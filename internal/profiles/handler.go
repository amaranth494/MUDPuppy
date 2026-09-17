package profiles

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/amaranth494/MudPuppy/internal/policy"
	"github.com/amaranth494/MudPuppy/internal/store"
	"github.com/google/uuid"
)

// profileStorage is the subset of *store.ProfileStore this package calls.
// Declaring it as an interface lets the handler be exercised in tests with
// a hand-written fake, without a live Postgres connection.
type profileStorage interface {
	GetProfile(userID, profileID uuid.UUID) (*store.Profile, error)
	GetProfileByConnection(userID, connectionID uuid.UUID) (*store.Profile, error)
	UpdateProfile(userID, profileID uuid.UUID, updates *store.ProfileUpdate) (*store.Profile, error)
	// UpdateSessionGoal is the goal box's own targeted save (code review
	// WR-08 of Phase 4); PutGoal never goes through UpdateProfile.
	UpdateSessionGoal(userID, profileID uuid.UUID, goal string) (*store.Profile, error)
	AcceptPolicy(userID, profileID uuid.UUID, version string) (*store.Profile, error)
}

// transcriptStorage is the subset of *store.TranscriptStore the log
// endpoints call (task 03-05-01). Declaring it as an interface, same
// discipline as profileStorage above, lets logs_test.go exercise the
// handler with a hand-written fake instead of a live Postgres connection.
type transcriptStorage interface {
	ListSessionsForConnection(connectionID uuid.UUID, limit int) ([]store.GameSessionSummary, error)
	GetSessionLines(gameSessionID, connectionID uuid.UUID, limit int) ([]store.GameSessionLine, error)
	// SessionMemoryForConnection is the read side of the ai-memory
	// sub-resource (D-10, plan 04-08) — same discipline as the two methods
	// above.
	SessionMemoryForConnection(connectionID uuid.UUID) ([]string, error)
}

// decisionsStorage is the subset of *store.DecisionStore the decisions
// endpoint calls (task 03-09-03). Same discipline as profileStorage and
// transcriptStorage above: an interface lets decisions_test.go exercise
// the handler with a hand-written fake instead of a live Postgres
// connection.
type decisionsStorage interface {
	ListForConnection(connectionID uuid.UUID, limit int) ([]store.Decision, error)
}

// questStorage is the subset of *store.QuestStore the goal endpoint calls
// (task 04-06-02, D-04). Same discipline as decisionsStorage above: an
// interface lets handler_test.go exercise PutGoal with a hand-written fake
// instead of a live Postgres connection.
type questStorage interface {
	EnsureActiveQuest(userID, connectionID uuid.UUID, goalText string) (store.Quest, error)
}

// retentionStorage is the subset of *store.RetentionJob the owner's
// immediate "delete captured text now" action calls (task 04-09-02, D-21).
// Same discipline as questStorage above: an interface lets handler tests
// exercise DeleteCapturedText with a hand-written fake instead of a live
// Postgres connection.
type retentionStorage interface {
	DeleteCapturedTextForConnection(connectionID uuid.UUID) (store.RetentionCounts, error)
}

// AIEvent is the payload PutGoal pushes through the goal-changed/
// goal-cleared notifier hook (D-03). It carries exactly what a "system" AI
// message needs to render the locked line -- no State/Calls/Failures/...
// fields, which are driver-loop-owned (D-14/D-15/D-17) and meaningless to a
// goal save that may happen while autopilot is off.
type AIEvent struct {
	ID        string
	Kind      string
	Outcome   string
	Message   string
	Timestamp string
}

// AINotifier delivers an AIEvent to the browser, mirroring
// internal/driver.NotifierFunc's shape so cmd/server/main.go can route both
// the driver's decision events and this handler's goal-changed event
// through the exact same wsHandler.PushAI call (D-18's single
// message-delivery path, not two).
type AINotifier func(userID string, ev AIEvent)

