# Phase 5: Coaching Channel - Research

**Researched:** 2026-09-17
**Domain:** Go backend extension of the Phase 4 continuous-play driver with a second, parallel model-call path (AI-chatter) plus a pause/resume state extension of the Phase 2 autopilot state machine (`internal/driver`, `internal/session`, `internal/gemini`, `internal/store`, `internal/icm`, `internal/config`, `internal/crypto`); TypeScript/React frontend extension of the AI Assist panel with a docked chat strip, a pop-out-window mechanism (`window.open` + `createPortal`, no new dependency), a Help page article, a Logs page section, and one new AI Settings field. Four carried-forward Phase 4 security defects are fixed in this phase's Go code. This is a brownfield codebase-investigation phase: no new library is required anywhere in the stack.
**Confidence:** HIGH for every code-level extension point (every claim below traces to a direct read this session of `driver.go` (1538 lines, full), `loop.go` (full), `memory.go` (full), `autopilot.go` (full), `manager.go` (full, 1340 lines), `websocket.go` (full, 818 lines), `handler.go` (relevant sections), `dispatcher.go` (full), `types.go` (SafetyLimits), `client.go` (gemini, full), `crypto.go` (full), `config.go` (full), `profile.go` (store, full), `AIAssistPanel.tsx` (full), `AIPlayerPanel.tsx` (settings section), `SessionContext.tsx` (full), `App.tsx` (routing/AuthGuard), `LogsPage.tsx` (header), `HelpPage.tsx` (header), `api.ts` (WebSocketManager, full), migrations 011/013/014, `RISK-REGISTER.md`, and a live `go build`/`go test ./internal/...` run on this machine). MEDIUM for the exact shape of AI-chatter's structured-output schema and for the pause/resume state-machine extension design (both are Claude's Discretion per CONTEXT.md; the recommendation below is a reasoned design proposal, not yet code-verified against a live Gemini call or a live pause/resume race). LOW for anything about React 18 `createPortal` + pop-out-window quirks in *this specific* app, since no live browser test was run this session — the design is standard React practice, cross-checked against the already-approved `05-UI-SPEC.md` code samples, not against a running instance.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**The two levels and the panel**
- **D-01:** Owner's words: "Chat between AI and User should sit at a meta level above the AI stream reading and sending commands into the game. however that chat should be able to influence the stream logic realtime in the game." The conversation is about the play, not part of it: its own view, its own stored history, the AI replies there. AI-player never waits on the conversation; it keeps its own pace (Phase 4 D-06).
- **D-02:** The names **AI-chatter** and **AI-player** are the owner's and are used in code comments, log lines, UI copy, the help article and planning documents.
- **D-03:** Chat lives **inside the existing AI Assist panel** (floating, 380 by 760, Phase 4 D-30), as a **split: AI-player's thinking stream on top, the AI-chatter conversation and message box below, both visible at once**. No tabs, no new pane, no docked column. The AI-player part keeps the goal box, status line, Session Memory section, and gains the Pause/Resume button (D-13) and the "Coaching in effect" list (D-09).
- **D-04:** The owner's messages and AI-chatter's replies appear **only in the conversation area**, never mixed into the thinking stream. This supersedes the roadmap's "one stream" wording (criterion 4), which is amended to "one panel, two views".
- **D-05:** When a coaching suggestion reaches AI-player, the **thinking stream prints a one-line marker** in the family of "Coaching received" at that moment. The message text stays in the chat. The marker, followed by the next decision's logged reasoning reflecting the coaching, is the evidence for success criterion 1.
- **D-06:** The **message box is usable whenever the panel shows**: autopilot ON, OFF or WAITING (paused or disconnected). A message sent while autopilot is off works the same: AI-chatter replies, and any coaching becomes a standing suggestion already in place at the next `#AUTO ON`.
- **D-07:** **Chat messages are plain text with no functionality.** Owner's words: "there's no functionality in the text... it's just text back and forth." No promote button, no per-message actions. (A browser's ordinary text selection and copy is all the owner needs.)
- **D-29 (added 2026-09-17):** Each of the two views can be **popped out into its own separate browser window**: one pop-out button on the AI-player view, one on the AI-chatter conversation. Whichever view is not popped out stays docked. A popped-out window is fed by the play screen's own live data (no second sign-in, no second game connection), so it lives only while the play screen tab is open; closing the pop-out returns that view to the panel. Everything that works docked works the same popped out. The docked conversation area stays a fixed height, not resizable.
- **Owner confirmations (2026-09-17):** the "Coaching received" marker does not repeat the coaching text; the Logs page shows the conversation as its own section below the transcript, not woven in; the Pause/Resume button is greyed out, not hidden, when autopilot is fully off.

**How coaching sticks**
- **D-08:** Each piece of coaching AI-chatter pushes down becomes a **standing suggestion that AI-player reads on every decision for the rest of the MUDPuppy login**, per connection profile: the same lifetime as Session Memory (Phase 4 D-31). A refresh, a game reconnect, `#AUTO OFF` then `#AUTO ON`, a wheel-grab or a redeploy keeps them; a new sign-in starts clean.
- **D-09:** **Nothing reaches AI-player that the owner cannot see.** When AI-chatter pushes a suggestion down, its reply quotes the exact line sent ("Sent to AI-player: avoid the north road"). AI-player's part of the panel shows a **read-only, collapsible "Coaching in effect" list** beside Session Memory, updated live and reloaded after a refresh.
- **D-10:** The owner **takes a suggestion back by telling AI-chatter in words**; AI-chatter withdraws that line and confirms which one it removed. No clear button.
- **D-11:** **AI-chatter always gives a short reply** to every message: what it understood and what AI-player will now do differently, or the answer to a question. Each reply is one model call and **counts against the session call cap** (Phase 4 D-14). While autopilot is off no stint is running, so replies are held to the same cap number on their own count.
- **D-12:** **Profile rules always hold in the game.** Coaching never overrides the conduct rules, the Never-issue list or the safety checker. When coaching conflicts with a rule, AI-chatter does not flatly refuse: it names which rule or filter is in the way and suggests the wording change; the owner makes the change himself on the Settings page.

**Pause and resume**
- **D-13:** Pause/Resume is **one button in the AI-player part of the panel**. It is instant and makes no model call. Nothing in chat pauses or resumes. No `#AUTO PAUSE` directive.
- **D-14:** **While paused AI-player reads only.** Game text keeps flowing into the Immediate Context window; no decisions, no model calls, no commands. The **first decision on resume reassesses the situation** exactly like a re-engage (Phase 4 D-08). AI-chatter keeps working while paused.
- **D-15:** **Pause shows as WAITING, with a reason.** The badge stays three states. The server tracks why it is waiting: paused by the owner, connection lost, or both. It resumes **only when every reason is cleared**: a pause never resumes by itself, and a reconnect alone does not un-pause. The status line gives the reason; the thinking stream prints a notice on pause and on resume. `#AUTO OFF` turns autopilot off from waiting as today. A page refresh keeps it paused.
- **D-16:** **Typing anything disengages autopilot, paused or not.** The wheel-grab rule is the same in every engaged state and lands OFF. Text typed into the chat message box is not a game command and is not a wheel-grab.

**What AI-chatter can see**
- **D-17:** AI-chatter reads **everything AI-player knows, read-only**: session goal, conduct rules, approach guidance, Never-issue list, AI-player's recent decisions and reasoning (including blocks and failures), recent game text (Immediate Context), Session Memory, the active Quest's memory, and coaching currently in effect. This delivers Quest Memory recall through chat (Phase 4 D-11 parked this).
- **D-18:** AI-chatter **changes nothing** except its own list of coaching suggestions. No write path to the profile, the goal, the memory layers, the autopilot state or the game.
- **D-19:** **Editing Session Memory by hand is dropped.** Session Memory stays read-only. A wrong note is corrected via chat coaching.

**No promotion; help article; document amendments**
- **D-20:** **REQ-promote-guidance as originally written is withdrawn.** No conduct-rules/approach-guidance editor is added to the play screen; Settings page is where the owner edits them by hand.
- **D-21:** A **help article is added to the Help page** (`frontend/src/pages/HelpPage.tsx`): what AI-chatter/AI-player are, how a chat message reaches the player, how long coaching lasts and how to take it back, that chat can never change settings, how Pause/Resume works, and how to copy a suggestion to Settings by hand.
- **D-22:** Design doc, REQUIREMENTS.md, ROADMAP.md, PROJECT.md amended (already reflected in the copies read for this research).

