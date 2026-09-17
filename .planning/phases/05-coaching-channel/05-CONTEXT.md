# Phase 5: Coaching Channel - Context

**Gathered:** 2026-09-17
**Status:** Ready for planning

<domain>
## Phase Boundary

The AI Assist panel gains a second level. Two names are locked by the owner and used everywhere from here on:

- **AI-player**: the existing server-side loop that reads the game, decides, and sends commands (Phases 3 and 4). Its thinking stream (reasoning, commands, notices) stays as it is.
- **AI-chatter**: a new conversation level that sits above AI-player. The owner talks to it in plain text; it always replies in plain text; and it can push coaching suggestions down into AI-player mid-stream so they take effect on AI-player's next decision. It can read everything AI-player knows. That is all it can do: **no other application permissions or functionality**. It cannot change settings, cannot work any control, and nothing typed in chat is ever sent to the game.

This phase also delivers Pause/Resume as an **AI-player** control (not a chat feature), a Help page article explaining AI-chatter, AI-player and how to apply a suggestion to AI settings by hand, and the fixes for all five security items carried from the Phase 4 review (three high, one low, one owner condition).

**Removed from this phase by owner decision (2026-09-17): promotion.** There is no "promote a chat instruction into the profile" feature. Chat is only ever plain text. If the owner wants a reply in the profile, he copies it and edits his AI settings by hand on the existing Settings page. The design doc, REQUIREMENTS.md and ROADMAP.md are amended to match (owner approved the amendment in this discussion).

Not in this phase: editing Session Memory by hand (dropped, see D-19); any button or action on a chat message; AI-chatter changing configuration of any kind; the progression tally, debrief, Historical Memory and quest closure (Phase 6); the study loader (Phase 7).

</domain>

<decisions>
## Implementation Decisions

### The two levels and the panel
- **D-01:** Owner's words: "Chat between AI and User should sit at a meta level above the AI stream reading and sending commands into the game. however that chat should be able to influence the stream logic realtime in the game." The conversation is about the play, not part of it: its own view, its own stored history, the AI replies there. AI-player never waits on the conversation; it keeps its own pace (Phase 4 D-06).
- **D-02:** The names **AI-chatter** and **AI-player** are the owner's and are used in code comments, log lines, UI copy, the help article and planning documents.
- **D-03:** Chat lives **inside the existing AI Assist panel** (floating, 380 by 760, Phase 4 D-30), as a **split: AI-player's thinking stream on top, the AI-chatter conversation and message box below, both visible at once**. No tabs, no new pane, no docked column. The AI-player part keeps the goal box, status line, Session Memory section, and gains the Pause/Resume button (D-13) and the "Coaching in effect" list (D-09).
- **D-04:** The owner's messages and AI-chatter's replies appear **only in the conversation area**, never mixed into the thinking stream. This supersedes the roadmap's "one stream" wording (criterion 4), which is amended to "one panel, two views".
- **D-05:** When a coaching suggestion reaches AI-player, the **thinking stream prints a one-line marker** in the family of "Coaching received" at that moment. The message text stays in the chat. The marker, followed by the next decision's logged reasoning reflecting the coaching, is the evidence for success criterion 1.
- **D-06:** The **message box is usable whenever the panel shows**: autopilot ON, OFF or WAITING (paused or disconnected). A message sent while autopilot is off works the same: AI-chatter replies, and any coaching becomes a standing suggestion already in place at the next `#AUTO ON`.
- **D-07:** **Chat messages are plain text with no functionality.** Owner's words: "there's no functionality in the text... it's just text back and forth." No promote button, no per-message actions. (A browser's ordinary text selection and copy is all the owner needs.)

