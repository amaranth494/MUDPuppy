# Phase 2: Autopilot Switch - Context

**Gathered:** 2026-09-15
**Status:** Ready for planning

<domain>
## Phase Boundary

The owner can flip autopilot from the terminal with `#AUTO ON` / `#AUTO OFF` and always see the true state. The state is owned server-side per user session and has three readings: **ON** (entered only by `#AUTO ON`, after the Phase 1 engage gate passes), **OFF** (entered only by `#AUTO OFF`), and **WAITING** (an ON autopilot whose game connection has dropped; it issues nothing and resumes to ON by itself when the connection returns). The play screen shows a header badge that re-syncs from the server after a page refresh, plus one bracketed terminal line per state change. Taking the wheel is instant and lossless: any line the owner types that is not a `#` directive disengages autopilot before the line is sent. Commands fired by browser aliases-of-triggers, triggers, or timers do not. The AI never opens a connection. No AI intelligence exists yet; nothing is driven while ON.

**Owner amendment made during this discussion (2026-09-15):** the design previously said a disconnect lands on disengaged and the owner must re-engage. The owner replaced that with the WAITING model above. Design v3 D2 and Definition of complete item 6, REQ-no-auto-reconnect, REQ-safety-limits-hold, PROJECT.md (new locked decision DEC-autopilot-waits-across-disconnect), and ROADMAP Phase 2 and Phase 4 text were amended and committed (`c0a8319`). Downstream agents follow the amended text.

</domain>

<decisions>
## Implementation Decisions

