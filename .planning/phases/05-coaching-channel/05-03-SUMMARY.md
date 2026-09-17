---
phase: 05-coaching-channel
plan: 03
subsystem: security
tags: [go, config, encryption-key, auth-guard, react-router]

# Dependency graph
requires:
  - phase: 04-continuous-play
    provides: the Railway-only vault-key gate (DR-3.1-04) and the AuthGuard-wrapped /logs/:connectionId route (AR-4-08) this plan closes
provides:
  - The vault-key requirement (ENCRYPTION_KEY_V1) holds on every host; the only opt-out is the explicit MUDPUPPY_LOCAL_DEV setting, announced at startup and never bypassing per-key validation
  - Confirmation, pinned in a source comment, that /logs/:connectionId already renders only inside AuthGuard
affects: [05-security-review, 05-11]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Config gate inversion: require-by-default with one named, announced opt-out env var, validated the same way regardless of the opt-out"

key-files:
  created: []
  modified:
    - internal/config/config.go
    - internal/config/config_test.go
    - frontend/src/App.tsx

key-decisions:
  - "MUDPUPPY_LOCAL_DEV accepts only '1' and 'true' (any letter case); every other value, including unset, is off — RAILWAY_ENVIRONMENT is no longer consulted in this decision at all"
  - "The opt-out skips the key requirement, never the per-key crypto.ParseKey validation: a malformed key present with the flag on is still a startup failure"
  - "No frontend guard was added — the log route was already inside AuthGuard as of this branch's HEAD; the task was to confirm this from the code and pin it with a comment"

patterns-established:
  - "A one-line startup log announces when the local-development opt-out is in effect, naming only the variable, never a value"

requirements-completed: [REQ-safety-limits-hold]

duration: 25min
completed: 2026-09-17
---

# Phase 5 Plan 03: Vault key everywhere and the log page's sign-in pin Summary

**The password-vault key is now required on every host (not just Railway), with one deliberately-named MUDPUPPY_LOCAL_DEV opt-out that announces itself at startup; the log page's existing AuthGuard placement is confirmed by reading the code and pinned with a comment naming AR-4-08.**

## Performance

- **Duration:** ~25 min
- **Tasks:** 2 completed
- **Files modified:** 3

## Accomplishments

- Closed DR-4-04 (D-26): `internal/config/config.go`'s vault-key gate no longer checks `RAILWAY_ENVIRONMENT`. A blank `ENCRYPTION_KEY_V1` now fails `Load()` on every host unless `MUDPUPPY_LOCAL_DEV` is `1` or `true` (case-insensitive); every other value, including unset, keeps the requirement in force. When the opt-out is on and the key is blank, the server logs one line naming `MUDPUPPY_LOCAL_DEV` and stating that saved MUD passwords will not survive a restart — no value is ever printed.
- The opt-out never weakens key *validation*: a key that is present but malformed (wrong length, not base64, quoted, trailing space, URL-safe encoding, or an unusable rotation key) still fails `Load()` even with `MUDPUPPY_LOCAL_DEV=true`, because the same `crypto.ParseKey` loop runs unconditionally on any non-blank key.
- Replaced `TestLoadRequiresEncryptionKeyOutsideDevelopment` with `TestLoadRequiresEncryptionKeyEverywhereUnlessLocalDev`, a table test covering: no key/no flag (fails), no key with `RAILWAY_ENVIRONMENT` set (still fails — the old escape route is gone), no key with `MUDPUPPY_LOCAL_DEV` = `true`/`TRUE`/`1` (succeeds), no key with `no`/`0`/empty (fails), a valid key with the flag off (succeeds), and every WR-07 malformed-key shape with the flag **on** (still fails, naming the variable, never the value). Added `TestLocalDevOptOutIsAnnounced`, which captures log output the way `internal/auth/handler_test.go` does and asserts the announcement line names `MUDPUPPY_LOCAL_DEV`.
- Confirmed from the code (not assumption) that a signed-out visitor to `/logs/<id>` never reaches `LogsPage`: `AppRoutes` wraps its entire `<Routes>` block — including `/logs/:connectionId` — inside `AuthGuard`, and `AuthGuard` renders `<LoginScreen />` in place of its children whenever `useSession()` reports no `user`. No route render happens before that check. Extended the existing block comment above the route with the required AR-4-08 pin: the route must stay inside `AuthGuard` because a log transcript is the owner's own game text, and moving it outside the guard is a security regression, not a refactor. No component code changed.

## Task Commits

1. **Task 05-03-01: Vault key required on every host** - `cf9a701` (fix)
2. **Task 05-03-02: Log page route pinned inside AuthGuard** - `fd82c77` (docs)

**Plan metadata:** (this commit, made by the orchestrator after all worktree agents in the wave complete)

## Files Created/Modified

- `internal/config/config.go` - Inverted the vault-key gate to require-everywhere-unless-`MUDPUPPY_LOCAL_DEV`; added `isLocalDevOptOut` helper; removed all `RAILWAY_ENVIRONMENT` references
- `internal/config/config_test.go` - Replaced the outside-development test with the everywhere-unless-local-dev table test; added the announcement test; added `MUDPUPPY_LOCAL_DEV=true` to four pre-existing AI-registry tests that don't concern the vault key so they keep passing under the stricter gate
- `frontend/src/App.tsx` - Comment-only change: extended the `/logs/:connectionId` route's existing block comment to name AR-4-08 and warn against moving the route outside `AuthGuard`

