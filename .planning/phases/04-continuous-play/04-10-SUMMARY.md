---
phase: 04-continuous-play
plan: 10
subsystem: testing
tags: [bash, harness, canned-report, fixtures, api]

# Dependency graph
requires:
  - phase: 04-continuous-play
    provides: "04-06's GET/PUT /api/v1/profiles/{connection_id}/ai-goal and its Quest reactivate-or-create; 04-08's GET /api/v1/profiles/{connection_id}/ai-memory (Session Memory, read-only); 04-09's DELETE /api/v1/profiles/{connection_id}/captured-text and its two-count response"
provides:
  - "scripts/verify-phase4.sh: the Phase 4 canned-report harness with four invocation modes (live, --self-test, --self-test-negative, --no-tests), PASS/FAIL/SKIP per criterion (C1, C2, C4 reachable; C3, C5 SKIP naming the exact go test and screenshot that prove them), no database query anywhere, and a restore step that leaves a live run's profile exactly as it found it"
  - "scripts/fixtures/phase4/ and scripts/fixtures/phase4-negative/: 16 fixture files each (plus a README naming the handler each mirrors), differing in exactly one file -- the decisions-after-delete response's surviving decision has its reasoning emptied, proving the harness can FAIL before its report is trusted (T-4-11)"
  - "internal/profiles/handler.go: GoalResponse's new PUT-only \"quest\" field (created/reactivated/none), exposing D-04's create-vs-reactivate distinction over HTTP for the first time -- previously only in the [AI-PLAYER] log line, which the evidence rule forbids querying"
affects: [04-continuous-play plan 04-11 (the staging evidence plan runs this harness live against staging for evidence/03-canned-report.txt and evidence/02-harness-selftest.txt)]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "The harness is scripts/verify-phase3-1.sh's exact shape (set -uo pipefail with errexit omitted, fixture-or-curl _http helper, flat-JSON extractors, PASS/FAIL/SKIP-per-criterion report) repointed at three new endpoints -- the same harness pattern now used four phases running"
    - "A create-vs-reactivate word that only ever reached a log line is promoted to an HTTP response field (GoalResponse.Quest, omitempty, PUT-only) specifically so a canned-report harness can prove it without a database query -- the evidence rule shapes the API, not just the test"
    - "The fake store powering a create-vs-reactivate unit test uses a monotonic tick counter instead of time.Now(), because two fake-store calls made back-to-back in a fast test can land on the same wall-clock instant on a coarse-resolution clock, silently making a genuine reactivation look like a fresh creation"

key-files:
  created:
    - scripts/verify-phase4.sh
    - scripts/fixtures/phase4/ (16 fixtures + README.md)
    - scripts/fixtures/phase4-negative/ (16 fixtures + README.md)
    - .planning/phases/04-continuous-play/deferred-items.md
  modified:
    - internal/profiles/handler.go
    - internal/profiles/handler_test.go

key-decisions:
  - "GoalResponse gained a PUT-only \"quest\" field (Rule 2 deviation, not in the plan's declared files_modified) because 04-06-PLAN.md shipped the create-vs-reactivate distinction only as a word in the [AI-PLAYER] log line; the harness -- and the project's own evidence rule, which forbids a database query as proof -- has no other way to observe D-04's reactivate-rather-than-duplicate behavior over HTTP"
  - "16 fixture files, one per HTTP call in the harness's exact live-mode call order (goal round trip x6, memory x2, decisions/delete/decisions x3, ownership x3, restore x1), so the self-test exercises the identical assertion code path the live run does"
  - "The negative fixture set changes exactly one file (12-decisions-get-after.json), emptying the surviving decision's reasoning -- the single most important thing C4 asserts (the audit record survives the prune), per the plan's own instruction"
  - "GOAL_A/GOAL_B are fixed literal strings in self-test modes (so static fixtures can match them) and timestamp-suffixed in live mode (so two live runs in the same minute never collide with a goal already on the profile)"
  - "PutAIMemory has no PUT handler at all (route registered GET-only in cmd/server/main.go); the harness's C2 PUT-refused check therefore expects a plain-text 405 body, not JSON, matching http.Error's real output rather than assuming a JSON error shape"

