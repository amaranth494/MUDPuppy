---
phase: 02-autopilot-switch
plan: 06
subsystem: ui
tags: [typescript, react, frontend, websocket, autopilot, ai-player]

# Dependency graph
requires:
  - phase: 02-autopilot-switch (plan 02-01)
    provides: "AutopilotState (off/on/waiting) three-value state, server-held per session"
  - phase: 02-autopilot-switch (plan 02-02)
    provides: "POST /api/v1/session/autopilot and StatusResponse.autopilot_state (no omitempty)"
  - phase: 02-autopilot-switch (plan 02-03)
    provides: "Server-side wheel-grab pushing WSMessage{Type: autopilot, Status: state, Data: 'wheel-grab'}"
  - phase: 02-autopilot-switch (plan 02-04)
    provides: "#AUTO directive grammar, api.ts's setAutopilot/onAutopilot/offAutopilot, SessionStatus.autopilot_state, automation.ts's setAutopilotControl seam, source-tagged sendCommand"
provides:
  - "autopilotState in SessionContext, populated by the 15s/visibility/mount status poll (refresh-correctness, D-10) and by the live websocket push (responsiveness layer)"
  - "AutopilotBadge.tsx: header badge beside SessionBadge reading Autopilot: On/Waiting/Off, driven strictly by server-held state"
  - "The #AUTO directive wired end-to-end: SessionContext's setAutopilotControl effect calls api.ts's setAutopilot with the tracked connection id"
  - "Wheel-grab notice, waiting/resume notices, and D-07's blank-Enter human tag in PlayScreen.tsx -- closes all four ROADMAP Phase 2 success criteria's owner-visible half"
affects: [02-07-evidence-capture]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "New useEffect keyed on the wsManager state variable (not a pre-existing effect) to pair onAutopilot/offAutopilot subscription with cleanup, since no existing effect 'owns' the WebSocketManager instance in this file"
    - "Transition effect holding the previous value in a ref, guarding the first render (ref starts undefined) so mount never prints a spurious notice -- the same shape as a diff/edge-detector, applied to autopilotState"

key-files:
  created:
    - frontend/src/components/AutopilotBadge.tsx
  modified:
    - frontend/src/context/SessionContext.tsx
    - frontend/src/components/Header.tsx
    - frontend/src/index.css
    - frontend/src/pages/PlayScreen.tsx
    - frontend/src/services/automation.ts
    - public/index.html
    - public/assets/index-CKt8wwg3.js (bundle, generated)
    - public/assets/index-CKt8wwg3.js.map (bundle, generated)
    - public/assets/index-DZFCatzZ.css (bundle, generated)

key-decisions:
  - "automation.ts's setAutopilotControl signature widened to accept undefined (Rule 3 blocking fix): the plan's task text explicitly requires SessionContext to clear the control when currentConnectionId is null (quick-connect sessions with no saved profile) so #AUTO prints its no-connected-game line without a server round trip; the pre-existing signature required a non-optional object, which would have been a type error. automation.ts is not in this plan's five-file scope but the one-line, backward-compatible widening (control ?? null) was necessary to satisfy the plan's own acceptance criteria."
  - "No dedicated effect 'owns' the WebSocketManager in SessionContext.tsx (unlike RESEARCH.md's assumption) -- wsManager is created inside the connect() callback and stored via useState, not inside a useEffect. Implemented the onAutopilot/offAutopilot subscription as a new useEffect keyed on the wsManager state variable instead, which is the correct React idiom for subscribing to an externally-created object stored in state."
  - "connect()'s existing end-of-function refreshStatus() call (pre-dating this plan, labelled 'Event-driven: verify connection status after connect action') already satisfies the acceptance criterion 'connect() calls refreshStatus() after the REST connect resolves' -- no additional call was needed."

requirements-completed: [REQ-autopilot-directives, REQ-wheel-grab, REQ-no-auto-reconnect, REQ-doc-hand-play-and-gate]

# Metrics
duration: ~40min
completed: 2026-09-15
---

# Phase 2 Plan 6: Autopilot Badge and Wheel-Grab Wiring Summary

**A header badge beside the connection badge reads `Autopilot: On/Waiting/Off` from the server's own status response and websocket push (never derived locally), while `PlayScreen.tsx` prints the wheel-grab, waiting, and resume notices verbatim from `02-UI-SPEC.md` and adds no output whatsoever when the switch is off.**

## Performance

