// Package driver is the phase's central orchestration: with autopilot
// engaged, take the rolling window of recent game text, ask the configured
// model once with the profile's conduct rules and approach guidance
// carried verbatim, validate that the answer is a single plain game
// command, put it through the command safety gate in the automation
// execution context, and — only then — send it to the game. The decision,
// the reasoning and the outcome are stored. Any failure sends nothing,
// says why, and lands the switch off.
//
// This package depends on internal/session, internal/icm, internal/store,
// internal/gemini and internal/config; internal/session declares the hook
// type the two callers (task 03-08-03) use to invoke HandleEngage and never
// imports this package, keeping the dependency direction one-way.
package driver

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/amaranth494/MudPuppy/internal/config"
	"github.com/amaranth494/MudPuppy/internal/gemini"
	"github.com/amaranth494/MudPuppy/internal/icm"
	"github.com/amaranth494/MudPuppy/internal/session"
	"github.com/amaranth494/MudPuppy/internal/store"
	"github.com/google/uuid"
)

// Failure kinds (D-13). Each maps to exactly one locked notice below.
const (
	failureAPIError     = "api-error"
	failureRateLimited  = "rate-limited"
	failureNoCommand    = "no-command"
	failureMultiCommand = "multi-command"
	failureNonGameLine  = "non-game-line"
	failureMalformed    = "malformed"
	failureICMRefused   = "icm-refused"
)

// failureNotices maps every D-13 failure kind to its locked notice. The
// first six sentences are reproduced verbatim from 03-UI-SPEC.md's
// Copywriting Contract; the seventh (icm-refused) is a new sentence in the
// same voice, added because D-13 covers "a first failure of any kind" and
// an ICM refusal is one, while the Copywriting Contract names only six.
var failureNotices = map[string]string{
	failureAPIError:     "AI decision failed: the model could not be reached. Autopilot disengaged.",
	failureRateLimited:  "AI decision failed: the model's rate limit was reached. Autopilot disengaged.",
	failureNoCommand:    "AI decision failed: the model did not return a command. Autopilot disengaged.",
	failureMultiCommand: "AI decision failed: the model returned more than one command. Autopilot disengaged.",
	failureNonGameLine:  "AI decision failed: the model tried to issue a non-game command. Autopilot disengaged.",
	failureMalformed:    "AI decision failed: the model's answer could not be understood. Autopilot disengaged.",
	failureICMRefused:   "AI decision failed: the command was refused by the command safety limits. Autopilot disengaged.",
}

// maxCommandBytes bounds a validated command's length (T-3-01); anything
// longer is malformed rather than silently truncated or half-sent.
const maxCommandBytes = 512

// Models is the one collaborator method the driver calls to ask the
// configured model for a decision. Its signature matches
// gemini.Client.GenerateContent exactly, so the real *gemini.Client
// satisfies it with no adapter, and tests supply a fake with no network.
type Models interface {
	GenerateContent(ctx context.Context, endpoint, model, apiKey, systemInstruction, userText string) (*gemini.Answer, error)
}

// Commands is the one collaborator method the driver calls to put the AI's
// command through the command safety gate. Its signature matches
// icm.Dispatcher.Dispatch exactly. The gate's own safety checker must
// actually run for this command, which is why the driver calls this method
// directly rather than the engine's alternate, higher-level entry point
// that short-circuits a plain command before this method is ever reached.
type Commands interface {
	Dispatch(ctx *icm.ExecutionContext, sessionID string, normalized *icm.NormalizedCommand) (*icm.CommandResult, *icm.ICMError)
}

// Sessions is the slice of *session.Manager the driver depends on.
type Sessions interface {
	RecentOutputSnapshot(userID string) string
	SendCommandAs(userID, command, source string) error
	DisengageAutopilot(userID, cause string) (session.AutopilotState, bool)
	CurrentGameSessionID(userID string) (uuid.UUID, bool)
}

// Profiles is the slice of *store.ProfileStore the driver depends on.
type Profiles interface {
	GetProfileByConnection(userID, connectionID uuid.UUID) (*store.Profile, error)
}

