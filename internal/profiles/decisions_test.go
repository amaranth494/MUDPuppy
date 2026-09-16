package profiles

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/amaranth494/MudPuppy/internal/store"
	"github.com/google/uuid"
)

// fakeDecisionStore is a hand-written fake satisfying decisionsStorage,
// used so these tests exercise the handler without a live Postgres
// connection — same discipline as fakeProfileStore (handler_test.go) and
// fakeTranscriptStore (logs_test.go).
type fakeDecisionStore struct {
	decisionsByConnection map[uuid.UUID][]store.Decision
	// lastLimit records the limit ListForConnection was called with, so
	// respects_the_limit can assert what the handler actually passed
	// through rather than re-implementing clamping in the fake.
	lastLimit int
}

func (f *fakeDecisionStore) ListForConnection(connectionID uuid.UUID, limit int) ([]store.Decision, error) {
	f.lastLimit = limit
	all := f.decisionsByConnection[connectionID]
	if limit > 0 && limit < len(all) {
		return all[:limit], nil
	}
	return all, nil
}

func TestDecisionsReload(t *testing.T) {
	t.Run("returns_decisions_oldest_first", func(t *testing.T) {
		userID := uuid.New()
		profileID := uuid.New()
		connectionID := uuid.New()
		profileFake := &fakeProfileStore{profile: newTestProfile(userID, profileID, connectionID)}

		first := store.Decision{
			ID: uuid.New(), ConnectionID: connectionID,
			Reasoning: "Heading north looked safest.", Command: "north",
			Outcome: "sent", CreatedAt: time.Date(2026, 1, 2, 3, 0, 0, 0, time.UTC),
		}
		second := store.Decision{
			ID: uuid.New(), ConnectionID: connectionID,
			Reasoning: "Resting to recover.", Command: "rest",
			Outcome: "sent", CreatedAt: time.Date(2026, 1, 2, 3, 5, 0, 0, time.UTC),
		}
		decisionsFake := &fakeDecisionStore{
			decisionsByConnection: map[uuid.UUID][]store.Decision{
				connectionID: {first, second}, // already oldest-first, as ListForConnection guarantees
			},
		}
		h := &Handler{profileStore: profileFake, decisions: decisionsFake}

		path := "/api/v1/profiles/" + connectionID.String() + "/decisions"
		req := newTestRequest(http.MethodGet, path, userID, nil)
		rec := httptest.NewRecorder()
		h.GetDecisions(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}

		var got DecisionsListResponse
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(got.Decisions) != 2 {
			t.Fatalf("Decisions = %+v, want 2", got.Decisions)
		}
		if got.Decisions[0].Command != "north" || got.Decisions[1].Command != "rest" {
			t.Errorf("Decisions = %+v, want north then rest (oldest first)", got.Decisions)
		}
		if got.Decisions[0].ID != first.ID.String() {
			t.Errorf("Decisions[0].ID = %s, want %s", got.Decisions[0].ID, first.ID.String())
		}
	})

	t.Run("not_owned_connection_is_refused", func(t *testing.T) {
		userID := uuid.New()
		profileID := uuid.New()
		connectionID := uuid.New()
		otherConnectionID := uuid.New()
		profileFake := &fakeProfileStore{profile: newTestProfile(userID, profileID, connectionID)}
		decisionsFake := &fakeDecisionStore{}
		h := &Handler{profileStore: profileFake, decisions: decisionsFake}

		// The signed-in user's profile is for connectionID, not
		// otherConnectionID — GetProfileByConnection must find nothing.
		path := "/api/v1/profiles/" + otherConnectionID.String() + "/decisions"
		req := newTestRequest(http.MethodGet, path, userID, nil)
		rec := httptest.NewRecorder()
		h.GetDecisions(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, body = %s, want 400", rec.Code, rec.Body.String())
		}
	})

	t.Run("respects_the_limit", func(t *testing.T) {
		userID := uuid.New()
		profileID := uuid.New()
		connectionID := uuid.New()
		profileFake := &fakeProfileStore{profile: newTestProfile(userID, profileID, connectionID)}

		many := make([]store.Decision, 0, 300)
		for i := 0; i < 300; i++ {
			many = append(many, store.Decision{
				ID: uuid.New(), ConnectionID: connectionID, Command: "look", Outcome: "sent",
				CreatedAt: time.Date(2026, 1, 2, 3, 0, 0, 0, time.UTC).Add(time.Duration(i) * time.Second),
			})
		}
		decisionsFake := &fakeDecisionStore{decisionsByConnection: map[uuid.UUID][]store.Decision{connectionID: many}}
		h := &Handler{profileStore: profileFake, decisions: decisionsFake}

		path := "/api/v1/profiles/" + connectionID.String() + "/decisions?limit=999"
		req := newTestRequest(http.MethodGet, path, userID, nil)
		rec := httptest.NewRecorder()
		h.GetDecisions(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}
		if decisionsFake.lastLimit != maxDecisionsLimit {
			t.Errorf("limit passed to the store = %d, want clamp to %d", decisionsFake.lastLimit, maxDecisionsLimit)
		}

		// No ?limit= at all defaults to 50.
		path = "/api/v1/profiles/" + connectionID.String() + "/decisions"
		req = newTestRequest(http.MethodGet, path, userID, nil)
		rec = httptest.NewRecorder()
		h.GetDecisions(rec, req)
		if decisionsFake.lastLimit != defaultDecisionsLimit {
			t.Errorf("default limit passed to the store = %d, want %d", decisionsFake.lastLimit, defaultDecisionsLimit)
		}
	})

	t.Run("never_returns_window_text", func(t *testing.T) {
		userID := uuid.New()
		profileID := uuid.New()
		connectionID := uuid.New()
		profileFake := &fakeProfileStore{profile: newTestProfile(userID, profileID, connectionID)}

		secretText := "SECRET-GAME-TEXT-NEVER-SHIPPED-TO-BROWSER"
		decisionsFake := &fakeDecisionStore{
			decisionsByConnection: map[uuid.UUID][]store.Decision{
				connectionID: {{
					ID: uuid.New(), ConnectionID: connectionID,
					WindowText: secretText, Reasoning: "why", Command: "north", Outcome: "sent",
					CreatedAt: time.Now(),
				}},
			},
		}
		h := &Handler{profileStore: profileFake, decisions: decisionsFake}

		path := "/api/v1/profiles/" + connectionID.String() + "/decisions"
		req := newTestRequest(http.MethodGet, path, userID, nil)
		rec := httptest.NewRecorder()
		h.GetDecisions(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}
		if got := rec.Body.String(); strings.Contains(got, secretText) {
			t.Fatalf("response body contains the game text window, want it kept server-side: %s", got)
		}
	})
}
