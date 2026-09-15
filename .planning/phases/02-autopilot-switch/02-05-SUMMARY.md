---
phase: 02-autopilot-switch
plan: 05
subsystem: testing
tags: [bash, curl, jq-optional, self-test-harness, canned-report, autopilot]

# Dependency graph
requires:
  - phase: 02-autopilot-switch (plan 02)
    provides: "POST /api/v1/session/autopilot (on/off/status), GET /api/v1/session/status carrying autopilot_state, AutopilotRequest/AutopilotResponse wire contract, the exact refusal sentence and outcome vocabulary"
  - phase: 01-profile-foundation-and-policy-gate (plan 06)
    provides: "scripts/verify-phase1.sh structural template: argument parsing, tee-to-report header, _get_field jq/grep-fallback helper, fixture-array self-test pattern, appended go test capture"
provides:
  - "scripts/verify-phase2.sh -- a single command that drives the eleven-step Phase 2 HTTP sequence (session/autopilot on/off/status, session/status, policy/accept) against any running server or a fixture set, writing a report with one PASS/FAIL/SKIP line per ROADMAP criterion C1..C4, the request/response body under every step, and (live mode) the full go test ./... -v output appended with its own exit status"
  - "scripts/fixtures/phase2/ -- 11 fixture files letting --self-test exercise every assertion with no server"
  - "scripts/fixtures/phase2-negative/ -- the same 11 files with one deliberately wrong response (pre-acceptance #AUTO ON coming back engaged/on), letting --self-test-negative prove the harness FAILs and exits non-zero"
affects: [02-07-staging-verification-and-evidence]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Shared-assertion-logic self-test: _http() is the only function that differs between live and fixture mode; every assertion, print, counter and exit-code line downstream is byte-identical in both modes (same pattern established by scripts/verify-phase1.sh)"
    - "Criterion-labelled assertions with a distinct skip channel: _check/_check_eq/_check_status tag PASS/FAIL against a ROADMAP criterion label; a new _skip helper tags SKIP separately so an unreachable criterion is never counted as passed (T-2-15)"

key-files:
  created:
    - scripts/verify-phase2.sh
    - scripts/fixtures/phase2/ (11 files)
    - scripts/fixtures/phase2-negative/ (11 files)
  modified: []

key-decisions:
  - "The complete script -- including --self-test/--self-test-negative flags and the appended go test ./... -v capture -- was authored in task 1's commit rather than split incrementally across the two task commits, because the plan's read_first instructed treating verify-phase1.sh as a structural copy and building the full seam (the _http fixture-vs-curl branch) at once was simpler and less error-prone than retrofitting it. Task 2's commit therefore covers only the fixture directories that exercise that seam. No behavior or acceptance criterion differs from a strictly incremental split; both tasks' acceptance criteria are independently verified against the final state."
  - "policy/accept's response is printed through a truncating _print_policy_response (mirroring phase1's T-1-11 pattern) rather than the plain _print_response, because AcceptPolicy's PolicyResponse carries the full policy prose in its text field -- printing it whole would violate this plan's own T-2-16 mitigation (\"no game text or profile prose\" in the report)."
  - "NOT_OWNED_CONNECTION_ID defaults to a fixed literal UUID (ffffffff-ffff-ffff-ffff-ffffffffffff) rather than a randomly generated one, so self-test fixtures and the live run against a real server both exercise the identical, reproducible id the plan specifies as indistinguishable from an unowned connection on the wire (T-2-02)."

requirements-completed: [REQ-autopilot-directives, REQ-no-auto-reconnect, REQ-doc-hand-play-and-gate]

# Metrics
duration: ~35min
completed: 2026-09-15
---

# Phase 2 Plan 5: Phase 2 Canned-Report Harness Summary

**`scripts/verify-phase2.sh` drives the eleven-step HTTP-reachable half of the autopilot switch (gate refusal before acceptance, the no-connected-game refusal after acceptance, the already-off no-op, the status endpoint, the not-owned-connection IDOR check, invalid-action rejection, and status/switch agreement after the fact) against any server or a fixture set, writing a report with a PASS/FAIL/SKIP line per ROADMAP criterion C1..C4, every response body underneath, and — in live mode — the full `go test ./... -v` output folded into the same file; proven correct by a zero-FAIL self-test (22 checks, 2 skips) and a two-FAIL negative self-test.**

