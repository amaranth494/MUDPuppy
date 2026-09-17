package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/amaranth494/MudPuppy/internal/crypto"
)

// AIModelEntry is one entry in the AI model registry: a slug, the vendor
// model identifier, its API endpoint and API key, and which provider
// implementation understands it. Phase 3 ships the "gemini" provider only;
// the registry shape (endpoint + key per entry) lets another vendor plug
// in later without touching profiles (D-18).
type AIModelEntry struct {
	Slug      string
	ModelName string
	Endpoint  string
	APIKey    string
	Provider  string
}

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

	// Encryption (SP03PH05)
	EncryptionKeyV1 string
	EncryptionKeyV2 string
	EncryptionKeyV3 string

	// Admin
	AdminMetricsSecret string

	// AI model registry (Phase 3, D-18). Model names, endpoints and API
	// keys come only from the environment (CON-model-config); absence is
	// not fatal (D-20) — the server starts normally and #AUTO ON is
	// refused later, by AIConfigured().
	AIDefaultModelSlug string
	AIModels           map[string]AIModelEntry

	// AuthLogOTP (DR-2-01). Off by default: the one-time sign-in code is
	// never printed to the log unless this is explicitly turned on for
	// local debugging. Staging's configuration does not set it. Never a
	// fail-fast case — an unparseable value warns and stays false.
	AuthLogOTP bool
}

// isLocalDevOptOut reports whether MUDPUPPY_LOCAL_DEV's value turns on the
// local-development opt-out from the vault-key requirement (D-26). Only "1"
// and "true" (any letter case) are on; every other value, including unset,
// is off.
func isLocalDevOptOut(value string) bool {
	switch strings.ToLower(value) {
	case "1", "true":
		return true
	default:
		return false
	}
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

	// Encryption keys for credential vault (SP03PH05)
	// Base64-encoded 32-byte keys for AES-256-GCM
	cfg.EncryptionKeyV1 = os.Getenv("ENCRYPTION_KEY_V1")
	cfg.EncryptionKeyV2 = os.Getenv("ENCRYPTION_KEY_V2")
	cfg.EncryptionKeyV3 = os.Getenv("ENCRYPTION_KEY_V3")

	// D-26, DR-4-04: the vault key is required on every host, not only on
	// Railway. Without this, internal/crypto.DefaultKeyStore silently
	// generates a fresh random key on every process start on any
	// non-Railway host, making every previously stored credential
	// undecryptable after a restart -- the exact failure that already hit
	// staging once (DR-3.1-04, when the Railway-only hosting signal was the
	// only thing checked). That hosting signal is no longer consulted here
	// at all: the only opt-out is the explicitly named MUDPUPPY_LOCAL_DEV
	// flag, read once below, which the operator must set on purpose. Never
	// log the key, its length, MUDPUPPY_LOCAL_DEV's value, or any other
	// environment value here.
	//
	// Code review WR-07 of Phase 4: "missing" was the only case checked, and
	// DefaultKeyStore ignores a key it cannot use exactly as it ignores one
	// that is absent -- so a key pasted with quotes or a trailing space, in
	// URL-safe base64, or of the wrong length started the server on a fresh
	// random key all the same. (A trailing line break is fine: Go's base64
	// decoder ignores it, the key store loads the key, and so this gate
	// accepts it too.) The gate asks the key store's own parser
	// (crypto.ParseKey) whether each key that is set is usable. V1 is
	// required (unless the local-development flag is on and V1 is blank);
	// V2 and V3 are optional but, when set, must be usable too -- flag on
	// or off -- because a rotation key that is silently dropped makes
	// everything encrypted under it unreadable in the same way. The
	// message names the variable and never the value.
	localDev := isLocalDevOptOut(os.Getenv("MUDPUPPY_LOCAL_DEV"))
	if cfg.EncryptionKeyV1 == "" {
		if !localDev {
			return nil, errors.New("ENCRYPTION_KEY_V1 environment variable is required; a local development machine can opt out by setting MUDPUPPY_LOCAL_DEV=true")
		}
		log.Print("Warning: starting without a password vault key because MUDPUPPY_LOCAL_DEV is set; saved MUD passwords will not survive a restart")
	}
	for _, k := range []struct{ name, value string }{
		{"ENCRYPTION_KEY_V1", cfg.EncryptionKeyV1},
		{"ENCRYPTION_KEY_V2", cfg.EncryptionKeyV2},
		{"ENCRYPTION_KEY_V3", cfg.EncryptionKeyV3},
	} {
		if k.value == "" {
			continue
		}
		if _, err := crypto.ParseKey(k.value); err != nil {
			return nil, fmt.Errorf("%s environment variable is set but unusable: it must be the standard base64 encoding of exactly 32 bytes, with no quotes or spaces", k.name)
		}
	}

	// AI model registry (Phase 3, D-18). Scanned from the environment once:
	// for every AI_MODEL_<SLUG>_NAME variable, the middle segment is the
	// slug, and AI_MODEL_<SLUG>_ENDPOINT / _KEY / _PROVIDER fill out the
	// rest of that entry. A blank PROVIDER defaults to "gemini". An entry
	// missing an endpoint or a key is skipped and warned about by slug
	// only — never by key value, key length, or an endpoint that might
	// carry a key. AI_MODEL_DEFAULT names the default slug; DEFAULT itself
	// is a reserved slug and may not be declared as an entry. None of this
	// is fatal (D-20): with the Gemini variables entirely absent, the
	// server still starts normally, and #AUTO ON is refused later, by
	// AIConfigured() — never here.
	cfg.AIModels = make(map[string]AIModelEntry)
	const aiModelPrefix = "AI_MODEL_"
	const aiModelNameSuffix = "_NAME"
	for _, envVar := range os.Environ() {
		key, value, found := strings.Cut(envVar, "=")
		if !found {
			continue
		}
		if !strings.HasPrefix(key, aiModelPrefix) || !strings.HasSuffix(key, aiModelNameSuffix) {
			continue
		}
		slug := strings.ToUpper(strings.TrimSuffix(strings.TrimPrefix(key, aiModelPrefix), aiModelNameSuffix))
		if slug == "" {
			continue
		}
		if slug == "DEFAULT" {
			log.Printf("Warning: AI_MODEL_DEFAULT_NAME uses the reserved slug DEFAULT and is ignored; DEFAULT names the default entry, not an entry itself")
			continue
		}

		modelName := value
		endpoint := os.Getenv(aiModelPrefix + slug + "_ENDPOINT")
		apiKey := os.Getenv(aiModelPrefix + slug + "_KEY")
		provider := os.Getenv(aiModelPrefix + slug + "_PROVIDER")
		if provider == "" {
			provider = "gemini"
		}
		if endpoint == "" || apiKey == "" {
			log.Printf("Warning: AI model entry %q is missing an endpoint or a key and will be ignored", slug)
			continue
		}

		cfg.AIModels[slug] = AIModelEntry{
			Slug:      slug,
			ModelName: modelName,
			Endpoint:  endpoint,
			APIKey:    apiKey,
			Provider:  provider,
		}
	}
	cfg.AIDefaultModelSlug = strings.ToUpper(os.Getenv("AI_MODEL_DEFAULT"))

	// One-time sign-in code logging (DR-2-01, optional, defaults to false).
	// Off is the deployed value: staging's configuration does not set this,
	// so the code never reaches the log by default. An unparseable value
	// warns and stays false — this must never become a fail-fast case.
	cfg.AuthLogOTP = false
	if otpLogStr := os.Getenv("AUTH_LOG_OTP"); otpLogStr != "" {
		otpLog, err := strconv.ParseBool(otpLogStr)
		if err != nil {
			log.Printf("Warning: Invalid AUTH_LOG_OTP %q, using default false", otpLogStr)
		} else {
			cfg.AuthLogOTP = otpLog
		}
	}

	return cfg, nil
}

