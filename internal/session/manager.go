package session

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/amaranth494/MudPuppy/internal/metrics"
	"github.com/google/uuid"
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
	ReasonUser             = "user"
	ReasonIdle             = "idle_timeout"
	ReasonHardCap          = "hard_cap"
	ReasonRemote           = "remote_close"
	ReasonError            = "error"
	ReasonSlowClient       = "slow_client"       // SP02PH04T02
	ReasonRateLimit        = "rate_limit"        // SP02PH04T02
	ReasonProtocolMismatch = "protocol_mismatch" // SP02PH04T07
)

// Session represents an active MUD connection session
type Session struct {
	UserID string `json:"user_id"`
	Host   string `json:"host"`
	Port   int    `json:"port"`
	// ConnectionID names the saved connection profile this session was opened
	// for, or is empty for a quick connect. Phase 2 code review C1/C2: the
	// autopilot switch engages only for the profile that is actually connected
	// and resumes only onto that same profile.
	ConnectionID   string    `json:"connection_id,omitempty"`
	State          string    `json:"state"`
	ConnectedAt    time.Time `json:"connected_at,omitempty"`
	LastActivityAt time.Time `json:"last_activity_at,omitempty"`
	DisconnectErr  string    `json:"disconnect_reason,omitempty"`
}

// Manager handles MUD session management
type Manager struct {
	portWhitelist         map[int]bool
	portDenylist          map[int]bool
	portAllowlistOverride map[int]bool
	idleTimeoutMinutes    int
	hardCapHours          int

	mu        sync.RWMutex
	sessions  map[string]*Session // userID -> session
	conns     map[string]net.Conn
	cleanups  map[string]context.CancelFunc
	autopilot map[string]*AutopilotRecord // userID -> autopilot state
	// outputWindow holds each user's recent-game-text ring, fed continuously
	// by ReadOutput from the moment a connection is live (D-04). It lives
	// here, not on Session, and is never cleared on Connect or Disconnect —
	// the same "must outlive the session" rule already documented on
	// AutopilotRecord.ConnectionID above, because a WAITING-to-ON resume
	// must read a window that spans the drop (D-02, RESEARCH Pitfall 4). It
	// is cleared only by a process restart. Do not "tidy" this away.
	outputWindow map[string]*ringBuffer // userID -> recent game text for the AI

	// outputSignals holds each user's output-arrived wakeup channel
	// (04-03-01, D-06). Created lazily under m.mu exactly like
	// outputWindow's ring buffer above. Buffered size 1: appendOutputWindow
	// performs a non-blocking send so a burst of game output coalesces into
	// one pending wakeup rather than blocking the hot MUD read path or
	// piling up sends no one is reading. Never cleared on Disconnect, for
	// the same "must outlive the session" reason as outputWindow.
	outputSignals map[string]chan struct{}

	// transcripts holds each user's open session transcript tap (D-14),
	// keyed by userID exactly like autopilot and outputWindow above — never
	// a field on Session, for the same "must not be destroyed by
	// Disconnect's delete(m.sessions, userID) before it can be closed"
	// reason already documented on AutopilotRecord.ConnectionID (RESEARCH
	// Pitfall 4). Opened in Connect, closed in Disconnect.
	transcripts map[string]*transcriptSession
	// transcriptSink is the store-backed side of the tap (implemented by
	// *store.TranscriptStore via a thin adapter in cmd/server/main.go). A
	// nil sink disables transcription entirely; every transcript hook
	// becomes a no-op so the server runs with no database sink wired
	// without panicking.
	transcriptSink TranscriptSink

	// engageHook is the Phase 3 AI driver's entry point (its HandleEngage
	// method, supplied by cmd/server/main.go — this package declares the
	// hook type and never imports the driver package itself, keeping the
	// dependency direction one-way). Fired in exactly one place in this
	// file: after a WAITING-to-ON resume actually changes state (D-02). A
	// nil hook is a no-op.
	engageHook EngageHook

	// disengageHook is the AI driver's symmetric stop signal (04-03-01,
	// D-27/Pattern 2): fired the instant autopilot leaves ON for any
	// reason, so a loop asleep in its pacing wait — or waiting on a model
	// call that can take up to two minutes — is cancelled at once rather
	// than only being noticed after its next decision completes. Fired
	// from exactly two places in this file: the changed-true path of
	// DisengageAutopilot (a wheel-grab or #AUTO OFF) and the changed-true
	// path of parkAutopilotLocked (a disconnect). A nil hook is a silent
	// no-op, and it is safe to call even when no loop is running for
	// userID — the driver's StopLoop target treats a miss as a no-op.
	disengageHook DisengageHook
}

// EngageHook is called once per real engagement: a WAITING-to-ON resume in
// this file, and a fresh #AUTO ON engage in internal/session/handler.go's
// Autopilot handler. userID and connectionID are always valid ids already
// used elsewhere in this Manager. epoch is the stint the engagement began
// (AutopilotRecord.Epoch, read under the same lock that made the
// transition), so the driver knows which stint every decision it makes
// belongs to (code review CR-01 of Phase 4).
type EngageHook func(userID, connectionID string, epoch uint64)

