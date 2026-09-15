---
phase: 02-autopilot-switch
plan: 03
subsystem: session
tags: [go, websocket, concurrency, session, ai-player]

# Dependency graph
requires:
  - phase: 02-autopilot-switch (plan 02-01)
    provides: "Manager.AutopilotStateFor/EngageAutopilot/DisengageAutopilot, the AutopilotState three-value type, and the [AI-PLAYER] autopilot log line format"
provides:
  - "WSMessage.Source (inbound data messages) and MsgTypeAutopilot (outbound push) on the websocket wire contract"
  - "IsHumanSource: the allowlist classifier (only trigger/timer are automation; absent/unrecognised is human)"
  - "applyWheelGrab(m, userID, source): the testable wheel-grab decision, wired into case MsgTypeData before the clientToMUD send"
affects: [02-04-autopilot-directives-frontend]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Package-level function taking *Manager rather than a method on WebSocketHandler, so the decision is testable without opening a socket (mirrors 02-01's pure-function style)"
    - "Allowlist classification (only named automation labels opt out) rather than denylist, so an absent/forged/garbled value fails toward the safe direction"

key-files:
  created:
    - internal/session/websocket_test.go
  modified:
    - internal/session/websocket.go

key-decisions:
  - "IsHumanSource and applyWheelGrab are exported/package-level functions, not methods, exactly as the plan specifies, so 02-03-02's test drives them directly against a real Manager without a websocket.Upgrader or httptest.NewServer"
  - "The best-effort outbound autopilot push uses the existing h.writeJSON and its error is discarded (not logged, not allowed to affect the channel send) per the plan's no-lost-keystrokes requirement (T-2-12)"

requirements-completed: [REQ-wheel-grab]

# Metrics
duration: 25min
completed: 2026-09-15
---

# Phase 2 Plan 3: Wheel-Grab Source Rule Summary

**Server-side wheel-grab at the single `case MsgTypeData:` command-ingress point in `internal/session/websocket.go`: an allowlist classifier (`IsHumanSource`) treats only `trigger`/`timer` as automation, so an absent or forged source flag can only cause an unwanted disengage, never a silent bypass, proven by `TestWheelGrabSourceRule` under `-race`.**

## Performance

- **Duration:** ~25 min
- **Started:** 2026-09-15
- **Completed:** 2026-09-15
- **Tasks:** 2 completed
- **Files modified:** 2 (1 modified, 1 created)

## Accomplishments
- `internal/session/websocket.go`: `WSMessage` gains `Source string` (`json:"source,omitempty"`, inbound-only); new `MsgTypeAutopilot = "autopilot"` constant; `IsHumanSource(source string) bool` (allowlist: only `trigger`/`timer`, case-insensitive after trim, are automation); `applyWheelGrab(m *Manager, userID, source string) (bool, AutopilotState)` (disengages only when human-sourced and currently on; never engages or resumes anything).
- The wheel-grab call is wired inside `case MsgTypeData:` after the existing rate-limit/message-size/`!connected` guards and before the unchanged, unconditional `clientToMUD` select. On a grab, a best-effort `WSMessage{Type: MsgTypeAutopilot, Status: <state>, Data: "wheel-grab"}` is pushed down the same `conn` via the existing `h.writeJSON`, its error discarded.
- `internal/session/websocket_test.go` (new): `TestWheelGrabSourceRule` — a classifier table (`""`, `user`, `alias`, `trigger`, `timer`, `TRIGGER`, `not-a-real-source`) plus seven decision subtests against a real `Manager` (`human_source_disengages`, `alias_source_disengages`, `blank_source_disengages`, `trigger_source_leaves_it_on`, `off_stays_off`, `waiting_is_not_grabbed`, `second_user_untouched`). No `websocket.Upgrader`/`websocket.Dial`/`httptest.NewServer` anywhere in the file.

## Task Commits

Each task was committed atomically:

1. **Task 02-03-01: The single command-ingress point reads who typed it and takes the wheel back before the command is sent** - `c017355` (feat)
2. **Task 02-03-02: A test proves the rule for every source label, including the ones nobody sends on purpose** - `ed5cea6` (test)

## Files Created/Modified
- `internal/session/websocket.go` - `WSMessage.Source`, `MsgTypeAutopilot`, `IsHumanSource`, `applyWheelGrab`, and the wheel-grab call site inside `case MsgTypeData:`
- `internal/session/websocket_test.go` - `TestWheelGrabSourceRule`, the classifier table and the seven Manager-backed decision subtests

## Decisions Made
- `IsHumanSource`/`applyWheelGrab` kept as package-level functions (not `WebSocketHandler` methods) per the plan, enabling `internal/session/websocket_test.go` to prove the rule with zero network/socket setup.
- The outbound autopilot push's error is silently discarded — never logged with command text, never causes a `continue` that would skip the channel send — satisfying the "no keystroke may be dropped" requirement (T-2-12) and the acceptance criterion that the log line only ever comes from `manager.go`'s fixed `[AI-PLAYER] autopilot` format (T-2-04).

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None. The repository's pre-existing `core.autocrlf=true` setting means all `internal/session/*.go` files have CRLF line endings on disk in this worktree; this is the established convention (verified against untouched files `manager.go`/`autopilot.go`, both also CRLF on disk) and git normalizes to LF on commit — not a defect introduced by this plan.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- The wire contract (`WSMessage.Source` on inbound `data` messages; `MsgTypeAutopilot` outbound with `Status`/`Data` carrying state/cause) is fixed and ready for plan 02-04's frontend work: threading `CommandSource` into the outbound `data` message and handling the new `autopilot` message type in the websocket client.
- `IsHumanSource`'s allowlist (`trigger`, `timer`) is the frontend contract: the browser's existing `CommandSource` values (`user`, `alias`, `trigger`) already map correctly; a future `timer` label (if the frontend ever splits `trigger` into `trigger`/`timer`) requires no server-side change.
- T-2-08 (bounded WAITING lifetime) and the WAITING-then-resume security concern remain deferred to the Phase 2 security review, as scoped in 02-01's summary — this plan makes no change to that.

## Self-Check: PASSED

- FOUND: internal/session/websocket.go
- FOUND: internal/session/websocket_test.go
- FOUND: commit c017355 (Task 1)
- FOUND: commit ed5cea6 (Task 2)
- `go build ./...` exit 0
- `go vet ./internal/session/...` exit 0
- `grep -c 'applyWheelGrab' internal/session/websocket.go` = 3 (decl, doc comment, call site)
- `grep -c 'source,omitempty'` present on `WSMessage`; `MsgTypeAutopilot = "autopilot"` present in the constant block
- `grep -n '\[AI-PLAYER\]' internal/session/websocket.go | grep -c 'wsMsg.Data'` = 0
- `applyWheelGrab` call site (line 418) is after the `if !connected` guard (line 408) and before the `clientToMUD` select (line 424)
- `case MsgTypeConnect:` contains no autopilot code (verified by grep against that block)
- `grep -c 'websocket.Upgrader\|websocket.Dial\|httptest.NewServer' internal/session/websocket_test.go` = 0
- `go test ./internal/session/... -run TestWheelGrabSourceRule -race -v` exit 0, all PASS (classifier: 7 subtests; decision: 7 subtests)
- `go test ./internal/session/... -race` exit 0, no data race reported
- `go test ./...` shows only the pre-existing, excused `internal/icm` `TestHandlerRegistration/CANCEL` baseline failure — no new failures
- `git diff --stat go.mod frontend/package.json` empty (T-2-SC)
- `git diff --diff-filter=D --name-only` empty for both task commits — no unintended deletions

---
*Phase: 02-autopilot-switch*
*Completed: 2026-09-15*
