---
phase: 2
slug: autopilot-switch
status: draft
nyquist_compliant: true
wave_0_complete: false
created: 2026-09-15
---

# Phase 2 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Derived from `02-RESEARCH.md` §Validation Architecture. The planner fills the Per-Task map with real task IDs; the executor flips statuses.
>
> **Evidence rule (owner-directed, binding).** Every success criterion and acceptance criterion is proven by one of exactly two artifact types: a **canned report** (a repeatable script or test run whose output is captured verbatim into a file) or a **screenshot** (an image of the browser as the owner sees it, no devtools, terminal, or raw JSON in frame). A server log excerpt captured to a file counts as a canned report. **Running a database query is not evidence** and no row in this document may cite one. Every row's Evidence column names a file under `.planning/phases/02-autopilot-switch/evidence/`.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go stdlib `testing` (same as Phase 1: `internal/store/profile_test.go`, `internal/policy/policy_test.go`, `internal/profiles/handler_test.go`); no frontend test tooling exists (`frontend/package.json` `"test": "echo 'No tests yet' && exit 0"`) |
| **Config file** | none — `go test` needs none |
| **Quick run command** | `go test ./internal/session/... -race -v` |
| **Full suite command** | `go test ./...` (same command CI runs in `.github/workflows/ci.yml`) |
| **Frontend check** | `cd frontend && npm run build` — `package.json`'s build script is `tsc && vite build`, so this is the type check as well as the build; it is the only automated frontend gate this repo has |
| **Canned report harness** | `scripts/verify-phase2.sh` (new, same shape as `scripts/verify-phase1.sh`: `set -uo pipefail`, PASS/FAIL/SKIP per ROADMAP criterion C1..C4, `--self-test` / `--self-test-negative` modes) — drives every REST-reachable check; websocket-only behaviours are proven by screenshot plus staging log excerpt |
| **Estimated runtime** | ~10 seconds for `go test`; ~40 seconds for `npm run build`; ~15 seconds for the harness against staging |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/session/... -race -v` for Go tasks, `cd frontend && npm run build` for frontend tasks
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd:verify-work`:** Full suite green and captured to `evidence/01-test-report.txt`; the harness self-test and negative self-test captured to `evidence/02-harness-selftest.txt`; `scripts/verify-phase2.sh` run against staging and captured to `evidence/03-canned-report.txt`; the staging `[AI-PLAYER]` autopilot transition log excerpt captured to `evidence/04-staging-ai-player.log`; end-user browser screenshots for the badge after refresh, the gate refusal before acceptance, the wheel-grab notice with the game's response, the trigger-fired command leaving autopilot ON, the WAITING badge after a drop, the resume-to-ON after a hand reconnect, `#AUTO OFF` while WAITING staying OFF after reconnect, and ordinary play unchanged with the AI off
- **Max feedback latency:** 30 seconds for Go tasks; accepted exception: `npm run build` (~40 seconds) is the only frontend gate this repo has, same as Phase 1

---

