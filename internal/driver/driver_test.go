package driver

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/amaranth494/MudPuppy/internal/config"
	"github.com/amaranth494/MudPuppy/internal/gemini"
	"github.com/amaranth494/MudPuppy/internal/icm"
	"github.com/amaranth494/MudPuppy/internal/session"
	"github.com/amaranth494/MudPuppy/internal/store"
	"github.com/google/uuid"
)

// orderLog is a shared, thread-safe event ordering recorder used by
// dispatch_precedes_send to prove Dispatch happens before the send.
type orderLog struct {
	mu     sync.Mutex
	events []string
}

func (o *orderLog) record(e string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.events = append(o.events, e)
}

func (o *orderLog) snapshot() []string {
	o.mu.Lock()
	defer o.mu.Unlock()
	out := make([]string, len(o.events))
	copy(out, o.events)
	return out
}

// fakeModels is a canned or error-returning Models double. block, when
// non-nil, is closed by the test to release a GenerateContent call that is
// deliberately held open (repeat_engage_fires_nothing).
type fakeModels struct {
	mu                    sync.Mutex
	calls                 int
	lastSystemInstruction string
	lastWindow            string
	answer                *gemini.Answer
	err                   error
	block                 chan struct{}
}

func (f *fakeModels) GenerateContent(ctx context.Context, endpoint, model, apiKey, systemInstruction, userText string) (*gemini.Answer, error) {
	f.mu.Lock()
	f.calls++
	f.lastSystemInstruction = systemInstruction
	f.lastWindow = userText
	block := f.block
	answer, err := f.answer, f.err
	f.mu.Unlock()

	if block != nil {
		<-block
	}
	if err != nil {
		return nil, err
	}
	return answer, nil
}

func (f *fakeModels) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func (f *fakeModels) lastSystemInstructionText() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.lastSystemInstruction
}

func (f *fakeModels) lastWindowText() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.lastWindow
}

// fakeCommands is a Commands double that can be told to refuse every
// dispatch, recording each attempted command and, optionally, its position
// in a shared orderLog.
type fakeCommands struct {
	mu     sync.Mutex
	refuse bool
	calls  []string
	order  *orderLog
}

func (f *fakeCommands) Dispatch(ctx *icm.ExecutionContext, sessionID string, normalized *icm.NormalizedCommand) (*icm.CommandResult, *icm.ICMError) {
	f.mu.Lock()
	f.calls = append(f.calls, normalized.Command)
	f.mu.Unlock()
	if f.order != nil {
		f.order.record("dispatch")
	}
	if f.refuse {
		return nil, icm.NewICMError(icm.E4001PermissionDenied, "refused for test", nil)
	}
	return nil, nil
}

type sendCall struct {
	userID  string
	command string
	source  string
}

// fakeSessions is a Sessions double backed by an in-memory window string
// that a test can update mid-run (resume_fires_one_fresh_decision).
type fakeSessions struct {
	mu             sync.Mutex
	window         string
	sends          []sendCall
	disengageCalls []string
	gameSessionID  uuid.UUID
	hasGameSession bool
	order          *orderLog
}

func (f *fakeSessions) RecentOutputSnapshot(userID string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.window
}

func (f *fakeSessions) setWindow(w string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.window = w
}

func (f *fakeSessions) SendCommandAs(userID, command, source string) error {
	f.mu.Lock()
	f.sends = append(f.sends, sendCall{userID: userID, command: command, source: source})
	f.mu.Unlock()
	if f.order != nil {
		f.order.record("send")
	}
	return nil
}

func (f *fakeSessions) DisengageAutopilot(userID, cause string) (session.AutopilotState, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.disengageCalls = append(f.disengageCalls, cause)
	return session.AutopilotOff, true
}

func (f *fakeSessions) CurrentGameSessionID(userID string) (uuid.UUID, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.gameSessionID, f.hasGameSession
}

