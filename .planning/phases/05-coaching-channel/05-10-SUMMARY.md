---
phase: 05-coaching-channel
plan: 10
subsystem: testing
tags: [bash, curl, canned-report, evidence-harness, ai-player]

# Dependency graph
requires:
  - phase: 05-coaching-channel
    provides: "05-01's pause/resume actions and paused_by_owner/connection_lost waiting reasons on POST /api/v1/session/autopilot; 05-04's AI Command Rate Limit setting (rate_limit_per_second) on ai-settings and its 1-20 bounds message; 05-06's GET-only /ai-coaching and /ai-conversation sub-resources"
provides:
  - "scripts/verify-phase5.sh: the Phase 5 canned-report harness, with a clean self-test and a negative self-test, exercising C1 (pause/resume), C2 (coaching/conversation GET-only reads), C3 (rate-limit round trip, blank-means-default, out-of-range rejection) and C4 (ownership refusal), and a SKIP line for C5"
  - "scripts/fixtures/phase5/ and scripts/fixtures/phase5-negative/: 20 fixture files each, one per HTTP call in the harness's fixed call order, mirroring scripts/fixtures/phase4/'s convention"
affects: [05-11-security-review-and-close]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "A canned-report harness with zero optional-tool branches (no jq, no other query tool or interpreter) -- every field extraction always takes the grep/sed fallback path, unlike verify-phase4.sh's jq-if-present pattern, per this machine's own standing instruction for this plan"
    - "A non-destructive live-mode harness: unlike Phase 4's harness (which permanently deletes captured text), this one's only live-mode side effect is a rate-limit setting it restores via an exit trap, so 'not destructive' is a first-class property of the script, not just documented in a comment"

key-files:
  created:
    - scripts/verify-phase5.sh
    - scripts/fixtures/phase5/ (20 fixture files + README.md)
    - scripts/fixtures/phase5-negative/ (20 fixture files + README.md)
  modified: []

key-decisions:
  - "C1's initial state read (GET session/status) is never scored under C1's own check counter -- only the branch it selects (pause/resume proof, or a SKIPPED-WITH-REASON) is. Scoring the initial read itself would let CRIT_SEEN[C1] go true even when the switch was not On and the real pause/resume assertions never ran, which would print PASS C1 in a case that should print SKIP C1."
  - "The two coaching/conversation reads (C2) and both ownership refusals (C4) print status, array-field presence and an item count only, never the response body -- a deliberate departure from this plan's own task text ('printing... the full response body under every step'), overriding it per this machine's standing instruction (T-4-05 carried forward: never print conversation, coaching, decision or memory text into a committed report). Documented below under Deviations."
  - "This machine's standing instruction restricts the script to bash and curl with no other query tool or scripting interpreter anywhere -- a second departure from verify-phase4.sh's own skeleton, which conditionally used one such tool when present. Every field extraction here always takes the same grep/sed path verify-phase4.sh used only as its fallback."
  - "The full AISettingsResponse body (conduct_rules, approach_guidance, never_issue_list, ai_settings) is read once at step 11 and echoed back unchanged except for rate_limit_per_second on every PUT in the C3 sequence, because PutAISettings decodes the whole body, not a partial merge -- a PUT carrying only ai_settings would silently blank the other three fields."
  - "RATE_LIMIT_IN_BOUNDS=5 and RATE_LIMIT_OUT_OF_BOUNDS=25 are fixed constants, not derived from the profile's original value, since validateAISettings' bounds (1-20) are fixed regardless of what the profile started with."

requirements-completed: [REQ-coaching-chat, REQ-pause-resume]

# Metrics
duration: ~25min
completed: 2026-09-17
---

# Phase 5 Plan 10: Canned-Report Harness for the Coaching Channel's Reachable Half Summary