**Security carry-forward (owner: "For any High level items, they need to be fixed in this Phase.")**
- **D-23:** **DR-4-01 (high) fixed here.** The checker is told plainly that fighting creatures/objects the game presents as targets is normal play and only attacking another player is off limits; the live false blocks `c chill touch golem` and `c static blast crystal` join the corpus as benign items; corpus rerun must hold STEERED 0 with those items not blocked; sample the new fight items more than once (no single-sample coin-flip, Phase 4 D-29).
- **D-24:** **DR-4-02 (high) fixed here.** A checker answer without its `blocked` yes-or-no is a failed review (already stops the command); the reviewer prompt gains its own sentence saying the memory notes were written by the other model. A test for each.
- **D-25:** **DR-4-03 (high) fixed here, as a configurable setting.** Every AI-player command is recorded with the rate limiter before it is sent and refused when the limiter says stop; a flood test proves it. The limit is an AI setting on the profile; blank means server default. `internal/icm/dispatcher.go` returns for a plain pass-through command before `RecordExecution`, and the circuit-breaker counter is never reset — both must be handled, not blindly routed through.
- **D-26:** **DR-4-04 (low) fixed here.** Require the vault key everywhere unless a setting says this is local development.
- **D-27:** **Log-page sign-in check fixed here.** Confirm `/logs/:connectionId` sends a signed-out visitor to sign-in, screenshot from a private window/separate profile; add the guard and a test if missing; record against AR-4-08 at the Phase 5 review.
- **D-28:** All four DR-4 rows re-presented at the Phase 5 security review with outcomes recorded in `.planning/RISK-REGISTER.md`. **New risk to carry:** AI-chatter reads untrusted game text and model-written memory (D-17) then writes lines AI-player reads as coaching (D-08) — a possible laundering channel. See Claude's Discretion below.

