---
phase: 02-autopilot-switch
reviewed: 2026-09-15T00:00:00Z
depth: standard
files_reviewed: 20
files_reviewed_list:
  - cmd/server/main.go
  - internal/session/autopilot.go
  - internal/session/autopilot_test.go
  - internal/session/handler.go
  - internal/session/handler_test.go
  - internal/session/manager.go
  - internal/session/manager_test.go
  - internal/session/websocket.go
  - internal/session/websocket_test.go
  - frontend/src/components/AutopilotBadge.tsx
  - frontend/src/components/Header.tsx
  - frontend/src/components/Sidebar.tsx
  - frontend/src/context/SessionContext.tsx
  - frontend/src/pages/PlayScreen.tsx
  - frontend/src/services/api.ts
  - frontend/src/services/automation.ts
  - frontend/src/services/automation/commands.ts
  - frontend/src/services/automation/evaluator.ts
  - frontend/src/types/index.ts
  - frontend/src/index.css
  - scripts/verify-phase2.sh
findings:
  critical: 3
  warning: 3
  info: 2
  total: 8
status: issues_found
---

# Phase 02: Code Review Report

**Reviewed:** 2026-09-15T00:00:00Z
**Depth:** standard
**Files Reviewed:** 20
**Status:** issues_found

## Summary

