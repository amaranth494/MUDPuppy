package metrics

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics holds all application metrics (SP02PH04T03)
type Metrics struct {
	// Connection counters
	connectsTotal    atomic.Int64
	disconnectsTotal atomic.Int64

	// Disconnect reasons (keyed by reason string)
	disconnectReasonsMu sync.RWMutex
	disconnectReasons   map[string]*atomic.Int64

	// WebSocket message counters
	wsMessagesIn  atomic.Int64
	wsMessagesOut atomic.Int64

	// MUD bytes transferred
	mudBytesIn  atomic.Int64
	mudBytesOut atomic.Int64

	// Blocked port counter
	blockedPortTotal atomic.Int64

	// Protocol mismatch counter
	protocolMismatchTotal atomic.Int64

	// Active sessions (gauge)
	activeSessions atomic.Int64

	// Slow client disconnects
	slowClientDisconnects atomic.Int64

	// Rate limited disconnects
	rateLimitedDisconnects atomic.Int64

	// Last metric reset time
	lastReset time.Time

	// Rate limiting events
	rateLimitEvents atomic.Int64
}

// NewMetrics creates a new metrics instance
func NewMetrics() *Metrics {
	m := &Metrics{
		disconnectReasons: make(map[string]*atomic.Int64),
		lastReset:         time.Now(),
	}

	// Initialize disconnect reason counters
	reasons := []string{"user", "idle_timeout", "hard_cap", "remote_close", "error", "protocol_mismatch", "slow_client", "rate_limit"}
	for _, r := range reasons {
		var counter atomic.Int64
		m.disconnectReasons[r] = &counter
	}

	return m
}

// IncConnect increments the connection counter
func (m *Metrics) IncConnect() {
	m.connectsTotal.Add(1)
	m.activeSessions.Add(1)
}

// IncDisconnect increments the disconnect counter by reason
func (m *Metrics) IncDisconnect(reason string) {
	m.disconnectsTotal.Add(1)
	m.activeSessions.Add(-1)

	m.disconnectReasonsMu.RLock()
	if counter, ok := m.disconnectReasons[reason]; ok {
		counter.Add(1)
	}
	m.disconnectReasonsMu.RUnlock()
}

// IncWSMessagesIn increments WebSocket messages received
func (m *Metrics) IncWSMessagesIn() {
	m.wsMessagesIn.Add(1)
}

// IncWSMessagesOut increments WebSocket messages sent
func (m *Metrics) IncWSMessagesOut() {
	m.wsMessagesOut.Add(1)
}

// AddMudBytesIn adds to MUD bytes received
func (m *Metrics) AddMudBytesIn(bytes int64) {
	m.mudBytesIn.Add(bytes)
}

// AddMudBytesOut adds to MUD bytes sent
func (m *Metrics) AddMudBytesOut(bytes int64) {
	m.mudBytesOut.Add(bytes)
}

// IncBlockedPort increments the blocked port counter
func (m *Metrics) IncBlockedPort() {
	m.blockedPortTotal.Add(1)
}

// IncProtocolMismatch increments the protocol mismatch counter
func (m *Metrics) IncProtocolMismatch() {
	m.protocolMismatchTotal.Add(1)
}

// IncSlowClient increments slow client disconnect counter
func (m *Metrics) IncSlowClient() {
	m.slowClientDisconnects.Add(1)
}

// IncRateLimited increments rate limited disconnect counter
func (m *Metrics) IncRateLimited() {
	m.rateLimitEvents.Add(1)
}