### Claude's Discretion
- The AI-chatter injection channel design: untrusted delimiting exactly like AI-player's/the reviewer's prompts; a pushed coaching line must come from what the owner said, not from untrusted blocks; coaching sits in its own marked section in AI-player's prompt, below standing text and goal, subordinate to conduct rules/Never-issue; reviewer sees it too; red-team corpus gains AI-chatter-channel attack items; size ceilings on suggestions, enforced in Go.
- How AI-chatter returns both a reply and zero-or-more push/withdraw actions in one structured model call; which configured model it uses (same env-configured registry).
- What happens to chat when the cap is reached: locked notice, no reply sent, owner's message still stored. Off-stint count reset timing (per login is fine).
- Storage: chat/replies against the AI session, auditable beside decisions; coaching suggestions per login and profile like Session Memory; migration numbers (015+). Whether the log page shows the conversation (default: yes, own section).
- Chat text is captured owner text plus model text, not game text; whether nightly retention touches it (default: keep with decision rows' kept fields).
- The websocket message type/fields for chat both directions; REST routes to reload conversation and "Coaching in effect" on attach; the pause/resume endpoint or message.
- How waiting reasons are held in the session manager; `[AI-PLAYER]` log lines for pause/resume/chat/coaching push/withdraw (ids, stages, outcomes, lengths only); an `[AI-CHATTER]` tag if it reads better.
- Split proportions/divider/message styling/button placement/notice wording, in `03-UI-SPEC.md`'s Copywriting Contract voice — **superseded for this phase by the already-approved `05-UI-SPEC.md`, which locks all of this**; see canonical ref below.
- The rate-limit setting's unit/default/bounds; where it sits among AI settings; how the dispatcher's counter reset and pass-through recording are fixed.
- Evidence for criterion 2: `[AI-PLAYER]` log excerpt showing the window still fed while paused with no model calls/sends, a canned report, and screenshots; never a database query.

### Deferred Ideas (OUT OF SCOPE)
- Promotion of chat text into the profile — withdrawn, not to be re-proposed.
- Editing Session Memory by hand — dropped.
- Conduct rules/approach guidance editable on the play screen — declined in favor of Settings + help article.
- A copy icon on chat messages, a Clear button on the coaching list, `#AUTO PAUSE`, typed "pause" in chat — all declined.
- Phase 6: debrief must record coaching amount per session. Phase 7: study loader's summary appears in AI-chatter's conversation area.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| REQ-coaching-chat | A chat message sent while the AI plays is reflected in its next decision, verifiable from logged reasoning | Architecture Pattern 1 (AI-chatter call path and coaching-in-effect prompt section); Code Examples (schema/prompt wiring); Don't Hand-Roll (structured push/withdraw answer) |
| REQ-pause-resume | Pause/resume from AI-player part of panel; pausing stops commands but keeps reading; shows WAITING with reason; never resumes by itself | Architecture Pattern 2 (waiting-reason state machine extension); Common Pitfall 1 (today's `EnterWaiting`/`Resume` transitions are too coarse for two independent reasons) |
| REQ-promote-guidance | Chat is plain text only, no promotion; help article explains applying a suggestion by hand | Architecture: Help page `SECTION_ORDER` extension (already locked in `05-UI-SPEC.md` §5); no backend work |
| REQ-doc-coaching | Cross-cutting Definition of Complete item 4 | Covered by the above three plus the Logs page conversation section (Architecture: WS/REST surface) |

</phase_requirements>

## Summary

Phase 5 adds a second, parallel model-call path (AI-chatter) beside the existing AI-player loop, without touching AI-player's per-iteration shape (`runIteration` in `driver.go`) except for one new prompt section (coaching-in-effect) and one new field the loop must check before every decision. AI-chatter is architecturally a **sibling** of the existing player/reviewer calls, not a wrapper around them: it is triggered by an inbound websocket chat message (never by the loop's own pacing), makes exactly one Gemini call per owner message using the same hand-written `internal/gemini` client and the same `propertyOrdering`-constrained-JSON mechanism already proven for `reasoning`/`command` and `reason`/`blocked`, and its output is a `{reply, push, withdraw}` triple that a new, small Go collaborator (a "coaching store", following the exact login-scoped inheritance pattern Phase 4 D-31 already built for Session Memory on `game_sessions.session_memory`) persists. AI-player's `promptContext` (already extended once for Session/Quest Memory in `memory.go`) gains one more field — `Coaching []string` — read the same nil-safe, clamped, untrusted-delimited way Session Memory is today, with one difference the reviewer's own untrusted-paragraph precedent does not fully cover: coaching is delimited as untrusted **content shape** (a bullet list a hostile game-text-reading process wrote) but must be framed to the model as **the owner's live guidance**, subordinate to conduct rules, not as arbitrary data to ignore — this is Phase 5's own prompt-injection design surface (D-28's new risk) and is spelled out in Architecture Pattern 1 below.

Pause/Resume is the phase's most delicate pure-logic problem: today's `session.AutopilotRecord` has exactly one `State` (`off`/`on`/`waiting`) and a single `WaitingSince` timestamp, and `EnterWaiting`/`Resume` are total pure-function transitions with no memory of *why* a record is waiting. D-15 requires two independent reasons (owner-paused, connection-lost) that must **both** clear before an On resume fires, and a reconnect must not silently cancel a pause the owner did not lift (or vice versa). This cannot be built by re-using `EnterWaiting`/`Resume` as-is; it requires two new boolean fields on `AutopilotRecord` and small new pure-transition-adjacent methods on `Manager` that update one reason bit at a time and only fire the engage hook when both are simultaneously clear. This is a genuinely new piece of state machine, not a bigger version of an existing one, and should be its own Wave.

Four Phase 4 security carry-forwards are concrete, scoped fixes with exact code locations already identified: DR-4-02 (missing `blocked` field silently sends) is a one-field type change (`bool` → `*bool`) in `gemini.ReviewAnswer` plus one call site in `driver.go`; DR-4-04 (vault key check) is a one-line inversion of `config.go`'s `RAILWAY_ENVIRONMENT` check to an explicit opt-out flag; DR-4-01 (fighting false-blocks) is a one-sentence addition to `reviewHarmDefinition`'s ordinary-guidance exception plus two new corpus items; DR-4-03 (AI commands bypass the ICM rate limiter) is the most involved of the four — `icm.Dispatcher.Dispatch` returns before `RecordExecution` for any command with no registered handler (which is every plain MUD command an AI sends), and `DefaultSafetyChecker.RecordExecution`'s `loopCounts` map is incremented but never reset by anything, so routing AI sends through the existing `CheckCircuitBreaker`/`RecordExecution` pair as-is would eventually refuse every AI command in a long-running process. The correct fix is a **dedicated, per-profile rate check** called explicitly by the driver immediately before `Dispatch`, independent of the dispatcher's internal (and differently-scoped) circuit breaker.

The frontend work (panel split, pop-out windows, Help/Logs pages, AI Settings field) is **fully specified** by the already-approved `05-UI-SPEC.md` — copy its class names, copy strings, and code samples verbatim; this research does not re-derive UI decisions, only flags the two places (pop-out lifecycle testing, and the `/logs/:connectionId` auth guard) where the UI spec's assumptions should be verified against what the code actually does.

**Primary recommendation:** Build AI-chatter as a new `Driver.HandleChat(userID, connectionID, message string)` method — a same-package sibling of `runIteration`, not a call into it — using a third `Models` interface method (`Chat`) with its own `propertyOrdering: [reply, push, withdraw]` schema; extend `promptContext` with `Coaching []string` read the same clamped/nil-safe way as `QuestBullets`/`SessionMemory`; add two boolean reason fields (`PausedByOwner`, `ConnectionLost`) to `session.AutopilotRecord` with two new `Manager` methods (`PauseAutopilot`, `ResumeAutopilot` distinct from the existing reconnect-driven `resumeAutopilotLocked`) that only fire the engage hook when both reasons are false; fix DR-4-03 with a small new `Driver`-owned token-bucket check (reusing `session.RateLimiter`, already written for the websocket ingress case) called explicitly before every `Dispatch`, keyed by user, with its limit resolved from a new `AISettings.RateLimitPerSecond *int` field.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| AI-chatter model call & reply | API/Backend (Go, `internal/driver`, new `HandleChat`) | — | Same trust boundary as AI-player's own calls; must never touch the game |
| Coaching suggestion storage & lifetime | API/Backend + Database | Browser (read-only "Coaching in effect" list) | Server owns truth; mirrors Session Memory's existing login-scoped pattern (D-31) exactly |
| Coaching injection into AI-player's prompt | API/Backend (Go, `driver.buildSystemInstruction`/`buildReviewSystemInstruction`) | — | Must be delimited and framed identically in both prompts, per the existing single-source-of-truth pattern (`untrustedDataParagraph`) |
| Pause/Resume control & waiting-reason state | API/Backend (Go, `internal/session`) | Browser (button, status-line reason text) | Server owns the state machine; browser reflects and calls one endpoint |
| Chat message transport | API/Backend (Go, `internal/session/websocket.go`, new message type) | Browser (chat strip / pop-out window) | New message type must bypass the wheel-grab path entirely (D-16) |
| Chat/coaching persistence | Database | API/Backend | New migration (015+); auditable beside decisions per D-08/discretion |
| Rate-limit-per-second AI setting | API/Backend (Go, `store.AISettings` + `internal/icm`) | Browser (Settings field) | Blank-means-default pattern already established for Call Cap/Disengage Threshold |
| Pop-out windows | Browser (React, `window.open` + `createPortal`) | — | No backend involvement at all — same live data, same websocket, different document |
| Help article content | API/Backend (Go, `internal/help`, JSON file) | Browser (renderer, unchanged) | Matches every existing Help section's server-authored-JSON pattern |
| Vault key / rate-limiter security fixes | API/Backend (Go, `internal/config`, `internal/icm`) | — | Pure backend hardening, no browser surface |

## Standard Stack

No new libraries anywhere in the stack. Backend: Go standard library (`sync`, `time`, `context`) plus the project's own existing `internal/gemini` hand-written client, extended with a third method. Frontend: React 18.3.1 (`react-dom`'s `createPortal`, already a transitive dependency of `react-dom@^18.3.1` — no new `package.json` entry) plus the browser's native `window.open`. This mirrors every prior phase's "no new go.mod/package.json entry" pattern.

**Version verification:**
```bash
$ cd /d/Projects/MUDPuppy && go version
go version go1.26.0 windows/amd64
$ node -e "console.log(require('./frontend/node_modules/react-dom/package.json').version)"
```
`frontend/package.json` pins `react: ^18.3.1`, `react-dom: ^18.3.1` `[VERIFIED: read frontend/package.json this session]`. `createPortal` has shipped in `react-dom` since React 16 and is unchanged in its React 18 signature (`createPortal(children, domNode, key?)`) `[CITED: react.dev/reference/react-dom/createPortal]` — this is training-data knowledge of a long-stable API, not independently re-verified against react.dev this session; flagged in the Assumptions Log.

### Package Legitimacy Audit

**Not applicable — no external packages are installed in this phase.** No `go.mod` or `frontend/package.json` change is anticipated; both `Chat` (gemini client method) and the pop-out mechanism (`window.open` + `createPortal`) are additions to files already in the tree, using APIs already present in already-installed dependencies. The plan-checker should verify this holds (empty dependency-drift section in the test report, matching every prior phase's AR-*-SC row).

## Architecture Patterns

### System Architecture Diagram

```
                    ┌───────────────────────────────────────────────────┐
                    │  Owner's browser — AI Assist panel (docked or      │
                    │  popped-out into a child window via createPortal)  │
                    │  ┌─────────────────┐   ┌─────────────────────────┐│
                    │  │ AI-player view  │   │ AI-chatter view          ││
                    │  │ (thinking       │   │ (chat log + input row)   ││
                    │  │  stream, goal,  │   │ types a message ─────────┼┼──┐
                    │  │  Pause/Resume,  │   │ sees replies, markers    ││  │
                    │  │  Coaching list) │   │                          ││  │
                    │  └────────┬────────┘   └──────────────────────────┘│  │
                    └───────────┼──────────────────────────────────────┘  │
                                │ existing onAI/offAI (ai message)         │ NEW: onChat/offChat
                                │                                          │ (chat message type,
                    ┌───────────▼──────────────────────────────────────┐  │  never wheel-grab)
                    │  internal/session.WebSocketHandler                │◄─┘
                    │  MsgTypeAI (unchanged) │ MsgTypeChat (NEW)         │
                    │  MsgTypeAutopilot (extends: waiting-reason data)  │
                    └───────┬─────────────────────┬────────────────────┘
                            │ existing hooks       │ NEW: chat inbound
              ┌─────────────▼───────────┐   ┌──────▼─────────────────────┐
              │ internal/session.Manager │   │ internal/driver.Driver     │
              │ AutopilotRecord gains:   │   │  runIteration (unchanged   │
              │  PausedByOwner bool      │   │   shape, +Coaching field   │
              │  ConnectionLost bool     │   │   in promptContext)        │
              │  PauseAutopilot()  ──────┼──►│  HandleChat (NEW, sibling  │
              │  ResumeAutopilot() ──────┼──►│   of runIteration, own     │
              │  (only fires EngageHook  │   │   in-flight guard, own     │
              │   when BOTH reasons      │   │   cap reservation)         │
              │   are false)             │   └──────┬─────────────────────┘
              └──────────────────────────┘          │ Models.Chat (NEW 3rd method)
                                                ┌─────▼──────────┐
                                                │ internal/gemini │
                                                │ {reply, push,   │
                                                │  withdraw}      │
                                                │ schema          │
                                                └────────┬────────┘
                                                         │ persists via new
                                                ┌────────▼────────────────┐
                                                │ internal/store           │
                                                │ CoachingStore (NEW,       │
                                                │ login-scoped like         │
                                                │ Session Memory D-31)      │
                                                │ ConversationStore (NEW,   │
                                                │ against game_session_id)  │
                                                └───────────────────────────┘
                            AI-player's next runIteration reads Coaching
                            via the same clampBullets/untrusted-delimit path
                            as Session/Quest Memory ───────────────────────►
```

### Recommended Project Structure

```
internal/driver/
├── driver.go          # promptContext gains Coaching []string; buildSystemInstruction/
│                       #  buildReviewSystemInstruction gain a coaching section; ReviewAnswer
│                       #  decode fix (DR-4-02); reviewHarmDefinition gains DR-4-01 sentence;
│                       #  Dispatch call site gains the new rate-limit check (DR-4-03)
├── chat.go            # NEW: HandleChat, its own in-flight guard, cap reservation, the
│                       #  {reply, push, withdraw} answer -> coaching store write, and the
│                       #  AI-chatter system-instruction builder (separate from buildSystemInstruction)
├── loop.go            # unchanged in shape; runLoop already reads AutopilotStateFor before
│                       #  each tick, which a WAITING (paused) state already stops
├── memory.go          # coachingBlock()/wrapCoaching() mirroring wrapSessionMemory exactly;
│                       #  size ceiling maxCoachingBullets alongside maxQuestBullets
├── ratelimit.go        # NEW (DR-4-03): a small per-user token-bucket check the driver calls
│                       #  explicitly before Dispatch, reusing session.RateLimiter's shape
├── driver_test.go / chat_test.go / ratelimit_test.go

internal/session/
├── autopilot.go       # AutopilotRecord gains PausedByOwner, ConnectionLost bool fields;
│                       #  new Pause/lift-reason pure-transition-adjacent helpers
├── manager.go          # PauseAutopilot(userID), ResumeAutopilot(userID) — distinct from the
│                       #  existing reconnect-driven resumeAutopilotLocked; parkAutopilotLocked
│                       #  sets ConnectionLost=true instead of assuming EnterWaiting's binary state
├── websocket.go        # MsgTypeChat constant; WSMessage gains a Chat payload field; the main
│                       #  read-loop switch gains a `case MsgTypeChat` that never calls
│                       #  applyWheelGrab (D-16); PushChat mirrors PushAI exactly
├── handler.go           # Autopilot handler gains "pause"/"resume" as valid actions;
│                        #  AutopilotResponse/StatusResponse gain waiting-reason fields

internal/store/
├── profile.go           # AISettings gains RateLimitPerSecond *int
├── coaching.go          # NEW: CoachingStore — CoachingFor(connectionID) / UpdateCoaching(...),
│                         #  same login-scoped SQL shape as transcripts.go's
│                         #  openGameSessionSQL/SessionMemoryForConnection
├── conversation.go       # NEW: ConversationStore — AppendChatLine(gameSessionID, ...) /
│                         #  ConversationFor(connectionID) for the Logs page section

internal/icm/
├── dispatcher.go         # no change required if ratelimit.go lives in internal/driver (see
│                         #  Architecture Pattern 3) — the DR-4-03 fix does not have to touch
│                         #  this file at all

internal/config/
├── config.go             # RAILWAY_ENVIRONMENT gate inverted to an explicit local-dev opt-out

internal/gemini/
├── client.go             # ReviewAnswer.Blocked -> *bool; new Chat() method + ChatAnswer type
│                          #  + its own responseSchema

migrations/
└── 015_add_coaching_and_conversation.up.sql / .down.sql

frontend/src/
├── components/AIAssistPanel.tsx   # per 05-UI-SPEC.md §1-3, §7 verbatim
├── components/AIPlayerPanel.tsx    # per 05-UI-SPEC.md §4 verbatim
├── services/popout.ts              # NEW, per 05-UI-SPEC.md §7 verbatim
├── services/api.ts                 # WebSocketManager gains sendChat/onChat/offChat, mirroring
│                                    #  sendCommand/onAI/offAI exactly; MsgTypeChat handling in
│                                    #  handleMessage's switch
├── pages/HelpPage.tsx               # SECTION_ORDER gains 'ai-coaching' per 05-UI-SPEC.md §5
├── pages/LogsPage.tsx                # conversation section per 05-UI-SPEC.md §6
```

### Pattern 1: AI-chatter is a sibling call path, not a wrapper — `HandleChat` never touches `runIteration`

**What:** AI-player's `HandleEngage`/`EngageLoop`/`runIteration` machinery (Phase 4) is a paced, cancellable, one-decision-at-a-time loop keyed by stint epoch. AI-chatter is fundamentally different: it fires once per owner-sent message, is not paced, has no stint epoch of its own, and must work even when autopilot is fully OFF (D-06). Building it as a variant of `runIteration` (adding an `isChat bool` branch, say) would entangle two independent concerns and risk AI-chatter's call being silently dropped by `staleStage`'s epoch check, which has no meaning for a message sent while autopilot is off.

**Recommendation:** A new `Driver.HandleChat(userID, connectionID, message string)` method, called directly from the new `case MsgTypeChat` in `websocket.go` (via a hook exactly like `EngageHook`/`DisengageHook` — e.g. `type ChatHook func(userID, connectionID, message string)`), with:
- its own in-flight guard (a small `chatInFlight map[string]bool` under the same `d.mu`, since one owner rarely sends two messages before the first reply lands, but nothing should assume that);
- its own cap reservation via the *same* `tryReserveCall` helper Pattern 3 of the Phase 4 research already established (D-11: "counts against the session call cap"), so a chat reply and a decision call share one pool per the resolved `AISettings.CallCap`;
- when the cap is exhausted, D-11/discretion says: no model call, a locked notice appended to the conversation, the owner's message still stored — this is a new terminal outcome, not routed through `recordFailure` (which disengages autopilot; AI-chatter's own cap-reached must never touch the autopilot switch, per D-18 "changes nothing except its own coaching list").

**Example (illustrative, not exact code):**
```go
// internal/driver/chat.go
func (d *Driver) HandleChat(userID, connectionID, message string) {
    if !d.beginChat(userID) {
        return // an earlier message from this user is still in flight
    }
    defer d.endChat(userID)

    profile, err := d.profiles.GetProfileByConnection(...)
    // ... same defensive checks as decide(), but no epoch, no staleStage ...

    resolved := store.ResolveAISettings(profile.AISettings, defaultModelName)
    if !d.tryReserveCall(userID, resolved) {
        d.notifyChatCapReached(userID, message) // stores owner's message, locked notice, no model call
        return
    }

    ctx := chatPromptContext{ /* everything promptContext has, PLUS RecentDecisions, Coaching */ }
    systemInstruction := buildChatSystemInstruction(ctx)
    answer, err := d.models.Chat(context.Background(), entry.Endpoint, entry.ModelName, entry.APIKey, systemInstruction, message)
    // ... store owner message + reply, apply answer.Push/answer.Withdraw to the coaching store,
    //     notify the browser over MsgTypeChat, and — if any push/withdraw happened — notify
    //     a "[Coaching received]" system line over the EXISTING MsgTypeAI channel so it lands
    //     in the thinking stream (D-05), not the chat channel.
}
```

**Anti-pattern to avoid:** Do not give `HandleChat` a stint epoch or route it through `staleStage`. AI-chatter has no "stint" — it must work identically whether autopilot is ON, OFF, or WAITING (D-06), which the epoch/stale-stage machinery is specifically designed to *not* allow for AI-player.

### Pattern 2: Two independent waiting reasons require new state, not a bigger `EnterWaiting`/`Resume`

**What:** `internal/session/autopilot.go`'s four pure functions (`Engage`, `Disengage`, `EnterWaiting`, `Resume`) are total, single-reason transitions: `AutopilotRecord` has one `State` and one `WaitingSince *time.Time`. D-15 needs the record to independently track "the owner clicked Pause" and "the connection dropped," with the rule that an On resume fires only when **both** are false, and clearing one reason while the other is still true must leave the state at WAITING with no engage hook fired. Neither `EnterWaiting` (On→Waiting, no memory of cause) nor `Resume` (Waiting→On, unconditional) can express "stay Waiting because one of two reasons is still active" — extending them to accept a reason parameter and still return a clean `(AutopilotState, bool)` pair loses exactly the information the caller needs (which reason(s), if any, remain).

**Recommendation:** Add two fields to `AutopilotRecord`:
```go
// internal/session/autopilot.go
type AutopilotRecord struct {
    State AutopilotState
    ConnectionID string
    WaitingSince *time.Time
    Epoch uint64
    // PausedByOwner and ConnectionLost are D-15's two independent waiting
    // reasons. Both false is not a valid combination while State is
    // AutopilotWaiting (parkAutopilotLocked/PauseAutopilot always set at
    // least one true on the way in); Resume only fires the engage hook
    // when a transition leaves BOTH false.
    PausedByOwner  bool
    ConnectionLost bool
}
```
And two new `Manager` methods, structurally parallel to `DisengageAutopilot`/`parkAutopilotLocked` (same lock discipline, same `disengageHook`/`engageHook` firing convention, same `logAutopilotTransition` line format extended with the reason):
- `PauseAutopilot(userID string) (AutopilotState, bool)` — On→Waiting, sets `PausedByOwner=true`; fires `disengageHook` (stops the loop, exactly like a wheel-grab, per D-14 "no model calls, no commands"). No-op (defensive) if already Off (D-13's button is disabled then anyway) or if already Waiting (setting `PausedByOwner=true` on an already-parked record is still meaningful — a disconnect happened first, the owner then also pauses — but does not change `State` or fire any hook, since the loop is already stopped).
- `ResumeAutopilotByOwner(userID string) (AutopilotState, bool)` — clears `PausedByOwner=false`; if `ConnectionLost` is also false, transitions Waiting→On (bumps `Epoch`, fires `engageHook`, exactly like today's `resumeAutopilotLocked`'s changed-true path); otherwise stays Waiting, no hook fires, and the status line now shows only "Connection lost".
- `parkAutopilotLocked` (existing, disconnect path) changes to set `ConnectionLost=true` on **every** call where the record exists and is not Off — not only when `EnterWaiting`'s pure transition reports `changed=true` — so a disconnect that happens while already paused-by-owner still records the reason, even though `State` was already Waiting.
- `resumeAutopilotLocked` (existing, reconnect path) changes to clear `ConnectionLost=false` and only transition to On (fire `engageHook`) when `PausedByOwner` is also false; the existing "different profile → force Off" branch is unchanged.

**When to use:** This is the correct shape specifically because D-15 requires *set semantics* ("every reason is cleared") not a *stack* or a *priority order* — the two booleans are the simplest structure that can never represent an invalid state (there is no ordering dependency between "the owner pauses while disconnected" and "the connection drops while paused").

**Anti-pattern to avoid:** Do not model this as a single `WaitingReason string` enum with values like `"paused"`, `"disconnected"`, `"both"` — that reintroduces exactly the illegal-state and races Phase 2's original design avoided by keeping `AutopilotState` a plain three-value type; two independent booleans compose correctly under concurrent pause+disconnect in a way a tri-state-plus string cannot without extra branching at every call site.

### Pattern 3: The DR-4-03 rate-limit fix belongs in the driver, not in `icm.Dispatcher`

**What:** `internal/icm/dispatcher.go`'s `Dispatch` method calls `checkSafety` (which calls `CheckCircuitBreaker`/`CheckRateLimit`/`CheckQueueDepth` against `DefaultSafetyChecker`'s single, server-wide `maxExecutions`/`loopCounts` state) **before** looking up a handler, but only calls `RecordExecution` **after** a handler ran successfully. Every command an AI-player sends is a plain MUD command string with no registered `CommandHandler` (`d.getHandler` returns `nil` for it), so `Dispatch` takes the `handler == nil` branch and returns `(nil, nil)` **before** `RecordExecution` is ever reached (confirmed by direct read this session — `dispatcher.go` lines 208-224). This means AI commands are checked against `CheckCircuitBreaker`/`CheckRateLimit` (using whatever count happens to be in the shared map from any *other* traffic) but never themselves recorded — the limiter is not "silently permissive," it is structurally blind to the one traffic class DR-4-03 is about.

Separately, `DefaultSafetyChecker.loopCounts[sessionID]` (used by `CheckCircuitBreaker`) is incremented in `RecordExecution` and **never decremented or reset anywhere in this codebase** (confirmed: no `loopCounts` write outside `RecordExecution`'s `d.loopCounts[sessionID]++`). If AI sends were simply made to also call `RecordExecution`, every AI session would permanently trip the circuit breaker after `MaxLoopCount` (5) total sends, ever, for the lifetime of the process — exactly the todo's own warning.

**Recommendation:** Do not route AI sends through `Dispatch`'s existing `checkSafety`/`RecordExecution` pair at all. Add a small, dedicated, per-user rate check owned by `internal/driver` (not `internal/icm`), called explicitly immediately before the existing `d.commands.Dispatch(...)` call in `runIteration`:
```go
// internal/driver/ratelimit.go — reuses session.RateLimiter's exact token-bucket
// shape (already written and tested for websocket ingress in
// internal/session/websocket.go), rather than inventing a second algorithm.
type aiRateLimiter struct {
    mu       sync.Mutex
    limiters map[string]*session.RateLimiter // userID -> limiter, lazily created
}

func (d *Driver) allowAISend(userID string, resolved store.ResolvedAISettings) bool {
    if !resolved.RateLimitSet {
        return true // blank means server default; server default itself is a real limit (see below), never "unlimited"
    }
    // ... lazily create/reuse a session.RateLimiter(resolved.RateLimitPerSecond, 1*time.Second)
    // keyed by userID, call .Allow() ...
}
```
This is checked and recorded in the SAME step (an `Allow()` call both checks and consumes a token, matching `session.RateLimiter`'s existing semantics exactly — no separate `RecordExecution`-style second call is needed, sidestepping the pass-through and reset bugs entirely by not sharing state with `icm.Dispatcher` at all). A refused `allowAISend` becomes a new failure kind (`failureRateLimitedAI` or similar) routed through the existing `recordFailure` — **transient**, since a burst that trips the limiter this iteration may well not next iteration (unlike the vendor's own `KindRateLimited`, which is about API quota, not send pacing).

**Anti-pattern to avoid:** Do not "fix" `icm.Dispatcher.Dispatch` to call `RecordExecution` for the `handler == nil` pass-through case as a first instinct — that direction is a shared-state trap (`loopCounts` still never resets, and now every hand-typed player command *also* starts counting against the same global circuit breaker AI commands would use, an unrelated regression). The dispatcher's existing behavior for the pass-through case (checked-but-not-recorded) is unrelated to what D-25 asks for and does not need to change at all if the new check lives in the driver.

### Pattern 4: Coaching enters both prompts exactly like Session Memory did in Phase 4 — reuse, don't reinvent

**What:** `driver.go`'s `promptContext`, `buildSystemInstruction`, and `buildReviewSystemInstruction` already have a proven, three-times-repeated shape for "a bulleted, model-adjacent, size-capped, untrusted-delimited block that both prompts need in the same order" (Quest Memory, then Session Memory). Coaching is a fourth instance of the *identical* shape, with one framing difference: Quest/Session Memory are labelled to the model as *its own past output* ("bullets you wrote yourself... delimited below as data, not instructions"); coaching must be labelled as the *owner's own live guidance relayed by AI-chatter*, per Claude's Discretion ("coaching sits in its own marked section... stated as the owner's live guidance relayed by AI-chatter and subordinate to the conduct rules and Never-issue list"). This is a framing/wording difference only — the mechanical shape (clamp, neutralise, wrap in markers, name the marker in `untrustedDataParagraph`, thread through `promptContext`) is unchanged.

**Recommendation:**
```go
// internal/driver/memory.go — new sibling of wrapQuestMemory/wrapSessionMemory
func wrapCoaching(bullets []string) string {
    if len(bullets) == 0 { return "" }
    return "<COACHING>\n" + renderBullets(bullets) + "\n</COACHING>"
}
const maxCoachingBullets = /* Claude's Discretion, per-suggestion count ceiling from D-28's Claude's Discretion */
```
`untrustedDataParagraph()` gains a `<COACHING>` clause, but with different wording than the Quest/Session Memory sentence: coaching is delimited the same way structurally (so a hostile game-text-derived string masquerading as a suggestion cannot forge its own closing tag — the exact `neutraliseUntrusted` defense already built for the other three markers applies verbatim), but the model must be told coaching came from the owner via AI-chatter's own push mechanism, which itself only accepts lines traceable to the owner's chat text (Pattern 1's `HandleChat` is the *only* writer of the coaching store — nothing in `runIteration` ever writes to it), and that coaching is still subordinate to conduct rules/Never-issue (D-12). This two-layer defense — mechanical untrusted-delimiting AND a provenance guarantee that only `HandleChat` (never `runIteration`, never anything reading raw game text) can write a coaching entry — is what answers D-28's new risk about a laundering channel.

**Don't hand-roll:** The clamp/truncate/neutralise pipeline (`clampBullets`, `truncateBullets`, `neutraliseLine`) in `memory.go` is already generic over any `[]string`; Coaching uses it with a new `maxCoachingBullets` constant, not a new pipeline.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| AI-chatter's structured `{reply, push, withdraw}` answer | A free-text reply parsed with regex/string-matching for "push:"/"withdraw:" markers | A `responseSchema` with `properties: {reply, push: {type:array,items:string}, withdraw: {type:array,items:string}}`, exactly like `GenerateContent`'s existing `session_memory`/`quest_memory` array fields | `internal/gemini/client.go` already proves array-of-string schema properties work and decode safely (`answerWire`'s tolerant partial-decode pattern); free-text parsing is a proven source of injection surface this project has spent two phases hardening against |
| Per-user AI send rate limiting (DR-4-03) | A new token-bucket implementation in `internal/driver` | `session.RateLimiter` (already written, already used for websocket ingress in `websocket.go`) — reuse the type, do not reinvent the algorithm | One rate-limiting primitive in the codebase, not two; the existing type is already correct and simple (refill-on-elapsed, no external deps) |
| Two independent "why is this waiting" flags | A `WaitingReason` string enum with combined values | Two `bool` fields on `AutopilotRecord` (Pattern 2) | Booleans compose correctly under concurrent pause+disconnect; a string enum needs extra parsing/combining logic at every call site for no benefit |
| Pop-out window content sync | A second `useSession()`-like context, or `postMessage` between windows | `createPortal(<SameComponent/>, childWindow.document.body)` — the exact same React component tree, same hooks, same `wsManager` instance, painted into a different `document` (05-UI-SPEC.md §7, already specified in detail) | A popped-out window is not a new page load and has no router; it is the same JS process painting into a second document object. `postMessage` would be solving a problem that does not exist here. |
| Chat message persistence keyed to "the current login" | A new session/expiry concept parallel to `users.login_started_at` | The exact SQL shape `transcripts.go`'s `openGameSessionSQL`/`sessionMemoryForConnectionSQL` already use (`JOIN users ON ... WHERE gs.started_at >= u.login_started_at`) | D-08 explicitly says coaching's lifetime equals Session Memory's (D-31); the login boundary column and its query pattern already exist and are already tested |

**Key insight:** Every non-trivial problem in this phase already has a same-shaped solved problem sitting in the Phase 3/3.1/4 code (structured JSON answers, untrusted-data delimiting, login-scoped persistence, rate limiting, engage/disengage hooks). Phase 5's job is disciplined reuse of four already-proven patterns applied to two new pieces of state (coaching, waiting-reasons), not new mechanism design.

## Common Pitfalls

### Pitfall 1: `EnterWaiting`/`Resume`'s pure-function contract silently assumes one reason, and callers that "just add a parameter" will get the boolean logic wrong

**What goes wrong:** A tempting, minimal-diff fix is to give `EnterWaiting(cur, cause string)` and `Resume(cur, cause string)` an extra parameter and have the *caller* decide whether to actually transition based on comparing `cause` against some remembered value — this reintroduces the exact bug class D-15 is designed to prevent (a reconnect naively calling the existing `Resume` pure function, which unconditionally maps Waiting→On, silently overriding a pause the owner never lifted).

**Why it happens:** The existing four pure functions in `autopilot.go` are elegant and total (every state maps to exactly one next state); it's natural to want to extend rather than replace them. But their totality is precisely what breaks once "waiting" needs to remember *why*.

**How to avoid:** Keep `Engage`/`Disengage`/`EnterWaiting`/`Resume` completely unchanged (they still correctly describe the Off/On/Waiting *state* transitions in isolation) and add the two reason booleans as **Manager-level bookkeeping alongside, not inside, the pure functions** — the `Manager` methods decide whether to call `Resume` at all (only when both booleans are about to be false), never asking `Resume` itself to know about reasons.

**Warning signs:** A code review that finds `Resume` or `EnterWaiting` in `autopilot.go` growing a `reason` parameter, or a test asserting `Resume(Waiting, "owner")` and `Resume(Waiting, "connection")` behave differently — that is a sign the reason logic leaked into the wrong layer.

### Pitfall 2: The reviewer's `ReviewAnswer.Blocked bool` zero-value silently means "not blocked" — fixing DR-4-02 requires checking for field *absence*, not just handling a decode error

**What goes wrong:** `internal/gemini/client.go`'s `ReviewCommand` decodes directly into `ReviewAnswer{Blocked bool; Reason string}` via `json.Unmarshal(inner, &answer)`. If Gemini's structured output ever omits the `blocked` key entirely (a malformed-but-syntactically-valid-JSON edge case `Required: []string{"blocked", "reason"}` is supposed to prevent but is a vendor-side promise, not a Go-side guarantee), `json.Unmarshal` succeeds with `Blocked` at its Go zero value, `false` — read by `driver.go`'s `if review.Blocked { ... }` as "not blocked," and the command is sent. A naive fix that only adds a `json.Unmarshal` error check does nothing, because there is no unmarshal error in this case — the JSON is valid, just missing a key.

**How to avoid:** Change `ReviewAnswer.Blocked` to `*bool` (or decode into a wire struct with `*bool` and map it), and in `ReviewCommand`, treat a `nil` `Blocked` after decode as a `*gemini.Error{Kind: KindMalformed}` — the exact same failure path a genuinely malformed JSON body already takes, which `driver.go`'s existing `recordFailure(..., failureMalformed, ...)` handling already stops from dispatching (D-24: "a checker answer that arrives without its blocked yes-or-no is a failed review, which already stops the command"). This is a two-file change (`client.go`'s struct + decode-time check, `driver.go` needs no change if the error path is used) rather than a `driver.go`-side special case.

**Warning signs:** A test named something like `TestReviewCommand_MissingBlockedField` that constructs a raw JSON response body `{"reason":"..."}`  (no `blocked` key) and asserts the command is never sent — this is the regression test D-24 explicitly asks for ("A test for each").

### Pitfall 3: A popped-out window's `document` has no linked stylesheet until JavaScript copies one in — a blank `window.open('', ...)` starts with zero CSS

**What goes wrong:** `05-UI-SPEC.md §7`'s `openPopout` helper correctly clones every `<link rel="stylesheet">` and `<style>` tag from the parent document into the child window's `<head>` — but this only works if it runs **after** the parent document's own stylesheets have loaded (a pop-out triggered before the app's CSS has painted would clone empty or partial style rules). Since the pop-out button only exists inside an already-rendered, already-connected play screen, this is unlikely to bite in practice, but a fast page-load-then-immediately-click sequence during manual testing could expose a race if any stylesheet is still loading asynchronously (unlikely with Vite's bundled CSS, which is typically inlined or blocking, but worth a defensive check).

**How to avoid:** No code change needed if Vite's build (`vite build`, confirmed in `frontend/package.json`'s `build` script) produces a single blocking `<link>` for the app's CSS, which is Vite's default behavior — flag this as a one-line manual-test note in the plan ("click Pop out immediately after a hard refresh; the popped-out window should not flash unstyled") rather than a code fix.

