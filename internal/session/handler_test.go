package session

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

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

// TestAutopilotHandler_PauseAndResumeActions drives the endpoint with
// "pause" then "resume" for the four sequences plan 05-01-02 names: pause
// from on, resume from paused, resume while the connection is also lost,
// and pause while off.
func TestAutopilotHandler_PauseAndResumeActions(t *testing.T) {
	t.Run("pause_from_on", func(t *testing.T) {
		m := newTestManager()
		userID := uuid.New()
		connID := uuid.New()
		seedConnectedSession(m, userID.String(), connID.String())
		h := newAutopilotHandler(m, alwaysAllow(""))

		onReq := newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "on", ConnectionID: connID}))
		h.Autopilot(httptest.NewRecorder(), onReq)

		req := newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "pause"}))
		rec := httptest.NewRecorder()
		h.Autopilot(rec, req)

		resp := decodeAutopilotResponse(t, rec)
		if resp.State != string(AutopilotWaiting) {
			t.Errorf("state = %q, want %q", resp.State, AutopilotWaiting)
		}
		if resp.Outcome != "paused" {
			t.Errorf("outcome = %q, want %q", resp.Outcome, "paused")
		}
		if !resp.PausedByOwner {
			t.Errorf("paused_by_owner = false, want true")
		}
		if resp.ConnectionLost {
			t.Errorf("connection_lost = true, want false")
		}
	})

	t.Run("resume_from_paused", func(t *testing.T) {
		m := newTestManager()
		userID := uuid.New()
		connID := uuid.New()
		seedConnectedSession(m, userID.String(), connID.String())
		h := newAutopilotHandler(m, alwaysAllow(""))

		onReq := newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "on", ConnectionID: connID}))
		h.Autopilot(httptest.NewRecorder(), onReq)
		pauseReq := newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "pause"}))
		h.Autopilot(httptest.NewRecorder(), pauseReq)

		req := newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "resume"}))
		rec := httptest.NewRecorder()
		h.Autopilot(rec, req)

		resp := decodeAutopilotResponse(t, rec)
		if resp.State != string(AutopilotOn) {
			t.Errorf("state = %q, want %q", resp.State, AutopilotOn)
		}
		if resp.Outcome != "resumed" {
			t.Errorf("outcome = %q, want %q", resp.Outcome, "resumed")
		}
		if resp.PausedByOwner || resp.ConnectionLost {
			t.Errorf("(paused_by_owner, connection_lost) = (%v, %v), want (false, false)", resp.PausedByOwner, resp.ConnectionLost)
		}
	})

	t.Run("resume_while_connection_also_lost", func(t *testing.T) {
		m := newTestManager()
		userID := uuid.New()
		connID := uuid.New()
		seedConnectedSession(m, userID.String(), connID.String())
		h := newAutopilotHandler(m, alwaysAllow(""))

		onReq := newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "on", ConnectionID: connID}))
		h.Autopilot(httptest.NewRecorder(), onReq)
		pauseReq := newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "pause"}))
		h.Autopilot(httptest.NewRecorder(), pauseReq)
		if err := m.Disconnect(userID.String(), ReasonRemote); err != nil {
			t.Fatalf("Disconnect: %v", err)
		}

		req := newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "resume"}))
		rec := httptest.NewRecorder()
		h.Autopilot(rec, req)

		resp := decodeAutopilotResponse(t, rec)
		if resp.State != string(AutopilotWaiting) {
			t.Errorf("state = %q, want %q", resp.State, AutopilotWaiting)
		}
		if resp.Outcome != "still-waiting" {
			t.Errorf("outcome = %q, want %q", resp.Outcome, "still-waiting")
		}
		if !resp.ConnectionLost {
			t.Errorf("connection_lost = false, want true")
		}
		if resp.PausedByOwner {
			t.Errorf("paused_by_owner = true, want false (the owner resume already cleared its own reason)")
		}
	})

	t.Run("pause_while_off", func(t *testing.T) {
		m := newTestManager()
		userID := uuid.New()
		h := newAutopilotHandler(m, alwaysAllow(""))

		req := newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "pause"}))
		rec := httptest.NewRecorder()
		h.Autopilot(rec, req)

		resp := decodeAutopilotResponse(t, rec)
		if resp.State != string(AutopilotOff) {
			t.Errorf("state = %q, want %q", resp.State, AutopilotOff)
		}
		if resp.Outcome != "already-off" {
			t.Errorf("outcome = %q, want %q", resp.Outcome, "already-off")
		}
		if resp.PausedByOwner || resp.ConnectionLost {
			t.Errorf("(paused_by_owner, connection_lost) = (%v, %v), want (false, false)", resp.PausedByOwner, resp.ConnectionLost)
		}
	})
}

