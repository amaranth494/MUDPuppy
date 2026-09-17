package driver

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/amaranth494/MudPuppy/internal/gemini"
	"github.com/amaranth494/MudPuppy/internal/session"
	"github.com/amaranth494/MudPuppy/internal/store"
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
		ignoreCancel: true, // worst case: StopLoop does not interrupt the old call
		answers: []*gemini.Answer{
			{Reasoning: "stint one, first", Command: "look"},
			{Reasoning: "stint one, decided before the wheel-grab", Command: "stale-north", SessionMemory: []string{"written by a decision that outlived its stint"}},
			{Reasoning: "stint two, reassessed", Command: "inventory"},
		},
		reviewAnswers: []*gemini.ReviewAnswer{
			{Blocked: boolPtr(false), Reason: "clear"},
			{Blocked: boolPtr(false), Reason: "clear"},
			{Blocked: boolPtr(false), Reason: "clear"},
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
		ignoreCancel: true, // worst case: StopLoop does not interrupt the old call
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
		reviewAnswer: &gemini.ReviewAnswer{Blocked: boolPtr(false), Reason: "clear"},
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
		reviewAnswer: &gemini.ReviewAnswer{Blocked: boolPtr(false), Reason: "clear"},
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

// inFlightCount reports how many iterations are in flight for userID, in any
// stint.
func inFlightCount(d *Driver, userID string) int {
	d.mu.Lock()
	defer d.mu.Unlock()
	n := 0
	for key := range d.inFlight {
		if key.userID == userID {
			n++
		}
	}
	return n
}

// waitForNoInFlight polls until userID has no iteration in flight.
func waitForNoInFlight(t *testing.T, d *Driver, userID string, within time.Duration) {
	t.Helper()
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		if inFlightCount(d, userID) == 0 {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("an iteration was still in flight %v after StopLoop", within)
}

// TestStint_StopLoopInterruptsAnInFlightModelCall is code review WR-02 of
// Phase 4: StopLoop's comment said a loop "waiting on a model call that can
// take up to two minutes" was cancelled at once, and it was not -- the call
// ran on context.Background(). The held call here is NEVER released: only
// the stint context's cancellation can end it.
func TestStint_StopLoopInterruptsAnInFlightModelCall(t *testing.T) {
	for _, tc := range []struct {
		name         string
		blockFirst   bool
		wantCallsMin int
	}{
		{"a_loop_tick", false, 2},
		{"the_stints_first_decision", true, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sessions := &fakeSessions{window: "a room"}
			sessions.engageState()
			decisions := &fakeDecisionsStore{}
			notifier := &fakeNotifier{}
			commands := &fakeCommands{}
			models := &fakeModels{
				answer:       &gemini.Answer{Reasoning: "looking around", Command: "look"},
				reviewAnswer: &gemini.ReviewAnswer{Blocked: boolPtr(false), Reason: "clear"},
			}
			d := newPacedDriver(sessions, decisions, notifier, commands, models, 2*time.Millisecond, 5*time.Second, 2*time.Millisecond)

			userID := uuid.New().String()
			connID := uuid.New().String()
			epoch := sessions.currentEpoch()
			block := make(chan struct{}) // never closed
			defer close(block)

			if tc.blockFirst {
				models.setBlock(block)
				go d.EngageLoop(userID, connID, epoch)
			} else {
				d.EngageLoop(userID, connID, epoch)
				waitForSendCount(t, sessions, 1)
				models.setBlock(block)
				sessions.fireOutput()
			}
			waitForCalls(t, models, tc.wantCallsMin) // the call is now held

			sendsBefore := len(sessions.sendCalls())
			eventsBefore := len(notifier.eventsSnapshot())
			reviewsBefore := models.reviewCallCount()

			sessions.disengageState()
			d.StopLoop(userID, epoch)

			waitForNoInFlight(t, d, userID, time.Second)

			if got := len(sessions.sendCalls()); got != sendsBefore {
				t.Fatalf("expected nothing sent by the interrupted decision, got %d further send(s)", got-sendsBefore)
			}
			if got := models.reviewCallCount(); got != reviewsBefore {
				t.Fatalf("expected no reviewer call after the interruption, got %d", got-reviewsBefore)
			}
			if got := len(notifier.eventsSnapshot()); got != eventsBefore {
				t.Fatalf("expected no notice for an interrupted decision (a cancelled call is not a model failure), got %d", got-eventsBefore)
			}
			if _, failures, _ := stintCounters(d, userID); failures != 0 {
				t.Fatalf("expected a cancelled call not to count as a failure, got %d", failures)
			}
			for _, row := range decisions.rows() {
				if row.Outcome == "failed" {
					t.Fatalf("an interrupted decision must not be stored as a failure")
				}
			}
		})
	}
}

// TestStint_RetryDelayIsCancellable proves the 503 retry wait no longer
// holds a stopped stint for its whole length and that no retry is made, and
// no call reserved, for a stint that ended during it (code review WR-02).
func TestStint_RetryDelayIsCancellable(t *testing.T) {
	sessions := &fakeSessions{window: "a room"}
	sessions.engageState()
	decisions := &fakeDecisionsStore{}
	notifier := &fakeNotifier{}
	models := &fakeModels{
		err: &gemini.Error{Kind: gemini.KindTransport, Status: 503, Message: "unavailable"},
	}
	d := newPacedDriver(sessions, decisions, notifier, &fakeCommands{}, models, 2*time.Millisecond, 5*time.Second, 2*time.Millisecond)
	d.retryDelay = 30 * time.Second

	userID := uuid.New().String()
	connID := uuid.New().String()
	epoch := sessions.currentEpoch()

	go d.EngageLoop(userID, connID, epoch)
	waitForCalls(t, models, 1)

	// Wait for the retrying notice: the iteration is now in its retry delay.
	deadline := time.Now().Add(2 * time.Second)
	for len(notifier.eventsSnapshot()) == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}

	sessions.disengageState()
	d.StopLoop(userID, epoch)
	waitForNoInFlight(t, d, userID, time.Second)

	if got := models.callCount(); got != 1 {
		t.Fatalf("expected no retry for a stint that ended during the retry delay, got %d calls", got)
	}
	if calls, failures, _ := stintCounters(d, userID); calls != 1 || failures != 0 {
		t.Fatalf("expected one reserved call and no failure, got calls=%d failures=%d", calls, failures)
	}
	for _, ev := range notifier.eventsSnapshot() {
		if ev.Outcome == "transient" || ev.Outcome == "failed" {
			t.Fatalf("expected no failure notice, got %q", ev.Message)
		}
	}
}

