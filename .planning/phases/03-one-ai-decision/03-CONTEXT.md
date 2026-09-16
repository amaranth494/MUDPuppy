# Phase 3: One AI Decision - Context

**Gathered:** 2026-09-15
**Status:** Ready for planning

<domain>
## Phase Boundary

The thinnest slice through the whole stack works once. With autopilot ON on a connected, AI-activated profile, the server-side Go driver takes the recent game text the server holds, sends one request to Gemini carrying the profile's conduct rules and approach guidance, and issues the single returned command through the ICM automation context so it passes the ICM dispatcher and its safety limits. The decision and a short reasoning appear on the play screen as they happen in a new floating **AI Assist** panel, are stored per decision, and reload into the panel after a page refresh. Model names and API keys come from environment configuration on staging through a small model registry; nothing is hard-coded, and `#AUTO ON` is refused with a clear notice when the configuration is absent. The dormant `internal/icm` engine is wired into the server and the frontend adapter's ICM calls stop falling back silently to browser-side logic.

Two things the owner added in this discussion are built here because the design already places them on the profile: a **session transcript** stored for every saved-profile game connection from connect to disconnect (game traffic only, human and AI input marked apart, no reasoning), reachable from a new **Logs** section in Settings that opens a per-profile log page in a new browser tab; and the security carry-forward from Phase 2 (DR-2-01 sign-in code in the staging log, marked must fix; DR-2-02 throwaway staging profiles).

Not in this phase: any loop or pacing (Phase 4), coaching input in the panel (Phase 5), learned notes and any AI-written memory (Phase 6), non-Gemini providers (registry shape allows them; implementation later).

</domain>

<decisions>
## Implementation Decisions

### When the AI acts and what it reads
- **D-01:** The one decision fires **immediately on `#AUTO ON`**, using whatever recent game text the server already holds. No waiting for fresh output, no separate step directive.
- **D-02:** A **WAITING to ON resume** after a reconnect counts as an engage and fires one fresh decision on the post-reconnect text. Consistent with Phase 4's rule that a resume reassesses rather than resuming a stale plan.
- **D-03:** After the single decision and its command, autopilot **stays ON and idle**. Nothing more is issued until Phase 4 adds the loop. Only `#AUTO OFF` or the Phase 2 wheel-grab turns it off (Phase 2 D-01, D-04 hold; a repeated `#AUTO ON` is still a no-op and must not fire a second decision).
- **D-04:** The "current game text" is a **rolling per-session window of recent game output** kept server-side: sized by a server constant (roughly the last screenful or two), **ANSI colour codes stripped**, telnet control bytes already stripped, and **including text from before the switch was flipped** so the AI's first look resembles what the owner sees on screen. Not a profile setting.
- **D-05:** The prompt has **two tiers** from the start. Tier one is the immediate window for reaction. Tier two is the profile's standing text: conduct rules and approach guidance, verbatim (Phase 1 D-12). Phase 6 adds learned notes to tier two and the AI's ability to write to it; the prompt shape does not change then, only what fills tier two. (Owner: "immediate buffer for reaction stuff, but also a memory file for longer termed goals and strategies.")

### How the decision shows on the play screen
- **D-06:** A **floating, minimizable AI Assist panel** on the play screen. It can be collapsed to a small tab and reopened; the terminal keeps its full width.
- **D-07:** The panel **opens automatically whenever the connected profile has AI activated** (the Phase 1 policy acceptance recorded), whether autopilot is ON or OFF. It is the AI's presence for that game. Profiles without acceptance see nothing new, so Phase 2's "hand play unchanged" criterion still holds for them.
- **D-08:** Panel content in this phase: each decision is one AI message showing the **short reasoning, then the command it issued**; state changes (engaged, waiting, resuming, disengaged, refusals, failures) appear as system lines. **No message input** until Phase 5 turns the panel into the coaching chat.
- **D-09:** In the terminal, the AI's issued command prints as **`[AI-ASSIST > north]`** and this line **replaces local echo** for AI-issued commands, so typed and AI-issued commands never look alike. The game's own echo and response follow as usual.
- **D-10:** The reasoning the owner reads is **a short plain-language "why" the model writes for the owner** (a sentence or three) plus exactly one command. The prompt asks for that shape; the model's internal thinking trace is not shown.

