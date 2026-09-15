# Constraints

Synthesized by gsd-doc-synthesizer on 2026-09-14 (MODE: new).

Both ingested documents are typed SPEC. Precedence within the set (from manifest): design v3 = 0 (higher),
policy v1.0 = 1. Where the two overlap they agree; policy section 7 additionally states that where the
policy conflicts with the connected game's rules or applicable law, the stricter rule applies.

Type legend: api-contract | schema | nfr | protocol.

---

## From the design document (precedence 0)

### CON-environment-staging-only
- source: .specify/specs/ai-game-player-design-v3.md (header: Environments)
- type: nfr
- content: All development and testing run against the MUDPuppy staging environment on Railway. Production is untouched until final acceptance.

### CON-foundation-existing-infra
- source: .specify/specs/ai-game-player-design-v3.md (header: Foundation)
- type: protocol
- content: Existing MUDPuppy infrastructure is the reference and the base: connection profiles, server-held telnet connections, the Internal Command Module's automation context and safety limits, the browser terminal, Postgres and Redis. New work extends these rather than replacing them.

### CON-directive-grammar
- source: .specify/specs/ai-game-player-design-v3.md (D2)
- type: api-contract
- content: `#AUTO ON` and `#AUTO OFF` are added to the existing `#` directive grammar. Engage/disengage mechanics are independent of any AI intelligence. A visible status indicator in the play screen must always match the true engaged/disengaged state.

### CON-command-path-automation-context
- source: .specify/specs/ai-game-player-design-v3.md (D3); owner instruction (server-side Go driver via ICM automation context)
- type: protocol
- content: Commands the AI issues go through the ICM automation context, which carries the existing safety limits. The driver runs server-side.

### CON-model-config
- source: .specify/specs/ai-game-player-design-v3.md (D3)
- type: nfr
- content: Model names and the API key come from environment configuration on staging; nothing is hard-coded.