patterns-established:
  - "A harness fixture directory always ships a README naming which handler each file mirrors, so a later response-shape change is traceable back to the fixture that needs updating (04-10-02's own acceptance criterion, generalizing scripts/fixtures/phase3-1/'s undocumented precedent)"

requirements-completed: [REQ-safety-limits-hold, REQ-doc-continuous-visible-play]

# Metrics
duration: 55min
completed: 2026-09-17
---

# Phase 4 Plan 10: Phase 4 Canned-Report Harness and Its Negative Self-Test Summary

**A four-mode bash harness (`scripts/verify-phase4.sh`) that drives the session-goal round trip and Quest reactivation, the Session Memory read-back, and the captured-text retention delete over HTTP with zero database queries, backed by matched 16-file fixture sets that prove the harness can FAIL before its PASS lines are trusted.**

## Performance

- **Duration:** ~55 min (context gathering through both self-tests green)
- **Started:** 2026-09-16T23:30:00-07:00 (approx, first file read)
- **Completed:** 2026-09-16T23:37:45-07:00 (approx, second commit)
- **Tasks:** 2 completed
- **Files modified:** 37 (1 new script, 32 new fixture files across two directories, 1 new deferred-items log, 2 modified Go source/test files, 1 new SUMMARY)

## Accomplishments

