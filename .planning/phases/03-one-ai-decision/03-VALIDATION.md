---
phase: 3
slug: one-ai-decision
status: planned
nyquist_compliant: true
wave_0_complete: false
created: 2026-09-15
---

# Phase 3 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Derived from `03-RESEARCH.md` §Validation Architecture. The planner fills the Per-Task map with real task IDs; the executor flips statuses.
>
> **Evidence rule (owner-directed, binding).** Every success criterion and acceptance criterion is proven by one of exactly two artifact types: a **canned report** (a repeatable script or test run whose output is captured verbatim into a file) or a **screenshot** (an image of the browser as the owner sees it, no devtools, terminal, or raw JSON in frame). A server log excerpt captured to a file counts as a canned report. **Running a database query is not evidence** and no row in this document may cite one. Every row's Evidence column names a file under `.planning/phases/03-one-ai-decision/evidence/`.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go stdlib `testing` (same as Phases 1 and 2); no mocking library anywhere in this codebase; no frontend test tooling exists (`frontend/package.json` `"test": "echo 'No tests yet' && exit 0"`) |
| **Config file** | none — `go test` needs none |
| **Quick run command** | `go test ./internal/session/... ./internal/icm/... ./internal/gemini/... ./internal/driver/... ./internal/store/... -race -v` |
| **Full suite command** | `go test ./...` (same command CI runs in `.github/workflows/ci.yml`) |
| **Frontend check** | `cd frontend && npm run build` — `tsc && vite build`, the type check and the build; the only automated frontend gate this repo has |
| **Canned report harness** | `scripts/verify-phase3.sh` (new, same shape as `scripts/verify-phase2.sh`: `set -uo pipefail`, PASS/FAIL/SKIP per ROADMAP criterion C1..C4, `--self-test` / `--self-test-negative` modes, plus a source-grep step proving no model name is a literal in Go source) — drives every REST-reachable check (missing-config refusal, decisions reload after refresh); websocket-only and live-Gemini behaviours are proven by screenshot plus staging `[AI-PLAYER]` log excerpt |
| **Estimated runtime** | ~15 seconds for `go test` with `-race`; ~40 seconds for `npm run build`; ~20 seconds for the harness against staging |

---

## Sampling Rate