// TestAutopilotHandler_ResumeConsultsTheEngageGate proves code review WR-09
// of Phase 5: the owner's Resume begins a new stint exactly as #AUTO ON
// does, so it passes the same gate -- asked about the connection the switch
// is parked on -- and when the gate refuses, the switch stays paused and the
// owner is told why in the same words.
func TestAutopilotHandler_ResumeConsultsTheEngageGate(t *testing.T) {
	// pausedHandler engages and pauses with the gate open, then hands the
	// test a switch for the gate, the connections it was asked about and
	// the notices it pushed.
	type rig struct {
		h       *Handler
		m       *Manager
		userID  uuid.UUID
		connID  uuid.UUID
		allow   *bool
		asked   *[]uuid.UUID
		notices *[]AIDecisionPayload
		engaged chan uint64
		cfg     *config.Config
	}
	pausedHandler := func(t *testing.T, cfg *config.Config) rig {
		t.Helper()
		allow := true
		var asked []uuid.UUID
		var notices []AIDecisionPayload
		m := newTestManager()
		userID, connID := uuid.New(), uuid.New()
		seedConnectedSession(m, userID.String(), connID.String())
		gate := func(connectionID, uid uuid.UUID) (bool, string, string) {
			asked = append(asked, connectionID)
			if allow {
				return true, "", "1.0"
			}
			return false, store.EngageGateRefusalMessage, ""
		}
		h := NewHandlerWithCallbacks(m, configuredConfig(), &HandlerCallbacks{EngageGate: gate})
		h.SetAINotifier(func(uid string, payload AIDecisionPayload) { notices = append(notices, payload) })

		h.Autopilot(httptest.NewRecorder(), newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "on", ConnectionID: connID})))
		h.Autopilot(httptest.NewRecorder(), newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "pause"})))
		if got := m.AutopilotStateFor(userID.String()); got != AutopilotWaiting {
			t.Fatalf("precondition: state = %q, want %q", got, AutopilotWaiting)
		}

		engaged := make(chan uint64, 4)
		m.SetEngageHook(func(uid, cid string, epoch uint64) { engaged <- epoch })
		asked, notices = nil, nil
		h.config = cfg
		return rig{h: h, m: m, userID: userID, connID: connID, allow: &allow, asked: &asked, notices: &notices, engaged: engaged, cfg: cfg}
	}
	resume := func(t *testing.T, r rig, body AutopilotRequest) AutopilotResponse {
		t.Helper()
		rec := httptest.NewRecorder()
		r.h.Autopilot(rec, newAutopilotRequest(http.MethodPost, &r.userID, jsonBody(t, body)))
		return decodeAutopilotResponse(t, rec)
	}
	assertStillPaused := func(t *testing.T, r rig, resp AutopilotResponse, wantOutcome string) {
		t.Helper()
		if resp.Outcome != wantOutcome {
			t.Errorf("outcome = %q, want %q", resp.Outcome, wantOutcome)
		}
		if resp.State != string(AutopilotWaiting) || !resp.PausedByOwner {
			t.Errorf("(state, paused_by_owner) = (%q, %v), want (%q, true): a refused resume leaves the switch paused", resp.State, resp.PausedByOwner, AutopilotWaiting)
		}
		if got := r.m.AutopilotStateFor(r.userID.String()); got != AutopilotWaiting {
			t.Errorf("manager state = %q, want %q", got, AutopilotWaiting)
		}
		if paused, _ := r.m.AutopilotWaitingReasons(r.userID.String()); !paused {
			t.Errorf("the owner's pause was cleared by a refused resume")
		}
		time.Sleep(20 * time.Millisecond)
		if len(r.engaged) != 0 {
			t.Errorf("the engage hook fired %d time(s) for a refused resume, want 0", len(r.engaged))
		}
	}

	t.Run("a_gate_that_now_refuses_leaves_the_switch_paused_and_says_why", func(t *testing.T) {
		r := pausedHandler(t, configuredConfig())
		*r.allow = false // e.g. policy acceptance withdrawn while paused

		resp := resume(t, r, AutopilotRequest{Action: "resume"})

		assertStillPaused(t, r, resp, "refused-gate")
		if resp.GateAllowed || resp.GateMessage != store.EngageGateRefusalMessage {
			t.Errorf("(gate_allowed, gate_message) = (%v, %q), want (false, %q)", resp.GateAllowed, resp.GateMessage, store.EngageGateRefusalMessage)
		}
		notices := *r.notices
		if len(notices) != 1 || notices[0].Kind != "system" || notices[0].Outcome != "refused" || notices[0].Message != store.EngageGateRefusalMessage {
			t.Fatalf("expected exactly one refused notice carrying the same words #AUTO ON uses, got %+v", notices)
		}
		for _, n := range notices {
			if n.Outcome == "resumed" {
				t.Errorf("a refused resume announced itself as resumed: %+v", n)
			}
		}
	})

	t.Run("the_gate_is_asked_about_the_parked_connection_never_the_request_bodys", func(t *testing.T) {
		r := pausedHandler(t, configuredConfig())
		other := uuid.New() // another of the owner's profiles, named in the body

		resume(t, r, AutopilotRequest{Action: "resume", ConnectionID: other})

		sawParked := false
		for _, id := range *r.asked {
			if id == r.connID {
				sawParked = true
			}
		}
		if !sawParked {
			t.Fatalf("the gate was never asked about the parked connection %s; asked about %v", r.connID, *r.asked)
		}
	})

	t.Run("another_profiles_acceptance_cannot_be_borrowed", func(t *testing.T) {
		allowOnly := uuid.New()
		m := newTestManager()
		userID, connID := uuid.New(), uuid.New()
		seedConnectedSession(m, userID.String(), connID.String())
		open := true
		gate := func(connectionID, uid uuid.UUID) (bool, string, string) {
			if open || connectionID == allowOnly {
				return true, "", "1.0"
			}
			return false, store.EngageGateRefusalMessage, ""
		}
		h := NewHandlerWithCallbacks(m, configuredConfig(), &HandlerCallbacks{EngageGate: gate})
		h.Autopilot(httptest.NewRecorder(), newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "on", ConnectionID: connID})))
		h.Autopilot(httptest.NewRecorder(), newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "pause"})))
		open = false // now only the OTHER profile passes the gate

		rec := httptest.NewRecorder()
		h.Autopilot(rec, newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "resume", ConnectionID: allowOnly})))
		resp := decodeAutopilotResponse(t, rec)

		if resp.Outcome != "refused-gate" || m.AutopilotStateFor(userID.String()) != AutopilotWaiting {
			t.Fatalf("outcome = %q, state = %q; naming an accepted profile in the body must not resume the paused one", resp.Outcome, m.AutopilotStateFor(userID.String()))
		}
	})

	t.Run("ai_no_longer_configured_refuses_too", func(t *testing.T) {
		r := pausedHandler(t, unconfiguredConfig())

		resp := resume(t, r, AutopilotRequest{Action: "resume"})

		assertStillPaused(t, r, resp, "refused-not-configured")
		if resp.GateMessage != "Autopilot refused: AI is not configured on this server" {
			t.Errorf("gate_message = %q, want the same words #AUTO ON uses", resp.GateMessage)
		}
	})

	t.Run("a_missing_gate_fails_closed", func(t *testing.T) {
		r := pausedHandler(t, configuredConfig())
		r.h.callbacks = &HandlerCallbacks{EngageGate: nil}

		resp := resume(t, r, AutopilotRequest{Action: "resume"})

		assertStillPaused(t, r, resp, "refused-gate")
	})

	t.Run("an_open_gate_resumes_as_before", func(t *testing.T) {
		r := pausedHandler(t, configuredConfig())

		resp := resume(t, r, AutopilotRequest{Action: "resume"})

		if resp.Outcome != "resumed" || resp.State != string(AutopilotOn) {
			t.Fatalf("(outcome, state) = (%q, %q), want (resumed, on)", resp.Outcome, resp.State)
		}
		if !pollUntil(t, 2*time.Second, func() bool { return len(r.engaged) >= 1 }) {
			t.Fatal("timed out waiting for the engage hook on an allowed resume")
		}
	})

	t.Run("nothing_to_resume_still_answers_not_waiting_even_with_the_gate_shut", func(t *testing.T) {
		m := newTestManager()
		userID := uuid.New()
		h := newAutopilotHandler(m, alwaysRefuse(store.EngageGateRefusalMessage))

		rec := httptest.NewRecorder()
		h.Autopilot(rec, newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "resume"})))

		if resp := decodeAutopilotResponse(t, rec); resp.Outcome != "not-waiting" {
			t.Fatalf("outcome = %q, want %q", resp.Outcome, "not-waiting")
		}
	})
}

