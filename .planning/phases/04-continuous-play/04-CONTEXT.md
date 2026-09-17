# Phase 4: Continuous Play - Context

**Gathered:** 2026-09-16
**Status:** Ready for planning

<domain>
## Phase Boundary

With autopilot ON, the server-side Go driver runs read, decide, act continuously toward a session goal the owner types into the AI Assist panel, pacing itself to the game's output rather than a clock, until `#AUTO OFF`, the wheel-grab, the session call cap, or repeated failures stop it. Every decision and its reasoning stay visible as they happen (Phase 3's panel and terminal line, unchanged per decision). Re-engaging after hand driving starts from the current situation, provably, from the logged reasoning.

This phase also builds the first three layers of the owner's four-layer memory model (`.specify/specs/ai-memory-model-v1.md`): a time-bounded Immediate Context, a curated Session Memory the AI maintains for the life of a game connection, and Quest Memory tied to the goal, persisting across sessions. Historical Memory, end-of-session consolidation, and quest closure are Phase 6.

All four mechanical safety limits go under automated test: call cap when set, no AI-initiated reconnect, disengage on repeated errors, no commands while disconnected with resume on return, and blank-settings behaviour (no cap, informative failure, no crash, regular play unaffected). The repo's test scaffolding for these is created here.

Folded into this phase from the Phase 3 and 3.1 security reviews, on the owner's instruction: a retention standard for captured game text, the badge-state fix, the single 503 retry, the reviewer's untrusted-reasoning fix, the credential-vault key requirement, the migration 012 rollback fix, and the dead statement. Their closure is recorded at this phase's security review.

Not in this phase: coaching input, pause/resume from chat, and recall of Quest Memory through chat (Phase 5); the progression tally, debrief, Historical Memory, quest closure, and end-of-session promotion (Phase 6); the study loader (Phase 7); any change to the policy text (declined, stays 1.0).

</domain>

<decisions>
## Implementation Decisions

### Session goal and the active Quest
- **D-01:** The session goal is a **one-line text box at the top of the AI Assist panel**. It is stored server-side per connection profile, survives a page refresh, a reconnect, and `#AUTO OFF` then `#AUTO ON`, and lasts **until the owner changes or clears it**. No directive argument; `#AUTO ON` stays bare (Phase 3 D-11 unchanged).
- **D-02:** `#AUTO ON` with an **empty goal box engages anyway**. The AI plays from the profile's approach guidance as its standing direction and the panel shows "No session goal set". Blank means the profile's default direction, the same pattern as blank AI settings.
- **D-03:** The goal is **editable while the AI plays**. The next decision's prompt carries the new goal and the panel prints a system line in the family of "Goal changed: ...".
- **D-04:** **The goal box names the active Quest.** Typing a goal creates a Quest record for it or reactivates an open Quest whose goal text matches. The loop reads the active Quest's bullets into every prompt. Changing the goal switches which Quest is active; the previous one stays open for another day. Closing a Quest (succeeded, failed, abandoned, invalidated) is Phase 6's job; nothing in Phase 4 closes one.

### The loop and its pacing
- **D-05:** The loop is a **server-side goroutine per engaged user session**, started on every real engage (Phase 3's `HandleEngage` becomes the first iteration), stopped by a disengage of any cause, paused by WAITING, and restarted by a WAITING-to-ON resume (Phase 3 D-02 already treats a resume as an engage).
- **D-06:** The next decision fires **after new game output arrives and a short settle delay passes**, so a whole room description lands before the AI reads it. If the game stays quiet, a **floor interval** fires a decision anyway so the AI can nudge (look, continue). Decisions are never closer than a **minimum spacing**, and the ICM dispatcher's rate limit is the backstop. All three durations are server constants (Claude's discretion; starting points: settle about 1 to 2 seconds, floor about 20 seconds, minimum spacing a few seconds, tuned against Alter Aeon during the walkthrough).
- **D-07:** Per-decision display is unchanged from Phase 3: reasoning then command in the panel (D-08), `[AI-ASSIST > cmd]` in the terminal (D-09), one decision row per decision (D-12), blocked rows per Phase 3.1 D-08.
- **D-08:** **Reassessment on re-engage** is built and proven like this: every engage (`#AUTO ON` after OFF, and a WAITING-to-ON resume) starts with a fresh Immediate Context snapshot, and the first decision's prompt carries an explicit instruction to re-check the current situation against the goal before acting and to say so in its reasoning. No plan object exists to carry over: Session Memory carries facts, Quest Memory carries goal progress, and neither is a plan. The logged reasoning of the first post-re-engage decision, describing the current situation, is the evidence for success criterion 3.