### How coaching sticks
- **D-08:** Each piece of coaching AI-chatter pushes down becomes a **standing suggestion that AI-player reads on every decision for the rest of the MUDPuppy login**, per connection profile: the same lifetime as Session Memory (Phase 4 D-31). A refresh, a game reconnect, `#AUTO OFF` then `#AUTO ON`, a wheel-grab or a redeploy keeps them; a new sign-in starts clean.
- **D-09:** **Nothing reaches AI-player that the owner cannot see.** When AI-chatter pushes a suggestion down, its reply quotes the exact line sent (in the family of "Sent to AI-player: avoid the north road"). The AI-player part of the panel shows a **read-only, collapsible "Coaching in effect" list** beside Session Memory, updated live and reloaded after a refresh.
- **D-10:** The owner **takes a suggestion back by telling AI-chatter in words** ("forget what I said about the north road"); AI-chatter withdraws that line and confirms which one it removed. The suggestions are AI-chatter's own working list, not configuration, so this stays inside "chat and push suggestions". No clear button.
- **D-11:** **AI-chatter always gives a short reply** to every message: what it understood and what AI-player will now do differently, or the answer to a question. Each reply is one model call and **counts against the session call cap** so the cap stays an honest cost limit (Phase 4 D-14). While autopilot is off no stint is running, so replies are held to the same cap number on their own count.
- **D-12:** **Profile rules always hold in the game.** Coaching never overrides the conduct rules, the Never-issue list or the safety checker; the checker keeps checking every command, coached or not. When coaching conflicts with a rule, AI-chatter does not flatly refuse: owner's words, "AI chat will not outright contradict the command, but it should give suggestions on how to better set up the filters to get the desired result," and "It only gives back a chat response suggestion and has no ability to affect the configuration itself." So the reply names which rule or filter is in the way and suggests the wording change that would get the result; the owner makes the change himself on the Settings page.

### Pause and resume
- **D-13:** Owner's words: "Pause/Resume should live with AI-Player, and not on AI-Chat interface." Pause/Resume is **one button in the AI-player part of the panel** (by the goal box and status line). It is instant and makes no model call. Nothing in chat pauses or resumes: typing "pause" in chat is just text. No `#AUTO PAUSE` directive.
- **D-14:** **While paused AI-player reads only.** Game text keeps flowing into the Immediate Context window so nothing is missed; no decisions are made, no model calls are spent, no commands are sent. The **first decision on resume reassesses the situation** exactly like a re-engage (Phase 4 D-08: fresh window snapshot, explicit re-check instruction). AI-chatter keeps working while paused.
- **D-15:** **Pause shows as WAITING, with a reason.** Owner: "Why not just WAITING ??" The badge stays three states (ON, OFF, WAITING). The server tracks why it is waiting: paused by the owner, connection lost, or both. It resumes **only when every reason is cleared**: a pause never resumes by itself, and a reconnect alone does not un-pause. The panel's status line gives the reason ("Paused by owner" / "Connection lost", exact wording Claude's discretion); the thinking stream prints a notice on pause and on resume. `#AUTO OFF` turns autopilot off from waiting as today. A page refresh keeps it paused.
- **D-16:** **Typing anything disengages autopilot, paused or not.** Owner's words: "if I type anything, AutoPilot disengages." The wheel-grab rule is the same in every engaged state and lands OFF. Text typed into the chat message box is not a game command and is not a wheel-grab.

### What AI-chatter can see
- **D-17:** AI-chatter reads **everything AI-player knows**, read-only: the session goal, the profile's conduct rules, approach guidance and Never-issue list, AI-player's recent decisions and reasoning (including blocks and failures), the recent game text (Immediate Context), Session Memory, the active Quest's memory, and the coaching currently in effect. So "why did you do that?", "what do you know about this quest?" and "which rule is stopping you?" get real answers. This delivers the **Quest Memory recall through chat** that Phase 4 D-11 parked for this phase.
- **D-18:** AI-chatter **changes nothing** except its own list of coaching suggestions. It has no write path to the profile, the goal, the memory layers, the autopilot state or the game.
- **D-19:** **Editing Session Memory by hand is dropped** (it was parked for this phase by Phase 4 D-10). Session Memory stays read-only on the panel. If a note is wrong the owner says so in chat; that goes down as a suggestion and AI-player corrects its own notes on its next decision.

