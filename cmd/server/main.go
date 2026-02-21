// SP01PH05: Security & Rate Limiting - Security headers middleware

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/amaranth494/MudPuppy/internal/auth"
	"github.com/amaranth494/MudPuppy/internal/config"
	"github.com/amaranth494/MudPuppy/internal/redis"
	"github.com/amaranth494/MudPuppy/internal/store"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
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

	// Create router with session middleware
	mux := http.NewServeMux()

	// Public endpoints
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/api/v1/register", authHandler.Register)
	mux.HandleFunc("/api/v1/send-otp", authHandler.SendOTP)
	mux.HandleFunc("/api/v1/login", authHandler.Login)
	mux.HandleFunc("/api/v1/logout", authHandler.Logout)
	mux.HandleFunc("/api/v1/me", authHandler.Me)

	// Serve static files from public directory
	mux.Handle("/", http.FileServer(http.Dir("./public")))

	// Protected endpoints - wrapped with session middleware
	protectedHandler := sessionMiddleware(redisClient, mux)

	// Add security headers middleware (SP01PH05T03)
	handler := securityHeadersMiddleware(protectedHandler)

	log.Printf("Server starting on port %s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, handler); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// sessionMiddleware validates session for protected routes
func sessionMiddleware(redisClient *redis.Client, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Route the request first to determine if it's a protected path
		path := r.URL.Path

		// Allow public endpoints through
		if path == "/health" || path == "/api/v1/register" || path == "/api/v1/login" {
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
			_, err := redisClient.GetSession(ctx, sessionToken)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Refresh session idle timer
			redisClient.RefreshSession(ctx, sessionToken)
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
