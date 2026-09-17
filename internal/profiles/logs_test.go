package profiles

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/amaranth494/MudPuppy/internal/store"
	"github.com/google/uuid"
)

// fakeTranscriptStore is a hand-written fake satisfying transcriptStorage,
// used so these tests exercise the handler without a live Postgres
// connection — same discipline as fakeProfileStore in handler_test.go.
// GetSessionLines mimics store.TranscriptStore's own defence-in-depth
// join: a session id whose recorded owner does not match connectionID
// returns an empty slice rather than an error (T-3-04).
type fakeTranscriptStore struct {
	sessionsByConnection map[uuid.UUID][]store.GameSessionSummary
	linesBySession       map[uuid.UUID][]store.GameSessionLine
	sessionOwner         map[uuid.UUID]uuid.UUID // gameSessionID -> connection_id
	// sessionMemoryByConnection backs GetSessionMemory (D-10, plan 04-08) —
	// same discipline as sessionsByConnection above.
	sessionMemoryByConnection map[uuid.UUID][]string
}

func (f *fakeTranscriptStore) ListSessionsForConnection(connectionID uuid.UUID, limit int) ([]store.GameSessionSummary, error) {
	return f.sessionsByConnection[connectionID], nil
}

func (f *fakeTranscriptStore) GetSessionLines(gameSessionID, connectionID uuid.UUID, limit int) ([]store.GameSessionLine, error) {
	if owner, ok := f.sessionOwner[gameSessionID]; !ok || owner != connectionID {
		return []store.GameSessionLine{}, nil
	}
	return f.linesBySession[gameSessionID], nil
}

func (f *fakeTranscriptStore) SessionMemoryForConnection(connectionID uuid.UUID) ([]string, error) {
	return f.sessionMemoryByConnection[connectionID], nil
}

func TestListSessionsScopedToOwner(t *testing.T) {
	t.Run("owner_sees_their_sessions", func(t *testing.T) {
		userID := uuid.New()
		profileID := uuid.New()
		connectionID := uuid.New()
		profileFake := &fakeProfileStore{profile: newTestProfile(userID, profileID, connectionID)}

		sessionID := uuid.New()
		started := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
		transcriptsFake := &fakeTranscriptStore{
			sessionsByConnection: map[uuid.UUID][]store.GameSessionSummary{
				connectionID: {{ID: sessionID, StartedAt: started, LineCount: 5}},
			},
		}
		h := &Handler{profileStore: profileFake, transcripts: transcriptsFake}

		path := "/api/v1/profiles/" + connectionID.String() + "/sessions"
		req := newTestRequest(http.MethodGet, path, userID, nil)
		rec := httptest.NewRecorder()
		h.ListSessions(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}

		var got SessionsListResponse
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(got.Sessions) != 1 {
			t.Fatalf("Sessions = %+v, want 1", got.Sessions)
		}
		if got.Sessions[0].ID != sessionID {
			t.Errorf("Sessions[0].ID = %v, want %v", got.Sessions[0].ID, sessionID)
		}
		if got.Sessions[0].LineCount != 5 {
			t.Errorf("Sessions[0].LineCount = %d, want 5", got.Sessions[0].LineCount)
		}
	})

	t.Run("not_owned_connection_is_refused", func(t *testing.T) {
		userID := uuid.New()
		profileID := uuid.New()
		connectionID := uuid.New()
		otherConnectionID := uuid.New()
		profileFake := &fakeProfileStore{profile: newTestProfile(userID, profileID, connectionID)}
		transcriptsFake := &fakeTranscriptStore{}
		h := &Handler{profileStore: profileFake, transcripts: transcriptsFake}

		// The signed-in user's profile is for connectionID, not
		// otherConnectionID — GetProfileByConnection must find nothing.
		path := "/api/v1/profiles/" + otherConnectionID.String() + "/sessions"
		req := newTestRequest(http.MethodGet, path, userID, nil)
		rec := httptest.NewRecorder()
		h.ListSessions(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, body = %s, want 400", rec.Code, rec.Body.String())
		}
	})
}

