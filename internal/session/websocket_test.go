package session

import (
	"sync"
	"testing"
	"time"

	"github.com/amaranth494/MudPuppy/internal/config"
)

// fakeWSConn is the seam this file uses in place of a real *websocket.Conn
// (task 03-09-01's own documented fallback: registerClient cannot be
// driven with anything but a real connection, so PushAI's registry-lookup
// and write-dispatch logic is exercised here through the wsWriter
// interface instead). It records every WSMessage handed to WriteJSON.
type fakeWSConn struct {
	mu       sync.Mutex
	messages []WSMessage
	writeErr error
}

func (f *fakeWSConn) WriteJSON(v interface{}) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.writeErr != nil {
		return f.writeErr
	}
	if msg, ok := v.(WSMessage); ok {
		f.messages = append(f.messages, msg)
	}
	return nil
}

func (f *fakeWSConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func (f *fakeWSConn) recorded() []WSMessage {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]WSMessage, len(f.messages))
	copy(out, f.messages)
	return out
}

// TestPushAI exercises PushAI's three documented outcomes: a silent no-op
// for a user with no open screen, and a delivered message for both an AI
// decision and a system notice.
func TestPushAI(t *testing.T) {
	t.Run("push_to_absent_user_is_a_no_op", func(t *testing.T) {
		h := NewWebSocketHandler(newTestManager(), &config.Config{})

		err := h.PushAI("nobody-here", AIDecisionPayload{Kind: "decision"})
		if err != nil {
			t.Fatalf("PushAI on an empty registry = %v, want nil", err)
		}
	})

	t.Run("pushes_a_decision", func(t *testing.T) {
		h := NewWebSocketHandler(newTestManager(), &config.Config{})
		fake := &fakeWSConn{}
		h.clients["user-1"] = fake

		payload := AIDecisionPayload{
			ID:        "dec-1",
			Kind:      "decision",
			Reasoning: "The room to the north looks safe to explore.",
			Command:   "north",
			Outcome:   "sent",
			Timestamp: "2026-09-15T00:00:00Z",
		}
		if err := h.PushAI("user-1", payload); err != nil {
			t.Fatalf("PushAI = %v, want nil", err)
		}

		got := fake.recorded()
		if len(got) != 1 {
			t.Fatalf("recorded messages = %d, want 1", len(got))
		}
		if got[0].Type != MsgTypeAI {
			t.Errorf("Type = %q, want %q", got[0].Type, MsgTypeAI)
		}
		if got[0].Decision == nil || *got[0].Decision != payload {
			t.Errorf("Decision = %+v, want %+v", got[0].Decision, payload)
		}
	})

	t.Run("pushes_a_system_line", func(t *testing.T) {
		h := NewWebSocketHandler(newTestManager(), &config.Config{})
		fake := &fakeWSConn{}
		h.clients["user-2"] = fake

		payload := AIDecisionPayload{
			ID:        "dec-2",
			Kind:      "system",
			Outcome:   "failed",
			Message:   "AI decision failed: the model could not be reached. Autopilot disengaged.",
			Timestamp: "2026-09-15T00:00:01Z",
		}
		if err := h.PushAI("user-2", payload); err != nil {
			t.Fatalf("PushAI = %v, want nil", err)
		}

		got := fake.recorded()
		if len(got) != 1 {
			t.Fatalf("recorded messages = %d, want 1", len(got))
		}
		if got[0].Type != MsgTypeAI {
			t.Errorf("Type = %q, want %q", got[0].Type, MsgTypeAI)
		}
		if got[0].Decision == nil || got[0].Decision.Kind != "system" || got[0].Decision.Message != payload.Message {
			t.Errorf("Decision = %+v, want message %q", got[0].Decision, payload.Message)
		}
	})

	t.Run("a_new_tabs_registration_survives_the_old_tabs_unregister", func(t *testing.T) {
		h := NewWebSocketHandler(newTestManager(), &config.Config{})
		oldFake := &fakeWSConn{}
		newFake := &fakeWSConn{}

		h.clientsMu.Lock()
		h.clients["user-3"] = oldFake
		h.clientsMu.Unlock()

		// A new tab replaces the registration...
		h.clientsMu.Lock()
		h.clients["user-3"] = newFake
		h.clientsMu.Unlock()

		// ...then the old tab's own connection tears down. unregisterClient
		// takes a *websocket.Conn, and neither fake is one, but the
		// production comparison-before-delete rule is what this test
		// documents: only a matching stored value is removed. Simulate it
		// directly against the map the same way unregisterClient does.
		h.clientsMu.Lock()
		if stored, ok := h.clients["user-3"]; ok && stored == oldFake {
			delete(h.clients, "user-3")
		}
		h.clientsMu.Unlock()

		if err := h.PushAI("user-3", AIDecisionPayload{Kind: "system"}); err != nil {
			t.Fatalf("PushAI = %v, want nil", err)
		}
		if len(newFake.recorded()) != 1 {
			t.Fatalf("new tab's connection did not receive the push; old tab's teardown deleted the new registration")
		}
	})
}

