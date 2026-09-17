---
phase: 04-continuous-play
plan: 09
subsystem: database
tags: [go, postgres, retention, react, typescript]

# Dependency graph
requires:
  - phase: 04-continuous-play
    provides: "04-08's Session Memory/Quest Memory columns (game_sessions.session_memory, quests.bullets) that this plan's retention job must never touch; the ai_decisions.window_text column and the decision audit columns (reasoning, command, outcome, failure_kind, notice) D-21 keeps forever"
provides:
  - "internal/store/retention.go: DecisionSnapshotRetentionDays=7, TranscriptRetentionDays=30 exported constants; RetentionJob.Run() clears ai_decisions.window_text and deletes game_session_lines past each window in one transaction, reporting RetentionCounts; RetentionJob.DeleteCapturedTextForConnection scopes both operations to one connection_id with no age condition"
  - "cmd/server/main.go: runRetentionLoop, one goroutine started once at server scope holding a 24-hour time.NewTicker, running once ~10s after startup; each run logs one [AI-PLAYER] stage=retention line with both counts, both window lengths in days, and duration_ms; a run error logs error=true and the loop continues"
  - "DELETE /api/v1/profiles/{connection_id}/captured-text (internal/profiles/handler.go DeleteCapturedText): ownership resolved via getProfileByConnectionID before any delete (T-4-09), logs stage=retention-manual, fails closed 503 with no retention store wired"
  - "frontend: deleteCapturedText() in services/api.ts, DeleteCapturedTextResponse type, and the .ai-player-danger-zone block in AIPlayerPanel.tsx (locked copy, two-step .delete-confirm/.btn-danger confirmation)"
affects: [04-continuous-play plan 04-10 (harness ownership/not-owned-connection evidence for the captured-text delete), plan 04-11 (staging walkthrough evidence/05-staging-ai-player.log's retention line, evidence/03-canned-report.txt's PASS C4 lines, evidence/12-delete-captured-text-confirm.png; Phase 4 security review records DR-3-01/DR-3.1-03 closure under D-28)]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Retention SQL statements are package-level constants (clearOldDecisionSnapshotsSQL, deleteOldTranscriptLinesSQL, clearConnectionDecisionSnapshotsSQL, deleteConnectionTranscriptLinesSQL) so retention_test.go can assert on their exact text with no database connection, mirroring quests.go's ensureActiveQuestSQL precedent"
    - "The nightly job is a single time.NewTicker goroutine started once at server scope (not per-user), copying internal/session/manager.go's startTimers shape -- the codebase's only background-timer idiom, no new infrastructure"
    - "A prune/delete operation's row counts (never row text) are the only thing that ever leaves the store layer or the [AI-PLAYER] log line -- RetentionCounts carries exactly SnapshotsCleared/TranscriptLinesDeleted"

key-files:
  created:
    - internal/store/retention.go
    - internal/store/retention_test.go
    - internal/profiles/retention_test.go
  modified:
    - cmd/server/main.go
    - internal/profiles/handler.go
    - frontend/src/services/api.ts
    - frontend/src/types/index.ts
    - frontend/src/components/AIPlayerPanel.tsx
    - frontend/src/index.css
    - public/index.html
    - public/assets/index-BVoB3fBB.css
    - public/assets/index-mz5qSCGX.js
    - public/assets/index-mz5qSCGX.js.map