func (f *fakeSessions) sendCalls() []sendCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]sendCall, len(f.sends))
	copy(out, f.sends)
	return out
}

func (f *fakeSessions) disengages() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.disengageCalls))
	copy(out, f.disengageCalls)
	return out
}

// fakeProfiles is a Profiles double returning one canned profile.
type fakeProfiles struct {
	profile *store.Profile
	err     error
}

func (f *fakeProfiles) GetProfileByConnection(userID, connectionID uuid.UUID) (*store.Profile, error) {
	return f.profile, f.err
}

// fakeDecisionsStore is an in-memory Decisions double.
type fakeDecisionsStore struct {
	mu         sync.Mutex
	rowsStored []store.DecisionRecord
}

func (f *fakeDecisionsStore) InsertDecision(rec store.DecisionRecord) (uuid.UUID, time.Time, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rowsStored = append(f.rowsStored, rec)
	return uuid.New(), time.Now(), nil
}

func (f *fakeDecisionsStore) rows() []store.DecisionRecord {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]store.DecisionRecord, len(f.rowsStored))
	copy(out, f.rowsStored)
	return out
}

// fakeNotifier is an in-memory Notifier double.
type fakeNotifier struct {
	mu     sync.Mutex
	events []Event
}

func (f *fakeNotifier) NotifyDecision(userID string, ev Event) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, ev)
}

func (f *fakeNotifier) eventsSnapshot() []Event {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]Event, len(f.events))
	copy(out, f.events)
	return out
}

func testProfile() *store.Profile {
	return &store.Profile{
		ConductRules:     "RULE: never grief another player. RULE: no scripting external bots.",
		ApproachGuidance: "GUIDE: prioritize quest completion over open-ended exploration.",
		NeverIssueList:   "",
		AISettings:       store.AISettings{},
	}
}

// testConfig returns a fully configured registry using obviously fake
// values — never a real model name or API key (Claude's Discretion,
// consistent with plan 03-02's test fixtures).
func testConfig() *config.Config {
	return &config.Config{
		AIDefaultModelSlug: "GEMINI",
		AIModels: map[string]config.AIModelEntry{
			"GEMINI": {
				Slug:      "GEMINI",
				ModelName: "test-model-not-a-real-name",
				Endpoint:  "https://example.invalid",
				APIKey:    "test-key-not-a-real-credential",
				Provider:  "gemini",
			},
		},
	}
}

func waitForCalls(t *testing.T, m *fakeModels, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if m.callCount() >= want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d model call(s), got %d", want, m.callCount())
}

