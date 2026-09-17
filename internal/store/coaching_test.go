package store

import (
	"strings"
	"testing"
)

// TestCoachingStoreSQL proves the coaching SQL shape without a database
// connection (D-08): the SQL constants name coaching_suggestions, the
// connection-scoped read carries the same login_started_at bound Session
// Memory uses, and openGameSessionSQL now seeds both session_memory and
// coaching_suggestions in the same INSERT.
func TestCoachingStoreSQL(t *testing.T) {
	if !strings.Contains(coachingForConnectionSQL, "coaching_suggestions") {
		t.Fatalf("expected coachingForConnectionSQL to name coaching_suggestions, got %q", coachingForConnectionSQL)
	}
	if !strings.Contains(coachingForConnectionSQL, "login_started_at") {
		t.Fatalf("expected coachingForConnectionSQL to carry the login_started_at bound, got %q", coachingForConnectionSQL)
	}
	if !strings.Contains(openGameSessionSQL, "session_memory") {
		t.Fatalf("expected openGameSessionSQL to still seed session_memory, got %q", openGameSessionSQL)
	}
	if !strings.Contains(openGameSessionSQL, "coaching_suggestions") {
		t.Fatalf("expected openGameSessionSQL to now seed coaching_suggestions, got %q", openGameSessionSQL)
	}
}
