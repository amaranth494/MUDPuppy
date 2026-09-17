package session

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/amaranth494/MudPuppy/internal/config"
	"github.com/google/uuid"
)

// Handler handles session HTTP requests
type Handler struct {
	manager   *Manager
	config    *config.Config
	callbacks *HandlerCallbacks
	// engageHook is the Phase 3 AI driver's entry point, fired from the
	// Autopilot handler's "on" branch — but only in the arm whose outcome
	// is "engaged" (see SetEngageHook).
	engageHook EngageHook
	// aiNotifier is the AI panel's thinking-stream channel (Phase 5,
	// D-15), mirroring internal/profiles.Handler's own aiNotifier
	// convention: nil until SetAINotifier is called, making the pause and
	// resume system-line notices a silent no-op — a missing notifier never
	// blocks a pause or resume itself.
	aiNotifier func(userID string, payload AIDecisionPayload)
	// autopilotNotifier tells every open play screen of a user what the
	// switch now reads, with both waiting reasons (D-15, code review WR-11
	// of Phase 5). nil until SetAutopilotNotifier is called, which makes the
	// push a silent no-op: a missing notifier never blocks a pause or resume.
	autopilotNotifier func(userID string, state AutopilotState, cause string, pausedByOwner, connectionLost bool)
}

// HandlerCallbacks provides callbacks for session events
type HandlerCallbacks struct {
	OnConnected     func(connectionID, userID uuid.UUID) error
	GetAutoLogin    func(connectionID uuid.UUID) (username, password string, err error)
	SendCredentials func(userID, username, password string) error
	// EngageGate resolves the Phase 1 policy-acceptance gate for the
	// profile owning connectionID, scoped to userID. A nil EngageGate (or a
	// nil HandlerCallbacks) must be treated as "gate refuses" by every
	// caller — a missing dependency fails closed, never open (T-2-10).
	EngageGate func(connectionID, userID uuid.UUID) (allowed bool, message string, policyVersion string)
}

// NewHandler creates a new session handler
func NewHandler(manager *Manager, cfg *config.Config) *Handler {
	return &Handler{
		manager:   manager,
		config:    cfg,
		callbacks: nil,
	}
}

// NewHandlerWithCallbacks creates a new session handler with callbacks
func NewHandlerWithCallbacks(manager *Manager, cfg *config.Config, callbacks *HandlerCallbacks) *Handler {
	return &Handler{
		manager:   manager,
		config:    cfg,
		callbacks: callbacks,
	}
}

// SetEngageHook wires the AI driver's entry point. A nil hook (the
// zero-value default) makes the fire site in Autopilot a no-op.
func (h *Handler) SetEngageHook(h2 EngageHook) {
	h.engageHook = h2
}

// SetAINotifier wires the AI system-line channel (Phase 5, D-15), following
// SetEngageHook's injection precedent. A nil fn (the zero-value default)
// makes notifyAutopilotSystemLine below a no-op.
func (h *Handler) SetAINotifier(fn func(userID string, payload AIDecisionPayload)) {
	h.aiNotifier = fn
}

// SetAutopilotNotifier wires the switch-state push (code review WR-11 of
// Phase 5), following SetAINotifier's injection precedent. cmd/server/main.go
// passes the websocket handler's PushAutopilot.
func (h *Handler) SetAutopilotNotifier(fn func(userID string, state AutopilotState, cause string, pausedByOwner, connectionLost bool)) {
	h.autopilotNotifier = fn
}

// pushAutopilotState reads the switch and its two waiting reasons and pushes
// them to every open play screen of userID. Called only after a transition
// that really happened (code review WR-11 of Phase 5): Pause and Resume used
// to send only a system line, which carries no state, so the badge and any
// other tab stayed wrong until the next fifteen-second poll.
func (h *Handler) pushAutopilotState(userID, cause string) {
	if h.autopilotNotifier == nil {
		return
	}
	state := h.manager.AutopilotStateFor(userID)
	pausedByOwner, connectionLost := h.manager.AutopilotWaitingReasons(userID)
	h.autopilotNotifier(userID, state, cause, pausedByOwner, connectionLost)
}

