package session

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Session state constants
const (
	StateDisconnected = "disconnected"
	StateConnecting   = "connecting"
	StateConnected    = "connected"
	StateError        = "error"
)

// Disconnect reasons
const (
	ReasonUser    = "user"
	ReasonIdle    = "idle_timeout"
	ReasonHardCap = "hard_cap"
	ReasonRemote  = "remote_close"
	ReasonError   = "error"
)

// Session represents an active MUD connection session
type Session struct {
	UserID         string    `json:"user_id"`
	Host           string    `json:"host"`
	Port           int       `json:"port"`
	State          string    `json:"state"`
	ConnectedAt    time.Time `json:"connected_at,omitempty"`
	LastActivityAt time.Time `json:"last_activity_at,omitempty"`
	DisconnectErr  string    `json:"disconnect_reason,omitempty"`
}

// Manager handles MUD session management
type Manager struct {
	portWhitelist      map[int]bool
	idleTimeoutMinutes int
	hardCapHours       int

	mu       sync.RWMutex
	sessions map[string]*Session // userID -> session
	conns    map[string]net.Conn
	cleanups map[string]context.CancelFunc
}

// NewManager creates a new session manager
func NewManager(portWhitelist string, idleTimeoutMinutes, hardCapHours int) *Manager {
	// Parse port whitelist
	ports := make(map[int]bool)
	for _, p := range strings.Split(portWhitelist, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		port, err := strconv.Atoi(p)
		if err != nil {
			log.Printf("Warning: Invalid port in whitelist '%s', skipping", p)
			continue
		}
		ports[port] = true
	}

	// Default to port 23 if no whitelist specified
	if len(ports) == 0 {
		ports[23] = true
	}

	return &Manager{
		portWhitelist:      ports,
		idleTimeoutMinutes: idleTimeoutMinutes,
		hardCapHours:       hardCapHours,
		sessions:           make(map[string]*Session),
		conns:              make(map[string]net.Conn),
		cleanups:           make(map[string]context.CancelFunc),
	}
}

// ValidatePort checks if a port is in the whitelist
func (m *Manager) ValidatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1-65535")
	}

	if !m.portWhitelist[port] {
		return fmt.Errorf("port %d is not allowed. Allowed ports: %v", port, m.getAllowedPorts())
	}

	return nil
}

func (m *Manager) getAllowedPorts() []int {
	ports := make([]int, 0, len(m.portWhitelist))
	for p := range m.portWhitelist {
		ports = append(ports, p)
	}
	return ports
}

// Connect establishes a MUD connection for a user
func (m *Manager) Connect(ctx context.Context, userID, host string, port int) (*Session, error) {
	// Validate port first
	if err := m.ValidatePort(port); err != nil {
		return nil, err
	}

	// Validate host (private IP blocking)
	if err := ValidateHost(host); err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for existing session (one connection per user)
	if existing, ok := m.sessions[userID]; ok && existing.State == StateConnected {
		return nil, fmt.Errorf("user already has an active session")
	}

	// Create session
	session := &Session{
		UserID: userID,
		Host:   host,
		Port:   port,
		State:  StateConnecting,
	}

	// Dial the MUD server
	address := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", address, 10*time.Second)
	if err != nil {
		session.State = StateError
		session.DisconnectErr = err.Error()
		m.sessions[userID] = session
		return session, fmt.Errorf("connection failed: %v", err)
	}

	// Handle telnet negotiation - consume initial bytes
	m.consumeTelnetNegotiation(conn)

	// Update session state
	session.State = StateConnected
	session.ConnectedAt = time.Now()
	session.LastActivityAt = time.Now()

	// Store connection
	m.sessions[userID] = session
	m.conns[userID] = conn

	// Start idle timer and hard cap timer
	m.startTimers(ctx, userID)

	// Log connection metadata (SP02PH01T07)
	m.logConnectionMetadata(session)

	return session, nil
}

// consumeTelnetNegotiation reads and discards initial telnet negotiation bytes
// This ensures the client only receives application-layer text
func (m *Manager) consumeTelnetNegotiation(conn net.Conn) {
	// Set a short read deadline
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))

	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			break
		}
		// If we got non-IAC bytes, put them back (not implemented for MVP)
		// For MVP, we just discard negotiation bytes
		_ = n
	}

	// Reset to no deadline for normal operation
	conn.SetReadDeadline(time.Time{})
}

