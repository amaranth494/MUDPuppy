---
phase: 04-continuous-play
plan: 03
subsystem: api
tags: [go, goroutine, pacing, autopilot, driver, session-manager]

# Dependency graph
requires:
  - phase: 04-continuous-play
    provides: "04-01's scripted-queue fakeModels, real-state-machine fakeSessions (engageState/disengageState/enterWaiting/resume/setState/AutopilotStateFor/OutputSignal/fireOutput) and loop_test.go's TestScriptedStint bench"
provides:
  - "Manager.OutputSignal: a per-user buffered-size-1 wakeup channel, fired non-blockingly from appendOutputWindow when new game bytes land, coalescing a burst of output into one pending wakeup"
  - "Manager.DisengageHook/SetDisengageHook: the symmetric stop signal to EngageHook, fired with go from DisengageAutopilot's and parkAutopilotLocked's changed-true paths"
  - "Driver.EngageLoop/StopLoop/runLoop in internal/driver/loop.go: the continuous, paced, cancellable loop that makes one #AUTO ON produce a stream of decisions instead of Phase 3's single one"
  - "Driver.runIteration(userID, connectionID, first): HandleEngage's unchanged body renamed and reused as the loop's per-iteration body; HandleEngage is now a thin first=false wrapper"
  - "reassessInstruction(): the fixed D-08 paragraph appended to the system instruction only on a stint's first iteration"
  - "Both real engage-hook wiring sites (cmd/server/main.go) retargeted from aiDriver.HandleEngage to aiDriver.EngageLoop; sessionManager.SetDisengageHook(aiDriver.StopLoop) newly wired"
  - "DR-3-07 closed: the dead `_ = curConnID` statement in EngageAutopilot is deleted"
affects: [04-continuous-play plans 04-04 through 04-11, especially 04-04 (call cap/error/block counters, which reset at the marked spot inside EngageLoop) and 04-11 (evidence capture: TestLoop_Pacing/TestLoop_StopsWhenWheelGrabbed/TestManager_DisengageHookFires PASS lines, and the staging walkthrough screenshots 07/09)]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Pacing durations (settleDelay, floorInterval, minSpacing) are Driver struct fields initialised by New, not package constants, so tests inject millisecond values with no fake-clock library"
    - "Per-user output-arrived signal: a lazily-created, buffered-size-1 chan struct{} with a non-blocking select-default send, mirroring outputWindow's own lazy-creation pattern"
    - "Symmetric hook pair (EngageHook/DisengageHook) fired with go from inside a held mutex, matching resumeAutopilotLocked's existing go m.engageHook(...) discipline exactly"
    - "Loop lifecycle: map[string]context.CancelFunc guarded by the same mutex that already guards inFlight; starting a new loop always cancels-and-replaces any existing one first"
    - "Cancellable wait via select{ case <-ctx.Done(): ...; case <-time.After(d): ... } used for both the post-signal settle wait and the minimum-spacing wait, so a wheel-grab or disconnect stops the loop mid-wait rather than only being noticed after the next decision completes"
    - "Test-only negative assertions (\"nothing further happened\") use a bounded polling helper (assertStableCallCount) defined in driver_test.go rather than a blind sleep, keeping loop_test.go itself free of any literal time.Sleep per its own acceptance criterion"

key-files:
  created:
    - internal/driver/loop.go
  modified:
    - internal/session/manager.go
    - internal/session/manager_test.go
    - internal/driver/driver.go
    - internal/driver/driver_test.go
    - internal/driver/loop_test.go
    - internal/driver/corpus_live_test.go
    - cmd/server/main.go

key-decisions:
  - "Pacing values shipped exactly as the plan's Claude's Discretion starting points: settleDelay 1500ms, floorInterval 20s, minSpacing 3s -- untouched, pending tuning against Alter Aeon during the staging walkthrough."
  - "EngageLoop records lastDecisionAt for its own synchronous first iteration (not only for later loop ticks), so D-06's minimum spacing applies from the very first decision of a stint onward, not just between the second and later ones -- a small addition beyond the plan's literal wording, made because 'decisions are never closer than a minimum spacing' (D-06) reads as unconditional."
  - "buildSystemInstruction's signature is unchanged; runIteration appends reassessInstruction() to its output with a blank-line separator only when first is true, rather than threading a first bool into buildSystemInstruction itself, keeping every existing TestBuildSystemInstruction case untouched."
  - "The counter-reset comment plan 04-04 needs sits as EngageLoop's very first statement, above the call to runIteration -- see internal/driver/loop.go's opening comment inside EngageLoop."
  - "-race ran clean on this machine for both internal/driver and internal/session (no race-detector-unavailable note needed)."
  - "DR-3-07's closure (the dead curConnID statement) is recorded here as done in code; per 04-CONTEXT.md D-28, its formal closure entry in RISK-REGISTER.md is written at the Phase 4 security review, not by this plan."

