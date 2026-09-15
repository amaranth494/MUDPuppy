# AI Game Player for MUDPuppy

## What This Is

Extend the MUDPuppy web MUD client so a game character can be driven either by its human owner or by a Gemini-backed reasoning AI, switched like a car's autopilot, supervised from the browser, learning from the game's own progression numbers. The AI driver runs server-side in Go and issues commands through the existing Internal Command Module (ICM) automation context; the browser terminal is the supervision surface (status indicator, live reasoning, coaching chat, and the wheel-grab). Everything game-specific lives on the connection profile, so the same engine adapts to Alter Aeon, the owner's own MUD, or any other text game.

## Core Value

The owner can hand the wheel to the AI and take it back instantly, always seeing what the AI is doing and why, with every mechanical safety limit holding, and the AI getting measurably better session over session by the game's own numbers.

## Requirements

### Validated

The existing MUDPuppy product is taken as-is per owner decision; its prior specs are excluded and nothing from them is tracked here.

- [x] D1 Profile foundation and policy gate: REQ-profile-ai-fields, REQ-policy-gate — Validated in Phase 1: Profile Foundation and Policy Gate (2026-09-15). Proof filed under `.planning/phases/01-profile-foundation-and-policy-gate/evidence/`: staging startup log showing migration 010 applied, canned report passing C1-C4, `[AI-PLAYER]` log excerpt, and seven end-user screenshots. Verification 4/4.

### Active

Requirement IDs come from `.planning/intel/requirements.md` and are quoted from the design's D1-D8 "Accepted when" bullets and Definition of complete. Full list with acceptance text in `.planning/REQUIREMENTS.md`.

- [ ] D2 Autopilot switch: REQ-autopilot-directives, REQ-wheel-grab, REQ-no-auto-reconnect
- [ ] D3 One AI decision: REQ-single-decision, REQ-reasoning-visibility, REQ-env-config
- [ ] D4 Continuous play: REQ-continuous-loop, REQ-call-cap-and-error-disengage, REQ-reengage-reassess
- [ ] D5 Coaching channel: REQ-coaching-chat, REQ-pause-resume, REQ-promote-guidance
- [ ] D6 Measurement and memory: REQ-progression-tally, REQ-session-debrief, REQ-memory-carryover
- [ ] D7 Study loader: REQ-study-loader, REQ-study-confirmation
- [ ] D8 Acceptance run on Alter Aeon: REQ-acceptance-reconnaissance, REQ-acceptance-goal-sessions, REQ-acceptance-definition-holds
- [ ] Definition of complete (cross-cutting): REQ-doc-hand-play-and-gate, REQ-doc-continuous-visible-play, REQ-doc-wheel-grab-and-reengage, REQ-doc-coaching, REQ-improvement-trend, REQ-safety-limits-hold

### Out of Scope

- Playing games that prohibit automation. Design v3 "Out of scope"; policy v1.0 section 1.
- Multiple AI characters at once. Design v3 is stricter than policy section 4 for this version; the stricter rule applies (policy section 7). One AI-driven character per game per profile, no exception path.
- Autonomous unattended operation. Design v3 "Out of scope"; policy section 2. Server-side placement of the driver does not license unattended running (locked decision).
- Production deployment. All development and testing run on Railway staging; production is untouched until final acceptance.
- Any engine-level game understanding (area detection, behavior analysis, intent detection). The model reasons about the game; the engine enforces numbers, flags, and the policy gate.
- Policy section 7 "disable AI features on the profile or account" enforcement mechanism. No deliverable in design v3 implements it; recorded as a known omission for this version (CON-policy-enforcement), not a conflict.
- Reconnect behaviour of the connection itself. Reconnect is a connection-profile concern, not an AI setting, and is unchanged by this work. Note: no reconnect toggle exists in the profile schema today (frontend/src/types/index.ts ProfileSettings); building one is outside this project. The AI-side obligation (never initiate a reconnect, issue nothing while disconnected, resume only once the connection is back) is in scope.
- Prior `.specify/` specs (SP00-SP06, PR01, PR02, constitution). Excluded by owner decision; treated as null. Downstream work must not read or cite them.

## Context

**Source of truth.** Exactly two documents: `.specify/specs/ai-game-player-design-v3.md` (precedence 0) and `.specify/specs/safety-and-abuse-policy-v1.md` (precedence 1, policy version 1.0). Ingest conflict report: 0 blockers, 0 warnings; the one warning (auto-reconnect flag in D1 AI settings) was resolved by the owner on 2026-09-14 and the design amended.

**Developer-facing success metric.** The design's Definition of complete items 1 through 6 all hold, demonstrated on Alter Aeon from the Railway staging environment during the D8 acceptance run.

