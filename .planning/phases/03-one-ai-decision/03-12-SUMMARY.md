---
phase: 03-one-ai-decision
plan: 12
subsystem: test
tags: [bash, harness, canned-report, self-test, fixtures]

# Dependency graph
requires:
  - phase: 03-one-ai-decision
    plan: 05
    provides: "GET /api/v1/profiles/{connection_id}/sessions[/{session_id}] response shapes (sessions[], lines[] with source human|ai|game|marker)"
  - phase: 03-one-ai-decision
    plan: 06
    provides: "the exact D-20 refusal sentence and refused-not-configured outcome on POST /api/v1/session/autopilot"
  - phase: 03-one-ai-decision
    plan: 09
    provides: "GET /api/v1/profiles/{connection_id}/decisions response shape (decisions[] with reasoning/command/outcome)"
provides:
  - "scripts/verify-phase3.sh: the Phase 3 canned-report harness (live mode, --self-test, --self-test-negative, --no-tests) producing a PASS/FAIL/SKIP line per ROADMAP C1-C4 with request/response printed under every step"
  - "scripts/fixtures/phase3/: a correct-server fixture set the self-test asserts against"
  - "scripts/fixtures/phase3-negative/: a deliberately wrong fixture set (one file differs) that produces FAIL C3"
affects: [03-13-evidence-capture]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Self-test modes assert a fixed, known world (Gemini variables unset) strictly, rather than accepting either of live mode's two valid worlds — this is what lets --self-test-negative's single swapped fixture reliably produce FAIL C3 instead of being silently accepted as 'the other valid world'"
    - "Fixture-backed _http() call with an ordered FIXTURE_FILES array, mirroring scripts/verify-phase2.sh's shape exactly so the same assertion logic runs against curl or against a fixture file with zero duplication"

key-files:
  created:
    - scripts/verify-phase3.sh
    - scripts/fixtures/phase3/01-autopilot-on-refused-not-configured.json
    - scripts/fixtures/phase3/02-status-agrees.json
    - scripts/fixtures/phase3/03-decisions-own.json
    - scripts/fixtures/phase3/04-decisions-not-owned.json
    - scripts/fixtures/phase3/05-sessions-own.json
    - scripts/fixtures/phase3/06-sessions-transcript.json
    - scripts/fixtures/phase3/07-icm-validate.json
    - scripts/fixtures/phase3-negative/01-autopilot-on-refused-not-configured.json
    - scripts/fixtures/phase3-negative/02-status-agrees.json
    - scripts/fixtures/phase3-negative/03-decisions-own.json
    - scripts/fixtures/phase3-negative/04-decisions-not-owned.json
    - scripts/fixtures/phase3-negative/05-sessions-own.json
    - scripts/fixtures/phase3-negative/06-sessions-transcript.json
    - scripts/fixtures/phase3-negative/07-icm-validate.json
  modified: []

key-decisions:
  - "The live-mode step 1 assertion accepts either of two valid server worlds (refused-not-configured when Gemini variables are unset, or engaged/refused-no-session when they are set) and prints a NOTE recording which world the run demonstrated — but self-test/self-test-negative modes bypass this either/or branching entirely and assert refused-not-configured strictly, because fixtures represent one fixed, known world. Without this split, the negative fixture's outcome:\"engaged\" would have been silently accepted as 'the configured world' instead of failing, defeating the negative self-test's purpose (T-3-10)."
  - "Step 6b (the per-session transcript read) is conditional on the sessions list being non-empty, matching the plan's own conditional language ('when non-empty, read the newest id'). Both fixture sets are constructed to always be non-empty, so the self-test/negative-self-test call sequence is fixed and deterministic; live mode against a profile with no prior game connection will simply skip this call, which is fine because fixture indexing is never consulted when FIXTURE_DIR is unset."
  - "The source-grep step (no model identifier as a Go source literal) and the appended go test runs are both skipped in --self-test and --self-test-negative, per the plan's explicit instruction that both self-test modes run with no server and no repository state — fixtures are the subject of those runs, not this repository's actual Go source or test suite."

patterns-established:
  - "scripts/verify-phase3.sh follows scripts/verify-phase2.sh's shape exactly: usage block, set -uo pipefail with the errexit-omission comment, the _get_field JSON helper, per-step request/response printing, tee to the report path, and a header recording BASE_URL and the git SHA."

requirements-completed: [REQ-env-config, REQ-reasoning-visibility]

# Metrics
duration: ~35min
completed: 2026-09-16
---

# Phase 3 Plan 12: Phase 3 Canned-Report Harness Summary

**`scripts/verify-phase3.sh` drives every Phase 3 behaviour reachable over HTTP or provable by a source grep — the D-20 configuration-gate refusal, the decisions and session-transcript read-back endpoints and their ownership scoping, and the now-live ICM validate route — prints a PASS/FAIL/SKIP line per ROADMAP criterion with the request and response body under every step, and proves it can fail via a fixture set that differs from the correct one by exactly one deliberately wrong response.**