patterns-established:
  - "Any test proving a negative ('the loop did not do X') on a millisecond-paced goroutine polls for a stable count over a bounded window (assertStableCallCount/waitForStableCallCount) rather than sleeping once and checking -- keeps the assertion itself un-flaky and keeps loop_test.go free of literal time.Sleep."
  - "A test that must block one specific model call without blocking an earlier synchronous one sets fakeModels.block only after waitForCalls confirms the earlier call already returned (fakeModels.setBlock), rather than setting block before New."

requirements-completed: [REQ-continuous-loop, REQ-doc-wheel-grab-and-reengage, REQ-reengage-reassess]

# Metrics
duration: 13min
completed: 2026-09-16
---

# Phase 4 Plan 03: The loop, the stop signal, and the reassess instruction Summary

**One `#AUTO ON` now produces a stream of paced decisions instead of Phase 3's single one, stopped instantly by a symmetric `DisengageHook` even mid-sleep, with every re-engage reassessing the current situation from a fresh window.**

## Performance

- **Duration:** 13 min (4c8ef84 base to final task commit 3abba50)
- **Started:** 2026-09-16T21:23:50-07:00
- **Completed:** 2026-09-16T21:37:03-07:00
- **Tasks:** 3 completed
- **Files modified:** 8 (1 created, 7 modified)

## Accomplishments
- The session manager can say "new output arrived" (`Manager.OutputSignal`, a per-user coalescing wakeup channel fired from `appendOutputWindow`) and "stop now" (`Manager.DisengageHook`/`SetDisengageHook`, fired from `DisengageAutopilot`'s and `parkAutopilotLocked`'s changed-true paths) -- proven by `TestManager_OutputSignalFires`, `TestManager_DisengageHookFires` and `TestManager_EngageHookStillFiresOnResume`.
- `internal/driver/loop.go` gives the driver a continuous, cancellable, paced loop (`EngageLoop`/`startLoop`/`runLoop`/`StopLoop`): decisions follow new game output after a settle wait, a floor interval nudges a quiet game, and a minimum spacing bounds the rate regardless of how many output signals arrive -- proven by `TestLoop_Pacing`'s three sub-cases, repeatable under `-count=5` with no flake.
- A wheel-grab or a disconnect stops the loop at once, whether it is asleep in its pacing wait or waiting on a model call already in flight -- proven by `TestLoop_StopsWhenWheelGrabbed` (both cases) and `TestManager_DisengageHookFires`.
- Every engage (a fresh `#AUTO ON` or a resume) starts with a fresh window snapshot and an explicit reassess instruction on its first decision only, with nothing from an earlier stint's answer leaking into the next prompt -- proven by `TestEngageLoop_Reassess` and by `TestLoop_NothingIssuedWhileWaiting`'s post-resume case.
- D-27/DR-3-07's dead `_ = curConnID` statement in `EngageAutopilot` is gone, `TestEngageAutopilot` unchanged and still green.
- `TestLoop_NoAIReconnect` gives DEC-reconnect-is-connection-toggle a mechanical, not narrative, guard: a counter on the session double that would catch a future interface change reintroducing a reconnect-shaped method.

## Task Commits

Each task was committed atomically:

1. **Task 04-03-01: The session manager can say "new output arrived" and "stop now" (D-27, DR-3-07)** - `4d85687` (feat)
2. **Task 04-03-02: The AI keeps playing until something stops it, paced to the game's output** - `3a8986a` (feat)
3. **Task 04-03-03: The pacing, the reassessment and the stopping are proven under test** - `3abba50` (test)

**Plan metadata:** (this commit, made by the orchestrator after all worktree agents in the wave complete)

## Files Created/Modified
- `internal/session/manager.go` - `OutputSignal`/lazy-created `outputSignals` map fired from `appendOutputWindow`; `DisengageHook`/`SetDisengageHook`, fired with `go` from `DisengageAutopilot` and `parkAutopilotLocked`'s changed-true paths; the dead `_ = curConnID` statement removed
- `internal/session/manager_test.go` - `TestManager_OutputSignalFires`, `TestManager_DisengageHookFires`, `TestManager_EngageHookStillFiresOnResume`, plus a shared `pollUntil` deadline-polling helper
- `internal/driver/driver.go` - `Sessions` widened with `OutputSignal`; `Driver` gains `loops`/`lastDecisionAt` maps and `settleDelay`/`floorInterval`/`minSpacing` fields (1.5s/20s/3s); `HandleEngage` becomes a thin wrapper over the new unexported `runIteration(userID, connectionID, first)`; `reassessInstruction()` added and appended to the system instruction only when `first` is true
- `internal/driver/loop.go` - New file: `EngageLoop`, `startLoop`, `runLoop`, `StopLoop`, and the shared `cancellableWait` helper -- the loop's lifecycle and pacing only, per its own file-level comment
- `internal/driver/driver_test.go` - `fakeSessions` gains `sentAt`/`reconnectCalls` fields and `sentAtSnapshot`/`reconnectCallCount` accessors; `fakeModels` gains `setBlock`; new `assertStableCallCount`/`waitForStableCallCount`/`waitForSendCount` polling helpers
- `internal/driver/loop_test.go` - `TestLoop_Pacing` (three sub-cases), `TestEngageLoop_Reassess`, `TestLoop_StopsWhenWheelGrabbed` (two cases), `TestLoop_NothingIssuedWhileWaiting`, `TestLoop_NoAIReconnect`, and the `newPacedDriver` test-setup helper
- `internal/driver/corpus_live_test.go` - `corpusSessions` gains a minimal `OutputSignal` to keep the live red-team harness building against the widened `Sessions` interface (Rule 3, following the 04-01 precedent for `AutopilotStateFor`)
- `cmd/server/main.go` - Both `SetEngageHook` call sites retargeted from `aiDriver.HandleEngage` to `aiDriver.EngageLoop`; `sessionManager.SetDisengageHook(aiDriver.StopLoop)` newly wired; the surrounding comment updated to describe the loop's start/stop triggers

