# Requirements: AI Game Player for MUDPuppy

**Defined:** 2026-09-14
**Core Value:** The owner can hand the wheel to the AI and take it back instantly, always seeing what the AI is doing and why, with every mechanical safety limit holding, and the AI getting measurably better session over session by the game's own numbers.

Requirement IDs and acceptance text come verbatim from `.planning/intel/requirements.md`, which extracted them from the design's D1-D8 "Accepted when" bullets and Definition of complete. Sources: `.specify/specs/ai-game-player-design-v3.md` (precedence 0) and `.specify/specs/safety-and-abuse-policy-v1.md` (precedence 1). Categories are the design's deliverables.

## v1 Requirements

Delivery ordering (from the source): D1 -> D2 -> D3 -> D4 strictly sequential; D5 and D6 both depend on D4 (D5 first); D7 depends on D6; D8 depends on everything.

### D1: Profile foundation and policy gate

- [x] **REQ-profile-ai-fields**: The connection profile stores conduct rules (text, handed to the model), approach guidance (text), and AI settings (model name, call cap per session, disengage threshold). Reconnect is not an AI setting. Acceptance: "A profile stores and returns all new fields. A blank model name means the server's configured default; a blank call cap means no cap; a blank disengage threshold means the engine's built-in error handling applies: any AI failure produces an informative error in the play screen, the AI disengages, and regular play continues. The AI must never crash the server or the session."
- [x] **REQ-policy-gate**: The Safety and Abuse policy acceptance flow. Acceptance: "The first time the owner opens the AI configuration for a profile, the policy is presented and must be accepted before AI settings can be edited or the AI engaged; attempting to engage the AI on a profile without a recorded acceptance is refused with a clear message." and "Acceptance is recorded once per profile with a timestamp and the policy version accepted. It does not expire and a later policy change does not require re-acceptance. Deleting the profile discards the acceptance."

### D2: Autopilot switch

- [x] **REQ-autopilot-directives**: `#AUTO ON` and `#AUTO OFF` in the existing `#` directive grammar, plus a visible status indicator in the play screen; engage/disengage mechanics independent of any AI intelligence. Acceptance: "`#AUTO ON` engages only when the profile passes the D1 gate; the indicator always matches the true state."
- [x] **REQ-wheel-grab**: Acceptance: "Any game command typed while engaged disengages before the command is sent, with no lost keystrokes."
- [x] **REQ-no-auto-reconnect**: Acceptance (design D2 amended by the owner 2026-09-15): "A disconnect while engaged does not turn autopilot off: it enters a waiting state, issues nothing while disconnected, and resumes on its own when the connection returns. The AI never initiates a reconnect; whether and how the connection itself reconnects is decided solely by the connection profile (today, the owner reconnecting by hand). Only `#AUTO OFF` moves autopilot to off, including while waiting."

### D3: One AI decision

- [x] **REQ-single-decision**: Acceptance: "With autopilot engaged, the driver reads the current game text, sends one request to Gemini with the profile's conduct rules and approach guidance included, and issues the returned command through the automation context."
- [x] **REQ-reasoning-visibility**: Acceptance: "The decision and the model's reasoning appear in the play screen as they happen and are stored in the session log."
- [x] **REQ-env-config**: Acceptance: "Model names and API key come from environment configuration on staging; nothing is hard-coded."

### D4: Continuous play

- [x] **REQ-continuous-loop**: The owner sets a session goal; the AI plays toward it until stopped, within the safety limits. Acceptance: "The loop runs read, decide, act continuously against a live game, pacing itself to the game's turn rhythm rather than flooding it."
- [ ] **REQ-call-cap-and-error-disengage**: Acceptance: "The session call cap halts the loop with a visible notice when reached; repeated errors or malformed model output disengage with a visible notice."
- [x] **REQ-reengage-reassess**: Acceptance: "Re-engagement after manual driving demonstrably reassesses the situation rather than resuming a stale plan."

