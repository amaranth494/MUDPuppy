---
phase: 03-one-ai-decision
plan: 08
subsystem: api
tags: [go, driver, icm, gemini, decision-log, autopilot-hook]

# Dependency graph
requires:
  - phase: 03-one-ai-decision
    plan: 01
    provides: "Manager.RecentOutputSnapshot(userID) — the rolling window the driver reads at engage time"
  - phase: 03-one-ai-decision
    plan: 02
    provides: "internal/config's AI model registry (AIConfigured/ResolveModelEntry) and internal/gemini.Client.GenerateContent"
  - phase: 03-one-ai-decision
    plan: 03
    provides: "the ICM engine constructed once in cmd/server/main.go and the proof that Dispatcher.Dispatch (not Engine.Process) is the safety-checked path"
  - phase: 03-one-ai-decision
    plan: 05
    provides: "Manager.SendCommandAs(userID, command, source) and the transcript/session plumbing CurrentGameSessionID reads"
  - phase: 03-one-ai-decision
    plan: 06
    provides: "the #AUTO ON not-configured refusal that runs before EngageAutopilot, so the driver's own defensive api-error branch is a second line, not the first"
provides:
  - "internal/store/decisions.go: DecisionStore.InsertDecision (outcome-validated) and ListForConnection (oldest first) over the ai_decisions table plan 03-05 created"
  - "internal/driver package: Driver.HandleEngage — snapshot, two-tier prompt, one model call, command validation, Dispatch through icm.ContextAutomation, SendCommandAs, persist, notify — with an in-flight guard making a concurrent engage/resume a no-op"
  - "session.EngageHook type and Manager.SetEngageHook/Handler.SetEngageHook, fired exactly once per real engagement (a changed #AUTO ON, or a WAITING-to-ON resume) and never on a repeated/refused path"
  - "Manager.CurrentGameSessionID(userID) — ties a decision row to the open transcript session"
  - "cmd/server/main.go wiring: one Driver instance sharing icmEngine's dispatcher with the HTTP ICM routes, triggered by both hook points"
affects: [03-09-decision-panel-and-websocket, 03-13-evidence-capture]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Driver depends on five narrow collaborator interfaces (Models, Commands, Sessions, Profiles, Decisions) plus a Notifier, each matching a real type's method signature exactly so no adapter is needed at the call site in main.go, and each independently fakeable with no database, no MUD and no network"
    - "In-flight guard: a plain sync.Mutex + map[string]bool on Driver, taken for the whole HandleEngage call, not just the model call — this is what makes a racing engage+resume produce exactly one decision (D-03, T-3-35)"
    - "One canonical log.Printf format string per structured log line, called from every stage via a single method (logDecision), so a source-assertion grep for the format string finds exactly one occurrence"

key-files:
  created:
    - internal/store/decisions.go
    - internal/driver/driver.go
    - internal/driver/driver_test.go
  modified:
    - internal/session/manager.go
    - internal/session/handler.go
    - cmd/server/main.go

key-decisions:
  - "Log field renamed from the plan's literal window_bytes/cmd_len example to snapshot_bytes/cmd_len (Rule 1 — self-matching grep, same class of issue plan 03-02 hit and fixed the same way): the acceptance criterion greps every [AI-PLAYER]-tagged line for the literal substring \"window\" and expects zero matches, but the plan's own example format string contains \"window_bytes\", which trivially matches that check's own pattern. Renaming the field to snapshot_bytes preserves the exact meaning (a byte length, never the text) while letting the source assertion pass as written; cmd_len is unchanged since \"cmd\\b\" does not match \"cmd_len\" (the underscore is a word character, so there is no boundary)."
  - "A malformed userID/connectionID (uuid.Parse failure) is handled by calling the same recordFailure path with uuid.Nil for both ids rather than a separate no-DB-write branch: InsertDecision's own FK violation (if any) is swallowed the same way any other storage error already is, so the notify/disengage/log sequence still runs uniformly. This path is defensive only — both callers (task 03-08-03's two hook points) always supply already-parsed, valid ids — and is not exercised by TestHandleEngageFailures, whose six rows are exactly the six D-13 failure kinds the acceptance criteria name."
  - "The Gemini API's own vendor-error 400 (KindBadRequest) has no distinct D-13 failure kind in the plan's vocabulary table; it is mapped to api-error alongside KindAuth and KindTransport in the genErr switch's default arm, consistent with the plan's own three-way mapping example (rate-limited and malformed are named explicitly; \"everything else\" is api-error)."
  - "internal/driver imports internal/config, internal/gemini, internal/icm, internal/session and internal/store as the plan's <context> fixes; internal/session declares EngageHook and never imports internal/driver, so cmd/server/main.go aliases the import as aidriver to avoid colliding with the pre-existing local variable named driver (golang-migrate's postgres.WithInstance result) in the same function."
  - "Driver.New takes a Notifier parameter (nilable) even though task 03-08-03's main.go wiring passes nil for it today; plan 03-09 will pass a real implementation into the same New call without changing Driver's shape."