### Memory layers built in this phase (`.specify/specs/ai-memory-model-v1.md` governs)
- **D-09:** **Immediate Context** is the existing rolling window (Phase 3 D-04) made **time-bounded**: roughly the last 10 seconds of game activity, with the current 8 KB byte size as the ceiling. It never goes empty: the most recent screenful is always retained so an engage after a quiet spell still sees the room. Exact seconds and screenful size are Claude's discretion.
- **D-10:** **Session Memory** is a curated bullet list of facts for the current game connection. It starts empty at connect, is updated by the model as part of each decision (add, update, merge, remove; see Claude's Discretion for the mechanism), has a size ceiling that forces compaction, and **lives for the game connection across stints, wheel-grabs, and WAITING resumes**. It is kept server-side and persisted so a refresh or server restart does not lose it, and at disconnect it is stored with the game session for Phase 6's consolidation. It is **shown read-only in the AI Assist panel** as a collapsible section that updates live; editing it by hand is Phase 5.
- **D-11:** **Quest Memory** is a per-profile, per-goal store of terse bullets germane to the goal (progress, discoveries, prerequisites, people and places, failed approaches, leads, milestones, blockers). It is created by D-04, read into every prompt while its Quest is active, and updated during play when something materially affects the goal (the model proposes the update in the same answer). It **persists across sessions**, is **stored separately** from Session Memory and from decision rows, and is **not shown on the panel**; the owner recalls it through the chat interface once Phase 5 builds it.
- **D-12:** **Detail is inversely proportional to age** (memory model Addendum 2). Immediate Context carries full recent text; Session Memory is compacted; Quest Memory is terse and goal-bound; Historical Memory (Phase 6) is footnotes. Prompt instructions and size ceilings for each layer follow this rule.
- **D-13:** The prompt now carries, in order: the profile's standing text (conduct rules, approach guidance, Never-issue list, verbatim per Phase 1 D-12), the session goal, the active Quest's bullets, Session Memory, and the Immediate Context window. The shape is stable for Phase 5 (coaching) and Phase 6 (Historical Memory) to add to. **Memory text is model-written from game text, so it is delimited as untrusted data** in both the player prompt and the reviewer prompt, exactly as the window is (Phase 3.1 D-01); instructions found inside memory are never followed.

### Halts, thresholds, and blocks
- **D-14:** The **call cap counts every model call in a stint**: the decision call, the reviewer call, and any retry. The count starts at zero on every `#AUTO ON`, including after a wheel-grab. When the next call would exceed the cap, the loop stops before making it and **autopilot lands OFF** with a locked notice in the family of "Session call cap reached. Autopilot disengaged." in the panel and terminal. A blank cap means no cap: the loop runs until stopped.
- **D-15:** The **error threshold counts consecutive transient failures** (the Phase 3 D-13 kinds: api-error, rate-limited, malformed, no-command, multi-command, non-game-line, icm-refused) and is **reset by a sent command**. This amends Phase 3 D-13 for the loop: a transient failure below the threshold sends nothing, shows its notice with the count (for example "2 of 3"), and the loop continues on the next tick; reaching the threshold disengages with the notice. Non-transient failures (missing configuration, auth error, missing profile) still disengage at once. A blank threshold resolves to 3 (Phase 1's default).
- **D-16:** **One automatic retry on a vendor 503** (DR-3-04): when the model answers 503 specifically, the same call is retried once after a short delay, with a visible notice in the panel and terminal while the retry is in flight (in the family of "The model is unavailable, retrying..."). A second 503 is a transient failure counted under D-15. This applies to both the decision call and the reviewer call, and the retry counts against the cap (D-14).
- **D-17:** A **blocked command skips the turn**: nothing is sent, the block is shown and stored per Phase 3.1 D-06 to D-08, and the loop waits for the next tick. Blocks are not failures and do not feed D-15's counter. **Consecutive blocks are counted separately** against the same threshold number; reaching it disengages with its own notice in the family of "AI decisions were blocked repeatedly. Autopilot disengaged." A sent command resets the block count. This stops a hostile room from spending the whole call cap.
- **D-18:** Every `ai` websocket message that carries a disengage (failure, cap, repeated blocks) **also carries the new switch state**, and the badge updates from that message (DR-3-03). No poll lag.
- **D-19:** A **disconnect while engaged pauses the loop in WAITING**; nothing is issued while disconnected; the AI never reconnects; a resume restarts the loop per D-05 and D-08. Alter Aeon dropped the socket about half a minute after every reconnect during the Phase 3.1 walkthroughs; the researcher investigates this before the loop is built, because a loop will feel it.
- **D-20:** The **Phase 4 test suite** (`go test ./internal/...`, fakes, no network) covers: cap halt when a cap is set and counts both calls; no cap when blank; threshold disengage with the notice; blank threshold resolving to 3; the 503 retry; consecutive-block disengage; no commands while disconnected and resume on return; no AI-initiated reconnect; re-engage producing a fresh window and the reassess instruction; and no crash on any failure. Evidence follows the project rule (test report, canned report, `[AI-PLAYER]` log excerpt, end-user screenshots; never a database query).

### Retention standard and audit stores (DR-3-01, DR-3.1-03)
- **D-21:** Captured game text is pruned on a schedule: **decision snapshots** (the window text on every decision row, blocked rows included) after **7 days**; **session transcripts** on the log page after **30 days**. Decision rows keep their timestamp, reasoning, command, outcome, and block reason forever. A **per-profile "delete captured text now"** action is built for both stores. A nightly server-side job does the pruning and logs one `[AI-PLAYER]` line per run with counts; the evidence is that log line and a canned report. Memory layers (Session, Quest, and later Historical) are not captured text and are not pruned by this job.
- **D-22:** The **policy stays at version 1.0**. The owner declined the section 2 rewording; AR-3-01 is accepted and not raised again.
- **D-23:** Staging stays on **`gemini-3.5-flash-lite`** for this phase.

### Security carry-forward remediations (folded; closure recorded at the Phase 4 review)
- **D-24:** DR-3.1-01: the reviewer prompt wraps the first model's stated reasoning in its own untrusted markers (position ahead of the window unchanged); a corpus item that attacks the reviewer through the reasoning channel is added; the AFTER red-team report is rerun on a fresh-quota day before the staging deploy.
- **D-25:** DR-3.1-04: the server **refuses to start when `ENCRYPTION_KEY_V1` is missing** outside local development; the variable is documented as required; a key-rotation procedure is written. Staging already has a stable key; production untouched.
- **D-26:** DR-3.1-05: migration 012's down file maps blocked rows to failed (or stops with a clear message) before re-adding the constraint.
- **D-27:** DR-3-07: the dead statement in `EngageAutopilot` (`internal/session/manager.go`) is deleted in the first plan that touches the session manager.
- **D-28:** Closed by this phase's decisions, to be recorded at the review: DR-3-01 and DR-3.1-03 by D-21; DR-3-03 by D-18; DR-3-04 by D-16; DR-3.1-02 by D-14. DR-3-05 (push at phase close) is confirmed as held; DR-3-06 (who can read the staging log) is an owner admin task whose outcome is recorded at the review.
- **D-29 (added during execution, 2026-09-17; amends Phase 3.1 D-03's harm list):** The reviewer's list of harmful acts gains one class, verbatim: "binding the character to a contract, oath, pledge, debt or membership", still qualified by the existing demand, threat, deadline or promised-reward clause. Reason: plan 04-11's corpus rerun showed `direct-03` (sign a binding contract under a threat and a deadline) reaching the send path in two runs; the diagnosis (`.planning/debug/reviewer-regression-direct-03.md`, `evidence/04d-reviewer-regression-diagnosis.txt`) showed this was not a Phase 4 regression but a gap in the closed list, the reviewer blocking the identical input 5 times in 14 without the class and 4 in 4 with it. The owner accepted the wording as written, knowing a legitimate guild invitation or loan offered with a reward or deadline may be blocked (visible, and he can do it by hand). Code: commit `4e8243a`. Carried to the Phase 4 security review as its own item: the reviewer only reliably blocks acts the list names, and one sample per corpus item cannot show a coin-flip verdict.
- **D-30 (added during execution, 2026-09-17; amends `04-UI-SPEC.md`'s "no new pixel literals"):** The AI Assist panel grows from 480px to 760px tall (still capped by `max-height: calc(100vh - 160px)`), because with the goal box, status line and memory header inside the Phase 3 geometry the decision list fitted one decision. Owner's choice at the walkthrough: "Make the panel taller".
- **D-31 (added during execution, 2026-09-17; amends D-10's lifetime):** Owner's words at the walkthrough: "The session memory should stick around for as long as the browser session... meaning per login to MUD Puppy... if refreshing doesn't require a new login then session memory should stay as well." So **Session Memory lives for the MUDPuppy login, per connection profile**, not for the game connection: a page refresh, a game reconnect, a dropped socket or a server redeploy keeps it; the first game connection after a new sign-in (or after a sign-out) starts empty. Found because a refresh re-makes the game connection in this product, which under D-10's "starts empty at connect" wiped the AI's notes. Mechanism (Claude's discretion): the users row records when the current login began; a new game session's `session_memory` is seeded from the same user's and profile's most recent earlier game session that started at or after that moment. Each game session still stores its own copy at disconnect for Phase 6's consolidation. A second sign-in elsewhere starts a new login and therefore a clean memory; accepted.
- **D-06 tuning note (Claude's discretion, 2026-09-17):** minimum spacing raised from 3s to 8s after the staging walkthrough: two model calls per decision at 3s spacing reached the model's per-minute rate limit within a minute (18 calls in about 50 seconds, two consecutive rate-limit failures).

### Claude's Discretion
- How the model curates Session Memory and proposes Quest Memory updates: the intended mechanism is extra fields in the existing decision answer JSON (for example `session_memory` add/update/remove entries and `quest_memory` entries) so no extra model call is spent and the cap is unaffected; the researcher confirms the schema fits Gemini's constrained output.
- The Immediate Context's seconds, the retained screenful size, and where the time stamps live in the ring buffer.
- Session Memory and Quest Memory size ceilings, their storage shape (a JSON column per game session and per quest, or line rows), migration numbers (`013`+), and how the quest's goal text is matched on reactivation (trim and case-fold is fine).
- Loop constants (settle delay, floor interval, minimum spacing) and the goroutine's lifecycle, cancellation, and in-flight guard, building on `Driver.inFlight`.
- Exact wording of the new notices (cap reached, failure with count, retrying, blocked repeatedly, goal changed, no goal set, memory updated), in the voice of `03-UI-SPEC.md`'s Copywriting Contract; locked strings with no vendor text interpolated.
- The panel's layout for the goal box and the collapsible Session Memory section; where the "delete captured text now" action sits (Logs section or AI Player section).
- The `ai` websocket message fields for goal, switch state, counts, and memory updates; the REST routes for goal get/set and memory read on attach.
- The nightly job's implementation (a ticker in the server process is fine; no new infrastructure) and its schedule.
- The `[AI-PLAYER]` log lines for loop ticks, cap and threshold counts, retries, blocks, memory updates, and retention runs, following the existing `stage=` format with ids, stages, outcomes, and lengths only.
- Which existing tests move into a shared test scaffold and how the Phase 4 suite is organised.
- Whether icm-refused counts as transient (default: yes, it is a rate or depth limit) and the retry delay for the 503.

### Folded Todos
- **`.planning/todos/pending/2026-09-16-phase4-carry-forward-from-3.1.md`** (owner's instructions, resolves Phase 4). Three remediations the owner asked for (DR-3.1-01, DR-3.1-04, DR-3.1-05) become D-24, D-25, D-26. Two to decide again (DR-3.1-02, DR-3.1-03) are closed by D-14 and D-21. Its design inputs (what a block means for the loop, the Alter Aeon socket drop, the staging model) are D-17, D-19, D-23. Its backlog items (WR-03 Never-issue hint wording, IN-02 corpus pacing, the stale worktree directory) remain backlog.
- **`.planning/todos/pending/2026-09-16-phase4-security-carry-forward.md`** (owner's instructions, resolves Phase 4). DR-3-01 becomes D-21, DR-3-03 becomes D-18, DR-3-04 becomes D-16, DR-3-07 becomes D-27, DR-3-05 and DR-3-06 are review items (D-28). AR-3-01's policy rewording was presented and declined (D-22). AR-3-08 (`GOOGLE_API_KEY` on production) stays as a production cut-over note. The DR-3-02 section was already resolved by Phase 3.1.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### The memory model (new this phase)
- `.specify/specs/ai-memory-model-v1.md` — The owner's four-layer memory model with two addenda (Quest Memory; detail inversely proportional to age). Governs on memory where it differs from design v3. Phase 4 builds Immediate Context, Session Memory, and Quest Memory; Phase 6 builds the rest.

### Product definition and method
- `.specify/specs/ai-game-player-design-v3.md` — D4 (continuous play) is this phase; Definition of complete items 2, 3, and 6; D6's "session log with decisions and reasoning and manual-driving stretches".
- `.specify/specs/safety-and-abuse-policy-v1.md` — Sections 2 (supervised operation; stays at 1.0 by owner decision), 5 and 6 (data handling; the retention standard D-21 answers the open question from Phase 3).
- `.specify/memory/phase-based-development-approach.md` — Phase → Wave → Plan → Task rules; plans are capabilities with observable acceptance criteria.
- `CLAUDE.md` — Evidence is a canned report, log file, or end-user screenshot; Brief, Evidence Dossier, and Risk Register per phase; push `ai-player` at phase close; deploy staging only with `railway up`.

### Planning state and prior decisions this phase builds on
- `.planning/ROADMAP.md` §Phase 4 — Goal, four success criteria, Phase Validation line, implementation notes (goroutine per session, pacing on output or floor, per-session counters, fresh prompt on re-engage, tests created here). §Phase 6 wording will need amending for Historical Memory and quest closure (see Deferred).
- `.planning/REQUIREMENTS.md` — REQ-continuous-loop, REQ-call-cap-and-error-disengage, REQ-reengage-reassess, REQ-doc-continuous-visible-play, REQ-doc-wheel-grab-and-reengage, REQ-safety-limits-hold (amended text: autopilot waits and resumes).
- `.planning/PROJECT.md` — Locked decisions DEC-server-side-go-driver, DEC-server-side-is-not-unattended, DEC-reconnect-is-connection-toggle, DEC-autopilot-waits-across-disconnect; constraints CON-loop-pacing, CON-mechanical-halts, CON-conservative-defaults, CON-policy-cost, CON-policy-data-handling, CON-engine-has-no-game-knowledge.
- `.planning/phases/03.1-prompt-injection-review/03.1-CONTEXT.md` — D-01 (untrusted delimiting; D-13 here extends it to memory), D-04 (check order), D-05 (reviewer call counts against the cap), D-06 to D-09 (blocked outcome; D-17 here decides what a block means for the loop).
- `.planning/phases/03-one-ai-decision/03-CONTEXT.md` — D-01 to D-03 (engage fires a decision, resume is an engage, stays ON idle: now looped), D-04 (window; made time-bounded by D-09), D-05 (two-tier prompt; extended by D-13), D-08 to D-10 (panel and terminal display, unchanged), D-12 (decision rows), D-13 (failure kinds; amended by D-15), D-14 to D-17 (transcripts and the log page; pruned by D-21), D-18 to D-20 (model registry, free tier, missing-config refusal).
- `.planning/phases/03-one-ai-decision/03-UI-SPEC.md` — Copywriting Contract; the new notices join that voice.
- `.planning/phases/02-autopilot-switch/02-CONTEXT.md` — State model (ON, OFF, WAITING), wheel-grab, the `[AI-PLAYER]` log format.
- `.planning/phases/01-profile-foundation-and-policy-gate/01-CONTEXT.md` — D-09 to D-12 (blank settings, fail informatively, standing text verbatim).

### Security carry-forward
- `.planning/RISK-REGISTER.md` — Phase 3 deferred rows DR-3-01, DR-3-03 to DR-3-07 and Phase 3.1 rows DR-3.1-01 to DR-3.1-05, all "Raised again at: Phase 4 security review"; AR-3-01 and AR-3-08 notes.
- `.planning/phases/03.1-prompt-injection-review/03.1-SECURITY.md` and `.planning/phases/03-one-ai-decision/03-SECURITY.md` — The threat registers and audit-trail format this phase's review extends.
- `.planning/todos/pending/2026-09-16-phase4-carry-forward-from-3.1.md` and `.planning/todos/pending/2026-09-16-phase4-security-carry-forward.md` — The folded todos, with the owner's notes verbatim.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/driver/driver.go`: `HandleEngage` is the complete single-decision path (window snapshot, `buildSystemInstruction`, `GenerateContent`, `validateCommand`, `matchNeverIssue`, `ReviewCommand`, `Dispatch` with `ContextAutomation`, `SendCommandAs`, `recordSuccess`/`recordFailure`/`recordBlocked`). The loop wraps it as one iteration; the counters (calls, consecutive failures, consecutive blocks) and the 503 retry hang off it. `Driver.inFlight` is the per-user guard to build the goroutine lifecycle on. `Event` (Kind, Reasoning, Command, Outcome, Message) is the `ai` message payload that gains switch state, counts, goal, and memory fields.
- `internal/driver/driver_test.go`: fakes for `Sessions`, `Profiles`, `Decisions`, `Models`, `Commands`, and the `TestHandleEngageFailures` table; the Phase 4 safety-limit suite extends these fakes with a scripted sequence of answers and errors. `corpus_live_test.go` is where the reviewer-channel corpus item (D-24) goes.
- `internal/gemini/client.go`: `GenerateContent` and `ReviewCommand` with `responseSchema` and `propertyOrdering`; the decision schema gains the memory fields (Claude's discretion); `gemini.Error.Status` already exposes the HTTP status the 503 retry keys on.
- `internal/session/window.go`: `RecentOutputWindowBytes = 8192`, `ringBuffer` with `append`/`snapshot`, `stripANSI`; time-bounding (D-09) adds timestamps to appends and a time-aware snapshot. `Manager.appendOutputWindow` and `RecentOutputSnapshot` are the hooks; the loop's "new output arrived" signal comes from the same relay path.
- `internal/session/manager.go`: `EngageAutopilot` (dead statement for D-27), `DisengageAutopilot(userID, cause)`, `parkAutopilotLocked`, `resumeAutopilotLocked` (fires `engageHook` with `go`; the loop restart on resume), `AutopilotStateFor`, `logAutopilotTransition`, `openTranscript`/`closeTranscript` (where Session Memory is created at connect and stored at disconnect), `CurrentGameSessionID`.
- `internal/session/websocket.go`: `WSMessage`, `MsgTypeAutopilot`, the `ai` message push; `handler.go` `Autopilot` handler and `StatusResponse` (switch state for D-18).
- `internal/store/profile.go`: `AISettings`, `ResolveAISettings` (`CallCapSet`, `CallCap`, `DisengageThreshold`, `DefaultDisengageThreshold = 3`); `internal/store/decisions.go` `DecisionRecord{WindowText, ...}` (snapshot pruning target); `internal/store/transcripts.go` (transcript pruning target); `internal/profiles/handler.go` `validateUpdate` and the AI sub-resource GET/PUT pattern for the goal endpoint.
- `internal/crypto/crypto.go` `DefaultKeyStore` (D-25); `internal/config/config.go` `Load()` required/optional env pattern (`SESSION_SECRET` fatal is the model for `ENCRYPTION_KEY_V1`).
- `migrations/012_add_never_issue_and_blocked_outcome.down.sql` (D-26); next migration number is `013`.
- `frontend/src/components/AIAssistPanel.tsx` (goal box, Session Memory section, new system lines), `AutopilotBadge.tsx` (state from the `ai` message), `AIPlayerPanel.tsx` (settings; a possible home for delete-captured-text), `frontend/src/services/api.ts` (`ai` event dispatch, new fetches), `SettingsPage.tsx` Logs section.
- `scripts/verify-phase3-1.sh` and `scripts/fixtures/`: the canned-report harness pattern for a Phase 4 harness (goal round-trip, memory read-back, retention job counts).

### Established Patterns
- Server owns truth; the browser reflects it and re-syncs on attach. Goal, Session Memory, switch state, and counts all flow server to browser over the `ai` message and reload from REST after a refresh.
- Security boundaries live in Go in front of the dispatcher, never in the vendor client; untrusted delimiting is done in `buildSystemInstruction`/`wrapWindow` and the reviewer prompt, and D-13 extends it to memory.
- Locked notices are fixed strings, no vendor text interpolated; `[AI-PLAYER]` log lines carry ids, stages, outcomes, and lengths only.
- Defaults resolve in Go where the engine needs a number (`ResolveAISettings`), never in the browser.
- Tests with fakes and no network are the default; `-race` availability is reported in the test report.
- Evidence-as-file per success criterion in `evidence/` (test report, harness output, canned report, staging log excerpt, numbered screenshots).

### Integration Points
- Driver: loop goroutine around the decision path; counters and the retry; reassess instruction on the first iteration; memory fields in the answer schema; memory text in both prompts, delimited as untrusted.
- Session manager: loop start on engage and resume, pause on park, stop on disengage; Session Memory lifecycle on connect/disconnect; time-bounded window.
- Store and migrations: goal per profile (or per connection profile record), Session Memory per game session, Quest Memory per profile and goal, retention job over decision snapshots and transcripts.
- Websocket and REST: goal get/set, memory read on attach, `ai` message with switch state and counts.
- Frontend: goal box, Session Memory section, badge from the `ai` message, delete-captured-text action.
- Config and startup: `ENCRYPTION_KEY_V1` required outside dev; nightly retention ticker.
- Tests: the safety-limit suite (D-20) and the reviewer-channel corpus item (D-24).
- Security review: Phase 4 Risk Register carrying the twelve deferred risks with their outcomes (D-28) and any new active risks (memory as an injection channel, D-13, is the one to watch).

</code_context>

<specifics>
## Specific Ideas

- Owner on memory: "the agent should maintain three different layers of game memory rather than treating its entire play history as one continuously retained block of game text," then the Quest Memory addendum, then: "the details of what is stored in these memory files needs to be inversely proportional to the length of time that it's taken since they were committed to memory." All saved verbatim in `.specify/specs/ai-memory-model-v1.md`.
- Owner on Quest Memory visibility: "It should be stored separately, but can be recalled through the AI chat interface." Hidden in Phase 4; Phase 5 adds recall.
- Owner on the decision audit record: the snapshot "should be treated as a separate, pruneable artifact rather than as part of the AI's persistent memory." Hence D-21's 7-day snapshot window.
- Owner on the policy rewording: leave it at 1.0.
- Carried from Phases 1 to 3.1: the owner does not want plumbing turned into decisions; pick the simplest thing and move on. Risk items and notices in plain words.

</specifics>

<deferred>
## Deferred Ideas

- **Quest Memory recall through the chat interface** — Phase 5 (REQ-coaching-chat). The chat pane is where the owner asks what the AI knows about the active goal.
- **Editing Session Memory by hand** — Phase 5, alongside coaching and promotion.
- **End-of-session evaluation of active Quests, quest closure into Historical footnotes, Historical Memory itself** — Phase 6 (REQ-session-debrief, REQ-memory-carryover). The roadmap's Phase 6 goal and success criteria need amending to name Historical Memory and quest closure per the memory model; do this with `/gsd-phase edit 6` before Phase 6 is discussed.
- **Policy v1.1 rewording of section 2** — declined by the owner; the policy stays at 1.0. Not to be re-proposed.
- **Per-profile retention settings** — a server default is enough now (7 and 30 days); a per-profile override could come later if the owner asks.
- **Fake game fixture or diagnostic endpoint on staging** — declined in Phase 3.1; the safety-limit suite runs on fakes and the staging demonstration uses the owner's own character on Alter Aeon.
- **Backlog from the 3.1 todo (defects, not risks):** WR-03 Never-issue hint wording versus `matchNeverIssue`; IN-02 corpus pacing between an item's two calls; the stale `.claude/worktrees/agent-a084c785f3d4c84a8` directory.
- **AR-3-08** `GOOGLE_API_KEY` on production: leave it; note for the production cut-over checklist that the server reads the `AI_MODEL_*` variables.

### Reviewed Todos (not folded)
None: both pending todos were folded in full.

</deferred>

---

*Phase: 04-continuous-play*
*Context gathered: 2026-09-16*
