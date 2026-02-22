package session

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/amaranth494/MudPuppy/internal/config"
	"github.com/gorilla/websocket"
)

// WebSocket message types
const (
	MsgTypeConnect    = "connect"
	MsgTypeDisconnect = "disconnect"
	MsgTypeData       = "data"
	MsgTypeError      = "error"
	MsgTypeStatus     = "status"
)

// WebSocket message structure
type WSMessage struct {
	Type   string `json:"type"`
	Host   string `json:"host,omitempty"`
	Port   int    `json:"port,omitempty"`
	Data   string `json:"data,omitempty"`
	Error  string `json:"error,omitempty"`
	Status string `json:"status,omitempty"`
}

// RateLimiter implements a simple token bucket rate limiter
type RateLimiter struct {
	mu         sync.Mutex
	tokens     int
	maxTokens  int
	refillRate time.Duration
	lastRefill time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxTokens int, refillRate time.Duration) *RateLimiter {
	return &RateLimiter{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow checks if a request is allowed under rate limit
func (r *RateLimiter) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(r.lastRefill)
	if elapsed >= r.refillRate {
		// Refill tokens
		r.tokens = r.maxTokens
		r.lastRefill = now
	}

	if r.tokens > 0 {
		r.tokens--
		return true
	}
	return false
}

// WebSocketHandler handles WebSocket connections for MUD session streaming
type WebSocketHandler struct {
	manager        *Manager
	config         *config.Config
	upgrader       websocket.Upgrader
	rateLimiters   map[string]*RateLimiter
	rateLimitersMu sync.RWMutex
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(manager *Manager, cfg *config.Config) *WebSocketHandler {
	return &WebSocketHandler{
		manager: manager,
		config:  cfg,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			// Allow all origins in development (CORS)
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		rateLimiters: make(map[string]*RateLimiter),
	}
}

// getRateLimiter gets or creates a rate limiter for a user
func (h *WebSocketHandler) getRateLimiter(userID string) *RateLimiter {
	h.rateLimitersMu.RLock()
	rl, ok := h.rateLimiters[userID]
	h.rateLimitersMu.RUnlock()

	if !ok {
		// Create new rate limiter: max tokens = command rate limit per second
		// refill rate = 1 second
		rl = NewRateLimiter(h.config.CommandRateLimitPerSec, 1*time.Second)

		h.rateLimitersMu.Lock()
		h.rateLimiters[userID] = rl
		h.rateLimitersMu.Unlock()
	}

	return rl
}

// removeRateLimiter removes a user's rate limiter
func (h *WebSocketHandler) removeRateLimiter(userID string) {
	h.rateLimitersMu.Lock()
	delete(h.rateLimiters, userID)
	h.rateLimitersMu.Unlock()
}

// HandleWebSocket handles WebSocket connections at /api/v1/session/stream
func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by session middleware)
	userID := r.Context().Value("user_id")
	if userID == nil {
		log.Printf("[SP02PH02] WebSocket connection rejected: no user_id in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userIDStr := userID.(string)

	log.Printf("[SP02PH02] WebSocket connection from user: %s", userIDStr)

	// Upgrade HTTP to WebSocket
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[SP02PH02] WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	// Set read limit to prevent memory exhaustion (SP02 hardening)
	conn.SetReadLimit(65536) // 64KB max message size

	// Write deadline for all outbound writes
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

	// Set up ping/pong handler for keepalive (SP02 hardening)
	pongChan := make(chan string, 1)
	conn.SetPongHandler(func(appData string) error {
		pongChan <- appData
		return nil
	})

	// Start ping ticker
	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	// Create context for this connection
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Channel for MUD output to send to client
	mudToClient := make(chan []byte, 100)
	// Channel for client commands to send to MUD
	clientToMUD := make(chan string, 10)
	// Channel for connection status
	statusChan := make(chan string, 2)

	// Track if we're connected to a MUD
	connected := false

	// Start goroutine to read from MUD and forward to client
	go h.readMUDOutput(ctx, userIDStr, mudToClient, statusChan)

	// Start goroutine to handle client commands and forward to MUD
	go h.handleClientCommands(ctx, userIDStr, clientToMUD, statusChan)

	// Main WebSocket message loop
	for {
		// Check if we got a disconnect status or ping ticker
		select {
		case status := <-statusChan:
			if status == "disconnected" {
				log.Printf("[SP02PH02] MUD connection closed for user %s", userIDStr)
				connected = false
				// Send disconnect message to client
				err := conn.WriteJSON(WSMessage{
					Type:   MsgTypeDisconnect,
					Status: "disconnected",
				})
				if err != nil {
					log.Printf("[SP02PH02] Error sending disconnect to client: %v", err)
					return
				}
			}
		case <-pingTicker.C:
			// Send ping to keep connection alive (SP02 hardening)
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("[SP02PH02] Ping failed: %v", err)
				return
			}
		case <-ctx.Done():
			// Context cancelled - clean up and exit (no goroutine leak)
			if connected {
				h.manager.Disconnect(userIDStr, ReasonRemote)
			}
			h.removeRateLimiter(userIDStr)
			return
		}

		// Set read deadline
		conn.SetReadDeadline(time.Now().Add(30 * time.Second))

		// Read message from client
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[SP02PH02] WebSocket read error: %v", err)
			}
			// Clean up MUD connection on close
			if connected {
				h.manager.Disconnect(userIDStr, ReasonRemote)
			}
			h.removeRateLimiter(userIDStr)
			return
		}

		if msgType != websocket.TextMessage {
			continue
		}

		// Parse the message
		var wsMsg WSMessage
		if err := json.Unmarshal(msg, &wsMsg); err != nil {
			log.Printf("[SP02PH02] Failed to parse WebSocket message: %v", err)
			h.sendError(conn, "Invalid message format")
			continue
		}

		switch wsMsg.Type {
		case MsgTypeConnect:
			var err error
			var session *Session

			// Check if session already exists (may have been created via REST API)
			session, err = h.manager.GetSession(userIDStr)
			if err == nil && session.State == StateConnected {
				// Session already exists from REST API - just use it
				connected = true
				log.Printf("[SP02PH02] Using existing session for user %s (from REST API)", userIDStr)
			} else {
				// No existing session - this is a WebSocket-only connect attempt
				// Validate host/port
				if wsMsg.Host == "" {
					h.sendError(conn, "Host is required")
					continue
				}

				// Default port to 23
				if wsMsg.Port == 0 {
					wsMsg.Port = 23
				}

				log.Printf("[SP02PH02T01] Connect request: user=%s, host=%s, port=%d", userIDStr, wsMsg.Host, wsMsg.Port)

				// Attempt connection via Manager
				session, err = h.manager.Connect(ctx, userIDStr, wsMsg.Host, wsMsg.Port)
				if err != nil {
					log.Printf("[SP02PH02] Connection failed: %v", err)
					h.sendError(conn, err.Error())
					continue
				}

				// Wait a moment for connection to establish
				time.Sleep(100 * time.Millisecond)

				// Check if actually connected
				session, _ = h.manager.GetSession(userIDStr)
				if session.State != StateConnected {
					h.sendError(conn, "Failed to establish connection")
					continue
				}

				connected = true
				log.Printf("[SP02PH02] Connected to %s:%d for user %s", wsMsg.Host, wsMsg.Port, userIDStr)
			}

			// Send success message (common for both paths)
			err = conn.WriteJSON(WSMessage{
				Type:   MsgTypeStatus,
				Status: StateConnected,
			})
			if err != nil {
				log.Printf("[SP02PH02] Error sending connected status: %v", err)
				continue
			}

			// Start the MUD->client relay
			go h.relayMUDToClient(ctx, userIDStr, conn, mudToClient)

		case MsgTypeDisconnect:
			if connected {
				log.Printf("[SP02PH02] User requested disconnect")
				h.manager.Disconnect(userIDStr, ReasonUser)
				connected = false
			}

		case MsgTypeData:
			// Rate limiting at WebSocket ingress (SP02PH02T04)
			rl := h.getRateLimiter(userIDStr)
			if !rl.Allow() {
				log.Printf("[SP02PH02T04] Rate limit exceeded for user %s", userIDStr)
				h.sendError(conn, "Rate limit exceeded")
				continue
			}

			// Message size enforcement (SP02PH02T03)
			if len(wsMsg.Data) > h.config.MaxMessageSizeBytes {
				log.Printf("[SP02PH02T03] Message too large: %d bytes", len(wsMsg.Data))
				h.sendError(conn, "Message too large")
				continue
			}

			if !connected {
				h.sendError(conn, "Not connected")
				continue
			}

			// Send command to MUD via channel
			select {
			case clientToMUD <- wsMsg.Data:
			default:
				h.sendError(conn, "Command queue full")
			}

		default:
			log.Printf("[SP02PH02] Unknown message type: %s", wsMsg.Type)
		}
	}
}

