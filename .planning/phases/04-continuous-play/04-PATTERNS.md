# Phase 4: Continuous Play - Pattern Map

**Mapped:** 2026-09-17
**Files analyzed:** 24 (Go: 14 new/modified, migration: 1 new + 1 modify, frontend: 7, harness: 1 new, config/crypto: 2)
**Analogs found:** 24 / 24 — every file this phase touches is a same-file self-extension of Phase 3/3.1 code or a direct sibling of an existing pattern already in the codebase. No net-new mechanism (goroutine loop, hook pair, memory schema) lacks a structural precedent to copy the shape of, even where the content is new. All line numbers below were re-verified by direct reads this session, current as of the `ai-player` branch HEAD (Phase 3.1 already merged: `NeverIssueList`, `matchNeverIssue`, `ReviewCommand`, the `blocked` outcome, and migration 012 all already exist in the code read below).

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `internal/driver/driver.go` (modify) | service (orchestration) | event-driven | itself — `HandleEngage`'s existing linear body becomes the loop's iteration; `Driver` struct gains counter maps sibling to `inFlight` | exact (self-extension) |
| `internal/driver/loop.go` (new) | service (goroutine lifecycle) | event-driven | `internal/session/manager.go`'s `startTimers`/ticker-goroutine shape (line 291-344) for the pacing `select`; `Driver.inFlight`'s mutex-guarded map (driver.go:147-149) for the per-user context/cancel map | role-match (new file, existing concurrency idiom) |
| `internal/driver/memory.go` (new) | service (prompt-context assembly) | transform | `driver.go`'s `buildSystemInstruction`/`buildReviewSystemInstruction`/`untrustedDataParagraph` (lines 496-596) — the exact "assemble fixed prose plus caller-supplied trusted/untrusted blocks" shape this file's `promptContext` struct extends | exact (self-extension of an existing function family) |
| `internal/driver/driver_test.go` (modify) | test | event-driven | itself — `fakeModels`' single-answer fields (lines 42-56), `fakeSessions.DisengageAutopilot` returning a fixed value (lines 198-203), `assertBlockedDecision`/`waitForCalls` helpers (lines 305-370) | exact (self-extension, needs a scripted-sequence upgrade) |
| `internal/driver/loop_test.go` (new) | test (safety-limit suite, D-20) | event-driven | `internal/driver/driver_test.go`'s `TestHandleEngageFailures` table shape (lines 861-943) — the direct template for cap/threshold/503/block-count tests, one table-driven test per D-14/D-15/D-16/D-17 behavior | role-match (new file, existing table-test idiom) |
| `internal/session/manager.go` (modify) | service (state machine host) | event-driven | itself — `EngageHook`/`SetEngageHook` (lines 98-110) is the exact type+setter shape `DisengageHook`/`SetDisengageHook` mirrors; `appendOutputWindow` (lines 489-502) is the exact call site the new `OutputSignal` fires from; line 689's `_ = curConnID` is the D-27 dead statement to delete | exact (self-extension) |
| `internal/session/window.go` (modify) | model (ring buffer) | transform | itself — `ringBuffer.append`/`snapshot` (lines 44-80), pure `[]byte`-in/`string`-out, no timestamps today; D-09 adds a parallel timestamp slice/floor rule to this exact struct | exact (self-extension) |
| `internal/session/manager_test.go` (modify) or `internal/session/autopilot_test.go` (modify) | test | event-driven | existing autopilot state-machine tests (`internal/session/autopilot.go`'s pure `Engage`/`Disengage`/`EnterWaiting`/`Resume` functions, lines 60-116, already tested) — `DisengageHook` firing needs a sibling test to whatever already asserts `EngageHook` firing on resume | role-match |
| `internal/store/profile.go` (modify) | model/service (`*Store`) | CRUD | itself — the `ConductRules`/`ApproachGuidance`/`NeverIssueList` five-touch-point pattern (`Profile` struct lines 21-24, `ProfileUpdate` lines 142-146, `GetProfile`/`GetProfileByConnection` SELECT+Scan lines 216-354, `UpdateProfile` declare/coalesce/UPDATE/write-back lines 376-503) — the goal field is a fourth sibling of this exact chain | exact (self-extension, fourth sibling field) |
| `internal/store/quests.go` (new) | model/service (`*QuestStore`) | CRUD | `internal/store/decisions.go` (full file, 146 lines) — same constructor shape (`NewDecisionStore`/`NewQuestStore`), same hand-written SQL with `database/sql`, same "no ORM" discipline; `internal/store/transcripts.go`'s `OpenGameSession`/`CloseGameSession` open/reactivate-by-lookup shape for "create or reactivate the matching open Quest" | role-match (new file, two direct sibling stores to copy from) |
| `internal/store/decisions.go` (no change expected) | model/service (`*Store`) | CRUD | itself — `validDecisionOutcomes` (lines 18-23) already includes `blocked`; a `call-cap`/`ai-blocked-repeatedly` disengage is recorded as `outcome="failed"` with a new `failure_kind` value (Pattern 3 below), which needs zero changes to this file's map or SQL | exact (confirmed no change needed) |
| `internal/store/transcripts.go` (modify) | model/service (`*Store`) | CRUD + batch | itself — `OpenGameSession`/`CloseGameSession` (lines 64-131) is where Session Memory's column lives per D-10's "stored with the game session"; add `UpdateSessionMemory(gameSessionID uuid.UUID, memory []string) error` and `PruneOlderThan(days int) (int64, error)` as new methods following `AppendGameLines`'s transaction shape (lines 81-121) | exact (self-extension) |
| `internal/config/config.go` (modify) | config | n/a | itself — `SessionSecret`'s exact "required, fail-fast" shape (lines 83-87: `if cfg.SessionSecret == "" { return nil, errors.New(...) }`) is the literal template for `ENCRYPTION_KEY_V1`'s D-25 gate, guarded by the existing `RAILWAY_ENVIRONMENT` signal already used at `internal/driver/corpus_live_test.go:810-812` | exact (self-extension, new guard condition) |
| `internal/crypto/crypto.go` (no change expected) | service (key store) | n/a | itself — `DefaultKeyStore` (lines 27-50) already silently falls back to a generated key when `ENCRYPTION_KEY_V1` is blank; D-25's gate belongs in `config.Load()` (the fail-fast site), not here — this file's fallback stays as the local-dev convenience path, confirmed unchanged | exact (confirmed no change needed here; the gate is one layer up) |
| `internal/profiles/handler.go` (modify) | controller | request-response | itself — `AISettingsResponse`/`GetAISettings`/`PutAISettings`/`validateAISettings` (lines 131-143, 584-653, 842-872) is the exact sub-resource GET/PUT/validate triplet the new goal endpoint (`GetGoal`/`PutGoal`) and memory-read endpoint copy field-for-field | exact (self-extension, new sibling sub-resource) |
| `internal/session/handler.go` (modify) | controller | request-response | itself — the `Autopilot` handler's `"on"` case (lines 395-415): `if h.engageHook != nil { go h.engageHook(userIDStr, req.ConnectionID.String()) }` is the exact call site whose hook target changes from `aiDriver.HandleEngage` to `aiDriver.EngageLoop` (a one-line change, matching `EngageHook`'s unchanged signature) | exact (self-extension) |
| `internal/session/websocket.go` (modify) | model + service (message types) | event-driven | itself — `AIDecisionPayload` (lines 60-68) and `WSMessage.Decision` (lines 51-54) are the exact struct the new switch-state/counts/memory fields join as siblings, following `Source`/`ConnectionID`'s existing "meaningful only on X message type" comment convention (lines 44-50) | exact (self-extension) |
| `migrations/013_add_goal_memory_and_quests.up.sql` / `.down.sql` (new) | migration | batch | `migrations/010_add_ai_fields.up.sql` (full file, `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` convention) for the `profiles.session_goal` column and `game_sessions.session_memory` column; `migrations/011_add_ai_session_tables.up.sql`'s `CREATE TABLE IF NOT EXISTS` + partial-unique-index convention for the new `quests` table | exact |
| `migrations/012_add_never_issue_and_blocked_outcome.down.sql` (modify, D-26) | migration | batch | itself — the existing `DROP CONSTRAINT`/re-`ADD CONSTRAINT` block (lines 4-20) needs one `UPDATE ai_decisions SET outcome='failed' WHERE outcome='blocked'` statement inserted before it, following the file's own `DO $$ ... END $$` style already present for the constraint lookup | exact (self-extension, one-statement gap fix) |
| `cmd/server/main.go` (modify) | config (wiring) | n/a | itself — the two `SetEngageHook(aiDriver.HandleEngage)` call sites (lines 270-271) become `aiDriver.EngageLoop`; the new `sessionManager.SetDisengageHook(aiDriver.StopLoop)` call mirrors `SetEngageHook`'s existing wiring line-for-line; the retention ticker mirrors `sessionManager.startTimers`'s existing ticker-in-goroutine shape (manager.go:302-344) | exact (self-extension) |
| `scripts/verify-phase4.sh` (new) | test (harness) | batch | `scripts/verify-phase3-1.sh` (full 3.1 shape: `set -uo pipefail`, `_get_field`, `tee`-to-report, `--self-test`/`--self-test-negative`/`--no-tests`, fixture-dir switch, PASS/FAIL/SKIP-per-criterion report) — the exact skeleton, pointed at Phase 4's goal/memory/retention endpoints | exact |
| `frontend/src/components/AIAssistPanel.tsx` (modify) | component | event-driven (websocket) + request-response (reload) | itself — `DecisionEntry`/`SystemEntry` types (lines 12-28), `loadHistory`'s reload mapping (lines 52-76), the live `handleAI` push (lines 82-115), the panel/tab markup (lines 124-180) — the goal box, status line, and Session Memory section are new siblings inside `.ai-assist-panel`, following the file's existing state-plus-effect idiom | exact (self-extension) |
| `frontend/src/components/AIPlayerPanel.tsx` (modify) | component | request-response (REST load/save) | itself — the `Never-Issue List` `.form-group`/`.form-label`/`.form-input.form-textarea`/`.form-hint` block (lines 229-243) is the exact pattern the "Delete Captured Text Now" danger-zone block sits beside; `ConnectionsHubModal.tsx`'s `.delete-confirm`/`.btn-danger` two-step inline confirm (lines 339-355) is the exact mechanism to reuse for the destructive action | exact (self-extension + cross-file reuse) |
| `frontend/src/pages/PlayScreen.tsx` (modify) | component (page) | event-driven | itself — `handleAI`'s existing three-branch `if/else if` (lines 415-424, already extended by 3.1 with the `blocked` branch) hard-codes `color: 'red'` for every `kind === 'system'` payload; this phase's outcome-keyed colour lookup replaces that one branch | exact (self-extension) |
| `frontend/src/types/index.ts` (modify) | model (types) | n/a | itself — `AIDecisionPayload.outcome` union (line ~74, already `'sent' | 'refused' | 'failed' | 'blocked'`) gains `'cap' | 'blocked-repeatedly' | 'transient' | 'retrying' | 'goal'`; `AISettingsResponse` (lines 298-303) is the exact sibling-field precedent, though the goal/memory fields live on new payload/response types, not this one | exact (self-extension) |
| `frontend/src/services/api.ts` (modify) | service (REST client) | request-response | itself — `getAISettings`/`putAISettings` (lines 649-677) is the exact GET/PUT-with-`credentials:'include'`/`handleAuthError` template for the new `getGoal`/`putGoal`/`getSessionMemory` functions; `onAI`/`offAI` (lines 264-268) is the existing websocket-handler-registration pattern, unchanged, since the goal/memory/counts ride the same `ai` message | exact (self-extension) |
| `frontend/src/index.css` (modify) | config (styles) | n/a | itself — the `.ai-assist-panel-header`/`.ai-assist-panel-body` block (lines 267-303) is the exact insertion point for the new `.ai-assist-panel-top` region; the `.ai-system-line.state-*` block (lines 346-355) is the exact block the five new `state-cap`/`state-blocked-repeatedly`/`state-transient`/`state-retrying`/`state-goal` rules join; `.form-group`/`.form-label`/`.form-input`/`.form-hint` (lines 1171-1210) are reused verbatim, no new rule | exact (self-extension) |