### Pitfall 4: There is no browser-side test framework at all — pop-out lifecycle correctness can only be proven by a staging screenshot, not a unit test

**What goes wrong:** A plan that promises `vitest`/jsdom coverage for `window.open`, `createPortal`, or the `beforeunload` cleanup handler will fail immediately: this repository has **zero** frontend test files (confirmed by `Glob frontend/**/*.test.*` returning only a stray `node_modules/gensync` file) and no `vitest.config.*` anywhere in `frontend/`. `jsdom` also does not implement real multi-window behavior (`window.open` in jsdom returns a mock or `null` depending on configuration, and never actually paints a second `document`), so even installing a test framework this phase would not meaningfully exercise the pop-out mechanism — it would only prove the component renders without throwing.

**How to avoid:** Per `05-UI-SPEC.md §7`'s own "Evidence note," the pop-out capability's proof is explicitly "a single end-user screenshot of the play screen showing both the AI-player and AI-chatter views live in their own separate browser windows at the same time" — this is already the correct plan and should not be second-guessed by trying to retrofit jsdom coverage. Backend logic (coaching persistence, waiting-reason state machine, rate limiting) gets Go unit tests as normal; the pop-out mechanism gets a staging screenshot, full stop.

**Warning signs:** A plan task named "add vitest and test the pop-out window" — this is scope creep this phase does not need and CLAUDE.md's "no speed bumps" principle argues against introducing a whole new test toolchain for one browser-only feature when a screenshot already satisfies the evidence rule.

