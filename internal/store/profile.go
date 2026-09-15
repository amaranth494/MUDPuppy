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
	AISettings            AISettings        `json:"ai_settings"`
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
	AISettings       *AISettings        `json:"ai_settings,omitempty"`
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
			conduct_rules, approach_guidance, ai_settings, policy_version_accepted, policy_accepted_at,
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
		&aiSettingsJSON,
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
			conduct_rules, approach_guidance, ai_settings, policy_version_accepted, policy_accepted_at,
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
		&aiSettingsJSON,
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

	if updates.AISettings != nil {
		aiSettingsJSON, _ = json.Marshal(*updates.AISettings)
	} else {
		aiSettingsJSON, _ = json.Marshal(existing.AISettings)
	}

	query := `
		UPDATE profiles
		SET keybindings = $1, settings = $2, aliases = $3, triggers = $4, variables = $5, timers = $6,
			conduct_rules = $7, approach_guidance = $8, ai_settings = $9, updated_at = NOW()
		WHERE id = $10 AND user_id = $11
		RETURNING updated_at
	`

	var updatedAt string
	err = s.db.QueryRow(query, keybindingsJSON, settingsJSON, aliasesJSON, triggersJSON, variablesJSON, timersJSON,
		conductRules, approachGuidance, aiSettingsJSON, profileID, userID).Scan(&updatedAt)
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
	if updates.AISettings != nil {
		existing.AISettings = *updates.AISettings
	}

	return existing, nil
}

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

// DeleteProfile deletes a profile by ID
func (s *ProfileStore) DeleteProfile(userID, profileID uuid.UUID) error {
	query := `DELETE FROM profiles WHERE id = $1 AND user_id = $2`
	_, err := s.db.Exec(query, profileID, userID)
	return err
}
