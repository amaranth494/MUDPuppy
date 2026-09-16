# Phase 3: One AI Decision - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-09-15
**Phase:** 03-one-ai-decision
**Areas discussed:** Security todo carry-forward, When the AI acts and what it reads, How the decision shows live and after refresh, Model answer failures and env config

---

## Security todo carry-forward

| Option | Description | Selected |
|--------|-------------|----------|
| Fold both | Plan the DR-2-01 log fix as Phase 3 work and schedule the profile cleanup before the Phase 3 security review | ✓ |
| Fold DR-2-01 only | Plan the log fix; leave the profile cleanup as a review-time item | |
| Re-raise at review only | Present both again at the Phase 3 security review | |

**User's choice:** Fold both.

---

## When the AI acts and what it reads

### Trigger

| Option | Description | Selected |
|--------|-------------|----------|
| Immediately on engage | Driver reads the recent text the server holds and decides right away | ✓ |
| On the next game output | Engage arms; first fresh line triggers | |
| On an explicit #AUTO STEP | Manual single-step directive | |

**User's choice:** Immediately on engage.

### Game text window

| Option | Description | Selected |
|--------|-------------|----------|
| Bounded recent window | Rolling server buffer of recent output, server-sized, includes pre-engage text | ✓ (after discussion) |
| Only text since #AUTO ON | Nothing before engage is sent | |
| You decide | Claude picks | |

**User's choice:** Asked to discuss. After the trade-offs (window reach, profile setting vs server constant, ANSI stripping, owner's typed lines present in the window) the owner said: "Go with your leaning, ANSI stripped, include pre-engage text." The owner also asked for a running memory per game profile: immediate buffer for reaction plus a memory file for longer-term goals and strategies.

### Memory tiers (follow-up)

| Option | Description | Selected |
|--------|-------------|----------|
| Two tiers now, writing in Phase 6 | Recent window plus conduct rules and approach guidance; learned notes and AI writes stay in Phase 6 | ✓ |
| Add a read-only learned-notes field now | Schema pull-forward, empty until Phase 6 | |

**User's choice:** Two tiers now, writing in Phase 6.

### After the one decision

| Option | Description | Selected |
|--------|-------------|----------|
| Stays ON and idle | Badge stays ON, nothing more issued until Phase 4; only #AUTO OFF or wheel-grab turns it off | ✓ |
| Disengages with a notice | Lands on OFF after the one command | |
| You decide | | |

**User's choice:** Stays ON and idle.

### Reconnect resume

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, resume fires one decision | Resume treated like engage on fresh post-reconnect text | ✓ |
| No, resume issues nothing in Phase 3 | Only a typed #AUTO ON fires | |
| You decide | | |

**User's choice:** Yes, resume fires one decision.

---

## How the decision shows, live and after refresh

### Live display

| Option | Description | Selected |
|--------|-------------|----------|
| Inline terminal lines | Bracketed coloured lines in the game scroll | |
| A reasoning pane beside the terminal | New side panel; future home of Phase 5 chat | |
| Both | Command line inline, reasoning in a pane | |

**User's choice:** Asked to discuss. Owner's direction: "if you have activated the AI assist from the settings on a profile, you get a pop-up chat interface with the AI model where you can see its reasoning and interact with it during the game." Interaction noted as Phase 5.

### Pane shape (follow-up)

| Option | Description | Selected |
|--------|-------------|----------|
| Floating panel, minimizable | Movable chat window, collapses to a tab | ✓ |
| Docked side pane | Fixed column beside the terminal | |
| You decide | | |

### When shown (follow-up)

| Option | Description | Selected |
|--------|-------------|----------|
| When the connected profile has AI activated | Opens for any policy-accepted profile, ON or OFF | ✓ |
| Only while autopilot is ON | | |
| Opened by the owner | Header button or directive | |

### Pane content (follow-up)

| Option | Description | Selected |
|--------|-------------|----------|
| Reasoning and commands as chat bubbles, no input yet | Decision = reasoning then command; state changes as system lines | ✓ |
| Same, plus a disabled input box | | |
| You decide | | |

### Command echo

| Option | Description | Selected |
|--------|-------------|----------|
| Marked as AI, e.g. [AI> north] | Visibly the AI's line | |
| Same as typed input | | |
| You decide | | |

**User's choice (free text):** "[AI-ASSIST > north] -- and that should substitute for local echo to differentiate between typed commands."

### After refresh

| Option | Description | Selected |
|--------|-------------|----------|
| #AUTO LOG replays the session's decisions | Directive prints stored decisions | |
| Automatic replay on reconnect | Server pushes on re-attach | |
| A history view in the AI Player settings section | | |