- `scripts/verify-phase4.sh` drives the three HTTP-reachable Phase 4 surfaces end to end: **C1** (D-01/D-03/D-04) — a distinctive goal round-trips byte for byte, a second goal proves an edit takes, the same goal re-cased and padded reactivates its Quest rather than duplicating it (now provable over HTTP via the new `quest` response field), and a 1001-character goal is refused with the exact cap sentence; **C2** (D-10) — Session Memory reads back as a well-formed array and refuses a hand-edit PUT with 405; **C4** (D-21) — the captured-text delete reports both counts and the decision audit record (id, outcome, command, failure_kind, reasoning) survives the prune afterward. Ownership refusal (T-4-09) is checked independently on all three endpoints against a fixed non-existent connection id. `SKIP C3` and `SKIP C5` each name the exact `go test -run` invocation and screenshot filename that prove the mechanical limits and the wheel-grab reassessment instead of falsely claiming them.
- `scripts/fixtures/phase4/` (16 files + README) and `scripts/fixtures/phase4-negative/` (the same, with the decisions-after-delete response's `reasoning` emptied) prove the harness's negative-test contract: the clean self-test produces 35 PASS-or-skip lines, zero `FAIL`, exit 0; the negative self-test produces exactly one `FAIL C4` line and exits 1 (T-4-11). Both run with no server, no cookie, and no connection id.
- **Rule 2 deviation:** `internal/profiles/handler.go`'s `GoalResponse` gained a PUT-only `quest` field so the harness (and any HTTP caller) can observe D-04's create-vs-reactivate distinction without a database query — previously this word existed only inside the `[AI-PLAYER] goal` log line, which the project's own evidence rule forbids querying as proof. `TestGoalRoundTrip` was extended to cover all three values (`created`, `reactivated`, `none`) and that `GetGoal` never sets the field.
- The harness contains no SQL, issues no Gemini call, never echoes the session cookie or a model key, and its restore step (step 16) decodes the original goal before ever re-encoding it into the restore PUT — the Phase 3.1 double-escaping defect (03.1-07-SUMMARY.md) deliberately not repeated.

## Task Commits

Each task was committed atomically:

1. **Task 04-10-01: One command checks Phase 4's reachable half and writes the report** - `1d100b5` (feat)
2. **Task 04-10-02: The harness proves it can fail before its report is trusted** - `dfa66c8` (test)

**Plan metadata:** (this commit)

## Files Created/Modified

- `scripts/verify-phase4.sh` — the harness: argument parsing, `_http` fixture-or-curl helper, `_get_field`/`_get_text_field`/`_count_array_items`/`_last_decision_field` extractors, 16 live-mode steps, the two SKIP lines, the appended `go test` run
- `scripts/fixtures/phase4/*.json` (16 files) + `README.md` — the correct-server fixture set, shaped from `internal/profiles/handler.go` and `decisions.go`'s real response structs
- `scripts/fixtures/phase4-negative/*.json` (16 files) + `README.md` — the same set with `12-decisions-get-after.json`'s `reasoning` emptied
- `internal/profiles/handler.go` — `GoalResponse.Quest` field; `PutGoal`'s final response now carries `questWord`
- `internal/profiles/handler_test.go` — `fakeQuestStore`'s monotonic-tick create-vs-reactivate simulation; `TestGoalRoundTrip` extended
- `.planning/phases/04-continuous-play/deferred-items.md` — the one out-of-scope flaky-test observation logged, not fixed

## Decisions Made

See `key-decisions` in the frontmatter above.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing critical] `GoalResponse` gained a `quest` field so D-04's create-vs-reactivate distinction is provable over HTTP**
- **Found during:** Task 04-10-01, while reading `internal/profiles/handler.go` per its `read_first` instruction
- **Issue:** The plan's task instructions say to "assert from the response that the same Quest was reactivated rather than a second one created, using whichever field 04-06-SUMMARY.md records the PUT returning for that purpose." Reading both `handler.go` and `04-06-SUMMARY.md` showed no such field exists: the create-vs-reactivate word (`questWord`) is computed in `PutGoal` but only ever reaches the `[AI-PLAYER] goal` log line, never the JSON response (`GoalResponse{Goal: ...}` only). Without a response field, this plan's must-have truth ("machine-checked over the wire... proven by the PASS C1 lines") and the project's own evidence rule (a database query is never evidence) would be impossible to satisfy for the Quest-reactivation half of C1.
- **Fix:** Added `Quest string \`json:"quest,omitempty"\`` to `GoalResponse`, populated only by `PutGoal` (values `"created"`, `"reactivated"`, `"none"`) using the already-computed `questWord`; `GetGoal` never sets it, so it is omitted on every read. This is additive and non-breaking: existing decoders of `GoalResponse` (the frontend's TS type, `TestGoalRoundTrip`'s pre-existing assertions) are unaffected by an extra field.
- **Files modified:** `internal/profiles/handler.go`, `internal/profiles/handler_test.go`
- **Verification:** `go build ./...`, `go vet ./...`, `go test ./internal/... -count=1` all green; `TestGoalRoundTrip` extended with three new assertions (created/reactivated/none) plus a check that `GetGoal`'s response never carries the field.
- **Committed in:** `1d100b5` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 missing critical HTTP-observability gap)
**Impact on plan:** Necessary for the plan's own C1 must-have truth to be checkable without a database query, per the project's binding evidence rule. No scope creep beyond the one field and its test coverage; no schema, route, or architectural change.

## Issues Encountered

- A flaky `internal/driver` test surfaced once during full-suite verification (`go test ./internal/... -count=1`), unrelated to this plan's diff (which touches only the harness, fixtures, and the goal endpoint). Three immediate reruns were all green. Logged in `deferred-items.md` rather than chased down, per the scope-boundary rule — this plan does not touch `internal/driver`.
- `time.Now().UTC()` called twice in quick succession inside `fakeQuestStore.EnsureActiveQuest` produced identical timestamps on the first attempt, making a genuine reactivation look like a fresh creation in the new `TestGoalRoundTrip` assertions. Fixed by switching the fake to a monotonic tick counter (`time.Unix(f.tick, 0)`) instead of wall-clock time — a within-task bug fix (Rule 1), not a deviation from the plan.

## User Setup Required

