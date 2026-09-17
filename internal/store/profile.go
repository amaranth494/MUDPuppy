package store

import (
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"
)

// Profile represents a per-connection user profile with keybindings, settings, and automation
type Profile struct {
	ID                    uuid.UUID         `json:"id"`
	UserID                uuid.UUID         `json:"user_id"`
	ConnectionID          uuid.UUID         `json:"connection_id"`
	Keybindings           map[string]string `json:"keybindings"`
	Settings              ProfileSettings   `json:"settings"`
	Aliases               Aliases           `json:"aliases"`
	Triggers              Triggers          `json:"triggers"`
	Variables             Variables         `json:"variables"`
	Timers                Timers            `json:"timers"`
	ConductRules          string            `json:"conduct_rules"`
	ApproachGuidance      string            `json:"approach_guidance"`
	NeverIssueList        string            `json:"never_issue_list"`
	AISettings            AISettings        `json:"ai_settings"`
	SessionGoal           string            `json:"session_goal"`
	PolicyVersionAccepted *string           `json:"policy_version_accepted"`
	PolicyAcceptedAt      *string           `json:"policy_accepted_at"`
	CreatedAt             string            `json:"created_at"`
	UpdatedAt             string            `json:"updated_at"`
}

// Alias represents a command alias for input transformation
type Alias struct {
	ID          string `json:"id"`
	Pattern     string `json:"pattern"`
	Replacement string `json:"replacement"`
	Enabled     bool   `json:"enabled"`
}

// Aliases wraps a list of aliases
type Aliases struct {
	Items []Alias `json:"items"`
}

// Trigger represents an output-driven automation trigger
type Trigger struct {
	ID       string `json:"id"`
	Match    string `json:"match"`
	Type     string `json:"type"`
	Action   string `json:"action"`
	Cooldown int    `json:"cooldown_ms"`
	Enabled  bool   `json:"enabled"`
}

// Triggers wraps a list of triggers
type Triggers struct {
	Items []Trigger `json:"items"`
}

// Variable represents an environment variable for automation
type Variable struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
	Type  string `json:"type,omitempty"`
}

// Variables wraps a list of environment variables
type Variables struct {
	Items []Variable `json:"items"`
}

// Timer represents a time-based automation trigger
type Timer struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Duration int    `json:"duration"` // in milliseconds
	Repeat   bool   `json:"repeat"`
	Commands string `json:"commands"`
	Enabled  bool   `json:"enabled"`
}

// Timers wraps a list of timers
type Timers struct {
	Items []Timer `json:"items"`
}

// AISettings holds the AI Player's per-profile configuration. A nil CallCap
// or DisengageThreshold and a blank ModelName are meaningful, user-visible
// "blank" states (D-09, D-10, D-11) and must round-trip unchanged; they are
// resolved to concrete values only by ResolveAISettings, never on read.
// There is no reconnect field, and none may be added (D-13).
type AISettings struct {
	ModelName          string `json:"model_name"`
	CallCap            *int   `json:"call_cap"`
	DisengageThreshold *int   `json:"disengage_threshold"`
}

// ProfileSettings contains UI and behavior settings for a profile
type ProfileSettings struct {
	ScrollbackLimit   int  `json:"scrollback_limit"`
	EchoInput         bool `json:"echo_input"`
	TimestampOutput   bool `json:"timestamp_output"`
	WordWrap          bool `json:"word_wrap"`
	AutomationEnabled bool `json:"automation_enabled"`
}

// DefaultProfileSettings returns the default profile settings
func DefaultProfileSettings() ProfileSettings {
	return ProfileSettings{
		ScrollbackLimit:   1000,
		EchoInput:         false,
		TimestampOutput:   false,
		WordWrap:          true,
		AutomationEnabled: true,
	}
}

// normalizeSettings ensures all settings fields are populated with defaults if empty/zero
func normalizeSettings(s ProfileSettings) ProfileSettings {
	if s.ScrollbackLimit == 0 {
		s.ScrollbackLimit = DefaultProfileSettings().ScrollbackLimit
	}
	// Note: bool fields default to false which matches DefaultProfileSettings for EchoInput, TimestampOutput
	// but WordWrap should be true - handle explicitly
	if !s.WordWrap && s.ScrollbackLimit == 0 {
		// Only set WordWrap default if we also would set scrollback (indicates empty settings)
		s.WordWrap = true
	}
	return s
}

