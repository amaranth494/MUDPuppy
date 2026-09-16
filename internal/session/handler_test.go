package session

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amaranth494/MudPuppy/internal/config"
	"github.com/amaranth494/MudPuppy/internal/store"
	"github.com/google/uuid"
)

// engageGateStub is the closure type HandlerCallbacks.EngageGate expects,
// aliased here only for readability in test setup.
type engageGateStub = func(connectionID, userID uuid.UUID) (allowed bool, message string, policyVersion string)

// newAutopilotHandler builds a Handler with no database, no live MUD and no
// real dial — the manager is the same seedable in-memory Manager
// manager_test.go uses, and gate is a subtest-controlled stub. The AI model
// registry is configured (AIConfigured() true) by default so the Phase 1/2
// gate and engage mechanics these existing tests exercise are unaffected by
// the D-20 configuration gate added in plan 03-06; TestAutopilotHandler_AIConfigRefusal
// builds its own handlers directly to vary AI configuration.
func newAutopilotHandler(m *Manager, gate engageGateStub) *Handler {
	return NewHandlerWithCallbacks(m, configuredConfig(), &HandlerCallbacks{EngageGate: gate})
}

// newAutopilotRequest builds an httptest request for the Autopilot handler.
// When userID is non-nil, "user_id" is set on the request context exactly
// like sessionMiddleware does in production.
func newAutopilotRequest(method string, userID *uuid.UUID, rawBody []byte) *http.Request {
	var buf *bytes.Buffer
	if rawBody != nil {
		buf = bytes.NewBuffer(rawBody)
	} else {
		buf = bytes.NewBuffer(nil)
	}
	req := httptest.NewRequest(method, "/api/v1/session/autopilot", buf)
	if userID != nil {
		req = req.WithContext(context.WithValue(req.Context(), "user_id", userID.String()))
	}
	return req
}

func jsonBody(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return b
}

func decodeAutopilotResponse(t *testing.T, rec *httptest.ResponseRecorder) AutopilotResponse {
	t.Helper()
	var resp AutopilotResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v (body=%s)", err, rec.Body.String())
	}
	return resp
}

func alwaysAllow(policyVersion string) engageGateStub {
	return func(connectionID, userID uuid.UUID) (bool, string, string) {
		return true, "", policyVersion
	}
}

func alwaysRefuse(message string) engageGateStub {
	return func(connectionID, userID uuid.UUID) (bool, string, string) {
		return false, message, ""
	}
}

