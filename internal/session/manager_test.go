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
		epochs := make(chan uint64, 4)
		m.SetDisengageHook(func(uid string, epoch uint64) { fired <- uid; epochs <- epoch })

		if _, changed := m.DisengageAutopilot(userID, "disengage"); !changed {
			t.Fatalf("expected the disengage to change state")
		}

		if !pollUntil(t, 2*time.Second, func() bool { return len(fired) >= 1 }) {
			t.Fatalf("timed out waiting for the disengage hook to fire")
		}
		if got := <-fired; got != userID {
			t.Fatalf("disengage hook fired with user id %q, want %q", got, userID)
		}
		// Code review WR-10 of Phase 4: the hook names the stint that ended.
		if got := <-epochs; got != 1 {
			t.Fatalf("disengage hook fired with epoch %d, want 1 (the stint that ended)", got)
		}
		if len(fired) != 0 {
			t.Fatalf("expected exactly one hook firing, got an extra one queued")
		}
	})

	t.Run("does_not_fire_when_already_off", func(t *testing.T) {
		m := newTestManager()
		const userID = "disengage-user-2"

		fired := make(chan string, 4)
		epochs := make(chan uint64, 4)
		m.SetDisengageHook(func(uid string, epoch uint64) { fired <- uid; epochs <- epoch })

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
		epochs := make(chan uint64, 4)
		m.SetDisengageHook(func(uid string, epoch uint64) { fired <- uid; epochs <- epoch })

		if err := m.Disconnect(userID, ReasonRemote); err != nil {
			t.Fatalf("Disconnect err = %v, want nil", err)
		}

		if !pollUntil(t, 2*time.Second, func() bool { return len(fired) >= 1 }) {
			t.Fatalf("timed out waiting for the disengage hook to fire on park")
		}
		if got := <-fired; got != userID {
			t.Fatalf("disengage hook fired with user id %q, want %q", got, userID)
		}
		if got := <-epochs; got != 1 {
			t.Fatalf("disengage hook fired with epoch %d on park, want 1 (the stint being parked)", got)
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

// TestAutopilotRecordCarriesBothWaitingReasons proves D-15's two
// independent waiting reasons round-trip through AutopilotWaitingReasons
// for all four combinations, and pins that Engage/Disengage/EnterWaiting/
// Resume still return exactly what they returned before this plan for
// every input -- a later refactor cannot quietly move reason logic into
// them (RESEARCH Pitfall 1).
func TestAutopilotRecordCarriesBothWaitingReasons(t *testing.T) {
	m := newTestManager()
	const userID = "reasons-user-1"

	combos := []struct {
		paused, lost bool
	}{
		{false, false},
		{true, false},
		{false, true},
		{true, true},
	}
	for _, c := range combos {
		m.mu.Lock()
		m.autopilot[userID] = &AutopilotRecord{State: AutopilotWaiting, PausedByOwner: c.paused, ConnectionLost: c.lost}
		m.mu.Unlock()

		gotPaused, gotLost := m.AutopilotWaitingReasons(userID)
		if gotPaused != c.paused || gotLost != c.lost {
			t.Errorf("combo %+v: AutopilotWaitingReasons = (%v, %v), want (%v, %v)", c, gotPaused, gotLost, c.paused, c.lost)
		}
	}

	if paused, lost := m.AutopilotWaitingReasons("never-engaged"); paused || lost {
		t.Errorf("AutopilotWaitingReasons for a never-engaged user = (%v, %v), want (false, false)", paused, lost)
	}

	pureTests := []struct {
		name        string
		fn          func(AutopilotState) (AutopilotState, bool)
		from        AutopilotState
		wantState   AutopilotState
		wantChanged bool
	}{
		{"off_Engage_on", Engage, AutopilotOff, AutopilotOn, true},
		{"on_Engage_unchanged", Engage, AutopilotOn, AutopilotOn, false},
		{"waiting_Engage_unchanged", Engage, AutopilotWaiting, AutopilotWaiting, false},
		{"on_Disengage_off", Disengage, AutopilotOn, AutopilotOff, true},
		{"waiting_Disengage_off", Disengage, AutopilotWaiting, AutopilotOff, true},
		{"off_Disengage_unchanged", Disengage, AutopilotOff, AutopilotOff, false},
		{"on_EnterWaiting_waiting", EnterWaiting, AutopilotOn, AutopilotWaiting, true},
		{"off_EnterWaiting_unchanged", EnterWaiting, AutopilotOff, AutopilotOff, false},
		{"waiting_EnterWaiting_unchanged", EnterWaiting, AutopilotWaiting, AutopilotWaiting, false},
		{"waiting_Resume_on", Resume, AutopilotWaiting, AutopilotOn, true},
		{"off_Resume_unchanged", Resume, AutopilotOff, AutopilotOff, false},
		{"on_Resume_unchanged", Resume, AutopilotOn, AutopilotOn, false},
	}
	for _, tt := range pureTests {
		gotState, gotChanged := tt.fn(tt.from)
		if gotState != tt.wantState || gotChanged != tt.wantChanged {
			t.Errorf("%s: (state, changed) = (%q, %v), want (%q, %v)", tt.name, gotState, gotChanged, tt.wantState, tt.wantChanged)
		}
	}
}

// TestManager_ResumeRequiresBothReasonsClear proves T-5-01's mitigation: a
// pause and a disconnect can arrive in either order, and the switch only
// leaves Waiting once both reasons have cleared.
func TestManager_ResumeRequiresBothReasonsClear(t *testing.T) {
	t.Run("pause_then_disconnect_then_reconnect_then_owner_resume", func(t *testing.T) {
		m := newTestManager()
		const userID = "reqclear-user-1"
		seedConnectedSession(m, userID, "conn-1")
		_, _, epoch1, err := m.EngageAutopilotEpoch(userID, "conn-1")
		if err != nil {
			t.Fatalf("engage err = %v, want nil", err)
		}

		fired := make(chan struct{}, 4)
		m.SetEngageHook(func(uid, connID string, epoch uint64) { fired <- struct{}{} })

		if state, changed := m.PauseAutopilot(userID); !changed || state != AutopilotWaiting {
			t.Fatalf("PauseAutopilot = (%q, %v), want (%q, true)", state, changed, AutopilotWaiting)
		}

		if err := m.Disconnect(userID, ReasonRemote); err != nil {
			t.Fatalf("Disconnect err = %v, want nil", err)
		}

		m.mu.Lock()
		m.sessions[userID] = &Session{UserID: userID, ConnectionID: "conn-1", State: StateConnected}
		m.resumeAutopilotLocked(userID, "conn-1")
		m.mu.Unlock()

		if got := m.AutopilotStateFor(userID); got != AutopilotWaiting {
			t.Fatalf("AutopilotStateFor after reconnect = %q, want %q (a reconnect alone must not un-pause)", got, AutopilotWaiting)
		}
		if paused, lost := m.AutopilotWaitingReasons(userID); !paused || lost {
			t.Fatalf("AutopilotWaitingReasons after reconnect = (%v, %v), want (true, false)", paused, lost)
		}
		time.Sleep(20 * time.Millisecond)
		if len(fired) != 0 {
			t.Fatalf("engage hook fired after a reconnect while the owner's pause still stands")
		}

		state, changed := m.ResumeAutopilotByOwner(userID)
		if !changed || state != AutopilotOn {
			t.Fatalf("ResumeAutopilotByOwner = (%q, %v), want (%q, true)", state, changed, AutopilotOn)
		}
		if !pollUntil(t, 2*time.Second, func() bool { return len(fired) >= 1 }) {
			t.Fatalf("timed out waiting for the engage hook to fire on owner resume")
		}
		<-fired
		if len(fired) != 0 {
			t.Fatalf("expected exactly one engage hook firing, got an extra one queued")
		}
		if _, epoch := m.AutopilotEpochFor(userID); epoch != epoch1+1 {
			t.Fatalf("epoch after owner resume = %d, want %d", epoch, epoch1+1)
		}
	})

	t.Run("disconnect_then_pause_then_owner_resume_then_reconnect", func(t *testing.T) {
		m := newTestManager()
		const userID = "reqclear-user-2"
		seedConnectedSession(m, userID, "conn-1")
		_, _, epoch1, err := m.EngageAutopilotEpoch(userID, "conn-1")
		if err != nil {
			t.Fatalf("engage err = %v, want nil", err)
		}

		fired := make(chan struct{}, 4)
		m.SetEngageHook(func(uid, connID string, epoch uint64) { fired <- struct{}{} })

		if err := m.Disconnect(userID, ReasonRemote); err != nil {
			t.Fatalf("Disconnect err = %v, want nil", err)
		}
		if state, changed := m.PauseAutopilot(userID); changed || state != AutopilotWaiting {
			t.Fatalf("PauseAutopilot while already waiting = (%q, %v), want (%q, false)", state, changed, AutopilotWaiting)
		}
		if paused, lost := m.AutopilotWaitingReasons(userID); !paused || !lost {
			t.Fatalf("AutopilotWaitingReasons after disconnect+pause = (%v, %v), want (true, true)", paused, lost)
		}

		// Owner resume first: PausedByOwner clears, but ConnectionLost still
		// stands, so the switch must stay Waiting with no hook fire.
		if state, changed := m.ResumeAutopilotByOwner(userID); changed || state != AutopilotWaiting {
			t.Fatalf("ResumeAutopilotByOwner while still disconnected = (%q, %v), want (%q, false)", state, changed, AutopilotWaiting)
		}
		time.Sleep(20 * time.Millisecond)
		if len(fired) != 0 {
			t.Fatalf("engage hook fired after an owner resume while the connection is still lost")
		}

		m.mu.Lock()
		m.sessions[userID] = &Session{UserID: userID, ConnectionID: "conn-1", State: StateConnected}
		m.resumeAutopilotLocked(userID, "conn-1")
		m.mu.Unlock()

		if !pollUntil(t, 2*time.Second, func() bool { return len(fired) >= 1 }) {
			t.Fatalf("timed out waiting for the engage hook to fire on reconnect")
		}
		if got := m.AutopilotStateFor(userID); got != AutopilotOn {
			t.Fatalf("AutopilotStateFor after reconnect clears the last reason = %q, want %q", got, AutopilotOn)
		}
		if _, epoch := m.AutopilotEpochFor(userID); epoch != epoch1+1 {
			t.Fatalf("epoch after reconnect resume = %d, want %d", epoch, epoch1+1)
		}
	})
}

// TestManager_ResumeByOwnerFiresEngageHookOnce proves the pause/resume
// hook-firing discipline with the connection healthy throughout: the
// disengage hook fires once on pause, the engage hook fires once on
// resume, and the resumed epoch is strictly greater than the paused one.
func TestManager_ResumeByOwnerFiresEngageHookOnce(t *testing.T) {
	m := newTestManager()
	const userID = "fireonce-user-1"
	seedConnectedSession(m, userID, "conn-1")
	_, _, pausedEpoch, err := m.EngageAutopilotEpoch(userID, "conn-1")
	if err != nil {
		t.Fatalf("engage err = %v, want nil", err)
	}

	disengageFired := make(chan uint64, 4)
	engageFired := make(chan uint64, 4)
	m.SetDisengageHook(func(uid string, epoch uint64) { disengageFired <- epoch })
	m.SetEngageHook(func(uid, connID string, epoch uint64) { engageFired <- epoch })

	if state, changed := m.PauseAutopilot(userID); !changed || state != AutopilotWaiting {
		t.Fatalf("PauseAutopilot = (%q, %v), want (%q, true)", state, changed, AutopilotWaiting)
	}
	if !pollUntil(t, 2*time.Second, func() bool { return len(disengageFired) >= 1 }) {
		t.Fatalf("timed out waiting for the disengage hook to fire on pause")
	}
	if got := <-disengageFired; got != pausedEpoch {
		t.Fatalf("disengage hook fired with epoch %d, want %d", got, pausedEpoch)
	}
	if len(disengageFired) != 0 {
		t.Fatalf("expected exactly one disengage hook firing on pause, got an extra one queued")
	}

	state, changed := m.ResumeAutopilotByOwner(userID)
	if !changed || state != AutopilotOn {
		t.Fatalf("ResumeAutopilotByOwner = (%q, %v), want (%q, true)", state, changed, AutopilotOn)
	}
	if !pollUntil(t, 2*time.Second, func() bool { return len(engageFired) >= 1 }) {
		t.Fatalf("timed out waiting for the engage hook to fire on resume")
	}
	resumedEpoch := <-engageFired
	if len(engageFired) != 0 {
		t.Fatalf("expected exactly one engage hook firing on resume, got an extra one queued")
	}
	if resumedEpoch <= pausedEpoch {
		t.Fatalf("resumed epoch %d is not strictly greater than the paused epoch %d", resumedEpoch, pausedEpoch)
	}
}

// TestManager_PauseNeverResumesByItself proves D-13/D-15's core promise:
// once paused, nothing but an owner resume starts the switch again.
func TestManager_PauseNeverResumesByItself(t *testing.T) {
	m := newTestManager()
	const userID = "neverself-user-1"
	seedConnectedSession(m, userID, "conn-1")
	if _, _, _, err := m.EngageAutopilotEpoch(userID, "conn-1"); err != nil {
		t.Fatalf("engage err = %v, want nil", err)
	}

	engageFired := make(chan struct{}, 4)
	m.SetEngageHook(func(uid, connID string, epoch uint64) { engageFired <- struct{}{} })

	if state, changed := m.PauseAutopilot(userID); !changed || state != AutopilotWaiting {
		t.Fatalf("PauseAutopilot = (%q, %v), want (%q, true)", state, changed, AutopilotWaiting)
	}

	// A reconnect on the same connection while the owner's pause stands
	// must not resume it.
	m.mu.Lock()
	m.sessions[userID] = &Session{UserID: userID, ConnectionID: "conn-1", State: StateConnected}
	m.resumeAutopilotLocked(userID, "conn-1")
	m.mu.Unlock()
	if got := m.AutopilotStateFor(userID); got != AutopilotWaiting {
		t.Fatalf("AutopilotStateFor after reconnect = %q, want %q", got, AutopilotWaiting)
	}

	// A second pause is a no-op that leaves the state exactly where it was.
	if state, changed := m.PauseAutopilot(userID); changed || state != AutopilotWaiting {
		t.Fatalf("second PauseAutopilot = (%q, %v), want (%q, false)", state, changed, AutopilotWaiting)
	}

	// A status change: a disconnect on top of an owner pause records its
	// own reason but must not resume anything either.
	if err := m.Disconnect(userID, ReasonRemote); err != nil {
		t.Fatalf("Disconnect err = %v, want nil", err)
	}
	if got := m.AutopilotStateFor(userID); got != AutopilotWaiting {
		t.Fatalf("AutopilotStateFor after a disconnect on top of a pause = %q, want %q", got, AutopilotWaiting)
	}

	time.Sleep(20 * time.Millisecond)
	if len(engageFired) != 0 {
		t.Fatalf("engage hook fired without an owner resume, got %d firing(s)", len(engageFired))
	}
}

// TestManager_WheelGrabWhilePausedLandsOff proves D-16: the wheel-grab rule
// is the same in every engaged state the owner can type in.
//
// Code review CR-01 of Phase 5: this test used to call DisengageAutopilot
// directly, the call applyWheelGrab never reached for a paused switch, so it
// passed while the behaviour it is named for did not exist. It now goes
// through applyWheelGrab itself, the only place a typed command arrives, and
// pins the guard (AutopilotGrabbable) in each state.
func TestManager_WheelGrabWhilePausedLandsOff(t *testing.T) {
	t.Run("the_guard_in_each_state", func(t *testing.T) {
		m := newTestManager()
		const userID = "wheelpause-guard-1"

		if state, ok := m.AutopilotGrabbable(userID); ok || state != AutopilotOff {
			t.Fatalf("never engaged: AutopilotGrabbable = (%q, %v), want (%q, false)", state, ok, AutopilotOff)
		}

		seedConnectedSession(m, userID, "conn-1")
		if _, _, _, err := m.EngageAutopilotEpoch(userID, "conn-1"); err != nil {
			t.Fatalf("engage err = %v, want nil", err)
		}
		if state, ok := m.AutopilotGrabbable(userID); !ok || state != AutopilotOn {
			t.Fatalf("on: AutopilotGrabbable = (%q, %v), want (%q, true)", state, ok, AutopilotOn)
		}

		m.PauseAutopilot(userID)
		if state, ok := m.AutopilotGrabbable(userID); !ok || state != AutopilotWaiting {
			t.Fatalf("paused: AutopilotGrabbable = (%q, %v), want (%q, true)", state, ok, AutopilotWaiting)
		}

		m.DisengageAutopilot(userID, "disengage")
		if state, ok := m.AutopilotGrabbable(userID); ok || state != AutopilotOff {
			t.Fatalf("off: AutopilotGrabbable = (%q, %v), want (%q, false)", state, ok, AutopilotOff)
		}
	})

	t.Run("a_disconnect_only_waiting_is_not_grabbable", func(t *testing.T) {
		m := newTestManager()
		const userID = "wheelpause-guard-2"
		seedConnectedSession(m, userID, "conn-1")
		if _, _, _, err := m.EngageAutopilotEpoch(userID, "conn-1"); err != nil {
			t.Fatalf("engage err = %v, want nil", err)
		}
		if err := m.Disconnect(userID, ReasonRemote); err != nil {
			t.Fatalf("Disconnect err = %v, want nil", err)
		}
		if state, ok := m.AutopilotGrabbable(userID); ok || state != AutopilotWaiting {
			t.Fatalf("connection lost only: AutopilotGrabbable = (%q, %v), want (%q, false)", state, ok, AutopilotWaiting)
		}
	})

	t.Run("typing_while_paused_lands_off_and_stops_the_stint", func(t *testing.T) {
		m := newTestManager()
		const userID = "wheelpause-user-1"
		seedConnectedSession(m, userID, "conn-1")
		if _, _, _, err := m.EngageAutopilotEpoch(userID, "conn-1"); err != nil {
			t.Fatalf("engage err = %v, want nil", err)
		}
		if state, changed := m.PauseAutopilot(userID); !changed || state != AutopilotWaiting {
			t.Fatalf("PauseAutopilot = (%q, %v), want (%q, true)", state, changed, AutopilotWaiting)
		}

		// Registered after the pause so the pause's own hook firing is not
		// counted: what is asserted is the grab's.
		time.Sleep(20 * time.Millisecond)
		disengageFired := make(chan uint64, 4)
		m.SetDisengageHook(func(uid string, epoch uint64) { disengageFired <- epoch })

		grabbed, state := applyWheelGrab(m, userID, "user")
		if !grabbed || state != AutopilotOff {
			t.Fatalf("applyWheelGrab while paused = (%v, %q), want (true, %q)", grabbed, state, AutopilotOff)
		}
		if got := m.AutopilotStateFor(userID); got != AutopilotOff {
			t.Fatalf("AutopilotStateFor after the grab = %q, want %q", got, AutopilotOff)
		}
		if paused, lost := m.AutopilotWaitingReasons(userID); paused || lost {
			t.Fatalf("AutopilotWaitingReasons after wheel-grab = (%v, %v), want (false, false)", paused, lost)
		}
		if !pollUntil(t, 2*time.Second, func() bool { return len(disengageFired) >= 1 }) {
			t.Fatalf("timed out waiting for the disengage hook to fire on a wheel-grab while paused")
		}

		// A second typed command finds the switch Off and grabs nothing.
		if grabbed, state := applyWheelGrab(m, userID, "user"); grabbed || state != AutopilotOff {
			t.Fatalf("second applyWheelGrab = (%v, %q), want (false, %q)", grabbed, state, AutopilotOff)
		}
	})
}

// TestManager_PauseStopsTheLoopAndKeepsReading is D-14's mechanical half:
// pausing stops the loop (the disengage hook fires, the engage hook does
// not) but the Immediate Context window keeps filling exactly as it does
// with autopilot fully on.
func TestManager_PauseStopsTheLoopAndKeepsReading(t *testing.T) {
	m := newTestManager()
	const userID = "pauseread-user-1"
	seedConnectedSession(m, userID, "conn-1")
	if _, _, _, err := m.EngageAutopilotEpoch(userID, "conn-1"); err != nil {
		t.Fatalf("engage err = %v, want nil", err)
	}

	disengageFired := make(chan struct{}, 4)
	engageFired := make(chan struct{}, 4)
	m.SetDisengageHook(func(uid string, epoch uint64) { disengageFired <- struct{}{} })
	m.SetEngageHook(func(uid, connID string, epoch uint64) { engageFired <- struct{}{} })

	if state, changed := m.PauseAutopilot(userID); !changed || state != AutopilotWaiting {
		t.Fatalf("PauseAutopilot = (%q, %v), want (%q, true)", state, changed, AutopilotWaiting)
	}
	if !pollUntil(t, 2*time.Second, func() bool { return len(disengageFired) >= 1 }) {
		t.Fatalf("timed out waiting for the disengage hook to fire on pause")
	}

	before := m.RecentOutputSnapshot(userID)
	m.appendOutputWindow(userID, []byte("A room. Exits: north.\n"))
	after := m.RecentOutputSnapshot(userID)
	if len(after) <= len(before) {
		t.Fatalf("RecentOutputSnapshot did not grow while paused: before %q, after %q", before, after)
	}

	time.Sleep(20 * time.Millisecond)
	if len(disengageFired) != 1 {
		t.Fatalf("expected exactly one disengage hook firing, got %d", len(disengageFired))
	}
	if len(engageFired) != 0 {
		t.Fatalf("expected no engage hook firing while paused, got %d", len(engageFired))
	}
}
