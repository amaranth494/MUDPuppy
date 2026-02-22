package config

import (
	"errors"
	"log"
	"os"
	"strconv"
)

// Config holds all configuration values
type Config struct {
	// Server
	Port string

	// Session
	SessionSecret string

	// SMTP
	SMTPHost         string
	SMTPPort         int
	SMTPUser         string
	SMTPPass         string
	EmailFromAddress string

	// OTP
	OTPExpiryMinutes int

	// MUD Session Proxy (SP02)
	PortWhitelist          string
	PortDenylist           string
	PortAllowlistOverride  string
	IdleTimeoutMinutes     int
	HardSessionCapHours    int
	MaxMessageSizeBytes    int
	CommandRateLimitPerSec int

	// Admin
	AdminMetricsSecret string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{}

	// Server port (optional, defaults to 8080)
	cfg.Port = os.Getenv("PORT")
	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	// Session secret (required - fail-fast)
	cfg.SessionSecret = os.Getenv("SESSION_SECRET")
	if cfg.SessionSecret == "" {
		return nil, errors.New("SESSION_SECRET environment variable is required")
	}

	// SMTP configuration (required for email delivery)
	cfg.SMTPHost = os.Getenv("SMTP_HOST")
	cfg.SMTPPort = 587 // Default SMTP port
	if portStr := os.Getenv("SMTP_PORT"); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			log.Printf("Warning: Invalid SMTP_PORT '%s', using default 587", portStr)
		} else {
			cfg.SMTPPort = port
		}
	}
	cfg.SMTPUser = os.Getenv("SMTP_USER")
	cfg.SMTPPass = os.Getenv("SMTP_PASS")
	cfg.EmailFromAddress = os.Getenv("EMAIL_FROM_ADDRESS")
	if cfg.EmailFromAddress == "" {
		cfg.EmailFromAddress = "noreply@" + cfg.SMTPHost
	}

	// OTP expiry (optional, defaults to 15 minutes)
	cfg.OTPExpiryMinutes = 15
	if expiryStr := os.Getenv("OTP_EXPIRY_MINUTES"); expiryStr != "" {
		expiry, err := strconv.Atoi(expiryStr)
		if err != nil {
			log.Printf("Warning: Invalid OTP_EXPIRY_MINUTES '%s', using default 15", expiryStr)
		} else {
			cfg.OTPExpiryMinutes = expiry
		}
	}

	// MUD Session Proxy configuration (SP02PH01)
	// Port whitelist (comma-separated, defaults to 23 for telnet)
	cfg.PortWhitelist = os.Getenv("MUD_PROXY_PORT_WHITELIST")
	if cfg.PortWhitelist == "" {
		cfg.PortWhitelist = "23" // Default: telnet only
	}

	// Idle timeout in minutes (defaults to 30)
	cfg.IdleTimeoutMinutes = 30
	if idleStr := os.Getenv("IDLE_TIMEOUT_MINUTES"); idleStr != "" {
		idle, err := strconv.Atoi(idleStr)
		if err != nil {
			log.Printf("Warning: Invalid IDLE_TIMEOUT_MINUTES '%s', using default 30", idleStr)
		} else {
			cfg.IdleTimeoutMinutes = idle
		}
	}

	// Hard session cap in hours (defaults to 24)
	cfg.HardSessionCapHours = 24
	if capStr := os.Getenv("HARD_SESSION_CAP_HOURS"); capStr != "" {
		cap, err := strconv.Atoi(capStr)
		if err != nil {
			log.Printf("Warning: Invalid HARD_SESSION_CAP_HOURS '%s', using default 24", capStr)
		} else {
			cfg.HardSessionCapHours = cap
		}
	}

	// Max message size in bytes (defaults to 64KB)
	cfg.MaxMessageSizeBytes = 65536
	if sizeStr := os.Getenv("MAX_MESSAGE_SIZE_BYTES"); sizeStr != "" {
		size, err := strconv.Atoi(sizeStr)
		if err != nil {
			log.Printf("Warning: Invalid MAX_MESSAGE_SIZE_BYTES '%s', using default 65536", sizeStr)
		} else {
			cfg.MaxMessageSizeBytes = size
		}
	}

	// Command rate limit per second (defaults to 10)
	cfg.CommandRateLimitPerSec = 10
	if rateStr := os.Getenv("COMMAND_RATE_LIMIT_PER_SECOND"); rateStr != "" {
		rate, err := strconv.Atoi(rateStr)
		if err != nil {
			log.Printf("Warning: Invalid COMMAND_RATE_LIMIT_PER_SECOND '%s', using default 10", rateStr)
		} else {
			cfg.CommandRateLimitPerSec = rate
		}
	}

	// Port denylist (SP02PH04T06) - comma-separated, defaults to dangerous ports
	cfg.PortDenylist = os.Getenv("MUD_PORT_DENYLIST")
	if cfg.PortDenylist == "" {
		// Default deny-list: Email (25,465,587,110,143,993,995), DNS (53), Web (80,443),
		// Databases (1433,1521,3306,5432,6379,27017), Remote admin (22,3389,5900), File sharing (445,139,2049)
		cfg.PortDenylist = "25,465,587,110,143,993,995,53,80,443,1433,1521,3306,5432,6379,27017,22,3389,5900,445,139,2049"
	}

	// Port allowlist override (optional - if set, only these ports are allowed)
	cfg.PortAllowlistOverride = os.Getenv("MUD_PORT_ALLOWLIST")

	// Admin metrics endpoint secret (required for /api/v1/admin/metrics)
	cfg.AdminMetricsSecret = os.Getenv("ADMIN_METRICS_SECRET")

	return cfg, nil
}
