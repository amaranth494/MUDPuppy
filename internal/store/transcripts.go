package store

import (
	"database/sql"
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

// OpenGameSession starts a new transcript row for a saved-profile
// connection (D-14). The caller (internal/session) is responsible for
// never calling this for a quick connect (no profile).
func (s *TranscriptStore) OpenGameSession(userID, connectionID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.db.QueryRow(
		`INSERT INTO game_sessions (user_id, connection_id) VALUES ($1, $2) RETURNING id`,
		userID, connectionID,
	).Scan(&id)
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