// Decisions is the slice of *store.DecisionStore the driver depends on.
type Decisions interface {
	InsertDecision(rec store.DecisionRecord) (uuid.UUID, time.Time, error)
}

// Notifier delivers a decision or system Event to the browser (plan
// 03-09's websocket push). A nil Notifier passed to New is a silent no-op,
// so this plan is independently runnable without plan 03-09.
type Notifier interface {
	NotifyDecision(userID string, ev Event)
}

// Event is the payload plan 03-09 puts on the websocket and plan 03-10
// renders. Kind is "decision" or "system".
type Event struct {
	ID        string
	Kind      string
	Reasoning string
	Command   string
	Outcome   string
	Message   string
	Timestamp string
}

// Driver holds the collaborators the phase's single decision needs and an
// in-flight guard so an engage and a resume racing for the same user never
// produce two decisions (D-03, T-3-35).
type Driver struct {
	sessions  Sessions
	profiles  Profiles
	decisions Decisions
	models    Models
	commands  Commands
	notifier  Notifier
	cfg       *config.Config

	mu       sync.Mutex
	inFlight map[string]bool
}

// New builds a Driver. notifier may be nil (plan 03-09 supplies the real
// implementation); every notify call becomes a silent no-op until then.
func New(sessions Sessions, profiles Profiles, decisions Decisions, models Models, commands Commands, notifier Notifier, cfg *config.Config) *Driver {
	return &Driver{
		sessions:  sessions,
		profiles:  profiles,
		decisions: decisions,
		models:    models,
		commands:  commands,
		notifier:  notifier,
		cfg:       cfg,
		inFlight:  make(map[string]bool),
	}
}

