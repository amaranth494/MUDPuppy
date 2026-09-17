package driver

import (
	"strings"
	"testing"
	"time"

	"github.com/amaranth494/MudPuppy/internal/gemini"
	"github.com/google/uuid"
)

// This file holds the stint-identity tests added by the Phase 4 code review
// fixes (CR-01, WR-02, WR-10): a decision belongs to the stint it started
// in, and a decision that outlived its stint is dropped quietly even when
// the switch reads On again.

// stintCounters reads the driver's three per-stint counters for userID.
func stintCounters(d *Driver, userID string) (calls, failures, blocks int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.callCounts[userID], d.failureCounts[userID], d.blockCounts[userID]
}

// TestStint_QuickReengageDropsTheOldDecision is code review CR-01's own
// sequence: stint 1's loop tick is waiting on the model, the owner takes the
// wheel and types #AUTO ON again before the model answers. The old decision
// must never be sent -- the switch reading On again is not enough -- and the
// new stint's first decision must run at once, carrying D-08's reassess
// instruction, rather than skipping itself because the old call is still in
// flight.
func TestStint_QuickReengageDropsTheOldDecision(t *testing.T) {
	gameSessionID := uuid.New()
	sessions := &fakeSessions{window: "a room", gameSessionID: gameSessionID, hasGameSession: true}
	sessions.engageState()
	decisions := &fakeDecisionsStore{}
	notifier := &fakeNotifier{}
	commands := &fakeCommands{}
	memory := newFakeMemoryStore()
	models := &fakeModels{
		answers: []*gemini.Answer{
			{Reasoning: "stint one, first", Command: "look"},
			{Reasoning: "stint one, decided before the wheel-grab", Command: "stale-north", SessionMemory: []string{"written by a decision that outlived its stint"}},
			{Reasoning: "stint two, reassessed", Command: "inventory"},
		},
		reviewAnswers: []*gemini.ReviewAnswer{
			{Blocked: false, Reason: "clear"},
			{Blocked: false, Reason: "clear"},
			{Blocked: false, Reason: "clear"},
		},
	}
	d := newPacedDriver(sessions, decisions, notifier, commands, models, 2*time.Millisecond, 5*time.Second, 2*time.Millisecond)
	d.SetMemory(memory)

	userID := uuid.New().String()
	connID := uuid.New().String()

	stint1 := sessions.currentEpoch()
	d.EngageLoop(userID, connID, stint1)
	waitForSendCount(t, sessions, 1)

	// Stint 1's second decision blocks inside the model call.
	block := make(chan struct{})
	models.setBlock(block)
	sessions.fireOutput()
	waitForCalls(t, models, 2)
	models.setBlock(nil) // only that one call is held; later calls answer at once

	// Wheel-grab, then #AUTO ON again, before the model answers.
	sessions.disengageState()
	d.StopLoop(userID, sessions.currentEpoch())
	sessions.engageState()
	stint2 := sessions.currentEpoch()
	if stint2 == stint1 {
		t.Fatalf("expected the re-engage to begin a new stint, got epoch %d twice", stint1)
	}

	// The new stint's first decision runs while the old call is STILL in
	// flight: EngageLoop returns only after that decision has been sent.
	d.EngageLoop(userID, connID, stint2)
	sends := sessions.sendCalls()
	if len(sends) != 2 || sends[1].command != "inventory" {
		t.Fatalf("expected the new stint's first decision to be sent while the old call is still blocked, got %+v", sends)
	}
	si := models.systemInstructionsSnapshot()
	if len(si) != 3 {
		t.Fatalf("expected exactly 3 player calls so far, got %d", len(si))
	}
	if !strings.Contains(si[2], reassessInstruction()) {
		t.Fatalf("expected the new stint's first decision to carry the reassess instruction")
	}

	callsBefore, _, _ := stintCounters(d, userID)
	reviewsBefore := models.reviewCallCount()
	eventsBefore := len(notifier.eventsSnapshot())
	rowsBefore := len(decisions.rows())

	// Now the old model call answers, into a switch that reads On.
	close(block)
	time.Sleep(100 * time.Millisecond)
	d.StopLoop(userID, sessions.currentEpoch())

	for _, s := range sessions.sendCalls() {
		if s.command == "stale-north" {
			t.Fatalf("the previous stint's decision was sent into the new stint: %+v", sessions.sendCalls())
		}
	}
	for _, c := range commands.dispatchCalls() {
		if c == "stale-north" {
			t.Fatalf("the previous stint's decision was dispatched: %v", commands.dispatchCalls())
		}
	}
	if got := models.reviewCallCount(); got != reviewsBefore {
		t.Fatalf("expected no reviewer call for the dropped decision, got %d further call(s)", got-reviewsBefore)
	}
	callsAfter, failures, blocks := stintCounters(d, userID)
	if callsAfter != callsBefore {
		t.Fatalf("the dropped decision was charged to the new stint's call count: %d before, %d after", callsBefore, callsAfter)
	}
	if failures != 0 || blocks != 0 {
		t.Fatalf("expected no failure or block counted for a dropped decision, got failures=%d blocks=%d", failures, blocks)
	}
	if got := len(notifier.eventsSnapshot()); got != eventsBefore {
		t.Fatalf("expected no notice for a dropped decision, got %d further event(s)", got-eventsBefore)
	}
	if got := len(decisions.rows()); got != rowsBefore {
		t.Fatalf("expected no decision row for a dropped decision, got %d further row(s)", got-rowsBefore)
	}
	for _, call := range memory.updateCallsSnapshot() {
		for _, b := range call.bullets {
			if strings.Contains(b, "outlived its stint") {
				t.Fatalf("a dropped decision wrote Session Memory: %v", call.bullets)
			}
		}
	}
}

