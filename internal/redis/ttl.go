package redis

import "time"

// TTL constants for Redis keys
const (
	// OTP TTL: 15 minutes (900 seconds)
	// Used for one-time password storage
	OTPTTL = 900 * time.Second

	// SessionTTL: 24 hours (86400 seconds)
	// Hard cap - absolute maximum session duration
	SessionTTL = 86400 * time.Second

	// SessionIdleTTL: 30 minutes (1800 seconds)
	// Sliding window - resets on user activity
	SessionIdleTTL = 1800 * time.Second

	// OTP rate limit: 1 hour (3600 seconds)
	// Max 5 OTP requests per email per hour
	OTPRateLimitTTL = 3600 * time.Second

	// Login rate limit: 10 minutes (600 seconds)
	// Max 10 login attempts per IP per 10 minutes
	LoginRateLimitTTL = 600 * time.Second
)

// TTLSeconds returns the TTL in seconds for each key type
var TTLSeconds = map[string]int64{
	"otp":              int64(OTPTTL.Seconds()),
	"session":          int64(SessionTTL.Seconds()),
	"session_idle":     int64(SessionIdleTTL.Seconds()),
	"otp_rate_limit":   int64(OTPRateLimitTTL.Seconds()),
	"login_rate_limit": int64(LoginRateLimitTTL.Seconds()),
}
