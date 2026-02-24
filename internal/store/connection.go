package store

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// SavedConnection represents a saved MUD server connection
type SavedConnection struct {
	ID              uuid.UUID `json:"id"`
	UserID          uuid.UUID `json:"user_id"`
	Name            string    `json:"name"`
	Host            string    `json:"host"`
	Port            int       `json:"port"`
	Protocol        string    `json:"protocol"`
	CreatedAt       string    `json:"created_at"`
	UpdatedAt       string    `json:"updated_at"`
	LastConnectedAt *string   `json:"last_connected_at,omitempty"`
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
	// Default protocol to telnet if not specified
	if conn.Protocol == "" {
		conn.Protocol = "telnet"
	}

	query := `
		INSERT INTO saved_connections (user_id, name, host, port, protocol)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`
	return s.db.QueryRow(query, conn.UserID, conn.Name, conn.Host, conn.Port, conn.Protocol).
		Scan(&conn.ID, &conn.CreatedAt, &conn.UpdatedAt)
}

// GetByID retrieves a connection by ID (for a specific user)
func (s *ConnectionStore) GetByID(id, userID uuid.UUID) (*SavedConnection, error) {
	query := `
		SELECT id, user_id, name, host, port, protocol, created_at, updated_at, last_connected_at
		FROM saved_connections
		WHERE id = $1 AND user_id = $2
	`
	conn := &SavedConnection{}
	var lastConnectedAt sql.NullTime
	err := s.db.QueryRow(query, id, userID).Scan(
		&conn.ID, &conn.UserID, &conn.Name, &conn.Host, &conn.Port,
		&conn.Protocol, &conn.CreatedAt, &conn.UpdatedAt, &lastConnectedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if lastConnectedAt.Valid {
		lastConnectedAtStr := lastConnectedAt.Time.Format(time.RFC3339)
		conn.LastConnectedAt = &lastConnectedAtStr
	}
	return conn, err
}

// GetByUserID retrieves all connections for a user
func (s *ConnectionStore) GetByUserID(userID uuid.UUID) ([]SavedConnection, error) {
	query := `
		SELECT id, user_id, name, host, port, protocol, created_at, updated_at, last_connected_at
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
		var lastConnectedAt sql.NullTime
		err := rows.Scan(
			&conn.ID, &conn.UserID, &conn.Name, &conn.Host, &conn.Port,
			&conn.Protocol, &conn.CreatedAt, &conn.UpdatedAt, &lastConnectedAt,
		)
		if err != nil {
			return nil, err
		}
		if lastConnectedAt.Valid {
			lastConnectedAtStr := lastConnectedAt.Time.Format(time.RFC3339)
			conn.LastConnectedAt = &lastConnectedAtStr
		}
		connections = append(connections, conn)
	}
	return connections, nil
}

// GetRecent retrieves recent connections (top 5 with last_connected_at not null, ordered by last_connected_at DESC)
func (s *ConnectionStore) GetRecent(userID uuid.UUID) ([]SavedConnection, error) {
	query := `
		SELECT id, user_id, name, host, port, protocol, created_at, updated_at, last_connected_at
		FROM saved_connections
		WHERE user_id = $1 AND last_connected_at IS NOT NULL
		ORDER BY last_connected_at DESC
		LIMIT 5
	`
	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var connections []SavedConnection
	for rows.Next() {
		var conn SavedConnection
		var lastConnectedAt sql.NullTime
		err := rows.Scan(
			&conn.ID, &conn.UserID, &conn.Name, &conn.Host, &conn.Port,
			&conn.Protocol, &conn.CreatedAt, &conn.UpdatedAt, &lastConnectedAt,
		)
		if err != nil {
			return nil, err
		}
		if lastConnectedAt.Valid {
			lastConnectedAtStr := lastConnectedAt.Time.Format(time.RFC3339)
			conn.LastConnectedAt = &lastConnectedAtStr
		}
		connections = append(connections, conn)
	}
	return connections, nil
}

// Update updates an existing connection
func (s *ConnectionStore) Update(conn *SavedConnection) error {
	// Default protocol to telnet if not specified
	if conn.Protocol == "" {
		conn.Protocol = "telnet"
	}

	query := `
		UPDATE saved_connections
		SET name = $1, host = $2, port = $3, protocol = $4, updated_at = NOW()
		WHERE id = $5 AND user_id = $6
		RETURNING updated_at
	`
	return s.db.QueryRow(query, conn.Name, conn.Host, conn.Port, conn.Protocol, conn.ID, conn.UserID).
		Scan(&conn.UpdatedAt)
}

// UpdateLastConnectedAt updates the last_connected_at timestamp
func (s *ConnectionStore) UpdateLastConnectedAt(id, userID uuid.UUID) error {
	query := `
		UPDATE saved_connections
		SET last_connected_at = NOW()
		WHERE id = $1 AND user_id = $2
	`
	_, err := s.db.Exec(query, id, userID)
	return err
}

// Delete deletes a connection by ID
func (s *ConnectionStore) Delete(id, userID uuid.UUID) error {
	query := `DELETE FROM saved_connections WHERE id = $1 AND user_id = $2`
	_, err := s.db.Exec(query, id, userID)
	return err
}