// TestSendFailureIsNotAModelFailure is code review WR-03 of Phase 4: the AI's
// write is what discovers the dropped socket. The session manager has
// already parked the switch at WAITING; the driver must not call that a
// model failure, must not count it toward D-15's threshold, and above all
// must not disengage -- OFF instead of WAITING means the reconnect no longer
// resumes (D-19).
func TestSendFailureIsNotAModelFailure(t *testing.T) {
	for _, sendErr := range []error{session.ErrSendFailed, session.ErrNoConnection} {
		t.Run(sendErr.Error(), func(t *testing.T) {
			sessions := &fakeSessions{window: "a room", sendErr: sendErr}
			sessions.engageState()
			decisions := &fakeDecisionsStore{}
			notifier := &fakeNotifier{}
			models := &fakeModels{
				answer:       &gemini.Answer{Reasoning: "heading north", Command: "north"},
				reviewAnswer: &gemini.ReviewAnswer{Blocked: boolPtr(false), Reason: "clear"},
			}
			d := New(sessions, &fakeProfiles{profile: testProfile()}, decisions, models, &fakeCommands{}, notifier, testConfig())

			userID := uuid.New().String()
			connID := uuid.New().String()

			// Put the failure count at threshold minus one first: the next
			// counted failure would disengage.
			d.mu.Lock()
			d.failureCounts[userID] = 2
			d.mu.Unlock()

			d.HandleEngage(userID, connID)

			if got := sessions.disengages(); len(got) != 0 {
				t.Fatalf("expected a lost connection to disengage nothing, got %v", got)
			}
			if got := sessions.AutopilotStateFor(userID); got != session.AutopilotWaiting {
				t.Fatalf("expected the switch left where the manager put it (%q), got %q", session.AutopilotWaiting, got)
			}
			if _, failures, _ := stintCounters(d, userID); failures != 2 {
				t.Fatalf("expected a lost connection not to count toward the failure threshold, got %d (was 2)", failures)
			}

			events := notifier.eventsSnapshot()
			if len(events) != 1 {
				t.Fatalf("expected exactly one notification, got %d", len(events))
			}
			if strings.Contains(events[0].Message, "model") || strings.Contains(events[0].Message, "disengaged") {
				t.Fatalf("a lost connection was reported as a model failure or a disengage: %q", events[0].Message)
			}
			if events[0].Message != sendFailedNotice || events[0].State != string(session.AutopilotWaiting) {
				t.Fatalf("expected %q with state %q, got %q with state %q", sendFailedNotice, session.AutopilotWaiting, events[0].Message, events[0].State)
			}

			rows := decisions.rows()
			if len(rows) != 1 || rows[0].Outcome != "failed" || rows[0].FailureKind != failureSendFailed || rows[0].Command != "north" {
				t.Fatalf("expected one failed row of kind %q carrying the unsent command, got %+v", failureSendFailed, rows)
			}
			if len(sessions.sendCalls()) != 0 {
				t.Fatalf("expected nothing recorded as sent")
			}
		})
	}
}

