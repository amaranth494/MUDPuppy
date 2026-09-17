// Package driver: this file is the home of the Phase 4 safety-limit suite
// (D-20). Every test here runs entirely on the fakes in driver_test.go --
// no network and no live model -- and the whole file must stay runnable in
// under a second so it can be the per-task feedback check every later
// Phase 4 plan runs before it touches anything else. It opens with the two
// tests that prove the harness itself can express a multi-decision stint
// and tell the truth about the autopilot switch; later plans extend this
// file with the cap, threshold, retry, and block-count tests those two
// tests are the bench for.
package driver

import (
	"strings"
	"testing"
	"time"

	"github.com/amaranth494/MudPuppy/internal/gemini"
	"github.com/amaranth494/MudPuppy/internal/session"
	"github.com/google/uuid"
)

// TestScriptedStint proves the harness can express a stint at all: three
// distinct decisions, each with its own scripted model answer and reviewer
// verdict, driven through the unchanged Phase 3 decision path in sequence
// for one user and one connection, with no network and no live model
// (D-20). This is the bench every later Phase 4 safety-limit test is built
// on top of.
func TestScriptedStint(t *testing.T) {
	sessions := &fakeSessions{window: "a room, exits north and south"}
	decisions := &fakeDecisionsStore{}
	notifier := &fakeNotifier{}
	commands := &fakeCommands{}
	models := &fakeModels{
		answers: []*gemini.Answer{
			{Reasoning: "looking around first", Command: "look"},
			{Reasoning: "heading north", Command: "north"},
			{Reasoning: "checking gear", Command: "inventory"},
		},
		reviewAnswers: []*gemini.ReviewAnswer{
			{Blocked: false, Reason: "clear, an ordinary look command"},
			{Blocked: false, Reason: "clear, ordinary movement"},
			{Blocked: false, Reason: "clear, checking inventory"},
		},
	}
	d := New(sessions, &fakeProfiles{profile: testProfile()}, decisions, models, commands, notifier, testConfig())

	userID := uuid.New().String()
	connID := uuid.New().String()

	d.HandleEngage(userID, connID)
	d.HandleEngage(userID, connID)
	d.HandleEngage(userID, connID)

	sends := sessions.sendCalls()
	if len(sends) != 3 {
		t.Fatalf("expected exactly 3 sent commands, got %d", len(sends))
	}
	wantCommands := []string{"look", "north", "inventory"}
	for i, want := range wantCommands {
		if sends[i].command != want {
			t.Fatalf("expected send %d to be %q, got %q", i, want, sends[i].command)
		}
	}

	rows := decisions.rows()
	if len(rows) != 3 {
		t.Fatalf("expected exactly 3 stored decision rows, got %d", len(rows))
	}
	for i, row := range rows {
		if row.Outcome != "sent" {
			t.Fatalf("expected row %d outcome %q, got %q", i, "sent", row.Outcome)
		}
	}

	events := notifier.eventsSnapshot()
	if len(events) != 3 {
		t.Fatalf("expected exactly 3 decision notifications, got %d", len(events))
	}
	for i, ev := range events {
		if ev.Kind != "decision" {
			t.Fatalf("expected event %d kind %q, got %q", i, "decision", ev.Kind)
		}
	}

	if got := models.overrunCount(); got != 0 {
		t.Fatalf("expected zero overruns for a stint that never exceeds its scripted queue, got %d", got)
	}

	si := models.systemInstructionsSnapshot()
	if len(si) != 3 {
		t.Fatalf("expected exactly 3 recorded system instructions (one per decision), got %d", len(si))
	}
}