- **After every task commit:** Run the quick run command for Go tasks, `cd frontend && npm run build` for frontend tasks
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd:verify-work`:** Full suite green and captured to `evidence/01-test-report.txt`; the harness self-test and negative self-test captured to `evidence/02-harness-selftest.txt`; `scripts/verify-phase3.sh` run against staging and captured to `evidence/03-canned-report.txt`; the staging `[AI-PLAYER]` decision log excerpt (request sent, answer received, command dispatched through ICM, failure and refusal lines, transcript open/close) captured to `evidence/04-staging-ai-player.log`; end-user browser screenshots for the AI Assist panel showing a live decision with reasoning and command, the `[AI-ASSIST > ...]` terminal line with the game's response, the panel reloaded after a page refresh, the D-20 refusal notice with the switch staying OFF, the D-11 unknown-option line, the D-13 failure notice landing on OFF, the Logs section in Settings, and the per-profile log page's two-pane view with human and AI lines marked apart
- **Max feedback latency:** 30 seconds for Go tasks; accepted exception: `npm run build` (~40 seconds) is the only frontend gate this repo has, same as Phases 1 and 2

---

## Phase Requirements → Test Map (from research)

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| REQ-single-decision | A plain AI-issued command in `ContextAutomation` is dispatched through `Dispatcher.Dispatch` and its `SafetyChecker` before `manager.SendCommand` is ever called (success criterion 4) | unit | `go test ./internal/icm/... -run TestDispatch_AutomationPassThrough -v` | ❌ Wave 0 |
| REQ-single-decision | The driver reads the ring-buffer window, assembles tier one + tier two, and calls Gemini with the correct request shape (`x-goog-api-key` header, `responseSchema`), decoding the double-JSON answer | unit (httptest-backed) | `go test ./internal/gemini/... -run TestGenerateContent -v` | ❌ Wave 0 |
| REQ-single-decision | Engage fires exactly one decision and issues exactly one command; a repeated `#AUTO ON` fires no second decision (D-01, D-03); a WAITING→ON resume fires one fresh decision (D-02); every D-13 failure kind sends nothing to the MUD and lands on OFF | unit (fake Gemini + fake Dispatcher) | `go test ./internal/driver/... -run TestHandleEngage -v` | ❌ Wave 0 |
| REQ-single-decision | The recent-text window is bounded, ANSI-stripped, and includes text from before engage (D-04) | unit | `go test ./internal/session/... -run TestWindow -race -v` | ❌ Wave 0 |
| REQ-single-decision | With autopilot engaged, one decision and its reasoning appear in the play screen and the game responds to the issued command | player-observable (browser, live MUD + live Gemini) + `[AI-PLAYER]` log excerpt | n/a | n/a — screenshot + staging log evidence |
| REQ-reasoning-visibility | A decision row is stored with window/reasoning/command/outcome and is returned by the decisions-reload endpoint after a page refresh (D-12) | unit (httptest) | `go test ./internal/session/... -run TestDecisionsReload -v` | ❌ Wave 0 |
| REQ-reasoning-visibility | The panel shows the reasoning live via the new AI websocket message, and the terminal shows `[AI-ASSIST > command]` in place of local echo (D-06 to D-10) | player-observable (browser) | n/a | n/a — screenshot evidence |
| REQ-reasoning-visibility | A decisions row exists for that decision and is returned after a page refresh | diagnostic (canned report against staging via the decisions REST endpoint, never a database query) | `scripts/verify-phase3.sh` decisions-reload step | ❌ Wave 0 (script) |
| REQ-reasoning-visibility | A session transcript is opened on connect and closed on disconnect for saved-profile connections only, with human, AI, and game lines marked apart and engage/disengage stint markers (D-14, D-15) | unit | `go test ./internal/session/... -run TestTranscript -race -v` | ❌ Wave 0 |
| REQ-reasoning-visibility | The Logs section opens the per-profile log page in a new tab; the page lists sessions by date/time and shows the selected transcript (D-16, D-17) | player-observable (browser) | n/a | n/a — screenshot evidence |
| REQ-env-config | With the Gemini environment variables unset, `#AUTO ON` is refused with a clear message, the server starts normally, and no command is sent (D-20) | unit | `go test ./internal/session/... -run TestAutopilotHandler_AIConfigRefusal -v` | ❌ Wave 0 |
| REQ-env-config | Same, demonstrated live on staging | player-observable + canned report | `scripts/verify-phase3.sh` refusal step | n/a — screenshot + report evidence |
| REQ-env-config | Model name and API key are read only from `internal/config` through the registry (D-18); never a literal in Go source | source assertion | `grep -rn "gemini-" internal/ --include=*.go` (expect zero matches outside `_test.go` fixtures and comments) | ❌ Wave 0 (grep step in the script) |
| DR-2-01 (folded security todo) | The staging one-time sign-in code no longer appears in the deploy log | unit + staging log excerpt | `go test ./internal/auth/... -run TestOTPNotLogged -v`; `evidence/04-staging-ai-player.log` shows no `code:` line on a fresh sign-in | ❌ Wave 0 |

---

## Per-Task Verification Map

