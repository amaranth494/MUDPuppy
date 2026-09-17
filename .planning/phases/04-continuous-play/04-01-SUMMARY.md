---
phase: 04-continuous-play
plan: 01
subsystem: testing
tags: [go, testing, driver, autopilot, fakes, table-driven]

# Dependency graph
requires:
  - phase: 03-one-ai-decision
    provides: HandleEngage's complete single-decision path (window snapshot, buildSystemInstruction, GenerateContent, validateCommand, matchNeverIssue, ReviewCommand, Dispatch, SendCommandAs, recordSuccess/recordFailure/recordBlocked) and the driver_test.go fakes this plan extends
  - phase: 02-autopilot-switch
    provides: internal/session/autopilot.go's pure Engage/Disengage/EnterWaiting/Resume functions and the AutopilotState values the session double now runs instead of a hard-coded return
provides:
  - A scripted queue on fakeModels (answers/errs, reviewAnswers/reviewErrs) that returns entries in call order, falls back to the original single-answer fields when empty, and reports an overrun count instead of panicking past the end
  - fakeSessions rebuilt so DisengageAutopilot and a full set of test-facing setters (engageState, disengageState, enterWaiting, resume, setState) run the real internal/session/autopilot.go transition functions against a stored state field, plus AutopilotStateFor and an OutputSignal/fireOutput double
  - internal/driver/loop_test.go, the new home of the Phase 4 safety-limit suite (D-20), opening with TestScriptedStint, TestFakeSessionsStateMachine, and TestScriptedStintSurvivesAFailureMidway
  - Sessions interface widened with AutopilotStateFor(userID string) session.AutopilotState, matching the real *session.Manager's existing method exactly
