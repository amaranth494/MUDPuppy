# Decisions

Synthesized by gsd-doc-synthesizer on 2026-09-14 (MODE: new).

Provenance note: no ADR-typed documents were ingested. Entries in the "Owner-settled" section come from the
owner's ingest instruction and are treated as LOCKED constraints per that instruction, not as conflicts.
Entries in the "Design-level" section are decisions asserted by the highest-precedence SPEC (precedence 0)
and are settled for this version but not owner-locked. Where a design or policy passage corroborates an
owner decision, it is cited as corroboration only; the owner instruction is the authoritative source.

---

## Owner-settled decisions (LOCKED)

### DEC-branch-ai-player
- source: owner instruction (ingest-docs prompt, 2026-09-14)
- corroborated by: .specify/specs/ai-game-player-design-v3.md (header: "Branch: `ai-player` (created from `staging`)")
- status: locked
- decision: Branch `ai-player` already exists, created from `staging`, and already carries the Internal Command Module (ICM). New work lands on this branch.
- scope: repository / branching

### DEC-server-side-go-driver
- source: owner instruction (ingest-docs prompt, 2026-09-14)
- corroborated by: .specify/specs/ai-game-player-design-v3.md (D3: "issues the returned command through the automation context")
- status: locked
- decision: The AI driver runs server-side, in Go, and issues game commands through the ICM automation context. It is not a browser-side or client-side agent.
- scope: AI driver runtime placement and command path

### DEC-browser-is-supervision-surface
- source: owner instruction (ingest-docs prompt, 2026-09-14)
- corroborated by: .specify/specs/ai-game-player-design-v3.md (D2 status indicator in play screen; D3 reasoning shown in play screen; D5 chat pane beside the terminal)
- status: locked
- decision: The browser terminal is the supervision surface. Status, live reasoning, coaching, and the wheel-grab all happen there.
- scope: user interface / supervision

### DEC-server-side-is-not-unattended
- source: owner instruction (ingest-docs prompt, 2026-09-14)
- corroborated by: .specify/specs/safety-and-abuse-policy-v1.md (section 2, "Supervise the AI"); .specify/specs/ai-game-player-design-v3.md (Out of scope: "autonomous unattended operation")
- status: locked
- decision: Running the driver server-side does not license unattended operation. Play is attended and supervised by the profile owner.
- scope: operating model

### DEC-prior-specs-excluded
- source: owner instruction (ingest-docs prompt, 2026-09-14)
- status: locked
- decision: All earlier specs under `.specify/` (SP00-SP06, PR01, PR02, constitution.md) are deliberately excluded from this ingest and treated as null. The existing MUDPuppy product is taken as-is as the foundation; the AI player work is net-new. Downstream consumers must not read or cite the excluded documents.
- scope: ingest boundary / planning inputs

---

## Design-level decisions (settled by SPEC, precedence 0)

### DEC-gemini-reasoning-model
- source: .specify/specs/ai-game-player-design-v3.md (Intended goal; D3)
- status: settled (design v3)
- decision: The AI plays using a language model (Gemini) that reasons about game text, rather than scripted triggers. Model names come from environment configuration, not code.
- scope: AI decision engine

### DEC-profile-scoped-game-knowledge
- source: .specify/specs/ai-game-player-design-v3.md (Intended goal; D1; Out of scope)
- status: settled (design v3)
- decision: Everything game-specific lives on the connection profile (conduct rules, approach guidance, AI settings, learned notes). The engine has no built-in game knowledge and no intent detection; it enforces numbers, flags, and the policy gate. The model reasons about the game.
- scope: architecture boundary between engine and profile

### DEC-staging-only-until-acceptance
- source: .specify/specs/ai-game-player-design-v3.md (header: Environments)
- status: settled (design v3)
- decision: All development and testing run against the MUDPuppy staging environment on Railway. Production is untouched until final acceptance (D8).
- scope: environments / deployment

### DEC-existing-infra-is-the-base
- source: .specify/specs/ai-game-player-design-v3.md (header: Foundation)
- status: settled (design v3)
- decision: Existing MUDPuppy infrastructure is the reference and base: connection profiles, server-held telnet connections, the ICM automation context and its safety limits, the browser terminal, Postgres, and Redis.
- scope: foundation / reuse

### DEC-policy-v1-is-current-version
- source: .specify/specs/ai-game-player-design-v3.md (Related documents); .specify/specs/safety-and-abuse-policy-v1.md (header: "Policy version: 1.0")
- status: settled (design v3 + policy v1.0)
- decision: Safety and Abuse Policy version 1.0 is the "current policy version" that the D1 gate checks. Changing the policy version forces re-acceptance on every profile.
- scope: policy gate versioning

### DEC-alter-aeon-acceptance-target
- source: .specify/specs/ai-game-player-design-v3.md (Definition of complete; D8)
- status: settled (design v3)
- decision: Project completion is demonstrated on Alter Aeon, from staging, under the owner's supervision.
- scope: acceptance / definition of done

### DEC-version-scope-exclusions
- source: .specify/specs/ai-game-player-design-v3.md (Out of scope for this version)
- status: settled (design v3)
- decision: Out of scope for this version: playing games that prohibit automation, multiple AI characters at once, autonomous unattended operation, production deployment, and any engine-level game understanding (area detection, behavior analysis).
- scope: version boundary