## Performance

- **Duration:** ~35 min (commit-to-commit; reading/context time not included)
- **Started:** 2026-09-16 (post worktree-base reset to `b309e04`)
- **Completed:** 2026-09-16
- **Tasks:** 2 completed
- **Files modified:** 15 (1 script + 14 fixture files)

## Accomplishments

- `scripts/verify-phase3.sh` gives the project a single command that answers, criterion by criterion, whether Phase 3's HTTP-reachable half holds: C3 (the D-20 configuration gate and the no-model-literal source grep), C2 (the decisions and session-transcript read-back endpoints, including their ownership scoping against a not-owned connection id), and C4 (the previously-404 ICM validate route now answering, plus the ICM dispatch and driver-engage diagnostic `go test` runs). C1 and the live half of C2 are printed as `SKIP`, naming the exact screenshot and log evidence that prove them instead, per the plan's own ground rule that they must never appear as `PASS`.
- The harness has demonstrated it can fail: `scripts/fixtures/phase3-negative/` differs from `scripts/fixtures/phase3/` in exactly one file — the autopilot response carries `outcome:"engaged"`/`state:"on"` where the correct server refuses — and running `--self-test-negative` against it prints 4 `FAIL C3` lines and exits non-zero, while `--self-test` against the correct set prints zero `FAIL` lines and exits 0.
- No `psql`, `DATABASE_URL` or SQL of any kind appears in the script; no API key, model name or registry detail is read, printed, or required — confirmed by the grep-based acceptance checks below.

## Task Commits

Each task was committed atomically:

1. **Task 03-12-01: One command produces a PASS/FAIL line per Phase 3 criterion against a running server** - `1e4ee60` (feat)
2. **Task 03-12-02: The harness proves it can fail before anyone trusts a clean run** - `8c6a34d` (test)

_No plan-metadata commit — the orchestrator owns STATE.md/ROADMAP.md updates after all wave worktrees complete, per this plan's parallel-execution instructions._

## Files Created/Modified

- `scripts/verify-phase3.sh` (new) — the harness: usage block, `set -uo pipefail`, `_get_field`, fixture-mode plumbing, 10 steps (C3 x3, C2 x5 including one conditional, C4 x2), `GO TEST EXIT:` line, exit-non-zero-on-any-FAIL.
- `scripts/fixtures/phase3/` (new, 7 files) — one JSON file per `_http` call in the live sequence, in the order the script consumes them: the D-20 refusal, a status response agreeing with it, a well-formed decision, a not-owned-connection refusal, a one-session list, a three-line transcript (`human`/`game`/`ai`), and an ICM validate answer.
- `scripts/fixtures/phase3-negative/` (new, 7 files) — an exact copy of the correct set with one deliberate change: `01-autopilot-on-refused-not-configured.json` carries `{"state":"on","outcome":"engaged","gate_allowed":true,"policy_version":"1.0"}` where the correct server refuses.

## Fixture files and the one that differs

| # | File | Correct set | Negative set |
|---|------|-------------|---------------|
| 1 | `01-autopilot-on-refused-not-configured.json` | `outcome:"refused-not-configured"`, `state:"off"`, D-20 message | **DIFFERS**: `outcome:"engaged"`, `state:"on"`, no gate_message |
| 2 | `02-status-agrees.json` | `{"state":"disconnected","autopilot_state":"off"}` | identical |
| 3 | `03-decisions-own.json` | one decision with `reasoning`, `command`, `outcome:"sent"` | identical |
| 4 | `04-decisions-not-owned.json` | `400 {"error":"Profile not found"}` | identical |
| 5 | `05-sessions-own.json` | one session, `line_count:3` | identical |
| 6 | `06-sessions-transcript.json` | 3 lines: `human`, `game`, `ai` | identical |
| 7 | `07-icm-validate.json` | `200 {"valid":true,...}` | identical |

`diff -rq scripts/fixtures/phase3 scripts/fixtures/phase3-negative` confirms exactly one file differs (`01-autopilot-on-refused-not-configured.json`).

## Decisions Made

See `key-decisions` in the frontmatter above for full rationale. In short: self-test modes assert the fixed unconfigured world strictly rather than accepting either of live mode's two valid worlds, which is what makes the negative fixture's single swapped response reliably produce `FAIL C3` instead of being silently accepted as the alternate configured world; the per-session transcript read (step 6b) is conditional on a non-empty sessions list, with both fixture sets constructed to always exercise it; and the source-grep and `go test` steps are both skipped in self-test modes per the plan's own instruction.

## Deviations from Plan