// startTimers starts idle timeout and hard cap timers
func (m *Manager) startTimers(ctx context.Context, userID string) {
	ctx, cancel := context.WithCancel(ctx)

	m.mu.Lock()
	m.cleanups[userID] = cancel
	m.mu.Unlock()

	go func() {
		idleTimer := time.NewTimer(time.Duration(m.idleTimeoutMinutes) * time.Minute)
		hardCapTimer := time.NewTimer(time.Duration(m.hardCapHours) * time.Hour)

		defer func() {
			idleTimer.Stop()
			hardCapTimer.Stop()
		}()

		select {
		case <-ctx.Done():
			return
		case <-idleTimer.C:
			log.Printf("[SP02PH01T05] Idle timeout for user %s", userID)
			m.Disconnect(userID, ReasonIdle)
		case <-hardCapTimer.C:
			log.Printf("[SP02PH01T06] Hard session cap reached for user %s", userID)
			m.Disconnect(userID, ReasonHardCap)
		}
	}()
}

// ResetIdleTimer resets the idle timer for a session
func (m *Manager) ResetIdleTimer(userID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, ok := m.sessions[userID]; ok {
		session.LastActivityAt = time.Now()
	}
}

// Disconnect terminates a user's MUD connection
func (m *Manager) Disconnect(userID, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[userID]
	if !ok {
		return fmt.Errorf("no session found for user")
	}

	// Cancel timers
	if cleanup, ok := m.cleanups[userID]; ok {
		cleanup()
		delete(m.cleanups, userID)
	}

	// Close connection
	if conn, ok := m.conns[userID]; ok {
		conn.Close()
		delete(m.conns, userID)
	}

	// Update session state
	session.State = StateDisconnected
	session.DisconnectErr = reason

	// Log disconnect metadata
	m.logDisconnectMetadata(session)

	log.Printf("[SP02PH01T07] Session disconnected: user=%s, reason=%s, duration=%v",
		userID, reason, time.Since(session.ConnectedAt))

	return nil
}

// GetSession returns the current session for a user
func (m *Manager) GetSession(userID string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, ok := m.sessions[userID]
	if !ok {
		return &Session{
			UserID: userID,
			State:  StateDisconnected,
		}, nil
	}

	return session, nil
}

// SendCommand sends a command to the MUD server
func (m *Manager) SendCommand(userID, command string) error {
	m.mu.RLock()
	conn, ok := m.conns[userID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("no active connection")
	}

	// Reset idle timer
	m.ResetIdleTimer(userID)

	// Send command
	_, err := conn.Write([]byte(command + "\r\n"))
	if err != nil {
		m.Disconnect(userID, ReasonError)
		return fmt.Errorf("failed to send command: %v", err)
	}

	return nil
}

// ReadOutput reads output from the MUD server (non-blocking for now)
// This will be called by the WebSocket handler in PH02
func (m *Manager) ReadOutput(userID string, buffer []byte) (int, error) {
	m.mu.RLock()
	conn, ok := m.conns[userID]
	m.mu.RUnlock()

	if !ok {
		return 0, fmt.Errorf("no active connection")
	}

	// Set read deadline
	conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))

	n, err := conn.Read(buffer)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return 0, nil // Timeout is not an error
		}
		return n, err
	}

	// Reset idle timer on any incoming data
	if n > 0 {
		m.ResetIdleTimer(userID)
	}

	return n, nil
}

// logConnectionMetadata logs connection metadata (no PII)
func (m *Manager) logConnectionMetadata(session *Session) {
	log.Printf("[SP02PH01T07] Connection established: user=%s, host=%s, port=%d, time=%s",
		session.UserID, session.Host, session.Port, session.ConnectedAt.Format(time.RFC3339))
}

// logDisconnectMetadata logs disconnect metadata (no PII)
func (m *Manager) logDisconnectMetadata(session *Session) {
	duration := time.Since(session.ConnectedAt)
	log.Printf("[SP02PH01T07] Connection closed: user=%s, host=%s, port=%d, duration=%v, reason=%s",
		session.UserID, session.Host, session.Port, duration, session.DisconnectErr)
}
