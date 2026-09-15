# Requirements

Synthesized by gsd-doc-synthesizer on 2026-09-14 (MODE: new).

Provenance note: no PRD-typed documents were ingested. The design document is typed SPEC (manifest,
precedence 0) but its Deliverables section carries numbered acceptance criteria. Those criteria are
extracted here verbatim so the roadmapper has requirement-level inputs. Acceptance criteria are quoted
as written; nothing has been merged or paraphrased into a combined criterion. Policy-derived system
behaviors are recorded in constraints.md and cross-referenced by ID below rather than duplicated.

Delivery ordering (from the source): D1 -> D2 -> D3 -> D4 strictly sequential; D5 and D6 both depend on
D4 in either order (D5 first recommended); D7 depends on D6; D8 depends on everything.

---

## D1: Profile foundation and policy gate

### REQ-profile-ai-fields
- source: .specify/specs/ai-game-player-design-v3.md (D1)
- description: The connection profile becomes the single per-game home for the AI. Add conduct rules (text, handed to the model), approach guidance (text), and AI settings (model names, call cap per session, disengage thresholds). Reconnect is not an AI setting; it stays governed by the connection profile's existing reconnect toggle, independent of the AI and unchanged by this work.
- acceptance: "A profile stores and returns all new fields; blank mechanical settings resolve to conservative engine defaults (capped calls, default disengage thresholds)."
- scope: connection profile schema and defaults
- see also: CON-profile-schema, CON-conservative-defaults (reconnect-flag ambiguity resolved by owner 2026-09-14, see INGEST-CONFLICTS.md INFO)

### REQ-policy-gate
- source: .specify/specs/ai-game-player-design-v3.md (D1); .specify/specs/safety-and-abuse-policy-v1.md (preamble)
- description: Add the Safety and Abuse policy acceptance flow. The AI cannot be engaged on any profile that has not accepted the current policy version.
- acceptance:
  - "Attempting to engage the AI on a profile without a recorded acceptance of the current policy version is refused with a clear message, and the policy is presented for acceptance."
  - "Acceptance is recorded per profile with timestamp and policy version; changing the policy version requires re-acceptance."
- scope: policy gate, acceptance record
- see also: CON-policy-acceptance-record

---

## D2: Autopilot switch

### REQ-autopilot-directives
- source: .specify/specs/ai-game-player-design-v3.md (D2)
- description: `#AUTO ON` and `#AUTO OFF` in the existing `#` directive grammar, plus a visible status indicator in the play screen. Engage/disengage mechanics are independent of any AI intelligence.
- acceptance: "`#AUTO ON` engages only when the profile passes the D1 gate; the indicator always matches the true state."
- scope: directive grammar, play-screen status indicator
- see also: CON-directive-grammar

### REQ-wheel-grab
- source: .specify/specs/ai-game-player-design-v3.md (D2; Definition of complete item 3)
- description: Typing any game command while engaged instantly disengages autopilot and the command goes through.
- acceptance: "Any game command typed while engaged disengages before the command is sent, with no lost keystrokes."
- scope: input handling while engaged

### REQ-no-auto-reconnect
- source: .specify/specs/ai-game-player-design-v3.md (D2; Definition of complete item 6); .specify/specs/safety-and-abuse-policy-v1.md (section 2)
- description: A disconnect while engaged lands the AI on disengaged. The AI never initiates a reconnect; whether the connection itself reconnects is decided solely by the connection profile's reconnect toggle, and if the connection comes back the AI stays disengaged until the owner re-engages it. The policy states the AI "will not reconnect on its own" if disconnected or kicked while engaged.
- acceptance: "Disconnect while engaged lands the AI on disengaged. The AI never initiates a reconnect; whether the connection itself reconnects is decided solely by the connection profile's reconnect toggle, and if the connection does come back the AI stays disengaged until the owner re-engages it."
- scope: disconnect handling
- see also: CON-policy-supervision (reconnect ownership settled by owner 2026-09-14, see INGEST-CONFLICTS.md INFO)

---

## D3: One AI decision

### REQ-single-decision
- source: .specify/specs/ai-game-player-design-v3.md (D3)
- description: The thinnest slice through the stack: Gemini connected, one decision made, one command issued.
- acceptance: "With autopilot engaged, the driver reads the current game text, sends one request to Gemini with the profile's conduct rules and approach guidance included, and issues the returned command through the automation context."
- scope: driver, model call, ICM automation context
- see also: CON-command-path