// SetEngageHook wires the AI driver's entry point. A nil hook (the
// zero-value default) makes every fire site below a no-op.
func (m *Manager) SetEngageHook(h EngageHook) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.engageHook = h
}

// DisengageHook is called once per real disengagement of any cause — a
// wheel-grab, #AUTO OFF, or a disconnect parking the switch at waiting —
// so a driver loop asleep in its pacing wait can be cancelled immediately
// rather than after its next decision completes (04-03-01, D-27). userID
// is always a valid id already used elsewhere in this Manager. It is safe
// to call when no loop is running for userID (a silent no-op) and safe to
// call from inside the very goroutine it targets, since cancelling a
// context only marks it Done and never blocks.
type DisengageHook func(userID string)

// SetDisengageHook wires the AI driver's stop signal, mirroring
// SetEngageHook exactly. A nil hook (the zero-value default) makes every
// fire site below a no-op.
func (m *Manager) SetDisengageHook(h DisengageHook) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.disengageHook = h
}

// NewManager creates a new session manager
func NewManager(portWhitelist string, portDenylist string, portAllowlistOverride string, idleTimeoutMinutes, hardCapHours int) *Manager {
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

	// Parse port denylist
	denylist := make(map[int]bool)
	for _, p := range strings.Split(portDenylist, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		port, err := strconv.Atoi(p)
		if err != nil {
			log.Printf("Warning: Invalid port in denylist '%s', skipping", p)
			continue
		}
		denylist[port] = true
	}

	// Parse port allowlist override (if set, only these ports are allowed)
	allowlistOverride := make(map[int]bool)
	for _, p := range strings.Split(portAllowlistOverride, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		port, err := strconv.Atoi(p)
		if err != nil {
			log.Printf("Warning: Invalid port in allowlist override '%s', skipping", p)
			continue
		}
		allowlistOverride[port] = true
	}

	return &Manager{
		portWhitelist:         ports,
		portDenylist:          denylist,
		portAllowlistOverride: allowlistOverride,
		idleTimeoutMinutes:    idleTimeoutMinutes,
		hardCapHours:          hardCapHours,
		sessions:              make(map[string]*Session),
		conns:                 make(map[string]net.Conn),
		cleanups:              make(map[string]context.CancelFunc),
		autopilot:             make(map[string]*AutopilotRecord),
		outputWindow:          make(map[string]*ringBuffer),
		outputSignals:         make(map[string]chan struct{}),
		transcripts:           make(map[string]*transcriptSession),
	}
}

// ValidatePort checks if a port is allowed (SP02PH04T06 - Port Denylist)
// Priority: allowlist override > denylist > allow-all (except denylisted)
func (m *Manager) ValidatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1-65535")
	}

	// Check allowlist override first (if set, only these ports are allowed)
	if len(m.portAllowlistOverride) > 0 {
		if !m.portAllowlistOverride[port] {
			metrics.Get().IncBlockedPort()
			log.Printf("[SP02PH04T06] Port %d blocked: not in allowlist override", port)
			return fmt.Errorf("port %d is not allowed. Allowed ports: %v", port, m.getAllowlistOverridePorts())
		}
		return nil
	}

	// Check denylist (always blocked)
	if m.portDenylist[port] {
		metrics.Get().IncBlockedPort()
		log.Printf("[SP02PH04T06] Port %d blocked: in denylist", port)
		return fmt.Errorf("port %d is blocked for security reasons", port)
	}

	// Port is not in denylist and no override - allow it (SP02 spec: allow non-deny-listed ports)
	return nil
}

func (m *Manager) getAllowlistOverridePorts() []int {
	ports := make([]int, 0, len(m.portAllowlistOverride))
	for p := range m.portAllowlistOverride {
		ports = append(ports, p)
	}
	return ports
}

// Connect establishes a MUD connection for a user
func (m *Manager) Connect(ctx context.Context, userID, host string, port int, connectionID string) (*Session, error) {
	log.Printf("[SP02PH01] Connect called: user=%s, host=%s, port=%d, connection_id=%q", userID, host, port, connectionID)

	// Validate port first
	if err := m.ValidatePort(port); err != nil {
		log.Printf("[SP02PH01] Port validation failed: %v", err)
		return nil, err
	}

	// Validate host (private IP blocking)
	if err := ValidateHost(host); err != nil {
		log.Printf("[SP02PH01] Host validation failed: %v", err)
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
		UserID:       userID,
		Host:         host,
		Port:         port,
		ConnectionID: connectionID,
		State:        StateConnecting,
	}

	// Dial the MUD server
	address := net.JoinHostPort(host, strconv.Itoa(port))
	log.Printf("[SP02PH01] Dialing %s...", address)
	conn, err := net.DialTimeout("tcp", address, 5*time.Second) // Reduced timeout
	if err != nil {
		log.Printf("[SP02PH01] Dial failed: %v", err)
		session.State = StateError
		session.DisconnectErr = err.Error()
		m.sessions[userID] = session
		return session, fmt.Errorf("connection failed: %v", err)
	}
	log.Printf("[SP02PH01] Dial succeeded")

	// Skip telnet negotiation consumption - let the WebSocket handler read all data
	// This ensures we don't lose any welcome text

	// Update session state
	session.State = StateConnected
	session.ConnectedAt = time.Now()
	session.LastActivityAt = time.Now()

	// Store connection
	m.sessions[userID] = session
	m.conns[userID] = conn

	// Resume a waiting autopilot now that the dial has actually succeeded,
	// but only onto the profile it was parked on.
	m.resumeAutopilotLocked(userID, connectionID)

	// Open a session transcript for this connection, if it is a
	// saved-profile connection (D-14). A quick connect (connectionID=="")
	// is never transcribed. openTranscript takes m.mu itself only for the
	// brief map insert; the OpenGameSession database round trip runs with
	// no lock held so a slow connect never stalls every other user's
	// session (code review CR-01).
	m.mu.Unlock()
	m.openTranscript(userID, connectionID)
	m.mu.Lock()

	// Record metrics
	metrics.Get().IncConnect()

	// Start idle timer and hard cap timer (non-blocking)
	go m.startTimers(context.Background(), userID)

	// Log connection metadata (SP02PH01T07)
	m.logConnectionMetadata(session)

	log.Printf("[SP02PH01] Connection established for user=%s", userID)
	return session, nil
}