## What a signed-out visitor to `/logs/<id>` actually gets (from the code)

`AppRoutes()` renders `<AuthGuard><Routes>...<Route path="/logs/:connectionId" .../>...</Routes></AuthGuard>`. `AuthGuard` checks `useSession()`: while loading it shows a spinner; once loaded, if there is no `user` it returns `<LoginScreen />` and renders nothing else — the `<Routes>` block (and therefore `LogsPage`) never mounts. A signed-out visitor who opens `/logs/<connectionId>` is shown the login screen, exactly as any other route in this app. The guard was already present; nothing was added or changed in behavior, only the comment pinning it in place.

## Decisions Made

- `MUDPUPPY_LOCAL_DEV` accepted spellings are exactly `1` and `true` (case-insensitive); this was the planner's discretion, not something decided during execution.
- Kept the WR-07-style malformed-key sub-tests (not base64, wrong length, trailing space, quoted, URL-safe, bad rotation key) inside the new table test rather than dropping them, since they are real regression coverage for `crypto.ParseKey` integration and the plan asked for "at least" the listed rows.
- Four pre-existing tests (`TestLoadAIRegistry`, `TestLoadAIRegistryIgnoresIncompleteEntries`, `TestAIConfigured`, `TestResolveModelEntry`) did not set `ENCRYPTION_KEY_V1` or any local-dev flag and started failing under the stricter gate; fixed by adding `t.Setenv("MUDPUPPY_LOCAL_DEV", "true")` to each, since none of them are about the vault key.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Four pre-existing config tests broke under the stricter gate**
- **Found during:** Task 1, after writing the new gate and test
- **Issue:** `TestLoadAIRegistry`, `TestLoadAIRegistryIgnoresIncompleteEntries`, `TestAIConfigured`, and `TestResolveModelEntry` call `Load()` without setting `ENCRYPTION_KEY_V1`, and previously passed because the gate only fired when `RAILWAY_ENVIRONMENT` was set (which these tests never set). Under the new everywhere-unless-local-dev gate they failed with the vault-key error, blocking `go test ./internal/config/...` from passing.
- **Fix:** Added `t.Setenv("MUDPUPPY_LOCAL_DEV", "true")` to each of the four tests, since none of them concern the vault key.
- **Files modified:** internal/config/config_test.go
- **Verification:** `go test ./internal/config/... -count=1 -v` — all tests PASS
- **Committed in:** cf9a701 (Task 1 commit)

**2. [Rule 1 - Bug] Acceptance criterion required zero `RAILWAY_ENVIRONMENT` mentions, including in comments**
- **Found during:** Task 1, running the source-assertion checks
- **Issue:** `grep -c "RAILWAY_ENVIRONMENT" internal/config/config.go` returned 2 because the explanatory comment above the gate still named the old variable in prose, even though the code no longer read it.
- **Fix:** Reworded the comment to describe the old signal without using the literal string.
- **Files modified:** internal/config/config.go
- **Verification:** `grep -c "RAILWAY_ENVIRONMENT" internal/config/config.go` returns 0 (exit 1, no match)
- **Committed in:** cf9a701 (Task 1 commit)

**3. [Rule 1 - Bug] AR-4-08 mentioned twice, acceptance criterion wanted exactly one**
- **Found during:** Task 2, running the source-assertion checks
- **Issue:** The first draft of the extended comment named "AR-4-08" twice; the acceptance criterion checks `grep -c "AR-4-08" frontend/src/App.tsx` returns 1.
- **Fix:** Reworded to one sentence naming AR-4-08 once.
- **Files modified:** frontend/src/App.tsx
- **Verification:** `grep -c "AR-4-08" frontend/src/App.tsx` returns 1
- **Committed in:** fd82c77 (Task 2 commit)

---

**Total deviations:** 3 auto-fixed (1 blocking test breakage, 2 acceptance-criterion wording fixes)
**Impact on plan:** All three are necessary for the stated acceptance criteria and test suite to pass. No scope creep.

## Issues Encountered

`npm run build` in `frontend/` regenerated `public/assets/index-*.js`, `.js.map`, and `.css` (pre-existing build output committed to the repo). These were reverted with `git checkout -- <path>` per the machine rule not to commit rebuilt bundle output unless the plan says to; only the source change in `frontend/src/App.tsx` was committed.

## User Setup Required

None - no external service configuration required. Staging's `ENCRYPTION_KEY_V1` is already set (Phase 4), so this stricter gate does not change staging's startup behavior; `MUDPUPPY_LOCAL_DEV` is not set anywhere on staging and must not be.

## Owner sign-out statement

The owner was not signed out of any session during this plan, and no private-window or separate-browser-profile capture was attempted here — that screenshot (`evidence/13-logs-signed-out.png`) is filed by plan 05-11 per the plan's objective.

## Next Phase Readiness

- DR-4-04 is closed in code, ready to be re-presented with its outcome at the Phase 5 security review (D-28).
- AR-4-08's code-side answer is in place; the remaining evidence (the signed-out screenshot) is plan 05-11's to file.
- No blockers for downstream Phase 5 plans.

## Self-Check: PASSED

All created/modified files found on disk (internal/config/config.go, internal/config/config_test.go, frontend/src/App.tsx, this SUMMARY.md); all commit hashes (cf9a701, fd82c77, 2dda3f7) found in `git log --oneline --all`.

---
*Phase: 05-coaching-channel*
*Completed: 2026-09-17*
