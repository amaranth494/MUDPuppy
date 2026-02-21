package store

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system
type User struct {
	ID              uuid.UUID  `json:"id"`
	Email           string     `json:"email"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// UserStore handles user database operations
type UserStore struct {
	db *sql.DB
}

// NewUserStore creates a new user store
func NewUserStore(db *sql.DB) *UserStore {
	return &UserStore{db: db}
}

// Create creates a new user
func (s *UserStore) Create(email string) (*User, error) {
	id := uuid.New()
	now := time.Now()

	_, err := s.db.Exec(`
		INSERT INTO users (id, email, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
	`, id, email, now, now)

	if err != nil {
		return nil, err
	}

	return &User{
		ID:        id,
		Email:     email,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// GetByEmail retrieves a user by email
func (s *UserStore) GetByEmail(email string) (*User, error) {
	var user User
	var emailVerifiedAt sql.NullTime

	err := s.db.QueryRow(`
		SELECT id, email, email_verified_at, created_at, updated_at
		FROM users
		WHERE email = $1
	`, email).Scan(&user.ID, &user.Email, &emailVerifiedAt, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if emailVerifiedAt.Valid {
		user.EmailVerifiedAt = &emailVerifiedAt.Time
	}

	return &user, nil
}

// GetByID retrieves a user by ID
func (s *UserStore) GetByID(id uuid.UUID) (*User, error) {
	var user User
	var emailVerifiedAt sql.NullTime

	err := s.db.QueryRow(`
		SELECT id, email, email_verified_at, created_at, updated_at
		FROM users
		WHERE id = $1
	`, id).Scan(&user.ID, &user.Email, &emailVerifiedAt, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if emailVerifiedAt.Valid {
		user.EmailVerifiedAt = &emailVerifiedAt.Time
	}

	return &user, nil
}

// MarkEmailVerified marks a user's email as verified
func (s *UserStore) MarkEmailVerified(id uuid.UUID) error {
	now := time.Now()
	_, err := s.db.Exec(`
		UPDATE users
		SET email_verified_at = $1, updated_at = $1
		WHERE id = $2
	`, now, id)
	return err
}

// EmailExists checks if an email already exists
func (s *UserStore) EmailExists(email string) (bool, error) {
	var exists bool
	err := s.db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)
	`, email).Scan(&exists)
	return exists, err
}
