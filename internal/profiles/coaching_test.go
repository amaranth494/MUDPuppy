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

// fakeCoachingStore is a hand-written fake satisfying coachingStorage (plan
// 05-06, D-09), used so these tests exercise the handler without a live
// Postgres connection -- same discipline as fakeTranscriptStore.
type fakeCoachingStore struct {
	coachingByConnection map[uuid.UUID][]string
}

func (f *fakeCoachingStore) CoachingForConnection(connectionID uuid.UUID) ([]string, error) {
	return f.coachingByConnection[connectionID], nil
}

// fakeConversationStore is a hand-written fake satisfying
// conversationStorage (plan 05-06/05-09, D-09/D-04). linesBySession is keyed
// by (gameSessionID, connectionID) so a test can prove a session id from
// another connection is refused, mirroring the real store's defence-in-depth
// JOIN.
type fakeConversationStore struct {
	linesByConnection map[uuid.UUID][]store.ConversationLine
	linesBySession     map[[2]uuid.UUID][]store.ConversationLine
}

func (f *fakeConversationStore) ConversationFor(connectionID uuid.UUID) ([]store.ConversationLine, error) {
	return f.linesByConnection[connectionID], nil
}

func (f *fakeConversationStore) ConversationForSession(gameSessionID, connectionID uuid.UUID) ([]store.ConversationLine, error) {
	return f.linesBySession[[2]uuid.UUID{gameSessionID, connectionID}], nil
}

// TestGetCoaching proves D-09's read-only endpoint: stored coaching
// round-trips, a connection with nothing stored yet answers with an empty
// array rather than an error, a connection id the caller does not own is
// refused (T-4-09), and a handler with no coaching store wired fails closed
// with 503 rather than panicking. There is deliberately no PUT test to
// match -- the coaching store's only writer anywhere in the codebase is
// internal/driver's HandleChat (D-18).
func TestGetCoaching(t *testing.T) {
	t.Run("returns_stored_coaching", func(t *testing.T) {
		userID := uuid.New()
		profileID := uuid.New()
		connectionID := uuid.New()
		profileFake := &fakeProfileStore{profile: newTestProfile(userID, profileID, connectionID)}
		coachingFake := &fakeCoachingStore{
			coachingByConnection: map[uuid.UUID][]string{
				connectionID: {"avoid the north road", "always check inventory first"},
			},
		}
		h := &Handler{profileStore: profileFake, coaching: coachingFake}

		path := "/api/v1/profiles/" + connectionID.String() + "/ai-coaching"
		req := newTestRequest(http.MethodGet, path, userID, nil)
		rec := httptest.NewRecorder()
		h.GetCoaching(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}
		var got CoachingResponse
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(got.Coaching) != 2 || got.Coaching[0] != "avoid the north road" {
			t.Fatalf("Coaching = %v, want the two stored lines", got.Coaching)
		}
	})

	t.Run("nothing_stored_returns_empty_array", func(t *testing.T) {
		userID := uuid.New()
		profileID := uuid.New()
		connectionID := uuid.New()
		profileFake := &fakeProfileStore{profile: newTestProfile(userID, profileID, connectionID)}
		h := &Handler{profileStore: profileFake, coaching: &fakeCoachingStore{}}

		path := "/api/v1/profiles/" + connectionID.String() + "/ai-coaching"
		req := newTestRequest(http.MethodGet, path, userID, nil)
		rec := httptest.NewRecorder()
		h.GetCoaching(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}
		var got CoachingResponse
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.Coaching == nil || len(got.Coaching) != 0 {
			t.Fatalf("Coaching = %v, want an empty (never nil) array", got.Coaching)
		}
	})

	t.Run("not_owned_connection_is_refused", func(t *testing.T) {
		userID := uuid.New()
		profileID := uuid.New()
		connectionID := uuid.New()
		otherConnectionID := uuid.New()
		profileFake := &fakeProfileStore{profile: newTestProfile(userID, profileID, connectionID)}
		h := &Handler{profileStore: profileFake, coaching: &fakeCoachingStore{}}

		path := "/api/v1/profiles/" + otherConnectionID.String() + "/ai-coaching"
		req := newTestRequest(http.MethodGet, path, userID, nil)
		rec := httptest.NewRecorder()
		h.GetCoaching(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, body = %s, want 400", rec.Code, rec.Body.String())
		}
	})

	t.Run("no_coaching_store_wired_fails_closed", func(t *testing.T) {
		userID := uuid.New()
		profileID := uuid.New()
		connectionID := uuid.New()
		profileFake := &fakeProfileStore{profile: newTestProfile(userID, profileID, connectionID)}
		h := &Handler{profileStore: profileFake}

		path := "/api/v1/profiles/" + connectionID.String() + "/ai-coaching"
		req := newTestRequest(http.MethodGet, path, userID, nil)
		rec := httptest.NewRecorder()
		h.GetCoaching(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503", rec.Code)
		}
	})

	t.Run("method_not_allowed", func(t *testing.T) {
		userID := uuid.New()
		profileID := uuid.New()
		connectionID := uuid.New()
		profileFake := &fakeProfileStore{profile: newTestProfile(userID, profileID, connectionID)}
		h := &Handler{profileStore: profileFake, coaching: &fakeCoachingStore{}}

		path := "/api/v1/profiles/" + connectionID.String() + "/ai-coaching"
		req := newTestRequest(http.MethodPut, path, userID, nil)
		rec := httptest.NewRecorder()
		h.GetCoaching(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want 405", rec.Code)
		}
	})
}

