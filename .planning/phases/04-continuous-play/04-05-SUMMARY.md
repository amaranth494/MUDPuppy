---
phase: 04-continuous-play
plan: 05
subsystem: api
tags: [go, memory-model, ring-buffer, session, immediate-context]

# Dependency graph
requires:
  - phase: 04-continuous-play
    provides: "04-03's continuous loop and Manager.RecentOutputSnapshot call site the driver reads through every iteration"
provides:
  - "ringBuffer.snapshotRecent(maxAge, minBytes): the age-bounded, never-empty read that implements the Immediate Context memory layer (D-09) alongside the existing byte-bounded snapshot()"
  - "windowMaxAge (10s) and windowMinRetainedBytes (2048) package constants naming D-09 and the memory model's Immediate Context layer"
  - "An injectable ringBuffer.now func() time.Time seam, defaulting to time.Now, for deterministic clock-driven tests with no fake-clock library"
  - "Manager.RecentOutputSnapshot now returns the reaction window instead of the whole ring; unchanged signature, locking, and single call site (internal/driver/driver.go)"
affects: [04-continuous-play plans 04-07, 04-08 (Session Memory and Quest Memory build on this Immediate Context layer per the memory model's flow), 04-11 (evidence capture references TestWindow_AgeBounded/TestWindow_NeverEmptyAfterQuiet PASS lines)]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "ringBuffer gains a parallel chunkMarker slice (cumulative byte offset + arrival time), appended on every append() call and trimmed the moment its bytes are fully overwritten in the byte ring — same lifecycle discipline as the byte ring itself, no separate cleanup pass"
    - "snapshotBytes() is the one shared chronological-ordering routine both snapshot() and snapshotRecent() build on, so the byte-ordering logic exists in exactly one place"
    - "Age-bound tests inject a package-private now func() time.Time field (windowTestClock in window_test.go) and call clock.advance(d) rather than sleeping or adding a fake-clock dependency"

key-files:
  created: []
  modified:
    - internal/session/window.go
    - internal/session/window_test.go
    - internal/session/manager.go

key-decisions:
  - "The age bound (10s) and never-empty floor (2048 bytes) are shipped exactly at the plan's Claude's-Discretion starting points (04-RESEARCH Assumption A2); no tuning performed here, consistent with 04-03's pacing-constants precedent."
  - "Chunk timestamps live as a parallel []chunkMarker slice tracking cumulative byte offsets, not per-byte timestamps, per the plan's explicit instruction that timestamps live on chunk boundaries."
  - "snapshotRecent's age-vs-floor resolution walks chunk markers newest-to-oldest and stops at the first chunk older than the cutoff, since chunks are time-ordered by construction; this keeps the implementation O(number of chunks currently in the ring) with no need for binary search or a separate index."
  - "TestWindow_AgeBounded pads its within-bound chunk past windowMinRetainedBytes so the never-empty floor cannot mask the age exclusion — an implementation detail discovered while writing the test (the first draft's chunks were all small enough that the floor pulled the old chunk back in, which is correct behavior per D-09 but not what that specific test needed to isolate)."

patterns-established:
  - "A memory-layer age bound and its never-empty floor are proven as two independent, separately-named tests (TestWindow_AgeBounded, TestWindow_NeverEmptyAfterQuiet) rather than one combined test, so a future regression in either rule fails with an unambiguous name."

requirements-completed: [REQ-continuous-loop, REQ-reengage-reassess, REQ-safety-limits-hold]

# Metrics
duration: 6min
completed: 2026-09-16
---

# Phase 4 Plan 05: The window knows how old its text is Summary

**`ringBuffer` gains a time dimension beside its existing byte ceiling: `snapshotRecent(windowMaxAge=10s, windowMinRetainedBytes=2048)` implements the memory model's Immediate Context layer, with an injectable clock so the age bound and never-empty floor are both proven by test without sleeping.**

## Performance

- **Duration:** 6 min (f514562 to 26fa9c0)
- **Started:** 2026-09-16T21:45:00-07:00 (approx, task 1 read/write/verify preceding first commit)
- **Completed:** 2026-09-16T21:48:33-07:00
- **Tasks:** 2 completed
- **Files modified:** 3

## Accomplishments
- `internal/session/window.go`'s `ringBuffer` can now answer "what happened in the last ten seconds" (`snapshotRecent`) as well as "what is on screen" (`snapshot`, unchanged) — proven by `TestWindow_AgeBounded` and `TestWindow_NeverEmptyAfterQuiet` PASS lines.
- The 8192-byte ceiling still wins when a flood of text all arrives within the age bound — proven by `TestWindow_CeilingStillHolds`.
- `internal/session/manager.go`'s `RecentOutputSnapshot` — the one call site `internal/driver/driver.go` reads through every decision — now returns the age-bounded reaction window instead of the whole ring, with zero changes to `internal/driver` (confirmed by an empty `git diff --stat internal/driver/`).
- All four pre-existing window tests (`TestWindowSnapshot`, `TestWindowRingBound`, `TestWindowIncludesPreEngageText`, `TestWindowSurvivesDisconnect`) and the four new ones pass unchanged; the full `internal/...` suite shows no new failures, and `-race` is clean on `internal/session`.

## Task Commits

Each task was committed atomically:

1. **Task 04-05-01: The window knows how old its text is and never empties** - `f514562` (feat)
2. **Task 04-05-02: Every decision reads the recent window, and the rules are pinned by test** - `26fa9c0` (test)

**Plan metadata:** (this commit, made by the orchestrator after all worktree agents in the wave complete)

## Files Created/Modified
- `internal/session/window.go` - Added `windowMaxAge`/`windowMinRetainedBytes` constants, `chunkMarker` type, `ringBuffer.now`/`totalWritten`/`chunks` fields, `recordChunk`, `snapshotBytes` (shared by `snapshot` and the new `snapshotRecent`), and `snapshotRecent(maxAge, minBytes) string`
- `internal/session/window_test.go` - Added `windowTestClock` (injectable clock helper) and `TestWindow_AgeBounded`, `TestWindow_NeverEmptyAfterQuiet`, `TestWindow_CeilingStillHolds`, `TestWindow_RecentSnapshotStripsANSI`
- `internal/session/manager.go` - `RecentOutputSnapshot` now calls `ring.snapshotRecent(windowMaxAge, windowMinRetainedBytes)` instead of `ring.snapshot()`; added a doc comment naming D-09 and the reaction-window behavior

## Decisions Made
- Age bound and never-empty floor shipped exactly at the plan's stated starting points (10s / 2048 bytes) — see key-decisions above for the full rationale and the test-design adjustment discovered while writing `TestWindow_AgeBounded`.

## Deviations from Plan

None - plan executed exactly as written. The one implementation detail worth flagging (chunk padding in `TestWindow_AgeBounded` to keep the age-exclusion assertion independent of the never-empty floor) is a test-construction choice within the plan's own instructions, not a deviation from them — the plan's `<action>` text already specifies both rules independently and names both as required test cases.

## Issues Encountered

The first draft of `TestWindow_AgeBounded` used small chunks throughout and failed: the never-empty floor (2048 bytes) correctly pulled the older chunk back into the result because the age-bounded portion alone was smaller than the floor. This was the floor working exactly as D-09 specifies, not a bug in `snapshotRecent` — the test itself needed a large chunk-new so the two rules (age bound, never-empty floor) could be asserted independently. Fixed by padding chunk-new past `windowMinRetainedBytes`; no production code changed.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `snapshotRecent` and the chunk-marker infrastructure are ready for 04-07/04-08 (Session Memory, Quest Memory) to build on per the memory model's `Game Output -> Immediate Context -> Session Memory` flow; nothing in this plan touches Session or Quest Memory.
- No blockers. `go build ./...`, `go vet ./internal/session/...`, `go test ./internal/session/... -count=1 -v`, `go test ./internal/... -count=1`, and `go test ./internal/... -race -count=1` all pass on this machine; `git diff --stat go.mod frontend/package.json` and `git diff --stat internal/driver/` are both empty.
- Ran as a parallel worktree executor alongside plan 04-04 (working in the main tree on `internal/driver/*`, `internal/session/websocket.go`, `cmd/server/main.go`, `frontend/*`). This plan touched only its declared files (`internal/session/window.go`, `internal/session/window_test.go`, `internal/session/manager.go`) — no file outside that list was modified, and `internal/session/manager.go`'s only change was the single `RecentOutputSnapshot` body line plus its doc comment, at a location distinct from 04-04's declared work.

---
*Phase: 04-continuous-play*
*Completed: 2026-09-16*

## Self-Check: PASSED

- FOUND: internal/session/window.go (modified, contains snapshotRecent)
- FOUND: internal/session/window_test.go (modified, contains TestWindow_AgeBounded)
- FOUND: internal/session/manager.go (modified, RecentOutputSnapshot calls snapshotRecent)
- FOUND: commit f514562 (Task 04-05-01)
- FOUND: commit 26fa9c0 (Task 04-05-02)
