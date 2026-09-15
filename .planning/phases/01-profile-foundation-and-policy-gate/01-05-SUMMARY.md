---
phase: 01-profile-foundation-and-policy-gate
plan: 05
subsystem: testing
tags: [evidence, staging, railway, screenshots, canned-report, golang-migrate]

# Dependency graph
requires:
  - phase: 01-profile-foundation-and-policy-gate (plan 01)
    provides: "Migration 010, store.ResolveAISettings, store.EngageGateAllowed, the startup log lines"
  - phase: 01-profile-foundation-and-policy-gate (plan 02)
    provides: "internal/policy.Text() and Version() == 1.0"
  - phase: 01-profile-foundation-and-policy-gate (plan 03)
    provides: "Five HTTP endpoints and the four [AI-PLAYER] log lines"
  - phase: 01-profile-foundation-and-policy-gate (plan 04)
    provides: "The AI Player browser panel (policy gate, then editor)"
  - phase: 01-profile-foundation-and-policy-gate (plan 06)
    provides: "scripts/verify-phase1.sh canned-report harness"
provides:
  - "evidence/01-test-report.txt — verbatim capture of the twelve-command diagnostic gate (go build, go test -v, policy diff, six targeted tests, harness self-test, npm run build, dependency-drift check)"
  - "evidence/02-staging-startup.log — Railway staging startup excerpt proving migration 010 and the AI Player column fallback confirmation"
  - "evidence/03-canned-report.txt — scripts/verify-phase1.sh run against staging, PASS for C1-C4, 29/29 checks, response bodies printed per step, plus the bundled go test ./... -v output"
  - "evidence/04-staging-ai-player.log — the seven [AI-PLAYER] lines for one staging connection: gate refusal, two blank-settings saves, acceptance, repeat-acceptance, gate allow, populated-settings save"
  - "evidence/05-policy-first.png through evidence/11-recreated-profile-policy-again.png — seven end-user screenshots covering every player-observable state in ROADMAP Phase 1's Phase Validation line"
  - "01-05-SUMMARY.md — the four-row ROADMAP criterion table with line-numbered citations, closing Phase 1"
affects: [02 (autopilot engage/disengage), all later phases citing Phase 1 as demonstrated]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Evidence-as-file: every ROADMAP success criterion is proven by a citable file on disk (report line, log line, or screenshot), never by a database query or by prose alone"
    - ".gitattributes text eol=lf pin for go:embed-sourced markdown, to survive Windows core.autocrlf checkouts"

key-files:
  created:
    - .planning/phases/01-profile-foundation-and-policy-gate/evidence/01-test-report.txt
    - .planning/phases/01-profile-foundation-and-policy-gate/evidence/02-staging-startup.log
    - .planning/phases/01-profile-foundation-and-policy-gate/evidence/03-canned-report.txt
    - .planning/phases/01-profile-foundation-and-policy-gate/evidence/04-staging-ai-player.log
    - .planning/phases/01-profile-foundation-and-policy-gate/evidence/05-policy-first.png
    - .planning/phases/01-profile-foundation-and-policy-gate/evidence/06-accepted-line.png
    - .planning/phases/01-profile-foundation-and-policy-gate/evidence/07-values-after-reload.png
    - .planning/phases/01-profile-foundation-and-policy-gate/evidence/08-values-new-session.png
    - .planning/phases/01-profile-foundation-and-policy-gate/evidence/09-blank-fields.png
    - .planning/phases/01-profile-foundation-and-policy-gate/evidence/10-after-timers-save.png
    - .planning/phases/01-profile-foundation-and-policy-gate/evidence/11-recreated-profile-policy-again.png
    - .planning/phases/01-profile-foundation-and-policy-gate/01-05-SUMMARY.md
  modified:
    - .gitattributes
    - internal/policy/safety-and-abuse-policy-v1.md (CRLF -> LF fix)

