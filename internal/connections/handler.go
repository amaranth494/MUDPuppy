package connections

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/amaranth494/MudPuppy/internal/crypto"
	"github.com/amaranth494/MudPuppy/internal/session"
	"github.com/amaranth494/MudPuppy/internal/store"
	"github.com/google/uuid"
)

// Handler handles connections HTTP requests
type Handler struct {
	connStore  *store.ConnectionStore
	credStore  *store.CredentialsStore
	crypto     *crypto.KeyStore
	sessionMgr *session.Manager
}

// NewHandler creates a new connections handler
func NewHandler(connStore *store.ConnectionStore, credStore *store.CredentialsStore, crypto *crypto.KeyStore, sessionMgr *session.Manager) *Handler {
	return &Handler{
		connStore:  connStore,
		credStore:  credStore,
		crypto:     crypto,
		sessionMgr: sessionMgr,
	}
}

// GetCredentialsForAutoLogin retrieves credentials for auto-login
// Returns username, password if auto_login is enabled; otherwise returns empty strings
func (h *Handler) GetCredentialsForAutoLogin(connectionID uuid.UUID) (username, password string, err error) {
	// Get credential status first
	status, err := h.credStore.GetStatus(connectionID)
	if err != nil {
		return "", "", err
	}

	// If auto-login is not enabled, return empty
	if status == nil || !status.AutoLoginEnabled {
		return "", "", nil
	}

	// Get the credentials
	cred, err := h.credStore.GetByConnectionID(connectionID)
	if err != nil {
		return "", "", err
	}
	if cred == nil {
		return "", "", nil
	}

	// Decrypt the password using the specific key version stored with the credential
	passwordBytes, err := h.crypto.DecryptWithVersion(cred.EncryptedPassword, cred.KeyVersion)
	if err != nil {
		log.Printf("[SP03PH05T08] Failed to decrypt password with key version %d: %v", cred.KeyVersion, err)
		return "", "", err
	}

	return cred.Username, string(passwordBytes), nil
}

// Request/Response types

type CreateConnectionRequest struct {
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
}

type UpdateConnectionRequest struct {
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
}

type ConnectionResponse struct {
	ID               uuid.UUID `json:"id"`
	UserID           uuid.UUID `json:"user_id"`
	Name             string    `json:"name"`
	Host             string    `json:"host"`
	Port             int       `json:"port"`
	Protocol         string    `json:"protocol"`
	CreatedAt        string    `json:"created_at"`
	UpdatedAt        string    `json:"updated_at"`
	LastConnectedAt  *string   `json:"last_connected_at,omitempty"`
	HasCredentials   bool      `json:"has_credentials"`
	AutoLoginEnabled bool      `json:"auto_login_enabled"`
}

type SetCredentialsRequest struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	AutoLogin bool   `json:"auto_login"`
}

