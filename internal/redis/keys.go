package redis

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// Key prefixes
const (
	OTPKeyPrefix      = "otp:email:"
	SessionKeyPrefix  = "session:"
	SessionIdlePrefix = "session_idle:"
	RateLimitPrefix   = "ratelimit:"
)

// OTPKey generates the Redis key for storing OTP
// Format: otp:email:{sha256(lower(email))}
func OTPKey(email string) string {
	hash := sha256.Sum256([]byte(strings.ToLower(email)))
	return OTPKeyPrefix + hex.EncodeToString(hash[:])
}

// SessionKey generates the Redis key for session data
// Format: session:{sessionID}
func SessionKey(sessionID string) string {
	return SessionKeyPrefix + sessionID
}

// SessionIdleKey generates the Redis key for session idle tracking
// Format: session_idle:{sessionID}
func SessionIdleKey(sessionID string) string {
	return SessionIdlePrefix + sessionID
}

// RateLimitKey generates the Redis key for rate limiting
// Format: ratelimit:{type}:{identifier}
// Examples: ratelimit:otp:user@example.com, ratelimit:login:192.168.1.1
func RateLimitKey(rateLimitType string, identifier string) string {
	return RateLimitPrefix + rateLimitType + ":" + identifier
}