// TestStint_StaleIterationIsNotAFailure proves a decision that outlived its
// stint changes no counter and prints no notice when its model call FAILS:
// the owner has already taken over, and "AI decision failed ... (1 of 3)"
// would be both wrong and counted against the next stint.
func TestStint_StaleIterationIsNotAFailure(t *testing.T) {
	sessions := &fakeSessions{window: "a room"}
	sessions.engageState()
	decisions := &fakeDecisionsStore{}
	notifier := &fakeNotifier{}
	commands := &fakeCommands{}
	models := &fakeModels{
		answers: []*gemini.Answer{
			{Reasoning: "stint one, first", Command: "look"},
			nil,
			{Reasoning: "stint two, reassessed", Command: "inventory"},
		},
		errs: []error{
			nil,
			&gemini.Error{Kind: gemini.KindTransport, Message: "dial tcp: i/o timeout"},
			nil,
		},
		reviewAnswer: &gemini.ReviewAnswer{Blocked: false, Reason: "clear"},
	}
	d := newPacedDriver(sessions, decisions, notifier, commands, models, 2*time.Millisecond, 5*time.Second, 2*time.Millisecond)

	userID := uuid.New().String()
	connID := uuid.New().String()

	d.EngageLoop(userID, connID, sessions.currentEpoch())
	waitForSendCount(t, sessions, 1)

	block := make(chan struct{})
	models.setBlock(block)
	sessions.fireOutput()
	waitForCalls(t, models, 2)
	models.setBlock(nil)

	sessions.disengageState()
	d.StopLoop(userID, sessions.currentEpoch())
	sessions.engageState()
	d.EngageLoop(userID, connID, sessions.currentEpoch())
	waitForSendCount(t, sessions, 2)

	close(block) // the old call now fails, into a switch that reads On
	time.Sleep(100 * time.Millisecond)
	d.StopLoop(userID, sessions.currentEpoch())

	_, failures, blocks := stintCounters(d, userID)
	if failures != 0 || blocks != 0 {
		t.Fatalf("expected a stale failure to change no counter, got failures=%d blocks=%d", failures, blocks)
	}
	for _, ev := range notifier.eventsSnapshot() {
		if ev.Outcome == "transient" || ev.Outcome == "failed" {
			t.Fatalf("expected no failure notice for a stale decision, got %q (%s)", ev.Message, ev.Outcome)
		}
	}
	for _, row := range decisions.rows() {
		if row.Outcome == "failed" {
			t.Fatalf("a stale decision must not be stored as a failure")
		}
	}
	if got := len(sessions.disengages()); got != 0 {
		t.Fatalf("expected a stale failure to disengage nothing, got %v", sessions.disengages())
	}
}

