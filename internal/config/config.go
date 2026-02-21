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

	return cfg, nil
}