## Pattern Assignments

### `internal/driver/driver.go` (modify — service, event-driven)

**Analog:** itself. `HandleEngage` (lines 186-332, full function verified this session) is the loop's iteration body, unchanged in its display/storage contract per D-07. Three structural changes, each with an exact precedent in this same file:

**1. Counter maps sibling to the existing `inFlight` guard (verified verbatim, lines 138-149):**
```go
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
```
Add `callCounts`, `failureCounts`, `blockCounts map[string]int` — three maps of the identical shape, guarded by the same `d.mu`, reset to zero in the new `EngageLoop`'s first step (mirroring `resetStintCounters` per D-14/D-15/D-17's "starts at zero on every #AUTO ON").

**2. The two model-call sites needing a cap check + 503 retry (verified verbatim, lines 246 and 287):**
```go
answer, genErr := d.models.GenerateContent(context.Background(), entry.Endpoint, entry.ModelName, entry.APIKey, systemInstruction, wrapped)
// ...
review, revErr := d.models.ReviewCommand(context.Background(), entry.Endpoint, entry.ModelName, entry.APIKey, reviewSystemInstruction, reviewUserText)
```
Both need, in order: (a) `d.tryReserveCall(userID, resolved)` before the call — locks `d.mu` exactly as `inFlight`'s check-and-set does (lines 187-194), compares against `resolved.CallCapSet`/`resolved.CallCap`; (b) on a `*gemini.Error` with `Status == 503`, one retry after a short delay, itself gated by a second `tryReserveCall`; (c) any other error (or a second 503) falls through to the existing `recordFailure` call unchanged.

**3. `recordFailure`'s unconditional disengage becomes conditional on transient-vs-non-transient (Pitfall 1 from RESEARCH.md; verified verbatim, lines 338-376):**
```go
func (d *Driver) recordFailure(userID, connectionID string, userUUID, connUUID uuid.UUID, gameSessionID *uuid.UUID, modelName, window, reasoning, command, failureKind string) {
    notice := failureNotices[failureKind]
    outcome := "failed"
    if failureKind == failureICMRefused {
        outcome = "refused"
    }
    // ... InsertDecision, notify ...
    d.sessions.DisengageAutopilot(userID, "ai-failure")   // <-- becomes conditional (D-15)
    d.logDecision(userID, connectionID, decisionID, "failed", modelName, outcome, failureKind, len(window), len(command))
}
```
The seven D-13/3.1 kinds (`failureAPIError`, `failureRateLimited`, `failureNoCommand`, `failureMultiCommand`, `failureNonGameLine`, `failureMalformed`, `failureICMRefused`) all currently disengage unconditionally on the first hit; D-15 keeps that behavior only for non-transient kinds (missing profile, missing model entry, auth) and adds a `d.failureCounts[userID]` increment + "N of M" notice + no-disengage path for the six/seven transient kinds, following the exact map-plus-mutex shape as `callCounts` above. Note the current failure-kind mapping (lines 256-263) collapses `KindAuth`/`KindBadRequest` into the same `failureAPIError` bucket as a genuine transport error — per RESEARCH Pitfall 1, a new `failureAuth` kind (non-transient) needs to be split out here so D-15's "auth disengages on the first hit" rule has something to match against.