## Code Examples

### Existing structured-output pattern to extend for AI-chatter (verified shipped code)
```go
// Source: this repo, internal/gemini/client.go:193-216 (GenerateContent) — the
// exact shape a new Chat method follows: a schema, doGenerate, then a
// tolerant decode.
func (c *Client) Chat(ctx context.Context, endpoint, model, apiKey, systemInstruction, userText string) (*ChatAnswer, error) {
    schema := responseSchema{
        Type: "object",
        Properties: map[string]schemaProperty{
            "reply":    {Type: "string"},
            "push":     {Type: "array", Items: &schemaProperty{Type: "string"}},
            "withdraw": {Type: "array", Items: &schemaProperty{Type: "string"}},
        },
        Required:         []string{"reply"}, // push/withdraw optional, exactly like session_memory/quest_memory
        PropertyOrdering: []string{"reply", "push", "withdraw"},
    }
    inner, err := c.doGenerate(ctx, endpoint, model, apiKey, systemInstruction, userText, schema)
    if err != nil { return nil, err }
    return decodeChatAnswer(inner) // tolerant partial-decode, mirroring decodeAnswer's pattern exactly
}
```

### Existing login-scoped persistence pattern to extend for coaching (verified shipped code)
```sql
-- Source: this repo, internal/store/transcripts.go:70-82 (openGameSessionSQL).
-- A new coaching_suggestions column (or table) follows this exact inheritance
-- shape so D-08's "lasts for the rest of the MUDPuppy login" is a database-level
-- guarantee, not an application-level convention that could be gotten wrong
-- per call site.
INSERT INTO game_sessions (user_id, connection_id, session_memory, coaching_suggestions)
VALUES ($1, $2,
  COALESCE((SELECT gs.session_memory FROM game_sessions gs JOIN users u ON u.id = gs.user_id
            WHERE gs.user_id=$1 AND gs.connection_id=$2 AND gs.started_at >= u.login_started_at
            ORDER BY gs.started_at DESC LIMIT 1), '[]'::jsonb),
  COALESCE((SELECT gs.coaching_suggestions FROM game_sessions gs JOIN users u ON u.id = gs.user_id
            WHERE gs.user_id=$1 AND gs.connection_id=$2 AND gs.started_at >= u.login_started_at
            ORDER BY gs.started_at DESC LIMIT 1), '[]'::jsonb)
)
RETURNING id;
```
(Whether this is a new column on `game_sessions`, following Session Memory's exact precedent, or a separate `coaching_suggestions` table keyed the same way, is Claude's Discretion per CONTEXT.md — the SQL inheritance *pattern* is what matters and is already proven.)

