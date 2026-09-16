# Phase 3 Brief — One AI Decision

**Status:** Awaiting the owner's planning approval (2026-09-15).

**Purpose:** Owner review before execution. Everything here is lifted from `.planning/ROADMAP.md` §Phase 3, `03-CONTEXT.md` and the thirteen PLAN.md files; nothing is new. Approve this, and Phase 3 goes to `/gsd-execute-phase 3`.

**Evidence rule (owner-directed, binding for this and every later phase):** every success criterion and acceptance criterion is proven by one of two artifact types, and nothing else.

1. **A canned report.** A repeatable script or test run whose output is captured verbatim to a file under `evidence/`. Server log excerpts captured to a file count here.
2. **A screenshot from the end-user perspective.** The MUDPuppy browser page as the owner sees it. No devtools pane, no terminal, no raw JSON in frame.

Database queries and inspections are not evidence anywhere in this phase. The one claim the ROADMAP phrases as "a decisions table row exists" is proven through the application's own decisions endpoint in the canned report, never by a query.

---

## Phase Goal

The thinnest slice through the whole stack works: with autopilot engaged, Gemini reads the game, makes one decision, and the command is issued through the ICM automation context, with the reasoning visible live and stored.

## Phase Success Criteria (what must be TRUE afterwards)

| # | Criterion | Proof (canned report or end-user screenshot) |
|---|-----------|----------------------------------------------|
| 1 | With autopilot engaged, the driver reads the current game text, sends one request to Gemini with the profile's conduct rules and approach guidance included, and issues the returned command through the ICM automation context so it passes the ICM dispatcher and its safety limits. | `evidence/01-test-report.txt` PASS lines for `TestHandleEngage`; `evidence/04-staging-ai-player.log` lines `stage=request`, `stage=dispatch`, `stage=sent`; screenshots `evidence/05-decision-in-panel.png` and `evidence/06-ai-assist-terminal-line.png` (the game's reply under the AI's command). |
| 2 | The decision and the model's reasoning appear in the play screen as they happen and are stored in the session log, where they survive a page refresh. | Screenshots `evidence/05-decision-in-panel.png` and `evidence/07-panel-after-refresh.png`; `evidence/03-canned-report.txt` RUN B decisions step returning the walkthrough's decision over HTTP; `evidence/12-log-page-two-pane.png`. |
| 3 | Model names and API key come from environment configuration on staging; nothing is hard-coded, and the server fails clearly if they are absent when engagement is attempted. | `evidence/03-canned-report.txt` RUN A `PASS C3` with the refusal sentence printed character for character; screenshot `evidence/08-refused-not-configured.png`; `evidence/01-test-report.txt` `### MODEL LITERALS` section empty and PASS lines for `TestLoadAIRegistry`, `TestAIConfigured`. |
| 4 | A command submitted on the server in the automation execution context is dispatched through the ICM engine's dispatcher and safety checker, demonstrable by a diagnostic test, and the frontend adapter's ICM calls no longer fall back silently to browser-side logic. | `evidence/01-test-report.txt` PASS lines for `TestDispatch_AutomationPassThrough_GoesThroughSafetyChecker` including the `refused_when_rate_limit_tripped` subtest; `evidence/03-canned-report.txt` ICM route step returning non-404; screenshot `evidence/13-hand-play-unchanged.png`. |

**Phase Validation line (from ROADMAP):** Player-observable on staging: with autopilot engaged, one decision and its reasoning appear in the play screen and the game responds to the issued command. Diagnostic: a decisions table row exists for that decision and is returned after a page refresh; a `go test` proves the command passed through the ICM dispatcher in the automation context; with the Gemini environment variables unset, engagement fails with a clear error and no command is sent.

**The finding that shaped the plan.** Research read the dormant ICM engine and found that its `Engine.Process()` classifies a plain game command such as `north` as pass-through and returns before `Dispatch` is ever called. A driver written the obvious way would ship a working demo and a false criterion 4. Plan 03-03 therefore writes the test that can tell those two worlds apart, in wave 1, before the driver exists; plan 03-08 calls `Dispatcher.Dispatch` directly with `ContextAutomation` and only sends to the game after it approves.