// ProfileUpdate represents fields that can be updated on a profile.
// Deliberately excludes PolicyVersionAccepted and PolicyAcceptedAt (T-1-01):
// acceptance is writable only through AcceptPolicy, never through a general
// profile update.
type ProfileUpdate struct {
	Keybindings      *map[string]string `json:"keybindings,omitempty"`
	Settings         *ProfileSettings   `json:"settings,omitempty"`
	Aliases          *Aliases           `json:"aliases,omitempty"`
	Triggers         *Triggers          `json:"triggers,omitempty"`
	Variables        *Variables         `json:"variables,omitempty"`
	Timers           *Timers            `json:"timers,omitempty"`
	ConductRules     *string            `json:"conduct_rules,omitempty"`
	ApproachGuidance *string            `json:"approach_guidance,omitempty"`
	NeverIssueList   *string            `json:"never_issue_list,omitempty"`
	AISettings       *AISettings        `json:"ai_settings,omitempty"`
	// There is deliberately no SessionGoal here (code review WR-08 of Phase
	// 4): the goal is saved only through ProfileStore.UpdateSessionGoal, a
	// targeted statement, never through UpdateProfile's whole-row write.
}

// DefaultAliases returns the default aliases structure
func DefaultAliases() Aliases {
	return Aliases{Items: []Alias{}}
}

// DefaultTriggers returns the default triggers structure
func DefaultTriggers() Triggers {
	return Triggers{Items: []Trigger{}}
}

// DefaultVariables returns the default variables structure
func DefaultVariables() Variables {
	return Variables{Items: []Variable{}}
}

// DefaultTimers returns the default timers structure
func DefaultTimers() Timers {
	return Timers{Items: []Timer{}}
}

// ProfileStore handles profiles database operations
type ProfileStore struct {
	db *sql.DB
}

// NewProfileStore creates a new profile store
func NewProfileStore(db *sql.DB) *ProfileStore {
	return &ProfileStore{db: db}
}

// CreateProfile creates a new profile for a connection
func (s *ProfileStore) CreateProfile(userID, connectionID uuid.UUID) (*Profile, error) {
	query := `
		INSERT INTO profiles (user_id, connection_id, keybindings, settings, aliases, triggers, variables)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`

	defaultKeybindings := "{}"
	defaultSettings := DefaultProfileSettings()
	settingsJSON, _ := json.Marshal(defaultSettings)
	defaultAliases := DefaultAliases()
	aliasesJSON, _ := json.Marshal(defaultAliases)
	defaultTriggers := DefaultTriggers()
	triggersJSON, _ := json.Marshal(defaultTriggers)
	defaultVariables := DefaultVariables()
	variablesJSON, _ := json.Marshal(defaultVariables)

	var profile Profile
	err := s.db.QueryRow(query, userID, connectionID, defaultKeybindings, settingsJSON, aliasesJSON, triggersJSON, variablesJSON).
		Scan(&profile.ID, &profile.CreatedAt, &profile.UpdatedAt)
	if err != nil {
		return nil, err
	}

	profile.UserID = userID
	profile.ConnectionID = connectionID
	profile.Keybindings = make(map[string]string)
	profile.Settings = defaultSettings
	profile.Aliases = defaultAliases
	profile.Triggers = defaultTriggers
	profile.Variables = defaultVariables

	return &profile, nil
}

