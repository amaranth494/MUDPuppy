---
phase: 4
slug: continuous-play
status: planned
nyquist_compliant: true
wave_0_complete: false
created: 2026-09-16
---

# Phase 4 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Derived from `04-RESEARCH.md` §Validation Architecture. The planner filled the Per-Task map with real task IDs; the executor flips statuses.
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
- **Before the loop is wired to the live session manager:** the Wave 1 fakes (scripted model queue, real autopilot state machine in `fakeSessions`, output-signal double) exist and the existing Phase 3 suite is still green against them
- **Before `/gsd:verify-work`:** Full suite green with `-race`; the safety-limit suite output captured verbatim to `evidence/01-test-report.txt`; harness self-test and staging report captured; staging screenshots and `[AI-PLAYER]` log excerpt filed for the walkthrough (goal set, paced decisions, cap halt, wheel-grab and re-engage reasoning); the D-24 corpus AFTER report filed
- **Max feedback latency:** 60 seconds (excluding the corpus rerun, which is a phase gate, not a per-task check)

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 04-01-01 | 04-01 | 1 | REQ-safety-limits-hold | T-4-17 | The session double runs the server's own autopilot state machine, so a limit test can no longer pass on a double that always reports Off | unit | `go test ./internal/driver/... -count=1 -v` | ✅ existing (extended) | ⬜ pending |
| 04-01-02 | 04-01 | 1 | REQ-call-cap-and-error-disengage | T-4-18 | A stint of several decisions, including a failure, runs on fakes with no network and no quota spend | unit | `go test ./internal/driver/... -run "TestScriptedStint\|TestFakeSessionsStateMachine" -count=1 -v` | ❌ W1 | ⬜ pending |
| 04-02-01 | 04-02 | 1 | REQ-safety-limits-hold | T-4-06, T-4-19 | Outside local development the server refuses to start without `ENCRYPTION_KEY_V1`; local development is unaffected | unit | `go test ./internal/config/... -run TestLoad -count=1 -v` | ❌ W1 | ⬜ pending |
| 04-02-02 | 04-02 | 1 | REQ-safety-limits-hold | T-4-07 | Migration 012's rollback reconciles blocked rows before the older constraint is restored | unit | `go test ./internal/store/... -run TestMigration -count=1 -v` | ❌ W1 | ⬜ pending |
| 04-03-01 | 04-03 | 2 | REQ-doc-wheel-grab-and-reengage, REQ-safety-limits-hold | T-4-01, T-4-14 | The disengage hook fires on a wheel-grab and on a park; the output signal coalesces and never blocks the relay; the dead statement is gone | unit | `go test ./internal/session/... -count=1 -v` | ✅ existing (extended) | ⬜ pending |
| 04-03-02 | 04-03 | 2 | REQ-continuous-loop | T-4-01, T-4-02, T-4-14 | Both engage hooks target the loop and the disengage hook cancels it; nothing in the loop reconnects | unit | `go vet ./internal/driver/... ./cmd/... && go build ./... && go test ./internal/driver/... -count=1` | ❌ W2 | ⬜ pending |
| 04-03-03 | 04-03 | 2 | REQ-continuous-loop, REQ-reengage-reassess | T-4-01, T-4-20 | Decisions fire on new output after a settle, a quiet game still gets one after the floor, spacing is never violated, a wheel-grab stops a sleeping loop, nothing is issued while waiting, and every engage carries the reassess instruction with a fresh window | unit | `go test ./internal/driver/... -run "TestLoop_\|TestEngageLoop_" -count=1 -v` | ❌ W2 | ⬜ pending |
| 04-04-01 | 04-04 | 3 | REQ-call-cap-and-error-disengage, REQ-safety-limits-hold | T-4-02, T-4-21, T-4-22 | The cap halts before the call that would exceed it and counts both calls; transient failures show "N of M" and reset on a sent command; non-transient failures disengage at once; consecutive blocks disengage with their own notice; blank settings mean no cap and no crash | unit | `go test ./internal/driver/... -run "TestLoop_CallCap\|TestLoop_ErrorThreshold\|TestLoop_ConsecutiveBlocks\|TestLoop_BlankSettings" -count=1 -v` | ❌ W2 | ⬜ pending |
| 04-04-02 | 04-04 | 3 | REQ-call-cap-and-error-disengage | T-4-16, T-4-23 | A 503 is retried exactly once with its notice and the retry is reserved against the cap; every disengage-carrying message reports the new switch state | unit | `go test ./internal/driver/... -run "TestHandleEngage_RetryOn503\|TestAIPayloadCarriesSwitchState" -count=1 -v` | ❌ W2 | ⬜ pending |
| 04-04-03 | 04-04 | 3 | REQ-doc-continuous-visible-play | T-4-16 | The status line, the outcome-keyed terminal colours and the badge-from-message path compile and type-check; the badge component is unchanged | build | `cd frontend && npm run build` | ✅ existing | ⬜ pending |
| 04-05-01 | 04-05 | 3 | REQ-continuous-loop | T-4-24, T-4-25, T-4-26 | The window is bounded by age, never empties, and still obeys the 8 KB ceiling | unit | `go test ./internal/session/... -run TestWindow -count=1 -v` | ✅ existing (extended) | ⬜ pending |
| 04-05-02 | 04-05 | 3 | REQ-reengage-reassess, REQ-safety-limits-hold | T-4-25 | Every decision's snapshot is the reaction window, and the Phase 2/3 window guarantees still hold | unit | `go test ./internal/session/... -count=1 -v` | ✅ existing | ⬜ pending |
| 04-06-01 | 04-06 | 4 | REQ-continuous-loop | T-4-07, T-4-27 | Migration 013 has a paired rollback and a partial unique index; no Phase 4 code path can close a Quest | unit | `go test ./internal/store/... -count=1 -v` | ❌ W4 | ⬜ pending |
| 04-06-02 | 04-06 | 4 | REQ-continuous-loop | T-4-09, T-4-10 | The goal endpoints resolve ownership before reading or writing and cap the goal's length | unit | `go test ./internal/profiles/... -count=1 -v` | ✅ existing (extended) | ⬜ pending |
| 04-06-03 | 04-06 | 4 | REQ-doc-continuous-visible-play | T-4-05 | The goal box compiles with its locked label, placeholder and hint and no Save button | build | `cd frontend && npm run build` | ✅ existing | ⬜ pending |
| 04-07-01 | 04-07 | 5 | REQ-continuous-loop | T-4-03, T-4-10, T-4-28 | Both prompts carry the goal, Quest bullets and Session Memory in D-13's order, with memory delimited as untrusted and bounded in size | unit | `go test ./internal/driver/... -run "TestPromptContextOrder\|TestUntrustedParagraphNamesEveryMarker\|TestMemoryIsWrappedInBothPrompts\|TestMemoryCeilingsAreEnforced" -count=1 -v` | ❌ W5 | ⬜ pending |
| 04-07-02 | 04-07 | 5 | REQ-reengage-reassess | T-4-04 | The reviewer reads the first model's stated reasoning inside untrusted markers, in its unchanged position | unit | `go test ./internal/driver/... -run "TestReviewPromptWrapsReasoning\|TestBuildReviewSystemInstruction\|TestHandleEngageReviewer" -count=1 -v` | ✅ existing (extended) | ⬜ pending |
| 04-07-03 | 04-07 | 5 | REQ-continuous-loop | T-4-04 | The reviewer-channel attack exists as corpus data and the live-corpus gate still holds with the flag unset | unit | `go test ./internal/driver/... -run "TestCorpusIsWellFormed\|TestLiveCorpus" -count=1 -v` | ✅ existing (extended) | ⬜ pending |
| 04-08-01 | 04-08 | 6 | REQ-continuous-loop | T-4-29 | The decision answer carries two optional string arrays; a malformed memory field never costs the command | unit | `go test ./internal/gemini/... -count=1 -v` | ✅ existing (extended) | ⬜ pending |
| 04-08-02 | 04-08 | 6 | REQ-continuous-loop | T-4-03, T-4-10, T-4-29 | Curated memory is truncated to the ceilings, replaces only when supplied, and a store error never stops a command | unit | `go test ./internal/driver/... -run "TestDriverPersistsCuratedMemory\|TestMemoryCeilingsAreEnforced" -count=1 -v` | ❌ W6 | ⬜ pending |
| 04-08-03 | 04-08 | 6 | REQ-doc-continuous-visible-play | T-4-09 | The memory endpoint is read-only and ownership-checked; the panel section has no editable control and shows no Quest Memory | build | `cd frontend && npm run build && go build ./... && go vet ./internal/profiles/... ./cmd/...` | ✅ existing | ⬜ pending |
| 04-09-01 | 04-09 | 7 | REQ-safety-limits-hold | T-4-08, T-4-15 | The retention statements touch only captured text, never a decision's reasoning or outcome and never a memory column, and the per-profile delete is scoped | unit | `go test ./internal/store/... -run TestRetention -count=1 -v` | ❌ W7 | ⬜ pending |
| 04-09-02 | 04-09 | 7 | REQ-safety-limits-hold | T-4-09, T-4-31 | The job runs on a ticker without affecting play, logs counts only, and the delete endpoint resolves ownership first | unit | `go vet ./internal/profiles/... ./cmd/... && go test ./internal/... -count=1` | ✅ existing | ⬜ pending |
| 04-09-03 | 04-09 | 7 | REQ-safety-limits-hold | T-4-30 | The destructive action sits in its own block with the locked scope hint and a two-step confirmation | build | `cd frontend && npm run build` | ✅ existing | ⬜ pending |
| 04-10-01 | 04-10 | 8 | REQ-safety-limits-hold, REQ-doc-continuous-visible-play | T-4-05, T-4-32, T-4-33 | The harness drives the goal, memory and captured-text endpoints, contains no database query, prints no cookie or key, restores what it changed, and marks unreachable criteria SKIP | script | `bash -n scripts/verify-phase4.sh` | ❌ W8 | ⬜ pending |
| 04-10-02 | 04-10 | 8 | REQ-safety-limits-hold | T-4-11 | The harness fails on the one thing it most needs to catch, before its report is trusted | script | `bash scripts/verify-phase4.sh --self-test /tmp/p4-selftest.txt && bash scripts/verify-phase4.sh --self-test-negative /tmp/p4-negative.txt; grep -c "FAIL C4" /tmp/p4-negative.txt` | ❌ W8 | ⬜ pending |
| 04-11-01 | 04-11 | 9 | REQ-safety-limits-hold, REQ-call-cap-and-error-disengage, REQ-continuous-loop | T-4-SC, T-4-11 | The whole suite is green with and without the race detector, the frontend builds, and no package was added | suite | `go test ./... -v` then `go test ./internal/... -race -count=1` then `cd frontend && npm run build` | ✅ existing | ⬜ pending |
| 04-11-02 | 04-11 | 9 | REQ-safety-limits-hold | T-4-04, T-4-18 | The unmodified corpus, now carrying the reviewer-channel attack, shows no steered command reaching the send path on the finished build | gated live | `MUDPUPPY_LIVE_CORPUS=1 MUDPUPPY_LIVE_CORPUS_STRICT=1 go test ./internal/driver/... -run TestLiveCorpus -v` (fresh-quota day only) | ✅ existing | ⬜ pending |
| 04-11-03 | 04-11 | 9 | REQ-continuous-loop, REQ-doc-continuous-visible-play | T-4-07, T-4-12, T-4-13 | Staging runs this build with migration 013 applied and the goal survives a refresh | player-observable + canned report | `bash scripts/verify-phase4.sh` against staging (RUN A) plus `evidence/06-goal-set.png` | n/a | ⬜ pending |
| 04-11-04 | 04-11 | 9 | REQ-continuous-loop, REQ-call-cap-and-error-disengage, REQ-reengage-reassess, REQ-doc-continuous-visible-play, REQ-doc-wheel-grab-and-reengage | T-4-05, T-4-12, T-4-13 | Every decision, notice and count reaches the panel and terminal as it happens; the badge follows the `ai` message; the cap halts on a live game; a wheel-grab and re-engage reassess | player-observable | staging screenshots `evidence/07` through `evidence/13` + RUN B + `evidence/05-staging-ai-player.log` | n/a | ⬜ pending |
| 04-11-05 | 04-11 | 9 | REQ-safety-limits-hold | T-4-03, T-4-34 | Every deferred risk returns to the owner with what was built against it, and the new memory-as-injection-channel risk is named | doc | `test -f .planning/phases/04-continuous-play/04-SECURITY-AGENDA.md && grep -c "Remediate Now" .planning/phases/04-continuous-play/04-SECURITY-AGENDA.md && grep -c "T-4-03" .planning/phases/04-continuous-play/04-SECURITY-AGENDA.md` | ❌ W9 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

