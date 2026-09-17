// SP01PH05: Security & Rate Limiting - Security headers middleware

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/amaranth494/MudPuppy/internal/auth"
	"github.com/amaranth494/MudPuppy/internal/config"
	"github.com/amaranth494/MudPuppy/internal/connections"
	"github.com/amaranth494/MudPuppy/internal/crypto"
	aidriver "github.com/amaranth494/MudPuppy/internal/driver"
	"github.com/amaranth494/MudPuppy/internal/gemini"
	"github.com/amaranth494/MudPuppy/internal/help"
	"github.com/amaranth494/MudPuppy/internal/icm"
	"github.com/amaranth494/MudPuppy/internal/metrics"
	"github.com/amaranth494/MudPuppy/internal/profiles"
	"github.com/amaranth494/MudPuppy/internal/redis"
	"github.com/amaranth494/MudPuppy/internal/session"
	"github.com/amaranth494/MudPuppy/internal/store"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type HealthResponse struct {
	Status string `json:"status"`
}

// transcriptSinkAdapter satisfies session.TranscriptSink by converting
// []session.TranscriptLine to []store.GameSessionLine. internal/session
// declares its own TranscriptLine type (rather than importing
// internal/store) so the interface lives in the consuming package; the two
// struct shapes are structurally identical, but Go interfaces require the
// declared method types to match exactly, so a thin adapter is required
// here in main, the one place that already imports both packages.
type transcriptSinkAdapter struct {
	store *store.TranscriptStore
}

func (a *transcriptSinkAdapter) OpenGameSession(userID, connectionID uuid.UUID) (uuid.UUID, error) {
	return a.store.OpenGameSession(userID, connectionID)
}

func (a *transcriptSinkAdapter) AppendGameLines(gameSessionID uuid.UUID, lines []session.TranscriptLine) error {
	storeLines := make([]store.GameSessionLine, len(lines))
	for i, l := range lines {
		storeLines[i] = store.GameSessionLine{Seq: l.Seq, Source: l.Source, Text: l.Text, CreatedAt: l.CreatedAt}
	}
	return a.store.AppendGameLines(gameSessionID, storeLines)
}

func (a *transcriptSinkAdapter) CloseGameSession(gameSessionID uuid.UUID) error {
	return a.store.CloseGameSession(gameSessionID)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := HealthResponse{Status: "ok"}
	json.NewEncoder(w).Encode(response)
}