## Performance

- **Duration:** ~35 min
- **Started:** 2026-09-15T20:10:00Z (approximate, session start)
- **Completed:** 2026-09-15T20:45:19Z
- **Tasks:** 2 completed
- **Files modified:** 23 (1 script, 22 fixture files)

## Accomplishments

- `scripts/verify-phase2.sh` (535 lines) drives the exact eleven-step sequence specified in the plan against `${BASE_URL}/api/v1/...` using a session cookie: fresh status, pre-acceptance status/on refusal, unchanged status, the Phase 1 policy/accept endpoint used to move the fixture profile to accepted, post-acceptance on-refused-no-session (D-03), post-acceptance status (the two halves of `#AUTO STATUS`), already-off no-op (D-04), not-owned-connection refusal (T-2-02, IDOR), invalid-action HTTP 400, and a closing status confirming agreement with the switch endpoint (D-10).
- Every assertion is labelled with its ROADMAP criterion (C1 or C4) via `_check`/`_check_eq`/`_check_status`; C2 (wheel-grab) and C3 (disconnect/waiting/resume) are printed via a new `_skip` helper that increments a separate counter, never a pass or fail (T-2-15), each naming the specific `evidence/` screenshots and `[AI-PLAYER]` log excerpt that prove them instead.
- `_get_field` reads JSON fields through `jq` when present, falling back to `grep -oE`/`sed` otherwise (no new dependency); `_http` performs the live `curl` call or, in `--self-test`/`--self-test-negative` mode, reads the next fixture file from an ordered eleven-entry list — every line downstream of that call is shared between both modes.
- `SESSION_COOKIE` is used only inside the two `curl` invocations, never echoed or printed; the policy/accept response's `text` field is truncated to its first 80 characters before printing (T-2-16), mirroring Phase 1's `_print_policy_response` pattern.
- `--self-test` against `scripts/fixtures/phase2/` (11 files, one correct server response per HTTP call) passes with 22/22 checks and zero `FAIL` lines, exit 0, both `SKIP C2`/`SKIP C3` lines present naming their evidence files. `--self-test-negative` against `scripts/fixtures/phase2-negative/` (identical except the pre-acceptance `#AUTO ON` answer comes back `engaged`/`on` instead of `refused-gate`/`off`) fails exactly two `FAIL C1` checks and exits 1 — the harness is demonstrably falsifiable (T-2-07).
- Live mode appends `go test ./... -v` to the same report with a `GO TEST EXIT: n` line, folding its exit status into the script's own exit code; both self-test modes and `--no-tests` skip it and state the reason in the report.

## Task Commits

Each task was committed atomically:

1. **Task 02-05-01: One command drives the Phase 2 sequence and writes a criterion-labelled report** - `d30d1e6` (feat)
2. **Task 02-05-02: The harness proves itself with fixtures, proves it can fail, and carries the Go suite into the same report** - `4ccb80d` (test)

**Plan metadata:** (this commit)

## Files Created/Modified

- `scripts/verify-phase2.sh` - the harness: argument/flag parsing (`--self-test`, `--self-test-negative`, `--no-tests`), `_get_field`, `_http`, `_check`/`_check_eq`/`_check_status`/`_skip`, the eleven steps plus two skip steps, the summary block, the appended Go suite
- `scripts/fixtures/phase2/01..11-*.json` - one fixture per HTTP call across the eleven steps, all correct-server responses
- `scripts/fixtures/phase2-negative/01..11-*.json` - the same set with file `03-on-refused-gate.json` changed to `outcome":"engaged","state":"on"` for the pre-acceptance `#AUTO ON` call

## Decisions Made

- The complete script (including both self-test flags and the Go test append) was authored in task 1's commit; task 2's commit covers the fixture directories that exercise the already-built seam. Both tasks' acceptance criteria are independently satisfied against the final committed state — see key-decisions above for the full rationale.
- `policy/accept`'s response goes through a truncating print helper (T-2-16), matching Phase 1's precedent for the same endpoint's prose field.
- `NOT_OWNED_CONNECTION_ID` defaults to a fixed literal UUID rather than a random one, for reproducibility across self-test and live runs.

## Deviations from Plan

### Auto-fixed Issues

None — no bugs, missing critical functionality, or blocking issues were encountered during implementation.