None — no external service configuration required. This plan runs no live server, no live Gemini call, and touches no staging state.

## Next Phase Readiness

- `scripts/verify-phase4.sh` is ready for plan 04-11 to run live against staging, producing `evidence/03-canned-report.txt` (the exact invocation is documented below) and, before that, `evidence/02-harness-selftest.txt` from re-running both self-test modes verbatim.
- The harness's C1/C2/C4 checks assume at least one decision already exists on the connection for the survival check (step 12) to exercise fully; if none exists yet, that one assertion pair is skipped with an explanatory line rather than fabricated — plan 04-11's walkthrough should generate at least one decision (a single engage/decision cycle) before running the live report.
- No blockers. `go build ./...`, `go vet ./...`, `go test ./internal/... -count=1`, and `bash -n scripts/verify-phase4.sh` all pass on this machine; `git diff --stat go.mod frontend/package.json` is empty (T-4-SC). No frontend changes were needed this plan (the new `quest` field is additive and unused by the frontend, which does not decode it).

### Live invocation for plan 04-11 (staging)

```
BASE_URL=<staging-url> \
SESSION_COOKIE="session=<owner's session cookie>" \
CONNECTION_ID=<connection_id of the Alter Aeon profile> \
scripts/verify-phase4.sh evidence/03-canned-report.txt
```

Optional: `NOT_OWNED_CONNECTION_ID` (defaults to `ffffffff-ffff-ffff-ffff-ffffffffffff`, a fixed literal UUID no profile will ever have). Add `--no-tests` to skip the appended `go test ./internal/driver/... ./internal/session/... -count=1` run if it was already captured separately.

### Criterion label table (as shipped)

| Label | What it checks | Reachable | This run's verdict source |
|-------|-----------------|-----------|----------------------------|
| C1 | Goal round trip, edit, Quest reactivate-vs-create (D-01, D-03, D-04) | yes | `PASS`/`FAIL` C1 lines |
| C2 | Session Memory read-back on attach, read-only (D-10) | yes | `PASS`/`FAIL` C2 lines |
| C3 | Call cap, threshold disengage, blocked-repeatedly, blank settings (D-14 to D-17) | no | `SKIP C3` naming `go test ./internal/driver/... -run "TestLoop_CallCap\|TestLoop_ErrorThreshold\|TestLoop_ConsecutiveBlocks\|TestLoop_BlankSettings" -count=1 -v` and `evidence/08-cap-halt-badge-off.png` |
| C4 | Retention delete counts + decision audit survival (D-21) | yes | `PASS`/`FAIL` C4 lines |
| C5 | Wheel-grab and re-engage reassessment (D-08) | no | `SKIP C5` naming `go test ./internal/driver/... -run "TestLoop_\|TestEngageLoop_" -count=1 -v` and `evidence/09-wheel-grab-reengage-reasoning.png` |

### Quest-reactivation field

`PUT /api/v1/profiles/{connection_id}/ai-goal`'s response now carries `"quest"`, one of `"created"` (a new normalized goal), `"reactivated"` (the same normalized goal as an existing active Quest), or `"none"` (a blank goal, creating nothing). `GET` never sets this field. This is the field the harness's step 6 checks equal `"reactivated"` to prove D-04.

### Negative fixture

`scripts/fixtures/phase4-negative/12-decisions-get-after.json` is the one file that differs from `scripts/fixtures/phase4/12-decisions-get-after.json`: the surviving decision's `"reasoning"` field is emptied (`""` instead of the descriptive sentence), which trips `FAIL C4: the decision's reasoning is still non-empty after the delete -- the audit record survives the prune`.

### Both self-test exit statuses

- `bash scripts/verify-phase4.sh --self-test <file>` → exit **0**, 35 checks, 0 FAIL, 2 SKIP.
- `bash scripts/verify-phase4.sh --self-test-negative <file>` → exit **1**, one `FAIL C4` line.

---
*Phase: 04-continuous-play*
*Completed: 2026-09-17*
