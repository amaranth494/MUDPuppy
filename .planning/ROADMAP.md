# Roadmap: AI Game Player for MUDPuppy

## Overview

Eight phases, one per deliverable in the design (D1-D8), in the design's stated order. Phases 1-4 are strictly sequential and each is a working, testable increment: profile fields and the policy gate, then the autopilot switch with no intelligence behind it, then one real Gemini decision through the ICM automation context, then the continuous loop with every mechanical safety limit under test. Phase 5 (coaching) and Phase 6 (measurement and memory) both build on Phase 4; Phase 5 goes first because the early sessions are collaboration-heavy. Phase 7 (study loader) needs Phase 6's learned notes. Phase 8 is the finish line: the acceptance run on Alter Aeon from Railway staging, where the Definition of complete items 1-6 all hold.

Granularity: standard (no `.planning/config.json` present; default applied). The phase count is fixed at eight by the owner's instruction to preserve the deliverable structure, which falls inside the standard band.

Branch: `ai-player` (exists from `staging`, carries the ICM). Environment: Railway project `mudpuppy`, environment `staging`. Source of truth: `.specify/specs/ai-game-player-design-v3.md` and `.specify/specs/safety-and-abuse-policy-v1.md` only.

## Phases

**Phase Numbering:**

- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Profile Foundation and Policy Gate** - Profiles gain conduct rules, approach guidance, AI settings with conservative defaults, and a one-time Safety and Abuse policy acceptance that gates AI configuration and engagement (completed 2026-09-15)
- [x] **Phase 2: Autopilot Switch** - `#AUTO ON` / `#AUTO OFF`, a server-owned engaged state with a truthful indicator, the wheel-grab rule, and a waiting state across disconnects that resumes on return, all with no AI behind it
 (completed 2026-09-15)

- [x] **Phase 3: One AI Decision** - ICM engine wired server-side, Gemini connected from env config, one decision made and issued through the automation context, reasoning shown live and persisted (completed 2026-09-16)
- [x] **Phase 3.1: Prompt Injection Review** (INSERTED) - Game text cannot steer the AI into a command the owner would not sanction; the investigation and any mitigation are proven by tests with hostile room text and demonstrated on staging (urgent, from the Phase 3 security review, DR-3-02) (completed 2026-09-16)
- [ ] **Phase 4: Continuous Play** - Session goal, paced read/decide/act loop, call cap and error disengage with visible notices, clean reassessment on re-engage, all safety limits under test
- [ ] **Phase 5: Coaching Channel** - Chat pane beside the terminal: guidance lands in the next decision, pause/resume, promote guidance into the profile
- [ ] **Phase 6: Measurement and Memory** - Progression tally from the game's status numbers, session debriefs, learned notes carried into the next session, manual driving captured as demonstrations
- [ ] **Phase 7: Study Loader** - Paste or upload material against a profile, AI summarizes in chat, confirmed lessons enter learned notes
- [ ] **Phase 8: Acceptance Run on Alter Aeon** - Reconnaissance then goal sessions from staging under the owner's supervision; Definition of complete items 1-6 hold, including the three-session improvement trend

## Phase Details

### Phase 1: Profile Foundation and Policy Gate

**Goal**: The connection profile is the single per-game home for the AI, and no profile can have the AI configured or engaged until its owner has accepted the Safety and Abuse policy on it, once.
**Depends on**: Nothing (first phase)
**Requirements**: REQ-profile-ai-fields, REQ-policy-gate
**Success Criteria** (what must be TRUE):

  1. A profile stores and returns conduct rules, approach guidance, and AI settings (model name, call cap per session, disengage threshold). A blank model name means the server default, a blank call cap means no cap, and a blank threshold means the engine's built-in error handling: any AI failure yields an informative error in the play screen and disengages without crashing or interrupting regular play. There is no reconnect field in AI settings.
  2. The owner can read and edit conduct rules, approach guidance, and AI settings for a profile from the browser, and the values survive a page reload and a new session.
  3. The first time the owner opens the AI Player configuration for a profile, the policy is presented and must be accepted before AI settings can be edited; the server-side engage gate refuses any profile without a recorded acceptance, with a clear message.
  4. Acceptance is recorded once per profile with timestamp and the policy version accepted (currently 1.0); it never expires, a later policy change does not require re-acceptance, and deleting the profile discards it.

