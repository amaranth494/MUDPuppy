package profiles

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/amaranth494/MudPuppy/internal/policy"
	"github.com/amaranth494/MudPuppy/internal/store"
	"github.com/google/uuid"
)

// fakeProfileStore is a hand-written fake satisfying profileStorage, used so
// these tests exercise the handler without a live Postgres connection.
type fakeProfileStore struct {
	profile    *store.Profile
	lastUpdate *store.ProfileUpdate

	// acceptPolicyReturnsNil, when true, makes AcceptPolicy return (nil, nil)
	// regardless of profile state — simulating store.ProfileStore.AcceptPolicy's
	// documented behavior when the profile row is missing (e.g. deleted
	// mid-request) (CR-01).
	acceptPolicyReturnsNil bool
}

func (f *fakeProfileStore) GetProfile(userID, profileID uuid.UUID) (*store.Profile, error) {
	if f.profile != nil && f.profile.ID == profileID && f.profile.UserID == userID {
		return f.profile, nil
	}
	return nil, nil
}

func (f *fakeProfileStore) GetProfileByConnection(userID, connectionID uuid.UUID) (*store.Profile, error) {
	if f.profile != nil && f.profile.ConnectionID == connectionID && f.profile.UserID == userID {
		return f.profile, nil
	}
	return nil, nil
}

func (f *fakeProfileStore) UpdateProfile(userID, profileID uuid.UUID, updates *store.ProfileUpdate) (*store.Profile, error) {
	if f.profile == nil || f.profile.ID != profileID || f.profile.UserID != userID {
		return nil, nil
	}
	f.lastUpdate = updates
	if updates.ConductRules != nil {
		f.profile.ConductRules = *updates.ConductRules
	}
	if updates.ApproachGuidance != nil {
		f.profile.ApproachGuidance = *updates.ApproachGuidance
	}
	if updates.AISettings != nil {
		f.profile.AISettings = *updates.AISettings
	}
	return f.profile, nil
}

func (f *fakeProfileStore) AcceptPolicy(userID, profileID uuid.UUID, version string) (*store.Profile, error) {
	if f.acceptPolicyReturnsNil {
		return nil, nil
	}
	if f.profile == nil || f.profile.ID != profileID || f.profile.UserID != userID {
		return nil, nil
	}
	if f.profile.PolicyAcceptedAt == nil {
		v := version
		ts := "2026-01-01T00:00:00Z"
		f.profile.PolicyVersionAccepted = &v
		f.profile.PolicyAcceptedAt = &ts
	}
	return f.profile, nil
}

func newTestProfile(userID, profileID, connectionID uuid.UUID) *store.Profile {
	return &store.Profile{
		ID:           profileID,
		UserID:       userID,
		ConnectionID: connectionID,
		Keybindings:  map[string]string{},
	}
}

func newTestRequest(method, path string, userID uuid.UUID, body interface{}) *http.Request {
	var buf *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		buf = bytes.NewBuffer(b)
	} else {
		buf = bytes.NewBuffer(nil)
	}
	req := httptest.NewRequest(method, path, buf)
	return req.WithContext(context.WithValue(req.Context(), "user_id", userID.String()))
}