// TestStint_DuplicateEngageStartsNothing proves an engage hook that arrives
// twice for one stint, or late for a stint that has been superseded, makes
// no model call and does not zero the running stint's counters.
func TestStint_DuplicateEngageStartsNothing(t *testing.T) {
	sessions := &fakeSessions{window: "a room"}
	sessions.engageState()
	decisions := &fakeDecisionsStore{}
	notifier := &fakeNotifier{}
	commands := &fakeCommands{}
	models := &fakeModels{
		answer:       &gemini.Answer{Reasoning: "looking around", Command: "look"},
		reviewAnswer: &gemini.ReviewAnswer{Blocked: false, Reason: "clear"},
	}
	d := newPacedDriver(sessions, decisions, notifier, commands, models, 2*time.Millisecond, 5*time.Second, 2*time.Millisecond)

	userID := uuid.New().String()
	connID := uuid.New().String()
	epoch := sessions.currentEpoch()

	d.EngageLoop(userID, connID, epoch)
	waitForSendCount(t, sessions, 1)

	d.EngageLoop(userID, connID, epoch)   // duplicate
	d.EngageLoop(userID, connID, epoch-1) // superseded
	d.StopLoop(userID, sessions.currentEpoch())

	if got := models.callCount(); got != 1 {
		t.Fatalf("expected a duplicate or superseded engage to make no model call, got %d calls", got)
	}
	if calls, _, _ := stintCounters(d, userID); calls != 2 {
		t.Fatalf("expected the running stint's call count (2) to be left alone, got %d", calls)
	}
}

// loopRegistered reports whether a pacing loop is registered for userID and,
// if so, for which stint.
func loopRegistered(d *Driver, userID string) (uint64, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	entry, ok := d.loops[userID]
	if !ok {
		return 0, false
	}
	return entry.epoch, true
}

// TestStint_LateStopLoopLeavesTheNextStintRunning is code review WR-10 of
// Phase 4. The manager fires the disengage hook with `go`, so the StopLoop
// for stint N can be scheduled after stint N+1's loop has started. It must
// not cancel that loop: the result used to be a badge reading On with no
// decisions and no notice until the owner toggled the switch.
func TestStint_LateStopLoopLeavesTheNextStintRunning(t *testing.T) {
	sessions := &fakeSessions{window: "a room"}
	sessions.engageState()
	decisions := &fakeDecisionsStore{}
	notifier := &fakeNotifier{}
	commands := &fakeCommands{}
	models := &fakeModels{
		answer:       &gemini.Answer{Reasoning: "looking around", Command: "look"},
		reviewAnswer: &gemini.ReviewAnswer{Blocked: false, Reason: "clear"},
	}
	d := newPacedDriver(sessions, decisions, notifier, commands, models, 2*time.Millisecond, 5*time.Second, 2*time.Millisecond)

	userID := uuid.New().String()
	connID := uuid.New().String()

	stint1 := sessions.currentEpoch()
	d.EngageLoop(userID, connID, stint1)
	waitForSendCount(t, sessions, 1)

	// Wheel-grab and immediate #AUTO ON; stint 1's StopLoop has not run yet.
	sessions.disengageState()
	sessions.engageState()
	stint2 := sessions.currentEpoch()
	d.EngageLoop(userID, connID, stint2)
	waitForSendCount(t, sessions, 2)

	// Now the late StopLoop for stint 1 lands.
	d.StopLoop(userID, stint1)

	if epoch, ok := loopRegistered(d, userID); !ok || epoch != stint2 {
		t.Fatalf("expected stint %d's loop to stay registered after a late StopLoop for stint %d, got (epoch %d, registered %v)", stint2, stint1, epoch, ok)
	}
	sessions.fireOutput()
	waitForSendCount(t, sessions, 3) // stint 2 is still deciding

	// The stint's own StopLoop does stop it.
	d.StopLoop(userID, stint2)
	if _, ok := loopRegistered(d, userID); ok {
		t.Fatalf("expected StopLoop for the current stint to remove its loop")
	}
	settled := models.callCount()
	sessions.fireOutput()
	assertStableCallCount(t, models, settled, 100*time.Millisecond)
}

// TestStint_LoopThatStopsItselfLeavesNoEntry proves the other half of WR-10:
// a loop that notices by itself that its stint is over removes its own
// registry entry instead of leaving a dead one behind.
func TestStint_LoopThatStopsItselfLeavesNoEntry(t *testing.T) {
	sessions := &fakeSessions{window: "a room"}
	sessions.engageState()
	models := &fakeModels{
		answer:       &gemini.Answer{Reasoning: "looking around", Command: "look"},
		reviewAnswer: &gemini.ReviewAnswer{Blocked: false, Reason: "clear"},
	}
	d := newPacedDriver(sessions, &fakeDecisionsStore{}, &fakeNotifier{}, &fakeCommands{}, models, 2*time.Millisecond, 5*time.Second, 2*time.Millisecond)

	userID := uuid.New().String()
	connID := uuid.New().String()
	d.EngageLoop(userID, connID, sessions.currentEpoch())
	waitForSendCount(t, sessions, 1)

	sessions.disengageState() // no StopLoop: the fake fires no hook
	sessions.fireOutput()     // wake the loop so it notices

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, ok := loopRegistered(d, userID); !ok {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("expected a loop that stopped by itself to remove its own registry entry")
}