### REQ-reasoning-visibility
- source: .specify/specs/ai-game-player-design-v3.md (D3; Definition of complete item 2)
- description: Every decision and its reasoning is visible as it happens and persisted.
- acceptance: "The decision and the model's reasoning appear in the play screen as they happen and are stored in the session log."
- scope: play screen, session log

### REQ-env-config
- source: .specify/specs/ai-game-player-design-v3.md (D3)
- description: Model configuration and secrets come from the environment.
- acceptance: "Model names and API key come from environment configuration on staging; nothing is hard-coded."
- scope: configuration
- see also: CON-model-config

---

## D4: Continuous play

### REQ-continuous-loop
- source: .specify/specs/ai-game-player-design-v3.md (D4; Definition of complete item 2)
- description: The owner sets a session goal; the AI plays toward it until stopped, within the safety limits.
- acceptance: "The loop runs read, decide, act continuously against a live game, pacing itself to the game's turn rhythm rather than flooding it."
- scope: decision loop, pacing
- see also: CON-policy-rate-limits

### REQ-call-cap-and-error-disengage
- source: .specify/specs/ai-game-player-design-v3.md (D4; Definition of complete item 6)
- description: Mechanical halts and disengages with visible notices.
- acceptance: "The session call cap halts the loop with a visible notice when reached; repeated errors or malformed model output disengage with a visible notice."
- scope: safety limits
- see also: CON-policy-cost, CON-policy-rate-limits

### REQ-reengage-reassess
- source: .specify/specs/ai-game-player-design-v3.md (D4; Definition of complete item 3)
- description: Re-engaging picks up cleanly from the current game situation.
- acceptance: "Re-engagement after manual driving demonstrably reassesses the situation rather than resuming a stale plan."
- scope: state handling on re-engage

---

## D5: Coaching channel

### REQ-coaching-chat
- source: .specify/specs/ai-game-player-design-v3.md (D5; Definition of complete item 4)
- description: A chat pane beside the terminal for live collaboration.
- acceptance: "A chat message sent while the AI plays is reflected in its next decision, and this is verifiable from the logged reasoning."
- scope: chat pane, prompt assembly, logged reasoning

### REQ-pause-resume
- source: .specify/specs/ai-game-player-design-v3.md (D5)
- description: Pause and resume control from chat.
- acceptance: "Pause and resume work from chat; pausing stops commands but keeps reading the game."
- scope: loop control

### REQ-promote-guidance
- source: .specify/specs/ai-game-player-design-v3.md (D5; Definition of complete item 4)
- description: Promote a piece of chat guidance into the profile permanently.
- acceptance: "The owner can promote a chat instruction into the profile's conduct rules or approach guidance without leaving the page, and it persists across sessions."
- scope: profile editing from chat

---

## D6: Measurement and memory

### REQ-progression-tally
- source: .specify/specs/ai-game-player-design-v3.md (D6)
- description: The driver measures progress using the game's own numbers.
- acceptance: "During play, the driver samples the game's status commands on an interval and stores the parsed numbers with timestamps against the profile and character."
- scope: tally sampling and storage

### REQ-session-debrief
- source: .specify/specs/ai-game-player-design-v3.md (D6; Definition of complete item 5)
- description: Every session ends with a recorded debrief and updated learned notes.
- acceptance: "Ending a session produces a stored debrief: goal, outcome, tally movement, what was learned, what went wrong; new lessons are appended to the profile's learned notes."
- scope: debrief, learned notes

### REQ-memory-carryover
- source: .specify/specs/ai-game-player-design-v3.md (D6)
- description: Learned notes feed the next session; manual play is captured as demonstration material.
- acceptance: "The next session's decisions include the learned notes, and manual-driving stretches are captured and available to the debrief as demonstrations."
- scope: prompt assembly, session capture

---

## D7: Study loader

### REQ-study-loader
- source: .specify/specs/ai-game-player-design-v3.md (D7)
- description: Learning from material outside live play.
- acceptance: "The owner can paste text or upload a file (transcript, help file, walkthrough) against a profile; the AI summarizes its conclusions in the chat pane."
- scope: upload/paste, summarization in chat

### REQ-study-confirmation
- source: .specify/specs/ai-game-player-design-v3.md (D7)
- description: Nothing enters learned notes without owner confirmation.
- acceptance: "Lessons the owner confirms are written into the learned notes; nothing is written without confirmation."
- scope: learned notes write gate