func TestHandleEngage(t *testing.T) {
	t.Run("one_command_per_engage", func(t *testing.T) {
		sessions := &fakeSessions{window: "a room, an exit north"}
		models := &fakeModels{answer: &gemini.Answer{Reasoning: "heading north", Command: "north"}}
		d := New(sessions, &fakeProfiles{profile: testProfile()}, &fakeDecisionsStore{}, models, &fakeCommands{}, &fakeNotifier{}, testConfig())

		d.HandleEngage(uuid.New().String(), uuid.New().String())

		calls := sessions.sendCalls()
		if len(calls) != 1 {
			t.Fatalf("expected exactly one SendCommandAs call, got %d", len(calls))
		}
		if calls[0].source != "ai" {
			t.Fatalf("expected source %q, got %q", "ai", calls[0].source)
		}
	})

	t.Run("dispatch_precedes_send", func(t *testing.T) {
		order := &orderLog{}
		commands := &fakeCommands{order: order}
		sessions := &fakeSessions{window: "a room, an exit north", order: order}
		models := &fakeModels{answer: &gemini.Answer{Reasoning: "heading north", Command: "north"}}
		d := New(sessions, &fakeProfiles{profile: testProfile()}, &fakeDecisionsStore{}, models, commands, &fakeNotifier{}, testConfig())

		d.HandleEngage(uuid.New().String(), uuid.New().String())

		events := order.snapshot()
		dispatchIdx, sendIdx := -1, -1
		for i, e := range events {
			if e == "dispatch" && dispatchIdx == -1 {
				dispatchIdx = i
			}
			if e == "send" && sendIdx == -1 {
				sendIdx = i
			}
		}
		if dispatchIdx == -1 || sendIdx == -1 || dispatchIdx > sendIdx {
			t.Fatalf("expected dispatch before send, got order %v", events)
		}
	})

	t.Run("repeat_engage_fires_nothing", func(t *testing.T) {
		models := &fakeModels{
			answer: &gemini.Answer{Reasoning: "heading north", Command: "north"},
			block:  make(chan struct{}),
		}
		sessions := &fakeSessions{window: "a room, an exit north"}
		d := New(sessions, &fakeProfiles{profile: testProfile()}, &fakeDecisionsStore{}, models, &fakeCommands{}, &fakeNotifier{}, testConfig())

		userID := uuid.New().String()
		connID := uuid.New().String()

		done := make(chan struct{})
		go func() {
			d.HandleEngage(userID, connID)
			close(done)
		}()

		waitForCalls(t, models, 1)

		// A second engage for the same user while the first is still
		// in flight must be a no-op (D-03, T-3-35).
		d.HandleEngage(userID, connID)

		close(models.block)
		<-done

		if got := models.callCount(); got != 1 {
			t.Fatalf("expected exactly 1 model call, got %d", got)
		}
		if got := len(sessions.sendCalls()); got != 1 {
			t.Fatalf("expected exactly 1 send, got %d", got)
		}
	})

	t.Run("resume_fires_one_fresh_decision", func(t *testing.T) {
		models := &fakeModels{answer: &gemini.Answer{Reasoning: "heading north", Command: "north"}}
		sessions := &fakeSessions{window: "first window text"}
		d := New(sessions, &fakeProfiles{profile: testProfile()}, &fakeDecisionsStore{}, models, &fakeCommands{}, &fakeNotifier{}, testConfig())

		userID := uuid.New().String()
		connID := uuid.New().String()

		d.HandleEngage(userID, connID)

		sessions.setWindow("second window text, much newer")
		d.HandleEngage(userID, connID)

		if got := models.callCount(); got != 2 {
			t.Fatalf("expected 2 model calls (engage + resume), got %d", got)
		}
		if !strings.Contains(models.lastWindowText(), "newer") {
			t.Fatalf("expected the resume's prompt to carry the newer window text, got %q", models.lastWindowText())
		}
	})

	t.Run("conduct_rules_and_guidance_are_verbatim", func(t *testing.T) {
		models := &fakeModels{answer: &gemini.Answer{Reasoning: "heading north", Command: "north"}}
		profile := testProfile()
		sessions := &fakeSessions{window: "a room"}
		d := New(sessions, &fakeProfiles{profile: profile}, &fakeDecisionsStore{}, models, &fakeCommands{}, &fakeNotifier{}, testConfig())

		d.HandleEngage(uuid.New().String(), uuid.New().String())

		si := models.lastSystemInstructionText()
		if !strings.Contains(si, profile.ConductRules) {
			t.Fatalf("system instruction missing conduct rules verbatim")
		}
		if !strings.Contains(si, profile.ApproachGuidance) {
			t.Fatalf("system instruction missing approach guidance verbatim")
		}
	})

	t.Run("icm_refusal_sends_nothing", func(t *testing.T) {
		commands := &fakeCommands{refuse: true}
		sessions := &fakeSessions{window: "a room"}
		decisions := &fakeDecisionsStore{}
		models := &fakeModels{answer: &gemini.Answer{Reasoning: "heading north", Command: "north"}}
		d := New(sessions, &fakeProfiles{profile: testProfile()}, decisions, models, commands, &fakeNotifier{}, testConfig())

		d.HandleEngage(uuid.New().String(), uuid.New().String())

		if got := len(sessions.sendCalls()); got != 0 {
			t.Fatalf("expected zero sends on ICM refusal, got %d", got)
		}
		rows := decisions.rows()
		if len(rows) != 1 {
			t.Fatalf("expected exactly one stored decision row, got %d", len(rows))
		}
		if rows[0].Outcome != "refused" {
			t.Fatalf("expected outcome %q, got %q", "refused", rows[0].Outcome)
		}
		if got := len(sessions.disengages()); got != 1 {
			t.Fatalf("expected exactly one disengage call, got %d", got)
		}
	})

	t.Run("wraps_window_as_untrusted_data", func(t *testing.T) {
		models := &fakeModels{answer: &gemini.Answer{Reasoning: "heading north", Command: "north"}}
		decisions := &fakeDecisionsStore{}
		sessions := &fakeSessions{window: "a room, an exit north"}
		d := New(sessions, &fakeProfiles{profile: testProfile()}, decisions, models, &fakeCommands{}, &fakeNotifier{}, testConfig())

		d.HandleEngage(uuid.New().String(), uuid.New().String())

		wt := models.lastWindowText()
		if !strings.HasPrefix(wt, "<GAME_TEXT>") {
			t.Fatalf("expected the model's window to start with <GAME_TEXT>, got %q", wt)
		}
		if !strings.HasSuffix(wt, "</GAME_TEXT>") {
			t.Fatalf("expected the model's window to end with </GAME_TEXT>, got %q", wt)
		}
		if !strings.Contains(wt, "a room, an exit north") {
			t.Fatalf("expected the model's window to contain the snapshot text unchanged, got %q", wt)
		}
		rows := decisions.rows()
		if len(rows) != 1 {
			t.Fatalf("expected exactly one stored decision row, got %d", len(rows))
		}
		if rows[0].WindowText != "a room, an exit north" {
			t.Fatalf("expected stored WindowText to be the unwrapped snapshot, got %q", rows[0].WindowText)
		}
	})
}