// TestMissingQuestIsRepairedAtTheNextDecision is the driver half of code
// review WR-09 of Phase 4: a goal with no active Quest (the goal endpoint's
// Quest upsert failed after the goal was saved, or the goal predates the
// Quest store) used to stay that way for ever, with every Quest Memory
// update dropped. The next decision now creates the Quest through the same
// ensure-or-create upsert the goal endpoint uses, and the update is stored.
func TestMissingQuestIsRepairedAtTheNextDecision(t *testing.T) {
	newDriver := func(quests *fakeQuestStore, goal string) (*Driver, *fakeSessions) {
		sessions := &fakeSessions{window: "a room"}
		sessions.engageState()
		models := &fakeModels{
			answer:       &gemini.Answer{Reasoning: "heading north", Command: "north", QuestMemory: []string{"the tower is north of the square"}},
			reviewAnswer: &gemini.ReviewAnswer{Blocked: boolPtr(false), Reason: "clear"},
		}
		profile := testProfile()
		profile.SessionGoal = goal
		d := New(sessions, &fakeProfiles{profile: profile}, &fakeDecisionsStore{}, models, &fakeCommands{}, &fakeNotifier{}, testConfig())
		d.SetQuests(quests)
		return d, sessions
	}

	t.Run("a_goal_with_no_quest_gets_one_and_its_memory_is_stored", func(t *testing.T) {
		quests := &fakeQuestStore{active: false}
		d, _ := newDriver(quests, "reach the tower")

		d.HandleEngage(uuid.New().String(), uuid.New().String())

		if got := quests.ensureCallsSnapshot(); len(got) != 1 || got[0] != "reach the tower" {
			t.Fatalf("expected exactly one repair of %q, got %v", "reach the tower", got)
		}
		updates := quests.updateBulletsCallsSnapshot()
		if len(updates) != 1 || len(updates[0].bullets) != 1 || updates[0].bullets[0] != "the tower is north of the square" {
			t.Fatalf("expected the Quest Memory update to be stored against the repaired Quest, got %+v", updates)
		}

		// The next decision finds the Quest: no second repair.
		d.HandleEngage(uuid.New().String(), uuid.New().String())
		if got := quests.ensureCallsSnapshot(); len(got) != 1 {
			t.Fatalf("expected no repair once the Quest exists, got %v", got)
		}
	})

	t.Run("an_existing_quest_is_never_touched_by_the_repair", func(t *testing.T) {
		quests := &fakeQuestStore{active: true, quest: store.Quest{ID: uuid.New(), Bullets: []string{}}}
		d, _ := newDriver(quests, "reach the tower")
		d.HandleEngage(uuid.New().String(), uuid.New().String())
		if got := quests.ensureCallsSnapshot(); len(got) != 0 {
			t.Fatalf("expected no ensure call when the Quest exists (the upsert bumps updated_at), got %v", got)
		}
	})

	t.Run("a_blank_goal_creates_nothing", func(t *testing.T) {
		quests := &fakeQuestStore{active: false}
		d, _ := newDriver(quests, "   ")
		d.HandleEngage(uuid.New().String(), uuid.New().String())
		if got := quests.ensureCallsSnapshot(); len(got) != 0 {
			t.Fatalf("expected a blank goal to create no Quest (D-04), got %v", got)
		}
	})

	t.Run("a_failed_repair_never_fails_the_decision", func(t *testing.T) {
		quests := &fakeQuestStore{active: false, ensureErr: errors.New("connection refused")}
		d, sessions := newDriver(quests, "reach the tower")
		d.HandleEngage(uuid.New().String(), uuid.New().String())
		if got := len(sessions.sendCalls()); got != 1 {
			t.Fatalf("expected the command to be sent despite the failed repair, got %d sends", got)
		}
		if got := quests.updateBulletsCallsSnapshot(); len(got) != 0 {
			t.Fatalf("expected no bullet write with no Quest, got %+v", got)
		}
	})
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
		reviewAnswer: &gemini.ReviewAnswer{Blocked: boolPtr(false), Reason: "clear"},
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
		reviewAnswer: &gemini.ReviewAnswer{Blocked: boolPtr(false), Reason: "clear"},
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
