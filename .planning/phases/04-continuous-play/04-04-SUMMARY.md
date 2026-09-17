---
phase: 04-continuous-play
plan: 04
subsystem: api
tags: [go, driver, safety-limits, websocket, react, typescript]

# Dependency graph
requires:
  - phase: 04-continuous-play
    provides: "04-01's scripted-queue fakeModels and real-state-machine fakeSessions (engageState/disengageState/enterWaiting/resume/setState/AutopilotStateFor/OutputSignal/fireOutput); 04-03's EngageLoop/StopLoop/runLoop continuous loop and the counter-reset marker it left inside EngageLoop"
provides:
  - "The split transient/non-transient failure taxonomy (transientFailureKinds), four new non-transient kinds (auth, bad-request, missing-profile, missing-model), and the icm-refused notice updated to 04-UI-SPEC's locked sentence"
  - "Three per-stint counter maps (callCounts, failureCounts, blockCounts) reset to zero by resetStintCounters on every EngageLoop"
  - "tryReserveCall: the D-14 cap reservation checked before every model call (decision, reviewer, and each call's own 503 retry)"
  - "recordFailure's conditional disengage: transient kinds show a running (N of M) notice below threshold and keep the loop going; non-transient kinds, the cap-reached halt, and the blocked-repeatedly halt all disengage through the same function with their own cause/outcome overrides"
  - "recordBlocked's D-17 consecutive-block counter, firing a second, distinct blocked-repeatedly notification through recordFailure's own machinery when the resolved threshold is reached, without touching recordBlocked's own no-disengage rule"
  - "The single 503 retry (is503/notifyRetrying, a 2-second fixed retryDelay Driver field) wrapping both the player call and the reviewer call, itself reserved against the cap"
  - "driver.Event and session.AIDecisionPayload gain State/Calls/CallCap/CallCapSet/Failures/Blocks/Threshold, decorated by decorateEvent and mapped field-for-field in cmd/server/main.go's NotifierFunc (D-18)"
  - "The AI Assist panel's standing status line (.ai-assist-panel-top/.ai-assist-status-line), the terminal's outcome-keyed system-line colour, and SessionContext's onAI badge-state effect"
affects: [04-continuous-play plans 04-05 through 04-11, especially 04-11 (staging evidence capture: TestLoop_CallCap/TestLoop_ErrorThreshold/TestLoop_ConsecutiveBlocks/TestLoop_BlankSettings/TestHandleEngage_RetryOn503/TestAIPayloadCarriesSwitchState PASS lines, and the cap-halt-badge-off screenshot), the Phase 4 security review (DR-3-03, DR-3-04, DR-3.1-02 closures recorded there per D-28)]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "One recordFailure function is reused three ways (an ordinary D-13/D-15 failure, the D-14 cap-reached halt, and the D-17 blocked-repeatedly halt) via two small lookup tables (failureEventOutcome, failureDisengageCause) keyed by failure kind, rather than three separate storage/notify/disengage/log functions"
    - "decorateEvent reads the switch state via Sessions.AutopilotStateFor after any disengage the same call already applied, so the message that announces a halt always carries the post-halt state (D-18)"
    - "retryDelay is a Driver struct field (like settleDelay/floorInterval/minSpacing), not a package constant, so tests inject a millisecond value with no fake-clock library"

key-files:
  created: []
  modified:
    - internal/driver/driver.go
    - internal/driver/loop.go
    - internal/driver/driver_test.go
    - internal/driver/loop_test.go
    - internal/session/websocket.go
    - cmd/server/main.go
    - frontend/src/types/index.ts
    - frontend/src/components/AIAssistPanel.tsx
    - frontend/src/pages/PlayScreen.tsx
    - frontend/src/context/SessionContext.tsx
    - frontend/src/index.css
    - public/index.html
    - public/assets/index-D7lOKakT.css
    - public/assets/index-gu7mAusA.js
    - public/assets/index-gu7mAusA.js.map

