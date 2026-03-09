package help

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// HelpSection represents a help content section
type HelpSection struct {
	Slug        string    `json:"slug"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Sections    []Section `json:"sections"`
}

// Section represents a subsection within a help topic
type Section struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// HelpSummary represents a summary of a help section for listing
type HelpSummary struct {
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// Handler handles help HTTP requests
type Handler struct {
	helpDir  string
	sections map[string]HelpSection
}

// NewHandler creates a new help handler
func NewHandler(helpDir string) *Handler {
	h := &Handler{
		helpDir:  helpDir,
		sections: make(map[string]HelpSection),
	}

	// Load all help content files
	if err := h.loadHelpContent(); err != nil {
		log.Printf("Warning: Failed to load help content: %v", err)
	}

	return h
}

// loadHelpContent loads all JSON files from the help directory
func (h *Handler) loadHelpContent() error {
	// Default to ./help if not provided
	if h.helpDir == "" {
		h.helpDir = "./help"
	}

	// Check if directory exists
	if _, err := os.Stat(h.helpDir); os.IsNotExist(err) {
		log.Printf("Help directory does not exist: %s", h.helpDir)
		return err
	}

	// Read all JSON files in the help directory
	files, err := os.ReadDir(h.helpDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		filename := filepath.Join(h.helpDir, file.Name())
		data, err := os.ReadFile(filename)
		if err != nil {
			log.Printf("Warning: Failed to read help file %s: %v", filename, err)
			continue
		}

		var section HelpSection
		if err := json.Unmarshal(data, &section); err != nil {
			log.Printf("Warning: Failed to parse help file %s: %v", filename, err)
			continue
		}

		h.sections[section.Slug] = section
		log.Printf("Loaded help section: %s (%s)", section.Title, section.Slug)
	}

	log.Printf("Loaded %d help sections", len(h.sections))
	return nil
}

// GetAll returns all help sections as summaries
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	summaries := make([]HelpSummary, 0, len(h.sections))

	for _, section := range h.sections {
		summaries = append(summaries, HelpSummary{
			Slug:        section.Slug,
			Title:       section.Title,
			Description: section.Description,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(summaries); err != nil {
		log.Printf("Error encoding help summaries: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// GetBySlug returns a specific help section by slug
func (h *Handler) GetBySlug(w http.ResponseWriter, r *http.Request) {
	// Extract slug from URL path - /api/v1/help/:slug
	parts := strings.Split(r.URL.Path, "/")
	slug := parts[len(parts)-1]

	if slug == "" || slug == "help" {
		// If no slug provided, return all
		h.GetAll(w, r)
		return
	}

	section, exists := h.sections[slug]
	if !exists {
		http.Error(w, "Help section not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(section); err != nil {
		log.Printf("Error encoding help section: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// HandlerFunc returns an http.HandlerFunc for the help handler
func (h *Handler) HandlerFunc() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			// Check if this is a specific slug request
			if strings.Contains(r.URL.Path, "/help/") && !strings.HasSuffix(r.URL.Path, "/help") {
				h.GetBySlug(w, r)
				return
			}

			// Otherwise return all help sections
			h.GetAll(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}