// Handler handles profiles HTTP requests
type Handler struct {
	profileStore profileStorage
	// transcripts is nil when no transcript store was wired (NewHandler),
	// making ListSessions/GetSessionTranscript answer 503 rather than
	// panic — a missing dependency fails closed.
	transcripts transcriptStorage
	// decisions is nil until SetDecisionStore is called, making
	// GetDecisions answer 503 rather than panic — a missing dependency
	// fails closed, same discipline as transcripts above.
	decisions decisionsStorage
	// quests is nil until SetQuestStore is called. PutGoal still saves the
	// goal text with a nil quests store (the goal box itself never depends
	// on Quest bookkeeping succeeding) but skips EnsureActiveQuest.
	quests questStorage
	// retention is nil until SetRetention is called, making
	// DeleteCapturedText answer 503 rather than panic -- a missing
	// dependency fails closed, same discipline as transcripts above.
	retention retentionStorage
	// aiNotifier is nil until SetAINotifier is called, making the
	// goal-changed/goal-cleared system line a silent no-op — a missing
	// notifier never blocks the goal save itself, same discipline as
	// internal/driver.Notifier being nil-safe.
	aiNotifier AINotifier
}

// SetDecisionStore wires the connection's-decisions read endpoint
// (task 03-09-03) to s. Assigning a nil *store.DecisionStore straight into
// the decisionsStorage interface field would produce a non-nil interface
// with a nil underlying pointer (the same well-known Go trap
// NewHandlerWithTranscripts guards against below), so a nil s is a no-op
// and h.decisions stays a true nil interface, keeping the endpoint's
// fail-closed 503 behavior.
func (h *Handler) SetDecisionStore(s *store.DecisionStore) {
	if s != nil {
		h.decisions = s
	}
}

// SetQuestStore wires the goal endpoint's Quest reactivate-or-create call
// (D-04) to s. Same nil-underlying-pointer guard as SetDecisionStore above.
func (h *Handler) SetQuestStore(s *store.QuestStore) {
	if s != nil {
		h.quests = s
	}
}

// SetRetention wires the owner's immediate "delete captured text now"
// action (D-21) to j. Same nil-underlying-pointer guard as SetDecisionStore
// above.
func (h *Handler) SetRetention(j *store.RetentionJob) {
	if j != nil {
		h.retention = j
	}
}

// SetAINotifier wires the goal-changed/goal-cleared system-line push (D-03)
// after construction, following SetDecisionStore's injection precedent. A
// nil hook is a silent no-op, so PutGoal stays independently testable
// without a live websocket handler.
func (h *Handler) SetAINotifier(n AINotifier) {
	h.aiNotifier = n
}

// NewHandler creates a new profiles handler with no transcript store
// wired; the two log endpoints answer 503 until NewHandlerWithTranscripts
// is used instead.
func NewHandler(profileStore *store.ProfileStore) *Handler {
	return NewHandlerWithTranscripts(profileStore, nil)
}