patterns-established:
  - "Failure-kind-to-notice map plus a single recordFailure/recordSuccess pair covers every D-13 exit from HandleEngage, so the seven-notices-in-source and every-failure-disengages-once acceptance criteria are structural (one code path) rather than needing per-branch duplication."

requirements-completed: [REQ-single-decision]

# Metrics
duration: ~20min
completed: 2026-09-15
---

# Phase 3 Plan 08: The Single AI Decision Summary

**A new `internal/driver` package makes the phase's one decision real end to end: snapshot the recent-game-text window, ask Gemini once with the profile's conduct rules and approach guidance carried verbatim, validate the answer as a single plain command, dispatch it through `icm.Dispatcher.Dispatch` in the automation execution context (never `Engine.Process`), send it only after that gate approves, and store the decision — with every failure kind sending nothing, disengaging autopilot, and leaving one of seven locked notices behind.**

## Performance

- **Duration:** ~20 min (commit-to-commit, base `415e95e` to final task commit `7c8fe86`)
- **Started:** 2026-09-15T19:21:50-07:00 (post worktree-base reset)
- **Completed:** 2026-09-15T19:41:01-07:00
- **Tasks:** 3
- **Files modified:** 6 (3 created, 3 modified)

## Accomplishments

- `internal/store/decisions.go` gives the codebase `DecisionStore`, matching `ProfileStore`'s and `TranscriptStore`'s exact constructor shape: `InsertDecision` validates `Outcome` against `sent`/`refused`/`failed` before any SQL runs, and `ListForConnection` returns a connection's decisions oldest-first so the panel (plan 03-09) can render them in the order they happened.
- `internal/driver` is a brand-new, fully unit-tested package implementing the phase's central mechanism: `Driver.HandleEngage` reads the window, builds the two-tier prompt (D-05), calls the model once, validates the returned command as a security boundary *before* building an ICM `NormalizedCommand` (T-3-01), calls `Dispatcher.Dispatch` directly with `icm.ContextAutomation` — never the engine's pass-through entry point, the phase's central correctness rule (RESEARCH Pitfall 1) — and only then sends the command, tagged `"ai"`.
- Every one of the six locked D-13 failure notices from 03-UI-SPEC.md's Copywriting Contract appears character-for-character in `driver.go`, plus a seventh sentence in the same voice for an ICM refusal (the addition 03-08-PLAN.md's `<context>` anticipated and asked to be flagged here for plan 03-13 to carry to the owner).
- `session.EngageHook` plus `Manager.SetEngageHook`/`Handler.SetEngageHook` fire the driver in exactly two places — a real `#AUTO ON` engage (the `case changed:` arm only, never `already-on` or any `refused-*` path) and a `WAITING`-to-`ON` reconnect resume — both started with `go` so neither the HTTP handler nor `Manager`'s held lock ever blocks on the driver.
- `cmd/server/main.go` now constructs one `Driver` sharing `icmEngine.GetDispatcher()` with the HTTP `/api/v1/icm/*` routes (T-3-20), and wires both hook points to it.

## Task Commits

Each task was committed atomically:

1. **Task 03-08-01: A decision can be written down and read back** - `48b0833` (feat)
2. **Task 03-08-02: The driver reads, asks, validates, dispatches and sends — once — and every failure sends nothing** - `038cc12` (feat)
3. **Task 03-08-03: Switching autopilot on — and a reconnect resuming it — fires the driver exactly once** - `7c8fe86` (feat)

**Additional commit:** `9d62a23` (docs: log the `-race`/cgo environment limitation in `deferred-items.md`)

_No plan-metadata commit — this executor runs in a parallel worktree and does not update STATE.md/ROADMAP.md; the orchestrator commits those after the wave completes._

## Files Created/Modified