- **Duration:** ~40 min
- **Completed:** 2026-09-15T20:56:31Z
- **Tasks:** 3 completed
- **Files modified:** 5 source files (3 tasks) + 1 automation.ts blocking fix + 3 generated bundle files (1 build commit)

## Accomplishments

- `frontend/src/context/SessionContext.tsx`: `autopilotState: 'on' | 'waiting' | 'off'` added to `SessionContextType`, defaulting to `'off'` (D-06, the safe direction for a fresh page or restarted server). Populated in three places: (1) `refreshStatus`'s existing `try` block now sets it from `status.autopilot_state || 'off'` -- this is D-10's entire refresh-correctness mechanism, riding on `SessionBadge.tsx`'s existing 15s/visibility/mount poll with no new polling loop; (2) a new `useEffect` keyed on the `wsManager` state variable subscribes `wsManager.onAutopilot`/unsubscribes `offAutopilot` in its cleanup -- the sub-15s responsiveness layer; (3) a new `useEffect` keyed on `[automationEngine, currentConnectionId]` wires `automationEngine.setAutopilotControl` to a `setState` closure that calls `api.ts`'s `setAutopilot(connectionId, action)` and updates `autopilotState` from the response, clearing the control (`setAutopilotControl(undefined)`) when `currentConnectionId` is null so `#AUTO` prints its no-connected-game line without a server round trip.
- `frontend/src/components/AutopilotBadge.tsx` (new): default-exported, no-props component reading `autopilotState` from `useSession()`. No interval, no visibility listener, no local state -- `SessionBadge.tsx` already keeps the shared context fresh. Renders `<div class="autopilot-badge state-{on|waiting|off}" title="Autopilot: {label}">` with a dot span and a text span reading `Autopilot: {On|Waiting|Off}`, matching `02-UI-SPEC.md` verbatim.
- `frontend/src/components/Header.tsx`: `<AutopilotBadge />` placed inside `.header-right` immediately after `<SessionBadge />` and before the account menu.
- `frontend/src/index.css`: `.autopilot-badge`, `.autopilot-badge-dot`, `.autopilot-badge.state-on/.state-waiting/.state-off` appended directly below `.session-badge.status-error` -- a verbatim structural copy of the connection badge's own rules, no new hex or spacing values.
- `frontend/src/pages/PlayScreen.tsx`: `setSubmitCommandCallback` registration widened to `(command, source)`, passed straight through to `wsManager.sendCommand(command + '\n', source)`. The direct-send `else` branch (blank Enter / no automation engine) now calls `wsManager.sendCommand(command + '\n', 'user')` explicitly, with a comment naming D-07, so a blank Enter still takes the wheel (RESEARCH Pitfall 2). A new `wsManager.onAutopilot`/`offAutopilot` pair lives in the same effect as the existing `onMessage`/`onError`/`onDisconnect` registrations; cause `'wheel-grab'` writes `[Autopilot disengaged: you took the wheel]` (white) through `echoLocal`. `handleDisconnectRef.current` now calls `refreshStatus()` immediately after its existing, unchanged `[Disconnected]` echo, so the context's `autopilotState` flips promptly. A new effect keyed on `autopilotState`, holding the previous value in a ref (undefined on first render, so mount prints nothing), handles exactly two transitions: on-to-waiting writes `[Autopilot waiting for reconnect]` (brightyellow); waiting-to-on writes `[Reconnected]` (white) then `[Autopilot resuming]` (brightgreen) as two separate `echoLocal` calls. Every other transition, including on-to-off and waiting-to-off (already announced by the directive response or the wheel-grab push), writes nothing -- holding D-12.
- `frontend/src/services/automation.ts`: `setAutopilotControl`'s parameter widened to accept `undefined` (see Deviations).
- Public bundle rebuilt and committed (`public/index.html`, `public/assets/index-CKt8wwg3.js[.map]`, `public/assets/index-DZFCatzZ.css`) so the running server serves this plan's changes.

## Task Commits

Each task was committed atomically:

1. **Task 02-06-01: The session context holds the server's answer and hands the directive its line to the server** - `7e6cb12` (feat)
2. **Task 02-06-02: A badge beside the connection badge states the switch position** - `f21d2ba` (feat)
3. **Task 02-06-03: Taking the wheel, parking, and resuming each say so in the terminal, and ordinary play says nothing new** - `7396ae3` (feat)
4. **Public bundle rebuild** - `2ab5e63` (build)

**Plan metadata:** (this commit)

## Files Created/Modified