// NewHandlerWithTranscripts creates a new profiles handler with the
// session-log endpoints wired to transcripts. A nil transcripts store
// disables those two endpoints (503) without panicking — a missing
// dependency fails closed, matching the discipline the package already
// uses for a nil EngageGate callback (internal/session/handler.go).
func NewHandlerWithTranscripts(profileStore *store.ProfileStore, transcripts *store.TranscriptStore) *Handler {
	h := &Handler{profileStore: profileStore}
	// Assigning a nil *store.TranscriptStore straight into the
	// transcriptStorage interface field would produce a non-nil interface
	// with a nil underlying pointer (a well-known Go trap) — checked
	// explicitly here so h.transcripts == nil stays a true nil interface.
	if transcripts != nil {
		h.transcripts = transcripts
	}
	return h
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

type TimersResponse struct {
	Items []store.Timer `json:"items"`
}

// AISettingsResponse is the GET response and PUT request body for the
// ai-settings sub-resource. It carries exactly conduct rules, approach
// guidance, the Never-issue list, and AI settings — no acceptance field and
// no reconnect field (D-13, T-1-01); acceptance is writable only through
// AcceptPolicy. NeverIssueList is an owner-written list of commands the AI
// must never issue, one entry per line, enforced mechanically in Go before
// dispatch (D-02, 03.1-CONTEXT.md); it is empty by default on every profile.
type AISettingsResponse struct {
	ConductRules     string           `json:"conduct_rules"`
	ApproachGuidance string           `json:"approach_guidance"`
	NeverIssueList   string           `json:"never_issue_list"`
	AISettings       store.AISettings `json:"ai_settings"`
}

// GoalResponse is the GET response and PUT request body for the ai-goal
// sub-resource (D-01). It carries exactly the session goal text — Quest
// bullets are not shown on the panel this phase (D-11) and have no field
// here.
//
// Quest is set only on a PUT response, to one of "created", "reactivated"
// or "none" — the same create-vs-reactivate word the "[AI-PLAYER] goal"
// log line already carries (04-06-SUMMARY.md), now also on the wire so a
// caller (the 04-10 harness) can prove D-04's reactivate-rather-than-
// duplicate behavior over HTTP instead of by a database query (the
// project's evidence rule forbids the latter). GetGoal never sets it
// (omitted via omitempty), since a plain read has no create-vs-reactivate
// event to report.
type GoalResponse struct {
	Goal  string `json:"goal"`
	Quest string `json:"quest,omitempty"`
}

// SessionMemoryResponse is the GET response for the ai-memory sub-resource
// (D-10, plan 04-08): the current game session's full curated Session
// Memory list. There is deliberately no PUT request body type to match —
// editing Session Memory by hand is Phase 5.
type SessionMemoryResponse struct {
	SessionMemory []string `json:"session_memory"`
}

// DeleteCapturedTextResponse is the response for the owner's immediate
// "delete captured text now" action (D-21): the two counts the harness and
// the panel's result line rely on. Never row text (T-4-05).
type DeleteCapturedTextResponse struct {
	SnapshotsCleared       int64 `json:"snapshots_cleared"`
	TranscriptLinesDeleted int64 `json:"transcript_lines_deleted"`
}

// maxGoalLength is the session goal's length cap (T-4-10), following the
// keybinding-command precedent (500 chars) rather than the 20000-char
// free-text fields, doubled per 04-CONTEXT.md's Claude's Discretion.
const maxGoalLength = 1000

// PolicyResponse is the response shape for both GetPolicy and AcceptPolicy.
type PolicyResponse struct {
	Text            string  `json:"text"`
	Version         string  `json:"version"`
	Accepted        bool    `json:"accepted"`
	AcceptedAt      *string `json:"accepted_at"`
	AcceptedVersion *string `json:"accepted_version"`
}

// EngageGateResponse is the response shape for GetEngageGate.
type EngageGateResponse struct {
	Allowed bool   `json:"allowed"`
	Message string `json:"message,omitempty"`
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

	log.Printf("[DEBUG GetEnvironment] Returning variables: %+v", profile.Variables.Items)

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
	log.Printf("[DEBUG PutEnvironment] Received variables: %+v", req.Items)
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

// GetTimers handles GET /api/v1/profiles/:connection_id/timers
func (h *Handler) GetTimers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	_, profile, err := h.getProfileByConnectionID(r)
	if err != nil {
		h.sendError(w, err.Error())
		return
	}

	h.sendJSON(w, TimersResponse{Items: profile.Timers.Items})
}

// PutTimers handles PUT /api/v1/profiles/:connection_id/timers
func (h *Handler) PutTimers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userUUID, profile, err := h.getProfileByConnectionID(r)
	if err != nil {
		h.sendError(w, err.Error())
		return
	}

	var req TimersResponse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid request body")
		return
	}

	// Validate timers
	if len(req.Items) > 50 {
		h.sendError(w, "Maximum 50 timers allowed")
		return
	}

	for _, timer := range req.Items {
		if strings.TrimSpace(timer.Name) == "" {
			h.sendError(w, "Timer name cannot be empty")
			return
		}
		if timer.Duration <= 0 {
			h.sendError(w, "Timer duration must be greater than 0")
			return
		}
	}

	// Update timers
	updates := &store.ProfileUpdate{
		Timers: &store.Timers{Items: req.Items},
	}

	updatedProfile, err := h.profileStore.UpdateProfile(userUUID, profile.ID, updates)
	if err != nil {
		log.Printf("[PR01PH06] Update timers failed: %v", err)
		h.sendError(w, "Failed to update timers")
		return
	}

	h.sendJSON(w, TimersResponse{Items: updatedProfile.Timers.Items})
}