type CredentialStatusResponse struct {
	Username         string `json:"username"`
	HasCredentials   bool   `json:"has_credentials"`
	AutoLoginEnabled bool   `json:"auto_login_enabled"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// Create handles POST /api/v1/connections
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
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

	var req CreateConnectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid request body")
		return
	}

	// Validate required fields
	if req.Name == "" || req.Host == "" {
		h.sendError(w, "Name and host are required")
		return
	}

	// Default port to 23 if not specified
	if req.Port == 0 {
		req.Port = 23
	}

	// Default protocol to telnet if not specified
	if req.Protocol == "" {
		req.Protocol = "telnet"
	}

	conn := &store.SavedConnection{
		UserID:   userUUID,
		Name:     req.Name,
		Host:     req.Host,
		Port:     req.Port,
		Protocol: req.Protocol,
	}

	if err := h.connStore.Create(conn); err != nil {
		log.Printf("[SP03PH05T02] Create connection failed: %v", err)
		h.sendError(w, "Failed to create connection")
		return
	}

	h.sendJSON(w, toResponse(conn, false, false))
}

// List handles GET /api/v1/connections
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
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
	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.sendError(w, "Invalid user ID")
		return
	}

	connections, err := h.connStore.GetByUserID(userUUID)
	if err != nil {
		log.Printf("[SP03PH05T02] List connections failed: %v", err)
		h.sendError(w, "Failed to list connections")
		return
	}

	// Get credential status for each connection
	resp := make([]ConnectionResponse, len(connections))
	for i, conn := range connections {
		status, _ := h.credStore.GetStatus(conn.ID)
		hasCreds := false
		autoLogin := false
		if status != nil {
			hasCreds = status.HasCredentials
			autoLogin = status.AutoLoginEnabled
		}
		resp[i] = toResponse(&conn, hasCreds, autoLogin)
	}

	h.sendJSON(w, resp)
}

// Get handles GET /api/v1/connections/:id
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
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
	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.sendError(w, "Invalid user ID")
		return
	}

	connID, err := h.getConnectionID(r)
	if err != nil {
		h.sendError(w, err.Error())
		return
	}

	conn, err := h.connStore.GetByID(connID, userUUID)
	if err != nil {
		log.Printf("[SP03PH05T02] Get connection failed: %v", err)
		h.sendError(w, "Failed to get connection")
		return
	}
	if conn == nil {
		h.sendError(w, "Connection not found")
		return
	}

	status, _ := h.credStore.GetStatus(conn.ID)
	hasCreds := false
	autoLogin := false
	if status != nil {
		hasCreds = status.HasCredentials
		autoLogin = status.AutoLoginEnabled
	}

	h.sendJSON(w, toResponse(conn, hasCreds, autoLogin))
}

// Update handles PUT /api/v1/connections/:id
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
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

	connID, err := h.getConnectionID(r)
	if err != nil {
		h.sendError(w, err.Error())
		return
	}

	var req UpdateConnectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid request body")
		return
	}

	// Validate required fields
	if req.Name == "" || req.Host == "" {
		h.sendError(w, "Name and host are required")
		return
	}

	// Default port to 23 if not specified
	if req.Port == 0 {
		req.Port = 23
	}

	// Default protocol to telnet if not specified
	if req.Protocol == "" {
		req.Protocol = "telnet"
	}

	conn := &store.SavedConnection{
		ID:       connID,
		UserID:   userUUID,
		Name:     req.Name,
		Host:     req.Host,
		Port:     req.Port,
		Protocol: req.Protocol,
	}

	if err := h.connStore.Update(conn); err != nil {
		log.Printf("[SP03PH05T02] Update connection failed: %v", err)
		h.sendError(w, "Failed to update connection")
		return
	}

	status, _ := h.credStore.GetStatus(conn.ID)
	hasCreds := false
	autoLogin := false
	if status != nil {
		hasCreds = status.HasCredentials
		autoLogin = status.AutoLoginEnabled
	}

	h.sendJSON(w, toResponse(conn, hasCreds, autoLogin))
}

// Delete handles DELETE /api/v1/connections/:id
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
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

	connID, err := h.getConnectionID(r)
	if err != nil {
		h.sendError(w, err.Error())
		return
	}

	// Delete credentials first (cascade should handle this, but be explicit)
	if err := h.credStore.Delete(connID); err != nil {
		log.Printf("[SP03PH05T02] Delete credentials failed: %v", err)
	}

	if err := h.connStore.Delete(connID, userUUID); err != nil {
		log.Printf("[SP03PH05T02] Delete connection failed: %v", err)
		h.sendError(w, "Failed to delete connection")
		return
	}

	h.sendJSON(w, map[string]string{"status": "deleted"})
}

// GetRecent handles GET /api/v1/connections/recent
func (h *Handler) GetRecent(w http.ResponseWriter, r *http.Request) {
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
	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.sendError(w, "Invalid user ID")
		return
	}

	connections, err := h.connStore.GetRecent(userUUID)
	if err != nil {
		log.Printf("[SP03PH05T04] Get recent connections failed: %v", err)
		h.sendError(w, "Failed to get recent connections")
		return
	}

	// Get credential status for each connection
	resp := make([]ConnectionResponse, len(connections))
	for i, conn := range connections {
		status, _ := h.credStore.GetStatus(conn.ID)
		hasCreds := false
		autoLogin := false
		if status != nil {
			hasCreds = status.HasCredentials
			autoLogin = status.AutoLoginEnabled
		}
		resp[i] = toResponse(&conn, hasCreds, autoLogin)
	}

	h.sendJSON(w, resp)
}

// SetCredentials handles POST /api/v1/connections/:id/credentials
func (h *Handler) SetCredentials(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
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

	connID, err := h.getConnectionID(r)
	if err != nil {
		h.sendError(w, err.Error())
		return
	}

	// Verify connection belongs to user
	conn, err := h.connStore.GetByID(connID, userUUID)
	if err != nil {
		h.sendError(w, "Failed to verify connection")
		return
	}
	if conn == nil {
		h.sendError(w, "Connection not found")
		return
	}

	var req SetCredentialsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid request body")
		return
	}

	// Encrypt password (only if not empty - empty password means "keep existing")
	var encryptedBytes []byte
	var version int
	if req.Password != "" {
		encryptedPassword, ver, err := h.crypto.EncryptString(req.Password)
		if err != nil {
			log.Printf("[SP03PH05T07] Encrypt password failed: %v", err)
			h.sendError(w, "Failed to encrypt password")
			return
		}

		encBytes, err := base64.StdEncoding.DecodeString(encryptedPassword)
		if err != nil {
			h.sendError(w, "Failed to encode password")
			return
		}
		encryptedBytes = encBytes
		version = ver
	}

	cred := &store.ConnectionCredential{
		ConnectionID:      connID,
		Username:          req.Username,
		EncryptedPassword: encryptedBytes,
		KeyVersion:        version,
		AutoLogin:         req.AutoLogin,
	}

	if err := h.credStore.Upsert(cred); err != nil {
		log.Printf("[SP03PH05T07] Set credentials failed: %v", err)
		h.sendError(w, "Failed to store credentials")
		return
	}

	// Return status only, never the actual credentials
	h.sendJSON(w, CredentialStatusResponse{
		HasCredentials:   true,
		AutoLoginEnabled: req.AutoLogin,
	})
}

// GetCredentialsStatus handles GET /api/v1/connections/:id/credentials/status
func (h *Handler) GetCredentialsStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Printf("[DEBUG] Credential status requested - connection=%s", r.PathValue("id"))

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

	connID, err := h.getConnectionID(r)
	if err != nil {
		h.sendError(w, err.Error())
		return
	}

	// Verify connection belongs to user
	conn, err := h.connStore.GetByID(connID, userUUID)
	if err != nil {
		h.sendError(w, "Failed to verify connection")
		return
	}
	if conn == nil {
		h.sendError(w, "Connection not found")
		return
	}

	status, err := h.credStore.GetStatus(connID)
	if err != nil {
		log.Printf("[SP03PH05T07] Get credentials status failed: %v", err)
		h.sendError(w, "Failed to get credentials status")
		return
	}

	// Also get the actual credentials to retrieve username (not password)
	var username string
	cred, err := h.credStore.GetByConnectionID(connID)
	log.Printf("[DEBUG] Credential row found - connection=%s username_present=%t", connID, cred != nil && cred.Username != "")
	if err == nil && cred != nil {
		username = cred.Username
	}

	h.sendJSON(w, CredentialStatusResponse{
		Username:         username,
		HasCredentials:   status.HasCredentials,
		AutoLoginEnabled: status.AutoLoginEnabled,
	})
}

// DeleteCredentials handles DELETE /api/v1/connections/:id/credentials
func (h *Handler) DeleteCredentials(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
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

	connID, err := h.getConnectionID(r)
	if err != nil {
		h.sendError(w, err.Error())
		return
	}

	// Verify connection belongs to user
	conn, err := h.connStore.GetByID(connID, userUUID)
	if err != nil {
		h.sendError(w, "Failed to verify connection")
		return
	}
	if conn == nil {
		h.sendError(w, "Connection not found")
		return
	}

	if err := h.credStore.Delete(connID); err != nil {
		log.Printf("[SP03PH05T07] Delete credentials failed: %v", err)
		h.sendError(w, "Failed to delete credentials")
		return
	}

	h.sendJSON(w, CredentialStatusResponse{
		HasCredentials:   false,
		AutoLoginEnabled: false,
	})
}

// Connect handles POST /api/v1/connections/:id/connect
// This connects to a saved connection using the session manager
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

	// Extract connection ID manually from URL path to handle /connect suffix
	// URL format: /api/v1/connections/{uuid}/connect
	path := r.URL.Path
	prefix := "/api/v1/connections/"
	if len(path) <= len(prefix) {
		h.sendError(w, "Invalid connection ID")
		return
	}
	rest := path[len(prefix):]
	rest = strings.TrimSuffix(rest, "/connect")

	connID, err := uuid.Parse(rest)
	if err != nil {
		h.sendError(w, "Invalid connection ID")
		return
	}

	// Get connection details
	conn, err := h.connStore.GetByID(connID, userUUID)
	if err != nil {
		log.Printf("[SP03PH06T05] Get connection failed: %v", err)
		h.sendError(w, "Failed to get connection")
		return
	}
	if conn == nil {
		h.sendError(w, "Connection not found")
		return
	}

	// Check if session manager is available
	if h.sessionMgr == nil {
		h.sendError(w, "Session manager not available")
		return
	}

	// Check if user already has an active session
	existingSession, _ := h.sessionMgr.GetSession(userUUID.String())
	if existingSession != nil && existingSession.State == session.StateConnected {
		h.sendError(w, "user already has active session")
		return
	}

	// Get credentials for auto-login (if available)
	var username, password string
	autoLogin := false

	status, err := h.credStore.GetStatus(connID)
	if err == nil && status != nil && status.AutoLoginEnabled {
		username, password, _ = h.GetCredentialsForAutoLogin(connID)
		autoLogin = true
	}

	// Connect to the MUD server using session manager
	ctx := context.Background()
	_, err = h.sessionMgr.Connect(ctx, userUUID.String(), conn.Host, conn.Port)
	if err != nil {
		log.Printf("[SP03PH06T05] Connect failed: %v", err)
		h.sendError(w, err.Error())
		return
	}

	// If auto-login is enabled, send credentials
	if autoLogin && username != "" && password != "" {
		err = h.sessionMgr.SendCredentials(userUUID.String(), username, password)
		if err != nil {
			log.Printf("[SP03PH06T05] Send credentials failed: %v", err)
			// Don't fail the connection for this
		}
	}

	// Update last_connected_at
	if err := h.connStore.UpdateLastConnectedAt(connID, userUUID); err != nil {
		log.Printf("[SP03PH06T05] Update last_connected_at failed: %v", err)
		// Don't fail the connection for this
	}

	// Return success response
	h.sendJSON(w, map[string]interface{}{
		"state":      "connected",
		"host":       conn.Host,
		"port":       conn.Port,
		"auto_login": autoLogin,
	})
}

// GetCredentials retrieves credentials for a connection (internal use, not exposed to API)
func (h *Handler) GetCredentials(connectionID uuid.UUID) (*store.ConnectionCredential, error) {
	return h.credStore.GetByConnectionID(connectionID)
}

// UpdateLastConnectedAt updates the last_connected_at timestamp for a connection
func (h *Handler) UpdateLastConnectedAt(connectionID, userID uuid.UUID) error {
	return h.connStore.UpdateLastConnectedAt(connectionID, userID)
}

// getConnectionID extracts the connection ID from the request using PathValue
func (h *Handler) getConnectionID(r *http.Request) (uuid.UUID, error) {
	idStr := r.PathValue("id")
	if idStr == "" {
		return uuid.Nil, fmt.Errorf("missing connection ID")
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid UUID length: %d", len(idStr))
	}
	return id, nil
}

func toResponse(conn *store.SavedConnection, hasCreds, autoLogin bool) ConnectionResponse {
	resp := ConnectionResponse{
		ID:               conn.ID,
		UserID:           conn.UserID,
		Name:             conn.Name,
		Host:             conn.Host,
		Port:             conn.Port,
		Protocol:         conn.Protocol,
		CreatedAt:        conn.CreatedAt,
		UpdatedAt:        conn.UpdatedAt,
		LastConnectedAt:  conn.LastConnectedAt,
		HasCredentials:   hasCreds,
		AutoLoginEnabled: autoLogin,
	}
	return resp
}

func (h *Handler) sendJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) sendError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	resp := ErrorResponse{Error: message}
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