**Plans**: 6 plans
Plans:
**Wave 1**

- [x] 01-01-PLAN.md — Profile row carries AI fields and a one-time policy acceptance record, with blanks resolved in Go
- [x] 01-02-PLAN.md — Policy text and version 1.0 ship inside the server binary

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 01-03-PLAN.md — AI settings, the policy, one-time acceptance and the engage gate are reachable over HTTP

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 01-04-PLAN.md — The owner accepts the policy and edits AI settings from the browser
- [x] 01-06-PLAN.md — One command turns the Phase 1 HTTP sequence into a canned PASS/FAIL report per success criterion

**Wave 4** *(blocked on Wave 3 completion)*

- [x] 01-05-PLAN.md — Phase 1 demonstrated on staging and filed as evidence: test report, staging log excerpts, canned report, screenshots

**UI hint**: yes

**Phase Validation** (how the success criteria are demonstrated): Diagnostic: `go test` covers default resolution for blank AI settings and the engage-gate decision (not accepted, accepted); server logs on staging show migration 010 applied at startup, one structured line per policy acceptance carrying the connection id, policy version 1.0 and the timestamp, and one line per engage-gate decision carrying the connection id and the outcome. Proof for every criterion is a UAT finding from the browser walkthrough or a server log excerpt; database queries are not accepted as evidence. Player-observable: open the AI Player section on a fresh profile and the policy appears first; accept once, then the settings editor is usable; edit the three fields, reload, values persist; call the engage gate on an un-accepted profile and receive the refusal message; on the accepted profile it passes; delete the profile, recreate it, and the policy is asked again.

Implementation notes for planning: new golang-migrate migration starting at `010` (profiles gain conduct_rules, approach_guidance, ai_settings JSONB, and policy acceptance columns: version accepted plus timestamp); new GET/PUT sub-resource(s) under `/api/v1/profiles/{connection_id}/...` in `internal/profiles`; a server-side engage-gate check that Phase 2's `#AUTO ON` will call; conservative defaults live in Go, not in the frontend. Policy text is served from the server from a markdown file embedded in the Go binary; the version string in its header is what gets recorded on acceptance.

### Phase 2: Autopilot Switch