// GetAISettings handles GET /api/v1/profiles/:connection_id/ai-settings
func (h *Handler) GetAISettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	_, profile, err := h.getProfileByConnectionID(r)
	if err != nil {
		h.sendError(w, err.Error())
		return
	}

	h.sendJSON(w, AISettingsResponse{
		ConductRules:     profile.ConductRules,
		ApproachGuidance: profile.ApproachGuidance,
		NeverIssueList:   profile.NeverIssueList,
		AISettings:       profile.AISettings,
	})
}

// PutAISettings handles PUT /api/v1/profiles/:connection_id/ai-settings
func (h *Handler) PutAISettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userUUID, profile, err := h.getProfileByConnectionID(r)
	if err != nil {
		h.sendError(w, err.Error())
		return
	}

	var req AISettingsResponse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid request body")
		return
	}

	if verr := validateAISettings(req); verr != nil {
		h.sendError(w, verr.Error())
		return
	}

	updates := &store.ProfileUpdate{
		ConductRules:     &req.ConductRules,
		ApproachGuidance: &req.ApproachGuidance,
		NeverIssueList:   &req.NeverIssueList,
		AISettings:       &req.AISettings,
	}

	updatedProfile, err := h.profileStore.UpdateProfile(userUUID, profile.ID, updates)
	if err != nil {
		log.Printf("[PH0103] Update AI settings failed: %v", err)
		h.sendError(w, "Failed to update AI settings")
		return
	}

	connectionID, _ := h.getConnectionIDFromPath(r)
	saved := updatedProfile
	log.Printf("[AI-PLAYER] ai settings saved connection_id=%s model_name_blank=%t call_cap_blank=%t threshold_blank=%t",
		connectionID, saved.AISettings.ModelName == "", saved.AISettings.CallCap == nil, saved.AISettings.DisengageThreshold == nil)

	h.sendJSON(w, AISettingsResponse{
		ConductRules:     updatedProfile.ConductRules,
		ApproachGuidance: updatedProfile.ApproachGuidance,
		NeverIssueList:   updatedProfile.NeverIssueList,
		AISettings:       updatedProfile.AISettings,
	})
}

// GetGoal handles GET /api/v1/profiles/:connection_id/ai-goal
func (h *Handler) GetGoal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	_, profile, err := h.getProfileByConnectionID(r)
	if err != nil {
		h.sendError(w, err.Error())
		return
	}

	h.sendJSON(w, GoalResponse{Goal: profile.SessionGoal})
}