func TestAISettingsRoundTrip(t *testing.T) {
	userID := uuid.New()
	profileID := uuid.New()
	connectionID := uuid.New()
	fake := &fakeProfileStore{profile: newTestProfile(userID, profileID, connectionID)}
	h := &Handler{profileStore: fake}

	callCap := 25
	threshold := 2
	putBody := AISettingsResponse{
		ConductRules:     "no PKing",
		ApproachGuidance: "level cautiously",
		AISettings: store.AISettings{
			ModelName:          "gemini-2.5-pro",
			CallCap:            &callCap,
			DisengageThreshold: &threshold,
		},
	}

	path := "/api/v1/profiles/" + connectionID.String() + "/ai-settings"
	putReq := newTestRequest(http.MethodPut, path, userID, putBody)
	putRec := httptest.NewRecorder()
	h.PutAISettings(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, body = %s", putRec.Code, putRec.Body.String())
	}

	getReq := newTestRequest(http.MethodGet, path, userID, nil)
	getRec := httptest.NewRecorder()
	h.GetAISettings(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET status = %d, body = %s", getRec.Code, getRec.Body.String())
	}

	var got AISettingsResponse
	if err := json.NewDecoder(getRec.Body).Decode(&got); err != nil {
		t.Fatalf("decode GET response: %v", err)
	}
	if got.ConductRules != "no PKing" {
		t.Errorf("ConductRules = %q, want %q", got.ConductRules, "no PKing")
	}
	if got.ApproachGuidance != "level cautiously" {
		t.Errorf("ApproachGuidance = %q, want %q", got.ApproachGuidance, "level cautiously")
	}
	if got.AISettings.ModelName != "gemini-2.5-pro" {
		t.Errorf("ModelName = %q, want %q", got.AISettings.ModelName, "gemini-2.5-pro")
	}
	if got.AISettings.CallCap == nil || *got.AISettings.CallCap != 25 {
		t.Errorf("CallCap = %v, want 25", got.AISettings.CallCap)
	}
	if got.AISettings.DisengageThreshold == nil || *got.AISettings.DisengageThreshold != 2 {
		t.Errorf("DisengageThreshold = %v, want 2", got.AISettings.DisengageThreshold)
	}
}

func TestAISettingsBlankRoundTrip(t *testing.T) {
	userID := uuid.New()
	profileID := uuid.New()
	connectionID := uuid.New()
	fake := &fakeProfileStore{profile: newTestProfile(userID, profileID, connectionID)}
	h := &Handler{profileStore: fake}

	putBody := AISettingsResponse{
		ConductRules:     "",
		ApproachGuidance: "",
		AISettings: store.AISettings{
			ModelName:          "",
			CallCap:            nil,
			DisengageThreshold: nil,
		},
	}

	path := "/api/v1/profiles/" + connectionID.String() + "/ai-settings"
	putReq := newTestRequest(http.MethodPut, path, userID, putBody)
	putRec := httptest.NewRecorder()
	h.PutAISettings(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, body = %s", putRec.Code, putRec.Body.String())
	}

	getReq := newTestRequest(http.MethodGet, path, userID, nil)
	getRec := httptest.NewRecorder()
	h.GetAISettings(getRec, getReq)

	var got AISettingsResponse
	if err := json.NewDecoder(getRec.Body).Decode(&got); err != nil {
		t.Fatalf("decode GET response: %v", err)
	}
	if got.ConductRules != "" {
		t.Errorf("ConductRules = %q, want blank", got.ConductRules)
	}
	if got.ApproachGuidance != "" {
		t.Errorf("ApproachGuidance = %q, want blank", got.ApproachGuidance)
	}
	if got.AISettings.ModelName != "" {
		t.Errorf("ModelName = %q, want blank", got.AISettings.ModelName)
	}
	if got.AISettings.CallCap != nil {
		t.Errorf("CallCap = %v, want nil", got.AISettings.CallCap)
	}
	if got.AISettings.DisengageThreshold != nil {
		t.Errorf("DisengageThreshold = %v, want nil", got.AISettings.DisengageThreshold)
	}
}

func TestAISettingsRejectsOverLengthText(t *testing.T) {
	userID := uuid.New()
	profileID := uuid.New()
	connectionID := uuid.New()
	fake := &fakeProfileStore{profile: newTestProfile(userID, profileID, connectionID)}
	h := &Handler{profileStore: fake}

	overLength := strings.Repeat("a", 20001)
	putBody := AISettingsResponse{ConductRules: overLength}

	path := "/api/v1/profiles/" + connectionID.String() + "/ai-settings"
	putReq := newTestRequest(http.MethodPut, path, userID, putBody)
	putRec := httptest.NewRecorder()
	h.PutAISettings(putRec, putReq)

	if putRec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", putRec.Code)
	}
	want := `{"error":"Conduct rules must be 20000 characters or less"}` + "\n"
	if putRec.Body.String() != want {
		t.Errorf("body = %q, want %q", putRec.Body.String(), want)
	}
	if fake.lastUpdate != nil {
		t.Errorf("fake store recorded an update, want none")
	}
}