// TestAutopilotHandler_PushesTheSwitchStateOnPauseAndResume proves code
// review WR-11 of Phase 5: Pause and Resume tell every open screen what the
// switch now reads, with both waiting reasons, instead of leaving the badge
// to the fifteen-second poll; and nothing is pushed when nothing changed.
func TestAutopilotHandler_PushesTheSwitchStateOnPauseAndResume(t *testing.T) {
	type push struct {
		userID         string
		state          AutopilotState
		cause          string
		pausedByOwner  bool
		connectionLost bool
	}
	engaged := func(t *testing.T) (*Handler, *Manager, uuid.UUID, *[]push) {
		t.Helper()
		m := newTestManager()
		userID, connID := uuid.New(), uuid.New()
		seedConnectedSession(m, userID.String(), connID.String())
		h := newAutopilotHandler(m, alwaysAllow(""))
		var pushes []push
		h.SetAutopilotNotifier(func(uid string, state AutopilotState, cause string, paused, lost bool) {
			pushes = append(pushes, push{uid, state, cause, paused, lost})
		})
		h.Autopilot(httptest.NewRecorder(), newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "on", ConnectionID: connID})))
		return h, m, userID, &pushes
	}
	act := func(t *testing.T, h *Handler, userID uuid.UUID, action string) {
		t.Helper()
		h.Autopilot(httptest.NewRecorder(), newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: action})))
	}

	t.Run("pause_pushes_waiting_with_the_owners_reason", func(t *testing.T) {
		h, _, userID, pushes := engaged(t)

		act(t, h, userID, "pause")

		want := []push{{userID.String(), AutopilotWaiting, "pause", true, false}}
		if !reflect.DeepEqual(*pushes, want) {
			t.Fatalf("pushes = %+v, want %+v", *pushes, want)
		}
	})

	t.Run("resume_pushes_on_with_both_reasons_cleared", func(t *testing.T) {
		h, _, userID, pushes := engaged(t)
		act(t, h, userID, "pause")
		*pushes = nil

		act(t, h, userID, "resume")

		want := []push{{userID.String(), AutopilotOn, "resume", false, false}}
		if !reflect.DeepEqual(*pushes, want) {
			t.Fatalf("pushes = %+v, want %+v", *pushes, want)
		}
	})

	t.Run("resume_with_the_connection_still_lost_pushes_the_reason_that_remains", func(t *testing.T) {
		h, m, userID, pushes := engaged(t)
		act(t, h, userID, "pause")
		if err := m.Disconnect(userID.String(), ReasonRemote); err != nil {
			t.Fatalf("Disconnect: %v", err)
		}
		*pushes = nil

		act(t, h, userID, "resume")

		want := []push{{userID.String(), AutopilotWaiting, "resume", false, true}}
		if !reflect.DeepEqual(*pushes, want) {
			t.Fatalf("pushes = %+v, want %+v (still waiting, but the owner's pause is gone)", *pushes, want)
		}
	})

	t.Run("pause_on_top_of_a_lost_connection_pushes_both_reasons", func(t *testing.T) {
		h, m, userID, pushes := engaged(t)
		if err := m.Disconnect(userID.String(), ReasonRemote); err != nil {
			t.Fatalf("Disconnect: %v", err)
		}

		act(t, h, userID, "pause")

		want := []push{{userID.String(), AutopilotWaiting, "pause", true, true}}
		if !reflect.DeepEqual(*pushes, want) {
			t.Fatalf("pushes = %+v, want %+v", *pushes, want)
		}
	})

	t.Run("nothing_is_pushed_when_nothing_changed", func(t *testing.T) {
		m := newTestManager()
		userID := uuid.New()
		h := newAutopilotHandler(m, alwaysAllow(""))
		pushed := 0
		h.SetAutopilotNotifier(func(string, AutopilotState, string, bool, bool) { pushed++ })

		act(t, h, userID, "pause")  // switch is off: already-off
		act(t, h, userID, "resume") // nothing to resume: not-waiting
		act(t, h, userID, "status")

		if pushed != 0 {
			t.Fatalf("pushed %d time(s) for actions that changed nothing, want 0", pushed)
		}
	})

	t.Run("a_refused_resume_pushes_nothing", func(t *testing.T) {
		m := newTestManager()
		userID, connID := uuid.New(), uuid.New()
		seedConnectedSession(m, userID.String(), connID.String())
		allow := true
		h := NewHandlerWithCallbacks(m, configuredConfig(), &HandlerCallbacks{EngageGate: func(uuid.UUID, uuid.UUID) (bool, string, string) {
			if allow {
				return true, "", ""
			}
			return false, store.EngageGateRefusalMessage, ""
		}})
		h.Autopilot(httptest.NewRecorder(), newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "on", ConnectionID: connID})))
		act(t, h, userID, "pause")
		pushed := 0
		h.SetAutopilotNotifier(func(string, AutopilotState, string, bool, bool) { pushed++ })
		allow = false

		act(t, h, userID, "resume")

		if pushed != 0 {
			t.Fatalf("pushed %d time(s) for a refused resume, want 0 (the switch did not move)", pushed)
		}
	})

	t.Run("no_notifier_wired_is_a_no_op", func(t *testing.T) {
		m := newTestManager()
		userID, connID := uuid.New(), uuid.New()
		seedConnectedSession(m, userID.String(), connID.String())
		h := newAutopilotHandler(m, alwaysAllow(""))
		h.Autopilot(httptest.NewRecorder(), newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "on", ConnectionID: connID})))

		act(t, h, userID, "pause") // must not panic
		act(t, h, userID, "resume")
	})
}

