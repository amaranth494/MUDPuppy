---
phase: 03
slug: one-ai-decision
status: verified
threats_open: 0
asvs_level: 1
created: 2026-09-16
register_authored_at_plan_time: true
accepted: 14
deferred: 7
remediate_now: 0
---

# Phase 03 — One AI Decision — Security

> Per-phase security contract: threat register, accepted risks, deferred risks, and audit trail.
> Register authored at plan time across 03-01-PLAN.md through 03-13-PLAN.md (45 threat ids, 76 verification rows because an id recurs across plans). Verified against implemented code, not documentation, on 2026-09-16 by gsd-security-auditor (`03-SECURITY-AUDIT.md`, status SECURED). The two critical and two warning findings of the code review (`03-REVIEW.md`) were fixed in cbf7d97, 0b74ea7, 48165ab and 2a2f349 before this review and independently re-verified in the tree.
>
> **This audit closes every threat whose plan disposition is `mitigate` and finds the declared mitigation present in code; `accept`, `transfer` and `defer` dispositions were confirmed documented.** The owner's decisions on every open item were made on the Phase 3 Risk Register (https://claude.ai/artifact/HauUDyWUkqutt7LuXWse1m) and read back from it; they are the Accepted Risks Log and the Deferred Risks below. No item was marked Remediate Now, so Phase 3 does not reopen.

---

## Trust Boundaries (consolidated across all 13 plans)