// startTimers starts idle timeout and hard cap timers
// Uses ticker-based approach to properly handle idle timer resets
func (m *Manager) startTimers(ctx context.Context, userID string) {
	ctx, cancel := context.WithCancel(ctx)

	m.mu.Lock()
	m.cleanups[userID] = cancel
	m.mu.Unlock()

	hardCapDuration := time.Duration(m.hardCapHours) * time.Hour

	// Hard cap is absolute; calculate when it should fire
	hardCapAt := time.Now().Add(hardCapDuration)

	go func() {
		ticker := time.NewTicker(1 * time.Minute) // Check every minute
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.mu.RLock()
				_, ok := m.sessions[userID]
				m.mu.RUnlock()

				if !ok {
					return // Session no longer exists
				}

				now := time.Now()

				// Check hard cap first (absolute limit)
				if now.After(hardCapAt) {
					log.Printf("[SP02PH01T06] Hard session cap reached for user %s", userID)
					m.Disconnect(userID, ReasonHardCap)
					return
				}

				// Idle timeout disabled - removed per user request
			}
		}
	}()
}

// ResetIdleTimer resets the idle timer for a session
// This should be called when there's any activity (inbound or outbound)
func (m *Manager) ResetIdleTimer(userID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, ok := m.sessions[userID]; ok {
		session.LastActivityAt = time.Now()
		log.Printf("[SP02PH01T05] Idle timer reset for user %s", userID)
	}
}

// ResetIdleTimerOnInbound resets the idle timer when MUD sends data to client
// Call this from the WebSocket relay when forwarding MUD→client data
func (m *Manager) ResetIdleTimerOnInbound(userID string) {
	m.ResetIdleTimer(userID)
}

// ResetIdleTimerOnOutbound resets the idle timer when client sends command to MUD
// Call this from the WebSocket relay when forwarding client→MUD data
func (m *Manager) ResetIdleTimerOnOutbound(userID string) {
	m.ResetIdleTimer(userID)
}

// Disconnect terminates a user's MUD connection
func (m *Manager) Disconnect(userID, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[userID]
	if !ok {
		// Session already removed - treat as already disconnected (success)
		return nil
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

	// Park an engaged autopilot at waiting before the session is deleted
	// below — the autopilot map is separate from m.sessions and survives
	// this deletion (RESEARCH Pitfall 1).
	m.parkAutopilotLocked(userID)

	// Close the session transcript, if one is open, before the session
	// itself is removed below (D-14). closeTranscript resolves and removes
	// the transcript from the map under m.mu, then runs the blocking
	// drain-wait and CloseGameSession call with no lock held, so a slow
	// database round trip on disconnect never stalls every other user's
	// session (code review CR-01). Releasing/re-acquiring m.mu here is
	// safe: nothing between this point and Disconnect's return depends on
	// m.transcripts, and closeTranscript's own locked resolve-then-remove
	// step means no concurrent caller can ever observe a half-torn-down
	// transcript.
	m.mu.Unlock()
	m.closeTranscript(userID)
	m.mu.Lock()

	// Remove session from map - this ensures clean state for reconnection
	// and prevents orphaned sessions
	delete(m.sessions, userID)

	// Record metrics
	metrics.Get().IncDisconnect(reason)

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

// logAutopilotTransition emits the one [AI-PLAYER] autopilot log line every
// autopilot state call produces. This format is a cross-plan contract:
// plans 02-02 and 02-03 append causes to this same line shape, never a
// different one.
func (m *Manager) logAutopilotTransition(userID, connectionID string, old, newState AutopilotState, cause string) {
	log.Printf("[AI-PLAYER] autopilot user_id=%s connection_id=%s old=%s new=%s cause=%s",
		userID, connectionID, old, newState, cause)
}

// AutopilotConnectionIDFor returns the saved-connection id the user's
// autopilot record was engaged on, or "" when the switch is off. The
// status poll carries it so the browser can still aim #AUTO OFF at the
// parked profile after a page refresh (code review C3).
func (m *Manager) AutopilotConnectionIDFor(userID string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if rec, ok := m.autopilot[userID]; ok && rec.State != AutopilotOff {
		return rec.ConnectionID
	}
	return ""
}

// AutopilotEpochFor returns the current autopilot state for a user
// together with the stint epoch it belongs to, read under one lock so the
// pair is consistent (code review CR-01 of Phase 4). A user who has never
// engaged reads as off with epoch 0. The driver compares the epoch it
// captured when a decision started against this one before every step that
// costs money or reaches the game: a different epoch means the decision's
// stint is over, whatever the switch reads now.
func (m *Manager) AutopilotEpochFor(userID string) (AutopilotState, uint64) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rec, ok := m.autopilot[userID]
	if !ok {
		return AutopilotOff, 0
	}
	return rec.State, rec.Epoch
}

// AutopilotStateFor returns the current autopilot state for a user. A user
// who has never engaged reads as off, which is also the state a server
// restart lands on (D-06).
func (m *Manager) AutopilotStateFor(userID string) AutopilotState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rec, ok := m.autopilot[userID]
	if !ok {
		return AutopilotOff
	}
	return rec.State
}