## Phase Requirements → Test Map (from research)

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| REQ-autopilot-directives | `#AUTO ON` engages only when the gate passes; `#AUTO OFF` disengages; already-on/already-off are no-ops with the D-04 wording; `#AUTO` / `#AUTO STATUS` prints state and gate result | unit | `go test ./internal/session/... -run TestEngageAutopilot -v` | ❌ Wave 0 |
| REQ-autopilot-directives | Indicator matches server state after a page refresh | player-observable (screenshot) | n/a | n/a |
| REQ-wheel-grab | Human-sourced `data` message disengages ON before the command is queued; `trigger`-sourced does not; absent/malformed source defaults to human | unit | `go test ./internal/session/... -run TestWheelGrabSourceRule -v` | ❌ Wave 0 |
| REQ-wheel-grab | Typing while engaged shows the disengage notice and the game's response; a trigger-fired command leaves autopilot ON | player-observable (screenshot) + log | n/a | n/a |
| REQ-no-auto-reconnect | `Disconnect` while ON transitions to WAITING and survives the `sessions` map deletion; `Connect` while WAITING resumes to ON; `#AUTO OFF` while WAITING lands on OFF and stays OFF after `Connect` | unit (end-to-end through `Manager`) | `go test ./internal/session/... -run TestWaitingSurvivesDisconnectAndResumesOnConnect -race -v` | ❌ Wave 0 |
| REQ-no-auto-reconnect | Drop shows WAITING and no command is sent while disconnected; hand reconnect shows resume to ON | player-observable (screenshot) + `[AI-PLAYER]` log excerpt | n/a | n/a |
| REQ-doc-hand-play-and-gate | Ordinary play with autopilot OFF is unchanged (D-12); engage refused without acceptance (Phase 1 `EngageGateAllowed`) | unit (regression) | `go test ./internal/store/... -run TestEngageGate -v` | ✅ exists (Phase 1) |
| REQ-doc-hand-play-and-gate | Owner creates a profile, accepts policy, hand-plays normally with the AI disengaged | player-observable (screenshot) | n/a | n/a |

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | Evidence File | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|---------------|--------|
| 02-01-01 | 02-01 | 1 | REQ-autopilot-directives, REQ-no-auto-reconnect | T-2-05, T-2-08 | A repeated Engage changes nothing; `WaitingSince` is recorded so a bounded waiting lifetime can be added later without redesign | unit | `go test ./internal/session/... -run TestAutopilotTransitions -race -v` | `evidence/01-test-report.txt` | ⬜ |
| 02-01-02 | 02-01 | 1 | REQ-no-auto-reconnect | T-2-04, T-2-09 | All autopilot access goes through methods that take `m.mu`; `EngageAutopilot` reads `m.sessions` inline rather than calling the RLock-taking `GetSession`; log lines carry ids, states and a cause only | build + source assertion | `go build ./... && go vet ./internal/session/...` | `evidence/01-test-report.txt` | ⬜ |
| 02-01-03 | 02-01 | 1 | REQ-no-auto-reconnect | T-2-05, T-2-09 | WAITING outlives the deleted session map entry; `#AUTO OFF` while waiting sticks across a reconnect; the package runs clean under `-race` | unit (end-to-end through `Manager`) | `go test ./internal/session/... -run TestWaitingSurvivesDisconnectAndResumesOnConnect -race -v` | `evidence/01-test-report.txt` | ⬜ |
| 02-01-03b | 02-01 | 1 | REQ-autopilot-directives | T-2-05 | Engage is refused without a connected session; a repeated engage leaves the stored record untouched | unit | `go test ./internal/session/... -run TestEngageAutopilot -race -v` | `evidence/01-test-report.txt` | ⬜ |
| 02-02-01 | 02-02 | 2 | REQ-autopilot-directives, REQ-doc-hand-play-and-gate | T-2-02, T-2-03, T-2-04, T-2-10, T-2-11 | User identity comes only from the request context; the action is allowlisted; a nil gate callback fails closed; the refusal sentence is passed through from Phase 1, never restated | build + source assertion | `go build ./... && go vet ./internal/session/...` | `evidence/01-test-report.txt` | ⬜ |
| 02-02-02 | 02-02 | 2 | REQ-doc-hand-play-and-gate | T-2-02 | The gate closure resolves the profile with `GetProfileByConnection(userID, connectionID)`, scoped on both columns | build + source assertion | `go build ./... && go vet ./cmd/...` | `evidence/01-test-report.txt` | ⬜ |
| 02-02-03 | 02-02 | 2 | REQ-autopilot-directives, REQ-doc-hand-play-and-gate | T-2-02, T-2-10, T-2-11 | Gate refusal, no-connection refusal, the two no-ops, invalid action, nil callback, and a body that tries to name another user are each asserted | unit (httptest) | `go test ./internal/session/... -run TestAutopilotHandler -race -v` | `evidence/01-test-report.txt` | ⬜ |
| 02-02-03b | 02-02 | 2 | REQ-autopilot-directives | T-2-03 | The status response carries `autopilot_state` on every response, including when off, so the badge is correct after a refresh | unit (httptest) | `go test ./internal/session/... -run TestStatusCarriesAutopilotState -race -v` | `evidence/01-test-report.txt` | ⬜ |
| 02-03-01 | 02-03 | 2 | REQ-wheel-grab | T-2-01, T-2-02, T-2-04, T-2-12 | Only `trigger` and `timer` count as automation; the grab can only move on to off for its own user; the command is queued on every path | build + source assertion | `go build ./... && go vet ./internal/session/...` | `evidence/01-test-report.txt` | ⬜ |
| 02-03-02 | 02-03 | 2 | REQ-wheel-grab | T-2-01, T-2-02 | Absent and garbled source flags are human; a human command never engages anything; one user's typing cannot move another user's switch | unit | `go test ./internal/session/... -run TestWheelGrabSourceRule -race -v` | `evidence/01-test-report.txt` | ⬜ |
| 02-04-01 | 02-04 | 3 | REQ-autopilot-directives | T-2-13, T-2-14 | `#AUTO` emits no game command; commands a typed directive emits are labelled automation; the refusal sentence is rendered from the server response and held nowhere in the frontend | type check + build | `cd frontend && npm run build` | `evidence/01-test-report.txt`, `evidence/06-refused-before-acceptance.png` | ⬜ |
| 02-04-02 | 02-04 | 3 | REQ-wheel-grab | T-2-01, T-2-03 | The source the engine already assigned reaches the data message; an omitted source is read as human server-side | type check + build | `cd frontend && npm run build` | `evidence/01-test-report.txt`, `evidence/07-wheel-grab.png` | ⬜ |
| 02-05-01 | 02-05 | 3 | REQ-autopilot-directives, REQ-doc-hand-play-and-gate | T-2-02, T-2-15, T-2-16, T-2-17 | The session cookie is never echoed; the not-owned-connection request is asserted refused; C2 and C3 are SKIP lines naming their screenshots, never PASS; no database query anywhere in the harness | harness syntax and permission check | `bash -n scripts/verify-phase2.sh && test -x scripts/verify-phase2.sh` | `evidence/02-harness-selftest.txt` | ⬜ |
| 02-05-02 | 02-05 | 3 | REQ-autopilot-directives, REQ-no-auto-reconnect | T-2-07 | The negative self-test proves the harness reports `FAIL C1` and exits non-zero on a wrong response, so a clean report carries information | harness self-test | `bash scripts/verify-phase2.sh --self-test /tmp/phase2-selftest.txt` | `evidence/02-harness-selftest.txt` | ⬜ |
| 02-06-01 | 02-06 | 4 | REQ-autopilot-directives | T-2-06 | The badge value comes only from the status response and the server push; no owner-visible copy lives in the session context | type check + build | `cd frontend && npm run build` | `evidence/05-badge-after-refresh.png` | ⬜ |
| 02-06-02 | 02-06 | 4 | REQ-autopilot-directives | T-2-06 | The badge derives nothing: it reads no `connectionState`, holds no local state and runs no poll of its own | type check + build + source assertion | `cd frontend && npm run build` | `evidence/05-badge-after-refresh.png`, `evidence/09-waiting-after-drop.png` | ⬜ |
| 02-06-03 | 02-06 | 4 | REQ-wheel-grab, REQ-no-auto-reconnect, REQ-doc-hand-play-and-gate | T-2-01, T-2-18, T-2-19 | The blank-Enter path is labelled human explicitly; one printing owner per transition; every notice goes through `echoLocal` so it can never reach the game; nothing prints while the switch is off | type check + build | `cd frontend && npm run build` | `evidence/07-wheel-grab.png`, `evidence/09-waiting-after-drop.png`, `evidence/10-resume-after-reconnect.png`, `evidence/12-hand-play-unchanged.png` | ⬜ |
| 02-07-01 | 02-07 | 5 | all four | T-2-07, T-2-SC | The harness is proven falsifiable before it is trusted; the dependency-drift section must be empty; no SQL appears in either report | canned report | `bash scripts/verify-phase2.sh --self-test /tmp/phase2-selftest.txt` | `evidence/01-test-report.txt`, `evidence/02-harness-selftest.txt` | ⬜ |
| 02-07-02 | 02-07 | 5 | all four | T-2-04, T-2-17 | Staging only, attributable by `BASE_URL` and git SHA in the report header; the log excerpt is grepped for game text and profile text before it is filed | canned report + log excerpt (blocking checkpoint) | live run: `BASE_URL=… SESSION_COOKIE=… CONNECTION_ID=… scripts/verify-phase2.sh .planning/phases/02-autopilot-switch/evidence/03-canned-report.txt` | `evidence/03-canned-report.txt`, `evidence/04-staging-ai-player.log` | ⬜ |
| 02-07-03 | 02-07 | 5 | all four | T-2-01, T-2-06 | Every shot is the end-user browser view with no devtools, terminal or raw JSON; the refresh shot is taken with no action after the reload; the staging session is never signed out | player-observable (blocking checkpoint) | n/a — human-check | `evidence/05-badge-after-refresh.png` through `evidence/12-hand-play-unchanged.png`, `02-07-SUMMARY.md` | ⬜ |
| 02-07-04 | 02-07 | 5 | all four | T-2-08, T-2-20, T-2-SC | The agenda lists three dispositions per item with none chosen and builds no remediation | file assertion | `test -f .planning/phases/02-autopilot-switch/02-SECURITY-AGENDA.md` | `02-SECURITY-AGENDA.md` | ⬜ |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