// AIConfigured reports whether the AI model registry names a usable
// default entry: the default slug is set, an entry exists for it, and
// that entry's ModelName, Endpoint and APIKey are all non-empty. A nil
// receiver returns false — a missing dependency fails closed, the same
// discipline EngageGate's doc comment states at
// internal/session/handler.go:27-30.
func (c *Config) AIConfigured() bool {
	if c == nil {
		return false
	}
	if c.AIDefaultModelSlug == "" {
		return false
	}
	entry, ok := c.AIModels[c.AIDefaultModelSlug]
	if !ok {
		return false
	}
	return entry.ModelName != "" && entry.Endpoint != "" && entry.APIKey != ""
}

// ResolveModelEntry resolves a requested model name to a registry entry.
//
//   - requested blank -> the default entry, returned as-is (Phase 1 D-09: a
//     blank profile model name means the server default).
//   - requested matching an entry's slug (case-insensitive) or an entry's
//     ModelName exactly -> that entry.
//   - requested non-blank and unmatched -> the default entry with
//     ModelName replaced by requested, so an unknown or retired model name
//     still reaches the vendor and comes back as a vendor error — a D-13
//     failure — rather than being silently substituted.
//   - no usable default entry at all -> the zero value and false.
func (c *Config) ResolveModelEntry(requested string) (AIModelEntry, bool) {
	if c == nil {
		return AIModelEntry{}, false
	}

	defaultEntry, hasDefault := c.AIModels[c.AIDefaultModelSlug]

	if requested == "" {
		if !hasDefault {
			return AIModelEntry{}, false
		}
		return defaultEntry, true
	}

	if entry, ok := c.AIModels[strings.ToUpper(requested)]; ok {
		return entry, true
	}
	for _, entry := range c.AIModels {
		if entry.ModelName == requested {
			return entry, true
		}
	}

	if !hasDefault {
		return AIModelEntry{}, false
	}
	substituted := defaultEntry
	substituted.ModelName = requested
	return substituted, true
}