key-decisions:
  - "The cap-reached and blocked-repeatedly halts are recorded as outcome='failed' rows with new failure_kind values (call-cap, blocked-repeatedly), not new outcome values -- no third migration touches the ai_decisions outcome CHECK constraint (04-RESEARCH A5, confirmed unchanged)"
  - "The four new non-transient kinds (auth, bad-request, missing-profile, missing-model) reuse the existing locked api-error sentence verbatim in failureNotices; only failure_kind and the immediate-disengage behaviour distinguish them, per the plan's Claude's Discretion"
  - "icm-refused's notice text changed to 04-UI-SPEC's locked sentence ('the game's own limits refused the command'), replacing Phase 3's 'refused by the command safety limits' wording; no existing test asserted the old string verbatim, so no test needed updating for the wording change itself"
  - "The 503 retry delay shipped at exactly 2 seconds, fixed, per 04-CONTEXT's Claude's Discretion -- as a Driver struct field (retryDelay) so tests override it to a millisecond value"
  - "tryReserveCall still increments callCounts even when no cap is set, so the panel's blank-cap 'Calls: {count}' status line is meaningful; only the halt comparison is skipped when CallCapSet is false"
  - "Four existing Phase 3 tests (TestHandleEngage's icm_refusal_sends_nothing, TestHandleEngageFailures, TestHandleEngageReviewer's reviewer_failures, and 04-01's TestScriptedStintSurvivesAFailureMidway) were rewritten to reflect D-15's amendment: a single transient failure no longer disengages on the first hit, it takes the resolved threshold's worth of consecutive failures. This is a deliberate, plan-anticipated change (D-15 amends Phase 3 D-13 for the loop), not a regression -- see Deviations."

patterns-established:
  - "A failure kind's disengage cause and browser-facing outcome are looked up from small kind-keyed maps (failureDisengageCause, failureEventOutcome) rather than branching in recordFailure itself, so a future halt kind is a two-line addition, not a new function"

requirements-completed: [REQ-call-cap-and-error-disengage, REQ-safety-limits-hold, REQ-doc-continuous-visible-play]

# Metrics
duration: 7min
completed: 2026-09-16
---

# Phase 4 Plan 04: Continuous Play Summary

**The session call cap, a three-strike transient-failure threshold, a consecutive-block counter and a single 503 retry now actually stop the AI driver's loop, each with its own locked notice, and the same message that announces a halt now carries the switch's post-halt state to the badge with no poll lag.**

## Performance

- **Duration:** 7 min (e755622 to 78da4db)
- **Started:** 2026-09-16T21:57:54-07:00
- **Completed:** 2026-09-16T22:04:14-07:00
- **Tasks:** 3 completed
- **Files modified:** 15 (11 source + 4 rebuilt production bundle files)

## Accomplishments

- The session call cap (D-14) halts the loop *before* the call that would exceed it, on both the decision call and the reviewer call (DR-3.1-02), with the locked notice `Session call cap reached. Autopilot disengaged.` and the switch landing off in the same message — proven by `TestLoop_CallCap`.
- Repeated transient failures (D-15) no longer burn the whole session on a single hiccup: below the resolved threshold the loop shows a running `(N of M)` count and keeps going; at the threshold it disengages with the kind's full locked sentence; a sent command resets the count; a blank threshold resolves to 3 — proven by `TestLoop_ErrorThreshold`.
- A hostile room forcing block after block can no longer spend a whole session's worth of calls (D-17, T-4-21): consecutive blocks are counted separately from failures and disengage with their own `AI decisions were blocked repeatedly. Autopilot disengaged.` notice at the same threshold — proven by `TestLoop_ConsecutiveBlocks`.
- A single vendor 503 costs a 2-second pause and a yellow `The model is unavailable, retrying...` line instead of the whole session; a second 503 is an ordinary transient failure; the retry itself counts against the cap (D-16, DR-3-04) — proven by `TestHandleEngage_RetryOn503`.
- The badge and the panel's status line can no longer disagree with the driver's own state: every event the driver emits carries the switch's state read *after* any disengage it applied, and `SessionContext`'s new `onAI` effect sets `autopilotState` from it with no poll lag (D-18, DR-3-03) — proven by `TestAIPayloadCarriesSwitchState`.
- The owner can now see how close the session is to each limit while it plays: a standing status line (`Calls: N of cap · Consecutive failures: N of M · Consecutive blocks: N of M`) sits under the goal box's future slot, hidden entirely while autopilot is off.

## Task Commits

Each task was committed atomically:

1. **Task 04-04-01: The cap, the failure threshold and the block count stop the loop and say why** - `e755622` (feat)
2. **Task 04-04-02: One retry on a 503, and every disengage carries the new switch state** - `52abc84` (feat)
3. **Task 04-04-03: The panel, the terminal and the badge show the limits as they move** - `e8fe10d` (feat)

