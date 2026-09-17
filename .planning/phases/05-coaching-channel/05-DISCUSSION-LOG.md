# Phase 5: Coaching Channel - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-09-17
**Phase:** 5-Coaching Channel
**Areas discussed:** Fold todos, Where the chat lives, How coaching sticks, Pause and resume, Making guidance permanent, Asking the AI and its memory

Every question also offered "I need to discuss this further before we move on."

---

## Fold todos

| Option | Description | Selected |
|--------|-------------|----------|
| Security carry-forward | The four risks deferred from the Phase 4 review (DR-4-01 to DR-4-04) | |
| Log page sign-in check | The owner's condition on AR-4-08 | |

**User's choice:** Free text: "For any High level items, they need to be fixed in this Phase. For anything else, figure out what Phase they would be appropriate to address in, and ensure they are covered."
**Notes:** DR-4-01, DR-4-02, DR-4-03 (all high) fixed in Phase 5; this supersedes the owner's earlier "later phases" notes on DR-4-01 and DR-4-03. DR-4-04 (low) and the log page check were placed in Phase 5 by Claude (small, and the owner's earlier note on DR-4-04 said next phase).

---

## Where the chat lives

| Option | Description | Selected |
|--------|-------------|----------|
| Grow the AI Assist panel | Message box added to the existing floating panel | ✓ |
| Docked pane beside the terminal | Real side column replacing the floating panel | |
| Separate chat pane, keep the panel | Second pane just for chat | |

| Option | Description | Selected |
|--------|-------------|----------|
| Whenever the panel shows | Usable in any autopilot state | ✓ |
| Only while autopilot is on or paused | Greyed out otherwise | |

| Option | Description | Selected |
|--------|-------------|----------|
| Same stream, clearly yours | Owner messages in time order among decisions | ✓ (superseded) |
| Chat bubbles, left and right | Messenger style | |
| You decide | | |

Persistence question (Panel and log page / Panel only): answered with free text instead: "Chat between User and AI should be separatee from the thinking stream and commands sent to the MUD."

**Clarification (plain text):** Claude offered two readings (two sections in the panel, or separate only in the records). Owner: "Chat between AI and User should sit at a meta level above the AI stream reading and sending commands into the game. however that chat should be able to influence the stream logic realtime in the game."

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, that's it | Two-level picture confirmed | ✓ |
| Close, but let me correct it | | |

| Option | Description | Selected |
|--------|-------------|----------|
| Split: stream on top, chat below | Both visible at once | ✓ |
| Two tabs: Activity and Chat | One at a time | |
| You decide | | |

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, a small marker | "Coaching received" line in the thinking stream | ✓ |
| No marker | | |

**Notes:** The roadmap's "one stream" criterion was amended to "one panel, two views". Mid-discussion the owner named the levels: "let's split these up and refer to AI-chatter and AI-player."

---

## How coaching sticks

| Option | Description | Selected |
|--------|-------------|----------|
| Until you sign out or remove it | Standing note for the login, like Session Memory | ✓ |
| Recent messages only | Older ones fall off | |
| Until autopilot goes off | Per stint | |

| Option | Description | Selected |
|--------|-------------|----------|
| Always a short reply | One model call each, counts against the cap | ✓ |
| Reply only to questions | Instructions get a plain acknowledgement | |

| Option | Description | Selected |
|--------|-------------|----------|
| Works the same, ready for next engage | Replies while off; coaching in place at next #AUTO ON | ✓ |
| Saved, but no reply until engaged | | |

| Option | Description | Selected |
|--------|-------------|----------|
| Profile rules win, and the AI says so | | |
| Live coaching wins over conduct rules | | |

**User's choice (conflicts):** Free text: "AI chat will not outright contradict the command, but it should give suggestions on how to better set up the filters to get the desired result." On Claude's read-back: "It only gives back a chat response suggestion and has no ability to affect the configuration itself."
**Notes:** Rules keep holding in the game; the reply suggests wording; the chat AI can change nothing.

