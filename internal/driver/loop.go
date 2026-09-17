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

	"github.com/amaranth494/MudPuppy/internal/session"
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
func (d *Driver) EngageLoop(userID, connectionID string) {
	// D-14/D-15/D-17: the call, consecutive-failure and consecutive-block
	// counters all start at zero on every #AUTO ON, including after a
	// wheel-grab, so this runs as this function's first statement, before
	// the first iteration runs.
	d.resetStintCounters(userID)

	d.runIteration(userID, connectionID, true)

	// D-06's minimum spacing is never bypassed just because a decision
	// happened to be the stint's first (synchronous) one rather than a
	// loop tick.
	d.mu.Lock()
	d.lastDecisionAt[userID] = time.Now()
	d.mu.Unlock()

	if d.sessions.AutopilotStateFor(userID) != session.AutopilotOn {
		return
	}

	d.startLoop(userID, connectionID)
}

// startLoop replaces (or creates) userID's pacing goroutine. Cancelling and
// replacing any loop already running for this user first means a resume
// racing an engage can never leave two loops alive for the same user.
// Guarded by the same d.mu that already guards inFlight and lastDecisionAt.
func (d *Driver) startLoop(userID, connectionID string) {
	d.mu.Lock()
	if cancel, ok := d.loops[userID]; ok {
		cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	d.loops[userID] = cancel
	d.mu.Unlock()

	go d.runLoop(ctx, userID, connectionID)
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
func (d *Driver) runLoop(ctx context.Context, userID, connectionID string) {
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
		if d.sessions.AutopilotStateFor(userID) != session.AutopilotOn {
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

		d.runIteration(userID, connectionID, false)

		d.mu.Lock()
		d.lastDecisionAt[userID] = time.Now()
		d.mu.Unlock()
	}
}
