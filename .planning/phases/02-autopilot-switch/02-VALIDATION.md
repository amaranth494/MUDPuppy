---
phase: 2
slug: autopilot-switch
status: draft
nyquist_compliant: false
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
| **Canned report harness** | `scripts/verify-phase2.sh` (new, same shape as `scripts/verify-phase1.sh`: `set -uo pipefail`, PASS/FAIL per ROADMAP criterion C1..C4, `--self-test` / `--self-test-negative` modes) — drives every REST-reachable check; websocket-only behaviours are proven by screenshot plus staging log excerpt |
| **Estimated runtime** | ~10 seconds for `go test`; ~15 seconds for the harness against staging |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/session/... -race -v`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd:verify-work`:** Full suite green and captured to `evidence/01-test-report.txt`; `scripts/verify-phase2.sh` run against staging and captured to `evidence/03-canned-report.txt`; the staging `[AI-PLAYER]` autopilot transition log excerpt captured to `evidence/04-staging-ai-player.log`; end-user browser screenshots for the badge after refresh, the wheel-grab notice with the game's response, the trigger-fired command leaving autopilot ON, the WAITING badge after a drop, the resume-to-ON after a hand reconnect, and `#AUTO OFF` while WAITING staying OFF after reconnect
- **Max feedback latency:** 30 seconds

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
| _(planner fills this table with real task IDs from the PLAN.md files; one row per task that carries an automated verify or a named evidence file)_ | | | | | | | | | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/session/autopilot.go` + `internal/session/autopilot_test.go` — pure autopilot state type and transitions (`Engage`, `Disengage`, `EnterWaiting`, `Resume`), table tests proving ON/OFF/WAITING per D-01/D-04 — REQ-autopilot-directives, REQ-no-auto-reconnect
- [ ] `internal/session/manager_test.go` — `Manager.Disconnect` → `Manager.Connect` end-to-end test proving WAITING survives the `sessions` map deletion — REQ-no-auto-reconnect
- [ ] `internal/session/websocket_test.go` — the human/automation source-flag classification as a pure function test — REQ-wheel-grab
- [ ] `scripts/verify-phase2.sh` plus `scripts/fixtures/phase2/` and `scripts/fixtures/phase2-negative/` — the canned-report harness and its self-test — REQ-autopilot-directives, REQ-no-auto-reconnect
- [ ] Framework install: none — `testing` is stdlib and already in use; the harness uses `curl` plus optional `jq` behind a `command -v` guard

---

## Manual-Only Verifications

Each row names the evidence file it produces. All files live under `.planning/phases/02-autopilot-switch/evidence/`. The planner assigns final file names; the ones below are the expected set.

| Behavior | Requirement | Why Manual | Test Instructions | Evidence File |
|----------|-------------|------------|-------------------|---------------|
| Badge reads ON after `#AUTO ON`, and still reads ON after a hard page refresh with no action taken | REQ-autopilot-directives | No frontend test tooling; visual state | Engage on an accepted profile, hard-refresh, screenshot the play screen with the header badge visible | `evidence/05-badge-after-refresh.png` |
| `#AUTO ON` on an un-accepted profile prints the gate refusal and the badge stays OFF | REQ-autopilot-directives, REQ-doc-hand-play-and-gate | End-user notice wording | Type `#AUTO ON` on a fresh profile; screenshot the terminal line and badge | `evidence/06-refused-before-acceptance.png` |
| Typing a game command while ON prints `[Autopilot disengaged: you took the wheel]` and the game's response follows | REQ-wheel-grab | Live MUD response is player-observable | While ON, type `look`; screenshot showing the notice, the command, and the game's reply | `evidence/07-wheel-grab.png` |
| A trigger-fired command is sent while the badge stays ON | REQ-wheel-grab | Trigger engine is browser-side only | Create a trigger that sends a command on a known prompt line; while ON, let it fire; screenshot badge still ON with the trigger's command visible | `evidence/08-trigger-keeps-on.png` |
| Dropping the connection shows `[Disconnected]` then `[Autopilot waiting for reconnect]` and the badge reads WAITING | REQ-no-auto-reconnect | Connection lifecycle | While ON, disconnect (button or idle); screenshot the two lines and the WAITING badge | `evidence/09-waiting-after-drop.png` |
| Reconnecting by hand shows `[Reconnected]` then `[Autopilot resuming]` and the badge reads ON without typing `#AUTO ON` | REQ-no-auto-reconnect | Connection lifecycle | Press Connect; screenshot the two lines and the ON badge | `evidence/10-resume-after-reconnect.png` |
| `#AUTO OFF` while WAITING lands on OFF and the badge stays OFF after a hand reconnect | REQ-no-auto-reconnect | Connection lifecycle | While WAITING, type `#AUTO OFF`, then Connect; screenshot the OFF badge after the connection returns | `evidence/11-off-while-waiting-stays-off.png` |
| Ordinary play with the AI OFF looks unchanged: no new lines, badge reads OFF | REQ-doc-hand-play-and-gate | Absence of behaviour is visual | Create a profile, accept the policy, connect, play a few commands; screenshot | `evidence/12-hand-play-unchanged.png` |
| Every autopilot transition emits one `[AI-PLAYER]` line with user/session id, old state, new state, and cause | all four | Captured from the platform log viewer, never from the database | After the walkthrough, capture the `[AI-PLAYER] autopilot` lines with `railway logs --environment staging` | `evidence/04-staging-ai-player.log` |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] Every Per-Task and Manual-Only row cites an evidence file, and no row cites a database query
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