func TestAISettingsCannotSetAcceptance(t *testing.T) {
	userID := uuid.New()
	profileID := uuid.New()
	connectionID := uuid.New()
	fake := &fakeProfileStore{profile: newTestProfile(userID, profileID, connectionID)}
	h := &Handler{profileStore: fake}

	// Body carries acceptance fields that AISettingsResponse does not define;
	// json.Decode simply ignores unknown fields, so this proves there is no
	// way for a client to reach the acceptance columns via this endpoint.
	rawBody := `{"conduct_rules":"x","approach_guidance":"y","ai_settings":{"model_name":""},"policy_accepted_at":"2020-01-01T00:00:00Z","policy_version_accepted":"9.9"}`

	path := "/api/v1/profiles/" + connectionID.String() + "/ai-settings"
	req := httptest.NewRequest(http.MethodPut, path, bytes.NewBufferString(rawBody))
	req = req.WithContext(context.WithValue(req.Context(), "user_id", userID.String()))
	rec := httptest.NewRecorder()
	h.PutAISettings(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if fake.profile.PolicyAcceptedAt != nil {
		t.Errorf("PolicyAcceptedAt = %v, want nil", fake.profile.PolicyAcceptedAt)
	}
	if fake.profile.PolicyVersionAccepted != nil {
		t.Errorf("PolicyVersionAccepted = %v, want nil", fake.profile.PolicyVersionAccepted)
	}
}

func TestPolicyAcceptUsesServerVersion(t *testing.T) {
	userID := uuid.New()
	profileID := uuid.New()
	connectionID := uuid.New()
	fake := &fakeProfileStore{profile: newTestProfile(userID, profileID, connectionID)}
	h := &Handler{profileStore: fake}

	path := "/api/v1/profiles/" + connectionID.String() + "/policy/accept"
	req1 := newTestRequest(http.MethodPost, path, userID, nil)
	rec1 := httptest.NewRecorder()
	h.AcceptPolicy(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("first accept status = %d, body = %s", rec1.Code, rec1.Body.String())
	}

	var first PolicyResponse
	if err := json.NewDecoder(rec1.Body).Decode(&first); err != nil {
		t.Fatalf("decode first accept response: %v", err)
	}
	if first.AcceptedVersion == nil || *first.AcceptedVersion != policy.Version() {
		t.Errorf("AcceptedVersion = %v, want %q", first.AcceptedVersion, policy.Version())
	}
	if first.AcceptedAt == nil {
		t.Fatal("AcceptedAt is nil after first accept")
	}
	firstTimestamp := *first.AcceptedAt

	req2 := newTestRequest(http.MethodPost, path, userID, nil)
	rec2 := httptest.NewRecorder()
	h.AcceptPolicy(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("second accept status = %d, body = %s", rec2.Code, rec2.Body.String())
	}

	var second PolicyResponse
	if err := json.NewDecoder(rec2.Body).Decode(&second); err != nil {
		t.Fatalf("decode second accept response: %v", err)
	}
	if second.AcceptedAt == nil || *second.AcceptedAt != firstTimestamp {
		t.Errorf("second AcceptedAt = %v, want unchanged %q", second.AcceptedAt, firstTimestamp)
	}
}

func TestPolicyAcceptHandlesProfileDeletedMidRequest(t *testing.T) {
	userID := uuid.New()
	profileID := uuid.New()
	connectionID := uuid.New()
	fake := &fakeProfileStore{profile: newTestProfile(userID, profileID, connectionID)}
	h := &Handler{profileStore: fake}

	// Simulate the profile being deleted between the initial
	// getProfileByConnectionID fetch and the AcceptPolicy write: the
	// fake's AcceptPolicy returns (nil, nil), matching store.ProfileStore's
	// documented not-found behavior (CR-01).
	fake.acceptPolicyReturnsNil = true

	path := "/api/v1/profiles/" + connectionID.String() + "/policy/accept"
	req := newTestRequest(http.MethodPost, path, userID, nil)
	rec := httptest.NewRecorder()

	h.AcceptPolicy(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s, want %d", rec.Code, rec.Body.String(), http.StatusBadRequest)
	}

	var got ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Error != "Profile not found" {
		t.Errorf("Error = %q, want %q", got.Error, "Profile not found")
	}
}

func TestEngageGateHandlerRefusesWithoutAcceptance(t *testing.T) {
	userID := uuid.New()
	profileID := uuid.New()
	connectionID := uuid.New()
	fake := &fakeProfileStore{profile: newTestProfile(userID, profileID, connectionID)}
	h := &Handler{profileStore: fake}

	path := "/api/v1/profiles/" + connectionID.String() + "/engage-gate"
	req := newTestRequest(http.MethodGet, path, userID, nil)
	rec := httptest.NewRecorder()
	h.GetEngageGate(rec, req)

	var got EngageGateResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Allowed {
		t.Errorf("Allowed = true, want false")
	}
	if got.Message != store.EngageGateRefusalMessage {
		t.Errorf("Message = %q, want %q", got.Message, store.EngageGateRefusalMessage)
	}
}

func TestEngageGateHandlerAllowsAfterAcceptance(t *testing.T) {
	userID := uuid.New()
	profileID := uuid.New()
	connectionID := uuid.New()
	profile := newTestProfile(userID, profileID, connectionID)
	version := "1.0"
	acceptedAt := "2026-01-01T00:00:00Z"
	profile.PolicyVersionAccepted = &version
	profile.PolicyAcceptedAt = &acceptedAt
	fake := &fakeProfileStore{profile: profile}
	h := &Handler{profileStore: fake}

	path := "/api/v1/profiles/" + connectionID.String() + "/engage-gate"
	req := newTestRequest(http.MethodGet, path, userID, nil)
	rec := httptest.NewRecorder()
	h.GetEngageGate(rec, req)

	var got EngageGateResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !got.Allowed {
		t.Errorf("Allowed = false, want true")
	}
	if got.Message != "" {
		t.Errorf("Message = %q, want empty", got.Message)
	}
}

func TestAIPlayerLogLinesAreEmitted(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	userID := uuid.New()
	profileID := uuid.New()
	connectionID := uuid.New()
	fake := &fakeProfileStore{profile: newTestProfile(userID, profileID, connectionID)}
	h := &Handler{profileStore: fake}

	acceptPath := "/api/v1/profiles/" + connectionID.String() + "/policy/accept"
	gatePath := "/api/v1/profiles/" + connectionID.String() + "/engage-gate"
	settingsPath := "/api/v1/profiles/" + connectionID.String() + "/ai-settings"

	// Gate call on unaccepted profile.
	req := newTestRequest(http.MethodGet, gatePath, userID, nil)
	h.GetEngageGate(httptest.NewRecorder(), req)

	// First acceptance.
	req = newTestRequest(http.MethodPost, acceptPath, userID, nil)
	h.AcceptPolicy(httptest.NewRecorder(), req)

	// Repeat acceptance.
	req = newTestRequest(http.MethodPost, acceptPath, userID, nil)
	h.AcceptPolicy(httptest.NewRecorder(), req)

	// Gate call on accepted profile.
	req = newTestRequest(http.MethodGet, gatePath, userID, nil)
	h.GetEngageGate(httptest.NewRecorder(), req)

	// Settings save with distinctive conduct rules text that must not leak.
	putBody := AISettingsResponse{
		ConductRules:     "Respect the game's rules and never PK",
		ApproachGuidance: "level cautiously",
	}
	req = newTestRequest(http.MethodPut, settingsPath, userID, putBody)
	h.PutAISettings(httptest.NewRecorder(), req)

	logOutput := buf.String()

	mustContain := []string{
		"[AI-PLAYER] policy accepted connection_id=",
		"[AI-PLAYER] policy already accepted connection_id=",
		"allowed=false",
		"allowed=true",
		"[AI-PLAYER] ai settings saved connection_id=",
	}
	for _, s := range mustContain {
		if !strings.Contains(logOutput, s) {
			t.Errorf("log output missing %q; got:\n%s", s, logOutput)
		}
	}

	mustNotContain := []string{
		"Respect the game's rules and never PK",
		"Respect the game's rules",
	}
	for _, s := range mustNotContain {
		if strings.Contains(logOutput, s) {
			t.Errorf("log output leaked conduct rules text %q; got:\n%s", s, logOutput)
		}
	}
}