**`recordBlocked`'s sibling shape (D-17's second, separate counter) — verified verbatim, lines 391-426:**
```go
// recordBlocked ... deliberately no autopilot-disengage call here — a block is
// the defence working, not the AI failing, and the switch must stay
// exactly where the owner left it. Do not "fix" this by adding one.
```
D-17 does not change this — it adds a *second*, independent counter (`d.blockCounts[userID]`) incremented by the two call sites that call `recordBlocked` today (`driver.go:276`, never-issue; `driver.go:309`, reviewer), with its own threshold check and its own notify+disengage on reaching it (a `failureCapReached`-shaped constant, e.g. `failureBlockedRepeatedly`, recorded via the existing `recordFailure`-shaped path, not by modifying `recordBlocked` itself).

**`recordSuccess` resets both counters (verified verbatim, lines 429-458):** add `d.failureCounts[userID] = 0` and `d.blockCounts[userID] = 0` at the top of this function — D-15's "reset by a sent command" and D-17's "a sent command resets the block count" are the same reset site.

---

### `internal/driver/loop.go` (new — service, event-driven)

**Analog:** `internal/session/manager.go`'s `startTimers` (lines 300-344, the only ticker-in-goroutine precedent anywhere in this codebase) for the pacing `select` shape, and `Driver.inFlight`'s `map[string]bool` guarded by `d.mu` (driver.go:147-149, 187-199) for the per-user lifecycle map this file needs (`map[string]context.CancelFunc`).

**The engage-hook refactor (RESEARCH Pattern 1, exact anchors verified this session):**
```go
// Both call sites today (verified verbatim):
// internal/session/manager.go:801-802 (resumeAutopilotLocked, WAITING-to-ON resume)
if m.engageHook != nil {
    go m.engageHook(userID, rec.ConnectionID)
}
// internal/session/handler.go:408-410 (Autopilot HTTP handler, #AUTO ON)
if h.engageHook != nil {
    go h.engageHook(userIDStr, req.ConnectionID.String())
}
```
Neither call site's code changes — `EngageHook`'s type (`func(userID, connectionID string)`) is unchanged, so `cmd/server/main.go`'s two `SetEngageHook(aiDriver.HandleEngage)` lines become `SetEngageHook(aiDriver.EngageLoop)`, and `EngageLoop` calls `d.HandleEngage(userID, connectionID)` synchronously for the first iteration (D-08's reassess instruction applies only there) before starting `runLoop` as its own goroutine.

**The new `DisengageHook`, built as `EngageHook`'s exact structural mirror (verified verbatim, `internal/session/manager.go:98-110`):**
```go
type EngageHook func(userID, connectionID string)
func (m *Manager) SetEngageHook(h EngageHook) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.engageHook = h
}
```
`type DisengageHook func(userID string)` / `SetDisengageHook` copy this shape exactly; fire it (with `go`, matching `manager.go:802`'s existing discipline) from the end of `DisengageAutopilot` (line 736, after `changed` is confirmed true) and `parkAutopilotLocked` (line 759, same condition) — both call sites already exist and already gate on `changed` before their own log/transcript calls, so the new hook call is a one-line addition at the same point.

---

### `internal/driver/memory.go` (new — service, transform)

**Analog:** `driver.go`'s existing prompt-assembly family — `buildSystemInstruction`, `buildReviewSystemInstruction`, `untrustedDataParagraph` (lines 487-596, full functions verified verbatim). The exact "fixed preamble, then trusted profile text verbatim, then an optional labelled block, then instruction" shape these three functions already use is what a new `promptContext` struct (bundling `*store.Profile`, `Goal string`, `QuestBullets []string`, `SessionMemory []string`) threads through in place of the bare `*store.Profile` parameter (RESEARCH Pitfall 3).

**The exact conditional-block precedent to copy for the goal/Quest/Session Memory sections (verified verbatim, lines 505-508):**
```go
if strings.TrimSpace(profile.NeverIssueList) != "" {
    b.WriteString("\n\nNever-issue commands (the owner has forbidden these; ...):\n")
    b.WriteString(profile.NeverIssueList)
}
```
D-13's ordering (standing text, goal, Quest bullets, Session Memory, window) is a chain of exactly this shape, one block per layer, each optional/blank-safe the same way.

**The one untrusted-data paragraph that must grow to cover memory (verified verbatim, lines 515-530):**
```go
// untrustedDataParagraph is the fixed statement that everything between the
// <GAME_TEXT> markers is untrusted ... this is the one place the wording
// lives; it must never be reworded independently in either caller.
func untrustedDataParagraph() string {
    var b strings.Builder
    b.WriteString("The game text you are shown is delimited between <GAME_TEXT> and </GAME_TEXT> markers. ...")
    ...
}
```
D-13 requires Session/Quest Memory delimited the same way (`<SESSION_MEMORY>`/`<QUEST_MEMORY>` markers, parallel to `<GAME_TEXT>`) — extend this one function with one more sentence naming the new markers, and wrap the memory text the same way `wrapWindow` wraps the window (lines 483-485):
```go
func wrapWindow(window string) string {
    return "<GAME_TEXT>\n" + window + "\n</GAME_TEXT>"
}
```

**Gemini schema extension — the exact existing mechanism, already shipped and tested (verified verbatim, `internal/gemini/client.go:179-197`):**
```go
func (c *Client) GenerateContent(...) (*Answer, error) {
    schema := responseSchema{
        Type: "object",
        Properties: map[string]schemaProperty{
            "reasoning": {Type: "string"},
            "command":   {Type: "string"},
        },
        Required: []string{"reasoning", "command"},
    }
    ...
}
```
Adding `session_memory`/`quest_memory` fields means: (a) `schemaProperty` (lines 108-110, currently `{Type string}` only) needs an `Items *schemaProperty` field for the `{"type":"array","items":{"type":"string"}}` shape RESEARCH.md's Code Examples section documents; (b) `Answer` (the decoded struct `GenerateContent` unmarshals into) gains `SessionMemory []string` / `QuestMemory []string` fields; (c) `PropertyOrdering` (already proven at `client.go:287` for the reviewer's `reason`-before-`blocked` ordering) is reused unchanged if a specific write order is wanted, but is not required for a flat array-of-strings design (RESEARCH's own recommendation over the op-list alternative).

---

### `internal/driver/driver_test.go` (modify — test, event-driven)

**Analog:** itself. `fakeModels`' current single-answer fields (verified verbatim, lines 42-56) are the direct precedent for a scripted-sequence upgrade:
```go
type fakeModels struct {
    mu                    sync.Mutex
    calls                 int
    lastSystemInstruction string
    lastWindow            string
    answer                *gemini.Answer
    err                   error
    block                 chan struct{}

    reviewCalls                 int
    lastReviewSystemInstruction string
    lastReviewUserText          string
    reviewAnswer                *gemini.ReviewAnswer
    reviewErr                   error
}
```
Phase 4's multi-iteration tests need this to consume a queue (`answers []*gemini.Answer`, `errs []error`, popped in call order) rather than one fixed pair — the field names and the `GenerateContent`/`ReviewCommand` method bodies (lines 58-89) keep their locking discipline unchanged, only the "return the same thing every call" behavior changes to "return the next queued thing."

**`fakeSessions.DisengageAutopilot`'s fixed-return gap (verified verbatim, lines 198-203):**
```go
func (f *fakeSessions) DisengageAutopilot(userID, cause string) (session.AutopilotState, bool) {
    f.mu.Lock()
    defer f.mu.Unlock()
    f.disengageCalls = append(f.disengageCalls, cause)
    return session.AutopilotOff, true   // <-- always this, regardless of actual prior state
}
```
Phase 4's `AutopilotStateFor`-dependent loop-stop check (Pattern 1's `EngageLoop`) needs `fakeSessions` to track real state, driven by the actual pure functions in `internal/session/autopilot.go` (`Engage`/`Disengage`/`EnterWaiting`/`Resume`, lines 60-116, already tested elsewhere) rather than hard-coding a return value — add a `state session.AutopilotState` field and an `AutopilotStateFor(userID string) session.AutopilotState` method to satisfy a widened `Sessions` interface.