**Production bundle rebuild:** `78da4db` (build)

**Plan metadata:** (this commit, made by the orchestrator after all worktree agents in the wave complete)

## Files Created/Modified

- `internal/driver/driver.go` — Failure-kind taxonomy split (transientFailureKinds, four new non-transient kinds, kindSentences, failureEventOutcome, failureDisengageCause); `callCounts`/`failureCounts`/`blockCounts` maps and `retryDelay` field on `Driver`; `resetStintCounters`, `tryReserveCall`, `is503`, `notifyRetrying`, `decorateEvent`; `recordFailure`/`recordBlocked`/`recordSuccess` reworked for conditional disengage, the block-repeated path, counter resets, and D-18 event decoration; `runIteration` gains cap reservations and the 503 retry at both model-call sites; `Event` gains `State`/`Calls`/`CallCap`/`CallCapSet`/`Failures`/`Blocks`/`Threshold`
- `internal/driver/loop.go` — `EngageLoop` calls `resetStintCounters` as its first statement
- `internal/driver/driver_test.go` — Four existing tests rewritten for D-15's threshold amendment; two new tests: `TestHandleEngage_RetryOn503`, `TestAIPayloadCarriesSwitchState`
- `internal/driver/loop_test.go` — `TestScriptedStintSurvivesAFailureMidway` updated for the new threshold behaviour; four new tests: `TestLoop_CallCap`, `TestLoop_ErrorThreshold`, `TestLoop_ConsecutiveBlocks`, `TestLoop_BlankSettings`
- `internal/session/websocket.go` — `AIDecisionPayload` gains the seven D-18 fields as optional JSON fields
- `cmd/server/main.go` — `NotifierFunc` maps the seven new `Event`/`AIDecisionPayload` fields
- `frontend/src/types/index.ts` — `AIDecisionPayload.outcome` union extended; seven optional state/count fields added
- `frontend/src/components/AIAssistPanel.tsx` — New `.ai-assist-panel-top` region with the standing status line, sourced from the latest `ai` message's fields, reset on autopilot-off
- `frontend/src/pages/PlayScreen.tsx` — `handleAI`'s system branch selects colour (brightyellow/white/red) from `outcome` instead of hard-coding red
- `frontend/src/context/SessionContext.tsx` — New `onAI` effect sets `autopilotState` from the `ai` message's `state` field
- `frontend/src/index.css` — `.ai-assist-panel-top`/`.ai-assist-status-line` and five new `.ai-system-line.state-*` rules, all referencing existing colour tokens
- `public/index.html`, `public/assets/index-D7lOKakT.css`, `public/assets/index-gu7mAusA.js[.map]` — Rebuilt production bundle (`npm run build`)

## Decisions Made

See `key-decisions` in the frontmatter above.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Four pre-existing Phase 3/04-01 tests asserted the amended-away single-strike disengage rule**

- **Found during:** Task 04-04-01, `go test ./internal/driver/...` after the failure-taxonomy split
- **Issue:** D-15 amends Phase 3 D-13's "any failure disengages on the first hit" rule to "transient kinds disengage only at the resolved threshold." All seven D-13/D-15 kinds are transient, so every case in `TestHandleEngageFailures`, `TestHandleEngageReviewer`'s `reviewer_failures` sub-test, `TestHandleEngage`'s `icm_refusal_sends_nothing` sub-test, and 04-01's `TestScriptedStintSurvivesAFailureMidway` — all of which called `HandleEngage` once and asserted an immediate disengage — began failing once the threshold logic existed, because a single transient failure with a fresh counter is 1 of 3, not 3 of 3.
- **Fix:** Rewrote the four tests to drive the same user through `store.DefaultDisengageThreshold` (3) consecutive calls (or, for `TestScriptedStintSurvivesAFailureMidway`, updated the single-failure assertion from "1 disengage" to "0 disengages, below threshold"), asserting the sub-threshold `(N of M)` notices and outcome `transient` along the way, and the full locked sentence with outcome `failed` plus exactly one disengage at the threshold call. This is a direct, plan-anticipated consequence of implementing D-15 as written, not a workaround.
- **Files modified:** `internal/driver/driver_test.go`, `internal/driver/loop_test.go`
- **Verification:** `go test ./internal/driver/... -count=1` and `go test ./internal/... -count=1` both green; `go test ./internal/... -race -count=1` green
- **Committed in:** `e755622` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 bug-class, spanning four test functions)
**Impact on plan:** Necessary and anticipated by the plan's own D-15 amendment language ("this amends Phase 3 D-13 for the loop"); no scope creep, no change to any locked notice string, no change to non-transient (first-hit) behaviour.

