---
phase: 4
slug: continuous-play
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-09-16
---

# Phase 4 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Derived from `04-RESEARCH.md` §Validation Architecture. The planner fills the Per-Task map with real task IDs; the executor flips statuses.
>
> **Evidence rule (owner-directed, binding).** Every success criterion and acceptance criterion is proven by one of exactly two artifact types: a **canned report** (a repeatable script or test run whose output is captured verbatim into a file) or a **screenshot** (an image of the browser as the owner sees it, no devtools, terminal, or raw JSON in frame). A server log excerpt captured to a file counts as a canned report. **Running a database query is not evidence** and no row in this document may cite one. Every row's evidence lands under `.planning/phases/04-continuous-play/evidence/`.
>
> **Model-call rule.** The safety-limit suite runs on fakes with no network. Live Gemini calls happen only on staging during the owner's walkthrough and in the gated corpus rerun (D-24); the free-tier quota is per model per day on `gemini-3.5-flash-lite`, so no test in this phase may call the real model unflagged.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go standard library `testing` (same as Phases 1 to 3.1); no mocking library, no fake-clock library; no frontend test tooling (`frontend/package.json` `"test": "echo 'No tests yet' && exit 0"`) |
| **Config file** | none — `go.mod` at repo root, `go 1.26` |
| **Quick run command** | `go test ./internal/driver/... ./internal/session/... -count=1` |
| **Full suite command** | `go test ./internal/... -count=1` |
| **Race run** | `go test ./internal/... -race -count=1` (MinGW-w64 is present on the dev machine; measured 9.4 s green on the Phase 3.1 build) |
| **Frontend check** | `cd frontend && npm run build` — `tsc && vite build`, the only automated frontend gate this repo has |
| **Canned report harness** | `scripts/verify-phase4.sh` (same shape as `scripts/verify-phase3-1.sh`: `set -uo pipefail`, PASS/FAIL/SKIP per criterion, `--self-test` / `--self-test-negative` / `--no-tests` modes) — proves the goal round trip, the Session Memory read-back on attach, and the retention job's counts through the HTTP surface and the `[AI-PLAYER]` log line, never a database query |
| **Live corpus rerun** | `MUDPUPPY_LIVE_CORPUS=1 go test ./internal/driver/... -run TestLiveCorpus -v` on a fresh-quota day for the D-24 AFTER report (gated; skipped otherwise; refuses on Railway) |
| **Estimated runtime** | ~2.2 s full Go suite without `-race`, ~9.4 s with; ~8.5 s cold `go build ./...`; ~40 s `npm run build`; ~30 s harness against staging |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/driver/... ./internal/session/... -count=1`
- **After every plan wave:** Run `go test ./internal/... -count=1` and `cd frontend && npm run build`
- **Before the loop is wired to the live session manager:** the Wave 0 fakes (scripted model queue, real autopilot state machine in `fakeSessions`, output-signal double) exist and the existing Phase 3 suite is still green against them
- **Before `/gsd:verify-work`:** Full suite green with `-race`; the safety-limit suite output captured verbatim to `evidence/`; harness self-test and staging report captured; staging screenshots and `[AI-PLAYER]` log excerpt filed for the walkthrough (goal set, paced decisions, cap halt, wheel-grab and re-engage reasoning); the D-24 corpus AFTER report filed
- **Max feedback latency:** 60 seconds (excluding the corpus rerun, which is a phase gate, not a per-task check)

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| _(planner fills)_ | | | REQ-continuous-loop | | Loop fires multiple decisions, paced by the output-arrived signal and the floor interval, never closer than the minimum spacing | unit | `go test ./internal/driver/... -run TestLoop_Pacing -v` | ❌ W0 | ⬜ pending |
| _(planner fills)_ | | | REQ-call-cap-and-error-disengage | | Cap halts the loop before the call that would exceed it, counts decision, reviewer and retry calls, lands OFF with the locked notice; blank cap never halts | unit | `go test ./internal/driver/... -run TestLoop_CallCap -v` | ❌ W0 | ⬜ pending |
| _(planner fills)_ | | | REQ-call-cap-and-error-disengage | | Consecutive transient failures show "N of M", reset on a sent command, disengage at the threshold; blank threshold resolves to 3; non-transient failures disengage at once | unit | `go test ./internal/driver/... -run TestLoop_ErrorThreshold -v` | ❌ W0 | ⬜ pending |
| _(planner fills)_ | | | REQ-call-cap-and-error-disengage | | A vendor 503 is retried exactly once with the retrying notice; the retry counts against the cap; a second 503 is a transient failure | unit | `go test ./internal/driver/... -run TestHandleEngage_RetryOn503 -v` | ❌ W0 | ⬜ pending |
| _(planner fills)_ | | | REQ-call-cap-and-error-disengage | | Consecutive blocks skip the turn, are counted separately, disengage at the threshold with their own notice, reset by a sent command | unit | `go test ./internal/driver/... -run TestLoop_ConsecutiveBlocks -v` | ❌ W0 | ⬜ pending |
| _(planner fills)_ | | | REQ-reengage-reassess | | The first decision after any engage carries a fresh window snapshot and the reassess instruction; no prior plan text is carried | unit | `go test ./internal/driver/... -run TestEngageLoop_Reassess -v` | ❌ W0 | ⬜ pending |
| _(planner fills)_ | | | REQ-safety-limits-hold | | Nothing is issued while WAITING; a resume restarts the loop; the AI never reconnects | unit | `go test ./internal/session/... -run "TestManager_LoopStopsOnPark\|TestAutopilot" -v` | ❌ W0 / ✅ existing | ⬜ pending |
| _(planner fills)_ | | | REQ-safety-limits-hold | | Blank settings: no cap, informative failure, no crash, regular play unaffected | unit | `go test ./internal/driver/... -run TestLoop_BlankSettings -v` | ❌ W0 | ⬜ pending |
| _(planner fills)_ | | | REQ-doc-wheel-grab-and-reengage | | The disengage hook cancels a loop asleep in its pacing wait or waiting on a model call, so a typed command disengages at once | unit | `go test ./internal/session/... -run TestManager_DisengageHookFires -v` | ❌ W0 | ⬜ pending |
| _(planner fills)_ | | | REQ-doc-continuous-visible-play | | Every decision, notice, and count reaches the panel and terminal as it happens; the badge follows the `ai` message | player-observable | staging screenshots + `[AI-PLAYER]` log excerpt | n/a | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/driver/loop_test.go` — new file; the safety-limit suite for REQ-continuous-loop, REQ-call-cap-and-error-disengage, REQ-reengage-reassess, REQ-safety-limits-hold
- [ ] `internal/driver/driver_test.go` — `fakeModels` gains a scripted queue of answers and errors consumed in call order (today it holds one canned pair, `driver_test.go:42-56`); `fakeSessions` gains a real `session.AutopilotState` driven by the pure `Engage`/`Disengage`/`EnterWaiting`/`Resume` functions (`internal/session/autopilot.go:60-116`) instead of always returning `(Off, true)`; an `AutopilotStateFor` method; an output-signal double the test can fire
- [ ] `internal/session/manager_test.go` (new or extended) — the disengage-hook and engage-hook interplay for REQ-doc-wheel-grab-and-reengage and the park/resume behaviour
- [ ] `scripts/verify-phase4.sh` — the canned-report harness (goal round trip, memory read-back, retention counts)
- [ ] Framework install: none — everything is additive to the existing `testing`-only setup

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Decisions paced to Alter Aeon's output rhythm rather than flooding it | REQ-continuous-loop, REQ-doc-continuous-visible-play | Needs a live MUD; the owner's own character on staging | Set a goal, `#AUTO ON`, watch several decisions arrive after room output settles; capture screenshots and the `[AI-PLAYER]` loop lines |
| Cap halt visible in the panel and terminal | REQ-call-cap-and-error-disengage | Player-observable notice on staging | Set a small cap (for example 4), engage, watch the halt notice and the badge go OFF; screenshot |
| Wheel-grab then re-engage reassesses | REQ-reengage-reassess, REQ-doc-wheel-grab-and-reengage | The proof is the logged reasoning of the first new decision on a live game | Type a game command while engaged, move somewhere else by hand, `#AUTO ON`, read the first decision's reasoning; screenshot and log excerpt |
| Session Memory section updates live and survives a refresh | REQ-doc-continuous-visible-play | Browser rendering | Play a few decisions, expand the section, refresh the page, reconnect, confirm the bullets return; screenshot |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