// TestFakeSessionsStateMachine proves the session double can no longer lie
// about the switch (T-4-17): every transition, driven by the double's own
// setters and by DisengageAutopilot itself, is asserted against the
// identical pure function in internal/session/autopilot.go for the same
// starting state, and AutopilotStateFor reads back exactly what each
// transition produced. The already-off case is the one the old fixed
// (session.AutopilotOff, true) return could never express honestly.
func TestFakeSessionsStateMachine(t *testing.T) {
	sessions := &fakeSessions{}
	userID := "u"

	// off -> on, via the double's engageState setter.
	gotState, gotChanged := sessions.engageState()
	wantState, wantChanged := session.Engage(session.AutopilotOff)
	if gotState != wantState || gotChanged != wantChanged {
		t.Fatalf("engage from off: got (%v, %v), want (%v, %v)", gotState, gotChanged, wantState, wantChanged)
	}
	if got := sessions.AutopilotStateFor(userID); got != wantState {
		t.Fatalf("AutopilotStateFor after engage: got %v, want %v", got, wantState)
	}

	// on -> waiting, via enterWaiting.
	gotState, gotChanged = sessions.enterWaiting()
	wantState, wantChanged = session.EnterWaiting(session.AutopilotOn)
	if gotState != wantState || gotChanged != wantChanged {
		t.Fatalf("enterWaiting from on: got (%v, %v), want (%v, %v)", gotState, gotChanged, wantState, wantChanged)
	}
	if got := sessions.AutopilotStateFor(userID); got != wantState {
		t.Fatalf("AutopilotStateFor after enterWaiting: got %v, want %v", got, wantState)
	}

	// waiting -> on, via resume.
	gotState, gotChanged = sessions.resume()
	wantState, wantChanged = session.Resume(session.AutopilotWaiting)
	if gotState != wantState || gotChanged != wantChanged {
		t.Fatalf("resume from waiting: got (%v, %v), want (%v, %v)", gotState, gotChanged, wantState, wantChanged)
	}
	if got := sessions.AutopilotStateFor(userID); got != wantState {
		t.Fatalf("AutopilotStateFor after resume: got %v, want %v", got, wantState)
	}

	// on -> off, via DisengageAutopilot itself -- the real caller path
	// HandleEngage's recordFailure exercises, not a bypass setter.
	gotState, gotChanged = sessions.DisengageAutopilot(userID, "test-disengage")
	wantState, wantChanged = session.Disengage(session.AutopilotOn)
	if gotState != wantState || gotChanged != wantChanged {
		t.Fatalf("DisengageAutopilot from on: got (%v, %v), want (%v, %v)", gotState, gotChanged, wantState, wantChanged)
	}
	if got := sessions.AutopilotStateFor(userID); got != wantState {
		t.Fatalf("AutopilotStateFor after DisengageAutopilot: got %v, want %v", got, wantState)
	}

	// Already off: DisengageAutopilot must report changed=false. This is
	// the case the old fixed (session.AutopilotOff, true) return could
	// never produce honestly, letting a safety-limit test pass against a
	// build that never actually disengages (T-4-17).
	gotState, gotChanged = sessions.DisengageAutopilot(userID, "test-disengage-again")
	if gotState != session.AutopilotOff || gotChanged != false {
		t.Fatalf("expected already-off DisengageAutopilot to return (off, false), got (%v, %v)", gotState, gotChanged)
	}
}

// TestScriptedStintSurvivesAFailureMidway proves the harness can describe
// the shape Phase 4's thresholds will later enforce: a stint where the
// middle decision fails must still leave the two clean decisions sent, the
// one failure stored with a failure kind, exactly one disengage recorded,
// and no panic. It asserts nothing about thresholds or counters -- neither
// exists yet; that is 04-03's work.
func TestScriptedStintSurvivesAFailureMidway(t *testing.T) {
	sessions := &fakeSessions{window: "a room"}
	decisions := &fakeDecisionsStore{}
	notifier := &fakeNotifier{}
	commands := &fakeCommands{}
	models := &fakeModels{
		answers: []*gemini.Answer{
			{Reasoning: "looking around first", Command: "look"},
			nil,
			{Reasoning: "heading north", Command: "north"},
		},
		errs: []error{
			nil,
			&gemini.Error{Kind: gemini.KindTransport, Message: "dial tcp: i/o timeout"},
			nil,
		},
		// Only two decisions ever reach the reviewer -- the middle one
		// fails at GenerateContent and returns before ReviewCommand is
		// ever called -- so this queue has two entries, indexed by
		// review-call order, not by decision number.
		reviewAnswers: []*gemini.ReviewAnswer{
			{Blocked: false, Reason: "clear"},
			{Blocked: false, Reason: "clear"},
		},
	}
	d := New(sessions, &fakeProfiles{profile: testProfile()}, decisions, models, commands, notifier, testConfig())

	userID := uuid.New().String()
	connID := uuid.New().String()

	d.HandleEngage(userID, connID)
	d.HandleEngage(userID, connID)
	d.HandleEngage(userID, connID)

	sends := sessions.sendCalls()
	if len(sends) != 2 {
		t.Fatalf("expected exactly 2 sent commands (the middle failure sends nothing), got %d", len(sends))
	}

	rows := decisions.rows()
	if len(rows) != 3 {
		t.Fatalf("expected exactly 3 stored decision rows, got %d", len(rows))
	}
	failureRows := 0
	for _, row := range rows {
		if row.Outcome == "failed" {
			failureRows++
			if row.FailureKind == "" {
				t.Fatalf("expected the failed row to carry a non-empty failure kind")
			}
		}
	}
	if failureRows != 1 {
		t.Fatalf("expected exactly 1 failure row, got %d", failureRows)
	}

	if got := len(sessions.disengages()); got != 1 {
		t.Fatalf("expected exactly 1 disengage call, got %d", got)
	}
}