*Rows suffixed `b` are a second named automated command belonging to the same task; both must be green for that task to count as verified.*

---

## Wave 0 Requirements

- [ ] `internal/session/autopilot.go` + `internal/session/autopilot_test.go` — pure autopilot state type and transitions (`Engage`, `Disengage`, `EnterWaiting`, `Resume`), table tests proving ON/OFF/WAITING per D-01/D-04 — REQ-autopilot-directives, REQ-no-auto-reconnect (plan 02-01, task 02-01-01)
- [ ] `internal/session/manager_test.go` — `Manager.Disconnect` → `Manager.Connect` end-to-end test proving WAITING survives the `sessions` map deletion — REQ-no-auto-reconnect (plan 02-01, task 02-01-03)
- [ ] `internal/session/handler_test.go` — httptest coverage of the autopilot endpoint's refusals, no-ops and authenticated-user-only rule — REQ-autopilot-directives, REQ-doc-hand-play-and-gate (plan 02-02, task 02-02-03)
- [ ] `internal/session/websocket_test.go` — the human/automation source-flag classification and the wheel-grab decision against a real `Manager` — REQ-wheel-grab (plan 02-03, task 02-03-02)
- [ ] `scripts/verify-phase2.sh` plus `scripts/fixtures/phase2/` and `scripts/fixtures/phase2-negative/` — the canned-report harness and its self-test — REQ-autopilot-directives, REQ-no-auto-reconnect (plan 02-05)
- [ ] Framework install: none — `testing` is stdlib and already in use; the harness uses `curl` plus optional `jq` behind a `command -v` guard

