package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

type HealthResponse struct {
	Status string `json:"status"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := HealthResponse{Status: "ok"}
	json.NewEncoder(w).Encode(response)
}

func debugHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	port := os.Getenv("PORT")
	if port == "" {
		port = "(not set)"
	}
	json.NewEncoder(w).Encode(map[string]string{
		"PORT":    port,
		"DATABASE_URL": os.Getenv("DATABASE_URL"),
		"REDIS_URL":   os.Getenv("REDIS_URL"),
	})
}

func main() {
	// Read database configuration from environment
	databaseURL := os.Getenv("DATABASE_URL")
	redisURL := os.Getenv("REDIS_URL")

	// Connect to PostgreSQL
	if databaseURL == "" {
		log.Println("WARNING: DATABASE_URL not set, skipping PostgreSQL connection")
	} else {
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
	}

	// Connect to Redis
	if redisURL == "" {
		log.Println("WARNING: REDIS_URL not set, skipping Redis connection")
	} else {
		log.Println("Connecting to Redis...")
		opt, err := redis.ParseURL(redisURL)
		if err != nil {
			log.Fatalf("Failed to parse Redis URL: %v", err)
		}

		client := redis.NewClient(opt)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := client.Ping(ctx).Err(); err != nil {
			log.Fatalf("Failed to connect to Redis: %v", err)
		}
		log.Println("Connected to Redis")
		defer client.Close()
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/debug", debugHandler)

	// Serve static frontend files
	fs := http.FileServer(http.Dir("public"))
	http.Handle("/", fs)

	log.Printf("PORT env: %s", os.Getenv("PORT"))
	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