// HandleEngage runs the phase's one decision for userID on connectionID:
// snapshot the recent-text window, ask the model once, validate the
// answer, put the validated command through the command safety gate in
// the automation execution context, and only then send it to the game.
// Any failure at any step sends nothing, stores the failure, notifies a
// system message and disengages autopilot (D-13). Callers (task 03-08-03)
// start this with `go` — it runs entirely on the caller's goroutine and
// takes its own in-flight guard so a concurrent call for the same userID
// is a no-op (D-03).
func (d *Driver) HandleEngage(userID, connectionID string) {
	d.mu.Lock()
	if d.inFlight[userID] {
		d.mu.Unlock()
		d.logDecision(userID, connectionID, "", "skipped", "", "", "", 0, 0)
		return
	}
	d.inFlight[userID] = true
	d.mu.Unlock()
	defer func() {
		d.mu.Lock()
		delete(d.inFlight, userID)
		d.mu.Unlock()
	}()

	window := d.sessions.RecentOutputSnapshot(userID)
	d.logDecision(userID, connectionID, "", "request", "", "", "", len(window), 0)

	var gameSessionID *uuid.UUID
	if gsID, ok := d.sessions.CurrentGameSessionID(userID); ok {
		gameSessionID = &gsID
	}

	userUUID, uErr := uuid.Parse(userID)
	connUUID, cErr := uuid.Parse(connectionID)
	if uErr != nil || cErr != nil {
		// Defensive only: both ids are already-parsed strings the caller
		// (task 03-08-03's two hook points) always supplies as valid ids.
		// No model call is made.
		d.recordFailure(userID, connectionID, uuid.Nil, uuid.Nil, gameSessionID, "", window, "", "", failureMalformed)
		return
	}

	profile, err := d.profiles.GetProfileByConnection(userUUID, connUUID)
	if err != nil || profile == nil {
		// A missing profile at this point means something changed under
		// the engage hook between the gate check and this call; treated
		// the same as an unreachable model since neither can proceed.
		d.recordFailure(userID, connectionID, userUUID, connUUID, gameSessionID, "", window, "", "", failureAPIError)
		return
	}

	defaultModelName := ""
	if d.cfg != nil {
		if entry, ok := d.cfg.AIModels[d.cfg.AIDefaultModelSlug]; ok {
			defaultModelName = entry.ModelName
		}
	}
	resolved := store.ResolveAISettings(profile.AISettings, defaultModelName)
	entry, ok := d.cfg.ResolveModelEntry(resolved.ModelName)
	if !ok {
		// Plan 03-06's engage-gate refusal should have prevented reaching
		// here at all; this is the defensive second line (D-20).
		d.recordFailure(userID, connectionID, userUUID, connUUID, gameSessionID, "", window, "", "", failureAPIError)
		return
	}

	systemInstruction := buildSystemInstruction(profile)

	answer, genErr := d.models.GenerateContent(context.Background(), entry.Endpoint, entry.ModelName, entry.APIKey, systemInstruction, window)
	if genErr != nil {
		kind := failureAPIError
		switch gemini.ErrorKind(genErr) {
		case gemini.KindRateLimited:
			kind = failureRateLimited
		case gemini.KindMalformed:
			kind = failureMalformed
		}
		d.recordFailure(userID, connectionID, userUUID, connUUID, gameSessionID, entry.ModelName, window, "", "", kind)
		return
	}

	d.logDecision(userID, connectionID, "", "answer", entry.ModelName, "", "", len(window), len(answer.Command))

	cmd, failKind := validateCommand(answer.Command)
	if failKind != "" {
		d.recordFailure(userID, connectionID, userUUID, connUUID, gameSessionID, entry.ModelName, window, answer.Reasoning, answer.Command, failKind)
		return
	}

	ctx := icm.ContextAutomation
	normalized := &icm.NormalizedCommand{
		Command:           cmd,
		Operator:          "",
		RequiresExecution: true,
	}
	if _, icmErr := d.commands.Dispatch(&ctx, userID, normalized); icmErr != nil {
		d.recordFailure(userID, connectionID, userUUID, connUUID, gameSessionID, entry.ModelName, window, answer.Reasoning, cmd, failureICMRefused)
		return
	}

	d.logDecision(userID, connectionID, "", "dispatch", entry.ModelName, "", "", len(window), len(cmd))

	if err := d.sessions.SendCommandAs(userID, cmd, "ai"); err != nil {
		d.recordFailure(userID, connectionID, userUUID, connUUID, gameSessionID, entry.ModelName, window, answer.Reasoning, cmd, failureAPIError)
		return
	}

	d.recordSuccess(userID, connectionID, userUUID, connUUID, gameSessionID, entry.ModelName, window, answer.Reasoning, cmd)
}

// recordFailure stores a failed or refused decision, notifies a system
// event carrying the matching locked notice, disengages autopilot, and
// logs the stage=failed line. A storage error is tolerated (the decision
// row is best-effort); the notify/disengage/log sequence still runs.
func (d *Driver) recordFailure(userID, connectionID string, userUUID, connUUID uuid.UUID, gameSessionID *uuid.UUID, modelName, window, reasoning, command, failureKind string) {
	notice := failureNotices[failureKind]
	outcome := "failed"
	if failureKind == failureICMRefused {
		outcome = "refused"
	}

	decisionID := ""
	if d.decisions != nil {
		id, _, err := d.decisions.InsertDecision(store.DecisionRecord{
			UserID:        userUUID,
			ConnectionID:  connUUID,
			GameSessionID: gameSessionID,
			ModelName:     modelName,
			WindowText:    window,
			Reasoning:     reasoning,
			Command:       command,
			Outcome:       outcome,
			FailureKind:   failureKind,
			Notice:        notice,
		})
		if err == nil {
			decisionID = id.String()
		}
	}

	d.notify(userID, Event{
		ID:        decisionID,
		Kind:      "system",
		Reasoning: reasoning,
		Outcome:   outcome,
		Message:   notice,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})

	d.sessions.DisengageAutopilot(userID, "ai-failure")

	d.logDecision(userID, connectionID, decisionID, "failed", modelName, outcome, failureKind, len(window), len(command))
}

