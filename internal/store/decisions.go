package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// validDecisionOutcomes mirrors the migration's
// CHECK (outcome IN ('sent','refused','failed')) constraint (D-12), so an
// invalid value is a Go error at the call site rather than a mid-insert SQL
// error.
var validDecisionOutcomes = map[string]bool{
	"sent":    true,
	"refused": true,
	"failed":  true,
}

// defaultDecisionListLimit applies when the caller passes limit <= 0.
const defaultDecisionListLimit = 200

// DecisionRecord is the write shape for one AI decision (D-12): the game
// text window the model saw, the reasoning it gave, the command it issued
// (if any) and the outcome. GameSessionID is nil when no transcript session
// is open at decision time (D-14 covers only saved-profile connections; a
// decision always has a connection, not always an open session row).
type DecisionRecord struct {
	UserID        uuid.UUID
	ConnectionID  uuid.UUID
	GameSessionID *uuid.UUID
	ModelName     string
	WindowText    string
	Reasoning     string
	Command       string
	Outcome       string
	FailureKind   string
	Notice        string
}

// Decision is the read shape for one stored AI decision: the same fields as
// DecisionRecord minus UserID (ownership is resolved by the caller before
// ListForConnection is ever reached), plus the two fields only the database
// assigns.
type Decision struct {
	ID            uuid.UUID
	ConnectionID  uuid.UUID
	GameSessionID *uuid.UUID
	ModelName     string
	WindowText    string
	Reasoning     string
	Command       string
	Outcome       string
	FailureKind   string
	Notice        string
	CreatedAt     time.Time
}

// DecisionStore handles ai_decisions database operations. Same constructor
// shape as ProfileStore and TranscriptStore: hand-written SQL with
// database/sql, no ORM, no query builder. This file adds no logging — the
// driver (internal/driver) owns the [AI-PLAYER] decision log lines.
type DecisionStore struct {
	db *sql.DB
}

// NewDecisionStore creates a new decision store.
func NewDecisionStore(db *sql.DB) *DecisionStore {
	return &DecisionStore{db: db}
}

// InsertDecision writes one decision row and returns its generated id and
// created_at. rec.Outcome is validated against the three allowed values
// before the SQL call, so a bad value is a call-site Go error rather than a
// CHECK-constraint violation. A nil GameSessionID inserts NULL.
func (s *DecisionStore) InsertDecision(rec DecisionRecord) (uuid.UUID, time.Time, error) {
	if !validDecisionOutcomes[rec.Outcome] {
		return uuid.Nil, time.Time{}, fmt.Errorf("store: invalid decision outcome %q", rec.Outcome)
	}

	var gameSessionID uuid.NullUUID
	if rec.GameSessionID != nil {
		gameSessionID = uuid.NullUUID{UUID: *rec.GameSessionID, Valid: true}
	}

	var id uuid.UUID
	var createdAt time.Time
	err := s.db.QueryRow(
		`INSERT INTO ai_decisions (user_id, connection_id, game_session_id, model_name, window_text, reasoning, command, outcome, failure_kind, notice)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 RETURNING id, created_at`,
		rec.UserID, rec.ConnectionID, gameSessionID, rec.ModelName, rec.WindowText, rec.Reasoning, rec.Command, rec.Outcome, rec.FailureKind, rec.Notice,
	).Scan(&id, &createdAt)
	if err != nil {
		return uuid.Nil, time.Time{}, err
	}
	return id, createdAt, nil
}

// ListForConnection lists a connection's decisions oldest first, so the
// panel renders them in the order they happened (D-12: "reloads the
// current connection's decisions after a page refresh"). Ownership of
// connectionID is resolved by the caller (plan 03-09's handler) before this
// is ever reached — this method takes an already-authorised connection id
// and performs no ownership check of its own. limit <= 0 defaults to 200.
func (s *DecisionStore) ListForConnection(connectionID uuid.UUID, limit int) ([]Decision, error) {
	if limit <= 0 {
		limit = defaultDecisionListLimit
	}

	rows, err := s.db.Query(
		`SELECT id, connection_id, game_session_id, model_name, window_text, reasoning, command, outcome, failure_kind, notice, created_at
		 FROM ai_decisions
		 WHERE connection_id = $1
		 ORDER BY created_at ASC, id ASC
		 LIMIT $2`,
		connectionID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	decisions := make([]Decision, 0)
	for rows.Next() {
		var d Decision
		var gameSessionID uuid.NullUUID
		if err := rows.Scan(
			&d.ID, &d.ConnectionID, &gameSessionID, &d.ModelName, &d.WindowText,
			&d.Reasoning, &d.Command, &d.Outcome, &d.FailureKind, &d.Notice, &d.CreatedAt,
		); err != nil {
			return nil, err
		}
		if gameSessionID.Valid {
			id := gameSessionID.UUID
			d.GameSessionID = &id
		}
		decisions = append(decisions, d)
	}
	return decisions, rows.Err()
}
