package store

import (
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"
)

// Profile represents a per-connection user profile with keybindings and settings
type Profile struct {
	ID           uuid.UUID         `json:"id"`
	UserID       uuid.UUID         `json:"user_id"`
	ConnectionID uuid.UUID         `json:"connection_id"`
	Keybindings  map[string]string `json:"keybindings"`
	Settings     ProfileSettings   `json:"settings"`
	CreatedAt    string            `json:"created_at"`
	UpdatedAt    string            `json:"updated_at"`
}

// ProfileSettings contains UI and behavior settings for a profile
type ProfileSettings struct {
	ScrollbackLimit int  `json:"scrollback_limit"`
	EchoInput       bool `json:"echo_input"`
	TimestampOutput bool `json:"timestamp_output"`
	WordWrap        bool `json:"word_wrap"`
}

// DefaultProfileSettings returns the default profile settings
func DefaultProfileSettings() ProfileSettings {
	return ProfileSettings{
		ScrollbackLimit: 1000,
		EchoInput:       false,
		TimestampOutput: false,
		WordWrap:        true,
	}
}

// ProfileUpdate represents fields that can be updated on a profile
type ProfileUpdate struct {
	Keybindings *map[string]string `json:"keybindings,omitempty"`
	Settings    *ProfileSettings   `json:"settings,omitempty"`
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
		INSERT INTO profiles (user_id, connection_id, keybindings, settings)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`

	defaultKeybindings := "{}"
	defaultSettings := DefaultProfileSettings()
	settingsJSON, _ := json.Marshal(defaultSettings)

	var profile Profile
	err := s.db.QueryRow(query, userID, connectionID, defaultKeybindings, settingsJSON).
		Scan(&profile.ID, &profile.CreatedAt, &profile.UpdatedAt)
	if err != nil {
		return nil, err
	}

	profile.UserID = userID
	profile.ConnectionID = connectionID
	profile.Keybindings = make(map[string]string)
	profile.Settings = defaultSettings

	return &profile, nil
}

// GetProfile retrieves a profile by ID for a specific user
func (s *ProfileStore) GetProfile(userID, profileID uuid.UUID) (*Profile, error) {
	query := `
		SELECT id, user_id, connection_id, keybindings, settings, created_at, updated_at
		FROM profiles
		WHERE id = $1 AND user_id = $2
	`

	var profile Profile
	var keybindingsJSON, settingsJSON []byte

	err := s.db.QueryRow(query, profileID, userID).Scan(
		&profile.ID,
		&profile.UserID,
		&profile.ConnectionID,
		&keybindingsJSON,
		&settingsJSON,
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

	return &profile, nil
}

// GetProfileByConnection retrieves a profile by connection ID for a specific user
func (s *ProfileStore) GetProfileByConnection(userID, connectionID uuid.UUID) (*Profile, error) {
	query := `
		SELECT id, user_id, connection_id, keybindings, settings, created_at, updated_at
		FROM profiles
		WHERE connection_id = $1 AND user_id = $2
	`

	var profile Profile
	var keybindingsJSON, settingsJSON []byte

	err := s.db.QueryRow(query, connectionID, userID).Scan(
		&profile.ID,
		&profile.UserID,
		&profile.ConnectionID,
		&keybindingsJSON,
		&settingsJSON,
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

	query := `
		UPDATE profiles
		SET keybindings = $1, settings = $2, updated_at = NOW()
		WHERE id = $3 AND user_id = $4
		RETURNING updated_at
	`

	var updatedAt string
	err = s.db.QueryRow(query, keybindingsJSON, settingsJSON, profileID, userID).Scan(&updatedAt)
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

	return existing, nil
}

// DeleteProfile deletes a profile by ID
func (s *ProfileStore) DeleteProfile(userID, profileID uuid.UUID) error {
	query := `DELETE FROM profiles WHERE id = $1 AND user_id = $2`
	_, err := s.db.Exec(query, profileID, userID)
	return err
}
