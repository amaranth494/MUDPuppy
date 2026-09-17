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
	"testing"

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