**Goal**: The owner can engage and disengage autopilot from the terminal and always see the true state; taking the wheel is instant and lossless; a disconnect never turns autopilot off and never issues commands, only `#AUTO OFF` turns it off. No AI intelligence exists yet.
**Depends on**: Phase 1
**Requirements**: REQ-autopilot-directives, REQ-wheel-grab, REQ-no-auto-reconnect, REQ-doc-hand-play-and-gate
**Success Criteria** (what must be TRUE):

  1. `#AUTO ON` engages only when the profile passes the Phase 1 gate (otherwise refused with the gate's message); `#AUTO OFF` disengages; the play-screen indicator always matches the true server-held state, including after a page refresh.
  2. Any game command typed by the human while engaged disengages autopilot before the command is sent, with no lost keystrokes; commands fired by browser aliases, triggers, or timers do not trip the wheel-grab.
  3. A disconnect while engaged moves autopilot to a waiting state that issues nothing; the AI never initiates a reconnect; when the connection comes back by any means autopilot resumes to ON by itself; `#AUTO OFF` while waiting lands on OFF and it stays OFF after the connection returns. (Owner amendment 2026-09-15.)
  4. The owner can create a game profile, accept the policy on it, and hand-play the character normally with the AI disengaged; nothing about ordinary play changes.

**Plans**: 7 plans
Plans:
**Wave 1**

- [x] 02-01-PLAN.md — Autopilot holds a position on the server and a dropped connection parks it instead of flipping it off

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 02-02-PLAN.md — The switch answers over HTTP and refuses without the policy gate or a connected game
- [x] 02-03-PLAN.md — Typing takes the wheel; triggers and timers do not

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 02-04-PLAN.md — `#AUTO` is a directive the owner can type, and the browser labels every command human or automation
- [x] 02-05-PLAN.md — One command turns the Phase 2 HTTP sequence into a canned PASS/FAIL report per success criterion

**Wave 4** *(blocked on Wave 3 completion)*

- [x] 02-06-PLAN.md — The badge and the notices tell the truth, including after a refresh

**Wave 5** *(blocked on Wave 4 completion)*

- [x] 02-07-PLAN.md — Phase 2 demonstrated on staging and filed as evidence: test report, canned report, staging log excerpt, screenshots, security-review agenda

**UI hint**: yes

**Phase Validation** (how the success criteria are demonstrated): Player-observable on staging against a live MUD: `#AUTO ON` is refused before acceptance and engages after it; the indicator matches the server state after a page refresh; typing a command while engaged shows the disengage notice and the command's game response; a trigger-fired command leaves autopilot engaged; dropping the connection shows the indicator as waiting and no command is sent while disconnected; reconnecting by hand shows it resuming to ON without typing `#AUTO ON`; `#AUTO OFF` while waiting lands on OFF and it stays OFF after reconnecting. Diagnostic: `go test` covers the engaged-state machine and the human-versus-automation source flag.

Implementation notes for planning: `#AUTO` is parsed in the browser directive grammar (`frontend/src/services/automation.ts`) and calls new server endpoints; engaged/disengaged state is owned server-side (per user session) and broadcast to the browser over the existing websocket (`status` or `sync` message, or a new AI message type) so refresh re-syncs; websocket input messages gain a source flag (human vs automation) so the server can apply the wheel-grab only to human-typed input; the session manager's disconnect path moves an engaged autopilot to waiting and the connect path resumes it; a server restart loses in-memory state and lands on OFF.

### Phase 3: One AI Decision

**Goal**: The thinnest slice through the whole stack works: with autopilot engaged, Gemini reads the game, makes one decision, and the command is issued through the ICM automation context, with the reasoning visible live and stored.
**Depends on**: Phase 2
**Requirements**: REQ-single-decision, REQ-reasoning-visibility, REQ-env-config
**Success Criteria** (what must be TRUE):

  1. With autopilot engaged, the driver reads the current game text, sends one request to Gemini with the profile's conduct rules and approach guidance included, and issues the returned command through the ICM automation context so it passes the ICM dispatcher and its safety limits.
  2. The decision and the model's reasoning appear in the play screen as they happen and are stored in the session log, where they survive a page refresh.
  3. Model names and API key come from environment configuration on staging (`internal/config/config.go`); nothing is hard-coded, and the server fails clearly if they are absent when engagement is attempted.
  4. A command submitted on the server in the automation execution context is dispatched through the ICM engine's dispatcher and safety checker, demonstrable by a diagnostic test, and the frontend adapter's ICM calls no longer fall back silently to browser-side logic.

**Plans**: 13 plans
Plans:
**Wave 1**

- [x] 03-01-PLAN.md — The server holds a rolling, ANSI-free window of the recent game text
- [x] 03-02-PLAN.md — The server can ask Gemini for one decision, with model and key taken only from the environment
- [x] 03-03-PLAN.md — The ICM engine is live and an automation-context command provably passes its safety checker
- [x] 03-04-PLAN.md — `#AUTO` answers only to ON and OFF

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 03-05-PLAN.md — Every saved-profile connection is transcribed and the transcript can be read back by its owner
- [x] 03-06-PLAN.md — `#AUTO ON` is refused with a clear notice when the AI is not configured on this server
- [x] 03-07-PLAN.md — The staging sign-in code never reaches the log (DR-2-01)

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 03-08-PLAN.md — With autopilot engaged the AI makes one decision and its command reaches the game through the ICM automation context

**Wave 4** *(blocked on Wave 3 completion)*

- [x] 03-09-PLAN.md — The decision and its reasoning reach the browser as they happen and can be re-read after a refresh

**Wave 5** *(blocked on Wave 4 completion)*

- [x] 03-10-PLAN.md — The owner watches the AI's reasoning in the AI Assist panel and sees its command marked in the terminal
- [x] 03-12-PLAN.md — One command turns the Phase 3 HTTP sequence into a canned PASS/FAIL report per success criterion

**Wave 6** *(blocked on Wave 5 completion)*

- [x] 03-11-PLAN.md — The owner opens a profile's session logs in a new tab and reads any past session

**Wave 7** *(blocked on Wave 6 completion)*

- [x] 03-13-PLAN.md — Phase 3 demonstrated on staging and filed as evidence: test report, canned report, staging log excerpt, screenshots, security-review agenda

**UI hint**: yes

**Phase Validation** (how the success criteria are demonstrated): Player-observable on staging: with autopilot engaged, one decision and its reasoning appear in the play screen and the game responds to the issued command. Diagnostic: a decisions table row exists for that decision and is returned after a page refresh; a `go test` proves the command passed through the ICM dispatcher in the automation context; with the Gemini environment variables unset, engagement fails with a clear error and no command is sent.

Implementation notes for planning: a server-side tap on the MUD output stream (`internal/session`) giving the driver a bounded buffer of recent game text; new tables for AI sessions and decisions (migration `011`+); a Gemini client package; a new websocket message type carrying decisions and reasoning to the play screen; the driver package is Go under `internal/` and calls the ICM dispatcher with `ContextAutomation`.

### Phase 3.1: Prompt Injection Review (INSERTED)

**Goal:** Game text, including other players' speech and hostile room descriptions, cannot steer the AI into issuing a command the owner would not sanction. The investigation establishes how far shape validation and the ICM safety checker already bound the risk and what a content-level mitigation would add; whatever is built is proven by tests with hostile game text and demonstrated on staging. Inserted from the Phase 3 security review (DR-3-02, owner: "dig into this as an emergency security task prior to the next Phase").
**Requirements**: None mapped — inserted security phase answering risk DR-3-02 (`.planning/RISK-REGISTER.md`); it protects REQ-single-decision's command path and REQ-safety-limits-hold's spirit without adding a requirement ID.
**Depends on:** Phase 3
**Success Criteria** (what must be TRUE):

  1. A hostile-text corpus (hostile room descriptions, another player's `say`/`tell`, item or sign text, text impersonating the game's system messages, text impersonating the owner or the conduct rules, multi-step setups spread across the window, plus benign control windows) runs as a flagged Go test from the dev machine against the real Gemini model, and a red-team report is filed for the pre-build baseline and again after the build, counting catches per layer and listing every wrongly blocked benign window.
  2. The prompt delimits the game-text window as untrusted data and the system instruction states that instructions found inside game text are never followed, while the profile's conduct rules and approach guidance still reach the model verbatim.
  3. The profile carries an owner-editable Never-issue list (a text box in the AI Player settings section, one entry per line, empty by default) that is stored and returned with the other AI fields, handed to the model, and enforced in Go before the reviewer pass and before ICM dispatch: a command that starts with an entry never reaches the dispatcher.
  4. A reviewer pass (a second Gemini call, constrained to a JSON verdict with a one-sentence reason) judges the chosen command against the conduct rules and the game text and blocks it if it breaks a rule or follows an instruction embedded in the game text; if the reviewer call itself fails, nothing is sent and autopilot disengages with the existing failure notice.
  5. A blocked command sends nothing to the game and leaves autopilot ON and idle; it is stored as a decision row with outcome `blocked` and its reason, reloads into the AI Assist panel after a page refresh, and the owner sees the blocked command and the reason in the panel and as one bracketed terminal line.
  6. After the build, zero steered commands from the corpus reach the send path, and on staging the owner's own `say` of an injection line on Alter Aeon is shown being blocked with its reason.

**Plans:** 7/7 plans complete

Plans:
**Wave 1**

- [x] 03.1-01-PLAN.md — The hostile-text corpus runs against today's build and files the baseline red-team report
- [x] 03.1-02-PLAN.md — The owner can forbid commands on a profile, and the server stores and returns the list

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 03.1-03-PLAN.md — Game text reaches the model marked untrusted, and a forbidden command never reaches the dispatcher

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 03.1-04-PLAN.md — A second model pass reviews the chosen command, and a review that cannot happen stops it
- [x] 03.1-05-PLAN.md — The owner sees the blocked command and the reason, live and after a refresh
- [x] 03.1-06-PLAN.md — One command turns the Phase 3.1 HTTP sequence into a canned PASS/FAIL report per success criterion

**Wave 4** *(blocked on Wave 3 completion)*

- [x] 03.1-07-PLAN.md — Phase 3.1 demonstrated on Alter Aeon from staging and filed as evidence: red-team before and after, test report, canned report, staging log excerpt, screenshots, security-review agenda

**UI hint**: yes

**Phase Validation** (how the success criteria are demonstrated): Diagnostic: `go test ./internal/...` passes a hostile-content table with a fake model (Never-issue prefix match, reviewer block on each of the two questions, reviewer failure fails closed and disengages, benign command passes all three layers) and the profile field round-trips; the flagged live corpus test writes the red-team report against the pre-build commit and again against the finished build, showing catches per layer and zero steered commands on the send path after the build. Player-observable on staging: the owner types `say` plus an injection line, then `#AUTO ON`; the AI Assist panel shows the blocked command and its reason, the terminal prints the blocked line, the autopilot indicator stays ON, a page refresh reloads the blocked decision, and the Never-issue box in AI Player settings persists its entries. Proof for every criterion is a canned report (test report, red-team report, harness output, `[AI-PLAYER]` staging log excerpt with ids, stages, verdicts and lengths only) or an end-user screenshot; database queries are not accepted as evidence.

Implementation notes for planning: the two new stages slot into `HandleEngage` in `internal/driver/driver.go` between `validateCommand` and `Dispatch`; the reviewer is a second use of `internal/gemini/client.go` with its own response schema; the Never-issue field joins `Profile` in `internal/store/profile.go` and the AI sub-resource in `internal/profiles/handler.go` with the next migration number; `blocked` is a third decision outcome in `internal/store/decisions.go`; the corpus test lives beside `internal/driver/driver_test.go` and is skipped unless an explicit environment variable and the Gemini key are set.

### Phase 4: Continuous Play

**Goal**: The owner sets a session goal and the AI plays toward it continuously until stopped, within safety limits that are proven under test; re-engaging after manual driving starts from the current situation, not a stale plan.
**Depends on**: Phase 3
**Requirements**: REQ-continuous-loop, REQ-call-cap-and-error-disengage, REQ-reengage-reassess, REQ-doc-continuous-visible-play, REQ-doc-wheel-grab-and-reengage, REQ-safety-limits-hold
**Success Criteria** (what must be TRUE):

  1. The owner sets a session goal, and the loop runs read, decide, act continuously against a live game toward that goal, pacing itself to the game's turn rhythm rather than flooding it, with every decision visible with its reasoning as it happens.
  2. The session call cap halts the loop with a visible notice when reached; repeated errors or malformed model output disengage with a visible notice.
  3. Typing any game command instantly disengages autopilot and the command goes through; re-engagement after manual driving demonstrably reassesses the situation rather than resuming a stale plan.
  4. All mechanical safety limits hold under automated test: call cap when one is set, no AI-initiated reconnect, disengage on repeated errors, no commands issued while disconnected (autopilot waits and resumes only once the connection returns), and safe behaviour with blank settings (no cap, informative failure, no crash, regular play unaffected).

**Plans**: TBD
**UI hint**: yes

**Phase Validation** (how the success criteria are demonstrated): Diagnostic: `go test ./internal/...` passes a suite that exercises all four mechanical limits (call cap halt when a cap is set, error disengage with an informative notice, no commands while disconnected with resume on return, and blank-settings behaviour: no cap, no crash). Player-observable on staging: set a goal and watch the loop issue decisions paced to game output; set a small call cap and see the loop halt with the notice; take the wheel, re-engage, and confirm from the logged reasoning that the first new decision describes the current situation, not the previous plan.

Implementation notes for planning: the loop is a server-side goroutine per engaged session, cancellable by disengage; pacing waits for new game output or a floor interval rather than issuing on a fixed clock; the call counter and error counter are per AI session and compared against the resolved (default or profile) settings; re-engage constructs a fresh prompt from current game text and does not carry the previous loop's plan; Go tests cover the four limits (this repo has near-zero test coverage, so the test scaffolding is created here).

### Phase 5: Coaching Channel

**Goal**: The owner can coach the AI from a chat pane while it plays, see the guidance take effect on the next decision, pause and resume it, and make a piece of guidance permanent on the profile.
**Depends on**: Phase 4
**Requirements**: REQ-coaching-chat, REQ-pause-resume, REQ-promote-guidance, REQ-doc-coaching
**Success Criteria** (what must be TRUE):

  1. A chat message sent while the AI plays is reflected in its next decision, and this is verifiable from the logged reasoning.
  2. Pause and resume work from chat; pausing stops commands but keeps reading the game.
  3. The owner can promote a chat instruction into the profile's conduct rules or approach guidance without leaving the page, and it persists across sessions.
  4. The chat pane sits beside the terminal in the play screen and shows the AI's decisions, reasoning, notices, and the owner's messages in one stream.

**Plans**: TBD
**UI hint**: yes

**Phase Validation** (how the success criteria are demonstrated): Player-observable on staging: send a coaching message and read it reflected in the next decision's logged reasoning; pause from chat and confirm decisions keep logging reads with no commands sent; promote a message and see it appear under the profile's conduct rules or approach guidance, still present in a fresh session. Diagnostic: chat messages are stored against the AI session and can be queried alongside the decisions they influenced.

Implementation notes for planning: chat messages travel over the existing websocket (new message type) and are persisted with the AI session so they are auditable against decisions; the prompt assembler includes recent coaching messages; pause is a loop state distinct from disengaged (still reading, not acting); promotion calls the Phase 1 profile sub-resource.

### Phase 6: Measurement and Memory

**Goal**: The AI measures its own progress by the game's numbers, every session ends with a stored debrief, lessons carry into the next session, and the owner's manual play is captured as demonstration material.
**Depends on**: Phase 4 (and Phase 5 for the chat surface the debrief is shown in)
**Requirements**: REQ-progression-tally, REQ-session-debrief, REQ-memory-carryover
**Success Criteria** (what must be TRUE):

  1. During play, the driver samples the game's status commands (configured on the profile, not built into the engine) on an interval and stores the parsed numbers with timestamps against the profile and character.
  2. Ending a session produces a stored debrief: goal, outcome, tally movement, what was learned, what went wrong; new lessons are appended to the profile's learned notes.
  3. The next session's decisions include the learned notes, and manual-driving stretches are captured and available to the debrief as demonstrations.
  4. The debrief records how much coaching the session required, so the Phase 8 trend (tally improves while required coaching declines) is measurable from stored data.

**Plans**: TBD
**UI hint**: yes

**Phase Validation** (how the success criteria are demonstrated): Diagnostic: progression sample rows with timestamps accumulate during play; a debrief row exists after session end with goal, outcome, tally movement, lessons, problems, and coaching count; the next session's first prompt, as logged, contains the learned notes; a manual-driving stretch is stored with the human source flag. Player-observable: the debrief is shown in the chat pane at session end.

Implementation notes for planning: new tables for progression samples (profile, character, timestamp, parsed numbers) and debriefs; learned notes as a profile field; status-command list and sampling interval are AI settings on the profile; parsing of status output is model-driven or profile-configured, never engine game knowledge; manual-driving capture reuses the session log with a source flag from Phase 2.

### Phase 7: Study Loader

**Goal**: The AI can learn from material outside live play, and nothing enters learned notes without the owner's confirmation.
**Depends on**: Phase 6
**Requirements**: REQ-study-loader, REQ-study-confirmation
**Success Criteria** (what must be TRUE):

  1. The owner can paste text or upload a file (transcript, help file, walkthrough) against a profile; the AI summarizes its conclusions in the chat pane.
  2. Lessons the owner confirms are written into the learned notes; nothing is written without confirmation.

**Plans**: TBD
**UI hint**: yes

**Phase Validation** (how the success criteria are demonstrated): Player-observable on staging: upload a help file against a profile and read the AI's summary in the chat pane; learned notes are unchanged until the owner confirms; after confirmation only the confirmed lessons appear in the notes. Diagnostic: proposed lessons are held server-side and are absent from the profile until confirmed.

Implementation notes for planning: upload endpoint under the profile; the summarization call counts toward the owner's cost but is not a play session, so it does not touch the play-session call cap; proposed lessons are held server-side pending an explicit confirm action from the chat pane.

### Phase 8: Acceptance Run on Alter Aeon

**Goal**: The proof: run from Railway staging under the owner's supervision on Alter Aeon, the Definition of complete items 1 through 6 all hold, including measurable improvement across three consecutive goal sessions.
**Depends on**: Phase 7 (and therefore all earlier phases)
**Requirements**: REQ-acceptance-reconnaissance, REQ-acceptance-goal-sessions, REQ-acceptance-definition-holds, REQ-improvement-trend
**Success Criteria** (what must be TRUE):

  1. Reconnaissance first: on a fresh profile, with the owner coaching, the AI's opening sessions discover and record how Alter Aeon measures progress (its status commands and numbers) into notes and configuration.
  2. Then goal sessions: the owner hand-plays a character through creation and the mud school, engages autopilot outside restricted areas (entered in conduct rules per policy section 1), and the AI completes small session goals honestly measured by the tally.
  3. Every session ends with a recorded debrief and updated progression tally, and across at least three consecutive goal sessions the tally shows improvement while required coaching declines.
  4. The Definition of complete, items 1 through 6, all hold during these sessions, and the evidence (session logs, debriefs, tally samples, test results for item 6) is on record.

**Plans**: TBD

**Phase Validation** (how the success criteria are demonstrated): Evidence on record from staging: reconnaissance session notes and configuration for Alter Aeon's status commands; session logs, debriefs, and tally samples for at least three consecutive goal sessions showing the tally improving while coaching declines; the Phase 4 safety-limit test output for item 6; the owner's sign-off that Definition of complete items 1 through 6 held during the sessions.

Implementation notes for planning: this phase is mostly operation and evidence gathering, not new code; defects found feed decimal insertion phases; Alter Aeon's automation rules must be confirmed by the owner before engagement (policy section 1) and any conditions recorded in the profile's conduct rules.

## Progress

**Execution Order:**
Phases execute in numeric order: 1 -> 2 -> 3 -> 4 -> 5 -> 6 -> 7 -> 8

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Profile Foundation and Policy Gate | 6/6 | Complete    | 2026-09-15 |
| 2. Autopilot Switch | 7/7 | Complete    | 2026-09-15 |
| 3. One AI Decision | 13/13 | Complete    | 2026-09-16 |
| 4. Continuous Play | 0/TBD | Not started | - |
| 5. Coaching Channel | 0/TBD | Not started | - |
| 6. Measurement and Memory | 0/TBD | Not started | - |
| 7. Study Loader | 0/TBD | Not started | - |
| 8. Acceptance Run on Alter Aeon | 0/TBD | Not started | - |

## Requirement Coverage

28 v1 requirements, 28 mapped, 0 orphaned, no duplicates.

| Phase | Requirements |
|-------|--------------|
| 1 | REQ-profile-ai-fields, REQ-policy-gate |
| 2 | REQ-autopilot-directives, REQ-wheel-grab, REQ-no-auto-reconnect, REQ-doc-hand-play-and-gate |
| 3 | REQ-single-decision, REQ-reasoning-visibility, REQ-env-config |
| 4 | REQ-continuous-loop, REQ-call-cap-and-error-disengage, REQ-reengage-reassess, REQ-doc-continuous-visible-play, REQ-doc-wheel-grab-and-reengage, REQ-safety-limits-hold |
| 5 | REQ-coaching-chat, REQ-pause-resume, REQ-promote-guidance, REQ-doc-coaching |
| 6 | REQ-progression-tally, REQ-session-debrief, REQ-memory-carryover |
| 7 | REQ-study-loader, REQ-study-confirmation |
| 8 | REQ-acceptance-reconnaissance, REQ-acceptance-goal-sessions, REQ-acceptance-definition-holds, REQ-improvement-trend |