key-decisions:
  - "Logs were captured with the Railway CLI (railway logs --environment staging), named explicitly per task 01-05-02's acceptance criterion, not the MCP tool or the dashboard pane"
  - "Migration 010's columns were applied by the golang-migrate step itself (evidence/02-staging-startup.log line 2: 'Migrations completed successfully (version=10, dirty=false)'); the AI Player column-ensure fallback (lines 5-6) ran afterward and reported the columns already present, confirming rather than repairing the schema — this settles RESEARCH Open Question 1 from the log, not a query"
  - "Screenshots were captured by the executor with a Chrome-window screen capture (full browser chrome: tab bar, URL bar, window frame) rather than by the owner clicking through by hand, per the plan's 'either is acceptable' allowance; the owner confirmed the resulting set at the checkpoint"
  - "The CRLF divergence in internal/policy/safety-and-abuse-policy-v1.md (found during task 01-05-01's diff-gate step) was fixed inline as a Rule 1 bug and pinned with a .gitattributes rule so this Windows checkout cannot reintroduce it on a future clone"

patterns-established:
  - "Four-row ROADMAP-criterion-to-evidence table as the shape of a phase's closing SUMMARY, with mandatory line numbers for text citations and filenames for screenshots"

requirements-completed: [REQ-profile-ai-fields, REQ-policy-gate]

# Metrics
duration: 61min
completed: 2026-09-15
---

# Phase 1 Plan 5: Evidence Filing Summary

**Phase 1's four ROADMAP success criteria are each proven by a named, line-numbered file on disk: a verbatim test report, two staging log excerpts, one canned HTTP report, and seven end-user screenshots — zero database queries anywhere in the evidence.**

## Performance

- **Duration:** 61 min (09:19:52-07:00 first task commit to 17:20 UTC SUMMARY authoring; continuation agent finished the last task)
- **Started:** 2026-09-15T16:19:52Z
- **Completed:** 2026-09-15T17:20:07Z (approx.)
- **Tasks:** 3 (2 checkpoints)
- **Files modified:** 13 (11 evidence files + .gitattributes + internal/policy/safety-and-abuse-policy-v1.md CRLF fix, across 3 task commits)

## Accomplishments

- Captured the full diagnostic gate (`go build`, `go test ./... -v`, the policy embed diff, six targeted test runs, the harness self-test, `npm run build`, dependency-drift check) verbatim into `evidence/01-test-report.txt` with zero new FAIL lines and an empty dependency-drift section
- Deployed to Railway staging and pulled three evidence files with no database access: the startup log proving migration 010 and the AI Player column fallback, the canned `scripts/verify-phase1.sh` report showing PASS for C1-C4 with 29/29 checks and every response body printed, and the filtered `[AI-PLAYER]` log excerpt showing the accept/already-accepted pair and both gate decisions for one connection
- Walked the AI Player section on staging end to end (policy-first, accept, populate, hard reload, new session, blank fields, unrelated timers save, delete-and-recreate) and captured one screenshot per observable state
- Filed the four-row ROADMAP criterion table below, closing Phase 1's Phase Validation line

## Task Commits

Each task was committed atomically:

1. **Task 01-05-01: The test report is captured verbatim as a file** - `4e2e3b0` (test)
2. **Task 01-05-02: Staging proves itself in its own logs and in the canned report** - `950c539` (test, owner-approved at checkpoint)
3. **Task 01-05-03: The owner sees it work, and each observable is a screenshot on disk** - screenshots + SUMMARY, this commit (docs)

**Plan metadata:** tracking commit follows this SUMMARY commit.

## Files Created/Modified

- `.planning/phases/01-profile-foundation-and-policy-gate/evidence/01-test-report.txt` - verbatim twelve-command diagnostic capture
- `.planning/phases/01-profile-foundation-and-policy-gate/evidence/02-staging-startup.log` - staging startup excerpt (migration 010, AI Player columns ensured)
- `.planning/phases/01-profile-foundation-and-policy-gate/evidence/03-canned-report.txt` - `scripts/verify-phase1.sh` run against staging (PASS C1-C4, 29/29 checks)
- `.planning/phases/01-profile-foundation-and-policy-gate/evidence/04-staging-ai-player.log` - filtered `[AI-PLAYER]` lines for the test connection
- `.planning/phases/01-profile-foundation-and-policy-gate/evidence/05-policy-first.png` through `11-recreated-profile-policy-again.png` - seven end-user screenshots
- `.gitattributes` - `internal/policy/*.md text eol=lf` rule added to pin LF endings on the go:embed-sourced policy document
- `internal/policy/safety-and-abuse-policy-v1.md` - CRLF stripped back to LF to match `.specify/specs/safety-and-abuse-policy-v1.md` byte for byte