---

## Pause and resume

| Option | Description | Selected |
|--------|-------------|----------|
| Button and plain words in chat | | ✓ (superseded) |
| Button only | | |
| Also a #AUTO PAUSE directive | | |

| Option | Description | Selected |
|--------|-------------|----------|
| Reads only, thinks on resume | No model calls while paused; reassess on resume | ✓ |
| Keeps thinking out loud, sends nothing | | |

Typing while paused (stay paused / turns autopilot off): free text: "if I type anything, AutoPilot disengages".

Badge while paused (fourth PAUSED state / keep ON): free text: "Why not just WAITING ??" Claude explained the catch (a pause must never resume by itself) and proposed waiting reasons.

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, WAITING with a reason | Three badge states; reason in the panel; resumes only when every reason clears | ✓ |
| Separate PAUSED badge after all | | |

Later, after the owner limited AI-chatter to chat and suggestions, the control question was re-asked (button only beside the message box / button and typed words recognised by the panel). Free text: "Pause/Resume should live with AI-Player, and not on AI-Chat interface".
**Notes:** Final: one button in the AI-player part of the panel; nothing in chat pauses or resumes.

---

## Making guidance permanent

Options presented: what can be promoted (own messages and AI suggestions / own messages only); promote flow (small editor / one click).

**User's choice:** Free text: "no, there's no functionality in the text... it's just text back and forth. It can give suggestions and the player is free to copy/paste from the chat into other settings." and "Discuss first" on the flow.

**Discussion (plain text):** Claude pointed out the design's "without leaving the page" wording and offered: (1) conduct rules and approach guidance editable on the play screen, (2) Settings only and amend the requirement, (3) option 1 plus a Copy icon. Owner: "There is no promotion of any text to do anything in the system. The text is only ever plain text. If a user wants to grab a reply and apply it to settings... they would copy/paste and manually adjust their AI settings. Feel free to write up a help article on this as well to let users know. AI Chat just does that... it chats and can push suggestions down into the AI-player mid-stream. No other application permissions or functionality should be in scope. Clear?"

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, amend them | Design doc, requirements and roadmap reworded now | ✓ |
| Record the decision, amend later | | |

**Notes:** Promotion withdrawn. Help article added to scope. Amendments made in the same commit as this log.

---

## Asking the AI and its memory

| Option | Description | Selected |
|--------|-------------|----------|
| Everything AI-player knows | Goal, profile text, decisions, game text, Session and Quest Memory | ✓ |
| Memory and decisions, not the game text | | |

| Option | Description | Selected |
|--------|-------------|----------|
| Drop it; correct the AI by coaching | Session Memory stays read-only | ✓ |
| Keep it as an AI-player feature | | |
| Push it to a later phase | | |

| Option | Description | Selected |
|--------|-------------|----------|
| Yes: shown in the reply and in a read-only list | Quoted line plus "Coaching in effect" list | ✓ |
| In the reply only | | |

| Option | Description | Selected |
|--------|-------------|----------|
| Tell AI-chatter in words | It withdraws the line and confirms | ✓ |
| Words, plus a 'clear all' button | | |

---

## Claude's Discretion

The AI-chatter injection channel's handling (required, flagged for the security review); the structured answer shape and model choice for AI-chatter; cap behaviour for chat; storage, migrations, websocket and REST shapes; whether the log page shows the conversation; waiting-reason mechanics and log lines; panel split proportions and all notice wording; the rate-limit setting's unit, default and bounds; evidence form for the pause criterion.

## Deferred Ideas

Promotion (withdrawn, not deferred); Session Memory hand-editing (dropped); profile text editable on the play screen, Copy icon, Clear button, `#AUTO PAUSE`, typed "pause" (all declined); Phase 6 note on what counts as "coaching amount"; Phase 7 note on lesson confirmation versus "chat text has no functionality"; Phase 6 roadmap wording for Historical Memory (carried).
