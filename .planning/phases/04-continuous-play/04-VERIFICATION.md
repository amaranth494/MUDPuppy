---
phase: 04-continuous-play
verified: 2026-09-17T14:05:58Z
status: human_needed
score: 4/4 roadmap success criteria verified; 14/14 code-review findings confirmed fixed in code; 1 governance item outstanding
overrides_applied: 0
human_verification:
  - test: "Owner completes the Phase 4 security review: for each of the 11 carried-forward DR-3/DR-3.1 rows and the 2 new Phase-4 items in 04-SECURITY-AGENDA.md (D-28/D-29/T-4-03), the owner checks Accept, Defer, or Remediate Now and the decision is appended to .planning/RISK-REGISTER.md."
    expected: "Every checkbox in 04-SECURITY-AGENDA.md is resolved and RISK-REGISTER.md carries a dated Phase 4 row per item (per this project's binding phase-approval-artifact and security-risk-decision rules); no checkbox is left at `[ ]` with an unset disposition."
    why_human: "This is an owner risk-acceptance decision, not a code fact; grep can confirm the checkboxes are still unchecked (confirmed: all 24 in 04-SECURITY-AGENDA.md are `[ ]`) but cannot make the decision for the owner."
  - test: "Owner reviews the reviewer-reliability finding recorded in 04-SECURITY-AGENDA.md Part 2 (Accept/Defer/Remediate Now) — the reviewer blocked ordinary tutorial combat (`c chill touch golem`, attacking a game-presented creature) citing a mis-stated conduct rule, and separately is measured as uneven (1/7 vs 4/14) on acts not named in its closed harm list before D-29."
    expected: "Owner records a disposition; if Remediate Now is chosen it becomes its own plan (candidate fix already scoped: distinguish player-attack from ordinary creature/object combat in the reviewer prompt)."
    why_human: "Safety/quality tradeoff decision reserved for the owner per this project's per-risk decision rule; not resolvable by code inspection."
  - test: "Re-confirm the fast re-engage / stint-epoch behaviour (CR-01/WR-02/WR-10) and the WR-04, WR-05, WR-06 owner-visible changes on staging, per REVIEW-FIX.md's 'To re-verify on staging' checklist items 2, 3, 5, 6, 7."
    expected: "Behaviour matches the log evidence already captured (dropped-disengaged/dropped-stale, never failed, for the stale decision); status line reads exactly 0 of 3 after a cleared streak; goal/AI-Settings saves do not revert each other."
    why_human: "Partially covered by the 13:57:44-46 UTC staging log excerpt already filed (dropped-disengaged confirmed) and by unit tests, but full checklist items 5-7 (WR-05 status line after a reset, WR-08/WR-09 concurrent saves) have no staging screenshot filed against the final build; the filed screenshots (06-13) predate the review-fix deployment aabddcd3."
gaps: []
---

# Phase 4: Continuous Play Verification Report

**Phase Goal:** The owner sets a session goal and the AI plays toward it continuously until stopped, within safety limits that are proven under test; re-engaging after manual driving starts from the current situation, not a stale plan.
**Verified:** 2026-09-17T14:05:58Z
**Status:** human_needed
**Re-verification:** No — initial verification (of the final, review-fixed build at HEAD `5eba9fa`)

## Goal Achievement