// readMUDOutput reads output from MUD and sends to mudToClient channel
func (h *WebSocketHandler) readMUDOutput(ctx context.Context, userID string, mudToClient chan<- []byte, statusChan chan<- string) {
	buffer := make([]byte, 8192)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		n, err := h.manager.ReadOutput(userID, buffer)
		if err != nil {
			log.Printf("[SP02PH02] Error reading from MUD: %v", err)
			statusChan <- "disconnected"
			return
		}

		if n > 0 {
			// Copy the data to send through channel
			data := make([]byte, n)
			copy(data, buffer[:n])
			select {
			case mudToClient <- data:
			case <-ctx.Done():
				return
			}
		}

		// Small sleep to prevent tight loop
		time.Sleep(10 * time.Millisecond)
	}
}

// relayMUDToClient relays MUD output to WebSocket client
func (h *WebSocketHandler) relayMUDToClient(ctx context.Context, userID string, conn *websocket.Conn, mudToClient <-chan []byte) {
	for {
		select {
		case <-ctx.Done():
			return
		case data := <-mudToClient:
			// Reset idle timer on inbound data
			h.manager.ResetIdleTimerOnInbound(userID)

			err := conn.WriteMessage(websocket.TextMessage, data)
			if err != nil {
				log.Printf("[SP02PH02] Error writing to WebSocket: %v", err)
				return
			}
		}
	}
}

// handleClientCommands handles commands from client and forwards to MUD
func (h *WebSocketHandler) handleClientCommands(ctx context.Context, userID string, clientToMUD <-chan string, statusChan chan<- string) {
	for {
		select {
		case <-ctx.Done():
			return
		case command := <-clientToMUD:
			err := h.manager.SendCommand(userID, command)
			if err != nil {
				log.Printf("[SP02PH02] Error sending command to MUD: %v", err)
				statusChan <- "disconnected"
				return
			}
		}
	}
}

// sendError sends an error message to the WebSocket client
func (h *WebSocketHandler) sendError(conn *websocket.Conn, errorMsg string) {
	err := conn.WriteJSON(WSMessage{
		Type:  MsgTypeError,
		Error: errorMsg,
	})
	if err != nil {
		log.Printf("[SP02PH02] Error sending error message: %v", err)
	}
}
