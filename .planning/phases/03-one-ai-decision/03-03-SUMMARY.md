---
phase: 03-one-ai-decision
plan: 03
subsystem: api
tags: [go, icm, dispatcher, safety-checker, react, typescript, adapter]

# Dependency graph
requires:
  - phase: 02-autopilot-switch
    provides: ContextAutomation-aware session/autopilot plumbing that the driver (plan 03-08) will trigger
provides:
  - "A single icm.Engine instance constructed once in cmd/server/main.go, with its four /api/v1/icm/* routes live on the existing sessionMiddleware-wrapped mux"
  - "icm.NewHandlerWithEngine(engine *Engine) *Handler so the HTTP routes and the future AI driver share one Dispatcher and one SafetyChecker state"
  - "A diagnostic test (TestDispatch_AutomationPassThrough_GoesThroughSafetyChecker) proving Dispatcher.Dispatch, called directly with ContextAutomation, actually runs checkSafety for a plain game command with no registered handler - and proving Engine.Process() does not"
  - "frontend/src/services/icm-adapter.ts no longer silently recomputes an ICM answer in the browser when the server call fails; validateAndNormalize propagates the rejection and processCommand's catch reports an ICMError with shouldPassThrough: false"
affects: [03-08-driver-plan, 03-12-evidence-harness, 03-13-evidence-capture]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "internal/icm engine constructed once in main.go and threaded via GetDispatcher(), mirroring how sessionManager/profileStore are each constructed once"
    - "NewXWithEngine/NewXWithY constructor pattern for sharing a dependency instance instead of each caller building its own"

key-files:
  created:
    - internal/icm/dispatcher_test.go
    - .planning/phases/03-one-ai-decision/deferred-items.md
  modified:
    - internal/icm/handler.go
    - cmd/server/main.go
    - frontend/src/services/icm-adapter.ts

key-decisions:
  - "Kept NewHandler() as a thin wrapper around NewHandlerWithEngine(NewEngine()) rather than deleting it, so any existing caller or test that constructs a standalone Handler is unaffected."
  - "Did not commit the frontend build output (public/assets/*) produced by npm run build: the two changed functions (processCommand, validateAndNormalize) are not imported anywhere in the app, so Vite tree-shakes them out and the bundle is byte-identical; the only diff was sourcemap minifier non-determinism, which was reverted rather than committed since it is not in this plan's files_modified list and reflects no real change."
  - "Ran npm ci to materialize node_modules (absent in this fresh worktree) before running npm run build - this installs exactly the locked dependency tree from the existing package-lock.json, not a new package, so it is not subject to the package-manager-install exclusion in Rule 3."

patterns-established:
  - "Dispatcher.Dispatch is called directly with a hand-built NormalizedCommand for automation-context commands; Engine.Process() is never used for this path (RESEARCH.md Pattern 3 / Pitfall 1) - the diagnostic test in dispatcher_test.go exists specifically to keep this true."

requirements-completed: [REQ-single-decision]

# Metrics
duration: 6min (from first commit to last commit; reading/context time not included)
completed: 2026-09-15
---

# Phase 3 Plan 3: Wire the dormant ICM engine into the server Summary

**Constructed the ICM engine exactly once in `cmd/server/main.go`, registered its four `/api/v1/icm/*` routes, added a diagnostic test proving a plain automation-context command really runs `Dispatcher.Dispatch`'s safety checker (and that `Engine.Process()` does not), and removed the browser adapter's silent fallback-to-frontend-logic on a failed ICM call.**

## Performance

- **Duration:** ~6 min (commit-to-commit; excludes context-reading time)
- **Started:** 2026-09-15T18:52:08-07:00 (first task commit)
- **Completed:** 2026-09-15T18:57:37-07:00 (last task commit)
- **Tasks:** 3 completed
- **Files modified:** 4 (3 source files + 1 new deferred-items log)

## Accomplishments

