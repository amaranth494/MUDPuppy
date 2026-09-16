---
phase: 3
slug: one-ai-decision
status: draft
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
| (planner fills) | | | | | | | | | ⬜ |

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
