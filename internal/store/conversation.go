package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ConversationLine is one line of the owner/AI-chatter conversation (D-01),
// append-only and scoped to the game session it happened during. Seq is a
// monotonically increasing ordering column assigned by AppendChatLine
// itself, the same kind game_session_lines' own seq column is, so the
// conversation can be read back in the order it happened.
type ConversationLine struct {
	ID        int64
	Seq       int64
	Speaker   string
	Text      string
	CreatedAt time.Time
}

// validConversationSpeakers mirrors the migration's
// CHECK (speaker IN ('owner','chatter','system')) constraint, so an invalid
// speaker is a Go error at the call site rather than a mid-insert SQL
// error.
var validConversationSpeakers = map[string]bool{
	"owner":   true,
	"chatter": true,
	"system":  true,
}

// ConversationStore handles conversation_lines database operations. Same
// constructor shape as QuestStore: hand-written SQL with database/sql, no
// ORM, no query builder.
type ConversationStore struct {
	db *sql.DB
}

// NewConversationStore creates a new conversation store.
func NewConversationStore(db *sql.DB) *ConversationStore {
	return &ConversationStore{db: db}
}

// nextConversationSeqSQL and insertConversationLineSQL are declared as
// package-level constants, rather than inline literals, so
// conversation_test.go (no database connection) can assert on their exact
// text.
const nextConversationSeqSQL = `SELECT COALESCE(MAX(seq), 0) + 1 FROM conversation_lines WHERE game_session_id = $1`

const insertConversationLineSQL = `INSERT INTO conversation_lines (game_session_id, seq, speaker, text)
	VALUES ($1, $2, $3, $4)
	RETURNING id, created_at`

// AppendChatLine inserts one conversation line, wrapped in an explicit
// transaction as this file's siblings are (assigning the next seq and
// inserting the row in the same transaction, so two concurrent appends to
// the same game session never collide), returning the stored row so the
// caller can notify the browser with the same id the database holds.
// Rejects a speaker outside the three allowed values with a Go error
// before any SQL runs.
func (s *ConversationStore) AppendChatLine(gameSessionID uuid.UUID, speaker, text string) (ConversationLine, error) {
	if !validConversationSpeakers[speaker] {
		return ConversationLine{}, fmt.Errorf("conversation: invalid speaker %q", speaker)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return ConversationLine{}, err
	}
	defer tx.Rollback()

	var seq int64
	if err := tx.QueryRow(nextConversationSeqSQL, gameSessionID).Scan(&seq); err != nil {
		return ConversationLine{}, err
	}

	line := ConversationLine{Seq: seq, Speaker: speaker, Text: text}
	if err := tx.QueryRow(insertConversationLineSQL, gameSessionID, seq, speaker, text).Scan(&line.ID, &line.CreatedAt); err != nil {
		return ConversationLine{}, err
	}

	if err := tx.Commit(); err != nil {
		return ConversationLine{}, err
	}
	return line, nil
}

// conversationForConnectionSQL reads back every conversation line for
// connectionID within the owner's current MUDPuppy login (D-31's same
// login_started_at boundary Session Memory and coaching share), oldest
// first, across every game session opened during that login -- the
// reload-on-attach read.
//
// Code review WR-03 of Phase 5: this read SPANS game sessions, and seq
// restarts at 1 in every one of them (nextConversationSeqSQL is scoped to a
// single game session). Ordering by seq alone therefore interleaved them:
// after one page refresh the panel showed question 1, question 3, answer 1,
// answer 3. The order is now by when the line was written, with the row id
// -- one sequence for the whole table, always increasing -- to settle two
// lines written in the same instant. seq stays the right key inside a
// single game session (conversationForSessionSQL, recentConversationSQL).
const conversationForConnectionSQL = `SELECT cl.id, cl.seq, cl.speaker, cl.text, cl.created_at
	FROM conversation_lines cl
	JOIN game_sessions gs ON gs.id = cl.game_session_id
	JOIN users u ON u.id = gs.user_id
	WHERE gs.connection_id = $1
	  AND gs.started_at >= u.login_started_at
	ORDER BY cl.created_at ASC, cl.id ASC`

// ConversationFor reads back the login-scoped conversation for
// connectionID, oldest first. Returns an empty, never nil, slice when this
// login has no conversation yet.
func (s *ConversationStore) ConversationFor(connectionID uuid.UUID) ([]ConversationLine, error) {
	return s.queryLines(conversationForConnectionSQL, connectionID)
}

// conversationForSessionSQL is the Logs page's per-session read, oldest
// first -- one specific game session's conversation. connectionID is joined
// through game_sessions in the WHERE clause as defence in depth behind the
// handler's own ownership check (T-3-04, mirroring GetSessionLines' exact
// precedent in transcripts.go): a session id belonging to another
// connection returns an empty result even if the caller guessed the
// session id correctly.
const conversationForSessionSQL = `SELECT cl.id, cl.seq, cl.speaker, cl.text, cl.created_at
	FROM conversation_lines cl
	JOIN game_sessions gs ON gs.id = cl.game_session_id
	WHERE cl.game_session_id = $1 AND gs.connection_id = $2
	ORDER BY cl.seq ASC`

// ConversationForSession reads back one game session's conversation, oldest
// first, for the Logs page. connectionID scopes the read the same way
// GetSessionLines does, so a session id from another connection returns an
// empty slice rather than that connection's own text.
func (s *ConversationStore) ConversationForSession(gameSessionID, connectionID uuid.UUID) ([]ConversationLine, error) {
	return s.queryLines(conversationForSessionSQL, gameSessionID, connectionID)
}

// recentConversationSQL is the bounded tail the chat prompt reads. Unlike
// every other read in this file, it orders newest first with a LIMIT --
// RecentConversation reverses the rows in Go before returning them, so the
// caller always receives them oldest first. This is deliberate, not a
// broken ORDER BY ... ASC convention: bounding to the newest N rows
// requires a descending order and a limit, then a reversal in Go, since SQL
// has no "limit from the end of an ascending order" clause.
const recentConversationSQL = `SELECT id, seq, speaker, text, created_at
	FROM conversation_lines
	WHERE game_session_id = $1
	ORDER BY seq DESC
	LIMIT $2`

// RecentConversation reads the newest limit lines of gameSessionID's
// conversation and returns them oldest first -- the bounded tail the chat
// prompt carries so a follow-up message makes sense (the chat call is a
// single shot with no memory of its own). A limit of zero or less returns
// an empty slice and runs no query.
func (s *ConversationStore) RecentConversation(gameSessionID uuid.UUID, limit int) ([]ConversationLine, error) {
	if limit <= 0 {
		return []ConversationLine{}, nil
	}

	lines, err := s.queryLines(recentConversationSQL, gameSessionID, limit)
	if err != nil {
		return nil, err
	}
	for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
		lines[i], lines[j] = lines[j], lines[i]
	}
	return lines, nil
}

// queryLines runs one of this file's SELECT statements and scans every row
// into a ConversationLine, returning an empty, never nil, slice when there
// are no rows.
func (s *ConversationStore) queryLines(query string, args ...interface{}) ([]ConversationLine, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lines := make([]ConversationLine, 0)
	for rows.Next() {
		var line ConversationLine
		if err := rows.Scan(&line.ID, &line.Seq, &line.Speaker, &line.Text, &line.CreatedAt); err != nil {
			return nil, err
		}
		lines = append(lines, line)
	}
	return lines, rows.Err()
}
