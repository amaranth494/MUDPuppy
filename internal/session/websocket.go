package session

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/amaranth494/MudPuppy/internal/config"
	"github.com/amaranth494/MudPuppy/internal/metrics"
	"github.com/gorilla/websocket"
)

// WebSocket message types
const (
	MsgTypeConnect    = "connect"
	MsgTypeDisconnect = "disconnect"
	MsgTypeData       = "data"
	MsgTypeError      = "error"
	MsgTypeStatus     = "status"
	// MsgTypeAutopilot is the outbound, best-effort autopilot-state push.
	// It reuses the Status and Data string fields (Status carries the
	// on/waiting/off state, Data carries the cause) rather than adding new
	// struct fields — see plan 02-03's wire contract.
	MsgTypeAutopilot = "autopilot"
	// MsgTypeAI is the outbound push of an AI decision or system notice
	// (plan 03-09, D-08). It carries a typed Decision payload rather than
	// cramming several variable-length text fields into Data/Status, which
	// an autopilot transition's two short values do not need.
	MsgTypeAI = "ai"
)

// WebSocket message structure
type WSMessage struct {
	Type   string `json:"type"`
	Host   string `json:"host,omitempty"`
	Port   int    `json:"port,omitempty"`
	Data   string `json:"data,omitempty"`
	Error  string `json:"error,omitempty"`
	Status string `json:"status,omitempty"`
	// Source is meaningful only on inbound "data" messages: the browser's
	// CommandSource tag ("user", "alias", "trigger") for the command being
	// sent. See IsHumanSource for the classification rule.
	Source string `json:"source,omitempty"`
	// ConnectionID is meaningful only on inbound "connect" messages: the
	// saved connection profile the browser is opening, if any.
	ConnectionID string `json:"connection_id,omitempty"`
	// Decision carries an AI decision or system notice (MsgTypeAI only,
	// plan 03-09, D-08); nil on every other message type.
	Decision *AIDecisionPayload `json:"decision,omitempty"`
	// PausedByOwner and ConnectionLost are meaningful only on a
	// MsgTypeAutopilot push (Phase 5, D-15): the same two independent
	// waiting reasons AutopilotResponse carries over REST, read through
	// AutopilotWaitingReasons on every autopilot push this file makes,
	// including the wheel-grab push below.
	PausedByOwner  bool `json:"paused_by_owner,omitempty"`
	ConnectionLost bool `json:"connection_lost,omitempty"`
}

// AIDecisionPayload is the payload of an outbound MsgTypeAI message
// (D-08). Kind is "decision" (Reasoning and Command are set, Outcome is
// "sent") or "system" (Message carries a locked failure/refusal notice,
// Outcome is one of "refused", "failed", "cap", "blocked-repeatedly",
// "transient" or "retrying" as of Phase 4). Never logged in full (T-3-07).
type AIDecisionPayload struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Reasoning string `json:"reasoning"`
	Command   string `json:"command"`
	Outcome   string `json:"outcome"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`

	// State, Calls, CallCap, CallCapSet, Failures, Blocks and Threshold are
	// meaningful on any event emitted during a stint (D-18, DR-3-03,
	// 04-UI-SPEC.md §3): State carries the autopilot switch's position read
	// after any disengage this message's own event already applied, so the
	// badge and the panel's status line never lag behind what actually
	// happened. The other fields are the standing call/failure/block counts
	// against their resolved cap/threshold.
	//
	// The counts and the memory list are POINTERS (code review WR-05 of
	// Phase 4). As plain omitempty values a zero count and an emptied list
	// were left out of the message, and the panel -- which updates a field
	// only when the message carries it -- went on showing "Consecutive
	// failures: 1 of 3" after the next sent command had reset the count to
	// 0, and the old bullets after the model emptied its memory. A pointer
	// separates the two cases the wire has to tell apart: nil means "this
	// message says nothing about the stint" (the goal-changed line) and is
	// omitted; non-nil is always sent, zero and [] included. Build the stint
	// half with WithStint, never field by field.
	State      string `json:"state,omitempty"`
	Calls      *int   `json:"calls,omitempty"`
	CallCap    *int   `json:"call_cap,omitempty"`
	CallCapSet bool   `json:"call_cap_set,omitempty"`
	Failures   *int   `json:"failures,omitempty"`
	Blocks     *int   `json:"blocks,omitempty"`
	Threshold  *int   `json:"threshold,omitempty"`

	// SessionMemory carries the current game session's full curated Session
	// Memory list (D-10, plan 04-08) on every driver-emitted ai message — the
	// whole list, not a diff — so the panel's collapsible section replaces
	// its displayed list wholesale from whichever message arrives. Present
	// (as [] when empty, never null) on every stint message; absent only on
	// a message that says nothing about the stint.
	SessionMemory *[]string `json:"session_memory,omitempty"`
}