**`waitForCalls`/`assertBlockedDecision` (verified verbatim, lines 305-370) are the exact helpers `loop_test.go` calls unchanged** — no new assertion helper needed for call-count polling; a new `waitForDisengage`/`waitForBlockCount` sibling follows the identical `deadline := time.Now().Add(...)` polling shape (lines 305-315) rather than any sleep-based test.

---

### `internal/driver/loop_test.go` (new — test, event-driven, D-20)

**Analog:** `internal/driver/driver_test.go`'s `TestHandleEngageFailures` table (verified verbatim, lines 861-943, full test read this session) — the direct template for every D-20 case:
```go
func TestHandleEngageFailures(t *testing.T) {
    cases := []struct {
        name        string
        answer      *gemini.Answer
        modelErr    error
        wantFailure string
        wantOutcome string
    }{ /* six cases */ }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            sessions := &fakeSessions{window: "a room"}
            decisions := &fakeDecisionsStore{}
            notifier := &fakeNotifier{}
            models := &fakeModels{answer: tc.answer, err: tc.modelErr}
            d := New(sessions, &fakeProfiles{profile: testProfile()}, decisions, models, &fakeCommands{}, notifier, testConfig())
            d.HandleEngage(uuid.New().String(), uuid.New().String())
            // assertions on sessions.sendCalls(), decisions.rows(), notifier.eventsSnapshot(), sessions.disengages()
        })
    }
}
```
Each D-20 requirement (cap set/blank, threshold+notice, blank-threshold-resolves-to-3, 503 retry, consecutive-block disengage, no-commands-while-disconnected+resume, no-AI-reconnect, reassess-on-reengage, no-crash) becomes one `t.Run` in a new table or a small dedicated test function following this exact five-line setup (`fakeSessions`/`fakeDecisionsStore`/`fakeNotifier`/`fakeModels`/`New(...)`) plus the same assertion style already proven at lines 917-940.

---

### `internal/session/manager.go` (modify — service, event-driven)

**Analog:** itself, three separate touch points, each already read in full this session.

**1. The `OutputSignal` fire site (verified verbatim, lines 484-502):**
```go
func (m *Manager) appendOutputWindow(userID string, p []byte) {
    if len(p) == 0 {
        return
    }
    m.mu.Lock()
    defer m.mu.Unlock()

    ring, ok := m.outputWindow[userID]
    if !ok {
        ring = newRingBuffer()
        m.outputWindow[userID] = ring
    }
    ring.append(p)
}
```
D-06's "new output arrived" signal fires from inside this exact function, after `ring.append(p)`: a non-blocking send on a per-user buffered-size-1 `chan struct{}` (created lazily the same way `ring` is), following the "don't hand-roll a poll loop" guidance in RESEARCH.md's Don't Hand-Roll table.

**2. The `DisengageHook` fire sites (verified verbatim, lines 706-736 and 738-759):**
```go
func (m *Manager) DisengageAutopilot(userID, cause string) (AutopilotState, bool) {
    ...
    newState, changed := Disengage(cur)
    if !changed {
        m.logAutopilotTransition(userID, curConnID, cur, newState, "already-off")
        return newState, false
    }
    ...
    m.logAutopilotTransition(userID, curConnID, cur, newState, cause)
    m.enqueueTranscriptLineLocked(userID, "marker", fmt.Sprintf("[AI-ASSIST disengaged: %s]", cause))
    return newState, true   // <-- new: fire m.disengageHook(userID) here, only in this changed-true path
}
```
`parkAutopilotLocked` (lines 742-759) has the identical `if !changed { return }` early-out shape — the new hook fires at the same "changed is true" point in both functions, mirroring `resumeAutopilotLocked`'s existing `go m.engageHook(...)` call (line 802) exactly, including the `go` (the caller holds `m.mu` in both cases, so a synchronous call would deadlock the moment the driver reads anything back from this `Manager`).

**3. The D-27 dead statement (verified verbatim, `EngageAutopilot`, lines 665-704):**
```go
cur := AutopilotOff
curConnID := ""
if rec, ok := m.autopilot[userID]; ok {
    cur = rec.State
    curConnID = rec.ConnectionID
}
...
if connectionID == "" || session.ConnectionID != connectionID {
    m.logAutopilotTransition(userID, connectionID, cur, cur, "refused-wrong-connection")
    return cur, false, ErrWrongConnection
}
_ = curConnID   // <-- D-27: delete this line; curConnID is read below only in the
                //     no-change branch (line 693, "already-on"), which the code
                //     reaches by falling through past this statement unconditionally
                //     — dead in the changed-state path, per RESEARCH's own analysis
```

---

### `internal/session/window.go` (modify — model, transform)

**Analog:** itself, full file (80 lines) verified verbatim this session. `ringBuffer` (lines 30-42) is a pure byte ring with no time dimension:
```go
type ringBuffer struct {
    buf    []byte
    offset int
    filled bool
}
func (r *ringBuffer) append(p []byte) { ... }
func (r *ringBuffer) snapshot() string { ... }
```
D-09 adds a second axis (age) alongside the existing byte ceiling (`RecentOutputWindowBytes = 8192`, line 10) with a "never below one screenful" floor. The append/snapshot method *signatures* stay pure `[]byte`/`string` (no context.Context, no error) per this file's own established style — the simplest addition is a parallel `[]time.Time` (or a ring of chunk-boundary timestamps) tracked alongside `buf`, with `snapshot` gaining an age-aware variant that trims from the oldest end down to the ~10s/screenful floor rather than always returning the full 8KB. `stripANSI` (lines 18-28) is unchanged — this file's ANSI-stripping and telnet-adjacent comments about "cosmetic noise, not a correctness bug" set the tone for how precisely the age floor needs to be enforced (approximately, not to the millisecond).

---

### `internal/store/profile.go` (modify — model/service, CRUD)

**Analog:** itself, following the exact `ConductRules`/`ApproachGuidance`/`NeverIssueList` five-touch-point chain (all three fields already present, confirmed this session — Phase 3.1 already added `NeverIssueList` as the third sibling). The session goal is a fourth sibling of the identical shape.

**1. `Profile` struct (verified verbatim, lines 21-24):**
```go
ConductRules          string            `json:"conduct_rules"`
ApproachGuidance      string            `json:"approach_guidance"`
NeverIssueList        string            `json:"never_issue_list"`
AISettings            AISettings        `json:"ai_settings"`
// NEW: SessionGoal string `json:"session_goal"`
```

**2. `ProfileUpdate` struct (verified verbatim, lines 142-146):**
```go
ConductRules     *string            `json:"conduct_rules,omitempty"`
ApproachGuidance *string            `json:"approach_guidance,omitempty"`
NeverIssueList   *string            `json:"never_issue_list,omitempty"`
AISettings       *AISettings        `json:"ai_settings,omitempty"`
// NEW: SessionGoal *string `json:"session_goal,omitempty"`
```

**3. `GetProfile`/`GetProfileByConnection` — both SELECT+Scan (structurally identical in both functions, verified this session at lines 216-354):** add `session_goal` to both column lists (after `never_issue_list`) and `&profile.SessionGoal` to both `Scan` calls.

**4. `UpdateProfile` — declare/coalesce/UPDATE/write-back, verified verbatim at lines 376-503 (the same four-touch-point shape 03.1-PATTERNS.md already documented for `NeverIssueList`):** add a fifth sibling `var sessionGoal string`, the identical nil-coalesce `if updates.SessionGoal != nil { sessionGoal = *updates.SessionGoal } else { sessionGoal = existing.SessionGoal }`, `session_goal = $N` appended to the `SET` clause (renumbering the trailing `profileID`/`userID` placeholders), and the matching write-back block.

---

### `internal/store/quests.go` (new — model/service, CRUD)

