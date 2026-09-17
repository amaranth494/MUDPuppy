package session

import (
	"errors"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

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

// pollUntil polls cond every millisecond until it returns true or the
// deadline elapses, following waitForCalls' exact deadline-polling shape
// (internal/driver/driver_test.go) rather than a fixed sleep-then-check.
func pollUntil(t *testing.T, timeout time.Duration, cond func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(time.Millisecond)
	}
	return cond()
}

// TestManager_OutputSignalFires proves the per-user output-arrived wakeup
// channel (04-03-01, D-06): a signal arrives after appendOutputWindow is
// called; a second append while the first signal is unread does not block
// and leaves exactly one pending signal (coalescing, so a burst of game
// output never piles up sends no one is reading); and a user with no
// output yet has an empty, non-nil channel.
func TestManager_OutputSignalFires(t *testing.T) {
	t.Run("signal_arrives_after_append", func(t *testing.T) {
		m := newTestManager()
		const userID = "signal-user-1"

		sig := m.OutputSignal(userID)
		m.appendOutputWindow(userID, []byte("a room"))

		select {
		case <-sig:
		default:
			t.Fatalf("expected a pending signal after appendOutputWindow, got none")
		}
	})

	t.Run("a_second_append_while_unread_does_not_block_and_stays_at_one", func(t *testing.T) {
		m := newTestManager()
		const userID = "signal-user-2"

		sig := m.OutputSignal(userID)

		done := make(chan struct{})
		go func() {
			m.appendOutputWindow(userID, []byte("first"))
			m.appendOutputWindow(userID, []byte("second"))
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatalf("appendOutputWindow blocked on a full signal channel")
		}

		select {
		case <-sig:
		default:
			t.Fatalf("expected exactly one pending signal after two appends, got none")
		}
		select {
		case <-sig:
			t.Fatalf("expected the second append to coalesce into the same pending signal, but a second signal was also readable")
		default:
		}
	})

	t.Run("a_user_with_no_output_has_an_empty_non_nil_channel", func(t *testing.T) {
		m := newTestManager()
		const userID = "signal-user-3"

		sig := m.OutputSignal(userID)
		if sig == nil {
			t.Fatalf("expected a non-nil channel for a user with no output yet")
		}
		select {
		case <-sig:
			t.Fatalf("expected no pending signal for a user with no output yet")
		default:
		}
	})
}

// TestManager_DisengageHookFires proves the symmetric stop signal
// (04-03-01, D-27/Pattern 2): the hook fires once with the right user id on
// a real disengage, does not fire when the switch was already off
// (changed=false), and fires on a park caused by a disconnect.
func TestManager_DisengageHookFires(t *testing.T) {
	t.Run("fires_once_on_a_real_disengage", func(t *testing.T) {
		m := newTestManager()
		const userID = "disengage-user-1"
		seedConnectedSession(m, userID, "conn-1")
		if _, _, err := m.EngageAutopilot(userID, "conn-1"); err != nil {
			t.Fatalf("engage err = %v, want nil", err)
		}

		fired := make(chan string, 4)
		m.SetDisengageHook(func(uid string) { fired <- uid })

		if _, changed := m.DisengageAutopilot(userID, "disengage"); !changed {
			t.Fatalf("expected the disengage to change state")
		}

		if !pollUntil(t, 2*time.Second, func() bool { return len(fired) >= 1 }) {
			t.Fatalf("timed out waiting for the disengage hook to fire")
		}
		if got := <-fired; got != userID {
			t.Fatalf("disengage hook fired with user id %q, want %q", got, userID)
		}
		if len(fired) != 0 {
			t.Fatalf("expected exactly one hook firing, got an extra one queued")
		}
	})

	t.Run("does_not_fire_when_already_off", func(t *testing.T) {
		m := newTestManager()
		const userID = "disengage-user-2"

		fired := make(chan string, 4)
		m.SetDisengageHook(func(uid string) { fired <- uid })

		if _, changed := m.DisengageAutopilot(userID, "disengage"); changed {
			t.Fatalf("expected an already-off disengage to report changed=false")
		}

		// Give any wrongly-fired goroutine a moment to land, then confirm
		// nothing did.
		time.Sleep(20 * time.Millisecond)
		if len(fired) != 0 {
			t.Fatalf("expected the disengage hook not to fire on an already-off disengage, got %d firing(s)", len(fired))
		}
	})

	t.Run("fires_on_a_park_caused_by_disconnect", func(t *testing.T) {
		m := newTestManager()
		const userID = "disengage-user-3"
		seedConnectedSession(m, userID, "conn-1")
		if _, _, err := m.EngageAutopilot(userID, "conn-1"); err != nil {
			t.Fatalf("engage err = %v, want nil", err)
		}

		fired := make(chan string, 4)
		m.SetDisengageHook(func(uid string) { fired <- uid })

		if err := m.Disconnect(userID, ReasonRemote); err != nil {
			t.Fatalf("Disconnect err = %v, want nil", err)
		}

		if !pollUntil(t, 2*time.Second, func() bool { return len(fired) >= 1 }) {
			t.Fatalf("timed out waiting for the disengage hook to fire on park")
		}
		if got := <-fired; got != userID {
			t.Fatalf("disengage hook fired with user id %q, want %q", got, userID)
		}
	})
}