// TestAutopilotHandler proves every refusal, every no-op, and that the
// request body can never name another user's session (T-2-02).
func TestAutopilotHandler(t *testing.T) {
	t.Run("unauthenticated_is_401", func(t *testing.T) {
		m := newTestManager()
		h := newAutopilotHandler(m, alwaysAllow(""))

		req := newAutopilotRequest(http.MethodPost, nil, jsonBody(t, AutopilotRequest{Action: "status"}))
		rec := httptest.NewRecorder()
		h.Autopilot(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("wrong_method_is_405", func(t *testing.T) {
		m := newTestManager()
		h := newAutopilotHandler(m, alwaysAllow(""))
		userID := uuid.New()

		req := newAutopilotRequest(http.MethodGet, &userID, nil)
		rec := httptest.NewRecorder()
		h.Autopilot(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})

	t.Run("refused_when_gate_refuses", func(t *testing.T) {
		m := newTestManager()
		userID := uuid.New()
		h := newAutopilotHandler(m, alwaysRefuse(store.EngageGateRefusalMessage))

		req := newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{
			Action:       "on",
			ConnectionID: uuid.New(),
		}))
		rec := httptest.NewRecorder()
		h.Autopilot(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		resp := decodeAutopilotResponse(t, rec)
		if resp.State != string(AutopilotOff) {
			t.Errorf("state = %q, want %q", resp.State, AutopilotOff)
		}
		if resp.Outcome != "refused-gate" {
			t.Errorf("outcome = %q, want %q", resp.Outcome, "refused-gate")
		}
		if resp.GateAllowed {
			t.Errorf("gate_allowed = true, want false")
		}
		if resp.GateMessage != store.EngageGateRefusalMessage {
			t.Errorf("gate_message = %q, want the store.EngageGateRefusalMessage constant verbatim", resp.GateMessage)
		}
	})

	t.Run("refused_when_gate_passes_but_no_connected_session", func(t *testing.T) {
		m := newTestManager()
		userID := uuid.New()
		h := newAutopilotHandler(m, alwaysAllow(""))

		req := newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{
			Action:       "on",
			ConnectionID: uuid.New(),
		}))
		rec := httptest.NewRecorder()
		h.Autopilot(rec, req)

		resp := decodeAutopilotResponse(t, rec)
		if resp.Outcome != "refused-no-session" {
			t.Errorf("outcome = %q, want %q", resp.Outcome, "refused-no-session")
		}
		if resp.State != string(AutopilotOff) {
			t.Errorf("state = %q, want %q", resp.State, AutopilotOff)
		}
	})

	t.Run("engages_when_gate_passes_and_session_connected", func(t *testing.T) {
		m := newTestManager()
		userID := uuid.New()
		connID := uuid.New()
		seedConnectedSession(m, userID.String(), connID.String())
		h := newAutopilotHandler(m, alwaysAllow(""))

		req := newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{
			Action:       "on",
			ConnectionID: connID,
		}))
		rec := httptest.NewRecorder()
		h.Autopilot(rec, req)

		resp := decodeAutopilotResponse(t, rec)
		if resp.Outcome != "engaged" {
			t.Errorf("outcome = %q, want %q", resp.Outcome, "engaged")
		}
		if resp.State != string(AutopilotOn) {
			t.Errorf("state = %q, want %q", resp.State, AutopilotOn)
		}
	})

	t.Run("repeat_on_is_already_on", func(t *testing.T) {
		m := newTestManager()
		userID := uuid.New()
		connID := uuid.New()
		seedConnectedSession(m, userID.String(), connID.String())
		h := newAutopilotHandler(m, alwaysAllow(""))

		req1 := newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "on", ConnectionID: connID}))
		rec1 := httptest.NewRecorder()
		h.Autopilot(rec1, req1)
		resp1 := decodeAutopilotResponse(t, rec1)
		if resp1.Outcome != "engaged" {
			t.Fatalf("first call outcome = %q, want %q", resp1.Outcome, "engaged")
		}

		req2 := newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "on", ConnectionID: connID}))
		rec2 := httptest.NewRecorder()
		h.Autopilot(rec2, req2)
		resp2 := decodeAutopilotResponse(t, rec2)
		if resp2.Outcome != "already-on" {
			t.Errorf("second call outcome = %q, want %q", resp2.Outcome, "already-on")
		}
		if resp2.State != string(AutopilotOn) {
			t.Errorf("second call state = %q, want %q", resp2.State, AutopilotOn)
		}
	})

	t.Run("off_then_off_again", func(t *testing.T) {
		m := newTestManager()
		userID := uuid.New()
		connID := uuid.New()
		seedConnectedSession(m, userID.String(), connID.String())
		h := newAutopilotHandler(m, alwaysAllow(""))

		onReq := newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "on", ConnectionID: connID}))
		h.Autopilot(httptest.NewRecorder(), onReq)

		req1 := newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "off"}))
		rec1 := httptest.NewRecorder()
		h.Autopilot(rec1, req1)
		resp1 := decodeAutopilotResponse(t, rec1)
		if resp1.Outcome != "disengaged" {
			t.Errorf("first off outcome = %q, want %q", resp1.Outcome, "disengaged")
		}

		req2 := newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "off"}))
		rec2 := httptest.NewRecorder()
		h.Autopilot(rec2, req2)
		resp2 := decodeAutopilotResponse(t, rec2)
		if resp2.Outcome != "already-off" {
			t.Errorf("second off outcome = %q, want %q", resp2.Outcome, "already-off")
		}
	})

	t.Run("off_is_allowed_even_when_gate_refuses", func(t *testing.T) {
		m := newTestManager()
		userID := uuid.New()
		connID := uuid.New()
		seedConnectedSession(m, userID.String(), connID.String())
		// Engage while allowed, then flip the stub to refuse and confirm
		// #AUTO OFF still works — the owner can always take the wheel back
		// (D-01).
		h := newAutopilotHandler(m, alwaysAllow(""))
		onReq := newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "on", ConnectionID: connID}))
		h.Autopilot(httptest.NewRecorder(), onReq)

		hRefuse := newAutopilotHandler(m, alwaysRefuse(store.EngageGateRefusalMessage))
		req := newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "off"}))
		rec := httptest.NewRecorder()
		hRefuse.Autopilot(rec, req)

		resp := decodeAutopilotResponse(t, rec)
		if resp.Outcome != "disengaged" {
			t.Errorf("outcome = %q, want %q (off must never be refused-gate)", resp.Outcome, "disengaged")
		}

		req2 := newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "off"}))
		rec2 := httptest.NewRecorder()
		hRefuse.Autopilot(rec2, req2)
		resp2 := decodeAutopilotResponse(t, rec2)
		if resp2.Outcome != "already-off" {
			t.Errorf("second off outcome = %q, want %q", resp2.Outcome, "already-off")
		}
	})

	t.Run("status_carries_gate_result", func(t *testing.T) {
		m := newTestManager()
		userID := uuid.New()
		h := newAutopilotHandler(m, alwaysAllow("1.0"))

		req := newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "status", ConnectionID: uuid.New()}))
		rec := httptest.NewRecorder()
		h.Autopilot(rec, req)

		resp := decodeAutopilotResponse(t, rec)
		if resp.Outcome != "status" {
			t.Errorf("outcome = %q, want %q", resp.Outcome, "status")
		}
		if !resp.GateAllowed {
			t.Errorf("gate_allowed = false, want true")
		}
		if resp.PolicyVersion != "1.0" {
			t.Errorf("policy_version = %q, want %q", resp.PolicyVersion, "1.0")
		}
		if resp.State != string(AutopilotOff) {
			t.Errorf("state = %q, want %q (status must not change state)", resp.State, AutopilotOff)
		}
	})

	t.Run("invalid_action_is_400", func(t *testing.T) {
		m := newTestManager()
		userID := uuid.New()
		h := newAutopilotHandler(m, alwaysAllow(""))

		req := newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "sideways"}))
		rec := httptest.NewRecorder()
		h.Autopilot(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
		if got := m.AutopilotStateFor(userID.String()); got != AutopilotOff {
			t.Errorf("AutopilotStateFor = %q, want %q (invalid action must change nothing)", got, AutopilotOff)
		}
	})

	t.Run("nil_gate_callback_refuses", func(t *testing.T) {
		m := newTestManager()
		userID := uuid.New()
		connID := uuid.New()
		seedConnectedSession(m, userID.String(), connID.String())
		h := NewHandlerWithCallbacks(m, &config.Config{}, &HandlerCallbacks{EngageGate: nil})

		req := newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "on", ConnectionID: connID}))
		rec := httptest.NewRecorder()
		h.Autopilot(rec, req)

		resp := decodeAutopilotResponse(t, rec)
		if resp.GateAllowed {
			t.Errorf("gate_allowed = true, want false (a missing dependency must fail closed)")
		}
		if resp.Outcome != "refused-gate" {
			t.Errorf("outcome = %q, want %q", resp.Outcome, "refused-gate")
		}
	})

	t.Run("body_cannot_name_another_user", func(t *testing.T) {
		m := newTestManager()
		userA := uuid.New()
		userB := uuid.New()
		connA := uuid.New()
		seedConnectedSession(m, userA.String(), connA.String())
		seedConnectedSession(m, userB.String(), uuid.New().String())
		h := newAutopilotHandler(m, alwaysAllow(""))

		// Raw JSON carrying extra fields AutopilotRequest doesn't declare —
		// a user id and a state — to prove the handler cannot be steered
		// toward another user's switch even if a client sends them.
		raw := []byte(`{"action":"on","connection_id":"` + connA.String() +
			`","user_id":"` + userB.String() + `","state":"on"}`)
		req := newAutopilotRequest(http.MethodPost, &userA, raw)
		rec := httptest.NewRecorder()
		h.Autopilot(rec, req)

		resp := decodeAutopilotResponse(t, rec)
		if resp.Outcome != "engaged" {
			t.Fatalf("outcome = %q, want %q (user A's own switch)", resp.Outcome, "engaged")
		}
		if got := m.AutopilotStateFor(userA.String()); got != AutopilotOn {
			t.Errorf("AutopilotStateFor(userA) = %q, want %q", got, AutopilotOn)
		}
		if got := m.AutopilotStateFor(userB.String()); got != AutopilotOff {
			t.Errorf("AutopilotStateFor(userB) = %q, want %q (must be untouched)", got, AutopilotOff)
		}
	})
}

