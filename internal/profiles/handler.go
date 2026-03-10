package profiles

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/amaranth494/MudPuppy/internal/store"
	"github.com/google/uuid"
)

// Handler handles profiles HTTP requests
type Handler struct {
	profileStore *store.ProfileStore
}

// NewHandler creates a new profiles handler
func NewHandler(profileStore *store.ProfileStore) *Handler {
	return &Handler{
		profileStore: profileStore,
	}
}

// Request/Response types

type UpdateProfileRequest struct {
	Keybindings *map[string]string     `json:"keybindings,omitempty"`
	Settings    *store.ProfileSettings `json:"settings,omitempty"`
}

type ProfileResponse struct {
	ID           uuid.UUID             `json:"id"`
	UserID       uuid.UUID             `json:"user_id"`
	ConnectionID uuid.UUID             `json:"connection_id"`
	Keybindings  map[string]string     `json:"keybindings"`
	Settings     store.ProfileSettings `json:"settings"`
	CreatedAt    string                `json:"created_at"`
	UpdatedAt    string                `json:"updated_at"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// Automation response types
type AliasesResponse struct {
	Items []store.Alias `json:"items"`
}

type TriggersResponse struct {
	Items []store.Trigger `json:"items"`
}

type VariablesResponse struct {
	Items []store.Variable `json:"items"`
}

// Variable name validation regex
var variableNameRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// Get handles GET /api/v1/profiles/:id
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

	profileID, err := h.getProfileID(r)
	if err != nil {
		h.sendError(w, err.Error())
		return
	}

	profile, err := h.profileStore.GetProfile(userUUID, profileID)
	if err != nil {
		log.Printf("[SP04PH02T03] Get profile failed: %v", err)
		h.sendError(w, "Failed to get profile")
		return
	}
	if profile == nil {
		h.sendError(w, "Profile not found")
		return
	}

	h.sendJSON(w, toResponse(profile))
}

// GetByConnection handles GET /api/v1/connections/:connectionID/profile
func (h *Handler) GetByConnection(w http.ResponseWriter, r *http.Request) {
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

	connectionID, err := h.getConnectionID(r)
	if err != nil {
		h.sendError(w, err.Error())
		return
	}

	profile, err := h.profileStore.GetProfileByConnection(userUUID, connectionID)
	if err != nil {
		log.Printf("[SP04PH02T03] Get profile by connection failed: %v", err)
		h.sendError(w, "Failed to get profile")
		return
	}
	if profile == nil {
		h.sendError(w, "Profile not found")
		return
	}

	h.sendJSON(w, toResponse(profile))
}

// Update handles PUT /api/v1/profiles/:id
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

	profileID, err := h.getProfileID(r)
	if err != nil {
		h.sendError(w, err.Error())
		return
	}

	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid request body")
		return
	}

	// Validate the update
	if err := h.validateUpdate(&req); err != nil {
		h.sendError(w, err.Error())
		return
	}

	updates := &store.ProfileUpdate{
		Keybindings: req.Keybindings,
		Settings:    req.Settings,
	}

	profile, err := h.profileStore.UpdateProfile(userUUID, profileID, updates)
	if err != nil {
		log.Printf("[SP04PH02T03] Update profile failed: %v", err)
		h.sendError(w, "Failed to update profile")
		return
	}
	if profile == nil {
		h.sendError(w, "Profile not found")
		return
	}

	h.sendJSON(w, toResponse(profile))
}

// GetAliases handles GET /api/v1/profiles/:connection_id/aliases
func (h *Handler) GetAliases(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	_, profile, err := h.getProfileByConnectionID(r)
	if err != nil {
		h.sendError(w, err.Error())
		return
	}

	h.sendJSON(w, AliasesResponse{Items: profile.Aliases.Items})
}

// PutAliases handles PUT /api/v1/profiles/:connection_id/aliases
func (h *Handler) PutAliases(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userUUID, profile, err := h.getProfileByConnectionID(r)
	if err != nil {
		h.sendError(w, err.Error())
		return
	}

	var req AliasesResponse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid request body")
		return
	}

	// Validate aliases
	if len(req.Items) > 200 {
		h.sendError(w, "Maximum 200 aliases allowed")
		return
	}

	for _, alias := range req.Items {
		if strings.TrimSpace(alias.Pattern) == "" {
			h.sendError(w, "Alias pattern cannot be empty")
			return
		}
		if strings.TrimSpace(alias.Replacement) == "" {
			h.sendError(w, "Alias replacement cannot be empty")
			return
		}
	}

	// Update aliases
	updates := &store.ProfileUpdate{
		Aliases: &store.Aliases{Items: req.Items},
	}

	updatedProfile, err := h.profileStore.UpdateProfile(userUUID, profile.ID, updates)
	if err != nil {
		log.Printf("[SP05PH01T02] Update aliases failed: %v", err)
		h.sendError(w, "Failed to update aliases")
		return
	}

	h.sendJSON(w, AliasesResponse{Items: updatedProfile.Aliases.Items})
}

// GetTriggers handles GET /api/v1/profiles/:connection_id/triggers
func (h *Handler) GetTriggers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	_, profile, err := h.getProfileByConnectionID(r)
	if err != nil {
		h.sendError(w, err.Error())
		return
	}

	h.sendJSON(w, TriggersResponse{Items: profile.Triggers.Items})
}

// PutTriggers handles PUT /api/v1/profiles/:connection_id/triggers
func (h *Handler) PutTriggers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userUUID, profile, err := h.getProfileByConnectionID(r)
	if err != nil {
		h.sendError(w, err.Error())
		return
	}

	var req TriggersResponse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid request body")
		return
	}

	// Validate triggers
	if len(req.Items) > 200 {
		h.sendError(w, "Maximum 200 triggers allowed")
		return
	}

	for _, trigger := range req.Items {
		if strings.TrimSpace(trigger.Match) == "" {
			h.sendError(w, "Trigger match cannot be empty")
			return
		}
		if strings.TrimSpace(trigger.Action) == "" {
			h.sendError(w, "Trigger action cannot be empty")
			return
		}
		if trigger.Type != "contains" {
			h.sendError(w, "Trigger type must be 'contains'")
			return
		}
		if trigger.Cooldown < 0 {
			h.sendError(w, "Trigger cooldown must be non-negative")
			return
		}
	}

	// Update triggers
	updates := &store.ProfileUpdate{
		Triggers: &store.Triggers{Items: req.Items},
	}

	updatedProfile, err := h.profileStore.UpdateProfile(userUUID, profile.ID, updates)
	if err != nil {
		log.Printf("[SP05PH01T04] Update triggers failed: %v", err)
		h.sendError(w, "Failed to update triggers")
		return
	}

	h.sendJSON(w, TriggersResponse{Items: updatedProfile.Triggers.Items})
}

// GetEnvironment handles GET /api/v1/profiles/:connection_id/environment
func (h *Handler) GetEnvironment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	_, profile, err := h.getProfileByConnectionID(r)
	if err != nil {
		h.sendError(w, err.Error())
		return
	}

	h.sendJSON(w, VariablesResponse{Items: profile.Variables.Items})
}

// PutEnvironment handles PUT /api/v1/profiles/:connection_id/environment
func (h *Handler) PutEnvironment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userUUID, profile, err := h.getProfileByConnectionID(r)
	if err != nil {
		h.sendError(w, err.Error())
		return
	}

	var req VariablesResponse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid request body")
		return
	}

	// Validate variables
	if len(req.Items) > 100 {
		h.sendError(w, "Maximum 100 variables allowed")
		return
	}

	// Check for duplicate names and validate names
	seenNames := make(map[string]bool)
	for _, v := range req.Items {
		if strings.TrimSpace(v.Name) == "" {
			h.sendError(w, "Variable name cannot be empty")
			return
		}
		if !variableNameRegex.MatchString(v.Name) {
			h.sendError(w, "Variable name must start with a letter or underscore and contain only letters, numbers, and underscores")
			return
		}
		if seenNames[v.Name] {
			h.sendError(w, "Duplicate variable name: "+v.Name)
			return
		}
		seenNames[v.Name] = true
	}

	// Update variables
	updates := &store.ProfileUpdate{
		Variables: &store.Variables{Items: req.Items},
	}

	updatedProfile, err := h.profileStore.UpdateProfile(userUUID, profile.ID, updates)
	if err != nil {
		log.Printf("[SP05PH01T06] Update environment failed: %v", err)
		h.sendError(w, "Failed to update environment")
		return
	}

	h.sendJSON(w, VariablesResponse{Items: updatedProfile.Variables.Items})
}

// getProfileByConnectionID is a helper that validates the user and fetches the profile by connection ID
func (h *Handler) getProfileByConnectionID(r *http.Request) (uuid.UUID, *store.Profile, error) {
	userID := r.Context().Value("user_id")
	if userID == nil {
		return uuid.Nil, nil, &ValidationError{Message: "Unauthorized"}
	}
	userIDStr := userID.(string)
	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, nil, &ValidationError{Message: "Invalid user ID"}
	}

	connectionID, err := h.getConnectionIDFromPath(r)
	if err != nil {
		return uuid.Nil, nil, err
	}

	profile, err := h.profileStore.GetProfileByConnection(userUUID, connectionID)
	if err != nil {
		log.Printf("[SP05PH01T07] Get profile by connection failed: %v", err)
		return uuid.Nil, nil, &ValidationError{Message: "Failed to get profile"}
	}
	if profile == nil {
		return uuid.Nil, nil, &ValidationError{Message: "Profile not found"}
	}

	return userUUID, profile, nil
}

// getConnectionIDFromPath extracts connection ID from URL path for automation endpoints
// Format: /api/v1/profiles/:connection_id/aliases, /api/v1/profiles/:connection_id/triggers, etc.
func (h *Handler) getConnectionIDFromPath(r *http.Request) (uuid.UUID, error) {
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 5 {
		return uuid.Nil, &ValidationError{Message: "Connection ID not found"}
	}
	return uuid.Parse(parts[4])
}

// validateUpdate validates a profile update request
func (h *Handler) validateUpdate(req *UpdateProfileRequest) error {
	// Validate keybindings if provided
	if req.Keybindings != nil {
		bindings := *req.Keybindings

		// Max 50 keybindings
		if len(bindings) > 50 {
			return &ValidationError{Message: "Maximum 50 keybindings allowed"}
		}

		// Validate each binding
		for key, command := range bindings {
			// Key must not be empty
			if strings.TrimSpace(key) == "" {
				return &ValidationError{Message: "Keybinding key cannot be empty"}
			}
			// Command must not be empty
			if strings.TrimSpace(command) == "" {
				return &ValidationError{Message: "Keybinding command cannot be empty"}
			}
			// Max command length 500
			if len(command) > 500 {
				return &ValidationError{Message: "Command must be 500 characters or less"}
			}
		}
	}

	// Validate settings if provided
	if req.Settings != nil {
		settings := *req.Settings

		// Validate scrollback limit
		if settings.ScrollbackLimit < 100 || settings.ScrollbackLimit > 10000 {
			return &ValidationError{Message: "Scrollback limit must be between 100 and 10000"}
		}
	}

	return nil
}

// ValidationError represents a validation error
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// getProfileID extracts profile ID from URL
func (h *Handler) getProfileID(r *http.Request) (uuid.UUID, error) {
	// Extract ID from path - format is /api/v1/profiles/:id
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 5 {
		return uuid.Nil, &ValidationError{Message: "Profile ID not found"}
	}
	return uuid.Parse(parts[4])
}

// getConnectionID extracts connection ID from URL
func (h *Handler) getConnectionID(r *http.Request) (uuid.UUID, error) {
	// Extract ID from path - format is /api/v1/connections/:connectionID/profile
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 5 {
		return uuid.Nil, &ValidationError{Message: "Connection ID not found"}
	}
	return uuid.Parse(parts[4])
}

// sendJSON sends a JSON response
func (h *Handler) sendJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("[SP04PH02T03] Failed to encode JSON: %v", err)
	}
}

// sendError sends an error response
func (h *Handler) sendError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}

// toResponse converts a Profile to a ProfileResponse
func toResponse(profile *store.Profile) ProfileResponse {
	return ProfileResponse{
		ID:           profile.ID,
		UserID:       profile.UserID,
		ConnectionID: profile.ConnectionID,
		Keybindings:  profile.Keybindings,
		Settings:     profile.Settings,
		CreatedAt:    profile.CreatedAt,
		UpdatedAt:    profile.UpdatedAt,
	}
}