// appendOutputWindow feeds p (already telnet- and ANSI-stripped) into the
// user's recent-output ring, lazily creating it on first use. A no-op for
// an empty p. Called from ReadOutput for every byte the game ever sends,
// so the window is kept continuously regardless of autopilot state (D-04's
// "including text from before the switch was flipped").
func (m *Manager) appendOutputWindow(userID string, p []byte) {
	if len(p) == 0 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	ring, ok := m.outputWindow[userID]
	if !ok {
		ring = newRingBuffer()
		m.outputWindow[userID] = ring
	}
	ring.append(p)

	// Wake a driver loop asleep in its pacing wait (04-03-01, D-06). A
	// non-blocking send with a default case: a burst of game output
	// coalesces into one pending wakeup rather than blocking this hot MUD
	// read path or piling up sends no one is reading.
	sig, ok := m.outputSignals[userID]
	if !ok {
		sig = make(chan struct{}, 1)
		m.outputSignals[userID] = sig
	}
	select {
	case sig <- struct{}{}:
	default:
	}
}

// OutputSignal returns userID's output-arrived wakeup channel, creating it
// lazily on first ask under m.mu, exactly as outputWindow's ring buffer is
// (04-03-01, D-06). A receive on the returned channel means new game bytes
// landed in appendOutputWindow since the last receive; the channel is never
// closed. A user with no output yet has an empty, non-nil channel.
func (m *Manager) OutputSignal(userID string) <-chan struct{} {
	m.mu.Lock()
	defer m.mu.Unlock()

	sig, ok := m.outputSignals[userID]
	if !ok {
		sig = make(chan struct{}, 1)
		m.outputSignals[userID] = sig
	}
	return sig
}

// RecentOutputSnapshot returns the current contents of the user's
// recent-output window, oldest-first, or "" when there is none. This is
// the only method the driver (plan 03-08) calls to get the AI's first
// look at the game. Same RLock/defer shape as AutopilotStateFor.
// RecentOutputSnapshot returns userID's Immediate Context reaction window
// (D-09, .specify/specs/ai-memory-model-v1.md §1): game text from roughly
// the last windowMaxAge, or the most recent windowMinRetainedBytes when a
// quiet spell would otherwise leave the AI with nothing — never the whole
// 8 KB ring regardless of age. The driver's per-iteration decision reads
// through this single call site, so every decision from now on is made
// against what just happened, and a decision taken after a long-quiet
// game still sees the most recent screenful rather than an empty window.
func (m *Manager) RecentOutputSnapshot(userID string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ring, ok := m.outputWindow[userID]
	if !ok {
		return ""
	}
	return ring.snapshotRecent(windowMaxAge, windowMinRetainedBytes)
}

// SetTranscriptSink wires the store-backed side of the session transcript
// tap (D-14). Called once at startup by cmd/server/main.go. A nil sink
// disables transcription entirely: every hook below becomes a no-op, so
// the server runs with no database sink wired without panicking.
func (m *Manager) SetTranscriptSink(sink TranscriptSink) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.transcriptSink = sink
}

// CurrentGameSessionID returns the game session id of userID's currently
// open transcript, so a decision row can be tied to the session it
// happened in (plan 03-08). Returns false when there is no open
// transcript — a quick connect with no profile, or no connection at all.
func (m *Manager) CurrentGameSessionID(userID string) (uuid.UUID, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	t, ok := m.transcripts[userID]
	if !ok {
		return uuid.Nil, false
	}
	return t.gameSessionID, true
}

