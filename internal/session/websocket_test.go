package session

import "testing"

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
			seedConnectedSession(m, userID)
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
			seedConnectedSession(m, userID)
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
			seedConnectedSession(m, userID)
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
			seedConnectedSession(m, userID)
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
			seedConnectedSession(m, userID)
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
			seedConnectedSession(m, userID)
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
			seedConnectedSession(m, userA)
			seedConnectedSession(m, userB)
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
