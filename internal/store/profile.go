package store

import (
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"
)

// Profile represents a per-connection user profile with keybindings, settings, and automation
type Profile struct {
	ID           uuid.UUID         `json:"id"`
	UserID       uuid.UUID         `json:"user_id"`
	ConnectionID uuid.UUID         `json:"connection_id"`
	Keybindings  map[string]string `json:"keybindings"`
	Settings     ProfileSettings   `json:"settings"`
	Aliases      Aliases           `json:"aliases"`
	Triggers     Triggers          `json:"triggers"`
	Variables    Variables         `json:"variables"`
	CreatedAt    string            `json:"created_at"`
	UpdatedAt    string            `json:"updated_at"`
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
}

// Variables wraps a list of environment variables
type Variables struct {
	Items []Variable `json:"items"`
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

// ProfileUpdate represents fields that can be updated on a profile
type ProfileUpdate struct {
	Keybindings *map[string]string `json:"keybindings,omitempty"`
	Settings    *ProfileSettings   `json:"settings,omitempty"`
	Aliases     *Aliases           `json:"aliases,omitempty"`
	Triggers    *Triggers          `json:"triggers,omitempty"`
	Variables   *Variables         `json:"variables,omitempty"`
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
		SELECT id, user_id, connection_id, keybindings, settings, aliases, triggers, variables, created_at, updated_at
		FROM profiles
		WHERE id = $1 AND user_id = $2
	`

	var profile Profile
	var keybindingsJSON, settingsJSON, aliasesJSON, triggersJSON, variablesJSON []byte

	err := s.db.QueryRow(query, profileID, userID).Scan(
		&profile.ID,
		&profile.UserID,
		&profile.ConnectionID,
		&keybindingsJSON,
		&settingsJSON,
		&aliasesJSON,
		&triggersJSON,
		&variablesJSON,
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

	// Normalize settings to defaults if empty/partial
	profile.Settings = normalizeSettings(profile.Settings)

	return &profile, nil
}

// GetProfileByConnection retrieves a profile by connection ID for a specific user
func (s *ProfileStore) GetProfileByConnection(userID, connectionID uuid.UUID) (*Profile, error) {
	query := `
		SELECT id, user_id, connection_id, keybindings, settings, aliases, triggers, variables, created_at, updated_at
		FROM profiles
		WHERE connection_id = $1 AND user_id = $2
	`

	var profile Profile
	var keybindingsJSON, settingsJSON, aliasesJSON, triggersJSON, variablesJSON []byte

	err := s.db.QueryRow(query, connectionID, userID).Scan(
		&profile.ID,
		&profile.UserID,
		&profile.ConnectionID,
		&keybindingsJSON,
		&settingsJSON,
		&aliasesJSON,
		&triggersJSON,
		&variablesJSON,
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

	query := `
		UPDATE profiles
		SET keybindings = $1, settings = $2, aliases = $3, triggers = $4, variables = $5, updated_at = NOW()
		WHERE id = $6 AND user_id = $7
		RETURNING updated_at
	`

	var updatedAt string
	err = s.db.QueryRow(query, keybindingsJSON, settingsJSON, aliasesJSON, triggersJSON, variablesJSON, profileID, userID).Scan(&updatedAt)
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

	return existing, nil
}

// DeleteProfile deletes a profile by ID
func (s *ProfileStore) DeleteProfile(userID, profileID uuid.UUID) error {
	query := `DELETE FROM profiles WHERE id = $1 AND user_id = $2`
	_, err := s.db.Exec(query, profileID, userID)
	return err
}