// openTranscript opens a session transcript for userID's new connection.
// The caller must NOT hold m.mu (code review CR-01): this method takes the
// lock itself, only for the brief moments that read m.transcriptSink and
// install the new transcriptSession into m.transcripts. The
// OpenGameSession database round trip runs with no lock held, so a slow
// connect never stalls every other user's session. It is a no-op when
// transcription is disabled (no sink configured), when connectionID is
// empty — a quick connect with no profile is never transcribed (D-14) —
// or when either id fails to parse as a UUID.
func (m *Manager) openTranscript(userID, connectionID string) {
	m.mu.RLock()
	sink := m.transcriptSink
	m.mu.RUnlock()

	if sink == nil || connectionID == "" {
		return
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return
	}
	connUUID, err := uuid.Parse(connectionID)
	if err != nil {
		return
	}

	gameSessionID, err := sink.OpenGameSession(userUUID, connUUID) // no lock held
	if err != nil {
		log.Printf("[AI-PLAYER] transcript user_id=%s connection_id=%s event=open-error error=%v",
			userID, connectionID, err)
		return
	}

	t := &transcriptSession{
		gameSessionID: gameSessionID,
		userID:        userID,
		connectionID:  connectionID,
		lines:         make(chan TranscriptLine, transcriptChannelCapacity),
		stop:          make(chan struct{}),
		done:          make(chan struct{}),
	}
	go t.writeLoop(sink)

	m.mu.Lock()
	m.transcripts[userID] = t
	m.mu.Unlock()

	log.Printf("[AI-PLAYER] transcript user_id=%s connection_id=%s game_session_id=%s event=open",
		userID, connectionID, gameSessionID)
}

// closeTranscript closes and removes userID's open transcript, if any. The
// caller must NOT hold m.mu (code review CR-01): this method resolves the
// transcript and removes it from m.transcripts under a single brief
// m.mu.Lock so a concurrent caller can never observe a half-torn-down
// transcript, then runs the blocking drain-wait and store close call with
// no lock held, so a slow disconnect never stalls every other user's
// session. A no-op when transcription is disabled or none was ever opened
// (a quick connect never had one). The blocking half mirrors the
// deliberate open/close lifecycle (D-14) rather than the never-block
// guarantee that applies to enqueue on the hot MUD read/write path
// (T-3-26).
func (m *Manager) closeTranscript(userID string) {
	m.mu.Lock()
	t, ok := m.transcripts[userID]
	if ok {
		delete(m.transcripts, userID)
	}
	sink := m.transcriptSink
	m.mu.Unlock()

	if !ok {
		return
	}
	t.close(sink)
}

// enqueueTranscriptLine enqueues one transcript line for userID's open
// transcript, if any, taking m.mu itself. Used by callers (SendCommandAs,
// feedTranscriptOutput) that do not already hold m.mu.
func (m *Manager) enqueueTranscriptLine(userID, source, text string) {
	m.mu.RLock()
	t, ok := m.transcripts[userID]
	m.mu.RUnlock()
	if ok {
		t.enqueue(source, text)
	}
}

// enqueueTranscriptLineLocked is the same as enqueueTranscriptLine except
// the caller must already hold m.mu (any mode) — used from inside
// EngageAutopilot, DisengageAutopilot, parkAutopilotLocked and
// resumeAutopilotLocked, which all hold the lock already and would
// deadlock calling RLock again (sync.RWMutex is not reentrant).
func (m *Manager) enqueueTranscriptLineLocked(userID, source, text string) {
	if t, ok := m.transcripts[userID]; ok {
		t.enqueue(source, text)
	}
}

// feedTranscriptOutput hands telnet-stripped game output to userID's open
// transcript, if any, for line-splitting and ANSI stripping (D-14). A
// no-op when no transcript is open.
func (m *Manager) feedTranscriptOutput(userID string, p []byte) {
	m.mu.RLock()
	t, ok := m.transcripts[userID]
	m.mu.RUnlock()
	if ok {
		t.feedGameOutput(p)
	}
}

// EngageAutopilot handles #AUTO ON. It reads m.sessions directly under the
// held lock rather than calling GetSession, which takes RLock itself and
// would deadlock against the Lock held here (sync.RWMutex is not
// reentrant). Engaging without a connected session is refused with
// ErrNoConnectedSession and the switch stays off (D-03). A repeated engage
// while already on leaves the stored record completely untouched — no
// rewritten ConnectionID, no cleared WaitingSince, no new allocation
// (D-04, threat T-2-05).
func (m *Manager) EngageAutopilot(userID, connectionID string) (AutopilotState, bool, error) {
	state, changed, _, err := m.EngageAutopilotEpoch(userID, connectionID)
	return state, changed, err
}