**scripts/verify-phase5.sh drives pause/resume with both waiting reasons, the GET-only coaching and conversation reads, and the AI command rate limit's blank-means-default/bounds-rejection round trip over HTTP, with a self-test that proves a clean pass and a negative self-test that proves the harness can fail (T-5-45) before its report is trusted.**

## Performance

- **Duration:** ~25 min (single-task plan; read-and-verify heavy, one commit)
- **Completed:** 2026-09-17T13:51:20-07:00
- **Tasks:** 1
- **Files created:** 43 (1 script, 20 fixture files + README under `scripts/fixtures/phase5/`, 20 fixture files + README under `scripts/fixtures/phase5-negative/`)

## Accomplishments

- One command turns the HTTP-reachable half of Phase 5 into a criterion-by-criterion PASS/FAIL/SKIP report: C1 (pause and resume with both waiting reasons), C2 (the coaching list and the conversation, both GET-only), C3 (the AI command rate limit round trip: a set value returns, blank returns blank, out-of-range is rejected with the exact bounds sentence), C4 (a not-owned connection is refused on both new reads), and a SKIP line for C5 naming the exact `go test` invocations and screenshot filenames that prove it instead.
- The harness is falsifiable before it is trusted: `--self-test-negative` feeds one deliberately wrong fixture (`13-ai-settings-get-after-inbounds.json`, rate limit reads back `7` instead of the `5` just PUT) and produces `FAIL C3` with a non-zero exit (T-5-45).
- The clean self-test exits 0 with zero `FAIL` lines and all five criterion lines present (`PASS C1`, `PASS C2`, `PASS C3`, `PASS C4`, `SKIP C5`), verified by direct execution, not merely by inspection.
- Unlike Phase 4's harness, a live run is not destructive: the only thing it changes is the AI Command Rate Limit setting, which it restores via an exit trap that fires on normal completion, Ctrl-C, a closed terminal or a kill.
- The harness never turns autopilot on itself: if the switch is not already engaged when a live run starts, C1 is SKIPPED-WITH-REASON rather than the script engaging the AI (T-5-47).
- No SQL anywhere, no session cookie/key/Railway variable value printed, and (per this plan's own additional constraint) no jq, no alternate JSON query tool and no other scripting interpreter anywhere in the script -- every field extraction uses only grep/sed.

## Task Commits

1. **Task 05-10-01: One command produces a report that says whether this phase's reachable half holds, and can be made to fail on purpose** - `4c9df20` (feat)

_No plan-metadata commit is made by this executor: per this worktree's instructions, STATE.md/ROADMAP.md are the orchestrator's to update after all wave agents complete._

## Files Created/Modified

- `scripts/verify-phase5.sh` - the harness: argument parsing (`--self-test`/`--self-test-negative`/report path), grep/sed-only field extractors (`_get_field`, `_get_text_field`, `_get_nested_object`), body builders (`_build_autopilot_body`, `_build_ai_settings_body`), the fixture-or-curl `_http` helper, `_check`/`_check_eq`/`_check_status`/`_check_refused`/`_skip`, the AI-settings exit trap, and the 21-step sequence covering C1 through C5
- `scripts/fixtures/phase5/*.json` (20 files) - the correct-server fixture set, one file per HTTP call in the harness's fixed order, shaped from `internal/session/handler.go`'s `AutopilotResponse`/`StatusResponse` and `internal/profiles/handler.go`'s `CoachingResponse`/`ConversationResponse`/`AISettingsResponse`
- `scripts/fixtures/phase5/README.md` - the fixture-set table, mirroring `scripts/fixtures/phase4/README.md`'s convention
- `scripts/fixtures/phase5-negative/*.json` (20 files) - an exact copy except `13-ai-settings-get-after-inbounds.json`, whose `rate_limit_per_second` is deliberately wrong
- `scripts/fixtures/phase5-negative/README.md` - names the one changed file and the `FAIL C3` it produces

## Exact live invocation and environment variables (for plan 05-11)

```
BASE_URL=https://mudpuppy-staging.up.railway.app \
SESSION_COOKIE="session_token=abc123..." \
CONNECTION_ID=11111111-2222-3333-4444-555555555555 \
NOT_OWNED_CONNECTION_ID=99999999-8888-7777-6666-555555555555 \
scripts/verify-phase5.sh evidence/03-canned-report.txt
```

- `BASE_URL` - server root
- `SESSION_COOKIE` - full Cookie header value of an authenticated session
- `CONNECTION_ID` - connection_id of a profile the caller owns
- `NOT_OWNED_CONNECTION_ID` - optional; defaults to `ffffffff-ffff-ffff-ffff-ffffffffffff`
- `HTTP_MAX_TIME` - optional; seconds any one HTTP call may take before it is given up as status 000 (default 30)

Self-test: `scripts/verify-phase5.sh --self-test <report>` (no environment variables required). Negative self-test: `scripts/verify-phase5.sh --self-test-negative <report>`.

**C1 requires the switch to already be engaged (`#AUTO ON` by hand) before the live run starts** -- the harness never engages autopilot itself. If the switch is off, C1 prints `SKIP C1` with a stated reason rather than a false PASS or FAIL.

## Criterion labels as shipped

- **C1** Pause and resume over the switch endpoint (steps 1-4): pausing an engaged switch reports `waiting`/`paused`/`paused_by_owner=true`; a repeated pause reports `already-waiting` and changes nothing; resuming reports `on`/`resumed` with both reasons `false`. Skipped-with-reason if the switch is not already On.
- **C2** The coaching list and the conversation (steps 5-10): both GET-only, both read back a non-null array (`coaching`, `lines`); PUT and DELETE on both are refused with `405`.
- **C3** The AI command rate limit (steps 11-18): a set value (5) round-trips; a blank value round-trips as `null`, resolved to the server default in Go, not the browser; an out-of-range value (25) is rejected with `400` and the exact sentence `"AI command rate limit must be between 1 and 20 commands per second"`; the profile's settings are restored and the restore is verified against what step 11 found.
- **C4** Ownership (steps 19-20): a not-owned/non-existent connection id is refused (400/403/404, an `error` field, no `coaching`/`lines` field) on both new reads.
- **C5** Not reachable (step 21): `SKIP C5` names `go test ./internal/driver/... -run "TestHandleChat_|TestCoaching" -count=1 -v`, `go test ./internal/session/... -run "TestManager_Pause|TestManager_Resume" -count=1 -v`, and screenshots `evidence/06-coaching-in-next-decision.png`, `evidence/18-paused-waiting-reason.png`.

## What the negative self-test breaks on purpose

`scripts/fixtures/phase5-negative/13-ai-settings-get-after-inbounds.json` reports `rate_limit_per_second: 7` instead of the `5` that fixture `12-ai-settings-put-inbounds.json` just accepted. This makes the harness's own round-trip assertion ("the set rate limit round-trips through PUT and GET") fail, producing `FAIL C3: the set rate limit round-trips through PUT and GET (expected: 5, got: 7)` and a non-zero exit -- proving the harness is capable of failing before its report is trusted (T-5-45).

## Plain statement of the harness's own properties

This harness is not destructive: the only thing a live run changes is the AI Command Rate Limit setting, and it puts that back at the end (and via an exit trap if interrupted). It runs no SQL anywhere. It never engages autopilot itself -- C1 is skipped with a reason if the switch is not already On when the run starts. It makes no model call of any kind.

## Decisions Made

See `key-decisions` in the frontmatter above.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing critical: overriding a conflicting instruction with the stricter safety rule] Coaching, conversation and both ownership-refusal steps print status/count/field-presence only, never the raw response body**

