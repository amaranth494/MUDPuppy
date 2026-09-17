package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// stripSQLComments removes every line whose first non-whitespace characters
// are "--" (a SQL line comment), so a comment mentioning a keyword (for
// example this file's own D-26 explanation) can never satisfy an
// assertion meant to check the executable SQL.
func stripSQLComments(text string) string {
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

// TestMigration012DownReconcilesBlockedRows proves D-26 (DR-3.1-05): the
// rollback of migration 012 maps any 'blocked' row to 'failed' before the
// older three-value CHECK constraint is re-added, so the rollback does not
// fail partway on a database where a command has actually been blocked.
// This test opens no database connection; it reads the migration file from
// disk and checks statement ordering by byte offset.
func TestMigration012DownReconcilesBlockedRows(t *testing.T) {
	path := filepath.Join("..", "..", "migrations", "012_add_never_issue_and_blocked_outcome.down.sql")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("could not read %s: %v", path, err)
	}
	text := string(raw)
	stripped := stripSQLComments(text)

	const updateStatement = "UPDATE ai_decisions SET outcome = 'failed' WHERE outcome = 'blocked'"
	updateOffset := strings.Index(stripped, updateStatement)
	if updateOffset == -1 {
		t.Fatalf("expected the down migration to contain %q (comments stripped)", updateStatement)
	}

	const addConstraintNeedle = "ADD CONSTRAINT ai_decisions_outcome_check"
	addConstraintOffset := strings.Index(stripped, addConstraintNeedle)
	if addConstraintOffset == -1 {
		t.Fatalf("expected the down migration to contain %q", addConstraintNeedle)
	}

	if updateOffset >= addConstraintOffset {
		t.Errorf("expected the blocked-row reconciliation (offset %d) to appear before %q (offset %d)", updateOffset, addConstraintNeedle, addConstraintOffset)
	}

	const restoredCheck = "CHECK (outcome IN ('sent','refused','failed'))"
	if !strings.Contains(stripped, restoredCheck) {
		t.Errorf("expected the re-added CHECK to list exactly sent, refused, failed; got file contents:\n%s", stripped)
	}
}

// TestMigrationFilesPairUp is a cheap standing guard against the same class
// of defect as D-26: every up migration must have a matching down migration,
// so a future migration cannot ship without a rollback.
func TestMigrationFilesPairUp(t *testing.T) {
	dir := filepath.Join("..", "..", "migrations")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("could not read migrations directory %s: %v", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".up.sql") {
			continue
		}
		base := strings.TrimSuffix(name, ".up.sql")
		downName := base + ".down.sql"
		downPath := filepath.Join(dir, downName)
		if _, err := os.Stat(downPath); err != nil {
			t.Errorf("migration %s has no matching down migration %s", name, downName)
		}
	}
}