// GetProfile retrieves a profile by ID for a specific user
func (s *ProfileStore) GetProfile(userID, profileID uuid.UUID) (*Profile, error) {
	query := `
		SELECT id, user_id, connection_id, keybindings, settings, aliases, triggers, variables, timers,
			conduct_rules, approach_guidance, never_issue_list, ai_settings, session_goal, policy_version_accepted, policy_accepted_at,
			created_at, updated_at
		FROM profiles
		WHERE id = $1 AND user_id = $2
	`

	var profile Profile
	var keybindingsJSON, settingsJSON, aliasesJSON, triggersJSON, variablesJSON, timersJSON, aiSettingsJSON []byte
	var policyVersionAcceptedNS, policyAcceptedAtNS sql.NullString

	err := s.db.QueryRow(query, profileID, userID).Scan(
		&profile.ID,
		&profile.UserID,
		&profile.ConnectionID,
		&keybindingsJSON,
		&settingsJSON,
		&aliasesJSON,
		&triggersJSON,
		&variablesJSON,
		&timersJSON,
		&profile.ConductRules,
		&profile.ApproachGuidance,
		&profile.NeverIssueList,
		&aiSettingsJSON,
		&profile.SessionGoal,
		&policyVersionAcceptedNS,
		&policyAcceptedAtNS,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Parse JSONB fields
	if err := json.Unmarshal(keybindingsJSON, &profile.Keybindings); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(settingsJSON, &profile.Settings); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(aliasesJSON, &profile.Aliases); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(triggersJSON, &profile.Triggers); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(variablesJSON, &profile.Variables); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(timersJSON, &profile.Timers); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(aiSettingsJSON, &profile.AISettings); err != nil {
		return nil, err
	}

	if policyVersionAcceptedNS.Valid {
		profile.PolicyVersionAccepted = &policyVersionAcceptedNS.String
	} else {
		profile.PolicyVersionAccepted = nil
	}
	if policyAcceptedAtNS.Valid {
		profile.PolicyAcceptedAt = &policyAcceptedAtNS.String
	} else {
		profile.PolicyAcceptedAt = nil
	}

	// Normalize settings to defaults if empty/partial
	profile.Settings = normalizeSettings(profile.Settings)

	return &profile, nil
}

// GetProfileByConnection retrieves a profile by connection ID for a specific user
func (s *ProfileStore) GetProfileByConnection(userID, connectionID uuid.UUID) (*Profile, error) {
	query := `
		SELECT id, user_id, connection_id, keybindings, settings, aliases, triggers, variables, timers,
			conduct_rules, approach_guidance, never_issue_list, ai_settings, session_goal, policy_version_accepted, policy_accepted_at,
			created_at, updated_at
		FROM profiles
		WHERE connection_id = $1 AND user_id = $2
	`

	var profile Profile
	var keybindingsJSON, settingsJSON, aliasesJSON, triggersJSON, variablesJSON, timersJSON, aiSettingsJSON []byte
	var policyVersionAcceptedNS, policyAcceptedAtNS sql.NullString

	err := s.db.QueryRow(query, connectionID, userID).Scan(
		&profile.ID,
		&profile.UserID,
		&profile.ConnectionID,
		&keybindingsJSON,
		&settingsJSON,
		&aliasesJSON,
		&triggersJSON,
		&variablesJSON,
		&timersJSON,
		&profile.ConductRules,
		&profile.ApproachGuidance,
		&profile.NeverIssueList,
		&aiSettingsJSON,
		&profile.SessionGoal,
		&policyVersionAcceptedNS,
		&policyAcceptedAtNS,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Parse JSONB fields
	if err := json.Unmarshal(keybindingsJSON, &profile.Keybindings); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(settingsJSON, &profile.Settings); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(aliasesJSON, &profile.Aliases); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(triggersJSON, &profile.Triggers); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(variablesJSON, &profile.Variables); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(timersJSON, &profile.Timers); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(aiSettingsJSON, &profile.AISettings); err != nil {
		return nil, err
	}

	if policyVersionAcceptedNS.Valid {
		profile.PolicyVersionAccepted = &policyVersionAcceptedNS.String
	} else {
		profile.PolicyVersionAccepted = nil
	}
	if policyAcceptedAtNS.Valid {
		profile.PolicyAcceptedAt = &policyAcceptedAtNS.String
	} else {
		profile.PolicyAcceptedAt = nil
	}

	// Normalize settings to defaults if empty/partial
	profile.Settings = normalizeSettings(profile.Settings)

	return &profile, nil
}

// UpdateProfile updates a profile with the given updates
func (s *ProfileStore) UpdateProfile(userID, profileID uuid.UUID, updates *ProfileUpdate) (*Profile, error) {
	// First get the existing profile
	existing, err := s.GetProfile(userID, profileID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, nil
	}

	// Build the update query dynamically based on what's being updated
	var keybindingsJSON []byte
	var settingsJSON []byte
	var aliasesJSON []byte
	var triggersJSON []byte
	var variablesJSON []byte
	var timersJSON []byte
	var aiSettingsJSON []byte
	var conductRules string
	var approachGuidance string
	var neverIssueList string

	if updates.Keybindings != nil {
		keybindingsJSON, _ = json.Marshal(*updates.Keybindings)
	} else {
		keybindingsJSON, _ = json.Marshal(existing.Keybindings)
	}

	if updates.Settings != nil {
		settingsJSON, _ = json.Marshal(*updates.Settings)
	} else {
		settingsJSON, _ = json.Marshal(existing.Settings)
	}

	if updates.Aliases != nil {
		aliasesJSON, _ = json.Marshal(*updates.Aliases)
	} else {
		aliasesJSON, _ = json.Marshal(existing.Aliases)
	}

	if updates.Triggers != nil {
		triggersJSON, _ = json.Marshal(*updates.Triggers)
	} else {
		triggersJSON, _ = json.Marshal(existing.Triggers)
	}

	if updates.Variables != nil {
		variablesJSON, _ = json.Marshal(*updates.Variables)
	} else {
		variablesJSON, _ = json.Marshal(existing.Variables)
	}

	if updates.Timers != nil {
		timersJSON, _ = json.Marshal(*updates.Timers)
	} else {
		timersJSON, _ = json.Marshal(existing.Timers)
	}

	if updates.ConductRules != nil {
		conductRules = *updates.ConductRules
	} else {
		conductRules = existing.ConductRules
	}

	if updates.ApproachGuidance != nil {
		approachGuidance = *updates.ApproachGuidance
	} else {
		approachGuidance = existing.ApproachGuidance
	}

	if updates.NeverIssueList != nil {
		neverIssueList = *updates.NeverIssueList
	} else {
		neverIssueList = existing.NeverIssueList
	}

	if updates.AISettings != nil {
		aiSettingsJSON, _ = json.Marshal(*updates.AISettings)
	} else {
		aiSettingsJSON, _ = json.Marshal(existing.AISettings)
	}

	var updatedAt string
	err = s.db.QueryRow(updateProfileSQL, keybindingsJSON, settingsJSON, aliasesJSON, triggersJSON, variablesJSON, timersJSON,
		conductRules, approachGuidance, neverIssueList, aiSettingsJSON, profileID, userID).Scan(&updatedAt)
	if err != nil {
		return nil, err
	}

	// Return the updated profile
	existing.UpdatedAt = updatedAt
	if updates.Keybindings != nil {
		existing.Keybindings = *updates.Keybindings
	}
	if updates.Settings != nil {
		existing.Settings = *updates.Settings
	}
	if updates.Aliases != nil {
		existing.Aliases = *updates.Aliases
	}
	if updates.Triggers != nil {
		existing.Triggers = *updates.Triggers
	}
	if updates.Variables != nil {
		existing.Variables = *updates.Variables
	}
	if updates.Timers != nil {
		existing.Timers = *updates.Timers
	}
	if updates.ConductRules != nil {
		existing.ConductRules = *updates.ConductRules
	}
	if updates.ApproachGuidance != nil {
		existing.ApproachGuidance = *updates.ApproachGuidance
	}
	if updates.NeverIssueList != nil {
		existing.NeverIssueList = *updates.NeverIssueList
	}
	if updates.AISettings != nil {
		existing.AISettings = *updates.AISettings
	}
	return existing, nil
}

// updateProfileSQL is UpdateProfile's whole-row write. It deliberately does
// NOT name session_goal (code review WR-08 of Phase 4): this statement
// writes back every column it names from a row read moments earlier, and the
// goal is edited during play through its own targeted statement
// (updateSessionGoalSQL). Were session_goal still listed here, any settings,
// alias, trigger or variable save that interleaved with a goal edit would
// write the OLD goal back over the new one.
// TestUpdateProfileLeavesTheGoalAlone pins this.
const updateProfileSQL = `
		UPDATE profiles
		SET keybindings = $1, settings = $2, aliases = $3, triggers = $4, variables = $5, timers = $6,
			conduct_rules = $7, approach_guidance = $8, never_issue_list = $9, ai_settings = $10, updated_at = NOW()
		WHERE id = $11 AND user_id = $12
		RETURNING updated_at
	`

// AcceptPolicy records one-time Safety and Abuse policy acceptance on a
// profile. The version is supplied by the caller from the server's embedded
// policy and is never read from a request body (T-1-02). Acceptance is
// one-time and never re-written (D-06): the guarded UPDATE only fires when
// policy_accepted_at IS NULL, and a "0 rows affected" second call is treated
// as a successful no-op rather than an error.
func (s *ProfileStore) AcceptPolicy(userID, profileID uuid.UUID, version string) (*Profile, error) {
	query := `
		UPDATE profiles
		SET policy_version_accepted = $1, policy_accepted_at = NOW(), updated_at = NOW()
		WHERE id = $2 AND user_id = $3 AND policy_accepted_at IS NULL
	`
	if _, err := s.db.Exec(query, version, profileID, userID); err != nil {
		return nil, err
	}

	return s.GetProfile(userID, profileID)
}

// updateSessionGoalSQL is the goal box's own statement (code review WR-08 of
// Phase 4), declared as a constant in this package's pinned-SQL style so
// TestUpdateSessionGoalTouchesOnlyTheGoal can assert on its exact text. It
// sets the goal column (and updated_at) and NOTHING else, scoped to the
// owner's own profile row.
const updateSessionGoalSQL = `UPDATE profiles
	 SET session_goal = $1, updated_at = NOW()
	 WHERE id = $2 AND user_id = $3`

// UpdateSessionGoal saves the session goal and only the session goal (code
// review WR-08 of Phase 4). The goal box used to save through
// UpdateProfile, which reads the whole row and then writes EVERY column
// back; the goal is saved on every blur, during play, so a goal save that
// interleaved with an AI Settings, alias, trigger or variable save wrote
// that request's stale columns back over it -- new conduct rules, the
// Never-issue list or the call cap silently reverted by a goal edit, or the
// goal by a settings save. A targeted UPDATE cannot do that in either
// direction. Returns the profile as stored afterwards, or (nil, nil) when
// the row does not exist or is not this user's, exactly as AcceptPolicy
// does.
func (s *ProfileStore) UpdateSessionGoal(userID, profileID uuid.UUID, goal string) (*Profile, error) {
	if _, err := s.db.Exec(updateSessionGoalSQL, goal, profileID, userID); err != nil {
		return nil, err
	}
	return s.GetProfile(userID, profileID)
}

// DeleteProfile deletes a profile by ID
func (s *ProfileStore) DeleteProfile(userID, profileID uuid.UUID) error {
	query := `DELETE FROM profiles WHERE id = $1 AND user_id = $2`
	_, err := s.db.Exec(query, profileID, userID)
	return err
}

// DefaultDisengageThreshold is the number of consecutive transient AI
// failures tolerated before disengage, used when a profile's disengage
// threshold is left blank. Claude's Discretion (CONTEXT.md): 3 is the
// chosen value. Non-transient failures (a missing key, an auth error)
// disengage on the first occurrence regardless of this threshold. Phase 1
// only resolves this value; the loop that consumes it is Phase 4.
const DefaultDisengageThreshold = 3

// EngageGateRefusalMessage is shown when AI engagement is refused because
// the profile has not recorded Safety and Abuse policy acceptance. This
// exact string is the Phase 2 contract (D-05, 01-UI-SPEC.md Copywriting
// Contract) — Phase 2's #AUTO ON handling surfaces it verbatim.
const EngageGateRefusalMessage = "AI Player has not been configured for this connection. Accept the Safety and Abuse policy in AI Player settings before engaging autopilot."

// ResolvedAISettings is the concrete, engine-usable resolution of a
// profile's AISettings: blank values replaced with their documented
// defaults. It must never be computed inside GetProfile/GetProfileByConnection
// — blank AI settings must round-trip to the browser unchanged (D-09,
// D-10, D-11); resolution happens only where the engine needs a concrete
// number (Phase 3/Phase 4).
type ResolvedAISettings struct {
	ModelName          string
	CallCapSet         bool
	CallCap            int
	DisengageThreshold int
}

// ResolveAISettings resolves a profile's (possibly blank) AISettings to
// concrete values: a blank ModelName resolves to serverDefaultModel, a nil
// CallCap means no cap at all (CallCapSet is false), and a nil
// DisengageThreshold resolves to DefaultDisengageThreshold. Non-nil/non-blank
// values pass through unchanged.
func ResolveAISettings(s AISettings, serverDefaultModel string) ResolvedAISettings {
	resolved := ResolvedAISettings{
		ModelName:          s.ModelName,
		DisengageThreshold: DefaultDisengageThreshold,
	}

	if resolved.ModelName == "" {
		resolved.ModelName = serverDefaultModel
	}

	if s.CallCap != nil {
		resolved.CallCapSet = true
		resolved.CallCap = *s.CallCap
	}

	if s.DisengageThreshold != nil {
		resolved.DisengageThreshold = *s.DisengageThreshold
	}

	return resolved
}

// EngageGateAllowed reports whether AI engagement may proceed for a profile,
// based solely on its recorded policy acceptance. It takes plain values read
// from the server-fetched row, never a client-supplied flag (T-1-05), and it
// compares nothing to the current policy version — a later policy change
// never re-gates (D-06).
func EngageGateAllowed(policyVersionAccepted *string, policyAcceptedAt *string) bool {
	return policyAcceptedAt != nil && policyVersionAccepted != nil && *policyVersionAccepted != ""
}