### D5: Coaching channel

- [ ] **REQ-coaching-chat**: A chat pane beside the terminal. Acceptance: "A chat message sent while the AI plays is reflected in its next decision, and this is verifiable from the logged reasoning."
- [ ] **REQ-pause-resume**: Acceptance: "Pause and resume work from chat; pausing stops commands but keeps reading the game."
- [ ] **REQ-promote-guidance**: Acceptance: "The owner can promote a chat instruction into the profile's conduct rules or approach guidance without leaving the page, and it persists across sessions."

### D6: Measurement and memory

- [ ] **REQ-progression-tally**: Acceptance: "During play, the driver samples the game's status commands on an interval and stores the parsed numbers with timestamps against the profile and character."
- [ ] **REQ-session-debrief**: Acceptance: "Ending a session produces a stored debrief: goal, outcome, tally movement, what was learned, what went wrong; new lessons are appended to the profile's learned notes."
- [ ] **REQ-memory-carryover**: Acceptance: "The next session's decisions include the learned notes, and manual-driving stretches are captured and available to the debrief as demonstrations."

### D7: Study loader

- [ ] **REQ-study-loader**: Acceptance: "The owner can paste text or upload a file (transcript, help file, walkthrough) against a profile; the AI summarizes its conclusions in the chat pane."
- [ ] **REQ-study-confirmation**: Acceptance: "Lessons the owner confirms are written into the learned notes; nothing is written without confirmation."

### D8: Acceptance run on Alter Aeon

- [ ] **REQ-acceptance-reconnaissance**: Acceptance: "Reconnaissance first: on a fresh profile, with the owner coaching, the AI's opening sessions discover and record how Alter Aeon measures progress (its status commands and numbers) into notes and configuration."
- [ ] **REQ-acceptance-goal-sessions**: Acceptance: "Then goal sessions: the owner hand-plays a character through creation and the mud school, engages autopilot outside restricted areas, and the AI completes small session goals honestly measured by the tally."
- [ ] **REQ-acceptance-definition-holds**: Acceptance: "The Definition of complete, items 1 through 6, all hold during these sessions."

### Definition of complete (cross-cutting)

- [x] **REQ-doc-hand-play-and-gate** (item 1): "The owner can create a game profile, accept the Safety and Abuse policy on it, and hand-play the character normally. The AI cannot be configured or engaged on any profile that has not accepted the policy." Maps to REQ-policy-gate, REQ-autopilot-directives.
- [ ] **REQ-doc-continuous-visible-play** (item 2): "With autopilot engaged, the AI plays continuously toward the session goal the owner set, and every decision it makes is visible with its reasoning as it happens." Maps to REQ-continuous-loop, REQ-reasoning-visibility.
- [x] **REQ-doc-wheel-grab-and-reengage** (item 3): "Typing any game command instantly disengages autopilot and the command goes through. Re-engaging picks up cleanly from the current game situation." Maps to REQ-wheel-grab, REQ-reengage-reassess.
- [ ] **REQ-doc-coaching** (item 4): "The owner can coach the AI from a chat pane while it plays, see the guidance take effect on the next decision, and promote a piece of guidance into the profile permanently." Maps to REQ-coaching-chat, REQ-promote-guidance.
- [ ] **REQ-improvement-trend** (item 5): "Every session ends with a recorded debrief and updated progression tally, and across at least three consecutive goal sessions the tally shows improvement while required coaching declines." Maps to REQ-session-debrief, REQ-progression-tally; adds the three-session trend as its own measurable criterion.
- [ ] **REQ-safety-limits-hold** (item 6, amended 2026-09-15): "All mechanical safety limits hold under test: call cap when one is set, no AI-initiated reconnect, disengage on repeated errors, no commands issued while disconnected (autopilot waits and resumes only once the connection returns), and safe behaviour when profile settings are blank (no cap, informative failure, no crash, regular play unaffected)." Maps to REQ-call-cap-and-error-disengage, REQ-no-auto-reconnect, REQ-profile-ai-fields.

