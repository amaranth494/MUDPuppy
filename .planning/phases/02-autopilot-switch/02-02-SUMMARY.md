---
phase: 02-autopilot-switch
plan: 02
subsystem: session
tags: [go, http, session, autopilot, ai-player, security]

# Dependency graph
requires:
  - phase: 02-autopilot-switch (plan 01)
    provides: "Manager.AutopilotStateFor/EngageAutopilot/DisengageAutopilot and the [AI-PLAYER] autopilot log line format"
  - phase: 01-profile-foundation-and-policy-gate
    provides: "store.EngageGateAllowed, store.EngageGateRefusalMessage, store.ProfileStore.GetProfileByConnection"
provides:
  - "POST /api/v1/session/autopilot: on/off/status behind the session-middleware mux"
  - "AutopilotRequest/AutopilotResponse wire contract for plans 02-04, 02-05, 02-06"
  - "HandlerCallbacks.EngageGate — the fail-closed injection seam reaching Phase 1's gate"
  - "StatusResponse.autopilot_state (no omitempty) on GET /api/v1/session/status"
affects: [02-04-directive-grammar-and-badge, 02-05-canned-report, 02-06-wheel-grab]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Callback-injection seam on HandlerCallbacks (EngageGate), mirroring the existing OnConnected/GetAutoLogin pattern, so session.Handler never takes a direct *store.ProfileStore dependency"
    - "Gate resolved once per request, before branching on action, so status can report gate_allowed/gate_message/policy_version too (D-05)"
    - "Ownership-scoped profile lookup (GetProfileByConnection(userID, connectionID)) as the IDOR mitigation for a connection id the caller does not own"

key-files:
  created:
    - internal/session/handler_test.go
  modified:
    - internal/session/handler.go
    - cmd/server/main.go

key-decisions:
  - "EngageGate closure treats a lookup error or nil profile identically to a gate-false profile (returns store.EngageGateRefusalMessage in both cases) — an unknown/unowned connection is indistinguishable from an unaccepted one on the wire, which is the correct ownership-check behavior per the plan's own instruction"
  - "A nil HandlerCallbacks.EngageGate (or nil callbacks struct) short-circuits to gate_allowed=false before EngageAutopilot is ever called — verified by the nil_gate_callback_refuses subtest"

requirements-completed: [REQ-autopilot-directives, REQ-doc-hand-play-and-gate]

# Metrics
duration: ~25min
completed: 2026-09-15
---

# Phase 2 Plan 2: Autopilot HTTP Endpoint and Gate Wiring Summary

**One `POST /api/v1/session/autopilot` endpoint (on/off/status) that resolves Phase 1's policy-acceptance gate via a new ownership-scoped `EngageGate` callback before ever touching the switch, and an extended `GET /api/v1/session/status` that now carries the server-held autopilot state unconditionally so the browser badge re-syncs after a refresh.**

## Performance

- **Duration:** ~25 min
- **Tasks:** 3 completed
- **Files modified:** 2 (`internal/session/handler.go`, `cmd/server/main.go`); 1 created (`internal/session/handler_test.go`)

## Accomplishments
- `internal/session/handler.go`: `HandlerCallbacks.EngageGate func(connectionID, userID uuid.UUID) (allowed bool, message string, policyVersion string)`; `AutopilotRequest{Action, ConnectionID}` and `AutopilotResponse{State, Outcome, GateAllowed, GateMessage, PolicyVersion}`; `func (h *Handler) Autopilot(w, r)` resolving the gate first for every action (`on`/`off`/`status`), then branching — `on` refuses with `store.EngageGateRefusalMessage` passed through verbatim when the gate fails, or calls `manager.EngageAutopilot` when it passes (answering `engaged`/`already-on`/`refused-no-session`); `off` always calls `manager.DisengageAutopilot` regardless of gate result (D-01); `status` changes nothing. `StatusResponse` gains `AutopilotState string` with tag `autopilot_state` (no `omitempty`), populated unconditionally in `Status`.
- `cmd/server/main.go`: registered `POST /api/v1/session/autopilot` inside the same `sessionMiddleware`-wrapped mux as the other session routes; added the `EngageGate` closure to the `session.NewHandlerWithCallbacks` literal, resolving the profile via `profileStore.GetProfileByConnection(userID, connectionID)` (ownership-scoped on both columns) and `store.EngageGateAllowed`, returning `store.EngageGateRefusalMessage` on every refusal path (lookup error, nil profile, or gate false) and the dereferenced (nil-guarded) policy version on success.
- `internal/session/handler_test.go` (first test file for this handler): `TestAutopilotHandler` with 12 named subtests — unauthenticated 401, wrong-method 405, gate refusal, no-connection refusal, engage + already-on, disengage + already-off, off-allowed-even-when-gate-refuses (D-01), status carrying the gate result, invalid-action 400, nil-callback fail-closed, and a body-cannot-name-another-user IDOR proof (T-2-02). `TestStatusCarriesAutopilotState` proves the raw JSON body always contains `"autopilot_state":"off"` with no `omitempty`.