func main() {
	// Load configuration (SP01PH03T08)
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("FATAL: %v", err)
	}
	log.Println("Configuration loaded successfully")

	// Read database configuration from environment
	databaseURL := os.Getenv("DATABASE_URL")
	redisURL := os.Getenv("REDIS_URL")

	// Fail-fast if DATABASE_URL is missing (SP01PH01T04)
	if databaseURL == "" {
		log.Fatal("FATAL: DATABASE_URL environment variable is required. Set it before starting the server.")
	}

	// Connect to PostgreSQL
	log.Println("Connecting to PostgreSQL...")
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}

	// Set connection timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("Failed to ping PostgreSQL: %v", err)
	}
	log.Println("Connected to PostgreSQL")
	defer db.Close()

	// Run migrations using golang-migrate (SP01PH01T01, SP01PH06T01)
	log.Println("Running database migrations (golang-migrate)...")

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Fatalf("Failed to create database driver: %v", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres",
		driver,
	)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	version, dirty, verr := m.Version()
	if verr == nil {
		log.Printf("Migrations completed successfully (version=%d, dirty=%v)", version, dirty)
	} else {
		log.Printf("Migrations completed successfully (version unavailable: %v)", verr)
	}

	// Ensure timers column exists (fallback for when migration 009 isn't in migrate.zip)
	log.Println("Ensuring timers column exists...")
	_, err = db.Exec(`ALTER TABLE profiles ADD COLUMN IF NOT EXISTS timers JSONB NOT NULL DEFAULT '{"items": []}'`)
	if err != nil {
		log.Printf("Warning: Failed to ensure timers column: %v", err)
	} else {
		log.Println("Timers column ensured")
	}

	// Ensure AI Player columns exist (fallback for when migration 010 isn't in migrate.zip)
	log.Println("Ensuring AI Player columns exist...")
	_, err = db.Exec(`ALTER TABLE profiles
		ADD COLUMN IF NOT EXISTS conduct_rules TEXT NOT NULL DEFAULT '',
		ADD COLUMN IF NOT EXISTS approach_guidance TEXT NOT NULL DEFAULT '',
		ADD COLUMN IF NOT EXISTS ai_settings JSONB NOT NULL DEFAULT '{}'::jsonb,
		ADD COLUMN IF NOT EXISTS policy_version_accepted TEXT,
		ADD COLUMN IF NOT EXISTS policy_accepted_at TIMESTAMPTZ`)
	if err != nil {
		log.Printf("Warning: Failed to ensure AI Player columns: %v", err)
	} else {
		log.Println("AI Player columns ensured")
	}

	// Fail-fast if REDIS_URL is missing (SP01PH02T04)
	if redisURL == "" {
		log.Fatal("FATAL: REDIS_URL environment variable is required. Set it before starting the server.")
	}

	// Connect to Redis
	log.Println("Connecting to Redis...")
	redisClient, err := redis.NewClient(redisURL)
	if err != nil {
		log.Fatalf("Failed to create Redis client: %v", err)
	}

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("Connected to Redis")
	defer redisClient.Close()

	// Initialize stores and handlers
	userStore := store.NewUserStore(db)
	authHandler := auth.NewHandler(userStore, redisClient, cfg)

	// Initialize connections handler FIRST (SP03PH05) - needed for session callbacks
	connectionStore := store.NewConnectionStore(db)
	credentialsStore := store.NewCredentialsStore(db)
	profileStore := store.NewProfileStore(db)
	// transcriptStore backs the session transcript (D-14): one store
	// instance, two consumers — the session manager's write-side tap and
	// the profiles handler's owner-scoped read endpoints below.
	transcriptStore := store.NewTranscriptStore(db)
	// Create encryption key store - uses DefaultKeyStore to generate default key if none configured
	keyStore, err := crypto.DefaultKeyStore()
	if err != nil {
		log.Fatalf("Failed to initialize encryption key store: %v", err)
	}

	// Initialize session manager first (SP02PH01)
	sessionManager := session.NewManager(
		cfg.PortWhitelist,
		cfg.PortDenylist,
		cfg.PortAllowlistOverride,
		cfg.IdleTimeoutMinutes,
		cfg.HardSessionCapHours,
	)

	// Initialize connections handler with session manager (SP03PH06)
	connectionsHandler := connections.NewHandler(connectionStore, credentialsStore, keyStore, sessionManager)

	// Wire the session transcript tap (D-14): every saved-profile
	// connection is transcribed from connect to disconnect. A thin adapter
	// bridges internal/session's own TranscriptLine type to
	// internal/store's GameSessionLine (see transcriptSinkAdapter above).
	sessionManager.SetTranscriptSink(&transcriptSinkAdapter{store: transcriptStore})

	// Initialize profiles handler (SP04PH02), with the session-log
	// endpoints wired to the same transcriptStore (D-17).
	profilesHandler := profiles.NewHandlerWithTranscripts(profileStore, transcriptStore)

	// Initialize the ICM engine exactly once (Phase 3, 03-03-02). internal/icm
	// has existed since before this project started (dispatcher, safety
	// checker, automation execution context) but nothing has ever imported
	// it - this is its first caller. icmEngine stays in scope for the rest
	// of main: plan 03-08's driver constructor receives
	// icmEngine.GetDispatcher() so the AI driver's commands and the HTTP
	// routes below share one Dispatcher and therefore one SafetyChecker
	// state (T-3-20).
	icmEngine := icm.NewEngine()
	icmHandler := icm.NewHandlerWithEngine(icmEngine)

	// Initialize help handler (SP06PH01T04)
	helpHandler := help.NewHandler("./help")

	// Create session handler with callbacks for SP03PH05 (connections integration)
	sessionHandler := session.NewHandlerWithCallbacks(sessionManager, cfg, &session.HandlerCallbacks{
		OnConnected: func(connectionID, userID uuid.UUID) error {
			return connectionsHandler.UpdateLastConnectedAt(connectionID, userID)
		},
		GetAutoLogin: func(connectionID uuid.UUID) (string, string, error) {
			return connectionsHandler.GetCredentialsForAutoLogin(connectionID)
		},
		SendCredentials: func(userID, username, password string) error {
			return sessionManager.SendCredentials(userID, username, password)
		},
		EngageGate: func(connectionID, userID uuid.UUID) (bool, string, string) {
			profile, err := profileStore.GetProfileByConnection(userID, connectionID)
			if err != nil || profile == nil {
				// An unknown or unowned connection is indistinguishable
				// from an unaccepted one on the wire (T-2-02).
				return false, store.EngageGateRefusalMessage, ""
			}
			if !store.EngageGateAllowed(profile.PolicyVersionAccepted, profile.PolicyAcceptedAt) {
				return false, store.EngageGateRefusalMessage, ""
			}
			policyVersion := ""
			if profile.PolicyVersionAccepted != nil {
				policyVersion = *profile.PolicyVersionAccepted
			}
			return true, "", policyVersion
		},
	})

	// Build the Phase 3/4 AI driver (03-08, 04-03): one instance shares
	// icmEngine's dispatcher with the HTTP ICM routes above (T-3-20), and
	// is started as a continuous, paced loop by exactly two callers — a
	// real #AUTO ON engage (sessionHandler.SetEngageHook, fired only on a
	// real state change) and a WAITING-to-ON reconnect resume
	// (sessionManager.SetEngageHook, D-02) — and stopped by the switch
	// leaving on, in either direction, from any cause (the owner's #AUTO
	// OFF, a wheel-grab, or a disconnect), via
	// sessionManager.SetDisengageHook. The notifier is nil at construction
	// and wired below once wsHandler exists (plan 03-09).
	decisionStore := store.NewDecisionStore(db)
	// questStore backs the goal endpoint's reactivate-or-create call (D-04,
	// plan 04-06). It is constructed beside decisionStore because both are
	// per-connection AI Player audit/memory stores sharing the same *sql.DB.
	questStore := store.NewQuestStore(db)
	// 120s: current Gemini flash models take well over the 30s default to return a
	// structured answer (seen on staging 2026-09-16: transport timeout at exactly 30s).
	geminiClient := gemini.NewClient(120 * time.Second)
	aiDriver := aidriver.New(sessionManager, profileStore, decisionStore, geminiClient, icmEngine.GetDispatcher(), nil, cfg)
	sessionManager.SetEngageHook(aiDriver.EngageLoop)
	sessionHandler.SetEngageHook(aiDriver.EngageLoop)
	sessionManager.SetDisengageHook(aiDriver.StopLoop)
	// Wire the real Quest and Session Memory stores (D-10, D-11, plan
	// 04-08) so the driver's prompt-context reads (plan 04-07) and its own
	// curation writes (this plan) land against the same questStore and
	// transcriptStore every other AI Player surface already uses.
	aiDriver.SetQuests(questStore)
	aiDriver.SetMemory(transcriptStore)

	// Wire the same decisionStore instance to the decisions-read endpoint
	// (plan 03-09) so a reloaded play screen reads back exactly what the
	// driver above wrote.
	profilesHandler.SetDecisionStore(decisionStore)
	// Wire questStore to the goal endpoint (plan 04-06, D-04).
	profilesHandler.SetQuestStore(questStore)

	// Initialize WebSocket handler (SP02PH02)
	wsHandler := session.NewWebSocketHandler(sessionManager, cfg)

	// Wire the driver's notifications to the owner's open play screen
	// (plan 03-09, D-08). The push's own error is deliberately discarded —
	// PushAI already returns nil for a user with no open screen, and a
	// closed browser tab must never affect the driver (T-3-36). The
	// decision row is already stored before this notification runs, so a
	// refresh recovers it through the decisions-read endpoint even if the
	// tab was closed at the moment the decision happened.
	aiDriver.SetNotifier(aidriver.NotifierFunc(func(userID string, ev aidriver.Event) {
		_ = wsHandler.PushAI(userID, session.AIDecisionPayload{
			ID:            ev.ID,
			Kind:          ev.Kind,
			Reasoning:     ev.Reasoning,
			Command:       ev.Command,
			Outcome:       ev.Outcome,
			Message:       ev.Message,
			Timestamp:     ev.Timestamp,
			State:         ev.State,
			Calls:         ev.Calls,
			CallCap:       ev.CallCap,
			CallCapSet:    ev.CallCapSet,
			Failures:      ev.Failures,
			Blocks:        ev.Blocks,
			Threshold:     ev.Threshold,
			SessionMemory: ev.SessionMemory,
		})
	}))

	// Wire the goal endpoint's goal-changed/goal-cleared system line
	// (plan 04-06, D-03) through the exact same wsHandler.PushAI path the
	// driver's own notifications take above — one message-delivery
	// mechanism, not two.
	profilesHandler.SetAINotifier(func(userID string, ev profiles.AIEvent) {
		_ = wsHandler.PushAI(userID, session.AIDecisionPayload{
			ID:        ev.ID,
			Kind:      ev.Kind,
			Outcome:   ev.Outcome,
			Message:   ev.Message,
			Timestamp: ev.Timestamp,
		})
	})

	// Initialize metrics (SP02PH04T03)
	metrics.Init()

	// Create router with session middleware
	mux := http.NewServeMux()

	// Add connections endpoints to mux (SP03PH05)
	mux.HandleFunc("/api/v1/connections", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			connectionsHandler.List(w, r)
		case http.MethodPost:
			connectionsHandler.Create(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	// Connections API with explicit routes (SP03PH06)
	mux.HandleFunc("/api/v1/connections/recent", connectionsHandler.GetRecent)
	mux.HandleFunc("/api/v1/connections/{id}/credentials/status", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			connectionsHandler.GetCredentialsStatus(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/v1/connections/{id}/credentials", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			connectionsHandler.GetCredentialsStatus(w, r)
		case http.MethodPost, http.MethodPut:
			connectionsHandler.SetCredentials(w, r)
		case http.MethodDelete:
			connectionsHandler.DeleteCredentials(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	// PR01PH08: Route for getting credentials for automation (only returns password if auto_login enabled)
	mux.HandleFunc("/api/v1/connections/{id}/credentials/auto", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			connectionsHandler.GetAutomationCredentials(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/v1/connections/{id}/connect", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			connectionsHandler.Connect(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// SP04PH07: Profile route must be registered BEFORE connections/{id} to avoid route conflict
	mux.HandleFunc("/api/v1/connections/{id}/profile", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			profilesHandler.GetByConnection(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/v1/connections/{id}", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			connectionsHandler.Get(w, r)
		case http.MethodPut:
			connectionsHandler.Update(w, r)
		case http.MethodDelete:
			connectionsHandler.Delete(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Public endpoints
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/api/v1/help", helpHandler.HandlerFunc())
	mux.HandleFunc("/api/v1/help/", helpHandler.HandlerFunc())
	mux.HandleFunc("/api/v1/register", authHandler.Register)
	mux.HandleFunc("/api/v1/send-otp", authHandler.SendOTP)
	mux.HandleFunc("/api/v1/login", authHandler.Login)
	mux.HandleFunc("/api/v1/logout", authHandler.Logout)
	mux.HandleFunc("/api/v1/me", authHandler.Me)

	// Add session endpoints to mux (SP02PH01)
	mux.HandleFunc("/api/v1/session/connect", sessionHandler.Connect)
	mux.HandleFunc("/api/v1/session/disconnect", sessionHandler.Disconnect)
	mux.HandleFunc("/api/v1/session/status", sessionHandler.Status)
	mux.HandleFunc("/api/v1/session/autopilot", sessionHandler.Autopilot)

	// Add ICM endpoints to mux (Phase 3, 03-03-02) - the four routes have
	// never been reachable before this. Registered on the same mux every
	// other /api/v1 route uses, so they sit inside the same
	// sessionMiddleware wrap below; no separate auth path is introduced.
	icmHandler.RegisterRoutes(mux)

	// Add profiles endpoints to mux (SP04PH02)
	mux.HandleFunc("/api/v1/profiles/{id}", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			profilesHandler.Get(w, r)
		case http.MethodPut:
			profilesHandler.Update(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Add automation endpoints to mux (SP05PH00)
	mux.HandleFunc("/api/v1/profiles/{connection_id}/aliases", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			profilesHandler.GetAliases(w, r)
		case http.MethodPut:
			profilesHandler.PutAliases(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/v1/profiles/{connection_id}/triggers", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			profilesHandler.GetTriggers(w, r)
		case http.MethodPut:
			profilesHandler.PutTriggers(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/v1/profiles/{connection_id}/environment", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			profilesHandler.GetEnvironment(w, r)
		case http.MethodPut:
			profilesHandler.PutEnvironment(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/v1/profiles/{connection_id}/timers", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			profilesHandler.GetTimers(w, r)
		case http.MethodPut:
			profilesHandler.PutTimers(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/v1/profiles/{connection_id}/ai-settings", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			profilesHandler.GetAISettings(w, r)
		case http.MethodPut:
			profilesHandler.PutAISettings(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/v1/profiles/{connection_id}/ai-goal", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			profilesHandler.GetGoal(w, r)
		case http.MethodPut:
			profilesHandler.PutGoal(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/v1/profiles/{connection_id}/ai-memory", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			profilesHandler.GetSessionMemory(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/v1/profiles/{connection_id}/policy", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			profilesHandler.GetPolicy(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/v1/profiles/{connection_id}/policy/accept", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			profilesHandler.AcceptPolicy(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/v1/profiles/{connection_id}/engage-gate", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			profilesHandler.GetEngageGate(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Session log endpoints (D-17): a profile's sessions listed by date and
	// time, and one session's transcript read back as text. Both resolve
	// ownership through GetProfileByConnection before any row is read
	// (T-3-04).
	mux.HandleFunc("/api/v1/profiles/{connection_id}/sessions", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			profilesHandler.ListSessions(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/v1/profiles/{connection_id}/sessions/{session_id}", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			profilesHandler.GetSessionTranscript(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// A connection's decisions, read back in the order they happened
	// (D-12, plan 03-09): this is what lets a reloaded play screen rebuild
	// what it missed. Ownership is resolved through GetProfileByConnection
	// before any row is read (T-3-03).
	mux.HandleFunc("/api/v1/profiles/{connection_id}/decisions", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			profilesHandler.GetDecisions(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// WebSocket endpoint (SP02PH02)
	mux.HandleFunc("/api/v1/session/stream", wsHandler.HandleWebSocket)

	// Admin metrics endpoint (SP02PH04T03)
	mux.HandleFunc("/api/v1/admin/metrics", metricsHandler(cfg))

	// Serve static files from public directory (SPA mode - serve index.html for non-file routes)
	publicDir := http.Dir("./public")
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the requested file
		path := r.URL.Path
		f, err := publicDir.Open(path)
		if err != nil {
			// File doesn't exist - serve index.html for SPA routing
			// This handles routes like /play, /connections, etc.
			index, err := publicDir.Open("/index.html")
			if err != nil {
				http.NotFound(w, r)
				return
			}
			defer index.Close()
			w.Header().Set("Content-Type", "text/html")
			http.ServeContent(w, r, "index.html", time.Now(), index.(io.ReadSeeker))
			return
		}
		defer f.Close()
		// Serve the file
		http.FileServer(publicDir).ServeHTTP(w, r)
	})

	// Protected endpoints - wrapped with session middleware
	protectedHandler := sessionMiddleware(redisClient, mux)

	// Add security headers middleware (SP01PH05T03)
	handler := securityHeadersMiddleware(protectedHandler)

	log.Printf("Server starting on port %s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, handler); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// metricsHandler handles the /api/v1/admin/metrics endpoint (SP02PH04T03)
func metricsHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check for secret if configured
		if cfg.AdminMetricsSecret != "" {
			secret := r.Header.Get("X-Admin-Secret")
			if secret != cfg.AdminMetricsSecret {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
		}

		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		metricsSnapshot := metrics.Get().GetSnapshot()
		w.Write([]byte(metricsSnapshot))
	}
}

// sessionMiddleware validates session for protected routes
func sessionMiddleware(redisClient *redis.Client, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Route the request first to determine if it's a protected path
		path := r.URL.Path

		// Allow public endpoints through
		if path == "/health" || path == "/api/v1/register" || path == "/api/v1/send-otp" || path == "/api/v1/login" {
			next.ServeHTTP(w, r)
			return
		}

		// For /api/v1/* paths (except register/login), require session
		if strings.HasPrefix(path, "/api/v1/") {
			sessionToken := getSessionToken(r)
			if sessionToken == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := context.Background()
			userID, err := redisClient.GetSession(ctx, sessionToken)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Refresh session idle timer
			redisClient.RefreshSession(ctx, sessionToken)

			// Add user ID to context for session handlers (SP02PH01)
			r = r.WithContext(context.WithValue(r.Context(), "user_id", userID))
		}

		next.ServeHTTP(w, r)
	})
}

// securityHeadersMiddleware adds security headers to all responses (SP01PH05T03)
func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")
		// XSS protection
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")
		// Content Security Policy
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self'")
		// Referrer policy
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		// Permissions policy
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		next.ServeHTTP(w, r)
	})
}

// getSessionToken extracts session token from cookie or Authorization header
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
