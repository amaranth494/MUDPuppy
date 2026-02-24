package session

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/amaranth494/MudPuppy/internal/config"
	"github.com/google/uuid"
)

// Handler handles session HTTP requests
type Handler struct {
	manager   *Manager
	config    *config.Config
	callbacks *HandlerCallbacks
}

// HandlerCallbacks provides callbacks for session events
type HandlerCallbacks struct {
	OnConnected     func(connectionID, userID uuid.UUID) error
	GetAutoLogin    func(connectionID uuid.UUID) (username, password string, err error)
	SendCredentials func(userID, username, password string) error
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
	session, err := h.manager.Connect(r.Context(), userIDStr, req.Host, req.Port)
	if err != nil {
		log.Printf("[SP02PH01] Connect failed: user=%s, error=%v", userIDStr, err)
		h.sendError(w, err.Error())
		return
	}

	// If connection_id was provided, update last_connected_at and handle auto-login
	if req.ConnectionID != uuid.Nil && h.callbacks != nil {
		// Update last_connected_at
		if h.callbacks.OnConnected != nil {
			if err := h.callbacks.OnConnected(req.ConnectionID, userUUID); err != nil {
				log.Printf("[SP03PH05T03] Failed to update last_connected_at: %v", err)
				// Don't fail the connection, just log the error
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

	h.sendJSON(w, resp)
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