---

## Breakdown by Success Criterion (review view)

### Criterion 1 — One decision, through the ICM automation context, with the profile's standing text in the prompt

| Plan | Plan acceptance criterion | Proof |
|------|---------------------------|-------|
| 03-01 | The server can produce the current game text on demand, ANSI-free and bounded, including what the game printed before autopilot was switched on | `01-test-report.txt`: PASS `TestWindowSnapshot`, `TestWindowRingBound`, `TestWindowIncludesPreEngageText` |
| 03-01 | A dropped connection does not empty the window, so a resume reads across the drop | `01-test-report.txt`: PASS `TestWindowSurvivesDisconnect` |
| 03-01 | The owner's terminal is unaffected: colour still reaches xterm.js | source assertion (no strip on the browser relay path); `13-hand-play-unchanged.png` |
| 03-02 | One request returns a reasoning and exactly one command, and the key travels in the `x-goog-api-key` header, never the URL | `01-test-report.txt`: PASS `TestGenerateContent` and its `no_api_key_in_url` subtest |
| 03-02 | Every vendor failure shape (transport, 400, 401, 403, 429, empty answer, unreadable answer) is a named error, never a crash | `01-test-report.txt`: PASS `TestGenerateContentErrors` |
| 03-08 | Engaging makes the driver read the window, ask the model once with conduct rules and approach guidance carried verbatim, and send the returned command | `01-test-report.txt`: PASS `TestHandleEngage`; `04-staging-ai-player.log`: `stage=request`, `stage=dispatch`, `stage=sent`; `06-ai-assist-terminal-line.png` |
| 03-08 | Exactly one decision per engagement; a repeated `#AUTO ON` fires nothing; the switch then stays on and idle | `01-test-report.txt`: PASS `one_command_per_engage`, `repeat_engage_fires_nothing` |
| 03-08 | A WAITING-to-ON resume after a reconnect fires one fresh decision on the post-reconnect text | `01-test-report.txt`: PASS `resume_fires_one_fresh_decision`; `04-staging-ai-player.log`: a `stage=request` line after a `cause=resume` line |
| 03-08 | Any first failure sends nothing to the game, records the failure, notices in the panel and terminal, and lands the switch off | `01-test-report.txt`: PASS `TestHandleEngageFailures` (one row per failure kind); `10-decision-failure-disengage.png` |
| 03-04 | `#AUTO ON` and `#AUTO OFF` still do exactly what Phase 2 made them do | `05-decision-in-panel.png`, `10-decision-failure-disengage.png` |

### Criterion 2 — The decision and reasoning on screen as they happen, stored, and back after a refresh