> Filled by the planner from the PLAN.md task lists. One row per task. Every task must have an automated command or an explicit evidence file; no three consecutive tasks may lack an automated verify.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | Evidence File | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|---------------|--------|
| 03-01-01 | 03-01 | 1 | REQ-single-decision | T-3-08 | The window is a fixed-size ring that cannot grow and holds no mutex of its own | unit | `go test ./internal/session/... -run "TestWindowSnapshot\|TestWindowRingBound" -race -v` | evidence/01-test-report.txt | ⬜ |
| 03-01-02 | 03-01 | 1 | REQ-single-decision | T-3-07, T-3-08 | The window lives on Manager under m.mu, never on Session, and is never written to the log | unit | `go test ./internal/session/... -run TestWindow -race -v` | evidence/01-test-report.txt | ⬜ |
| 03-02-01 | 03-02 | 1 | REQ-env-config | T-3-02 | Registry read from env with warn-and-default, never fatal; no key detail ever logged | unit | `go test ./internal/config/... -run "TestLoadAIRegistry\|TestAIConfigured\|TestResolveModelEntry" -v` | evidence/01-test-report.txt | ⬜ |
| 03-02-02 | 03-02 | 1 | REQ-single-decision, REQ-env-config | T-3-02, T-3-16 | Key in the x-goog-api-key header only, never in a URL; the package logs nothing | unit (httptest) | `go test ./internal/gemini/... -run TestGenerateContent -race -v` | evidence/01-test-report.txt | ⬜ |
| 03-03-01 | 03-03 | 1 | REQ-single-decision | T-3-23 | An automation-context command reaches checkSafety, and is refused once its limits trip | unit | `go test ./internal/icm/... -run TestDispatch_AutomationPassThrough -race -v` | evidence/01-test-report.txt | ⬜ |
| 03-03-02 | 03-03 | 1 | REQ-single-decision | T-3-20, T-3-22 | One engine shared by the routes and the driver; routes inside the existing sessionMiddleware | source + canned report | `go build ./... && grep -c "icm.NewEngine()" cmd/server/main.go` | evidence/03-canned-report.txt | ⬜ |
| 03-03-03 | 03-03 | 1 | REQ-single-decision | T-3-21 | A failed ICM call never becomes a command sent to the game | build + screenshot | `cd frontend && npm run build` | evidence/13-hand-play-unchanged.png | ⬜ |
| 03-04-01 | 03-04 | 1 | REQ-single-decision | T-3-24 | An unknown #AUTO argument prints one local line and reaches neither the game nor the server | build + screenshot | `cd frontend && npm run build` | evidence/09-auto-unknown-option.png | ⬜ |
| 03-05-01 | 03-05 | 2 | REQ-reasoning-visibility | T-3-28 | Source and outcome vocabularies enforced by a CHECK constraint and a Go-side rejection | build | `go build ./... && go vet ./internal/store/...` | evidence/01-test-report.txt | ⬜ |
| 03-05-02 | 03-05 | 2 | REQ-reasoning-visibility | T-3-26, T-3-27, T-3-08 | Non-blocking enqueue off the MUD read path; transcript text never reaches the log | unit | `go test ./internal/session/... -run TestTranscript -race -v` | evidence/01-test-report.txt | ⬜ |
| 03-05-03 | 03-05 | 2 | REQ-reasoning-visibility | T-3-04, T-3-09 | Every transcript read resolves through GetProfileByConnection and filters on the connection id | unit (httptest) | `go test ./internal/profiles/... -run "TestListSessions\|TestGetSessionTranscript" -v` | evidence/03-canned-report.txt | ⬜ |
| 03-06-01 | 03-06 | 2 | REQ-env-config | T-3-29, T-3-30, T-3-31 | The refusal returns before EngageAutopilot; a nil config fails closed; the policy gate still refuses first | unit | `go test ./internal/session/... -run TestAutopilotHandler_AIConfigRefusal -race -v` | evidence/08-refused-not-configured.png | ⬜ |
| 03-07-01 | 03-07 | 2 | REQ-env-config | T-3-05 (DR-2-01), T-3-32, T-3-33 | The one-time sign-in code is suppressed unless AUTH_LOG_OTP is explicitly on; the issuance line carries no identifier | unit | `go test ./internal/auth/... -run TestOTPNotLogged -v` | evidence/04-staging-ai-player.log | ⬜ |
| 03-08-01 | 03-08 | 3 | REQ-single-decision | T-3-15 | Outcome validated before the insert; a decision row is never updated or deleted | build | `go build ./... && go vet ./internal/store/...` | evidence/01-test-report.txt | ⬜ |
| 03-08-02 | 03-08 | 3 | REQ-single-decision | T-3-01, T-3-06, T-3-07 | The command is validated in Go before Dispatch; the send happens only after approval; log lines carry lengths only | unit | `go test ./internal/driver/... -run TestHandleEngage -race -v` | evidence/01-test-report.txt | ⬜ |
| 03-08-03 | 03-08 | 3 | REQ-single-decision | T-3-35, T-3-14 | One trigger per path, fired only on a real state change, always asynchronously | unit | `go test ./internal/session/... -race -v` | evidence/04-staging-ai-player.log | ⬜ |
| 03-09-01 | 03-09 | 4 | REQ-reasoning-visibility | T-3-11, T-3-36, T-3-08 | The client registry is keyed by the authenticated user id; an absent screen is a silent no-op | unit | `go test ./internal/session/... -run TestPushAI -race -v` | evidence/01-test-report.txt | ⬜ |
| 03-09-02 | 03-09 | 4 | REQ-reasoning-visibility | T-3-36 | The row is stored before the push and the push error is discarded, so a closed tab loses nothing | unit | `go test ./internal/driver/... -race -v` | evidence/05-decision-in-panel.png | ⬜ |
| 03-09-03 | 03-09 | 4 | REQ-reasoning-visibility | T-3-03, T-3-15, T-3-07 | Ownership resolved before any row is read; window_text is never serialised to the browser | unit (httptest) | `go test ./internal/profiles/... -run TestDecisionsReload -v` | evidence/03-canned-report.txt | ⬜ |
| 03-10-01 | 03-10 | 5 | REQ-reasoning-visibility | T-3-11 | The decisions fetch is per connection id with credentials and the shared auth-error path | build | `cd frontend && npm run build` | evidence/01-test-report.txt | ⬜ |
| 03-10-02 | 03-10 | 5 | REQ-reasoning-visibility | T-3-37, T-3-11 | Model-authored text is rendered as escaped JSX children; no innerHTML anywhere | build + screenshot | `cd frontend && npm run build` | evidence/05-decision-in-panel.png | ⬜ |
| 03-10-03 | 03-10 | 5 | REQ-reasoning-visibility | T-3-38, T-3-39 | One reserved colour and label for AI-issued commands; the panel mounts only where the AI is activated | build + screenshot | `cd frontend && npm run build` | evidence/06-ai-assist-terminal-line.png | ⬜ |
| 03-11-01 | 03-11 | 6 | REQ-reasoning-visibility | T-3-41 | The log route sits inside AuthGuard and outside the play-screen shell | build | `cd frontend && npm run build` | evidence/12-log-page-two-pane.png | ⬜ |
| 03-11-02 | 03-11 | 6 | REQ-reasoning-visibility | T-3-37, T-3-04 | Transcript text is escaped by React; the connection id comes only from the route | build + screenshot | `cd frontend && npm run build` | evidence/12-log-page-two-pane.png | ⬜ |
| 03-11-03 | 03-11 | 6 | REQ-reasoning-visibility | T-3-40 | The new tab is opened by an anchor carrying rel="noopener noreferrer" | build + screenshot | `cd frontend && npm run build` | evidence/11-logs-section.png | ⬜ |
| 03-12-01 | 03-12 | 5 | REQ-env-config, REQ-reasoning-visibility | T-3-43, T-3-17, T-3-42 | No SQL client and no key anywhere in the harness; the header records BASE_URL and the git SHA | script | `bash -n scripts/verify-phase3.sh` | evidence/03-canned-report.txt | ⬜ |
| 03-12-02 | 03-12 | 5 | REQ-env-config | T-3-10, T-3-42 | The negative fixture set makes the harness print FAIL C3 and exit non-zero | script | `bash scripts/verify-phase3.sh --self-test /tmp/phase3-selftest.txt` | evidence/02-harness-selftest.txt | ⬜ |
| 03-13-01 | 03-13 | 7 | REQ-single-decision, REQ-reasoning-visibility, REQ-env-config | T-3-SC, T-3-10 | The dependency-drift and model-literal sections are both empty; the harness has proven it can fail | canned report | `bash scripts/verify-phase3.sh --self-test /tmp/phase3-selftest.txt && grep -c "GO TEST EXIT" .planning/phases/03-one-ai-decision/evidence/01-test-report.txt` | evidence/01-test-report.txt | ⬜ |
| 03-13-02 | 03-13 | 7 | REQ-env-config | T-3-02, T-3-17 | The key is set on staging only, by the owner, and its value is never recorded anywhere | checkpoint (canned report) | `BASE_URL=... SESSION_COOKIE=... CONNECTION_ID=... scripts/verify-phase3.sh /tmp/phase3-run-a.txt` | evidence/03-canned-report.txt | ⬜ |
| 03-13-03 | 03-13 | 7 | REQ-single-decision, REQ-reasoning-visibility | T-3-07, T-3-05, T-3-44 | The log excerpt is grepped for prose, keys and sign-in codes before filing; the six leftover profiles are deleted through the app | checkpoint (screenshots + log) | `grep -ciE "conduct_rules\|approach_guidance\|AIza\|code:" .planning/phases/03-one-ai-decision/evidence/04-staging-ai-player.log` | evidence/04-staging-ai-player.log | ⬜ |
| 03-13-04 | 03-13 | 7 | REQ-single-decision, REQ-reasoning-visibility, REQ-env-config | T-3-01, T-3-04, T-3-06, T-3-09, T-3-14, T-3-15, T-3-19 | Every deferred item is raised with three dispositions and none pre-chosen | doc | `test -f .planning/phases/03-one-ai-decision/03-SECURITY-AGENDA.md && grep -c "Remediate Now" .planning/phases/03-one-ai-decision/03-SECURITY-AGENDA.md` | 03-SECURITY-AGENDA.md | ⬜ |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/gemini/client.go` + `internal/gemini/client_test.go` — new package; `httptest.Server`-backed tests for the request shape, header auth, the double-JSON-decode, and every error status code (400/401/403/429) — REQ-single-decision, REQ-env-config
- [ ] `internal/driver/driver.go` + `internal/driver/driver_test.go` — new package; fake-Gemini + fake-Dispatcher table tests proving exactly-one-decision-per-engage (D-01, D-02, D-03) and every D-13 failure kind sends nothing to the MUD — REQ-single-decision
- [ ] `internal/icm/dispatcher_test.go` addition — `TestDispatch_AutomationPassThrough_GoesThroughSafetyChecker`, proving a plain command in `ContextAutomation` reaches `Dispatch` and the `SafetyChecker` — REQ-single-decision (success criterion 4)
- [ ] `internal/session/window.go` + `internal/session/window_test.go` — ring buffer bound, ANSI strip, and the "text from before engage is included" property (D-04) — REQ-single-decision
- [ ] `internal/session/transcript_test.go` — open-on-connect/close-on-disconnect, human/AI/game line tagging, stint markers, quick-connect exclusion (D-14, D-15) — REQ-reasoning-visibility
- [ ] `internal/session/handler_test.go` addition — `TestAutopilotHandler_AIConfigRefusal`, proving D-20's refusal on the engage path — REQ-env-config
- [ ] `internal/session/...` decisions reload test — `TestDecisionsReload`, proving the current connection's decisions are returned on attach (D-12) — REQ-reasoning-visibility
- [ ] `internal/auth/...` test — `TestOTPNotLogged`, proving the sign-in code is gated or hashed (DR-2-01)
- [ ] `scripts/verify-phase3.sh` (new, following `scripts/verify-phase2.sh`'s exact shape) — REQ-env-config, REQ-reasoning-visibility
- [ ] Framework install: none — `testing` is stdlib, already in use

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| One live decision with reasoning appears in the AI Assist panel and the game responds to the `[AI-ASSIST > ...]` command | REQ-single-decision, REQ-reasoning-visibility | Needs a live MUD and live Gemini; websocket push is not REST-reachable | On staging with an AI-activated profile connected: type `#AUTO ON`; screenshot the panel with the reasoning and command, and the terminal line with the game's reply; capture the `[AI-PLAYER]` log lines |
| The panel reloads the decision after a page refresh | REQ-reasoning-visibility | Browser behaviour | Refresh the play screen; screenshot the panel showing the same decision |
| `#AUTO ON` refused when AI is not configured | REQ-env-config | Needs the staging env variables unset for the capture | With the Gemini variables unset on staging: type `#AUTO ON`; screenshot the refusal notice with the badge still OFF |
| `#AUTO FOO` prints the D-11 unknown-option line | REQ-single-decision (grammar) | Browser-local line | Type `#AUTO STATUS`; screenshot the local line |
| Logs section and per-profile log page in a new tab | REQ-reasoning-visibility (session log) | Browser navigation | In Settings pick Logs, press Open logging; screenshot the new tab's two-pane page with a session selected showing human and AI lines marked apart |
| Hand play unchanged for a profile without acceptance | Phase 2 regression | Browser behaviour | Connect a profile without policy acceptance; screenshot the play screen with no panel |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s (accepted exception: `npm run build`)
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