### `#AUTO` grammar (amends Phase 2 D-05)
- **D-11:** `#AUTO` accepts **only `ON` and `OFF`**. A bare `#AUTO`, `#AUTO STATUS`, or any other argument prints one local line to the effect of "Don't know what you're talking about. Your options are ON or OFF." The Phase 2 `#AUTO STATUS` diagnostic line is retired; the badge and the panel are the status surfaces. No directive starts, stops, or prints logging.

### Where decisions and reasoning are stored
- **D-12:** Each decision is stored on its own row: timestamp, profile and game connection, the game text window the model saw, the reasoning, the command issued, and the outcome (sent, refused, failed). The panel shows it live and **reloads the current connection's decisions after a page refresh** from these rows. This is how REQ-reasoning-visibility's "stored in the session log" is met without putting reasoning in the transcript.
- **D-13:** A first failure of any kind (API error, no command, several commands, a `#` directive or other non-game line, malformed answer) **sends nothing to the game**, shows an informative notice in the panel and the terminal, lands autopilot on OFF, and leaves regular play untouched (Phase 1 D-11). No retry in this phase; Phase 4 adds the threshold behaviour on top.

### Session transcript and the Logs section
- **D-14:** A **session transcript is stored for every saved-profile game connection**, from connect to disconnect, AI activated or not. Quick connects with no profile are not logged. It captures everything sent to the game and everything received from the game, with each sent line **marked as human-typed or AI-issued**. It does not contain reasoning. Sessions are keyed by profile and start date-time. (Owner: "logging is automatic based on the connection... capturing all of the session itself from connection to disconnection.")
- **D-15:** AI engage and disengage moments are recorded inside the transcript so a stint at the wheel (`#AUTO ON` to `#AUTO OFF`, the owner's chosen unit for an AI session) can be found later for Phase 6 debriefs. A WAITING stretch stays inside the same stint.
- **D-16:** Settings gets a **Logs** section in the left nav (per connection profile, like the other sections). Selecting it shows an **"Open logging"** button on the right. The button opens the log page **in a new browser tab; the current tab does not navigate**.
- **D-17:** The log page is **per game profile**: a left pane lists that profile's sessions by date/time, a right pane shows the selected session's transcript as text. Sign-in protected like the rest of the app and scoped to the signed-in owner's profile. No cross-profile listing, no share links.

### Model configuration and failure when absent
- **D-18:** A **model registry from environment configuration**: a default model plus any number of named entries, each with its API endpoint and API key. A profile's blank model name resolves to the default (Phase 1 D-09); a non-blank name selects a registry entry. Phase 3 implements the **Gemini API only**; other vendors plug in later without changing profiles. Nothing is hard-coded.
- **D-19:** The owner's key is on the **Gemini free tier** for this test. Its rate limits are a fact the planner records: the per-session call cap is the owner's cost and rate protection, and a rate-limit answer from Gemini is a failure under D-13.
- **D-20:** When the key or default model is missing from the environment, **the server starts normally** and **`#AUTO ON` is refused** with a clear notice such as `[Autopilot refused: AI is not configured on this server]`; the switch stays OFF and hand play is unaffected.

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

### Folded Todos
- **`2026-09-15-phase3-security-carry-forward.md` (security).** Two risks deferred at the Phase 2 security review must be handled in this phase, not merely re-presented:
  - **DR-2-01** (medium, owner: "MUST FIX in next Phase"): staging prints the one-time sign-in code in the deploy log; anyone with staging log access can sign in as any staging user. Plan the remediation into Phase 3 (gate the line behind an env flag that is off by default, or log a hash; confirm production has no equivalent; review Railway log access). Its closure is recorded at the Phase 3 security review.
  - **DR-2-02** (low, owner: "Clean this up in Phase 3"): six throwaway connection profiles remain on the owner's staging account (Phase 2 Harness, Harness B, Walkthrough, Refusal, Hand Play; Phase 1 Evidence Walkthrough). Delete them, or use them as Phase 3 fixtures and delete them before the Phase 3 security review. Note: profiles are deleted through the app with the owner's signed-in session; never sign the owner out.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Product definition
- `.specify/specs/ai-game-player-design-v3.md` — Deliverable D3 (one AI decision) is this phase; D6 (measurement and memory) defines learned notes and the debrief that D-05 and D-15 leave room for; Definition of complete items 2 and 6. The design's "session log with decisions and reasoning and manual-driving stretches" is met by D-12 plus D-14.
- `.specify/specs/safety-and-abuse-policy-v1.md` — Sections 2 (supervised operation), 5 and 6 (data handling: captured game text stored for tool operation only; stored credentials). D-14 stores full transcripts for every profile connection and must be raised at the Phase 3 security review against the data-handling sections.

### Method
- `.specify/memory/phase-based-development-approach.md` — Binding Phase → Wave → Plan → Task rules; plans are capabilities with observable acceptance criteria; the phase must be provable.
- `CLAUDE.md` — Distilled method rules and project facts. Evidence is a canned report, log file, or end-user screenshot; database queries never count as proof. Every phase has a Brief for planning approval, an Evidence Dossier for acceptance, and a Risk Register for the security review.

### Planning state
- `.planning/ROADMAP.md` §Phase 3 — Goal, four success criteria, Phase Validation line, implementation notes (output tap, migration `011`+, Gemini client, new websocket message type, driver under `internal/`).
- `.planning/REQUIREMENTS.md` — REQ-single-decision, REQ-reasoning-visibility, REQ-env-config; REQ-safety-limits-hold (Phase 4) names the blank-setting and disconnect behaviour this phase must not break.
- `.planning/PROJECT.md` — Locked decisions (server-side Go driver through the ICM automation context; server-side is not unattended; autopilot waits across a disconnect), constraints CON-model-config, CON-profile-schema, CON-conservative-defaults, CON-engine-has-no-game-knowledge, CON-policy-data-handling.
- `.planning/phases/01-profile-foundation-and-policy-gate/01-CONTEXT.md` — D-09 (blank model name means the server default env var, added here), D-11 (fail informatively, disengage, never crash), D-12 (conduct rules and guidance verbatim).
- `.planning/phases/02-autopilot-switch/02-CONTEXT.md` — State model D-01 to D-06 (D-05 `#AUTO STATUS` is retired by this phase's D-11), wheel-grab D-07 to D-09, notices D-10 to D-12; the `[AI-PLAYER]` log format; the Phase 3 hook noted in its integration points.
- `.planning/phases/02-autopilot-switch/02-SECURITY.md` and `.planning/RISK-REGISTER.md` — DR-2-01 and DR-2-02, the deferred risks folded above; AR-2-01 (unbounded WAITING) is accepted and not reopened.
- `.planning/todos/pending/2026-09-15-phase3-security-carry-forward.md` — The folded todo, with the owner's notes verbatim.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/icm`: `Engine` (`NewEngine`, `Process`, `Classify`, `GetDispatcher`), `Dispatcher.Dispatch(ctx, sessionID, normalized)` with `checkSafety` and a pluggable `SafetyChecker` (`DefaultSafetyChecker`: recursion depth, circuit breaker, rate limit, queue depth, timeout), `ExecutionContext` with `ContextAutomation = "automation"` and its permission table in `types.go`, `Classifier` in `pass_through.go` (plain game text is `ShouldPassThrough`), and `Handler.RegisterRoutes(mux)` for `/api/v1/icm/{validate|normalize|execute|preview}`. Nothing imports any of it today; `icm_test.go` is the only existing test.
- `internal/session/manager.go`: one session per user; `Session{UserID, Host, Port, ConnectionID, State, ...}`; `SendCommand(userID, command)` and `ReadOutput(userID, buffer)` are the only paths to and from the MUD, so the transcript tap and the recent-text window hang off them and catch both browser-sent and driver-sent commands. `EngageAutopilot`, `DisengageAutopilot(userID, cause)`, `AutopilotStateFor`, `AutopilotConnectionIDFor`, `parkAutopilotLocked`, `resumeAutopilotLocked` are the hooks for D-01, D-02, D-13; `logAutopilotTransition` is the `[AI-PLAYER]` line pattern.
- `internal/session/websocket.go`: `WSMessage{Type, Data, Status, Source, ConnectionID, ...}` with `MsgTypeAutopilot` already pushed and handled by the browser (`wsManager.onAutopilot`); `readMUDOutput` and `relayMUDToClient` are the output path (telnet IAC stripped in `stripTelnetIAC`; ANSI not stripped); `handleClientCommands` is the single ingress with `IsHumanSource` and `applyWheelGrab`. A driver-issued command must bypass the wheel-grab (it is not human) but still be rate-limited and logged.
- `internal/session/handler.go`: `Autopilot` handler for `POST /api/v1/session/autopilot` (engage/disengage/status) with the engage-gate callback; `StatusResponse` carries `autopilot_state` and connection id; the missing-config refusal (D-20) belongs on the engage path here.
- `internal/config/config.go`: `Config` struct and `Load()` with the required/optional env pattern (`SESSION_SECRET` fatal, others defaulted with a warning). The model registry and Gemini settings join it; absence must not be fatal (D-20). The sign-in code log line for DR-2-01 is in `internal/auth`.
- `internal/store/profile.go`: `Profile.ConductRules`, `Profile.ApproachGuidance`, `AISettings{ModelName, CallCap *int, DisengageThreshold *int}`, `EngageGateAllowed`; `GetProfile` by connection id gives the driver tier two of the prompt.
- `migrations/010_add_ai_fields.up.sql`: the `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` pattern; next number is `011`. New tables follow the same up/down pair.
- `frontend/src/services/icm-adapter.ts`: `validateCommandRemote`, `normalizeCommandRemote`, `executeCommandRemote`, `previewCommandRemote` call `/api/v1/icm/*` and `processCommand` / `validateAndNormalize` fall back to browser logic on failure; success criterion 4 removes the silent fallback.
- `frontend/src/services/automation.ts`: the `#AUTO` directive (02-04-01, `autopilotControl` callback) is where D-11 narrows the grammar; `echoLocal` writes bracketed local lines.
- `frontend/src/services/api.ts`: `wsManager` message dispatch (`case 'autopilot'`), `onAutopilot`/`offAutopilot`, and the `session/autopilot` fetch; the new decision messages and the decisions reload fetch go here.
- `frontend/src/pages/PlayScreen.tsx` and `frontend/src/components/AutopilotBadge.tsx`, `Header.tsx`: the play screen and badge the AI Assist panel joins; `previousAutopilotStateRef` transition effect writes the Phase 2 notices.
- `frontend/src/pages/SettingsPage.tsx` (`SettingsSection` union and nav list) and `frontend/src/components/AIPlayerPanel.tsx`: the Logs section is one more entry following the AI Player section's per-connection pattern; the "AI activated" signal for D-07 is the acceptance status the AI Player section already reads.

### Established Patterns
- Server owns truth; the browser reflects it. Autopilot state and now decisions flow server → browser over the existing websocket; after a refresh the browser re-syncs from the server.
- Local notices are bracketed lines written through the automation engine's echo path, never sent to the MUD.
- One structured `[AI-PLAYER]` log line per event is the diagnostic evidence format; add lines per decision (request, answer, dispatch, failure) and per transcript session open/close.
- Defaults resolve in Go (`normalizeSettings`, `DefaultProfileSettings`); blank profile fields are resolved server-side, so blank model name → registry default happens in the driver, not the browser.
- Validation and refusals are server-side with specific messages; the frontend shows the returned string.
- Tests: `internal/icm/icm_test.go`, Phase 1 store/handler tests, Phase 2 `autopilot_test.go`, `handler_test.go`, `websocket_test.go`, `manager_test.go`. The ICM diagnostic test (success criterion 4), the window and ANSI-strip tests, the answer-parsing and failure tests, and the missing-config refusal test continue that.
- Evidence-as-file: every success criterion is proven by a report line, log line, or screenshot on staging; the Phase 2 canned-report harness (`scripts/verify-phase2.sh`) is the model for a Phase 3 harness.

### Integration Points
- ICM engine constructed in `cmd/server/main.go`, routes registered, and the driver calling `Dispatch` with `ContextAutomation` before `manager.SendCommand`.
- Recent-text window and transcript tap on `Manager.ReadOutput` / `SendCommand` (or the relay goroutines that wrap them), opened on `Connect`, closed on `Disconnect`.
- Driver trigger on the engage path (`EngageAutopilot` success) and on `resumeAutopilotLocked`; failure path calls `DisengageAutopilot(userID, cause)` with a new cause and pushes a notice.
- New websocket outbound messages for decisions, system lines, and the `[AI-ASSIST > ...]` echo; a REST read for the current connection's decisions on attach.
- Gemini env config and registry in `internal/config`; refusal wired into the `Autopilot` handler's engage branch.
- New routes: decisions read, transcript list and read per profile, and the log page; the Logs section and button in `SettingsPage.tsx` with `target="_blank"`.
- DR-2-01: the sign-in code log line in `internal/auth`, gated or hashed.
- Phase 4 will loop D-01's single decision; Phase 5 adds the panel's input; Phase 6 adds learned notes to tier two and hangs the debrief off the stint markers in D-15.

</code_context>

<specifics>
## Specific Ideas

- Owner on memory: "there should be a running memory kept for the game profile for the AI that it references. It should reference immediate buffer for reaction stuff, but also a memory file for longer termed goals and strategies." Built as the two-tier prompt (D-05); the writable long-term tier is Phase 6.
- Owner on the panel: "if you have activated the AI assist from the settings on a profile, you get a pop-up chat interface with the AI model where you can see its reasoning and interact with it during the game." Reasoning now (D-06 to D-08); interaction in Phase 5.
- Owner on the terminal line: "`[AI-ASSIST > north]` -- and that should substitute for local echo to differentiate between typed commands."
- Owner on `#AUTO`: "I don't think auto status should do anything. It should just be auto on, auto off. And if you put something else in there, it just says, don't know what you're talking about. Your options are on or off."
- Owner on the log: "logging is automatic based on the connection, not based on when auto is on or off... capturing all of the session itself from connection to disconnection." "The log only tracks what was sent to the game and what was sent back from the game. But it will differentiate the user typing from the AI typing." "Logs should be stored by date time connections."
- Owner on the Logs UI: "create under settings a logs section on the left... the modal on the right lights up with open up logging, and then it'll jump you over to the page. But importantly, I don't want it to move away. The current tab stays where it is. It has to open it in a new tab." "It will only show the logs associated with the game profile you clicked the button from... a list of date/time log links. Maybe we build a web interface that allows you to highlight them on the left pane and it displays the text on the right."
- Owner on models: "For this test we were using Gemini free tier, but I would like the option for adding more models using API keys/endpoints."
- The owner does not want plumbing turned into decisions; pick the simplest thing and move on (carried from Phases 1 and 2).

</specifics>

<deferred>
## Deferred Ideas

- **Coaching input in the AI Assist panel** — Phase 5 (REQ-coaching-chat). The panel built here is its home; Phase 5 adds the message box and the pause/resume controls.
- **Learned notes and an AI-written running memory** — Phase 6 (REQ-session-debrief, REQ-memory-carryover). Tier two of the prompt gains learned notes then; D-05 keeps the prompt shape stable.
- **Non-Gemini model vendors** — the registry (D-18) carries endpoint and key per entry so another vendor's client can be added later; no implementation in this version unless the owner asks.
- **Per-profile tuning of the recent-text window** — a server constant now (D-04); revisit in Phase 4 if a game's output volume makes it matter.
- **Cross-profile log listing, share links without sign-in, transcript search or export** — not built; the per-profile page (D-17) is the whole surface.
- **Security review items for this phase (raise, do not decide here):** D-14 stores full game transcripts for every saved-profile connection, including hand play, against policy sections on data handling; the Gemini key on staging and the free-tier limits; the log page's authorization scope; the driver bypassing the human wheel-grab check; and the closure of DR-2-01 and DR-2-02.
- Shelf copy: `D:\Projects\ai-mud-player\documents\` is the owner's approved-version shelf; the repo copy of the design is the source of truth. Phase 2 D-05's retirement (D-11) and the transcript scope (D-14) are implementation decisions, not design amendments; no shelf change is needed.

</deferred>

---

*Phase: 03-one-ai-decision*
*Context gathered: 2026-09-15*