// TestManager_EngageHookStillFiresOnResume proves the new DisengageHook did
// not disturb the existing resume path (04-03-01): a WAITING-to-ON resume
// still fires the engage hook exactly as it did before this plan.
func TestManager_EngageHookStillFiresOnResume(t *testing.T) {
	m := newTestManager()
	const userID = "resume-user-1"
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

	type engageCall struct {
		userID, connectionID string
		epoch                uint64
	}
	fired := make(chan engageCall, 4)
	m.SetEngageHook(func(uid, connID string, epoch uint64) { fired <- engageCall{uid, connID, epoch} })

	m.mu.Lock()
	m.sessions[userID] = &Session{UserID: userID, ConnectionID: "conn-1", State: StateConnected}
	m.resumeAutopilotLocked(userID, "conn-1")
	m.mu.Unlock()

	if !pollUntil(t, 2*time.Second, func() bool { return len(fired) >= 1 }) {
		t.Fatalf("timed out waiting for the engage hook to fire on resume")
	}
	got := <-fired
	if got.userID != userID || got.connectionID != "conn-1" {
		t.Fatalf("engage hook fired with (%q, %q), want (%q, %q)", got.userID, got.connectionID, userID, "conn-1")
	}
	// Code review CR-01 of Phase 4: the engage was stint 1, the resume is
	// stint 2, and the hook is told so.
	if got.epoch != 2 {
		t.Fatalf("engage hook fired with epoch %d on resume, want 2", got.epoch)
	}
	if state, epoch := m.AutopilotEpochFor(userID); state != AutopilotOn || epoch != 2 {
		t.Fatalf("AutopilotEpochFor = (%q, %d), want (%q, 2)", state, epoch, AutopilotOn)
	}
}

// TestManager_StintEpoch pins the stint identity code review CR-01 of Phase
// 4 added: the epoch goes up by one every time the switch BECOMES On (an
// engage and a resume alike), is untouched by a repeated engage, a
// disengage or a park, and never repeats for a user.
func TestManager_StintEpoch(t *testing.T) {
	m := newTestManager()
	const userID = "epoch-user-1"
	seedConnectedSession(m, userID, "conn-1")

	if state, epoch := m.AutopilotEpochFor(userID); state != AutopilotOff || epoch != 0 {
		t.Fatalf("never engaged: AutopilotEpochFor = (%q, %d), want (%q, 0)", state, epoch, AutopilotOff)
	}

	_, changed, epoch, err := m.EngageAutopilotEpoch(userID, "conn-1")
	if err != nil || !changed || epoch != 1 {
		t.Fatalf("first engage = (changed %v, epoch %d, err %v), want (true, 1, nil)", changed, epoch, err)
	}

	_, changed, epoch, err = m.EngageAutopilotEpoch(userID, "conn-1")
	if err != nil || changed || epoch != 1 {
		t.Fatalf("repeated engage = (changed %v, epoch %d, err %v), want (false, 1, nil)", changed, epoch, err)
	}

	m.DisengageAutopilot(userID, "wheel-grab")
	if state, epoch := m.AutopilotEpochFor(userID); state != AutopilotOff || epoch != 1 {
		t.Fatalf("after disengage: AutopilotEpochFor = (%q, %d), want (%q, 1)", state, epoch, AutopilotOff)
	}

	_, changed, epoch, err = m.EngageAutopilotEpoch(userID, "conn-1")
	if err != nil || !changed || epoch != 2 {
		t.Fatalf("re-engage = (changed %v, epoch %d, err %v), want (true, 2, nil)", changed, epoch, err)
	}
}

// TestSendAICommand_RefusedOnEpochMismatch is the manager half of code
// review CR-01 of Phase 4: the owner takes the wheel and engages again while
// stint 1's decision is still waiting on the model. The switch reads On, so
// the state-only rule would send it; SendAICommand refuses it because it
// belongs to stint 1 and the switch is in stint 2.
func TestSendAICommand_RefusedOnEpochMismatch(t *testing.T) {
	m := newTestManager()
	userID := uuid.New().String()
	connID := uuid.New().String()
	seedConnectedSession(m, userID, connID)

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	m.mu.Lock()
	m.conns[userID] = client
	m.mu.Unlock()

	var mu sync.Mutex
	var wire strings.Builder
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := server.Read(buf)
			if err != nil {
				return
			}
			mu.Lock()
			wire.Write(buf[:n])
			mu.Unlock()
		}
	}()
	written := func() string {
		mu.Lock()
		defer mu.Unlock()
		return wire.String()
	}

	_, _, stint1, err := m.EngageAutopilotEpoch(userID, connID)
	if err != nil {
		t.Fatalf("engage: %v", err)
	}
	m.DisengageAutopilot(userID, "wheel-grab")
	_, _, stint2, err := m.EngageAutopilotEpoch(userID, connID)
	if err != nil {
		t.Fatalf("re-engage: %v", err)
	}
	if stint2 == stint1 {
		t.Fatalf("expected the re-engage to begin a new stint, got epoch %d twice", stint1)
	}
	if got := m.AutopilotStateFor(userID); got != AutopilotOn {
		t.Fatalf("AutopilotStateFor = %q, want %q", got, AutopilotOn)
	}

	if err := m.SendAICommand(userID, "stale-east", stint1); !errors.Is(err, ErrAutopilotNotOn) {
		t.Fatalf("expected ErrAutopilotNotOn for the previous stint's command while the switch is On, got %v", err)
	}
	if err := m.SendAICommand(userID, "fresh-north", stint2); err != nil {
		t.Fatalf("SendAICommand for the current stint: %v", err)
	}

	if !pollUntil(t, time.Second, func() bool { return strings.Contains(written(), "fresh-north\r\n") }) {
		t.Fatalf("expected the current stint's command on the wire, got %q", written())
	}
	if strings.Contains(written(), "stale-east") {
		t.Fatalf("the previous stint's command reached the wire: %q", written())
	}
}