**User's choice (free text):** "It needs to dump to a log that is stored on Railway and accessible from a browser tab via URL." Claude proposed Postgres rows plus a sign-in-protected per-session URL, link printed by the panel and #AUTO STATUS; the owner asked to discuss further.

### Stored log (discussion)

Claude laid out rows vs flat file, page contents, how much game text to keep, authorization, and how the owner finds the link. Owner's direction (verbatim in CONTEXT.md specifics): no CLI directive kicks off or prints logging; #AUTO is ON/OFF only with an unknown-option line for anything else; logging is thorough; logs stored by date-time per connection; a button in Settings opens the log listing. Owner then corrected: logging is automatic per game connection (connect to disconnect), not per AI stint; it captures the whole session; a Logs section in Settings with an "Open logging" button that opens a new tab and never navigates the current tab.

### Session unit

| Option | Description | Selected |
|--------|-------------|----------|
| From #AUTO ON to #AUTO OFF | Each engage opens a session row | ✓ (as the AI stint inside the connection log) |
| One per game connection | | (became the transcript unit per the owner's correction) |
| You decide | | |

### Which sessions are logged

| Option | Description | Selected |
|--------|-------------|----------|
| Every connection, every profile | Any saved-profile connection, AI or not; quick connects not logged | ✓ |
| Only profiles with AI activated | | |

### Logs page

| Option | Description | Selected |
|--------|-------------|----------|
| All the owner's games, sessions grouped under each | One page per user | |
| Only the current profile's sessions | Per-profile page | ✓ (free text) |

**User's choice (free text):** "Since Logs are based on the Game Profile it will only show the logs associated with the game profile you clicked the button from. And then you would see a list of date/time log links. Maybe we build a web interface that allows you to highlight them on the left pane and it displays the text on the right."

### #AUTO grammar

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, ON and OFF only | Retires the Phase 2 STATUS line | ✓ |
| Keep STATUS as well | | |

---

## Model answer, failures, and env config

### Reasoning (first pass)

| Option | Description | Selected |
|--------|-------------|----------|
| A short explanation the model writes for the owner | Brief why plus one command | |
| The model's full thinking trace | | |
| Both: short why in the panel, full trace in the log | | |

**User's choice (free text):** "None. The log only tracks what was sent to the game and what was sent back from the game. But it will differentiate the user typing from the AI typing." Claude flagged the conflict with REQ-reasoning-visibility ("stored in the session log") and proposed storing reasoning per decision outside the transcript.

### Reasoning kept (follow-up)

| Option | Description | Selected |
|--------|-------------|----------|
| Stored per decision, shown in the panel, not in the transcript | Satisfies the design as written | ✓ |
| Shown live only, never stored | Would amend design D3 | |

### Panel text (follow-up)

| Option | Description | Selected |
|--------|-------------|----------|
| A short why the model writes for the owner | | ✓ |
| The model's full thinking trace | | |
| You decide | | |

### Bad answer

| Option | Description | Selected |
|--------|-------------|----------|
| Notice and disengage on the first failure | Nothing sent; informative error; OFF; play continues | ✓ |
| Retry once, then disengage | | |
| You decide | | |

### Missing env

| Option | Description | Selected |
|--------|-------------|----------|
| #AUTO ON is refused with a clear message | Server starts normally | ✓ |
| Server refuses to start | Fatal at boot | |

### Default model (first pass)

| Option | Description | Selected |
|--------|-------------|----------|
| Fast, low-cost tier | Flash-class | |
| Strongest reasoning tier | Pro-class | |
| You decide | | |

**User's choice (free text):** "For this test we were using Gemini free tier, but I would like the option for adding more models using API keys/endpoints."

### Model list (follow-up)

| Option | Description | Selected |
|--------|-------------|----------|
| Named entries with endpoint and key; Gemini only for now | Env registry; other vendors plug in later | ✓ |
| Several Gemini models, one key | | |
| Other vendors in Phase 3 too | Flagged as scope growth | |

---

## Claude's Discretion

Window size and buffer implementation; ANSI stripping location; table shapes and migration numbers; websocket message type for decisions and the panel reload; how a pass-through game command is dispatched through the ICM dispatcher in the automation context and how the frontend adapter surfaces errors; registry env format and Gemini client implementation; prompt wording and structured answer format; log page delivery and transcript line marking; panel visuals and the exact wording of system lines and the unknown-option line; where the DR-2-01 fix lands; `[AI-PLAYER]` log line format per decision.

## Deferred Ideas

Coaching input in the panel (Phase 5); learned notes and AI-written memory (Phase 6); non-Gemini vendors; per-profile window tuning; cross-profile log listing, share links, search or export; security review items (full transcripts for all profile connections vs policy data handling, staging Gemini key and free-tier limits, log page authorization, driver bypassing the wheel-grab check, closure of DR-2-01 and DR-2-02).