### Process Note (not a Rule 1-4 deviation)

**Task-commit boundary shifted:** the plan's task breakdown implies task 1 creates the base eleven-step live-mode script and task 2 incrementally adds `--self-test`/`--self-test-negative`/Go-suite-append on top of it. This executor instead wrote the complete script (including the fixture-vs-curl `_http` seam and both self-test flags) in task 1's commit, then added only the fixture directories in task 2's commit. This is a commit-sequencing choice, not a scope, correctness, or acceptance-criteria change: every acceptance criterion listed under both task 1 and task 2 in the plan was independently re-verified against the final committed state (see Self-Check below) and all pass. Flagged here for transparency, not as a Rule 1-4 auto-fix.

---

**Total deviations:** 0 auto-fixed. **Impact on plan:** None — plan executed as specified; only the task-commit boundary for otherwise-identical final content differs from a strictly literal reading of the two task descriptions.

## Issues Encountered

None. `jq` is not installed in this execution environment, so every self-test and negative self-test run in this session exercised the `grep`/`sed` fallback path of `_get_field`, not the `jq` path — matching Phase 1's own noted coverage gap; both paths are exercised by the same call sites, so this is adequate coverage, but a reviewer with `jq` installed should note the `jq` path itself was not separately re-run here.

## User Setup Required

None for this plan. Running the harness live (plan 02-07's job) requires `BASE_URL`, `SESSION_COOKIE`, `CONNECTION_ID` and optionally `NOT_OWNED_CONNECTION_ID` — no owner action needed to produce those beyond what plan 02-07 already does.

## Live Run Status

Per this plan's own objective ("Plan 02-07 runs this script against Railway staging and files its output as `evidence/03-canned-report.txt`"), no live server was exercised in this environment. The live invocation was therefore **not** run here; a `bash -n` syntax check, `test -x`, and the full self-test / negative-self-test cycle (which exercises 100% of the assertion logic against fixtures, per this plan's `<verification>` items 1-3) were run instead. Item 4 (the live run against Railway staging) is explicitly deferred to plan 02-07, exactly as this plan's own objective section states.

## Next Phase Readiness

- Plan 02-07 can run `scripts/verify-phase2.sh` directly against Railway staging with no further setup; the four invocations (live, `--self-test`, `--self-test-negative`, `--no-tests`) are documented in the script's own header comment.
- The known pre-existing baseline failure `go test ./internal/icm/... -run TestHandlerRegistration/CANCEL` will surface as a non-zero `GO TEST EXIT` in any live run's report until a future phase fixes it; this is expected, out of scope for this plan, and does not indicate a Phase 2 regression (C1/C4 are evaluated independently of the Go suite's exit code within the report, even though both roll into the script's own final exit code).
- C2 and C3 remain SKIP by design in every mode this script can run; plan 02-07 is responsible for the screenshots and `[AI-PLAYER]` log excerpt that prove them, exactly as the SKIP lines name.

No blockers.

---
*Phase: 02-autopilot-switch*
*Completed: 2026-09-15*

## Self-Check: PASSED

- FOUND: scripts/verify-phase2.sh
- FOUND: scripts/fixtures/phase2/ (11 files)
- FOUND: scripts/fixtures/phase2-negative/ (11 files)
- FOUND: commit d30d1e6 (Task 1)
- FOUND: commit 4ccb80d (Task 2)
- `bash -n scripts/verify-phase2.sh` clean; `test -x scripts/verify-phase2.sh` exit 0
- `bash scripts/verify-phase2.sh --self-test /tmp/phase2-selftest.txt` exit 0; 0 `FAIL` lines; 16 `PASS C1` lines; 6 `PASS C4` lines; `SKIP C2`/`SKIP C3` present, each naming `evidence/` files
- `bash scripts/verify-phase2.sh --self-test-negative /tmp/phase2-negative.txt` exit 1; 2 `FAIL C1` lines (T-2-07)
- `diff -rq scripts/fixtures/phase2 scripts/fixtures/phase2-negative` shows exactly one differing file (`03-on-refused-gate.json`)
- `git diff --stat go.mod frontend/package.json` empty (T-2-SC)
- `grep -ci 'psql\|SELECT \|information_schema' scripts/verify-phase2.sh` = 0
- `git ls-files -s scripts/verify-phase2.sh` shows mode `100755`