// EngageAutopilotEpoch is EngageAutopilot that also reports the stint epoch
// of the record after the call (code review CR-01 of Phase 4): the new
// stint's epoch when the engage changed the state, the unchanged current
// one otherwise. The #AUTO ON handler passes it to the engage hook, so the
// epoch the driver is told about is the one created by this very
// transition, not one read afterwards under a second lock.
func (m *Manager) EngageAutopilotEpoch(userID, connectionID string) (AutopilotState, bool, uint64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cur := AutopilotOff
	curConnID := ""
	var curEpoch uint64
	if rec, ok := m.autopilot[userID]; ok {
		cur = rec.State
		curConnID = rec.ConnectionID
		curEpoch = rec.Epoch
	}

	session, ok := m.sessions[userID]
	if !ok || session.State != StateConnected {
		m.logAutopilotTransition(userID, connectionID, cur, cur, "refused-no-session")
		return cur, false, curEpoch, ErrNoConnectedSession
	}

	// Code review C2: the gate was resolved for connectionID, so the live
	// session must be the one opened for that same profile. Otherwise a
	// profile that accepted the policy could lend its acceptance to another.
	if connectionID == "" || session.ConnectionID != connectionID {
		m.logAutopilotTransition(userID, connectionID, cur, cur, "refused-wrong-connection")
		return cur, false, curEpoch, ErrWrongConnection
	}

	newState, changed := Engage(cur)
	if !changed {
		m.logAutopilotTransition(userID, curConnID, cur, newState, "already-on")
		return newState, false, curEpoch, nil
	}

	// The record is replaced, but the epoch carries over and goes up by
	// one: it must never repeat for a user while the process lives.
	m.autopilot[userID] = &AutopilotRecord{
		State:        newState,
		ConnectionID: connectionID,
		Epoch:        curEpoch + 1,
	}
	m.logAutopilotTransition(userID, connectionID, cur, newState, "engage")
	m.enqueueTranscriptLineLocked(userID, "marker", "[AI-ASSIST engaged]")
	return newState, true, curEpoch + 1, nil
}

// DisengageAutopilot handles #AUTO OFF and the wheel-grab. Permitted cause
// values are "disengage", "wheel-grab" and "already-off"; when the
// transition does not change the state, the log line always uses
// "already-off" regardless of what the caller passed.
func (m *Manager) DisengageAutopilot(userID, cause string) (AutopilotState, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cur := AutopilotOff
	curConnID := ""
	if rec, ok := m.autopilot[userID]; ok {
		cur = rec.State
		curConnID = rec.ConnectionID
	}

	newState, changed := Disengage(cur)
	if !changed {
		m.logAutopilotTransition(userID, curConnID, cur, newState, "already-off")
		return newState, false
	}

	if rec, ok := m.autopilot[userID]; ok {
		rec.State = newState
		rec.WaitingSince = nil
	} else {
		m.autopilot[userID] = &AutopilotRecord{State: newState}
	}
	m.logAutopilotTransition(userID, curConnID, cur, newState, cause)
	m.enqueueTranscriptLineLocked(userID, "marker", fmt.Sprintf("[AI-ASSIST disengaged: %s]", cause))

	// 04-03-01/D-27: fire the symmetric stop signal so a driver loop
	// asleep in its pacing wait is cancelled at once. Started with `go`,
	// matching resumeAutopilotLocked's existing discipline below — the
	// caller holds m.mu here, so a synchronous call could deadlock the
	// moment the driver reads anything back from this Manager.
	if m.disengageHook != nil {
		go m.disengageHook(userID)
	}
	return newState, true
}

// parkAutopilotLocked applies the on->waiting transition for a disconnect.
// The caller must already hold m.mu. It is a no-op with no log line when
// there is no record or the transition does not change the state, so
// ordinary play with autopilot off adds nothing to the log (D-12).
func (m *Manager) parkAutopilotLocked(userID string) {
	rec, ok := m.autopilot[userID]
	if !ok {
		return
	}

	newState, changed := EnterWaiting(rec.State)
	if !changed {
		return
	}

	old := rec.State
	rec.State = newState
	now := time.Now()
	rec.WaitingSince = &now
	m.logAutopilotTransition(userID, rec.ConnectionID, old, newState, "disconnect")
	m.enqueueTranscriptLineLocked(userID, "marker", "[AI-ASSIST waiting]")

	// 04-03-01/D-27: same stop signal as DisengageAutopilot's changed-true
	// path — a disconnect must cancel a sleeping loop immediately too
	// (D-19), not only be noticed the next time it wakes.
	if m.disengageHook != nil {
		go m.disengageHook(userID)
	}
}

// resumeAutopilotLocked applies the waiting->on transition for a connect.
// The caller must already hold m.mu. Same no-op rule as
// parkAutopilotLocked. A parked switch resumes only when the new session
// is for the same saved connection it was parked on; a different profile,
// or a quick connect with no profile, lands the switch on off instead, so
// the Phase 1 gate of the profile that accepted the policy is never
// borrowed by another (code review C1).
func (m *Manager) resumeAutopilotLocked(userID, connectionID string) {
	rec, ok := m.autopilot[userID]
	if !ok || rec.State != AutopilotWaiting {
		return
	}

	old := rec.State
	if connectionID == "" || connectionID != rec.ConnectionID {
		rec.State = AutopilotOff
		rec.WaitingSince = nil
		m.logAutopilotTransition(userID, rec.ConnectionID, old, AutopilotOff, "connection-changed")
		m.enqueueTranscriptLineLocked(userID, "marker", "[AI-ASSIST disengaged: connection-changed]")
		return
	}

	newState, changed := Resume(rec.State)
	if !changed {
		return
	}
	rec.State = newState
	rec.WaitingSince = nil
	// A resume is an engagement (D-02): it begins a new stint, so the epoch
	// goes up exactly as it does on #AUTO ON (code review CR-01 of Phase 4).
	rec.Epoch++
	m.logAutopilotTransition(userID, rec.ConnectionID, old, newState, "resume")
	m.enqueueTranscriptLineLocked(userID, "marker", "[AI-ASSIST resumed]")

	// D-02: a resume counts as an engagement and fires one fresh decision
	// on the post-reconnect text. Started with `go` because m.mu is held
	// here — calling the hook synchronously would deadlock the moment the
	// driver reads anything back from this Manager (RecentOutputSnapshot,
	// SendCommandAs, DisengageAutopilot, CurrentGameSessionID all take
	// m.mu themselves). `go` returns immediately; the goroutine acquires
	// the lock on its own schedule, after this function returns and
	// releases it. Never fired from EngageAutopilot — the HTTP handler
	// owns that trigger, so a single code path fires each real engage.
	if m.engageHook != nil {
		go m.engageHook(userID, rec.ConnectionID, rec.Epoch)
	}
}

