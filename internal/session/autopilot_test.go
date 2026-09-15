package session

import "testing"

// TestAutopilotTransitions proves every legal move between the three
// switch positions. Subtest names read as sentences in -v output; these
// PASS lines are the canned-report evidence plan 02-07 captures for this
// plan's first two truths.
func TestAutopilotTransitions(t *testing.T) {
	tests := []struct {
		name        string
		fn          func(AutopilotState) (AutopilotState, bool)
		from        AutopilotState
		wantState   AutopilotState
		wantChanged bool
	}{
		// Engage
		{"off_Engage_on", Engage, AutopilotOff, AutopilotOn, true},
		{"on_Engage_unchanged", Engage, AutopilotOn, AutopilotOn, false},
		{"waiting_Engage_unchanged", Engage, AutopilotWaiting, AutopilotWaiting, false},

		// Disengage
		{"on_Disengage_off", Disengage, AutopilotOn, AutopilotOff, true},
		{"waiting_Disengage_off", Disengage, AutopilotWaiting, AutopilotOff, true},
		{"off_Disengage_unchanged", Disengage, AutopilotOff, AutopilotOff, false},

		// EnterWaiting
		{"on_EnterWaiting_waiting", EnterWaiting, AutopilotOn, AutopilotWaiting, true},
		{"off_EnterWaiting_unchanged", EnterWaiting, AutopilotOff, AutopilotOff, false},
		{"waiting_EnterWaiting_unchanged", EnterWaiting, AutopilotWaiting, AutopilotWaiting, false},

		// Resume
		{"waiting_Resume_on", Resume, AutopilotWaiting, AutopilotOn, true},
		{"off_Resume_unchanged", Resume, AutopilotOff, AutopilotOff, false},
		{"on_Resume_unchanged", Resume, AutopilotOn, AutopilotOn, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotState, gotChanged := tt.fn(tt.from)
			if gotState != tt.wantState {
				t.Errorf("from %q: state = %q, want %q", tt.from, gotState, tt.wantState)
			}
			if gotChanged != tt.wantChanged {
				t.Errorf("from %q: changed = %v, want %v", tt.from, gotChanged, tt.wantChanged)
			}
		})
	}
}