None — plan executed exactly as written. No auto-fixes were needed; every acceptance criterion in both tasks passed on first implementation once the self-test-mode world-fixing decision above was made during task 2 (this is a design decision the plan's own action text anticipated implicitly by requiring the negative self-test to reliably fail, not a bug fix to prior code).

## Verification Evidence

```
$ bash -n scripts/verify-phase3.sh && grep -c "PASS C3\|FAIL C3" scripts/verify-phase3.sh
SYNTAX OK
2

$ bash scripts/verify-phase3.sh --self-test /tmp/phase3-selftest.txt; echo "EXIT=$?"
... (14 checks, 0 FAIL, 2 SKIP)
EXIT=0

$ bash scripts/verify-phase3.sh --self-test-negative /tmp/phase3-negative.txt; echo "EXIT=$?"
... (14 checks, 4 FAIL — all C3, 2 SKIP)
EXIT=1

$ grep -ciE "psql|DATABASE_URL|SELECT |INSERT |sqlcmd" scripts/verify-phase3.sh
0
$ grep -ciE "AI_MODEL_.*_KEY|GEMINI_API_KEY|x-goog-api-key" scripts/verify-phase3.sh
0
$ grep -c "Autopilot refused: AI is not configured on this server" scripts/verify-phase3.sh
1
$ diff -rq scripts/fixtures/phase3/ scripts/fixtures/phase3-negative/
Files scripts/fixtures/phase3/01-autopilot-on-refused-not-configured.json and scripts/fixtures/phase3-negative/01-autopilot-on-refused-not-configured.json differ
$ grep -rciE "@|AIza|session=[a-f0-9]{16}" scripts/fixtures/phase3 scripts/fixtures/phase3-negative
(all 0)
$ git diff --stat go.mod frontend/package.json
(empty)
```

## Issues Encountered

None. This plan touches only `scripts/verify-phase3.sh` and `scripts/fixtures/phase3{,-negative}/`; no Go or frontend source was read or modified, so the `-race`/cgo environment limitation logged by prior Phase 3 plans does not apply here (the harness's own `go test` invocations intentionally omit `-race`, matching this plan's `<action>` text verbatim, which names `go test ./internal/icm/... -run TestDispatch_AutomationPassThrough -v` and `go test ./internal/driver/... -run TestHandleEngage -v` with no `-race` flag).

## User Setup Required

None for this plan. Running the harness live against staging (`evidence/03-canned-report.txt`) still requires `BASE_URL`, `SESSION_COOKIE`, and a `CONNECTION_ID` naming a policy-accepted profile, per the script's own usage block — that live run is plan 03-13's evidence-capture work, not this plan's.

## Next Phase Readiness

- `scripts/verify-phase3.sh` is ready for plan 03-13 to run live against staging in both the unconfigured world (Gemini variables unset, demonstrating D-20) and, once the owner supplies a key, the configured world.
- `evidence/02-harness-selftest.txt` (the negative self-test output) and the `--self-test` output are both reproducible on demand from the commands above; plan 03-13 can capture them directly.
- No blockers for plan 03-13: this plan's `files_modified` list (`scripts/verify-phase3.sh`, `scripts/fixtures/phase3/`, `scripts/fixtures/phase3-negative/`) is exactly what the plan named, with no overlap against plan 03-10's `frontend/src/**` files running in parallel.

## Known Stubs

None. The harness and both fixture sets are complete artifacts that run standalone; nothing here waits on a later plan to become real.

## Threat Flags

None beyond what this plan's own `<threat_model>` already covers (T-3-10, T-3-42, T-3-17, T-3-43 mitigated; T-3-SC accepted, confirmed by the empty `git diff --stat go.mod frontend/package.json`) — this plan introduces no new network endpoints, auth paths, or schema changes; it only adds a test harness and fixture files.

---
*Phase: 03-one-ai-decision*
*Completed: 2026-09-16*

## Self-Check: PASSED

- FOUND: scripts/verify-phase3.sh
- FOUND: scripts/fixtures/phase3/01-autopilot-on-refused-not-configured.json
- FOUND: scripts/fixtures/phase3/02-status-agrees.json
- FOUND: scripts/fixtures/phase3/03-decisions-own.json
- FOUND: scripts/fixtures/phase3/04-decisions-not-owned.json
- FOUND: scripts/fixtures/phase3/05-sessions-own.json
- FOUND: scripts/fixtures/phase3/06-sessions-transcript.json
- FOUND: scripts/fixtures/phase3/07-icm-validate.json
- FOUND: scripts/fixtures/phase3-negative/01-autopilot-on-refused-not-configured.json
- FOUND: scripts/fixtures/phase3-negative/02-status-agrees.json
- FOUND: scripts/fixtures/phase3-negative/03-decisions-own.json
- FOUND: scripts/fixtures/phase3-negative/04-decisions-not-owned.json
- FOUND: scripts/fixtures/phase3-negative/05-sessions-own.json
- FOUND: scripts/fixtures/phase3-negative/06-sessions-transcript.json
- FOUND: scripts/fixtures/phase3-negative/07-icm-validate.json
- FOUND: 1e4ee60 (Task 1 commit)
- FOUND: 8c6a34d (Task 2 commit)