// notifyAutopilotSystemLine builds the fixed-shape AIDecisionPayload every
// pause/resume notice uses (mirroring internal/profiles.Handler's own
// goal-changed notice verbatim) and pushes it through h.aiNotifier when one
// is wired. It carries no ids, no reasons and no free text beyond the fixed
// message — there is nothing to redact because nothing variable goes in.
func (h *Handler) notifyAutopilotSystemLine(userID, outcome, message string) {
	if h.aiNotifier == nil {
		return
	}
	h.aiNotifier(userID, AIDecisionPayload{
		ID:        uuid.New().String(),
		Kind:      "system",
		Outcome:   outcome,
		Message:   message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// Request/Response types
type ConnectRequest struct {
	Host         string    `json:"host"`
	Port         int       `json:"port"`
	ConnectionID uuid.UUID `json:"connection_id,omitempty"`
}

type ConnectResponse struct {
	State     string `json:"state"`
	SessionID string `json:"session_id,omitempty"`
	Error     string `json:"error,omitempty"`
}

type DisconnectRequest struct {
	Reason string `json:"reason,omitempty"`
}

type DisconnectResponse struct {
	State  string `json:"state"`
	Reason string `json:"reason,omitempty"`
	Error  string `json:"error,omitempty"`
}

type StatusResponse struct {
	State            string  `json:"state"`
	ConnectedAt      *string `json:"connected_at,omitempty"`
	Host             string  `json:"host,omitempty"`
	Port             int     `json:"port,omitempty"`
	LastActivityAt   *string `json:"last_activity_at,omitempty"`
	LastError        string  `json:"last_error,omitempty"`
	DisconnectReason string  `json:"disconnect_reason,omitempty"`
	// AutopilotState carries the server-held switch position on every
	// response, no omitempty, so "off" is always transmitted and the
	// browser badge can trust the field's presence after a page refresh
	// (D-10).
	AutopilotState string `json:"autopilot_state"`
	// AutopilotConnectionID names the profile the switch is engaged or parked
	// on, so #AUTO OFF can still reach it after a page refresh while waiting.
	AutopilotConnectionID string `json:"autopilot_connection_id,omitempty"`
}

// AutopilotRequest is the request body for POST /api/v1/session/autopilot.
// It carries only the action and the connection the caller wants to
// engage/disengage/query — the acting user identity is never read from
// this body, only from the authenticated request context (threat T-2-02).
type AutopilotRequest struct {
	Action       string    `json:"action"`
	ConnectionID uuid.UUID `json:"connection_id"`
}

// AutopilotResponse is the response body for POST /api/v1/session/autopilot.
// Outcome is exactly one of engaged, disengaged, already-on, already-off,
// refused-gate, refused-no-session, refused-not-configured, status, paused,
// already-waiting, resumed, still-waiting, not-waiting (Phase 5, D-13/D-15).
// GateMessage, when set, is Phase 1's store.EngageGateRefusalMessage passed
// through verbatim for refused-gate, or the fixed D-20 sentence for
// refused-not-configured — this package never composes its own refusal
// wording.
type AutopilotResponse struct {
	State         string `json:"state"`
	Outcome       string `json:"outcome"`
	GateAllowed   bool   `json:"gate_allowed"`
	GateMessage   string `json:"gate_message,omitempty"`
	PolicyVersion string `json:"policy_version,omitempty"`
	// PausedByOwner and ConnectionLost are D-15's two independent waiting
	// reasons, read fresh from AutopilotWaitingReasons on every response
	// this handler sends (including "on" and "off"), no omitempty, so the
	// browser can always tell the false case from an absent field.
	PausedByOwner  bool `json:"paused_by_owner"`
	ConnectionLost bool `json:"connection_lost"`
}

// Connect handles POST /api/v1/session/connect
func (h *Handler) Connect(w http.ResponseWriter, r *http.Request) {
	// Only allow POST method
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user ID from context (set by session middleware)
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

	// Parse request body
	var req ConnectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid request body")
		return
	}

	// Validate host
	if req.Host == "" {
		h.sendError(w, "Host is required")
		return
	}

	// Default port to 23 if not specified
	if req.Port == 0 {
		req.Port = 23
	}

	log.Printf("[SP02PH01] Connect request: user=%s, host=%s, port=%d", userIDStr, req.Host, req.Port)

	// Attempt connection
	connIDStr := ""
	if req.ConnectionID != uuid.Nil {
		connIDStr = req.ConnectionID.String()
	}
	session, err := h.manager.Connect(r.Context(), userIDStr, req.Host, req.Port, connIDStr)
	if err != nil {
		log.Printf("[SP02PH01] Connect failed: user=%s, error=%v", userIDStr, err)
		h.sendError(w, err.Error())
		return
	}

	// If connection_id was provided, update last_connected_at and handle auto-login
	if req.ConnectionID != uuid.Nil && h.callbacks != nil {
		log.Printf("[SP03PH05T03] Connection request has connectionID=%s for userID=%s", req.ConnectionID, userUUID)

		// Update last_connected_at
		if h.callbacks.OnConnected != nil {
			if err := h.callbacks.OnConnected(req.ConnectionID, userUUID); err != nil {
				log.Printf("[SP03PH05T03] Failed to update last_connected_at: %v", err)
				// Don't fail the connection, just log the error
			} else {
				log.Printf("[SP03PH05T03] Updated last_connected_at for connectionID=%s, userID=%s", req.ConnectionID, userUUID)
			}
		}

		// Handle auto-login
		if h.callbacks.GetAutoLogin != nil {
			username, password, err := h.callbacks.GetAutoLogin(req.ConnectionID)
			if err != nil {
				log.Printf("[SP03PH05T08] Failed to get auto-login credentials: %v", err)
			} else if username != "" && password != "" && h.callbacks.SendCredentials != nil {
				// Wait a bit for the connection to establish before sending credentials
				time.Sleep(100 * time.Millisecond)
				if err := h.callbacks.SendCredentials(userIDStr, username, password); err != nil {
					log.Printf("[SP03PH05T08] Failed to send auto-login credentials: %v", err)
				}
			}
		}
	}

	// Return success response
	resp := ConnectResponse{
		State:     session.State,
		SessionID: userIDStr,
	}
	h.sendJSON(w, resp)
}

// Disconnect handles POST /api/v1/session/disconnect
func (h *Handler) Disconnect(w http.ResponseWriter, r *http.Request) {
	// Only allow POST method
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user ID from context
	userID := r.Context().Value("user_id")
	if userID == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userIDStr := userID.(string)

	log.Printf("[SP02PH01] Disconnect request: user=%s", userIDStr)

	// Parse optional reason from request body
	var req DisconnectRequest
	json.NewDecoder(r.Body).Decode(&req)

	reason := ReasonUser
	if req.Reason != "" {
		reason = req.Reason
	}

	// Disconnect
	if err := h.manager.Disconnect(userIDStr, reason); err != nil {
		log.Printf("[SP02PH01] Disconnect failed: user=%s, error=%v", userIDStr, err)
		h.sendError(w, err.Error())
		return
	}

	// Return success response
	resp := DisconnectResponse{
		State:  StateDisconnected,
		Reason: reason,
	}
	h.sendJSON(w, resp)
}

// Status handles GET /api/v1/session/status
func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	// Only allow GET method
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user ID from context
	userID := r.Context().Value("user_id")
	if userID == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userIDStr := userID.(string)

	// Get session status
	session, err := h.manager.GetSession(userIDStr)
	if err != nil {
		h.sendError(w, err.Error())
		return
	}

	// Build response
	resp := StatusResponse{
		State: session.State,
	}

	if session.State == StateConnected {
		resp.Host = session.Host
		resp.Port = session.Port
		connectedAt := session.ConnectedAt.Format(time.RFC3339)
		resp.ConnectedAt = &connectedAt
		lastActivity := session.LastActivityAt.Format(time.RFC3339)
		resp.LastActivityAt = &lastActivity
	}

	if session.DisconnectErr != "" {
		resp.LastError = session.DisconnectErr
		resp.DisconnectReason = session.DisconnectErr
	}

	resp.AutopilotState = string(h.manager.AutopilotStateFor(userIDStr))
	resp.AutopilotConnectionID = h.manager.AutopilotConnectionIDFor(userIDStr)

	h.sendJSON(w, resp)
}

// Autopilot handles POST /api/v1/session/autopilot: #AUTO ON, #AUTO OFF and
// #AUTO STATUS all funnel through this one endpoint. The gate is always
// resolved first, so gate_allowed/gate_message/policy_version are on every
// response including status (D-05); only the "on" action's outcome depends
// on the gate result.
func (h *Handler) Autopilot(w http.ResponseWriter, r *http.Request) {
	// Only allow POST method
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user ID from context (set by session middleware). The user
	// identity comes from the request context only — never from the body.
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

	// Parse request body
	var req AutopilotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid request body")
		return
	}

	action := strings.ToLower(req.Action)
	if action != "on" && action != "off" && action != "status" && action != "pause" && action != "resume" {
		http.Error(w, "Invalid action", http.StatusBadRequest)
		return
	}

	// Resolve the gate first, for every action, so #AUTO STATUS can print
	// the gate result too (D-05). A nil callbacks struct or a nil
	// EngageGate callback fails closed: gateAllowed stays false and
	// EngageAutopilot is never reached below (T-2-10).
	var gateAllowed bool
	var gateMessage string
	var policyVersion string
	if h.callbacks != nil && h.callbacks.EngageGate != nil {
		gateAllowed, gateMessage, policyVersion = h.callbacks.EngageGate(req.ConnectionID, userUUID)
	}

	resp := AutopilotResponse{
		GateAllowed:   gateAllowed,
		GateMessage:   gateMessage,
		PolicyVersion: policyVersion,
	}

	switch action {
	case "on":
		if !gateAllowed {
			cur := h.manager.AutopilotStateFor(userIDStr)
			resp.State = string(cur)
			resp.Outcome = "refused-gate"
			log.Printf("[AI-PLAYER] autopilot user_id=%s connection_id=%s old=%s new=%s cause=%s",
				userIDStr, req.ConnectionID.String(), cur, cur, "refused-gate")
			resp.PausedByOwner, resp.ConnectionLost = h.manager.AutopilotWaitingReasons(userIDStr)
			h.sendJSON(w, resp)
			return
		}

		// A nil config or an unconfigured AI model registry refuses here,
		// checked only after the policy gate allows and before
		// EngageAutopilot is ever called — the switch cannot move and no
		// driver can be invoked on this path (D-20, T-3-29). A nil h.config
		// refuses too: a missing dependency fails closed, never open, the
		// same discipline EngageGate's doc comment states above (T-2-10).
		// Never log the model name, endpoint, key or any registry detail on
		// this path (T-3-02) — the log line below carries ids, states and
		// the cause only.
		if h.config == nil || !h.config.AIConfigured() {
			cur := h.manager.AutopilotStateFor(userIDStr)
			resp.State = string(cur)
			resp.Outcome = "refused-not-configured"
			resp.GateMessage = "Autopilot refused: AI is not configured on this server"
			log.Printf("[AI-PLAYER] autopilot user_id=%s connection_id=%s old=%s new=%s cause=refused-not-configured",
				userIDStr, req.ConnectionID.String(), cur, cur)
			resp.PausedByOwner, resp.ConnectionLost = h.manager.AutopilotWaitingReasons(userIDStr)
			h.sendJSON(w, resp)
			return
		}

		newState, changed, epoch, engageErr := h.manager.EngageAutopilotEpoch(userIDStr, req.ConnectionID.String())
		resp.State = string(newState)
		switch {
		case engageErr == ErrNoConnectedSession:
			resp.Outcome = "refused-no-session"
		case engageErr == ErrWrongConnection:
			resp.Outcome = "refused-wrong-connection"
		case changed:
			resp.Outcome = "engaged"
			// D-03: only a real engage fires the driver — a repeated
			// #AUTO ON lands in the "already-on" arm below and fires
			// nothing. Started with `go` so the HTTP response returns at
			// once; the decision arrives later over the websocket.
			if h.engageHook != nil {
				go h.engageHook(userIDStr, req.ConnectionID.String(), epoch)
			}
		default:
			resp.Outcome = "already-on"
		}
		resp.PausedByOwner, resp.ConnectionLost = h.manager.AutopilotWaitingReasons(userIDStr)
		h.sendJSON(w, resp)
		return

	case "off":
		// Turning the switch off is always allowed, including while the
		// gate refuses — the owner can always take the wheel back (D-01).
		newState, changed := h.manager.DisengageAutopilot(userIDStr, "disengage")
		resp.State = string(newState)
		if changed {
			resp.Outcome = "disengaged"
		} else {
			resp.Outcome = "already-off"
		}
		resp.PausedByOwner, resp.ConnectionLost = h.manager.AutopilotWaitingReasons(userIDStr)
		h.sendJSON(w, resp)
		return

	case "status":
		resp.State = string(h.manager.AutopilotStateFor(userIDStr))
		resp.Outcome = "status"
		resp.PausedByOwner, resp.ConnectionLost = h.manager.AutopilotWaitingReasons(userIDStr)
		h.sendJSON(w, resp)
		return

	case "pause":
		// D-13: pausing is instant, makes no model call and can never start
		// anything, so there is no policy-gate check here — unlike "on",
		// this arm runs unconditionally. PauseAutopilot is defensive on its
		// own when the switch is already off (D-13's button is disabled
		// client-side in that case anyway).
		newState, changed := h.manager.PauseAutopilot(userIDStr)
		resp.State = string(newState)
		if changed {
			resp.Outcome = "paused"
			// D-15: the stream prints a notice only on a real transition —
			// a pause that returned already-waiting or already-off never
			// claims a transition that did not happen.
			h.notifyAutopilotSystemLine(userIDStr, "paused", "Autopilot paused")
			h.pushAutopilotState(userIDStr, "pause")
		} else if newState == AutopilotOff {
			resp.Outcome = "already-off"
		} else {
			// Already Waiting (a lost connection): no transition, but the
			// owner's pause is now recorded as a second reason (D-15), so the
			// screens are told. Telling them twice is harmless.
			resp.Outcome = "already-waiting"
			h.pushAutopilotState(userIDStr, "pause")
		}
		resp.PausedByOwner, resp.ConnectionLost = h.manager.AutopilotWaitingReasons(userIDStr)
		h.sendJSON(w, resp)
		return

	case "resume":
		// Code review WR-09 of Phase 5: the owner's Resume is an engagement
		// -- it begins a new stint exactly as #AUTO ON does -- so it passes
		// the same gate. A pause can last the whole login; policy acceptance
		// withdrawn, a policy version bump or AI being unconfigured in the
		// meantime used to go unnoticed, because the gate result resolved
		// above was simply ignored on this arm.
		//
		// The gate is asked about the connection the switch is PARKED on,
		// never the one named in the request body: otherwise a profile that
		// accepted the policy could lend its acceptance to the paused one
		// (the same rule code review C2 set for "on"). It is asked only when
		// there is an owner pause to lift, so a Resume with nothing to resume
		// still answers "not-waiting" as before. When the gate refuses, the
		// switch is left exactly as it was -- still paused -- and the owner
		// is told why, in the same words #AUTO ON uses.
		if paused, _ := h.manager.AutopilotWaitingReasons(userIDStr); paused && h.manager.AutopilotStateFor(userIDStr) == AutopilotWaiting {
			parkedID := h.manager.AutopilotConnectionIDFor(userIDStr)
			refusal, refusalMessage := "", ""
			parkedUUID, parseErr := uuid.Parse(parkedID)
			switch {
			case parseErr != nil || h.callbacks == nil || h.callbacks.EngageGate == nil:
				// A missing dependency fails closed, never open (T-2-10).
				refusal, refusalMessage = "refused-gate", gateMessage
				resp.GateAllowed = false
			default:
				allowed, message, version := h.callbacks.EngageGate(parkedUUID, userUUID)
				resp.GateAllowed, resp.GateMessage, resp.PolicyVersion = allowed, message, version
				if !allowed {
					refusal, refusalMessage = "refused-gate", message
				} else if h.config == nil || !h.config.AIConfigured() {
					refusal, refusalMessage = "refused-not-configured", "Autopilot refused: AI is not configured on this server"
					resp.GateMessage = refusalMessage
				}
			}
			if refusal != "" {
				resp.State = string(AutopilotWaiting)
				resp.Outcome = refusal
				log.Printf("[AI-PLAYER] autopilot user_id=%s connection_id=%s old=%s new=%s cause=%s",
					userIDStr, parkedID, AutopilotWaiting, AutopilotWaiting, refusal+"-resume")
				// The Resume button has no terminal directive to print the
				// reason, so it goes to the thinking stream (and, through the
				// play screen, the terminal) as a refused line.
				if refusalMessage != "" {
					h.notifyAutopilotSystemLine(userIDStr, "refused", refusalMessage)
				}
				resp.PausedByOwner, resp.ConnectionLost = h.manager.AutopilotWaitingReasons(userIDStr)
				h.sendJSON(w, resp)
				return
			}
		}

		// D-13: the engage hook is never called from this handler directly —
		// only ResumeAutopilotByOwner's own changed-true path fires it, so
		// "one code path fires each real engage" holds for the owner-resume
		// case exactly as it does for a reconnect resume.
		newState, changed := h.manager.ResumeAutopilotByOwner(userIDStr)
		resp.State = string(newState)
		switch {
		case changed:
			resp.Outcome = "resumed"
			// D-15: ResumeAutopilotByOwner reports changed==false whenever
			// ConnectionLost is still standing, so gating the notice on
			// changed alone is automatically "only after every waiting
			// reason has cleared" — no reason condition is re-derived here.
			h.notifyAutopilotSystemLine(userIDStr, "resumed", "Autopilot resumed")
			h.pushAutopilotState(userIDStr, "resume")
		case newState == AutopilotWaiting:
			// Another reason (a lost connection) is still standing (D-15).
			// The state did not change but a REASON did -- the owner's pause
			// is gone -- so every screen is told that too.
			resp.Outcome = "still-waiting"
			h.pushAutopilotState(userIDStr, "resume")
		default:
			resp.Outcome = "not-waiting"
		}
		resp.PausedByOwner, resp.ConnectionLost = h.manager.AutopilotWaitingReasons(userIDStr)
		h.sendJSON(w, resp)
		return
	}
}

// sendJSON sends a JSON response
func (h *Handler) sendJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// sendError sends an error response
func (h *Handler) sendError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	resp := map[string]string{"error": message}
	json.NewEncoder(w).Encode(resp)
}

// contextKey is a type for context keys
type contextKey string

const userIDKey contextKey = "user_id"

// UserIDFromContext extracts user ID from context
func UserIDFromContext(ctx context.Context) (string, bool) {
	userID := ctx.Value(userIDKey)
	if userID == nil {
		return "", false
	}
	id, ok := userID.(string)
	return id, ok
}

// WithUserID adds user ID to context
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}