// WithStint returns p carrying the stint's standing status (code review
// WR-05 of Phase 4): every count is present on the wire even when it is
// zero, and the Session Memory list is present even when it is empty (a nil
// list is sent as [], never null).
func (p AIDecisionPayload) WithStint(state string, calls, callCap int, callCapSet bool, failures, blocks, threshold int, sessionMemory []string) AIDecisionPayload {
	if sessionMemory == nil {
		sessionMemory = []string{}
	}
	p.State = state
	p.Calls = &calls
	p.CallCap = &callCap
	p.CallCapSet = callCapSet
	p.Failures = &failures
	p.Blocks = &blocks
	p.Threshold = &threshold
	p.SessionMemory = &sessionMemory
	return p
}

// IsHumanSource is the wheel-grab's classification rule. Absent or
// unrecognised means human, because the failure in that direction is an
// unwanted disengage the owner will notice immediately, while the opposite
// direction — treating an unrecognised source as automation — would be a
// silent bypass of the one protection this phase exists to provide
// (RESEARCH Pitfall 3). Only "trigger" and "timer" are automation; "timer"
// is included even though the browser currently labels timer-fired
// commands "trigger", so a later frontend rename cannot quietly turn
// timers into wheel-grabs.
func IsHumanSource(source string) bool {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "trigger", "timer":
		return false
	default:
		return true
	}
}

// applyWheelGrab decides whether a browser-sent command takes the wheel
// back from autopilot. It takes a *Manager argument rather than being a
// method on WebSocketHandler so the rule is testable without opening a
// socket. It can only move a user's own autopilot from on to off — never
// engage or resume anything — so a client that forges a source label can
// at most keep its own already-on autopilot on (threat T-2-01).
func applyWheelGrab(m *Manager, userID, source string) (grabbed bool, state AutopilotState) {
	if !IsHumanSource(source) {
		return false, m.AutopilotStateFor(userID)
	}
	if m.AutopilotStateFor(userID) != AutopilotOn {
		return false, m.AutopilotStateFor(userID)
	}
	newState, changed := m.DisengageAutopilot(userID, "wheel-grab")
	if !changed {
		return false, newState
	}
	return true, newState
}