### No promotion; help article; document amendments
- **D-20:** Owner's words: "There is no promotion of any text to do anything in the system. The text is only ever plain text. If a user wants to grab a reply and apply it to settings... they would copy/paste and manually adjust their AI settings." and "AI Chat just does that... it chats and can push suggestions down into the AI-player mid-stream. No other application permissions or functionality should be in scope." **REQ-promote-guidance as originally written is withdrawn.** No conduct-rules or approach-guidance editor is added to the play screen; the existing Settings page is where the owner edits them.
- **D-21:** A **help article is added to the app's Help page** (`frontend/src/pages/HelpPage.tsx`), in plain words: what AI-chatter and AI-player are, how a chat message reaches the player (the marker, the "Coaching in effect" list, the quoted line), how long coaching lasts and how to take it back, that chat can never change settings, how Pause/Resume works, and how to copy a suggestion from chat into Conduct rules or Approach guidance on the Settings page by hand.
- **D-22:** **Approved amendments (made with this context, owner's decision 2026-09-17):** design v3 D5's second and third bullets and Definition of complete item 4; `REQUIREMENTS.md` REQ-pause-resume, REQ-promote-guidance (becomes the plain-text and help-article requirement) and REQ-doc-coaching; `ROADMAP.md` Phase 5 goal, success criteria 2 to 4, Phase Validation and implementation notes; `PROJECT.md` Active line. The shelf copy in `D:\Projects\ai-mud-player\documents` is updated only when the owner approves the new design version.

### Security carry-forward (owner: "For any High level items, they need to be fixed in this Phase.")
- **D-23:** **DR-4-01 (high) is fixed here.** This supersedes the owner's earlier "later phases" note. Per the register's recommended fix: the checker is told plainly that fighting creatures and objects the game presents as targets is normal play and only attacking another player is off limits; harmless fight examples (including the live false blocks `c chill touch golem` and `c static blast crystal`) join the corpus as benign items; the corpus is rerun and must hold STEERED 0 with those items not blocked. Because one sample per item cannot show a coin-flip verdict (Phase 4 D-29), the rerun samples the new fight items more than once.
- **D-24:** **DR-4-02 (high) is fixed here**, per the recommended fix: a checker answer that arrives without its `blocked` yes-or-no is a failed review, which already stops the command; the reviewer prompt gains its own sentence saying the memory notes were written by the other model, not by itself. A test for each.
- **D-25:** **DR-4-03 (high) is fixed here, as a configurable setting.** Owner: "the limiter needs to be a setting that can be configured." Every AI-player command is recorded with the rate limiter before it is sent and the send is refused when the limiter says stop; a flood test proves it. The limit is an **AI setting on the profile** (Settings page, with the other AI settings); **blank means the server default**, the same pattern as the other blank settings. The audit's notes must be handled: `internal/icm/dispatcher.go` returns for a plain pass-through command before `RecordExecution`, and the dispatcher's circuit-breaker counter is never reset, so counting AI sends against it as it stands would eventually refuse every AI command in a long-running process.
- **D-26:** **DR-4-04 (low) is placed in this phase** (owner's earlier note: "Employ this fix next Phase with the recommended fix."): require the vault key everywhere unless a setting says this is local development.
- **D-27:** **The log page sign-in check is placed in this phase**: confirm `/logs/:connectionId` sends a signed-out visitor to sign-in, with an end-user screenshot from a private window or separate browser profile (never sign the owner out); add the guard and a test if it is missing; record the outcome against AR-4-08 at the Phase 5 security review.
- **D-28:** At the Phase 5 security review all four DR-4 rows are re-presented with what was built and the outcome recorded in an `Outcome` column on the Phase 04 Deferred table in `.planning/RISK-REGISTER.md`. **New risk to carry to that review:** AI-chatter reads untrusted game text and model-written memory (D-17) and then writes lines that AI-player reads as coaching (D-08). That is a possible laundering channel from game text into trusted-looking instructions. See Claude's Discretion for the required handling.

### Claude's Discretion
- **The AI-chatter injection channel (must be designed, not skipped).** AI-chatter's prompt delimits game text, Session Memory, Quest Memory and AI-player's reasoning as untrusted exactly as AI-player's and the reviewer's prompts do (Phase 3.1 D-01, Phase 4 D-13, D-24). A pushed coaching line must come from what the owner said, not from anything found in the untrusted blocks. In AI-player's prompt, coaching sits in its own marked section below the profile's standing text and the goal, stated as the owner's live guidance relayed by AI-chatter and subordinate to the conduct rules and Never-issue list (D-12); the reviewer sees it too. The red-team corpus gains items that attack through AI-chatter (hostile game text trying to get a line pushed down). Size ceilings on the number and length of standing suggestions, enforced in Go.
- How AI-chatter returns both a reply and zero or more push/withdraw actions (one structured answer from one model call is the intended shape; the researcher confirms it fits Gemini's constrained output). Which configured model AI-chatter uses (the same env-configured registry; nothing hard-coded).
- What happens to chat when the cap is reached: a locked notice in the conversation area, no reply sent, the owner's message still stored. How the off-stint count resets (per login is fine).
- Storage: chat messages and AI-chatter replies stored against the AI session so they are auditable beside the decisions they influenced; coaching suggestions stored per login and profile like Session Memory; migration numbers (`015`+). Whether the log page shows the conversation (default: yes, as its own section, since the owner wants it separate from the thinking stream).
- Chat text is captured owner text plus model text, not game text; whether the nightly retention job touches it (default: keep with the decision rows' kept fields; the "delete captured text now" action does not need to cover it).
- The websocket message type and fields for chat in both directions, the REST routes to reload the conversation and the "Coaching in effect" list on attach, and the pause/resume endpoint or message.
- How the waiting reasons are held in the session manager (a small set of reasons beside `AutopilotWaiting`), the `[AI-PLAYER]` log lines for pause, resume, chat received, reply sent, coaching pushed and withdrawn (ids, stages, outcomes, lengths only; never message text), and an `[AI-CHATTER]` tag if it reads better.
- The split's proportions, whether the divider can be dragged, the look of owner versus AI-chatter messages in the 380-wide panel, the Pause/Resume button's place and label, and all new notice wording, in the voice of `03-UI-SPEC.md`'s Copywriting Contract with no vendor text interpolated.
- The rate-limit setting's unit, server default and bounds; where it sits among the AI settings; how the dispatcher's counter reset and pass-through recording are fixed.
- Evidence for criterion 2 now that pause makes no decisions: the `[AI-PLAYER]` log excerpt showing the window still being fed while paused with no model calls and no sends, a canned report, and screenshots; never a database query.

### Folded Todos
- **`.planning/todos/pending/2026-09-17-phase5-security-carry-forward.md`**: folded in full. DR-4-01 becomes D-23, DR-4-02 becomes D-24, DR-4-03 becomes D-25, DR-4-04 becomes D-26; the review obligation is D-28.
- **`.planning/todos/pending/2026-09-17-confirm-log-page-login-gate.md`**: folded in full as D-27.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Product definition and method
- `.specify/specs/ai-game-player-design-v3.md` — D5 (coaching channel) and Definition of complete item 4, **as amended 2026-09-17** (no promotion; plain-text chat; Pause/Resume is an AI-player control). D7 still says the study loader summarises "in the chat pane": that is AI-chatter's conversation area.
- `.specify/specs/ai-memory-model-v1.md` — The four-layer memory model. AI-chatter reads Session and Quest Memory (D-17); it writes to neither.
- `.specify/specs/safety-and-abuse-policy-v1.md` — Sections 2, 5 and 6; policy stays at 1.0.
- `.specify/memory/phase-based-development-approach.md` — Phase, Wave, Plan, Task rules; plans are capabilities with observable acceptance criteria.
- `CLAUDE.md` — Evidence is a canned report, log file or end-user screenshot; Brief, Evidence Dossier and Risk Register per phase; push `ai-player` at phase close; deploy staging only with `railway up`.

### Planning state and prior decisions this phase builds on
- `.planning/ROADMAP.md` §Phase 5 — Goal, success criteria, Phase Validation and implementation notes, as amended 2026-09-17.
- `.planning/REQUIREMENTS.md` — REQ-coaching-chat, REQ-pause-resume, REQ-promote-guidance (reworded), REQ-doc-coaching, as amended 2026-09-17.
- `.planning/PROJECT.md` — Locked decisions and constraints: DEC-server-side-go-driver, DEC-autopilot-waits-across-disconnect, CON-mechanical-halts, CON-policy-cost, CON-engine-has-no-game-knowledge.
- `.planning/phases/04-continuous-play/04-CONTEXT.md` — D-06 (pacing), D-08 (reassess on re-engage; reused on resume), D-10 and D-31 (Session Memory lifetime; coaching copies it), D-11 (Quest Memory recall parked for chat), D-13 (prompt order and untrusted delimiting; coaching joins it), D-14 (cap counts every model call), D-17 (blocks), D-18 (switch state rides the `ai` message), D-29 (reviewer reliability), D-30 (panel height).
- `.planning/phases/03.1-prompt-injection-review/03.1-CONTEXT.md` — D-01 (untrusted delimiting), D-03 as amended (harm-aimed reviewer; D-23 here extends the ordinary-play exception to fighting), D-06 to D-08 (blocked outcome), D-12 (corpus pass bar).
- `.planning/phases/03-one-ai-decision/03-CONTEXT.md` and `.planning/phases/03-one-ai-decision/03-UI-SPEC.md` — Panel and terminal display decisions; the Copywriting Contract the new notices join.
- `.planning/phases/02-autopilot-switch/02-CONTEXT.md` — State model (ON, OFF, WAITING), wheel-grab, the `[AI-PLAYER]` log format. D-15 and D-16 here extend it.

### Security carry-forward
- `.planning/RISK-REGISTER.md` — DR-4-01 to DR-4-04 ("Raised again at: Phase 5 security review") and AR-4-08 with the owner's condition.
- `.planning/phases/04-continuous-play/04-SECURITY.md` — The threat register and audit notes for T-4-02, T-4-04, T-4-06 and review note IN-07.
- `.planning/todos/pending/2026-09-17-phase5-security-carry-forward.md` and `.planning/todos/pending/2026-09-17-confirm-log-page-login-gate.md` — The folded todos, with the owner's notes verbatim and the recommended fixes.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `frontend/src/components/AIAssistPanel.tsx` (401 lines): header, goal box, status line (call count, cap, failure and block counts), collapsible read-only Session Memory section, and the decision stream body. The split (D-03) divides this panel; the "Coaching in effect" list copies the Session Memory section's shape; the Pause/Resume button joins the top area. `frontend/src/index.css` `.ai-assist-panel` is `position: fixed`, 380 by 760.
- `frontend/src/pages/PlayScreen.tsx` mounts the panel when connected on an AI-activated profile; `frontend/src/context/SessionContext.tsx` registers the `onAI`/`offAI` websocket handlers; `frontend/src/services/api.ts` holds the `ai` event dispatch and the goal and memory fetches to copy.
- `frontend/src/pages/HelpPage.tsx`: the existing Help page the article (D-21) joins. `frontend/src/components/AIPlayerPanel.tsx`: the AI settings form where the rate-limit setting (D-25) goes.
- `internal/driver/driver.go`: `buildSystemInstruction(ctx promptContext)`, `buildReviewSystemInstruction`, `untrustedDataParagraph()` naming `<GAME_TEXT>`, `<QUEST_MEMORY>`, `<SESSION_MEMORY>`. `internal/driver/memory.go`: `promptContext`, `wrapQuestMemory`, `wrapSessionMemory`, `renderBullets`, `clampBullets`/`truncateBullets` ceilings. Coaching becomes a new field on `promptContext` with its own section; AI-chatter gets its own instruction builder reusing the untrusted paragraph.
- `internal/driver/loop.go` (275 lines): the loop goroutine, settle/floor/min-spacing pacing, call, failure and block counters. Pause is a loop state that keeps the window fed and skips the decision; resume reuses the first-iteration reassess path.
- `internal/session/autopilot.go`: `AutopilotState` (`off`, `on`, `waiting`) and the transition functions; `internal/session/manager.go`: `parkAutopilotLocked`, `resumeAutopilotLocked`, `DisengageAutopilot`, `SendCommandAs` (refuses to act as ai unless On, stint epoch). The waiting reasons (D-15) extend these.
- `internal/session/websocket.go`: `WSMessage`, `MsgTypeAI`, `MsgTypeAutopilot`, `AIDecisionPayload`; the wheel-grab in the `MsgTypeData` branch. Chat needs a new inbound and outbound message type that is never treated as game input.
- `internal/gemini/client.go`: `GenerateContent`, `ReviewCommand`, `responseSchema` with `propertyOrdering`; `ReviewAnswer.Blocked` is a plain bool today (D-24 target). AI-chatter's call is a third entry point in the same client.
- `internal/profiles/handler.go` and `cmd/server/main.go`: the `ai-settings`, `ai-goal`, `ai-memory` sub-resource pattern under `/api/v1/profiles/{connection_id}/` to copy for the conversation and coaching reads; `internal/store/profile.go` `AISettings`/`ResolveAISettings` for the new rate-limit field; `internal/store/decisions.go`, `quests.go`, `transcripts.go` as storage patterns.
- `internal/icm/dispatcher.go`: the rate limiter, `RecordExecution`, the pass-through early return and the never-reset circuit-breaker counter (D-25). `internal/config/config.go` and `internal/crypto/crypto.go` `DefaultKeyStore` (D-26).
- `internal/driver/corpus_live_test.go` and `internal/driver/testdata/`: the red-team corpus (benign fight items for D-23, AI-chatter channel items for D-28). `internal/driver/driver_test.go` and `loop_test.go` fakes for the pause, coaching and limiter tests. `scripts/verify-phase3-1.sh` and the Phase 4 harness as the canned-report pattern.

### Established Patterns
- Server owns truth; the browser reflects it and re-syncs on attach. The conversation, the coaching list and the waiting reason all flow server to browser and reload from REST after a refresh.
- Security boundaries live in Go in front of the dispatcher; untrusted delimiting is done in the instruction builders; the owner's own text (goal, and now his chat messages) is trusted, model-written text is not.
- Locked notices are fixed strings with no vendor text interpolated; log lines carry ids, stages, outcomes and lengths only.
- Blank setting means the server default, resolved in Go (`ResolveAISettings`), never in the browser.
- Tests with fakes and no network by default; evidence as files per success criterion in `evidence/`.

### Integration Points
- Driver: coaching section in both prompts; AI-chatter call path with read access to decisions, window, memory and profile text; pause state in the loop; cap counting for chatter replies.
- Session manager and websocket: waiting reasons; pause/resume control; chat message types; wheel-grab unchanged and applying while paused.
- Store and migrations (`015`+): conversation rows per AI session, coaching suggestions per login and profile, the rate-limit AI setting.
- REST: conversation and coaching reload on attach; pause/resume; AI settings gains the limiter field.
- Frontend: split panel, conversation area and message box, "Coaching in effect" list, Pause/Resume button, status-line waiting reason, Help article, limiter field in AI settings, log page conversation section and sign-in guard check.
- ICM dispatcher: AI sends recorded and refused by the limiter; counter reset; flood test.
- Reviewer: missing-`blocked` is a failed review; memory-authorship sentence; fighting is normal play; corpus additions and rerun.
- Config: vault key required everywhere unless a local-development setting says otherwise.

</code_context>

<specifics>
## Specific Ideas

- Owner on the architecture: "Chat between AI and User should sit at a meta level above the AI stream reading and sending commands into the game. however that chat should be able to influence the stream logic realtime in the game."
- Owner on naming: "let's split these up and refer to AI-chatter and AI-player."
- Owner on AI-chatter's limits: "AI Chat just does that... it chats and can push suggestions down into the AI-player mid-stream. No other application permissions or functionality should be in scope."
- Owner on promotion: "There is no promotion of any text to do anything in the system. The text is only ever plain text. If a user wants to grab a reply and apply it to settings... they would copy/paste and manually adjust their AI settings. Feel free to write up a help article on this as well to let users know."
- Owner on rule conflicts: "AI chat will not outright contradict the command, but it should give suggestions on how to better set up the filters to get the desired result." and "It only gives back a chat response suggestion and has no ability to affect the configuration itself."
- Owner on pause: "Pause/Resume should live with AI-Player, and not on AI-Chat interface." and "Why not just WAITING ??" and "if I type anything, AutoPilot disengages".
- Owner on the carried risks: "For any High level items, they need to be fixed in this Phase. For anything else, figure out what Phase they would be appropriate to address in, and ensure they are covered."
- Carried from earlier phases: the owner does not want plumbing turned into decisions; pick the simplest thing and move on. Owner-facing text in plain words.

</specifics>

<deferred>
## Deferred Ideas

- **Promotion of chat text into the profile** — withdrawn by the owner, not deferred. Not to be re-proposed.
- **Editing Session Memory by hand** — dropped (D-19); correcting the AI is done by coaching. Not carried to Phase 6 unless the owner asks.
- **Conduct rules and approach guidance editable on the play screen** — considered as a way to keep "without leaving the page"; the owner chose the Settings page and a help article instead.
- **A Copy icon on chat messages, a Clear button on the coaching list, a `#AUTO PAUSE` directive, typed "pause" in chat** — all declined; chat text has no functionality and pause is one AI-player button.
- **Phase 6 note:** the debrief must record the amount of coaching per session (PROJECT.md roadmap-level decision). With this phase's model the natural count is coaching suggestions pushed per session (and possibly owner messages); Phase 6 decides which.
- **Phase 7 note:** the study loader's summary appears in AI-chatter's conversation area; the lesson confirmation it needs is a Phase 7 decision and must respect "chat text has no functionality" or reopen it with the owner.
- **Phase 6 roadmap wording** still needs amending for Historical Memory and quest closure (carried from Phase 4's deferred list).

### Reviewed Todos (not folded)
None: both pending todos were folded in full.

</deferred>

---

*Phase: 05-coaching-channel*
*Context gathered: 2026-09-17*
