package store

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

// TestRecentDecisionsSQL pins code review WR-02 of Phase 5 without a
// database: AI-chatter's read takes the NEWEST rows (descending order and a
// limit) and it leaves the window text out.
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

// TestListDecisionsSQL pins the panel reload's read without a database: it
// takes the NEWEST page (an inner descending order with the limit) and hands
// it back oldest first (an outer ascending order with no limit of its own).
// An ascending order carrying the limit is the defect: it reads the oldest
// page, so a refreshed panel on a long session showed stale decisions.
func TestListDecisionsSQL(t *testing.T) {
	stmt := strings.Join(strings.Fields(sqlWithoutComments(listDecisionsForConnectionSQL)), " ")

	inner := "WHERE connection_id = $1 ORDER BY created_at DESC, id DESC LIMIT $2 ) newest"
	if !strings.Contains(stmt, inner) {
		t.Errorf("expected the inner query to take the newest page (%q):\n%s", inner, stmt)
	}
	if !strings.HasSuffix(stmt, "ORDER BY created_at ASC, id ASC") {
		t.Errorf("expected the outer query to return the page oldest first:\n%s", stmt)
	}
	if strings.Count(stmt, "LIMIT") != 1 {
		t.Errorf("expected exactly one LIMIT, on the inner newest-first query:\n%s", stmt)
	}
	if !strings.Contains(stmt, "window_text") {
		t.Errorf("the panel/Logs read keeps window_text in the row:\n%s", stmt)
	}
}
