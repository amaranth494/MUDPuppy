package store

import (
	"strings"
	"testing"
)

// TestNormalizeGoal proves the single definition of goal identity (D-04):
// trim, case-fold, leave internal whitespace alone, blank stays blank.
func TestNormalizeGoal(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"trims leading and trailing whitespace", "  Find the Rod  ", "find the rod"},
		{"case-folds", "REACH LEVEL 10", "reach level 10"},
		{"leaves internal whitespace alone", "reach   level 10", "reach   level 10"},
		{"blank stays blank", "", ""},
		{"whitespace-only becomes blank", "   \t  ", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizeGoal(tc.in)
			if got != tc.want {
				t.Errorf("NormalizeGoal(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// sqlWithoutComments strips SQL line comments (lines whose first
// non-whitespace characters are "--"), mirroring migrations_test.go's own
// stripSQLComments helper, so a comment mentioning a forbidden word can
// never satisfy an assertion meant to check executable SQL. None of this
// file's three statement constants currently contain a "--" comment line,
// but stripping first keeps the assertion robust against a future edit that
// adds one.
func sqlWithoutComments(text string) string {
	lines := strings.Split(text, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "--") {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}

// allQuestStatements is every SQL statement this store issues, gathered so
// TestQuestStore_NeverCloses can scan all of them in one pass without a
// database connection.
var allQuestStatements = []string{ensureActiveQuestSQL, activeQuestForSQL, updateBulletsSQL}

// TestQuestStore_EnsureActiveReactivates proves D-04's atomic
// reactivate-or-create shape without a database connection: the upsert
// names the partial index's own conflict target (connection_id,
// goal_text_normalized) WHERE status = 'active', so a repeated goal
// reactivates the existing row rather than racing a SELECT-then-INSERT.
func TestQuestStore_EnsureActiveReactivates(t *testing.T) {
	stmt := sqlWithoutComments(ensureActiveQuestSQL)

	const conflictTarget = "ON CONFLICT (connection_id, goal_text_normalized) WHERE status = 'active'"
	if !strings.Contains(stmt, conflictTarget) {
		t.Errorf("expected EnsureActiveQuest's statement to name the partial index's conflict target %q; got:\n%s", conflictTarget, stmt)
	}

	if !strings.Contains(stmt, "DO UPDATE SET updated_at = NOW()") {
		t.Errorf("expected a repeated goal to reactivate via DO UPDATE, not duplicate via a plain INSERT; got:\n%s", stmt)
	}

	if !strings.Contains(stmt, "INSERT INTO quests") {
		t.Errorf("expected the statement to be an INSERT against the quests table; got:\n%s", stmt)
	}
}

// TestQuestStore_NeverCloses proves D-04: Phase 4 closes nothing. No
// statement in this store ever sets status to anything but 'active', and no
// statement contains any of the four terminal-state words the memory model
// reserves for Phase 6 (succeeded, failed, abandoned, invalidated).
func TestQuestStore_NeverCloses(t *testing.T) {
	terminalWords := []string{"succeeded", "failed", "abandoned", "invalidated"}

	for i, raw := range allQuestStatements {
		stmt := sqlWithoutComments(raw)
		lower := strings.ToLower(stmt)

		for _, word := range terminalWords {
			if strings.Contains(lower, word) {
				t.Errorf("statement %d contains forbidden terminal-state word %q (D-04: Phase 4 never closes a Quest):\n%s", i, word, stmt)
			}
		}

		// Every occurrence of "status" assigned a value in this file must
		// assign it 'active' -- SET status = '...' or VALUES (...,
		// 'active') are the only two shapes any statement here uses.
		if strings.Contains(lower, "status =") && !strings.Contains(lower, "status = 'active'") {
			t.Errorf("statement %d assigns status to something other than 'active':\n%s", i, stmt)
		}
	}
}