// TestAutopilotHandler_EmitsPausedAndResumedNotices proves D-15's thinking-
// stream notices: a locked, unbracketed message on every real pause and
// resume, silence otherwise, and no panic with no notifier wired at all.
func TestAutopilotHandler_EmitsPausedAndResumedNotices(t *testing.T) {
	t.Run("pause_from_on_emits_exactly_one_paused_notice", func(t *testing.T) {
		m := newTestManager()
		userID := uuid.New()
		connID := uuid.New()
		seedConnectedSession(m, userID.String(), connID.String())
		h := newAutopilotHandler(m, alwaysAllow(""))

		var got []AIDecisionPayload
		h.SetAINotifier(func(uid string, payload AIDecisionPayload) { got = append(got, payload) })

		onReq := newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "on", ConnectionID: connID}))
		h.Autopilot(httptest.NewRecorder(), onReq)

		pauseReq := newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "pause"}))
		h.Autopilot(httptest.NewRecorder(), pauseReq)

		if len(got) != 1 {
			t.Fatalf("got %d notices, want exactly 1: %+v", len(got), got)
		}
		if got[0].Kind != "system" {
			t.Errorf("Kind = %q, want %q", got[0].Kind, "system")
		}
		if got[0].Outcome != "paused" {
			t.Errorf("Outcome = %q, want %q", got[0].Outcome, "paused")
		}
		if got[0].Message != "Autopilot paused" {
			t.Errorf("Message = %q, want exactly %q", got[0].Message, "Autopilot paused")
		}
		if strings.ContainsAny(got[0].Message, "[]") {
			t.Errorf("Message %q contains a bracket; the panel adds brackets, not this handler", got[0].Message)
		}
	})

	t.Run("resume_from_paused_only_emits_exactly_one_resumed_notice", func(t *testing.T) {
		m := newTestManager()
		userID := uuid.New()
		connID := uuid.New()
		seedConnectedSession(m, userID.String(), connID.String())
		h := newAutopilotHandler(m, alwaysAllow(""))

		onReq := newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "on", ConnectionID: connID}))
		h.Autopilot(httptest.NewRecorder(), onReq)
		pauseReq := newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "pause"}))
		h.Autopilot(httptest.NewRecorder(), pauseReq)

		var got []AIDecisionPayload
		h.SetAINotifier(func(uid string, payload AIDecisionPayload) { got = append(got, payload) })

		resumeReq := newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "resume"}))
		h.Autopilot(httptest.NewRecorder(), resumeReq)

		if len(got) != 1 {
			t.Fatalf("got %d notices, want exactly 1: %+v", len(got), got)
		}
		if got[0].Outcome != "resumed" {
			t.Errorf("Outcome = %q, want %q", got[0].Outcome, "resumed")
		}
		if got[0].Message != "Autopilot resumed" {
			t.Errorf("Message = %q, want exactly %q", got[0].Message, "Autopilot resumed")
		}
		if strings.ContainsAny(got[0].Message, "[]") {
			t.Errorf("Message %q contains a bracket; the panel adds brackets, not this handler", got[0].Message)
		}
	})

	t.Run("resume_while_connection_also_lost_emits_nothing_until_the_last_reason_clears", func(t *testing.T) {
		m := newTestManager()
		userID := uuid.New()
		connID := uuid.New()
		seedConnectedSession(m, userID.String(), connID.String())
		h := newAutopilotHandler(m, alwaysAllow(""))

		onReq := newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "on", ConnectionID: connID}))
		h.Autopilot(httptest.NewRecorder(), onReq)
		pauseReq := newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "pause"}))
		h.Autopilot(httptest.NewRecorder(), pauseReq)
		if err := m.Disconnect(userID.String(), ReasonRemote); err != nil {
			t.Fatalf("Disconnect: %v", err)
		}

		var got []AIDecisionPayload
		h.SetAINotifier(func(uid string, payload AIDecisionPayload) { got = append(got, payload) })

		resumeReq := newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "resume"}))
		h.Autopilot(httptest.NewRecorder(), resumeReq)

		if len(got) != 0 {
			t.Fatalf("resume while the connection is still lost emitted %d notice(s), want 0: %+v", len(got), got)
		}

		// A later reconnect (resumeAutopilotLocked's path) clears the last
		// reason and completes the resume, but that path fires the engage
		// hook, never this handler's notifier — no notice from a reconnect.
		m.mu.Lock()
		m.sessions[userID.String()] = &Session{UserID: userID.String(), ConnectionID: connID.String(), State: StateConnected}
		m.resumeAutopilotLocked(userID.String(), connID.String())
		m.mu.Unlock()
		if got := m.AutopilotStateFor(userID.String()); got != AutopilotOn {
			t.Fatalf("AutopilotStateFor after reconnect = %q, want %q", got, AutopilotOn)
		}
	})

	t.Run("pause_while_already_waiting_from_a_disconnect_emits_nothing", func(t *testing.T) {
		m := newTestManager()
		userID := uuid.New()
		connID := uuid.New()
		seedConnectedSession(m, userID.String(), connID.String())
		h := newAutopilotHandler(m, alwaysAllow(""))

		onReq := newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "on", ConnectionID: connID}))
		h.Autopilot(httptest.NewRecorder(), onReq)
		if err := m.Disconnect(userID.String(), ReasonRemote); err != nil {
			t.Fatalf("Disconnect: %v", err)
		}

		var got []AIDecisionPayload
		h.SetAINotifier(func(uid string, payload AIDecisionPayload) { got = append(got, payload) })

		pauseReq := newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "pause"}))
		h.Autopilot(httptest.NewRecorder(), pauseReq)

		if len(got) != 0 {
			t.Fatalf("pause while already waiting from a disconnect emitted %d notice(s), want 0: %+v", len(got), got)
		}
	})

	t.Run("pause_while_off_emits_nothing", func(t *testing.T) {
		m := newTestManager()
		userID := uuid.New()
		h := newAutopilotHandler(m, alwaysAllow(""))

		var got []AIDecisionPayload
		h.SetAINotifier(func(uid string, payload AIDecisionPayload) { got = append(got, payload) })

		pauseReq := newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "pause"}))
		h.Autopilot(httptest.NewRecorder(), pauseReq)

		if len(got) != 0 {
			t.Fatalf("pause while off emitted %d notice(s), want 0: %+v", len(got), got)
		}
	})

	t.Run("no_notifier_set_still_returns_normal_responses_and_does_not_panic", func(t *testing.T) {
		m := newTestManager()
		userID := uuid.New()
		connID := uuid.New()
		seedConnectedSession(m, userID.String(), connID.String())
		h := newAutopilotHandler(m, alwaysAllow("")) // SetAINotifier never called

		onReq := newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "on", ConnectionID: connID}))
		h.Autopilot(httptest.NewRecorder(), onReq)

		pauseReq := newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "pause"}))
		pauseRec := httptest.NewRecorder()
		h.Autopilot(pauseRec, pauseReq)
		pauseResp := decodeAutopilotResponse(t, pauseRec)
		if pauseResp.Outcome != "paused" {
			t.Errorf("pause outcome = %q, want %q", pauseResp.Outcome, "paused")
		}

		resumeReq := newAutopilotRequest(http.MethodPost, &userID, jsonBody(t, AutopilotRequest{Action: "resume"}))
		resumeRec := httptest.NewRecorder()
		h.Autopilot(resumeRec, resumeReq)
		resumeResp := decodeAutopilotResponse(t, resumeRec)
		if resumeResp.Outcome != "resumed" {
			t.Errorf("resume outcome = %q, want %q", resumeResp.Outcome, "resumed")
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