---

## D8: Acceptance run on Alter Aeon

### REQ-acceptance-reconnaissance
- source: .specify/specs/ai-game-player-design-v3.md (D8)
- description: Opening sessions discover how the game measures progress.
- acceptance: "Reconnaissance first: on a fresh profile, with the owner coaching, the AI's opening sessions discover and record how Alter Aeon measures progress (its status commands and numbers) into notes and configuration."
- scope: acceptance run, phase 1

### REQ-acceptance-goal-sessions
- source: .specify/specs/ai-game-player-design-v3.md (D8)
- description: Goal sessions after reconnaissance.
- acceptance: "Then goal sessions: the owner hand-plays a character through creation and the mud school, engages autopilot outside restricted areas, and the AI completes small session goals honestly measured by the tally."
- scope: acceptance run, phase 2
- see also: CON-policy-game-rules (restricted areas are conduct-rule conditions)

### REQ-acceptance-definition-holds
- source: .specify/specs/ai-game-player-design-v3.md (D8; Definition of complete)
- description: The whole Definition of complete holds during the acceptance sessions.
- acceptance: "The Definition of complete, items 1 through 6, all hold during these sessions."
- scope: acceptance run, exit criterion

---

## Cross-cutting completion criteria (Definition of complete)

### REQ-doc-hand-play-and-gate
- source: .specify/specs/ai-game-player-design-v3.md (Definition of complete item 1)
- acceptance: "The owner can create a game profile, accept the Safety and Abuse policy on it, and hand-play the character normally. The AI cannot be engaged on any profile that has not accepted the current policy version."
- maps to: REQ-policy-gate, REQ-autopilot-directives

### REQ-doc-continuous-visible-play
- source: .specify/specs/ai-game-player-design-v3.md (Definition of complete item 2)
- acceptance: "With autopilot engaged, the AI plays continuously toward the session goal the owner set, and every decision it makes is visible with its reasoning as it happens."
- maps to: REQ-continuous-loop, REQ-reasoning-visibility

### REQ-doc-wheel-grab-and-reengage
- source: .specify/specs/ai-game-player-design-v3.md (Definition of complete item 3)
- acceptance: "Typing any game command instantly disengages autopilot and the command goes through. Re-engaging picks up cleanly from the current game situation."
- maps to: REQ-wheel-grab, REQ-reengage-reassess

### REQ-doc-coaching
- source: .specify/specs/ai-game-player-design-v3.md (Definition of complete item 4)
- acceptance: "The owner can coach the AI from a chat pane while it plays, see the guidance take effect on the next decision, and promote a piece of guidance into the profile permanently."
- maps to: REQ-coaching-chat, REQ-promote-guidance

### REQ-improvement-trend
- source: .specify/specs/ai-game-player-design-v3.md (Definition of complete item 5)
- acceptance: "Every session ends with a recorded debrief and updated progression tally, and across at least three consecutive goal sessions the tally shows improvement while required coaching declines."
- maps to: REQ-session-debrief, REQ-progression-tally; adds the three-session improvement trend as its own measurable criterion

### REQ-safety-limits-hold
- source: .specify/specs/ai-game-player-design-v3.md (Definition of complete item 6)
- acceptance: "All mechanical safety limits hold under test: call cap, no AI-initiated reconnect, disengage on repeated errors or disconnect, conservative defaults when profile settings are blank."
- maps to: REQ-call-cap-and-error-disengage, REQ-no-auto-reconnect, REQ-profile-ai-fields

---

## Requirement ID index (28)

REQ-profile-ai-fields, REQ-policy-gate, REQ-autopilot-directives, REQ-wheel-grab, REQ-no-auto-reconnect,
REQ-single-decision, REQ-reasoning-visibility, REQ-env-config, REQ-continuous-loop,
REQ-call-cap-and-error-disengage, REQ-reengage-reassess, REQ-coaching-chat, REQ-pause-resume,
REQ-promote-guidance, REQ-progression-tally, REQ-session-debrief, REQ-memory-carryover, REQ-study-loader,
REQ-study-confirmation, REQ-acceptance-reconnaissance, REQ-acceptance-goal-sessions,
REQ-acceptance-definition-holds, REQ-doc-hand-play-and-gate, REQ-doc-continuous-visible-play,
REQ-doc-wheel-grab-and-reengage, REQ-doc-coaching, REQ-improvement-trend, REQ-safety-limits-hold
