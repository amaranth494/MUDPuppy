package store

import (
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"
)

// CoachingStore handles game_sessions.coaching_suggestions database
// operations (D-08, plan 05-06): the standing coaching list AI-chatter
// pushes to and withdraws from, following TranscriptStore's Session Memory
// trio (SessionMemoryFor/UpdateSessionMemory/SessionMemoryForConnection)
// field for field, on the coaching_suggestions column migration 015 already
// added -- the same login-scoped lifetime Session Memory has (D-08 ties
// coaching's lifetime to Session Memory's exactly, D-31).
type CoachingStore struct {
	db *sql.DB
}

// NewCoachingStore creates a new coaching store.
func NewCoachingStore(db *sql.DB) *CoachingStore {
	return &CoachingStore{db: db}
}

// CoachingFor reads back a game session's standing coaching list, returning
// an empty slice rather than nil when the column holds an empty JSON array,
// SQL NULL, or no row matches gameSessionID at all -- mirroring
// SessionMemoryFor's exact read-failure tolerance: a read failure here is
// never fatal to the caller (internal/driver's prompt assembly and
// HandleChat's own push/withdraw step), it just means no coaching reaches
// this iteration.
func (s *CoachingStore) CoachingFor(gameSessionID uuid.UUID) ([]string, error) {
	var coachingJSON []byte
	err := s.db.QueryRow(
		`SELECT coaching_suggestions FROM game_sessions WHERE id = $1`,
		gameSessionID,
	).Scan(&coachingJSON)
	if err == sql.ErrNoRows {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	if len(coachingJSON) == 0 {
		return []string{}, nil
	}
	var bullets []string
	if err := json.Unmarshal(coachingJSON, &bullets); err != nil {
		return nil, err
	}
	if bullets == nil {
		bullets = []string{}
	}
	return bullets, nil
}

// UpdateCoaching stores gameSessionID's standing coaching list (D-08) in one
// UPDATE, mirroring UpdateSessionMemory's exact single-statement shape and
// nil-safety: a nil coaching list is stored as an empty JSON array, never a
// SQL NULL. This is the store-side half of the coaching channel's single
// writer -- internal/driver/chat.go's HandleChat is the only caller of this
// method anywhere in the codebase (T-5-26, TestCoachingHasOneWriter).
func (s *CoachingStore) UpdateCoaching(gameSessionID uuid.UUID, coaching []string) error {
	if coaching == nil {
		coaching = []string{}
	}
	coachingJSON, err := json.Marshal(coaching)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		`UPDATE game_sessions SET coaching_suggestions = $1 WHERE id = $2`,
		coachingJSON, gameSessionID,
	)
	return err
}

// coachingForConnectionSQL reads connectionID's newest game session within
// the owner's current MUDPuppy login (D-08, the same login_started_at
// boundary sessionMemoryForConnectionSQL uses, D-31), declared as a
// package-level constant so coaching_test.go (no database connection) can
// assert on its exact text.
const coachingForConnectionSQL = `SELECT gs.coaching_suggestions
	FROM game_sessions gs
	JOIN users u ON u.id = gs.user_id
	WHERE gs.connection_id = $1
	  AND gs.started_at >= u.login_started_at
	ORDER BY gs.started_at DESC
	LIMIT 1`

// CoachingForConnection reads back the standing coaching list for
// connectionID within the current MUDPuppy login (D-08), for the panel's
// read-only "Coaching in effect" reload (plan 05-06-03's GET endpoint).
// Returns an empty, never nil, slice and a nil error when this login has no
// game session for the connection yet -- the same "nothing yet" shape
// SessionMemoryForConnection already returns.
func (s *CoachingStore) CoachingForConnection(connectionID uuid.UUID) ([]string, error) {
	var coachingJSON []byte
	err := s.db.QueryRow(coachingForConnectionSQL, connectionID).Scan(&coachingJSON)
	if err == sql.ErrNoRows {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	if len(coachingJSON) == 0 {
		return []string{}, nil
	}
	var bullets []string
	if err := json.Unmarshal(coachingJSON, &bullets); err != nil {
		return nil, err
	}
	if bullets == nil {
		bullets = []string{}
	}
	return bullets, nil
}