// newPacedDriver builds a Driver over the given fakes with the three
// pacing durations set to millisecond values (04-RESEARCH Don't
// Hand-Roll), so every test in this file drives the loop on a timescale
// under a second with no fake-clock library. No test in this file relies
// on New's production defaults.
func newPacedDriver(sessions *fakeSessions, decisions *fakeDecisionsStore, notifier *fakeNotifier, commands *fakeCommands, models *fakeModels, settle, floor, spacing time.Duration) *Driver {
	d := New(sessions, &fakeProfiles{profile: testProfile()}, decisions, models, commands, notifier, testConfig())
	d.settleDelay = settle
	d.floorInterval = floor
	d.minSpacing = spacing
	return d
}

// TestLoop_Pacing proves D-05/D-06 in three sub-cases: the loop keeps
// deciding as output arrives, paced by a settle wait; a quiet game still
// gets one nudge after the floor interval; and rapid-fire output never
// raises the decision rate past the minimum spacing.
func TestLoop_Pacing(t *testing.T) {
	t.Run("output_driven_decisions_follow_each_signal_after_settling", func(t *testing.T) {
		sessions := &fakeSessions{window: "a room, exits north and south"}
		sessions.engageState()
		decisions := &fakeDecisionsStore{}
		notifier := &fakeNotifier{}
		commands := &fakeCommands{}
		models := &fakeModels{
			answers: []*gemini.Answer{
				{Reasoning: "looking around first", Command: "look"},
				{Reasoning: "heading north", Command: "north"},
				{Reasoning: "checking gear", Command: "inventory"},
			},
			reviewAnswers: []*gemini.ReviewAnswer{
				{Blocked: false, Reason: "clear"},
				{Blocked: false, Reason: "clear"},
				{Blocked: false, Reason: "clear"},
			},
		}
		const settle = 20 * time.Millisecond
		d := newPacedDriver(sessions, decisions, notifier, commands, models, settle, 5*time.Second, time.Millisecond)

		userID := uuid.New().String()
		connID := uuid.New().String()

		d.EngageLoop(userID, connID)
		waitForCalls(t, models, 1)

		fireTime := time.Now()
		sessions.fireOutput()
		waitForCalls(t, models, 2)
		if elapsed := time.Since(fireTime); elapsed < settle {
			t.Fatalf("expected the second decision to land no sooner than the settle wait (%v), landed after %v", settle, elapsed)
		}

		fireTime = time.Now()
		sessions.fireOutput()
		waitForCalls(t, models, 3)
		if elapsed := time.Since(fireTime); elapsed < settle {
			t.Fatalf("expected the third decision to land no sooner than the settle wait (%v), landed after %v", settle, elapsed)
		}

		d.StopLoop(userID)

		sends := sessions.sendCalls()
		if len(sends) != 3 {
			t.Fatalf("expected exactly 3 sent commands, got %d", len(sends))
		}
		wantCommands := []string{"look", "north", "inventory"}
		for i, want := range wantCommands {
			if sends[i].command != want {
				t.Fatalf("expected send %d to be %q, got %q", i, want, sends[i].command)
			}
		}
	})

	t.Run("a_quiet_game_still_gets_nudged_after_the_floor_interval", func(t *testing.T) {
		sessions := &fakeSessions{window: "a quiet room"}
		sessions.engageState()
		decisions := &fakeDecisionsStore{}
		notifier := &fakeNotifier{}
		commands := &fakeCommands{}
		models := &fakeModels{
			answers: []*gemini.Answer{
				{Reasoning: "looking around", Command: "look"},
				{Reasoning: "nudging a quiet game", Command: "look"},
			},
			reviewAnswers: []*gemini.ReviewAnswer{
				{Blocked: false, Reason: "clear"},
				{Blocked: false, Reason: "clear"},
			},
		}
		d := newPacedDriver(sessions, decisions, notifier, commands, models, 2*time.Millisecond, 30*time.Millisecond, 2*time.Millisecond)

		userID := uuid.New().String()
		connID := uuid.New().String()

		d.EngageLoop(userID, connID)
		waitForCalls(t, models, 1)

		// No output signal is ever fired in this sub-case; the second
		// decision can only come from the floor interval's own nudge.
		waitForCalls(t, models, 2)

		d.StopLoop(userID)

		assertStableCallCount(t, models, 2, 100*time.Millisecond)
		if got := len(sessions.sendCalls()); got != 2 {
			t.Fatalf("expected exactly one further decision (2 sends total) after the floor interval, got %d", got)
		}
	})

	t.Run("minimum_spacing_bounds_the_decision_rate", func(t *testing.T) {
		sessions := &fakeSessions{window: "a room"}
		sessions.engageState()
		decisions := &fakeDecisionsStore{}
		notifier := &fakeNotifier{}
		commands := &fakeCommands{}
		models := &fakeModels{
			answer:       &gemini.Answer{Reasoning: "looking around", Command: "look"},
			reviewAnswer: &gemini.ReviewAnswer{Blocked: false, Reason: "clear"},
		}
		const minSpacing = 25 * time.Millisecond
		d := newPacedDriver(sessions, decisions, notifier, commands, models, 2*time.Millisecond, 500*time.Millisecond, minSpacing)

		userID := uuid.New().String()
		connID := uuid.New().String()

		d.EngageLoop(userID, connID)
		waitForCalls(t, models, 1)

		// Fire the output signal continuously for a fixed window -- far
		// more often than minSpacing could ever let a decision through --
		// and prove the decision rate is bounded by the spacing, not by
		// how many signals arrived.
		const firingWindow = 150 * time.Millisecond
		ticker := time.NewTicker(time.Millisecond)
		stop := time.After(firingWindow)
	firingLoop:
		for {
			select {
			case <-ticker.C:
				sessions.fireOutput()
			case <-stop:
				break firingLoop
			}
		}
		ticker.Stop()

		waitForStableCallCount(models, 2*d.floorInterval)
		d.StopLoop(userID)

		got := models.callCount()
		if got <= 1 {
			t.Fatalf("expected more than the single synchronous decision once output kept arriving, got %d", got)
		}
		maxExpected := int(firingWindow/minSpacing) + 3
		if got > maxExpected {
			t.Fatalf("expected decisions bounded by the minimum spacing (~%d from %v of continuous signalling), got %d", maxExpected, firingWindow, got)
		}

		sends := sessions.sentAtSnapshot()
		for i := 1; i < len(sends); i++ {
			if gap := sends[i].Sub(sends[i-1]); gap < minSpacing {
				t.Fatalf("expected consecutive sends at least %v apart, got %v between send %d and %d", minSpacing, gap, i-1, i)
			}
		}
	})
}

