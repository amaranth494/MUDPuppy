// This file is D-25/DR-4-03's fix: the security audit found that the
// last-resort speed limit on commands never counted the AI's own sends.
// The reason is in internal/icm/dispatcher.go: Dispatch returns for a
// command with no registered handler — which is every plain MUD command
// the AI sends — before RecordExecution is ever reached, so an AI send
// never contributes to that counter. Worse, that counter is never reset,
// so routing AI sends through it as it stands would eventually refuse
// every AI command in a long-running process.
//
// This file deliberately does NOT share state with icm.Dispatcher's
// circuit breaker for either reason above. It is a small, dedicated,
// per-user check the driver makes itself, immediately before every
// Dispatch call, reusing session.RateLimiter (the same token-bucket
// primitive the websocket ingress path already uses) rather than writing
// a second implementation. internal/icm/dispatcher.go is not touched by
// this fix; its behaviour for hand-typed commands is unchanged.
package driver

import (
	"time"

	"github.com/amaranth494/MudPuppy/internal/session"
	"github.com/amaranth494/MudPuppy/internal/store"
)

// aiSendLimiter pairs a per-user session.RateLimiter with the limit it was
// built for, so allowAISend can tell when the owner has edited the AI
// Command Rate Limit setting and rebuild the bucket at the new size,
// rather than silently keep enforcing a stale one.
type aiSendLimiter struct {
	limiter *session.RateLimiter
	limit   int
}

// allowAISend is the one check the driver makes, immediately before every
// d.commands.Dispatch call, that actually counts an AI-issued command
// against a speed limit (D-25). It resolves the effective limit from the
// profile's AI settings — resolved.RateLimitPerSecond when
// resolved.RateLimitSet is true, store.DefaultAIRateLimitPerSecond
// otherwise, never "unlimited" (T-5-16) — lazily builds or rebuilds a
// per-user token bucket sized to that limit, and returns whether this send
// may proceed. The limiter is keyed by user id alone, so it is shared
// across a user's stints; resetAISendLimiter below is what gives a new
// stint a full bucket.
func (d *Driver) allowAISend(userID string, resolved store.ResolvedAISettings) bool {
	limit := store.DefaultAIRateLimitPerSecond
	if resolved.RateLimitSet {
		limit = resolved.RateLimitPerSecond
	}

	d.mu.Lock()
	entry, ok := d.aiSendLimiters[userID]
	if !ok || entry.limit != limit {
		entry = &aiSendLimiter{
			limiter: session.NewRateLimiter(limit, time.Second),
			limit:   limit,
		}
		d.aiSendLimiters[userID] = entry
	}
	limiter := entry.limiter
	d.mu.Unlock()

	return limiter.Allow()
}

// resetAISendLimiter drops userID's rate limiter so the next allowAISend
// call rebuilds it from scratch with a full bucket. Called wherever a
// stint ends (StopLoop, internal/driver/loop.go), so a fresh #AUTO ON or a
// resume never inherits tokens already spent by the stint before it.
func (d *Driver) resetAISendLimiter(userID string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.aiSendLimiters, userID)
}
