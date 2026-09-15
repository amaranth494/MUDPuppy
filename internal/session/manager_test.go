package session

import "testing"

// newTestManager builds a Manager with no port restrictions and no
// timeouts, matching Connect's real construction path but without any
// autopilot-specific fixtures.
func newTestManager() *Manager {
	return NewManager("", "", "", 0, 0)
}

// seedConnectedSession writes a connected session directly into the
// manager's session map under m.mu, standing in for a real Connect call.
// Manager.Connect dials for real and validates the host, rejecting
// loopback and private addresses, so a unit test cannot drive a genuine
// connect — this is legitimate white-box setup since the test is in the
// same package.
func seedConnectedSession(m *Manager, userID, connID string) {
	m.mu.Lock()
	m.sessions[userID] = &Session{UserID: userID, ConnectionID: connID, State: StateConnected}
	m.mu.Unlock()
}

// TestEngageAutopilot exercises the real Manager, never a bare Session
// struct — a test that constructs a Session and asserts on it would pass
// while the production path silently lost the state, which is exactly the
// failure RESEARCH Pitfall 1 warns about.
func TestEngageAutopilot(t *testing.T) {
	t.Run("refused_without_connected_session", func(t *testing.T) {
		m := newTestManager()
		const userID = "user-1"

		state, changed, err := m.EngageAutopilot(userID, "conn-1")
		if err != ErrNoConnectedSession {
			t.Fatalf("err = %v, want ErrNoConnectedSession", err)
		}
		if changed {
			t.Fatalf("changed = true, want false")
		}
		if state != AutopilotOff {
			t.Fatalf("state = %q, want %q", state, AutopilotOff)
		}
		if got := m.AutopilotStateFor(userID); got != AutopilotOff {
			t.Fatalf("AutopilotStateFor = %q, want %q", got, AutopilotOff)
		}
	})

	t.Run("engages_with_connected_session", func(t *testing.T) {
		m := newTestManager()
		const userID = "user-2"
		seedConnectedSession(m, userID, "conn-1")

		state, changed, err := m.EngageAutopilot(userID, "conn-1")
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		if !changed {
			t.Fatalf("changed = false, want true")
		}
		if state != AutopilotOn {
			t.Fatalf("state = %q, want %q", state, AutopilotOn)
		}
	})

	t.Run("engage_for_a_profile_other_than_the_connected_one_is_refused", func(t *testing.T) {
		// Code review C2: the gate is resolved for the connection_id in the
		// request, so the live session must be for that same profile.
		m := newTestManager()
		const userID = "user-3b"
		seedConnectedSession(m, userID, "conn-1")

		state, changed, err := m.EngageAutopilot(userID, "conn-2")
		if err != ErrWrongConnection {
			t.Fatalf("err = %v, want ErrWrongConnection", err)
		}
		if changed || state != AutopilotOff {
			t.Fatalf("(state, changed) = (%q, %v), want (%q, false)", state, changed, AutopilotOff)
		}
		if got := m.AutopilotStateFor(userID); got != AutopilotOff {
			t.Fatalf("AutopilotStateFor = %q, want %q", got, AutopilotOff)
		}
	})

	t.Run("engage_on_a_quick_connect_with_no_profile_is_refused", func(t *testing.T) {
		m := newTestManager()
		const userID = "user-3c"
		seedConnectedSession(m, userID, "")

		if _, _, err := m.EngageAutopilot(userID, "conn-1"); err != ErrWrongConnection {
			t.Fatalf("err = %v, want ErrWrongConnection", err)
		}
	})

	t.Run("repeat_engage_is_a_no_op", func(t *testing.T) {
		m := newTestManager()
		const userID = "user-3"
		seedConnectedSession(m, userID, "conn-1")

		if _, _, err := m.EngageAutopilot(userID, "conn-1"); err != nil {
			t.Fatalf("first engage err = %v, want nil", err)
		}

		state, changed, err := m.EngageAutopilot(userID, "conn-1")
		if err != nil {
			t.Fatalf("second engage err = %v, want nil", err)
		}
		if changed {
			t.Fatalf("changed = true, want false")
		}
		if state != AutopilotOn {
			t.Fatalf("state = %q, want %q", state, AutopilotOn)
		}

		m.mu.RLock()
		rec, ok := m.autopilot[userID]
		m.mu.RUnlock()
		if !ok {
			t.Fatalf("no autopilot record stored for %s", userID)
		}
		if rec.ConnectionID != "conn-1" {
			t.Errorf("ConnectionID = %q, want %q (repeated engage must not reset it)", rec.ConnectionID, "conn-1")
		}
		if rec.WaitingSince != nil {
			t.Errorf("WaitingSince = %v, want nil", rec.WaitingSince)
		}
	})

	t.Run("disengage_then_disengage_again", func(t *testing.T) {
		m := newTestManager()
		const userID = "user-4"
		seedConnectedSession(m, userID, "conn-1")
		if _, _, err := m.EngageAutopilot(userID, "conn-1"); err != nil {
			t.Fatalf("engage err = %v, want nil", err)
		}

		state, changed := m.DisengageAutopilot(userID, "disengage")
		if !changed {
			t.Fatalf("first disengage changed = false, want true")
		}
		if state != AutopilotOff {
			t.Fatalf("state = %q, want %q", state, AutopilotOff)
		}

		state, changed = m.DisengageAutopilot(userID, "disengage")
		if changed {
			t.Fatalf("second disengage changed = true, want false")
		}
		if state != AutopilotOff {
			t.Fatalf("state = %q, want %q", state, AutopilotOff)
		}
	})
}