// TestEngageLoop_Reassess proves D-08: every engage's first decision
// carries the reassess instruction and a fresh window snapshot; later
// decisions in the same stint carry neither the instruction nor anything
// from the first decision's own answer, so there is no plan to carry over.
func TestEngageLoop_Reassess(t *testing.T) {
	sessions := &fakeSessions{window: "first window: a dim room"}
	sessions.engageState()
	decisions := &fakeDecisionsStore{}
	notifier := &fakeNotifier{}
	commands := &fakeCommands{}
	models := &fakeModels{
		answers: []*gemini.Answer{
			{Reasoning: "PLAN-STAGE-ONE: heading to the vault next", Command: "look"},
			{Reasoning: "second decision, no plan carried over", Command: "north"},
		},
		reviewAnswers: []*gemini.ReviewAnswer{
			{Blocked: false, Reason: "clear"},
			{Blocked: false, Reason: "clear"},
		},
	}
	d := newPacedDriver(sessions, decisions, notifier, commands, models, 2*time.Millisecond, 5*time.Second, 2*time.Millisecond)

	userID := uuid.New().String()
	connID := uuid.New().String()

	d.EngageLoop(userID, connID)
	waitForCalls(t, models, 1)

	sessions.setWindow("second window: a very different, brightly lit hall")
	sessions.fireOutput()
	waitForCalls(t, models, 2)
	d.StopLoop(userID)

	si := models.systemInstructionsSnapshot()
	if len(si) != 2 {
		t.Fatalf("expected exactly 2 recorded system instructions, got %d", len(si))
	}
	if !strings.Contains(si[0], reassessInstruction()) {
		t.Fatalf("expected the first decision's system instruction to carry the reassess instruction")
	}
	if strings.Contains(si[1], reassessInstruction()) {
		t.Fatalf("expected the second decision's system instruction not to carry the reassess instruction")
	}

	windows := models.windowsSnapshot()
	if len(windows) != 2 {
		t.Fatalf("expected exactly 2 recorded windows (one per iteration), got %d", len(windows))
	}
	if !strings.Contains(windows[1], "second window") {
		t.Fatalf("expected the second iteration's window to reflect the updated snapshot, got %q", windows[1])
	}
	if strings.Contains(windows[1], "first window") {
		t.Fatalf("expected the second iteration's window not to carry the first iteration's text, got %q", windows[1])
	}
	if strings.Contains(si[1], "PLAN-STAGE-ONE") || strings.Contains(windows[1], "PLAN-STAGE-ONE") {
		t.Fatalf("expected nothing from the first iteration's answer to appear in the second iteration's prompt (no plan carried over)")
	}
}

