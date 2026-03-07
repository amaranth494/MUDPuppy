package redis

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	// ErrOTPExpired indicates the OTP has expired
	ErrOTPExpired = errors.New("otp expired")
	// ErrOTPNotFound indicates no OTP was found for the email
	ErrOTPNotFound = errors.New("otp not found")
	// ErrInvalidSession indicates the session is invalid or expired
	ErrInvalidSession = errors.New("invalid session")
	// ErrSessionExpired indicates the session has expired (hard cap)
	ErrSessionExpired = errors.New("session expired")
	// ErrSessionIdle indicates the session is idle (idle timeout exceeded)
	ErrSessionIdle = errors.New("session idle timeout exceeded")
	// ErrRateLimited indicates the rate limit has been exceeded
	ErrRateLimited = errors.New("rate limit exceeded")
)

// Client wraps the Redis client and provides authentication-specific methods
type Client struct {
	rdb *redis.Client
}

// NewClient creates a new Redis client from a URL
func NewClient(redisURL string) (*Client, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	rdb := redis.NewClient(opt)
	return &Client{rdb: rdb}, nil
}

// NewClientFromExisting creates a Client wrapper from an existing Redis client
func NewClientFromExisting(rdb *redis.Client) *Client {
	return &Client{rdb: rdb}
}

// Ping checks the Redis connection
func (c *Client) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

// Close closes the Redis connection
func (c *Client) Close() error {
	return c.rdb.Close()
}

// ============================================================================
// OTP Operations (SP01PH02T02)
// ============================================================================

// StoreOTP stores an OTP in Redis with 15-minute TTL
// The OTP is stored using atomic SET with EX option
func (c *Client) StoreOTP(ctx context.Context, email string, otp string) error {
	key := OTPKey(email)
	return c.rdb.Set(ctx, key, otp, OTPTTL).Err()
}

// GetOTP retrieves and deletes an OTP from Redis (single-use)
func (c *Client) GetOTP(ctx context.Context, email string) (string, error) {
	key := OTPKey(email)

	// Use GETDEL to atomically get and delete the OTP
	result, err := c.rdb.GetDel(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", ErrOTPNotFound
		}
		return "", err
	}
	return result, nil
}

// VerifyOTP verifies an OTP and consumes it (single-use)
func (c *Client) VerifyOTP(ctx context.Context, email string, inputOTP string) error {
	storedOTP, err := c.GetOTP(ctx, email)
	if err != nil {
		return err
	}

	if storedOTP != inputOTP {
		return errors.New("invalid otp")
	}

	return nil
}

// ============================================================================
// Session Operations (SP01PH02T03 - Dual Key)
// ============================================================================

// SessionData represents the session data stored in Redis
type SessionData struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// StoreSession stores a session using dual-key approach:
// - session:{id} - holds session data, hard cap 24h TTL
// - session_idle:{id} - idle marker, sliding 30min TTL
func (c *Client) StoreSession(ctx context.Context, sessionID string, data SessionData) error {
	sessionKey := SessionKey(sessionID)
	idleKey := SessionIdleKey(sessionID)

	// Store session data with hard cap TTL (24 hours)
	if err := c.rdb.Set(ctx, sessionKey, data.UserID, SessionTTL).Err(); err != nil {
		return err
	}

	// Store idle marker with sliding TTL (30 minutes)
	if err := c.rdb.Set(ctx, idleKey, "active", SessionIdleTTL).Err(); err != nil {
		return err
	}

	return nil
}

// GetSession retrieves session data if both session key and idle key exist
// Returns ErrSessionExpired if hard cap exceeded, ErrSessionIdle if idle exceeded
func (c *Client) GetSession(ctx context.Context, sessionID string) (string, error) {
	sessionKey := SessionKey(sessionID)
	idleKey := SessionIdleKey(sessionID)

	// Check both keys exist
	exists, err := c.rdb.Exists(ctx, sessionKey, idleKey).Result()
	if err != nil {
		return "", err
	}

	// If neither exists, session is completely invalid
	if exists == 0 {
		return "", ErrInvalidSession
	}

	// If only session exists (idle expired), it's an idle timeout
	if exists == 1 {
		// Check which one exists
		sessionExists, _ := c.rdb.Exists(ctx, sessionKey).Result()
		if sessionExists == 1 {
			return "", ErrSessionIdle
		}
		return "", ErrInvalidSession
	}

	// Both exist - get user ID
	userID, err := c.rdb.Get(ctx, sessionKey).Result()
	if err != nil {
		if err == redis.Nil {
			return "", ErrSessionExpired
		}
		return "", err
	}

	return userID, nil
}

// RefreshSession updates the idle TTL on activity (sliding window)
func (c *Client) RefreshSession(ctx context.Context, sessionID string) error {
	idleKey := SessionIdleKey(sessionID)

	// Update idle marker TTL (sliding window)
	return c.rdb.Expire(ctx, idleKey, SessionIdleTTL).Err()
}

// DeleteSession removes both session keys (logout)
func (c *Client) DeleteSession(ctx context.Context, sessionID string) error {
	sessionKey := SessionKey(sessionID)
	idleKey := SessionIdleKey(sessionID)

	_, err := c.rdb.Del(ctx, sessionKey, idleKey).Result()
	return err
}

// ============================================================================
// Generic Key-Value Operations (SP02)
// ============================================================================

// Set stores a string value with TTL
func (c *Client) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return c.rdb.Set(ctx, key, value, ttl).Err()
}

// Get retrieves a string value
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	return c.rdb.Get(ctx, key).Result()
}

// Delete removes a key
func (c *Client) Delete(ctx context.Context, key string) error {
	return c.rdb.Del(ctx, key).Err()
}

// Exists checks if a key exists
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	n, err := c.rdb.Exists(ctx, key).Result()
	return n > 0, err
}

// ============================================================================
// Rate Limiting Operations (SP01PH05)
// ============================================================================

// CheckRateLimit checks if the rate limit has been exceeded
// Returns ErrRateLimited if limit exceeded
func (c *Client) CheckRateLimit(ctx context.Context, rateLimitType string, identifier string, maxAttempts int, ttl time.Duration) (int, error) {
	key := RateLimitKey(rateLimitType, identifier)

	// Increment counter
	count, err := c.rdb.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	// Set expiry on first request
	if count == 1 {
		c.rdb.Expire(ctx, key, ttl)
	}

	// Check if over limit
	if int(count) > maxAttempts {
		return int(count), ErrRateLimited
	}

	return int(count), nil
}

// GetRateLimitRemaining returns remaining attempts for a rate limit key
func (c *Client) GetRateLimitRemaining(ctx context.Context, rateLimitType string, identifier string, maxAttempts int) (int, error) {
	key := RateLimitKey(rateLimitType, identifier)

	count, err := c.rdb.Get(ctx, key).Int()
	if err == redis.Nil {
		return maxAttempts, nil
	}
	if err != nil {
		return 0, err
	}

	remaining := maxAttempts - count
	if remaining < 0 {
		return 0, nil
	}
	return remaining, nil
}

// ResetRateLimit resets the rate limit counter
func (c *Client) ResetRateLimit(ctx context.Context, rateLimitType string, identifier string) error {
	key := RateLimitKey(rateLimitType, identifier)
	return c.rdb.Del(ctx, key).Err()
}
