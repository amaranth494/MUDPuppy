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
// the shape Phase 4's thresholds enforce: a stint where the middle decision
// fails must still leave the two clean decisions sent, the one failure
// stored with a failure kind, and no panic. The middle failure (a transient
// transport error, D-15) is the stint's first consecutive failure, one
// short of the default threshold of three, so it does not disengage --
// updated by 04-04 from this test's original assertion of an immediate
// disengage, which predates the threshold this plan adds.
func TestScriptedStintSurvivesAFailureMidway(t *testing.T) {
	sessions := &fakeSessions{window: "a room"}
	sessions.engageState()
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

	if got := len(sessions.disengages()); got != 0 {
		t.Fatalf("expected zero disengage calls (one transient failure is below the default threshold of 3), got %d", got)
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

		d.EngageLoop(userID, connID, sessions.currentEpoch())
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

		// waitForCalls returns when the third model call starts; the send
		// follows the review, so wait for it before reading sendCalls.
		waitForSendCount(t, sessions, 3)
		d.StopLoop(userID, sessions.currentEpoch())

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

		d.EngageLoop(userID, connID, sessions.currentEpoch())
		waitForCalls(t, models, 1)

		// No output signal is ever fired in this sub-case; the second
		// decision can only come from the floor interval's own nudge.
		waitForCalls(t, models, 2)

		// waitForCalls returns when the second model call STARTS. Since code
		// review WR-02 of Phase 4, StopLoop cancels the stint's context and a
		// decision still in flight is dropped rather than sent, so wait for
		// the nudge's send before stopping -- exactly as the sub-test above
		// does -- or a StopLoop that wins the race leaves 1 send, not 2.
		waitForSendCount(t, sessions, 2)
		d.StopLoop(userID, sessions.currentEpoch())

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

		d.EngageLoop(userID, connID, sessions.currentEpoch())
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
		d.StopLoop(userID, sessions.currentEpoch())

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

	d.EngageLoop(userID, connID, sessions.currentEpoch())
	waitForCalls(t, models, 1)

	sessions.setWindow("second window: a very different, brightly lit hall")
	sessions.fireOutput()
	waitForCalls(t, models, 2)
	d.StopLoop(userID, sessions.currentEpoch())

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

		d.EngageLoop(userID, connID, sessions.currentEpoch())
		waitForCalls(t, models, 1)

		// Let the loop settle into its pacing select (asleep, waiting for
		// output or the floor) before grabbing the wheel -- proven by
		// polling for a stable call count, never a fixed sleep.
		waitForStableCallCount(models, 100*time.Millisecond)

		d.StopLoop(userID, sessions.currentEpoch())

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

		d.EngageLoop(userID, connID, sessions.currentEpoch())
		waitForCalls(t, models, 1)

		block := make(chan struct{})
		models.setBlock(block)
		sessions.fireOutput()
		waitForCalls(t, models, 2) // the second call is now blocked mid-flight

		// The real order of events: the manager flips the switch off, then
		// its disengage hook stops the loop, and only then does the model
		// answer. The decision that was in flight must never reach the game
		// (found on staging 2026-09-17: it was sent one second after off),
		// and being dropped is not a failure.
		sessions.disengageState()
		d.StopLoop(userID, sessions.currentEpoch())
		close(block)

		assertStableCallCount(t, models, 2, 100*time.Millisecond)
		if got := len(sessions.sendCalls()); got != 1 {
			t.Fatalf("expected only the first decision's send, the in-flight one dropped; got %d sends", got)
		}
		if got := len(commands.dispatchCalls()); got != 1 {
			t.Fatalf("expected no ICM dispatch for the dropped decision, got %d dispatches", got)
		}
		for _, row := range decisions.rows() {
			if row.Outcome == "failed" {
				t.Fatalf("a dropped in-flight decision must not be stored as a failure")
			}
		}
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

	d.EngageLoop(userID, connID, sessions.currentEpoch())
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
	d.EngageLoop(userID, connID, sessions.currentEpoch())
	waitForCalls(t, models, 2)
	d.StopLoop(userID, sessions.currentEpoch())

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

	d.EngageLoop(userID, connID, sessions.currentEpoch())
	waitForCalls(t, models, 1)

	sessions.enterWaiting()
	sessions.fireOutput()
	assertStableCallCount(t, models, 1, 50*time.Millisecond)

	sessions.resume()
	d.EngageLoop(userID, connID, sessions.currentEpoch())
	waitForCalls(t, models, 2)

	d.StopLoop(userID, sessions.currentEpoch())

	if got := sessions.reconnectCallCount(); got != 0 {
		t.Fatalf("expected zero reconnect-shaped calls on the session double, got %d", got)
	}
}

// TestLoop_CallCap proves D-14/DR-3.1-02: the loop halts before making the
// call that would exceed the session call cap, autopilot lands off, and the
// panel/terminal notice is the locked cap-reached sentence. A cap of 2
// permits exactly one full iteration (the decision call plus the reviewer
// call, DR-3.1-02's own counting rule), so the second iteration's very
// first reservation -- the decision call -- is the cap's third call and is
// refused before it is made. A fresh EngageLoop resets the counter to zero.
func TestLoop_CallCap(t *testing.T) {
	sessions := &fakeSessions{window: "a room"}
	sessions.engageState()
	decisions := &fakeDecisionsStore{}
	notifier := &fakeNotifier{}
	commands := &fakeCommands{}
	models := &fakeModels{
		answer:       &gemini.Answer{Reasoning: "heading out", Command: "look"},
		reviewAnswer: &gemini.ReviewAnswer{Blocked: false, Reason: "clear"},
	}
	profile := testProfile()
	callCap := 2
	profile.AISettings.CallCap = &callCap
	d := New(sessions, &fakeProfiles{profile: profile}, decisions, models, commands, notifier, testConfig())
	d.settleDelay = 2 * time.Millisecond
	d.floorInterval = 5 * time.Second
	d.minSpacing = 2 * time.Millisecond

	userID := uuid.New().String()
	connID := uuid.New().String()

	d.EngageLoop(userID, connID, sessions.currentEpoch())
	waitForSendCount(t, sessions, 1)

	// The driver's own D-14 counter (player call + reviewer call) is the
	// one that actually enforces the cap; fakeModels.callCount() only
	// counts the player call, so read the cap counter directly.
	callCount := func() int {
		d.mu.Lock()
		defer d.mu.Unlock()
		return d.callCounts[userID]
	}
	if got := callCount(); got != 2 {
		t.Fatalf("expected exactly 2 reserved calls (decision + reviewer) for the cap's one permitted iteration, got %d", got)
	}

	sessions.fireOutput()
	waitForDisengages(t, sessions, 1)

	assertStableCallCount(t, models, 1, 100*time.Millisecond)
	if got := callCount(); got != 2 {
		t.Fatalf("expected the reserved call count to stay at the cap (2), got %d", got)
	}
	if got := len(sessions.sendCalls()); got != 1 {
		t.Fatalf("expected exactly 1 sent command total, got %d", got)
	}

	disengages := sessions.disengages()
	if len(disengages) != 1 || disengages[0] != "ai-call-cap" {
		t.Fatalf("expected exactly one ai-call-cap disengage, got %v", disengages)
	}
	if got := sessions.AutopilotStateFor(userID); got != session.AutopilotOff {
		t.Fatalf("expected autopilot off after the cap halt, got %v", got)
	}

	events := notifier.eventsSnapshot()
	last := events[len(events)-1]
	if last.Message != "Session call cap reached. Autopilot disengaged." {
		t.Fatalf("expected the locked cap-reached notice, got %q", last.Message)
	}
	if last.Outcome != "cap" {
		t.Fatalf("expected event outcome %q, got %q", "cap", last.Outcome)
	}
	if last.State != "off" {
		t.Fatalf("expected the cap-halt event to carry state %q, got %q", "off", last.State)
	}

	// D-14: the cap counter starts again at zero on every #AUTO ON.
	sessions.setState(session.AutopilotOff)
	sessions.engageState()
	d.EngageLoop(userID, connID, sessions.currentEpoch())
	waitForSendCount(t, sessions, 2)
	if got := callCount(); got != 2 {
		t.Fatalf("expected the cap to reset to zero on a fresh EngageLoop, got %d calls this stint", got)
	}
}

// TestLoop_ErrorThreshold proves D-15: two consecutive transient failures
// continue the loop with a running "(N of M)" count and send nothing; the
// third disengages with the kind's full locked sentence; a sent command
// resets the count; a blank threshold resolves to 3; and a non-transient
// (auth) failure disengages on the first hit rather than being counted.
func TestLoop_ErrorThreshold(t *testing.T) {
	t.Run("two_transient_failures_continue_then_the_third_disengages", func(t *testing.T) {
		sessions := &fakeSessions{window: "a room"}
		sessions.engageState()
		decisions := &fakeDecisionsStore{}
		notifier := &fakeNotifier{}
		commands := &fakeCommands{}
		models := &fakeModels{err: &gemini.Error{Kind: gemini.KindTransport, Message: "dial tcp: i/o timeout"}}
		d := New(sessions, &fakeProfiles{profile: testProfile()}, decisions, models, commands, notifier, testConfig())

		userID := uuid.New().String()
		connID := uuid.New().String()

		d.HandleEngage(userID, connID)
		d.HandleEngage(userID, connID)
		d.HandleEngage(userID, connID)

		events := notifier.eventsSnapshot()
		if len(events) != 3 {
			t.Fatalf("expected exactly 3 notifications, got %d", len(events))
		}
		wantFirst := "AI decision failed: the model could not be reached. (1 of 3)"
		wantSecond := "AI decision failed: the model could not be reached. (2 of 3)"
		wantThird := "AI decision failed: the model could not be reached. Autopilot disengaged."
		if events[0].Message != wantFirst || events[0].Outcome != "transient" {
			t.Fatalf("expected first notice %q with outcome %q, got %q / %q", wantFirst, "transient", events[0].Message, events[0].Outcome)
		}
		if events[1].Message != wantSecond || events[1].Outcome != "transient" {
			t.Fatalf("expected second notice %q with outcome %q, got %q / %q", wantSecond, "transient", events[1].Message, events[1].Outcome)
		}
		if events[2].Message != wantThird || events[2].Outcome != "failed" {
			t.Fatalf("expected third notice %q with outcome %q, got %q / %q", wantThird, "failed", events[2].Message, events[2].Outcome)
		}
		if got := len(sessions.disengages()); got != 1 {
			t.Fatalf("expected exactly one disengage call (at the threshold), got %d", got)
		}
	})

	t.Run("a_sent_command_resets_the_count", func(t *testing.T) {
		sessions := &fakeSessions{window: "a room"}
		sessions.engageState()
		decisions := &fakeDecisionsStore{}
		notifier := &fakeNotifier{}
		commands := &fakeCommands{}
		transientErr := &gemini.Error{Kind: gemini.KindTransport, Message: "dial tcp: i/o timeout"}
		models := &fakeModels{
			answers: []*gemini.Answer{
				nil,
				nil,
				{Reasoning: "recovered", Command: "look"},
				nil,
			},
			errs: []error{
				transientErr,
				transientErr,
				nil,
				transientErr,
			},
			reviewAnswers: []*gemini.ReviewAnswer{
				{Blocked: false, Reason: "clear"},
			},
		}
		d := New(sessions, &fakeProfiles{profile: testProfile()}, decisions, models, commands, notifier, testConfig())

		userID := uuid.New().String()
		connID := uuid.New().String()

		d.HandleEngage(userID, connID) // 1 of 3
		d.HandleEngage(userID, connID) // 2 of 3
		d.HandleEngage(userID, connID) // sent -- resets the count
		d.HandleEngage(userID, connID) // back to 1 of 3, not 3 of 3

		events := notifier.eventsSnapshot()
		if len(events) != 4 {
			t.Fatalf("expected exactly 4 notifications, got %d", len(events))
		}
		wantAfterReset := "AI decision failed: the model could not be reached. (1 of 3)"
		if events[3].Message != wantAfterReset {
			t.Fatalf("expected the count to restart at 1 after a sent command, got %q", events[3].Message)
		}
		if got := len(sessions.disengages()); got != 0 {
			t.Fatalf("expected zero disengage calls (the count never reached 3 after the reset), got %d", got)
		}
	})

	t.Run("blank_threshold_resolves_to_3", func(t *testing.T) {
		profile := testProfile()
		profile.AISettings.DisengageThreshold = nil
		sessions := &fakeSessions{window: "a room"}
		sessions.engageState()
		decisions := &fakeDecisionsStore{}
		notifier := &fakeNotifier{}
		commands := &fakeCommands{}
		models := &fakeModels{err: &gemini.Error{Kind: gemini.KindTransport, Message: "dial tcp: i/o timeout"}}
		d := New(sessions, &fakeProfiles{profile: profile}, decisions, models, commands, notifier, testConfig())

		userID := uuid.New().String()
		connID := uuid.New().String()

		d.HandleEngage(userID, connID)
		d.HandleEngage(userID, connID)
		if got := len(sessions.disengages()); got != 0 {
			t.Fatalf("expected no disengage before the third failure with a blank threshold, got %d", got)
		}
		d.HandleEngage(userID, connID)
		if got := len(sessions.disengages()); got != 1 {
			t.Fatalf("expected exactly one disengage at the third failure with a blank threshold, got %d", got)
		}
	})

	t.Run("auth_failure_disengages_on_the_first_hit", func(t *testing.T) {
		sessions := &fakeSessions{window: "a room"}
		sessions.engageState()
		decisions := &fakeDecisionsStore{}
		notifier := &fakeNotifier{}
		commands := &fakeCommands{}
		models := &fakeModels{err: &gemini.Error{Kind: gemini.KindAuth, Message: "invalid API key"}}
		d := New(sessions, &fakeProfiles{profile: testProfile()}, decisions, models, commands, notifier, testConfig())

		d.HandleEngage(uuid.New().String(), uuid.New().String())

		if got := len(sessions.disengages()); got != 1 {
			t.Fatalf("expected exactly one disengage call on the first auth failure, got %d", got)
		}
		events := notifier.eventsSnapshot()
		if len(events) != 1 {
			t.Fatalf("expected exactly one notification, got %d", len(events))
		}
		if events[0].Outcome != "failed" {
			t.Fatalf("expected event outcome %q, got %q", "failed", events[0].Outcome)
		}
	})
}

// TestLoop_ConsecutiveBlocks proves D-17: three consecutive blocks
// disengage with their own locked notice, distinct from D-15's failure
// notice; a sent command resets the block count; and blocks never move
// D-15's failure counter, so a hostile room cannot spend a whole session's
// worth of calls by forcing block after block.
func TestLoop_ConsecutiveBlocks(t *testing.T) {
	t.Run("three_consecutive_blocks_disengage_with_their_own_notice", func(t *testing.T) {
		profile := testProfile()
		profile.NeverIssueList = "give"
		sessions := &fakeSessions{window: "a room"}
		sessions.engageState()
		decisions := &fakeDecisionsStore{}
		notifier := &fakeNotifier{}
		commands := &fakeCommands{}
		models := &fakeModels{answer: &gemini.Answer{Reasoning: "handing it over", Command: "give sword to bob"}}
		d := New(sessions, &fakeProfiles{profile: profile}, decisions, models, commands, notifier, testConfig())

		userID := uuid.New().String()
		connID := uuid.New().String()

		d.HandleEngage(userID, connID)
		d.HandleEngage(userID, connID)
		d.HandleEngage(userID, connID)

		if got := len(sessions.sendCalls()); got != 0 {
			t.Fatalf("expected zero sends, got %d", got)
		}
		events := notifier.eventsSnapshot()
		// 3 blocked-decision events plus the separate blocked-repeatedly
		// notice fired on the third.
		if len(events) != 4 {
			t.Fatalf("expected 4 notifications (3 blocks + 1 blocked-repeatedly), got %d", len(events))
		}
		last := events[3]
		if last.Message != "AI decisions were blocked repeatedly. Autopilot disengaged." {
			t.Fatalf("expected the locked blocked-repeatedly notice, got %q", last.Message)
		}
		if last.Outcome != "blocked-repeatedly" {
			t.Fatalf("expected event outcome %q, got %q", "blocked-repeatedly", last.Outcome)
		}
		disengages := sessions.disengages()
		if len(disengages) != 1 || disengages[0] != "ai-blocked-repeatedly" {
			t.Fatalf("expected exactly one ai-blocked-repeatedly disengage, got %v", disengages)
		}
	})

	t.Run("a_sent_command_resets_the_block_count", func(t *testing.T) {
		profile := testProfile()
		profile.NeverIssueList = "give"
		sessions := &fakeSessions{window: "a room"}
		sessions.engageState()
		decisions := &fakeDecisionsStore{}
		notifier := &fakeNotifier{}
		commands := &fakeCommands{}
		models := &fakeModels{
			answers: []*gemini.Answer{
				{Reasoning: "handing it over", Command: "give sword to bob"},
				{Reasoning: "handing it over", Command: "give sword to bob"},
				{Reasoning: "looking around", Command: "look"},
				{Reasoning: "handing it over", Command: "give sword to bob"},
				{Reasoning: "handing it over", Command: "give sword to bob"},
			},
			reviewAnswers: []*gemini.ReviewAnswer{
				{Blocked: false, Reason: "clear"},
			},
		}
		d := New(sessions, &fakeProfiles{profile: profile}, decisions, models, commands, notifier, testConfig())

		userID := uuid.New().String()
		connID := uuid.New().String()

		d.HandleEngage(userID, connID) // block 1 of 3
		d.HandleEngage(userID, connID) // block 2 of 3
		d.HandleEngage(userID, connID) // sent -- resets the block count
		d.HandleEngage(userID, connID) // block 1 of 3 again
		d.HandleEngage(userID, connID) // block 2 of 3 again

		if got := len(sessions.disengages()); got != 0 {
			t.Fatalf("expected zero disengage calls (the block count never reached 3 after the reset), got %d", got)
		}
	})

	t.Run("blocks_never_move_the_failure_counter", func(t *testing.T) {
		profile := testProfile()
		profile.NeverIssueList = "give"
		sessions := &fakeSessions{window: "a room"}
		sessions.engageState()
		decisions := &fakeDecisionsStore{}
		notifier := &fakeNotifier{}
		commands := &fakeCommands{}
		models := &fakeModels{answer: &gemini.Answer{Reasoning: "handing it over", Command: "give sword to bob"}}
		d := New(sessions, &fakeProfiles{profile: profile}, decisions, models, commands, notifier, testConfig())

		userID := uuid.New().String()
		connID := uuid.New().String()

		d.HandleEngage(userID, connID)
		d.HandleEngage(userID, connID)

		d.mu.Lock()
		failureCount := d.failureCounts[userID]
		d.mu.Unlock()
		if failureCount != 0 {
			t.Fatalf("expected the failure counter to stay at 0 after blocks, got %d", failureCount)
		}
	})
}

// TestLoop_BlankSettings proves the blank-settings limit (REQ-safety-limits-hold):
// a blank cap lets the loop run on past any number of calls, a failure is
// still informative with blank settings, nothing panics, and a session that
// never engages autopilot makes no model call and sends nothing -- hand
// play is untouched by this plan's counters and halts.
func TestLoop_BlankSettings(t *testing.T) {
	t.Run("blank_cap_runs_past_many_calls", func(t *testing.T) {
		sessions := &fakeSessions{window: "a room"}
		sessions.engageState()
		decisions := &fakeDecisionsStore{}
		notifier := &fakeNotifier{}
		commands := &fakeCommands{}
		models := &fakeModels{
			answer:       &gemini.Answer{Reasoning: "looking around", Command: "look"},
			reviewAnswer: &gemini.ReviewAnswer{Blocked: false, Reason: "clear"},
		}
		d := newPacedDriver(sessions, decisions, notifier, commands, models, 2*time.Millisecond, 5*time.Second, time.Millisecond)

		userID := uuid.New().String()
		connID := uuid.New().String()

		d.EngageLoop(userID, connID, sessions.currentEpoch())
		waitForSendCount(t, sessions, 1)

		const manyIterations = 20
		for i := 0; i < manyIterations; i++ {
			sessions.fireOutput()
			waitForSendCount(t, sessions, i+2)
		}

		d.StopLoop(userID, sessions.currentEpoch())

		// Read the driver's own D-14 counter (every reserved call, player
		// and reviewer alike) rather than fakeModels.callCount(), which
		// only counts the player call -- the point of this assertion is
		// that a blank cap never halts no matter how many calls accumulate.
		d.mu.Lock()
		totalCalls := d.callCounts[userID]
		d.mu.Unlock()
		if want := (manyIterations + 1) * 2; totalCalls < want {
			t.Fatalf("expected the blank cap to let the loop run past %d reserved calls, got %d", want, totalCalls)
		}
		if got := len(sessions.disengages()); got != 0 {
			t.Fatalf("expected zero disengage calls with a blank cap, got %d", got)
		}
	})

	t.Run("a_failure_is_still_informative_with_blank_settings", func(t *testing.T) {
		sessions := &fakeSessions{window: "a room"}
		sessions.engageState()
		decisions := &fakeDecisionsStore{}
		notifier := &fakeNotifier{}
		commands := &fakeCommands{}
		models := &fakeModels{err: &gemini.Error{Kind: gemini.KindAuth, Message: "invalid API key"}}
		d := New(sessions, &fakeProfiles{profile: testProfile()}, decisions, models, commands, notifier, testConfig())

		// Nothing panics with every AI setting left blank (testProfile's
		// AISettings{} zero value): no model call, either, since the
		// failure below shows nothing about this test depends on a set
		// cap or threshold.
		d.HandleEngage(uuid.New().String(), uuid.New().String())

		events := notifier.eventsSnapshot()
		if len(events) != 1 {
			t.Fatalf("expected exactly one notification, got %d", len(events))
		}
		if events[0].Message == "" {
			t.Fatalf("expected a non-empty, informative failure notice")
		}
		if got := len(sessions.disengages()); got != 1 {
			t.Fatalf("expected exactly one disengage call, got %d", got)
		}
	})

	t.Run("hand_play_is_untouched_by_blank_settings", func(t *testing.T) {
		// A human-typed command never reaches the driver at all -- the
		// wheel-grab and ordinary echo paths are internal/session's job.
		// This proves the negative mechanically: a Driver that is never
		// engaged makes zero model calls and zero sends, so none of this
		// plan's counters or halts can fire for a session that never turns
		// autopilot on.
		sessions := &fakeSessions{window: "a room"}
		sessions.engageState()
		decisions := &fakeDecisionsStore{}
		notifier := &fakeNotifier{}
		commands := &fakeCommands{}
		models := &fakeModels{answer: &gemini.Answer{Reasoning: "n/a", Command: "look"}}
		_ = New(sessions, &fakeProfiles{profile: testProfile()}, decisions, models, commands, notifier, testConfig())

		if got := models.callCount(); got != 0 {
			t.Fatalf("expected zero model calls with autopilot never engaged, got %d", got)
		}
		if got := len(sessions.sendCalls()); got != 0 {
			t.Fatalf("expected zero sends with autopilot never engaged, got %d", got)
		}
	})
}