---

## Manual-Only Verifications

Each row names the evidence file it produces. All files live under `.planning/phases/02-autopilot-switch/evidence/`. All eight screenshots are captured by plan 02-07, task 02-07-03.

| Behavior | Requirement | Why Manual | Test Instructions | Evidence File |
|----------|-------------|------------|-------------------|---------------|
| Badge reads ON after `#AUTO ON`, and still reads ON after a hard page refresh with no action taken | REQ-autopilot-directives | No frontend test tooling; visual state | Engage on an accepted profile, hard-refresh, screenshot the play screen with the header badge visible | `evidence/05-badge-after-refresh.png` |
| `#AUTO ON` on an un-accepted profile prints the gate refusal and the badge stays OFF | REQ-autopilot-directives, REQ-doc-hand-play-and-gate | End-user notice wording | Type `#AUTO ON` on a fresh profile; screenshot the terminal line and badge | `evidence/06-refused-before-acceptance.png` |
| Typing a game command while ON prints `[Autopilot disengaged: you took the wheel]` and the game's response follows; a blank Enter does the same | REQ-wheel-grab | Live MUD response is player-observable | While ON, type `look`; screenshot showing the notice, the command, and the game's reply; then engage again and press Enter on an empty line | `evidence/07-wheel-grab.png` |
| A trigger-fired command is sent while the badge stays ON | REQ-wheel-grab | Trigger engine is browser-side only | Create a trigger that sends a command on a known prompt line; while ON, let it fire; screenshot badge still ON with the trigger's command visible | `evidence/08-trigger-keeps-on.png` |
| Dropping the connection shows `[Disconnected]` then `[Autopilot waiting for reconnect]` and the badge reads WAITING | REQ-no-auto-reconnect | Connection lifecycle | While ON, disconnect; screenshot the two lines and the WAITING badge | `evidence/09-waiting-after-drop.png` |
| Reconnecting by hand shows `[Reconnected]` then `[Autopilot resuming]` and the badge reads ON without typing `#AUTO ON` | REQ-no-auto-reconnect | Connection lifecycle | Press Connect; screenshot the two lines and the ON badge | `evidence/10-resume-after-reconnect.png` |
| `#AUTO OFF` while WAITING lands on OFF and the badge stays OFF after a hand reconnect | REQ-no-auto-reconnect | Connection lifecycle | While WAITING, type `#AUTO OFF`, then Connect; screenshot the OFF badge after the connection returns | `evidence/11-off-while-waiting-stays-off.png` |
| Ordinary play with the AI OFF looks unchanged: no new lines, badge reads OFF | REQ-doc-hand-play-and-gate | Absence of behaviour is visual | Create a profile, accept the policy, connect, play a few commands; screenshot | `evidence/12-hand-play-unchanged.png` |
| Every autopilot transition emits one `[AI-PLAYER]` line with user/session id, old state, new state, and cause | all four | Captured from the platform log viewer, never from the database | After the walkthrough, capture the `[AI-PLAYER] autopilot` lines with `railway logs --environment staging` | `evidence/04-staging-ai-player.log` |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies (plan-checker pass, 2026-09-15)
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 30s for Go tasks (`npm run build` exception noted above)
- [x] Every Per-Task and Manual-Only row cites an evidence file, and no row cites a database query
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