## Decisions Made
- Pacing constants shipped exactly at the plan's suggested starting points (1500ms settle, 20s floor, 3s minimum spacing); no tuning performed here, as instructed -- that happens against Alter Aeon during the staging walkthrough (plan 04-11).
- `EngageLoop` records `lastDecisionAt` for its own synchronous first iteration too, not only for loop ticks, so the minimum spacing is never bypassed for the second decision of a stint landing unusually soon after the first. This is a small addition beyond the plan's literal task-2 wording (which only names `runLoop` as the enforcement site), justified by D-06's own unconditional phrasing ("decisions are never closer than a minimum spacing") and verified by `TestLoop_Pacing`'s minimum-spacing sub-case.
- `reassessInstruction()` is appended as a separate string after `buildSystemInstruction(profile)` rather than threading a `first` parameter into `buildSystemInstruction` itself, so every pre-existing `TestBuildSystemInstruction` case needed zero changes.
- The plan 04-04 counter-reset marker sits as `EngageLoop`'s very first statement (a comment above the call to `runIteration`), matching the plan's instruction to "leave a one-line comment marking that spot."
- `-race` availability: `go test ./internal/driver/... ./internal/session/... -race -count=1` ran clean on this machine (2.5s / 1.3s respectively); no race-detector-unavailable note is needed.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] corpus_live_test.go's separate Sessions double needed OutputSignal too**
- **Found during:** Task 04-03-02, `go build ./...` verification
- **Issue:** `corpus_live_test.go` defines its own file-local `corpusSessions` double for the live red-team harness, which also implements `driver.Sessions`. Widening the interface with `OutputSignal` broke the build because `corpusSessions` had no such method -- the same shape of gap 04-01 hit for `AutopilotStateFor`.
- **Fix:** Added a minimal `func (c *corpusSessions) OutputSignal(userID string) <-chan struct{} { return make(chan struct{}) }`. The corpus harness calls `HandleEngage` directly and never the loop, so a fresh never-fired channel changes nothing about what that harness measures.
- **Files modified:** `internal/driver/corpus_live_test.go`
- **Verification:** `go build ./...` exits 0; `go vet ./internal/driver/...` clean; full driver suite (including this file's own tests, network-gated and not run live here) compiles as one package.
- **Committed in:** `3a8986a` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Necessary to keep the build green after the interface widening the plan itself specifies; no scope creep, no behaviour change to the corpus harness's own red-team logic.

## Issues Encountered

None beyond the deviation above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `internal/driver/loop.go`'s `EngageLoop`/`StopLoop`/`runLoop` and the `loops`/`lastDecisionAt` maps are ready for plan 04-04 to add call-cap, error-threshold and consecutive-block counters and their reset at the marked spot inside `EngageLoop`.
- `Manager.OutputSignal` and `DisengageHook` are wired end to end in `cmd/server/main.go`; no further wiring is needed for those two collaborators.
- DR-3-07's code-level closure is done; per 04-CONTEXT.md D-28, its entry in `.planning/RISK-REGISTER.md` is written at the Phase 4 security review, not by this plan -- flagged here so that review step does not overlook it.
- No blockers. `go build ./...`, `go vet ./internal/driver/... ./internal/session/... ./cmd/...`, `go test ./internal/... -count=1`, `go test ./internal/driver/... -run "TestLoop_|TestEngageLoop_" -count=1 -v`, `go test ./internal/driver/... -count=5 -run "TestLoop_Pacing"`, and `go test ./internal/driver/... ./internal/session/... -race -count=1` all pass on this machine; `git diff --stat go.mod frontend/package.json` is empty (T-4-SC).

---
*Phase: 04-continuous-play*
*Completed: 2026-09-16*

## Self-Check: PASSED

- FOUND: internal/driver/loop.go
- FOUND: .planning/phases/04-continuous-play/04-03-SUMMARY.md
- FOUND: commit 4d85687 (Task 04-03-01)
- FOUND: commit 3a8986a (Task 04-03-02)
- FOUND: commit 3abba50 (Task 04-03-03)
