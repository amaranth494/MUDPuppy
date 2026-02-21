package store

import (
	"database/sql"

	"github.com/google/uuid"
)

// SavedConnection represents a saved MUD server connection
type SavedConnection struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Name      string    `json:"name"`
	Host      string    `json:"host"`
	Port      int       `json:"port"`
	CreatedAt string    `json:"created_at"`
	UpdatedAt string    `json:"updated_at"`
}

// ConnectionStore handles saved connections database operations
type ConnectionStore struct {
	db *sql.DB
}

// NewConnectionStore creates a new connection store
func NewConnectionStore(db *sql.DB) *ConnectionStore {
	return &ConnectionStore{db: db}
}

// Create saves a new connection for a user
func (s *ConnectionStore) Create(conn *SavedConnection) error {
	query := `
		INSERT INTO saved_connections (user_id, name, host, port)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`
	return s.db.QueryRow(query, conn.UserID, conn.Name, conn.Host, conn.Port).
		Scan(&conn.ID, &conn.CreatedAt, &conn.UpdatedAt)
}

// GetByID retrieves a connection by ID
func (s *ConnectionStore) GetByID(id uuid.UUID) (*SavedConnection, error) {
	query := `
		SELECT id, user_id, name, host, port, created_at, updated_at
		FROM saved_connections
		WHERE id = $1
	`
	conn := &SavedConnection{}
	err := s.db.QueryRow(query, id).Scan(
		&conn.ID, &conn.UserID, &conn.Name, &conn.Host, &conn.Port,
		&conn.CreatedAt, &conn.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return conn, err
}

// GetByUserID retrieves all connections for a user
func (s *ConnectionStore) GetByUserID(userID uuid.UUID) ([]SavedConnection, error) {
	query := `
		SELECT id, user_id, name, host, port, created_at, updated_at
		FROM saved_connections
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var connections []SavedConnection
	for rows.Next() {
		var conn SavedConnection
		err := rows.Scan(
			&conn.ID, &conn.UserID, &conn.Name, &conn.Host, &conn.Port,
			&conn.CreatedAt, &conn.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		connections = append(connections, conn)
	}
	return connections, nil
}

// Update updates an existing connection
func (s *ConnectionStore) Update(conn *SavedConnection) error {
	query := `
		UPDATE saved_connections
		SET name = $1, host = $2, port = $3, updated_at = NOW()
		WHERE id = $4 AND user_id = $5
		RETURNING updated_at
	`
	return s.db.QueryRow(query, conn.Name, conn.Host, conn.Port, conn.ID, conn.UserID).
		Scan(&conn.UpdatedAt)
}

// Delete deletes a connection by ID
func (s *ConnectionStore) Delete(id, userID uuid.UUID) error {
	query := `DELETE FROM saved_connections WHERE id = $1 AND user_id = $2`
	_, err := s.db.Exec(query, id, userID)
	return err
}