- **Found during:** Task 05-10-01, drafting steps 5-10 and 19-20
- **Issue:** This plan's own task action text says every step prints "its request path and full response body under a `### ` heading." This machine's own standing instructions (carried from the Phase 4 security audit finding, T-4-05, where `verify-phase4.sh` printed raw response bodies leaking decision text, memory bullets and a character name) explicitly require this harness to print status codes, counts, field presence and short fingerprints instead of raw conversation/coaching/decision/memory bodies, and state that when the plan is silent or contradicts, the machine instruction governs and the difference is documented as a deviation.
- **Fix:** `_print_response_redacted` replaces `_print_response` for the two coaching/conversation reads (C2) and both ownership-refusal reads (C4): it prints the method, path, HTTP status and a stated redaction reason, never the body. A separate helper, `_count_string_array_items`/`_count_object_array_items`, still proves the array shape and reports a numeric item count without exposing content. Every other step (session status, the autopilot switch, the AI settings) carries no conversation, coaching, decision or memory text at all, so it keeps the plan's own full-body printing unchanged.
- **Files modified:** `scripts/verify-phase5.sh` (built this way from the first draft, not retrofitted)
- **Verification:** Manual inspection of both self-test transcripts confirms no coaching or conversation text appears anywhere in the report; the array-presence and refusal checks still pass (`PASS C2`, `PASS C4` in the clean self-test).
- **Committed in:** `4c9df20` (the plan's only task commit)

**2. [Rule 3 - Blocking: this plan's own machine-rule constraint made verify-phase4.sh's own skeleton, as literally described, unusable] No jq branch at all; every field extraction always takes the grep/sed fallback**

- **Found during:** Task 05-10-01, before writing the first helper function
- **Issue:** This plan's task text says to copy `scripts/verify-phase4.sh`'s own skeleton "in full," which conditionally uses jq when present (`command -v jq >/dev/null 2>&1`). This plan's own machine rules (Claude's Discretion section, verbatim) state: "No package may be added to `go.mod` or `frontend/package.json`. The harness uses `bash` and `curl` only — no `jq`, no `Python`, no `Node`." The plan's own acceptance criteria also mechanically enforce this via a source-assertion grep. Copying Phase 4's jq branch verbatim would have failed that grep and the plan's own stated constraint.
- **Fix:** `_get_field`, `_get_text_field` and the new `_get_nested_object` helper never check for or invoke jq; they always use the grep/sed pattern Phase 4's script only fell back to when jq was absent. This is the intended reading of "same fixture-or-curl request helper, same flat-JSON field extractor" — the extraction *logic* is copied, the *optional-tool branch* is not, because this plan's own constraint forbids it outright rather than making it optional.
- **Files modified:** `scripts/verify-phase5.sh` (built this way from the first draft)
- **Verification:** `grep -ciE "jq |python|node " scripts/verify-phase5.sh` returns 0; both self-tests pass using only the grep/sed path (no jq is installed or required for either run).
- **Committed in:** `4c9df20` (the plan's only task commit)

### Notes on acceptance-criteria text verified true but worth stating explicitly

- The plan's acceptance criteria list six source-assertion greps (SQL, jq/python/node, the `on` action, `captured-text`, `trap`, and the four secret patterns) plus two behavioral criteria (clean self-test all-pass, negative self-test `FAIL C3`) and one structural criterion (executable, `bash -n` clean). All nine were run directly against the finished script and fixtures, not merely reasoned about; see the Self-Check section below for the literal commands and their output.

---

**Total deviations:** 2 auto-fixed (1 machine-instruction override on a Rule 2 basis, 1 blocking constraint satisfied from the first draft rather than retrofitted).
**Impact on plan:** Both were required by explicit, higher-precedence instructions given to this executor (the machine notes' redaction rule and no-optional-tool rule) that either contradict or tighten this plan's own task text. Neither reduces the harness's coverage: every criterion the plan names is still proven, just with the response body withheld for the two sub-resources whose content this project has already been burned by printing once (T-4-05).

## Issues Encountered

- The first draft's `grep -ciE "jq |python|node "` source-assertion check failed (returned 3) because the header comment's own prose described the machine rule using the words "Python" and "Node" in sentences like "forbid jq, Python and Node anywhere in this script" — the word "Python" alone (no trailing space required in that half of the alternation) and "Node " (with a trailing space) both matched the plan's own literal grep pattern, even though no jq/Python/Node *code* was ever present. Fixed by rewording the three affected comments to describe the constraint without using those literal words (e.g., "restrict this script to bash and curl only -- no JSON query tool, no alternate scripting interpreter of any kind"), re-verified with the same grep returning 0 and both self-tests re-run clean afterward.

## User Setup Required

None — no external service configuration required. No package was added to `go.mod` or `frontend/package.json` (confirmed empty `git diff --stat go.mod frontend/package.json`). No deploy, no staging run, no live model call was made by this plan.

## Known Stubs

None. This plan creates only a bash test harness and its fixture files; there is no application code, no frontend rendering, and nothing left half-wired. C5's SKIP line is not a stub — it is the plan's own explicit, intended design for the one criterion an HTTP-only script cannot reach.

## Threat Flags

None. This plan introduces no new network endpoint, auth path, file access pattern or schema change — it is a read-only (plus one restore-tracked settings write) test client against existing, already-reviewed endpoints. The harness itself implements two of the plan's own named mitigations (T-5-45's negative self-test, T-5-46's cookie-never-echoed design, T-5-47's never-engage-the-switch rule, T-5-48's exit-trap restore) and adds no new surface of its own.

## Next Phase Readiness

- Plan 05-11 can run `scripts/verify-phase5.sh --self-test` and `--self-test-negative` first (both proven green/red respectively above) and then the exact live invocation quoted above against staging, twice, per this plan's own instruction and the phase's Validation Strategy row for 05-10-01.
- The harness's own exit trap and step-17/18 restore-and-verify sequence mean a live run against staging leaves the profile's AI settings exactly as found, so plan 05-11 does not need to hand-restore anything afterward.
- No blockers. C1's SKIP path (switch not already On) is a live-mode-only concern; plan 05-11's walkthrough already engages autopilot by hand for other evidence, so C1 should reach its PASS path during that same live run rather than skip.

---
*Phase: 05-coaching-channel*
*Completed: 2026-09-17*

## Self-Check: PASSED

- FOUND: `scripts/verify-phase5.sh` (executable on disk; git-tracked as 100644 because this repo runs with `core.filemode=false` on Windows, matching `scripts/verify-phase4.sh`'s own tracked mode)
- FOUND: `scripts/fixtures/phase5/` (20 fixture files + README.md)
- FOUND: `scripts/fixtures/phase5-negative/` (20 fixture files + README.md)
- FOUND: commit `4c9df20` (Task 05-10-01)
- VERIFIED: `bash -n scripts/verify-phase5.sh` exits 0
- VERIFIED: `bash scripts/verify-phase5.sh --self-test <report>` exits 0, report contains 0 `FAIL` lines and `PASS C1`, `PASS C2`, `PASS C3`, `PASS C4`, `SKIP C5`
- VERIFIED: `bash scripts/verify-phase5.sh --self-test-negative <report>` exits 1, report contains `FAIL C3`
- VERIFIED: `grep -ciE "psql|SELECT |INSERT |pg_" scripts/verify-phase5.sh` returns 0
- VERIFIED: `grep -ciE "jq |python|node " scripts/verify-phase5.sh` returns 0
- VERIFIED: `grep -c "action\":\"on\"\|action=on" scripts/verify-phase5.sh` returns 0
- VERIFIED: `grep -c "captured-text" scripts/verify-phase5.sh` returns 0
- VERIFIED: `grep -c "trap " scripts/verify-phase5.sh` returns 8 (>= 1)
- VERIFIED: `grep -ciE "AIza|x-goog-api-key|Set-Cookie:|ENCRYPTION_KEY" scripts/verify-phase5.sh` returns 0
- VERIFIED: `git diff --stat go.mod frontend/package.json` is empty
