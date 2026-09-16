# Phase 3: One AI Decision - Research

**Researched:** 2026-09-15
**Domain:** Go backend (`internal/session` output-tap and transcript, `internal/icm` wiring — a dormant engine imported by no other package until now, a new `internal/gemini` hand-written REST client, new Postgres tables via golang-migrate `011`), TypeScript frontend (new AI Assist panel, Logs section, websocket decision messages) — brownfield extension of three subsystems that already work (Phase 1's profile/policy store, Phase 2's autopilot state machine and wheel-grab, the never-wired ICM engine) plus one new external call (Gemini `generateContent` over hand-written `net/http`).
**Confidence:** HIGH for every backend extension point and for the ICM dispatch mechanics (all verified by direct code read this session, including the single most consequential finding: `Engine.Process()` never reaches `Dispatcher.Dispatch()` for a plain pass-through command, so the driver must call `Dispatcher.Dispatch()` directly). MEDIUM for the Gemini REST contract (official docs fetched and cross-checked this session, but the exact free-tier numeric rate limits are account-specific and undocumented by Google; treated as ASSUMED). MEDIUM for table shapes and the websocket message extension (genuine "Claude's Discretion" per CONTEXT.md, one clearly-simplest option identified and justified against this codebase's own conventions, not a pre-existing pattern to copy verbatim).

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**When the AI acts and what it reads**
- **D-01:** The one decision fires immediately on `#AUTO ON`, using whatever recent game text the server already holds. No waiting for fresh output, no separate step directive.
- **D-02:** A WAITING to ON resume after a reconnect counts as an engage and fires one fresh decision on the post-reconnect text. Consistent with Phase 4's rule that a resume reassesses rather than resuming a stale plan.
- **D-03:** After the single decision and its command, autopilot stays ON and idle. Nothing more is issued until Phase 4 adds the loop. Only `#AUTO OFF` or the Phase 2 wheel-grab turns it off (Phase 2 D-01, D-04 hold; a repeated `#AUTO ON` is still a no-op and must not fire a second decision).
- **D-04:** The "current game text" is a rolling per-session window of recent game output kept server-side: sized by a server constant (roughly the last screenful or two), ANSI colour codes stripped, telnet control bytes already stripped, and including text from before the switch was flipped so the AI's first look resembles what the owner sees on screen. Not a profile setting.
- **D-05:** The prompt has two tiers from the start. Tier one is the immediate window for reaction. Tier two is the profile's standing text: conduct rules and approach guidance, verbatim (Phase 1 D-12). Phase 6 adds learned notes to tier two and the AI's ability to write to it; the prompt shape does not change then, only what fills tier two.

**How the decision shows on the play screen**
- **D-06:** A floating, minimizable AI Assist panel on the play screen. It can be collapsed to a small tab and reopened; the terminal keeps its full width.
- **D-07:** The panel opens automatically whenever the connected profile has AI activated (the Phase 1 policy acceptance recorded), whether autopilot is ON or OFF. Profiles without acceptance see nothing new.
- **D-08:** Panel content in this phase: each decision is one AI message showing the short reasoning, then the command it issued; state changes (engaged, waiting, resuming, disengaged, refusals, failures) appear as system lines. No message input until Phase 5.
- **D-09:** In the terminal, the AI's issued command prints as `[AI-ASSIST > north]` and this line replaces local echo for AI-issued commands. The game's own echo and response follow as usual.
- **D-10:** The reasoning is a short plain-language "why" (a sentence or three) plus exactly one command. The prompt asks for that shape; the model's internal thinking trace is not shown.

**`#AUTO` grammar (amends Phase 2 D-05)**
- **D-11:** `#AUTO` accepts only `ON` and `OFF`. A bare `#AUTO`, `#AUTO STATUS`, or any other argument prints one local line: "Don't know what you're talking about. Your options are ON or OFF." The Phase 2 `#AUTO STATUS` diagnostic line is retired; the badge and the panel are the status surfaces. No directive starts, stops, or prints logging.

**Where decisions and reasoning are stored**
- **D-12:** Each decision is stored on its own row: timestamp, profile and game connection, the game text window the model saw, the reasoning, the command issued, and the outcome (sent, refused, failed). The panel shows it live and reloads the current connection's decisions after a page refresh from these rows.
- **D-13:** A first failure of any kind (API error, no command, several commands, a `#` directive or other non-game line, malformed answer) sends nothing to the game, shows an informative notice in the panel and the terminal, lands autopilot on OFF, and leaves regular play untouched. No retry in this phase; Phase 4 adds the threshold behaviour on top.

**Session transcript and the Logs section**
- **D-14:** A session transcript is stored for every saved-profile game connection, from connect to disconnect, AI activated or not. Quick connects with no profile are not logged. It captures everything sent to the game and everything received from the game, with each sent line marked as human-typed or AI-issued. It does not contain reasoning. Sessions are keyed by profile and start date-time.
- **D-15:** AI engage and disengage moments are recorded inside the transcript so a stint at the wheel (`#AUTO ON` to `#AUTO OFF`) can be found later for Phase 6 debriefs. A WAITING stretch stays inside the same stint.
- **D-16:** Settings gets a Logs section in the left nav (per connection profile). Selecting it shows an "Open logging" button on the right. The button opens the log page in a new browser tab; the current tab does not navigate.
- **D-17:** The log page is per game profile: a left pane lists that profile's sessions by date/time, a right pane shows the selected session's transcript as text. Sign-in protected, scoped to the signed-in owner's profile. No cross-profile listing, no share links.

**Model configuration and failure when absent**
- **D-18:** A model registry from environment configuration: a default model plus any number of named entries, each with its API endpoint and API key. A profile's blank model name resolves to the default (Phase 1 D-09); a non-blank name selects a registry entry. Phase 3 implements the Gemini API only. Nothing is hard-coded.
- **D-19:** The owner's key is on the Gemini free tier for this test. Its rate limits are a fact the planner records: the per-session call cap is the owner's cost and rate protection, and a rate-limit answer from Gemini is a failure under D-13.
- **D-20:** When the key or default model is missing from the environment, the server starts normally and `#AUTO ON` is refused with a clear notice such as `[Autopilot refused: AI is not configured on this server]`; the switch stays OFF and hand play is unaffected.

### Claude's Discretion
- Window size and buffer implementation (ring buffer per user session fed from the same relay path the websocket uses), where ANSI stripping happens, and how the driver reads a snapshot.
- Table shapes and migration numbers (`011`+) for AI sessions or stints, decisions, and session transcripts; whether the transcript is one row per session with appended text or one row per line.
- The websocket message type that carries decisions, reasoning, system lines, and the `[AI-ASSIST > ...]` echo to the browser (a new `ai` type or extending `autopilot`), and how the panel reloads after refresh (REST fetch on attach).
- How the ICM dispatcher is invoked for a plain game command (the classifier marks it pass-through; it must still go through `Dispatch` with `ContextAutomation` and the `SafetyChecker` so the diagnostic `go test` proves it), where the ICM engine instance lives, which `/api/v1/icm` routes are registered, and how the frontend adapter surfaces a server error instead of falling back.
- Registry format in env (one JSON variable versus a small set of `AI_MODEL_*` variables), env variable names, and the Gemini client package (hand-written `net/http` against the REST API is fine; no SDK is required).
- Prompt wording and the structured answer format (for example JSON with `reasoning` and `command`), validation that the command is a single plain game line.
- Log page delivery (a small server-rendered page or an SPA route), the transcript's line marking format, and where the Logs nav entry sits.
- Panel visuals, colours, default position and size, minimized tab look, and the exact wording of system lines and the D-11 unknown-option line.
- Where the DR-2-01 fix lands (gate the sign-in code log line behind an env flag off by default, or log a hash) and confirming production has no equivalent.
- The `[AI-PLAYER]` log line format for each decision (request sent, answer received, command dispatched, failure), following Phases 1 and 2.

### Deferred Ideas (OUT OF SCOPE)
- Coaching input in the AI Assist panel — Phase 5 (REQ-coaching-chat).
- Learned notes and an AI-written running memory — Phase 6 (REQ-session-debrief, REQ-memory-carryover). Tier two of the prompt gains learned notes then; D-05 keeps the prompt shape stable.
- Non-Gemini model vendors — the registry (D-18) carries endpoint and key per entry; no implementation this version.
- Per-profile tuning of the recent-text window — a server constant now (D-04); revisit in Phase 4.
- Cross-profile log listing, share links without sign-in, transcript search or export — not built.
- Security review items for this phase (raise, do not decide here): D-14's full transcript capture (including hand play) against the policy's data-handling sections; the Gemini key on staging and free-tier limits; the log page's authorization scope; the driver bypassing the human wheel-grab check; the closure of DR-2-01 and DR-2-02.
- Shelf copy `D:\Projects\ai-mud-player\documents\` — not touched by this phase; D-11's retirement of `#AUTO STATUS` and D-14's transcript scope are implementation decisions, not design amendments.
- **Folded todo `2026-09-15-phase3-security-carry-forward.md`:** DR-2-01 (medium, MUST FIX: staging deploy log prints the one-time sign-in code — gate behind an env flag off by default or log a hash; confirm production has no equivalent) and DR-2-02 (low: six throwaway staging profiles from Phases 1-2 — delete via the app, or use as Phase 3 fixtures and delete before the Phase 3 security review; never sign the owner out).
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| REQ-single-decision | With autopilot engaged, the driver reads the current game text, sends one request to Gemini with conduct rules and approach guidance, and issues the returned command through the automation context. | `Pattern 1` (output tap + ring buffer on `Manager`) gives the read side. `Pattern 2` (Gemini REST client, hand-written `net/http`) gives the request side, verified against official docs (`## Gemini API Reference` below). `Pattern 3` (the ICM dispatch pitfall) is the load-bearing finding for "through the automation context": `Engine.Process()` never calls `Dispatch()` for a plain command, so the driver must call `Dispatcher.Dispatch()` directly with a hand-built `NormalizedCommand`, then call `manager.SendCommand` itself once `Dispatch` returns no error. |
| REQ-reasoning-visibility | The decision and reasoning appear in the play screen as they happen and are stored in the session log, surviving a page refresh. | `Pattern 4` (new `MsgTypeAI` websocket type carrying a `Decision` payload) for live display; `Pattern 5` (migration `011`, `ai_decisions` table) plus a new `GET /api/v1/session/decisions` REST endpoint (mirroring `session.Handler.Status`'s attach-time re-sync pattern already proven in Phase 2) for refresh survival. |
| REQ-env-config | Model names and API key come from environment configuration; nothing hard-coded; the server fails clearly if absent when engagement is attempted. | `internal/config.Load()`'s existing required/optional env pattern (fail-fast for `SESSION_SECRET`, warn-and-default for everything else) extends directly; D-20 requires a *third* pattern — optional at startup, refused only at engage time — documented in `Pattern 6` below with the exact `EngageGate`-callback precedent already used for the Phase 1 policy gate. |
</phase_requirements>

## Summary

Phase 3 is a wiring-and-integration phase across three previously separate, already-working subsystems, plus one new external HTTP call. Two of the three integration points (the ICM engine, the Gemini API) have never been exercised by this codebase at all — `internal/icm` is dormant (imported by nothing outside its own package and its own test file, confirmed by an exhaustive grep this session), and there is no `internal/gemini` or equivalent. The third (the session/websocket/automation stack Phase 2 built) is proven and extends cleanly along exactly the same "single choke point, new map on `Manager` under the existing mutex" pattern Phase 2 established.

The single most consequential finding this session is that **the ICM engine's own `Process()` pipeline is not the integration point for the AI's game commands.** `Engine.Process()` classifies a plain string like `"north"` as pass-through in `passThroughClassifier.ClassifyCommand` and returns immediately (`engine.go:94-105`) — it never reaches `Dispatcher.Dispatch()`, and therefore never touches `SafetyChecker.CheckCircuitBreaker`/`CheckRateLimit`/`CheckQueueDepth` at all. If the driver called `engine.Process()` with the AI's returned command the way the frontend's `icm-adapter.ts` calls `/api/v1/icm/execute`, Phase 3's fourth success criterion — "dispatched through the ICM engine's dispatcher and safety checker" — would be **false** even though the code compiles and the command reaches the MUD (because nothing in `Process()`'s early-return path sends anything anywhere; a caller relying on `Process()` alone would need its own separate `manager.SendCommand` call regardless). The correct integration is for the driver to call `Dispatcher.Dispatch(&ctx, sessionID, normalizedCmd)` **directly** — bypassing `Engine.Process()` entirely — with a hand-built `*icm.NormalizedCommand{Command: theCommand, Operator: "", RequiresExecution: true}` and `ctx := icm.ContextAutomation`. `Dispatch` unconditionally runs `checkSafety` (circuit breaker, rate limit, queue depth) before it ever looks up a handler; since no handler is registered under the empty operator family, `getHandler` returns `nil` and `Dispatch` returns `(nil, nil)` — "safety-approved, no handler, nothing more to do here." The driver reads that `nil` error as "approved" and *only then* calls `manager.SendCommand(userID, command)` itself, exactly the way the browser's own typed commands already reach the MUD. This is also why `Dispatch`'s safety counters (`RecordExecution`, which only fires after a handler successfully executes) will **not** accumulate for the AI's plain commands under this scheme — noted as a Common Pitfall below and flagged as an Open Question for Phase 4, where a real call-counter is needed; it does not affect Phase 3's single-decision success criteria.

The second material finding is that this codebase's evidence-format and single-choke-point conventions (Phase 1's `EngageGateAllowed`/`HandlerCallbacks` injection pattern, Phase 2's "new map on `Manager` under `m.mu`, never a field on `Session`" rule) extend to every new piece of Phase 3 state without modification: the recent-text ring buffer and the transcript tap are both new `map[string]*T` fields on `Manager`, guarded by the existing `m.mu`; the missing-Gemini-config refusal is a new `EngageGate`-style callback resolved at `#AUTO ON` time, not a startup fatal (D-20 explicitly forbids `config.Load()` failing when Gemini env is absent); and the new `ai_decisions`/`ai_sessions`/`game_sessions` tables follow migration `010`'s exact `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` / `CREATE TABLE IF NOT EXISTS` conventions already established in `006_create_profiles.up.sql`.

The third material finding concerns the Gemini API itself: as of this research date the API accepts the key via either a `?key=` query parameter (documented, still works) or the `x-goog-api-key` HTTP header (Google's now-preferred form). **The header form must be used**, not the query parameter — this project already has one deferred security finding (DR-2-01) about a secret value appearing in a log line; a `?key=...` query parameter would appear in any access log, proxy log, or error trace that records the request URL, recreating the same class of problem this phase is contractually obligated to close. Exact numeric free-tier rate limits (RPM/RPD) are **not published** by Google's own docs — they are surfaced only per-account in the AI Studio dashboard — so any specific number is an assumption, not a verified fact; the design already protects against this correctly via the profile's `CallCap` and `EngageGateRefusalMessage`-style failure-is-safe posture (D-19: a rate-limit answer is simply a D-13 failure).

**Primary recommendation:** Build a small `internal/gemini` package (hand-written `net/http`, one `GenerateContent(ctx, model, apiKey, systemInstruction, userText) (*Decision, error)` function using `x-goog-api-key`, `responseMimeType: "application/json"`, and a `responseSchema` constraining the answer to `{reasoning: string, command: string}`); add a `driver` type (new `internal/driver` package, or a new file inside `internal/session` — recommend a new package to avoid growing `session` further, see Architectural Responsibility Map) that on `EngageAutopilot` success (and on `resumeAutopilotLocked`'s WAITING→ON transition per D-02) takes the ring-buffer snapshot, calls Gemini, validates the answer against D-13's failure list, calls `icm.GetDispatcher().Dispatch(&automationCtx, sessionID, normalizedCmd)` directly (never `Engine.Process()`), and on approval calls `manager.SendCommand` itself; store the decision row; push a new `MsgTypeAI` websocket message with a `Decision` payload; extend the missing-config check into the existing `EngageGate`-callback shape already proven for the Phase 1 policy gate, composed with it (both must pass for `#AUTO ON` to succeed) rather than duplicating the refusal plumbing.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Recent-text ring buffer (D-04) | API / Backend (`internal/session.Manager`, new map keyed by userID, fed from `ReadOutput`) | — | Same tap point CONTEXT.md's own code_context names (`SendCommand`/`ReadOutput` are the only paths to and from the MUD); must be a new map under `m.mu`, not a `Session` field, for the same reason Phase 2's autopilot state cannot live on `Session` (survives nothing across a reconnect if it does) |
| Gemini request/response (D-18) | API / Backend (new `internal/gemini` package, hand-written `net/http`) | — | CON-model-config; no SDK needed for one endpoint; keeps the dependency surface at zero new `go.mod` entries |
| Prompt assembly (tier one + tier two, D-05) | API / Backend (new `internal/driver` package or a `session` sub-file) | Database / Storage (`profiles.conduct_rules`/`approach_guidance`, already migrated) | The driver needs both the live ring-buffer snapshot and the profile row; neither `gemini` nor `session` alone should own composing the prompt |
| Command dispatch + safety check (REQ-single-decision, success criterion 4) | API / Backend (`internal/icm.Dispatcher.Dispatch`, called directly, not via `Engine.Process`) | — | See Summary: `Process()` short-circuits pass-through before `Dispatch`; the diagnostic test and the real path must both go through `Dispatch` directly with `ContextAutomation` |
| Actually sending the command to the MUD | API / Backend (`internal/session.Manager.SendCommand`, called by the driver after `Dispatch` approves) | — | `Dispatch` never sends anything to any connection; it is a pure safety/authority gate, confirmed by reading `dispatcher.go` in full |
| Decision + reasoning persistence (D-12) | Database / Storage (new `ai_decisions` table, migration `011`) | API / Backend (new store package) | Matches Phase 1's `ProfileStore` pattern exactly |
| Decision + reasoning live display (D-06 to D-10) | Browser / Client (new `AIAssistPanel.tsx`) | API / Backend (new `MsgTypeAI` websocket push + `GET .../decisions` REST reload) | Same server-owns-truth, browser-reflects pattern as the Phase 2 autopilot badge |
| `[AI-ASSIST > ...]` terminal echo replacing local echo (D-09) | Browser / Client (`PlayScreen.tsx`, on receipt of the `MsgTypeAI` "command" event) | — | The terminal write is purely presentational; the command itself already went to the MUD server-side, so the browser is only rendering what happened, not re-sending it |
| Session transcript (D-14, D-15) | API / Backend (tap on `Manager.SendCommand`/`ReadOutput`, new store) | Database / Storage (new `game_sessions`/`game_session_lines` tables) | Same tap points as the ring buffer; a saved-profile connection with `ConnectionID != ""` gates whether a transcript session opens (quick connects excluded, D-14) |
| Logs section + log page (D-16, D-17) | Browser / Client (new Settings nav entry + new route opened in a new tab) | API / Backend (new scoped-read endpoints) | Matches `AIPlayerPanel.tsx`'s existing per-connection Settings-section pattern |
| Model registry resolution (D-18) | API / Backend (`internal/config`, extended) | — | Same `Load()` pattern already resolves every other env-sourced setting; the registry is a map, not new machinery |
| Missing-config refusal at engage time (D-20) | API / Backend (`session.HandlerCallbacks`, a second callback composed with `EngageGate`) | — | D-20 explicitly forbids a startup fatal; reuses the exact "resolve at action time, fail closed on nil" shape already proven for the Phase 1 policy gate |
| DR-2-01 remediation (OTP code in staging log) | API / Backend (`internal/auth/handler.go`, two call sites) | — | Existing `if h.emailSender != nil && h.emailSender.IsConfigured()` branch already distinguishes staging-with-SMTP from dev-mode; gate the `log.Printf("STAGING: OTP sent to user, code: %s", otp)` line behind a new env flag, default off |

## Standard Stack

No new Go module dependency. No new frontend package.

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `net/http` (stdlib) | Go 1.26 stdlib [VERIFIED: go.mod `go 1.26`] | Gemini REST client (`generateContent`) | CONTEXT.md's own discretion note: "hand-written `net/http` against the REST API is fine; no SDK is required." No official Google Go SDK is in `go.mod` today, and adding one for a single endpoint is unjustified surface area. |
| `encoding/json` (stdlib) | Go 1.26 stdlib | Building the Gemini request body and parsing `{reasoning, command}` out of the structured JSON response | Already the only JSON library in this codebase (`internal/store`, `internal/session` all use it directly) |
| `regexp` (stdlib) | Go 1.26 stdlib | ANSI CSI-sequence stripping for the AI's text window (see Pattern 1) | A single compiled pattern is sufficient for this use case (see Common Pitfalls; a full terminal-emulation state machine is unnecessary hand-rolling for a text snapshot, not a rendered terminal) |
| `github.com/gorilla/websocket` | `v1.5.3` [VERIFIED: go.mod, already imported] | New `MsgTypeAI` outbound message | Already the only websocket library; Phase 2 already added `MsgTypeAutopilot` the same way |
| `database/sql` + `github.com/lib/pq` | `v1.11.2` [VERIFIED: go.mod] | New `ai_decisions`/`ai_sessions`/`game_sessions` tables via a new store package | Unchanged from Phases 1-2; same `NewXStore(db *sql.DB) *XStore` constructor shape as `NewProfileStore`/`NewConnectionStore` |
| `github.com/golang-migrate/migrate/v4` | (existing, from go.mod) | Migration `011` | Already the only migration tool; `main.go` already runs `m.Up()` against `file://migrations` |
| React state/context (existing) | package.json, unchanged | New `AIAssistPanel.tsx`, Logs section, decisions reload | Matches every existing component in this codebase |

**Installation:** none required.

## Package Legitimacy Audit

**Not applicable.** This phase installs zero new packages in `go.mod` or `frontend/package.json`. The Gemini integration is a hand-written REST client per CONTEXT.md's explicit discretion note ("no SDK is required"), which also avoids the slopsquatting/hallucinated-package risk category entirely — there is no third-party package name to verify. Per the Package Legitimacy Gate's own instructions ("Required whenever this phase installs external packages"), the audit is skipped, identical to Phases 1 and 2.

## Gemini API Reference

Verified against official Google AI for Developers documentation this session (`ai.google.dev`), not training-data recall, because model names and API surface details in this space move fast (confirmed: `gemini-2.5-flash-lite` itself is scheduled to shut down October 16, 2026, within the life of this project).

### Endpoint

```
POST https://generativelanguage.googleapis.com/v1beta/models/{model}:generateContent
```
`{model}` is a bare model ID (e.g. `gemini-2.5-flash`), not a full resource path. [CITED: ai.google.dev/api/generate-content]

### Authentication

Two forms both work; **use the header form**:
```
x-goog-api-key: YOUR_API_KEY
```
The legacy `?key=YOUR_API_KEY` query parameter also works but must be avoided in this project: any component that logs the request URL (access logs, reverse-proxy logs, Go's own `http.Client` error strings which sometimes include the URL) would leak the key into a log file, which is the exact class of problem DR-2-01 already flags for the OTP code. [CITED: ai.google.dev/gemini-api/docs/generate-content/api-key; cross-checked via WebSearch against community/API-reference sources]

### Request body shape

```json
{
  "systemInstruction": { "parts": [{ "text": "<tier two: conduct rules + approach guidance>" }] },
  "contents": [
    { "role": "user", "parts": [{ "text": "<tier one: the recent-text window>" }] }
  ],
  "generationConfig": {
    "responseMimeType": "application/json",
    "responseSchema": {
      "type": "object",
      "properties": {
        "reasoning": { "type": "string" },
        "command":   { "type": "string" }
      },
      "required": ["reasoning", "command"]
    }
  }
}
```
`systemInstruction` currently supports text-only parts. `responseSchema` requires `responseMimeType: "application/json"` to be set alongside it. [CITED: ai.google.dev/api/generate-content; firebase.google.com/docs/ai-logic/generate-structured-output]

### Response body shape (success)

```json
{
  "candidates": [
    { "content": { "parts": [{ "text": "{\"reasoning\":\"...\",\"command\":\"north\"}" }] } }
  ],
  "usageMetadata": { "...": "token counts" }
}
```
The model's structured-JSON answer arrives as a **string** inside `candidates[0].content.parts[0].text` — it must be JSON-unmarshaled a second time (the outer HTTP response is JSON; the inner `text` field is also a JSON string, per Gemini's `responseSchema` contract). This double-decode is a real implementation detail, not an edge case, and is exactly the kind of thing D-13's "malformed answer" failure category exists to catch (empty `candidates`, a `text` field that fails the second `json.Unmarshal`, a `command` field containing more than one line, etc. are all D-13 failures). [CITED: ai.google.dev/api/generate-content]

### Error shapes

```json
{ "error": { "code": 429, "message": "...", "status": "RESOURCE_EXHAUSTED" } }
```
- **400** — malformed request body or invalid `generationConfig` (e.g. a broken `responseSchema`). Treat as a D-13 failure.
- **401 / 403** — API key missing, invalid, expired, or lacking permission for the model. This is the D-20 "configured but wrong" case — distinct from D-20's "absent" case (which is caught before the call is ever made) — and must still land as a D-13 failure, not a crash.
- **429 `RESOURCE_EXHAUSTED`** — rate or quota exceeded (D-19: "a rate-limit answer from Gemini is a failure under D-13"). [CITED: ai.google.dev/gemini-api/docs/api-errors, ai.google.dev/gemini-api/docs/troubleshooting]

### Free-tier rate limits — ASSUMED, not verified

Google's own rate-limits documentation explicitly declines to publish fixed numbers: "Rate limits depend on a variety of factors (such as your usage tier) and can be viewed in Google AI Studio" — it directs the developer to a per-account dashboard rather than stating RPM/RPD figures. [CITED: ai.google.dev/gemini-api/docs/rate-limits] Third-party aggregator sites report figures such as "10 RPM / 250 RPD for `gemini-2.5-flash`" as of 2026, but these are **not** an authoritative source and are **not corroborated** by Google's own docs — tagged `[ASSUMED]` per this project's provenance rule. **Do not hard-code a specific numeric assumption about the free tier's request rate anywhere in the implementation.** The design already has the correct mechanical answer regardless of the exact number: the profile's `CallCap` and D-13's "any failure disengages, no retry" rule mean a 429 is handled correctly without the code needing to know the limit in advance.

### Model naming — time-sensitive, do not hard-code a default in Go

As of this research date, `gemini-2.5-flash-lite` (a natural "cheap free-tier default" candidate) is scheduled for shutdown October 16, 2026 (Developer API) / October 20, 2026 (Agent Platform), and `gemini-flash-latest` is documented as a rolling alias that currently resolves to a newer model family. [CITED: ai.google.dev, cross-checked via WebSearch] This directly reinforces D-18/CON-model-config: the default model name must come from environment configuration (e.g. `AI_MODEL_DEFAULT_NAME=gemini-flash-latest` or a specific pinned ID the owner chooses at deploy time), and the Go code must never hard-code a fallback model ID as a last resort — D-20 already requires refusing engagement cleanly when the default model name is absent from the environment, which is the correct behavior here too if the owner's configured model is ever retired without the env var being updated (the failure surfaces as a Gemini 400/404 for an unknown model, which is a D-13 failure, not a crash).

## Architecture Patterns

### System Architecture Diagram

```
#AUTO ON (Phase 2 grammar, engage gate already passes)
   │
   ▼
internal/session.Handler.Autopilot()  action == "on"
   │  NEW: after EngageAutopilot succeeds, invoke the driver
   │  (composed with the existing EngageGate callback — see Pattern 6)
   ▼
NEW driver.HandleEngage(userID, connectionID) — fires once per D-01/D-02,
never on an already-on no-op (D-03)
   │
   ├─► internal/session.Manager.RecentOutputSnapshot(userID)  (Pattern 1)
   │      returns ANSI-stripped, telnet-already-stripped text,
   │      up to the ring buffer's bound, INCLUDING pre-engage text (D-04)
   │
   ├─► internal/store.ProfileStore.GetProfileByConnection(...)
   │      → ConductRules, ApproachGuidance (tier two, D-05)
   │
   ▼
internal/gemini.GenerateContent(ctx, model, apiKey, systemInstruction, window)
   │  POST https://generativelanguage.googleapis.com/v1beta/models/{model}:generateContent
   │  header: x-goog-api-key   body: systemInstruction + contents + responseSchema
   │
   ├─ error (network, 400, 401/403, 429) ──► D-13 failure path (see below)
   │
   ▼
parse candidates[0].content.parts[0].text as {reasoning, command}
   │
   ├─ empty candidates / bad JSON / empty command / multi-line command /
   │  command starts with '#','@','$','%' ──► D-13 failure path
   │
   ▼
NEW: build *icm.NormalizedCommand{Command: command, Operator: "", RequiresExecution: true}
ctx := icm.ContextAutomation
result, err := dispatcher.Dispatch(&ctx, sessionID, normalizedCmd)
   │  NOTE: this is internal/icm.Dispatcher.Dispatch called DIRECTLY —
   │  NEVER internal/icm.Engine.Process(), which short-circuits pass-through
   │  commands before Dispatch is ever reached (see Summary/Pitfall 1)
   │
   ├─ err != nil (circuit breaker / rate limit / queue depth tripped)
   │      ──► D-13 failure path — the driver's own AI call never reaches
   │          the MUD; ICM's safety limits are what stopped it
   │
   ▼ err == nil (result is nil too — no handler for a plain command,
   │             which IS "approved, pass through")
manager.SendCommand(userID, command)     ← the driver sends it, ICM did not
   │
   ▼
NEW: store ai_decisions row (window, reasoning, command, outcome=sent)
NEW: push MsgTypeAI websocket message {reasoning, command, outcome}
   │
   ▼
Browser: AIAssistPanel shows reasoning+command; PlayScreen prints
`[AI-ASSIST > north]` in place of local echo for this line; the game's
own response to "north" arrives via the existing MUD→client relay
unchanged (the driver never touches that path)

D-13 failure path (any failure kind):
   NEW: store ai_decisions row (outcome=refused|failed, reasoning/command
        as available, error text)
   NEW: push MsgTypeAI system-line message with the failure notice
   manager.DisengageAutopilot(userID, "ai-failure")   ← lands on OFF
   Browser: panel + terminal both show the notice (D-13); badge flips to
   Off via the existing autopilot-state websocket/REST path (Phase 2,
   unchanged)

Separately, the always-on transcript tap (D-14, independent of AI/autopilot):
Manager.SendCommand(userID, command, source)  ← NEW source param threaded in
Manager.ReadOutput(userID, buffer)            ← NEW: also appends to the
                                                 per-user transcript buffer
   both write to a NEW per-connection game_session_lines store, tagged
   human/ai/game, keyed to the CURRENT open game_sessions row for that
   (userID, connectionID) pair; opened in Manager.Connect (profile
   connections only, D-14), closed in Manager.Disconnect
```

### Recommended Project Structure

```
internal/
├── driver/                    # NEW package — the phase's orchestration
│   ├── driver.go              #   HandleEngage(userID, connectionID string):
│   │                          #   snapshot → prompt → Gemini → Dispatch →
│   │                          #   SendCommand → persist → broadcast
│   └── driver_test.go         #   table tests against fake Gemini client +
│                              #   fake Dispatcher (interface-satisfying stub)
├── gemini/                    # NEW package — the REST client only
│   ├── client.go              #   GenerateContent(ctx, cfg, systemInstruction,
│   │                          #   userText) (*Answer, error); no MUD/ICM
│   │                          #   knowledge lives here
│   └── client_test.go         #   httptest.Server-backed tests for the
│                              #   request shape, the double-JSON-decode,
│                              #   and every error status code
├── session/
│   ├── manager.go             # extended: outputWindow map[string]*ringBuffer,
│   │                          #   RecentOutputSnapshot(userID) string,
│   │                          #   SendCommand gains a source param
│   ├── window.go              # NEW — ring buffer + ANSI strip (Pattern 1)
│   ├── window_test.go         # NEW
│   ├── transcript.go          # NEW — game_sessions/game_session_lines tap,
│   │                          #   opened in Connect, closed in Disconnect,
│   │                          #   gated on ConnectionID != "" (D-14)
│   ├── transcript_test.go     # NEW
│   ├── handler.go             # extended: Autopilot's "on" branch invokes
│   │                          #   driver.HandleEngage after EngageAutopilot
│   │                          #   succeeds; a second callback (missing-config
│   │                          #   refusal, Pattern 6) composed with EngageGate
│   └── websocket.go           # extended: new MsgTypeAI outbound type
├── icm/                       # UNCHANGED — Dispatch/SafetyChecker/
│                              #   ExecutionContext/ContextAutomation all
│                              #   already exist and need no new code;
│                              #   only cmd/server/main.go wires it in
├── config/
│   └── config.go              # extended: ModelRegistry (map[string]ModelEntry),
│                              #   DefaultModelName; absence is NOT fatal (D-20)
├── store/
│   ├── decisions.go           # NEW — DecisionStore, same NewXStore(db) shape
│   │                          #   as ProfileStore/ConnectionStore
│   └── transcripts.go         # NEW — game_sessions/game_session_lines reads
│                              #   for the Logs page
└── auth/
    └── handler.go              # extended: gate the two "STAGING: OTP sent to
                                 #   user, code: %s" log.Printf calls (lines
                                 #   ~181, ~270) behind a new env flag,
                                 #   default off (DR-2-01)

migrations/
├── 011_add_ai_decisions.up.sql    # ai_sessions (stints), ai_decisions
└── 011_add_ai_decisions.down.sql
├── 012_add_session_transcripts.up.sql   # game_sessions, game_session_lines
└── 012_add_session_transcripts.down.sql
   (or a single 011 covering all four tables — see Table Shapes below;
   either is Claude's Discretion, one migration is simpler and matches
   "no speed bumps")

cmd/server/
└── main.go                     # extended: icm.NewEngine() constructed once,
                                 #   icmHandler.RegisterRoutes(mux), driver
                                 #   wired with sessionManager + profileStore +
                                 #   geminiClient + decisionStore, passed into
                                 #   session.HandlerCallbacks

frontend/src/
├── components/
│   ├── AIAssistPanel.tsx       # NEW — floating/minimizable, D-06 to D-08
│   └── AutopilotBadge.tsx      # UNCHANGED — reference pattern only
├── services/
│   ├── api.ts                  # extended: onAI/offAI handler list (mirrors
│   │                           #   onAutopilot exactly); getDecisions(connectionId)
│   │                           #   REST call for refresh reload
│   └── automation/evaluator.ts # extended: D-11's narrower AUTO grammar
│                                #   (reject anything but ON/OFF with the
│                                #   exact wording) replaces Phase 2's
│                                #   STATUS branch
├── pages/
│   ├── PlayScreen.tsx           # extended: mounts AIAssistPanel when the
│   │                            #   connected profile is AI-activated (D-07);
│   │                            #   on an AI-issued command event, writes
│   │                            #   `[AI-ASSIST > command]` instead of the
│   │                            #   normal typed-echo path
│   └── SettingsPage.tsx         # extended: 'logs' added to SettingsSection
│                                #   union; nav entry; "Open logging" button
│                                #   with target="_blank"
└── pages/
    └── LogsPage.tsx              # NEW — SPA route (recommended over a
                                   #   server-rendered page, see Pattern 7),
                                   #   left pane session list, right pane
                                   #   transcript text, both from new
                                   #   scoped-read endpoints
```

### Pattern 1: Ring buffer + ANSI strip lives on `Manager`, fed from `ReadOutput`, snapshotted at engage time

**What:** A new `map[string]*ringBuffer` field on `Manager`, guarded by the existing `m.mu` (the same pattern Phase 2 used for the `autopilot` map — see `internal/session/manager.go:60-65`, four parallel maps under one `sync.RWMutex` already). `Manager.ReadOutput` (the sole MUD-read path, already confirmed exhaustively in Phase 2's research) appends every chunk it reads to the buffer for that `userID`, after telnet-IAC-stripping (call the existing `stripTelnetIAC` function, already in package `session`, directly — no need to duplicate its logic) and ANSI-CSI-stripping. A bounded byte-length ring (recommend ~8KB, roughly 2-4 terminal screens of plain text at 80 columns — comfortably inside Gemini's per-request token budget and satisfying D-04's "roughly the last screenful or two") overwrites oldest content first.

**ANSI stripping — a compiled regex is sufficient, not a state machine:**
```go
// NEW: internal/session/window.go
var ansiCSI = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripANSI(b []byte) []byte {
    return ansiCSI.ReplaceAll(b, nil)
}
```
This matches the standard CSI (`ESC [ ... letter`) form MUD servers use for color codes. A sequence split across two separate reads from the MUD socket (a real possibility given `readMUDOutput`'s 8KB-buffer, 50ms-poll loop) would fail to match and leave a partial escape fragment in the AI's window — a cosmetic imperfection (a few stray bytes of noise for the model to ignore), not a correctness bug, and not worth a full state-machine parser for a text snapshot that only needs to be "good enough for an LLM to read," unlike the browser's xterm.js rendering (which needs exact fidelity and is explicitly NOT touched by this stripping — the raw ANSI still flows to the browser terminal unchanged, exactly as it does today).

**Snapshot at engage time (D-04's "including text from before the switch was flipped"):** because the buffer is fed continuously from `ReadOutput` regardless of autopilot state, the very first `RecentOutputSnapshot(userID)` call the driver makes on `#AUTO ON` already contains pre-engage text for free — no special-casing needed.

### Pattern 2: Gemini client is a thin, testable `net/http` wrapper — no MUD or ICM knowledge

**What:** `internal/gemini.GenerateContent` takes only `(ctx context.Context, endpoint, apiKey, model, systemInstruction, userText string) (*Answer, error)` and returns a parsed `{Reasoning, Command string}` or a typed error distinguishing "config/auth problem" (400/401/403 — a D-20/D-13 boundary case) from "rate limited" (429 — D-19/D-13) from "malformed answer" (JSON decode failure on the *inner* `text` field — D-13). Keep the double-JSON-decode (outer HTTP body, then the model's `text` field, which is itself a JSON string per the `responseSchema` contract) inside this package so callers never see it.

```go
// Source: pattern derived from ai.google.dev/api/generate-content (verified this session)
req, _ := http.NewRequestWithContext(ctx, "POST",
    fmt.Sprintf("%s/v1beta/models/%s:generateContent", endpoint, model),
    bytes.NewReader(bodyJSON))
req.Header.Set("x-goog-api-key", apiKey)   // NOT ?key=... — see Gemini API Reference
req.Header.Set("Content-Type", "application/json")
resp, err := httpClient.Do(req)
// resp.StatusCode == 429 -> RateLimitedError
// resp.StatusCode == 401 || 403 -> AuthError
// resp.StatusCode == 400 -> BadRequestError
// resp.StatusCode == 200 -> unmarshal outer body, then unmarshal
//   candidates[0].content.parts[0].text as {reasoning, command} a second time
```

### Pattern 3: `Dispatcher.Dispatch` must be called directly for the AI's command — never `Engine.Process()`

**What goes wrong if ignored:** `Engine.Process()` (`internal/icm/engine.go:67-141`, verified) recognizes a plain string like `"north"` as `!recognizeResult.IsInternal`, classifies it via `passThroughClassifier.ClassifyCommand`, sees `ShouldPassThrough == true`, and **returns immediately** at line 102 with `resp.Normalized = req.Raw` — `e.dispatcher.Dispatch(...)` at line 130 is never reached, because that call only executes when `authority.CanExecute && normalized.RequiresExecution`, and the pass-through early-return happens before `normalized` is even computed. A driver that called `engine.Process(&icm.ICMRequest{Raw: command, Context: icm.ContextAutomation, ...})` expecting the safety checker to run would see it silently *not* run, while still believing (because the call "succeeded") that the command passed through ICM's safety limits. Success criterion 4 requires this to be provably false — the diagnostic `go test` must exercise the actual code path the driver uses.

**The correct call (verified against `dispatcher.go` in full, this session):**
```go
// Source: pattern derived from internal/icm/dispatcher.go's actual Dispatch signature
ctx := icm.ContextAutomation
normalized := &icm.NormalizedCommand{
    Command:           command,   // e.g. "north" — the AI's returned command, already
                                   // validated to be a single plain line (D-13)
    Operator:          "",        // empty family: no # / @ / $ / % prefix
    RequiresExecution: true,      // irrelevant to Dispatch itself, but keeps the
                                   // struct honest if anything else inspects it later
}
result, icmErr := dispatcher.Dispatch(&ctx, sessionID, normalized)
if icmErr != nil {
    // circuit breaker tripped, rate limited, or queue depth exceeded —
    // treat as a D-13 failure; the command is NOT sent to the MUD
    return
}
// result is nil here too (no handler registered for an empty operator
// family) — that IS "approved, pass through, nothing further ICM needs to do"
if err := manager.SendCommand(userID, command); err != nil { ... }
```
`Dispatch`'s own `checkSafety` (`dispatcher.go:240-259`) runs unconditionally for any non-preview context, in this exact order: circuit breaker → rate limit → queue depth — before `getHandler` is even consulted. This is genuinely, provably "dispatched through the ICM engine's dispatcher and safety checker" in the sense success criterion 4 requires, and is trivially unit-testable with `NewDispatcher(NewRegistry())` directly (see `icm_test.go`'s own `TestDispatcher_SafetyEnforcement`, which already exercises `d.safety` this same way without any HTTP or MUD machinery).

**Where the ICM engine instance lives:** `cmd/server/main.go` today constructs zero `icm.Engine` instances — an exhaustive grep this session (`grep -rn "icm\." cmd/ internal/session internal/profiles`) returns nothing outside `internal/icm` itself. Phase 3 must call `icm.NewEngine()` once in `main.go` (mirroring how `sessionManager`, `profileStore`, etc. are each constructed once and threaded through), register `/api/v1/icm/*` routes via `icmHandler.RegisterRoutes(mux)` (the method already exists, unused, at `internal/icm/handler.go:23`), and pass `engine.GetDispatcher()` into the new `driver` package's constructor.

### Pattern 4: New `MsgTypeAI` websocket message — a typed payload field, not string-cramming

**What:** Phase 2's `MsgTypeAutopilot` reused the existing `Status`/`Data` string fields because an autopilot transition only ever carries two short strings (state, cause). A decision carries three logically distinct pieces of variable-length text (reasoning, command, outcome) plus an id for later reference — cramming these into `Data` as an escaped JSON string works but fights Go's type system for no benefit, since `WSMessage` is already extensible. Add one new optional field:

```go
// Source: extends internal/session/websocket.go's existing WSMessage
type WSMessage struct {
    Type   string `json:"type"`
    // ... existing fields unchanged ...
    // Decision carries an AI decision or system notice for MsgTypeAI.
    // Nil on every other message type.
    Decision *AIDecisionPayload `json:"decision,omitempty"`
}

type AIDecisionPayload struct {
    ID        string `json:"id"`
    Kind      string `json:"kind"`      // "decision" | "system"
    Reasoning string `json:"reasoning,omitempty"`
    Command   string `json:"command,omitempty"`
    Outcome   string `json:"outcome,omitempty"`   // "sent" | "refused" | "failed"
    Message   string `json:"message,omitempty"`   // system-line text (D-13 notices, D-11 unknown-option line if surfaced here)
    Timestamp string `json:"timestamp"`
}
```
This follows the exact same "add one field, reuse the existing switch" extension Phase 2 used for `Source`/`ConnectionID`. Frontend: `WebSocketManager` gains `onAI`/`offAI` handler lists, mirroring `onAutopilot`/`offAutopilot` verbatim (`api.ts:246-251` is the template).

### Pattern 5: Decision persistence and refresh-reload — REST-on-attach, exactly like Phase 2's status re-sync

**What:** Phase 2's own research (`02-RESEARCH.md`) already established that the correctness mechanism for "state survives a page refresh" in this codebase is a REST GET called on mount/attach, not a websocket-attach message (no per-user connection registry exists to make the latter reliable). The same applies here: `GET /api/v1/session/decisions?connection_id=...` (new `session.Handler` method, or a new small handler — recommend keeping it on `session.Handler` alongside `Status`/`Autopilot` since it answers "what has this session's AI done," the same conceptual family) returns the ordered list of `ai_decisions` rows for the current connection, and `AIAssistPanel.tsx` calls it once on mount, exactly mirroring `SessionContext.tsx`'s `refreshStatus()` pattern.

### Pattern 6: The missing-config refusal composes with the existing `EngageGate` callback, it does not replace it

**What:** D-20 requires `#AUTO ON` to refuse with a clear message when Gemini config is absent, while the server itself starts normally (no `config.Load()` fatal). `session.HandlerCallbacks.EngageGate` already exists as exactly this shape: `func(connectionID, userID uuid.UUID) (allowed bool, message string, policyVersion string)`, called first for every `#AUTO` action including `status` (`handler.go:334-343`, verified). The cleanest composition is a second, independent callback checked in the same place:

```go
// Source: extends internal/session/handler.go's existing pattern
type HandlerCallbacks struct {
    // ... existing fields unchanged ...
    // AIConfigured reports whether a default model and its API key are
    // present in the environment (D-20). A nil callback must be treated
    // as "not configured" — fail closed, same discipline as EngageGate.
    AIConfigured func() (ok bool, message string)
}
```
In `Autopilot`'s `"on"` branch, check `AIConfigured` alongside `gateAllowed`: both must pass. This keeps Phase 1's policy-gate refusal message and Phase 3's missing-config refusal message as two distinct, independently-testable checks rather than one handler trying to encode two unrelated failure reasons in one string — and matches CONTEXT.md's own wording, which gives each its own exact notice (`store.EngageGateRefusalMessage` for the gate, `[Autopilot refused: AI is not configured on this server]` for D-20).

### Pattern 7: Log page as an SPA route, not a server-rendered page

**What:** This codebase has exactly one delivery mechanism for browser UI today: the React SPA served by `main.go`'s catch-all static-file handler (`publicDir`, falls back to `index.html` for any non-file path — verified at `main.go`'s closing block). A server-rendered log page would be a second, parallel rendering mechanism for one page, in a codebase that has never had one. `target="_blank"` on a plain `<a href="/logs/{connection_id}">` (or a `window.open` call) opening a new SPA route (`LogsPage.tsx`, added to the existing React Router configuration) satisfies D-16's "opens in a new tab, current tab does not navigate" with zero new server-side rendering machinery — the new tab is simply a second instance of the same SPA, landing directly on the logs route, protected by the same `sessionMiddleware` and per-user scoping every other `/api/v1/*` route already has.

### Anti-Patterns to Avoid

- **Calling `icm.Engine.Process()` for the AI's command.** See Pattern 3 — this is the single most consequential mistake available in this phase; it compiles, appears to work in a screen-recorded demo (the driver's own `manager.SendCommand` call, if written defensively "just in case," would still send the command even though `Process()` did nothing), and only fails the diagnostic test that specifically checks whether the safety checker ran.
- **Hard-coding any Gemini model ID as a Go-level fallback default.** D-18/D-20 already require env-sourced configuration with a clean refusal when absent; a hard-coded fallback both violates CON-model-config and risks silently calling a model Google has since retired (see the `gemini-2.5-flash-lite` shutdown date above).
- **Passing the Gemini API key as a `?key=` query parameter.** Use the `x-goog-api-key` header — see Gemini API Reference and DR-2-01's precedent.
- **Storing the ring buffer or the transcript buffer as a field on `Session`.** Same Phase 2 Pitfall 1 applies verbatim: anything that needs to survive a disconnect (the transcript does not need to survive a disconnect — it closes at disconnect, D-14 — but the *decision* the AI made must survive it since D-12 says decisions persist across a refresh) must not live on the struct that `Manager.Disconnect` deletes. The ring buffer specifically may reasonably be cleared or preserved across a disconnect (D-02's WAITING→ON resume implies the window should probably persist into the reconnect, so the post-reconnect decision's window includes the pre-drop tail) — recommend keeping it on the same independent `Manager`-level map as autopilot state, not on `Session`, for exactly this reason.
- **Building a bespoke JSON-schema validation library for the Gemini answer.** `encoding/json`'s `Unmarshal` into a `struct{ Reasoning, Command string }` already fails cleanly on missing/wrong-typed fields; combined with Gemini's own `responseSchema` constraining the *model's* output shape server-side, no additional schema-validation package is justified.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Command safety gating for the AI's issued command | A second, parallel rate-limiter/circuit-breaker inside the new `driver` package | `internal/icm.Dispatcher.Dispatch` + its `SafetyChecker` (already built, already unit-tested in `icm_test.go`) | This is the entire point of "issues the returned command through the automation context"; a parallel safety mechanism would make the ICM wiring cosmetic |
| Gemini JSON-schema-constrained output | A custom prompt-engineering "please reply in this exact format" with regex post-parsing | `generationConfig.responseSchema` (official, documented, server-enforced by Google) | Far more reliable than string-parsing free text; also directly gives D-13's "malformed answer" a clean failure signal (JSON decode error) instead of ambiguous text-parsing heuristics |
| ANSI-aware terminal rendering for the AI's text window | A full terminal-emulation library (e.g. a Go port of xterm state machine) | A single compiled regex stripping CSI sequences (Pattern 1) | The AI window is a one-shot text snapshot for an LLM prompt, not a rendered display; xterm.js already owns real terminal rendering for the human on the browser side and is untouched |
| Env var parsing for the model registry | A generic config/env library (viper, envconfig, etc.) | Extend `internal/config.Load()`'s existing hand-rolled `os.Getenv` + `strconv` pattern | Zero other config in this codebase uses a config library; adding one for four new variables is disproportionate and inconsistent |
| Decision/transcript persistence | An ORM or query builder | `database/sql` + hand-written SQL, same as every existing `*Store` type (`ProfileStore`, `ConnectionStore`, `CredentialsStore`) | Matches the established convention exactly; this codebase has zero ORM usage anywhere |

**Key insight:** every piece of new infrastructure this phase needs either already exists and is dormant (`internal/icm`'s dispatcher and safety checker), or is a single, narrow addition that follows an existing convention in this codebase exactly (a new `*Store`, a new `Manager`-level map under the existing mutex, a new `HandlerCallbacks` field, a new `WSMessage` field). The phase's actual engineering risk is entirely in the ICM call-shape subtlety (Pattern 3) and in not letting the Gemini integration accumulate scope creep (an SDK, a config library, a schema-validation library) that this codebase's own conventions do not support.

## Common Pitfalls

### Pitfall 1: Calling `Engine.Process()` instead of `Dispatcher.Dispatch()` directly (see Pattern 3)
**What goes wrong:** The safety checker never runs; success criterion 4 is false even though the feature appears to work end-to-end in a demo.
**Why it happens:** `Process()` is the "obvious" public entry point (it is what the frontend's `icm-adapter.ts` already calls for every other command path), and its pass-through short-circuit is not visible without reading `engine.go` line-by-line.
**How to avoid:** Call `dispatcher.Dispatch()` directly with a hand-built `NormalizedCommand`, as shown in Pattern 3. Write the diagnostic `go test` against `Dispatcher` directly (as `icm_test.go`'s own `TestDispatcher_SafetyEnforcement` already does), not against `Engine.Process()`.
**Warning signs:** A test that calls `engine.Process()` and asserts on `resp.Result` would pass trivially (because pass-through always "succeeds") without ever exercising `d.safety`.

### Pitfall 2: `Dispatch`'s `RecordExecution` never fires for the AI's plain commands
**What goes wrong:** `Dispatch` only calls `d.safety.RecordExecution(sessionID, normalized.Command)` *after* a handler successfully executes (`dispatcher.go:220-221`). Since no handler is registered for an empty operator family, that line is never reached for the AI's `"north"`-style commands — meaning the AI's own command history does **not** accumulate against ICM's own rate-limit/circuit-breaker counters, even though `checkSafety` (which reads those same counters) does run and gates the call.
**Why it happens:** `Dispatch`'s safety-check and safety-record steps are asymmetric by design — they were built for `#`-directive handlers, not passthrough game commands, and Phase 3 is the first caller to route a passthrough command through `Dispatch` at all.
**How to avoid:** For Phase 3 (a single decision, at most one command per engage), this has no observable effect — success criterion 4 only requires the safety *check* to run, which it does. Flag this explicitly for Phase 4: once the AI issues commands in a loop, either `RecordExecution` needs to be called explicitly by the driver after a successful `SendCommand` (so ICM's circuit breaker/rate-limit state actually reflects AI activity), or Phase 4's own call-cap mechanism (a separate, profile-level counter, not ICM's) is the sole rate protection and ICM's counters are understood to be decorative for this command class. This is an Open Question for the planner to carry into Phase 4, not a Phase 3 blocker.
**Warning signs:** A test that issues the AI's command 200 times in a tight loop and expects ICM's own circuit breaker to trip based on that traffic alone would fail; nothing in Phase 3 does this (D-03: exactly one decision per engage), so it is out of scope here.

### Pitfall 3: Treating a `?key=` query parameter as equivalent to the `x-goog-api-key` header
**What goes wrong:** The API key ends up in any log, trace, or error message that records the request URL — recreating DR-2-01's exact failure class (a secret visible in a log an unrelated party with log access could read) for a brand-new secret.
**Why it happens:** Most tutorial-era Gemini examples (and this project's own training-data-era knowledge) show `?key=...` because it is the older, still-functional form; the header form is Google's current preference but less prominently shown in casual examples.
**How to avoid:** Always set `x-goog-api-key` as a request header; never format the API key into the URL string anywhere, including in log lines that print the request for debugging.
**Warning signs:** Any `log.Printf` that includes a full outbound request URL for the Gemini call — the URL must be logged with the key redacted or omitted entirely, consistent with how this project already avoids logging PII/secrets elsewhere (`internal/session/manager.go`'s `logConnectionMetadata`/`logDisconnectMetadata` comments explicitly say "no PII").

### Pitfall 4: Storing the AI's text window or the transcript buffer keyed only by `Session`
**What goes wrong:** Same class of bug as Phase 2's Pitfall 1 (autopilot state on `Session` gets deleted by `Manager.Disconnect`). If the ring buffer is a `Session` field, it is destroyed and recreated empty on every reconnect — breaking D-02's "resume fires a fresh decision on the post-reconnect text," since "post-reconnect text" is supposed to build on whatever text existed right up to the drop, not start from nothing at the exact moment of resume.
**Why it happens:** Same as Phase 2 — `Session` looks like the natural home for "this session's recent output."
**How to avoid:** A separate `map[string]*ringBuffer` on `Manager`, independent of the `sessions` map's lifecycle, exactly like the `autopilot` map (Pattern 4 in `02-RESEARCH.md`, directly reusable here).
**Warning signs:** A test that disconnects and reconnects mid-buffer and asserts the pre-disconnect tail is still present after resume would fail if the buffer lives on `Session`.

### Pitfall 5: The double-JSON-decode for Gemini's structured answer
**What goes wrong:** A naive implementation unmarshals the outer HTTP response body and stops, treating `candidates[0].content.parts[0].text` as the final answer — but that field is itself a JSON-encoded string (per the `responseSchema` contract), not already-parsed `{reasoning, command}` fields. Forgetting the second `Unmarshal` produces a "reasoning" that is literally the string `{"reasoning":"...","command":"..."}` displayed verbatim to the owner in the panel.
**Why it happens:** It is easy to assume `responseSchema` makes the outer JSON directly contain the structured fields; in fact Gemini always wraps model output in the `candidates[].content.parts[].text` envelope regardless of `responseMimeType`.
**How to avoid:** Always decode twice: once for the Gemini API envelope, once for the model's own JSON text.
**Warning signs:** The AI Assist panel showing raw JSON-looking text as the "reasoning" instead of a plain sentence.

## Code Examples

### ICM Dispatch call for a plain AI-issued command (the phase's central mechanism)
```go
// Source: pattern derived from internal/icm/dispatcher.go (verified in full this session)
func (d *Driver) dispatchAndSend(userID, sessionID, command string) error {
    ctx := icm.ContextAutomation
    normalized := &icm.NormalizedCommand{
        Command:           command,
        Operator:          "",
        RequiresExecution: true,
    }
    if _, icmErr := d.dispatcher.Dispatch(&ctx, sessionID, normalized); icmErr != nil {
        return fmt.Errorf("icm refused: %s", icmErr.Message) // D-13 failure
    }
    return d.manager.SendCommand(userID, command)
}
```

### Diagnostic test shape proving success criterion 4 (mirrors `icm_test.go`'s existing `TestDispatcher_SafetyEnforcement`)
```go
// Source: pattern, following internal/icm/icm_test.go's own established style
func TestDispatch_AutomationPassThrough_GoesThroughSafetyChecker(t *testing.T) {
    d := icm.NewDispatcher(icm.NewRegistry())
    ctx := icm.ContextAutomation
    // Trip the rate limiter first, exactly as TestDispatcher_SafetyEnforcement does
    for i := 0; i < 100; i++ {
        d.GetSafetyChecker().RecordExecution("sess-1", "north")
    }
    _, icmErr := d.Dispatch(&ctx, "sess-1", &icm.NormalizedCommand{Command: "north"})
    if icmErr == nil {
        t.Fatal("expected Dispatch to refuse once the rate limit is tripped")
    }
}
```

### Gemini request construction (the header form)
```go
// Source: pattern derived from ai.google.dev/api/generate-content (verified this session)
body, _ := json.Marshal(map[string]any{
    "systemInstruction": map[string]any{"parts": []map[string]string{{"text": tierTwo}}},
    "contents": []map[string]any{{"role": "user", "parts": []map[string]string{{"text": tierOne}}}},
    "generationConfig": map[string]any{
        "responseMimeType": "application/json",
        "responseSchema": map[string]any{
            "type": "object",
            "properties": map[string]any{
                "reasoning": map[string]string{"type": "string"},
                "command":   map[string]string{"type": "string"},
            },
            "required": []string{"reasoning", "command"},
        },
    },
})
req, _ := http.NewRequestWithContext(ctx, "POST",
    endpoint+"/v1beta/models/"+model+":generateContent", bytes.NewReader(body))
req.Header.Set("x-goog-api-key", apiKey) // never ?key=... in the URL
req.Header.Set("Content-Type", "application/json")
```

### Existing `EngageGate` callback shape, the precedent for D-20's composed refusal (Pattern 6)
```go
// Source: internal/session/handler.go (verbatim, existing)
EngageGate func(connectionID, userID uuid.UUID) (allowed bool, message string, policyVersion string)
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| Gemini API key via `?key=` query parameter | `x-goog-api-key` HTTP header | Current Google guidance as of this research date | Avoids the key appearing in any URL-logging log line — directly relevant given DR-2-01 |
| `gemini-2.5-flash-lite` as a "safe default cheap model" | Model scheduled for shutdown Oct 16/20, 2026; `gemini-flash-latest` is a rolling alias | Announced ahead of this research date | Reinforces why the default model name must be env-configured, never hard-coded in Go |
| ICM engine unused | Wired into `cmd/server/main.go` for the first time this phase | This phase | The dormant `internal/icm` package (confirmed unimported anywhere outside itself) finally has a caller |

**Deprecated/outdated:** `gemini-2.5-flash-lite` — do not recommend as a Go-level fallback constant given its announced shutdown date within this project's likely timeline.

## Runtime State Inventory

Not applicable — this is a greenfield-within-brownfield feature phase (new tables, new packages, new UI), not a rename/refactor/migration phase. No existing stored data, live service config, OS-registered state, or build artifact carries a name that this phase changes.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Gemini free-tier numeric rate limits (commonly cited as ~10 RPM / 250 RPD for `gemini-2.5-flash`-class models) | Gemini API Reference | Low — the design does not depend on knowing the exact number; D-19 already treats any 429 as a D-13 failure and the profile's `CallCap` is the owner-controlled protection regardless of the true limit. Flagged `[ASSUMED]` because Google's own docs decline to publish it; do not present this number to the owner as verified fact. |
| A2 | `x-goog-api-key` header is preferred over `?key=` query parameter and both currently work | Gemini API Reference, Pattern 2 | Low — cross-checked against two independent sources (official docs fetch + WebSearch corroboration); if the header form were ever deprecated the fallback (`?key=`) still exists as a documented alternative, so the client can be updated without redesign. |
| A3 | The recommended model naming (`gemini-flash-latest` as a rolling alias, `gemini-2.5-flash-lite` shutdown dates) | Gemini API Reference, State of the Art | Low — this affects only what the owner types into the env var as the default model name, not any code path; D-20's refusal-on-failure behavior already covers the case where a configured model name stops working. |
| A4 | `internal/driver` as a new top-level package (rather than adding driver logic directly inside `internal/session`) | Architectural Responsibility Map, Recommended Project Structure | Low — CONTEXT.md leaves this to Claude's Discretion implicitly (it specifies file/function touch points in `internal/session` for the *existing* hooks the driver calls, not where the driver's own orchestration code must live); a new package keeps `internal/session` from growing a Gemini/ICM/prompt-assembly dependency graph it does not otherwise need, but the planner may reasonably choose to inline it into `internal/session` instead with no behavioral difference. |
| A5 | Two separate tables (`ai_sessions`/stints and `ai_decisions`) plus two more (`game_sessions`/`game_session_lines`) rather than a single combined schema | Recommended Project Structure, Table Shapes | Low — CONTEXT.md explicitly defers this to Claude's Discretion ("table shapes and migration numbers... whether the transcript is one row per session with appended text or one row per line"); the four-table shape is recommended because it directly matches the Logs page's two-pane UI (D-17) and the stint-based debrief grouping Phase 6 will need (D-15), but a simpler two-table version (one for decisions, one appended-text-blob transcript table) would also satisfy Phase 3's own success criteria alone. |
| A6 | DR-2-01's fix is an env flag gating the existing `log.Printf("STAGING: OTP sent to user, code: %s", otp)` lines, rather than hashing the code | Architectural Responsibility Map | Low — CONTEXT.md's own discretion note lists both options ("gate the sign-in code log line behind an env flag that is off by default, or log a hash") as acceptable; an env flag is recommended because it requires no change to what the log line contains when explicitly enabled for debugging, only whether it prints at all, and is a one-line change at each of the two call sites identified this session (`internal/auth/handler.go` lines ~181 and ~270). |

**If this table is empty:** N/A — six assumptions are logged above, none rated above Low risk.

## Open Questions (RESOLVED — see plans 03-05 and 03-13)

1. **ICM's `RecordExecution` gap for AI-issued passthrough commands (Pitfall 2).**
   - What we know: `Dispatch`'s safety *check* runs for every non-preview-context call regardless of whether a handler exists; `RecordExecution` (which feeds the circuit breaker and rate-limiter's own counters) only runs after a handler successfully executes, and no handler exists for plain passthrough commands.
   - What's unclear: whether Phase 4's continuous loop should call `d.safety.RecordExecution` explicitly after every AI-issued command (making ICM's own counters meaningful for AI traffic), or rely entirely on Phase 4's own profile-level call cap and treat ICM's counters as decorative for this command class.
   - Recommendation: not a Phase 3 decision (D-03 caps this phase at exactly one command per engage, so the gap has zero observable effect here) — carry this forward explicitly into Phase 4's research/planning rather than silently discovering it there.

2. **Single migration vs. two migrations for the four new tables.**
   - What we know: CONTEXT.md says "migration numbers (`011`+)" (plural-friendly), and this codebase's convention (`006_create_profiles`, `010_add_ai_fields`) has historically been one migration per cohesive feature addition, not one per table.
   - What's unclear: whether the planner prefers one `011` migration covering `ai_sessions`, `ai_decisions`, `game_sessions`, `game_session_lines` together (simpler, matches "no speed bumps"), or splits AI-decision tables from transcript tables into `011`/`012` for cleaner rollback granularity.
   - Recommendation: one `011` migration — this phase ships all four tables together as one coherent capability; a split rollback granularity has no demonstrated need yet.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | Building/testing the backend | ✓ | go1.26 [VERIFIED: go.mod `go 1.26`] | — |
| Gemini API reachability from Railway staging | The actual `#AUTO ON` decision call | Not verified this session (network access at execution time, not research time) | — | The diagnostic `go test` for ICM dispatch and for the Gemini client's request/parsing logic (via `httptest.Server`) require no live Gemini call at all; only the staging player-observable evidence needs one |
| `GEMINI_API_KEY` / default model env vars on staging | Engagement to succeed at all | **Not yet set** [confirmed by STATE.md's own Phase 3 blocker note: "Gemini env vars (model names, API key) must be set on Railway staging before Phase 3 verification can run"] | — | None for the live/player-observable evidence; the missing-config-refusal path (D-20) is itself testable *without* the vars being set, and should be the first thing verified on staging before they are added |
| A live MUD reachable from staging | Player-observable verification of the issued command's game response | Not verified this session | — | Same as Phase 2: the Go tests need no live MUD; only the staging screenshot/log evidence does |
| Railway staging environment | Deployment target for evidence capture | ✓ (same environment Phases 1-2 already used successfully) | — | — |
| Frontend test tooling (vitest/jest) | N/A this phase | ✗ | — | Unchanged from Phases 1-2: none exists; frontend verification is player-observable screenshots |

**Missing dependencies with no fallback:** `GEMINI_API_KEY`/default-model env vars are not yet set on staging (confirmed by `STATE.md`) — this blocks the player-observable half of Phase 3's evidence (a live decision and its reasoning on staging) until the owner or the planner arranges for them to be set. It does **not** block writing or unit-testing any of the phase's code, including the D-20 refusal path, which is exercisable today precisely because the vars are absent.

**Missing dependencies with fallback:** frontend automated testing (fallback: screenshots, unchanged from Phases 1-2).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` [VERIFIED: same as Phases 1-2; no mocking library anywhere in this codebase] |
| Config file | none — `go test` needs none |
| Quick run command | `go test ./internal/session/... ./internal/icm/... ./internal/gemini/... ./internal/driver/... ./internal/store/... -v` |
| Full suite command | `go test ./...` (matches `.github/workflows/ci.yml`, unchanged) |
| Recommended addition | `-race` on `internal/session/...` given the new ring-buffer and transcript maps are additional concurrent state under `m.mu`, same discipline Phase 2's research already flagged |

**Frontend test tooling:** still none (`frontend/package.json`'s `"test": "echo 'No tests yet' && exit 0"`, unchanged). Frontend verification for this phase is player-observable browser walkthrough plus screenshots, exactly as ROADMAP's Phase Validation line specifies ("Player-observable on staging").

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| REQ-single-decision | A plain AI-issued command in `ContextAutomation` is dispatched through `Dispatcher.Dispatch` and its `SafetyChecker` before `manager.SendCommand` is ever called | unit | `go test ./internal/icm/... -run TestDispatch_AutomationPassThrough -v` | ❌ Wave 0 |
| REQ-single-decision | The driver reads the ring-buffer window, assembles tier one + tier two, and calls Gemini with the correct request shape (header auth, `responseSchema`) | unit (httptest-backed) | `go test ./internal/gemini/... -run TestGenerateContent -v` | ❌ Wave 0 |
| REQ-single-decision | End-to-end: engage fires exactly one decision, issues exactly one command, and a repeated `#AUTO ON` fires no second decision (D-03) | unit (fake Gemini + fake Dispatcher) | `go test ./internal/driver/... -run TestHandleEngage -v` | ❌ Wave 0 |
| REQ-single-decision | With autopilot engaged, one decision and its reasoning appear in the play screen and the game responds to the issued command | player-observable (browser, live MUD + live Gemini) | ROADMAP Phase Validation line | n/a — screenshot + staging log evidence |
| REQ-reasoning-visibility | A decision row is stored with window/reasoning/command/outcome and is returned by the decisions-reload endpoint after a page refresh | unit (httptest) | `go test ./internal/session/... -run TestDecisionsReload -v` | ❌ Wave 0 |
| REQ-reasoning-visibility | The panel shows the reasoning live via `MsgTypeAI`, and the terminal shows `[AI-ASSIST > command]` in place of local echo | player-observable (browser) | ROADMAP Phase Validation line | n/a — screenshot evidence |
| REQ-reasoning-visibility | A decisions table row exists for that decision and is returned after a page refresh | diagnostic (canned report against staging, via the decisions REST endpoint — never a direct database query) | `scripts/verify-phase3.sh` decisions-reload step | ❌ Wave 0 (script) |
| REQ-env-config | With the Gemini environment variables unset, `#AUTO ON` is refused with a clear message, the server starts normally, and no command is sent | unit | `go test ./internal/session/... -run TestAutopilotHandler_AIConfigRefusal -v` | ❌ Wave 0 |
| REQ-env-config | Same, demonstrated live on staging | player-observable + canned report | ROADMAP Phase Validation line; `scripts/verify-phase3.sh` | n/a — screenshot + report evidence |
| REQ-env-config | Model name/API key are read only from `internal/config`, never a literal in Go source | source assertion | `grep -rn "gemini-" internal/ --include=*.go` (expect zero matches outside `_test.go` fixtures and comments) | ❌ Wave 0 (grep step in the script) |

### Sampling Rate
- **Per task commit:** `go test ./internal/session/... ./internal/icm/... ./internal/gemini/... ./internal/driver/... -race -v`
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green, captured to `evidence/01-test-report.txt`; `scripts/verify-phase3.sh` run against staging for every REST-reachable behavior (missing-config refusal, decisions-reload); a staging `[AI-PLAYER]` log excerpt for the decision request/answer/dispatch/failure lines; end-user screenshots for the AI Assist panel showing a live decision, the `[AI-ASSIST > ...]` terminal line, the panel surviving a refresh, the D-20 refusal notice, and the Logs page's two-pane view — matching the owner's evidence rule (canned report or screenshot only, never a database query).

### Wave 0 Gaps
- [ ] `internal/gemini/client.go` + `internal/gemini/client_test.go` — new package; `httptest.Server`-backed tests for the request shape, header auth, the double-JSON-decode, and every error status code (400/401/403/429) — REQ-single-decision, REQ-env-config
- [ ] `internal/driver/driver.go` + `internal/driver/driver_test.go` — new package; fake-Gemini + fake-Dispatcher table tests proving exactly-one-decision-per-engage (D-01, D-02, D-03) and every D-13 failure kind sends nothing to the MUD — REQ-single-decision
- [ ] `internal/icm/dispatcher_test.go` addition (or a new file) — `TestDispatch_AutomationPassThrough_GoesThroughSafetyChecker`, proving the exact mechanism in Pattern 3 — REQ-single-decision (success criterion 4)
- [ ] `internal/session/window.go` + `internal/session/window_test.go` — ring buffer bound, ANSI-strip regex, and the "text from before engage is included" property (D-04) — REQ-single-decision
- [ ] `internal/session/transcript_test.go` — game_sessions/game_session_lines open-on-connect/close-on-disconnect, human/AI/game line tagging, quick-connect exclusion (D-14) — REQ-reasoning-visibility (session-log half)
- [ ] `internal/session/handler_test.go` addition — `TestAutopilotHandler_AIConfigRefusal`, proving D-20's composed-callback refusal — REQ-env-config
- [ ] `scripts/verify-phase3.sh` (new, following `scripts/verify-phase2.sh`'s exact shape: `set -uo pipefail`, PASS/FAIL per ROADMAP criterion, `--self-test`/`--self-test-negative` modes, a source-grep step for hard-coded model names) — REQ-env-config, REQ-reasoning-visibility (decisions-reload half)
- [ ] Framework install: none — `testing` is stdlib, already in use

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No (unchanged this phase) | Existing `sessionMiddleware` already wraps every new route |
| V3 Session Management | Yes | The new ring-buffer/transcript maps must be keyed by the authenticated `user_id` from context, same discipline as every existing `Manager` map |
| V4 Access Control | Yes | New decisions/transcript-read endpoints must resolve `connection_id` through `GetProfileByConnection(userID, connectionID)`, never trust a client-supplied "this is my profile" claim — same pattern as Phase 1/2's IDOR mitigation (T-1-03, T-2-02) |
| V5 Input Validation | Yes | The Gemini answer's `command` field must be validated as a single plain line with no leading `#`/`@`/`$`/`%` and no embedded newline before it ever reaches `Dispatch` — this is D-13's "malformed answer" and "a `#` directive or other non-game line" failure categories, and is also a security boundary (the model must never be allowed to issue an internal directive as if the owner typed it) |
| V6 Cryptography | No | The Gemini API key is a bearer secret transmitted over HTTPS to Google's endpoint; no new cryptographic primitive is implemented in this phase |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Prompt injection via game text: a hostile MUD room description or another player's speech could contain text designed to make Gemini "issue" a `#`-prefixed directive or a multi-command string instead of a single game command | Tampering / Elevation of Privilege | D-13's validation (single plain line, no `#`/`@`/`$`/`%` prefix, no newline) is the mechanical backstop regardless of what the model was tricked into producing; this must be enforced in Go before the command ever reaches `Dispatch`, not trusted to the model's own good behavior or to `responseSchema` alone (schema only constrains JSON shape, not command content) |
| API key exposure via URL-based auth or via debug logging of the outbound request | Information Disclosure | Use `x-goog-api-key` header, never `?key=`; never log the full outbound request URL or headers verbatim (Pitfall 3) |
| Autopilot re-engaging with a stale/incorrect owner-intent understanding after WAITING→ON resume (carried forward from Phase 2's own accepted risk AR-2-01, now compounded by an actual AI decision firing on resume, D-02) | (Policy/process risk, not strict STRIDE) | Not newly mitigated by this research — Phase 2's security review already accepted this risk (AR-2-01) at the state-machine level with no AI behind it; Phase 3 raises it again in concrete form ("the AI will now actually act on WAITING→ON resume, not just flip a badge") and it belongs on this phase's own security-review agenda, not silently inherited as already-closed |
| Full transcript capture of all game text, including other players' speech, for every saved-profile connection (D-14), against the policy's own data-handling section 6 language ("game text captured during play, including other players' words, is stored... for this tool's operation. Do not use it to profile or target other players.") | (Policy/data-handling risk) | Not decided by this research — this is explicitly named in CONTEXT.md's Deferred section as a required Phase 3 security-review agenda item; the technical implementation (Pattern: transcript tap) is the same regardless of the review's outcome, but retention/access-scope decisions belong to that review, not to this document |
| Carry-forward from Phase 2 (must be re-raised, not re-researched) | — | **DR-2-01** (staging log prints the OTP sign-in code, MUST FIX this phase per the owner) and **DR-2-02** (six throwaway staging profiles) — both must appear on this phase's security-review agenda; DR-2-01's remediation is now also a build task (Architectural Responsibility Map), not merely a re-presented risk |

## Project Constraints (from CLAUDE.md)

- Every phase must be provable by player-observable behavior or diagnostic verification; ROADMAP.md's Phase Validation line for Phase 3 is the binding demonstration text — planning must produce tasks whose completion maps directly to that line, not merely to "code exists."
- Plans are named for the capability they create; "backend wiring" or "frontend work" are not valid plan names. Candidate capability-named plans this research suggests: "the driver makes one Gemini decision and issues it through the ICM automation context," "the decision and its reasoning appear live and survive a refresh," "the server refuses to engage without Gemini configuration," "every game connection is transcribed with human/AI lines marked apart."
- Every plan needs acceptance criteria describing observable results (test, directive, endpoint, UI, log, or database inspection) — never "code is complete."
- Waves are dependency order only; a plan sits in the earliest wave where its `depends_on` are satisfied. Given this research: the ring buffer, the Gemini client, and the ICM engine wiring have no dependencies on each other and can be Wave 1; the driver (needs all three) is Wave 2; the websocket/panel/decisions-reload UI (needs the driver's persisted output) is Wave 3; the transcript tap and Logs page are independent of the driver and could run in parallel from Wave 1; DR-2-01's remediation is independent of everything else and can be any wave.
- The AI driver runs server-side in Go and issues commands through the ICM automation context (locked project decision, `DEC-server-side-go-driver`) — Pattern 3 is the literal mechanism satisfying this.
- Evidence is a canned report, log file, or end-user screenshot; database queries never count as proof — the Validation Architecture section above follows this exactly, including for "a decisions table row exists... after a page refresh," which must be demonstrated via the REST reload endpoint's response, not a `SELECT`.
- Every phase has a Brief for planning approval, an Evidence Dossier for acceptance, and a Risk Register for the security review — this RESEARCH.md's Security Domain section is the direct input to that review; DR-2-01/DR-2-02 carry-forward and the D-14 data-handling item must appear on it.

## Sources

### Primary (HIGH confidence)
- `internal/session/manager.go` — full file read, this session (Manager struct and its four existing maps, `Connect`, `Disconnect`, `SendCommand`, `ReadOutput`, `GetSession`, `EngageAutopilot`, `DisengageAutopilot`, `parkAutopilotLocked`, `resumeAutopilotLocked`, `logAutopilotTransition`)
- `internal/session/websocket.go` — full file read, this session (`WSMessage`, `MsgTypeAutopilot`, `IsHumanSource`, `applyWheelGrab`, `HandleWebSocket`'s full message loop, `readMUDOutput`, `relayMUDToClient`, `stripTelnetIAC`, `handleClientCommands`)
- `internal/session/handler.go` — full file read, this session (`Handler`, `HandlerCallbacks`, `EngageGate`, `StatusResponse`, `AutopilotRequest`/`AutopilotResponse`, the full `Autopilot` handler)
- `internal/session/autopilot.go` — read this session (`AutopilotState`, `AutopilotRecord`, `Engage`/`Disengage` pure transitions)
- `internal/icm/types.go`, `internal/icm/engine.go`, `internal/icm/dispatcher.go`, `internal/icm/pass_through.go`, `internal/icm/handler.go`, `internal/icm/errors.go` — full files read, this session (confirmed the `Process()`/`Dispatch()` short-circuit, `SafetyChecker` interface and `DefaultSafetyChecker` implementation, `ContextAutomation`/`AuthorityLevels`, `RegisterRoutes` unused by `main.go`)
- `internal/icm/icm_test.go` (relevant sections) — read this session (`TestDispatcher_SafetyEnforcement`'s exact test shape, reused as the Pattern 3 diagnostic-test template)
- `internal/config/config.go` — full file read, this session (`Config` struct, `Load()`'s required-vs-optional env pattern)
- `internal/store/profile.go` (lines 1-100, 505-582) — read this session (`Profile`, `AISettings`, `EngageGateAllowed`, `EngageGateRefusalMessage`, `ResolveAISettings`)
- `internal/auth/handler.go` (lines 140-190) — read this session, confirmed DR-2-01's exact log lines (`"STAGING: OTP sent to user, code: %s"`) at both `/send-otp`-family call sites
- `cmd/server/main.go` — full file read, this session (confirmed `icm` is never constructed or routed; confirmed the exact `HandlerCallbacks` wiring, migration-running block, static-file/SPA-fallback handler, `sessionMiddleware`)
- `migrations/010_add_ai_fields.up.sql`, `migrations/006_create_profiles.up.sql` — read this session (exact `ALTER TABLE`/`CREATE TABLE` conventions, `gen_random_uuid()`, `REFERENCES ... ON DELETE CASCADE`)
- `frontend/src/services/icm-adapter.ts` — full file read, this session (confirmed the frontend fallback-to-browser-logic pattern the phase must remove per success criterion 4)
- `frontend/src/services/automation.ts`, `frontend/src/services/automation/evaluator.ts` (targeted reads/greps) — confirmed `CommandSource`, `autopilotControl`'s existing threading (already built in Phase 2, not hypothetical), `case 'AUTO'` location
- `frontend/src/services/api.ts` (lines 1-260) — full relevant section read, confirmed `WebSocketManager`'s `handleMessage` switch, `onAutopilot`/`offAutopilot` template for the new `onAI`/`offAI`
- `frontend/src/components/AutopilotBadge.tsx` — full file read, this session (the "server owns truth, component derives nothing" pattern to replicate for the AI panel)
- `frontend/src/components/AIPlayerPanel.tsx` (lines 1-60), `frontend/src/pages/SettingsPage.tsx` (targeted greps) — confirmed `SettingsSection` union, nav list shape, the AI-activated/policy-accepted signal source for D-07
- `frontend/src/pages/PlayScreen.tsx` (targeted greps + lines 505-545) — confirmed the `.play-screen` layout root for the floating panel's mount point, `setSubmitCommandCallback`/`echoLocal` wiring
- `.planning/phases/02-autopilot-switch/02-RESEARCH.md`, `02-CONTEXT.md`, `02-VALIDATION.md` — read in full, this session, as the structural and evidentiary precedent this document follows
- `.planning/phases/03-one-ai-decision/03-CONTEXT.md`, `.planning/REQUIREMENTS.md`, `.planning/ROADMAP.md`, `.planning/PROJECT.md`, `.planning/STATE.md` — read in full, this session
- `.planning/todos/pending/2026-09-15-phase3-security-carry-forward.md` — read in full, this session
- `CLAUDE.md`, `.specify/memory/phase-based-development-approach.md` — read in full, this session
- `.specify/specs/ai-game-player-design-v3.md` §D3, `.specify/specs/safety-and-abuse-policy-v1.md` §§5-6 — read this session (D3's acceptance text verbatim; policy's data-handling and cost-ownership language, directly relevant to D-14's transcript scope and D-19's rate-limit-as-failure rule)
- `scripts/verify-phase2.sh` (header/usage comments) — read this session, the harness-shape template for `verify-phase3.sh`

### Secondary (MEDIUM confidence)
- [Generating content | Gemini API | Google AI for Developers](https://ai.google.dev/api/generate-content) — endpoint URL shape, `systemInstruction`, `responseMimeType`/`responseSchema`, response envelope shape, error status meanings — fetched and read this session
- [Using Gemini API keys | Google AI for Developers](https://ai.google.dev/gemini-api/docs/generate-content/api-key) — `x-goog-api-key` header vs `?key=` query parameter — cross-checked via WebSearch corroboration this session
- [Gemini API rate limits | Google AI for Developers](https://ai.google.dev/gemini-api/docs/rate-limits) — confirms Google does not publish fixed free-tier numbers, directs to per-account AI Studio dashboard instead — fetched this session
- [Gemini API errors | Google AI for Developers](https://ai.google.dev/gemini-api/docs/api-errors) — error object shape (`code`, `message`), 429 `RESOURCE_EXHAUSTED` — fetched this session
- [Generate structured output (JSON/enums) | Firebase AI Logic](https://firebase.google.com/docs/ai-logic/generate-structured-output) — corroborates `responseSchema` + `responseMimeType` requirement pairing — fetched this session

### Tertiary (LOW confidence)
- Third-party rate-limit aggregator sites (e.g. aipromptshub.co, aifreeapi.com) citing "10 RPM / 250 RPD for gemini-2.5-flash" — **not** corroborated by Google's own docs, tagged `[ASSUMED]` throughout this document, not presented as verified fact anywhere (see Assumption A1)
- WebSearch-surfaced model names (`gemini-3.1-flash-lite`, `gemini-3.5-flash`, `gemini-3.6-flash`, `gemini-flash-latest`) — the shutdown date for `gemini-2.5-flash-lite` and the existence of `gemini-flash-latest` as a rolling alias are corroborated across two independent fetches/searches this session (MEDIUM), but the full current model lineup naming is volatile enough that no specific model ID from this research should be hard-coded as a Go-level default — see State of the Art and the Anti-Patterns section

## Metadata

**Confidence breakdown:**
- Standard stack / ICM dispatch mechanics: HIGH — every claim traces to a full file read this session, including the single most load-bearing finding (`Process()` never reaches `Dispatch()` for pass-through commands), which was verified by reading `engine.go`'s actual control flow line-by-line, not inferred from documentation or comments
- Gemini API contract: MEDIUM — official docs fetched and cross-checked this session for the request/response/error shapes and auth header preference (HIGH-confidence sub-claims), but exact numeric free-tier limits are explicitly undocumented by Google and only available third-hand (LOW-confidence sub-claim, clearly flagged)
- Architecture (new tables, new websocket message, new packages): MEDIUM — these are "Claude's Discretion" per CONTEXT.md; the recommendations follow this codebase's own established conventions exactly (verified against Phase 1/2 precedent) rather than being pre-existing patterns to copy verbatim
- Pitfalls: HIGH for Pitfalls 1, 3, 4, 5 (directly observed in actual code/docs this session); MEDIUM for Pitfall 2 (reasoned from `dispatcher.go`'s actual control flow, but its Phase 4 implication is a forward-looking recommendation, not an observed failure)

**Research date:** 2026-09-15
**Valid until:** 14 days (shorter than Phase 2's 30-day estimate specifically because the Gemini model-naming landscape is confirmed to be changing within this project's own timeline — `gemini-2.5-flash-lite`'s October 2026 shutdown falls inside a plausible Phase 3 build-and-verify window — re-check model availability and the free-tier dashboard before staging verification if more than two weeks elapse between this research and execution)