- `internal/icm` — dormant since before this project started — now has exactly one caller in the running server: `cmd/server/main.go` constructs `icm.NewEngine()` once and registers its routes on the same mux every other `/api/v1` route uses.
- Wrote the one test that can tell apart "a driver built on `Dispatcher.Dispatch`" from "a driver built on `Engine.Process()`" — the exact distinction ROADMAP criterion 4 depends on, and the central finding of 03-RESEARCH.md.
- The frontend ICM adapter's two remote-call functions (`validateAndNormalize`, `processCommand`) now surface a failed server call as a caller-visible error/rejection instead of quietly recomputing an answer client-side.

## Task Commits

Each task was committed atomically:

1. **Task 03-03-01: Diagnostic test for the automation-context safety-checker path** - `8cf3dc9` (test)
2. **Task 03-03-02: One ICM engine, four live routes** - `2ed4dfd` (feat)
3. **Task 03-03-03: Browser adapter reports server failures instead of recomputing them** - `da934f1` (fix)

_No plan-metadata commit was made — the orchestrator owns STATE.md/ROADMAP.md updates after all wave worktrees complete, per this plan's parallel-execution instructions._

## Files Created/Modified

- `internal/icm/dispatcher_test.go` (new) - `TestDispatch_AutomationPassThrough_GoesThroughSafetyChecker` with three subtests: `approved_when_limits_are_clear`, `refused_when_rate_limit_tripped`, `process_does_not_reach_dispatch_for_a_plain_command`.
- `internal/icm/handler.go` - added `NewHandlerWithEngine(engine *Engine) *Handler`; `NewHandler()` now delegates to it.
- `cmd/server/main.go` - added the `internal/icm` import, constructed `icmEngine := icm.NewEngine()` and `icmHandler := icm.NewHandlerWithEngine(icmEngine)` alongside the other single-construction dependencies, and called `icmHandler.RegisterRoutes(mux)` in the route block next to the `/api/v1/session/*` registrations, before the `sessionMiddleware` wrap.
- `frontend/src/services/icm-adapter.ts` - `validateAndNormalize` no longer catches and swallows a failed `normalizeCommandRemote` call; `processCommand`'s trailing catch now returns `shouldPassThrough: false` with an `ICMError` carrying the failure message instead of `shouldPassThrough: true`.
- `.planning/phases/03-one-ai-decision/deferred-items.md` (new) - logs one pre-existing, out-of-scope test failure and the `-race`/cgo unavailability in this environment.

## Decisions Made

- Kept `NewHandler()` in place (delegating to `NewHandlerWithEngine`) rather than removing it, per the plan's explicit instruction to leave existing behavior/callers untouched.
- Left `recognizeCommand`, `validateCommand`, `normalizeCommand` completely untouched in `icm-adapter.ts` (verified via `git diff`, no changes inside those function bodies) since they are the only symbols imported by `PlayScreen.tsx`, `SettingsPage.tsx`, `automation.ts`, and `automation/evaluator.ts`.
- Did not commit the rebuilt `public/assets/*` output. `npm run build` was run twice (once per this plan's verification and once to re-confirm before summary) and both times the `.css`/`.js` bundle content was byte-identical to what is already committed; only the `.js.map` differed, purely from minifier non-determinism (confirmed by inspecting the diff: same length, scrambled short variable names, no semantic change). Grepping the built bundle for `E5000` (the new error code introduced in `processCommand`'s catch) found zero occurrences, confirming `processCommand`/`validateAndNormalize` are tree-shaken out of the shipped bundle because nothing currently imports them — exactly the condition the plan's own acceptance criteria rely on to call this change "provably safe for ordinary play." Committing the sourcemap diff would have added worktree-path-flavored build noise with no functional content, so it was reverted with `git checkout --`.

## Grep proof required by the plan's `<output>` instruction

Per the plan: "quoting the grep that proves nothing outside `icm-adapter.ts` imports the two changed functions":

```
$ grep -rn "processCommand\|validateAndNormalize" frontend/src --include=*.ts --include=*.tsx | grep -v "icm-adapter.ts"
frontend/src/services/automation.ts:1018:      this.processCommandQueue();
frontend/src/services/automation.ts:1027:  private async processCommandQueue(): Promise<void> {
```