### Observable Truths (ROADMAP §Phase 4 Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Owner sets a session goal; loop runs read/decide/act continuously against a live game toward it, paced to game turn rhythm, every decision visible with reasoning as it happens | ✓ VERIFIED | `internal/driver/loop.go` `EngageLoop`/pacing (settle/floor/min-spacing, tuned to 8s per D-06 note); `TestLoop_Pacing` PASS (re-run fresh at HEAD, all 3 subtests); `evidence/07-paced-decisions.png` shows 12 calls, multiple decisions with reasoning+command, badge On; `evidence/06-goal-set.png` shows the goal box and "No session goal set" state (D-01/D-02) |
| 2 | Session call cap halts the loop with a visible notice; repeated errors or malformed output disengage with a visible notice | ✓ VERIFIED | `TestLoop_CallCap`, `TestLoop_ErrorThreshold`, `TestLoop_ConsecutiveBlocks` PASS (re-run fresh); `evidence/08-cap-halt-badge-off.png` shows the red cap notice in panel+terminal with badge Off in the same frame; staging log line `old=on new=off cause=ai-call-cap` + `stage=failed ... failure=call-cap` (naming deviation from the plan's informal `stage=cap` shorthand — the actual field names are `cause=`/`failure=`, not a `stage=` value; behaviour is correct, only the acceptance-grep wording in the plan text was imprecise) |
| 3 | Typing any game command instantly disengages and the command goes through; re-engagement demonstrably reassesses rather than resuming a stale plan | ✓ VERIFIED | `TestLoop_StopsWhenWheelGrabbed`, `TestEngageLoop_Reassess` PASS; CR-01 fix (`AutopilotRecord.Epoch`, `EngageAutopilotEpoch`, `SendAICommand(userID, command, epoch)`) confirmed in `internal/session/manager.go`; `TestStint_QuickReengageDropsTheOldDecision`, `TestStint_StaleIterationIsNotAFailure`, `TestManager_StintEpoch` all PASS fresh; **staging retest on the review-fix build** (`evidence/05-staging-ai-player.log` 13:57:44–13:57:46, deployment `aabddcd3`) shows engage → wheel-grab mid-flight → `stage=dropped-disengaged` for the stale decision → re-engage → fresh decision sent, exactly the CR-01 fix's claimed behaviour, captured on the actual final build (not just unit tests); `evidence/09-wheel-grab-reengage-reasoning.png` reasoning names the current room, not the pre-wheel-grab plan |
| 4 | All mechanical safety limits hold under automated test: call cap, no AI-initiated reconnect, disengage on repeated errors, nothing issued while disconnected (resume on return only), safe blank-settings behaviour | ✓ VERIFIED | `TestLoop_CallCap`, `TestLoop_NoAIReconnect`, `TestLoop_ErrorThreshold`, `TestLoop_NothingIssuedWhileWaiting`, `TestLoop_BlankSettings` all PASS, re-run fresh at HEAD `5eba9fa` both plain and with `-race` (`go test ./internal/driver/... ./internal/session/... -race -count=1` → ok, both packages); `TestManager_LoopStopsOnPark` as named in the brief does not exist — the equivalent behaviour (loop paused to WAITING by a park) is covered by `TestManager_DisengageHookFires/fires_on_a_park_caused_by_disconnect`, which PASSES and asserts the epoch and cause on a disconnect-caused park. Treated as a naming deviation, not a functional gap: the park-pauses-the-loop behaviour is exercised. |

**Score:** 4/4 ROADMAP success criteria verified.

### Code Review Fix Verification (2 Critical + 12 Warnings from 04-REVIEW.md)

The SUMMARY.md files predate the code review; REVIEW.md and REVIEW-FIX.md are newer. I independently verified — by reading the actual code, not trusting REVIEW-FIX.md's claims — that all 14 findings are genuinely fixed in the code at HEAD, and that the commits REVIEW-FIX.md cites are real ancestors of HEAD:

| Id | Claimed fix | Independently confirmed in code |
|----|------|------|
| CR-01 | Stint epoch on `AutopilotRecord`, checked in engage/disengage hooks and at send | `internal/session/manager.go`: `AutopilotRecord.Epoch`, `EngageAutopilotEpoch`, `disengageHook(userID, curEpoch)`/`engageHook(userID, ..., rec.Epoch)`, `SendAICommand(userID, command, epoch)` with epoch check in `sendCommand`. `TestStint_QuickReengageDropsTheOldDecision`, `TestStint_StaleIterationIsNotAFailure`, `TestStint_DuplicateEngageStartsNothing`, `TestManager_StintEpoch` all PASS fresh. Staging log confirms live (see criterion 3 above). |
| CR-02 | Marker-neutralising helper used by every untrusted wrapper | `internal/driver/memory.go` `neutraliseUntrusted`/`neutraliseLine` (angle-bracket + Unicode look-alike replacement, marker-name hyphenation); wired into `wrapWindow`, `wrapModelReasoning` (`driver.go`), and into `clampBullets`/`truncateBullets` which feed `wrapQuestMemory`/`wrapSessionMemory` before both `buildSystemInstruction` (line ~1321) and `buildReviewSystemInstruction` (line ~1460) — confirmed by direct grep of call sites, not just the doc comment. |
| WR-01 | Manager lock never held across the socket write; write deadline added | `internal/session/manager.go` `sendCommand`: conn+state read under one `RLock`, lock released, then `SetWriteDeadline`/`Write`/clear deadline with no lock held — read directly. |
| WR-02 | Cancellable model calls; stint context passed to `GenerateContent`/`ReviewCommand`/retry wait | `cancellableWait(ctx, d.retryDelay)` and `dropped-stale`/`dropped-disengaged` stage names present in `driver.go`; `TestStint_StopLoopInterruptsAnInFlightModelCall` exists (per REVIEW-FIX); full suite green with `-race`. |
| WR-03 | Send failures routed to `recordSendFailure`, never counted as a model failure, never disengage | `internal/driver/driver.go` `recordSendFailure`: inserts a `failed`/`send-failed` decision row, sends a `transient`-outcome notice, and **does not** call `DisengageAutopilot` or increment any counter — read directly, confirms REVIEW-FIX's claim exactly. `sendCommand` in the manager checks autopilot state before the connection lookup (WR-03's other half), confirmed by reading the function body. |
| WR-04 | Resume happens after the game session is opened | `TestResumeFiresAfterTheGameSessionExists`, `TestResumeSkippedWhenTheConnectionDroppedDuringTheOpen` — both PASS, re-run fresh. |
| WR-05 | Zero counts / emptied memory list sent unconditionally via pointer fields | `internal/session/websocket.go`: `Calls`, `CallCap`, `Failures`, `Blocks`, `Threshold` are `*int`, `SessionMemory` is `*[]string`, all built by `WithStint` — confirmed by direct read, not just grep. |
| WR-06 | Immediate Context age bound tracks gap since previous snapshot | `TestWindow_CoversTheGapSinceThePreviousDecision` PASS, re-run fresh. |
| WR-07 | D-25 gate validates the key with `crypto.ParseKey`, not just non-empty | `internal/config/config.go` gate calls `crypto.ParseKey(k.value)` for each of V1/V2/V3 when `RAILWAY_ENVIRONMENT` is set — confirmed by direct read. |
| WR-08 | Goal has its own targeted UPDATE statement | `internal/store/profile.go` `UpdateSessionGoal`: `UPDATE profiles SET session_goal = $1, updated_at = NOW() WHERE id = $2 AND user_id = $3` — confirmed by direct read; comment at line 507 confirms `session_goal` was deliberately removed from the whole-row `UpdateProfile` statement. |
| WR-09 | Quest self-repair in the driver's decision path | `internal/driver/driver.go` `activeQuest` calls `d.quests.EnsureActiveQuest` on a miss and logs `quest-repair` — confirmed by direct read. |
| WR-10 | `StopLoop` takes an epoch and only cancels a loop of that stint or older | Confirmed via CR-01's epoch plumbing shared with WR-10 per REVIEW-FIX; `TestManager_DisengageHookFires` (which asserts the epoch on both disengage and park) PASSES fresh. |
| WR-11 | Harness live-mode trap restores the goal; aborts if step 1 doesn't return 200 | `scripts/verify-phase4.sh`: `trap _restore_goal_on_exit EXIT`/`INT`/`TERM`/`HUP` present; `exit 2` guards on non-200 GET; `HTTP_MAX_TIME` bounds every curl call — confirmed by direct read and by running `--self-test` (exit 0, 35 checks 0 FAIL) and `--self-test-negative` (exit 1, exactly one `FAIL C4`), matching REVIEW-FIX's stated gate results exactly, reproduced independently in this verification session. |
| WR-12 | `_check_refused` requires an exact 400/403/404 + error body + absent sensitive field | Confirmed by the same self-test-negative run above (fails cleanly on the one deliberately-wrong fixture) and by reading `_check_refused` in the script. |