| Plan | Plan acceptance criterion | Proof |
|------|---------------------------|-------|
| 03-09 | The moment the AI decides, the server pushes the reasoning, the command and the outcome to the owner's open play screen | `01-test-report.txt`: PASS `TestPushAI`; `05-decision-in-panel.png` |
| 03-09 | A failure or refusal is pushed the same way as a system line with the exact notice | `01-test-report.txt`: PASS `pushes_a_system_line`; `10-decision-failure-disengage.png` |
| 03-09 | A connection's stored decisions can be fetched back in order, which is what makes them survive a refresh | `01-test-report.txt`: PASS `TestDecisionsReload`; `03-canned-report.txt`: RUN B decisions step |
| 03-09 | A caller who does not own the connection gets nothing from the decisions endpoint | `01-test-report.txt`: PASS `not_owned_connection_is_refused` |
| 03-10 | On an AI-activated profile the play screen carries a floating AI Assist panel that collapses to a tab; the terminal keeps its full width | `05-decision-in-panel.png` |
| 03-10 | The panel shows the short reasoning, then the command it issued | `05-decision-in-panel.png` |
| 03-10 | The AI's command prints in the terminal as `[AI-ASSIST > command]` with the game's reply beneath, so a typed and an AI-issued command never look alike | `06-ai-assist-terminal-line.png` |
| 03-10 | After a page refresh the panel shows the same decision again, rebuilt from the server | `07-panel-after-refresh.png` |
| 03-10 | The panel has no message box, no send button, no text input of any kind (Phase 5's) | source assertion in task 03-10-02 |
| 03-05 | Every saved-profile connection is transcribed from connect to disconnect; quick connects are not | `01-test-report.txt`: PASS `TestTranscriptOpensOnConnectAndClosesOnDisconnect`, `quick_connect_is_not_transcribed` |
| 03-05 | Sent lines are stored marked human or ai; game output marked game; engage and disengage leave stint markers | `01-test-report.txt`: PASS `TestTranscriptLineSources`, `TestTranscriptStintMarkers` |
| 03-05 | Transcript writes never block the MUD read loop | `01-test-report.txt`: PASS `TestTranscriptDoesNotBlockReader` |
| 03-05 | A profile's sessions can be listed by date and time and any one read back over HTTP, by its owner and nobody else | `01-test-report.txt`: PASS `TestListSessionsScopedToOwner`, `TestGetSessionTranscript`, the two `not_owned` refusals; `03-canned-report.txt` sessions steps |
| 03-11 | Settings has a Logs section whose Open Logging button opens the log page in a new tab, leaving the Settings tab where it was | `11-logs-section.png` plus the walkthrough note |
| 03-11 | The log page lists the profile's sessions on the left and shows the selected transcript on the right, human, AI and game lines told apart | `12-log-page-two-pane.png` |
| 03-11 | The log page is sign-in protected, scoped to the profile in its URL, and never mounts the play screen | source assertions on the route |

### Criterion 3 — Model and key from the environment only; a clear refusal when absent

| Plan | Plan acceptance criterion | Proof |
|------|---------------------------|-------|
| 03-02 | A model registry is read from environment variables at startup; a blank profile model name resolves to the default; an absent registry is not fatal | `01-test-report.txt`: PASS `TestLoadAIRegistry`, `TestResolveModelEntry`, `TestAIConfigured` |
| 03-02 | No model identifier is a literal anywhere in Go source | `01-test-report.txt`: `### MODEL LITERALS` empty; `03-canned-report.txt` grep step zero matches |
| 03-06 | With no model or key in the environment the server starts normally and answers requests | `04-staging-ai-player.log` startup lines from the unconfigured deploy |
| 03-06 | `#AUTO ON` on that server is refused with `[Autopilot refused: AI is not configured on this server]`, the switch stays off and nothing is sent to the game | `01-test-report.txt`: PASS `TestAutopilotHandler_AIConfigRefusal`; `03-canned-report.txt` RUN A `PASS C3`; `08-refused-not-configured.png` |
| 03-06 | The Phase 1 policy gate still refuses first with its own unchanged sentence | `01-test-report.txt`: PASS `gate_refusal_takes_precedence` |
| 03-06 | `#AUTO OFF` and the badge keep working with the AI unconfigured | `01-test-report.txt`: PASS `off_is_unaffected_when_unconfigured` |
| 03-06 | Each refusal is one `[AI-PLAYER]` line with `cause=refused-not-configured` and no key, model name or game text on it | `04-staging-ai-player.log` |
| 03-12 | One command turns the HTTP-reachable half of the phase into a PASS/FAIL line per criterion, and the harness proves it can fail | `03-canned-report.txt`; `02-harness-selftest.txt` with `FAIL C3` and a non-zero exit |
| 03-07 | The staging one-time sign-in code never reaches the log by default; only `AUTH_LOG_OTP` turns it back on for local debugging (DR-2-01) | `01-test-report.txt`: PASS `TestOTPNotLogged`; `04-staging-ai-player.log`: no `code:` line for a fresh sign-in |

### Criterion 4 — The ICM dispatcher gates automation-context commands, provably, and the browser no longer answers for the server

| Plan | Plan acceptance criterion | Proof |
|------|---------------------------|-------|
| 03-03 | A plain game command in the automation execution context passes the dispatcher's safety checker before anything else happens to it | `01-test-report.txt`: PASS `TestDispatch_AutomationPassThrough_GoesThroughSafetyChecker` |
| 03-03 | The same command is refused once the safety checker's limits are tripped, so the check is load-bearing, not decorative | `01-test-report.txt`: PASS `refused_when_rate_limit_tripped` |
| 03-03 | The server owns exactly one ICM engine, shared by the HTTP routes and the driver, and its four routes answer instead of 404 | source assertion (one `icm.NewEngine()` in `main.go`); `03-canned-report.txt` ICM route step |
| 03-03 | The browser's ICM adapter surfaces a server error instead of quietly recomputing an answer itself | source assertion in task 03-03-03; green frontend build in `01-test-report.txt`; `13-hand-play-unchanged.png` |
| 03-08 | The command reaches the game only after `Dispatch` approves; an ICM refusal means nothing is sent at all | `01-test-report.txt`: PASS `dispatch_precedes_send`, `icm_refusal_sends_nothing` |

Plan 03-13 adds no capability. It runs the phase on staging in two passes (unconfigured, then configured with the owner's key), files the evidence above, writes `03-13-SUMMARY.md` with one PASS/FAIL row per criterion citing these files by name and line, closes DR-2-02, and writes the security-review agenda.

---

## Plans by wave

Waves are dependency order only. Plans in the same wave touch no common files. Seven waves because `cmd/server/main.go`, `internal/session/manager.go`, `internal/session/handler.go` and `frontend/src/services/api.ts` each chain several plans.

| Wave | Plan | Capability | Depends on | What the owner will be able to see afterwards |
|------|------|------------|------------|-----------------------------------------------|
| 1 | 03-01 — The server holds a rolling, ANSI-free window of the recent game text | A bounded per-user ring of recent MUD output, telnet and ANSI stripped, kept from the moment a connection is live and surviving a disconnect | — | Nothing in the browser. The test report shows the window bounded, colour-free, holding pre-engage text and spanning a drop. |
| 1 | 03-02 — The server can ask Gemini for one decision, with model and key taken only from the environment | A hand-written `net/http` client for `generateContent` constrained to a `{reasoning, command}` answer, plus a model registry from `AI_MODEL_*` variables with a default entry; absence is not fatal | — | Nothing in the browser. The test report shows the request shape, header auth, every failure named, and the registry parsed. |
| 1 | 03-03 — The ICM engine is live and an automation-context command provably passes its safety checker | One engine constructed in `main.go`, four `/api/v1/icm` routes registered, the criterion-4 diagnostic test, and the browser adapter's silent fallback removed | — | The ICM routes answer; the diagnostic test in the report can fail if the safety checker is skipped. |
| 1 | 03-04 — `#AUTO` answers only to ON and OFF | The narrowed directive grammar; `#AUTO STATUS` retired in the browser; the server's status wire action untouched | — | Typing `#AUTO STATUS` prints the one red line. |
| 2 | 03-05 — Every saved-profile connection is transcribed and readable back by its owner | Migration `011` (game_sessions, game_session_lines, ai_decisions), a transcript tap on the two MUD choke points with human / ai / game tagging and stint markers, and two owner-scoped read endpoints | 03-01, 03-03 | The canned report lists the walkthrough's session and reads it back. |
| 2 | 03-06 — `#AUTO ON` is refused when the AI is not configured on this server | A second engage gate beside the Phase 1 policy gate, with the D-20 sentence and a `cause=refused-not-configured` log line | 03-02 | On an unconfigured staging, `#AUTO ON` is refused and the badge stays Off. |
| 2 | 03-07 — The staging sign-in code never reaches the log (DR-2-01) | Both OTP log call sites funnelled through one helper gated by `AUTH_LOG_OTP`, off by default, with the first test `internal/auth` has had | 03-02 | A fresh sign-in leaves an issuance line but no code in the staging log. |
| 3 | 03-08 — The AI makes one decision and its command reaches the game through the ICM automation context | The `internal/driver` package: snapshot, two-tier prompt, model call, single-plain-line validation, `Dispatcher.Dispatch` in `ContextAutomation`, `SendCommandAs(..., "ai")`, decision storage, and the engage and resume hooks that fire it exactly once | 03-01, 03-02, 03-03, 03-05, 03-06 | The game answers a command the owner did not type. |
| 4 | 03-09 — The decision reaches the browser as it happens and can be re-read after a refresh | A new `ai` websocket message with a typed payload, a per-user client registry, the driver's notifier, and the owner-scoped decisions endpoint | 03-08 | Nothing new yet; the report proves the push and the reload. |
| 5 | 03-10 — The owner watches the reasoning in the AI Assist panel and sees the command marked in the terminal | The floating, minimizable panel with decision cards and system lines, the two new colour tokens, the `[AI-ASSIST > command]` terminal line replacing local echo, gated to AI-activated profiles | 03-09 | The panel, the reasoning, the magenta AI line, and the same decision after a refresh. |
| 5 | 03-12 — One command turns the Phase 3 HTTP sequence into a canned PASS/FAIL report | `scripts/verify-phase3.sh` with fixtures, a self-test and a negative self-test; no database, no live model | 03-05, 03-06, 03-09 | A report file with a verdict per criterion and the response body under each step. |
| 6 | 03-11 — The owner opens a profile's session logs in a new tab and reads any past session | A Logs section in Settings with an Open Logging button (`target="_blank"`), the `/logs/:connectionId` route inside AuthGuard and outside the play-screen shell, the two-pane page | 03-05, 03-10 | The Logs section, the new tab, sessions by date on the left, the transcript on the right. |
| 7 | 03-13 — Phase 3 demonstrated on staging and filed as evidence | Two staging passes (unconfigured, then configured), fourteen evidence files, the four-row criterion table, DR-2-02 closed, the security-review agenda. Two tasks are blocking owner checkpoints. | all | The walkthrough itself, then a folder of reports, a log excerpt and ten screenshots. |

---

## Decisions honoured

Every decision from the Phase 3 discussion is cited by ID in at least one plan's truths or success criteria (checked by grep: all twenty plus DR-2-01 and DR-2-02).

- **D-01:** The one decision fires immediately on `#AUTO ON`, using the recent game text the server already holds. Plan 03-08.
- **D-02:** A WAITING-to-ON resume after a reconnect counts as an engage and fires one fresh decision on the post-reconnect text. Plans 03-01, 03-08.
- **D-03:** After the single decision autopilot stays ON and idle; a repeated `#AUTO ON` is a no-op and fires nothing. Plan 03-08.
- **D-04:** The current game text is a server-side rolling window, sized by a constant, ANSI stripped, including text from before the switch was flipped. Plan 03-01.
- **D-05:** Two prompt tiers: the immediate window, and the profile's conduct rules and approach guidance verbatim. Phase 6 fills tier two further without changing the shape. Plan 03-08.
- **D-06:** A floating, minimizable AI Assist panel on the play screen; the terminal keeps its full width. Plan 03-10.
- **D-07:** The panel opens automatically whenever the connected profile has AI activated, ON or OFF; profiles without acceptance see nothing new. Plan 03-10.
- **D-08:** Each decision is one message: short reasoning, then the command; state changes are system lines; no message input until Phase 5. Plans 03-09, 03-10.
- **D-09:** The AI's command prints in the terminal as `[AI-ASSIST > north]` and replaces local echo for AI-issued commands. Plans 03-10, 03-11 (same label in the transcript).
- **D-10:** The reasoning is a short plain-language "why" the model writes for the owner, plus exactly one command; the internal thinking trace is not shown. Plans 03-02, 03-08.
- **D-11:** `#AUTO` accepts only ON and OFF; anything else prints `[Autopilot: don't know what you're talking about. Your options are ON or OFF]`. The Phase 2 `#AUTO STATUS` line is retired. Plan 03-04.
- **D-12:** Each decision is stored on its own row (time, profile and connection, the window the model saw, reasoning, command, outcome); the panel reloads the current connection's decisions after a refresh. Plans 03-05 (table), 03-08, 03-09, 03-10.
- **D-13:** A first failure of any kind sends nothing to the game, notices in the panel and the terminal, lands autopilot on OFF, and leaves regular play untouched. No retry in this phase. Plans 03-02, 03-08, 03-09, 03-10.
- **D-14:** A session transcript is stored for every saved-profile game connection, connect to disconnect, AI activated or not; sent lines marked human or AI; no reasoning in it; quick connects not logged. Plan 03-05.
- **D-15:** Engage and disengage moments are recorded inside the transcript so a stint at the wheel can be found later. Plan 03-05.
- **D-16:** Settings gets a Logs section per connection profile with an Open Logging button that opens the log page in a new tab; the current tab does not navigate. Plan 03-11.
- **D-17:** The log page is per game profile: sessions by date and time on the left, the selected transcript on the right; sign-in protected and scoped to the owner's profile; no cross-profile listing, no share links. Plans 03-05, 03-11.
- **D-18:** A model registry from environment configuration: a default plus any number of named entries, each with endpoint and key; blank profile model name resolves to the default; Gemini only in this phase. Plan 03-02.
- **D-19:** The owner's key is on the Gemini free tier; a rate-limit answer is a failure under D-13; the per-session call cap is Phase 4's. Plans 03-02, 03-08.
- **D-20:** When the key or default model is missing, the server starts normally and `#AUTO ON` is refused with `[Autopilot refused: AI is not configured on this server]`; the switch stays OFF. Plans 03-02, 03-06, 03-12.
- **DR-2-01 (must fix):** The staging sign-in code is gated out of the log by default. Plan 03-07; closure recorded at the security review by 03-13.
- **DR-2-02:** The six throwaway staging profiles are deleted through the app at the end of the walkthrough, with the owner never signed out. Plan 03-13.

Two plumbing calls made under Claude's Discretion and recorded in the plans: the server keeps its `status` wire action on the autopilot endpoint even though the typed `#AUTO STATUS` is retired; and a seventh failure notice exists for an ICM refusal, `the command was refused by the command safety limits`, copy the owner has not yet seen and which plan 03-13 carries to the review.

---

## What the owner must supply

One thing, at the first checkpoint of plan 03-13: **the Gemini API key.** The executor never invents, guesses or reuses one. The evidence is captured in two passes so the phase does not stall on it: first RUN A with the Gemini variables absent (proving D-20 and the `#AUTO` grammar), then the key is set and RUN B and the walkthrough follow. The five variables set on the `staging` environment of service `MudPuppy`, values never recorded anywhere:

| Variable | Meaning |
|----------|---------|
| `AI_MODEL_DEFAULT` | the slug of the default registry entry, for example `GEMINI` |
| `AI_MODEL_GEMINI_NAME` | the model identifier sent to the vendor |
| `AI_MODEL_GEMINI_ENDPOINT` | the API base URL for that entry |
| `AI_MODEL_GEMINI_KEY` | the API key (header auth only, never the URL, never logged) |
| `AI_MODEL_GEMINI_PROVIDER` | `gemini`; blank means gemini |

Production is untouched.

---

## What the owner will do during the walkthrough

All of this is on Railway staging, in the browser, signed in as the owner, against a live MUD. The staging session is never signed out. Each step produces one screenshot; the file names are fixed because the plans cite them.

**Pass A, before the key is set (checkpoint 03-13-02):**

1. On a profile that has accepted the policy, connect to the MUD and type `#AUTO ON`. The terminal prints `[Autopilot refused: AI is not configured on this server]` and the badge still reads `Autopilot: Off`. Screenshot: `evidence/08-refused-not-configured.png`.
2. Type `#AUTO STATUS`. One red line: `[Autopilot: don't know what you're talking about. Your options are ON or OFF]`. Nothing else changes. Screenshot: `evidence/09-auto-unknown-option.png`.
3. Supply the Gemini key; the executor sets the five variables on staging and restarts the service. Approve the checkpoint.

**Pass B, with the key set (checkpoint 03-13-03):**

4. Connect an AI-activated profile and hand-play a couple of commands so the window has text. The AI Assist panel is present with its empty state.
5. Type `#AUTO ON`. The panel shows one decision: the reasoning in plain language, then `→ command`. The badge reads `Autopilot: On`. Screenshot: `evidence/05-decision-in-panel.png`.
6. The terminal shows `[AI-ASSIST > command]` in magenta and, beneath it, the game's own reply. Screenshot: `evidence/06-ai-assist-terminal-line.png`.
7. Hard-refresh the page and do nothing else. The same decision is back in the panel. Screenshot: `evidence/07-panel-after-refresh.png`. The badge still reads On and no second command appeared.
8. Force one failure without touching the server: set the profile's model name in Settings to one that does not exist, type `#AUTO OFF` then `#AUTO ON`. The same failure sentence appears in the panel and the terminal and the badge reads Off. Screenshot: `evidence/10-decision-failure-disengage.png`. Set the model name back to blank.
9. In Settings, select Logs. The section shows its description and the Open Logging button. Press it; the Settings tab stays put. Screenshot: `evidence/11-logs-section.png`.
10. In the new tab, select this walkthrough's session. Sessions on the left, transcript on the right, with a `> command` human line in cyan, the `[AI-ASSIST > command]` line in magenta, and unprefixed game output. Screenshot: `evidence/12-log-page-two-pane.png`.
11. Create a fresh profile, do not accept the policy, connect and hand-play. No panel, no tab, no new colour, badge Off. Screenshot: `evidence/13-hand-play-unchanged.png`.
12. The executor runs the harness again (RUN B), captures the `[AI-PLAYER]` lines to `evidence/04-staging-ai-player.log`, and greps the excerpt for anything it must not contain: game text, reasoning, conduct rules, the key, the endpoint, a sign-in code.
13. Close DR-2-02: the six throwaway profiles (Phase 2 Harness, Harness B, Walkthrough, Refusal, Hand Play, Phase 1 Evidence Walkthrough) are deleted through the app. Screenshot: `evidence/14-connections-cleaned.png`. Approve the checkpoint.

Before either pass, the executor captures the full test report to `evidence/01-test-report.txt` and the harness self-tests to `evidence/02-harness-selftest.txt`.

---

## Security

### Threat register summary

Forty-two numbered threats (`T-3-01` to `T-3-45`, with three numbers unused) plus the standing supply-chain item were identified across the thirteen plans. Each is mitigated by a named task, accepted at plan level, or deferred to the phase-close security review. The high-severity ones:

| Threat | What it is | Handled by |
|--------|------------|------------|
| T-3-01 | Prompt injection through game text: a hostile room description making the model "issue" a `#`/`@`/`$`/`%`-prefixed or multi-line command | 03-08: Go-side validation before the dispatcher, one failure-table row per attack shape; the response schema constrains JSON shape only, so the check is never delegated to the model. Residual (steering which legitimate command is chosen) on the agenda as item 6. |
| T-3-02 | The Gemini API key in a URL, a log line, an error string, a screenshot or a report | 03-02: header auth, no `?key=` construction, zero log calls in the client; 03-06 and 03-08: fixed log formats with no registry detail; 03-13: names only recorded, log excerpt grepped before filing. |
| T-3-03 / T-3-04 | IDOR on the decisions and transcript endpoints and the log page | 03-05, 03-09, 03-11: every read resolves through `GetProfileByConnection(userID, connectionID)`; the transcript read also filters on connection id; refusal subtests. |
| T-3-05 (DR-2-01) | The one-time sign-in code in the staging deploy log | 03-07: one gated helper, off by default, tested. Railway log access itself is an access question carried to the review. |
| T-3-06 | The driver sends without passing the browser's human wheel-grab | 03-08: the driver reaches the MUD only through `SendCommandAs(..., "ai")` after `Dispatch` approves, so ICM's safety checker gates it and the transcript marks it. Residual on the agenda as item 5. |
| T-3-10 | A canned report that always passes | 03-12 and 03-13: the negative self-test produces `FAIL C3` and a non-zero exit before the staging report is trusted. |
| T-3-11 | A push delivered to the wrong user's screen; decisions from another connection in the panel | 03-09: registry keyed by the authenticated user from the upgraded connection's own context; 03-10: per-connection fetch, no merging. |
| T-3-20 | Two ICM engines with independent safety-checker state | 03-03: `NewHandlerWithEngine`, one `icm.NewEngine()` in `main.go`, so routes and driver share one checker. |
| T-3-23 | A criterion-4 test that would pass even if the safety checker were skipped | 03-03: `refused_when_rate_limit_tripped` fails if `checkSafety` is bypassed. |
| T-3-29 | Engaging with no model configured, leaving a half-live driver | 03-06: the refusal returns before `EngageAutopilot`; the switch never moves. |
| T-3-38 | An AI-issued command looking like something the owner typed | 03-10: one reserved colour and the fixed `[AI-ASSIST > ...]` label used identically in the terminal, the panel and the log page. |

Medium and low items (data races on the new maps, transcript writes on the read path, script injection of model text, reverse tabnabbing from the Logs tab, the wrong deploy target, log lines carrying game text) are each mitigated by a named task in the plan files, and every plan asserts that no package is installed (T-3-SC).

### Security-review agenda (decided by the owner at phase close)

Plan 03-13 writes `03-SECURITY-AGENDA.md` as the input to this review, not its output. Six open items, each with the same three dispositions and none pre-chosen or recommended.

| # | Item | Accept | Defer | Remediate Now |
|---|------|--------|-------|---------------|
| 1 | **Transcript and window retention against the policy's data handling.** Every saved-profile connection's full game traffic, including other players' speech and hand play, is now stored (D-14), plus the window the model saw per decision (D-12). No retention limit, deletion path or export exists; the decisions endpoint deliberately does not serve the window text to the browser. Candidate remediation: a retention window, length left to the owner. T-3-09, T-3-15. | open | open | open |
| 2 | **A resume now acts.** Phase 2 accepted an unbounded WAITING state that returns to ON by itself when nothing acted on it (AR-2-01). From this phase a resume fires a real decision and a real command (D-02). `WaitingSince` is still stored so a bounded lifetime remains cheap. T-3-14. | open | open | open |
| 3 | **The log page's authorization scope.** Per-profile, signed-in scoping is enforced server-side; the open question the owner asked to have raised is whether that is the right boundary for stored game text at all. T-3-04. | open | open | open |
| 4 | **The Gemini key on staging and the free-tier limits.** Header-only, never logged, never in the repo or the evidence; Google publishes no fixed free-tier numbers; a rate-limit answer is a D-13 failure with no retry; the call cap is Phase 4's. T-3-02, T-3-19. | open | open | open |
| 5 | **The driver sends without passing the human wheel-grab.** By construction, since it is not human input; ICM gates it and the transcript marks it; the residual is a server-side send path no browser-side control mediates. Re-confirms Phase 2's T-2-01 as that agenda required. T-3-06. | open | open | open |
| 6 | **Prompt injection through game text.** Mitigated for command shape; a hostile description can still steer which legitimate command is chosen. T-3-01. | open | open | open |

Closures for the owner to confirm rather than decide: **DR-2-01** closed by plan 03-07 (with the non-code part, review Railway log access on the staging project, carried forward with its own three dispositions) and **DR-2-02** closed by plan 03-13. Plan-level acceptances listed for confirmation: T-3-SC, T-3-19, T-3-25. Two forward notes for Phase 4: whether the driver should call ICM's `RecordExecution` after a send, since ICM's own counters do not accumulate for pass-through commands (harmless at one command per engage, not in a loop); and the seventh failure notice.

The project cannot close while any item on this agenda remains deferred. If the owner chooses Remediate Now on any item, that becomes its own plan or phase; nothing is remediated inside Phase 3.

---

## Out of scope / deferred

- **No loop, no pacing.** One decision per engagement, then idle. Phase 4.
- **No coaching input.** The panel has no message box; Phase 5 adds it.
- **No learned notes, no AI-written memory.** Tier two of the prompt is conduct rules and approach guidance only; Phase 6 adds learned notes and the debrief that hangs off the stint markers.
- **No non-Gemini vendor.** The registry carries endpoint and key per entry so another client can be added later.
- **No per-profile window size, no cross-profile log listing, no share links, no transcript search or export.**
- **Shelf copy unchanged.** The retirement of `#AUTO STATUS` and the transcript scope are implementation decisions, not design amendments; `D:\Projects\ai-mud-player\documents\` needs no update for this phase.

---

Approve this, and Phase 3 goes to `/gsd-execute-phase 3`.
