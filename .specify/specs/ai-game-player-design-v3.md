# AI Game Player for MUDPuppy — Design Document, Version 3

Date: 2026-09-14
Branch: `ai-player` (created from `staging`) in github.com/amaranth494/MudPuppy
Environments: all development and testing run against the mudpuppy staging environment on Railway. Production is untouched until final acceptance.
Foundation: existing MUDPuppy infrastructure is the reference and the base. Connection profiles, server-held telnet connections, the Internal Command Module's automation context and safety limits, the browser terminal, Postgres and Redis.

## Intended goal

Extend MUDPuppy so a game character can be driven either by its human owner or by a reasoning AI, switched as easily as engaging and disengaging a car's autopilot. The AI takes instructions on how to approach a game, accepts a small achievable goal per session, plays toward it using a language model (Gemini) rather than scripted triggers, and gets measurably better over time using the game's own progression numbers. Everything game-specific lives on the connection profile, so the same engine adapts to Alter Aeon, the owner's own MUD, or any other text game, with no game knowledge built into the plumbing and no intent detection in the engine.

## Definition of complete

The project is complete when all of the following are true, demonstrated on Alter Aeon from the staging environment:

1. The owner can create a game profile, accept the Safety and Abuse policy on it, and hand-play the character normally. The AI cannot be engaged on any profile that has not accepted the current policy version.
2. With autopilot engaged, the AI plays continuously toward the session goal the owner set, and every decision it makes is visible with its reasoning as it happens.
3. Typing any game command instantly disengages autopilot and the command goes through. Re-engaging picks up cleanly from the current game situation.
4. The owner can coach the AI from a chat pane while it plays, see the guidance take effect on the next decision, and promote a piece of guidance into the profile permanently.
5. Every session ends with a recorded debrief and updated progression tally, and across at least three consecutive goal sessions the tally shows improvement while required coaching declines.
6. All mechanical safety limits hold under test: call cap, no AI-initiated reconnect, disengage on repeated errors or disconnect, conservative defaults when profile settings are blank.

## Deliverables

### D1: Profile foundation and policy gate

The connection profile becomes the single per-game home for the AI. Add to profiles: conduct rules (text, handed to the model), approach guidance (text), and AI settings (model names, call cap per session, disengage thresholds). Add the Safety and Abuse policy acceptance flow. Reconnect is not an AI setting: it remains governed by the connection profile's existing reconnect toggle, which is independent of the AI and unchanged by this work.

Accepted when:
- A profile stores and returns all new fields; blank mechanical settings resolve to conservative engine defaults (capped calls, default disengage thresholds).
- Attempting to engage the AI on a profile without a recorded acceptance of the current policy version is refused with a clear message, and the policy is presented for acceptance.
- Acceptance is recorded per profile with timestamp and policy version; changing the policy version requires re-acceptance.

### D2: Autopilot switch

The engage and disengage mechanics, independent of any AI intelligence. `#AUTO ON` and `#AUTO OFF` in the existing `#` directive grammar, a visible status indicator in the play screen, and the wheel-grab rule.

Accepted when:
- `#AUTO ON` engages only when the profile passes the D1 gate; the indicator always matches the true state.
- Any game command typed while engaged disengages before the command is sent, with no lost keystrokes.
- Disconnect while engaged lands the AI on disengaged. The AI never initiates a reconnect; whether the connection itself reconnects is decided solely by the connection profile's reconnect toggle, and if the connection does come back the AI stays disengaged until the owner re-engages it.

### D3: One AI decision

The thinnest slice through the whole stack: Gemini connected, one decision made, one command issued. This proves the architecture before any looping.

Accepted when:
- With autopilot engaged, the driver reads the current game text, sends one request to Gemini with the profile's conduct rules and approach guidance included, and issues the returned command through the automation context.
- The decision and the model's reasoning appear in the play screen as they happen and are stored in the session log.
- Model names and API key come from environment configuration on staging; nothing is hard-coded.

### D4: Continuous play

The full loop: the owner sets a session goal, the AI plays toward it until stopped, within the safety limits.

Accepted when:
- The loop runs read, decide, act continuously against a live game, pacing itself to the game's turn rhythm rather than flooding it.
- The session call cap halts the loop with a visible notice when reached; repeated errors or malformed model output disengage with a visible notice.
- Re-engagement after manual driving demonstrably reassesses the situation rather than resuming a stale plan.

### D5: Coaching channel

The live collaboration layer: a chat pane beside the terminal.

Accepted when:
- A chat message sent while the AI plays is reflected in its next decision, and this is verifiable from the logged reasoning.
- Pause and resume work from chat; pausing stops commands but keeps reading the game.
- The owner can promote a chat instruction into the profile's conduct rules or approach guidance without leaving the page, and it persists across sessions.

### D6: Measurement and memory

The learning substrate: progression tally, session debriefs, learned notes.

Accepted when:
- During play, the driver samples the game's status commands on an interval and stores the parsed numbers with timestamps against the profile and character.
- Ending a session produces a stored debrief: goal, outcome, tally movement, what was learned, what went wrong; new lessons are appended to the profile's learned notes.
- The next session's decisions include the learned notes, and manual-driving stretches are captured and available to the debrief as demonstrations.

### D7: Study loader

Learning from material outside live play.

Accepted when:
- The owner can paste text or upload a file (transcript, help file, walkthrough) against a profile; the AI summarizes its conclusions in the chat pane.
- Lessons the owner confirms are written into the learned notes; nothing is written without confirmation.

### D8: Acceptance run on Alter Aeon

The proof, run from staging under the owner's supervision.

Accepted when:
- Reconnaissance first: on a fresh profile, with the owner coaching, the AI's opening sessions discover and record how Alter Aeon measures progress (its status commands and numbers) into notes and configuration.
- Then goal sessions: the owner hand-plays a character through creation and the mud school, engages autopilot outside restricted areas, and the AI completes small session goals honestly measured by the tally.
- The Definition of complete, items 1 through 6, all hold during these sessions.

## Order and dependencies

D1 through D4 are strictly sequential; each is a working, testable increment. D5 and D6 both build on D4 and can proceed in either order, though D5 first matches the collaboration-heavy early phase. D7 needs D6's notes. D8 needs everything and is the finish line.

## Out of scope for this version

Playing games that prohibit automation, multiple AI characters at once, autonomous unattended operation, production deployment, and any engine-level game understanding (area detection, behavior analysis). The model reasons about the game; the engine enforces numbers, flags, and the policy gate.

## Related documents

- Safety and Abuse Policy, version 1.0: `.specify/specs/safety-and-abuse-policy-v1.md`. Per-profile acceptance required before AI engagement; referenced by D1.
