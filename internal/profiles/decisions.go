package profiles

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// defaultDecisionsLimit and maxDecisionsLimit bound the ?limit= query
// parameter (D-12): enough for this phase's one-decision-per-engage
// reality with room for Phase 4's loop, never unbounded.
const (
	defaultDecisionsLimit = 50
	maxDecisionsLimit     = 200
)

// DecisionResponseItem is one entry of the GetDecisions response. The
// game text the model saw is deliberately omitted (T-3-15) — see
// GetDecisions' own doc comment.
type DecisionResponseItem struct {
	ID          string `json:"id"`
	CreatedAt   string `json:"created_at"`
	Reasoning   string `json:"reasoning"`
	Command     string `json:"command"`
	Outcome     string `json:"outcome"`
	FailureKind string `json:"failure_kind"`
	Notice      string `json:"notice"`
}

// DecisionsListResponse is the GetDecisions response body.
type DecisionsListResponse struct {
	Decisions []DecisionResponseItem `json:"decisions"`
}

// GetDecisions handles GET /api/v1/profiles/{connection_id}/decisions
// (D-12). This is how a reloaded play screen rebuilds the connection's
// decision history: the driver already pushes each decision live over the
// socket as it happens, and this endpoint answers the one-time fetch on
// mount that recovers whatever a closed or refreshed tab missed.
//
// The recent-game-text snapshot the model saw is never returned here. It
// exists so the owner can audit what the model was shown, and putting it
// on an endpoint the play screen calls on every mount would ship a
// screenful of other players' speech to the browser on every page load
// for no benefit this response's caller uses — the panel renders only the
// reasoning and the command (T-3-15).
func (h *Handler) GetDecisions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.decisions == nil {
		http.Error(w, "Decision history is not available", http.StatusServiceUnavailable)
		return
	}

	userIDVal := r.Context().Value("user_id")
	if userIDVal == nil {
		h.sendError(w, "Unauthorized")
		return
	}
	userIDStr, _ := userIDVal.(string)
	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.sendError(w, "Invalid user ID")
		return
	}

	connectionID, err := h.getConnectionIDFromPath(r)
	if err != nil {
		h.sendError(w, err.Error())
		return
	}

	// Ownership is resolved through the profile store before any decision
	// row is read — a caller can never read another owner's decisions by
	// guessing a connection id (T-3-03).
	profile, err := h.profileStore.GetProfileByConnection(userUUID, connectionID)
	if err != nil {
		log.Printf("[AI-PLAYER] decisions-read user_id=%s connection_id=%s outcome=%s count=%d",
			userUUID, connectionID, "refused", 0)
		h.sendError(w, "Failed to get profile")
		return
	}
	if profile == nil {
		log.Printf("[AI-PLAYER] decisions-read user_id=%s connection_id=%s outcome=%s count=%d",
			userUUID, connectionID, "refused", 0)
		h.sendError(w, "Profile not found")
		return
	}

	limit := defaultDecisionsLimit
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, perr := strconv.Atoi(raw); perr == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > maxDecisionsLimit {
		limit = maxDecisionsLimit
	}

	decisions, err := h.decisions.ListForConnection(profile.ConnectionID, limit)
	if err != nil {
		log.Printf("[AI-PLAYER] decisions-read user_id=%s connection_id=%s outcome=%s count=%d",
			userUUID, profile.ConnectionID, "failed", 0)
		h.sendError(w, "Failed to get decisions")
		return
	}

	resp := DecisionsListResponse{Decisions: make([]DecisionResponseItem, len(decisions))}
	for i, d := range decisions {
		resp.Decisions[i] = DecisionResponseItem{
			ID:          d.ID.String(),
			CreatedAt:   d.CreatedAt.UTC().Format(time.RFC3339),
			Reasoning:   d.Reasoning,
			Command:     d.Command,
			Outcome:     d.Outcome,
			FailureKind: d.FailureKind,
			Notice:      d.Notice,
		}
	}

	log.Printf("[AI-PLAYER] decisions-read user_id=%s connection_id=%s outcome=%s count=%d",
		userUUID, profile.ConnectionID, "listed", len(resp.Decisions))

	h.sendJSON(w, resp)
}
