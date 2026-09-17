package driver

import (
	"fmt"
	"testing"
	"time"

	"github.com/amaranth494/MudPuppy/internal/gemini"
	"github.com/amaranth494/MudPuppy/internal/store"
	"github.com/google/uuid"
)

// newRateLimitTestDriver builds a Driver with every collaborator faked, for
// tests that only exercise allowAISend directly and never call HandleEngage.
func newRateLimitTestDriver() *Driver {
	return New(&fakeSessions{}, &fakeProfiles{}, &fakeDecisionsStore{}, &fakeModels{}, &fakeCommands{}, &fakeNotifier{}, testConfig())
}

// TestAllowAISend_FloodRefused proves D-25/DR-4-03's core guarantee: a
// burst of AI-issued sends past the configured limit is refused, a
// different user's bucket is untouched by that flood, and the limiter
// refills once its own refill interval has elapsed.
func TestAllowAISend_FloodRefused(t *testing.T) {
	d := newRateLimitTestDriver()
	resolved := store.ResolvedAISettings{RateLimitSet: true, RateLimitPerSecond: 2}

	userA := "user-a"
	userB := "user-b"

	if !d.allowAISend(userA, resolved) {
		t.Fatalf("expected the first send to be allowed")
	}
	if !d.allowAISend(userA, resolved) {
		t.Fatalf("expected the second send to be allowed")
	}
	if d.allowAISend(userA, resolved) {
		t.Fatalf("expected a third send within the same second to be refused")
	}

	// A different user has its own bucket: the flood above must not have
	// consumed any of userB's tokens.
	if !d.allowAISend(userB, resolved) {
		t.Fatalf("expected a different user's first send to be allowed")
	}

	// session.RateLimiter (internal/session/websocket.go) refills to a full
	// bucket once its configured refill interval has elapsed since its last
	// refill — allowAISend builds it with a one-second interval regardless
	// of the configured limit, so one second (plus a small margin for test
	// scheduling slack) is the smallest wait this assertion can drive off
	// the limiter's own behaviour, not an arbitrary sleep.
	time.Sleep(1100 * time.Millisecond)
	if !d.allowAISend(userA, resolved) {
		t.Fatalf("expected a send to be allowed again after the refill interval")
	}
}

// TestAllowAISend_BlankMeansServerDefault proves a blank (RateLimitSet
// false) resolved setting still enforces a real, non-zero limit —
// store.DefaultAIRateLimitPerSecond — rather than reading as unlimited
// (T-5-16).
func TestAllowAISend_BlankMeansServerDefault(t *testing.T) {
	d := newRateLimitTestDriver()
	resolved := store.ResolvedAISettings{RateLimitSet: false}

	userID := "user-blank"
	for i := 0; i < store.DefaultAIRateLimitPerSecond; i++ {
		if !d.allowAISend(userID, resolved) {
			t.Fatalf("send %d of %d should be allowed under the server default", i+1, store.DefaultAIRateLimitPerSecond)
		}
	}
	if d.allowAISend(userID, resolved) {
		t.Fatalf("expected a burst past the server default (%d) to be refused, not treated as unlimited", store.DefaultAIRateLimitPerSecond)
	}
}

// TestAllowAISend_LimitChangeRebuildsBucket proves editing the resolved
// limit mid-session rebuilds the bucket at the new size rather than
// leaving it capped at whatever limit it was first built for.
func TestAllowAISend_LimitChangeRebuildsBucket(t *testing.T) {
	d := newRateLimitTestDriver()
	userID := "user-rebuild"

	limitTwo := store.ResolvedAISettings{RateLimitSet: true, RateLimitPerSecond: 2}
	if !d.allowAISend(userID, limitTwo) {
		t.Fatalf("expected the first send at limit 2 to be allowed")
	}
	d.allowAISend(userID, limitTwo) // consumes the rest of the limit-2 bucket

	d.mu.Lock()
	before, ok := d.aiSendLimiters[userID]
	d.mu.Unlock()
	if !ok || before.limit != 2 {
		t.Fatalf("expected a limit-2 bucket to exist before the change, got %+v (ok=%v)", before, ok)
	}

	limitFive := store.ResolvedAISettings{RateLimitSet: true, RateLimitPerSecond: 5}
	allowed := 0
	for i := 0; i < 5; i++ {
		if d.allowAISend(userID, limitFive) {
			allowed++
		}
	}
	if allowed != 5 {
		t.Fatalf("expected all 5 sends to be allowed once the bucket rebuilt at limit 5, got %d", allowed)
	}

	d.mu.Lock()
	after, ok := d.aiSendLimiters[userID]
	d.mu.Unlock()
	if !ok || after.limit != 5 {
		t.Fatalf("expected the stored limiter to be rebuilt at limit 5, got %+v (ok=%v)", after, ok)
	}
}

// TestDriverRateLimitRefusalIsTransient drives two immediate decisions
// through the real decide/Dispatch path with a resolved limit of 1 per
// second, proving D-25's owner-visible guarantees end to end: exactly one
// command reaches the session double, the second decision is stored with
// failure kind rate-limited-ai, the switch is not disengaged on that single
// refusal, and the panel notice sent is the locked transient-count string.
func TestDriverRateLimitRefusalIsTransient(t *testing.T) {
	sessions := &fakeSessions{window: "a room"}
	sessions.engageState()
	decisions := &fakeDecisionsStore{}
	notifier := &fakeNotifier{}
	commands := &fakeCommands{}
	models := &fakeModels{
		answer:       &gemini.Answer{Reasoning: "heading out", Command: "north"},
		reviewAnswer: &gemini.ReviewAnswer{Blocked: boolPtr(false), Reason: "clear"},
	}
	profile := testProfile()
	limit := 1
	profile.AISettings.RateLimitPerSecond = &limit

	d := New(sessions, &fakeProfiles{profile: profile}, decisions, models, commands, notifier, testConfig())

	userID := uuid.New().String()
	connID := uuid.New().String()

	d.HandleEngage(userID, connID)
	d.HandleEngage(userID, connID)

	if got := len(sessions.sendCalls()); got != 1 {
		t.Fatalf("expected exactly one command to reach the session double, got %d", got)
	}

	rows := decisions.rows()
	if len(rows) != 2 {
		t.Fatalf("expected two stored decision rows, got %d", len(rows))
	}
	if rows[0].Outcome != "sent" {
		t.Fatalf("expected the first decision's outcome to be %q, got %q", "sent", rows[0].Outcome)
	}
	if rows[1].FailureKind != failureRateLimitedAI {
		t.Fatalf("expected the second decision's failure kind to be %q, got %q", failureRateLimitedAI, rows[1].FailureKind)
	}

	if got := len(sessions.disengages()); got != 0 {
		t.Fatalf("expected the switch not to be disengaged on a single rate-limited refusal, got %d disengage calls", got)
	}

	wantNotice := fmt.Sprintf("AI decision failed: %s (%d of %d)", kindSentences[failureRateLimitedAI], 1, store.DefaultDisengageThreshold)
	events := notifier.eventsSnapshot()
	found := false
	for _, ev := range events {
		if ev.Message == wantNotice {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a notification with the locked rate-limited notice %q, got %+v", wantNotice, events)
	}
	if rows[1].Notice != wantNotice {
		t.Fatalf("expected the stored decision row's notice to match, got %q", rows[1].Notice)
	}
}