| Boundary | Plans | Description |
|----------|-------|-------------|
| MUD socket → `Manager` memory / Postgres | 03-01, 03-05 | Untrusted remote text (including other players' words) is copied into a server-side ring and, for saved profiles, persisted |
| Websocket/REST goroutines → `Manager` maps | 03-01, 03-05, 03-09 | Concurrent goroutines read/write shared state under `m.mu` (or a dedicated mutex) |
| Go process → server log | 03-01, 03-05, 03-07, 03-08, 03-09, 03-13 | Anything logged from these paths could be MUD content, reasoning, a key, or a sign-in code |
| Go process → Google's API (public internet) | 03-02 | The Gemini API key crosses this boundary on every call |
| Environment → process config | 03-02, 03-06 | Key, model identity and the engage gate arrive as env vars set on Railway staging |
| Vendor response → process | 03-02, 03-08 | The model's answer is untrusted text that becomes a game command |
| Browser → `/api/v1/icm/*`, `/api/v1/session/autopilot`, `/api/v1/profiles/{id}/decisions`, `/api/v1/profiles/{id}/sessions[/{id}]` | 03-03, 03-06, 03-09, 03-05 | Newly reachable routes inside the existing session middleware |
| AI driver → ICM dispatcher / websocket wheel-grab | 03-03, 03-08 | The AI's command must pass the same safety gate and never bypass the human send path |
| Browser fallback logic → the game | 03-03 | A browser that recomputes an ICM answer locally could send an unapproved command |
| Typed input → directive grammar → the MUD | 03-04 | A `#` directive must never leak through as a game command |
| Driver goroutine → owner's websocket | 03-09 | A background goroutine writes to a connection owned by the request loop |
| Server payload → rendered DOM | 03-10, 03-11 | Model- and transcript-derived text is untrusted and rendered in the browser |
| Settings tab → opened tab | 03-11 | A `target="_blank"` anchor can reach back via `window.opener` |
| Operator shell → staging over HTTP; harness/fixtures → repository | 03-12 | The harness carries a real session cookie; committed fixtures/reports are disclosed |
| Repository → Railway staging; owner's API key → Railway env vars; evidence files → readers | 03-13 | Deployed binary may diverge from repo; a real credential is set; committed evidence is disclosed |

---

## Threat Verification Register

Legend: **M**=mitigate, **A**=accept, **T**=transfer, **D**=defer. All rows CLOSED.

| Threat ID | Category | Plan(s) | Disp. | Evidence | Status |
|-----------|----------|---------|-------|----------|--------|
| T-3-01 | Tampering/EoP (prompt injection → command) | 03-02 | T | `internal/gemini/client.go:21-24` doc comment transfers validation to the driver, naming plan 03-08/T-3-01 | CLOSED |
| T-3-01 | (same) | 03-08 | M | `internal/driver/driver.go:409-431` `validateCommand` rejects empty/multi-line/`#@$%`-prefixed/oversize/control-char commands, called at `driver.go:261` before `Dispatch` (line 273) and `SendCommandAs` (line 280); `go test ./internal/driver/... -run TestHandleEngageFailures -race` PASS (`hash_prefixed_command`, `two_line_command`, `empty_command`, `unparseable_answer` subtests) | CLOSED |
| T-3-02 | Info Disclosure (API key) | 03-02 | M | `internal/gemini/client.go:205` `req.Header.Set("x-goog-api-key", ...)`, zero `log.` calls anywhere in `internal/gemini/*.go`, no `https://` literal in the package | CLOSED |
| T-3-02 | (same, refusal message) | 03-06 | M | `internal/session/handler.go:388` fixed string `"Autopilot refused: AI is not configured on this server"`, no interpolation; `go test ./internal/session/... -run TestAutopilotHandler_AIConfigRefusal -race` PASS | CLOSED |
| T-3-02 | (same, driver log) | 03-08 | M | `internal/driver/driver.go:247` logs `model=%s kind=%s http=%d` only, never key/endpoint | CLOSED |
| T-3-02 | (same, staging evidence) | 03-13 | M | `grep -inE "apikey|x-goog-api-key|generativelanguage|AIza"` on `evidence/04-staging-ai-player.log` → 0 matches | CLOSED |
| T-3-03 | Info Disclosure (IDOR, decisions) | 03-09 | M | `internal/profiles/decisions.go:82-94` resolves `GetProfileByConnection` before any row read; refusal subtest in `decisions_test.go` | CLOSED |
| T-3-04 | Info Disclosure (IDOR, sessions/transcript) | 03-05 | M | `internal/profiles/logs.go:40-42,90-94`; SQL-level defence-in-depth `internal/store/transcripts.go:143-149,182-190` (`WHERE connection_id = $2` / `gs.connection_id = $2`) | CLOSED |
| T-3-04 | (client-side, log page) | 03-11 | M | `frontend/src/pages/LogsPage.tsx:14-22` — connection id taken only from `useParams`, never merged; server-side check above is the actual gate | CLOSED |
| T-3-05 (DR-2-01) | Info Disclosure/Spoofing (OTP in log) | 03-07 | M | `internal/auth/handler.go:56-61` `logOTPIssued` gated by `h.logOTPCode` (from `AUTH_LOG_OTP`, default false); `go test ./internal/auth/... -run TestOTPNotLogged -race` PASS (3 subtests) — **see residual note below** | CLOSED |
| T-3-05 (DR-2-01) | (staging evidence) | 03-13 | M | `grep -n "code:\|@"` on `evidence/04-staging-ai-player.log` → 0 matches | CLOSED |
| T-3-06 | EoP (driver bypassing human wheel-grab) | 03-08 | M+raise | `internal/driver/driver.go:273,280` — `Dispatch` called before `SendCommandAs(..., "ai")`; `SendCommandAs` (`internal/session/manager.go:820-845`) marks the transcript with `source` | CLOSED |
| T-3-06 | (residual, unsupervised send path) | 03-13 | D | `03-SECURITY-AGENDA.md` Item 5 ("The driver sends without passing the human wheel-grab"), undecided, 3 dispositions offered | CLOSED (deferred, documented) |
| T-3-07 | Info Disclosure (window/reasoning/log) | 03-01 | M | `internal/session/window.go` — zero `log.` calls in the file | CLOSED |
| T-3-07 | (same) | 03-08 | M | `internal/driver/driver.go:247,378` — ids/model/stage/kind/http/lengths only, no reasoning/command/window text | CLOSED |
| T-3-07 | (same) | 03-09 | M | `internal/session/websocket.go:228` `ai-push user_id=%s outcome=failed error_class=%T`; `internal/profiles/decisions.go:84-91,108-109,127-128` — ids/outcome/count only | CLOSED |
| T-3-07 | (staging evidence) | 03-13 | M | `evidence/04-staging-ai-player.log` reviewed — only ids/events/stages/counts present, no reasoning or game text | CLOSED |
| T-3-08 | DoS (data race) | 03-01 | M | `ringBuffer` fixed at `RecentOutputWindowBytes`, all access via `m.mu`-guarded methods | CLOSED |
| T-3-08 | (same) | 03-05 | M | transcript map access via locked helpers | CLOSED |
| T-3-08 | (same) | 03-09 | M | `internal/session/websocket.go:166-169` `clientsMu sync.RWMutex`, separate from write mutex | CLOSED |
| T-3-08 | (verification) | all above | — | `go test ./internal/session/... -race -count=1` PASS; `go test ./internal/... -race -count=1` PASS (all packages) | CLOSED |
| T-3-09 | Info Disclosure (policy, retention) | 03-05 | D | `03-SECURITY-AGENDA.md` Part A Item 1, quoting policy §6 verbatim, undecided | CLOSED (deferred, documented) |
| T-3-09 | (same) | 03-11 | D | Same Item 1 | CLOSED (deferred, documented) |
| T-3-09/T-3-15 | (same, combined) | 03-13 | D | Same Item 1, explicitly combined row | CLOSED (deferred, documented) |
| T-3-10 | Repudiation (unfalsifiable harness) | 03-12 | M | `evidence/02-harness-selftest.txt` — positive run 0 failures; negative-fixture run: 4 real `FAIL C3` lines, non-zero exit | CLOSED |
| T-3-10 | (same) | 03-13 | M | Same evidence file, re-cited | CLOSED |
| T-3-11 | Spoofing (push to wrong user) | 03-09 | M | `internal/session/websocket.go:191-211` registry keyed by authenticated `userID` from request context only; `unregisterClient` compares connection identity before deleting | CLOSED |
| T-3-11 | (panel merge) | 03-10 | M | `frontend/src/components/AIAssistPanel.tsx:44,52,72` — fetch/effect scoped to `connectionId` prop, never merges | CLOSED |
| T-3-14 (AR-2-01) | Policy (resume fires real command) | 03-08 | D | `03-SECURITY-AGENDA.md` Item 2, quoting policy §2 | CLOSED (deferred, documented) |
| T-3-14 | (same) | 03-13 | D | Same Item 2 | CLOSED (deferred, documented) |
| T-3-15 | Info Disclosure (window_text retained/served) | 03-01 | D | Agenda Item 1 | CLOSED (deferred, documented) |
| T-3-15 | (same) | 03-08 | D | Agenda Item 1 | CLOSED (deferred, documented) |
| T-3-15 | (endpoint omission) | 03-09 | M | `internal/profiles/decisions.go:20-31` `DecisionResponseItem` has no `window_text` field | CLOSED |
| T-3-16 | Info Disclosure (key in test fixture) | 03-02 | M | `internal/gemini/client_test.go:14` `testFakeKey = "test-key-not-a-real-credential"`; no `https://` in package | CLOSED |
| T-3-17 | Tampering (wrong target) | 03-12 | M | `scripts/verify-phase3.sh:165,177,180` records `BASE_URL` and `git rev-parse --short HEAD` in the report header | CLOSED |
| T-3-17 | (same) | 03-13 | M | Same mechanism; staging-only `railway up` deploys confirmed by header in `evidence/03-canned-report.txt` | CLOSED (see note on Agenda Item 12 below) |
| T-3-19 | DoS (vendor cost/rate limit) | 03-02 | A | `internal/gemini/client.go` `KindRateLimited` (const `rate_limited`) surfaced, no retry loop anywhere in `client.go` or `driver.go` | CLOSED |
| T-3-20 | EoP (two independent ICM engines) | 03-03 | M | `cmd/server/main.go:227-228,273` — exactly one `icm.NewEngine()` call in production wiring; `NewHandler()`/`GetDefaultEngine()` confirmed unused outside tests; driver and HTTP routes share `icmEngine.GetDispatcher()` | CLOSED |
| T-3-21 | Tampering (browser recompute on failure) | 03-03 | M | `frontend/src/services/icm-adapter.ts:592-634` — `useBackend=true` path's `catch` returns `shouldPassThrough:false` with an error; local recompute only reachable via explicit `useBackend=false` opt-out | CLOSED |
| T-3-22 | Spoofing/EoP (unauthenticated ICM routes) | 03-03 | M | `cmd/server/main.go:404,567` — `icmHandler.RegisterRoutes(mux)` on the same `mux` later wrapped by `sessionMiddleware` | CLOSED |
| T-3-23 | Repudiation (unfalsifiable criterion-4 test) | 03-03 | M | `internal/icm/dispatcher_test.go` `TestDispatch_AutomationPassThrough_GoesThroughSafetyChecker` — `go test ./internal/icm/... -run ... -v` PASS, all 3 subtests including `refused_when_rate_limit_tripped` and `process_does_not_reach_dispatch_for_a_plain_command` | CLOSED |
| T-3-24 | Tampering (`#AUTO` unknown arg leaks) | 03-04 | M | `frontend/src/services/automation/evaluator.ts:1175-1188` — unrecognised arg returns via `outputMessage` only, `setState`/fetch only reached in the `else` branch for a valid action | CLOSED |
| T-3-25 | Repudiation (retired `#AUTO STATUS` line) | 03-04 | A | Confirmed `rawArg === 'STATUS'` falls into `action === null` branch (no status dispatch); badge/panel are the status surfaces | CLOSED |
| T-3-26 | DoS (DB write on MUD read path) | 03-05 | M | `internal/session/transcript.go:15-17` buffered channel + non-blocking `enqueue`; `go test ./internal/session/... -run TestTranscriptDoesNotBlockReader -race` PASS | CLOSED |
| T-3-27 | Info Disclosure (transcript text in log) | 03-05 | M | `internal/session/manager.go:573,592`, `internal/session/transcript.go:154,207,212` — ids/event/counts/error type only | CLOSED |
| T-3-28 | Tampering (invalid `source` value) | 03-05 | M | `migrations/011_add_ai_session_tables.up.sql:32` `CHECK (source IN (...))`; `internal/store/transcripts.go:85-88` Go-side rejection before the transaction | CLOSED |
| T-3-29 | EoP (engage with no model configured) | 03-06 | M | `internal/session/handler.go:384-389` refusal precedes `EngageAutopilot`; `TestAutopilotHandler_AIConfigRefusal/refuses_on_when_unconfigured` PASS | CLOSED |
| T-3-30 | Spoofing (blurred refusal reasons) | 03-06 | M | `gate_refusal_takes_precedence` subtest PASS — policy gate and config gate keep distinct causes (`refused-gate` vs `refused-not-configured`) | CLOSED |
| T-3-31 | DoS (nil/empty config breaks switch) | 03-06 | M | `off_is_unaffected_when_unconfigured` and `nil_config_refuses` subtests PASS | CLOSED |
| T-3-32 | Info Disclosure (replacement audit line) | 03-07 | M | `TestOTPNotLogged/suppressed_line_carries_no_identifiers` PASS | CLOSED |
| T-3-33 | Repudiation (losing issuance record) | 03-07 | M | `internal/auth/handler.go:61` issuance line still printed when suppressed; `logs_issuance_without_the_code` subtest PASS | CLOSED |
| T-3-34 | Info Disclosure (Railway log access) | 03-07 | D | `03-SECURITY-AGENDA.md` Part C, DR-2-01 closure section, "review Railway log access" — 3 dispositions, undecided | CLOSED (deferred, documented) |
| T-3-35 | DoS (double fire, engage+resume race) | 03-08 | M | `internal/driver/driver.go:140-190` `inFlight map[string]bool` guard; `TestHandleEngage/repeat_engage_fires_nothing` and `resume_fires_one_fresh_decision` PASS | CLOSED |
| T-3-36 | DoS (slow/dead browser blocks driver) | 03-09 | M | `internal/session/websocket.go:219-232` `PushAI` no-op for absent user; driver discards push error (verified: `TestPushAI` cited in review/verification, driver code does not wait on push) | CLOSED |
| T-3-37 | Tampering (script injection, panel) | 03-10 | M | `grep -rn "dangerouslySetInnerHTML\|innerHTML"` on `AIAssistPanel.tsx` → 0 matches; text rendered as JSX children | CLOSED |
| T-3-37 | (log page) | 03-11 | M | Same grep on `LogsPage.tsx` → 0 matches | CLOSED |
| T-3-38 | Spoofing (AI command looks typed) | 03-10 | M | `frontend/src/pages/PlayScreen.tsx:418` `[AI-ASSIST > ...]` + `brightmagenta`; `frontend/src/index.css:22,325-328,444` — `--color-ai-accent` used identically in terminal, panel (`.ai-decision-command`) and log page (`.logs-transcript-line.ai`); `LogsPage.tsx:162` same bracket format | CLOSED |
| T-3-39 | Info Disclosure (panel on non-opted-in profile) | 03-10 | M | `frontend/src/pages/PlayScreen.tsx:45,54,565-566` — mount gated on `policy.accepted` | CLOSED |
| T-3-40 | Tampering (reverse tabnabbing) | 03-11 | M | `frontend/src/pages/SettingsPage.tsx:1706-1710` `rel="noopener noreferrer"` on the `target="_blank"` anchor | CLOSED |
| T-3-41 | DoS (second tab disturbs live session) | 03-11 | M | `frontend/src/App.tsx:189-190` — `/logs/:connectionId` is a sibling `Route` to the `*` catch-all that renders `AppContent`; only one JSX `<PlayScreen />` render exists, inside `AppContent` (note: the plan's literal `grep -n "PlayScreen"` count of "exactly one" is stale — the substring now also matches an import and 3 comments, 5 total — but the security-relevant fact, a single render site inside `AppContent`, holds; documentation imprecision, not a code gap) | CLOSED |
| T-3-42 | Info Disclosure (cookie/key/game text in harness output) | 03-12 | M | No SQL/key/cookie patterns in `scripts/verify-phase3.sh`; no address/key/cookie patterns in `scripts/fixtures/phase3{,-negative}/*.json` | CLOSED |
| T-3-43 | Repudiation (proving via direct DB query) | 03-12 | M | `grep -inE "psql|database/sql|pq\.|sql\.Open|lib/pq"` on `verify-phase3.sh` → 0 matches; harness calls only application endpoints | CLOSED |
| T-3-44 (DR-2-02) | Info Disclosure (throwaway profiles left on staging) | 03-13 | M | `03-SECURITY-AGENDA.md` Part C — profiles deleted through the app with the owner's own session; `evidence/14-connections-cleaned.png` present | CLOSED |
| T-3-45 | Repudiation (carried-forward risks forgotten) | 03-13 | M | `03-SECURITY-AGENDA.md` Part C — both DR-2-01 and DR-2-02 closures recorded with original wording and cited evidence | CLOSED |
| T-3-SC | Tampering (supply chain) | all 13 plans | A | `git diff --stat 6e98595 HEAD -- go.mod go.sum frontend/package.json frontend/package-lock.json` → empty; `evidence/01-test-report.txt` `### DEPENDENCY DRIFT` section empty, `EXIT: 0` | CLOSED |

**Row count:** 76 threat-occurrence rows (one per threat id × plan it appears in), all CLOSED. 0 OPEN.

---

## Code Review Fix Verification (03-REVIEW.md)

All four findings independently confirmed fixed in the current tree (not merely present as commits — checked `git merge-base --is-ancestor` against HEAD, then read the resulting code):

| Finding | Commit | Verified in code |
|---------|--------|-------------------|
| CR-01 (DB I/O under manager-wide lock) | `cbf7d97` | `internal/session/manager.go:277-286` (`Connect`) and `:402-414` (`Disconnect`) release `m.mu` around `openTranscript`/`closeTranscript`'s blocking I/O |
| CR-02 (send-on-closed-channel panic) | `0b74ea7` | `internal/session/transcript.go:66,84,92,169,202` — `close()` closes a `stop` channel, never `t.lines` |
| WR-01 (React key collision) | `2a2f349` | `frontend/src/components/AIAssistPanel.tsx:155` `const key = entry.id \|\| \`no-id-${index}\`` |
| WR-02 (internal error text leaked) | `48165ab` | `internal/profiles/handler.go:223-227` — logs `err`, returns fixed `"Failed to get profile"` |

`go test ./internal/... -race -count=1` — all packages PASS (auth, config, driver, gemini, icm, policy, profiles, session, store).

---

## Accepted Risks Log

Decisions recorded by the owner on the Phase 3 Risk Register (https://claude.ai/artifact/HauUDyWUkqutt7LuXWse1m) on 2026-09-16 and read back from it. Accepted risks are closed for good unless the owner reopens them; they are not re-presented at later reviews.

| Risk ID | Register item | Criticality | Risk | Proposed remediation (not taken) | Owner note | Accepted By | Date |
|---------|---------------|-------------|------|----------------------------------|------------|-------------|------|
| AR-3-01 | R-02 / Item 2 / T-3-14, AR-2-01 | medium | A resume from Waiting now acts, and it has no time bound | A bounded Waiting lifetime after which the switch lands on Off instead of resuming. | I've already accepted this risk in previous Phases. If the policy is the reason this keeps coming up, then adjust the policy. | Owner, Accept Risk on the Phase 3 Risk Register | 2026-09-16 |
| AR-3-02 | R-03 / Item 3 / T-3-04 | low | The log page trusts the session cookie alone | A share link with its own expiry, or a confirmation step before the log page opens. | — | Owner, Accept Risk on the Phase 3 Risk Register | 2026-09-16 |
| AR-3-03 | R-04 / Item 4 / T-3-02, T-3-19 | medium | The Gemini key on staging, the free tier, and no call cap until Phase 4 | Nothing beyond Phase 4's call cap, or an early stopgap cap at the driver. | Having an option and we can set it whenever we want. This is not a Risk. | Owner, Accept Risk on the Phase 3 Risk Register | 2026-09-16 |
| AR-3-04 | R-05 / Item 5 / T-3-06, T-2-01 | medium | The driver sends to the game without passing the human wheel-grab | A browser-side control that can interrupt an in-flight driver send, or an explicit acceptance that the ICM safety checker is the mediation for this path. | This is by design. Not a risk. | Owner, Accept Risk on the Phase 3 Risk Register | 2026-09-16 |
| AR-3-05 | R-07 / Item 7 | low | A server-only plan promised a browser behaviour and the gap was only caught live | A plan-checker rule that flags a server-only plan whose acceptance criterion describes something visible in the browser. | — | Owner, Accept Risk on the Phase 3 Risk Register | 2026-09-16 |
| AR-3-06 | R-09 / Item 9 / D-12 | low | "Survives a refresh" means reconnect the same profile, then it comes back | State the reconnect in the requirement, or keep the panel mounted for the last-selected profile across a refresh. | — | Owner, Accept Risk on the Phase 3 Risk Register | 2026-09-16 |
| AR-3-07 | R-11 / Item 11 / T-3-07 | low | The new model-error log line carries a kind and an HTTP status | None if the field set stands; otherwise redact further. | — | Owner, Accept Risk on the Phase 3 Risk Register | 2026-09-16 |
| AR-3-08 | R-12 / Item 12 | medium | A key was set on production and production redeployed | Remove GOOGLE_API_KEY from production and note the incident in the deploy runbook. | We're going to need it in Prod. Leave it there. | Owner, Accept Risk on the Phase 3 Risk Register | 2026-09-16 |
| AR-3-09 | R-14 / Item 14 | medium | Two staging session tokens were pasted into the chat | The owner signs out or rotates the current session at their convenience; future harness runs by the owner or through a short-lived token. | — | Owner, Accept Risk on the Phase 3 Risk Register | 2026-09-16 |
| AR-3-10 | R-15 / Item 15 | low | The harness self-test could not see the live-shape bugs it shipped with | A recorded-live fixture in the self-test suite. | — | Owner, Accept Risk on the Phase 3 Risk Register | 2026-09-16 |
| AR-3-11 | R-16 / Item 16 / DR-2-01 residual | low | Two dev-mode lines still print the sign-in code when no mail server is configured | Route the two lines through the same gated helper. | — | Owner, Accept Risk on the Phase 3 Risk Register | 2026-09-16 |
| AR-3-12 | R-17 / Item 17 / AR-2-11 | low | The race detector depends on one machine's compiler | A standing CI step that runs -race with its own compiler. | — | Owner, Accept Risk on the Phase 3 Risk Register | 2026-09-16 |
| AR-3-13 | R-19 / T-3-SC | low | Supply chain: no new packages this phase | Nothing to remediate. Keep the drift check in every phase's test report. | — | Owner, Accept Risk on the Phase 3 Risk Register | 2026-09-16 |
| AR-3-14 | R-21 / IN-02 | low | The harness's no-jq fallback can match the wrong JSON field | Require jq for live mode, or parse with a small embedded script. | — | Owner, Accept Risk on the Phase 3 Risk Register | 2026-09-16 |

*Accepted risks do not resurface in future audit runs.*

---

## Deferred Risks

Temporarily accepted so Phase 3 can close. Each is raised again at the Phase 4 security review with the same three choices. The project cannot be considered closed while any deferral is active. The owner's notes carry instructions into Phase 4 (see `.planning/todos/pending/2026-09-16-phase4-security-carry-forward.md`).

| Risk ID | Register item | Criticality | Risk | Proposed remediation | Owner note | Deferred By | Date | Raised again at |
|---------|---------------|-------------|------|----------------------|------------|-------------|------|-----------------|
| DR-3-01 | R-01 / Item 1 / T-3-09, T-3-15 | medium | Every session transcript and every decision window is kept forever | A retention window after which transcript lines and decision windows are deleted, with the length left to the owner, plus a per-profile delete path. | Make a retention policy standard in next Phase | Owner | 2026-09-16 | Phase 4 security review |
| DR-3-02 | R-06 / Item 6 / T-3-01 | high | Prompt injection through game text steers which command the AI issues | A content-level mitigation: a second model pass reviewing the chosen command against the room text, or a denylist of high-risk verbs, or both. | I don't want to hold up this Phase with this, but I want to dig into this as a emergency security task prior to the next Phase. | Owner | 2026-09-16 | Phase 4 security review |
| DR-3-03 | R-08 / Item 8 / D-13 | medium | The badge reads On for a few seconds after an AI-failure disengage | Have the ai message that carries a failure notice also carry the new switch state, so the badge updates in the same message. | Proposed remediation: have the ai message that carries a failure notice also carry the new switch state, so the badge updates in the same message. Do this in next Phase | Owner | 2026-09-16 | Phase 4 security review |
| DR-3-04 | R-10 / Item 10 / D-03, D-13 | medium | A transient vendor 503 counts as a failed decision and disengages autopilot | One automatic retry on a 503 specifically, distinct from the other failure kinds, with a user-visible notice while it retries; the model name kept as an owner-set environment variable. | A single retry is fine, but it needs to inform the user while establishing a connection. | Owner | 2026-09-16 | Phase 4 security review |
| DR-3-05 | R-13 / Item 13 | medium | A dashboard Deploy rebuilds staging from a branch that was never pushed | Push ai-player to GitHub so dashboard deploys are safe, or document that only railway up may deploy this branch. | Please ensure that as Phases close that we're pushing to github appropriately. | Owner | 2026-09-16 | Phase 4 security review |
| DR-3-06 | R-18 / DR-2-01 carry-forward / T-3-34 | medium | Who can read the staging log has not been reviewed | Review and, if needed, restrict who can run railway logs against staging. | — | Owner | 2026-09-16 | Phase 4 security review |
| DR-3-07 | R-20 / IN-01 | low | A dead statement in EngageAutopilot | Delete the statement. | Fix this in next Phase. | Owner | 2026-09-16 | Phase 4 security review |

---

## Closures confirmed at this review

| Carried in | Closed by | Evidence |
|-----------|-----------|----------|
| DR-2-01 (sign-in code printed in the staging log; owner: must fix) | Plan 03-07: `AUTH_LOG_OTP`, off by default, gates one helper used by both send paths; the suppressed line carries no identifiers | `evidence/01-test-report.txt` `TestOTPNotLogged` (3 subtests PASS); `evidence/04-staging-ai-player.log` has no `code:` line for the fresh sign-in during the walkthrough. Residuals recorded as AR-3 (dev-mode lines, R-16) and DR-3 (log-access review, R-18). |
| DR-2-02 (six throwaway staging profiles) | Plan 03-13: used as fixtures, then deleted through the app in the owner's session; the post-review fixture deleted the same way | `evidence/14-connections-cleaned.png`; `03-canned-report.txt` RUN C header records the fixture's deletion and an empty Connections list |

---

## Security Audit Trail

| Audit Date | Threats Total (register rows) | Closed | Open-for-decision | Open (mitigation absent) | Run By |
|------------|-------------------------------|--------|--------------------|-----------------------------|--------|
| 2026-09-16 | 76 rows (45 unique T-3 ids) | 76 | 15 items listed for the owner (7 register deferrals, 5 walkthrough items, 3 plan-level acceptances to confirm) | 0 | gsd-security-auditor |
| 2026-09-16 | 21 register items (R-01 to R-21) | 14 accepted (AR-3-01 to AR-3-14) | 7 deferred (DR-3-01 to DR-3-07) | 0 | Owner, on the Phase 3 Risk Register artifact |

After the owner's decisions every item has a disposition: every plan-time mitigation verified in code, 14 risks accepted, 7 deferred to the Phase 4 review, none remediated now. `threats_open: 0`.

---

## Sign-Off

- [x] All 76 register rows have a disposition (mitigate / accept / transfer / defer to security review) and every one is CLOSED with cited evidence
- [x] The four post-review fixes (CR-01, CR-02, WR-01, WR-02) were independently re-verified in current code
- [x] Accepted risks documented in the Accepted Risks Log (AR-3-01 to AR-3-14) with the owner's notes
- [x] Deferred risks documented and carried to the Phase 4 review (DR-3-01 to DR-3-07), with the owner's instructions filed as a Phase 4 carry-forward todo
- [x] DR-2-01 and DR-2-02 closures confirmed with evidence
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-09-16 — owner decisions recorded on the Phase 3 Risk Register (https://claude.ai/artifact/HauUDyWUkqutt7LuXWse1m); no Remediate Now decision, so Phase 3 closes.