## Task Commits

Each task was committed atomically:

1. **Task 02-02-01: One endpoint turns the switch, answers its position, and refuses with Phase 1's own sentence** - `67ead06` (feat)
2. **Task 02-02-02: The route exists and the gate callback reaches the profile the caller actually owns** - `dfa2882` (feat)
3. **Task 02-02-03: A handler test proves every refusal, every no-op, and that the body can never name another user** - `8fa1c3e` (test)

**Plan metadata:** (this commit)

## Files Created/Modified
- `internal/session/handler.go` - `HandlerCallbacks.EngageGate`, `AutopilotRequest`/`AutopilotResponse`, the `Autopilot` handler, `StatusResponse.AutopilotState`
- `cmd/server/main.go` - the `/api/v1/session/autopilot` route registration and the `EngageGate` closure wiring `profileStore.GetProfileByConnection` + `store.EngageGateAllowed`
- `internal/session/handler_test.go` (new) - `TestAutopilotHandler` (12 subtests) and `TestStatusCarriesAutopilotState`

## Decisions Made
- The `EngageGate` closure collapses "lookup error", "nil profile" and "gate false" into the same refusal response (`false`, `store.EngageGateRefusalMessage`, `""`) — an unowned or unknown connection id is indistinguishable from an unaccepted policy on the wire, which is the plan's specified ownership-check behavior (T-2-02), not an information leak about which connections exist.
- A nil `EngageGate` callback (or nil `HandlerCallbacks`) is treated as a hard refusal before `EngageAutopilot` is ever called, matching the fail-closed requirement for a missing dependency (T-2-10).

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- `go test -race` requires `CGO_ENABLED=1` and a cgo-capable C compiler; none was on `PATH` in this worktree's shell by default. Reused the same user-scope WinLibs MinGW-w64 toolchain plan 02-01 installed via `winget install --scope user` (no elevation), prepending its `bin/` to `PATH` for the `-race` runs. No repo files changed by this; `go.mod`/`frontend/package.json` are unchanged (verified below). Not logged as a plan deviation since it is host tooling already installed by 02-01 and this plan added nothing new.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- The `POST /api/v1/session/autopilot` contract (`AutopilotRequest`/`AutopilotResponse`, the `on`/`off`/`status` action set, and the exact `Outcome` vocabulary) is stable and ready for plan 02-04 (directive grammar + badge, which calls this endpoint from `#AUTO ON/OFF/STATUS`), plan 02-05 (canned report, which drives this endpoint via HTTP fixtures), and plan 02-06 (wheel-grab).
- `StatusResponse.autopilot_state` is ready for the frontend's 15-second poll / on-mount / visibility-change refresh to consume (D-10).
- The `[AI-PLAYER] autopilot ... cause=refused-gate` log line is the new cause this plan adds to plan 02-01's fixed log format, ready to be captured as staging evidence by plan 02-07.

## Self-Check: PASSED

- FOUND: internal/session/handler.go (modified)
- FOUND: cmd/server/main.go (modified)
- FOUND: internal/session/handler_test.go (created)
- FOUND: commit 67ead06 (Task 1)
- FOUND: commit dfa2882 (Task 2)
- FOUND: commit 8fa1c3e (Task 3)
- `go build ./...` exit 0
- `go vet ./internal/session/... ./cmd/...` exit 0
- `go test ./internal/session/... -run "TestAutopilotHandler|TestStatusCarriesAutopilotState" -race -v` exit 0, all 12 `TestAutopilotHandler` subtests PASS plus `TestStatusCarriesAutopilotState` PASS
- `go test ./internal/session/... -race` exit 0, no data race reported
- `go test ./...` shows only the pre-existing, excused `internal/icm` `TestHandlerRegistration/CANCEL` baseline failure — no new failures
- `grep -c '"/api/v1/session/autopilot"' cmd/server/main.go` = 1, inside the same session-middleware mux block as `/api/v1/session/status`
- `grep -rc 'AI Player has not been configured' internal/session/` = 0 for every file (refusal sentence only ever passed through from `store`)
- `git diff --stat go.mod frontend/package.json` empty (T-2-SC)

---
*Phase: 02-autopilot-switch*
*Completed: 2026-09-15*