// RateLimiter implements a simple token bucket rate limiter
type RateLimiter struct {
	mu         sync.Mutex
	tokens     int
	maxTokens  int
	refillRate time.Duration
	lastRefill time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxTokens int, refillRate time.Duration) *RateLimiter {
	return &RateLimiter{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow checks if a request is allowed under rate limit
func (r *RateLimiter) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(r.lastRefill)
	if elapsed >= r.refillRate {
		// Refill tokens
		r.tokens = r.maxTokens
		r.lastRefill = now
	}

	if r.tokens > 0 {
		r.tokens--
		return true
	}
	return false
}

// wsWriter is the minimal websocket write capability writeJSON needs. The
// real *websocket.Conn satisfies it. Declaring it as an interface is the
// one seam PushAI's tests use to exercise the registry-lookup-and-write
// path with a fake connection, since registerClient's production callers
// only ever have a real *websocket.Conn to register (task 03-09-01).
type wsWriter interface {
	WriteJSON(v interface{}) error
	SetWriteDeadline(t time.Time) error
}

// WebSocketHandler handles WebSocket connections for MUD session streaming
type WebSocketHandler struct {
	manager        *Manager
	config         *config.Config
	upgrader       websocket.Upgrader
	rateLimiters   map[string]*RateLimiter
	rateLimitersMu sync.RWMutex
	wsWriteMu      sync.Mutex // Protects WebSocket writes from concurrent goroutines

	// clients is the per-user registry PushAI reads (plan 03-09). Its own
	// RWMutex guards map access only — wsWriteMu above serialises the
	// actual writes, a distinct concern (T-3-08).
	clients   map[string]wsWriter
	clientsMu sync.RWMutex
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(manager *Manager, cfg *config.Config) *WebSocketHandler {
	return &WebSocketHandler{
		manager: manager,
		config:  cfg,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			// Allow all origins in development (CORS)
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		rateLimiters: make(map[string]*RateLimiter),
		clients:      make(map[string]wsWriter),
	}
}

// registerClient records userID's open play-screen connection so PushAI
// can find it later. Keyed only by the authenticated user id taken from
// the upgraded connection's own request context — never a client-supplied
// field (T-3-11).
func (h *WebSocketHandler) registerClient(userID string, conn *websocket.Conn) {
	h.clientsMu.Lock()
	defer h.clientsMu.Unlock()
	h.clients[userID] = conn
}

// unregisterClient removes userID's registration only if it still points
// at conn. A new tab registers its own connection before the old tab's
// deferred teardown runs; without this comparison the old tab's
// unregister would delete the new tab's entry (T-3-11).
func (h *WebSocketHandler) unregisterClient(userID string, conn *websocket.Conn) {
	h.clientsMu.Lock()
	defer h.clientsMu.Unlock()
	if stored, ok := h.clients[userID]; ok && stored == wsWriter(conn) {
		delete(h.clients, userID)
	}
}

// PushAI sends an AI decision or system notice to userID's open play
// screen (D-08). A user with no open screen is a silent no-op — the
// driver goroutine calling this must never be blocked or broken by a
// closed browser tab (T-3-36). The payload is never logged; a write
// failure logs at most one line naming the user and an error class only
// (T-3-07).
func (h *WebSocketHandler) PushAI(userID string, payload AIDecisionPayload) error {
	h.clientsMu.RLock()
	conn, ok := h.clients[userID]
	h.clientsMu.RUnlock()
	if !ok {
		return nil
	}

	if err := h.writeJSON(conn, WSMessage{Type: MsgTypeAI, Decision: &payload}); err != nil {
		log.Printf("[AI-PLAYER] ai-push user_id=%s outcome=failed error_class=%T", userID, err)
		return err
	}
	return nil
}

// getRateLimiter gets or creates a rate limiter for a user
func (h *WebSocketHandler) getRateLimiter(userID string) *RateLimiter {
	h.rateLimitersMu.RLock()
	rl, ok := h.rateLimiters[userID]
	h.rateLimitersMu.RUnlock()

	if !ok {
		// Create new rate limiter: max tokens = command rate limit per second
		// refill rate = 1 second
		rl = NewRateLimiter(h.config.CommandRateLimitPerSec, 1*time.Second)

		h.rateLimitersMu.Lock()
		h.rateLimiters[userID] = rl
		h.rateLimitersMu.Unlock()
	}

	return rl
}

// removeRateLimiter removes a user's rate limiter
func (h *WebSocketHandler) removeRateLimiter(userID string) {
	h.rateLimitersMu.Lock()
	delete(h.rateLimiters, userID)
	h.rateLimitersMu.Unlock()
}

// writeJSON writes JSON to the WebSocket with proper locking and deadline.
// conn is typed wsWriter rather than the concrete *websocket.Conn so this
// is still the single write path (T-3-08) whether the caller is the real
// connect/data/status flow or PushAI's registry lookup.
func (h *WebSocketHandler) writeJSON(conn wsWriter, v interface{}) error {
	h.wsWriteMu.Lock()
	defer h.wsWriteMu.Unlock()
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return conn.WriteJSON(v)
}

// writeMessage writes a message to the WebSocket with proper locking and deadline
func (h *WebSocketHandler) writeMessage(conn *websocket.Conn, msgType int, data []byte) error {
	h.wsWriteMu.Lock()
	defer h.wsWriteMu.Unlock()
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return conn.WriteMessage(msgType, data)
}

// HandleWebSocket handles WebSocket connections at /api/v1/session/stream
func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by session middleware)
	userID := r.Context().Value("user_id")
	if userID == nil {
		log.Printf("[SP02PH02] WebSocket connection rejected: no user_id in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userIDStr := userID.(string)

	log.Printf("[SP02PH02] WebSocket connection from user: %s at %v", userIDStr, time.Now().UnixNano())

	// Upgrade HTTP to WebSocket
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[SP02PH02] WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	// Register this connection as userIDStr's open play screen so PushAI
	// (plan 03-09) can find it, and unregister on every exit path below.
	h.registerClient(userIDStr, conn)
	defer h.unregisterClient(userIDStr, conn)

	// Set read limit to prevent memory exhaustion (SP02 hardening)
	conn.SetReadLimit(65536) // 64KB max message size

	// Write deadline for all outbound writes
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

	// Note: Don't use pongChan - it blocks and prevents deadline refresh
	conn.SetPongHandler(func(appData string) error {
		return nil
	})

	// Start ping ticker
	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	// Create context for this connection
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Channel for MUD output to send to client
	mudToClient := make(chan []byte, 100)
	// Channel for client commands to send to MUD (buffered to prevent blocking)
	clientToMUD := make(chan string, 64)
	// Channel for connection status
	statusChan := make(chan string, 2)

	// Track if we're connected to a MUD
	connected := false

	// Start goroutine to read from MUD and forward to client
	go h.readMUDOutput(ctx, userIDStr, mudToClient, statusChan)

	// Start goroutine to handle client commands and forward to MUD
	go h.handleClientCommands(ctx, userIDStr, clientToMUD, statusChan)

	// Main WebSocket message loop
	for {
		// Check if we got a disconnect status or ping ticker
		select {
		case status := <-statusChan:
			if status == "disconnected" {
				log.Printf("[SP02PH02] MUD connection closed for user %s", userIDStr)
				connected = false
				// Send disconnect message to client (using helper for thread-safe writes)
				err := h.writeJSON(conn, WSMessage{
					Type:   MsgTypeDisconnect,
					Status: "disconnected",
				})
				if err != nil {
					log.Printf("[SP02PH02] Error sending disconnect to client: %v", err)
					return
				}
			}
		case <-pingTicker.C:
			// Send ping to keep connection alive (using helper for thread-safe writes)
			if err := h.writeMessage(conn, websocket.PingMessage, nil); err != nil {
				log.Printf("[SP02PH02] Ping failed: %v", err)
				return
			}
		case <-ctx.Done():
			// Context cancelled - clean up and exit (no goroutine leak)
			if connected {
				h.manager.Disconnect(userIDStr, ReasonRemote)
			}
			h.removeRateLimiter(userIDStr)
			return
		default:
			// No blocking - continue to read message immediately
		}

		// No read deadline - let MUD server handle idle timeout

		// Read message from client
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[SP02PH02] WebSocket read error: %v", err)
			}
			// Clean up MUD connection on close
			if connected {
				h.manager.Disconnect(userIDStr, ReasonRemote)
			}
			h.removeRateLimiter(userIDStr)
			return
		}

		// Record metrics for incoming message
		metrics.Get().IncWSMessagesIn()

		if msgType != websocket.TextMessage {
			continue
		}

		// Parse the message
		var wsMsg WSMessage
		if err := json.Unmarshal(msg, &wsMsg); err != nil {
			log.Printf("[SP02PH02] Failed to parse WebSocket message: %v", err)
			h.sendError(conn, "Invalid message format")
			continue
		}

		switch wsMsg.Type {
		case MsgTypeConnect:
			var err error
			var session *Session

			// Check if session already exists (may have been created via REST API)
			session, err = h.manager.GetSession(userIDStr)
			if err == nil && session.State == StateConnected {
				// Session already exists from REST API - just use it
				connected = true
				log.Printf("[SP02PH02] Using existing session for user %s (from REST API) at %v", userIDStr, time.Now().UnixNano())
			} else {
				// No existing session - this is a WebSocket-only connect attempt
				// Validate host/port
				if wsMsg.Host == "" {
					h.sendError(conn, "Host is required")
					continue
				}

				// Default port to 23
				if wsMsg.Port == 0 {
					wsMsg.Port = 23
				}

				log.Printf("[SP02PH02T01] Connect request: user=%s, host=%s, port=%d", userIDStr, wsMsg.Host, wsMsg.Port)

				// Attempt connection via Manager
				session, err = h.manager.Connect(ctx, userIDStr, wsMsg.Host, wsMsg.Port, wsMsg.ConnectionID)
				if err != nil {
					log.Printf("[SP02PH02] Connection failed: %v", err)
					h.sendError(conn, err.Error())
					continue
				}

				// Wait a moment for connection to establish
				time.Sleep(100 * time.Millisecond)

				// Check if actually connected
				session, _ = h.manager.GetSession(userIDStr)
				if session.State != StateConnected {
					h.sendError(conn, "Failed to establish connection")
					continue
				}

				connected = true
				log.Printf("[SP02PH02] Connected to %s:%d for user %s", wsMsg.Host, wsMsg.Port, userIDStr)

				// Protocol plausibility check (SP02PH04T07)
				// Check after connection to ensure it's actually a MUD server
				go func() {
					time.Sleep(500 * time.Millisecond) // Brief delay to let initial data arrive
					if h.manager.CheckAndDisconnectProtocolMismatch(userIDStr) {
						log.Printf("[SP02PH04T07] Protocol mismatch detected for user %s, disconnected", userIDStr)
						h.sendError(conn, "Protocol mismatch: server appears to be non-MUD")
					}
				}()
			}

			// Send success message (common for both paths) - using helper for thread-safe writes
			err = h.writeJSON(conn, WSMessage{
				Type:   MsgTypeStatus,
				Status: StateConnected,
			})
			if err != nil {
				log.Printf("[SP02PH02] Error sending connected status: %v", err)
				continue
			}

			// Start the MUD->client relay (for both new and existing sessions)
			go h.relayMUDToClient(ctx, userIDStr, conn, mudToClient)
			log.Printf("[SP02PH02] Started relay at %v", time.Now().UnixNano())

		case MsgTypeDisconnect:
			if connected {
				log.Printf("[SP02PH02] User requested disconnect")
				h.manager.Disconnect(userIDStr, ReasonUser)
				connected = false
			}

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

			// Wheel-grab: a human-sourced command disengages autopilot
			// before it is sent (D-07/D-09, threat T-2-01). Best-effort
			// push of the new state to this tab; the command is queued on
			// every path below regardless of whether the grab happened or
			// the push succeeded (no lost keystrokes, T-2-12).
			if grabbed, state := applyWheelGrab(h.manager, userIDStr, wsMsg.Source); grabbed {
				pausedByOwner, connectionLost := h.manager.AutopilotWaitingReasons(userIDStr)
				_ = h.writeJSON(conn, WSMessage{
					Type:           MsgTypeAutopilot,
					Status:         string(state),
					Data:           "wheel-grab",
					PausedByOwner:  pausedByOwner,
					ConnectionLost: connectionLost,
				})
			}

			// Send command to MUD via channel
			log.Printf("[SP02PH02] TRACE: Received WebSocket message at %v: %q", time.Now().UnixNano(), wsMsg.Data)
			select {
			case clientToMUD <- wsMsg.Data:
				log.Printf("[SP02PH02] TRACE: Queued command to channel at %v", time.Now().UnixNano())
				// Record metrics for outgoing message
				metrics.Get().IncWSMessagesOut()
			default:
				h.sendError(conn, "Command queue full")
			}

		default:
			log.Printf("[SP02PH02] Unknown message type: %s", wsMsg.Type)
		}
	}
}