// TestLoop_StopsWhenWheelGrabbed proves REQ-doc-wheel-grab-and-reengage's
// loop half (RESEARCH Pattern 2): StopLoop (the DisengageHook's target)
// cancels a loop asleep in its pacing wait at once, and also stops a loop
// from starting another iteration after a model call already in flight at
// the moment of the wheel-grab returns.
func TestLoop_StopsWhenWheelGrabbed(t *testing.T) {
	t.Run("stops_while_asleep_in_the_pacing_wait", func(t *testing.T) {
		sessions := &fakeSessions{window: "a room"}
		sessions.engageState()
		decisions := &fakeDecisionsStore{}
		notifier := &fakeNotifier{}
		commands := &fakeCommands{}
		models := &fakeModels{
			answer:       &gemini.Answer{Reasoning: "heading out", Command: "look"},
			reviewAnswer: &gemini.ReviewAnswer{Blocked: false, Reason: "clear"},
		}
		d := newPacedDriver(sessions, decisions, notifier, commands, models, 5*time.Millisecond, 200*time.Millisecond, 5*time.Millisecond)

		userID := uuid.New().String()
		connID := uuid.New().String()

		d.EngageLoop(userID, connID)
		waitForCalls(t, models, 1)

		// Let the loop settle into its pacing select (asleep, waiting for
		// output or the floor) before grabbing the wheel -- proven by
		// polling for a stable call count, never a fixed sleep.
		waitForStableCallCount(models, 100*time.Millisecond)

		d.StopLoop(userID)

		for i := 0; i < 5; i++ {
			sessions.fireOutput()
		}

		assertStableCallCount(t, models, 1, 100*time.Millisecond)
		if got := len(sessions.sendCalls()); got != 1 {
			t.Fatalf("expected exactly the one synchronous send, got %d", got)
		}
	})

	t.Run("stops_after_an_in_flight_model_call_returns", func(t *testing.T) {
		sessions := &fakeSessions{window: "a room"}
		sessions.engageState()
		decisions := &fakeDecisionsStore{}
		notifier := &fakeNotifier{}
		commands := &fakeCommands{}
		models := &fakeModels{
			answers: []*gemini.Answer{
				{Reasoning: "first", Command: "look"},
				{Reasoning: "second, mid-flight when grabbed", Command: "north"},
			},
			reviewAnswers: []*gemini.ReviewAnswer{
				{Blocked: false, Reason: "clear"},
				{Blocked: false, Reason: "clear"},
			},
		}
		d := newPacedDriver(sessions, decisions, notifier, commands, models, 2*time.Millisecond, 500*time.Millisecond, 2*time.Millisecond)

		userID := uuid.New().String()
		connID := uuid.New().String()

		d.EngageLoop(userID, connID)
		waitForCalls(t, models, 1)

		block := make(chan struct{})
		models.setBlock(block)
		sessions.fireOutput()
		waitForCalls(t, models, 2) // the second call is now blocked mid-flight

		d.StopLoop(userID)
		close(block)

		waitForSendCount(t, sessions, 2)
		assertStableCallCount(t, models, 2, 100*time.Millisecond)
	})
}