affects: [04-continuous-play plans 04-02 through 04-11, especially 04-03 (the loop itself) which builds directly on loop_test.go's three tests]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Scripted-queue test double: parallel slices indexed by call order (before increment), overrun returns the last entry again plus a counter, empty queue falls back to the original single-value fields for zero-touch backward compatibility"
    - "Test doubles delegate to the real package's pure state-transition functions (session.Engage/Disengage/EnterWaiting/Resume) rather than reimplementing or hard-coding transition outcomes, so a test can no longer pass on a lie about state"

key-files:
  created:
    - internal/driver/loop_test.go
  modified:
    - internal/driver/driver_test.go
    - internal/driver/driver.go
    - internal/driver/corpus_live_test.go

key-decisions:
  - "The scripted queue is two parallel slices (answers/errs) indexed by the same call-order counter, not a single slice of {answer,err} pairs, matching the plan's literal field list; an index is 'the error' when errs[idx] is non-nil, otherwise 'the answer', so a test can express both by only populating the relevant slice at a given index."
  - "reviewAnswers/reviewErrs are indexed by ReviewCommand's own call count, not by the outer decision number -- when a decision fails before the reviewer is ever called (TestScriptedStintSurvivesAFailureMidway's middle decision), the reviewer queue has one fewer entry than the decision count, exactly matching how many times ReviewCommand actually ran."
  - "fakeSessions' zero-value AutopilotState field ('') is normalized to session.AutopilotOff by a private currentStateLocked() helper on every read and transition, so a &fakeSessions{} literal that never sets state behaves like a freshly booted switch without needing a constructor function every existing test call site would have had to adopt."

patterns-established:
  - "Pattern: any future Sessions-interface addition needs `git grep -l DisengageAutopilot` across _test.go files, not just driver_test.go -- corpus_live_test.go's separate corpusSessions double satisfies the same interface and silently needed the same widening."

requirements-completed: [REQ-safety-limits-hold, REQ-call-cap-and-error-disengage]

duration: 9min
completed: 2026-09-16
---

# Phase 4 Plan 01: Test scaffolding for a multi-decision stint Summary

**Scripted model-call queue plus a session double that runs the real autopilot state machine, proven end to end by three new tests in a fresh `loop_test.go`, with zero changes to Phase 3/3.1 production behaviour.**

## Performance

- **Duration:** 9 min (dd15a98 base to final task commit)
- **Started:** 2026-09-16T21:12:56-07:00
- **Completed:** 2026-09-16T21:21:31-07:00
- **Tasks:** 2 completed
- **Files modified:** 4 (1 created, 3 modified)

## Accomplishments
- A stint of three consecutive AI decisions runs end to end on fakes with no network, each getting its own scripted answer and reviewer verdict, proven by `TestScriptedStint` (evidence-of-record for evidence/01-test-report.txt, filed by plan 04-11).
- The session double can no longer lie about the autopilot switch: `DisengageAutopilot` and every setter now runs `internal/session/autopilot.go`'s own pure transition functions, proven by `TestFakeSessionsStateMachine`, including the already-off `changed=false` case the old fixed return could never produce (T-4-17).
- The harness can describe a stint that survives a mid-stint failure -- two sends, one failure row with a failure kind, one disengage, no panic -- proven by `TestScriptedStintSurvivesAFailureMidway`, the shape later Phase 4 plans' thresholds will enforce.
- Zero production behaviour changes beyond the single, already-implemented `AutopilotStateFor` method being declared on the `Sessions` interface; the entire pre-existing driver test suite passes unedited, and `go test ./internal/... -count=1` shows no new failures anywhere in the repo.

## Task Commits

Each task was committed atomically:

1. **Task 04-01-01: The test doubles can script a stint and tell the truth about the switch** - `97d6e82` (test)
2. **Task 04-01-02: A stint of several decisions runs end to end with no network** - `9fa3903` (test)

**Plan metadata:** (this commit, made by the orchestrator after all worktree agents in the wave complete)

## Files Created/Modified
- `internal/driver/loop_test.go` - New file, package driver: the Phase 4 safety-limit suite's home (D-20), opening with `TestScriptedStint`, `TestFakeSessionsStateMachine`, and `TestScriptedStintSurvivesAFailureMidway`
- `internal/driver/driver_test.go` - `fakeModels` gained a scripted `answers`/`errs`/`reviewAnswers`/`reviewErrs` queue with an `overruns` counter and `systemInstructions`/`windows` recording slices; `fakeSessions` gained a real `state` field driven by `internal/session/autopilot.go`'s pure functions (`engageState`, `disengageState`, `enterWaiting`, `resume`, `setState`, `AutopilotStateFor`), an `OutputSignal`/`fireOutput` double, and a `waitForDisengages` polling helper beside `waitForCalls`
- `internal/driver/driver.go` - `Sessions` interface widened with `AutopilotStateFor(userID string) session.AutopilotState`, matching the real `*session.Manager`'s existing method; no other behaviour changed
- `internal/driver/corpus_live_test.go` - `corpusSessions` (a separate file-local `Sessions` double for the live red-team harness) given a minimal `AutopilotStateFor` returning a fixed `AutopilotOff`, required to keep the build green after the interface widened (Rule 3)

## Decisions Made
- The scripted queue's exhaustion behaviour matches the plan's Claude's Discretion exactly: return the last queued item again and increment a recorded `overruns` counter, never panic.
- `reviewAnswers`/`reviewErrs` are indexed by `ReviewCommand`'s own call count, not by decision number -- documented in `loop_test.go`'s `TestScriptedStintSurvivesAFailureMidway` comment since a decision that fails before the reviewer runs shifts every later review-queue index down by one.
- Which existing tests needed no edit: every test in `driver_test.go` predating this plan (`TestHandleEngage`, `TestBuildSystemInstruction`, `TestBuildReviewSystemInstruction`, `TestMatchNeverIssue`, `TestHandleEngageNeverIssue`, `TestHandleEngageFailures`, `TestHandleEngageReviewer`) passed unedited because the scripted queue and the real state machine both fall back to the original single-value fields/fixed-off-start behaviour when a test never populates the new fields.
- `-race` availability: `go test ./internal/driver/... -race -count=1` ran clean on this machine (1.77s), so no race-detector-unavailable note is needed.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] corpus_live_test.go's separate Sessions double needed AutopilotStateFor too**
- **Found during:** Task 04-01-01, `go build ./...` verification
- **Issue:** `corpus_live_test.go` defines its own file-local `corpusSessions` double for the live red-team harness, which also implements `driver.Sessions`. Widening the interface with `AutopilotStateFor` (as the plan's task 1 requires) broke `go build ./...` because `corpusSessions` had no such method. The plan's `<files>` list for this task named only `driver_test.go`/`driver.go`, and its Claude's Discretion note says "`corpus_live_test.go`'s separate `corpus*` fakes are not touched" — but that note predates the interface-widening requirement and the two are in tension.
- **Fix:** Added a minimal `func (c *corpusSessions) AutopilotStateFor(userID string) session.AutopilotState { return session.AutopilotOff }` to `corpus_live_test.go`. The corpus harness classifies every item solely on sends/dispatches/disengages and never reads switch state, so a fixed return changes nothing about what that harness measures.
- **Files modified:** `internal/driver/corpus_live_test.go`
- **Verification:** `go build ./...` exits 0; `go vet ./internal/driver/...` clean; the full pre-existing driver suite (including `corpus_live_test.go`'s own tests, which are network-gated and did not run live here) still compiles as one package.
- **Committed in:** `97d6e82` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Necessary to keep the build green after the interface widening the plan itself specifies; no scope creep, no behaviour change to the corpus harness's actual red-team logic.

## Issues Encountered

Two of task 1's own acceptance-criteria grep assertions do not hold, for reasons that predate this plan or are inherent to the plan's own required test names, not code defects:

1. **`grep -c "time.Sleep" internal/driver/driver_test.go` returns 2, not 0.** The pre-existing `waitForCalls` helper (present before this plan, line ~312) already contains a `time.Sleep(time.Millisecond)` poll-interval inside its deadline-polling loop -- confirmed present on the pre-plan `dd15a98` base before any edits. This plan's `read_first` explicitly directs new helpers to copy `waitForCalls`' exact polling shape, so the new `waitForDisengages` helper adds a second, identical poll-interval sleep. Neither is a sleep-based assertion (the loop tests `s.disengages() >= n`>, not a fixed sleep-then-check); the grep is not distinguishing "sleep as an assertion" from "sleep as a poll interval inside a bounded deadline loop." `internal/driver/loop_test.go` itself (task 2's own file, and the one the plan's overall `<verification>` section 5 checks) correctly has zero `time.Sleep` occurrences.
2. **`grep -c "func TestScriptedStint" internal/driver/loop_test.go` returns 2, not 1.** The unanchored grep pattern also matches `func TestScriptedStintSurvivesAFailureMidway`, which task 2 explicitly requires the file to contain. This is a substring-match artifact in the acceptance criterion's own grep pattern, not a naming problem in the code -- `TestScriptedStint` and `TestScriptedStintSurvivesAFailureMidway` are exactly the two distinctly-named tests the plan's `<action>` section asks for.

Neither affects the plan's actual `<verification>` and `<success_criteria>` sections, which this plan satisfies in full (see below).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `internal/driver/loop_test.go` exists with the three tests plan 04-03 (the loop itself) will extend with cap, threshold, retry, and block-count cases.
- `fakeModels`' scripted queue and `fakeSessions`' real state machine are ready for every later Phase 4 plan's tests without further doubling work.
- No blockers. The plan's own `<verification>` steps all pass on this machine: `go build ./...` clean, `go vet ./internal/driver/...` clean, the full pre-existing suite plus the three new tests all PASS, `go test ./internal/...` shows no new failures, `grep -c "return session.AutopilotOff, true"` returns 0, and `git diff --stat go.mod frontend/package.json` is empty.

---
*Phase: 04-continuous-play*
*Completed: 2026-09-16*