// readMUDOutput reads output from MUD and sends to mudToClient channel
func (h *WebSocketHandler) readMUDOutput(ctx context.Context, userID string, mudToClient chan<- []byte, statusChan chan<- string) {
	defer log.Printf("[SP02PH02] WS reader (readMUDOutput) exiting for user %s at %v", userID, time.Now().UnixNano())
	log.Printf("[SP02PH02] readMUDOutput started at %v", time.Now().UnixNano())
	buffer := make([]byte, 8192)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		n, err := h.manager.ReadOutput(userID, buffer)
		if err != nil {
			log.Printf("[SP02PH02] Error reading from MUD: %v", err)
			statusChan <- "disconnected"
			return
		}

		if n > 0 {
			// Copy the data to send through channel
			data := make([]byte, n)
			copy(data, buffer[:n])
			log.Printf("[SP02PH02] TRACE: Read %d bytes from MUD at %v", n, time.Now().UnixNano())

			// Record metrics for incoming bytes from MUD
			metrics.Get().AddMudBytesIn(int64(n))

			select {
			case mudToClient <- data:
			case <-ctx.Done():
				return
			}
			// No sleep when we have data - process immediately
			continue
		}

		// Small sleep to prevent tight loop when no data
		time.Sleep(1 * time.Millisecond)
	}
}

