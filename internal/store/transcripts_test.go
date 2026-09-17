package store

import (
	"strings"
	"testing"
)

// TestOpenGameSession_IsOneStatement proves D-31's shape without a database
// connection: OpenGameSession's SQL is one INSERT with a seeding subquery,
// never a SELECT followed by a separate INSERT, so seeding session_memory
// and creating the game_sessions row can never race.
func TestOpenGameSession_IsOneStatement(t *testing.T) {
	stmt := sqlWithoutComments(openGameSessionSQL)

	if got := strings.Count(stmt, "INSERT INTO"); got != 1 {
		t.Errorf("expected exactly one INSERT INTO in openGameSessionSQL, got %d:\n%s", got, stmt)
	}
	if strings.Count(stmt, "INSERT INTO game_sessions") != 1 {
		t.Errorf("expected the statement to insert into game_sessions exactly once:\n%s", stmt)
	}
}

// TestOpenGameSession_SeedsFromSameUserAndConnection proves the seed
// subquery is scoped to the same user_id and connection_id being inserted
// -- $1 and $2 -- so one profile's Session Memory can never leak into
// another user's or another connection's new game session.
func TestOpenGameSession_SeedsFromSameUserAndConnection(t *testing.T) {
	stmt := sqlWithoutComments(openGameSessionSQL)

	for _, needle := range []string{"gs.user_id = $1", "gs.connection_id = $2"} {
		if !strings.Contains(stmt, needle) {
			t.Errorf("expected the seed subquery to contain %q:\n%s", needle, stmt)
		}
	}
}

// TestOpenGameSession_ComparesAgainstLoginStartedAt proves D-31's login
// boundary: the seed subquery only considers an earlier game session that
// started at or after the same user's login_started_at (migration 014), so
// a game session opened before the current login can never be inherited
// from.
func TestOpenGameSession_ComparesAgainstLoginStartedAt(t *testing.T) {
	stmt := sqlWithoutComments(openGameSessionSQL)

	for _, needle := range []string{"JOIN users u ON u.id = gs.user_id", "gs.started_at >= u.login_started_at"} {
		if !strings.Contains(stmt, needle) {
			t.Errorf("expected the seed subquery to contain %q:\n%s", needle, stmt)
		}
	}
}

// TestOpenGameSession_OrdersNewestFirstWithLimitOne proves the seed
// subquery picks the single most recent qualifying earlier game session,
// never an arbitrary or an averaged one.
func TestOpenGameSession_OrdersNewestFirstWithLimitOne(t *testing.T) {
	stmt := sqlWithoutComments(openGameSessionSQL)

	for _, needle := range []string{"ORDER BY gs.started_at DESC", "LIMIT 1"} {
		if !strings.Contains(stmt, needle) {
			t.Errorf("expected the seed subquery to contain %q:\n%s", needle, stmt)
		}
	}
}

// TestOpenGameSession_FallsBackToEmptyArray proves that when no earlier
// game session qualifies (a new sign-in, or after a sign-out, per D-31),
// the new row's session_memory seeds as an empty JSON array, never SQL
// NULL.
func TestOpenGameSession_FallsBackToEmptyArray(t *testing.T) {
	stmt := sqlWithoutComments(openGameSessionSQL)

	for _, needle := range []string{"COALESCE(", "'[]'::jsonb"} {
		if !strings.Contains(stmt, needle) {
			t.Errorf("expected the statement to contain %q:\n%s", needle, stmt)
		}
	}
}