## Failure-Kind Table (final)

| Kind | Transient? | Notice (verbatim) |
|------|------------|--------------------|
| `api-error` | Yes | AI decision failed: the model could not be reached. Autopilot disengaged. |
| `rate-limited` | Yes | AI decision failed: the model's rate limit was reached. Autopilot disengaged. |
| `no-command` | Yes | AI decision failed: the model did not return a command. Autopilot disengaged. |
| `multi-command` | Yes | AI decision failed: the model returned more than one command. Autopilot disengaged. |
| `non-game-line` | Yes | AI decision failed: the model tried to issue a non-game command. Autopilot disengaged. |
| `malformed` | Yes | AI decision failed: the model's answer could not be understood. Autopilot disengaged. |
| `icm-refused` | Yes | AI decision failed: the game's own limits refused the command. Autopilot disengaged. |
| `auth` | No | AI decision failed: the model could not be reached. Autopilot disengaged. (api-error sentence reused) |
| `bad-request` | No | AI decision failed: the model could not be reached. Autopilot disengaged. (api-error sentence reused) |
| `missing-profile` | No | AI decision failed: the model could not be reached. Autopilot disengaged. (api-error sentence reused) |
| `missing-model` | No | AI decision failed: the model could not be reached. Autopilot disengaged. (api-error sentence reused) |
| `call-cap` | No (own halt, not in transient set) | Session call cap reached. Autopilot disengaged. |
| `blocked-repeatedly` | No (own halt, not in transient set) | AI decisions were blocked repeatedly. Autopilot disengaged. |

Sub-threshold transient notice shape (below the resolved threshold): `AI decision failed: {kind sentence} (N of M)` — e.g. `AI decision failed: the model could not be reached. (2 of 3)`.

Retry notice (D-16): `The model is unavailable, retrying...` — fixed 2-second delay (`Driver.retryDelay`).

## `Event` / `AIDecisionPayload` fields added (D-18)

Both structs gain the identical seven fields, mapped field-for-field in `cmd/server/main.go`'s `NotifierFunc`:

- `State` (`state`) — the autopilot switch's position (`on`/`waiting`/`off`), read after any disengage the same event's call already applied
- `Calls` (`calls`) — the stint's reserved call count so far
- `CallCap` (`call_cap`) / `CallCapSet` (`call_cap_set`) — the resolved cap, when set
- `Failures` (`failures`) — the consecutive-failure count
- `Blocks` (`blocks`) — the consecutive-block count
- `Threshold` (`threshold`) — the resolved disengage threshold

## Issues Encountered

None beyond the deviation above.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- DR-3-03 (badge updates from the disengage-carrying message), DR-3-04 (single 503 retry with notice), and DR-3.1-02 (reviewer and retry calls count against the cap) are closed in code by this plan; per `04-CONTEXT.md` D-28, their formal closure entries in `.planning/RISK-REGISTER.md` are written at the Phase 4 security review, not by this plan — flagged here so that review step does not overlook it.
- The three per-stint counters, the taxonomy split, and the switch-state-carrying `Event`/`AIDecisionPayload` are ready for plan 04-06 (session goal) and 04-08 (Session Memory) to extend the same `.ai-assist-panel-top` region and the same `ai` message shape without further plumbing changes.
- No blockers. `go build ./...`, `go vet ./internal/... ./cmd/...`, `go test ./internal/... -count=1`, `go test ./internal/... -race -count=1`, the six named tests (`TestLoop_CallCap`, `TestLoop_ErrorThreshold`, `TestLoop_ConsecutiveBlocks`, `TestLoop_BlankSettings`, `TestHandleEngage_RetryOn503`, `TestAIPayloadCarriesSwitchState`), and `cd frontend && npm run build` all pass on this machine; `git diff --stat go.mod frontend/package.json` is empty (T-4-SC).
- `internal/session/manager.go`, `internal/session/window.go` and `internal/session/window_test.go` were not touched by this plan — confirmed by `git log` over this plan's commit range — leaving them untouched for the parallel 04-05 worktree.

---
*Phase: 04-continuous-play*
*Completed: 2026-09-16*
