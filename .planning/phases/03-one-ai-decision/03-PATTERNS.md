# Phase 3: One AI Decision - Pattern Map

**Mapped:** 2026-09-15
**Files analyzed:** 24 (Go: 15 new/modified, frontend: 9 new/modified)
**Analogs found:** 19 exact/role-match / 24 (5 are net-new mechanisms — the Gemini client, the driver orchestration, the ring buffer, the transcript tap, and the AI Assist panel's decision-card layout — for which the closest structural analog and RESEARCH.md's/03-UI-SPEC.md's own verbatim sketch are both given)

All line numbers below were re-verified by direct file reads this session (not copied blind from RESEARCH.md), current as of `ai-player` branch HEAD at research time. Several RESEARCH.md citations of `internal/session/handler.go`/`manager.go` describe a slightly earlier state than HEAD; this document uses the as-verified-this-session line numbers and signatures throughout (e.g. `AutopilotRequest` already carries `ConnectionID`, `EngageGate` already returns three values, the `autopilot` map is already `map[string]*AutopilotRecord`).

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `internal/gemini/client.go` (new) | service (external REST client) | request-response | no direct analog in this codebase (first external HTTP client); closest shape is any `*Store`'s `NewXStore(db)` constructor-plus-method convention | role-match (structural convention only) |
| `internal/gemini/client_test.go` (new) | test | request-response | `internal/icm/icm_test.go` (table-driven, plain `testing`) + Go stdlib `httptest.Server` (no existing httptest precedent in this codebase — first one) | role-match |
| `internal/driver/driver.go` (new) | service (orchestration) | event-driven (fires once per engage) | `internal/store/profile.go`'s `ResolveAISettings` (pure function assembling multiple upstream values) is the closest compositional analog; no existing "orchestrate three subsystems" file exists | role-match |
| `internal/driver/driver_test.go` (new) | test | event-driven | `internal/icm/icm_test.go` table shape, with fake Gemini + fake Dispatcher | role-match |
| `internal/session/window.go` (new) | model (ring buffer + ANSI strip) | transform | `internal/session/websocket.go`'s `stripTelnetIAC` (pure `[]byte -> []byte` transform, no receiver) | exact (same transform shape, sibling function) |
| `internal/session/window_test.go` (new) | test | transform | `internal/icm/icm_test.go` | role-match |
| `internal/session/transcript.go` (new) | model/service (session-scoped tap + store-backed open/close) | event-driven | `internal/session/autopilot.go` (pure state) + `Manager`'s existing map-under-`m.mu` convention combined | role-match |
| `internal/session/transcript_test.go` (new) | test | event-driven | `internal/icm/icm_test.go` | role-match |
| `internal/session/manager.go` (modify) | model/service (session lifecycle) | CRUD (in-memory maps) | itself — the existing four-map (`sessions`/`conns`/`cleanups`/`autopilot`) pattern becomes five/six maps | exact (self-extension) |
| `internal/session/websocket.go` (modify) | controller (websocket message loop) | streaming/event-driven | itself — `case MsgTypeData:`'s wheel-grab insertion point, `WSMessage` struct extension | exact (self-extension) |
| `internal/session/handler.go` (modify) | controller | request-response | itself — `Autopilot` handler's gate-composition pattern, `StatusResponse` extension pattern | exact (self-extension) |
| `internal/icm` (test addition only — `dispatcher_test.go` or new file) | test | transform | `internal/icm/icm_test.go`'s `TestDispatcher_SafetyEnforcement` (lines 659-685) | exact |
| `internal/config/config.go` (modify) | config | transform | itself — the optional/warn-and-default `os.Getenv` blocks (e.g. `SMTPPort`, `IdleTimeoutMinutes`) | exact (self-extension) |
| `internal/store/decisions.go` (new) | model/service (`*Store`) | CRUD | `internal/store/profile.go`'s `ProfileStore` (`NewProfileStore(db)`, `GetProfile`, `GetProfileByConnection`) | exact |
| `internal/store/transcripts.go` (new) | model/service (`*Store`) | CRUD | `internal/store/profile.go`'s `ProfileStore` (same constructor/query shape) | exact |
| `internal/auth/handler.go` (modify — DR-2-01) | controller | request-response | itself — the two `log.Printf("STAGING: OTP sent to user, code: %s", otp)` call sites | exact (self-extension) |
| `migrations/011_*.up/down.sql` (new) | migration | batch | `migrations/010_add_ai_fields.up/down.sql` (`ALTER TABLE ... ADD COLUMN IF NOT EXISTS`) + `migrations/006_create_profiles.up.sql` (`CREATE TABLE IF NOT EXISTS`, `gen_random_uuid()`, `REFERENCES ... ON DELETE CASCADE`) | exact |
| `cmd/server/main.go` (modify) | route/config | request-response | itself — the existing `HandlerCallbacks` composition-literal block (lines 182-208) and route-registration block (lines 304-307) | exact (self-extension) |
| `scripts/verify-phase3.sh` (new) | test (harness) | batch | `scripts/verify-phase2.sh` (full shape: `set -uo pipefail`, `_get_field`, fixture arrays, `--self-test`/`--self-test-negative`) | exact |
| `frontend/src/components/AIAssistPanel.tsx` (new) | component | event-driven (websocket) + request-response (REST reload on mount) | `frontend/src/components/AutopilotBadge.tsx` (server-owns-truth, derives nothing) for the state-reading half; `03-UI-SPEC.md`'s own verbatim markup for the panel body | exact (spec gives verbatim markup) |
| Logs page (`frontend/src/pages/LogsPage.tsx`, new) | component (page) | request-response | `frontend/src/pages/SettingsPage.tsx`'s `.settings-layout` two-pane pattern + `AIPlayerPanel.tsx`'s per-connection load-on-mount pattern | role-match |
| `frontend/src/services/api.ts` (modify) | service (API client + websocket transport) | request-response/streaming | itself — `onAutopilot`/`offAutopilot` (lines 246-255), `case 'autopilot':` (lines 160-164), `setAutopilot` (line 702) | exact (self-extension) |
| `frontend/src/services/automation.ts` (modify) | service (command queue/engine) | event-driven | itself — `autopilotControl` field (line 167) and its threading into `executeAutomationAction` calls (lines 535, 583) | exact (self-extension) |
| `frontend/src/services/icm-adapter.ts` (modify) | service | request-response | itself — `validateAndNormalize`/`processCommand`'s existing `catch { /* frontend fallback */ }` blocks (lines 540-545, 605-620) | exact (self-extension, removal of fallback) |
| `frontend/src/pages/PlayScreen.tsx` (modify) | component (page) | event-driven | itself — `handleDisconnectRef`/`previousAutopilotStateRef` transition effect (lines 306-379), `submitCommand`'s `echoLocal` call (lines 485-492) | exact (self-extension) |
| `frontend/src/pages/SettingsPage.tsx` (modify) | component (page) | request-response | itself — `SECTIONS` array (lines 24-32) and the `ai-player` section's render block (lines 1691-1693) | exact (self-extension) |
| `frontend/src/index.css` (modify) | config (styles) | n/a | `.autopilot-badge*` block (lines 208-239) for the minimized tab; `03-UI-SPEC.md` gives the AI Assist panel/Logs page CSS verbatim | exact |
| `frontend/src/App.tsx` (modify — new route) | route config | n/a | existing React Router route list (not read this session in full; add `/logs/:connectionId` the same way every other page route is registered) | role-match |

## Pattern Assignments

### `internal/gemini/client.go` (new — service, request-response)

**Analog:** no existing HTTP client to an external service in this codebase; the shape below is RESEARCH.md's own verified-against-official-docs design, not copied from an existing file. Follow this codebase's zero-dependency convention (`net/http` + `encoding/json` only, no SDK).

**Exact request construction (the header form, never `?key=`):**
```go
// Source: RESEARCH.md Pattern 2 / Code Examples, verified against
// ai.google.dev/api/generate-content this session
body, _ := json.Marshal(map[string]any{
    "systemInstruction": map[string]any{"parts": []map[string]string{{"text": tierTwo}}},
    "contents": []map[string]any{{"role": "user", "parts": []map[string]string{{"text": tierOne}}}},
    "generationConfig": map[string]any{
        "responseMimeType": "application/json",
        "responseSchema": map[string]any{
            "type": "object",
            "properties": map[string]any{
                "reasoning": map[string]string{"type": "string"},
                "command":   map[string]string{"type": "string"},
            },
            "required": []string{"reasoning", "command"},
        },
    },
})
req, _ := http.NewRequestWithContext(ctx, "POST",
    endpoint+"/v1beta/models/"+model+":generateContent", bytes.NewReader(body))
req.Header.Set("x-goog-api-key", apiKey) // never ?key=... in the URL — DR-2-01 precedent
req.Header.Set("Content-Type", "application/json")
```

**Double-JSON-decode (the load-bearing subtlety, Pitfall 5):** the outer HTTP body decodes to `{candidates: [{content: {parts: [{text: string}]}}]}`; that inner `text` field is itself a JSON string and must be `Unmarshal`-ed a second time into `{reasoning, command}`. Forgetting this produces raw JSON text shown to the owner as "reasoning."

**Error classification to return as typed errors (D-13/D-20 boundary):** 400 → bad-request/malformed; 401/403 → auth (D-20's "configured but wrong" case, still a D-13 failure not a crash); 429 `RESOURCE_EXHAUSTED` → rate-limited (D-19). Never log the full outbound request URL (it never contains the key under the header form, but keep the discipline explicit in a comment, mirroring `manager.go`'s `logConnectionMetadata`/`logDisconnectMetadata` "no PII" comment style at `internal/session/manager.go:637-641`).

---

### `internal/gemini/client_test.go` (new — test, request-response)

**Analog:** `internal/icm/icm_test.go`'s table-driven `testing` style, combined with Go stdlib `httptest.Server` (first use of `httptest` in this codebase — no in-repo precedent, use it directly per stdlib convention). Table cases: 200 success (verify double-decode), 400, 401/403, 429, malformed inner JSON, empty `candidates`. Assert the request the fake server received used the `x-goog-api-key` header, never a `?key=` query string.

---

### `internal/driver/driver.go` (new — service, event-driven)

**Analog:** no existing orchestration file; assemble from three already-read primitives exactly as RESEARCH.md's Architecture Patterns diagram and Code Examples specify. This is the single most consequential file in the phase (RESEARCH.md's central finding).

