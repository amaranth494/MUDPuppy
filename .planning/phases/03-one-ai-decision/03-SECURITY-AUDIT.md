---
phase: 03-one-ai-decision
document: security-audit
audited: 2026-09-16
auditor: Claude (gsd-security-auditor)
asvs_level: 1
block_on: critical
status: SECURED
threats_total_rows: 76
threats_closed_rows: 76
threats_open_rows: 0
unregistered_flags: 0
---

# Phase 3: Security Audit Report

**Method:** Every `<threat_model>` block in `03-01-PLAN.md` through `03-13-PLAN.md` was read and every threat row extracted (register authored at plan time — an id appearing in several plans was verified separately in each plan's context). For `mitigate` rows the cited file/test was read and, where a test was cited, re-run with `go test ... -race -count=1` to confirm it exists, exercises the claim, and passes (not merely that a matching name exists). For `accept` rows the acceptance is checked against the actual repository state (e.g. `git diff --stat` for `go.mod`/`package.json`). For `transfer` rows the transfer documentation (a doc comment naming the receiving package/plan) was located and the receiving plan's own mitigation was independently verified. For `defer` rows, `03-SECURITY-AGENDA.md` was checked for a matching, undecided agenda item carrying the original wording — that is what "documented" means for a deferred disposition; the owner's Accept/Defer/Remediate choice itself is outside this audit's scope and is listed below for the owner. All four `03-REVIEW.md` findings (2 critical, 2 warning) were independently confirmed fixed in the current tree, not just present as commits.

No implementation file was modified. This file and nothing else was written.

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

## Code Review Fix Verification (03-REVIEW.md)

All four findings independently confirmed fixed in the current tree (not merely present as commits — checked `git merge-base --is-ancestor` against HEAD, then read the resulting code):

| Finding | Commit | Verified in code |
|---------|--------|-------------------|
| CR-01 (DB I/O under manager-wide lock) | `cbf7d97` | `internal/session/manager.go:277-286` (`Connect`) and `:402-414` (`Disconnect`) release `m.mu` around `openTranscript`/`closeTranscript`'s blocking I/O |
| CR-02 (send-on-closed-channel panic) | `0b74ea7` | `internal/session/transcript.go:66,84,92,169,202` — `close()` closes a `stop` channel, never `t.lines` |
| WR-01 (React key collision) | `2a2f349` | `frontend/src/components/AIAssistPanel.tsx:155` `const key = entry.id \|\| \`no-id-${index}\`` |
| WR-02 (internal error text leaked) | `48165ab` | `internal/profiles/handler.go:223-227` — logs `err`, returns fixed `"Failed to get profile"` |

`go test ./internal/... -race -count=1` — all packages PASS (auth, config, driver, gemini, icm, policy, profiles, session, store).

## Unregistered Flags

None. Every `## Threat Flags` section across `03-01` through `03-13-SUMMARY.md` maps its plan's new attack surface to an existing threat id already in the register above (verified: `03-03` → T-3-20/21/22/SC; `03-08` → T-3-01/02/06/07/35/14/15/SC; `03-09` → T-3-03/07/08/11/15/36/SC; `03-10` → T-3-37/38/11/39/SC; `03-11` → T-3-04/37/40/41/SC; `03-12` → T-3-10/42/17/43/SC). Independent cross-check of `git diff 6e98595 HEAD -- cmd/server/main.go` for new route registrations found exactly the three routes covered by T-3-03/T-3-04, plus the ICM routes covered by T-3-22 — no unmapped route.

## Residual Note: T-3-05 / DEV MODE OTP lines (not a registered threat gap, flagged for completeness)

Independent code reading found two log lines (`internal/auth/handler.go:204,293`, `log.Printf("DEV MODE - OTP for %s: %s", email, otp)`) that print the sign-in code **unconditionally**, bypassing the `AUTH_LOG_OTP` gate entirely, whenever `h.emailSender` is nil or `IsConfigured()` is false. This is outside 03-07-01's declared mitigation scope (which named only the two `"STAGING: ..."` call sites) and was independently found and disclosed by the executor in `03-07-SUMMARY.md`'s repository-wide grep confirmation, then carried to `03-SECURITY-AGENDA.md` as **Item 16**. `IsConfigured()` requires `host`/`username`/`password`/`fromAddress` all non-empty, which staging and production both require for OTP email delivery to function at all — so this branch is provably unreachable in either environment as currently configured, but it is not code-level-closed. This does not reopen T-3-05 (the two call sites the plan actually named are gated and verified), but it is a live, undecided item on the agenda and belongs in the owner's review alongside T-3-05.

## Items Needing an Owner Decision (Accept / Defer / Remediate Now)

Per `03-SECURITY-AGENDA.md`, none of the following are decided yet — every checkbox is unchecked. These are the deferred-disposition threats plus the residual/process items the executor surfaced during the walkthrough that bear directly on this register:

**Deferred threats from the register (Part A):**
1. T-3-09 / T-3-15 — Item 1: transcript + `window_text` retention against policy §6 (no retention limit, no deletion path, no export)
2. T-3-14 — Item 2: a WAITING→ON resume now fires a real command, not just a badge flip
3. T-3-04 (philosophical boundary question, not a code gap) — Item 3: is per-profile session scoping the right boundary for stored game text at all
4. T-3-19 (re-confirmation) — Item 4: Gemini key/free-tier posture on staging
5. T-3-06 — Item 5: the driver's send path still bypasses the human wheel-grab entirely (Phase 2's T-2-01 re-confirmed)
6. T-3-01 (residual) — Item 6: shape-only validation doesn't stop content-level prompt injection (which legitimate command gets chosen)
7. T-3-34 — DR-2-01 closure section: Railway staging log access review, never done

**Residual/process items raised live during the walkthrough (Part B), relevant to this audit:**
8. Item 16 — the DEV MODE OTP lines (see Residual Note above)
9. Item 12 — a production env var (`GOOGLE_API_KEY`) was set and triggered an unplanned production redeploy (operator error, not a code defect; inert on the currently-running production build, but touches the "production untouched until acceptance" rule and the T-3-17 trust boundary)
10. Item 13 — Railway dashboard deploys would crash staging (undocumented deploy-path divergence)
11. Item 14 — staging session tokens were pasted into the chat transcript during the walkthrough
12. Item 11 — the new `model-error` diagnostic log line (added live, outside any plan's file list) — confirmed compliant with T-3-07 by this audit, but never formally reviewed against that rule until now

**Accepted-at-plan-level threats the review should confirm still hold (not re-litigate):**
13. T-3-SC — confirmed by this audit: no package installed anywhere in Phase 3
14. T-3-19 — confirmed by this audit: no retry loop, `KindRateLimited` surfaced correctly
15. T-3-25 — confirmed by this audit: `#AUTO STATUS` no longer dispatches

None of items 1–15 are BLOCKER findings under this audit's adversarial-verification standard — each declared `mitigate` disposition has a concrete, independently-verified control in code and/or a passing test, and each `defer`/`accept` disposition is properly documented on `03-SECURITY-AGENDA.md` awaiting the owner's Accept/Defer/Remediate Now choice per the project's binding risk-decision process. `block_on: critical` — no critical (BLOCKER) finding exists.