**Target runtime (taken as-is).** Go 1.26 backend (`net/http` mux with path patterns, gorilla/websocket, lib/pq Postgres, go-redis), React + TypeScript + Vite frontend served as a static SPA by the Go server, golang-migrate SQL migrations (`migrations/001`-`009` exist; new work starts at `010`), deployed on Railway (project `mudpuppy`, environment `staging`). Config is env vars via `internal/config/config.go`. Gemini model names and API key will be new env vars on staging.

**Codebase facts the roadmap accounts for (verified 2026-09-14):**

- `internal/icm` (engine, dispatcher with `SafetyChecker`, `ExecutionContext` with `ContextAutomation = "automation"`, `SafetyLimits`) exists but is NOT wired into the server: nothing imports it and no `/api/v1/icm` routes are registered in `cmd/server/main.go`. `frontend/src/services/icm-adapter.ts` calls those routes and silently falls back to browser-side logic. Live aliases/triggers/timers currently execute in the browser (`frontend/src/services/automation.ts`). Wiring the ICM engine server-side is part of the D3 slice.
- The telnet connection is server-held (`internal/session/manager.go`: one session per user, `SendCommand`, `ReadOutput`). MUD output is relayed to the browser over a websocket (`internal/session/websocket.go`, message types connect/disconnect/data/status/error/sync/log) with no server-side buffer; the driver needs a tap on that stream.
- The `#` directive grammar is parsed browser-side in `frontend/src/services/automation.ts`; `#AUTO ON/OFF` is added there and must call the server, which owns the true engaged/disengaged state.
- Browser-side triggers and timers send commands over the same websocket path as typed input, so the D2 wheel-grab needs a source flag distinguishing human-typed input from automation-fired commands.
- No session log or transcript persistence exists. Persistence today: users, otp_challenges, saved_connections, connection credentials (encrypted), profiles (1:1 with saved_connections; JSONB columns keybindings, settings, aliases, triggers, variables, timers). Profile REST pattern: GET/PUT sub-resources under `/api/v1/profiles/{connection_id}/{aliases|triggers|environment|timers}`. New tables are needed for AI sessions/decisions, progression samples, and debriefs.
- Test coverage is near zero (one Go test file, `internal/icm/icm_test.go`; no frontend tests). Definition of complete item 6 says the safety limits "hold under test", so the D4 phase must add tests for them.

**Supervision model.** The driver runs server-side so a page refresh never kills or confuses it, but supervision is enforced by the policy gate and mechanical settings (call cap, disengage thresholds, wheel-grab, no commands while disconnected), not by process placement. The policy expects the owner to be reachable and checking in.

**Policy section 1 confirmation.** The policy requires the owner to confirm the game permits automated play before engaging. Design v3 D1 specifies only the acceptance record as the gate. This roadmap treats accepting the policy (whose text includes section 1) as that confirmation; conditional permissions (restricted areas, attended-only) go into the profile's conduct rules, which D8's goal sessions exercise on Alter Aeon.

## Constraints

- **Environment**: Staging only on Railway until final acceptance; production untouched. (CON-environment-staging-only)
- **Foundation**: Extend existing infrastructure (profiles, server-held telnet, ICM automation context and safety limits, browser terminal, Postgres, Redis); do not replace it. (CON-foundation-existing-infra)
- **Command path**: Every AI-issued command goes through the ICM automation context and inherits its safety limits; the driver is server-side Go. (CON-command-path-automation-context, locked decision 2)
- **Directive grammar**: `#AUTO ON` / `#AUTO OFF` join the existing `#` grammar; the play-screen indicator must always match the true state. (CON-directive-grammar)
- **Model config**: Model names and API key from environment; nothing hard-coded. (CON-model-config)
- **Profile schema**: New per-profile data: conduct rules, approach guidance, AI settings (model names, call cap per session, disengage thresholds; no reconnect field), policy acceptance record (timestamp + version), learned notes, progression samples (keyed by profile and character), session debriefs, session log with decisions and reasoning and manual-driving stretches. (CON-profile-schema)
- **Blank-setting defaults**: A blank model name means the server default; a blank call cap means no cap; a blank disengage threshold means the engine's built-in error handling: any AI failure produces an informative error, disengages, and never crashes or interrupts regular play. The AI never initiates a reconnect regardless of settings. (CON-conservative-defaults, owner decision 2026-09-15)
- **Loop pacing**: Read/decide/act paces to the game's turn rhythm; never floods the server. (CON-loop-pacing, CON-policy-rate-limits)
- **Mechanical halts**: Call cap halts with a visible notice; repeated errors or malformed model output disengage with a visible notice; a disconnect while engaged puts autopilot in a waiting state that issues nothing and resumes when the connection returns; only `#AUTO OFF` turns autopilot off. (CON-mechanical-halts, amended by owner decision 2026-09-15)
- **No game knowledge in the engine**: No intent detection, area detection, or behaviour analysis in the plumbing. Status commands used for the progression tally are profile configuration, not engine code. (CON-engine-has-no-game-knowledge)
- **Learned-notes write gate**: Study-loader lessons enter learned notes only after owner confirmation. (CON-learned-notes-write-gate)
- **Policy obligations enforced mechanically**: per-profile acceptance with version (CON-policy-acceptance-record); rate/safety limits and one AI character per game (CON-policy-rate-limits); owner-editable call cap as cost protection (CON-policy-cost); stored credentials used only to connect that profile (CON-policy-credentials); captured game text stored for tool operation only (CON-policy-data-handling).
- **Cost**: AI play consumes paid Gemini calls under the owner's key; the per-session call cap is the protection.