**Analog:** `internal/store/decisions.go` (full file, 146 lines, verified verbatim) for the store-shape convention:
```go
type DecisionStore struct {
    db *sql.DB
}
func NewDecisionStore(db *sql.DB) *DecisionStore {
    return &DecisionStore{db: db}
}
```
`QuestStore`/`NewQuestStore` copy this constructor exactly. The "create or reactivate the matching row" operation has its closest precedent in `transcripts.go`'s `OpenGameSession` (lines 64-74, a plain `INSERT ... RETURNING id`) combined with the partial-unique-index technique RESEARCH.md's Don't Hand-Roll table recommends over a `SELECT`-then-`INSERT` race:
```sql
-- RESEARCH-recommended shape (not yet in the codebase, first use this phase):
CREATE UNIQUE INDEX idx_quests_active_goal
    ON quests(connection_id, goal_text_normalized) WHERE status = 'active';
```
A Go-side `INSERT ... ON CONFLICT (connection_id, goal_text_normalized) WHERE status='active' DO UPDATE SET status='active' RETURNING id` (Postgres upsert) is the atomic reactivate-or-create in one round trip — no existing file in this codebase uses `ON CONFLICT` yet, so this is the one genuinely new SQL idiom this phase introduces; `decisions.go`'s plain `QueryRow(...).Scan(&id, &createdAt)` result-handling shape (lines 91-102) still applies to the call site itself.

---

### `internal/store/transcripts.go` (modify — model/service, CRUD + batch)

**Analog:** itself. `OpenGameSession`/`CloseGameSession` (verified verbatim, lines 64-131) is the exact table (`game_sessions`) D-10 asks Session Memory to live on ("stored with the game session"). A new `UpdateSessionMemory` method follows the same single-statement `Exec` shape as `CloseGameSession`:
```go
func (s *TranscriptStore) CloseGameSession(gameSessionID uuid.UUID) error {
    _, err := s.db.Exec(
        `UPDATE game_sessions SET ended_at = NOW() WHERE id = $1 AND ended_at IS NULL`,
        gameSessionID,
    )
    return err
}
```
`UpdateSessionMemory(gameSessionID uuid.UUID, memory []string) error` is the identical one-`UPDATE`-statement shape, storing a JSON-marshaled `[]string` into a new `session_memory JSONB` column. A new `PruneOlderThan(days int) (int64, error)` for D-21's retention job follows `AppendGameLines`'s transaction discipline (lines 91-121, `tx.Begin()`/`defer tx.Rollback()`/`tx.Commit()`) even though pruning is a single `DELETE`/`UPDATE`, not a multi-row insert, because this file's established convention is "every mutating method that isn't a single trivial statement takes an explicit transaction."

---

### `internal/config/config.go` (modify — config, n/a)

**Analog:** itself. `SessionSecret`'s exact fail-fast shape (verified verbatim, lines 83-87):
```go
cfg.SessionSecret = os.Getenv("SESSION_SECRET")
if cfg.SessionSecret == "" {
    return nil, errors.New("SESSION_SECRET environment variable is required")
}
```
D-25's `ENCRYPTION_KEY_V1` gate is the same shape, guarded by the environment-detection signal already proven at `internal/driver/corpus_live_test.go:810-812`:
```go
if os.Getenv("RAILWAY_ENVIRONMENT") != "" {
    t.Fatal("live corpus test refuses to run when RAILWAY_ENVIRONMENT is set (D-11): ...")
}
```
Recommended shape in `Load()`, placed after the existing `cfg.EncryptionKeyV1 = os.Getenv("ENCRYPTION_KEY_V1")` line (line 185):
```go
cfg.EncryptionKeyV1 = os.Getenv("ENCRYPTION_KEY_V1")
if cfg.EncryptionKeyV1 == "" && os.Getenv("RAILWAY_ENVIRONMENT") != "" {
    return nil, errors.New("ENCRYPTION_KEY_V1 environment variable is required outside local development")
}
```
This is the one place in the whole codebase that already distinguishes "running on Railway" from "running on the owner's own machine" (Assumption A6 in RESEARCH.md) — no new detection mechanism is invented, the existing signal is reused verbatim.

---

### `internal/profiles/handler.go` (modify — controller, request-response)

