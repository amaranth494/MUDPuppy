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
	"github.com/amaranth494/MudPuppy/internal/metrics"
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

	log.Println("Migrations completed successfully")

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
	})

	// Initialize WebSocket handler (SP02PH02)
	wsHandler := session.NewWebSocketHandler(sessionManager, cfg)

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
	mux.HandleFunc("/api/v1/connections/{id}/connect", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			connectionsHandler.Connect(w, r)
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
	mux.HandleFunc("/api/v1/register", authHandler.Register)
	mux.HandleFunc("/api/v1/send-otp", authHandler.SendOTP)
	mux.HandleFunc("/api/v1/login", authHandler.Login)
	mux.HandleFunc("/api/v1/logout", authHandler.Logout)
	mux.HandleFunc("/api/v1/me", authHandler.Me)

	// Add session endpoints to mux (SP02PH01)
	mux.HandleFunc("/api/v1/session/connect", sessionHandler.Connect)
	mux.HandleFunc("/api/v1/session/disconnect", sessionHandler.Disconnect)
	mux.HandleFunc("/api/v1/session/status", sessionHandler.Status)

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