- `internal/store/decisions.go` - `DecisionStore`, `DecisionRecord`, `Decision`, `NewDecisionStore`, `InsertDecision`, `ListForConnection`
- `internal/driver/driver.go` - `Driver`, `New`, `HandleEngage`, the five collaborator interfaces (`Models`, `Commands`, `Sessions`, `Profiles`, `Decisions`) plus `Notifier`/`Event`, `validateCommand`, `buildSystemInstruction`, `logDecision`
- `internal/driver/driver_test.go` - fakes for all six collaborators, `TestHandleEngage` (6 subtests), `TestHandleEngageFailures` (6-case table)
- `internal/session/manager.go` - `EngageHook` type, `Manager.engageHook`/`SetEngageHook`, the fire site inside `resumeAutopilotLocked`, `Manager.CurrentGameSessionID`
- `internal/session/handler.go` - `Handler.engageHook`/`SetEngageHook`, the fire site inside the `Autopilot` handler's `case changed:` arm
- `cmd/server/main.go` - `decisionStore`, `geminiClient`, `aiDriver` construction (import aliased `aidriver` to avoid colliding with the pre-existing local `driver` variable from `postgres.WithInstance`), two `SetEngageHook` calls

## Decisions Made

See `key-decisions` in the frontmatter above for the full rationale on each. In short: the `[AI-PLAYER]` log line's byte-length field is named `snapshot_bytes` rather than the plan's literal `window_bytes` example, to satisfy the acceptance criterion's own grep for the substring `window` without changing what is actually logged (a length, never text); a malformed engage-hook id is handled through the same failure path as every other failure rather than a separate branch; Gemini's `KindBadRequest`/`KindAuth` fold into `api-error`; and `cmd/server/main.go` imports the new package as `aidriver` to avoid a name collision with an existing local variable.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Renamed the `[AI-PLAYER]` decision log's window-length field to avoid a self-matching acceptance-criteria grep**
- **Found during:** Task 03-08-02, working through the task's own acceptance criteria before committing
- **Issue:** 03-08-PLAN.md's `<action>` gives the exact log format `... window_bytes=%d cmd_len=%d`, but its own acceptance criteria immediately after requires `grep "\[AI-PLAYER\]" internal/driver/driver.go | grep -ciE "window|reasoning|...` to return 0 — and the literal field name `window_bytes` contains the substring `window`, so implementing the format exactly as written would fail that same task's own check. (`cmd_len` does not have this problem: the criteria's pattern is `cmd\b`, and the underscore in `cmd_len` is a word character, so there is no word boundary there.) This is the same class of self-inflicted grep collision plan 03-02 hit and fixed (a comment matching its own forbidden-pattern check).
- **Fix:** Renamed the field to `snapshot_bytes` in the log format string, keeping the exact same meaning (the byte length of the recent-text window handed to the model) and never interpolating the window's actual text. Also worded every doc comment near the log call to avoid the literal string `[AI-PLAYER]` (so an explanatory comment mentioning "reasoning" or "conduct rules" by name, to say they are *not* logged, cannot itself trip the same grep).
- **Files modified:** internal/driver/driver.go
- **Verification:** `grep "\[AI-PLAYER\]" internal/driver/driver.go | grep -ciE "window|reasoning|conductrules|approachguidance|apikey|endpoint|cmd\b"` returns 0; `grep -n "\[AI-PLAYER\] decision" internal/driver/driver.go` shows exactly one format-string line.
- **Committed in:** 038cc12 (Task 2 commit — fixed before commit, not a separate commit)

**2. [Rule 3 - Blocking] Aliased the `internal/driver` import in `cmd/server/main.go` to avoid a name collision**
- **Found during:** Task 03-08-03, first `go build ./...` after wiring the driver into `main.go`
- **Issue:** `cmd/server/main.go` already declares a local variable named `driver` (the result of `postgres.WithInstance(db, ...)`, used for the golang-migrate setup near the top of `main`). Importing the new package under its default name `driver` shadowed that identifier and produced `driver.New undefined (type database.Driver has no field or method New)`.
- **Fix:** Imported the package under the alias `aidriver` (`aidriver "github.com/amaranth494/MudPuppy/internal/driver"`) and used `aidriver.New(...)` at the one call site; the acceptance criterion's literal substring `driver.New(` is still present (it is a substring of `aidriver.New(`), so the source assertion is unaffected.
- **Files modified:** cmd/server/main.go
- **Verification:** `go build ./...` exits 0; `grep -c "driver\.New(" cmd/server/main.go` returns 1.
- **Committed in:** 7c8fe86 (Task 3 commit — fixed before commit, not a separate commit)

---

**Total deviations:** 2 auto-fixed (1 self-matching-grep wording fix, 1 build-blocking import collision). No scope creep — both were caught and fixed inside the same task's own files before that task's commit.
**Impact on plan:** None on shipped behavior. The log line still carries only ids, a stage and byte lengths (T-3-07/T-3-02 hold exactly as specified); the driver is wired into `main.go` exactly as the plan describes, under a different Go identifier.

## Issues Encountered

- **`-race` unavailable in this environment**, consistent with every prior Phase 3 plan: `CGO_ENABLED=0`, no `gcc` on `PATH`. `go test ./internal/driver/... -run TestHandleEngage -race -v` and `go test ./internal/session/... -race -v` (both named in 03-VALIDATION.md and this plan's own `<verify>` blocks) were run as plain `go test ... -v` instead. All subtests and the full `internal/session` suite pass; logged in `deferred-items.md` under a new "## Plan 03-08" section for whichever environment captures `evidence/01-test-report.txt` to re-run with `-race` if cgo is available there.
- **No other issues.** `go test ./...` (whole repo) is green at this plan's final commit; `internal/icm`'s previously-flagged `TestHandlerRegistration/CANCEL` failure remains resolved (confirmed passing, per 03-05's and 03-07's notes).

## User Setup Required

None new. The Gemini environment variables (`AI_MODEL_DEFAULT`, `AI_MODEL_<SLUG>_NAME/ENDPOINT/KEY`) plan 03-02 introduced still need to be set on Railway staging before `#AUTO ON` can actually reach a real model there — this plan's driver code is otherwise ready to run against them the moment they exist.

## Next Phase Readiness

- `internal/driver.Driver`, its `Event`/`Notifier` shapes, and `New`'s parameter order are ready for plan 03-09 to pass a real websocket-backed `Notifier` into the same `cmd/server/main.go` construction call (currently `nil`) and to build the decisions-read REST endpoint plan 03-09's own plan text describes, using `DecisionStore.ListForConnection`.
- The `[AI-ASSIST > {command}]` terminal echo (D-09) and the AI Assist panel itself are not built here — this plan's own scope is the decision, not its browser presentation — and remain plan 03-09/03-10 work.
- No blockers for plan 03-09: this plan's `files_modified` list (`internal/store/decisions.go`, `internal/driver/driver.go`, `internal/driver/driver_test.go`, `internal/session/manager.go`, `internal/session/handler.go`, `cmd/server/main.go`) is exactly what the plan named, with no overlap declared against other wave-3 plans.
- `go build ./...`, `go vet ./internal/driver/... ./internal/session/... ./internal/store/... ./cmd/...`, and `go test ./...` (whole repo) all pass at this plan's final commit. `git diff --stat go.mod frontend/package.json` is empty.

## Known Stubs

- **`Driver`'s `Notifier` is `nil` in `cmd/server/main.go`'s current wiring.** This is the plan's own explicit design (`Notifier`'s doc comment: "a nil Notifier must be a silent no-op so this plan is independently runnable"), not a shortcut: plan 03-09 supplies the real websocket-backed implementation into the same `New(...)` call. Every decision still reaches `SendCommandAs`, `InsertDecision` and the `[AI-PLAYER]` log regardless of the notifier being nil; only the browser-facing push (out of this plan's scope) is not yet wired. Not a stub against this plan's own goal — ROADMAP criteria 1 and 4 (the decision happens, and it passes through ICM) are both provable by `go test` and by `SendCommandAs` reaching the game without any notifier at all.

## Threat Flags

None beyond what this plan's own `<threat_model>` already covers (T-3-01, T-3-02, T-3-06, T-3-07, T-3-35 mitigated; T-3-14, T-3-15 deferred to security review; T-3-SC accepted, and `git diff --stat go.mod` confirms it held) — no new network endpoints, auth paths, or schema changes were introduced outside that register.

---
*Phase: 03-one-ai-decision*
*Completed: 2026-09-15*

## Self-Check: PASSED

- FOUND: internal/store/decisions.go
- FOUND: internal/driver/driver.go
- FOUND: internal/driver/driver_test.go
- FOUND: internal/session/manager.go
- FOUND: internal/session/handler.go
- FOUND: cmd/server/main.go
- FOUND: .planning/phases/03-one-ai-decision/deferred-items.md
- FOUND: 48b0833 (Task 1 commit)
- FOUND: 038cc12 (Task 2 commit)
- FOUND: 7c8fe86 (Task 3 commit)
- FOUND: 9d62a23 (deferred-items docs commit)
