---
phase: 04-continuous-play
plan: 02
subsystem: infra
tags: [go, config, postgres, migrations, encryption, credential-vault]

# Dependency graph
requires:
  - phase: 03.1-prompt-injection-review
    provides: the `blocked` outcome on ai_decisions and migration 012 that introduced it, plus the owner's carry-forward remediation instructions (DR-3.1-04, DR-3.1-05)
provides:
  - A startup gate in internal/config/config.go that refuses to start outside local development when ENCRYPTION_KEY_V1 is absent
  - A fixed rollback for migration 012 that reconciles blocked rows to failed before restoring the older three-value CHECK constraint
  - A no-database test proving the rollback statement ordering, plus a standing guard that every up migration has a matching down migration
  - ENCRYPTION_KEY_V1 documented as required outside local development, with a key-rotation procedure
affects: [04-continuous-play remaining plans, Phase 4 security review (D-28)]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Config fail-fast gate conditioned on RAILWAY_ENVIRONMENT, mirroring SessionSecret's required/fail-fast shape"
    - "No-database migration test: read the .sql file from disk, strip comment lines, assert statement ordering by byte offset"

key-files:
  created:
    - internal/store/migrations_test.go
  modified:
    - internal/config/config.go
    - internal/config/config_test.go
    - migrations/012_add_never_issue_and_blocked_outcome.down.sql
    - .specify/ENVIRONMENT-VARIABLES.md

key-decisions:
  - "The gate lives in config.Load() only; internal/crypto/crypto.go's DefaultKeyStore generated-key fallback is left unchanged so local development is unaffected (Claude's Discretion, plan 04-02)."
  - "RAILWAY_ENVIRONMENT != \"\" reused verbatim as the outside-local-development signal, the only such signal already in the codebase (internal/driver/corpus_live_test.go)."

patterns-established:
  - "Migration rollback tests read the .sql file from disk with no database connection, matching the project's evidence rule that a database query is never proof."

requirements-completed: [REQ-safety-limits-hold]

# Metrics
duration: ~20min
completed: 2026-09-17
---

# Phase 4 Plan 2: Credential-vault startup gate and migration 012 rollback fix Summary

**Server refuses to start outside local development without ENCRYPTION_KEY_V1, and migration 012's rollback reconciles blocked decision rows before restoring the older CHECK constraint — both proven by no-database/no-network tests.**

## Performance

- **Duration:** ~20 min
- **Completed:** 2026-09-17T04:18:43Z
- **Tasks:** 2 completed
- **Files modified:** 5 (4 modified, 1 created)

## Accomplishments

- `internal/config/config.go`'s `Load()` now returns `"ENCRYPTION_KEY_V1 environment variable is required outside local development"` and a nil config when `ENCRYPTION_KEY_V1` is empty and `RAILWAY_ENVIRONMENT` is set — closing D-25 (DR-3.1-04). On the owner's own machine (`RAILWAY_ENVIRONMENT` unset), nothing changes.
- `migrations/012_add_never_issue_and_blocked_outcome.down.sql` now runs `UPDATE ai_decisions SET outcome = 'failed' WHERE outcome = 'blocked';` before re-adding the three-value CHECK constraint, so the rollback no longer fails partway on a database with a blocked row — closing D-26 (DR-3.1-05).
- `internal/store/migrations_test.go` (new) proves the fix with two no-database tests: `TestMigration012DownReconcilesBlockedRows` (reads the file from disk, strips comments, asserts the `UPDATE` statement's byte offset precedes `ADD CONSTRAINT`'s, and that the restored CHECK still lists exactly `sent`, `refused`, `failed`) and `TestMigrationFilesPairUp` (every `*.up.sql` has a matching `*.down.sql`).
- `.specify/ENVIRONMENT-VARIABLES.md` marks `ENCRYPTION_KEY_V1` "Yes (outside local development)" and adds a "Rotating the credential vault key" section with a numbered procedure; no key value or key-shaped string was added (confirmed by grep).

## Task Commits

Each task was committed atomically:

1. **Task 04-02-01: The server will not start outside development without the vault key (DR-3.1-04)** - `2954960` (feat)
2. **Task 04-02-02: Migration 012 can be rolled back after a command has been blocked (DR-3.1-05)** - `d8ae8a3` (fix)

## Files Created/Modified

- `internal/config/config.go` - Added the D-25 gate immediately after the `EncryptionKeyV*` reads
- `internal/config/config_test.go` - Added `TestLoadRequiresEncryptionKeyOutsideDevelopment` with three sub-cases (staging without key fails; staging with key succeeds; local development unaffected)
- `migrations/012_add_never_issue_and_blocked_outcome.down.sql` - Inserted the one reconciliation `UPDATE` statement before the existing `DO $$` constraint-lookup block; nothing else in the file changed
- `internal/store/migrations_test.go` - New file: `TestMigration012DownReconcilesBlockedRows` and `TestMigrationFilesPairUp`
- `.specify/ENVIRONMENT-VARIABLES.md` - `ENCRYPTION_KEY_V1` row updated; new "Rotating the credential vault key" section added

## Decisions Made

- The startup gate error text names only the variable, never a key value, its length, or any other environment value, per the plan's binding instruction and D-25.
- No SQL was executed and no database was connected at any point in Task 2; the proof is the disk-read test, per the project's evidence rule (a database query is never evidence).

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## Confirmation on staging's key (per the plan's `<output>` requirement)

This plan does not deploy to or query Railway staging (machine_notes for this execution explicitly reserve staging changes for plan 04-11, owned by the orchestrator). STATE.md's Phase 3.1 history records that the staging vault key was found ephemeral during the Phase 3.1 walkthroughs and that "an operational (non-code) workaround let walkthrough 2 proceed" — i.e., a stable `ENCRYPTION_KEY_V1` was already set on staging by that point through an out-of-band (non-code) step, not by this plan. Whether that value is still set, and whether the new gate will pass cleanly on the phase's own staging deploy (plan 04-11), should be confirmed by the orchestrator immediately before that deploy step, since this plan's worktree has no visibility into Railway's current variable state.

## User Setup Required

None - no external service configuration required by this plan. If, at deploy time (plan 04-11), staging's `ENCRYPTION_KEY_V1` is found unset, the deploy will fail fast with this plan's new error rather than silently generating a throwaway key; the fix is to set a stable base64-encoded 32-byte value on the `ENCRYPTION_KEY_V1` Railway variable before deploying (see the new "Rotating the credential vault key" procedure in `.specify/ENVIRONMENT-VARIABLES.md` for the shape of the value, though rotation itself is not needed if none has ever been set).

## Next Phase Readiness

- D-25 and D-26 are ready to be recorded as closed at the Phase 4 security review under D-28.
- Both fixes are self-contained (config and one migration file) and touch no driver code, so they carry no dependency risk for the rest of Phase 4's wave 1 or later plans.
- Open item for the orchestrator: confirm staging's `ENCRYPTION_KEY_V1` is set before plan 04-11's deploy, since this plan's gate will otherwise stop that deploy from starting.

---
*Phase: 04-continuous-play*
*Completed: 2026-09-17*