**The ICM dispatch call — must be `Dispatcher.Dispatch` directly, never `Engine.Process()` (verified against `internal/icm/dispatcher.go:187-224` this session):**
```go
// Source: internal/icm/dispatcher.go:187-212 (verified verbatim, the actual
// Dispatch method the driver must call directly)
func (d *Dispatcher) Dispatch(ctx *ExecutionContext, sessionID string, normalized *NormalizedCommand) (*CommandResult, *ICMError) {
    if normalized == nil {
        return nil, NewICMError(E5000InternalError, "nil command", nil)
    }
    if !d.checkSafety(ctx, sessionID, normalized.Command) {
        return nil, NewICMError(E4003CircuitBreaker, "Circuit breaker tripped", nil)
    }
    authority, ok := AuthorityLevels[*ctx]
    if !ok {
        authority = AuthorityLevels[ContextPreview]
    }
    if !authority.CanExecute && normalized.RequiresExecution {
        return nil, NewICMError(E4001PermissionDenied, "Not authorized to execute", nil)
    }
    handler := d.getHandler(normalized.Operator, normalized.Command)
    if handler == nil {
        // No handler - this might be pass-through case
        return nil, nil
    }
    // ... handler.Handle + RecordExecution, not reached for plain commands ...
}
```
`checkSafety` (`dispatcher.go:240-259`, verified) runs circuit breaker → rate limit → queue depth, in that order, before `getHandler` is even consulted — this is what makes the direct `Dispatch` call "genuinely dispatched through the ICM engine's dispatcher and safety checker" for a plain command with no registered handler (`getHandler` returns `nil`, `Dispatch` returns `(nil, nil)` = "approved, pass through").

**The driver's own call (build exactly this shape, per RESEARCH.md's Code Examples, using the real `NormalizedCommand` fields verified at `internal/icm/types.go:78-97`):**
```go
ctx := icm.ContextAutomation
normalized := &icm.NormalizedCommand{
    Command:           command, // AI's returned command, single plain line
    Operator:          "",
    RequiresExecution: true,
}
if _, icmErr := dispatcher.Dispatch(&ctx, sessionID, normalized); icmErr != nil {
    return fmt.Errorf("icm refused: %s", icmErr.Message) // D-13 failure — never sent
}
return manager.SendCommand(userID, command) // manager.go:545-569, verified
```
`manager.SendCommand(userID, command string) error` (verified verbatim at `internal/session/manager.go:545-569`) is the only path to the MUD; it trims trailing newlines and writes `command + "\r\n"`. The driver calls this itself — `Dispatch` never sends anything.

**Where the profile's tier-two text comes from (verified, `internal/store/profile.go:293` signature + fields at lines 21-22):** `ProfileStore.GetProfileByConnection(userID, connectionID uuid.UUID) (*Profile, error)` returns `Profile.ConductRules string` / `Profile.ApproachGuidance string`, verbatim, no resolution needed (D-05 tier two).