// PutGoal handles PUT /api/v1/profiles/:connection_id/ai-goal. Saving a
// non-blank goal names its Quest (D-04): EnsureActiveQuest creates or
// reactivates the open Quest whose goal text matches, trimmed and
// case-folded. A blank goal clears the field and creates nothing — the
// previously active Quest, if any, stays open and untouched (Phase 4 closes
// nothing). Ownership is resolved through getProfileByConnectionID, the
// same check every other profile sub-resource uses (T-4-09).
func (h *Handler) PutGoal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userUUID, profile, err := h.getProfileByConnectionID(r)
	if err != nil {
		h.sendError(w, err.Error())
		return
	}

	var req GoalResponse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid request body")
		return
	}

	if len(req.Goal) > maxGoalLength {
		h.sendError(w, "Session goal must be 1000 characters or less")
		return
	}

	// A targeted save of the goal column alone (code review WR-08 of Phase
	// 4), never UpdateProfile's whole-row read-modify-write: the goal is
	// saved on every blur, during play, and must not be able to write another
	// request's stale columns back, nor be overwritten by one.
	updatedProfile, err := h.profileStore.UpdateSessionGoal(userUUID, profile.ID, req.Goal)
	if err != nil {
		log.Printf("[PH0106] Update session goal failed: %v", err)
		h.sendError(w, "Failed to update session goal")
		return
	}
	if updatedProfile == nil {
		h.sendError(w, "Profile not found")
		return
	}

	connectionID, _ := h.getConnectionIDFromPath(r)
	trimmedGoal := strings.TrimSpace(req.Goal)

	questWord := "none"
	if trimmedGoal != "" && h.quests != nil {
		if q, qerr := h.quests.EnsureActiveQuest(userUUID, connectionID, req.Goal); qerr != nil {
			log.Printf("[PH0106] Ensure active quest failed: %v", qerr)
		} else if q.CreatedAt.Equal(q.UpdatedAt) {
			questWord = "created"
		} else {
			questWord = "reactivated"
		}
	}

	log.Printf("[AI-PLAYER] goal connection_id=%s user_id=%s quest=%s goal_length=%d", connectionID, userUUID, questWord, len(req.Goal))

	if h.aiNotifier != nil {
		message := "Goal cleared"
		if trimmedGoal != "" {
			message = "Goal changed: " + updatedProfile.SessionGoal
		}
		h.aiNotifier(userUUID.String(), AIEvent{
			ID:        uuid.New().String(),
			Kind:      "system",
			Outcome:   "goal",
			Message:   message,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
	}

	h.sendJSON(w, GoalResponse{Goal: updatedProfile.SessionGoal, Quest: questWord})
}

// GetSessionMemory handles GET /api/v1/profiles/:connection_id/ai-memory
// (D-10). GET-only, following GetAISettings' exact shape: ownership is
// resolved through getProfileByConnectionID, the same check every other
// profile sub-resource uses (T-4-09). A nil transcripts store (no
// TranscriptStore wired) fails closed with 503 rather than panicking, the
// same discipline the two log endpoints already use. There is deliberately
// no PutSessionMemory — editing memory by hand is Phase 5 (D-10).
func (h *Handler) GetSessionMemory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	_, _, err := h.getProfileByConnectionID(r)
	if err != nil {
		h.sendError(w, err.Error())
		return
	}

	if h.transcripts == nil {
		http.Error(w, "Session Memory unavailable", http.StatusServiceUnavailable)
		return
	}

	connectionID, err := h.getConnectionIDFromPath(r)
	if err != nil {
		h.sendError(w, err.Error())
		return
	}

	bullets, err := h.transcripts.SessionMemoryForConnection(connectionID)
	if err != nil {
		log.Printf("[PH0107] Get session memory failed: %v", err)
		h.sendError(w, "Failed to load session memory")
		return
	}
	if bullets == nil {
		bullets = []string{}
	}

	h.sendJSON(w, SessionMemoryResponse{SessionMemory: bullets})
}

// DeleteCapturedText handles DELETE
// /api/v1/profiles/:connection_id/captured-text (D-21). Ownership is
// resolved through getProfileByConnectionID before anything is deleted
// (T-4-09) -- the same check every other profile sub-resource uses. A nil
// retention store fails closed with 503 rather than panicking, the same
// discipline GetSessionMemory already uses for a nil transcripts store. A
// failure returns the existing error shape; it never partially reports
// success.
func (h *Handler) DeleteCapturedText(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userUUID, _, err := h.getProfileByConnectionID(r)
	if err != nil {
		h.sendError(w, err.Error())
		return
	}

	if h.retention == nil {
		http.Error(w, "Delete captured text unavailable", http.StatusServiceUnavailable)
		return
	}

	connectionID, err := h.getConnectionIDFromPath(r)
	if err != nil {
		h.sendError(w, err.Error())
		return
	}

	counts, err := h.retention.DeleteCapturedTextForConnection(connectionID)
	if err != nil {
		log.Printf("[PH0109] Delete captured text failed: %v", err)
		h.sendError(w, "Failed to delete captured text")
		return
	}

	log.Printf("[AI-PLAYER] stage=retention-manual user_id=%s connection_id=%s snapshots_cleared=%d transcript_lines_deleted=%d",
		userUUID, connectionID, counts.SnapshotsCleared, counts.TranscriptLinesDeleted)

	h.sendJSON(w, DeleteCapturedTextResponse{
		SnapshotsCleared:       counts.SnapshotsCleared,
		TranscriptLinesDeleted: counts.TranscriptLinesDeleted,
	})
}

