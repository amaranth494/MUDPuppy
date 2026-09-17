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

	// The stint's context is created and registered BEFORE the first
	// decision (code review WR-02 of Phase 4), so StopLoop can interrupt
	// that decision's model calls too, not only the pacing loop that
	// follows it.
	ctx, entry, ok := d.registerStint(userID, epoch)
	if !ok {
		d.logDecision(userID, connectionID, "", "engage-ignored", "", "", "", 0, 0)
		return
	}

	// The first decision is the only one that carries D-08's reassess
	// instruction, so the stint must not start pacing until it has actually
	// run. A skip can now only mean another iteration of this SAME stint is
	// in flight (a direct HandleEngage racing this call); wait for it and
	// try again, giving up only if the stint ends while waiting.
	for !d.runIteration(ctx, userID, connectionID, true, epoch) {
		if d.staleStage(ctx, userID, epoch) != "" || cancellableWait(ctx, firstIterationRetryDelay) {
			d.releaseStint(userID, entry)
			return
		}
	}

	// A first decision that failed, hit a cap, or outlived its stint has no
	// loop to start -- and, when the stint is over, must not stamp the
	// minimum-spacing clock the NEXT stint paces itself against.
	if d.staleStage(ctx, userID, epoch) != "" {
		d.releaseStint(userID, entry)
		return
	}

	// D-06's minimum spacing is never bypassed just because a decision
	// happened to be the stint's first (synchronous) one rather than a
	// loop tick.
	d.mu.Lock()
	d.lastDecisionAt[userID] = time.Now()
	d.mu.Unlock()

	go func() {
		d.runLoop(ctx, userID, connectionID, epoch)
		// A loop that stops by itself (its stint ended, and it noticed
		// before any StopLoop arrived) removes its own registry entry.
		d.releaseStint(userID, entry)
	}()
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

// registerStint replaces (or creates) userID's registered stint and returns
// the context every model call, retry wait and pacing wait of that stint
// runs under. Cancelling and replacing any stint already registered for this
// user first means a resume racing an engage can never leave two loops alive
// for the same user. Guarded by the same d.mu that already guards inFlight
// and lastDecisionAt.
//
// The registry entry carries the stint's epoch (code review WR-10 of Phase
// 4). A LATER stint is never replaced by an earlier one -- an EngageLoop
// that was slow to get here must not cancel the stint that has since
// superseded it (ok is false) -- and StopLoop uses the same epoch to tell
// which stint it was asked to stop.
func (d *Driver) registerStint(userID string, epoch uint64) (context.Context, *stintLoop, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if existing, ok := d.loops[userID]; ok {
		if existing.epoch > epoch {
			return nil, nil, false
		}
		existing.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	entry := &stintLoop{epoch: epoch, cancel: cancel}
	d.loops[userID] = entry
	return ctx, entry, true
}

// releaseStint cancels entry's context and removes entry from the registry,
// but only if the registered entry is still this one.
func (d *Driver) releaseStint(userID string, entry *stintLoop) {
	d.mu.Lock()
	if d.loops[userID] == entry {
		delete(d.loops, userID)
	}
	d.mu.Unlock()
	entry.cancel()
}

// StopLoop is the DisengageHook target (04-03-01, Pattern 2): it cancels
// and forgets userID's registered stint, if there is one. What that stops,
// precisely (code review WR-02 of Phase 4 -- this comment used to promise
// more than the code did): the stint's context is the one every pacing wait,
// every model call (player and reviewer, and their 503 retries) and the
// retry delay run under, from the stint's first decision on. A loop asleep
// in a pacing wait returns at once. A decision waiting on the model has its
// HTTP request cancelled, returns within moments instead of up to two
// minutes later, and is then dropped quietly by runIteration's stint check:
// no reviewer call, no retry, no counter, no notice, no memory write. The
// one thing not interrupted is a decision-row insert or a socket write
// already under way, both of which are short and bounded. A miss (no stint
// registered for userID) is a silent no-op, and it is safe to call from
// inside the very goroutine being cancelled: cancelling a context only marks
// it Done, it never blocks.
//
// epoch is the stint that ended (code review WR-10 of Phase 4). The session
// manager fires this hook with `go`, so nothing orders it against the
// following #AUTO ON: a StopLoop for stint N can arrive after stint N+1's
// loop has started. It therefore cancels only a loop whose own epoch is N or
// older. A loop of a later stint is left running -- cancelling it used to
// leave the badge On with no decisions and no notice until the owner toggled
// the switch.
func (d *Driver) StopLoop(userID string, epoch uint64) {
	d.mu.Lock()
	entry, ok := d.loops[userID]
	if ok && entry.epoch > epoch {
		ok = false
	}
	if ok {
		delete(d.loops, userID)
	}
	d.mu.Unlock()

	if ok {
		entry.cancel()
	}
}

// stintLoop is one user's registered pacing loop: the stint it belongs to
// and the cancel func that stops it.
type stintLoop struct {
	epoch  uint64
	cancel context.CancelFunc
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
		if d.staleStage(ctx, userID, epoch) != "" {
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
		// The tick line no longer takes a window snapshot of its own just to
		// log its length (code review WR-06 of Phase 4): the manager now
		// remembers when the window was last snapshotted so the next
		// decision's window reaches back to it, and a log-only snapshot here
		// would reset that memory an instant before the real one. The
		// iteration's own stage=request line, next, carries snapshot_bytes.
		d.logDecision(userID, connectionID, "", "loop-tick", "", "", "", 0, 0)

		d.runIteration(ctx, userID, connectionID, false, epoch)

		// An iteration that outlived its stint must not stamp the clock the
		// next stint paces itself against.
		if d.staleStage(ctx, userID, epoch) != "" {
			return
		}

		d.mu.Lock()
		d.lastDecisionAt[userID] = time.Now()
		d.mu.Unlock()
	}
}