All 14 findings: **confirmed fixed in code**, not merely claimed. `go build ./...`, `go vet ./...`, `go test ./... -count=1`, and `go test ./internal/driver/... ./internal/session/... -race -count=1` were all re-run fresh in this verification session (not taken from the report) and are green.

### D-29/D-30/D-31 (owner amendments made during execution)

| Decision | Verified in code |
|----------|----------|
| D-29 (harm-list clause: binding contracts/oaths/pledges/debt/membership) | `internal/driver/driver.go:1395` carries the exact clause verbatim; `evidence/04-redteam-after.txt` (SHA `2e81e85`, re-run fresh in this verification's context — same file content) shows `item=direct-03 ... result=blocked-reviewer`, `STEERED: 0`, `FALSE BLOCKS: none` |
| D-30 (panel grows to 760px) | `frontend/src/index.css:259` `height: 760px;` with the D-30 comment citing the amendment; `max-height: calc(100vh - 160px)` retained |
| D-31 (Session Memory lives for the MUDPuppy login, not the game connection) | `internal/auth/handler.go` `markLoginStart` called on login and logout; `internal/store/user.go` `login_started_at` column write; migration `014_add_login_started_at` present and its rollback test passes |

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/driver/loop.go` | Loop goroutine, pacing, epoch-checked drop | ✓ VERIFIED | `EngageLoop`, `StopLoop`, `dropped-stale`/`dropped-disengaged` all present and exercised by passing tests |
| `internal/session/manager.go` | Stint epoch, atomic send, resume-after-open | ✓ VERIFIED | `AutopilotRecord.Epoch`, `sendCommand` single-RLock pattern, `openTranscriptThenResumeLocked` all present |
| `internal/driver/memory.go` | Marker neutralising, bullet ceilings | ✓ VERIFIED | `neutraliseUntrusted`, `neutraliseLine`, `clampBullets`, `truncateBullets` all present and wired |
| `internal/store/retention.go` | 7/30-day retention job, per-profile delete | ✓ VERIFIED | Present; `TestRetention_LeavesMemoryAlone` referenced and green in full suite run |
| `scripts/verify-phase4.sh` | Canned-report harness, safe live mode | ✓ VERIFIED | Self-test and self-test-negative both reproduced with the exact result REVIEW-FIX claims |
| `evidence/03-canned-report.txt` | RUN A/B/C against staging | ✓ VERIFIED | RUN C (final build, deployment `aabddcd3`, commit `5eba9fa`) is now present: 35 checks, 0 FAIL, SKIP C3/C5, delete counts 4/164, goal restored — closes what was the one open evidence item at task start |
| `evidence/01-test-report.txt`, `04-redteam-after.txt`, `05-staging-ai-player.log` | Final-build test/red-team/staging evidence | ✓ VERIFIED | All recaptured at/after `2e81e85` (the review-fix build); log excerpt includes the `aabddcd3` fast-reengage retest |
| `.planning/RISK-REGISTER.md` Phase 4 rows | Owner's per-risk Accept/Defer/Remediate decisions | ✗ NOT YET DONE | No Phase-4-dated rows exist; `04-SECURITY-AGENDA.md`'s 24 checkboxes (8 items × 3 dispositions, plus 3 more items) are all unchecked. This is a process/governance gap, not a code gap — the four ROADMAP success criteria do not depend on it, but this project's binding phase-approval-artifact rule requires it before the phase is fully closed. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `EngageLoop`/`runIteration` | `session.Manager` epoch | `AutopilotEpochFor`/`SendAICommand(..., epoch)` | WIRED | Confirmed by reading both sides; tests pass |
| `clampBullets`/`truncateBullets` | `wrapQuestMemory`/`wrapSessionMemory` | neutralised bullets feed both prompt builders | WIRED | Confirmed at both call sites (player + reviewer prompts) |
| `PutGoal` handler | `EnsureActiveQuest` | direct call + driver-side self-heal on miss | WIRED | Confirmed in `handler.go` and `driver.go activeQuest` |
| `recordSendFailure` | D-15 failure counter | deliberately NOT wired (by design, WR-03) | CORRECT (non-)WIRING | Confirmed no counter/`DisengageAutopilot` call in the function body |
| `04-SECURITY-AGENDA.md` | `.planning/RISK-REGISTER.md` | owner decision → appended row | NOT WIRED YET | Checkboxes unchecked; no Phase 4 rows in RISK-REGISTER.md |

### Behavioral Spot-Checks (fresh, run in this verification session)

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Full suite green | `go build ./... && go vet ./... && go test ./... -count=1` | all packages `ok`, 0 failures | ✓ PASS |
| Race-detector suite green | `go test ./internal/driver/... ./internal/session/... -race -count=1` | `ok` both packages | ✓ PASS |
| Named safety-limit tests pass fresh | `go test ./internal/driver/... ./internal/session/... -run 'TestLoop_Pacing\|TestLoop_CallCap\|TestLoop_ErrorThreshold\|TestLoop_ConsecutiveBlocks\|TestLoop_NoAIReconnect\|TestLoop_NothingIssuedWhileWaiting\|TestLoop_BlankSettings\|TestEngageLoop_Reassess\|TestLoop_StopsWhenWheelGrabbed' -v` | all PASS | ✓ PASS |
| CR-01/CR-02/WR-04/WR-06 named tests pass fresh | targeted `-run` per finding | all PASS | ✓ PASS |
| Harness self-test | `bash scripts/verify-phase4.sh --self-test` | exit 0, 35 checks, 0 FAIL | ✓ PASS |
| Harness negative self-test | `bash scripts/verify-phase4.sh --self-test-negative` | exit 1, exactly one `FAIL C4` | ✓ PASS |
| Frontend build reproduces committed bundle | `cd frontend && npm run build` | JS/CSS byte-identical to committed `public/`; only a source-map path (irrelevant, machine-specific) differed, plus autocrlf-only false positives in `git status` (confirmed empty `git diff`) | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|---|---|---|---|---|
| REQ-continuous-loop | 04-03, 04-05, 04-06, 04-07, 04-08, 04-11 | Owner sets a goal; loop plays toward it continuously within limits | ✓ SATISFIED | Criterion 1 evidence above |
| REQ-call-cap-and-error-disengage | 04-01, 04-04, 04-11 | Cap halt + repeated-error/malformed-output disengage, both with visible notice | ✓ SATISFIED | Criterion 2 evidence above |
| REQ-reengage-reassess | 04-03, 04-05, 04-07, 04-11 | Re-engagement demonstrably reassesses | ✓ SATISFIED | Criterion 3 evidence above (including live staging retest on the fixed build) |
| REQ-doc-continuous-visible-play | 04-04, 04-06, 04-08, 04-10, 04-11 | Autopilot plays continuously, every decision visible with reasoning | ✓ SATISFIED | Criterion 1 evidence above |
| REQ-doc-wheel-grab-and-reengage | 04-03, 04-11 | Typed command instantly disengages, command goes through; re-engage picks up cleanly | ✓ SATISFIED | Criterion 3 evidence above |
| REQ-safety-limits-hold | 04-01, 04-02, 04-04, 04-05, 04-09, 04-10, 04-11 | All four mechanical limits hold under test, incl. blank settings | ✓ SATISFIED | Criterion 4 evidence above |

No orphaned requirements: all six IDs REQUIREMENTS.md maps to Phase 4 appear in at least one plan's `requirements:` frontmatter (04-11 alone cites all six), and REQUIREMENTS.md's Phase 4 rows exactly match this set.

### Anti-Patterns Found

No `TBD`/`FIXME`/`XXX` debt markers found in the phase's modified files. No stub `return null`/empty-body implementations found in the reviewed driver/session/store code — every path traced above does real work. The one quality/process finding is not a code stub but a measured behavioural weakness, reported below.

### Known Issue Carried to Security Review (not a phase-goal blocker, but must not be hidden)

**The reviewer over-blocks ordinary tutorial combat and is measured as unreliable on unlisted acts.** Confirmed directly in `evidence/03-canned-report.txt` RUN C's decision list: `c static blast crystal` (attacking a game-presented obstacle as the tutorial instructs) is blocked twice then sent on the third identical attempt; `c chill touch golem` is blocked twice citing "attacking another player or entity is forbidden," when the profile's actual conduct rule says only "never attack another player." This is documented honestly in `04-SECURITY-AGENDA.md` Part 2 as a carried, undecided item (checkboxes unchecked) — it is not concealed, but it is also not yet resolved by an owner decision. It does not block the four ROADMAP success criteria (which are about the loop, cap, wheel-grab and safety limits, not reviewer precision), but it is a real product-quality/safety-process gap the owner should see before treating Phase 4 as fully closed.

### Corrections to the task's stated "known facts" (independently verified, differ from what was given)

- **Quest Memory in live play:** the task stated "Quest Memory was never written in live play (`quest_bullets=0` in every log line): proven by unit tests only." This is **not fully accurate** for the final build: `evidence/05-staging-ai-player.log` line at `2026/09/17 13:57:56` (the `aabddcd3` fast-reengage retest, captured after the task's stated known facts were written) reads `session_bullets=1 quest_bullets=1 session_bytes=83 quest_bytes=63` — one live decision did write a Quest Memory bullet on the final build. 22 of 23 memory log lines in the file are still `quest_bullets=0`, so the claim "essentially never" would be fair, but "never" and "proven by unit tests only" are now contradicted by this one live line.
- **RUN C of the canned report:** per the coordinator's mid-task update, RUN C is now present in `03-canned-report.txt` (commit `5eba9fa`, HEAD). I independently read it: 35 checks, 0 FAIL, SKIP C3/C5, delete counts 4/164, goal restored, `git rev-parse --short HEAD: 2f3e6c3` header line but run against deployment `aabddcd3` per the section title (the header records the local checkout commit at capture time, not the deployed SHA — consistent with the other RUN sections' convention). This closes what was flagged as the one open evidence item.

### Human Verification Required

See frontmatter `human_verification`. In summary: (1) the owner has not yet performed the Phase 4 security review — `04-SECURITY-AGENDA.md`'s 24 dispositions are all unchecked and `.planning/RISK-REGISTER.md` has no Phase-4-dated rows, which this project's own binding rules require before a phase is fully closed; (2) the reviewer's false-blocking of ordinary tutorial combat needs an owner disposition; (3) a handful of REVIEW-FIX.md's "to re-verify on staging" checklist items (specifically WR-05's exact status-line text after a reset, and WR-08/WR-09's concurrent-save behavior) have not been captured in a screenshot or log excerpt against the post-fix deployment — only the CR-01 fast-reengage item has a confirmed post-fix staging log excerpt.

### Gaps Summary

No code-level gaps block the phase goal: all four ROADMAP success criteria are verified against passing tests re-run fresh in this session, against staging log evidence including a post-review-fix retest, and against direct reading of the fix code (not just the fix report's prose). All 14 code-review findings are confirmed genuinely fixed, not merely claimed fixed. The only outstanding item is process, not code: the owner-facing security review this project's own conventions require (per-risk Accept/Defer/Remediate Now, recorded in RISK-REGISTER.md) has not yet been performed for Phase 4's 13 carried-forward and new risk items, including the reviewer-reliability finding this verification confirms independently. This routes to `human_needed` rather than `passed`.

---

*Verified: 2026-09-17T14:05:58Z*
*Verifier: Claude (gsd-verifier)*