// TestWaitingSurvivesDisconnectAndResumesOnConnect is the phase's most
// important test: it drives the real Manager.Disconnect and proves the
// waiting state outlives the session map entry it would have been
// destroyed with if stored on Session (RESEARCH Pitfall 1).
func TestWaitingSurvivesDisconnectAndResumesOnConnect(t *testing.T) {
	t.Run("waiting_survives_disconnect_and_resumes_on_connect", func(t *testing.T) {
		m := newTestManager()
		const userID = "user-5"
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

		sess, err := m.GetSession(userID)
		if err != nil {
			t.Fatalf("GetSession err = %v, want nil", err)
		}
		if sess.State != StateDisconnected {
			t.Fatalf("session state = %q, want %q (session was really removed and refabricated)", sess.State, StateDisconnected)
		}

		m.mu.RLock()
		rec, ok := m.autopilot[userID]
		m.mu.RUnlock()
		if !ok {
			t.Fatalf("no autopilot record after disconnect")
		}
		if rec.WaitingSince == nil {
			t.Fatalf("WaitingSince = nil, want non-nil")
		}

		// Simulate the connection returning exactly the way Connect does.
		m.mu.Lock()
		m.sessions[userID] = &Session{UserID: userID, ConnectionID: "conn-1", State: StateConnected}
		m.resumeAutopilotLocked(userID, "conn-1")
		m.mu.Unlock()

		if got := m.AutopilotStateFor(userID); got != AutopilotOn {
			t.Fatalf("AutopilotStateFor after reconnect = %q, want %q", got, AutopilotOn)
		}
		m.mu.RLock()
		rec = m.autopilot[userID]
		m.mu.RUnlock()
		if rec.WaitingSince != nil {
			t.Fatalf("WaitingSince after resume = %v, want nil", rec.WaitingSince)
		}
	})

	t.Run("off_while_waiting_stays_off_after_reconnect", func(t *testing.T) {
		m := newTestManager()
		const userID = "user-6"
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

		if state, changed := m.DisengageAutopilot(userID, "disengage"); !changed || state != AutopilotOff {
			t.Fatalf("DisengageAutopilot = (%q, %v), want (%q, true)", state, changed, AutopilotOff)
		}

		m.mu.Lock()
		m.sessions[userID] = &Session{UserID: userID, ConnectionID: "conn-1", State: StateConnected}
		m.resumeAutopilotLocked(userID, "conn-1")
		m.mu.Unlock()

		if got := m.AutopilotStateFor(userID); got != AutopilotOff {
			t.Fatalf("AutopilotStateFor after reconnect = %q, want %q (a connection returning must never engage an autopilot the owner turned off)", got, AutopilotOff)
		}
	})

	t.Run("reconnect_to_a_different_profile_lands_off_not_on", func(t *testing.T) {
		// Code review C1: a parked switch resumes only onto the profile it
		// was parked on. Another profile, which may never have accepted the
		// policy, must not inherit the engagement.
		m := newTestManager()
		const userID = "user-6b"
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

		m.mu.Lock()
		m.sessions[userID] = &Session{UserID: userID, ConnectionID: "conn-2", State: StateConnected}
		m.resumeAutopilotLocked(userID, "conn-2")
		m.mu.Unlock()

		if got := m.AutopilotStateFor(userID); got != AutopilotOff {
			t.Fatalf("AutopilotStateFor after connecting another profile = %q, want %q", got, AutopilotOff)
		}
		if got := m.AutopilotConnectionIDFor(userID); got != "" {
			t.Fatalf("AutopilotConnectionIDFor after landing off = %q, want empty", got)
		}
	})

	t.Run("reconnect_by_quick_connect_lands_off_not_on", func(t *testing.T) {
		m := newTestManager()
		const userID = "user-6c"
		seedConnectedSession(m, userID, "conn-1")
		if _, _, err := m.EngageAutopilot(userID, "conn-1"); err != nil {
			t.Fatalf("engage err = %v, want nil", err)
		}
		if err := m.Disconnect(userID, ReasonRemote); err != nil {
			t.Fatalf("Disconnect err = %v, want nil", err)
		}
		if got := m.AutopilotConnectionIDFor(userID); got != "conn-1" {
			t.Fatalf("AutopilotConnectionIDFor while waiting = %q, want %q", got, "conn-1")
		}

		m.mu.Lock()
		m.sessions[userID] = &Session{UserID: userID, ConnectionID: "", State: StateConnected}
		m.resumeAutopilotLocked(userID, "")
		m.mu.Unlock()

		if got := m.AutopilotStateFor(userID); got != AutopilotOff {
			t.Fatalf("AutopilotStateFor after a quick connect = %q, want %q", got, AutopilotOff)
		}
	})

	t.Run("disconnect_while_off_changes_nothing", func(t *testing.T) {
		m := newTestManager()
		const userID = "user-7"
		seedConnectedSession(m, userID, "conn-1")

		if err := m.Disconnect(userID, ReasonRemote); err != nil {
			t.Fatalf("Disconnect err = %v, want nil", err)
		}

		if got := m.AutopilotStateFor(userID); got != AutopilotOff {
			t.Fatalf("AutopilotStateFor = %q, want %q", got, AutopilotOff)
		}
		m.mu.RLock()
		_, ok := m.autopilot[userID]
		m.mu.RUnlock()
		if ok {
			t.Fatalf("an autopilot record was created for a disconnect while off; ordinary play must be untouched (D-12)")
		}
	})
}