// ErrAutopilotNotOn is what SendCommandAs and SendAICommand return for an
// AI command when autopilot is not On at the moment of the write, or (for
// SendAICommand) when it is On but for a later stint than the one the
// command was decided in.
var ErrAutopilotNotOn = errors.New("autopilot is not on; AI command not sent")

// SendCommand sends a command to the MUD server, tagged in the session
// transcript as human-typed. This is the browser's only send path
// (websocket.go:653, covering typed input, aliases, triggers and timers
// alike) and its signature is unchanged so that call site needs no edit;
// it delegates to SendCommandAs with source "human".
func (m *Manager) SendCommand(userID, command string) error {
	return m.SendCommandAs(userID, command, "human")
}

// SendCommandAs sends a command to the MUD server and tags the transcript
// line with source, which must be "human" or "ai" (plan 03-08's driver is
// the only caller that passes "ai"). The transcript line is enqueued only
// after the write to the socket succeeds, so a failed send is never
// recorded as sent.
//
// An "ai" command is written only while autopilot is On, and the check and
// the write happen under one read lock: DisengageAutopilot takes the write
// lock, so a decision that was already in flight when the owner took the
// wheel (or typed #AUTO OFF) can never reach the game after the switch has
// gone off. It returns ErrAutopilotNotOn instead. The staging walkthrough
// of 2026-09-17 found exactly that: a command sent one second after off.
func (m *Manager) SendCommandAs(userID, command, source string) error {
	return m.sendCommand(userID, command, source, false, 0)
}

// SendAICommand is the driver's send path (code review CR-01 of Phase 4).
// It is SendCommandAs with source "ai" plus one more condition checked under
// the same lock: the autopilot record's Epoch must still equal epoch, the
// stint the command was decided in. "The switch reads On" is not enough: an
// owner who takes the wheel for one command and types #AUTO ON again before
// a slow model call returns has the switch On again, in a new stint, and the
// old decision -- made from text that predates the wheel-grab -- must not be
// sent into it. A mismatch returns ErrAutopilotNotOn, exactly as a switch
// that is off does; the driver treats both as "this decision outlived its
// stint" and drops it quietly.
func (m *Manager) SendAICommand(userID, command string, epoch uint64) error {
	return m.sendCommand(userID, command, "ai", true, epoch)
}

// sendCommand is the one body behind SendCommandAs and SendAICommand.
// checkEpoch is true only for SendAICommand.
func (m *Manager) sendCommand(userID, command, source string, checkEpoch bool, epoch uint64) error {
	m.mu.RLock()
	conn, ok := m.conns[userID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("no active connection")
	}

	// Reset idle timer
	m.ResetIdleTimer(userID)

	// Strip trailing newlines from command to avoid double-newline issue
	// Frontend may already include \n, and we add \r\n, so we need to prevent \n\r\n
	command = strings.TrimRight(command, "\n")

	// Send command with CRLF (standard for MUD servers)
	var err error
	if source == "ai" {
		m.mu.RLock()
		rec, recOK := m.autopilot[userID]
		if !recOK || rec.State != AutopilotOn || (checkEpoch && rec.Epoch != epoch) {
			m.mu.RUnlock()
			return ErrAutopilotNotOn
		}
		_, err = conn.Write([]byte(command + "\r\n"))
		m.mu.RUnlock()
	} else {
		_, err = conn.Write([]byte(command + "\r\n"))
	}
	if err != nil {
		m.Disconnect(userID, ReasonError)
		return fmt.Errorf("failed to send command: %v", err)
	}

	m.enqueueTranscriptLine(userID, source, command)

	return nil
}