// TestStatusCarriesAutopilotState proves StatusResponse.AutopilotState is
// populated unconditionally, including when off (no omitempty) — the
// mechanism the browser badge relies on after a page refresh (D-10).
func TestStatusCarriesAutopilotState(t *testing.T) {
	m := newTestManager()
	userID := uuid.New()
	connID := uuid.New()
	seedConnectedSession(m, userID.String(), connID.String())
	h := newAutopilotHandler(m, alwaysAllow(""))

	if _, _, err := m.EngageAutopilot(userID.String(), connID.String()); err != nil {
		t.Fatalf("EngageAutopilot: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/session/status", nil)
	req = req.WithContext(context.WithValue(req.Context(), "user_id", userID.String()))
	rec := httptest.NewRecorder()
	h.Status(rec, req)

	var statusResp StatusResponse
	if err := json.NewDecoder(bytes.NewReader(rec.Body.Bytes())).Decode(&statusResp); err != nil {
		t.Fatalf("decode status response: %v", err)
	}
	if statusResp.AutopilotState != string(AutopilotOn) {
		t.Errorf("autopilot_state = %q, want %q", statusResp.AutopilotState, AutopilotOn)
	}

	m.DisengageAutopilot(userID.String(), "disengage")

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/session/status", nil)
	req2 = req2.WithContext(context.WithValue(req2.Context(), "user_id", userID.String()))
	rec2 := httptest.NewRecorder()
	h.Status(rec2, req2)

	body := rec2.Body.String()
	if !bytes.Contains([]byte(body), []byte(`"autopilot_state":"off"`)) {
		t.Errorf("raw status body missing literal `\"autopilot_state\":\"off\"` (no omitempty required); body=%s", body)
	}
}

// TestEngageRefusedForAnotherProfile proves code review C2: a request that
// names a profile other than the one the live session was opened for is
// refused as refused-wrong-connection and the switch stays off.
func TestEngageRefusedForAnotherProfile(t *testing.T) {
	m := newTestManager()
	userID := uuid.New()
	seedConnectedSession(m, userID.String(), uuid.New().String())
	h := newAutopilotHandler(m, alwaysAllow(""))

	req := newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "on", ConnectionID: uuid.New()}))
	rec := httptest.NewRecorder()
	h.Autopilot(rec, req)

	resp := decodeAutopilotResponse(t, rec)
	if resp.Outcome != "refused-wrong-connection" {
		t.Fatalf("outcome = %q, want %q", resp.Outcome, "refused-wrong-connection")
	}
	if resp.State != string(AutopilotOff) {
		t.Fatalf("state = %q, want %q", resp.State, AutopilotOff)
	}
}