// TestLoop_NothingIssuedWhileWaiting proves D-19 plus D-08's "a resume is
// an engage": a loop whose session state drops to waiting mid-run issues
// nothing further and stops itself (the post-wake AutopilotStateFor check,
// defence in depth alongside StopLoop -- RESEARCH Pattern 2); a resume
// starts a fresh loop whose first decision carries the reassess
// instruction.
func TestLoop_NothingIssuedWhileWaiting(t *testing.T) {
	sessions := &fakeSessions{window: "a room"}
	sessions.engageState()
	decisions := &fakeDecisionsStore{}
	notifier := &fakeNotifier{}
	commands := &fakeCommands{}
	models := &fakeModels{
		answers: []*gemini.Answer{
			{Reasoning: "before the drop", Command: "look"},
			{Reasoning: "after the resume", Command: "north"},
		},
		reviewAnswers: []*gemini.ReviewAnswer{
			{Blocked: false, Reason: "clear"},
			{Blocked: false, Reason: "clear"},
		},
	}
	d := newPacedDriver(sessions, decisions, notifier, commands, models, 2*time.Millisecond, 500*time.Millisecond, 2*time.Millisecond)

	userID := uuid.New().String()
	connID := uuid.New().String()

	d.EngageLoop(userID, connID)
	waitForCalls(t, models, 1)

	sessions.enterWaiting()
	for i := 0; i < 5; i++ {
		sessions.fireOutput()
	}

	assertStableCallCount(t, models, 1, 100*time.Millisecond)
	if got := len(sessions.sendCalls()); got != 1 {
		t.Fatalf("expected zero commands issued while waiting (only the pre-disconnect send present), got %d", got)
	}

	sessions.resume()
	d.EngageLoop(userID, connID)
	waitForCalls(t, models, 2)
	d.StopLoop(userID)

	sends := sessions.sendCalls()
	if len(sends) != 2 {
		t.Fatalf("expected exactly 2 sends (before the drop, and after the resume), got %d", len(sends))
	}

	si := models.systemInstructionsSnapshot()
	if len(si) == 0 || !strings.Contains(si[len(si)-1], reassessInstruction()) {
		t.Fatalf("expected the first post-resume decision's system instruction to carry the reassess instruction")
	}
}

// TestLoop_NoAIReconnect proves, mechanically rather than by claim, that
// nothing exercised by this file's own engage/wait/resume/wheel-grab
// scenarios ever calls a reconnect-shaped method on the session double
// (D-19, DEC-reconnect-is-connection-toggle). See fakeSessions'
// reconnectCalls field comment (internal/driver/driver_test.go): the
// Sessions interface has no such method today, so the driver has no way to
// call one, and this counter exists so a future interface change could not
// silently reintroduce one without a test noticing it move off zero.
// TestManager_LoopStopsOnPark (internal/session/manager_test.go, named
// TestManager_DisengageHookFires there) covers the manager half of this
// same guarantee; it is not duplicated here.
func TestLoop_NoAIReconnect(t *testing.T) {
	sessions := &fakeSessions{window: "a room"}
	sessions.engageState()
	decisions := &fakeDecisionsStore{}
	notifier := &fakeNotifier{}
	commands := &fakeCommands{}
	models := &fakeModels{
		answers: []*gemini.Answer{
			{Reasoning: "first", Command: "look"},
			{Reasoning: "resumed", Command: "north"},
		},
		reviewAnswers: []*gemini.ReviewAnswer{
			{Blocked: false, Reason: "clear"},
			{Blocked: false, Reason: "clear"},
		},
	}
	d := newPacedDriver(sessions, decisions, notifier, commands, models, 2*time.Millisecond, 500*time.Millisecond, 2*time.Millisecond)

	userID := uuid.New().String()
	connID := uuid.New().String()

	d.EngageLoop(userID, connID)
	waitForCalls(t, models, 1)

	sessions.enterWaiting()
	sessions.fireOutput()
	assertStableCallCount(t, models, 1, 50*time.Millisecond)

	sessions.resume()
	d.EngageLoop(userID, connID)
	waitForCalls(t, models, 2)

	d.StopLoop(userID)

	if got := sessions.reconnectCallCount(); got != 0 {
		t.Fatalf("expected zero reconnect-shaped calls on the session double, got %d", got)
	}
}