### Existing pass-through gap in the dispatcher (DR-4-03's exact root cause, verified this session)
```go
// Source: this repo, internal/icm/dispatcher.go:208-224 (Dispatch)
handler := d.getHandler(normalized.Operator, normalized.Command)
if handler == nil {
    // No handler - this might be pass-through case
    return nil, nil   // <-- every plain MUD command an AI sends exits HERE,
}                       //     before RecordExecution below is ever reached.
result, err := handler.Handle(ctx, normalized)
if err != nil { return nil, err }
d.safety.RecordExecution(sessionID, normalized.Command) // never reached for AI sends
```

## State of the Art

| Old Approach (Phase 4) | New Approach (Phase 5) | When Changed | Impact |
|--------------------------|------------------------|---------------|--------|
| One model-call path per stint (player + reviewer), keyed by stint epoch | A second, independent model-call path (AI-chatter) with no epoch, triggered by inbound chat, coexisting with the loop | This phase | New concurrency surface: a chat reply and a decision call can be in flight for the same user at once, both drawing from the same call-cap counter (Pattern 1) |
| `AutopilotState` alone (off/on/waiting) fully describes the switch | Two additional reason booleans needed to describe *why* waiting, for correct multi-cause resume | This phase | Every place that reads `AutopilotStateFor` and assumes "waiting" has one meaning must now also read the reason fields for correct UI/behavior (status line text, D-15) |
| Memory layers (Session/Quest) are the only model-adjacent, untrusted-delimited prompt sections | Coaching becomes a fourth such section, but framed as trusted-owner-guidance-relayed-by-a-trusted-relay rather than model's-own-past-output | This phase | The `untrustedDataParagraph` sentence for `<COACHING>` must be worded differently from the Quest/Session Memory sentences — copy-pasting the exact wording would incorrectly tell the model to distrust the owner's own coaching |
| AI commands are subject to the ICM dispatcher's shared, global rate-limit/circuit-breaker state (in practice: not really, per DR-4-03) | AI commands get a dedicated, per-user, per-profile-configurable rate check independent of the dispatcher | This phase | Closes a real gap; also means AI-player's send path now has two limiter layers to reason about (the dispatcher's pre-existing checks for pass-through commands, which still run but don't record, and the new driver-owned one) — document this clearly so a future phase does not "simplify" by merging them incorrectly |