// GetSnapshot returns a snapshot of all metrics in Prometheus format
func (m *Metrics) GetSnapshot() string {
	// Prometheus text format
	output := "# HELP mudpuppy_active_sessions Current number of active MUD sessions\n"
	output += "# TYPE mudpuppy_active_sessions gauge\n"
	output += fmt.Sprintf("mudpuppy_active_sessions %d\n\n", m.activeSessions.Load())

	output += "# HELP mudpuppy_connects_total Total number of connection attempts\n"
	output += "# TYPE mudpuppy_connects_total counter\n"
	output += fmt.Sprintf("mudpuppy_connects_total %d\n\n", m.connectsTotal.Load())

	output += "# HELP mudpuppy_disconnects_total Total number of disconnections\n"
	output += "# TYPE mudpuppy_disconnects_total counter\n"
	output += fmt.Sprintf("mudpuppy_disconnects_total %d\n\n", m.disconnectsTotal.Load())

	output += "# HELP mudpuppy_disconnect_reason_total Number of disconnections by reason\n"
	output += "# TYPE mudpuppy_disconnect_reason_total counter\n"
	m.disconnectReasonsMu.RLock()
	for reason, counter := range m.disconnectReasons {
		output += fmt.Sprintf("mudpuppy_disconnect_reason_total{reason=\"%s\"} %d\n", reason, counter.Load())
	}
	m.disconnectReasonsMu.RUnlock()
	output += "\n"

	output += "# HELP mudpuppy_ws_messages_in_total Total WebSocket messages received from clients\n"
	output += "# TYPE mudpuppy_ws_messages_in_total counter\n"
	output += fmt.Sprintf("mudpuppy_ws_messages_in_total %d\n\n", m.wsMessagesIn.Load())

	output += "# HELP mudpuppy_ws_messages_out_total Total WebSocket messages sent to clients\n"
	output += "# TYPE mudpuppy_ws_messages_out_total counter\n"
	output += fmt.Sprintf("mudpuppy_ws_messages_out_total %d\n\n", m.wsMessagesOut.Load())

	output += "# HELP mudpuppy_mud_bytes_in_total Total bytes received from MUD servers\n"
	output += "# TYPE mudpuppy_mud_bytes_in_total counter\n"
	output += fmt.Sprintf("mudpuppy_mud_bytes_in_total %d\n\n", m.mudBytesIn.Load())

	output += "# HELP mudpuppy_mud_bytes_out_total Total bytes sent to MUD servers\n"
	output += "# TYPE mudpuppy_mud_bytes_out_total counter\n"
	output += fmt.Sprintf("mudpuppy_mud_bytes_out_total %d\n\n", m.mudBytesOut.Load())

	output += "# HELP mudpuppy_blocked_port_total Total number of blocked port connection attempts\n"
	output += "# TYPE mudpuppy_blocked_port_total counter\n"
	output += fmt.Sprintf("mudpuppy_blocked_port_total %d\n\n", m.blockedPortTotal.Load())

	output += "# HELP mudpuppy_protocol_mismatch_total Total number of protocol mismatch disconnects\n"
	output += "# TYPE mudpuppy_protocol_mismatch_total counter\n"
	output += fmt.Sprintf("mudpuppy_protocol_mismatch_total %d\n\n", m.protocolMismatchTotal.Load())

	output += "# HELP mudpuppy_slow_client_disconnects_total Total number of slow client disconnects\n"
	output += "# TYPE mudpuppy_slow_client_disconnects_total counter\n"
	output += fmt.Sprintf("mudpuppy_slow_client_disconnects_total %d\n\n", m.slowClientDisconnects.Load())

	output += "# HELP mudpuppy_rate_limit_events_total Total number of rate limit events\n"
	output += "# TYPE mudpuppy_rate_limit_events_total counter\n"
	output += fmt.Sprintf("mudpuppy_rate_limit_events_total %d\n\n", m.rateLimitEvents.Load())

	output += "# HELP mudpuppy_uptime_seconds Seconds since metrics last reset\n"
	output += "# TYPE mudpuppy_uptime_seconds gauge\n"
	output += fmt.Sprintf("mudpuppy_uptime_seconds %.0f\n", time.Since(m.lastReset).Seconds())

	return output
}

// Global metrics instance
var globalMetrics *Metrics

// once is used to initialize global metrics once
var once sync.Once

// Init initializes the global metrics instance
func Init() {
	once.Do(func() {
		globalMetrics = NewMetrics()
	})
}

// Get returns the global metrics instance
func Get() *Metrics {
	if globalMetrics == nil {
		Init()
	}
	return globalMetrics
}