Reviewed the Phase 2 autopilot-switch diff (`628a038..HEAD`) across the Go session package (autopilot state machine, HTTP handler, Manager, websocket handler) and the frontend wiring (SessionContext, PlayScreen, WebSocketManager, evaluator's `#AUTO` command). The pure state-transition functions in `autopilot.go` are correct and well covered by `autopilot_test.go`. Cross-user isolation (T-2-02, "the body cannot name another user's session") is genuinely enforced — `EngageGate`'s SQL scopes by `(connection_id, user_id)` together, and the handler never trusts a user id from the request body.

However, two related design gaps undermine the Phase 1 policy-acceptance gate that this phase is built on top of: the engage-time gate check never verifies that the `connection_id` supplied in the request actually corresponds to the live game session being engaged (a same-user, cross-connection bypass), and the reconnect "resume" path (`waiting` → `on`) re-engages autopilot on *any* future connection with **no gate check at all**, because `Manager` (unlike `Handler`) holds no reference to `EngageGate`. On the frontend, `#AUTO OFF` — the one command the whole design insists must "always work" (D-01) — is unreachable after a browser refresh while autopilot is parked, because the connection id needed to make the call is never persisted or recoverable across a reload. Also found a benign-but-real TOCTOU race in the websocket wheel-grab check-then-act sequence, and a type-safety gap on the outbound `WSMessage.source` field.

## Critical Issues

### CR-01: Reconnecting after any drop silently resumes autopilot with no gate re-check, for any future connection

**File:** `internal/session/manager.go:488-503` (`resumeAutopilotLocked`), called from `internal/session/manager.go:221` (`Connect`)

**Issue:** `Manager.Connect` unconditionally calls `m.resumeAutopilotLocked(userID)` on every successful dial. `resumeAutopilotLocked` only inspects the stored `AutopilotRecord.State` and applies the pure `Resume()` transition (`waiting` → `on`) — it never calls back into `HandlerCallbacks.EngageGate`, because `Manager` holds no reference to it at all (only `Handler` does, see `handler.go:31`). Once a user has engaged autopilot for connection A (which required an accepted policy), any subsequent drop simply parks the record as `waiting` with no expiry (`WaitingSince` is set but read by nothing, per the comment on `autopilot.go:39-41`). The **next** successful `Connect` for that user — to connection A again, to a completely different saved connection B whose policy was never accepted, or even to an ad-hoc Quick Connect host with no saved profile at all — silently flips the switch back to `on`, dispatching the AI player against a game the owner never consented to for AI use. `EngageAutopilot` (the only path that does call the gate) is bypassed entirely on resume.

This is a stronger version of CR-02 below: CR-02 requires the attacker to own a second accepted connection and craft a raw API call; this one fires automatically, from the ordinary UI, for any user who disconnects without first typing `#AUTO OFF` and then plays a different game later in the same session's lifetime (no session restart needed — `AutopilotOff` is the only state a restart resets, per D-06).

**Fix:** Re-validate the gate on resume, or bind the parked record to the connection it was engaged for and refuse to resume unless the reconnecting `connection_id` matches:
```go
// resumeAutopilotLocked needs either:
// (a) a gate-check callback threaded into Manager (mirroring HandlerCallbacks.EngageGate), or
// (b) the caller (Handler.Connect / Handler's websocket path) supplying the connectionID being
//     dialed, compared against rec.ConnectionID, with resume refused on mismatch and the
//     record simply staying "waiting" (or moving to "off") until #AUTO ON is re-issued and the
//     gate is re-checked.
```
At minimum, give `WaitingSince` the bounded lifetime the comment already anticipates (T-2-08) so a week-old parked engagement cannot silently reactivate.

### CR-02: `#AUTO ON`'s gate check is not bound to the live session's connection

**File:** `internal/session/handler.go:326-366` (`Autopilot`, case `"on"`); `cmd/server/main.go:192-207` (`EngageGate` closure); `internal/session/manager.go:401-430` (`EngageAutopilot`)

**Issue:** `EngageGate(req.ConnectionID, userUUID)` only proves that the caller *owns* `req.ConnectionID` and that *that* connection's profile has accepted the policy — it has no way to prove `req.ConnectionID` is the connection driving the caller's current live TCP session, because `session.Session` (manager.go:37-45) stores only `Host`/`Port`/`State`, never a connection id. A user with two saved connections — A (policy accepted) and B (policy never accepted) — who is currently playing on B can call `POST /api/v1/session/autopilot {"action":"on","connection_id":"<A>"}` directly (bypassing the frontend, which normally only ever sends `currentConnectionId`) and engage autopilot against the live session actually connected to B. The Phase 1 per-connection consent gate is thereby satisfied by borrowing acceptance from an unrelated connection.

**Fix:** Correlate the live session to a connection id server-side (e.g. store the connection id used at `Connect` time on the `Session` struct, or look up the `connections` row by `Host`/`Port` for that user) and reject `#AUTO ON` when `req.ConnectionID` does not match the connection that actually produced the live session, rather than trusting the client-supplied id verbatim.

### CR-03: `#AUTO OFF` is unreachable from the UI after a page refresh while autopilot is `waiting`

**File:** `frontend/src/context/SessionContext.tsx:267-287`

**Issue:** `parkedConnectionIdRef` (line 267) and `currentConnectionId` (line 97) are both plain in-memory React state — they reset to `null` on every fresh mount of `SessionProvider`, i.e. on every browser refresh. `StatusResponse` (`handler.go:75-88`) never carries a connection id, so `refreshStatus()` (which does correctly restore `autopilotState` to `"waiting"` after a reload) has no way to repopulate either ref. The result: after a refresh while parked, `boundId` at line 274 evaluates to `null`, `automationEngine.setAutopilotControl(undefined)` is called, and typing `#AUTO OFF` hits the "no connection to autopilot control yet" branch in `evaluator.ts` (`"[Autopilot needs a connected game; connect first]"`) instead of actually turning the switch off. This directly breaks D-01 ("the owner can always take the wheel back") in a routine scenario — a user parks autopilot, closes/refreshes the tab, comes back, and cannot disengage from the UI at all (the badge still correctly shows "Waiting", making the failure invisible until the user tries `#AUTO OFF` and it silently does nothing useful). Note the server side does not need a real `connection_id` for `"off"` at all (`handler.go:368-379` never reads `req.ConnectionID` in that branch), so this is purely a frontend wiring gap, not a server limitation.

**Fix:** Don't gate `#AUTO OFF` availability on knowing a connection id. E.g. always keep `autopilotControl` wired whenever `autopilotState !== 'off'`, using a placeholder/zero connection id when none is known and none is required by the action:
```ts
useEffect(() => {
  if (!automationEngine) return;
  const boundId = currentConnectionId ?? parkedConnectionIdRef.current ?? (autopilotState !== 'off' ? '00000000-0000-0000-0000-000000000000' : null);
  ...
```
or better, add a connection id to `StatusResponse` so the frontend can actually recover which connection is parked after a reload.

## Warnings

### WR-01: Wheel-grab check-then-act is not atomic across the two lock acquisitions

**File:** `internal/session/websocket.go:69-81` (`applyWheelGrab`)

**Issue:** `applyWheelGrab` calls `m.AutopilotStateFor(userID)` (acquires and releases `m.mu.RLock()`) and then, if `AutopilotOn`, separately calls `m.DisengageAutopilot(userID, "wheel-grab")` (acquires `m.mu.Lock()`). Between the two calls, a concurrent goroutine (e.g. a REST call to `POST /api/v1/session/autopilot {"action":"off"}` racing with a websocket data message from the same user/tab) can change the record's state, so the read that gated the decision is stale by the time the write happens. The user-visible effect is benign today (worst case: a wheel-grab push is skipped once, or `DisengageAutopilot` is a harmless no-op), but this is exactly the kind of check-then-act pattern the rest of the package deliberately avoids by doing all read-modify-write work under a single lock acquisition inside `Manager` methods (see `EngageAutopilot`, which reads and writes under one `Lock()`).

**Fix:** Add a `Manager` method that performs the wheel-grab's read-and-maybe-disengage atomically under one lock, e.g. `func (m *Manager) WheelGrab(userID string) (grabbed bool, state AutopilotState)`, so `applyWheelGrab` in websocket.go becomes a thin wrapper around it instead of two independently-locked calls.

### WR-02: `WSMessage.source` (frontend) is typed as a bare `string`, not the `CommandSource` union

**File:** `frontend/src/types/index.ts:54-58`; contrast with `frontend/src/services/automation.ts:20` (`export type CommandSource = 'user' | 'alias' | 'trigger';`)

**Issue:** The outbound websocket `source` field — which the server's `IsHumanSource` (`websocket.go:45-61`) uses to decide whether to disengage autopilot — is typed as `source?: string` in `types/index.ts` rather than `CommandSource`. `WebSocketManager.sendCommand` (`api.ts`) does accept a typed `source?: CommandSource` parameter and forwards it into the message correctly today, but the wire-level `WSMessage` type itself doesn't enforce this, so a future caller constructing a `WSMessage` directly (bypassing `sendCommand`) gets no compile-time protection against sending an arbitrary string into a field whose value gates a safety-relevant server decision.

**Fix:**
```ts
export interface WSMessage {
  type: WSMessageType;
  ...
  source?: CommandSource; // import from services/automation
}
```

### WR-03: D-08's "typed `#` directive relabels its own output as automation" rule is line-scoped, not command-scoped

**File:** `frontend/src/services/automation.ts:518-523, 547-552`

**Issue:** `isDirectiveInput` is computed once per top-level input string (`input.trimStart().startsWith('#')`) and applied to *every* command any internal directive on that line emits (`source: isDirectiveInput ? 'trigger' : 'user'`). Today this is safe in practice only because the one directive capable of conditionally emitting literal game text from CLI, `#IF`, is explicitly blocked when `context.source === 'cli'` (`evaluator.ts:864-873`). If a future `#`-command is added that can emit a real game command from typed CLI input (e.g. a macro-expansion directive), it will be silently mislabeled `'trigger'` (automation) instead of `'user'` (human) by this rule, which — per `IsHumanSource` on the server — means a typed command that should take the wheel back from autopilot would not. There is no test asserting this rule stays scoped correctly as new `#` commands are added.

**Fix:** Either scope the relabeling to the specific commands known to be side-effect-only (documented as "no command is ever emitted here" in the `#AUTO` case itself), or add a regression test that fails if any future CLI-reachable directive emits a `result.commands` entry while `isDirectiveInput` is true, so the D-08 invariant is enforced rather than merely coincidental.

## Info

### IN-01: `#AUTO`'s outcome switch silently no-ops on an unrecognized `outcome`

**File:** `frontend/src/services/automation/evaluator.ts:1206-1226`

**Issue:** The `switch (answer.outcome)` in the `AUTO` case has a `default: break` with no `outputMessage` call and no console warning. If the server ever returns an outcome value the frontend doesn't recognize (contract drift, a typo in a future server change, a proxy/error page returning unexpected JSON with `outcome` coerced to something odd), the user who just typed `#AUTO ON/OFF/STATUS` gets no terminal feedback at all — it will look like the command did nothing.

**Fix:** Log to console and/or emit a generic `[Autopilot: unexpected response]` line in the `default` branch so contract drift is visible rather than silent.

### IN-02: Redundant `refreshStatus()` calls on user-initiated disconnect

**File:** `frontend/src/context/SessionContext.tsx:611-645`; `frontend/src/pages/PlayScreen.tsx` (`handleDisconnectRef`)

**Issue:** `disconnect()` calls `wsManager.notifyDisconnect()` (which synchronously fires `PlayScreen`'s `handleDisconnectRef`, which itself calls `refreshStatus()`), and then `disconnect()` itself calls `await refreshStatus()` again a few lines later. Both calls hit `GET /api/v1/session/status`. Harmless (not in scope per the performance carve-out) but worth trimming for clarity — a reader has to trace both call sites to confirm there's no ordering dependency.

**Fix:** No functional fix required; consider a comment noting the double-call is intentional (belt-and-braces for the synchronous vs. async firing order) so a future edit doesn't "simplify" it into a race.

---

_Reviewed: 2026-09-15T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