### CON-profile-schema
- source: .specify/specs/ai-game-player-design-v3.md (D1, D6); .specify/specs/safety-and-abuse-policy-v1.md (preamble)
- type: schema
- content: New per-profile data:
  - conduct rules (text, handed to the model)
  - approach guidance (text)
  - AI settings: model names, call cap per session, disengage thresholds (reconnect is NOT an AI setting; it belongs to the connection profile's existing reconnect toggle)
  - policy acceptance record: timestamp + policy version (per profile)
  - learned notes (appended by debriefs and confirmed study lessons)
  - progression tally samples: parsed status numbers with timestamps, keyed by profile and character
  - session debriefs: goal, outcome, tally movement, what was learned, what went wrong
  - session log: decisions with model reasoning; manual-driving stretches captured as demonstrations
  Reconnect-flag ambiguity resolved by owner 2026-09-14: no reconnect field in AI settings.

### CON-conservative-defaults
- source: .specify/specs/ai-game-player-design-v3.md (D1; Definition of complete item 6)
- type: nfr
- content: Blank mechanical settings resolve to conservative engine defaults: capped calls, default disengage thresholds. The AI never initiates a reconnect regardless of settings.

### CON-loop-pacing
- source: .specify/specs/ai-game-player-design-v3.md (D4)
- type: nfr
- content: The read/decide/act loop paces itself to the game's turn rhythm rather than flooding it.

### CON-mechanical-halts
- source: .specify/specs/ai-game-player-design-v3.md (D4; Definition of complete item 6)
- type: nfr
- content: Session call cap halts the loop with a visible notice. Repeated errors or malformed model output disengage with a visible notice. Disconnect while engaged lands the AI on disengaged with no AI-initiated reconnect; connection-level reconnect is the profile toggle's business and leaves the AI disengaged.

### CON-engine-has-no-game-knowledge
- source: .specify/specs/ai-game-player-design-v3.md (Intended goal; Out of scope)
- type: nfr
- content: No game knowledge is built into the plumbing and no intent detection in the engine. No engine-level game understanding (area detection, behavior analysis). The model reasons about the game; the engine enforces numbers, flags, and the policy gate.

### CON-learned-notes-write-gate
- source: .specify/specs/ai-game-player-design-v3.md (D7)
- type: protocol
- content: Study-loader lessons are written to learned notes only after owner confirmation; nothing is written without confirmation.

### CON-version-out-of-scope
- source: .specify/specs/ai-game-player-design-v3.md (Out of scope for this version)
- type: nfr
- content: Not in this version: playing games that prohibit automation, multiple AI characters at once, autonomous unattended operation, production deployment, engine-level game understanding.

---

## From the Safety and Abuse Policy v1.0 (precedence 1)

### CON-policy-acceptance-record
- source: .specify/specs/safety-and-abuse-policy-v1.md (preamble)
- type: protocol
- content: The policy must be accepted for each game profile before the AI player can be engaged on that profile. Acceptance is recorded per profile with timestamp and policy version. Re-acceptance is required when the policy changes. Enforced by REQ-policy-gate.

### CON-policy-game-rules
- source: .specify/specs/safety-and-abuse-policy-v1.md (section 1)
- type: nfr
- content: Before engaging the AI on a profile, the owner must confirm the game permits automated play and is responsible for knowing and tracking that game's bot/scripting/automation rules. The AI must not be engaged on a game that prohibits automation, and must not be used to conceal automated play or defeat bot detection. Where a game permits automation with conditions (restricted areas, attended play only, behavior requirements), those conditions must be entered into the profile's conduct rules before engagement and bind the AI's play.

### CON-policy-supervision
- source: .specify/specs/safety-and-abuse-policy-v1.md (section 2); owner instruction (server-side does not mean unattended)
- type: nfr
- content: The AI plays under the owner's supervision; the owner is expected to be reachable and to check on behavior at reasonable intervals. No extended unattended running; disengage before stepping away for long. If disconnected or kicked while engaged, the AI will not reconnect on its own, and the owner must not re-engage until the cause is understood. Enforced mechanically by REQ-no-auto-reconnect.

### CON-policy-respect-players
- source: .specify/specs/safety-and-abuse-policy-v1.md (section 3)
- type: nfr
- content: The AI must not be used to harass, deceive, impersonate, or interfere with other players; to camp or drain shared resources in ways the community considers abusive; or to gain unfair advantage in PvP. If staff or players raise a concern, disengage first and discuss second.

### CON-policy-rate-limits
- source: .specify/specs/safety-and-abuse-policy-v1.md (section 4)
- type: nfr
- content: The AI must operate within the tool's rate and safety limits. Profile limits must not be raised to levels that could flood a game server with commands or connections. One AI-driven character per game at a time under a profile, unless the game's rules expressly allow more. (Design v3 is stricter for this version: multiple AI characters at once are out of scope regardless.)

### CON-policy-cost
- source: .specify/specs/safety-and-abuse-policy-v1.md (section 5)
- type: nfr
- content: AI play consumes paid model API calls under the owner's API key. The per-session call cap exists to protect the owner; raising it is the owner's choice and cost. Implies the cap is owner-editable on the profile (REQ-profile-ai-fields) while still subject to CON-policy-rate-limits.

### CON-policy-credentials
- source: .specify/specs/safety-and-abuse-policy-v1.md (section 6)
- type: nfr
- content: Game credentials stored in a profile are used only to connect that profile. The AI does not use stored credentials for any other purpose.

### CON-policy-data-handling
- source: .specify/specs/safety-and-abuse-policy-v1.md (section 6)
- type: nfr
- content: Game text captured during play, including other players' words, is stored in session logs and learned notes for the tool's operation. It must not be used to profile or target other players.

### CON-policy-enforcement
- source: .specify/specs/safety-and-abuse-policy-v1.md (section 7)
- type: nfr
- content: Violating the policy is grounds for disabling AI features on the profile involved, or on the account. Nothing in the policy overrides the connected game's own rules or applicable law; where they conflict, the stricter rule applies. No deliverable in design v3 implements the disable mechanism; see INGEST-CONFLICTS.md INFO.
