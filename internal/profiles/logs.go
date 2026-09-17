package profiles

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SessionSummaryResponse is one entry of the ListSessions response (D-17):
// a profile's session, listed by date and time.
type SessionSummaryResponse struct {
	ID        uuid.UUID `json:"id"`
	StartedAt string    `json:"started_at"`
	EndedAt   *string   `json:"ended_at"`
	LineCount int       `json:"line_count"`
}

// SessionsListResponse is the ListSessions response body.
type SessionsListResponse struct {
	Sessions []SessionSummaryResponse `json:"sessions"`
}

// SessionLineResponse is one line of a GetSessionTranscript response.
type SessionLineResponse struct {
	Seq    int64  `json:"seq"`
	Source string `json:"source"`
	Text   string `json:"text"`
}

// SessionTranscriptResponse is the GetSessionTranscript response body.
type SessionTranscriptResponse struct {
	SessionID uuid.UUID             `json:"session_id"`
	Lines     []SessionLineResponse `json:"lines"`
}

// ListSessions handles GET /api/v1/profiles/{connection_id}/sessions
// (D-17). Ownership is resolved through GetProfileByConnection before any
// session row is read — a caller never gets to claim ownership of a
// connection id it does not own (T-3-04).
func (h *Handler) ListSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.transcripts == nil {
		http.Error(w, "Session logging is not available", http.StatusServiceUnavailable)
		return
	}

	userUUID, profile, err := h.getProfileByConnectionID(r)
	if err != nil {
		h.sendError(w, err.Error())
		return
	}

	sessions, err := h.transcripts.ListSessionsForConnection(profile.ConnectionID, 0)
	if err != nil {
		log.Printf("[SP05PH07] List sessions failed: %v", err)
		h.sendError(w, "Failed to list sessions")
		return
	}

	resp := SessionsListResponse{Sessions: make([]SessionSummaryResponse, len(sessions))}
	for i, s := range sessions {
		summary := SessionSummaryResponse{
			ID:        s.ID,
			StartedAt: s.StartedAt.UTC().Format(time.RFC3339),
			LineCount: s.LineCount,
		}
		if s.EndedAt != nil {
			ended := s.EndedAt.UTC().Format(time.RFC3339)
			summary.EndedAt = &ended
		}
		resp.Sessions[i] = summary
	}

	log.Printf("[AI-PLAYER] transcript-read user_id=%s connection_id=%s game_session_id=%s outcome=%s lines=%d",
		userUUID, profile.ConnectionID, "", "listed", len(resp.Sessions))

	h.sendJSON(w, resp)
}

// GetSessionTranscript handles
// GET /api/v1/profiles/{connection_id}/sessions/{session_id} (D-17). The
// same ownership resolution as ListSessions runs first — via
// getProfileByConnectionID, which calls GetProfileByConnection — then
// GetSessionLines also filters on connection_id, so a session id that
// exists but belongs to another connection returns an empty lines array
// rather than another profile's text (defence in depth behind this
// handler's own check, T-3-04).
func (h *Handler) GetSessionTranscript(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.transcripts == nil {
		http.Error(w, "Session logging is not available", http.StatusServiceUnavailable)
		return
	}

	userUUID, profile, err := h.getProfileByConnectionID(r)
	if err != nil {
		h.sendError(w, err.Error())
		return
	}

	sessionID, err := h.getSessionIDFromPath(r)
	if err != nil {
		h.sendError(w, err.Error())
		return
	}

	lines, err := h.transcripts.GetSessionLines(sessionID, profile.ConnectionID, 0)
	if err != nil {
		log.Printf("[SP05PH07] Get session transcript failed: %v", err)
		h.sendError(w, "Failed to get session transcript")
		return
	}

	resp := SessionTranscriptResponse{SessionID: sessionID, Lines: make([]SessionLineResponse, len(lines))}
	for i, l := range lines {
		resp.Lines[i] = SessionLineResponse{Seq: l.Seq, Source: l.Source, Text: l.Text}
	}

	outcome := "read"
	if len(lines) == 0 {
		outcome = "empty"
	}
	log.Printf("[AI-PLAYER] transcript-read user_id=%s connection_id=%s game_session_id=%s outcome=%s lines=%d",
		userUUID, profile.ConnectionID, sessionID, outcome, len(resp.Lines))

	h.sendJSON(w, resp)
}

// GetSessionConversation handles
// GET /api/v1/profiles/{connection_id}/sessions/{session_id}/conversation
// (D-04, D-27, plan 05-09). The Logs page's own section, requested
// separately from the transcript so the two stay visually and structurally
// separate (D-04: "one panel, two views" carried into the read-only
// record). Ownership resolution mirrors GetSessionTranscript exactly: the
// same getProfileByConnectionID check runs first, then
// ConversationForSession also filters on connection_id, so a session id
// belonging to another connection returns an empty lines array (T-3-04,
// T-5-42).
func (h *Handler) GetSessionConversation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.conversation == nil {
		http.Error(w, "Conversation unavailable", http.StatusServiceUnavailable)
		return
	}

	_, profile, err := h.getProfileByConnectionID(r)
	if err != nil {
		h.sendError(w, err.Error())
		return
	}

	sessionID, err := h.getSessionIDFromPath(r)
	if err != nil {
		h.sendError(w, err.Error())
		return
	}

	lines, err := h.conversation.ConversationForSession(sessionID, profile.ConnectionID)
	if err != nil {
		log.Printf("[SP05PH09] Get session conversation failed: %v", err)
		h.sendError(w, "Failed to get session conversation")
		return
	}

	resp := ConversationResponse{Lines: make([]ConversationLineResponse, len(lines))}
	for i, l := range lines {
		resp.Lines[i] = ConversationLineResponse{
			ID:        l.ID,
			Speaker:   l.Speaker,
			Text:      l.Text,
			Timestamp: l.CreatedAt.UTC().Format(time.RFC3339),
		}
	}

	h.sendJSON(w, resp)
}

// getSessionIDFromPath extracts the session id from
// /api/v1/profiles/{connection_id}/sessions/{session_id} — and, since it
// only ever reads parts[6], this same helper also works unmodified for
// /api/v1/profiles/{connection_id}/sessions/{session_id}/conversation
// (one more path segment after session_id changes nothing at index 6).
func (h *Handler) getSessionIDFromPath(r *http.Request) (uuid.UUID, error) {
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 7 {
		return uuid.Nil, &ValidationError{Message: "Session ID not found"}
	}
	return uuid.Parse(parts[6])
}