**Analog:** itself. `AISettingsResponse`/`GetAISettings`/`PutAISettings`/`validateAISettings` (verified verbatim, lines 131-143, 584-653, 842-872) is the exact GET/PUT/validate triplet a new goal sub-resource copies field-for-field:
```go
type AISettingsResponse struct {
    ConductRules     string           `json:"conduct_rules"`
    ApproachGuidance string           `json:"approach_guidance"`
    NeverIssueList   string           `json:"never_issue_list"`
    AISettings       store.AISettings `json:"ai_settings"`
}

func (h *Handler) GetAISettings(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet { http.Error(w, "Method not allowed", http.StatusMethodNotAllowed); return }
    _, profile, err := h.getProfileByConnectionID(r)
    if err != nil { h.sendError(w, err.Error()); return }
    h.sendJSON(w, AISettingsResponse{ ConductRules: profile.ConductRules, ... })
}

func (h *Handler) PutAISettings(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPut { http.Error(w, "Method not allowed", http.StatusMethodNotAllowed); return }
    userUUID, profile, err := h.getProfileByConnectionID(r)
    if err != nil { h.sendError(w, err.Error()); return }
    var req AISettingsResponse
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil { h.sendError(w, "Invalid request body"); return }
    if verr := validateAISettings(req); verr != nil { h.sendError(w, verr.Error()); return }
    updates := &store.ProfileUpdate{ ConductRules: &req.ConductRules, ... }
    updatedProfile, err := h.profileStore.UpdateProfile(userUUID, profile.ID, updates)
    ...
    h.sendJSON(w, AISettingsResponse{ ... })
}
```
A new `GoalResponse{Goal string}` + `GetGoal`/`PutGoal` pair (or folding `session_goal` into the existing `AISettingsResponse`/sub-resource, executor discretion per CONTEXT.md) follows this exact structure, with `h.getProfileByConnectionID(r)` (the shared ownership-check helper, lines 762-791) reused unchanged. `validateAISettings`'s length-check pattern (verified verbatim, lines 842-872) is the template for a goal-length cap (RESEARCH's V5 recommendation, ~500-1000 chars):
```go
func validateAISettings(req AISettingsResponse) *ValidationError {
    if len(req.ConductRules) > 20000 {
        return &ValidationError{Message: "Conduct rules must be 20000 characters or less"}
    }
    ...
}
```
A memory-read endpoint (`GetSessionMemory`, read-only per D-10) follows `GetAISettings`'s GET-only shape with no PUT counterpart, sourced from `TranscriptStore`'s new `session_memory` column via the already-injected `h.transcripts` (this handler is already constructed via `NewHandlerWithTranscripts`, confirmed at `cmd/server/main.go`'s wiring section).

---

### `internal/session/handler.go` (modify — controller, request-response)

**Analog:** itself. The exact call site whose hook target changes (verified verbatim, lines 395-415):
```go
newState, changed, engageErr := h.manager.EngageAutopilot(userIDStr, req.ConnectionID.String())
resp.State = string(newState)
switch {
case engageErr == ErrNoConnectedSession:
    resp.Outcome = "refused-no-session"
case engageErr == ErrWrongConnection:
    resp.Outcome = "refused-wrong-connection"
case changed:
    resp.Outcome = "engaged"
    if h.engageHook != nil {
        go h.engageHook(userIDStr, req.ConnectionID.String())   // <-- becomes aiDriver.EngageLoop, wired in main.go
    }
default:
    resp.Outcome = "already-on"
}
```
No code change needed in this file itself beyond what `EngageHook`'s type already supports — the retarget is entirely in `cmd/server/main.go`'s `SetEngageHook` call. Listed here to confirm the call site was read and requires no structural change (same "no analog needed because no change needed" confirmation style as 03.1-PATTERNS.md's `internal/profiles/decisions.go` entry).

---

### `internal/session/websocket.go` (modify — model + service, event-driven)

**Analog:** itself. `AIDecisionPayload` and `WSMessage.Decision` (verified verbatim, lines 37-68):
```go
type WSMessage struct {
    Type   string `json:"type"`
    ...
    Decision *AIDecisionPayload `json:"decision,omitempty"`
}

type AIDecisionPayload struct {
    ID        string `json:"id"`
    Kind      string `json:"kind"`
    Reasoning string `json:"reasoning"`
    Command   string `json:"command"`
    Outcome   string `json:"outcome"`
    Message   string `json:"message"`
    Timestamp string `json:"timestamp"`
}
```
D-18's "every disengage-carrying `ai` message also carries switch state" and the status-line counts (calls/failures/blocks) and the Session Memory snapshot all join `AIDecisionPayload` as new optional fields, following the exact "meaningful only on X" comment convention this struct's sibling field `Source` on `WSMessage` already uses (lines 44-47: `// Source is meaningful only on inbound "data" messages`). No new message *type* is needed — `MsgTypeAI` (line 33) already carries this payload; D-18 is a field-addition, not a wire-shape change.

---

### `migrations/013_add_goal_memory_and_quests.up.sql` / `.down.sql` (new — migration, batch)

**Analog:** `migrations/010_add_ai_fields.up.sql` (full 9-line file, verified verbatim) for the two new columns:
```sql
-- +migrate Up
ALTER TABLE profiles
ADD COLUMN IF NOT EXISTS conduct_rules TEXT NOT NULL DEFAULT '',
ADD COLUMN IF NOT EXISTS approach_guidance TEXT NOT NULL DEFAULT '',
...
```
`profiles.session_goal TEXT NOT NULL DEFAULT ''` and `game_sessions.session_memory JSONB NOT NULL DEFAULT '[]'::jsonb` follow this exact `ADD COLUMN IF NOT EXISTS` convention. `migrations/011_add_ai_session_tables.up.sql` (full file, verified verbatim) is the `CREATE TABLE IF NOT EXISTS` + index convention for the new `quests` table:
```sql
CREATE TABLE IF NOT EXISTS game_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    connection_id UUID NOT NULL REFERENCES saved_connections(id) ON DELETE CASCADE,
    started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMP WITH TIME ZONE,
    line_count INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_game_sessions_connection_started ON game_sessions(connection_id, started_at DESC);
```
`quests` follows this shape (`id UUID PRIMARY KEY DEFAULT gen_random_uuid()`, `connection_id UUID NOT NULL REFERENCES saved_connections(id) ON DELETE CASCADE`, `goal_text TEXT NOT NULL`, `goal_text_normalized TEXT NOT NULL`, `status TEXT NOT NULL DEFAULT 'active'`, `bullets JSONB NOT NULL DEFAULT '[]'::jsonb`, `created_at`/`updated_at`), plus the new partial unique index (`CREATE UNIQUE INDEX IF NOT EXISTS idx_quests_active_goal ON quests(connection_id, goal_text_normalized) WHERE status = 'active';`) — a first use of a `WHERE`-qualified unique index in this codebase's migrations, needed for D-04's atomic reactivate-or-create.

**Down migration mirrors `migrations/011_add_ai_session_tables.down.sql`'s plain-drop shape:**
```sql
-- +migrate Down
DROP TABLE IF EXISTS ai_decisions;
DROP TABLE IF EXISTS game_session_lines;
DROP TABLE IF EXISTS game_sessions;
```
`013`'s down drops `quests`, then the two new columns (`session_memory`, `session_goal`), in that order (dependent-first, matching 011's own child-before-parent ordering).

---

### `migrations/012_add_never_issue_and_blocked_outcome.down.sql` (modify — migration, batch, D-26)

**Analog:** itself, full file (24 lines, verified verbatim this session):
```sql
-- +migrate Down
DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT conname
        FROM pg_constraint
        WHERE conrelid = 'ai_decisions'::regclass
          AND contype = 'c'
          AND pg_get_constraintdef(oid) LIKE '%outcome%'
    LOOP
        EXECUTE format('ALTER TABLE ai_decisions DROP CONSTRAINT %I', r.conname);
    END LOOP;
END $$;

ALTER TABLE ai_decisions
ADD CONSTRAINT ai_decisions_outcome_check CHECK (outcome IN ('sent','refused','failed'));

ALTER TABLE profiles
DROP COLUMN IF EXISTS never_issue_list;
```
**The D-26 gap, confirmed by this exact read:** re-adding the three-value CHECK on the line above will fail (or silently corrupt data if `NOT VALID` were used, which it is not here) if any row already has `outcome='blocked'` — no `UPDATE` statement exists in this file today to reconcile those rows first. The fix is one statement inserted immediately before the `DO $$` block:
```sql
-- +migrate Down
UPDATE ai_decisions SET outcome = 'failed' WHERE outcome = 'blocked';

DO $$
...
```
This is the smallest possible change — one line, using the same table/column this file already touches, with no new style introduced.

---

### `cmd/server/main.go` (modify — config, wiring)

**Analog:** itself. The two engage-hook wiring lines (verified verbatim, this session's earlier read of the wiring section):
```go
sessionManager.SetEngageHook(aiDriver.HandleEngage)
sessionHandler.SetEngageHook(aiDriver.HandleEngage)
```
Both become `aiDriver.EngageLoop` — a one-line change per site, since `EngageLoop` satisfies `EngageHook`'s unchanged signature. The new symmetric wiring:
```go
sessionManager.SetDisengageHook(aiDriver.StopLoop)
```
mirrors this exact line shape. The retention job's ticker mirrors `sessionManager.startTimers`'s existing per-user ticker-goroutine idiom (`internal/session/manager.go:300-344`, the only ticker precedent in this codebase) but runs once at server scope, not per-user — closer in shape to the one-time `go m.engageHook(...)` dispatch already used elsewhere in `main.go`'s own startup sequence (e.g. `go sessionManager.startTimers(...)` at line 291) than to any per-request pattern.

---

### `scripts/verify-phase4.sh` (new — test harness, batch)

**Analog:** `scripts/verify-phase3-1.sh` (full file header verified verbatim this session, lines 1-60+) — the exact structural skeleton:
```bash
#!/usr/bin/env bash
# scripts/verify-phase3-1.sh -- Phase 3.1 canned-report harness ...
set -uo pipefail
# ... _get_field, tee-to-report, --self-test/--self-test-negative/--no-tests,
# FIXTURE_DIR switch, PASS/FAIL/SKIP-per-criterion report, RUN_TS/GIT_SHA header ...
```
`scripts/verify-phase4.sh` copies this skeleton verbatim, pointed at Phase 4's own reachable endpoints (goal GET/PUT round trip — D-01/D-03; Session Memory GET read-back — D-10; the "delete captured text now" REST action — D-21) with new fixture directories (`scripts/fixtures/phase4/`, `scripts/fixtures/phase4-negative/`) the plan must create, following exactly the same three-invocation-mode contract (live / `--self-test` / `--self-test-negative`) plus `--no-tests`. The loop's own pacing/counter/reengage behaviors (D-14 to D-20) are **not** reachable by this HTTP-only harness — they are proven by `go test ./internal/driver/...`'s `loop_test.go` and printed as SKIP lines naming that test file, exactly as `verify-phase3-1.sh` already does for its own C1/C2/C4/C6 (live red-team, hostile-content Go tests) that a curl-based script cannot drive.

---

### `frontend/src/components/AIAssistPanel.tsx` (modify — component, event-driven + request-response)

**Analog:** itself, full file (180 lines) verified verbatim this session. `DecisionEntry`/`SystemEntry`, `loadHistory`, `handleAI`, and the panel markup are the four touch points:

**State additions follow the file's existing `useState` idiom (verified verbatim, lines 46-50):**
```tsx
const { wsManager, autopilotState } = useSession();
const [collapsed, setCollapsed] = useState(false);
const [entries, setEntries] = useState<PanelEntry[]>([]);
const [isLoading, setIsLoading] = useState(true);
const bodyRef = useRef<HTMLDivElement>(null);
```
New siblings: `goal`, `memoryOpen`, `sessionMemory`, `callCount`/`failureCount`/`blockCount`/`callCap`/`disengageThreshold` (or a single status object) — same flat `useState` style, no new state-management library.

**The panel's outer markup (verified verbatim, lines 133-146) is where `.ai-assist-panel-top` (04-UI-SPEC.md's new region) is inserted, between the header and the existing scrollable body:**
```tsx
return (
  <div className="ai-assist-panel">
    <div className="ai-assist-panel-header">...</div>
    {/* NEW: <div className="ai-assist-panel-top"> goal box, status line, Session Memory </div> */}
    <div className="ai-assist-panel-body" ref={bodyRef}>...</div>
  </div>
);
```
`loadHistory`'s reload mapping (lines 52-76) and `handleAI`'s live-push branch (lines 84-115) are unchanged in shape — the `outcome` union simply grows (per `types/index.ts` below) and the existing `.ai-system-line.state-${entry.outcome}` mechanism (line 172) already renders any new outcome string with the right CSS class with zero JSX change, exactly as it did for 3.1's `blocked` addition.

---

### `frontend/src/components/AIPlayerPanel.tsx` (modify — component, request-response)

**Analog:** itself, full file (293 lines) verified verbatim this session. The `Never-Issue List` `.form-group` block (lines 229-243) is the exact pattern the danger-zone sits beside, not merges into:
```tsx
<div className="form-group">
  <label className="form-label">Never-Issue List</label>
  <textarea className="form-input form-textarea" style={{ minHeight: '120px' }}
    value={neverIssueList} onChange={(e) => setNeverIssueList(e.target.value)} />
  <p className="form-hint">One entry per line. ...</p>
</div>
```
The `.settings-actions`/"Save AI Settings" button (lines 284-288) is the boundary the danger zone is deliberately kept apart from (04-UI-SPEC.md's own instruction), via a `border-top` div inserted after it, not inside `.settings-actions`.

**`ConnectionsHubModal.tsx`'s two-step inline delete confirm (verified verbatim, lines 339-355) is the exact mechanism to reuse:**
```tsx
{deleteConfirmId === conn.id ? (
  <div className="delete-confirm">
    <span>Delete?</span>
    <button className="btn btn-sm btn-danger" onClick={() => handleDelete(conn.id)} disabled={isLoading}>Yes</button>
    <button className="btn btn-sm" onClick={() => setDeleteConfirmId(null)}>No</button>
  </div>
) : (
  <button className="btn btn-sm btn-icon icon-red" onClick={() => setDeleteConfirmId(conn.id)} title="Delete">...</button>
)}
```
"Delete Captured Text Now" copies this `.delete-confirm`/`.btn-danger` pair verbatim (confirmed present in `frontend/src/index.css` at lines 1161 and 1624), substituting a text button for the icon button per 04-UI-SPEC.md's locked copy, and adding the unconditional scope-stating hint paragraph above it that a saved-connection delete does not need.

---

### `frontend/src/pages/PlayScreen.tsx` (modify — component, event-driven)

**Analog:** itself. The exact branch to replace (verified verbatim this session):
```tsx
const handleAI = (payload: AIDecisionPayload) => {
  if (!automationEngine) return;
  if (payload.kind === 'decision' && payload.outcome === 'sent') {
    automationEngine.echoLocal(`[AI-ASSIST > ${payload.command}]`, { color: 'brightmagenta' });
  } else if (payload.kind === 'decision' && payload.outcome === 'blocked') {
    automationEngine.echoLocal(`[AI-ASSIST blocked > ${payload.command}]`, { color: 'brightyellow' });
  } else if (payload.kind === 'system') {
    automationEngine.echoLocal(`[${payload.message}]`, { color: 'red' });   // <-- hard-coded, must become outcome-keyed
  }
};
```
04-UI-SPEC.md's locked replacement for the last branch:
```tsx
} else if (payload.kind === 'system') {
  const color =
    payload.outcome === 'transient' || payload.outcome === 'retrying'
      ? 'brightyellow'
      : payload.outcome === 'goal'
      ? 'white'
      : 'red'; // failed, refused, cap, blocked-repeatedly
  automationEngine.echoLocal(`[${payload.message}]`, { color });
}
```
`brightyellow` and `white` are already-used ANSI names elsewhere in this same file (`[Autopilot waiting for reconnect]`, `[Reconnected]`) — confirmed by the adjacent code this session (`previous === 'on' && autopilotState === 'waiting'` block, immediately above `handleAI`), so no new colour name is introduced.

---

### `frontend/src/types/index.ts` (modify — model, n/a)

**Analog:** itself. `AIDecisionPayload.outcome` (verified verbatim):
```tsx
export interface AIDecisionPayload {
  id: string;
  kind: 'decision' | 'system';
  reasoning?: string;
  command?: string;
  outcome?: 'sent' | 'refused' | 'failed' | 'blocked';   // NEW: | 'cap' | 'blocked-repeatedly' | 'transient' | 'retrying' | 'goal'
  message?: string;
  timestamp: string;
}
```
`AISettingsResponse` (verified verbatim, lines 298-303) is the sibling-field precedent style, though the goal/memory/counts fields likely live on new interfaces (`GoalResponse`, `SessionMemoryResponse`) rather than growing this one, per executor discretion — `StoredDecision` (lines already generic per 3.1-PATTERNS.md's confirmation) needs no change since decision-row outcomes are unaffected by Phase 4's new system-line-only outcomes.

---

### `frontend/src/services/api.ts` (modify — service, request-response)

**Analog:** itself. `getAISettings`/`putAISettings` (verified verbatim, lines 649-677) is the exact template:
```ts
export async function getAISettings(connectionId: string): Promise<AISettingsResponse> {
  const response = await fetch(`${API_BASE}/profiles/${connectionId}/ai-settings`, { credentials: 'include' });
  handleAuthError(response);
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || 'Failed to load AI settings');
  }
  return await response.json();
}
```
New `getGoal`/`putGoal`/`getSessionMemory`/`deleteCapturedText` functions copy this GET/PUT `fetch` + `credentials:'include'` + `handleAuthError` + `response.ok` shape verbatim, pointed at the new sub-resource routes `internal/profiles/handler.go` adds. `onAI`/`offAI` (verified verbatim, lines 264-268) need no change — the goal/memory/counts fields ride the same `ai` websocket message the existing handler registration already dispatches.

---

### `frontend/src/index.css` (modify — config, n/a)

**Analog:** itself. The exact insertion point for `.ai-assist-panel-top` (verified verbatim, lines 267-303):
```css
.ai-assist-panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--spacing-sm) var(--spacing-md);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}
/* NEW: .ai-assist-panel-top { flex-shrink: 0; padding: ...; border-bottom: ...; } */
.ai-assist-panel-body {
  flex: 1;
  overflow-y: auto;
  ...
}
```
The exact block the five new `.ai-system-line.state-*` rules join (verified verbatim, lines 346-355):
```css
.ai-system-line {
  font-size: var(--font-size-sm);
  padding: var(--spacing-xs) 0;
}
.ai-system-line.state-engaged,
.ai-system-line.state-resuming { color: var(--color-success); }
.ai-system-line.state-waiting { color: var(--color-warning); }
.ai-system-line.state-disengaged { color: #888; }
.ai-system-line.state-refused,
.ai-system-line.state-failed { color: var(--color-error); }
/* NEW, appended: state-cap/state-blocked-repeatedly (red), state-transient/state-retrying (yellow), state-goal (#888) */
```
`.form-group`/`.form-label`/`.form-input`/`.form-hint` (lines 1171-1210, confirmed present) are reused verbatim for the goal box — no new rule, matching the `.ai-assist-panel-top` markup's plan to use these exact classes for the input.

## Shared Patterns

### `EngageHook`/`DisengageHook` symmetric pair
**Source:** `internal/session/manager.go:98-110` (`EngageHook` type + `SetEngageHook`, full definition verified verbatim)
**Apply to:** `internal/driver/loop.go` (new `DisengageHook` type + `SetDisengageHook`), `internal/session/manager.go` (fire sites in `DisengageAutopilot` and `parkAutopilotLocked`), `cmd/server/main.go` (the new `SetDisengageHook(aiDriver.StopLoop)` wiring line). Same type shape, same nil-is-no-op discipline, same `go` dispatch under a held lock.

### Counter-map-plus-mutex, three siblings of `inFlight`
**Source:** `internal/driver/driver.go:147-149` (`Driver.mu`/`inFlight map[string]bool`), `187-199` (check-and-set-and-defer-delete pattern)
**Apply to:** `callCounts`, `failureCounts`, `blockCounts map[string]int` on the same `Driver` struct, guarded by the same `d.mu` — one mechanism, reused three times, not three different concurrency primitives (RESEARCH's own "Don't Hand-Roll" key insight, reused here for the counters too even though RESEARCH's own table did not list this one explicitly).

### `recordFailure`/`recordBlocked` sibling shape (Phase 3.1 precedent, extended)
**Source:** `internal/driver/driver.go:338-426` (`recordFailure` and `recordBlocked`, full methods verified verbatim; 03.1-PATTERNS.md already documented this pair)
**Apply to:** the cap-reached and blocked-repeatedly notices (D-14, D-17) — both are new `recordFailure`-shaped calls with a new `failureKind` constant (`failureCapReached`, `failureBlockedRepeatedly`) rather than a new outcome value or a new method, reusing `validDecisionOutcomes`'s existing `"failed"` bucket (`internal/store/decisions.go:18-23`, confirmed unchanged) so no migration touches this table's CHECK constraint a third time.

### `[AI-PLAYER]` structured log line, ids/stage/lengths only
**Source:** `internal/driver/driver.go:473-476` (`logDecision`, verified verbatim) and `internal/session/manager.go:452-455` (`logAutopilotTransition`, verified verbatim)
**Apply to:** every new stage this phase adds (loop tick, cap-reached, threshold-notice, retry, block-count, memory-update, retention-run) — reuse these two exact signatures unchanged; never interpolate the goal text, memory bullets, window text, or command (T-3-07's discipline, unchanged this phase).

### `AISettingsResponse`-shaped sub-resource GET/PUT
**Source:** `internal/profiles/handler.go:131-143,584-653,842-872` (struct + both handlers + validator, full read this session) and `frontend/src/services/api.ts:649-677` (`getAISettings`/`putAISettings`) and `frontend/src/components/AIPlayerPanel.tsx:20-100` (state/apply/save-body)
**Apply to:** the new goal sub-resource (`GetGoal`/`PutGoal`, `getGoal`/`putGoal`) and the read-only Session Memory sub-resource (`GetSessionMemory`, `getSessionMemory`) — same six-file chain shape 03.1-PATTERNS.md already used for `NeverIssueList`, now a fourth application of the identical pattern.

### `ringBuffer`/pure-transform, no I/O
**Source:** `internal/session/window.go` (full file, 80 lines, verified verbatim)
**Apply to:** the time-bounding addition (D-09) — stays a pure `[]byte`/`string` transform with no `context.Context`, no error return, no I/O, matching this file's own established minimalism (RESEARCH's own note that a terminal-emulation state machine was deliberately rejected here).

### `.delete-confirm`/`.btn-danger` two-step inline confirmation
**Source:** `frontend/src/components/ConnectionsHubModal.tsx:339-355` (full block, verified verbatim) and `frontend/src/index.css:1161,1624` (confirmed present)
**Apply to:** "Delete Captured Text Now" (D-21) — identical mechanism, not a new modal, per 04-UI-SPEC.md's own explicit instruction.

### `verify-phaseN.sh` canned-report harness shape
**Source:** `scripts/verify-phase3-1.sh` (full header verified verbatim, `set -uo pipefail`/`_get_field`/`tee`/three-invocation-mode/`--no-tests` contract)
**Apply to:** `scripts/verify-phase4.sh` — identical skeleton, new fixture directories under `scripts/fixtures/phase4/`, SKIP lines naming `go test ./internal/driver/... -run TestLoop_*` for the loop-pacing/counter/reengage behaviors this HTTP-only harness cannot reach.

## No Analog Found

None. Every file this phase touches is either a same-file self-extension of Phase 3/3.1 code (the majority — driver.go, manager.go, window.go, profile.go, transcripts.go, config.go, profiles/handler.go, session/handler.go, session/websocket.go, all five frontend files) or a new file with a direct, structurally identical sibling already in the codebase (`loop.go`/`memory.go` next to `driver.go`'s own function families; `quests.go` next to `decisions.go`/`transcripts.go`; `loop_test.go` next to `driver_test.go`'s existing table tests; `verify-phase4.sh` next to `verify-phase3-1.sh`). The two genuinely novel technical elements this phase introduces — a goroutine-per-user pacing loop with cancellation, and a Postgres `ON CONFLICT ... WHERE` partial-unique-index upsert — have no *literal* prior instance in this codebase, but both compose directly from patterns that already exist here (the ticker-goroutine shape in `startTimers`, and the already-proven `CREATE UNIQUE INDEX ... WHERE` migration idiom is new, but plain `CREATE UNIQUE INDEX IF NOT EXISTS` is not), so neither is flagged as "no analog" — RESEARCH.md's own "Don't Hand-Roll" table already worked out the composition for both.

## Metadata

**Analog search scope:** `internal/driver/`, `internal/session/`, `internal/store/`, `internal/config/`, `internal/crypto/`, `internal/profiles/`, `internal/gemini/`, `migrations/`, `cmd/server/`, `frontend/src/components/`, `frontend/src/pages/`, `frontend/src/services/`, `frontend/src/types/`, `frontend/src/index.css`, `scripts/`

**Files read this session (full or targeted, line numbers verified against current HEAD):** `internal/driver/driver.go` (full, 656 lines), `internal/driver/driver_test.go` (lines 1-260, 300-375, 861-945), `internal/session/manager.go` (lines 85-205, 440-620, 620-810), `internal/session/window.go` (full, 80 lines), `internal/session/handler.go` (lines 270-420), `internal/session/websocket.go` (lines 1-100), `internal/gemini/client.go` (lines 96-306), `internal/store/profile.go` (lines 1-110, 540-596), `internal/store/decisions.go` (full, 146 lines), `internal/store/transcripts.go` (full, 205 lines), `internal/config/config.go` (lines 1-60, 74-204), `internal/crypto/crypto.go` (lines 1-70), `internal/profiles/handler.go` (lines 131-370, 584-655), `cmd/server/main.go` (lines 180-310), `migrations/011_add_ai_session_tables.up.sql` (full), `migrations/012_add_never_issue_and_blocked_outcome.down.sql` (full), `frontend/src/components/AIAssistPanel.tsx` (full, 180 lines), `frontend/src/components/AIPlayerPanel.tsx` (full, 293 lines), `frontend/src/components/AutopilotBadge.tsx` (full, 54 lines), `frontend/src/pages/PlayScreen.tsx` (lines 395-435), `frontend/src/services/api.ts` (grep + lines 645-724), `frontend/src/types/index.ts` (lines 54-100, 290-320), `frontend/src/index.css` (lines 251-360), `frontend/src/components/ConnectionsHubModal.tsx` (lines 325-360), `scripts/verify-phase3-1.sh` (lines 1-60)

**Also read (upstream inputs, not re-cited as analogs):** `.planning/phases/04-continuous-play/04-CONTEXT.md` (full), `.planning/phases/04-continuous-play/04-RESEARCH.md` (full, 519 lines), `.planning/phases/04-continuous-play/04-UI-SPEC.md` (full, 426 lines), `.planning/phases/03.1-prompt-injection-review/03.1-PATTERNS.md` (full, format and depth precedent for this document)

**Pattern extraction date:** 2026-09-17