key-decisions:
  - "The decision snapshot is cleared in place (window_text = '') rather than the row being deleted -- the decision row keeps every other column (timestamp, reasoning, command, outcome, failure_kind, notice) forever, per D-21's audit-trail requirement"
  - "Transcript pruning deletes game_session_lines rows only; game_sessions summary rows are never deleted, so the log page still lists a pruned session and shows it as empty"
  - "The nightly schedule is a single time.NewTicker(24 * time.Hour) goroutine started once in cmd/server/main.go, running once ~10s after startup and then every 24 hours -- server-scope, not per-user, mirroring internal/session/manager.go's startTimers, the codebase's only prior background-timer idiom"
  - "Route is DELETE /api/v1/profiles/{connection_id}/captured-text, a sibling of the other profile sub-resources (ai-goal, ai-memory, ai-settings), registered with a DELETE-only method switch"
  - "The [AI-PLAYER] retention log line (stage=retention) carries only counts, the two window lengths in days, and a duration in milliseconds -- no ids, no row text; the manual per-profile delete's line (stage=retention-manual) adds the user_id/connection_id, still no row text"
  - "A nil retention store fails closed with 503 on DeleteCapturedText (matching GetSessionMemory's nil-transcripts precedent), rather than panicking, even though the plan's acceptance criteria did not explicitly require this -- consistent with the codebase's existing missing-dependency discipline"

patterns-established:
  - "A background job whose only output is aggregate counts (RetentionCounts) is exposed identically through two call paths -- a scheduled Run() and an on-demand per-scope DeleteCapturedTextForConnection() -- sharing one execRetentionPair helper so the two never drift in which columns they touch"

requirements-completed: [REQ-safety-limits-hold]

# Metrics
duration: 15min
completed: 2026-09-16
---

# Phase 4 Plan 09: Retention Standard and On-Demand Delete Summary

**A nightly in-process ticker clears decision-window snapshots after 7 days and deletes session-transcript lines after 30 (D-21), leaving the decision audit trail and both memory layers untouched, plus a DELETE /api/v1/profiles/{connection_id}/captured-text route and a two-step "Delete Captured Text Now" danger zone for the owner's immediate, per-profile version of the same prune.**

## Performance

- **Duration:** ~15 min across three task commits (7466bb9 to 9cfe816); research/context-gathering preceding execution not included
- **Started:** 2026-09-16T23:16:23-07:00
- **Completed:** 2026-09-16T23:20:25-07:00
- **Tasks:** 3 completed
- **Files modified:** 14 (6 new/modified Go source+test, 4 frontend source, 4 rebuilt production bundle files)

## Accomplishments

- **The retention standard as code (D-21, DR-3-01, DR-3.1-03):** `internal/store/retention.go` defines `DecisionSnapshotRetentionDays = 7` and `TranscriptRetentionDays = 30` as exported constants, each documented with what it governs and deliberately leaves alone. `RetentionJob.Run()` clears `ai_decisions.window_text` on rows older than the snapshot window (only where the snapshot is not already empty) and deletes `game_session_lines` for sessions started before the transcript window, both in one transaction, reporting a `RetentionCounts{SnapshotsCleared, TranscriptLinesDeleted}`. No statement anywhere in the file deletes an `ai_decisions` row or names `session_memory`, `quests`, `bullets`, `reasoning`, `command`, `outcome`, `failure_kind`, or `notice` — proven by `retention_test.go`'s four no-database tests (`TestRetentionWindows`, `TestRetention_TouchesOnlyCapturedText`, `TestRetention_LeavesMemoryAlone`, `TestRetention_PerProfileDeleteIsScoped`), which scan the package-level SQL constants directly.
- **The schedule and the manual trigger (D-21):** `cmd/server/main.go` constructs the job beside the other stores and starts one goroutine (`runRetentionLoop`) holding a `time.NewTicker(24 * time.Hour)`, running once ~10 seconds after startup so a fresh deploy proves the path immediately. Each run logs exactly one `[AI-PLAYER] stage=retention` line carrying both counts, both window lengths in days, and the run's duration in milliseconds; an error logs `error=true` and the loop continues — retention can never stop the server or affect play. `DELETE /api/v1/profiles/{connection_id}/captured-text` (`Handler.DeleteCapturedText`) resolves ownership through `getProfileByConnectionID` before any delete (T-4-09), calls `DeleteCapturedTextForConnection`, logs `[AI-PLAYER] stage=retention-manual` with the user/connection ids and counts, fails closed with 503 when no retention store is wired, and never partially reports success on a store error.
- **The danger zone (D-21, T-4-30, T-4-09):** `frontend/src/components/AIPlayerPanel.tsx` gains `.ai-player-danger-zone`, its own block after the `.settings-actions`/"Save AI Settings" row, separated by a top border. The unconditional hint reads the locked scope statement verbatim; the `Delete Captured Text Now` button swaps into the `.delete-confirm`/`.btn-danger` two-step confirmation (`Delete captured text now?`, Yes/No) reused unchanged from `ConnectionsHubModal.tsx`'s saved-connection delete. Success shows `Captured text deleted.` and failure shows `Failed to delete captured text — refresh the page to try again`, both through the panel's existing message state. No new modal, no new CSS custom property, no change to `SettingsPage.tsx`.