## Evidence Inventory

| File | Produced by | Proves |
|------|-------------|--------|
| `evidence/01-test-report.txt` | Task 01-05-01 | Diagnostic gate: `go build`, full `go test -v`, the policy diff gate, `npm run build`, dependency-drift — all green, no package added |
| `evidence/02-staging-startup.log` | Task 01-05-02 | Migration 010 live on staging, AI Player columns present |
| `evidence/03-canned-report.txt` | Task 01-05-02 (`scripts/verify-phase1.sh`) | All four ROADMAP criteria hold over HTTP against staging, with response bodies |
| `evidence/04-staging-ai-player.log` | Task 01-05-02 | Acceptance recorded once, both engage-gate decisions, on the strength of stored state alone |
| `evidence/05-policy-first.png` … `evidence/11-recreated-profile-policy-again.png` | Task 01-05-03 | Every player-observable state in the ROADMAP Phase Validation line |

**Log capture tool:** Railway CLI, `railway logs --environment staging` (filtered on `[AI-PLAYER]` for `04-staging-ai-player.log`; unfiltered, excerpted from `Running database migrations` through `AI Player columns ensured`, for `02-staging-startup.log`).

**Migration mechanism (RESEARCH Open Question 1, closed):** the golang-migrate step itself applied migration 010 — `evidence/02-staging-startup.log` line 2 reads `Migrations completed successfully (version=10, dirty=false)`. The separate "AI Player columns ensured" fallback (lines 5-6) ran immediately after and reported the columns already present; it *confirmed* the schema rather than repairing it. Both facts come from the startup log, not a query.

**Staging test data created during 01-05-02:** a throwaway staging test account with two connection profiles — one taken through the full accept/edit/reload/session/blank/timers sequence (`connection_id=563b3448-066d-40b0-9e61-3408a5085a46`, per `evidence/03-canned-report.txt` and `evidence/04-staging-ai-player.log`), the other used only to prove the pre-acceptance refusal state before being superseded by the first. Neither is a production account.

**Owner walkthrough connection (01-05-03):** a separate connection named "Evidence Walkthrough (Alter Aeon)", created under the owner's own staging login, used for the seven screenshots. It was deleted and recreated once (screenshot 11) to prove a fresh profile for the same game re-asks the policy; the recreated profile exists on staging today with the policy not yet accepted.