// SendCredentials sends username and password to the MUD server for auto-login
// This sends the credentials followed by a newline, suitable for login prompts
func (m *Manager) SendCredentials(userID, username, password string) error {
	m.mu.RLock()
	conn, ok := m.conns[userID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("no active connection")
	}

	// Reset idle timer
	m.ResetIdleTimer(userID)

	// Send username then password (each followed by newline)
	// Common login flow: username -> enter -> password -> enter
	if username != "" {
		_, err := conn.Write([]byte(username + "\r\n"))
		if err != nil {
			log.Printf("[SP03PH05T08] Failed to send username: %v", err)
			return fmt.Errorf("failed to send username: %v", err)
		}
	}

	if password != "" {
		_, err := conn.Write([]byte(password + "\r\n"))
		if err != nil {
			log.Printf("[SP03PH05T08] Failed to send password: %v", err)
			return fmt.Errorf("failed to send password: %v", err)
		}
	}

	log.Printf("[SP03PH05T08] Credentials sent for user=%s", userID)
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

	// Set read deadline - shorter for faster response
	conn.SetReadDeadline(time.Now().Add(50 * time.Millisecond))

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
		// Feed the AI's recent-output window and the session transcript
		// (D-14) from the bytes actually read. ReadOutput does not hold
		// m.mu here (the RLock above was already released), so
		// appendOutputWindow's and feedTranscriptOutput's own locking are
		// safe to call. A new slice is built for the window/transcript;
		// buffer itself is not mutated, since the same slice is what
		// relayMUDToClient forwards to the browser and the browser must
		// keep receiving full ANSI.
		telnetStripped := stripTelnetIAC(buffer[:n])
		m.appendOutputWindow(userID, stripANSI(telnetStripped))
		m.feedTranscriptOutput(userID, telnetStripped)
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

// Protocol plausibility check constants (SP02PH04T07)
const (
	protocolCheckBytes   = 512             // First N bytes to check
	protocolCheckTimeout = 2 * time.Second // Timeout for reading initial data
)

// detectProtocolMismatch checks if the initial data from MUD server looks like non-MUD protocol
// Returns true if protocol mismatch detected, false if it looks like valid MUD/telnet
func (m *Manager) detectProtocolMismatch(conn net.Conn) (bool, string) {
	// Set read deadline
	conn.SetReadDeadline(time.Now().Add(protocolCheckTimeout))

	buffer := make([]byte, protocolCheckBytes)
	n, err := conn.Read(buffer)

	// Reset deadline for normal operation
	conn.SetReadDeadline(time.Time{})

	if err != nil || n == 0 {
		// No data or error - can't determine, allow it
		return false, ""
	}

	data := buffer[:n]

	// Check for TLS/SSL handshake (starts with 0x16 = TLS Handshake)
	if len(data) >= 1 && data[0] == 0x16 {
		log.Printf("[SP02PH04T07] Protocol mismatch detected: TLS handshake")
		metrics.Get().IncProtocolMismatch()
		return true, "TLS/SSL not supported"
	}

	// Check for HTTP response (starts with "HTTP/")
	if len(data) >= 5 && string(data[:5]) == "HTTP/" {
		log.Printf("[SP02PH04T07] Protocol mismatch detected: HTTP response")
		metrics.Get().IncProtocolMismatch()
		return true, "HTTP not supported"
	}

	// Check for SSH (starts with "SSH-")
	if len(data) >= 4 && string(data[:4]) == "SSH-" {
		log.Printf("[SP02PH04T07] Protocol mismatch detected: SSH")
		metrics.Get().IncProtocolMismatch()
		return true, "SSH not supported"
	}

	// Check for FTP (starts with "220-" or "220 ")
	if len(data) >= 4 && (string(data[:4]) == "220-" || string(data[:4]) == "220 ") {
		log.Printf("[SP02PH04T07] Protocol mismatch detected: FTP")
		metrics.Get().IncProtocolMismatch()
		return true, "FTP not supported"
	}

	// Check for SMTP (starts with "220-" from SMTP but different from FTP)
	if len(data) >= 4 && string(data[:4]) == "220-" {
		// Could be SMTP - check for ESMTP markers
		if containsSMTPMarker(data) {
			log.Printf("[SP02PH04T07] Protocol mismatch detected: SMTP")
			metrics.Get().IncProtocolMismatch()
			return true, "SMTP not supported"
		}
	}

	// Check for high concentration of null bytes (binary protocol)
	nullCount := 0
	for _, b := range data {
		if b == 0 {
			nullCount++
		}
	}
	// If more than 10% are null bytes, likely binary
	if float64(nullCount)/float64(len(data)) > 0.1 {
		log.Printf("[SP02PH04T07] Protocol mismatch detected: binary data (null bytes)")
		metrics.Get().IncProtocolMismatch()
		return true, "Binary protocol not supported"
	}

	// Default: looks like valid text/telnet/MUD
	return false, ""
}

// containsSMTPMarker checks if data contains SMTP-specific markers
func containsSMTPMarker(data []byte) bool {
	dataStr := string(data)
	smtpMarkers := []string{"ESMTP", "SMTP", "mail", "EHLO"}
	for _, marker := range smtpMarkers {
		if strings.Contains(dataStr, marker) {
			return true
		}
	}
	return false
}

// CheckAndDisconnectProtocolMismatch checks for protocol mismatch and disconnects if detected
// Returns true if disconnected due to protocol mismatch
func (m *Manager) CheckAndDisconnectProtocolMismatch(userID string) bool {
	m.mu.RLock()
	conn, ok := m.conns[userID]
	m.mu.RUnlock()

	if !ok {
		return false
	}

	isMismatch, message := m.detectProtocolMismatch(conn)
	if isMismatch {
		m.Disconnect(userID, ReasonError)
		log.Printf("[SP02PH04T07] Protocol mismatch disconnect: user=%s, reason=%s", userID, message)
		return true
	}

	return false
}
