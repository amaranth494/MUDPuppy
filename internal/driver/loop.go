// This file holds the driver's continuous-play loop lifecycle and pacing
// only (D-05, D-06): when a stint starts, when it stops, and how long it
// waits between decisions. The decision itself -- the window snapshot, the
// model calls, validation, the reviewer pass, dispatch and send -- stays in
// driver.go's runIteration, unchanged in shape by this file. The loop
// exists per engaged user: one goroutine, one cancel func, started by
// EngageLoop and stopped by StopLoop from any cause (the owner's #AUTO OFF,
// a wheel-grab, a disconnect, or the iteration's own disengage).
package driver

import (
	"context"
	"time"
)

// EngageLoop is the new EngageHook target for both real triggers of a stint
// (D-05): a fresh #AUTO ON engage and a WAITING-to-ON resume. It has the
// same signature as HandleEngage, so SetEngageHook(aiDriver.EngageLoop) is
// a drop-in retarget everywhere HandleEngage used to be wired directly.
// Phase 3's single decision becomes the stint's first iteration
// (first=true, carrying D-08's reassess instruction), run synchronously so
// its outcome is known before anything else starts; only if autopilot is
// still on afterward does the pacing loop begin. A first decision that
// already failed, hit a cap, or otherwise disengaged the switch has no
// loop to start.
//
// epoch is the stint this engagement began, as the session manager numbered
// it (code review CR-01 of Phase 4). Every decision of the stint carries it,
// and a decision whose epoch is no longer the switch's is dropped -- so an
// iteration of the PREVIOUS stint that is still waiting on a slow model call
// when the owner re-engages neither blocks this stint's reassess decision
// (the in-flight guard is keyed by epoch) nor gets its stale command sent
// into this stint (runIteration's stint checks and SendAICommand).
func (d *Driver) EngageLoop(userID, connectionID string, epoch uint64) {
	// D-14/D-15/D-17: the call, consecutive-failure and consecutive-block
	// counters all start at zero on every #AUTO ON, including after a
	// wheel-grab, so beginStint runs first, before the first iteration
	// runs. It refuses an epoch that has already begun or been superseded:
	// hooks are fired with `go`, so they can arrive late or out of order.
	if !d.beginStint(userID, epoch) {
		d.logDecision(userID, connectionID, "", "engage-ignored", "", "", "", 0, 0)
		return
	}

	// The first decision is the only one that carries D-08's reassess
	// instruction, so the stint must not start pacing until it has actually
	// run. A skip can now only mean another iteration of this SAME stint is
	// in flight (a direct HandleEngage racing this call); wait for it and
	// try again, giving up only if the stint ends while waiting.
	for !d.runIteration(userID, connectionID, true, epoch) {
		if d.staleStage(userID, epoch) != "" {
			return
		}
		time.Sleep(firstIterationRetryDelay)
	}

	// A first decision that failed, hit a cap, or outlived its stint has no
	// loop to start -- and, when the stint is over, must not stamp the
	// minimum-spacing clock the NEXT stint paces itself against.
	if d.staleStage(userID, epoch) != "" {
		return
	}

	// D-06's minimum spacing is never bypassed just because a decision
	// happened to be the stint's first (synchronous) one rather than a
	// loop tick.
	d.mu.Lock()
	d.lastDecisionAt[userID] = time.Now()
	d.mu.Unlock()

	d.startLoop(userID, connectionID, epoch)
}

// firstIterationRetryDelay is how long EngageLoop waits before retrying a
// first decision that was skipped because the same stint already had an
// iteration in flight.
const firstIterationRetryDelay = 25 * time.Millisecond

// beginStint records epoch as userID's newest begun stint and zeroes the
// stint counters, reporting false -- and changing nothing -- when that
// epoch, or a later one, has already begun.
func (d *Driver) beginStint(userID string, epoch uint64) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if begun, ok := d.begunEpoch[userID]; ok && epoch <= begun {
		return false
	}
	d.begunEpoch[userID] = epoch
	d.callCounts[userID] = 0
	d.failureCounts[userID] = 0
	d.blockCounts[userID] = 0
	return true
}