// TestGetConversation proves D-09's read-only endpoint: stored lines
// round-trip in order with the fields the panel needs, a connection with no
// conversation yet answers with an empty array, a connection id the caller
// does not own is refused, and a handler with no conversation store wired
// fails closed with 503.
func TestGetConversation(t *testing.T) {
	t.Run("returns_stored_lines", func(t *testing.T) {
		userID := uuid.New()
		profileID := uuid.New()
		connectionID := uuid.New()
		profileFake := &fakeProfileStore{profile: newTestProfile(userID, profileID, connectionID)}
		when := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
		conversationFake := &fakeConversationStore{
			linesByConnection: map[uuid.UUID][]store.ConversationLine{
				connectionID: {
					{ID: 1, Seq: 1, Speaker: "owner", Text: "why did you go north?", CreatedAt: when},
					{ID: 2, Seq: 2, Speaker: "chatter", Text: "the door was open.", CreatedAt: when},
				},
			},
		}
		h := &Handler{profileStore: profileFake, conversation: conversationFake}

		path := "/api/v1/profiles/" + connectionID.String() + "/ai-conversation"
		req := newTestRequest(http.MethodGet, path, userID, nil)
		rec := httptest.NewRecorder()
		h.GetConversation(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}
		var got ConversationResponse
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(got.Lines) != 2 {
			t.Fatalf("Lines = %+v, want 2", got.Lines)
		}
		if got.Lines[0].Speaker != "owner" || got.Lines[1].Speaker != "chatter" {
			t.Fatalf("Lines = %+v, want owner then chatter", got.Lines)
		}
	})

	t.Run("nothing_stored_returns_empty_array", func(t *testing.T) {
		userID := uuid.New()
		profileID := uuid.New()
		connectionID := uuid.New()
		profileFake := &fakeProfileStore{profile: newTestProfile(userID, profileID, connectionID)}
		h := &Handler{profileStore: profileFake, conversation: &fakeConversationStore{}}

		path := "/api/v1/profiles/" + connectionID.String() + "/ai-conversation"
		req := newTestRequest(http.MethodGet, path, userID, nil)
		rec := httptest.NewRecorder()
		h.GetConversation(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}
		var got ConversationResponse
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.Lines == nil || len(got.Lines) != 0 {
			t.Fatalf("Lines = %v, want an empty (never nil) array", got.Lines)
		}
	})

	t.Run("not_owned_connection_is_refused", func(t *testing.T) {
		userID := uuid.New()
		profileID := uuid.New()
		connectionID := uuid.New()
		otherConnectionID := uuid.New()
		profileFake := &fakeProfileStore{profile: newTestProfile(userID, profileID, connectionID)}
		h := &Handler{profileStore: profileFake, conversation: &fakeConversationStore{}}

		path := "/api/v1/profiles/" + otherConnectionID.String() + "/ai-conversation"
		req := newTestRequest(http.MethodGet, path, userID, nil)
		rec := httptest.NewRecorder()
		h.GetConversation(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, body = %s, want 400", rec.Code, rec.Body.String())
		}
	})

	t.Run("no_conversation_store_wired_fails_closed", func(t *testing.T) {
		userID := uuid.New()
		profileID := uuid.New()
		connectionID := uuid.New()
		profileFake := &fakeProfileStore{profile: newTestProfile(userID, profileID, connectionID)}
		h := &Handler{profileStore: profileFake}

		path := "/api/v1/profiles/" + connectionID.String() + "/ai-conversation"
		req := newTestRequest(http.MethodGet, path, userID, nil)
		rec := httptest.NewRecorder()
		h.GetConversation(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503", rec.Code)
		}
	})

	t.Run("method_not_allowed", func(t *testing.T) {
		userID := uuid.New()
		profileID := uuid.New()
		connectionID := uuid.New()
		profileFake := &fakeProfileStore{profile: newTestProfile(userID, profileID, connectionID)}
		h := &Handler{profileStore: profileFake, conversation: &fakeConversationStore{}}

		path := "/api/v1/profiles/" + connectionID.String() + "/ai-conversation"
		req := newTestRequest(http.MethodPost, path, userID, nil)
		rec := httptest.NewRecorder()
		h.GetConversation(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want 405", rec.Code)
		}
	})
}