func TestGetSessionTranscript(t *testing.T) {
	t.Run("returns_lines_in_sequence_order", func(t *testing.T) {
		userID := uuid.New()
		profileID := uuid.New()
		connectionID := uuid.New()
		sessionID := uuid.New()
		profileFake := &fakeProfileStore{profile: newTestProfile(userID, profileID, connectionID)}
		transcriptsFake := &fakeTranscriptStore{
			sessionOwner: map[uuid.UUID]uuid.UUID{sessionID: connectionID},
			linesBySession: map[uuid.UUID][]store.GameSessionLine{
				sessionID: {
					{Seq: 1, Source: "human", Text: "look"},
					{Seq: 2, Source: "game", Text: "You see a room."},
				},
			},
		}
		h := &Handler{profileStore: profileFake, transcripts: transcriptsFake}

		path := "/api/v1/profiles/" + connectionID.String() + "/sessions/" + sessionID.String()
		req := newTestRequest(http.MethodGet, path, userID, nil)
		rec := httptest.NewRecorder()
		h.GetSessionTranscript(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}

		var got SessionTranscriptResponse
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(got.Lines) != 2 {
			t.Fatalf("Lines = %+v, want 2", got.Lines)
		}
		if got.Lines[0].Seq != 1 || got.Lines[0].Source != "human" {
			t.Errorf("Lines[0] = %+v", got.Lines[0])
		}
		if got.Lines[1].Seq != 2 || got.Lines[1].Source != "game" {
			t.Errorf("Lines[1] = %+v", got.Lines[1])
		}
	})

	t.Run("not_owned_session_is_refused", func(t *testing.T) {
		userID := uuid.New()
		profileID := uuid.New()
		connectionID := uuid.New()
		otherConnectionID := uuid.New()
		sessionID := uuid.New()
		profileFake := &fakeProfileStore{profile: newTestProfile(userID, profileID, connectionID)}
		transcriptsFake := &fakeTranscriptStore{
			// sessionID actually belongs to a different connection; the
			// caller owns connectionID but is guessing at sessionID.
			sessionOwner: map[uuid.UUID]uuid.UUID{sessionID: otherConnectionID},
			linesBySession: map[uuid.UUID][]store.GameSessionLine{
				sessionID: {{Seq: 1, Source: "human", Text: "secret"}},
			},
		}
		h := &Handler{profileStore: profileFake, transcripts: transcriptsFake}

		path := "/api/v1/profiles/" + connectionID.String() + "/sessions/" + sessionID.String()
		req := newTestRequest(http.MethodGet, path, userID, nil)
		rec := httptest.NewRecorder()
		h.GetSessionTranscript(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}

		var got SessionTranscriptResponse
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(got.Lines) != 0 {
			t.Fatalf("Lines = %+v, want empty — a session belonging to another connection must never leak its text", got.Lines)
		}
	})
}

// TestGetSessionMemory proves D-10's read-only endpoint (plan 04-08):
// stored bullets round-trip, a connection with nothing stored yet answers
// with an empty array rather than an error, a connection id the caller does
// not own is refused (T-4-09), and a handler with no transcripts store
// wired fails closed with 503 rather than panicking.
func TestGetSessionMemory(t *testing.T) {
	t.Run("returns_stored_bullets", func(t *testing.T) {
		userID := uuid.New()
		profileID := uuid.New()
		connectionID := uuid.New()
		profileFake := &fakeProfileStore{profile: newTestProfile(userID, profileID, connectionID)}
		transcriptsFake := &fakeTranscriptStore{
			sessionMemoryByConnection: map[uuid.UUID][]string{
				connectionID: {"spoke with captain reyes", "learned the bridge is out"},
			},
		}
		h := &Handler{profileStore: profileFake, transcripts: transcriptsFake}

		path := "/api/v1/profiles/" + connectionID.String() + "/ai-memory"
		req := newTestRequest(http.MethodGet, path, userID, nil)
		rec := httptest.NewRecorder()
		h.GetSessionMemory(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}
		var got SessionMemoryResponse
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(got.SessionMemory) != 2 || got.SessionMemory[0] != "spoke with captain reyes" {
			t.Fatalf("SessionMemory = %v, want the two stored bullets", got.SessionMemory)
		}
	})

	t.Run("no_open_game_session_returns_empty_array", func(t *testing.T) {
		userID := uuid.New()
		profileID := uuid.New()
		connectionID := uuid.New()
		profileFake := &fakeProfileStore{profile: newTestProfile(userID, profileID, connectionID)}
		transcriptsFake := &fakeTranscriptStore{}
		h := &Handler{profileStore: profileFake, transcripts: transcriptsFake}

		path := "/api/v1/profiles/" + connectionID.String() + "/ai-memory"
		req := newTestRequest(http.MethodGet, path, userID, nil)
		rec := httptest.NewRecorder()
		h.GetSessionMemory(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}
		var got SessionMemoryResponse
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.SessionMemory == nil || len(got.SessionMemory) != 0 {
			t.Fatalf("SessionMemory = %v, want an empty (never nil) array", got.SessionMemory)
		}
	})

	t.Run("not_owned_connection_is_refused", func(t *testing.T) {
		userID := uuid.New()
		profileID := uuid.New()
		connectionID := uuid.New()
		otherConnectionID := uuid.New()
		profileFake := &fakeProfileStore{profile: newTestProfile(userID, profileID, connectionID)}
		transcriptsFake := &fakeTranscriptStore{}
		h := &Handler{profileStore: profileFake, transcripts: transcriptsFake}

		path := "/api/v1/profiles/" + otherConnectionID.String() + "/ai-memory"
		req := newTestRequest(http.MethodGet, path, userID, nil)
		rec := httptest.NewRecorder()
		h.GetSessionMemory(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, body = %s, want 400", rec.Code, rec.Body.String())
		}
	})

	t.Run("no_transcripts_store_wired_fails_closed", func(t *testing.T) {
		userID := uuid.New()
		profileID := uuid.New()
		connectionID := uuid.New()
		profileFake := &fakeProfileStore{profile: newTestProfile(userID, profileID, connectionID)}
		h := &Handler{profileStore: profileFake}

		path := "/api/v1/profiles/" + connectionID.String() + "/ai-memory"
		req := newTestRequest(http.MethodGet, path, userID, nil)
		rec := httptest.NewRecorder()
		h.GetSessionMemory(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503", rec.Code)
		}
	})
}