Both matches are the unrelated `processCommandQueue` method name (substring match only) — no file outside `icm-adapter.ts` imports `processCommand` or `validateAndNormalize`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking issue] `node_modules` absent in this fresh worktree, blocking `npm run build`**
- **Found during:** Task 03-03-03 verification
- **Issue:** `npm run build` failed with `'tsc' is not recognized` because this worktree had never had `npm install`/`npm ci` run in it.
- **Fix:** Ran `npm ci`, which installs exactly the dependency tree already locked in the existing, unmodified `package-lock.json` — no new package was added or changed, so this is package-tree materialization, not a new install subject to the package-legitimacy exclusion in Rule 3.
- **Files affected:** none tracked (`node_modules/` is gitignored; `git status` confirmed no new untracked files after the install).
- **Verification:** `npm run build` then completed with exit 0.
- **Committed in:** N/A (no tracked files changed by this fix).

---

**Total deviations:** 1 auto-fixed (Rule 3, environment setup only)
**Impact on plan:** No scope creep; the fix only made the existing verification command runnable in this worktree.

## Issues Encountered

- `go test ./internal/icm/... -run TestDispatch_AutomationPassThrough -race -v` (the exact command in 03-VALIDATION.md) fails in this environment with `go: -race requires cgo; enable cgo by setting CGO_ENABLED=1` — no C compiler is present on this Windows/Git-Bash worktree. Verified the same test passes without `-race`: all three subtests PASS (`go test ./internal/icm/... -run TestDispatch_AutomationPassThrough -v`). Logged in `deferred-items.md` for the phase-level evidence capture step (plan 03-13), which may run on a host with cgo available.
- `go test ./internal/icm/...` (full package) shows one pre-existing failure, `TestHandlerRegistration/CANCEL`, unrelated to this plan's files (`internal/icm/dispatcher.go:170` comments out the `CANCEL` handler registration but the existing test still expects it registered). Confirmed pre-existing by temporarily removing this plan's new `dispatcher_test.go` and re-running the full package test — the failure persisted. Out of scope for `files_modified` (`internal/icm/dispatcher_test.go` only); logged in `deferred-items.md`, not fixed.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `icmEngine.GetDispatcher()` is available in `cmd/server/main.go`'s scope for plan 03-08's driver constructor.
- The diagnostic test in `internal/icm/dispatcher_test.go` is the artifact plan 03-13's evidence capture will run to produce `evidence/01-test-report.txt` for ROADMAP criterion 4.
- The ICM routes are live behind `sessionMiddleware`; plan 03-12's staging harness can hit `POST /api/v1/icm/validate` for `evidence/03-canned-report.txt`.
- No blockers for downstream plans in this wave (03-01, 03-02, 03-04) — this plan's `files_modified` list (`internal/icm/handler.go`, `internal/icm/dispatcher_test.go`, `cmd/server/main.go`, `frontend/src/services/icm-adapter.ts`) had no overlap with any sibling plan's declared files.

## Known Stubs

None. This plan wires existing, functioning infrastructure; it introduces no placeholder UI, hardcoded empty data, or "coming soon" text.

## Threat Flags

None beyond what the plan's own `<threat_model>` already covers (T-3-20, T-3-21, T-3-22, T-3-SC) — no new network endpoints, auth paths, or schema changes were introduced outside that register. `/api/v1/icm/*` becoming reachable and the adapter's fallback removal are exactly T-3-22 and T-3-21, both already mitigated per the threat model's disposition.

## Self-Check: PASSED

Files verified present: `internal/icm/dispatcher_test.go`, `internal/icm/handler.go`, `cmd/server/main.go`, `frontend/src/services/icm-adapter.ts`, `.planning/phases/03-one-ai-decision/deferred-items.md`, `.planning/phases/03-one-ai-decision/03-03-SUMMARY.md`.
Commits verified present in `git log`: `8cf3dc9`, `2ed4dfd`, `da934f1`.

---
*Phase: 03-one-ai-decision*
*Completed: 2026-09-15*