**Deprecated/outdated:** None — Phase 5 only extends Phase 1-4 code; nothing is retired. The `RAILWAY_ENVIRONMENT`-only vault-key gate (D-26) is replaced by an explicit local-dev flag, but the underlying `crypto.ParseKey`/`DefaultKeyStore` mechanism is unchanged.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `createPortal`'s React 18 signature and behavior (unchanged since React 16) is training-data knowledge, not independently re-verified against react.dev this session | Standard Stack | Low — this is a long-stable, widely-documented API; `05-UI-SPEC.md`'s own code samples already use it in exactly this shape and were independently checker-approved |
| A2 | `HandleChat` as a new sibling method (not a `runIteration` branch) is the right architectural boundary | Architecture Pattern 1 | Medium — this is a design recommendation, not something read directly from existing code (no AI-chatter code exists yet); if the planner instead extends `runIteration`, the epoch/staleStage entanglement described in the pitfall must be explicitly worked around, not just "made to work" |
| A3 | Two boolean reason fields (rather than a reason-count integer, a set type, or a small state-machine library) is the simplest correct structure for D-15's two-reason waiting state | Architecture Pattern 2 | Low — this is a small, easily-revisable data-shape choice; the *behavioral* contract (both must clear before resume) is what D-15 actually requires and is independent of which Go type expresses it |
| A4 | The DR-4-03 fix should live entirely in `internal/driver` and never touch `internal/icm/dispatcher.go` | Architecture Pattern 3 | Medium — this is the research's strongest opinion in this document; an alternative (adding a dedicated `RecordAISend`/`CheckAISend` pair to `icm.SafetyChecker` with its own independent, resettable counter, called explicitly by the driver but *implemented* in icm) is equally valid and might be preferred if the planner wants the limiter's implementation colocated with the dispatcher's other safety checks — the recommendation here is "don't reuse the existing pass-through/circuit-breaker path," not "the new code must live in package driver specifically" |
| A5 | Vite's production build emits a single blocking `<link rel="stylesheet">` (not async/deferred), so Pitfall 3's stylesheet-cloning race is not a real concern | Common Pitfall 3 | Low — Vite's default CSS handling is well-documented and this project's existing `vite build` output was not directly inspected this session for its exact `<link>` tag emission; if wrong, the fix is a one-line manual-test note in the plan, not a design change |

**If this table is empty:** N/A — see rows above.

## Open Questions