*File Exists column: "✅ existing" means the test file is already in the repository and this task extends it; "❌ Wn" means the file is created in that wave.*

---

## Wave 1 Requirements (the phase's own test scaffolding)

- [ ] `internal/driver/loop_test.go` — new file (plan 04-01); the safety-limit suite for REQ-continuous-loop, REQ-call-cap-and-error-disengage, REQ-reengage-reassess, REQ-safety-limits-hold, extended by plans 04-03 and 04-04
- [ ] `internal/driver/driver_test.go` — `fakeModels` gains a scripted queue of answers and errors consumed in call order (today it holds one canned pair, `driver_test.go:42-56`); `fakeSessions` gains a real `session.AutopilotState` driven by the pure `Engage`/`Disengage`/`EnterWaiting`/`Resume` functions (`internal/session/autopilot.go:60-116`) instead of always returning `(Off, true)`; an `AutopilotStateFor` method; an output-signal double the test can fire (plan 04-01)
- [ ] `internal/session/manager_test.go` — the disengage-hook and engage-hook interplay and the output signal, for REQ-doc-wheel-grab-and-reengage and the park/resume behaviour (plan 04-03)
- [ ] `internal/store/migrations_test.go` — the no-database migration assertions, including the pairing guard every later migration must satisfy (plan 04-02)
- [ ] `scripts/verify-phase4.sh` — the canned-report harness: goal round trip, memory read-back, retention counts (plan 04-10)
- [ ] Framework install: none — everything is additive to the existing `testing`-only setup

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Decisions paced to Alter Aeon's output rhythm rather than flooding it | REQ-continuous-loop, REQ-doc-continuous-visible-play | Needs a live MUD; the owner's own character on staging | Set a goal, `#AUTO ON`, watch several decisions arrive after room output settles; capture `evidence/07-paced-decisions.png` and the `[AI-PLAYER]` loop lines (task 04-11-04) |
| Cap halt visible in the panel and terminal | REQ-call-cap-and-error-disengage | Player-observable notice on staging | Set a small cap (for example 4), engage, watch the halt notice and the badge go Off; capture `evidence/08-cap-halt-badge-off.png` (task 04-11-04) |
| Wheel-grab then re-engage reassesses | REQ-reengage-reassess, REQ-doc-wheel-grab-and-reengage | The proof is the logged reasoning of the first new decision on a live game | Type a game command while engaged, move somewhere else by hand, `#AUTO ON`, read the first decision's reasoning; capture `evidence/09-wheel-grab-reengage-reasoning.png` and the log excerpt (task 04-11-04) |
| Session Memory section updates live and survives a refresh | REQ-doc-continuous-visible-play | Browser rendering | Play a few decisions, expand the section, refresh the page, reconnect, confirm the bullets return; capture `evidence/10-session-memory-expanded.png` and `evidence/11-session-memory-after-refresh.png` (task 04-11-04) |
| Delete Captured Text Now, and the decision record surviving it | REQ-safety-limits-hold | Destructive browser action with a confirmation the owner must read | Open AI Player settings, click the action, capture the confirmation state, confirm, then check the decision list still shows reasoning and commands; capture `evidence/12-delete-captured-text-confirm.png` (task 04-11-04) |
| Hand play unchanged on a profile with no policy acceptance | REQ-doc-continuous-visible-play | Regression invariant, browser-visible only | Connect on an unaccepted profile and hand-play; capture `evidence/13-hand-play-unchanged.png` (task 04-11-04) |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or a named manual-only row
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 1 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 60s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