## Task Commits

Each task was committed atomically:

1. **Task 04-09-01: The retention standard exists as code, with its windows and its limits pinned** - `7466bb9` (feat)
2. **Task 04-09-02: The job runs on a schedule and the owner can run it now for his own profile** - `8609dd1` (feat)
3. **Task 04-09-03: Delete Captured Text Now sits where it cannot be hit by accident** - `9cfe816` (feat)

## Files Created/Modified

- `internal/store/retention.go` — `DecisionSnapshotRetentionDays`, `TranscriptRetentionDays`, `RetentionCounts`, `RetentionJob`, `NewRetentionJob`, `Run`, `DeleteCapturedTextForConnection`, the four package-level SQL constants, `execRetentionPair`
- `internal/store/retention_test.go` — `TestRetentionWindows`, `TestRetention_TouchesOnlyCapturedText`, `TestRetention_LeavesMemoryAlone`, `TestRetention_PerProfileDeleteIsScoped`
- `cmd/server/main.go` — `retentionJob` construction, `profilesHandler.SetRetention` wiring, the `captured-text` route (DELETE only), `runRetentionLoop`
- `internal/profiles/handler.go` — `retentionStorage` interface, `retention` field, `SetRetention`, `DeleteCapturedTextResponse`, `DeleteCapturedText`
- `internal/profiles/retention_test.go` — `fakeRetentionStore`, `TestDeleteCapturedText` (success, not-owned refusal before any delete, store-error all-or-nothing, fail-closed 503, wrong-method rejection)
- `frontend/src/services/api.ts`, `types/index.ts` — `deleteCapturedText`, `DeleteCapturedTextResponse`
- `frontend/src/components/AIPlayerPanel.tsx` — `confirmingDelete`/`isDeleting` state, `handleDeleteCapturedText`, the danger-zone markup
- `frontend/src/index.css` — `.ai-player-danger-zone`
- `public/index.html`, `public/assets/index-BVoB3fBB.css`, `index-mz5qSCGX.js[.map]` — rebuilt production bundle (`npm run build`)

## Decisions Made

See `key-decisions` in the frontmatter above.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing critical] Added test coverage for `DeleteCapturedText`**
- **Found during:** Task 04-09-02
- **Issue:** The task's acceptance criteria list only source assertions and a green build for `DeleteCapturedText`; no Go test was named. A new destructive HTTP endpoint touching ownership (T-4-09) with no test coverage would be a correctness/security gap, matching the precedent already set in 04-08's own deviation for `GetSessionMemory`.
- **Fix:** Added `internal/profiles/retention_test.go` with a hand-written `fakeRetentionStore` and `TestDeleteCapturedText` covering: the success path with count round-trip, a not-owned connection refused before `DeleteCapturedTextForConnection` is ever called, a store error returning the existing error shape with no partial success, a nil retention store failing closed with 503, and a wrong-method request being rejected.
- **Files modified:** `internal/profiles/retention_test.go`
- **Verification:** `go test ./internal/profiles/... -run TestDeleteCapturedText -v` — all five cases PASS.
- **Committed in:** `8609dd1` (Task 2 commit)