func TestBuildSystemInstruction(t *testing.T) {
	t.Run("untrusted_data_paragraph_present", func(t *testing.T) {
		profile := testProfile()
		si := buildSystemInstruction(profile)
		if !strings.Contains(si, "<GAME_TEXT>") || !strings.Contains(si, "</GAME_TEXT>") {
			t.Fatalf("expected system instruction to mention both GAME_TEXT markers, got %q", si)
		}
		if !strings.Contains(si, "untrusted") {
			t.Fatalf("expected system instruction to state the game text is untrusted, got %q", si)
		}
	})

	t.Run("conduct_rules_and_guidance_verbatim", func(t *testing.T) {
		profile := testProfile()
		si := buildSystemInstruction(profile)
		if !strings.Contains(si, profile.ConductRules) {
			t.Fatalf("system instruction missing conduct rules verbatim")
		}
		if !strings.Contains(si, profile.ApproachGuidance) {
			t.Fatalf("system instruction missing approach guidance verbatim")
		}
	})

	t.Run("non_blank_never_issue_list_appears_under_heading", func(t *testing.T) {
		profile := testProfile()
		profile.NeverIssueList = "give\nopen vault"
		si := buildSystemInstruction(profile)
		if !strings.Contains(si, "Never-issue") {
			t.Fatalf("expected a Never-issue heading, got %q", si)
		}
		if !strings.Contains(si, "give") || !strings.Contains(si, "open vault") {
			t.Fatalf("expected every Never-issue entry present, got %q", si)
		}
	})

	t.Run("blank_list_emits_no_heading", func(t *testing.T) {
		profile := testProfile()
		profile.NeverIssueList = ""
		si := buildSystemInstruction(profile)
		if strings.Contains(si, "Never-issue") {
			t.Fatalf("expected no Never-issue heading for a blank list, got %q", si)
		}
	})

	t.Run("whitespace_only_list_emits_no_heading", func(t *testing.T) {
		profile := testProfile()
		profile.NeverIssueList = "   \n\t  "
		si := buildSystemInstruction(profile)
		if strings.Contains(si, "Never-issue") {
			t.Fatalf("expected no Never-issue heading for a whitespace-only list, got %q", si)
		}
	})

	t.Run("answer_shape_instruction_is_last", func(t *testing.T) {
		profile := testProfile()
		profile.NeverIssueList = "give"
		si := buildSystemInstruction(profile)
		wantSuffix := "with no leading '#', '@', '$' or '%' character."
		if !strings.HasSuffix(si, wantSuffix) {
			t.Fatalf("expected system instruction to end with the answer-shape instruction, got %q", si)
		}
	})
}