// configuredConfig returns a *config.Config whose AIConfigured() reports
// true: a default slug naming a registry entry with a non-empty model
// name, endpoint and key.
func configuredConfig() *config.Config {
	return &config.Config{
		AIDefaultModelSlug: "GEMINI",
		AIModels: map[string]config.AIModelEntry{
			"GEMINI": {
				Slug:      "GEMINI",
				ModelName: "gemini-test-model",
				Endpoint:  "https://example.invalid",
				APIKey:    "test-key-not-a-real-credential",
				Provider:  "gemini",
			},
		},
	}
}

// unconfiguredConfig returns a *config.Config whose AIConfigured() reports
// false: an empty registry and no default slug.
func unconfiguredConfig() *config.Config {
	return &config.Config{}
}

// TestAutopilotHandler_AIConfigRefusal proves the D-20 configuration gate:
// #AUTO ON is refused with the locked sentence when the AI is not
// configured, the Phase 1 policy gate still refuses first with its own
// unchanged message, and #AUTO OFF is completely unaffected (T-3-29,
// T-3-30, T-3-31).
func TestAutopilotHandler_AIConfigRefusal(t *testing.T) {
	const lockedRefusal = "Autopilot refused: AI is not configured on this server"

	t.Run("refuses_on_when_unconfigured", func(t *testing.T) {
		m := newTestManager()
		userID := uuid.New()
		connID := uuid.New()
		seedConnectedSession(m, userID.String(), connID.String())
		h := NewHandlerWithCallbacks(m, unconfiguredConfig(), &HandlerCallbacks{EngageGate: alwaysAllow("")})

		req := newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{
			Action:       "on",
			ConnectionID: connID,
		}))
		rec := httptest.NewRecorder()
		h.Autopilot(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		resp := decodeAutopilotResponse(t, rec)
		if resp.Outcome != "refused-not-configured" {
			t.Errorf("outcome = %q, want %q", resp.Outcome, "refused-not-configured")
		}
		if resp.GateMessage != lockedRefusal {
			t.Errorf("gate_message = %q, want %q", resp.GateMessage, lockedRefusal)
		}
		if resp.State != string(AutopilotOff) {
			t.Errorf("state = %q, want %q", resp.State, AutopilotOff)
		}
		if got := m.AutopilotStateFor(userID.String()); got != AutopilotOff {
			t.Errorf("AutopilotStateFor = %q, want %q (the switch must stay off)", got, AutopilotOff)
		}
	})

	t.Run("nil_config_refuses", func(t *testing.T) {
		m := newTestManager()
		userID := uuid.New()
		connID := uuid.New()
		seedConnectedSession(m, userID.String(), connID.String())
		h := NewHandlerWithCallbacks(m, nil, &HandlerCallbacks{EngageGate: alwaysAllow("")})

		req := newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{
			Action:       "on",
			ConnectionID: connID,
		}))
		rec := httptest.NewRecorder()
		h.Autopilot(rec, req)

		resp := decodeAutopilotResponse(t, rec)
		if resp.Outcome != "refused-not-configured" {
			t.Errorf("outcome = %q, want %q (a nil config must not panic)", resp.Outcome, "refused-not-configured")
		}
		if got := m.AutopilotStateFor(userID.String()); got != AutopilotOff {
			t.Errorf("AutopilotStateFor = %q, want %q", got, AutopilotOff)
		}
	})

	t.Run("gate_refusal_takes_precedence", func(t *testing.T) {
		m := newTestManager()
		userID := uuid.New()
		connID := uuid.New()
		seedConnectedSession(m, userID.String(), connID.String())
		h := NewHandlerWithCallbacks(m, unconfiguredConfig(), &HandlerCallbacks{EngageGate: alwaysRefuse(store.EngageGateRefusalMessage)})

		req := newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{
			Action:       "on",
			ConnectionID: connID,
		}))
		rec := httptest.NewRecorder()
		h.Autopilot(rec, req)

		resp := decodeAutopilotResponse(t, rec)
		if resp.Outcome != "refused-gate" {
			t.Errorf("outcome = %q, want %q (the policy gate refuses first)", resp.Outcome, "refused-gate")
		}
		if resp.GateMessage != store.EngageGateRefusalMessage {
			t.Errorf("gate_message = %q, want the store.EngageGateRefusalMessage constant verbatim (the two refusals must never blur)", resp.GateMessage)
		}
	})

	t.Run("engages_when_configured", func(t *testing.T) {
		m := newTestManager()
		userID := uuid.New()
		connID := uuid.New()
		seedConnectedSession(m, userID.String(), connID.String())
		h := NewHandlerWithCallbacks(m, configuredConfig(), &HandlerCallbacks{EngageGate: alwaysAllow("")})

		req := newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{
			Action:       "on",
			ConnectionID: connID,
		}))
		rec := httptest.NewRecorder()
		h.Autopilot(rec, req)

		resp := decodeAutopilotResponse(t, rec)
		if resp.Outcome != "engaged" {
			t.Errorf("outcome = %q, want %q", resp.Outcome, "engaged")
		}
		if resp.State != string(AutopilotOn) {
			t.Errorf("state = %q, want %q", resp.State, AutopilotOn)
		}
	})

	t.Run("off_is_unaffected_when_unconfigured", func(t *testing.T) {
		m := newTestManager()
		userID := uuid.New()
		connID := uuid.New()
		seedConnectedSession(m, userID.String(), connID.String())
		h := NewHandlerWithCallbacks(m, unconfiguredConfig(), &HandlerCallbacks{EngageGate: alwaysAllow("")})

		req := newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "off"}))
		rec := httptest.NewRecorder()
		h.Autopilot(rec, req)

		resp := decodeAutopilotResponse(t, rec)
		if resp.Outcome != "disengaged" && resp.Outcome != "already-off" {
			t.Errorf("outcome = %q, want %q or %q (the AI's absence must not touch #AUTO OFF)", resp.Outcome, "disengaged", "already-off")
		}
		if resp.Outcome == "refused-not-configured" {
			t.Errorf("outcome = %q, #AUTO OFF must never be refused for a missing AI config", resp.Outcome)
		}
	})
}