## v2 Requirements

Deferred; tracked but not in the current roadmap.

### Policy enforcement

- **ENF-01**: Disable AI features on a profile or account after a policy violation (policy v1.0 section 7). No deliverable in design v3 implements this; recorded as a known omission (CON-policy-enforcement).

### Connection reconnect

- **CONN-01**: A connection-profile reconnect toggle. The owner's decision treats reconnect as the connection profile's business, unchanged by this work; no such toggle exists in the profile schema today. Outside the AI scope.

## Out of Scope

| Feature | Reason |
|---------|--------|
| Playing games that prohibit automation | Design v3 out of scope; policy section 1 |
| Multiple AI characters at once | Design v3 out of scope; stricter than policy section 4 and the stricter rule applies |
| Autonomous unattended operation | Design v3 out of scope; policy section 2; server-side placement does not license it (locked) |
| Production deployment | Staging only until D8 acceptance |
| Engine-level game understanding (area detection, behaviour analysis, intent detection) | The model reasons about the game; the engine enforces numbers, flags, and the gate |
| Reading or citing prior `.specify/` specs (SP00-SP06, PR01, PR02, constitution) | Excluded by owner; product taken as-is |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| REQ-profile-ai-fields | Phase 1 | Complete |
| REQ-policy-gate | Phase 1 | Complete |
| REQ-autopilot-directives | Phase 2 | Complete |
| REQ-wheel-grab | Phase 2 | Complete |
| REQ-no-auto-reconnect | Phase 2 | Complete |
| REQ-doc-hand-play-and-gate | Phase 2 | Complete |
| REQ-single-decision | Phase 3 | Complete |
| REQ-reasoning-visibility | Phase 3 | Complete |
| REQ-env-config | Phase 3 | Complete |
| REQ-continuous-loop | Phase 4 | Complete |
| REQ-call-cap-and-error-disengage | Phase 4 | Pending |
| REQ-reengage-reassess | Phase 4 | Complete |
| REQ-doc-continuous-visible-play | Phase 4 | Pending |
| REQ-doc-wheel-grab-and-reengage | Phase 4 | Complete |
| REQ-safety-limits-hold | Phase 4 | Pending |
| REQ-coaching-chat | Phase 5 | Pending |
| REQ-pause-resume | Phase 5 | Pending |
| REQ-promote-guidance | Phase 5 | Pending |
| REQ-doc-coaching | Phase 5 | Pending |
| REQ-progression-tally | Phase 6 | Pending |
| REQ-session-debrief | Phase 6 | Pending |
| REQ-memory-carryover | Phase 6 | Pending |
| REQ-study-loader | Phase 7 | Pending |
| REQ-study-confirmation | Phase 7 | Pending |
| REQ-acceptance-reconnaissance | Phase 8 | Pending |
| REQ-acceptance-goal-sessions | Phase 8 | Pending |
| REQ-acceptance-definition-holds | Phase 8 | Pending |
| REQ-improvement-trend | Phase 8 | Pending |

**Coverage:**

- v1 requirements: 28 total
- Mapped to phases: 28
- Unmapped: 0

Mapping notes:

- Definition of complete items map to the earliest phase that can fully satisfy them; Phase 8 re-verifies all six on Alter Aeon through REQ-acceptance-definition-holds.
- REQ-improvement-trend sits in Phase 8 because its three consecutive goal sessions are the D8 goal sessions. Phase 6 must record the amount of coaching per session in the debrief so the trend is measurable.
- REQ-safety-limits-hold sits in Phase 4 because that is the first phase where all four limits (call cap, no AI reconnect, disengage on errors and no commands while disconnected, conservative defaults) exist and can be put under automated test.

---
*Requirements defined: 2026-09-14*
*Last updated: 2026-09-15 after the owner amended D2 disconnect behaviour (autopilot waits across a disconnect and resumes on return)*
