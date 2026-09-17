package store

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Quest is the read shape of one row in the quests table (D-04, D-11): a
// per-connection, per-goal store of terse bullets that outlives the game
// session the goal was set in. There is deliberately no method anywhere in
// this file that sets Status to anything other than "active" -- Phase 4
// never closes a Quest (D-04, memory model Quest closure is Phase 6).
// Closing a Quest (succeeded, failed, abandoned, invalidated) and promoting
// its bullets into Historical Memory is Phase 6's job, not this file's.
type Quest struct {
	ID                 uuid.UUID
	UserID             uuid.UUID
	ConnectionID       uuid.UUID
	GoalText           string
	GoalTextNormalized string
	Status             string
	Bullets            []string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// QuestStore handles quests database operations. Same constructor shape as
// DecisionStore and TranscriptStore: hand-written SQL with database/sql, no
// ORM, no query builder.
type QuestStore struct {
	db *sql.DB
}

// NewQuestStore creates a new quest store.
func NewQuestStore(db *sql.DB) *QuestStore {
	return &QuestStore{db: db}
}

// The three statements below are declared as constants, rather than inline
// string literals, so TestQuestStore_EnsureActiveReactivates and
// TestQuestStore_NeverCloses (this package's own tests, no database
// connection) can assert on their exact text: that the upsert names the
// partial index's conflict target, and that no statement anywhere in this
// file ever sets status to anything but 'active' -- D-04's "Phase 4 never
// closes a Quest" made mechanically checkable.
const ensureActiveQuestSQL = `INSERT INTO quests (user_id, connection_id, goal_text, goal_text_normalized, status)
	 VALUES ($1, $2, $3, $4, 'active')
	 ON CONFLICT (connection_id, goal_text_normalized) WHERE status = 'active'
	 DO UPDATE SET updated_at = NOW()
	 RETURNING id, user_id, connection_id, goal_text, goal_text_normalized, status, bullets, created_at, updated_at`

const activeQuestForSQL = `SELECT id, user_id, connection_id, goal_text, goal_text_normalized, status, bullets, created_at, updated_at
	 FROM quests
	 WHERE connection_id = $1 AND goal_text_normalized = $2 AND status = 'active'`

const updateBulletsSQL = `UPDATE quests SET bullets = $1, updated_at = NOW() WHERE id = $2`

// NormalizeGoal is the single definition of goal identity (D-04): trim and
// case-fold. Exported so the profiles handler and this package's own tests
// use the exact same function, never two independent implementations of the
// same idea. Internal whitespace is left alone -- only leading/trailing
// whitespace is trimmed.
func NormalizeGoal(goal string) string {
	return strings.ToLower(strings.TrimSpace(goal))
}

// EnsureActiveQuest creates or reactivates the active Quest named by
// goalText for a connection (D-04). A blank goal (after trimming) is a
// no-op returning a zero Quest and no error -- clearing the goal box
// creates nothing, and the previously active Quest, if any, stays open and
// untouched (Phase 4 closes nothing). A repeated goal (matched by
// NormalizeGoal) reactivates the existing active Quest rather than
// duplicating it, via one INSERT ... ON CONFLICT round trip against the
// partial unique index on (connection_id, goal_text_normalized) WHERE
// status = 'active' (migration 013), so this is race-free without a
// SELECT-then-INSERT.
func (s *QuestStore) EnsureActiveQuest(userID, connectionID uuid.UUID, goalText string) (Quest, error) {
	normalized := NormalizeGoal(goalText)
	if normalized == "" {
		return Quest{}, nil
	}

	var q Quest
	var bulletsJSON []byte
	err := s.db.QueryRow(
		ensureActiveQuestSQL,
		userID, connectionID, goalText, normalized,
	).Scan(&q.ID, &q.UserID, &q.ConnectionID, &q.GoalText, &q.GoalTextNormalized, &q.Status, &bulletsJSON, &q.CreatedAt, &q.UpdatedAt)
	if err != nil {
		return Quest{}, err
	}
	if err := json.Unmarshal(bulletsJSON, &q.Bullets); err != nil {
		return Quest{}, err
	}
	return q, nil
}

// ActiveQuestFor reads back the active Quest for a connection whose
// normalized goal text matches goalText. The second return value reports
// whether an active Quest was found.
func (s *QuestStore) ActiveQuestFor(connectionID uuid.UUID, goalText string) (Quest, bool, error) {
	normalized := NormalizeGoal(goalText)
	if normalized == "" {
		return Quest{}, false, nil
	}

	var q Quest
	var bulletsJSON []byte
	err := s.db.QueryRow(
		activeQuestForSQL,
		connectionID, normalized,
	).Scan(&q.ID, &q.UserID, &q.ConnectionID, &q.GoalText, &q.GoalTextNormalized, &q.Status, &bulletsJSON, &q.CreatedAt, &q.UpdatedAt)
	if err == sql.ErrNoRows {
		return Quest{}, false, nil
	}
	if err != nil {
		return Quest{}, false, err
	}
	if err := json.Unmarshal(bulletsJSON, &q.Bullets); err != nil {
		return Quest{}, false, err
	}
	return q, true, nil
}

// UpdateBullets stores a Quest's curated bullet list in one UPDATE. Unused
// until plan 04-08 (Quest Memory curation during play) -- present now so
// the Quest's storage shape is settled before that plan extends the prompt
// and answer schema.
func (s *QuestStore) UpdateBullets(questID uuid.UUID, bullets []string) error {
	if bullets == nil {
		bullets = []string{}
	}
	bulletsJSON, err := json.Marshal(bullets)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		updateBulletsSQL,
		bulletsJSON, questID,
	)
	return err
}