// startLoop replaces (or creates) userID's pacing goroutine. Cancelling and
// replacing any loop already running for this user first means a resume
// racing an engage can never leave two loops alive for the same user.
// Guarded by the same d.mu that already guards inFlight and lastDecisionAt.
func (d *Driver) startLoop(userID, connectionID string, epoch uint64) {
	d.mu.Lock()
	if cancel, ok := d.loops[userID]; ok {
		cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	d.loops[userID] = cancel
	d.mu.Unlock()

	go d.runLoop(ctx, userID, connectionID, epoch)
}

// StopLoop is the DisengageHook target (04-03-01, Pattern 2): it cancels
// and forgets userID's pacing goroutine, if one is running, so a loop
// asleep in its pacing wait -- or waiting on a model call that can take up
// to two minutes -- stops at once rather than only being noticed after its
// next decision completes. A miss (no loop running for userID) is a silent
// no-op, and it is safe to call from inside the very goroutine being
// cancelled: cancelling a context only marks it Done, it never blocks.
func (d *Driver) StopLoop(userID string) {
	d.mu.Lock()
	cancel, ok := d.loops[userID]
	if ok {
		delete(d.loops, userID)
	}
	d.mu.Unlock()

	if ok {
		cancel()
	}
}

// cancellableWait waits for dur or ctx's cancellation, whichever comes
// first, reporting whether ctx fired first (true) so the caller can return
// immediately instead of running one more iteration after being told to
// stop. A non-positive dur returns immediately without cancellation.
func cancellableWait(ctx context.Context, dur time.Duration) bool {
	if dur <= 0 {
		return false
	}
	select {
	case <-ctx.Done():
		return true
	case <-time.After(dur):
		return false
	}
}

// runLoop is the pacing goroutine started by startLoop for one engaged
// user. It wakes on whichever comes first: new game output (then a short
// settle wait so a whole room description lands before the next decision
// reads it, D-06), or the floor interval (a nudge for a quiet game). Either
// wakeup is cancellable via ctx, so a wheel-grab or a disconnect stops the
// loop immediately even mid-wait rather than after one more decision. After
// waking, it enforces the minimum spacing against the previous decision's
// finish time -- also cancellably -- before running the next iteration.
func (d *Driver) runLoop(ctx context.Context, userID, connectionID string, epoch uint64) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-d.sessions.OutputSignal(userID):
			if cancellableWait(ctx, d.settleDelay) {
				return
			}
		case <-time.After(d.floorInterval):
			// Quiet-game nudge: fall through to the tick below.
		}

		if ctx.Err() != nil {
			return
		}
		// The switch must still be On AND still in this loop's own stint: a
		// loop that outlived its stint stops itself here even if a later
		// stint has the switch On again.
		if d.staleStage(userID, epoch) != "" {
			return
		}

		d.mu.Lock()
		last, haveLast := d.lastDecisionAt[userID]
		d.mu.Unlock()
		if haveLast {
			if remaining := d.minSpacing - time.Since(last); remaining > 0 {
				if cancellableWait(ctx, remaining) {
					return
				}
			}
		}

		// One [AI-PLAYER] line per tick, ids/stage/lengths only -- never
		// the window, a command, a goal or any reasoning text (T-4-05).
		window := d.sessions.RecentOutputSnapshot(userID)
		d.logDecision(userID, connectionID, "", "loop-tick", "", "", "", len(window), 0)

		d.runIteration(userID, connectionID, false, epoch)

		// An iteration that outlived its stint must not stamp the clock the
		// next stint paces itself against.
		if d.staleStage(userID, epoch) != "" {
			return
		}

		d.mu.Lock()
		d.lastDecisionAt[userID] = time.Now()
		d.mu.Unlock()
	}
}
