package store

import (
	"strings"
	"testing"
)

// allRetentionStatements is every SQL statement RetentionJob issues,
// gathered so the tests below can scan all of them in one pass without a
// database connection, mirroring quests_test.go's allQuestStatements.
var allRetentionStatements = []string{
	clearOldDecisionSnapshotsSQL,
	deleteOldTranscriptLinesSQL,
	clearConnectionDecisionSnapshotsSQL,
	deleteConnectionTranscriptLinesSQL,
}

// perConnectionRetentionStatements is the subset of allRetentionStatements
// used by DeleteCapturedTextForConnection -- the owner's immediate,
// per-profile delete.
var perConnectionRetentionStatements = []string{
	clearConnectionDecisionSnapshotsSQL,
	deleteConnectionTranscriptLinesSQL,
}

// TestRetentionWindows proves D-21's two pruning windows are exactly what
// the owner locked: seven days for a decision's snapshot, thirty for a
// session's transcript lines.
func TestRetentionWindows(t *testing.T) {
	if DecisionSnapshotRetentionDays != 7 {
		t.Errorf("DecisionSnapshotRetentionDays = %d, want 7", DecisionSnapshotRetentionDays)
	}
	if TranscriptRetentionDays != 30 {
		t.Errorf("TranscriptRetentionDays = %d, want 30", TranscriptRetentionDays)
	}
}

// TestRetention_TouchesOnlyCapturedText proves every statement RetentionJob
// issues updates or deletes only window_text or game_session_lines -- and
// names none of the columns D-21 keeps forever (the decision's audit trail)
// -- so a prune cannot silently widen its own reach. Comments are stripped
// first (mirroring migrations_test.go's stripSQLComments), so a comment
// mentioning a forbidden word can never satisfy this assertion.
func TestRetention_TouchesOnlyCapturedText(t *testing.T) {
	preservedColumns := []string{"reasoning", "command", "outcome", "failure_kind", "notice"}

	for i, raw := range allRetentionStatements {
		stmt := stripSQLComments(raw)
		lower := strings.ToLower(stmt)

		touchesSnapshot := strings.Contains(lower, "window_text")
		touchesTranscript := strings.Contains(lower, "game_session_lines")
		if !touchesSnapshot && !touchesTranscript {
			t.Errorf("statement %d touches neither window_text nor game_session_lines:\n%s", i, stmt)
		}

		if strings.Contains(lower, "delete from ai_decisions") {
			t.Errorf("statement %d deletes from ai_decisions -- D-21 keeps the decision row forever:\n%s", i, stmt)
		}

		for _, col := range preservedColumns {
			if strings.Contains(lower, col) {
				t.Errorf("statement %d names preserved column %q, which D-21 keeps forever:\n%s", i, col, stmt)
			}
		}
	}
}

// TestRetention_LeavesMemoryAlone proves D-21's own instruction: memory
// layers (Session Memory, Quest Memory) are not captured text and are not
// pruned by this job. No statement anywhere in this file mentions
// session_memory, quests, or bullets.
func TestRetention_LeavesMemoryAlone(t *testing.T) {
	forbidden := []string{"session_memory", "quests", "bullets"}

	for i, raw := range allRetentionStatements {
		stmt := stripSQLComments(raw)
		lower := strings.ToLower(stmt)
		for _, word := range forbidden {
			if strings.Contains(lower, word) {
				t.Errorf("statement %d mentions forbidden word %q (D-21: memory is not captured text):\n%s", i, word, stmt)
			}
		}
	}
}

// TestRetention_PerProfileDeleteIsScoped proves the owner's immediate
// per-connection delete can never reach past its own connection: both of
// DeleteCapturedTextForConnection's statements carry a connection_id
// predicate, so a delete can never be global.
func TestRetention_PerProfileDeleteIsScoped(t *testing.T) {
	for i, raw := range perConnectionRetentionStatements {
		stmt := stripSQLComments(raw)
		lower := strings.ToLower(stmt)
		if !strings.Contains(lower, "connection_id") {
			t.Errorf("per-connection statement %d has no connection_id predicate:\n%s", i, stmt)
		}
	}
}
