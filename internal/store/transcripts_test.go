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

// TestCloseOrphanedGameSessions_OnlyEndsOpenRows pins the startup sweep's
// reach without a database connection: one UPDATE of game_sessions that sets
// ended_at and nothing else, only where ended_at is still null -- so it can
// never re-stamp a session that closed normally -- and that never names
// session_memory, so an interrupted session keeps the bullets it last wrote.
func TestCloseOrphanedGameSessions_OnlyEndsOpenRows(t *testing.T) {
	stmt := sqlWithoutComments(closeOrphanedGameSessionsSQL)
	fields := strings.Join(strings.Fields(stmt), " ")

	if got := strings.Count(fields, "UPDATE "); got != 1 {
		t.Errorf("expected exactly one UPDATE in closeOrphanedGameSessionsSQL, got %d:\n%s", got, stmt)
	}
	if !strings.HasPrefix(fields, "UPDATE game_sessions SET ended_at = NOW() WHERE ") {
		t.Errorf("expected the statement to set ended_at, and only ended_at, on game_sessions:\n%s", stmt)
	}
	if !strings.HasSuffix(fields, " WHERE ended_at IS NULL") {
		t.Errorf("expected the statement's whole predicate to be ended_at IS NULL:\n%s", stmt)
	}
	for _, forbidden := range []string{"session_memory", "DELETE", "INSERT", ",", ";"} {
		if strings.Contains(fields, forbidden) {
			t.Errorf("closeOrphanedGameSessionsSQL must not contain %q:\n%s", forbidden, stmt)
		}
	}
}

// TestSessionMemoryForConnection_ReadsThisLoginsNewestSession pins the
// read-back the panel and the harness use to the same boundary
// openGameSessionSQL seeds from (D-31). It must not pick "the newest row
// without an end time": a server restart leaves the interrupted session's
// row open forever, and on staging that stale row was read back instead of
// the session that had just closed.
func TestSessionMemoryForConnection_ReadsThisLoginsNewestSession(t *testing.T) {
	stmt := sqlWithoutComments(sessionMemoryForConnectionSQL)

	if strings.Contains(stmt, "ended_at") {
		t.Errorf("the read-back must not filter on ended_at (orphaned open rows survive a restart):\n%s", stmt)
	}
	for _, want := range []string{
		"gs.connection_id = $1",
		"JOIN users u ON u.id = gs.user_id",
		"gs.started_at >= u.login_started_at",
		"ORDER BY gs.started_at DESC",
		"LIMIT 1",
	} {
		if !strings.Contains(stmt, want) {
			t.Errorf("expected sessionMemoryForConnectionSQL to contain %q:\n%s", want, stmt)
		}
	}
}
