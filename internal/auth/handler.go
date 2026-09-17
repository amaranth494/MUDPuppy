package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/amaranth494/MudPuppy/internal/config"
	"github.com/amaranth494/MudPuppy/internal/email"
	"github.com/amaranth494/MudPuppy/internal/redis"
	"github.com/amaranth494/MudPuppy/internal/store"
	"github.com/google/uuid"
)

// UserStore is the slice of *store.UserStore this handler depends on
// (mirrors internal/driver's per-consumer interface pattern), so tests can
// supply a fake with no database connection. The real *store.UserStore
// satisfies this with no adapter.
type UserStore interface {
	Create(email string) (*store.User, error)
	GetByEmail(email string) (*store.User, error)
	GetByID(id uuid.UUID) (*store.User, error)
	// MarkLoginStarted records the moment userID's current MUDPuppy login
	// began (D-31), read back by internal/store.TranscriptStore's
	// OpenGameSession to decide which earlier game session's Session
	// Memory a new one may inherit.
	MarkLoginStarted(userID uuid.UUID) error
}

// Handler handles authentication HTTP requests
type Handler struct {
	userStore     UserStore
	redisClient   *redis.Client
	emailSender   *email.Sender
	sessionSecret string
	logOTPCode    bool

	// loginBoundaryHook, when set, is told every time a user's login
	// boundary moves -- a sign-in or a sign-out, the same moment
	// markLoginStart records (code review WR-08 of Phase 5). The AI driver
	// uses it to start AI-chatter's per-login call count afresh. This
	// package declares the hook type and never imports the driver.
	loginBoundaryHook func(userID string)
}

// SetLoginBoundaryHook wires the hook markLoginStart calls. A nil hook (the
// zero-value default) is a no-op. Set once at start-up, before the handler
// serves its first request.
func (h *Handler) SetLoginBoundaryHook(hook func(userID string)) {
	h.loginBoundaryHook = hook
}

// NewHandler creates a new auth handler
func NewHandler(userStore UserStore, redisClient *redis.Client, cfg *config.Config) *Handler {
	var emailSender *email.Sender
	if cfg.SMTPHost != "" && cfg.SMTPUser != "" && cfg.SMTPPass != "" {
		emailSender = email.NewSender(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.EmailFromAddress)
	}

	return &Handler{
		userStore:     userStore,
		redisClient:   redisClient,
		emailSender:   emailSender,
		sessionSecret: cfg.SessionSecret,
		logOTPCode:    cfg.AuthLogOTP,
	}
}

// This is the single place in this package allowed to print a
// one-time sign-in code to the log (DR-2-01). Staging previously printed
// the code on every OTP issuance, which meant anyone with staging log
// access could sign in as any staging user; deferred at the Phase 1 and
// Phase 2 security reviews, this closes it. By default (h.logOTPCode
// false, the deployed value) only the audit fact that a code was issued
// is printed — never the code, an email address, the user id, the code's
// length, or any prefix of it. The code prints only when AUTH_LOG_OTP is
// explicitly turned on for local debugging. No caller may print the code
// directly ever again; both former call sites route through this helper.
func (h *Handler) logOTPIssued(otp string) {
	if h.logOTPCode {
		log.Printf("STAGING: OTP sent to user, code: %s", otp)
		return
	}
	log.Printf("[AUTH] one-time sign-in code issued (code suppressed; set AUTH_LOG_OTP=1 to print)")
}

// Request/Response types
type RegisterRequest struct {
	Email string `json:"email"`
}

type RegisterResponse struct {
	Message string `json:"message"`
}

type LoginRequest struct {
	Email string `json:"email"`
	OTP   string `json:"otp"`
}

type LoginResponse struct {
	SessionToken string `json:"session_token"`
}

type LogoutResponse struct {
	Message string `json:"message"`
}

type UserResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// generateOTP generates a 6-digit OTP
func generateOTP() (string, error) {
	buf := make([]byte, 3) // 3 bytes = 24 bits, gives us 6 decimal digits
	_, err := rand.Read(buf)
	if err != nil {
		return "", err
	}

	// Convert to integer and ensure 6 digits
	num := int(buf[0])<<16 | int(buf[1])<<8 | int(buf[2])
	otp := num % 1000000
	return fmt.Sprintf("%06d", otp), nil
}

// generateSessionID generates a unique session ID
func generateSessionID() (string, error) {
	buf := make([]byte, 32)
	_, err := rand.Read(buf)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// Register handles POST /api/v1/register
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate email
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" || !strings.Contains(email, "@") {
		http.Error(w, "Invalid email address", http.StatusBadRequest)
		return
	}

	ctx := context.Background()

	// Check if user exists - for registration, user MUST NOT exist
	existingUser, err := h.userStore.GetByEmail(email)
	if err != nil {
		log.Printf("Error checking user: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// If user already exists, return error for registration
	if existingUser != nil {
		http.Error(w, "Email already registered. Please login instead.", http.StatusConflict)
		return
	}

	// Create new user
	_, err = h.userStore.Create(email)
	if err != nil {
		log.Printf("Error creating user: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Check OTP rate limit (5 OTP requests per email per hour)
	// Log rate limit violations without PII (SP01PH05T04)
	_, err = h.redisClient.CheckRateLimit(ctx, "otp", email, 100, redis.OTPRateLimitTTL)
	if err == redis.ErrRateLimited {
		log.Printf("ABUSE: OTP rate limit exceeded for email hash, endpoint: %s, timestamp: %s",
			r.URL.Path, time.Now().UTC().Format(time.RFC3339))
		http.Error(w, "Too many OTP requests. Please try again later.", http.StatusTooManyRequests)
		return
	}
	if err != nil {
		log.Printf("Error checking rate limit: %v", err)
	}

	// Generate OTP
	otp, err := generateOTP()
	if err != nil {
		log.Printf("Error generating OTP: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Store OTP in Redis (15 min TTL)
	if err := h.redisClient.StoreOTP(ctx, email, otp); err != nil {
		log.Printf("Error storing OTP: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Send OTP via email
	if h.emailSender != nil && h.emailSender.IsConfigured() {
		if err := h.emailSender.SendOTP(email, otp); err != nil {
			// If SMTP fails, delete the OTP from Redis and return 503
			h.redisClient.CheckRateLimit(ctx, "otp", email, 100, redis.OTPRateLimitTTL) // Just to reset, error ignored
			log.Printf("Failed to send OTP email: %v", err)
			http.Error(w, "Service unavailable. Please try again later.", http.StatusServiceUnavailable)
			return
		}
		// Log OTP issuance for staging verification, gated by DR-2-01
		h.logOTPIssued(otp)
	} else {
		// In development, log the OTP
		log.Printf("DEV MODE - OTP for %s: %s", email, otp)
	}

	// Return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(RegisterResponse{
		Message: "Verification code sent to your email",
	})
}

// SendOTP handles POST /api/v1/send-otp - sends OTP to existing users only
func (h *Handler) SendOTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate email
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" || !strings.Contains(email, "@") {
		http.Error(w, "Invalid email address", http.StatusBadRequest)
		return
	}

	ctx := context.Background()

	// Check if user exists - for SendOTP, user MUST exist
	existingUser, err := h.userStore.GetByEmail(email)
	if err != nil {
		log.Printf("Error checking user: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// If user doesn't exist, return error
	if existingUser == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Email not found. Please register first."})
		return
	}

	// Check OTP rate limit (5 OTP requests per email per hour)
	_, err = h.redisClient.CheckRateLimit(ctx, "otp", email, 100, redis.OTPRateLimitTTL)
	if err == redis.ErrRateLimited {
		log.Printf("ABUSE: OTP rate limit exceeded for email hash, endpoint: %s, timestamp: %s",
			r.URL.Path, time.Now().UTC().Format(time.RFC3339))
		http.Error(w, "Too many OTP requests. Please try again later.", http.StatusTooManyRequests)
		return
	}
	if err != nil {
		log.Printf("Error checking rate limit: %v", err)
	}

	// Generate OTP
	otp, err := generateOTP()
	if err != nil {
		log.Printf("Error generating OTP: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Store OTP in Redis (15 min TTL)
	if err := h.redisClient.StoreOTP(ctx, email, otp); err != nil {
		log.Printf("Error storing OTP: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Send OTP via email
	if h.emailSender != nil && h.emailSender.IsConfigured() {
		if err := h.emailSender.SendOTP(email, otp); err != nil {
			// If SMTP fails, delete the OTP from Redis and return 503
			h.redisClient.CheckRateLimit(ctx, "otp", email, 100, redis.OTPRateLimitTTL) // Just to reset, error ignored
			log.Printf("Failed to send OTP email: %v", err)
			http.Error(w, "Service unavailable. Please try again later.", http.StatusServiceUnavailable)
			return
		}
		// Log OTP issuance for staging verification, gated by DR-2-01
		h.logOTPIssued(otp)
	} else {
		// In development, log the OTP
		log.Printf("DEV MODE - OTP for %s: %s", email, otp)
	}

	// Return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(RegisterResponse{
		Message: "Verification code sent to your email",
	})
}

// markLoginStart records userID's login boundary (D-31: Session Memory now
// lives for the MUDPuppy login, per connection profile, not the game
// connection) by calling UserStore.MarkLoginStarted. stage identifies the
// caller ("login" or "logout") in the failure log line. A storage error
// here must never fail the login or logout response the owner is waiting
// on, so it is only logged -- with ids and stage, never an email address or
// the underlying error text -- and never returned.
func (h *Handler) markLoginStart(userID uuid.UUID, stage string) {
	if err := h.userStore.MarkLoginStarted(userID); err != nil {
		log.Printf("[AUTH] stage=%s user_id=%s failed to mark login start", stage, userID)
	}
	// Told whether or not the store call worked: the boundary is the owner
	// signing in or out, not the row update (code review WR-08 of Phase 5).
	if h.loginBoundaryHook != nil {
		h.loginBoundaryHook(userID.String())
	}
}

// Login handles POST /api/v1/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate input
	email := strings.ToLower(strings.TrimSpace(req.Email))
	otp := strings.TrimSpace(req.OTP)

	if email == "" || otp == "" {
		http.Error(w, "Email and OTP are required", http.StatusBadRequest)
		return
	}

	ctx := context.Background()

	// Check login rate limit (10 attempts per IP per 10 minutes)
	// Log rate limit violations without PII (SP01PH05T04)
	clientIP := getClientIP(r)
	_, err := h.redisClient.CheckRateLimit(ctx, "login", clientIP, 10, redis.LoginRateLimitTTL)
	if err == redis.ErrRateLimited {
		log.Printf("ABUSE: Login rate limit exceeded for IP: %s, endpoint: %s, timestamp: %s",
			clientIP, r.URL.Path, time.Now().UTC().Format(time.RFC3339))
		http.Error(w, "Too many login attempts. Please try again later.", http.StatusTooManyRequests)
		return
	}

	// Verify OTP
	err = h.redisClient.VerifyOTP(ctx, email, otp)
	if err != nil {
		if err == redis.ErrOTPNotFound {
			http.Error(w, "Invalid or expired OTP", http.StatusUnauthorized)
			return
		}
		if err.Error() == "invalid otp" {
			http.Error(w, "Invalid OTP", http.StatusUnauthorized)
			return
		}
		log.Printf("Error verifying OTP: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Get user from database
	user, err := h.userStore.GetByEmail(email)
	if err != nil {
		log.Printf("Error getting user: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if user == nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	// Generate session ID
	sessionID, err := generateSessionID()
	if err != nil {
		log.Printf("Error generating session ID: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Store session in Redis
	sessionData := redis.SessionData{
		UserID:    user.ID.String(),
		Email:     user.Email,
		CreatedAt: time.Now(),
	}

	if err := h.redisClient.StoreSession(ctx, sessionID, sessionData); err != nil {
		log.Printf("Error storing session: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// D-31: mark this sign-in as a new login boundary now that the session
	// exists, so a new game session opened during this login knows where
	// its Session Memory inheritance window starts.
	h.markLoginStart(user.ID, "login")

	// Log successful login (without sensitive data)
	log.Printf("User %s logged in successfully", email)

	// Set session cookie (HTTP-only, secure, samesite lax, 24h max age)
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400, // 24 hours - matches session hard cap
	})

	// Return success (no token in body)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(LoginResponse{
		SessionToken: "", // Token is in cookie, not body
	})
}

// Logout handles DELETE /api/v1/logout
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := getSessionToken(r)
	if sessionID == "" {
		http.Error(w, "No session token provided", http.StatusUnauthorized)
		return
	}

	ctx := context.Background()

	// D-31: capture which user is signing out before the session is
	// deleted, so the login boundary can be bumped afterward -- nothing
	// carries across a sign-out even if some future path opens a game
	// session before the next Login. A lookup failure here (an
	// already-expired or invalid session) is not fatal to logout; there is
	// simply no login boundary left to mark.
	loggedOutUserID, sessionLookupErr := h.redisClient.GetSession(ctx, sessionID)

	// Delete session from Redis
	if err := h.redisClient.DeleteSession(ctx, sessionID); err != nil {
		log.Printf("Error deleting session: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if sessionLookupErr == nil && loggedOutUserID != "" {
		if id, err := uuid.Parse(loggedOutUserID); err == nil {
			h.markLoginStart(id, "logout")
		}
	}

	// Clear the session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	log.Printf("User logged out, session: %s", sessionID[:8]+"...")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(LogoutResponse{
		Message: "Logged out successfully",
	})
}

// Me handles GET /api/v1/me
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := getSessionToken(r)
	if sessionID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	ctx := context.Background()

	// Validate session
	userID, err := h.redisClient.GetSession(ctx, sessionID)
	if err != nil {
		if err == redis.ErrInvalidSession || err == redis.ErrSessionExpired || err == redis.ErrSessionIdle {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		log.Printf("Error getting session: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Refresh session idle timer on activity
	h.redisClient.RefreshSession(ctx, sessionID)

	// Get user from database
	id, err := uuid.Parse(userID)
	if err != nil {
		log.Printf("Error parsing user ID: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	user, err := h.userStore.GetByID(id)
	if err != nil {
		log.Printf("Error getting user: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if user == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(UserResponse{
		ID:    user.ID.String(),
		Email: user.Email,
	})
}

// SessionMiddleware validates session for protected routes
func (h *Handler) SessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip if not a protected path
		if !strings.HasPrefix(r.URL.Path, "/api/v1/") {
			next.ServeHTTP(w, r)
			return
		}

		// Skip auth endpoints themselves
		if r.URL.Path == "/api/v1/register" || r.URL.Path == "/api/v1/login" {
			next.ServeHTTP(w, r)
			return
		}

		sessionID := getSessionToken(r)
		if sessionID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.Background()

		// Validate session
		_, err := h.redisClient.GetSession(ctx, sessionID)
		if err != nil {
			if err == redis.ErrInvalidSession || err == redis.ErrSessionExpired || err == redis.ErrSessionIdle {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			log.Printf("Error validating session: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Refresh session idle timer
		h.redisClient.RefreshSession(ctx, sessionID)

		next.ServeHTTP(w, r)
	})
}

// getSessionToken gets the session token from cookie or header
func getSessionToken(r *http.Request) string {
	// First try cookie
	cookie, err := r.Cookie("session_token")
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}

	// Then try Authorization header
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	return ""
}

// getClientIP gets the client IP from request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for reverse proxy)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}
