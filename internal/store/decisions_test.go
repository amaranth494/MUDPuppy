package store

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

// TestRecentDecisionsSQL pins code review WR-02 of Phase 5 without a
// database: AI-chatter's read takes the NEWEST rows (descending order and a
// limit), where ListForConnection pages from the oldest, and it leaves the
// window text out.
func TestRecentDecisionsSQL(t *testing.T) {
	stmt := strings.Join(strings.Fields(sqlWithoutComments(recentDecisionsForConnectionSQL)), " ")

	for _, want := range []string{
		"FROM ai_decisions",
		"WHERE connection_id = $1",
		"ORDER BY created_at DESC, id DESC",
		"LIMIT $2",
	} {
		if !strings.Contains(stmt, want) {
			t.Errorf("expected recentDecisionsForConnectionSQL to contain %q:\n%s", want, stmt)
		}
	}
	if strings.Contains(stmt, " ASC") {
		t.Errorf("an ascending order with a limit reads the OLDEST rows, which is the bug:\n%s", stmt)
	}
	if strings.Contains(stmt, "window_text") {
		t.Errorf("the chat read must not load window_text:\n%s", stmt)
	}
	for _, col := range []string{"reasoning", "command", "outcome", "failure_kind"} {
		if !strings.Contains(stmt, col) {
			t.Errorf("expected the chat read to select %q:\n%s", col, stmt)
		}
	}
}

// TestRecentForConnection_NoLimitNoQuery proves a limit of zero or less
// answers an empty list without touching the database (the store here has a
// nil *sql.DB, so any query would panic).
func TestRecentForConnection_NoLimitNoQuery(t *testing.T) {
	s := NewDecisionStore(nil)
	for _, limit := range []int{0, -1} {
		got, err := s.RecentForConnection(uuid.New(), limit)
		if err != nil || got == nil || len(got) != 0 {
			t.Fatalf("RecentForConnection(limit=%d) = (%v, %v), want an empty non-nil slice and no error", limit, got, err)
		}
	}
}