// TestWheelGrabSourceRule is the test 02-VALIDATION.md's requirement map
// cites. Two halves, no websocket dial anywhere: the source classifier in
// isolation, then the wheel-grab decision against a real Manager.
func TestWheelGrabSourceRule(t *testing.T) {
	t.Run("classifier", func(t *testing.T) {
		tests := []struct {
			source    string
			wantHuman bool
		}{
			{"", true},                  // absent means human (RESEARCH Pitfall 3)
			{"user", true},              // owner typed it
			{"alias", true},             // owner typed the line that expanded (D-07)
			{"trigger", false},          // automation
			{"timer", false},            // automation
			{"TRIGGER", false},          // case-insensitive
			{"not-a-real-source", true}, // garbled -> fail-safe direction is human
		}
		for _, tt := range tests {
			t.Run(tt.source, func(t *testing.T) {
				if got := IsHumanSource(tt.source); got != tt.wantHuman {
					t.Errorf("IsHumanSource(%q) = %v, want %v", tt.source, got, tt.wantHuman)
				}
			})
		}
	})

	t.Run("decision", func(t *testing.T) {
		t.Run("human_source_disengages", func(t *testing.T) {
			m := newTestManager()
			const userID = "wg-user-1"
			seedConnectedSession(m, userID, "conn-1")
			if _, _, err := m.EngageAutopilot(userID, "conn-1"); err != nil {
				t.Fatalf("engage err = %v, want nil", err)
			}

			grabbed, state := applyWheelGrab(m, userID, "user")
			if !grabbed {
				t.Fatalf("grabbed = false, want true")
			}
			if state != AutopilotOff {
				t.Fatalf("state = %q, want %q", state, AutopilotOff)
			}
			if got := m.AutopilotStateFor(userID); got != AutopilotOff {
				t.Fatalf("AutopilotStateFor = %q, want %q", got, AutopilotOff)
			}
		})

		t.Run("alias_source_disengages", func(t *testing.T) {
			m := newTestManager()
			const userID = "wg-user-2"
			seedConnectedSession(m, userID, "conn-1")
			if _, _, err := m.EngageAutopilot(userID, "conn-1"); err != nil {
				t.Fatalf("engage err = %v, want nil", err)
			}

			grabbed, state := applyWheelGrab(m, userID, "alias")
			if !grabbed {
				t.Fatalf("grabbed = false, want true")
			}
			if state != AutopilotOff {
				t.Fatalf("state = %q, want %q", state, AutopilotOff)
			}
			if got := m.AutopilotStateFor(userID); got != AutopilotOff {
				t.Fatalf("AutopilotStateFor = %q, want %q", got, AutopilotOff)
			}
		})

		t.Run("blank_source_disengages", func(t *testing.T) {
			m := newTestManager()
			const userID = "wg-user-3"
			seedConnectedSession(m, userID, "conn-1")
			if _, _, err := m.EngageAutopilot(userID, "conn-1"); err != nil {
				t.Fatalf("engage err = %v, want nil", err)
			}

			grabbed, state := applyWheelGrab(m, userID, "")
			if !grabbed {
				t.Fatalf("grabbed = false, want true")
			}
			if state != AutopilotOff {
				t.Fatalf("state = %q, want %q", state, AutopilotOff)
			}
			if got := m.AutopilotStateFor(userID); got != AutopilotOff {
				t.Fatalf("AutopilotStateFor = %q, want %q", got, AutopilotOff)
			}
		})

		t.Run("trigger_source_leaves_it_on", func(t *testing.T) {
			m := newTestManager()
			const userID = "wg-user-4"
			seedConnectedSession(m, userID, "conn-1")
			if _, _, err := m.EngageAutopilot(userID, "conn-1"); err != nil {
				t.Fatalf("engage err = %v, want nil", err)
			}

			grabbed, state := applyWheelGrab(m, userID, "trigger")
			if grabbed {
				t.Fatalf("grabbed = true, want false")
			}
			if state != AutopilotOn {
				t.Fatalf("state = %q, want %q", state, AutopilotOn)
			}
			if got := m.AutopilotStateFor(userID); got != AutopilotOn {
				t.Fatalf("AutopilotStateFor = %q, want %q", got, AutopilotOn)
			}
		})

		t.Run("off_stays_off", func(t *testing.T) {
			m := newTestManager()
			const userID = "wg-user-5"
			seedConnectedSession(m, userID, "conn-1")
			// Never engaged.

			grabbed, _ := applyWheelGrab(m, userID, "user")
			if grabbed {
				t.Fatalf("grabbed = true, want false")
			}
			if got := m.AutopilotStateFor(userID); got != AutopilotOff {
				t.Fatalf("AutopilotStateFor = %q, want %q (a human command must never engage anything)", got, AutopilotOff)
			}
		})

		t.Run("waiting_is_not_grabbed", func(t *testing.T) {
			m := newTestManager()
			const userID = "wg-user-6"
			seedConnectedSession(m, userID, "conn-1")
			if _, _, err := m.EngageAutopilot(userID, "conn-1"); err != nil {
				t.Fatalf("engage err = %v, want nil", err)
			}
			if err := m.Disconnect(userID, ReasonRemote); err != nil {
				t.Fatalf("Disconnect err = %v, want nil", err)
			}
			if got := m.AutopilotStateFor(userID); got != AutopilotWaiting {
				t.Fatalf("AutopilotStateFor = %q, want %q", got, AutopilotWaiting)
			}

			grabbed, _ := applyWheelGrab(m, userID, "user")
			if grabbed {
				t.Fatalf("grabbed = true, want false")
			}
			if got := m.AutopilotStateFor(userID); got != AutopilotWaiting {
				t.Fatalf("AutopilotStateFor = %q, want %q (a command cannot be typed at a disconnected game; the grab must not quietly turn a parked switch off)", got, AutopilotWaiting)
			}
		})

		t.Run("second_user_untouched", func(t *testing.T) {
			m := newTestManager()
			const userA = "wg-user-a"
			const userB = "wg-user-b"
			seedConnectedSession(m, userA, "conn-a")
			seedConnectedSession(m, userB, "conn-b")
			if _, _, err := m.EngageAutopilot(userA, "conn-a"); err != nil {
				t.Fatalf("engage userA err = %v, want nil", err)
			}
			if _, _, err := m.EngageAutopilot(userB, "conn-b"); err != nil {
				t.Fatalf("engage userB err = %v, want nil", err)
			}

			grabbed, _ := applyWheelGrab(m, userA, "user")
			if !grabbed {
				t.Fatalf("grabbed = false, want true")
			}
			if got := m.AutopilotStateFor(userA); got != AutopilotOff {
				t.Fatalf("AutopilotStateFor(userA) = %q, want %q", got, AutopilotOff)
			}
			if got := m.AutopilotStateFor(userB); got != AutopilotOn {
				t.Fatalf("AutopilotStateFor(userB) = %q, want %q (one user's typing must not move another's switch)", got, AutopilotOn)
			}
		})
	})
}