**2. [Rule 1 - Bug] Reworded two code comments that duplicated the locked button label**
- **Found during:** Task 04-09-03
- **Issue:** `AIPlayerPanel.tsx`'s own explanatory comments happened to contain the literal string "Delete Captured Text Now", making the acceptance criterion's `grep -c "Delete Captured Text Now"` return 3 instead of the required 1 (button text only).
- **Fix:** Reworded both comments to describe "the captured-text danger zone" instead of repeating the locked label verbatim.
- **Files modified:** `frontend/src/components/AIPlayerPanel.tsx`
- **Verification:** `grep -c "Delete Captured Text Now" frontend/src/components/AIPlayerPanel.tsx` returns 1.
- **Committed in:** `9cfe816` (Task 3 commit)

---

**Total deviations:** 2 auto-fixed (1 missing critical test coverage, 1 bug in a grep-checkable acceptance criterion)
**Impact on plan:** Both necessary for a passing acceptance check and for baseline security-relevant test coverage of a new destructive endpoint. No scope creep — no new behavior beyond what the plan specified.

## Issues Encountered

None beyond the two deviations above.

**Note on one acceptance-criteria check:** Task 3's acceptance criterion `git diff frontend/src/index.css | grep -c "^+.*--color-"` (intended to catch a *new custom property declaration*) will also match the one CSS rule this plan adds, because `border-top: 1px solid var(--color-border);` — a *reuse* of an existing property, exactly as `04-UI-SPEC.md` specifies verbatim — contains the substring `--color-` inside its `var()` call. The rule adds zero new `--custom-property:` declarations (confirmed by reading the diff directly); this is a known limitation of the literal grep pattern, not a plan violation.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- The retention standard is proven at the unit level (four no-database tests) and wired end to end (scheduled job, manual endpoint, owner-facing control); the staging walkthrough (plan 04-11) still needs to produce the live evidence artifacts named in this plan's `<output>`: the `stage=retention` line in `evidence/05-staging-ai-player.log`, the PASS C4 lines of `evidence/03-canned-report.txt`, and `evidence/12-delete-captured-text-confirm.png`.
- DR-3-01 and DR-3.1-03 closures are ready to record at the Phase 4 security review under D-28, alongside this plan's own code-level proof.
- No blockers. `go build ./...`, `go vet ./internal/... ./cmd/...`, `go test ./internal/... -count=1`, `go test ./internal/... -race -count=1`, and `cd frontend && npm run build` all pass on this machine; `git diff --stat go.mod frontend/package.json` is empty (T-4-SC).
- Plan 04-10 (harness) needs a not-owned-connection step against `DELETE /api/v1/profiles/{connection_id}/captured-text`, and a decisions-read-back-after-delete step proving reasoning/command/outcome survive.

## Threat Flags

None new beyond what this plan's own `<threat_model>` already declares (T-4-08, T-4-15, T-4-09, T-4-30, T-4-31, T-4-05, T-4-SC) — every file touched is inside that register.

---
*Phase: 04-continuous-play*
*Completed: 2026-09-16*

## Self-Check: PASSED

- FOUND: internal/store/retention.go
- FOUND: internal/store/retention_test.go
- FOUND: internal/profiles/retention_test.go
- FOUND: cmd/server/main.go
- FOUND: internal/profiles/handler.go
- FOUND: frontend/src/services/api.ts
- FOUND: frontend/src/types/index.ts
- FOUND: frontend/src/components/AIPlayerPanel.tsx
- FOUND: frontend/src/index.css
- FOUND: public/index.html
- FOUND: public/assets/index-BVoB3fBB.css
- FOUND: public/assets/index-mz5qSCGX.js
- FOUND: public/assets/index-mz5qSCGX.js.map
- FOUND: commit 7466bb9 (Task 04-09-01)
- FOUND: commit 8609dd1 (Task 04-09-02)
- FOUND: commit 9cfe816 (Task 04-09-03)