**Screenshot capture method:** all seven PNGs were captured by the executor as full Chrome-window screen captures (tab bar, URL bar, and window frame all visible, confirming end-user context per the plan's requirement), not taken by the owner clicking through by hand. The plan's `<action>` explicitly allows either method; the owner confirmed the resulting set of seven at the checkpoint. A Chrome automation info bar ("Claude started debugging this browser") is visible at the top of the browser window in files 06 through 11 — this is Chrome's own automation banner, not a devtools pane, and does not violate the "no devtools" acceptance criterion.

## Phase Validation

| Criterion | What it claims | Evidence | Verdict |
|-----------|----------------|----------|---------|
| 1 | Blank model name / call cap / threshold resolve to server default / no cap / built-in error handling; over-length conduct rules rejected | `evidence/01-test-report.txt` lines 293-302 and 332-342 (`TestResolveAISettings`, all four subtests PASS); `evidence/03-canned-report.txt` lines 28-31 (blank round-trip: empty model_name, null call_cap, null disengage_threshold) and lines 36-37 (over-length rejected HTTP 400 with the exact error string) and line 93 (`C1: PASS`); `evidence/09-blank-fields.png` (three blank fields with hint text, no substituted zeros) | PASS |
| 2 | Owner can read/edit conduct rules, approach guidance and AI settings from the browser; values survive a hard reload and a new login session | `evidence/03-canned-report.txt` lines 71-75 (round-trip identical) and lines 84-88 (unchanged after an unrelated timers save) and line 94 (`C2: PASS`); `evidence/07-values-after-reload.png`, `evidence/08-values-new-session.png`, `evidence/10-after-timers-save.png` | PASS |
| 3 | Policy presented first, editor only unlocks after acceptance, engage gate refuses then allows on stored acceptance alone | `evidence/05-policy-first.png`, `evidence/06-accepted-line.png`; `evidence/03-canned-report.txt` lines 14, 20-21, 63-64 (C3 refusal-then-allow with the exact Phase 2 contract string) and line 95 (`C3: PASS`); `evidence/04-staging-ai-player.log` line 1 (`allowed=false`) and line 6 (`allowed=true`), same `connection_id` | PASS |
| 4 | Acceptance recorded once with version 1.0 and a timestamp; a recreated profile is asked again | `evidence/03-canned-report.txt` lines 44-45 (forged acceptance rejected), 50-52 (first accept), 57-58 (repeat accept echoes the same `accepted_at`), line 96 (`C4: PASS`); `evidence/04-staging-ai-player.log` line 4 (`policy accepted ... version=1.0 accepted_at=2026-09-15T16:31:38.095576Z`) and line 5 (`policy already accepted` with the identical timestamp); `evidence/11-recreated-profile-policy-again.png` (fresh profile for the same game shows the policy again) | PASS |

Migration 010 live on staging (a precondition for all four criteria, not a criterion itself): `evidence/02-staging-startup.log` line 2 (`Migrations completed successfully (version=10, dirty=false)`) and line 6 (`AI Player columns ensured`).

## Decisions Made

- Log capture tool named explicitly per acceptance criteria: Railway CLI (`railway logs --environment staging`), not the MCP log tool or the dashboard pane.
- Migration mechanism named explicitly: the golang-migrate step applied the schema; the fallback only confirmed it — closing RESEARCH Open Question 1 from the log.
- Screenshots captured by the executor's own browser-automation tool as full Chrome-window captures rather than by the owner's hand, using the plan's explicit "either is acceptable" allowance; owner confirmed the set at the checkpoint.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed CRLF divergence in the embedded policy document**
- **Found during:** Task 01-05-01, step 3 (`diff -u` between the approved source and the embedded copy)
- **Issue:** This Windows checkout's `core.autocrlf=true` had silently converted `internal/policy/safety-and-abuse-policy-v1.md` (a `go:embed`-sourced file served verbatim to end users) to CRLF line endings on checkout, while the approved source `.specify/specs/safety-and-abuse-policy-v1.md` remained LF — a byte-for-byte divergence between the approved policy text and what the server actually embeds and serves.
- **Fix:** Stripped the CR characters (`sed -i 's/\r$//'`) and added a `.gitattributes` rule (`internal/policy/*.md text eol=lf`) so a future checkout on any machine cannot reintroduce the divergence.
- **Files modified:** `.gitattributes`, `internal/policy/safety-and-abuse-policy-v1.md`
- **Verification:** The `diff -u` step re-run with empty output, confirmed byte-identical; recorded as the passing result in `evidence/01-test-report.txt`.
- **Committed in:** `4e2e3b0` (Task 01-05-01 commit)

---

**Total deviations:** 1 auto-fixed (1 Rule 1 bug fix)
**Impact on plan:** Necessary for correctness — the policy actually served to end users must match the approved source exactly. No scope creep.

## Issues Encountered

None beyond the CRLF deviation above, which was resolved inline before the diff-gate step completed.

## Known Stubs

None. All evidence files and screenshots reflect real staging behavior against a live deployment; no placeholder values, mock data, or unwired UI states are present in this plan's output.

## Threat Flags

None. All work in this plan is evidence capture against surfaces built and threat-modeled in plans 01-01 through 01-04 and 01-06; no new network endpoint, auth path, file-access pattern, or schema change was introduced.

## User Setup Required

None - no external service configuration required beyond the existing Railway staging deployment used throughout Phase 1.

## Next Phase Readiness

Phase 1's ROADMAP Phase Validation line has been demonstrated end to end on staging with no database queries anywhere in the evidence trail. All four success criteria carry a PASS verdict with a citable file. Phase 2 (autopilot engage/disengage) can proceed: it depends on the engage gate proven here (`evidence/04-staging-ai-player.log` lines 1 and 6) and the stored acceptance record (`evidence/04-staging-ai-player.log` lines 4-5).

No blockers. The recreated "Evidence Walkthrough (Alter Aeon)" connection on staging has not accepted the policy and can be reused or discarded freely by a later phase's evidence work.

---
*Phase: 01-profile-foundation-and-policy-gate*
*Completed: 2026-09-15*