func TestHandleEngageFailures(t *testing.T) {
	cases := []struct {
		name        string
		answer      *gemini.Answer
		modelErr    error
		wantFailure string
		wantOutcome string
	}{
		{
			name:        "hash_prefixed_command",
			answer:      &gemini.Answer{Reasoning: "using a directive", Command: "#SET foo bar"},
			wantFailure: failureNonGameLine,
			wantOutcome: "failed",
		},
		{
			name:        "two_line_command",
			answer:      &gemini.Answer{Reasoning: "two moves", Command: "north\nsouth"},
			wantFailure: failureMultiCommand,
			wantOutcome: "failed",
		},
		{
			name:        "empty_command",
			answer:      &gemini.Answer{Reasoning: "unsure", Command: "   "},
			wantFailure: failureNoCommand,
			wantOutcome: "failed",
		},
		{
			name:        "unparseable_answer",
			modelErr:    &gemini.Error{Kind: gemini.KindMalformed, Message: "could not decode the model's structured answer"},
			wantFailure: failureMalformed,
			wantOutcome: "failed",
		},
		{
			name:        "transport_error",
			modelErr:    &gemini.Error{Kind: gemini.KindTransport, Message: "dial tcp: i/o timeout"},
			wantFailure: failureAPIError,
			wantOutcome: "failed",
		},
		{
			name:        "rate_limit_error",
			modelErr:    &gemini.Error{Kind: gemini.KindRateLimited, Message: "quota exceeded"},
			wantFailure: failureRateLimited,
			wantOutcome: "failed",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sessions := &fakeSessions{window: "a room"}
			decisions := &fakeDecisionsStore{}
			notifier := &fakeNotifier{}
			models := &fakeModels{answer: tc.answer, err: tc.modelErr}
			d := New(sessions, &fakeProfiles{profile: testProfile()}, decisions, models, &fakeCommands{}, notifier, testConfig())

			d.HandleEngage(uuid.New().String(), uuid.New().String())

			if got := len(sessions.sendCalls()); got != 0 {
				t.Fatalf("expected zero sends, got %d", got)
			}
			rows := decisions.rows()
			if len(rows) != 1 {
				t.Fatalf("expected exactly one stored decision row, got %d", len(rows))
			}
			if rows[0].FailureKind != tc.wantFailure {
				t.Fatalf("expected failure kind %q, got %q", tc.wantFailure, rows[0].FailureKind)
			}
			if rows[0].Outcome != tc.wantOutcome {
				t.Fatalf("expected outcome %q, got %q", tc.wantOutcome, rows[0].Outcome)
			}
			wantNotice := failureNotices[tc.wantFailure]
			events := notifier.eventsSnapshot()
			if len(events) != 1 {
				t.Fatalf("expected exactly one notification, got %d", len(events))
			}
			if events[0].Message != wantNotice {
				t.Fatalf("expected notice %q, got %q", wantNotice, events[0].Message)
			}
			if got := len(sessions.disengages()); got != 1 {
				t.Fatalf("expected exactly one disengage call, got %d", got)
			}
		})
	}
}