// recordSuccess stores a sent decision, notifies a decision event, and
// logs the stage=sent line.
func (d *Driver) recordSuccess(userID, connectionID string, userUUID, connUUID uuid.UUID, gameSessionID *uuid.UUID, modelName, window, reasoning, command string) {
	decisionID := ""
	if d.decisions != nil {
		id, _, err := d.decisions.InsertDecision(store.DecisionRecord{
			UserID:        userUUID,
			ConnectionID:  connUUID,
			GameSessionID: gameSessionID,
			ModelName:     modelName,
			WindowText:    window,
			Reasoning:     reasoning,
			Command:       command,
			Outcome:       "sent",
		})
		if err == nil {
			decisionID = id.String()
		}
	}

	d.notify(userID, Event{
		ID:        decisionID,
		Kind:      "decision",
		Reasoning: reasoning,
		Command:   command,
		Outcome:   "sent",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})

	d.logDecision(userID, connectionID, decisionID, "sent", modelName, "sent", "", len(window), len(command))
}

// notify is a nil-safe wrapper around Notifier.NotifyDecision.
func (d *Driver) notify(userID string, ev Event) {
	if d.notifier == nil {
		return
	}
	d.notifier.NotifyDecision(userID, ev)
}

// logDecision emits one structured log line per stage of a decision. It
// carries ids, a stage name and byte lengths only. The model's answer, the
// command text, and any profile-specific text are never interpolated here
// (T-3-07, T-3-02) — they are proven by a stored row, a websocket event, or
// an end-user screenshot elsewhere, never by a log line.
func (d *Driver) logDecision(userID, connectionID, decisionID, stage, modelName, outcome, failureKind string, snapshotBytes, cmdLen int) {
	log.Printf("[AI-PLAYER] decision user_id=%s connection_id=%s decision_id=%s stage=%s model=%s outcome=%s failure=%s snapshot_bytes=%d cmd_len=%d",
		userID, connectionID, decisionID, stage, modelName, outcome, failureKind, snapshotBytes, cmdLen)
}

// buildSystemInstruction assembles tier two of the two-tier prompt (D-05):
// a short fixed preamble naming the assistant's job, then the profile's
// standing text carried character for character (Phase 1 D-12 — no
// summarising, truncation or rewriting), then the answer-shape
// instruction. Phase 6 adds learned notes to this tier; the prompt's shape
// does not change then, only what fills it.
func buildSystemInstruction(profile *store.Profile) string {
	var b strings.Builder
	b.WriteString("You are playing a text-based multiplayer game on behalf of its owner. ")
	b.WriteString("You are shown the game's most recent output and must decide the single next command to send.\n\n")
	b.WriteString("Conduct rules:\n")
	b.WriteString(profile.ConductRules)
	b.WriteString("\n\nApproach guidance:\n")
	b.WriteString(profile.ApproachGuidance)
	b.WriteString("\n\nRespond with a short plain-language reasoning of one to three sentences written for the owner, ")
	b.WriteString("and exactly one command, written exactly as it would be typed at the game's prompt, ")
	b.WriteString("with no leading '#', '@', '$' or '%' character.")
	return b.String()
}

// validateCommand checks the model's returned command before anything
// else touches it (T-3-01) — a security boundary, not a formatting
// nicety, since a hostile room description or another player's speech can
// try to talk the model into issuing a directive, and this check is the
// mechanical backstop that does not care whether it worked. It returns the
// trimmed command and an empty failure kind on success, or an empty
// command and one of the D-13 failure kinds on failure.
func validateCommand(raw string) (string, string) {
	trimmed := strings.TrimRight(raw, "\r")
	trimmed = strings.TrimSpace(trimmed)

	if trimmed == "" {
		return "", failureNoCommand
	}
	if strings.ContainsAny(trimmed, "\n\r") {
		return "", failureMultiCommand
	}
	switch trimmed[0] {
	case '#', '@', '$', '%':
		return "", failureNonGameLine
	}
	if len(trimmed) > maxCommandBytes {
		return "", failureMalformed
	}
	for _, r := range trimmed {
		if r < 0x20 && r != '\t' {
			return "", failureMalformed
		}
	}
	return trimmed, ""
}
