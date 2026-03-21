package icm

import (
	"encoding/json"
	"net/http"
	"sync"
)

// Handler handles ICM HTTP requests
type Handler struct {
	engine *Engine
	mu     sync.RWMutex
}

// NewHandler creates a new ICM HTTP handler
func NewHandler() *Handler {
	return &Handler{
		engine: NewEngine(),
	}
}

// RegisterRoutes registers ICM routes with the http.ServeMux
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/icm/validate", h.handleValidate)
	mux.HandleFunc("/api/v1/icm/normalize", h.handleNormalize)
	mux.HandleFunc("/api/v1/icm/execute", h.handleExecute)
	mux.HandleFunc("/api/v1/icm/preview", h.handlePreview)
}

// handleValidate validates a command without execution
func (h *Handler) handleValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeMethodNotAllowed(w, "POST")
		return
	}

	var req ICMRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	req.Context = ContextPreview
	resp := h.engine.Process(&req)
	h.writeResponse(w, resp)
}

// handleNormalize normalizes a command without execution
func (h *Handler) handleNormalize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeMethodNotAllowed(w, "POST")
		return
	}

	var req ICMRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	req.Context = ContextPreview
	resp := h.engine.Process(&req)
	h.writeResponse(w, resp)
}

// handleExecute executes a command with full processing
func (h *Handler) handleExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeMethodNotAllowed(w, "POST")
		return
	}

	var req ICMRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Set default context to submission if not specified
	if req.Context == "" {
		req.Context = ContextSubmission
	}

	resp := h.engine.Process(&req)
	h.writeResponse(w, resp)
}

// handlePreview previews a command without execution
func (h *Handler) handlePreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeMethodNotAllowed(w, "POST")
		return
	}

	var req ICMRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	req.Context = ContextPreview
	resp := h.engine.Process(&req)
	h.writeResponse(w, resp)
}

// writeResponse writes a JSON response
func (h *Handler) writeResponse(w http.ResponseWriter, resp *ICMResponse) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.writeError(w, http.StatusInternalServerError, "Failed to encode response")
	}
}

// writeError writes an error response
func (h *Handler) writeError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	resp := map[string]interface{}{
		"error": map[string]string{
			"code":        "E5000",
			"message":     message,
			"userMessage": message,
		},
	}
	json.NewEncoder(w).Encode(resp)
}

// writeMethodNotAllowed writes a method not allowed error
func (h *Handler) writeMethodNotAllowed(w http.ResponseWriter, allowed string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Allow", allowed)
	w.WriteHeader(http.StatusMethodNotAllowed)
	resp := map[string]interface{}{
		"error": map[string]string{
			"code":        "E5000",
			"message":     "Method not allowed",
			"userMessage": "Method not allowed. Use " + allowed,
		},
	}
	json.NewEncoder(w).Encode(resp)
}

// GetEngine returns the ICM engine
func (h *Handler) GetEngine() *Engine {
	return h.engine
}

// SetEngine sets a custom ICM engine
func (h *Handler) SetEngine(engine *Engine) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.engine = engine
}