// TestGetSessionConversation proves the Logs page's own per-session read
// (D-04, D-27, plan 05-09): stored lines for the selected session round-trip
// in order, a session with nothing stored yet answers with an empty array,
// a connection id the caller does not own is refused, a session id from
// another connection returns no lines (T-3-04, T-5-42 defence in depth),
// and a handler with no conversation store wired fails closed with 503.
func TestGetSessionConversation(t *testing.T) {
	t.Run("returns_stored_lines_for_the_selected_session", func(t *testing.T) {
		userID := uuid.New()
		profileID := uuid.New()
		connectionID := uuid.New()
		sessionID := uuid.New()
		profileFake := &fakeProfileStore{profile: newTestProfile(userID, profileID, connectionID)}
		when := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
		conversationFake := &fakeConversationStore{
			linesBySession: map[[2]uuid.UUID][]store.ConversationLine{
				{sessionID, connectionID}: {
					{ID: 1, Seq: 1, Speaker: "owner", Text: "why did you go north?", CreatedAt: when},
					{ID: 2, Seq: 2, Speaker: "chatter", Text: "the door was open.", CreatedAt: when},
				},
			},
		}
		h := &Handler{profileStore: profileFake, conversation: conversationFake}

		path := "/api/v1/profiles/" + connectionID.String() + "/sessions/" + sessionID.String() + "/conversation"
		req := newTestRequest(http.MethodGet, path, userID, nil)
		rec := httptest.NewRecorder()
		h.GetSessionConversation(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}
		var got ConversationResponse
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(got.Lines) != 2 {
			t.Fatalf("Lines = %+v, want 2", got.Lines)
		}
		if got.Lines[0].Speaker != "owner" || got.Lines[1].Speaker != "chatter" {
			t.Fatalf("Lines = %+v, want owner then chatter", got.Lines)
		}
	})

	t.Run("nothing_stored_returns_empty_array", func(t *testing.T) {
		userID := uuid.New()
		profileID := uuid.New()
		connectionID := uuid.New()
		sessionID := uuid.New()
		profileFake := &fakeProfileStore{profile: newTestProfile(userID, profileID, connectionID)}
		h := &Handler{profileStore: profileFake, conversation: &fakeConversationStore{}}

		path := "/api/v1/profiles/" + connectionID.String() + "/sessions/" + sessionID.String() + "/conversation"
		req := newTestRequest(http.MethodGet, path, userID, nil)
		rec := httptest.NewRecorder()
		h.GetSessionConversation(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}
		var got ConversationResponse
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.Lines == nil || len(got.Lines) != 0 {
			t.Fatalf("Lines = %v, want an empty (never nil) array", got.Lines)
		}
	})

	t.Run("not_owned_connection_is_refused", func(t *testing.T) {
		userID := uuid.New()
		profileID := uuid.New()
		connectionID := uuid.New()
		otherConnectionID := uuid.New()
		sessionID := uuid.New()
		profileFake := &fakeProfileStore{profile: newTestProfile(userID, profileID, connectionID)}
		h := &Handler{profileStore: profileFake, conversation: &fakeConversationStore{}}

		path := "/api/v1/profiles/" + otherConnectionID.String() + "/sessions/" + sessionID.String() + "/conversation"
		req := newTestRequest(http.MethodGet, path, userID, nil)
		rec := httptest.NewRecorder()
		h.GetSessionConversation(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, body = %s, want 400", rec.Code, rec.Body.String())
		}
	})

	t.Run("session_id_from_another_connection_returns_no_lines", func(t *testing.T) {
		userID := uuid.New()
		profileID := uuid.New()
		connectionID := uuid.New()
		otherConnectionID := uuid.New()
		sessionID := uuid.New()
		profileFake := &fakeProfileStore{profile: newTestProfile(userID, profileID, connectionID)}
		when := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
		// The fake stores this session's lines keyed to otherConnectionID --
		// the caller owns connectionID, not otherConnectionID, so the read
		// must come back empty even though the session id is correct
		// (T-3-04, T-5-42).
		conversationFake := &fakeConversationStore{
			linesBySession: map[[2]uuid.UUID][]store.ConversationLine{
				{sessionID, otherConnectionID}: {
					{ID: 1, Seq: 1, Speaker: "owner", Text: "another profile's line", CreatedAt: when},
				},
			},
		}
		h := &Handler{profileStore: profileFake, conversation: conversationFake}

		path := "/api/v1/profiles/" + connectionID.String() + "/sessions/" + sessionID.String() + "/conversation"
		req := newTestRequest(http.MethodGet, path, userID, nil)
		rec := httptest.NewRecorder()
		h.GetSessionConversation(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}
		var got ConversationResponse
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(got.Lines) != 0 {
			t.Fatalf("Lines = %+v, want empty (session belongs to another connection)", got.Lines)
		}
	})

	t.Run("no_conversation_store_wired_fails_closed", func(t *testing.T) {
		userID := uuid.New()
		profileID := uuid.New()
		connectionID := uuid.New()
		sessionID := uuid.New()
		profileFake := &fakeProfileStore{profile: newTestProfile(userID, profileID, connectionID)}
		h := &Handler{profileStore: profileFake}

		path := "/api/v1/profiles/" + connectionID.String() + "/sessions/" + sessionID.String() + "/conversation"
		req := newTestRequest(http.MethodGet, path, userID, nil)
		rec := httptest.NewRecorder()
		h.GetSessionConversation(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503", rec.Code)
		}
	})

	t.Run("method_not_allowed", func(t *testing.T) {
		userID := uuid.New()
		profileID := uuid.New()
		connectionID := uuid.New()
		sessionID := uuid.New()
		profileFake := &fakeProfileStore{profile: newTestProfile(userID, profileID, connectionID)}
		h := &Handler{profileStore: profileFake, conversation: &fakeConversationStore{}}

		path := "/api/v1/profiles/" + connectionID.String() + "/sessions/" + sessionID.String() + "/conversation"
		req := newTestRequest(http.MethodPost, path, userID, nil)
		rec := httptest.NewRecorder()
		h.GetSessionConversation(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want 405", rec.Code)
		}
	})
}