// TestStatusCarriesAutopilotConnectionID proves code review C3: while the
// switch is on or waiting, the status poll names the profile it is bound
// to, and it is omitted once the switch is off.
func TestStatusCarriesAutopilotConnectionID(t *testing.T) {
	m := newTestManager()
	userID := uuid.New()
	connID := uuid.New()
	seedConnectedSession(m, userID.String(), connID.String())
	h := newAutopilotHandler(m, alwaysAllow(""))

	status := func() StatusResponse {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/session/status", nil)
		req = req.WithContext(context.WithValue(req.Context(), "user_id", userID.String()))
		rec := httptest.NewRecorder()
		h.Status(rec, req)
		var resp StatusResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode status: %v", err)
		}
		return resp
	}

	if got := status().AutopilotConnectionID; got != "" {
		t.Fatalf("autopilot_connection_id while off = %q, want empty", got)
	}
	if _, _, err := m.EngageAutopilot(userID.String(), connID.String()); err != nil {
		t.Fatalf("EngageAutopilot: %v", err)
	}
	if got := status().AutopilotConnectionID; got != connID.String() {
		t.Fatalf("autopilot_connection_id while on = %q, want %q", got, connID)
	}
	if err := m.Disconnect(userID.String(), ReasonRemote); err != nil {
		t.Fatalf("Disconnect: %v", err)
	}
	if got := status().AutopilotConnectionID; got != connID.String() {
		t.Fatalf("autopilot_connection_id while waiting = %q, want %q", got, connID)
	}
}