- `frontend/src/context/SessionContext.tsx` - `autopilotState` field, three population points (poll, live push, `#AUTO` control wiring)
- `frontend/src/components/AutopilotBadge.tsx` (new) - the header badge, three states, server-driven only
- `frontend/src/components/Header.tsx` - `<AutopilotBadge />` placement
- `frontend/src/index.css` - the three badge state classes
- `frontend/src/pages/PlayScreen.tsx` - source-tagged submission paths, wheel-grab notice, waiting/resume notice sequences
- `frontend/src/services/automation.ts` - `setAutopilotControl` widened to accept `undefined` (deviation, see below)
- `public/index.html`, `public/assets/index-CKt8wwg3.js[.map]`, `public/assets/index-DZFCatzZ.css` - rebuilt production bundle

## Decisions Made

- `automation.ts`'s `setAutopilotControl` signature widened to accept `undefined` -- see Deviations.
- The `onAutopilot`/`offAutopilot` websocket subscription lives in a *new* `useEffect` keyed on the `wsManager` state variable, not inside an existing effect, because no existing effect in `SessionContext.tsx` "owns" the `WebSocketManager` instance (it's created inside the `connect()` callback and stored via `useState`). This is the correct React idiom for the shape this codebase actually has, not a deviation from RESEARCH.md's intent.
- `connect()`'s pre-existing end-of-function `refreshStatus()` call already satisfies "connect() calls refreshStatus() after the REST connect resolves" -- no additional call was added, since one already existed and already runs after `await connectToMud(...)`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Widened `automation.ts`'s `setAutopilotControl` to accept `undefined`**
- **Found during:** Task 02-06-01
- **Issue:** The plan's task text instructs: "When `currentConnectionId` is null, call `setAutopilotControl(undefined)` (or pass a control whose `setState` is absent) so the directive prints its no-connected-game line without a server round trip." The existing signature from plan 02-04 (`setAutopilotControl(control: { setState: ... }): void`) required a non-optional object -- calling it with `undefined` was a TypeScript compile error, blocking the task.
- **Fix:** Widened the parameter to `control: { setState: ... } | undefined`, storing `control ?? null` internally (the field was already nullable). One-line, additive, backward-compatible; no existing call site is affected (there were none prior to this plan).
- **Files modified:** `frontend/src/services/automation.ts`
- **Verification:** `cd frontend && npm run build` (tsc + vite) exits 0; `grep -c 'setAutopilotControl' src/context/SessionContext.tsx` = 2 (the `undefined` branch and the populated-control branch)
- **Committed in:** `7e6cb12` (Task 1 commit)

**2. [Rule 1 - Bug] Reworded a doc comment that defeated its own literal acceptance-criteria grep**
- **Found during:** Task 02-06-02 verification
- **Issue:** `AutopilotBadge.tsx`'s explanatory comment said "it does not read `connectionState`", which is prose, not code -- but the acceptance criterion's literal grep (`grep -c 'connectionState\|useState\|setInterval' frontend/src/components/AutopilotBadge.tsx` must return 0) matched the comment text itself, not any real usage.
- **Fix:** Reworded the comment to describe the same guarantee without using the literal substring `connectionState`.
- **Files modified:** `frontend/src/components/AutopilotBadge.tsx`
- **Verification:** `grep -c 'connectionState\|useState\|setInterval' frontend/src/components/AutopilotBadge.tsx` now returns 0; `npm run build` still exits 0.
- **Committed in:** `f21d2ba` (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (1 blocking, 1 bug)
**Impact on plan:** Both are small, contained, and necessary for the plan's own acceptance criteria to pass as literally written. No scope creep -- neither changes any owner-visible behavior beyond what the plan specifies.

## Issues Encountered

None beyond the deviations above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- All four ROADMAP Phase 2 success criteria's owner-visible mechanisms are now wired: badge re-sync after refresh (criterion 1), wheel-grab notice with the game still answering (criterion 2), the waiting/resume two-line sequences (criterion 3), and D-12's silent-when-off invariant (criterion 4). Screenshot evidence capture (`evidence/05-badge-after-refresh.png` through `evidence/12-hand-play-unchanged.png`) is explicitly out of this plan's scope per prior plans' summaries ("ready to be captured as staging evidence by plan 02-07") -- this plan built the mechanism, not the proof artifacts.
- The Phase 2 security review (WAITING-then-resume against policy section 2, plus the carried-forward R-02/R-04 items from Phase 1) remains open and is not this plan's concern -- flagged in `02-CONTEXT.md`'s deferred section for a later plan/gate.
- `go test ./...` shows only the pre-existing, excused `internal/icm` `TestHandlerRegistration/CANCEL` baseline failure -- no new failures from this plan (no Go files were touched).

## Self-Check: PASSED

- FOUND: frontend/src/context/SessionContext.tsx (modified)
- FOUND: frontend/src/components/AutopilotBadge.tsx (created)
- FOUND: frontend/src/components/Header.tsx (modified)
- FOUND: frontend/src/index.css (modified)
- FOUND: frontend/src/pages/PlayScreen.tsx (modified)
- FOUND: frontend/src/services/automation.ts (modified)
- FOUND: commit 7e6cb12 (Task 1)
- FOUND: commit f21d2ba (Task 2)
- FOUND: commit 7396ae3 (Task 3)
- FOUND: commit 2ab5e63 (public bundle rebuild)
- `cd frontend && npm ci` succeeded (fresh worktree, no node_modules)
- `cd frontend && npx tsc --noEmit` exit 0
- `cd frontend && npm run build` (tsc + vite build) exit 0, after every task and again as the final check -- deterministic output (same file hashes) on the repeat run, confirming a clean working tree
- No `test` script exists in `frontend/package.json` (only `dev`/`build`/`preview`) -- frontend verification for this phase is player-observable/screenshot per Phase 1's established precedent (02-RESEARCH.md), not a gap this plan needed to close
- `grep -c 'autopilot_state' frontend/src/context/SessionContext.tsx` = 1
- `grep -c 'wsManager.onAutopilot(' frontend/src/context/SessionContext.tsx` = 1; `grep -c 'offAutopilot(' frontend/src/context/SessionContext.tsx` = 1 (paired, same effect)
- `grep -c '\[Autopilot' frontend/src/context/SessionContext.tsx` = 0 (no owner-visible copy in the context)
- `grep -c 'setAutopilotControl' frontend/src/context/SessionContext.tsx` = 2
- `grep -c 'autopilot-badge.state-waiting' frontend/src/index.css` = 1
- `grep -c 'connectionState\|useState\|setInterval' frontend/src/components/AutopilotBadge.tsx` = 0 (D-10: derives nothing, polls nothing)
- Header.tsx: `<AutopilotBadge />` on the line after `<SessionBadge />` and before `<div className="account-menu">` (verified by line number)
- index.css contains `.autopilot-badge`, `.autopilot-badge-dot`, `.autopilot-badge.state-on`, `.autopilot-badge.state-waiting`, `.autopilot-badge.state-off`, all appended after `.session-badge.status-error`
- Badge labels are exactly `Autopilot: On`, `Autopilot: Waiting`, `Autopilot: Off`
- `grep -c "sendCommand(command + '\n')" frontend/src/pages/PlayScreen.tsx` (untagged) = 0 -- both remaining `sendCommand` calls carry an explicit source (`source` variable, or literal `'user'`)
- Direct-send branch carries the literal `'user'` tag and a comment naming D-07
- `grep -n "wsManager.onAutopilot(\|offAutopilot("` in PlayScreen.tsx shows the pair inside the same effect as `onMessage`/`onError`/`onDisconnect`
- Literal strings found character-for-character: `[Autopilot disengaged: you took the wheel]`, `[Autopilot waiting for reconnect]`, `[Reconnected]`, `[Autopilot resuming]`, each via `automationEngine.echoLocal`
- Transition effect has exactly two conditional branches calling `echoLocal` (on-to-waiting; waiting-to-on), with a first-render guard (`previous === undefined`) printing nothing
- `handleDisconnectRef.current` calls `refreshStatus()` after its existing `[Disconnected]` echo; that echo's wording/colour unchanged
- `git diff --stat go.mod frontend/package.json frontend/package-lock.json` empty (T-2-SC) -- no dependency added
- `go build ./...` exit 0
- `go test ./...` shows only the pre-existing, excused `internal/icm` `TestHandlerRegistration/CANCEL` baseline failure -- no new failures
- `git diff --diff-filter=D --name-only` empty for every task commit (7e6cb12, f21d2ba, 7396ae3, 2ab5e63) -- no unintended deletions
- Worktree HEAD assertion passed at spawn (branch `worktree-agent-aa950891efc64c241`, not a protected ref); base corrected via sanctioned `git reset --hard` to `a9e38e082a99f4f377ad8a87a518bf2eb12413cc` before any edits

---
*Phase: 02-autopilot-switch*
*Completed: 2026-09-15*