**Model name resolution (verified, `internal/store/profile.go:548-573`):** `store.ResolveAISettings(profile.AISettings, serverDefaultModel) ResolvedAISettings` — blank `ModelName` resolves to the env-sourced default; call this in the driver, not in Go source, per D-18/D-20.

**Diagnostic test proving success criterion 4 (place in `internal/icm`, mirrors the existing `TestDispatcher_SafetyEnforcement` shape verified at `icm_test.go:659-685`):**
```go
// Source: pattern following internal/icm/icm_test.go:659-685's own established style
func TestDispatch_AutomationPassThrough_GoesThroughSafetyChecker(t *testing.T) {
    d := NewDispatcher(NewRegistry())
    ctx := ContextAutomation
    for i := 0; i < 100; i++ {
        d.safety.RecordExecution("sess-1", "north")
    }
    _, icmErr := d.Dispatch(&ctx, "sess-1", &NormalizedCommand{Command: "north", RequiresExecution: true})
    if icmErr == nil {
        t.Fatal("expected Dispatch to refuse once the rate limit is tripped")
    }
}
```

**D-13 validation before `Dispatch` is ever called:** the command must be a single line with no leading `#`/`@`/`$`/`%` and no embedded newline — this is both D-13's "malformed answer"/"non-game line" failure category and a security boundary (V5 Input Validation in RESEARCH.md's Security Domain); enforce in the driver package before building `NormalizedCommand`, never rely on the model's own good behavior.

---

### `internal/driver/driver_test.go` (new — test, event-driven)

**Analog:** `internal/icm/icm_test.go`'s table shape, exercised against a fake Gemini client (an interface the driver depends on, satisfied by a test double returning canned `{reasoning, command}` or an error) and a fake `Dispatcher`-shaped interface (or the real `icm.NewDispatcher(icm.NewRegistry())`, which needs no I/O). Table cases per RESEARCH.md's Wave 0 Gaps: exactly-one-decision-per-engage (D-01, D-03 no-op on repeated `#AUTO ON`), WAITING→ON resume fires exactly one fresh decision (D-02), and every D-13 failure kind (API error, no command, multiple commands, `#`-prefixed non-game line, malformed JSON) sends nothing to the MUD and disengages.

---

### `internal/session/window.go` (new — model, transform)

**Analog:** `internal/session/websocket.go`'s `stripTelnetIAC` — a pure `[]byte -> []byte` transform with no receiver, already living in package `session` (verified at `internal/session/websocket.go:585-`, called from `relayMUDToClient` at line 512: `cleanData := stripTelnetIAC(data)`). Mirror this exact shape for ANSI stripping and add a bounded ring buffer type alongside it.

```go
// Source: internal/session/websocket.go:585-587 (verbatim shape to mirror)
// stripTelnetIAC removes telnet IAC (Interpret As Command) sequences from the data
// IAC is byte 255 (0xFF). Telnet commands are: IAC + command + [option]
func stripTelnetIAC(data []byte) []byte {
```
New sibling function, per RESEARCH.md Pattern 1 (verified-sufficient regex approach, no state machine needed):
```go
// NEW: internal/session/window.go
var ansiCSI = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripANSI(b []byte) []byte {
    return ansiCSI.ReplaceAll(b, nil)
}
```
**Call site:** `relayMUDToClient` already calls `stripTelnetIAC(data)` at `websocket.go:512` before forwarding to the browser (browser output is unaffected — ANSI stays in the browser-bound stream). The ring buffer's own feed point is a *second*, independent call on the same `cleanData` (telnet-stripped) bytes, further passed through `stripANSI`, appended to a new `map[string]*ringBuffer` field on `Manager` — never inserted into the `relayMUDToClient` path that reaches the browser.

**Ring buffer bound:** ~8KB per RESEARCH.md, bytes overwrite oldest-first — a small fixed-size struct, testable with no I/O, same "pure enough to unit test standalone" discipline as `internal/session/autopilot.go`'s pure transition functions (`Engage`/`Disengage`/`EnterWaiting`/`Resume`, verified at `internal/session/autopilot.go:60-120`).

---

### `internal/session/window_test.go` (new — test, transform)

**Analog:** `internal/icm/icm_test.go` table shape, or `internal/session/autopilot.go`'s own pure-function-test precedent (no test file for `autopilot.go` was found in this session's file listing, so `icm_test.go`'s style is the concrete template). Table cases: ANSI CSI sequence stripped, sequence split across two appends is tolerated (cosmetic, not a crash), buffer overwrites oldest bytes first at the bound, snapshot includes pre-engage text (D-04) with no special-casing needed because the buffer is fed continuously regardless of autopilot state.

---

### `internal/session/transcript.go` (new — model/service, event-driven)

**Analog:** the same "new map on `Manager`, guarded by `m.mu`, never a field on `Session`" discipline `internal/session/manager.go` already uses for `autopilot map[string]*AutopilotRecord` (verified at `manager.go:64`, alongside `sessions`/`conns`/`cleanups` at lines 61-64). This is Pitfall 4 (RESEARCH.md) restated: the transcript tap must not live on `Session`, or `Manager.Disconnect`'s `delete(m.sessions, userID)` (verified at `manager.go:347`) would destroy it before it can be flushed/closed — though unlike the ring buffer, the transcript's own lifecycle (open at Connect, close at Disconnect) is intentional, so the tap needs its own `Open(userID, connectionID string)`/`Close(userID string)` methods called from the exact points below.

**Open-on-connect insertion point (verified verbatim, `internal/session/manager.go:222-228`):**
```go
// Store connection
m.sessions[userID] = session
m.conns[userID] = conn

// Resume a waiting autopilot now that the dial has actually succeeded,
// but only onto the profile it was parked on.
m.resumeAutopilotLocked(userID, connectionID)
```
Insert transcript-open immediately after this block, gated on `connectionID != ""` (D-14: quick connects excluded) — `Manager.Connect`'s own signature already carries `connectionID string` as its fifth parameter (verified at `manager.go:169`), so no new plumbing is needed to know which connections are profile-backed.

**Close-on-disconnect insertion point (verified verbatim, `internal/session/manager.go:340-347`):**
```go
// Park an engaged autopilot at waiting before the session is deleted
// below — the autopilot map is separate from m.sessions and survives
// this deletion (RESEARCH Pitfall 1).
m.parkAutopilotLocked(userID)

// Remove session from map - this ensures clean state for reconnection
// and prevents orphaned sessions
delete(m.sessions, userID)
```
Insert transcript-close immediately before or alongside `m.parkAutopilotLocked(userID)` — both are keyed off the same `session` variable already in scope at this point in `Disconnect`, which still holds `session.ConnectionID` before the map delete.

