package store

import (
	"database/sql"

	"github.com/google/uuid"
)

// DecisionSnapshotRetentionDays governs how long a decision's window_text
// snapshot (the game text the model was reading, blocked rows included)
// survives before it is cleared in place; every other column on the row --
// timestamp, reasoning, command, outcome, block reason -- is kept forever
// (D-21, closing DR-3-01 and DR-3.1-03).
const DecisionSnapshotRetentionDays = 7

// TranscriptRetentionDays governs how long a session's game_session_lines
// rows survive before they are deleted; the game_sessions summary row
// (started/ended timestamps, line_count) is left untouched, so the log
// page still lists the session and shows it as empty (D-21).
const TranscriptRetentionDays = 30

// RetentionCounts is what one retention run or one per-profile delete
// reports: how many decision snapshots were cleared and how many
// transcript lines were deleted. Counts only -- never a row's text -- so
// the [AI-PLAYER] retention log line (cmd/server/main.go) never carries
// captured text (T-4-05).
type RetentionCounts struct {
	SnapshotsCleared       int64
	TranscriptLinesDeleted int64
}

// The four statements below are declared as package-level constants,
// rather than inline literals, so retention_test.go (no database
// connection) can assert on their exact text: that each names only
// window_text or game_session_lines, that neither per-profile statement
// can reach past its own connection_id, and that no statement anywhere in
// this file touches session_memory, quests, bullets, or any of the audit
// columns (reasoning, command, outcome, failure_kind, notice) D-21 keeps
// forever. The decision row itself is never deleted by any statement here
// -- D-21 keeps it forever.
const clearOldDecisionSnapshotsSQL = `UPDATE ai_decisions
	SET window_text = ''
	WHERE created_at < NOW() - INTERVAL '7 days'
	  AND window_text <> ''`

const deleteOldTranscriptLinesSQL = `DELETE FROM game_session_lines
	WHERE game_session_id IN (
		SELECT id FROM game_sessions WHERE started_at < NOW() - INTERVAL '30 days'
	)`

const clearConnectionDecisionSnapshotsSQL = `UPDATE ai_decisions
	SET window_text = ''
	WHERE connection_id = $1
	  AND window_text <> ''`

const deleteConnectionTranscriptLinesSQL = `DELETE FROM game_session_lines
	WHERE game_session_id IN (
		SELECT id FROM game_sessions WHERE connection_id = $1
	)`

// RetentionJob prunes captured game text on a schedule (D-21): the
// decision snapshot after DecisionSnapshotRetentionDays, the session
// transcript after TranscriptRetentionDays. Same constructor shape as
// DecisionStore/TranscriptStore/QuestStore: hand-written SQL with
// database/sql, no ORM, no query builder.
type RetentionJob struct {
	db *sql.DB
}

// NewRetentionJob creates a new retention job.
func NewRetentionJob(db *sql.DB) *RetentionJob {
	return &RetentionJob{db: db}
}

// Run prunes every decision snapshot older than
// DecisionSnapshotRetentionDays and every transcript line older than
// TranscriptRetentionDays, in one transaction, and reports how many rows
// each statement touched. The decision row itself is never deleted --
// D-21 keeps it forever.
func (j *RetentionJob) Run() (RetentionCounts, error) {
	return j.runInTx(clearOldDecisionSnapshotsSQL, deleteOldTranscriptLinesSQL)
}

// DeleteCapturedTextForConnection runs the owner's immediate "delete
// captured text now" action (D-21) for one connection: the same two
// operations Run performs, scoped to connectionID with no age condition.
// Both statements carry a connection_id predicate, so a delete can never
// reach another owner's data (T-4-09).
func (j *RetentionJob) DeleteCapturedTextForConnection(connectionID uuid.UUID) (RetentionCounts, error) {
	tx, err := j.db.Begin()
	if err != nil {
		return RetentionCounts{}, err
	}
	defer tx.Rollback()

	counts, err := execRetentionPair(tx, clearConnectionDecisionSnapshotsSQL, deleteConnectionTranscriptLinesSQL, connectionID)
	if err != nil {
		return RetentionCounts{}, err
	}

	if err := tx.Commit(); err != nil {
		return RetentionCounts{}, err
	}
	return counts, nil
}

// runInTx is the shared transaction shape Run uses: begin, run both
// statements (no arguments -- the age condition is baked into the SQL
// text), commit. Mirrors AppendGameLines' tx.Begin()/defer
// tx.Rollback()/tx.Commit() discipline.
func (j *RetentionJob) runInTx(clearSnapshotsSQL, deleteLinesSQL string) (RetentionCounts, error) {
	tx, err := j.db.Begin()
	if err != nil {
		return RetentionCounts{}, err
	}
	defer tx.Rollback()

	counts, err := execRetentionPair(tx, clearSnapshotsSQL, deleteLinesSQL)
	if err != nil {
		return RetentionCounts{}, err
	}

	if err := tx.Commit(); err != nil {
		return RetentionCounts{}, err
	}
	return counts, nil
}

// execRetentionPair runs the snapshot-clear statement followed by the
// transcript-line-delete statement against tx, with the same args passed
// to both (empty for the age-bounded pair, one connectionID for the
// per-profile pair), and returns both statements' row counts.
func execRetentionPair(tx *sql.Tx, clearSnapshotsSQL, deleteLinesSQL string, args ...interface{}) (RetentionCounts, error) {
	var counts RetentionCounts

	res, err := tx.Exec(clearSnapshotsSQL, args...)
	if err != nil {
		return RetentionCounts{}, err
	}
	counts.SnapshotsCleared, err = res.RowsAffected()
	if err != nil {
		return RetentionCounts{}, err
	}

	res, err = tx.Exec(deleteLinesSQL, args...)
	if err != nil {
		return RetentionCounts{}, err
	}
	counts.TranscriptLinesDeleted, err = res.RowsAffected()
	if err != nil {
		return RetentionCounts{}, err
	}

	return counts, nil
}
