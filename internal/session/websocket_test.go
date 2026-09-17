package session

import (
	"encoding/json"
	"errors"
	"reflect"
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
		if got[0].Decision == nil || !reflect.DeepEqual(*got[0].Decision, payload) {
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

// TestPushAutopilot proves code review WR-11 of Phase 5 at the socket layer:
// the switch state, with both waiting reasons, reaches EVERY open play
// screen of the user -- not only the newest tab, which is all PushAI and
// PushChat write to.
func TestPushAutopilot(t *testing.T) {
	t.Run("every_open_screen_of_the_user_is_told", func(t *testing.T) {
		h := NewWebSocketHandler(newTestManager(), &config.Config{})
		tabOne, tabTwo, stranger := &fakeWSConn{}, &fakeWSConn{}, &fakeWSConn{}
		h.registerClient("user-1", tabOne)
		h.registerClient("user-1", tabTwo)
		h.registerClient("user-2", stranger)

		h.PushAutopilot("user-1", AutopilotWaiting, "pause", true, false)

		for name, tab := range map[string]*fakeWSConn{"first tab": tabOne, "second tab": tabTwo} {
			got := tab.recorded()
			if len(got) != 1 {
				t.Fatalf("%s: recorded %d message(s), want 1", name, len(got))
			}
			want := WSMessage{Type: MsgTypeAutopilot, Status: "waiting", Data: "pause", PausedByOwner: true}
			if !reflect.DeepEqual(got[0], want) {
				t.Errorf("%s: message = %+v, want %+v", name, got[0], want)
			}
		}
		if got := stranger.recorded(); len(got) != 0 {
			t.Fatalf("another user's screen was told: %+v", got)
		}
	})

	t.Run("the_wire_message_carries_the_state_and_both_reasons", func(t *testing.T) {
		h := NewWebSocketHandler(newTestManager(), &config.Config{})
		tab := &fakeWSConn{}
		h.registerClient("user-1", tab)

		h.PushAutopilot("user-1", AutopilotWaiting, "resume", false, true)

		raw, err := json.Marshal(tab.recorded()[0])
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var wire map[string]interface{}
		if err := json.Unmarshal(raw, &wire); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if wire["type"] != "autopilot" || wire["status"] != "waiting" || wire["data"] != "resume" || wire["connection_lost"] != true {
			t.Errorf("wire message = %s", raw)
		}
		if _, present := wire["paused_by_owner"]; present {
			t.Errorf("paused_by_owner should be left out when false (the browser reads absent as false): %s", raw)
		}
	})

	t.Run("one_tabs_failed_write_does_not_stop_the_others", func(t *testing.T) {
		h := NewWebSocketHandler(newTestManager(), &config.Config{})
		dead := &fakeWSConn{writeErr: errors.New("broken pipe")}
		alive := &fakeWSConn{}
		h.registerClient("user-1", dead)
		h.registerClient("user-1", alive)

		h.PushAutopilot("user-1", AutopilotOn, "resume", false, false)

		if got := alive.recorded(); len(got) != 1 || got[0].Status != "on" {
			t.Fatalf("the healthy tab was not told: %+v", got)
		}
	})

	t.Run("a_closed_tab_is_forgotten_and_the_rest_still_hear", func(t *testing.T) {
		h := NewWebSocketHandler(newTestManager(), &config.Config{})
		closed, open := &fakeWSConn{}, &fakeWSConn{}
		h.registerClient("user-1", closed)
		h.registerClient("user-1", open)
		h.unregisterClient("user-1", closed)

		h.PushAutopilot("user-1", AutopilotWaiting, "pause", true, false)

		if len(closed.recorded()) != 0 || len(open.recorded()) != 1 {
			t.Fatalf("closed tab got %d, open tab got %d; want 0 and 1", len(closed.recorded()), len(open.recorded()))
		}

		h.unregisterClient("user-1", open)
		h.clientsMu.RLock()
		_, left := h.allClients["user-1"]
		h.clientsMu.RUnlock()
		if left {
			t.Fatal("expected no entry left once the user's last tab closed")
		}
	})

	t.Run("no_open_screen_is_a_no_op", func(t *testing.T) {
		h := NewWebSocketHandler(newTestManager(), &config.Config{})
		h.PushAutopilot("nobody", AutopilotOff, "pause", false, false) // must not panic
	})

	t.Run("the_ai_and_chat_pushes_still_go_to_the_newest_tab_only", func(t *testing.T) {
		h := NewWebSocketHandler(newTestManager(), &config.Config{})
		older, newer := &fakeWSConn{}, &fakeWSConn{}
		h.registerClient("user-1", older)
		h.registerClient("user-1", newer)

		if err := h.PushAI("user-1", AIDecisionPayload{Kind: "system"}); err != nil {
			t.Fatalf("PushAI = %v", err)
		}
		if len(older.recorded()) != 0 || len(newer.recorded()) != 1 {
			t.Fatalf("PushAI reached older=%d newer=%d; want 0 and 1 (unchanged behaviour)", len(older.recorded()), len(newer.recorded()))
		}

		// The newer tab closing must not strand the registration it replaced
		// the older one in: the older tab's own teardown still only removes
		// itself.
		h.unregisterClient("user-1", older)
		if err := h.PushAI("user-1", AIDecisionPayload{Kind: "system"}); err != nil {
			t.Fatalf("PushAI = %v", err)
		}
		if len(newer.recorded()) != 2 {
			t.Fatalf("the newest tab lost its registration when an older tab closed")
		}
	})
}

// TestAIPayloadSendsZeroAndEmpty is code review WR-05 of Phase 4, on the
// wire. The panel updates a field only when the message carries it, so a
// stint message must carry its counts when they are 0 and its memory list
// when it is empty; a message that says nothing about the stint (the
// goal-changed line) must carry none of them, or it would zero the panel.
func TestAIPayloadSendsZeroAndEmpty(t *testing.T) {
	decode := func(t *testing.T, p AIDecisionPayload) map[string]json.RawMessage {
		t.Helper()
		raw, err := json.Marshal(WSMessage{Type: MsgTypeAI, Decision: &p})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var msg struct {
			Decision map[string]json.RawMessage `json:"decision"`
		}
		if err := json.Unmarshal(raw, &msg); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		return msg.Decision
	}

	t.Run("a_stint_message_carries_zero_counts_and_an_empty_list", func(t *testing.T) {
		got := decode(t, AIDecisionPayload{ID: "dec-1", Kind: "decision", Outcome: "sent"}.
			WithStint("on", 0, 0, false, 0, 0, 3, nil))
		for field, want := range map[string]string{
			"calls": "0", "failures": "0", "blocks": "0", "threshold": "3", "session_memory": "[]", "state": `"on"`,
		} {
			if string(got[field]) != want {
				t.Errorf("%s = %s, want %s (present even when zero or empty)", field, got[field], want)
			}
		}
	})

	t.Run("a_stint_message_carries_its_values", func(t *testing.T) {
		got := decode(t, AIDecisionPayload{Kind: "system"}.
			WithStint("on", 7, 20, true, 1, 2, 3, []string{"the guard wants a pass"}))
		for field, want := range map[string]string{
			"calls": "7", "call_cap": "20", "call_cap_set": "true", "failures": "1", "blocks": "2",
			"session_memory": `["the guard wants a pass"]`,
		} {
			if string(got[field]) != want {
				t.Errorf("%s = %s, want %s", field, got[field], want)
			}
		}
	})

	t.Run("a_message_with_no_stint_data_carries_none_of_it", func(t *testing.T) {
		got := decode(t, AIDecisionPayload{ID: "goal-1", Kind: "system", Outcome: "goal", Message: "Goal changed: reach the vault"})
		for _, field := range []string{"calls", "call_cap", "failures", "blocks", "threshold", "session_memory", "state"} {
			if _, present := got[field]; present {
				t.Errorf("%s is present on a message that says nothing about the stint: %s", field, got[field])
			}
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

		// Code review CR-01 of Phase 5 (D-16): a human command typed while the
		// owner's pause stands takes the wheel exactly as it does while On.
		// This goes THROUGH applyWheelGrab, the only place typing arrives.
		t.Run("paused_is_grabbed", func(t *testing.T) {
			m := newTestManager()
			const userID = "wg-user-7"
			seedConnectedSession(m, userID, "conn-1")
			_, _, pausedEpoch, err := m.EngageAutopilotEpoch(userID, "conn-1")
			if err != nil {
				t.Fatalf("engage err = %v, want nil", err)
			}
			if state, changed := m.PauseAutopilot(userID); !changed || state != AutopilotWaiting {
				t.Fatalf("PauseAutopilot = (%q, %v), want (%q, true)", state, changed, AutopilotWaiting)
			}

			engageFired := make(chan struct{}, 4)
			m.SetEngageHook(func(uid, connID string, epoch uint64) { engageFired <- struct{}{} })

			grabbed, state := applyWheelGrab(m, userID, "user")
			if !grabbed {
				t.Fatalf("grabbed = false, want true (D-16: typing takes the wheel, paused or not)")
			}
			if state != AutopilotOff {
				t.Fatalf("state = %q, want %q", state, AutopilotOff)
			}
			if got := m.AutopilotStateFor(userID); got != AutopilotOff {
				t.Fatalf("AutopilotStateFor = %q, want %q", got, AutopilotOff)
			}
			if paused, lost := m.AutopilotWaitingReasons(userID); paused || lost {
				t.Fatalf("AutopilotWaitingReasons after the grab = (%v, %v), want (false, false)", paused, lost)
			}

			// Off means Off: Resume must not bring AI-player back without a
			// fresh #AUTO ON, and must not start a stint.
			if state, changed := m.ResumeAutopilotByOwner(userID); changed || state != AutopilotOff {
				t.Fatalf("ResumeAutopilotByOwner after the grab = (%q, %v), want (%q, false)", state, changed, AutopilotOff)
			}
			time.Sleep(20 * time.Millisecond)
			if len(engageFired) != 0 {
				t.Fatalf("engage hook fired %d time(s) after a wheel-grab while paused, want 0", len(engageFired))
			}
			if _, epoch := m.AutopilotEpochFor(userID); epoch != pausedEpoch {
				t.Fatalf("epoch after the grab = %d, want unchanged %d (a grab never begins a stint)", epoch, pausedEpoch)
			}
		})

		t.Run("paused_trigger_source_is_not_grabbed", func(t *testing.T) {
			m := newTestManager()
			const userID = "wg-user-8"
			seedConnectedSession(m, userID, "conn-1")
			if _, _, err := m.EngageAutopilot(userID, "conn-1"); err != nil {
				t.Fatalf("engage err = %v, want nil", err)
			}
			if _, changed := m.PauseAutopilot(userID); !changed {
				t.Fatalf("PauseAutopilot changed = false, want true")
			}

			grabbed, state := applyWheelGrab(m, userID, "trigger")
			if grabbed {
				t.Fatalf("grabbed = true, want false (automation never takes the wheel)")
			}
			if state != AutopilotWaiting {
				t.Fatalf("state = %q, want %q", state, AutopilotWaiting)
			}
			if paused, _ := m.AutopilotWaitingReasons(userID); !paused {
				t.Fatalf("paused_by_owner cleared by an automation command, want it to stand")
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

// TestResolveChatConnection proves code review WR-10 of Phase 5: the
// connection a chat message belongs to comes from the server's own state and
// an id in the message can never override it; and code review WR-01: that
// state survives the game connection going down, so chat still has a target
// while disconnected (D-06).
func TestResolveChatConnection(t *testing.T) {
	t.Run("the_live_session_decides_and_the_browser_need_not_name_it", func(t *testing.T) {
		m := newTestManager()
		seedConnectedSession(m, "u1", "conn-a")

		if got, ok := m.ResolveChatConnection("u1", ""); !ok || got != "conn-a" {
			t.Fatalf("ResolveChatConnection = (%q, %v), want (conn-a, true)", got, ok)
		}
		if got, ok := m.ResolveChatConnection("u1", "conn-a"); !ok || got != "conn-a" {
			t.Fatalf("a matching supplied id: got (%q, %v), want (conn-a, true)", got, ok)
		}
	})

	t.Run("a_supplied_id_for_another_connection_is_refused", func(t *testing.T) {
		m := newTestManager()
		seedConnectedSession(m, "u1", "conn-a")

		if got, ok := m.ResolveChatConnection("u1", "conn-b"); ok || got != "" {
			t.Fatalf("ResolveChatConnection = (%q, %v), want (\"\", false): the message must never reach profile B's text under connection A's session", got, ok)
		}
	})

	t.Run("disconnected_with_the_switch_parked_resolves_to_the_parked_connection", func(t *testing.T) {
		m := newTestManager()
		seedConnectedSession(m, "u1", "conn-a")
		if _, _, err := m.EngageAutopilot("u1", "conn-a"); err != nil {
			t.Fatalf("engage err = %v", err)
		}
		if err := m.Disconnect("u1", ReasonRemote); err != nil {
			t.Fatalf("Disconnect err = %v", err)
		}
		// GetSession answers a blank placeholder, not an error, once the
		// session is gone: this is the state the old fallback read "" from.
		if sess, _ := m.GetSession("u1"); sess.ConnectionID != "" || sess.State == StateConnected {
			t.Fatalf("precondition: expected no live session after the disconnect, got %+v", sess)
		}

		if got, ok := m.ResolveChatConnection("u1", ""); !ok || got != "conn-a" {
			t.Fatalf("ResolveChatConnection while WAITING / connection lost = (%q, %v), want (conn-a, true)", got, ok)
		}
		if _, ok := m.ResolveChatConnection("u1", "conn-b"); ok {
			t.Fatal("a different supplied id must still be refused while disconnected")
		}
	})

	t.Run("disconnected_with_the_switch_off_resolves_to_the_last_connection", func(t *testing.T) {
		m := newTestManager()
		m.mu.Lock()
		m.rememberConnectionLocked("u1", "conn-a")
		m.mu.Unlock()

		if got, ok := m.ResolveChatConnection("u1", ""); !ok || got != "conn-a" {
			t.Fatalf("ResolveChatConnection = (%q, %v), want (conn-a, true)", got, ok)
		}
		if _, ok := m.ResolveChatConnection("u1", "conn-b"); ok {
			t.Fatal("a different supplied id must be refused when the server remembers the connection")
		}
	})

	t.Run("a_quick_connect_is_never_remembered", func(t *testing.T) {
		m := newTestManager()
		m.mu.Lock()
		m.rememberConnectionLocked("u1", "")
		m.mu.Unlock()

		if got, ok := m.ResolveChatConnection("u1", ""); ok || got != "" {
			t.Fatalf("ResolveChatConnection = (%q, %v), want (\"\", false)", got, ok)
		}
	})

	t.Run("when_the_server_knows_nothing_the_supplied_id_is_used_for_the_driver_to_verify", func(t *testing.T) {
		m := newTestManager()

		if got, ok := m.ResolveChatConnection("u1", "conn-a"); !ok || got != "conn-a" {
			t.Fatalf("ResolveChatConnection = (%q, %v), want (conn-a, true)", got, ok)
		}
		if got, ok := m.ResolveChatConnection("u1", ""); ok || got != "" {
			t.Fatalf("nothing known and nothing supplied: got (%q, %v), want (\"\", false)", got, ok)
		}
	})

	t.Run("one_users_connection_never_answers_for_another", func(t *testing.T) {
		m := newTestManager()
		seedConnectedSession(m, "u1", "conn-a")

		if got, ok := m.ResolveChatConnection("u2", ""); ok || got != "" {
			t.Fatalf("ResolveChatConnection for another user = (%q, %v), want (\"\", false)", got, ok)
		}
	})
}

// TestWebSocket_ChatIsNotAWheelGrab proves D-16: a chat message never
// disengages autopilot and is never treated as a game command, in every
// autopilot state, including disconnected. It exercises Manager.FireChatHook
// directly -- the exact call the websocket read loop's `case MsgTypeChat:`
// branch makes -- following TestWheelGrabSourceRule's own precedent above of
// testing the read loop's decision logic against a real *Manager with no
// websocket dial anywhere (this package has no harness for a live
// *websocket.Conn round trip; clientToMUD is a channel local to
// HandleWebSocket's own call frame, not a Manager field, so "nothing
// written to the MUD channel" is proven by source inspection -- the
// MsgTypeChat case contains no reference to clientToMUD at all -- rather
// than by a runtime assertion here).
func TestWebSocket_ChatIsNotAWheelGrab(t *testing.T) {
	t.Run("autopilot_on_stays_on_and_the_hook_fires_once", func(t *testing.T) {
		m := newTestManager()
		const userID = "chat-user-1"
		seedConnectedSession(m, userID, "conn-1")
		if _, _, err := m.EngageAutopilot(userID, "conn-1"); err != nil {
			t.Fatalf("engage err = %v, want nil", err)
		}

		type call struct{ userID, connectionID, message string }
		var calls []call
		m.SetChatHook(func(userID, connectionID, message string) {
			calls = append(calls, call{userID, connectionID, message})
		})

		m.FireChatHook(userID, "conn-1", "why did you go north?")

		if got := m.AutopilotStateFor(userID); got != AutopilotOn {
			t.Fatalf("AutopilotStateFor = %q, want still %q (a chat message must never disengage autopilot)", got, AutopilotOn)
		}
		if len(calls) != 1 {
			t.Fatalf("chat hook fired %d time(s), want 1: %+v", len(calls), calls)
		}
		if calls[0] != (call{userID, "conn-1", "why did you go north?"}) {
			t.Fatalf("chat hook call = %+v, want {%q, %q, %q}", calls[0], userID, "conn-1", "why did you go north?")
		}
	})

	t.Run("accepted_and_fires_even_when_not_connected_to_a_game", func(t *testing.T) {
		m := newTestManager()
		const userID = "chat-user-2"
		// No seedConnectedSession call at all -- there is no game connection
		// for this user, exactly the disconnected case D-06 requires the
		// message box to keep working in.

		fired := false
		m.SetChatHook(func(userID, connectionID, message string) {
			fired = true
		})

		m.FireChatHook(userID, "", "hello while disconnected")

		if !fired {
			t.Fatal("expected the chat hook to fire even while not connected to a game")
		}
	})

	t.Run("a_nil_hook_is_a_silent_no_op", func(t *testing.T) {
		m := newTestManager()
		// SetChatHook is never called -- the zero-value default.
		m.FireChatHook("nobody", "nowhere", "hello")
	})
}
