package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// GameSessionSummary is one row of a profile's session list (D-17): a
// saved-profile game connection's start/end and how many lines it holds.
type GameSessionSummary struct {
	ID        uuid.UUID  `json:"id"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at"`
	LineCount int        `json:"line_count"`
}

// GameSessionLine is one line of a session transcript, tagged by source
// (human, ai, game or marker — D-14, D-15). Seq is assigned by the caller
// (internal/session), not by the database, so it survives batching.
type GameSessionLine struct {
	Seq       int64     `json:"seq"`
	Source    string    `json:"source"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

// validTranscriptSources mirrors the migration's
// CHECK (source IN ('human','ai','game','marker')) constraint (T-3-28), so
// an invalid source is a Go error at the call site rather than a
// mid-batch SQL error.
var validTranscriptSources = map[string]bool{
	"human":  true,
	"ai":     true,
	"game":   true,
	"marker": true,
}

// defaultSessionListLimit and defaultLineListLimit apply when the caller
// passes limit <= 0.
const (
	defaultSessionListLimit = 200
	defaultLineListLimit    = 10000
)

// TranscriptStore handles game_sessions / game_session_lines database
// operations. Same constructor shape as ProfileStore: hand-written SQL
// with database/sql, no ORM, no query builder.
type TranscriptStore struct {
	db *sql.DB
}

// NewTranscriptStore creates a new transcript store.
func NewTranscriptStore(db *sql.DB) *TranscriptStore {
	return &TranscriptStore{db: db}
}

// openGameSessionSQL is declared as a package-level constant, rather than
// an inline literal, so transcripts_test.go (no database connection) can
// assert on its exact text: one INSERT, no SELECT-then-INSERT race, that
// seeds session_memory AND coaching_suggestions (plan 05-06, D-08) from the
// same user_id and connection_id's most recent earlier game session started
// at or after users.login_started_at, falling back to an empty array for
// either column when there is none (D-31, amending D-10 -- Session Memory
// now lives for the MUDPuppy login, not the game connection; D-08 ties
// coaching's lifetime to Session Memory's exactly).
const openGameSessionSQL = `INSERT INTO game_sessions (user_id, connection_id, session_memory, coaching_suggestions)
	VALUES ($1, $2,
		COALESCE(
			(SELECT gs.session_memory
			 FROM game_sessions gs
			 JOIN users u ON u.id = gs.user_id
			 WHERE gs.user_id = $1
			   AND gs.connection_id = $2
			   AND gs.started_at >= u.login_started_at
			 ORDER BY gs.started_at DESC
			 LIMIT 1),
			'[]'::jsonb
		),
		COALESCE(
			(SELECT gs.coaching_suggestions
			 FROM game_sessions gs
			 JOIN users u ON u.id = gs.user_id
			 WHERE gs.user_id = $1
			   AND gs.connection_id = $2
			   AND gs.started_at >= u.login_started_at
			 ORDER BY gs.started_at DESC
			 LIMIT 1),
			'[]'::jsonb
		)
	)
	RETURNING id`

// OpenGameSession starts a new transcript row for a saved-profile
// connection (D-14). The caller (internal/session) is responsible for
// never calling this for a quick connect (no profile).
//
// The new row's session_memory is seeded, in this same single INSERT, from
// the most recent earlier game_sessions row for the same user_id and
// connection_id whose started_at is at or after that user's
// login_started_at -- so Session Memory lives for the MUDPuppy login, per
// connection profile (D-31, amending D-10), not for the game connection: a
// page refresh, a game reconnect, a dropped socket, or a server redeploy
// all keep it, because the game session(s) they open still start after the
// same login boundary. The first game connection after a new sign-in (or
// after a sign-out, which also bumps login_started_at) finds no such row
// and seeds '[]'::jsonb.
func (s *TranscriptStore) OpenGameSession(userID, connectionID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.db.QueryRow(openGameSessionSQL, userID, connectionID).Scan(&id)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

// AppendGameLines appends a batch of transcript lines in one multi-row
// INSERT (never one statement per line) and bumps game_sessions.line_count
// by the batch size, in the same transaction. A zero-length batch is a
// no-op returning nil. Rejects any source outside human/ai/game/marker
// with a Go error before any SQL runs (T-3-28).
func (s *TranscriptStore) AppendGameLines(gameSessionID uuid.UUID, lines []GameSessionLine) error {
	if len(lines) == 0 {
		return nil
	}
	for _, line := range lines {
		if !validTranscriptSources[line.Source] {
			return fmt.Errorf("transcript: invalid source %q", line.Source)
		}
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var sb strings.Builder
	sb.WriteString("INSERT INTO game_session_lines (game_session_id, seq, source, text) VALUES ")
	args := make([]interface{}, 0, len(lines)*4)
	for i, line := range lines {
		if i > 0 {
			sb.WriteString(", ")
		}
		base := i * 4
		fmt.Fprintf(&sb, "($%d, $%d, $%d, $%d)", base+1, base+2, base+3, base+4)
		args = append(args, gameSessionID, line.Seq, line.Source, line.Text)
	}

	if _, err := tx.Exec(sb.String(), args...); err != nil {
		return err
	}

	if _, err := tx.Exec(
		`UPDATE game_sessions SET line_count = line_count + $1 WHERE id = $2`,
		len(lines), gameSessionID,
	); err != nil {
		return err
	}

	return tx.Commit()
}

// CloseGameSession sets ended_at to now, only when it is still null (a
// repeat call is a harmless no-op).
func (s *TranscriptStore) CloseGameSession(gameSessionID uuid.UUID) error {
	_, err := s.db.Exec(
		`UPDATE game_sessions SET ended_at = NOW() WHERE id = $1 AND ended_at IS NULL`,
		gameSessionID,
	)
	return err
}

// closeOrphanedGameSessionsSQL is a package-level constant so
// transcripts_test.go (no database connection) can pin its reach: it sets
// ended_at, only on rows whose ended_at is still null, and names no other
// column -- Session Memory in particular is left exactly as the interrupted
// session last wrote it.
const closeOrphanedGameSessionsSQL = `UPDATE game_sessions SET ended_at = NOW() WHERE ended_at IS NULL`

// CloseOrphanedGameSessions closes every game session a previous server
// process left open, returning how many rows it closed. A stopped or
// redeployed process never reaches CloseGameSession for its live
// connections, so without this their rows keep a null ended_at forever
// (seen on staging 2026-09-17: each redeploy left one), and anything that
// reads "closed game sessions" -- Phase 6's Session Memory consolidation --
// would skip or misread them.
//
// Assumption: the server runs as a single instance that holds every live
// game connection in memory, so at process start no game session can be
// live and every open row is an orphan. Call it once, at startup, before
// the server accepts connections -- never while sessions may be open. If a
// redeploy ever overlaps two processes, a session still draining in the old
// one is merely stamped a little early: its own CloseGameSession becomes
// the no-op it is already written to be.
func (s *TranscriptStore) CloseOrphanedGameSessions() (int64, error) {
	res, err := s.db.Exec(closeOrphanedGameSessionsSQL)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// SessionMemoryFor reads back a game session's curated Session Memory
// bullets -- which now live for the MUDPuppy login (D-31, amending D-10),
// not the game session's own row, though each game session still keeps its
// own copy here for Phase 6 -- returning an empty slice rather than nil
// when the column holds an empty JSON array, when it holds SQL NULL, or when no
// row matches gameSessionID at all -- a read failure here is never fatal
// to the caller (internal/driver's prompt assembly), it just means no
// Session Memory reaches the prompt this iteration. Unused by any real
// writer before plan 04-08 curates Session Memory during play; plan 04-07
// reads it into the prompt so the column's shape is proven correct before
// anything writes to it.
func (s *TranscriptStore) SessionMemoryFor(gameSessionID uuid.UUID) ([]string, error) {
	var memoryJSON []byte
	err := s.db.QueryRow(
		`SELECT session_memory FROM game_sessions WHERE id = $1`,
		gameSessionID,
	).Scan(&memoryJSON)
	if err == sql.ErrNoRows {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	if len(memoryJSON) == 0 {
		return []string{}, nil
	}
	var bullets []string
	if err := json.Unmarshal(memoryJSON, &bullets); err != nil {
		return nil, err
	}
	if bullets == nil {
		bullets = []string{}
	}
	return bullets, nil
}

// UpdateSessionMemory stores gameSessionID's curated Session Memory bullets
// (D-10, plan 04-08) in one UPDATE, following CloseGameSession's exact
// single-statement shape. A nil memory is stored as an empty JSON array,
// mirroring QuestStore.UpdateBullets' own nil-safety, so "replace with
// nothing" is always a valid array, never a SQL NULL.
func (s *TranscriptStore) UpdateSessionMemory(gameSessionID uuid.UUID, memory []string) error {
	if memory == nil {
		memory = []string{}
	}
	memoryJSON, err := json.Marshal(memory)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		`UPDATE game_sessions SET session_memory = $1 WHERE id = $2`,
		memoryJSON, gameSessionID,
	)
	return err
}

// sessionMemoryForConnectionSQL reads connectionID's newest game session
// within the owner's current MUDPuppy login (D-31): the same boundary
// openGameSessionSQL seeds from, so what the panel reads back is exactly
// what the next decision will be given. It deliberately does not filter on
// ended_at: a server restart leaves the interrupted game session's row
// without an end time, and "the newest open row" then picked a stale one
// (seen on staging 2026-09-17, RUN B read four bullets from an orphaned
// session instead of the two from the session that had just closed).
const sessionMemoryForConnectionSQL = `SELECT gs.session_memory
	FROM game_sessions gs
	JOIN users u ON u.id = gs.user_id
	WHERE gs.connection_id = $1
	  AND gs.started_at >= u.login_started_at
	ORDER BY gs.started_at DESC
	LIMIT 1`

// SessionMemoryForConnection reads back the Session Memory bullets for
// connectionID within the current MUDPuppy login (D-31, amending D-10) for
// the owner's read-only memory endpoint (plan 04-08). Returns an empty,
// never nil, slice and a nil error when this login has no game session for
// the connection yet — the same "nothing yet" shape SessionMemoryFor already
// returns for a missing row, so a profile that has never engaged autopilot
// reads back an empty list rather than an error.
func (s *TranscriptStore) SessionMemoryForConnection(connectionID uuid.UUID) ([]string, error) {
	var memoryJSON []byte
	err := s.db.QueryRow(sessionMemoryForConnectionSQL, connectionID).Scan(&memoryJSON)
	if err == sql.ErrNoRows {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	if len(memoryJSON) == 0 {
		return []string{}, nil
	}
	var bullets []string
	if err := json.Unmarshal(memoryJSON, &bullets); err != nil {
		return nil, err
	}
	if bullets == nil {
		bullets = []string{}
	}
	return bullets, nil
}

// latestGameSessionForConnectionSQL names connectionID's newest game session
// within the owner's current MUDPuppy login: the SAME row
// sessionMemoryForConnectionSQL and coachingForConnectionSQL read, selected
// by id (code review WR-01 of Phase 5). It deliberately does not filter on
// ended_at, for the same reason they do not: the row is wanted whether the
// game connection is up or down.
const latestGameSessionForConnectionSQL = `SELECT gs.id
	FROM game_sessions gs
	JOIN users u ON u.id = gs.user_id
	WHERE gs.connection_id = $1
	  AND gs.started_at >= u.login_started_at
	ORDER BY gs.started_at DESC
	LIMIT 1`

// LatestGameSessionForConnection returns the id of connectionID's newest
// game session within the current MUDPuppy login, and false when this login
// has none yet (code review WR-01 of Phase 5). AI-chatter attaches a chat
// exchange to this row, so chat keeps working while the game connection is
// down (D-06), and -- because the row is found FROM the connection -- the
// conversation and coaching it writes always belong to the same connection
// whose profile it read (code review WR-10 of Phase 5).
func (s *TranscriptStore) LatestGameSessionForConnection(connectionID uuid.UUID) (uuid.UUID, bool, error) {
	var id uuid.UUID
	err := s.db.QueryRow(latestGameSessionForConnectionSQL, connectionID).Scan(&id)
	if err == sql.ErrNoRows {
		return uuid.UUID{}, false, nil
	}
	if err != nil {
		return uuid.UUID{}, false, err
	}
	return id, true, nil
}

// ListSessionsForConnection lists a connection's sessions newest-first.
// Ownership of connectionID is resolved by the caller (internal/profiles,
// via ProfileStore.GetProfileByConnection) before this is ever reached
// (T-3-04) — this method takes an already-authorised connection id and
// performs no ownership check of its own.
func (s *TranscriptStore) ListSessionsForConnection(connectionID uuid.UUID, limit int) ([]GameSessionSummary, error) {
	if limit <= 0 {
		limit = defaultSessionListLimit
	}

	rows, err := s.db.Query(
		`SELECT id, started_at, ended_at, line_count
		 FROM game_sessions
		 WHERE connection_id = $1
		 ORDER BY started_at DESC
		 LIMIT $2`,
		connectionID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	summaries := make([]GameSessionSummary, 0)
	for rows.Next() {
		var summary GameSessionSummary
		var endedAt sql.NullTime
		if err := rows.Scan(&summary.ID, &summary.StartedAt, &endedAt, &summary.LineCount); err != nil {
			return nil, err
		}
		if endedAt.Valid {
			t := endedAt.Time
			summary.EndedAt = &t
		}
		summaries = append(summaries, summary)
	}
	return summaries, rows.Err()
}

// GetSessionLines reads one session's lines, oldest-first. connectionID is
// joined through game_sessions in the WHERE clause as defence in depth
// behind the handler's ownership check (T-3-04): a session id belonging
// to another connection returns an empty result even if the caller
// guessed the session id correctly.
func (s *TranscriptStore) GetSessionLines(gameSessionID, connectionID uuid.UUID, limit int) ([]GameSessionLine, error) {
	if limit <= 0 {
		limit = defaultLineListLimit
	}

	rows, err := s.db.Query(
		`SELECT l.seq, l.source, l.text, l.created_at
		 FROM game_session_lines l
		 JOIN game_sessions gs ON gs.id = l.game_session_id
		 WHERE l.game_session_id = $1 AND gs.connection_id = $2
		 ORDER BY l.seq ASC
		 LIMIT $3`,
		gameSessionID, connectionID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lines := make([]GameSessionLine, 0)
	for rows.Next() {
		var line GameSessionLine
		if err := rows.Scan(&line.Seq, &line.Source, &line.Text, &line.CreatedAt); err != nil {
			return nil, err
		}
		lines = append(lines, line)
	}
	return lines, rows.Err()
}
