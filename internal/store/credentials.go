package store

import (
	"database/sql"

	"github.com/google/uuid"
)

// ConnectionCredential represents stored credentials for a connection
type ConnectionCredential struct {
	ID                uuid.UUID `json:"id"`
	ConnectionID      uuid.UUID `json:"connection_id"`
	Username          string    `json:"username,omitempty"`
	EncryptedPassword []byte    `json:"encrypted_password"`
	KeyVersion        int       `json:"key_version"`
	AutoLogin         bool      `json:"auto_login"`
	CreatedAt         string    `json:"created_at"`
	UpdatedAt         string    `json:"updated_at"`
}

// CredentialStatus represents the status of credentials (returned to UI, never the actual credentials)
type CredentialStatus struct {
	HasCredentials   bool `json:"has_credentials"`
	AutoLoginEnabled bool `json:"auto_login_enabled"`
}

// CredentialsStore handles connection credentials database operations
type CredentialsStore struct {
	db *sql.DB
}

// NewCredentialsStore creates a new credentials store
func NewCredentialsStore(db *sql.DB) *CredentialsStore {
	return &CredentialsStore{db: db}
}

// Create stores new credentials for a connection
func (s *CredentialsStore) Create(cred *ConnectionCredential) error {
	query := `
		INSERT INTO connection_credentials (connection_id, username, encrypted_password, key_version, auto_login)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`
	return s.db.QueryRow(query, cred.ConnectionID, cred.Username, cred.EncryptedPassword, cred.KeyVersion, cred.AutoLogin).
		Scan(&cred.ID, &cred.CreatedAt, &cred.UpdatedAt)
}

// GetByConnectionID retrieves credentials for a connection
func (s *CredentialsStore) GetByConnectionID(connectionID uuid.UUID) (*ConnectionCredential, error) {
	query := `
		SELECT id, connection_id, username, encrypted_password, key_version, auto_login, created_at, updated_at
		FROM connection_credentials
		WHERE connection_id = $1
	`
	cred := &ConnectionCredential{}
	err := s.db.QueryRow(query, connectionID).Scan(
		&cred.ID, &cred.ConnectionID, &cred.Username, &cred.EncryptedPassword,
		&cred.KeyVersion, &cred.AutoLogin, &cred.CreatedAt, &cred.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return cred, err
}

// GetStatus returns the credential status (has_credentials, auto_login_enabled) without exposing credentials
func (s *CredentialsStore) GetStatus(connectionID uuid.UUID) (*CredentialStatus, error) {
	query := `
		SELECT auto_login
		FROM connection_credentials
		WHERE connection_id = $1
	`
	var autoLogin bool
	err := s.db.QueryRow(query, connectionID).Scan(&autoLogin)
	if err == sql.ErrNoRows {
		return &CredentialStatus{HasCredentials: false, AutoLoginEnabled: false}, nil
	}
	if err != nil {
		return nil, err
	}
	return &CredentialStatus{HasCredentials: true, AutoLoginEnabled: autoLogin}, nil
}

// Update updates credentials for a connection
func (s *CredentialsStore) Update(cred *ConnectionCredential) error {
	// If no password provided, we need to get the existing password to preserve it
	if len(cred.EncryptedPassword) == 0 {
		// Get existing credentials
		existing, err := s.GetByConnectionID(cred.ConnectionID)
		if err != nil {
			return err
		}
		if existing != nil {
			cred.EncryptedPassword = existing.EncryptedPassword
			cred.KeyVersion = existing.KeyVersion
		}
	}

	query := `
		UPDATE connection_credentials
		SET username = $1, encrypted_password = $2, key_version = $3, auto_login = $4, updated_at = NOW()
		WHERE connection_id = $5
		RETURNING updated_at
	`
	return s.db.QueryRow(query, cred.Username, cred.EncryptedPassword, cred.KeyVersion, cred.AutoLogin, cred.ConnectionID).
		Scan(&cred.UpdatedAt)
}

// Delete removes credentials for a connection
func (s *CredentialsStore) Delete(connectionID uuid.UUID) error {
	query := `DELETE FROM connection_credentials WHERE connection_id = $1`
	_, err := s.db.Exec(query, connectionID)
	return err
}

// Upsert creates or updates credentials for a connection
func (s *CredentialsStore) Upsert(cred *ConnectionCredential) error {
	// First try to update
	err := s.Update(cred)
	if err == sql.ErrNoRows {
		// If no rows updated, create new
		return s.Create(cred)
	}
	return err
}