**Line-tagging call sites:** `SendCommand(userID, command string) error` (verified, `manager.go:545`) and `ReadOutput(userID string, buffer []byte) (int, error)` (verified, `manager.go:609`) are the only two paths; RESEARCH.md's recommendation to thread a `source` parameter into `SendCommand` (human/AI) is the natural extension point since `SendCommand` is already the driver's own send path (see `driver.go` above) and the browser's own `wsMsg.Data` send path (via `websocket.go`'s `case MsgTypeData:`, `clientToMUD <- wsMsg.Data`) — both funnel through this one function.

**AI engage/disengage markers inside the transcript (D-15):** record a marker line at the same points `logAutopilotTransition` already fires (verified, `manager.go:381-384`) — `EngageAutopilot` (line 421), `DisengageAutopilot` (line 465), so a stint (`#AUTO ON` to `#AUTO OFF`) is reconstructable from the transcript itself, not a separate join.

---

### `internal/session/transcript_test.go` (new — test, event-driven)

**Analog:** `internal/icm/icm_test.go` shape, exercised against a real `Manager` (constructed via `NewManager(...)`, same as Phase 2's `manager_test.go` precedent) — open on `Connect` with a non-empty `connectionID`, assert no transcript session for `connectionID == ""` (quick connect, D-14), assert human vs. AI line tagging round-trips, assert close on `Disconnect`.

---

### `internal/session/manager.go` (modify — model/service, CRUD)

**Analog:** self-extension — the existing `Manager` struct (verified verbatim, `manager.go:53-65`):
```go
// Manager handles MUD session management
type Manager struct {
	portWhitelist         map[int]bool
	portDenylist          map[int]bool
	portAllowlistOverride map[int]bool
	idleTimeoutMinutes    int
	hardCapHours          int

	mu        sync.RWMutex
	sessions  map[string]*Session // userID -> session
	conns     map[string]net.Conn
	cleanups  map[string]context.CancelFunc
	autopilot map[string]*AutopilotRecord // userID -> autopilot state
}
```
Add two more maps the same way `autopilot` was added on top of the original three (`outputWindow map[string]*ringBuffer`, `transcript map[string]*transcriptSession` or similar), all guarded by the same `m.mu`, all initialized in `NewManager` alongside the existing four. **Do not add fields to `Session`** — same rule Phase 2's own code comments state explicitly at `manager.go:41-44` ("It lives here, not on Session, because autopilot state must outlive the session that engaged it") for `AutopilotRecord.ConnectionID`.

**`RecentOutputSnapshot(userID string) string` — new public method, same visibility/locking convention as `AutopilotStateFor` (verified verbatim, `manager.go:402-411`):**
```go
func (m *Manager) AutopilotStateFor(userID string) AutopilotState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rec, ok := m.autopilot[userID]
	if !ok {
		return AutopilotOff
	}
	return rec.State
}
```
Mirror this exactly for the ring-buffer snapshot: `RLock`, look up the per-user ring buffer, return its current bytes as a string (or `""` if absent), `RUnlock` via `defer`.

---

### `internal/session/websocket.go` (modify — controller, streaming/event-driven)

**Analog:** self-extension — the existing `WSMessage` struct and its wheel-grab insertion point (both verified verbatim this session, current HEAD, already includes Phase 2's `Source`/`ConnectionID` fields):
```go
// WebSocket message structure (internal/session/websocket.go:31-46, verbatim)
type WSMessage struct {
	Type   string `json:"type"`
	Host   string `json:"host,omitempty"`
	Port   int    `json:"port,omitempty"`
	Data   string `json:"data,omitempty"`
	Error  string `json:"error,omitempty"`
	Status string `json:"status,omitempty"`
	Source string `json:"source,omitempty"`
	ConnectionID string `json:"connection_id,omitempty"`
}
```
Add a new `Decision *AIDecisionPayload `json:"decision,omitempty"`` field (RESEARCH.md Pattern 4) and a new `MsgTypeAI = "ai"` constant alongside the existing `MsgTypeAutopilot = "autopilot"` constant (verified, `websocket.go:24-29`).

**The wheel-grab insertion point the driver's own send must NOT trigger (verified verbatim, `websocket.go:416-434`):**
```go
// Wheel-grab: a human-sourced command disengages autopilot
// before it is sent (D-07/D-09, threat T-2-01). Best-effort
// push of the new state to this tab; the command is queued on
// every path below regardless of whether the grab happened or
// the push succeeded (no lost keystrokes, T-2-12).
if grabbed, state := applyWheelGrab(h.manager, userIDStr, wsMsg.Source); grabbed {
	_ = h.writeJSON(conn, WSMessage{Type: MsgTypeAutopilot, Status: string(state), Data: "wheel-grab"})
}

// Send command to MUD via channel
...
select {
case clientToMUD <- wsMsg.Data:
```
This block only runs for browser-originated `MsgTypeData` messages that arrive over the websocket's `HandleWebSocket` loop. The driver's own `manager.SendCommand` call (see `driver.go` above) bypasses this loop entirely — it calls the `Manager` method directly, never goes through `clientToMUD`, and therefore never risks tripping its own wheel-grab check. `IsHumanSource` (verified, `websocket.go:57-64`) already treats anything other than `"trigger"`/`"timer"` as human — the driver's calls never pass through `IsHumanSource` at all, confirming no accidental self-disengage is possible by construction.

**Broadcasting the new `MsgTypeAI` message:** add a `writeJSON`-based push from the driver's caller (likely `handler.go`'s `Autopilot` "on" branch, or a callback the driver invokes) — mirror the exact `h.writeJSON(conn, WSMessage{...})` call shape already used for the wheel-grab push above. Since the websocket connection lives in `WebSocketHandler`, not `Manager`, the driver itself cannot call `writeJSON` directly; route the decision push through a callback/channel the `WebSocketHandler` already owns, the same indirection Phase 2 used for the wheel-grab's "best-effort push."

---

### `internal/session/handler.go` (modify — controller, request-response)

**Analog:** self-extension — the existing `Autopilot` handler's gate-composition already matches RESEARCH.md Pattern 6's recommendation almost exactly (verified verbatim, `handler.go:295-397`; the codebase is already one step ahead of RESEARCH.md's sketch — `EngageGate` already returns `(allowed bool, message string, policyVersion string)` and is already resolved once for every action).

**`HandlerCallbacks` — the exact composition point for D-20's new callback (verified verbatim, `handler.go:22-32`):**
```go
type HandlerCallbacks struct {
	OnConnected     func(connectionID, userID uuid.UUID) error
	GetAutoLogin    func(connectionID uuid.UUID) (username, password string, err error)
	SendCredentials func(userID, username, password string) error
	EngageGate func(connectionID, userID uuid.UUID) (allowed bool, message string, policyVersion string)
}
```
Add `AIConfigured func() (ok bool, message string)` here, following the same "nil callback fails closed" discipline the doc comment on `EngageGate` already states (`handler.go:27-30`: "A nil EngageGate ... must be treated as 'gate refuses' by every caller — a missing dependency fails closed, never open").

**The exact `"on"` branch to extend (verified verbatim, `handler.go:351-376`):**
```go
switch action {
case "on":
	if !gateAllowed {
		cur := h.manager.AutopilotStateFor(userIDStr)
		resp.State = string(cur)
		resp.Outcome = "refused-gate"
		log.Printf("[AI-PLAYER] autopilot user_id=%s connection_id=%s old=%s new=%s cause=%s",
			userIDStr, req.ConnectionID.String(), cur, cur, "refused-gate")
		h.sendJSON(w, resp)
		return
	}

	newState, changed, engageErr := h.manager.EngageAutopilot(userIDStr, req.ConnectionID.String())
	resp.State = string(newState)
	switch {
	case engageErr == ErrNoConnectedSession:
		resp.Outcome = "refused-no-session"
	case engageErr == ErrWrongConnection:
		resp.Outcome = "refused-wrong-connection"
	case changed:
		resp.Outcome = "engaged"
	default:
		resp.Outcome = "already-on"
	}
	h.sendJSON(w, resp)
	return
```
Insert the `AIConfigured` check as a second gate, checked alongside `gateAllowed` (both must pass, per RESEARCH.md Pattern 6 — "two distinct, independently-testable checks", each with its own exact wording: `store.EngageGateRefusalMessage` for the policy gate, `[Autopilot refused: AI is not configured on this server]` (D-20, verbatim) for the config gate). When `changed` is true (a real engage, not a no-op), invoke the driver's `HandleEngage(userID, connectionID)` here — this is the D-01/D-03 "fires once per real engage, never on an already-on no-op" guarantee, already structurally guaranteed by `EngageAutopilot`'s own `changed` return value (verified, `manager.go:447-451`: "on stays on (changed false, D-04 — a repeated #AUTO ON must reset nothing)").

**`AutopilotResponse`'s existing shape (verified verbatim, `handler.go:107-113`)** already documents its `Outcome` enum in a comment (`"engaged, disengaged, already-on, already-off, refused-gate, refused-no-session, status"`) — add `refused-not-configured` to this enum and its doc comment, following the exact same naming convention (`refused-` + reason).

**New `GET /api/v1/session/decisions` handler:** follow the `Status` handler's exact shape (verified verbatim, `handler.go:247-293`: method check, `r.Context().Value("user_id")`, `h.manager.GetSession`-equivalent lookup, `h.sendJSON`) — add alongside `Status`/`Autopilot` in this same file, per RESEARCH.md Pattern 5's recommendation ("the same conceptual family").

---

### `internal/icm` (test addition only, e.g. `internal/icm/dispatcher_test.go` or appended to `icm_test.go`)

**Analog:** `TestDispatcher_SafetyEnforcement` (verified verbatim, `icm_test.go:659-685`, shown in full above under `driver.go`'s diagnostic-test section) — apply the exact same `NewDispatcher(NewRegistry())` + manual `RecordExecution` loop + `Dispatch` call shape, proving success criterion 4's "dispatched through the ICM engine's dispatcher and safety checker" claim with zero HTTP/MUD machinery, matching this test file's own existing style precisely.

---

### `internal/config/config.go` (modify — config, transform)

**Analog:** self-extension — every existing optional-env block already follows one shape (verified verbatim, e.g. `config.go:80-89` for `OTPExpiryMinutes`):
```go
// OTP expiry (optional, defaults to 15 minutes)
cfg.OTPExpiryMinutes = 15
if expiryStr := os.Getenv("OTP_EXPIRY_MINUTES"); expiryStr != "" {
	expiry, err := strconv.Atoi(expiryStr)
	if err != nil {
		log.Printf("Warning: Invalid OTP_EXPIRY_MINUTES '%s', using default 15", expiryStr)
	} else {
		cfg.OTPExpiryMinutes = expiry
	}
}
```
The model registry and default-model-name fields must follow this exact "warn and default, never fail `Load()`" pattern — **critically, unlike `SessionSecret` (verified, `config.go:56-60`, which is the one `return nil, errors.New(...)` fail-fast case in this file)**, the Gemini config must never trigger that branch (D-20: absence is not fatal). Add `DefaultModelName string`, `ModelRegistry map[string]ModelEntry` (endpoint + API key per entry), reading from either one JSON env var or a small set of `AI_MODEL_*` variables (Claude's Discretion) — either way, `Load()` returns a valid `*Config` with empty/zero values when absent, and `AIConfigured()` (used by `handler.go`'s new callback) is a simple `len(cfg.DefaultModelName) > 0 && cfg.ModelRegistry[cfg.DefaultModelName].APIKey != ""`-style check, computed at `main.go`'s wiring time, not inside `Load()` itself.

---

### `internal/store/decisions.go` (new — model/service, CRUD)

**Analog:** `internal/store/profile.go`'s `ProfileStore` (verified verbatim, `profile.go:168-178`):
```go
type ProfileStore struct {
	db *sql.DB
}

func NewProfileStore(db *sql.DB) *ProfileStore {
	return &ProfileStore{db: db}
}
```
`DecisionStore` follows this exact constructor shape. Query methods follow `GetProfileByConnection`'s shape (verified verbatim, `profile.go:293-`, a single-row `db.QueryRow(...).Scan(...)` into a struct) for a single decision, and a `db.Query(...)` + row-scan loop (no existing multi-row example was read this session in `profile.go`, but `database/sql`'s standard `rows.Next()` loop is the only convention this codebase uses anywhere — `Don't Hand-Roll` in RESEARCH.md explicitly rules out an ORM).

---

### `internal/store/transcripts.go` (new — model/service, CRUD)

**Analog:** same `NewXStore(db *sql.DB) *XStore` shape as `DecisionStore` above and `ProfileStore` (`profile.go:168-178`). Two query surfaces per D-17: list sessions for a profile (ordered by start time), and read one session's lines (ordered by sequence/timestamp) — both scoped through `GetProfileByConnection`-style ownership checks (userID + connectionID), never a client-supplied "this is my profile" claim, per RESEARCH.md's V4 Access Control note (same IDOR discipline as Phase 1/2's `getProfileByConnectionID`).

---

### `internal/auth/handler.go` (modify — DR-2-01)

**Analog:** self-extension — the exact two call sites (verified verbatim this session, `internal/auth/handler.go:172-181` and `:261-270`, both inside the same `if h.emailSender != nil && h.emailSender.IsConfigured() { ... } else { ... log.Printf("STAGING: OTP sent to user, code: %s", otp) }` shape):
```go
if h.emailSender != nil && h.emailSender.IsConfigured() {
	if err := h.emailSender.SendOTP(email, otp); err != nil {
		// ... existing error handling ...
	}
} else {
	// existing dev/staging fallback branch
	log.Printf("STAGING: OTP sent to user, code: %s", otp)
}
```
Gate both `log.Printf("STAGING: OTP sent to user, code: %s", otp)` lines behind a new env flag (e.g. `cfg.DebugLogOTP`, default `false`, read in `config.go` the same optional-env way as every other flag in that file) — when the flag is off (the default), print a line confirming an OTP was sent without the code (or a hash), matching CONTEXT.md's own discretion note ("gate the sign-in code log line behind an env flag that is off by default, or log a hash"). Do not change what the line prints when explicitly enabled for debugging — only whether the code appears by default.

---

### `migrations/011_*.up.sql` / `.down.sql` (new — migration, batch)

**Analog:** `migrations/010_add_ai_fields.up.sql` (full file, verified verbatim) for the `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` convention, and `migrations/006_create_profiles.up.sql` (verified, lines 1-20) for the `CREATE TABLE IF NOT EXISTS` convention:
```sql
-- Source: migrations/006_create_profiles.up.sql:1-17 (verbatim shape)
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    connection_id UUID NOT NULL REFERENCES saved_connections(id) ON DELETE CASCADE,
    keybindings JSONB NOT NULL DEFAULT '{}'::jsonb,
    ...
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_profiles_connection_id ON profiles(connection_id);
CREATE INDEX IF NOT EXISTS idx_profiles_user_id ON profiles(user_id);
```
One `011` migration (RESEARCH.md's recommendation, "no speed bumps") creating `ai_decisions` (one row per decision: timestamp, `profile_id`/`connection_id` FK, the text window, reasoning, command, outcome), `game_sessions` (one row per profile connection: `profile_id` FK, `started_at`, `ended_at` nullable), and `game_session_lines` (one row per line: `game_session_id` FK `ON DELETE CASCADE`, `source` enum-like text `human|ai|game`, `text`, `sequence`/`created_at`). Every FK follows `profiles`'/`saved_connections`' own `REFERENCES ... ON DELETE CASCADE` convention. The down migration mirrors `010_add_ai_fields.down.sql`'s exact `DROP COLUMN IF EXISTS`/here `DROP TABLE IF EXISTS` shape.

---

### `cmd/server/main.go` (modify — route/config, request-response)

**Analog:** self-extension — the existing `HandlerCallbacks` composition literal and route block (both verified verbatim this session, current HEAD):
```go
// Source: cmd/server/main.go:182-208 (verbatim, current HEAD)
sessionHandler := session.NewHandlerWithCallbacks(sessionManager, cfg, &session.HandlerCallbacks{
	OnConnected: func(connectionID, userID uuid.UUID) error {
		return connectionsHandler.UpdateLastConnectedAt(connectionID, userID)
	},
	GetAutoLogin: func(connectionID uuid.UUID) (string, string, error) {
		return connectionsHandler.GetCredentialsForAutoLogin(connectionID)
	},
	SendCredentials: func(userID, username, password string) error {
		return sessionManager.SendCredentials(userID, username, password)
	},
	EngageGate: func(connectionID, userID uuid.UUID) (bool, string, string) {
		profile, err := profileStore.GetProfileByConnection(userID, connectionID)
		if err != nil || profile == nil {
			return false, store.EngageGateRefusalMessage, ""
		}
		if !store.EngageGateAllowed(profile.PolicyVersionAccepted, profile.PolicyAcceptedAt) {
			return false, store.EngageGateRefusalMessage, ""
		}
		policyVersion := ""
		if profile.PolicyVersionAccepted != nil {
			policyVersion = *profile.PolicyVersionAccepted
		}
		return true, "", policyVersion
	},
})
```
Add `AIConfigured: func() (bool, string) { ... }` to this same composite literal, checking `cfg.DefaultModelName`/`cfg.ModelRegistry` per the D-20 message. Construct `icm.NewEngine()` once here (confirmed unimported anywhere in `cmd/`/`internal/session`/`internal/profiles` this session — the dormant engine finally gets a caller), call `icmHandler.RegisterRoutes(mux)` (verified existing at `internal/icm/handler.go:22-27`, unused today), and construct the `internal/driver` package with `sessionManager`, `profileStore`, the new `gemini` client, and the new `decisions`/`transcripts` stores, then thread the driver into `session.HandlerCallbacks` (either as a new field, or the `Autopilot` handler holds a reference directly — Claude's Discretion).

**Route block to extend (verified verbatim, `cmd/server/main.go:304-307`):**
```go
mux.HandleFunc("/api/v1/session/connect", sessionHandler.Connect)
mux.HandleFunc("/api/v1/session/disconnect", sessionHandler.Disconnect)
mux.HandleFunc("/api/v1/session/status", sessionHandler.Status)
mux.HandleFunc("/api/v1/session/autopilot", sessionHandler.Autopilot)
```
Add `mux.HandleFunc("/api/v1/session/decisions", sessionHandler.Decisions)` here, plus new transcript-list/transcript-read routes (recommend nesting under `/api/v1/profiles/{connection_id}/sessions` and `/api/v1/profiles/{connection_id}/sessions/{session_id}`, following the exact `{connection_id}` path-parameter convention already used at `main.go:322-395` for aliases/triggers/environment/timers/ai-settings/policy/engage-gate). All new routes sit inside the same `mux` wrapped by `sessionMiddleware` (verified, line 435) — no separate auth wiring needed.

---

### `scripts/verify-phase3.sh` (new — test harness, batch)

**Analog:** `scripts/verify-phase2.sh`'s full shape (per `02-PATTERNS.md`, itself modeled on `scripts/verify-phase1.sh`) — `set -uo pipefail`, `_get_field` JSON-extraction helper, ordered `FIXTURE_FILES=()` array under `scripts/fixtures/phase3/` and `scripts/fixtures/phase3-negative/`, `--self-test`/`--self-test-negative`/`--no-tests` flags, `tee`-to-report. New criteria per RESEARCH.md's Validation Architecture: D-20 missing-config refusal (before Gemini env vars are set on staging — this is explicitly verifiable today), decisions-reload via the new REST endpoint, and a source-grep step (`grep -rn "gemini-" internal/ --include=*.go`, expect zero matches outside test fixtures) proving no hard-coded model ID reached Go source (REQ-env-config).

## Shared Patterns

### `[AI-PLAYER]` structured log line
**Source:** `internal/session/manager.go:377-384` (`logAutopilotTransition`, verified verbatim: `log.Printf("[AI-PLAYER] autopilot user_id=%s connection_id=%s old=%s new=%s cause=%s", ...)`)
**Apply to:** every new decision event in the driver (request sent, answer received, command dispatched, failure) and every transcript session open/close — one `[AI-PLAYER]` line per event, same key=value shape, per RESEARCH.md's Established Patterns note and CONTEXT.md's own discretion note on this exact format.

### State/buffers live on `Manager`, never on `Session`
**Source:** `internal/session/manager.go:53-65` (four existing maps under `m.mu`), reinforced by the `AutopilotRecord.ConnectionID` doc comment at `manager.go:34-37` ("It lives here, not on Session, because autopilot state must outlive the session that engaged it")
**Apply to:** the new ring-buffer map and transcript map — RESEARCH.md's Pitfall 4, the single highest-risk copy-paste point in this phase, exactly parallel to Phase 2's own Pitfall 1.

### `Dispatch` called directly, never `Engine.Process()`
**Source:** `internal/icm/dispatcher.go:187-224` (`Dispatch`) vs. `internal/icm/engine.go` (not re-read this session in full — RESEARCH.md's line-by-line trace of `Process()`'s early return is treated as verified/HIGH confidence per its own Sources section)
**Apply to:** `internal/driver/driver.go` exclusively — this is the phase's single most consequential correctness rule, and the diagnostic test in `internal/icm` exists specifically to prove it.

### Gate composition: nil callback fails closed
**Source:** `internal/session/handler.go:27-30` (`EngageGate`'s doc comment, verbatim: "A nil EngageGate ... must be treated as 'gate refuses' by every caller — a missing dependency fails closed, never open")
**Apply to:** the new `AIConfigured` callback — same discipline, same doc-comment convention, checked alongside `gateAllowed` in the `Autopilot` handler's `"on"` branch (Pattern 6).

### Env var parsing: warn-and-default, one fatal exception
**Source:** `internal/config/config.go:56-60` (the sole `errors.New` fatal case, `SessionSecret`) vs. every other field in the same file (e.g. `:80-89`, `:98-107`, warn-and-default)
**Apply to:** the new Gemini/model-registry config fields — must follow the warn-and-default majority pattern, never the `SessionSecret` fatal exception (D-20 explicitly forbids a startup fatal here).

### `*Store` constructor shape
**Source:** `internal/store/profile.go:168-178` (`type ProfileStore struct { db *sql.DB }`, `func NewProfileStore(db *sql.DB) *ProfileStore`)
**Apply to:** the new `DecisionStore` and `TranscriptStore` (or combined `SessionLogStore`) — identical constructor shape, `database/sql` + hand-written SQL, no ORM.

### Migration conventions
**Source:** `migrations/010_add_ai_fields.up/down.sql` (full files, verified verbatim) + `migrations/006_create_profiles.up.sql:1-20` (verified)
**Apply to:** the new `011_*.up/down.sql` — `CREATE EXTENSION IF NOT EXISTS pgcrypto`, `gen_random_uuid()` PKs, `REFERENCES ... ON DELETE CASCADE` FKs, `IF NOT EXISTS` on every DDL statement for idempotent re-runs (matches `main.go`'s own fallback `ALTER TABLE` blocks at lines 105-114 that guard against migrations not yet applied).

### Websocket message extension: reuse the switch, add one field
**Source:** `internal/session/websocket.go:31-46` (`WSMessage`) + `internal/session/websocket.go:24-29` (`MsgTypeAutopilot` added as a sibling constant to the original five message types) + `frontend/src/services/api.ts:143-169` (`handleMessage`'s switch) + `:246-255` (`onAutopilot`/`offAutopilot`)
**Apply to:** the new `MsgTypeAI`/`Decision` field on both the Go `WSMessage` and the TypeScript `WSMessage` (`frontend/src/types/index.ts:53-63`, verified verbatim) — add one case to the existing switch, one new handler-array pair (`onAI`/`offAI`), exactly mirroring the autopilot precedent added in Phase 2.

### Fetch wrapper (`credentials: 'include'` + `handleAuthError`)
**Source:** `frontend/src/services/api.ts:44-53` (`getSessionStatus`, verified verbatim: `fetch(..., { credentials: 'include' })`, `handleAuthError(response)` before the `response.ok` check) and `:56-72` (`connectToMud`, same shape with a POST body and `data.error || '<fallback>'` error message)
**Apply to:** the new `getDecisions(connectionId)`, transcript-list, and transcript-read fetch functions in `api.ts`.

### Local bracketed notice via `echoLocal`
**Source:** `frontend/src/pages/PlayScreen.tsx:306-379` (verified verbatim this session: `automationEngine.echoLocal('[Disconnected]', { color: 'white' })`, the `previousAutopilotStateRef` transition effect producing `[Autopilot waiting for reconnect]`/`[Reconnected]`/`[Autopilot resuming]`/`[Autopilot disengaged: you took the wheel]`)
**Apply to:** every new D-13 failure notice and the D-20 refusal notice in the terminal — same mechanism, same color-name convention (`red`, `white`, `brightyellow`, `brightgreen`), guarantees "never sent to the MUD" for free.

### `#AUTO` grammar: single `case 'AUTO':` block, narrowed in place
**Source:** `frontend/src/services/automation/evaluator.ts:1162-1230` (verified verbatim, full current case body, including the existing `rawArg === '' || rawArg === 'STATUS'` branch mapping to `action = 'status'` at line 1175, and the existing red invalid-argument line at line 1188: `context.outputMessage?.(...[Autopilot: use #AUTO ON, #AUTO OFF, or #AUTO STATUS]...)`)
**Apply to:** D-11's grammar narrowing — retire the `'' -> status'`/`'STATUS' -> status'` branches (and the `'status'` outcome-handling switch arm at lines 1218-1225), replace the existing red line's wording with the exact D-11 string, reusing the same `getAnsiColorCode('red')` call already at line 1167. **Note for the planner:** the frontend's `AutopilotAnswer`/`setState` signature already accepts `'on'|'off'|'status'`(`automation.ts:167`, `api.ts:702`) and the server's `Autopilot` handler still accepts and returns a `"status"` action/outcome (`handler.go:328-329,391-395`) — D-11 retires `#AUTO STATUS` as a *typed directive*, not necessarily the underlying wire action; confirm with the planner whether the server-side `"status"` action is also removed or merely made unreachable from the CLI grammar (badge/panel become the status surfaces per D-11, but nothing in CONTEXT.md forbids keeping the wire action for internal/future use).

### `AutopilotBadge`-style "server owns truth, derive nothing" component
**Source:** `frontend/src/components/AutopilotBadge.tsx` (full file, verified verbatim — reads `autopilotState` straight from `useSession()`, runs no local poll/interval)
**Apply to:** `AIAssistPanel.tsx`'s minimized-tab state dot (03-UI-SPEC.md's own markup already specifies `state-${autopilotState}` reading from the same context) — the panel adds its own one-time REST fetch on mount (Pattern 5) for decision history, but tracks live autopilot state exactly like the badge, no duplicate polling.

### Per-connection Settings section (load-on-mount, `connectionId` prop)
**Source:** `frontend/src/components/AIPlayerPanel.tsx:1-52` (full pattern verified verbatim: `interface AIPlayerPanelProps { connectionId: string }`, `useCallback(loadPanel, [connectionId])`, `useEffect(() => { loadPanel() }, [loadPanel])`) + `frontend/src/pages/SettingsPage.tsx:15-32` (`SettingsSection` union and `SECTIONS` array, verified verbatim) + `:1691-1693` (the `ai-player` render block, verified verbatim: `{activeSection === 'ai-player' && connectionId && (<AIPlayerPanel connectionId={connectionId} />)}`)
**Apply to:** the new `'logs'` entry in `SettingsSection`/`SECTIONS` and its render block — `03-UI-SPEC.md` gives the exact section markup (`<h3>Logs</h3>`, description paragraph, `<a className="btn btn-primary" href={...} target="_blank">Open Logging</a>`) to slot into this same conditional-render pattern, placed after `'ai-player'` per D-16.

### `.play-screen` as the floating panel's mount root
**Source:** `frontend/src/pages/PlayScreen.tsx:506-511` (verified verbatim: `<div className="play-screen"><div className="output-panel output-panel-full">...`)
**Apply to:** `<AIAssistPanel />` mounts as a sibling inside `<div className="play-screen">`, positioned via `position: fixed` per `03-UI-SPEC.md`'s CSS (not part of the flex flow), so `.output-panel-full`'s full-width terminal is structurally undisturbed — confirmed by the existing `output-panel-full` class name itself, which already asserts the terminal owns the full width regardless of what floats over it.

### `icm-adapter.ts` fallback removal
**Source:** `frontend/src/services/icm-adapter.ts` (verified via grep this session: `validateAndNormalize` at lines 532-545 and `processCommand` at lines 565-620 each end in a `catch { ... // Frontend fallback ... }` block after their `fetch(...)` call)
**Apply to:** success criterion 4's "the frontend adapter's ICM calls stop falling back silently" — remove the fallback branches (or make them surface the server error instead of computing a browser-side answer), per RESEARCH.md's Architectural Responsibility Map row for "Command dispatch + safety check."

## No Analog Found

| File | Role | Data Flow | Reason |
|---|---|---|---|
| `internal/gemini/client.go` (the Gemini REST contract itself: header auth, `responseSchema`, double-JSON-decode) | service | request-response | First external HTTP integration in this codebase; no existing outbound `net/http` client to any third-party API was found. RESEARCH.md's Gemini API Reference (fetched and cross-checked against official docs this session) is the only available "analog," not a codebase file. |
| `internal/driver/driver.go` (the orchestration of ring-buffer + profile + Gemini + ICM dispatch + persistence + broadcast into one sequence) | service | event-driven | No existing file in this codebase orchestrates this many subsystems in one call; RESEARCH.md's own Architecture Patterns diagram is the design document to build from, assembled here from the individually-verified primitives above. |
| `frontend/src/components/AIAssistPanel.tsx` (the decision-card + system-line layout itself) | component | event-driven | No existing component renders a live list of heterogeneous timeline entries (decision cards mixed with system lines); `03-UI-SPEC.md` supplies the exact verbatim markup and CSS to implement directly, so no further codebase analog search is productive. |
| `frontend/src/pages/LogsPage.tsx` (the two-pane session-list/transcript layout) | component | request-response | No existing page has a click-to-load two-pane master/detail layout; `SettingsPage.tsx`'s `.settings-layout`/`.keybindings-list` two-pane precedent is structurally similar (nav list + content pane) but not the same interaction (click-a-row-to-load-detail); `03-UI-SPEC.md` supplies the exact verbatim CSS and markup shape. |

## Metadata

**Analog search scope:** `internal/session/`, `internal/icm/`, `internal/store/`, `internal/config/`, `internal/auth/`, `cmd/server/`, `migrations/`, `frontend/src/services/`, `frontend/src/services/automation/`, `frontend/src/pages/`, `frontend/src/components/`, `frontend/src/index.css`, `scripts/`

**Files scanned this session (targeted or full reads, line numbers verified against current HEAD):** `internal/session/manager.go` (lines 1-70, 169-241, 314-370, 381-635, 638-648), `internal/session/websocket.go` (lines 1-90, 380-460, 493-598), `internal/session/handler.go` (full file, 431 lines), `internal/session/autopilot.go` (full file, 120 lines), `internal/icm/types.go` (lines 1-100), `internal/icm/dispatcher.go` (lines 1-60, 187-266), `internal/icm/errors.go` (lines 1-55), `internal/icm/handler.go` (lines 1-40), `internal/icm/icm_test.go` (lines 659-700), `internal/config/config.go` (full file, 163 lines), `internal/store/profile.go` (lines 1-30, 521-582), `internal/auth/handler.go` (grep for OTP lines, confirmed 172-181 and 261-270), `cmd/server/main.go` (lines 150-220, 405-462, plus route grep), `migrations/010_add_ai_fields.up/down.sql` (full files), `migrations/006_create_profiles.up.sql` (lines 1-20), `frontend/src/services/api.ts` (lines 1-53, 143-273, plus export grep), `frontend/src/services/automation/evaluator.ts` (lines 1162-1241, plus `autopilotControl` grep), `frontend/src/services/automation.ts` (grep for `autopilotControl`/`CommandSource`), `frontend/src/services/icm-adapter.ts` (grep for fallback blocks), `frontend/src/pages/PlayScreen.tsx` (grep for echoLocal/autopilot, lines 440-535, 506-511), `frontend/src/pages/SettingsPage.tsx` (grep for SECTIONS, lines 1691-1700), `frontend/src/components/AutopilotBadge.tsx` (full file), `frontend/src/components/AIPlayerPanel.tsx` (lines 1-70), `frontend/src/types/index.ts` (lines 12-68), `frontend/src/index.css` (grep for badge selectors)

**Also read (upstream inputs, not re-cited as analogs):** `03-CONTEXT.md` (full), `03-RESEARCH.md` (full, 782 lines), `03-UI-SPEC.md` (full, 508 lines), `02-PATTERNS.md` (full, format precedent)

**Pattern extraction date:** 2026-09-15