// Soft backpressure constants (SP02PH04T02)
const (
	maxCoalesceSize = 64 * 1024             // Max bytes to coalesce before sending
	coalesceTimeout = 50 * time.Millisecond // Max time to wait before sending partial data
	slowClientDrops = 100                   // Number of drops before warning
	sustainedDrops  = 500                   // Number of sustained drops before disconnect
)

// relayMUDToClient relays MUD output to WebSocket client with soft backpressure
func (h *WebSocketHandler) relayMUDToClient(ctx context.Context, userID string, conn *websocket.Conn, mudToClient <-chan []byte) {
	defer log.Printf("[SP02PH02] WS reader (relayMUDToClient) exiting for user %s at %v", userID, time.Now().UnixNano())
	log.Printf("[SP02PH02] relayMUDToClient started at %v", time.Now().UnixNano())

	var coalesceBuffer []byte
	lastSendTime := time.Now()
	dropCount := 0
	sustainedDropCount := 0

	for {
		select {
		case <-ctx.Done():
			return
		case data := <-mudToClient:
			// Reset idle timer on inbound data
			h.manager.ResetIdleTimerOnInbound(userID)

			// Strip telnet IAC sequences before sending to client
			cleanData := stripTelnetIAC(data)

			log.Printf("[SP02PH02] TRACE: Forwarding %d bytes to WebSocket at %v", len(cleanData), time.Now().UnixNano())

			// Coalesce data: append to buffer if it fits
			if len(coalesceBuffer)+len(cleanData) <= maxCoalesceSize {
				coalesceBuffer = append(coalesceBuffer, cleanData...)
			} else {
				// Buffer full - send current and start new
				if len(coalesceBuffer) > 0 {
					h.sendCoalescedData(conn, userID, coalesceBuffer, &dropCount, &sustainedDropCount)
					coalesceBuffer = nil
				}
				coalesceBuffer = append(coalesceBuffer, cleanData...)
			}

			// Record metrics for outgoing bytes to client
			metrics.Get().AddMudBytesOut(int64(len(cleanData)))

			// Check if we should send due to time
			sinceLastSend := time.Since(lastSendTime)
			if sinceLastSend >= coalesceTimeout && len(coalesceBuffer) > 0 {
				h.sendCoalescedData(conn, userID, coalesceBuffer, &dropCount, &sustainedDropCount)
				coalesceBuffer = nil
			}

		default:
			// No data available - check if we should send coalesced data
			sinceLastSend := time.Since(lastSendTime)
			if sinceLastSend >= coalesceTimeout && len(coalesceBuffer) > 0 {
				h.sendCoalescedData(conn, userID, coalesceBuffer, &dropCount, &sustainedDropCount)
				coalesceBuffer = nil
			}

			// Small sleep to prevent tight loop
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// sendCoalescedData sends coalesced data to WebSocket with drop handling
func (h *WebSocketHandler) sendCoalescedData(conn *websocket.Conn, userID string, data []byte, dropCount *int, sustainedDropCount *int) {
	err := h.writeJSON(conn, WSMessage{
		Type: MsgTypeData,
		Data: string(data),
	})

	if err != nil {
		log.Printf("[SP02PH04T02] Error writing to WebSocket (slow client?): %v", err)
		*dropCount++
		*sustainedDropCount++

		// If sustained drops, log warning
		if *sustainedDropCount >= sustainedDrops {
			log.Printf("[SP02PH04T02] Slow client detected for user %s: %d sustained drops", userID, *sustainedDropCount)
			metrics.Get().IncSlowClient()
			// Don't disconnect yet - let it continue and see if client recovers
		}

		// Log warning every slowClientDrops
		if *dropCount >= slowClientDrops {
			log.Printf("[SP02PH04T02] Slow client warning for user %s: %d drops in buffer", userID, *dropCount)
			*dropCount = 0 // Reset for next warning cycle
		}

		return
	}

	// Successful send - reset counters
	*dropCount = 0
	*sustainedDropCount = 0
}

// stripTelnetIAC removes telnet IAC (Interpret As Command) sequences from the data
// IAC is byte 255 (0xFF). Telnet commands are: IAC + command + [option]
func stripTelnetIAC(data []byte) []byte {
	if len(data) == 0 {
		return data
	}

	// First pass: check if there are any IAC bytes
	hasIAC := false
	for _, b := range data {
		if b == 255 {
			hasIAC = true
			break
		}
	}
	if !hasIAC {
		return data
	}

	// Second pass: strip IAC sequences
	result := make([]byte, 0, len(data))
	i := 0
	for i < len(data) {
		if data[i] == 255 { // IAC
			if i+1 < len(data) {
				cmd := data[i+1]
				switch cmd {
				case 251, 252, 253, 254: // WILL, WON'T, DO, DON'T - skip command + option
					if i+2 < len(data) {
						i += 3 // Skip IAC + command + option
						continue
					}
					// Incomplete (IAC + cmd but no option) - skip both bytes
					i += 2
					continue
				case 255: // IAC IAC - escaped literal 255
					i += 2
					result = append(result, 255)
					continue
				default:
					// Unknown command, skip command byte only
					i += 2
					continue
				}
			}
			// Incomplete IAC at end of data (just 255 with no command) - skip it
			i++
			continue
		}
		// Regular byte (not part of IAC sequence)
		result = append(result, data[i])
		i++
	}

	return result
}

// handleClientCommands handles commands from client and forwards to MUD
func (h *WebSocketHandler) handleClientCommands(ctx context.Context, userID string, clientToMUD <-chan string, statusChan chan<- string) {
	defer log.Printf("[SP02PH02] WS reader (handleClientCommands) exiting for user %s at %v", userID, time.Now().UnixNano())
	log.Printf("[SP02PH02] handleClientCommands started at %v", time.Now().UnixNano())
	for {
		select {
		case <-ctx.Done():
			log.Printf("[SP02PH02] handleClientCommands: context cancelled for user %s", userID)
			return
		case command := <-clientToMUD:
			log.Printf("[SP02PH02] TRACE: Received command from client at %v: %q", time.Now().UnixNano(), command)
			err := h.manager.SendCommand(userID, command)
			if err != nil {
				log.Printf("[SP02PH02] Error sending command to MUD: %v - sending disconnect status", err)
				statusChan <- "disconnected"
				return
			}
			log.Printf("[SP02PH02] TRACE: Sent command to MUD at %v", time.Now().UnixNano())
		}
	}
}

// sendError sends an error message to the WebSocket client
func (h *WebSocketHandler) sendError(conn *websocket.Conn, errorMsg string) {
	err := h.writeJSON(conn, WSMessage{
		Type:  MsgTypeError,
		Error: errorMsg,
	})
	if err != nil {
		log.Printf("[SP02PH02] Error sending error message: %v", err)
	}
}