// GetPolicy handles GET /api/v1/profiles/:connection_id/policy
func (h *Handler) GetPolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	_, profile, err := h.getProfileByConnectionID(r)
	if err != nil {
		h.sendError(w, err.Error())
		return
	}

	h.sendJSON(w, PolicyResponse{
		Text:            policy.Text(),
		Version:         policy.Version(),
		Accepted:        profile.PolicyAcceptedAt != nil,
		AcceptedAt:      profile.PolicyAcceptedAt,
		AcceptedVersion: profile.PolicyVersionAccepted,
	})
}

// AcceptPolicy handles POST /api/v1/profiles/:connection_id/policy/accept.
// The version is taken only from policy.Version() (never the request body,
// which is not decoded at all), so there is nothing for a client to forge
// (D-04, T-1-02). A repeat POST is a harmless no-op that echoes the original
// timestamp and version (D-06).
func (h *Handler) AcceptPolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userUUID, profile, err := h.getProfileByConnectionID(r)
	if err != nil {
		h.sendError(w, err.Error())
		return
	}

	wasAccepted := profile.PolicyAcceptedAt != nil

	updatedProfile, err := h.profileStore.AcceptPolicy(userUUID, profile.ID, policy.Version())
	if err != nil {
		log.Printf("[PH0103] Accept policy failed: %v", err)
		h.sendError(w, "Failed to record acceptance")
		return
	}
	if updatedProfile == nil {
		h.sendError(w, "Profile not found")
		return
	}

	connectionID, _ := h.getConnectionIDFromPath(r)
	userIDVal := r.Context().Value("user_id")
	userIDStr, _ := userIDVal.(string)

	var version, acceptedAt string
	if updatedProfile.PolicyVersionAccepted != nil {
		version = *updatedProfile.PolicyVersionAccepted
	}
	if updatedProfile.PolicyAcceptedAt != nil {
		acceptedAt = *updatedProfile.PolicyAcceptedAt
	}

	if wasAccepted {
		log.Printf("[AI-PLAYER] policy already accepted connection_id=%s version=%s accepted_at=%s", connectionID, version, acceptedAt)
	} else {
		log.Printf("[AI-PLAYER] policy accepted connection_id=%s user_id=%s version=%s accepted_at=%s", connectionID, userIDStr, version, acceptedAt)
	}

	h.sendJSON(w, PolicyResponse{
		Text:            policy.Text(),
		Version:         policy.Version(),
		Accepted:        updatedProfile.PolicyAcceptedAt != nil,
		AcceptedAt:      updatedProfile.PolicyAcceptedAt,
		AcceptedVersion: updatedProfile.PolicyVersionAccepted,
	})
}

// GetEngageGate handles GET /api/v1/profiles/:connection_id/engage-gate.
// It reads acceptance only from the server-fetched row and accepts no
// client input beyond the path (T-1-05).
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

// validateAISettings validates a PutAISettings request body. Nil pointers
// (CallCap, DisengageThreshold) are valid and mean blank — a blank field is
// never rejected (D-09, D-10, D-11); only over-length text and out-of-range
// numeric values (when set) are rejected (T-1-04).
func validateAISettings(req AISettingsResponse) *ValidationError {
	if len(req.ConductRules) > 20000 {
		return &ValidationError{Message: "Conduct rules must be 20000 characters or less"}
	}
	if len(req.ApproachGuidance) > 20000 {
		return &ValidationError{Message: "Approach guidance must be 20000 characters or less"}
	}
	if len(req.NeverIssueList) > 20000 {
		return &ValidationError{Message: "Never-issue list must be 20000 characters or less"}
	}
	if len(req.AISettings.ModelName) > 200 {
		return &ValidationError{Message: "Model name must be 200 characters or less"}
	}
	if req.AISettings.CallCap != nil && *req.AISettings.CallCap < 1 {
		return &ValidationError{Message: "Call cap must be at least 1 when set"}
	}
	if req.AISettings.DisengageThreshold != nil && *req.AISettings.DisengageThreshold < 1 {
		return &ValidationError{Message: "Disengage threshold must be at least 1 when set"}
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
