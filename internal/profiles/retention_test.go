package profiles

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amaranth494/MudPuppy/internal/store"
	"github.com/google/uuid"
)

// fakeRetentionStore is a hand-written fake satisfying retentionStorage,
// used so these tests exercise the handler without a live Postgres
// connection — same discipline as fakeTranscriptStore in logs_test.go.
type fakeRetentionStore struct {
	countsByConnection map[uuid.UUID]store.RetentionCounts
	err                error
	calledWith         uuid.UUID
}

func (f *fakeRetentionStore) DeleteCapturedTextForConnection(connectionID uuid.UUID) (store.RetentionCounts, error) {
	f.calledWith = connectionID
	if f.err != nil {
		return store.RetentionCounts{}, f.err
	}
	return f.countsByConnection[connectionID], nil
}

// TestDeleteCapturedText proves D-21's owner-triggered immediate delete:
// ownership is resolved before anything is deleted (T-4-09), the counts
// round-trip, a store error never reports partial success, and a handler
// with no retention store wired fails closed with 503 rather than
// panicking (same discipline as GetSessionMemory's nil-transcripts case).
func TestDeleteCapturedText(t *testing.T) {
	t.Run("deletes_and_returns_counts", func(t *testing.T) {
		userID := uuid.New()
		profileID := uuid.New()
		connectionID := uuid.New()
		profileFake := &fakeProfileStore{profile: newTestProfile(userID, profileID, connectionID)}
		retentionFake := &fakeRetentionStore{
			countsByConnection: map[uuid.UUID]store.RetentionCounts{
				connectionID: {SnapshotsCleared: 4, TranscriptLinesDeleted: 12},
			},
		}
		h := &Handler{profileStore: profileFake, retention: retentionFake}

		path := "/api/v1/profiles/" + connectionID.String() + "/captured-text"
		req := newTestRequest(http.MethodDelete, path, userID, nil)
		rec := httptest.NewRecorder()
		h.DeleteCapturedText(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}
		if retentionFake.calledWith != connectionID {
			t.Fatalf("DeleteCapturedTextForConnection called with %v, want %v", retentionFake.calledWith, connectionID)
		}
		var got DeleteCapturedTextResponse
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.SnapshotsCleared != 4 || got.TranscriptLinesDeleted != 12 {
			t.Fatalf("got = %+v, want SnapshotsCleared=4 TranscriptLinesDeleted=12", got)
		}
	})

	t.Run("not_owned_connection_is_refused_before_any_delete", func(t *testing.T) {
		userID := uuid.New()
		profileID := uuid.New()
		connectionID := uuid.New()
		otherConnectionID := uuid.New()
		profileFake := &fakeProfileStore{profile: newTestProfile(userID, profileID, connectionID)}
		retentionFake := &fakeRetentionStore{}
		h := &Handler{profileStore: profileFake, retention: retentionFake}

		path := "/api/v1/profiles/" + otherConnectionID.String() + "/captured-text"
		req := newTestRequest(http.MethodDelete, path, userID, nil)
		rec := httptest.NewRecorder()
		h.DeleteCapturedText(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, body = %s, want 400", rec.Code, rec.Body.String())
		}
		if retentionFake.calledWith != uuid.Nil {
			t.Fatalf("DeleteCapturedTextForConnection must not be called for an unowned connection, was called with %v", retentionFake.calledWith)
		}
	})

	t.Run("store_error_never_reports_partial_success", func(t *testing.T) {
		userID := uuid.New()
		profileID := uuid.New()
		connectionID := uuid.New()
		profileFake := &fakeProfileStore{profile: newTestProfile(userID, profileID, connectionID)}
		retentionFake := &fakeRetentionStore{err: errors.New("boom")}
		h := &Handler{profileStore: profileFake, retention: retentionFake}

		path := "/api/v1/profiles/" + connectionID.String() + "/captured-text"
		req := newTestRequest(http.MethodDelete, path, userID, nil)
		rec := httptest.NewRecorder()
		h.DeleteCapturedText(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, body = %s, want 400 (error shape, no partial success)", rec.Code, rec.Body.String())
		}
	})

	t.Run("no_retention_store_wired_fails_closed", func(t *testing.T) {
		userID := uuid.New()
		profileID := uuid.New()
		connectionID := uuid.New()
		profileFake := &fakeProfileStore{profile: newTestProfile(userID, profileID, connectionID)}
		h := &Handler{profileStore: profileFake}

		path := "/api/v1/profiles/" + connectionID.String() + "/captured-text"
		req := newTestRequest(http.MethodDelete, path, userID, nil)
		rec := httptest.NewRecorder()
		h.DeleteCapturedText(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503", rec.Code)
		}
	})

	t.Run("wrong_method_is_rejected", func(t *testing.T) {
		userID := uuid.New()
		profileID := uuid.New()
		connectionID := uuid.New()
		profileFake := &fakeProfileStore{profile: newTestProfile(userID, profileID, connectionID)}
		h := &Handler{profileStore: profileFake, retention: &fakeRetentionStore{}}

		path := "/api/v1/profiles/" + connectionID.String() + "/captured-text"
		req := newTestRequest(http.MethodGet, path, userID, nil)
		rec := httptest.NewRecorder()
		h.DeleteCapturedText(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want 405", rec.Code)
		}
	})
}
