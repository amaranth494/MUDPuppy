# Phase 2: Autopilot Switch - Pattern Map

**Mapped:** 2026-09-15
**Files analyzed:** 18 (Go: 8, frontend: 9, script: 1)
**Analogs found:** 16 exact/role-match / 18 (2 are net-new mechanisms — `internal/session/autopilot.go`'s pure state machine and the CSS badge-state blocks — for which the closest structural analog and the RESEARCH.md sketch are both given)

All line numbers below were re-verified by direct file reads this session (not copied blind from RESEARCH.md), current as of `ai-player` branch HEAD at research time.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `internal/session/autopilot.go` (new) | model (state machine) | transform (pure functions) | `internal/store/profile.go`'s `EngageGateAllowed`/`ResolveDisengageThreshold` (pure, receiver-less functions) | role-match |
| `internal/session/autopilot_test.go` (new) | test | transform | `internal/icm/icm_test.go` (table-driven, plain `testing`) | role-match |
| `internal/session/manager.go` (modify) | model / service (session lifecycle) | CRUD (in-memory map) | itself — `Connect`/`Disconnect`/`GetSession`'s existing three-map pattern | exact (self-extension) |
| `internal/session/manager_test.go` (new) | test | event-driven (Disconnect→Connect sequence) | `internal/icm/icm_test.go` | role-match — first test file in this package |
| `internal/session/websocket.go` (modify) | controller (websocket message loop) | streaming / event-driven | itself — `case MsgTypeData:`/`case MsgTypeConnect:` in `HandleWebSocket` | exact (self-extension) |
| `internal/session/websocket_test.go` (new) | test | transform (pure classifier function) | `internal/icm/icm_test.go` | role-match |
| `internal/session/handler.go` (modify) | controller | request-response | itself — `Connect`/`Disconnect`/`Status` handlers in the same file | exact (self-extension) |
| `cmd/server/main.go` (modify) | route / config | request-response | Phase 1's `timers` `mux.HandleFunc` block (same file) | exact |
| `frontend/src/services/automation/commands.ts` (modify) | config (command registry) | transform | `'HELP'` entry (same file) | exact |
| `frontend/src/services/automation/evaluator.ts` (modify) | service (directive evaluator) | request-response (async HTTP call inside a switch case) | `case 'HELP':` + the `helpResolver` trailing-param precedent (same file) | exact |
| `frontend/src/services/automation.ts` (modify) | service (command queue / engine) | event-driven (queue → callback) | itself — `processCommandQueue`/`setSubmitCommandCallback`/`echoLocal` | exact (self-extension) |
| `frontend/src/services/api.ts` (modify) | service (API client + websocket transport) | request-response / streaming | itself — `sendCommand`/`WebSocketManager.handleMessage`/`getSessionStatus` | exact (self-extension) |
| `frontend/src/pages/PlayScreen.tsx` (modify) | component (page) | event-driven (websocket handlers + submit path) | itself — `submitCommand`, `handleDisconnectRef`, `setSubmitCommandCallback` registration | exact (self-extension) |
| `frontend/src/context/SessionContext.tsx` (modify) | provider (React context) | request-response (poll) / event-driven (websocket) | itself — `refreshStatus`/`connect`/`SessionContextType` | exact (self-extension) |
| `frontend/src/components/AutopilotBadge.tsx` (new) | component | request-response (reads context, no local fetch) | `frontend/src/components/SessionBadge.tsx` | exact |
| `frontend/src/components/Header.tsx` (modify) | component | request-response | itself — `.header-right` placement of `<SessionBadge />` | exact (self-extension) |
| `frontend/src/types/index.ts` (modify) | model (types) | transform | `SessionStatus` interface (same file) | exact |
| `frontend/src/index.css` (modify) | config (styles) | n/a | `.session-badge*` block (same file, lines 137-206) | exact |
| `scripts/verify-phase2.sh` (new) + `scripts/fixtures/phase2/`, `scripts/fixtures/phase2-negative/` | test (harness) | batch (scripted HTTP sequence) | `scripts/verify-phase1.sh` | exact |

## Pattern Assignments

### `internal/session/autopilot.go` (new — model, transform)

**Analog:** no direct precedent for a state-machine type; closest shape is `internal/store/profile.go`'s pure, receiver-less helper functions (`EngageGateAllowed`, `ResolveDisengageThreshold`, verified at `internal/store/profile.go:580` and the `DefaultDisengageThreshold` block immediately above it) — same "no `*sql.DB`/no I/O, independently unit-testable" shape RESEARCH.md Pattern 4 calls for.

**Verified precedent (the shape to mirror — pure function, no receiver, no I/O):**
```go
// Source: internal/store/profile.go:580 (verbatim signature)
func EngageGateAllowed(policyVersionAccepted *string, policyAcceptedAt *string) bool {
```

**What this new file must contain (per RESEARCH.md's Recommended Project Structure and Pattern 3):** an `AutopilotState` type with three values (`AutopilotOn`, `AutopilotOff`, `AutopilotWaiting` — string constants, matching the existing `StateConnected`/`StateDisconnected` string-constant convention at `internal/session/manager.go:17-22`) plus pure transition functions (`Engage`, `Disengage`, `EnterWaiting`, `Resume`) that take and return state values, no map/mutex access — the map/mutex living on `Manager` (see below) is what calls these pure functions. Include a `WaitingSince *time.Time` (or similar) field on whatever struct carries the state, per RESEARCH.md's Open Question 1 recommendation: store a timestamp even though nothing reads it yet, so a future bounded-WAITING-lifetime remediation needs no redesign.

**Existing string-constant convention to match exactly** (`internal/session/manager.go:16-22`, verified):
```go
// Source: internal/session/manager.go:16-22 (verbatim)
const (
	StateDisconnected = "disconnected"
	StateConnecting   = "connecting"
	StateConnected    = "connected"
	StateError        = "error"
)
```

---

### `internal/session/autopilot_test.go` (new — test, transform)

**Analog:** `internal/icm/icm_test.go` (table-driven, plain `testing`, `t.Run` subtests) — the same shape Phase 1's `01-PATTERNS.md` used for `internal/store/profile_test.go`.

```go
// Source: internal/icm/icm_test.go (verbatim shape, table-driven)
package icm

import (
	"testing"
)

func TestRecognizer_RecognizeStructured(t *testing.T) {
	tests := []struct {
		input    string
		wantOp   OperatorFamily
	}{
		{"#echo hello", OperatorStructured},
		// ...
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			// ...
		})
	}
}
```
Apply this shape to `package session`: one table proving D-01/D-04 transitions (OFF --Engage--> ON, ON --Engage--> ON no-op, ON --Disengage--> OFF, OFF --Disengage--> OFF no-op, ON --EnterWaiting--> WAITING, WAITING --Resume--> ON, WAITING --Disengage--> OFF). No fixtures, no `*sql.DB`, pure function tests only.

---

### `internal/session/manager.go` (modify — model/service, CRUD)

**Analog:** self-extension. This is RESEARCH.md's single most load-bearing pattern (Pitfall 1 / Pattern 3): the autopilot map MUST be a fourth map on `Manager`, structurally identical to the existing three, never a field on `Session`.

**Existing three-map pattern to extend** (`internal/session/manager.go:47-59`, verified verbatim):
```go
// Session represents an active MUD connection session
type Session struct {
	UserID         string    `json:"user_id"`
	Host           string    `json:"host"`
	Port           int       `json:"port"`
	State          string    `json:"state"`
	ConnectedAt    time.Time `json:"connected_at,omitempty"`
	LastActivityAt time.Time `json:"last_activity_at,omitempty"`
	DisconnectErr  string    `json:"disconnect_reason,omitempty"`
}

// Manager handles MUD session management
type Manager struct {
	portWhitelist         map[int]bool
	portDenylist          map[int]bool
	portAllowlistOverride map[int]bool
	idleTimeoutMinutes    int
	hardCapHours          int

	mu       sync.RWMutex
	sessions map[string]*Session // userID -> session
	conns    map[string]net.Conn
	cleanups map[string]context.CancelFunc
}
```
**Do NOT add `AutopilotState` to `Session`.** Add a fourth map (`autopilot map[string]*AutopilotState`, or equivalent) guarded by the same `m.mu`, initialized in `NewManager` alongside `sessions`/`conns`/`cleanups`.

**`Disconnect` — the ON→WAITING hook point** (`internal/session/manager.go:302-342`, verified verbatim):
```go
func (m *Manager) Disconnect(userID, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[userID]
	if !ok {
		// Session already removed - treat as already disconnected (success)
		return nil
	}

	// Cancel timers
	if cleanup, ok := m.cleanups[userID]; ok {
		cleanup()
		delete(m.cleanups, userID)
	}

	// Close connection
	if conn, ok := m.conns[userID]; ok {
		conn.Close()
		delete(m.conns, userID)
	}

	// Update session state
	session.State = StateDisconnected
	session.DisconnectErr = reason

	// Remove session from map - this ensures clean state for reconnection
	// and prevents orphaned sessions
	delete(m.sessions, userID)   // <-- Session struct is gone after this line

	// Record metrics
	metrics.Get().IncDisconnect(reason)

	// Log disconnect metadata
	m.logDisconnectMetadata(session)

	log.Printf("[SP02PH01T07] Session disconnected: user=%s, reason=%s, duration=%v",
		userID, reason, time.Since(session.ConnectedAt))

	return nil
}
```
Insert the ON→WAITING transition as an `if` block reading/writing the **separate** `m.autopilot` map, anywhere between `m.mu.Lock()` and the `defer`'s eventual unlock — it survives `delete(m.sessions, userID)` because it lives in a different map entirely. This runs for **every** disconnect reason (`ReasonUser`, `ReasonIdle`, `ReasonHardCap`, `ReasonRemote`, `ReasonError`, `ReasonSlowClient`, `ReasonRateLimit`, `ReasonProtocolMismatch` — all defined at `manager.go:26-33`), confirmed as the single exhaustive choke point.

**`Connect` — the WAITING→ON resume hook point** (`internal/session/manager.go:162-229`, verified verbatim excerpt):
```go
func (m *Manager) Connect(ctx context.Context, userID, host string, port int) (*Session, error) {
	log.Printf("[SP02PH01] Connect called: user=%s, host=%s, port=%d", userID, host, port)
	// ... port/host validation ...

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for existing session (one connection per user)
	if existing, ok := m.sessions[userID]; ok && existing.State == StateConnected {
		return nil, fmt.Errorf("user already has an active session")
	}

	session := &Session{UserID: userID, Host: host, Port: port, State: StateConnecting}
	// ... dial ...
	session.State = StateConnected
	session.ConnectedAt = time.Now()
	session.LastActivityAt = time.Now()
	m.sessions[userID] = session
	m.conns[userID] = conn

	metrics.Get().IncConnect()
	go m.startTimers(context.Background(), userID)
	m.logConnectionMetadata(session)
	return session, nil
}
```
Insert the WAITING→ON resume as an `if m.autopilot[userID] is WAITING { set to ON }` block after the dial succeeds (after `session.State = StateConnected`), still inside the held lock.

**`GetSession`'s zero-value fabrication** (`internal/session/manager.go:344-358`, verified — the reason a `Session`-embedded field would never be observable):
```go
func (m *Manager) GetSession(userID string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, ok := m.sessions[userID]
	if !ok {
		return &Session{
			UserID: userID,
			State:  StateDisconnected,
		}, nil
	}
	return session, nil
}
```

**Concurrency discipline (Pitfall 4):** implement all new autopilot access as `Manager` methods (`EngageAutopilot`, `DisengageAutopilot`, `AutopilotStateFor(userID)`) that take `m.mu` internally, exactly like `GetSession`/`SendCommand` do — never inline map access from `websocket.go`.

---

### `internal/session/manager_test.go` (new — test, event-driven)

**Analog:** `internal/icm/icm_test.go` shape, but exercised end-to-end against the real `Manager`, not a bare `Session` struct (per Pitfall 1's warning sign). Construct a `Manager` via `NewManager(...)`, call `Connect`, `EngageAutopilot`, `Disconnect(userID, ReasonRemote)`, assert `AutopilotStateFor(userID) == WAITING`, call `Connect` again, assert it flips to ON. Run with `-race` (RESEARCH.md's Validation Architecture recommends `go test ./internal/session/... -race -v` specifically because this package has no existing `-race`-covered test).

---

### `internal/session/websocket.go` (modify — controller, streaming/event-driven)

**Analog:** self-extension — the single command-ingress choke point.

**`WSMessage` struct to extend** (`internal/session/websocket.go:16-33`, verified verbatim):
```go
const (
	MsgTypeConnect    = "connect"
	MsgTypeDisconnect = "disconnect"
	MsgTypeData       = "data"
	MsgTypeError      = "error"
	MsgTypeStatus     = "status"
)

type WSMessage struct {
	Type   string `json:"type"`
	Host   string `json:"host,omitempty"`
	Port   int    `json:"port,omitempty"`
	Data   string `json:"data,omitempty"`
	Error  string `json:"error,omitempty"`
	Status string `json:"status,omitempty"`
}
```
Add `Source string `json:"source,omitempty"`` for the inbound wheel-grab flag, and add a new outbound `MsgTypeAutopilot = "autopilot"` constant (RESEARCH.md's recommendation: reuse the `Status`-field string-carrying convention rather than adding a fifth struct field).

**`case MsgTypeData:` — the exact insertion point for the wheel-grab check** (`internal/session/websocket.go:344-374`, verified verbatim):
```go
case MsgTypeData:
	// Rate limiting at WebSocket ingress (SP02PH02T04)
	rl := h.getRateLimiter(userIDStr)
	if !rl.Allow() {
		log.Printf("[SP02PH02T04] Rate limit exceeded for user %s", userIDStr)
		h.sendError(conn, "Rate limit exceeded")
		continue
	}

	// Message size enforcement (SP02PH02T03)
	if len(wsMsg.Data) > h.config.MaxMessageSizeBytes {
		log.Printf("[SP02PH02T03] Message too large: %d bytes", len(wsMsg.Data))
		h.sendError(conn, "Message too large")
		continue
	}

	if !connected {
		h.sendError(conn, "Not connected")
		continue
	}

	// NEW: wheel-grab check goes here — wsMsg.Source is available on the
	// already-unmarshaled wsMsg. Allowlist "trigger" as automation; treat
	// everything else (absent, "", "user", "alias", garbled) as human,
	// per Pitfall 3's fail-safe-direction rule.
	// if isHumanSource(wsMsg.Source) && h.manager.AutopilotStateFor(userIDStr) == session.AutopilotOn {
	//     h.manager.DisengageAutopilot(userIDStr, "wheel-grab")
	//     h.sendJSON(conn, WSMessage{Type: MsgTypeAutopilot, Status: "off"})  // best-effort live push
	// }

	// Send command to MUD via channel
	select {
	case clientToMUD <- wsMsg.Data:
		metrics.Get().IncWSMessagesOut()
	default:
		h.sendError(conn, "Command queue full")
	}
```
**No race is possible here:** this executes serially within `HandleWebSocket`'s single per-connection goroutine loop; the disengage call (itself locking `m.mu` inside the `Manager` method) completes in program order strictly before the line reaches `clientToMUD`.

**`case MsgTypeConnect:`** is at `websocket.go:266` (verified via grep) — this is where the reattach-vs-fresh-dial branch lives; per RESEARCH.md Pattern 4, the WAITING→ON resume hook belongs in `Manager.Connect` itself (already covered above), not duplicated here, since the normal frontend flow calls `Manager.Connect` via the REST handler first.

---

### `internal/session/websocket_test.go` (new — test, transform)

**Analog:** `internal/icm/icm_test.go` table shape, applied to a pure classifier function (no actual websocket dial needed, per RESEARCH.md Wave 0 Gaps):
```go
func TestIsHumanSource(t *testing.T) {
	tests := []struct{ source string; want bool }{
		{"", true},        // absent -> human (Pitfall 3, fail-safe direction)
		{"user", true},
		{"alias", true},
		{"trigger", false},
		{"garbled-nonsense", true}, // anything not explicitly "trigger" is human
	}
	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			if got := isHumanSource(tt.source); got != tt.want {
				t.Errorf("isHumanSource(%q) = %v, want %v", tt.source, got, tt.want)
			}
		})
	}
}
```

---

### `internal/session/handler.go` (modify — controller, request-response)

**Analog:** self-extension — `Connect`/`Status` handler shapes in the same file.

**`Handler`/`HandlerCallbacks` struct to extend** (`internal/session/handler.go:14-26`, verified verbatim):
```go
type Handler struct {
	manager   *Manager
	config    *config.Config
	callbacks *HandlerCallbacks
}

type HandlerCallbacks struct {
	OnConnected     func(connectionID, userID uuid.UUID) error
	GetAutoLogin    func(connectionID uuid.UUID) (username, password string, err error)
	SendCredentials func(userID, username, password string) error
}
```
Add `EngageGate func(connectionID, userID uuid.UUID) (allowed bool, message string)` here, wired in `cmd/server/main.go` the same way `OnConnected`/`GetAutoLogin` already are — this is the injection seam that reaches `store.EngageGateAllowed`/`store.EngageGateRefusalMessage` without giving `session.Handler` a direct `*store.ProfileStore` dependency (RESEARCH.md Assumption A1).

**`Connect` handler shape to mirror for the new `Autopilot` handler** (`internal/session/handler.go:79-98`, verified verbatim, auth + body-parse prologue):
```go
func (h *Handler) Connect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID := r.Context().Value("user_id")
	if userID == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userIDStr := userID.(string)
	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.sendError(w, "Invalid user ID")
		return
	}
	// Parse request body ...
}
```
`ConnectRequest`'s optional `ConnectionID uuid.UUID` field (`handler.go:47-51`, verified) is the precedent for the new `AutopilotRequest{ Action string; ConnectionID uuid.UUID }` body shape (RESEARCH.md Open Question 2's recommendation: require `connection_id` in the body, resolved from the frontend's already-tracked `currentConnectionId`).

**`Status` handler shape to extend for `StatusResponse`** (`internal/session/handler.go:69-77, 207-239`, verified verbatim):
```go
type StatusResponse struct {
	State            string  `json:"state"`
	ConnectedAt      *string `json:"connected_at,omitempty"`
	Host             string  `json:"host,omitempty"`
	Port             int     `json:"port,omitempty"`
	LastActivityAt   *string `json:"last_activity_at,omitempty"`
	LastError        string  `json:"last_error,omitempty"`
	DisconnectReason string  `json:"disconnect_reason,omitempty"`
}

func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID := r.Context().Value("user_id")
	if userID == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userIDStr := userID.(string)
	session, err := h.manager.GetSession(userIDStr)
	if err != nil {
		h.sendError(w, err.Error())
		return
	}
	resp := StatusResponse{State: session.State}
	if session.State == StateConnected {
		resp.Host = session.Host
		resp.Port = session.Port
		connectedAt := session.ConnectedAt.Format(time.RFC3339)
		resp.ConnectedAt = &connectedAt
	}
	// ...
}
```
Add `AutopilotState string `json:"autopilot_state,omitempty"`` to `StatusResponse`, populated from `h.manager.AutopilotStateFor(userIDStr)` — this is D-10's authoritative refresh-correctness path (the 15s-polled `SessionBadge.tsx` mechanism, extended, not a new push mechanism).

**Phase 1's exact engage-gate call, reused verbatim (do not re-derive) — `internal/profiles/handler.go:674-696`, verified verbatim:**
```go
func (h *Handler) GetEngageGate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	_, profile, err := h.getProfileByConnectionID(r)
	if err != nil {
		h.sendError(w, err.Error())
		return
	}
	allowed := store.EngageGateAllowed(profile.PolicyVersionAccepted, profile.PolicyAcceptedAt)
	connectionID, _ := h.getConnectionIDFromPath(r)
	log.Printf("[AI-PLAYER] engage gate connection_id=%s allowed=%t", connectionID, allowed)
	resp := EngageGateResponse{Allowed: allowed}
	if !allowed {
		resp.Message = store.EngageGateRefusalMessage
	}
	h.sendJSON(w, resp)
}
```
`store.EngageGateAllowed` is at `internal/store/profile.go:580`; `store.EngageGateRefusalMessage` (the exact locked D-11 refusal string) is the package-level `const` at `internal/store/profile.go:533`. `#AUTO ON`'s server-side handler calls these two directly via the new `HandlerCallbacks.EngageGate` callback — do not re-derive the policy-acceptance check inside `session`.

**`[AI-PLAYER]` log-line format to reuse for every autopilot transition** (verified verbatim above): `log.Printf("[AI-PLAYER] <event> connection_id=%s ...", ...)` — Phase 1's established prefix and key=value shape; extend with `user_id`, old state, new state, and cause per RESEARCH.md's Established Patterns note.

---

### `cmd/server/main.go` (modify — route/config, request-response)

**Analog:** Phase 1's `timers` route block (`01-PATTERNS.md` cites `cmd/server/main.go:319-328`) — same file, same shape, to be replicated for the new autopilot endpoint(s).

```go
// Source: cmd/server/main.go (verbatim shape, Phase 1's timers block)
mux.HandleFunc("/api/v1/profiles/{connection_id}/timers", func(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		profilesHandler.GetTimers(w, r)
	case http.MethodPut:
		profilesHandler.PutTimers(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
})
```
Register the new `/api/v1/session/autopilot` route (POST-only, `action` in body per Handler section above) the same way; wire `HandlerCallbacks.EngageGate` at `session.NewHandlerWithCallbacks(...)` construction to a closure calling `profileStore.GetProfileByConnection` + `store.EngageGateAllowed`, mirroring how `OnConnected`/`GetAutoLogin` are already wired (grep `NewHandlerWithCallbacks` in this file for the exact call site). All new routes sit inside the same `sessionMiddleware`-wrapped `mux` Phase 1 used — no separate auth wiring.

---

### `frontend/src/services/automation/commands.ts` (modify — config, transform)

**Analog:** the `'HELP'` entry (verified verbatim, `commands.ts:123-128`):
```typescript
'HELP': {
  name: 'HELP',
  category: CommandCategories.OUTPUT,
  requiresArgs: false,
  description: 'Show help',
},
```
Add `'AUTO': { name: 'AUTO', category: CommandCategories.OUTPUT, requiresArgs: false, description: 'Engage or disengage autopilot' }` following this exact shape (`requiresArgs: false` because bare `#AUTO` is valid per D-05). `CommandCategories` (`commands.ts:13-19`, verified) has no `AUTOPILOT` bucket today; reusing `OUTPUT` (as `HELP`/`ECHO`/`LOG`/`GAG` already do) is the lowest-friction choice and is explicitly Claude's Discretion per CONTEXT.md.

---

### `frontend/src/services/automation/evaluator.ts` (modify — service, request-response)

**Analog:** `case 'HELP':` (verified verbatim, `evaluator.ts:1131-1142`):
```typescript
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
`case 'AUTO':` follows this exact shape but is `async` (an HTTP call via the new `context.autopilotControl` callback) — `executeTokenList` (the enclosing function, confirmed at `evaluator.ts:780` as `executeTokens`, which calls the switch) is already `async` and already `await`s inside other cases, so `await context.autopilotControl?.setState(...)` is a direct, precedented extension.

**`ExecutionContext` — the trailing-optional-param precedent to replicate (verified verbatim, `evaluator.ts:158-172`):**
```typescript
export interface ExecutionContext {
  variables: VariableStore;
  depth: number;
  maxDepth: number;
  timerManager?: TimerManager;
  executeCommands?: (commands: string[]) => void;
  aliasResolver?: (aliasName: string) => Promise<string[]>;
  outputMessage?: (message: string) => void;
  // PR02PH09: For fetching help content from backend
  helpResolver?: (topic?: string) => Promise<{ title: string; description: string; sections: { title: string; content: string }[] } | null>;
  // PR02PH09: Source context to distinguish CLI from Triggers/Aliases/Timers
  source?: 'cli' | 'trigger' | 'alias' | 'timer';
}
```
Add `autopilotControl?: { setState: (action: 'on'|'off'|'status', connectionId: string) => Promise<{ state: string; message?: string }> }` the same way `helpResolver` was added — a new trailing optional field, and a new trailing optional parameter threaded through `executeTokens` (signature at `evaluator.ts:780-790`, verified: `helpResolver` is param 8, `source` is param 9) and `executeAutomationAction` (signature at `evaluator.ts:1702-1711`, same param ordering). Both call sites in `automation.ts` (`processUserInput`) already pass `'cli'` as the `source` argument and need the one new trailing argument added.

**Note on `#HELP`'s current body:** the live code's `case 'HELP':` comment says "PR02PH09: Commented out - help should come from Help modal, not CLI" — the case still executes the fallback text shown above; this is the template's actual current behavior, not a broken example.

---

### `frontend/src/services/automation.ts` (modify — service, event-driven)

**Analog:** self-extension — `CommandSource`, `processCommandQueue`, `setSubmitCommandCallback`, `echoLocal` are all in this file.

**`CommandSource` type to extend nowhere (already correct per D-09) — verified verbatim (`automation.ts:19-26`):**
```typescript
export type CommandSource = 'user' | 'alias' | 'trigger';

export interface ProcessedCommand {
  command: string;
  source: CommandSource;
}
```
This type is already correct and already assigned at every origination point; the only change needed in this file is plumbing it through, per RESEARCH.md Pattern 5.

**`setSubmitCommandCallback` — the signature that drops `source` today (verified verbatim, `automation.ts:151, 284-286`):**
```typescript
private onSubmitCommand: ((command: string) => void) | null = null;
// ...
setSubmitCommandCallback(callback: (command: string) => void): void {
  this.onSubmitCommand = callback;
}
```
Change to `(callback: (command: string, source: CommandSource) => void): void`.

**`processCommandQueue` — the single call site that must pass `cmd.source` through (verified verbatim, `automation.ts:1002-1021`):**
```typescript
private async processCommandQueue(): Promise<void> {
  if (this.isProcessingQueue || this.commandQueue.length === 0) {
    return;
  }
  this.isProcessingQueue = true;
  while (this.commandQueue.length > 0) {
    if (this.circuitBreaker.isTripped) {
      break;
    }
    const cmd = this.commandQueue.shift();
    if (!cmd) continue;

    // Submit the command
    if (this.onSubmitCommand) {
      this.onSubmitCommand(cmd.command);   // <-- drops cmd.source; change to (cmd.command, cmd.source)
    }
    this.lastDispatchTime = Date.now();
    await this.sleep(50);
  }
```

**`echoLocal` — the exact mechanism for every D-11 bracketed notice (verified verbatim, `automation.ts:309-331`):**
```typescript
async echoLocal(message: string, options?: {
  color?: string;
  background?: string;
  bold?: boolean;
  underline?: boolean;
}): Promise<void> {
  let echoCmd = '#ECHO';
  if (options && (options.color || options.background || options.bold || options.underline)) {
    const styleParts: string[] = [];
    if (options.color) styleParts.push(`color:${options.color}`);
    if (options.background) styleParts.push(`background:${options.background}`);
    if (options.bold) styleParts.push('bold');
    if (options.underline) styleParts.push('underline');
    echoCmd += ` (${styleParts.join(',')})`;
  }
  echoCmd += ` ${message}`;
  // Process through the evaluator (same path as user-typed #ECHO)
  await this.processUserInput(echoCmd);
}
```
Every D-11 notice (`[Autopilot engaged]`, `[Autopilot waiting for reconnect]`, etc.) is written via `automationEngine.echoLocal(text, { color })` with the exact color names from `02-UI-SPEC.md`'s ANSI colour mapping table — this guarantees the "never sent to the MUD" property for free, with zero new code, exactly matching `[Disconnected]`'s existing mechanism.

---

### `frontend/src/services/api.ts` (modify — service, request-response/streaming)

**Analog:** self-extension — `sendCommand`, `handleMessage`, `getSessionStatus`.

**`sendCommand` — the source-flag insertion point (verified verbatim, `api.ts:154-161`):**
```typescript
sendCommand(command: string): void {
  if (this.ws && this.ws.readyState === WebSocket.OPEN) {
    this.ws.send(JSON.stringify({
      type: 'data',
      data: command,
    }));
  }
}
```
Change to `sendCommand(command: string, source?: CommandSource): void` and include `source` in the JSON payload: `{ type: 'data', data: command, source }`.

**`WebSocketManager.handleMessage` — the switch to extend with the new `autopilot` message type (verified verbatim, `api.ts:131-152`):**
```typescript
private handleMessage(message: WSMessage): void {
  switch (message.type) {
    case 'data':
      if (message.data) {
        this.messageHandlers.forEach(handler => handler(message.data!));
      }
      break;
    case 'error':
      if (message.error) {
        this.errorHandlers.forEach(handler => handler(message.error!));
      }
      break;
    case 'status':
      if (message.status) {
        this.statusHandlers.forEach(handler => handler(message.status!));
      }
      break;
    case 'disconnect':
      this.disconnectHandlers.forEach(handler => handler());
      break;
  }
}
```
Add `case 'autopilot':` following the `status` case's exact shape, plus a parallel `autopilotHandlers: ((state: string) => void)[]` array and `onAutopilot`/`offAutopilot` methods, mirroring `onStatus`/`offStatus` (`api.ts:203-212`, verified) exactly.

**`SessionStatus` type to extend (verified verbatim, `frontend/src/types/index.ts:12-20`):**
```typescript
export interface SessionStatus {
  state: ConnectionState;
  connected_at?: string;
  host?: string;
  port?: number;
  last_activity_at?: string;
  last_error?: string;
  disconnect_reason?: string;
}
```
Add `autopilot_state?: 'on' | 'waiting' | 'off';` — this is the field the REST `GET /api/v1/session/status` extension (Go `StatusResponse.AutopilotState`) lands in on the frontend, and is D-10's actual refresh-correctness mechanism (not the websocket push).

**New functions to add, following `getTimers`/`putTimers`'s established fetch-wrapper shape** (Phase 1's `01-PATTERNS.md` documents this exact shape at `api.ts:558-584`: `credentials: 'include'`, `handleAuthError(response)` before the `response.ok` check, `throw new Error(data.error || '<fallback>')`) — add `setAutopilot(connectionId: string, action: 'on'|'off'|'status')` calling the new `POST /api/v1/session/autopilot` endpoint.

---

### `frontend/src/pages/PlayScreen.tsx` (modify — component, event-driven)

**Analog:** self-extension — `handleDisconnectRef`, `setSubmitCommandCallback` registration, `submitCommand`.

**`[Disconnected]` notice — the exact mechanism D-11's WAITING/reconnect sequence reuses (verified verbatim, `PlayScreen.tsx:302-317`):**
```typescript
// PR02PH09: Use automation engine's echoLocal for consistent #ECHO styling
handleErrorRef.current = async (err: string) => {
  if (automationEngine) {
    await automationEngine.echoLocal(`[ERROR] ${err}`, { color: 'red' });
  } else {
    terminal.writeln(`\r\n\x1b[91m[ERROR] ${err}\x1b[0m\r\n`);
  }
};

handleDisconnectRef.current = async () => {
  if (automationEngine) {
    await automationEngine.echoLocal('[Disconnected]', { color: 'white' });
  } else {
    terminal.writeln('\r\n\x1b[37m[Disconnected]\x1b[0m\r\n');
  }
};
```
Extend `handleDisconnectRef.current` (or add a sibling handler fired right after it) to also emit `[Autopilot waiting for reconnect]` in `brightyellow` **only if** autopilot was ON at disconnect time (D-11's two-line, not-concatenated sequence) — the server-authoritative state for "was it ON" should come from the same websocket/REST source as everywhere else, not be inferred client-side.

**`setSubmitCommandCallback` registration — the call site that must pass `source` through (verified verbatim, `PlayScreen.tsx:339-364`):**
```typescript
useEffect(() => {
  if (automationEngine && wsManager) {
    automationEngine.setSubmitCommandCallback((command: string) => {
      wsManager.sendCommand(command + '\n');
      // ... echo the command ...
    });
    // ...
  }
}, [automationEngine, wsManager, profile]);
```
Change the callback signature to `(command: string, source: CommandSource) => { wsManager.sendCommand(command + '\n', source); ... }`.

**`submitCommand`'s direct-send (blank-command / no-engine) branch — MUST be tagged `'user'` explicitly (Pitfall 2; verified verbatim, `PlayScreen.tsx:379-434`):**
```typescript
const submitCommand = useCallback(async (_source: 'typing' | 'keybinding', text: string) => {
  // ...
  const command = text;   // Don't trim - allow blank lines for MUDs
  // ... ICM validation ...
  if (wsManager && connectionState === 'connected') {
    if (automationEngine && command !== '' && !automationDisabled) {
      const processedCommands = await automationEngine.processUserInput(command);
      if (processedCommands.length === 0) {
        return;
      }
      // Don't echo here - the automation engine's callback will echo
    } else {
      // No automation engine OR empty command - send directly to WebSocket
      // Empty commands bypass automation (some MUDs need blank lines)
      wsManager.sendCommand(command + '\n');   // <-- MUST become sendCommand(command + '\n', 'user')
      // ... local echo via automationEngine.echoLocal ...
    }
  }
```
This `else` branch is unconditionally human-typed by construction (`submitCommand` is only invoked from typed input at line ~567 and keybindings at line ~442) — per D-07, a blank Enter here must still take the wheel, so this call needs the explicit `'user'` tag even though it bypasses the automation engine entirely (Pitfall 2's exact warning).

---

### `frontend/src/context/SessionContext.tsx` (modify — provider, request-response/event-driven)

**Analog:** self-extension — `refreshStatus`, `SessionContextType`, mount effect.

**`refreshStatus` — the extension point for `autopilotState` (verified verbatim, `SessionContext.tsx:221-235`):**
```typescript
const refreshStatus = useCallback(async () => {
  try {
    const status = await getSessionStatus();
    setConnectionState(status.state);
    setHost(status.host || '');
    setPort(status.port || 23);
    if (status.last_error) {
      setError(mapBackendError(status.last_error));
    } else {
      setError(null);
    }
  } catch (err) {
    console.error('Failed to get session status:', err);
  }
}, []);
```
Add `setAutopilotState(status.autopilot_state || 'off')` inside the `try` block — this is called on mount (`SessionContext.tsx:210-219`, `init()`), and `SessionBadge.tsx` already calls it on a 15s interval plus on visibility-change, so extending this one function is D-10's entire refresh-correctness mechanism; no new polling loop needed.

**`SessionContextType` interface** (`SessionContext.tsx:26-42`, confirmed present with `connectionState`, `refreshStatus`, `currentConnectionId`) — add `autopilotState: 'on' | 'waiting' | 'off'` alongside `connectionState`, following the exact naming/placement convention already used for `connectionState`.

---

### `frontend/src/components/AutopilotBadge.tsx` (new — component)

**Analog:** `frontend/src/components/SessionBadge.tsx` (full file, 81 lines, verified verbatim below) — the structural template named explicitly by `02-UI-SPEC.md`.

```tsx
// Source: frontend/src/components/SessionBadge.tsx (verbatim, full file)
import { useState, useEffect, useCallback } from 'react';
import { useSession } from '../context/SessionContext';

export default function SessionBadge() {
  const { connectionState, refreshStatus } = useSession();
  const [lastRefresh, setLastRefresh] = useState<Date>(new Date());

  const handleVisibilityChange = useCallback(() => {
    if (!document.hidden) {
      refreshStatus().then(() => setLastRefresh(new Date()));
    }
  }, [refreshStatus]);

  useEffect(() => {
    document.addEventListener('visibilitychange', handleVisibilityChange);
    return () => {
      document.removeEventListener('visibilitychange', handleVisibilityChange);
    };
  }, [handleVisibilityChange]);

  useEffect(() => {
    const interval = setInterval(() => {
      refreshStatus().then(() => setLastRefresh(new Date()));
    }, 15000); // 15 seconds - reduced from 5s
    return () => clearInterval(interval);
  }, [refreshStatus]);

  const getStateDisplay = () => {
    switch (connectionState) {
      case 'connected': return 'Connected';
      case 'connecting': return 'Connecting';
      case 'disconnected': return 'Disconnected';
      case 'error': return 'Error';
      default: return 'Disconnected';
    }
  };

  const getStatusClass = () => {
    switch (connectionState) {
      case 'connected': return 'status-connected';
      case 'connecting': return 'status-connecting';
      case 'error': return 'status-error';
      default: return 'status-disconnected';
    }
  };

  return (
    <div className={`session-badge ${getStatusClass()}`} title={`Last updated: ${lastRefresh.toLocaleTimeString()}`}>
      <span className="session-badge-dot" />
      <span className="session-badge-text">{getStateDisplay()}</span>
    </div>
  );
}
```
`AutopilotBadge.tsx` is simpler: no local polling of its own (it reads `autopilotState` straight from `useSession()`, which is already kept fresh by `SessionBadge.tsx`'s existing 15s/visibility-change refresh of the same context) — per `02-UI-SPEC.md`'s exact markup:
```tsx
<div className={`autopilot-badge state-${stateClass}`} title={`Autopilot: ${label}`}>
  <span className="autopilot-badge-dot" />
  <span className="autopilot-badge-text">Autopilot: {label}</span>
</div>
```
where `stateClass` is `on`/`waiting`/`off` and `label` is `On`/`Waiting`/`Off`, both derived from `autopilotState`, never inferred locally.

---

### `frontend/src/components/Header.tsx` (modify — component)

**Analog:** self-extension — the exact `.header-right` placement of `<SessionBadge />` (verified verbatim, `Header.tsx:56-59`):
```tsx
<div className="header-right">
  {/* Session Badge - Persistent status indicator derived from API */}
  <SessionBadge />

  <div className="account-menu">
```
Per `02-UI-SPEC.md`, insert `<AutopilotBadge />` immediately after `<SessionBadge />` and before `<div className="account-menu">`, plus `import AutopilotBadge from './AutopilotBadge';` alongside the existing `import SessionBadge from './SessionBadge';` (`Header.tsx:5`).

---

### `frontend/src/index.css` (modify — config)

**Analog:** `.session-badge`/`.session-badge-dot`/`.session-badge.status-*` (verified verbatim, `index.css:137-206`):
```css
.session-badge {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  padding: var(--spacing-xs) var(--spacing-md);
  border-radius: 999px;
  font-size: var(--font-size-sm);
  font-weight: bold;
  text-transform: uppercase;
}

.session-badge-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: currentColor;
}

.session-badge.status-disconnected {
  background: rgba(128, 128, 128, 0.2);
  color: #888;
}

.session-badge.status-connecting {
  background: rgba(255, 255, 0, 0.2);
  color: var(--color-warning);
}

.session-badge.status-connected {
  background: rgba(0, 255, 0, 0.2);
  color: var(--color-success);
}

.session-badge.status-error {
  background: rgba(255, 68, 68, 0.2);
  color: var(--color-error);
}
```
`02-UI-SPEC.md` gives the exact new block verbatim (append directly below `.session-badge.status-error` at line 206): `.autopilot-badge`, `.autopilot-badge-dot`, `.autopilot-badge.state-on/.state-waiting/.state-off` — a structural copy renamed for the autopilot concept, reduced from four states to three, no new spacing/colour token values (every value already exists in this file for the connection badge).

---

### `scripts/verify-phase2.sh` (new — test harness, batch)

**Analog:** `scripts/verify-phase1.sh` (full header + argument-parsing + helper section read verbatim this session).

**Invocation-mode / usage pattern to replicate exactly (verified verbatim, `verify-phase1.sh:1-125`):**
```bash
#!/usr/bin/env bash
# scripts/verify-phase1.sh -- Phase 1 canned-report harness ...
set -uo pipefail   # errexit intentionally omitted: a failed check must not
                    # abort the report -- the report must list every result.

SELF_TEST=0
SELF_TEST_NEGATIVE=0
NO_TESTS=0
REPORT=""

usage() { cat >&2 <<'USAGE'
Usage: scripts/verify-phase1.sh [--self-test|--self-test-negative] [--no-tests] <report-file>
USAGE
}

for arg in "$@"; do
  case "$arg" in
    --self-test) SELF_TEST=1 ;;
    --self-test-negative) SELF_TEST_NEGATIVE=1 ;;
    --no-tests) NO_TESTS=1 ;;
    --*) echo "Unknown flag: $arg" >&2; usage; exit 2 ;;
    *) REPORT="$arg" ;;
  esac
done
# ... fixture-dir selection, env-var validation ...

exec > >(tee "$REPORT") 2>&1
RUN_TS=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_SHA=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
echo "================================================================"
echo "Phase 1 canned-report harness -- scripts/verify-phase1.sh"
echo "Run started (UTC): $RUN_TS"
# ...
```

**`_get_field` JSON-extraction helper (jq-if-available, grep fallback) — reuse verbatim, verified (`verify-phase1.sh:156-181`):**
```bash
_get_field() {
  local json="$1" field="$2"
  if command -v jq >/dev/null 2>&1; then
    printf '%s' "$json" | jq -r --arg f "$field" '
      (getpath($f | split("."))) as $v
      | if $v == null then "null"
        elif ($v | type) == "string" then $v
        else ($v | tostring)
        end' 2>/dev/null
  else
    local leaf="${field##*.}"
    printf '%s' "$json" \
      | grep -oE "\"$leaf\"[[:space:]]*:[[:space:]]*(\"[^\"]*\"|null|true|false|-?[0-9]+(\.[0-9]+)?)" \
      | head -n 1 \
      | sed -E "s/^\"$leaf\"[[:space:]]*:[[:space:]]*//" \
      | sed -E 's/^"(.*)"$/\1/'
  fi
}
```

**Fixture-file list pattern (verified, `verify-phase1.sh:183-199`):** an ordered `FIXTURE_FILES=(...)` bash array, one file per `_http` call, under `scripts/fixtures/phase1/` and `scripts/fixtures/phase1-negative/`. `verify-phase2.sh` needs parallel `scripts/fixtures/phase2/` and `scripts/fixtures/phase2-negative/` directories with one JSON fixture per REST call in its sequence (engage-gate refusal before acceptance, `#AUTO ON` after acceptance, `#AUTO ON` with no connected session refused per D-03, already-on/already-off no-ops, `#AUTO STATUS`).

**Explicit out-of-scope note to carry over into the new script's header comment** (per RESEARCH.md Wave 0 Gaps): websocket-only behaviors (wheel-grab, live badge push, disconnect/reconnect) are OUT of this script's scope and proven by screenshot/log instead — state this in the header exactly as `verify-phase1.sh`'s own comments establish the "canned report or screenshot, never a database query" rule.

## Shared Patterns

### `[AI-PLAYER]` structured log line
**Source:** `internal/profiles/handler.go:689` — `log.Printf("[AI-PLAYER] engage gate connection_id=%s allowed=%t", connectionID, allowed)`
**Apply to:** every new autopilot state transition in `manager.go` (`Disconnect`'s ON→WAITING, `Connect`'s WAITING→ON) and `handler.go`'s new `Autopilot` handler (`#AUTO ON`/`#AUTO OFF`/no-op) — one `[AI-PLAYER]` line per transition with user/session id, old state, new state, and cause, per RESEARCH.md's Established Patterns note.

### State lives on `Manager`, never on `Session`
**Source:** `internal/session/manager.go:47-59` (three existing parallel maps under `m.mu`), Pitfall 1/Pattern 3 in `02-RESEARCH.md`
**Apply to:** `internal/session/manager.go`'s new autopilot map — the single highest-risk copy-paste point in this phase, exactly parallel to Phase 1's "partial-update-via-full-marshal" shared pattern in `01-PATTERNS.md`.

### Local bracketed notice via `echoLocal`
**Source:** `frontend/src/services/automation.ts:309-331` (`echoLocal`), `frontend/src/pages/PlayScreen.tsx:311-317` (`[Disconnected]` usage)
**Apply to:** every D-11 notice line across `PlayScreen.tsx`/wherever the `#AUTO` response or `onAutopilot` handler lands — guarantees "never sent to the MUD" for free.

### `#` directive registration (two-file edit)
**Source:** `frontend/src/services/automation/commands.ts:123-128` (`CommandRegistry['HELP']`), `frontend/src/services/automation/evaluator.ts:1131-1142` (`case 'HELP':`)
**Apply to:** `#AUTO`'s registration — exactly two edits, no new parser path, matching four prior directives (`ECHO`/`LOG`/`HELP`/`TIMER`).

### Auth/scoping via context `user_id` + ownership-scoped lookup
**Source:** `internal/session/handler.go:87-98` (`r.Context().Value("user_id")` + `uuid.Parse`), `internal/profiles/handler.go:698+` (`getProfileByConnectionID`)
**Apply to:** the new `Autopilot` handler and the `EngageGate` callback — never a client-supplied user identifier, per RESEARCH.md's V3/V4 Security Domain findings.

### Fetch wrapper (`credentials: 'include'` + `handleAuthError`)
**Source:** `frontend/src/services/api.ts` (established in `01-PATTERNS.md` at lines 558-584, unchanged this phase)
**Apply to:** the new `setAutopilot`/status-fetching functions in `api.ts`.

### Canned-report harness shape
**Source:** `scripts/verify-phase1.sh` (full file structure: usage/argparse, `tee`-to-report, `_get_field`, fixture-array, PASS/FAIL-per-criterion)
**Apply to:** `scripts/verify-phase2.sh` — identical shape, new criteria (C1-C4 mapped to `02-ROADMAP.md`'s Phase 2 success criteria), new fixture set.

## No Analog Found

| File | Role | Data Flow | Reason |
|---|---|---|---|
| `internal/session/autopilot.go` (the `AutopilotState` type + pure transition functions themselves) | model | transform | No existing state-machine type in the codebase; `store.EngageGateAllowed`'s pure-function shape is the closest structural precedent (given above) but nothing today models a three-value machine with transition functions. RESEARCH.md Pattern 3/Recommended Project Structure gives the target shape directly; no further codebase search is useful. |
| CSS badge-state blocks (`.autopilot-badge*` in `index.css`) | config | n/a | No existing three-state (as opposed to four-state) badge variant; `02-UI-SPEC.md` already supplies the exact verbatim CSS to append, itself a structural copy of `.session-badge.status-*`, so no additional analog search is needed beyond what's cited above. |

## Metadata

**Analog search scope:** `internal/session/`, `internal/store/`, `internal/profiles/`, `internal/icm/`, `cmd/server/`, `frontend/src/services/`, `frontend/src/services/automation/`, `frontend/src/pages/`, `frontend/src/components/`, `frontend/src/context/`, `frontend/src/types/`, `frontend/src/index.css`, `scripts/`
**Files scanned (this session, targeted or full reads, line numbers verified against current HEAD):** `internal/session/manager.go` (lines 1-75, 162-241, 302-370), `internal/session/websocket.go` (lines 14-58, 340-379), `internal/session/handler.go` (lines 1-100, 190-239), `internal/store/profile.go` (targeted grep for `EngageGateAllowed`/`EngageGateRefusalMessage`, lines 533, 580), `internal/profiles/handler.go` (lines 674-699), `frontend/src/services/automation/commands.ts` (full file, 150 lines), `frontend/src/services/automation/evaluator.ts` (lines 150-172, 780-810, 1120-1170, 1700-1795), `frontend/src/services/automation.ts` (lines 1-35, 280-335, 995-1030), `frontend/src/pages/PlayScreen.tsx` (lines 295-435), `frontend/src/services/api.ts` (lines 92-217), `frontend/src/components/SessionBadge.tsx` (full file, 81 lines), `frontend/src/components/Header.tsx` (full file, 103 lines), `frontend/src/context/SessionContext.tsx` (targeted grep + lines 210-250), `frontend/src/types/index.ts` (lines 12-21), `frontend/src/index.css` (lines 137-206), `scripts/verify-phase1.sh` (lines 1-200)
**Also read (upstream inputs, not re-cited as analogs):** `02-CONTEXT.md`, `02-RESEARCH.md` (full, 587 lines), `02-UI-SPEC.md` (full, 257 lines), `01-PATTERNS.md` (full, format precedent)
**Pattern extraction date:** 2026-09-15
