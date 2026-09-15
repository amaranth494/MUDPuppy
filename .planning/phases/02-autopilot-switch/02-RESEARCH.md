# Phase 2: Autopilot Switch - Research

**Researched:** 2026-09-15
**Domain:** Go backend (`internal/session` connection-lifecycle state machine, `net/http`/websocket message extension), TypeScript frontend (`#` directive grammar in `internal/automation`, React session context) — brownfield extension of two existing, working subsystems (Phase 1's engage-gate store functions; the pre-existing session/websocket/automation stack). No AI, no ICM wiring, no new external dependency.
**Confidence:** HIGH for the Go connect/disconnect single-path facts and the frontend directive-grammar extension point (all verified by direct code read this session); MEDIUM for the exact websocket message shape and multi-tab broadcast mechanism, which are genuine "Claude's Discretion" design choices with one clearly simplest option identified, not a pre-existing pattern to copy verbatim.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**State model and the switch**
- **D-01:** Autopilot has two owner-set positions and one derived one. `#AUTO ON` is the only way into ON; `#AUTO OFF` is the only way into OFF. A dropped connection moves ON to WAITING; the connection returning moves WAITING back to ON without any owner action. `#AUTO OFF` while WAITING lands on OFF and it stays OFF after reconnecting.
- **D-02:** `#AUTO ON` engages only when the profile passes the Phase 1 engage gate (`EngageGateAllowed` on the profile's policy acceptance columns). A refusal prints the gate's message from `store.EngageGateRefusalMessage` and the switch stays OFF.
- **D-03:** `#AUTO ON` with no connected game session is refused with a notice such as `[Autopilot needs a connected game; connect first]`. The switch stays OFF. There is no "arm before connecting".
- **D-04:** `#AUTO ON` while already ON and `#AUTO OFF` while already OFF are no-ops that print `[Autopilot is already on]` / `[Autopilot is already off]`. A repeated `#AUTO ON` must never restart or reset anything in later phases.
- **D-05:** `#AUTO` with no argument (and `#AUTO STATUS`) prints one line: the current state (ON, WAITING, or OFF) and the engage-gate result for the current profile. This is the phase's diagnostic surface; later phases may append call count and last decision. It is the Phase 1 deferred idea, landed here.
- **D-06:** The state is held server-side per user session (one session per user today) and is the single truth. A server restart loses it and lands on OFF; that is acceptable and is the safe direction.

**Wheel-grab boundaries**
- **D-07:** Any line the owner types that is sent to the game takes the wheel: raw commands, alias expansions of a typed command (the owner typed it), command-history recall, and a blank Enter (a blank line is a client command and human interaction). Disengage happens before the line is sent; no keystroke is lost; the game's response to the typed command appears as usual.
- **D-08:** Lines beginning with `#` are client-internal commands and are ignored by the wheel-grab. They never disengage autopilot. This covers `#AUTO` itself (so `#AUTO ON` cannot toggle itself off) and every other directive such as `#ECHO`, `#HELP`, `#LOG`, `#IF`. Only `#AUTO OFF` turns autopilot off.
- **D-09:** Commands fired by browser triggers and timers pass through with autopilot still ON. The websocket `data` message gains a source flag (human vs automation) so the server applies the wheel-grab only to human-sourced input; the browser's existing `CommandSource` tagging (`user`, `alias`, `trigger`) is the origin of that flag, with `user` and `alias` mapping to human.

**Indicator and notices**
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

### Deferred Ideas (OUT OF SCOPE)
- **Security review item (this phase, must be raised, not resolved by research):** the WAITING-then-resume rule means autopilot can return to ON when a connection comes back while the owner is away. Raise at the Phase 2 security review against policy section 2 with Accept / Defer / Remediate Now; a possible mitigation is a bounded WAITING lifetime after which it lands on OFF. Not decided here — see Security Domain below.
- `D:\Projects\ai-mud-player\documents\ai-game-player-design-v3.md` shelf copy still holds pre-amendment text; not touched by this phase.
- Carried from Phase 1, MUST be re-raised at this phase's security review with Accept/Defer/Remediate Now (project cannot close while deferred): **R-02** (`AIPlayerPanel.tsx` numeric fields silently save as blank on invalid input) and **R-04** (staging deploy log prints the one-time OTP code).
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| REQ-autopilot-directives | `#AUTO ON`/`#AUTO OFF` in the `#` grammar, plus a visible status indicator; engage/disengage independent of AI. Acceptance: engages only when the D1 gate passes; indicator always matches true state. | `Pattern 1` (CommandRegistry + evaluator switch extension point, exact file/line verified) gives the directive-grammar mechanism. `Pattern 3` (state ownership on `Manager`, not `Session`) and `Pattern 4` (REST endpoint + `HandlerCallbacks` wiring) give the engage/disengage/status mechanism, reusing `store.EngageGateAllowed`/`EngageGateRefusalMessage` verbatim from Phase 1. `Pitfall 1` is the single most important finding for this requirement: state must not live on the `Session` struct. |
| REQ-wheel-grab | Any typed game command disengages before send, no lost keystrokes; alias/trigger/timer-fired commands do not trip it. | `Pattern 2` (single command choke point: `case MsgTypeData` in `websocket.go`) and `Pattern 5` (frontend `CommandSource` threading from `queueCommand`/`submitCommand` through to the `data` message) give the exact, minimal-touch wiring. `Common Pitfalls` #2 and #3 cover the two ways this could silently misfire (blank-Enter bypass path, default-source-on-absence). |
| REQ-no-auto-reconnect | Disconnect while engaged enters WAITING, issues nothing, resumes on its own on reconnect; AI never reconnects; `#AUTO OFF` while WAITING lands on OFF and stays OFF. | `Pattern 3`/`Pitfall 1` (state must survive `Manager.Disconnect`'s session-map deletion) is the load-bearing finding. `Pattern 4` traces both the REST and websocket connect entry points to the single `Manager.Connect` hook point, and confirms `Manager.Disconnect` is the single hook for every disconnect reason (verified exhaustively against every `ReasonX` constant and call site). |
| REQ-doc-hand-play-and-gate | Owner can create a profile, accept policy, hand-play with AI disengaged; nothing about ordinary play changes; AI cannot engage without acceptance. | Already built by Phase 1 (`EngageGateAllowed`, `AIPlayerPanel.tsx`); this phase's only obligation is D-12 (no regression) plus the gate call in `#AUTO ON`, both covered by `Pattern 4`. |
</phase_requirements>

## Summary

Phase 2 is a state-machine-and-wiring phase, not a new-subsystem phase: every piece it needs already exists and works (Phase 1's engage gate, the session manager's single `Connect`/`Disconnect`/`SendCommand` choke points, the websocket's single `data`-message ingress, and the frontend's `#` directive grammar with its `CommandRegistry`/evaluator-switch extension pattern already used four times over for `ECHO`/`LOG`/`HELP`/`TIMER`). The phase's real difficulty is not "where do I add code" — every extension point is a verified, single, unambiguous location — it is **getting the WAITING state's survival correct**: `internal/session/manager.go`'s `Disconnect` deletes the user's `*Session` from its map on every disconnect (`delete(m.sessions, userID)`, verified at `manager.go:330`), and `GetSession` fabricates a fresh zero-value `Session` for any missing entry. If autopilot state is naively added as a field on `Session`, WAITING is destroyed the instant the connection drops — the exact moment the phase's central new behaviour needs it to persist. The correct design keeps autopilot state in a **separate map on `Manager`**, keyed by `userID`, independent of the `sessions`/`conns` maps' lifecycle, so it survives the disconnect-triggered deletion and is read back by `Connect` to resume.

The second material finding is that **no cross-connection broadcast mechanism exists today**: each `HandleWebSocket` invocation holds its own `*websocket.Conn` in local closures with no per-user registry, so there is no existing way to push a message to "all of user X's open browser tabs." Phase 2 should not build one. Instead: state changes triggered by the *acting* tab (typing `#AUTO ON`, typing a command that trips the wheel-grab) are already known to that tab synchronously (the HTTP response, or the same websocket connection that sent the triggering message); the connection-drop→WAITING and reconnect→ON transitions happen inside code paths (`HandleWebSocket`'s own main loop, and REST `session.Handler.Connect`) that either already hold a `conn` in scope or are the same request/response cycle the acting tab is waiting on. The pre-existing, already-working precedent for "badge always correct after a page refresh" is **not** a websocket-attach message at all — it is `SessionBadge.tsx`'s 15-second poll plus on-mount/visibility-change calls to `GET /api/v1/session/status` (verified in `SessionContext.tsx` and `SessionBadge.tsx`) — extending that same `StatusResponse` with the autopilot field is the simplest, lowest-risk way to satisfy "the badge never shows a state the server does not hold," and a best-effort live websocket push (piggybacked wherever a `conn` is already in scope) is the responsiveness layer on top, not the correctness mechanism.

**Primary recommendation:** Add autopilot state to `internal/session.Manager` as a new `map[string]*AutopilotState` field guarded by the existing `m.mu`, independent of `sessions`; hook the transition into `Manager.Connect` (WAITING→ON) and `Manager.Disconnect` (ON→WAITING) directly, since both are already the single, exhaustively-used choke points for every connect/disconnect reason in the codebase; add one new field to `WSMessage` (`Source string` on inbound `data` messages, read in the existing `case MsgTypeData:` block before the message reaches `clientToMUD`) for the wheel-grab, and one new outbound message type (`autopilot`) reusing the `Status` field's string-carrying convention for best-effort live push; extend `StatusResponse`/`GET /api/v1/session/status` with the autopilot field as the authoritative refresh-correctness path; add `#AUTO` to `frontend/src/services/automation/commands.ts`'s `CommandRegistry` and a `case 'AUTO':` in `evaluator.ts`'s `executeTokenList` switch, following the `HELP`/`ECHO` precedent exactly, threading a new optional `autopilotControl` callback through `executeAutomationAction`/`ExecutionContext` the same way `helpResolver` was added; thread `CommandSource` from `ProcessedCommand`/`queueCommand` through the `setSubmitCommandCallback` signature and into `wsManager.sendCommand` so the human/automation distinction reaches the `data` message without touching every call site.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Autopilot ON/OFF/WAITING state | API / Backend (`internal/session.Manager`, new map keyed by userID) | — | D-06: single server-held truth, survives across browser tabs and page refresh; must NOT live on the `Session` struct (see Pitfall 1) |
| Engage-gate decision on `#AUTO ON` | API / Backend (reuses `store.EngageGateAllowed`/`EngageGateRefusalMessage`, unchanged from Phase 1) | Database / Storage (`profiles` row, already migrated) | Phase 1 already built and proved this; Phase 2 only calls it |
| Wheel-grab decision (human vs automation source) | API / Backend (`internal/session/websocket.go`, `case MsgTypeData`) | Browser / Client (tags the source at origin) | The server is the enforcement point (a malicious/buggy client cannot self-report "automation" to bypass the grab in a way the server can't at least default-safely reject); the browser is where the distinction is first known |
| `#AUTO` directive parsing and local notice lines | Browser / Client (`frontend/src/services/automation/{commands,evaluator}.ts`) | — | Matches every existing `#` directive (`ECHO`, `LOG`, `HELP`, `TIMER`); D-11's bracketed lines go through the same `echoLocal`/`#ECHO` path already used for `[Disconnected]` |
| Badge display and refresh-correctness | Browser / Client (`SessionContext.tsx` polling + `Header.tsx`/new badge component) | API / Backend (`GET /api/v1/session/status`, extended) | Exact existing pattern for the connection badge (`SessionBadge.tsx`); no new mechanism needed for correctness, only a new field |
| Live (sub-15s) badge push | API / Backend (best-effort, piggybacked on existing `conn` in scope) | Browser / Client (new `onAutopilot`/message-type handler) | No per-user connection registry exists; building one is scope creep for this phase — see Summary |
| Disconnect → WAITING, reconnect → ON hook | API / Backend (`internal/session.Manager.Disconnect` / `.Connect`) | — | Both are the exhaustive, single choke points for every disconnect reason and every connect entry point in the existing codebase (verified) |

## Standard Stack

No new external dependency in either `go.mod` or `frontend/package.json`. Everything this phase needs is already present and already used by the exact subsystems it touches:

| Library | Version (from go.mod / package.json) | Purpose | Why Standard |
|---------|------|---------|--------------|
| `github.com/gorilla/websocket` | `v1.5.3` [VERIFIED: go.mod, already imported by `internal/session/websocket.go`] | Websocket transport carrying `data`/`status`/new `autopilot` messages | Already the only websocket library in the codebase |
| `database/sql` + `github.com/lib/pq` | `v1.11.2` [VERIFIED: go.mod] | Reading `profiles.policy_version_accepted`/`policy_accepted_at` via `store.EngageGateAllowed` | Unchanged from Phase 1; no new query shape needed, the gate is a pure function over two already-fetched values |
| `sync.RWMutex` (stdlib) | Go 1.26 stdlib | Guarding the new per-user autopilot map on `Manager`, exactly like the existing `sessions`/`conns`/`cleanups` maps | Matches `Manager`'s existing concurrency pattern (`m.mu sync.RWMutex` already guards three parallel maps; a fourth is the same pattern, not a new one) |
| `net/http.ServeMux` (stdlib) | Go 1.26 stdlib | New route(s) for `#AUTO`'s server call(s) | Already the only router; Phase 1 registered five routes this exact way |
| React state/context (existing) | package.json, unchanged | New badge component, autopilot-state field in `SessionContext` | Matches `SessionBadge.tsx`/`useSession()` exactly |

No new packages needed on either side. `frontend/package.json`'s `"test": "echo 'No tests yet' && exit 0"` [VERIFIED: read this session, unchanged since Phase 1] confirms frontend test tooling is still absent — this phase's frontend verification is player-observable/screenshot, matching Phase 1's precedent exactly, not a gap Phase 2 needs to close.

**Installation:** none required.

## Package Legitimacy Audit

**Not applicable.** This phase installs zero new packages in `go.mod` or `frontend/package.json`. Per the Package Legitimacy Gate's own instructions ("Required whenever this phase installs external packages"), the audit is skipped — identical situation to Phase 1.

## Architecture Patterns

### System Architecture Diagram

```
Browser: typed "#AUTO ON" / "#AUTO OFF" / "#AUTO" / "#AUTO STATUS"
   │
   ▼
frontend/src/services/automation/evaluator.ts
  executeTokenList() switch — new `case 'AUTO':`
  (registered in commands.ts CommandRegistry, same as ECHO/LOG/HELP/TIMER)
   │  calls a new autopilotControl callback (threaded through
   │  executeAutomationAction, mirrors the helpResolver pattern)
   ▼
POST /api/v1/session/autopilot  { action: "on"|"off"|"status", connection_id }
   │  (new session.Handler method, alongside existing Connect/Disconnect/Status;
   │   needs a profile-gate check — wired via a new HandlerCallbacks entry,
   │   same injection pattern already used for OnConnected/GetAutoLogin)
   ▼
internal/session.Manager
  new: autopilot map[string]*AutopilotState   (guarded by existing m.mu,
       independent of the sessions/conns/cleanups maps' lifecycle)
  EngageAutopilot(userID) -> calls store.EngageGateAllowed via callback,
       checks GetSession(userID).State == StateConnected (D-03),
       sets autopilot[userID] = ON
  DisengageAutopilot(userID) -> sets autopilot[userID] = OFF (always allowed, D-01)
   │
   ▼ (HTTP response carries the new state + message back to the acting tab)

Separately, the wheel-grab and WAITING/resume hooks:

Browser: typed command (human)          Browser: trigger/timer fires (automation)
   │  submitCommand() -> automationEngine    │  fireTrigger()/timer callback ->
   │  .processUserInput() -> queueCommand    │  queueCommand({source:'trigger'})
   │  ({source:'user'|'alias'})              │
   └──────────────┬───────────────────────────┘
                   ▼
     onSubmitCommand(command, source)  (NEW: source param added to the
     existing setSubmitCommandCallback signature, PlayScreen.tsx passes
     it straight through to wsManager.sendCommand(command, source))
                   ▼
     WebSocket data message: {type:'data', data, source:'user'|'alias'|'trigger'}
                   ▼
internal/session/websocket.go  HandleWebSocket()  case MsgTypeData:
  NEW: read wsMsg.Source; if human (absent/"user"/"alias") AND
  manager.IsAutopilotOn(userID): manager.DisengageAutopilot(userID, "wheel-grab"),
  push a best-effort autopilot notice down this same conn, THEN queue to
  clientToMUD exactly as today — ordering is guaranteed by single-goroutine
  program order, no race (see Pitfall 4)
                   ▼
             clientToMUD channel -> handleClientCommands() -> manager.SendCommand()
                   ▼
                MUD server

Manager.Disconnect(userID, reason)  <- every disconnect reason funnels here
  NEW: if autopilot[userID] == ON, set to WAITING (state map survives the
  `delete(m.sessions, userID)` a few lines later because it's a separate map)

Manager.Connect(ctx, userID, host, port)  <- both REST /session/connect and
  the websocket "connect" message's fresh-dial branch funnel here
  NEW: if autopilot[userID] == WAITING, set to ON (resume)

Badge correctness (page refresh):
  GET /api/v1/session/status  (existing endpoint, extended)
     StatusResponse gains AutopilotState string
  SessionContext.tsx refreshStatus() (existing, called on mount/visibility/
     15s interval/every connect-disconnect action) -> setAutopilotState
     — this is the actual refresh-correctness mechanism, not a websocket push
```

### Recommended Project Structure

```
internal/
├── session/
│   ├── manager.go       # extended: autopilot map, EngageAutopilot/DisengageAutopilot/
│   │                     #   autopilotState methods, hooks in Connect/Disconnect
│   ├── autopilot.go      # NEW — AutopilotState type + pure transition functions,
│   │                     #   zero-fixture unit-testable (mirrors Phase 1 Pattern 4)
│   ├── autopilot_test.go # NEW — go test proves ON/OFF/WAITING transitions and
│   │                     #   the human/automation source-flag rule (Phase Validation line)
│   ├── handler.go        # extended: Autopilot(w,r) POST handler, alongside
│   │                     #   Connect/Disconnect/Status; HandlerCallbacks gains
│   │                     #   EngageGate func(connectionID, userID) (bool, string)
│   └── websocket.go      # extended: WSMessage gains Source (inbound) and a new
│                          #   MsgTypeAutopilot outbound type; case MsgTypeData gains
│                          #   the wheel-grab check
└── store/
    └── profile.go         # UNCHANGED — EngageGateAllowed/EngageGateRefusalMessage
                            #   already exist from Phase 1, reused as-is

frontend/src/
├── services/
│   ├── automation/
│   │   ├── commands.ts    # extended: 'AUTO' entry in CommandRegistry
│   │   └── evaluator.ts   # extended: case 'AUTO' in executeTokenList; new
│   │                       #   autopilotControl param threaded through
│   │                       #   executeTokens/executeAutomationAction (mirrors
│   │                       #   the helpResolver precedent, PR02PH09)
│   ├── automation.ts       # extended: setSubmitCommandCallback signature gains
│   │                       #   source param; queueCommand/processCommandQueue
│   │                       #   pass cmd.source through
│   └── api.ts               # extended: sendCommand(command, source) includes
│                             #   source in the data message; WebSocketManager
│                             #   gains onAutopilot/offAutopilot handler list;
│                             #   getSessionStatus's SessionStatus type gains
│                             #   autopilot_state
├── components/
│   ├── SessionBadge.tsx     # UNCHANGED — reference pattern only
│   └── AutopilotBadge.tsx    # NEW — same shape as SessionBadge.tsx, reads
│                              #   autopilotState from useSession()
├── pages/
│   └── PlayScreen.tsx        # extended: setSubmitCommandCallback call site
│                              #   passes source through; wheel-grab notices
│                              #   arrive via the same echoLocal path as
│                              #   [Disconnected] already does
└── context/
    └── SessionContext.tsx    # extended: autopilotState field, populated by
                               #   refreshStatus() (extended StatusResponse)
                               #   and by the new onAutopilot websocket handler
```

### Pattern 1: `#` directive registration (existing, four-times-proven, to be replicated for `AUTO`)

**What:** A new directive requires exactly two edits: an entry in `CommandRegistry` (`frontend/src/services/automation/commands.ts`) so the parser recognizes `#AUTO` as a `COMMAND` token rather than falling through to plain text, and a `case 'AUTO':` in the `executeTokenList` switch (`frontend/src/services/automation/evaluator.ts`, the same switch that already handles `IF`/`SET`/`TIMER`/`ECHO`/`LOG`/`HELP`/`ELSE`/`ENDIF`).

**Verified existing registry entry (the template):**
```typescript
// Source: frontend/src/services/automation/commands.ts (verbatim)
'HELP': {
  name: 'HELP',
  category: CommandCategories.OUTPUT,
  requiresArgs: false,
  description: 'Show help',
},
```
`AUTO` should be `requiresArgs: false` (bare `#AUTO` is valid per D-05) with its own category (e.g. add `AUTOPILOT: 'autopilot'` to `CommandCategories`, or reuse `OUTPUT` — a genuine, low-stakes discretion call).

**Verified existing switch-case shape (the template, `HELP`, which is also side-effect-only and never sends a MUD command):**
```typescript
// Source: frontend/src/services/automation/evaluator.ts (verbatim)
case 'HELP':
  // Handle #HELP command - show help
  {
    const helpText = 'Use the Help menu in the sidebar for documentation';
    const brightYellow = '\x1b[93m';
    const reset = '\x1b[0m';
    context.outputMessage?.(`\r\n${brightYellow}${helpText}${reset}\r\n`);
  }
  i++;
  continue;
```
`case 'AUTO':` follows this exact shape but is `async` work (an HTTP call to the new endpoint) — since `executeTokenList` is already `async` (it `await`s inside the `IF`/`SET` cases), an `await context.autopilotControl?.setState(...)` call is a direct, precedented extension, not a new pattern.

**Threading the new callback (mirrors the `helpResolver` precedent exactly — verified, not speculative):** `helpResolver` was added to `ExecutionContext`, `executeTokens`, and `executeAutomationAction` as a new trailing optional parameter in a prior PR (PR02PH09, still in the current code). `autopilotControl` should be added the same way, one parameter after `source`:
```typescript
// Source: frontend/src/services/automation/evaluator.ts (verbatim signature, current)
export async function executeAutomationAction(
  actionText: string,
  variables: VariableStore,
  timerManager?: TimerManager,
  aliasResolver?: (aliasName: string) => Promise<string[]>,
  outputMessage?: (message: string) => void,
  helpResolver?: (topic?: string) => Promise<{...} | null>,
  source?: 'cli' | 'trigger' | 'alias' | 'timer'
  // NEW trailing param goes here: autopilotControl?: { ... }
): Promise<ExecutionResult>
```
Both call sites in `automation.ts` (`processUserInput`, lines ~512 and ~558, both already passing `'cli'` as `source`) need the new argument added — a two-line diff, not a redesign.

### Pattern 2: The single command-ingress choke point for the wheel-grab (existing, verified)

**What:** Every browser-typed command that reaches the MUD passes through exactly one place server-side: `case MsgTypeData:` inside `HandleWebSocket`'s main `for` loop (`internal/session/websocket.go`, lines 344-374, verified this session). This is true regardless of whether the command originated from `submitCommand`'s automation-engine path or its direct-send (blank command / no engine) path, because both converge on the browser's single `wsManager.sendCommand(...)` call before crossing the network.

**Verified existing code (the exact insertion point):**
```go
// Source: internal/session/websocket.go (verbatim, case MsgTypeData)
case MsgTypeData:
    // Rate limiting at WebSocket ingress (SP02PH02T04)
    rl := h.getRateLimiter(userIDStr)
    if !rl.Allow() { ... }
    // Message size enforcement (SP02PH02T03)
    if len(wsMsg.Data) > h.config.MaxMessageSizeBytes { ... }
    if !connected {
        h.sendError(conn, "Not connected")
        continue
    }
    // NEW: wheel-grab check goes here, before the channel send below —
    // wsMsg.Source is now available on the parsed WSMessage
    select {
    case clientToMUD <- wsMsg.Data:
        metrics.Get().IncWSMessagesOut()
    default:
        h.sendError(conn, "Command queue full")
    }
```
**No race is possible:** this entire block executes serially within `HandleWebSocket`'s single main loop for one websocket connection — no other goroutine reads or mutates `wsMsg` between JSON-unmarshal and the channel send. The disengage call (a synchronous `manager.DisengageAutopilot(userID, ...)` guarded by `m.mu`) completes in program order strictly before the line is pushed onto `clientToMUD`, and `clientToMUD` is itself a single-consumer FIFO channel (`handleClientCommands` goroutine), so command ordering across rapid keystrokes is also preserved.

### Pattern 3: Autopilot state must live independent of `Session`, not on it (the phase's central pitfall, elevated to a pattern)

**What goes wrong if ignored:** `Manager.Disconnect` (`internal/session/manager.go:302-342`, verified) does `delete(m.sessions, userID)` unconditionally for every disconnect reason. `Manager.GetSession` (`manager.go:344-358`, verified) returns a **freshly constructed zero-value** `&Session{UserID: userID, State: StateDisconnected}` whenever no map entry exists. If `AutopilotState` were a field added to the `Session` struct, the very act of disconnecting (the trigger for WAITING) would delete the struct carrying that state one statement later, and the next `GetSession`/`Connect` call would see a brand-new zero-value struct — WAITING would never be observable.

**The correct shape (new, but following the `Manager`'s own established multi-map pattern):**
```go
// Source: internal/session/manager.go (verbatim, the pattern being extended)
type Manager struct {
    ...
    mu       sync.RWMutex
    sessions map[string]*Session // userID -> session
    conns    map[string]net.Conn
    cleanups map[string]context.CancelFunc
    // NEW, same shape, same mutex, independent lifecycle:
    // autopilot map[string]*AutopilotState // userID -> autopilot state
}
```
`Manager` already carries three parallel `userID`-keyed maps guarded by one `sync.RWMutex`; a fourth, independently-lived map is not a new concurrency pattern, it is the existing one applied once more. `Connect` and `Disconnect` already are the two places (verified, see Pattern 4) where every session lifecycle transition happens — adding four lines to each (read-and-resume in `Connect`, read-and-wait in `Disconnect`) is the entire wiring cost, once the map is separate.

**Server restart (D-06):** an in-memory map is lost on restart exactly like `sessions`/`conns` already are today — no extra work needed to satisfy "a server restart loses it and lands on OFF."

### Pattern 4: Single connect/disconnect choke points (existing, verified exhaustively)

**Disconnect — every reason funnels through one function.** Grepping every `Disconnect(` call site and every `Reason*` constant in the package confirms all eight disconnect reasons (`ReasonUser`, `ReasonIdle` [currently disabled per a code comment], `ReasonHardCap`, `ReasonRemote`, `ReasonError`, `ReasonSlowClient`, `ReasonRateLimit`, `ReasonProtocolMismatch`) call `Manager.Disconnect(userID, reason string)` and nothing else terminates a session. This is the single hook point for ON→WAITING.

**Connect — two HTTP-level entry points, one low-level function.**
1. `session.Handler.Connect` (`internal/session/handler.go:80`, `POST /api/v1/session/connect`) calls `h.manager.Connect(r.Context(), userIDStr, req.Host, req.Port)` directly.
2. `WebSocketHandler.HandleWebSocket`'s `case MsgTypeConnect:` (`websocket.go:266-321`) either (a) reuses an already-`StateConnected` session via `GetSession` with **no** call to `Manager.Connect` — this is the page-refresh-while-still-connected reattach path, not a new connection — or (b) calls `h.manager.Connect(ctx, userIDStr, wsMsg.Host, wsMsg.Port)` itself for a websocket-only connect attempt (a legacy/alternate path; the frontend's actual `connect()` flow, verified in `SessionContext.tsx`, always calls the REST endpoint first and only opens the websocket afterward, sending a `connect` message that hits branch (a)).

**Verified frontend connect flow** (`SessionContext.tsx`, `connect()` callback): `await connectToMud(connectRequest)` (REST) happens first, *then* a new `WebSocketManager` is created and `manager.sendConnect(mudHost, mudPort)` is sent — meaning in the normal product flow, `Manager.Connect` is called exactly once, from the REST handler, and the websocket's own `connect` message reuses that session. **This means the WAITING→ON resume hook belongs in `Manager.Connect` itself** (reached via the REST path when the owner clicks Connect/reconnects), not in the websocket message-loop's reuse branch — the reuse branch by definition means the connection never actually dropped from the server's perspective, so there is nothing to resume.

### Pattern 5: Threading `CommandSource` to the outbound `data` message without touching every call site

**What:** `ProcessedCommand.source` (`'user' | 'alias' | 'trigger'`, `frontend/src/services/automation.ts:20`, verified) already exists and is already correctly assigned at every command-origination point (typed input → `'user'`, alias expansion → `'alias'`, trigger fire and timer fire → `'trigger'`, verified at four separate call sites: `processUserInput`, `fireTrigger`, and both `TimerManager` callback registrations). The loss point is `processCommandQueue` (`automation.ts`, private method), which calls `this.onSubmitCommand(cmd.command)` — **dropping `cmd.source`** — and `PlayScreen.tsx`'s registration of that same callback (`automationEngine.setSubmitCommandCallback((command: string) => { wsManager.sendCommand(command + '\n'); ... })`), which also only accepts a bare string.

**Minimal-touch fix (two functions, not every call site):**
1. `setSubmitCommandCallback(callback: (command: string, source: CommandSource) => void)` — signature gains one parameter.
2. `processCommandQueue`'s single call site: `this.onSubmitCommand(cmd.command, cmd.source)`.
3. `PlayScreen.tsx`'s single registration: `automationEngine.setSubmitCommandCallback((command, source) => { wsManager.sendCommand(command + '\n', source); ... })`.
4. `api.ts`'s `sendCommand(command: string, source?: CommandSource)` includes `source` in the JSON payload: `{type: 'data', data: command, source}`.
5. `submitCommand`'s own direct-send branches in `PlayScreen.tsx` (the blank-command / no-automation-engine paths, `submitCommand` lines ~407-434) are **always** human-typed by construction (they only fire for the human-facing `submitCommand` function itself) — call `wsManager.sendCommand(command + '\n', 'user')` explicitly there.

This touches five call sites total, all already enumerated by name — no broad refactor, and it exactly answers "can `CommandSource` be threaded to the `data` message without touching every call site" (yes: the queue and the one callback registration are the only chokepoints).

### Anti-Patterns to Avoid

- **Storing `AutopilotState` as a field on `Session`.** See Pattern 3 — this is the single most consequential design mistake available in this phase, because it would compile, pass a naive unit test of the transition logic in isolation, and only fail the moment a real disconnect happens, which is exactly the scenario the phase exists to prove.
- **Building a new per-user websocket connection registry for broadcast.** No such registry exists today (verified: `WebSocketHandler` holds no `map[string][]*websocket.Conn` or equivalent); adding one is a meaningful new piece of concurrent infrastructure for a phase whose CONTEXT.md explicitly says "the owner does not want plumbing turned into decisions; pick the simplest thing and move on." The REST poll (`GET /api/v1/session/status`, already 15s-cadenced) is suffient for correctness; best-effort pushes reuse `conn` objects already in scope.
- **Re-deriving the engage-gate logic instead of calling `store.EngageGateAllowed`/`store.EngageGateRefusalMessage` verbatim.** Both already exist, are already unit-tested, and their exact wording is a locked Phase 2 contract per Phase 1's own doc comment ("This exact string is the Phase 2 contract").
- **Checking `#AUTO`'s source/gate logic inside the frontend.** D-02/D-03's refusal conditions (policy gate, no connected session) must be authoritative server-side checks, not frontend guesses — the frontend only renders whatever message the server returns, exactly like `AIPlayerPanel.tsx` already does for the Phase 1 gate.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Engage-gate decision | A second policy-acceptance check inside `session` package | `store.EngageGateAllowed(policyVersionAccepted, policyAcceptedAt *string) bool` + `store.EngageGateRefusalMessage` (both already exist, `internal/store/profile.go:575-582`) | Already built, unit-tested, and named as the Phase 2 contract in its own doc comment |
| Local bracketed notice lines (`[Autopilot engaged]`, etc.) | A new terminal-write path bypassing the automation engine | `automationEngine.echoLocal(message, {color})`, the exact mechanism already producing `[Disconnected]` (`PlayScreen.tsx:313`, verified) and `[ERROR]` (`PlayScreen.tsx:305`) | Guarantees the same "never sent to the MUD" guarantee D-11 requires, for free, with zero new code |
| `#` directive parsing/dispatch | A second parser or a regex-based `#AUTO` special-case outside the evaluator | `CommandRegistry` + `executeTokenList`'s switch (Pattern 1) | Four other directives already prove this scales; a bespoke path would fork the grammar |
| Per-user concurrent-map safety for the new autopilot state | A new mutex or a sync.Map | The existing `Manager.mu sync.RWMutex`, already guarding three parallel maps | One more map under the same lock is zero new concurrency surface; a second lock risks a new deadlock ordering |

**Key insight:** every piece of this phase's mechanism is either "the same four-times-proven `#` directive pattern" or "the same single choke-point that already exists for connect/disconnect/command-ingress" — the phase's actual engineering risk is entirely in *not* introducing a new, parallel, incorrect version of a mechanism that already exists correctly.

## Common Pitfalls

### Pitfall 1: Storing autopilot state on `Session` (see Pattern 3 for the full mechanism)
**What goes wrong:** WAITING is deleted the instant it's created, because `Disconnect` deletes the `Session` map entry a few lines after any `Session`-embedded state would be set to WAITING.
**Why it happens:** `Session` looks like the obvious place for "session-scoped" state, and the struct is right there with `State string` already on it.
**How to avoid:** A separate `map[string]*AutopilotState` on `Manager`, independent of `sessions`.
**Warning signs:** A `go test` that constructs a `Session` directly and asserts WAITING would pass while the real `Disconnect`→`Connect` cycle silently loses the state — write the test against `Manager.Disconnect`/`Manager.Connect` end-to-end, not against a bare `Session` struct.

### Pitfall 2: Blank Enter bypassing the wheel-grab because it bypasses the automation engine
**What goes wrong:** `PlayScreen.tsx`'s `submitCommand` explicitly does **not** trim input ("Don't trim - allow blank lines for MUDs") and routes a blank command around `automationEngine.processUserInput` entirely (`if (automationEngine && command !== '' && !automationDisabled)` — the `command !== ''` guard sends blank Enter straight to `wsManager.sendCommand` in the `else` branch). D-07 explicitly requires blank Enter to take the wheel. If the wheel-grab's source-tagging is only added inside the automation-engine path (Pattern 1/5), a blank Enter would reach the server with no source tag at all.
**Why it happens:** The direct-send branch exists specifically to bypass automation processing for a reason unrelated to autopilot (some MUDs need a bare newline); it predates this phase.
**How to avoid:** Tag the direct-send branch's `wsManager.sendCommand` call with `'user'` explicitly (Pattern 5, point 5) — it is unconditionally human by construction, since it only fires inside `submitCommand`, which is only called from typed input and keybindings.
**Warning signs:** Pressing Enter on an empty input line while autopilot is ON does not disengage it.

### Pitfall 3: Treating an absent/malformed source field as automation instead of human
**What goes wrong:** If the server-side default for a missing `Source` field on an inbound `data` message is "automation" (i.e., skip the wheel-grab), any client bug, version skew, or malicious client that omits the field entirely engages the AI's protection bypass — a human command would silently not take the wheel.
**Why it happens:** It's easy to write `if wsMsg.Source == "automation" { skip }` (opt-out) instead of `if wsMsg.Source == "trigger" { skip }` (opt-in) — the two look symmetric but fail differently on a missing/garbled field.
**How to avoid:** CONTEXT.md's own discretion note is explicit: "its default when absent (treat absent as human, the safe direction)." Implement as an allowlist: only `"trigger"` (and possibly `"alias"`/`"user"` are just the two human labels already used interchangeably per D-09) skips the grab; anything else, including empty string, is human.
**Warning signs:** A frontend build that forgets to send the new field (e.g., a stale cached bundle) causes triggers/timers to stop working (grabbed on every fire) rather than causing the wheel-grab to silently stop working — verify the fail-safe direction is the annoying one, not the dangerous one.

### Pitfall 4: Assuming the REST `#AUTO ON` endpoint and the websocket wheel-grab share one mutex acquisition
**What goes wrong:** `Manager.Connect`, `Manager.Disconnect`, and the new autopilot-map read/write in `HandleWebSocket`'s `case MsgTypeData:` all touch the same `Manager.mu`. If the new autopilot methods (`EngageAutopilot`, `DisengageAutopilot`, a state-read helper) are added without taking `m.mu` (e.g., because they're implemented as bare map access inline in `websocket.go` instead of as `Manager` methods), a data race exists between the websocket goroutine's read and the REST handler's write, even though neither one individually looks unsafe.
**Why it happens:** `Connect`/`Disconnect` already take the lock as their first statement (`m.mu.Lock(); defer m.mu.Unlock()`), but a developer adding a "quick" state check inside `websocket.go`'s existing loop (which does not hold `m.mu`) might reach for `m.sessions[userID].SomeField` directly out of habit, forgetting the new autopilot map needs the same discipline.
**How to avoid:** Implement all autopilot state access as `Manager` methods (`(m *Manager) AutopilotState(userID string) AutopilotState`, `(m *Manager) EngageAutopilot(...)`, `(m *Manager) DisengageAutopilot(...)`) that take `m.mu` internally, exactly like `GetSession`/`SendCommand` already do — never inline map access from `websocket.go`.
**Warning signs:** `go test -race ./internal/session/...` (recommended addition to the Wave 0 test command, since no existing test in this package currently runs under `-race`).

## Code Examples

### Existing single-choke-point disconnect (verbatim, the hook for ON→WAITING)
```go
// Source: internal/session/manager.go (verbatim, Disconnect)
func (m *Manager) Disconnect(userID, reason string) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    session, ok := m.sessions[userID]
    if !ok {
        return nil // already disconnected
    }
    if cleanup, ok := m.cleanups[userID]; ok {
        cleanup()
        delete(m.cleanups, userID)
    }
    if conn, ok := m.conns[userID]; ok {
        conn.Close()
        delete(m.conns, userID)
    }
    session.State = StateDisconnected
    session.DisconnectErr = reason
    delete(m.sessions, userID) // <-- Session struct is gone after this line
    metrics.Get().IncDisconnect(reason)
    m.logDisconnectMetadata(session)
    return nil
}
```
The ON→WAITING hook is one `if` block inserted before `defer m.mu.Unlock()` releases, reading/writing the *separate* autopilot map — not `session`.

### Existing REST connect handler (verbatim, the hook for WAITING→ON, and where the profile-gate callback would attach)
```go
// Source: internal/session/handler.go (verbatim, relevant excerpt)
func (h *Handler) Connect(w http.ResponseWriter, r *http.Request) {
    ...
    session, err := h.manager.Connect(r.Context(), userIDStr, req.Host, req.Port)
    if err != nil { ... }
    if req.ConnectionID != uuid.Nil && h.callbacks != nil {
        if h.callbacks.OnConnected != nil {
            h.callbacks.OnConnected(req.ConnectionID, userUUID)
        }
        ...
    }
    resp := ConnectResponse{State: session.State, SessionID: userIDStr}
    h.sendJSON(w, resp)
}
```
`HandlerCallbacks` (`internal/session/handler.go:22-26`) already demonstrates the exact injection pattern needed for a new `EngageGate func(connectionID, userID uuid.UUID) (allowed bool, message string)` callback, wired in `cmd/server/main.go` the same way `OnConnected`/`GetAutoLogin`/`SendCredentials` already are — no new coupling pattern, reuse of an existing one.

### Existing local-echo mechanism for bracketed notices (verbatim, D-11's mechanism)
```typescript
// Source: frontend/src/pages/PlayScreen.tsx (verbatim)
handleDisconnectRef.current = async () => {
  if (automationEngine) {
    await automationEngine.echoLocal('[Disconnected]', { color: 'white' });
  } else {
    terminal.writeln('\r\n\x1b[37m[Disconnected]\x1b[0m\r\n');
  }
};
```
```typescript
// Source: frontend/src/services/automation.ts (verbatim, echoLocal)
async echoLocal(message: string, options?: {...}): Promise<void> {
  let echoCmd = '#ECHO';
  // ... builds "#ECHO (color:xxx) message" ...
  await this.processUserInput(echoCmd); // same path as user-typed #ECHO — never reaches the MUD
}
```
Every D-11 notice line (`[Autopilot engaged]`, `[Autopilot waiting for reconnect]`, etc.) should call `automationEngine.echoLocal(...)` exactly like this, from whichever place in `PlayScreen.tsx`/`SessionContext.tsx` learns of the transition (the `#AUTO` HTTP response for the acting tab, the `onAutopilot` websocket handler for live pushes).

## State of the Art

Not applicable in the "library version drift" sense — no external library changed. The one relevant fact is that this codebase's own conventions have already evolved once in a directly analogous way: `helpResolver` was added as a new trailing optional parameter to `ExecutionContext`/`executeTokens`/`executeAutomationAction` in a prior change (tagged `PR02PH09` in comments) specifically to let one more `#` directive (`HELP`) reach an external resource (the help content) without restructuring the evaluator. `autopilotControl` for `#AUTO` is the same move a second time, not a new kind of change.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The new `#AUTO` REST endpoint should live on `session.Handler` (alongside Connect/Disconnect/Status) rather than on `profiles.Handler`, with the engage-gate check reached via a new `HandlerCallbacks` entry rather than injecting `*store.ProfileStore` into `session.Handler` directly | Architecture Patterns, Pattern 4 / diagram | Low — this is explicitly Claude's Discretion per CONTEXT.md ("endpoint paths for engage/disengage/status"); either wiring works, the callback approach was chosen because it exactly matches the existing `OnConnected`/`GetAutoLogin` injection pattern already proven in this file, minimizing new coupling. If the planner prefers injecting `*store.ProfileStore` into `session.Handler` instead, no behavior changes, only which file holds the dependency. |
| A2 | The live websocket push for autopilot-state changes should reuse whatever `conn` is already in scope (best-effort, no new registry) rather than building a per-user connection registry | Summary, Anti-Patterns | Medium — if the owner actually uses multiple browser tabs simultaneously against the same profile regularly, best-effort push means a second tab could show a stale badge for up to 15 seconds after a wheel-grab in the first tab. This is judged acceptable because: (a) the product's own code already assumes "one session per user" (comment in `manager.go`), (b) the 15s REST poll is the existing, already-shipped correctness mechanism for the connection badge too, and (c) CONTEXT.md's "no speed bumps" instruction weighs against building new cross-connection infrastructure for an edge case not named in any success criterion. Flagged for the planner to confirm is in scope of Claude's Discretion, not requiring owner sign-off, since it does not change any of the four ROADMAP success criteria's observable behavior for the single-tab case those criteria describe. |
| A3 | `#AUTO`'s new `CommandCategories` entry (or reuse of `OUTPUT`) and exact bracketed-notice color choices are unconstrained beyond D-11's literal wording | Pattern 1 | Low — CONTEXT.md's discretion list explicitly includes "exact badge visuals, colours, and placement"; no owner-visible behavior depends on the category label used internally in `CommandRegistry`. |
| A4 | `ReasonIdle` is currently dead code (idle timeout disabled per a code comment in `startTimers`, "Idle timeout disabled - removed per user request") and does not need special handling for the WAITING transition, since it never fires today | Pattern 4 | Low — verified directly in `manager.go`'s `startTimers` goroutine, the idle-timeout branch is commented out; only `ReasonHardCap` actually fires from that goroutine today. If idle timeout is re-enabled in a future phase, it already funnels through the same `Disconnect` choke point Phase 2 hooks, so no additional Phase 2 work would be needed regardless. |

## Open Questions (RESOLVED)

1. **RESOLVED: deferred to the Phase 2 security review per the binding security-risk-decision rule; plan 02-07 task 4 writes it into `02-SECURITY-AGENDA.md` with Accept / Defer / Remediate Now and nothing pre-chosen; `WaitingSince` is stored so a bound can be added later.** Does the owner want a bounded WAITING lifetime (auto-land-on-OFF after N minutes disconnected), given the deferred security concern that WAITING→ON resume can happen while the owner is away?**
   - What we know: CONTEXT.md's Deferred section names this explicitly as a security-review item, not a design decision made here. Policy section 2 ("If the AI is disconnected or kicked from a game while engaged, it will not reconnect on its own, and you must not re-engage it until you understand why it was disconnected") is arguably in tension with an automatic WAITING→ON resume with no time bound and no re-assessment — the policy text says *the owner* must not re-engage without understanding why; DEC-autopilot-waits-across-disconnect has the *system* do exactly that automatically.
   - What's unclear: whether the owner, when this is raised at the Phase 2 security review, will choose Accept / Defer / Remediate Now, and if Remediate Now, what the bound should be.
   - Recommendation: the planner should NOT decide a numeric bound as an implementation default — this is a security-review decision point requiring the owner's Accept/Defer/Remediate choice, per the project's binding `security-risk-decisions` rule. Build the state machine so a bound could be added later without redesign (i.e., store a WAITING-entered timestamp on the autopilot state struct even if nothing reads it yet in Phase 2) but do not implement a timeout without the owner's explicit choice at the review.

2. **RESOLVED: plan 02-02 adopts the recommendation verbatim, `connection_id` in the `AutopilotRequest` body.** Exact shape of the `#AUTO` REST request/response and whether `connection_id` travels in the body or is resolved some other way.**
   - What we know: the frontend already tracks `currentConnectionId` in `SessionContext.tsx` state, set at connect time (verified); `ConnectRequest` (`internal/session/handler.go`) already demonstrates an optional `ConnectionID uuid.UUID` field carried in a session-scoped REST body for a similar purpose (auto-login lookup).
   - What's unclear: whether the endpoint should require `connection_id` in the body (mirroring `ConnectRequest`) or derive it from the currently-connected session's own stored connection reference (which does not exist today — `Session` has no `ConnectionID` field). Adding one would be a second, smaller instance of the same "don't couple lifecycle-sensitive state to the wrong struct" question as Pitfall 1, though lower stakes since a `ConnectionID` field on `Session` would only be lost on disconnect, which is an acceptable time to lose it (you can't engage autopilot on a disconnected session anyway, per D-03).
   - Recommendation: require `connection_id` in the `#AUTO ON` request body, resolved from the frontend's already-tracked `currentConnectionId`, exactly like `ConnectRequest.ConnectionID` already does for auto-login. This needs no new field on `Session` and reuses the existing per-request pattern rather than adding session-lifecycle-coupled state for a fifth time in this phase.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | Building/testing the backend | ✓ | go1.26 [VERIFIED: go.mod `go 1.26`] | — |
| A live MUD reachable from Railway staging (Alter Aeon or equivalent) | Player-observable verification of connect/disconnect/wheel-grab/reconnect | Not verified this session (requires network access at execution time, not research time) | — | The phase's Go tests (state machine, source-flag rule) require no live MUD at all; only the staging screenshot/log evidence needs one, exactly as Phase 1's evidence plan did |
| Railway staging environment | Deployment target for evidence capture | ✓ (per CLAUDE.md/PROJECT.md, same environment Phase 1 already used successfully) | — | — |
| Frontend test tooling (vitest/jest) | N/A this phase | ✗ | — | Unchanged from Phase 1: none exists; frontend verification is player-observable screenshots, not automated, matching the existing accepted gap |

**Missing dependencies with no fallback:** none — the phase's core diagnostic (`go test`) needs no live MUD, no Postgres beyond what `store.EngageGateAllowed`'s pure-function tests already need (none), and no frontend test framework.

**Missing dependencies with fallback:** frontend automated testing (fallback: screenshots, exact Phase 1 precedent, already proven to satisfy the owner's evidence rule).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` [VERIFIED: same as Phase 1; `internal/store/profile_test.go`, `internal/policy/policy_test.go`, `internal/profiles/handler_test.go` all use plain `testing`, no mocking library] |
| Config file | none — `go test` needs none |
| Quick run command | `go test ./internal/session/... -v` |
| Full suite command | `go test ./...` (matches `.github/workflows/ci.yml`'s "Run backend tests" step, `continue-on-error: true`, unchanged) |
| Recommended addition | `go test ./internal/session/... -race -v` for the new autopilot map's concurrency, since Pitfall 4 is specifically a data-race risk this package has not previously needed to guard against (no existing test in this package runs under `-race`) |

**Frontend test tooling:** still none (`frontend/package.json`'s `"test": "echo 'No tests yet' && exit 0"`, unchanged since Phase 1, [VERIFIED: read this session]). Frontend verification for this phase is player-observable browser walkthrough plus screenshots, exactly as ROADMAP's Phase Validation line for Phase 2 specifies ("Player-observable on staging against a live MUD").

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| REQ-autopilot-directives | `#AUTO ON` engages only when the gate passes; `#AUTO OFF` disengages; already-on/already-off are no-ops with the D-04 wording | unit | `go test ./internal/session/... -run TestEngageAutopilot -v` | ❌ Wave 0 |
| REQ-autopilot-directives | Indicator matches server state after a page refresh | player-observable (browser) | refresh the play screen after engaging, confirm badge reads ON with no manual re-sync action | n/a — screenshot evidence |
| REQ-wheel-grab | Human-sourced `data` message disengages ON before the command is queued; `trigger`-sourced does not; absent/malformed source defaults to human (Pitfall 3) | unit | `go test ./internal/session/... -run TestWheelGrabSourceRule -v` | ❌ Wave 0 |
| REQ-wheel-grab | Typing a command while engaged shows the disengage notice and the command's game response; a trigger-fired command leaves autopilot engaged | player-observable (browser, live MUD) | ROADMAP Phase Validation line, staging walkthrough | n/a — screenshot/log evidence |
| REQ-no-auto-reconnect | `Disconnect` while autopilot ON transitions to WAITING and survives the `sessions` map deletion (Pitfall 1); `Connect` while WAITING resumes to ON; `#AUTO OFF` while WAITING lands on OFF and stays OFF after `Connect` | unit (the phase's most important test, exercises `Manager.Disconnect`→`Manager.Connect` end-to-end, not a bare struct) | `go test ./internal/session/... -run TestWaitingSurvivesDisconnectAndResumesOnConnect -race -v` | ❌ Wave 0 |
| REQ-no-auto-reconnect | Dropping the connection shows waiting and no command is sent while disconnected; reconnecting by hand shows resuming to ON | player-observable (browser, live MUD) + log excerpt | ROADMAP Phase Validation line; `[AI-PLAYER]` log lines for the two transitions | n/a — screenshot + staging log evidence |
| REQ-doc-hand-play-and-gate | Ordinary play with autopilot OFF is unchanged (D-12); AI cannot engage without policy acceptance (reuses Phase 1's `EngageGateAllowed`, already tested) | unit (regression) | `go test ./internal/store/... -run TestEngageGateAllowed -v` (Phase 1 test, unchanged) | ✅ exists (Phase 1) |
| REQ-doc-hand-play-and-gate | Owner creates a profile, accepts policy, hand-plays normally with the AI disengaged, nothing visibly different | player-observable (browser, live MUD) | ROADMAP Phase Validation success criterion 4 | n/a — screenshot evidence |

### Sampling Rate
- **Per task commit:** `go test ./internal/session/... -race -v`
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green, captured to `evidence/01-test-report.txt`; the canned-report harness (extended from `scripts/verify-phase1.sh` or a new `scripts/verify-phase2.sh`, see below) run against staging for every REST-reachable behavior (gate refusal, `#AUTO ON`/`OFF`/`STATUS` responses); a staging `[AI-PLAYER]` log excerpt for both WAITING transitions; end-user screenshots for the badge, the wheel-grab notice, and the reconnect-resume sequence — matching the owner's evidence rule (canned report or screenshot only, never a database query), identical in kind to Phase 1's evidence plan.

### Wave 0 Gaps
- [ ] `internal/session/autopilot.go` + `internal/session/autopilot_test.go` — new files; pure `AutopilotState` type and transition functions (`Engage`, `Disengage`, `EnterWaiting`, `Resume`), zero-fixture table tests proving ON/OFF/WAITING transitions per D-01/D-04 — REQ-autopilot-directives, REQ-no-auto-reconnect
- [ ] `internal/session/manager_test.go` (new, or extend if one exists after this phase's own Wave 0 work) — `Manager.Disconnect`→`Manager.Connect` end-to-end test proving WAITING survives the `sessions` map deletion (Pitfall 1) — REQ-no-auto-reconnect
- [ ] `internal/session/websocket_test.go` (new) — the human/automation source-flag rule as a pure function test (does not require an actual websocket dial; test the classification function directly, e.g. `IsHumanSource(source string) bool`) — REQ-wheel-grab
- [ ] `scripts/verify-phase2.sh` (new, following `scripts/verify-phase1.sh`'s exact shape: `set -uo pipefail`, PASS/FAIL per ROADMAP criterion C1-C4, `--self-test`/`--self-test-negative` modes) — drives every REST-reachable check (engage-gate refusal before acceptance, `#AUTO ON` after acceptance, `#AUTO ON` with no connected session refused per D-03, already-on/already-off no-ops, `#AUTO STATUS`); explicitly documents that websocket-only behaviors (wheel-grab, live badge push, disconnect/reconnect) are OUT of this script's scope and proven by screenshot/log instead, exactly as `verify-phase1.sh`'s header comments already establish the "no database query" ground rule for this project — REQ-autopilot-directives, REQ-no-auto-reconnect
- [ ] Framework install: none — `testing` is stdlib, already in use

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No (unchanged this phase) | Existing `sessionMiddleware` already wraps every new route |
| V3 Session Management | Yes (new: autopilot is server-held session state) | The new autopilot map must be keyed by the authenticated `user_id` from context, never a client-supplied user identifier — same discipline `Manager`'s existing maps already use |
| V4 Access Control | Yes | The `#AUTO ON` endpoint's engage-gate check must resolve `connection_id` through the same ownership-scoped lookup Phase 1 already uses (`GetProfileByConnection(userID, connectionID)`), never trust a client-supplied "allowed" flag |
| V5 Input Validation | Yes | The new `Source` field on inbound `data` messages must be validated against a small allowlist (`""`, `"user"`, `"alias"`, `"trigger"`) — reject/log anything else rather than silently passing it through as either classification |
| V6 Cryptography | No | Unaffected |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Wheel-grab bypass: a compromised or buggy client tags a human-typed command as `"trigger"` to keep autopilot engaged against the owner's intent | Spoofing / Tampering | Server-side enforcement only (Pitfall 3's allowlist default); the client's self-reported source is a convenience for the common case, not a trust boundary the server should rely on for anything beyond UX — however, note that since Phase 2 has no AI decisions yet (no autonomous behavior to protect against), the actual harm of this specific spoof in Phase 2 alone is limited to "autopilot stays ON when the owner expected it to disengage," not "the AI executes a malicious command"; this threat becomes materially higher-stakes starting Phase 3, and should be re-confirmed in that phase's security review |
| Autopilot-state forgery: a request to `#AUTO ON`'s endpoint on a `connection_id` the caller does not own | Elevation of Privilege / Information Disclosure (IDOR) | Same mitigation as Phase 1's T-1-03: resolve `connection_id` through `GetProfileByConnection(userID, connectionID)`, scoped by both columns, never a bespoke query |
| WAITING→ON auto-resume while the owner is away, in tension with policy section 2's "you must not re-engage it until you understand why it was disconnected" | (Policy/process risk, not a STRIDE category) | **Not mitigated by this research — explicitly an Open Question and a required Phase 2 security-review agenda item** (Accept / Defer / Remediate Now per the project's binding risk-decision rule); a bounded WAITING lifetime is the proposed remediation if chosen, but is NOT implemented by default |
| Carry-forward from Phase 1 (must be re-raised, not re-researched) | — | **R-02** (`AIPlayerPanel.tsx` invalid numeric input silently saves as blank) and **R-04** (staging log prints the OTP code) — both MUST appear on this phase's security-review agenda with the same three-choice disposition; the project cannot close while either remains deferred |

## Sources

### Primary (HIGH confidence)
- `internal/session/manager.go` — full file read, this session (Session struct, Manager struct and its three maps, `Connect`, `Disconnect`, `SendCommand`, `GetSession`, disconnect reason constants, `startTimers`)
- `internal/session/websocket.go` — full file read, this session (`WSMessage` struct, `HandleWebSocket`'s full message loop, `case MsgTypeConnect`/`MsgTypeData`, `handleClientCommands`, `readMUDOutput`, `relayMUDToClient`)
- `internal/session/handler.go` — full file read, this session (`Handler`, `HandlerCallbacks`, `Connect`, `Disconnect`, `Status`, `StatusResponse`)
- `internal/store/profile.go` (lines 515-583) — `DefaultDisengageThreshold`, `EngageGateRefusalMessage`, `ResolveAISettings`, `EngageGateAllowed`, read verbatim this session
- `internal/profiles/handler.go` (lines 660-730) — `GetEngageGate`, `getProfileByConnectionID`, read verbatim this session
- `internal/icm/dispatcher.go`, `internal/icm/types.go` — read this session; confirmed `internal/icm` is imported by no other package (`grep -rn "icm\."` across `cmd/`, `internal/session`, `internal/profiles` returns nothing), confirming Phase 2 does not need to touch it
- `cmd/server/main.go` (route registration block, handler construction block) — read this session, confirmed exact route list and handler wiring, confirmed `profilesHandler` does not currently receive `sessionManager`
- `frontend/src/services/automation.ts` — full file read plus targeted greps, this session (`CommandSource`, `processUserInput`, `queueCommand`, `processCommandQueue`, `fireTrigger`, timer-source assignments, `echoLocal`)
- `frontend/src/services/automation/commands.ts` — full file read, this session (`CommandRegistry`, `KNOWN_COMMANDS`)
- `frontend/src/services/automation/evaluator.ts` — targeted reads, this session (`executeTokenList`'s full switch, `executeTokens`, `executeAutomationAction` signature, `ExecutionContext`)
- `frontend/src/pages/PlayScreen.tsx` — targeted reads, this session (imports, `submitCommand`, websocket handler registration incl. `[Disconnected]`, `setSubmitCommandCallback` registration)
- `frontend/src/services/api.ts` — targeted reads, this session (`WebSocketManager` class in full: `connect`, `handleMessage`, `sendCommand`, `sendConnect`, `sendDisconnect`, all `on*`/`off*` handler-list methods)
- `frontend/src/context/SessionContext.tsx` — targeted reads, this session (`connect()`, `refreshStatus()`, `disconnect()`, mount effect, `currentConnectionId` tracking)
- `frontend/src/components/SessionBadge.tsx`, `Header.tsx` — full file reads, this session
- `.github/workflows/ci.yml`, `frontend/package.json`, root `package.json`, `go.mod` — read this session (test commands, confirmed no new dependency needed, confirmed frontend test tooling still absent)
- `scripts/verify-phase1.sh` — header/usage comments read this session (harness shape and invocation modes, template for `verify-phase2.sh`)
- `.specify/specs/ai-game-player-design-v3.md` §D2, `.specify/specs/safety-and-abuse-policy-v1.md` §§1-2 — read this session (amended D2 text, policy sections 1 and 2 verbatim)
- Phase 1 artifacts (`01-RESEARCH.md`, `01-VALIDATION.md`, `01-CONTEXT.md`, all six `01-*-SUMMARY.md`) — read this session for precedent (evidence-file conventions, `[AI-PLAYER]` log format, pure-function test pattern, `HandlerCallbacks` injection pattern)

### Secondary (MEDIUM confidence)
- None required — every substantive architectural claim in this document traces to a file read this session; no WebSearch or Context7 lookup was needed because this phase introduces no new external library or framework.

### Tertiary (LOW confidence)
- Whether the owner regularly uses multiple browser tabs against the same profile simultaneously (Assumption A2) — not verified in this session, flagged as an assumption with a stated, bounded risk if wrong.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — zero new dependencies; every touched file and function read directly this session
- Architecture: HIGH for the state-ownership finding (Pitfall 1/Pattern 3, directly derived from reading `Disconnect`'s and `GetSession`'s actual code, not inferred); MEDIUM for the exact websocket message shape and REST endpoint placement (Claude's Discretion per CONTEXT.md, one clearly-simplest option identified and justified, not a pre-existing pattern to copy)
- Pitfalls: HIGH for Pitfalls 1, 2, and 4 (each directly observed in the existing code's actual control flow); MEDIUM for Pitfall 3 (the correct default is explicitly stated in CONTEXT.md's discretion note, but the specific failure mode described is reasoned from that note plus the code structure, not independently observed)

**Research date:** 2026-09-15
**Valid until:** 30 days (stable brownfield codebase; no fast-moving external dependency involved; the one time-sensitive item is the Open Question requiring an owner decision at the Phase 2 security review, which should happen before or during planning, not after)