1. **Should the coaching store be a new column on `game_sessions` (mirroring `session_memory` exactly) or a separate table?**
   - What we know: D-08 ties coaching's lifetime to Session Memory's lifetime exactly (Phase 4 D-31's login-scoped inheritance). The SQL pattern for inheriting a JSONB column across the login boundary is already written and tested (`transcripts.go`).
   - What's unclear: whether coaching's separate concerns (it needs individual add/withdraw semantics driven by AI-chatter's structured answer, versus Session Memory's whole-list-replace semantics driven by AI-player) argue for its own table with per-entry rows (enabling, e.g., an audit trail of *when* each suggestion was pushed/withdrawn) rather than a JSONB array column.
   - Recommendation: Claude's Discretion, explicitly deferred by CONTEXT.md ("coaching suggestions stored per login and profile like Session Memory"). A JSONB column matching `session_memory`'s exact shape is the lower-effort, more-consistent choice; a dedicated table is only worth the extra migration/query complexity if the planner wants per-suggestion timestamps surfaced somewhere (nothing in the UI spec currently shows one).

2. **Does the "Coaching received" marker (D-05) ride the existing `MsgTypeAI` channel or does it need its own?**
   - What we know: `05-UI-SPEC.md` confirms the marker is a plain `.ai-system-line` in the thinking stream, styled exactly like the existing goal-changed neutral line, and "carries no suggestion text."
   - What's unclear: whether this marker is emitted by `HandleChat` (immediately, when the push happens) or by the *next* `runIteration` (when the coaching is actually read into a prompt) — D-05 says "at that moment" (implying immediate, from `HandleChat`) but also "the marker, followed by the next decision's logged reasoning reflecting the coaching, is the evidence for success criterion 1" (implying the marker precedes, not depends on, the next decision).
   - Recommendation: Emit the marker from `HandleChat` at the moment of push/withdraw, over the existing `MsgTypeAI`/`Notifier.NotifyDecision` path (Kind: "system", Outcome: something new like "coaching-received") — this matches "at that moment" literally and requires no new websocket message type for this one line, since `AIAssistPanel.tsx`'s existing `handleAI` switch already renders any `kind: 'system'` entry as a bracketed line in the thinking stream.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | All backend work | Yes | go1.26.0 windows/amd64 | — |
| `go test ./internal/...` | All new unit tests | Yes, confirmed green this session (full suite ~7.7s combined across packages) | — | — |
| React 18 + `createPortal` | Pop-out windows | Yes | react/react-dom `^18.3.1` in `frontend/package.json` | — |
| Frontend test framework (vitest/jsdom) | N/A this phase | No — zero test files, no config anywhere in `frontend/` | — | Pop-out capability is proven by staging screenshot per `05-UI-SPEC.md §7`'s own evidence note (Common Pitfall 4); no fallback needed since no test was ever planned to cover this |
| Gemini API reachability (`gemini-3.5-flash-lite`, per Phase 4 STATE.md) | AI-chatter's live model call, staging walkthrough only | Not tested this session (no live model calls made during research, per standing instruction) | confirmed working in Phase 4/3.1 staging deploys | Unit tests for `HandleChat`/`Chat` are fake-only by design, matching every prior phase's test discipline |
| Railway CLI / `railway up` staging deploy | Staging walkthrough, not development | Assumed available per every prior phase's evidence trail | — | — |

**Missing dependencies with no fallback:** None.

**Missing dependencies with fallback:** Frontend test framework (not needed — screenshot evidence is the correct, already-specified proof mechanism for the one browser-only capability this phase adds).

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go standard library `testing` (backend); no frontend framework exists or is needed this phase (see Common Pitfall 4) |
| Config file | none — `go.mod` at repo root, `go 1.26` |
| Quick run command | `go test ./internal/driver/... ./internal/session/... -count=1` |
| Full suite command | `go test ./internal/... -count=1` |

Measured this session (fresh run, no `-race`): full suite **~7.7 seconds** across all packages (`driver` 2.0s, `session` 2.5s, remainder ~3.2s combined), all green against the current Phase 4 build. `go build ./...` completes cleanly.

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| REQ-coaching-chat | A pushed coaching line appears in the next decision's prompt and is reflected in logged reasoning | unit | `go test ./internal/driver/... -run TestHandleChat_PushReachesNextPrompt -v` | ❌ Wave 0 |
| REQ-coaching-chat | Withdraw removes a suggestion from the coaching list and the next prompt | unit | `go test ./internal/driver/... -run TestHandleChat_Withdraw -v` | ❌ Wave 0 |
| REQ-coaching-chat | Chat cap-reached: no model call, owner message still stored, locked notice | unit | `go test ./internal/driver/... -run TestHandleChat_CapReached -v` | ❌ Wave 0 |
| REQ-pause-resume | Pause stops the loop with no model calls/sends; window keeps filling | unit | `go test ./internal/driver/... -run TestLoop_PauseNoDecisions -v` | ❌ Wave 0 |
| REQ-pause-resume | Resume fires only when both PausedByOwner and ConnectionLost are false | unit | `go test ./internal/session/... -run TestManager_ResumeRequiresBothReasonsClear -v` | ❌ Wave 0 |
| REQ-pause-resume | First decision after resume carries the reassess instruction (reuses Phase 4 D-08 path) | unit | `go test ./internal/driver/... -run TestEngageLoop_Reassess -v` | ✅ existing (Phase 4), regression only |
| DR-4-02 (carried) | A reviewer answer missing `blocked` is treated as a failed review, command not sent | unit | `go test ./internal/gemini/... -run TestReviewCommand_MissingBlockedField -v` | ❌ Wave 0 |
| DR-4-01 (carried) | Corpus items `chill touch golem`/`static blast crystal` pass as benign, sampled more than once | live (flagged, gated) | `MUDPUPPY_LIVE_CORPUS=1 MUDPUPPY_LIVE_CORPUS_ONLY=<new-ids>,<new-ids> go test ./internal/driver/... -run TestLiveCorpus_HostileText -v` | ❌ Wave 0 (new corpus items) |
| DR-4-03 (carried) | AI sends are refused when the configured per-second limit is exceeded; a flood test proves it | unit | `go test ./internal/driver/... -run TestAllowAISend_FloodRefused -v` | ❌ Wave 0 |
| DR-4-04 (carried) | Server refuses to start without a vault key on any host, unless explicit local-dev flag is set | unit | `go test ./internal/config/... -run TestLoadRequiresEncryptionKeyEverywhereUnlessLocalDev -v` | ❌ Wave 0 |
| D-27 (carried) | `/logs/:connectionId` sends a signed-out visitor to sign-in | manual (screenshot) | N/A — `AuthGuard` in `App.tsx` already wraps this route (confirmed by direct read this session); this may require zero code change, only a verification screenshot | ✅ existing code path, evidence-only |

### Sampling Rate

- **Per task commit:** `go test ./internal/driver/... ./internal/session/... ./internal/gemini/... ./internal/config/... -count=1`
- **Per wave merge:** `go test ./internal/... -count=1` (full suite, ~7.7s)
- **Phase gate:** Full suite green, plus the live corpus rerun (DR-4-01, flagged/gated, spends real quota) before `/gsd:verify-work`

### Wave 0 Gaps

- [ ] `internal/driver/chat_test.go` — new file, covers REQ-coaching-chat's three rows above
- [ ] `internal/driver/ratelimit_test.go` — new file, covers DR-4-03's flood test
- [ ] `internal/session/manager_test.go` (extend) — covers the new `PauseAutopilot`/`ResumeAutopilotByOwner` pair and the both-reasons-clear rule
- [ ] `internal/gemini/client_test.go` (extend) — covers DR-4-02's missing-field regression test
- [ ] `internal/config/config_test.go` (extend) — covers DR-4-04's everywhere-unless-local-dev rule
- [ ] `fakeModels` extension (`driver_test.go`) — needs a `Chat` method on the fake to support `HandleChat` tests, mirroring its existing `GenerateContent`/`ReviewCommand` fake methods
- [ ] Framework install: none — everything above is additive to the existing `testing`-only setup

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V1 Architecture, Design and Threat Modeling | Yes | AI-chatter's provenance guarantee (Pattern 4: only `HandleChat` writes coaching, never anything reading raw game text) is the primary architectural control against the laundering channel D-28 names |
| V4 Access Control | Yes | New chat/coaching REST/WS routes reuse the existing `getProfileByConnectionID` ownership-check pattern, unchanged from every prior AI sub-resource |
| V5 Input Validation | Yes | Coaching bullets go through the existing `clampBullets`/`neutraliseLine` pipeline (size ceiling, no embedded markers); the owner's chat message itself needs a length cap (recommend matching the session-goal precedent, ~1000 chars) before it ever reaches a model call |
| V6 Cryptography | Yes | DR-4-04's vault-key-everywhere fix (D-26) — never weaken `crypto.ParseKey`'s strictness as part of this change |
| V8 Data Protection | Yes | Chat/coaching text falls under the same retention discipline as decision rows (Claude's Discretion: "whether the nightly retention job touches it — default: keep with the decision rows' kept fields") |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Coaching-channel laundering: hostile game text convinces AI-player or AI-chatter to write something that later re-enters AI-player's prompt as apparently-owner-authored coaching | Tampering / Elevation of Privilege | Provenance guarantee (only `HandleChat`, driven by the owner's own inbound chat message, ever writes the coaching store) plus the existing untrusted-delimiting mechanism applied to `<COACHING>` too (Pattern 4); red-team corpus gains AI-chatter-channel attack items per Claude's Discretion |
| Rate-limit bypass via the dispatcher's handler-nil pass-through (DR-4-03) | Denial of Service (against the game server / the owner's own account standing) | Dedicated driver-owned rate check independent of the dispatcher (Pattern 3) |
| Reviewer fail-open via a missing `blocked` field (DR-4-02) | Tampering (of the safety-review outcome) | Treat missing field as a failed review, same fail-closed path as any other malformed reviewer answer (Pitfall 2) |
| Silent vault-key regeneration outside Railway (DR-4-04) | Tampering (of the credential vault's integrity) | Fail-closed everywhere unless an explicit, named local-dev opt-out is set |

## Project Constraints (from CLAUDE.md)

- Plans are capabilities with observable acceptance criteria (Phase → Wave → Plan → Task); a plan named for a layer ("backend chat work") is invalid — Phase 5's plans should be named for capabilities (e.g., "A coaching message reaches the next decision" not "Add HandleChat method").
- Evidence is a canned report, `[AI-PLAYER]` log excerpt, or end-user screenshot — never a database query. Every Phase 5 plan's acceptance criteria must specify one of these three forms; the pop-out capability specifically requires a screenshot (Common Pitfall 4).
- Push `ai-player` to GitHub at phase close, after security-review commits land, per the standing rule from the Phase 3 review (DR-3 R-13).
- Deploy staging only via `railway up`; the dashboard's Deploy rebuilds from GitHub and will crash staging if the branch is ever behind.
- Do not read or cite prior `.specify/` specs beyond `ai-game-player-design-v3.md`, `safety-and-abuse-policy-v1.md`, `ai-memory-model-v1.md`, and `phase-based-development-approach.md` — product is taken as-is.
- No package may be added to `go.mod` or `frontend/package.json` without it showing up in the test report's dependency-drift section (empty is expected and required for Phase 5).
- Never log out of an active owner session; the `/logs/:connectionId` sign-in-gate screenshot (D-27) must come from a private/incognito window or a separate browser profile, never from signing the owner out.
- Every question posed to the owner must include "I need to discuss this further before we move on" as an explicit option (repo-wide user instruction, applies to the plan/discuss phase, not to this research document itself).

## Sources

### Primary (HIGH confidence)
- Direct code read this session: `internal/driver/driver.go` (full, 1538 lines), `internal/driver/loop.go` (full), `internal/driver/memory.go` (full), `internal/session/autopilot.go` (full), `internal/session/manager.go` (full, 1340 lines), `internal/session/websocket.go` (full, 818 lines), `internal/session/handler.go` (Autopilot handler + response structs), `internal/icm/dispatcher.go` (full), `internal/icm/types.go` (SafetyLimits), `internal/gemini/client.go` (full, 382 lines), `internal/crypto/crypto.go` (full), `internal/config/config.go` (full), `internal/store/profile.go` (full), `internal/driver/corpus_live_test.go` (header + corpusItem shape), `frontend/src/components/AIAssistPanel.tsx` (full), `frontend/src/components/AIPlayerPanel.tsx` (AI Settings section), `frontend/src/context/SessionContext.tsx` (full), `frontend/src/App.tsx` (routing/AuthGuard), `frontend/src/pages/LogsPage.tsx` (header), `frontend/src/pages/HelpPage.tsx` (header), `frontend/src/services/api.ts` (WebSocketManager, full), `frontend/package.json`, migrations `011`, `013`, `014`, `internal/store/transcripts.go` (OpenGameSession/SessionMemoryFor/SessionMemoryForConnection), `internal/profiles/handler.go` (GetAISettings/PutAISettings/validateAISettings).
- `.planning/RISK-REGISTER.md` (full) — exact wording and criticality of DR-4-01 through DR-4-04, and the owner's verbatim notes on each.
- `.planning/todos/pending/2026-09-17-phase5-security-carry-forward.md` — the exact recommended fixes and the dispatcher root-cause note quoted verbatim in this research.
- Live commands run this session: `go version` (go1.26.0 windows/amd64), `go build ./...` (clean), `go test ./internal/... -count=1` (~7.7s, all green).
- `05-UI-SPEC.md` (full, approved 2026-09-17) — the complete, locked frontend contract this research defers to rather than re-deriving.
- `05-CONTEXT.md` (full, approved 2026-09-17) — the locked decisions and discretion areas reproduced verbatim in `<user_constraints>` above.
- `.specify/specs/ai-memory-model-v1.md` (full) — governs where coaching sits relative to the four memory layers (it does not; coaching is explicitly outside this model, a live-guidance channel, per D-17/D-08).

### Secondary (MEDIUM confidence)
- Training-data knowledge of React 18's `createPortal` API surface, cross-checked against `05-UI-SPEC.md`'s own already-checker-approved code samples using it in the identical shape — not independently re-fetched from react.dev this session.

### Tertiary (LOW confidence)
- None identified as needing flagging beyond what is already in the Assumptions Log.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies anywhere; every extension point verified by direct code read this session
- Architecture (AI-chatter call path, pause/resume state machine, rate-limit fix): HIGH for the mechanism and exact code locations (all read directly), MEDIUM for the specific design recommendations (HandleChat as sibling, two-boolean reasons, driver-owned rate check) since no Phase 5 code exists yet to verify against — these are reasoned proposals building on proven Phase 3/4 patterns, not confirmed facts
- Security carry-forwards (DR-4-01 to DR-4-04): HIGH — every root cause is a directly-read, line-cited code fact (the dispatcher's pass-through return, the `Blocked bool` zero-value trap, the `RAILWAY_ENVIRONMENT`-only gate), not speculation
- Frontend (pop-out windows, panel split): HIGH for what the already-approved `05-UI-SPEC.md` locks (this research defers to it entirely), MEDIUM for the two flagged pitfalls (stylesheet-load race, no test framework) since these are risk assessments, not code-verified facts

**Research date:** 2026-09-17
**Valid until:** 14 days (matching Phase 4's own precedent — the Gemini model-naming/behavior landscape moves within this project's timeline; re-verify `gemini-3.5-flash-lite` is still serving before the staging walkthrough, and re-confirm no Phase 4 code drifted underneath this research if more than two weeks elapse before planning executes)