### State model and the switch
- **D-01:** Autopilot has two owner-set positions and one derived one. `#AUTO ON` is the only way into ON; `#AUTO OFF` is the only way into OFF. A dropped connection moves ON to WAITING; the connection returning moves WAITING back to ON without any owner action. `#AUTO OFF` while WAITING lands on OFF and it stays OFF after reconnecting.
- **D-02:** `#AUTO ON` engages only when the profile passes the Phase 1 engage gate (`EngageGateAllowed` on the profile's policy acceptance columns). A refusal prints the gate's message from `store.EngageGateRefusalMessage` and the switch stays OFF.
- **D-03:** `#AUTO ON` with no connected game session is refused with a notice such as `[Autopilot needs a connected game; connect first]`. The switch stays OFF. There is no "arm before connecting".
- **D-04:** `#AUTO ON` while already ON and `#AUTO OFF` while already OFF are no-ops that print `[Autopilot is already on]` / `[Autopilot is already off]`. A repeated `#AUTO ON` must never restart or reset anything in later phases.
- **D-05:** `#AUTO` with no argument (and `#AUTO STATUS`) prints one line: the current state (ON, WAITING, or OFF) and the engage-gate result for the current profile. This is the phase's diagnostic surface; later phases may append call count and last decision. It is the Phase 1 deferred idea, landed here.
- **D-06:** The state is held server-side per user session (one session per user today) and is the single truth. A server restart loses it and lands on OFF; that is acceptable and is the safe direction.

### Wheel-grab boundaries
- **D-07:** Any line the owner types that is sent to the game takes the wheel: raw commands, alias expansions of a typed command (the owner typed it), command-history recall, and a blank Enter (a blank line is a client command and human interaction). Disengage happens before the line is sent; no keystroke is lost; the game's response to the typed command appears as usual.
- **D-08:** Lines beginning with `#` are client-internal commands and are ignored by the wheel-grab. They never disengage autopilot. This covers `#AUTO` itself (so `#AUTO ON` cannot toggle itself off) and every other directive such as `#ECHO`, `#HELP`, `#LOG`, `#IF`. Only `#AUTO OFF` turns autopilot off. (Owner's words: "# commands should not interrupt the AI; those internal commands should be ignored.")
- **D-09:** Commands fired by browser triggers and timers pass through with autopilot still ON. The websocket `data` message gains a source flag (human vs automation) so the server applies the wheel-grab only to human-sourced input; the browser's existing `CommandSource` tagging (`user`, `alias`, `trigger`) is the origin of that flag, with `user` and `alias` mapping to human.

### Indicator and notices
- **D-10:** The play-screen indicator is a header badge beside the existing connection badge, reading ON, WAITING, or OFF, driven by a server broadcast over the existing websocket so it re-syncs after a page refresh and never shows a state the server does not hold.
- **D-11:** Every state change also prints one short bracketed line in the terminal through the local-echo path (`#ECHO` style, never sent to the game), matching the existing `[Disconnected]` line. Wording set by the owner where given:
  - engage: `[Autopilot engaged]`
  - `#AUTO OFF`: `[Autopilot disengaged]`
  - wheel-grab: `[Autopilot disengaged: you took the wheel]`
  - refusal: the engage gate's message, or the no-connection notice from D-03
  - connection drops while ON: `[Disconnected]` then `[Autopilot waiting for reconnect]`
  - connection returns while WAITING: `[Reconnected]` then `[Autopilot resuming]`
  - already on/off: D-04 wording
- **D-12:** Ordinary play with autopilot OFF is unchanged: no new lines, no new behaviour, and the badge simply reads OFF. Phase 2 success criterion 4 (create profile, accept policy, hand-play normally) must hold with nothing else visible.

### Claude's Discretion
- Exact badge visuals, colours, and placement next to `SessionBadge`; the websocket message shape that carries the state (extend `status`, add a `sync` field, or a new `ai` message type); endpoint paths for engage/disengage/status; how the WAITING-to-ON resume is detected (the session manager's connect path vs the websocket `connect` handler); the Go state-machine package location and its tests; the source-flag field name on the `data` message and its default when absent (treat absent as human, the safe direction); log line format (`[AI-PLAYER]` prefix as in Phase 1); what a typed `#` directive that itself emits game commands does (treat its emitted commands as automation, consistent with D-08, unless it proves confusing in practice).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Product definition (amended 2026-09-15 for the WAITING model)
- `.specify/specs/ai-game-player-design-v3.md` — Deliverable D2 (autopilot switch) and Definition of complete items 1, 3, and 6. D2's third bullet and item 6 were amended today; read the current text, not memory of the old one.
- `.specify/specs/safety-and-abuse-policy-v1.md` — Sections 1 and 2 (permitted games, supervised operation). The WAITING-then-resume behaviour must be raised at the Phase 2 security review against section 2.

### Method
- `.specify/memory/phase-based-development-approach.md` — Binding Phase → Wave → Plan → Task rules.
- `CLAUDE.md` — Distilled method rules and project facts.

### Planning state
- `.planning/ROADMAP.md` §Phase 2 — Goal, success criteria (criterion 3 amended), Phase Validation line, implementation notes.
- `.planning/REQUIREMENTS.md` — REQ-autopilot-directives, REQ-wheel-grab, REQ-no-auto-reconnect (amended), REQ-doc-hand-play-and-gate.
- `.planning/PROJECT.md` — Locked decisions including the new DEC-autopilot-waits-across-disconnect, and CON-mechanical-halts (amended).
- `.planning/phases/01-profile-foundation-and-policy-gate/01-CONTEXT.md` — Phase 1 decisions D-05 (refused engage points to AI Player settings) and D-08 (the gate is the only thing Phase 2 needs).
- `.planning/phases/01-profile-foundation-and-policy-gate/01-SECURITY.md` and `.planning/todos/pending/2026-09-15-deferred-security-risks.md` — R-02 and R-04 must be re-raised at this phase's security review.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/store/profile.go`: `EngageGateAllowed(policyVersionAccepted, policyAcceptedAt)` and `EngageGateRefusalMessage`; `internal/profiles/handler.go` `GetEngageGate` (GET `/api/v1/profiles/{connection_id}/engage-gate`) and its `[AI-PLAYER] engage gate` log line. `#AUTO ON` calls the Go function directly server-side.
- `internal/session/manager.go`: one session per user; `Connect`, `Disconnect(userID, reason)` (single path for every disconnect reason: user, remote, error, idle, hard cap), `SendCommand`, `GetSession`. The WAITING transition hangs off `Disconnect`; the resume hangs off the connect path.
- `internal/session/websocket.go`: `WSMessage{Type, Host, Port, Data, Error, Status}` with types `connect`, `disconnect`, `data`, `error`, `status`; `handleClientCommands` is where every browser-sent `data` message becomes `manager.SendCommand`. The wheel-grab check and the source flag live here. `writeJSON` is the thread-safe writer for the state broadcast.
- `frontend/src/services/automation.ts`: `CommandSource` (`user`, `alias`, `trigger`, `cli`); `processUserInput` (typed input, alias expansion) and the trigger/timer paths all funnel into one `setSubmitCommandCallback`; `#` directives are parsed here (`parseCommandString`, the `segment.startsWith('#')` branch). `#AUTO` is a new directive in this grammar. `echoLocal` / `#ECHO` writes local lines.
- `frontend/src/pages/PlayScreen.tsx`: the submit callback (`wsManager.sendCommand(command + '\n')`) and the typed-input path; the `[Disconnected]` line is written here.
- `frontend/src/services/api.ts`: `sendCommand` builds the `{type: 'data', data}` message; the source flag is added here.
- `frontend/src/components/SessionBadge.tsx` and `Header.tsx`: the connection badge pattern the autopilot badge sits beside; `useSession()` context is the model for an autopilot-state context.

### Established Patterns
- Server owns truth; the browser reflects it. Connection state already flows server → browser as `status` messages; autopilot state follows the same direction.
- Local notices are bracketed lines written through the automation engine's echo path, never sent to the MUD.
- Structured `[AI-PLAYER]` log lines per decision (Phase 1) are the diagnostic evidence format; add one per state transition with user/session id, old state, new state, and cause.
- Only `internal/icm/icm_test.go` and the Phase 1 store/handler tests exist; the Phase 2 state machine and source-flag tests continue that.

### Integration Points
- New `#AUTO` handling in `automation.ts` calling new server endpoints (or websocket messages) for on/off/status.
- Source flag on the websocket `data` message (browser `api.ts` → server `websocket.go`).
- Wheel-grab in `handleClientCommands` before `manager.SendCommand`.
- WAITING/resume hooks in `manager.Disconnect` and the connect path; state broadcast to the browser on every transition and on websocket (re)attach for refresh re-sync.
- New badge in `Header.tsx` next to `SessionBadge`.
- Phase 3 will read this state to know when to drive; Phase 4's re-engage reassessment also applies to a WAITING → ON resume (fresh look at the game, no stale plan).

</code_context>

<specifics>
## Specific Ideas

- Owner's state model, verbatim intent: "`#AUTO ON` is a response to an off state. Off state can be engaged with `#AUTO OFF`." A dropped connection is not `#AUTO OFF`.
- Owner's wording for the drop and return: `[Disconnected]  [Autopilot waiting for Reconnect]` and `[Reconnected] [Autopilot resuming]`.
- Owner on `#` lines: "# commands should not interrupt the AI then. Those internal commands should be ignored."
- Owner on blank Enter: "A blank line is a Client command and should be considered human interaction and AI would disengage."
- The owner does not want plumbing turned into decisions; pick the simplest thing and move on (carried from Phase 1).

</specifics>

<deferred>
## Deferred Ideas

- **Security review item (this phase):** the WAITING-then-resume rule means autopilot can return to ON when a connection comes back while the owner is away from the screen. Raise it at the Phase 2 security review against policy section 2 with Accept / Defer / Remediate Now; a possible mitigation is a bounded WAITING lifetime after which it lands on OFF. Not decided here.
- Shelf copy: `D:\Projects\ai-mud-player\documents\ai-game-player-design-v3.md` is the owner's approved-version shelf and still holds the pre-amendment text. Copy the repo version there only when the owner says the amended version is approved.

### Reviewed Todos (not folded)
- `2026-09-15-deferred-security-risks.md` (R-02 invalid numeric input in the AI Player panel saves as blank; R-04 staging deploy log prints the one-time login code). Not Phase 2 build work; the todo's instruction is to re-raise both at the Phase 2 security review with Accept / Defer / Remediate Now. The project cannot close while either is deferred.

</deferred>

---

*Phase: 02-autopilot-switch*
*Context gathered: 2026-09-15*