## Key Decisions

### Locked (owner-settled, 2026-09-14)

| ID | Decision | Rationale | Outcome |
|----|----------|-----------|---------|
| DEC-branch-ai-player | Branch `ai-player` already exists from `staging` and already carries the ICM; no merge or re-branch. | Work lands where the ICM foundation already is. | Locked |
| DEC-server-side-go-driver | The AI driver runs server-side in Go and issues commands through the ICM automation context (`internal/icm`) so it inherits ICM safety limits. The browser is the supervision surface only (terminal, autopilot switch, chat pane); a page refresh must never kill or confuse the driver. | Safety limits live in one place; driver state is not tied to a browser tab. | Locked |
| DEC-server-side-is-not-unattended | Server-side does not mean unattended; supervision is enforced by the policy gate and mechanical settings, not by process placement. | Policy section 2; design lists unattended operation as out of scope. | Locked |
| DEC-reconnect-is-connection-toggle | Reconnect is a connection-profile toggle, not an AI decision; the AI never initiates a reconnect. | Resolves the only ingest warning; keeps the AI's failure mode passive. | Locked |
| DEC-autopilot-waits-across-disconnect | Autopilot has exactly two owner-set states: ON is entered only by `#AUTO ON`, OFF only by `#AUTO OFF`. A dropped connection is not `#AUTO OFF`: autopilot holds in a waiting state, issues nothing while disconnected, and resumes on its own when the connection returns. (Owner, 2026-09-15; design D2 and Definition of complete item 6 amended. Supersedes the earlier "stays disengaged until re-engaged" rule.) | The owner's mental model: the switch position is the owner's choice, and a connection blip should not flip it. | Locked |
| DEC-prior-specs-excluded | Prior `.specify/` specs (SP00-SP06, PR01, PR02, constitution) are excluded; the product is taken as-is and new work is net-new. | Clean planning inputs; existing behaviour is the base, not a subject. | Locked |

### Settled by design v3

| ID | Decision | Outcome |
|----|----------|---------|
| DEC-gemini-reasoning-model | The AI plays using Gemini reasoning over game text, not scripted triggers; model names from env. | Settled |
| DEC-profile-scoped-game-knowledge | All game-specific knowledge lives on the connection profile; the engine enforces numbers, flags, and the gate. | Settled |
| DEC-staging-only-until-acceptance | Develop and test on Railway staging; production untouched until D8 passes. | Settled |
| DEC-existing-infra-is-the-base | Existing MUDPuppy infrastructure is the reference and base. | Settled |
| DEC-policy-accept-once | Policy acceptance is one-time per profile: shown the first time AI configuration is opened, never expires, never re-asked on policy change, discarded with the profile. Version 1.0 is recorded for the record only. (Owner, 2026-09-15; design D1 amended.) | Locked |
| DEC-alter-aeon-acceptance-target | Completion is demonstrated on Alter Aeon, from staging, under the owner's supervision. | Settled |
| DEC-version-scope-exclusions | Out of scope list above governs this version. | Settled |

### Roadmap-level decisions (this document)

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Phases map 1:1 to deliverables D1-D8 in the design's stated order (D5 before D6). | Owner instruction: preserve deliverable structure and dependencies; each deliverable is a working, testable increment. | Pending |
| Definition of complete items map to the earliest phase that can fully satisfy them (items 1, 3-part-a to Phase 2 via the gate; items 2, 3, 6 to Phase 4; item 4 to Phase 5) and item 5's three-session trend to Phase 8, with Phase 8 re-verifying all six on Alter Aeon. | Every requirement maps to exactly one phase; the demonstration is the acceptance run. | Pending |
| Wiring the dormant ICM engine server-side is Phase 3 work, not a separate phase. | It is the command path the single decision must use; nothing earlier needs it. | Pending |
| The Phase 6 debrief records the amount of coaching required in the session. | Definition of complete item 5 requires "required coaching declines" to be measurable across sessions; without a recorded count the Phase 8 trend cannot be shown. Derived from REQ-improvement-trend, not invented. | Pending |

---
*Last updated: 2026-09-15 after Phase 1 completed and verified on staging and the owner amended D2 disconnect behaviour during the Phase 2 discussion (DEC-autopilot-waits-across-disconnect)*
