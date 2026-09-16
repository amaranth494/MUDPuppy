---
phase: 03-one-ai-decision
plan: 01
subsystem: session
tags: [go, ring-buffer, ansi, session-management, concurrency]

# Dependency graph
requires: []
provides:
  - "internal/session/window.go: RecentOutputWindowBytes constant (8192), stripANSI CSI stripper, pure ringBuffer type (append/snapshot)"
  - "internal/session/manager.go: Manager.outputWindow map (userID -> *ringBuffer), Manager.RecentOutputSnapshot(userID) string for the driver, Manager.appendOutputWindow fed continuously from ReadOutput"
affects: [03-08-driver-single-decision]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Pure transform sibling to stripTelnetIAC (websocket.go): stripANSI(b []byte) []byte, no receiver, no I/O"
    - "Fifth userID-keyed map on Manager under the existing m.mu, same 'lives here not on Session' discipline as autopilot map"
    - "RecentOutputSnapshot mirrors AutopilotStateFor's RLock/defer shape exactly"

key-files:
  created:
    - internal/session/window.go
    - internal/session/window_test.go
  modified:
    - internal/session/manager.go

key-decisions:
  - "ReadOutput does not hold m.mu when it returns (RLock released right after the conn lookup), so appendOutputWindow's own m.mu.Lock() is called directly rather than inlining append under an already-held lock"
  - "The window is never cleared on Connect or Disconnect, only on process restart, matching D-02's requirement that a WAITING-to-ON resume reads text spanning the drop"
  - "-race could not be run on this dev machine (CGO_ENABLED=0, no gcc); all verification ran as plain go test instead, logged in deferred-items.md for whoever captures evidence/01-test-report.txt"

patterns-established:
  - "Ring buffer bound is a server constant (RecentOutputWindowBytes), not a profile setting, matching D-04"

requirements-completed: [REQ-single-decision]

# Metrics
duration: ~20min
completed: 2026-09-15
---

# Phase 3 Plan 01: Bounded Recent-Output Window Summary

**A per-user, 8KB, ANSI-free ring buffer of recent MUD output, fed continuously by `Manager.ReadOutput` and exposed to the future AI driver via `Manager.RecentOutputSnapshot`, surviving both disconnects and pre-engage history.**

## Performance

- **Duration:** ~20 min
- **Started:** 2026-09-15T18:45:00-07:00 (approx.)
- **Completed:** 2026-09-15T18:55:41-07:00
- **Tasks:** 2
- **Files modified:** 3 (2 created, 1 modified)

## Accomplishments
- `internal/session/window.go` gives the codebase a pure, testable ring buffer plus an ANSI CSI stripper, mirroring the existing `stripTelnetIAC` shape exactly, with no mutex and no per-user map of its own.
- `Manager` grew a fifth userID-keyed map (`outputWindow`), fed from the sole MUD-read path (`ReadOutput`) on telnet- and ANSI-stripped bytes, and exposes exactly one accessor (`RecentOutputSnapshot`) for the driver plan 03-08 will build.
- The window is proven (by `go test`) to include text from before autopilot is ever engaged and to survive a real `Manager.Disconnect` call, so a WAITING-to-ON resume reads across the drop.
- The browser-bound stream (`relayMUDToClient`) is untouched — verified by source assertion (`stripANSI` does not appear in `websocket.go`) and by the fact that `ReadOutput`'s new call builds a separate stripped slice rather than mutating `buffer`.

## Task Commits

Each task was committed atomically:

1. **Task 03-01-01: A bounded ring of plain game text exists, and a test proves colour codes never reach it** - `d338ada` (feat)
2. **Task 03-01-02: Manager keeps the window continuously and hands the driver a snapshot on request** - `9c2f24c` (feat)

**Additional commit:** `253b2b8` (docs: log out-of-scope discoveries — pre-existing `internal/icm` test failure and the `-race`/cgo environment limitation)

## Files Created/Modified
- `internal/session/window.go` - `RecentOutputWindowBytes = 8192`, `stripANSI`, `ringBuffer` (`append`/`snapshot`)
- `internal/session/window_test.go` - `TestWindowSnapshot` (4 subtests), `TestWindowRingBound`, `TestWindowIncludesPreEngageText`, `TestWindowSurvivesDisconnect`
- `internal/session/manager.go` - `outputWindow` map on `Manager`, `appendOutputWindow`, `RecentOutputSnapshot`, one call site inside `ReadOutput`

## Decisions Made
- `ReadOutput` releases `m.mu.RLock()` immediately after looking up the connection and does not re-acquire any lock before returning, so the plan's conditional "call inline under the held lock" branch was not needed — `appendOutputWindow` is called directly and takes its own `m.mu.Lock()`.
- The ring is bounded with distinguishable, uniquely-labeled test chunks (`|chunk00000|`, etc.) rather than a raw incrementing-byte-mod-256 pattern, because the latter is periodic within a single byte's 256-value range and can produce false "still present" matches when checking for overwritten (oldest) content — this was caught during test-writing (see Deviations).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed a self-inflicted test bug: periodic byte pattern in `TestWindowRingBound` produced false negatives for the "oldest bytes absent" assertion**
- **Found during:** Task 03-01-01 (writing `TestWindowRingBound`)
- **Issue:** The first draft used `byte((len(written)+i) % 256)` to generate "distinguishable" bytes, but any modulo-256 byte sequence is periodic with period 256, so a substring-based "oldest bytes must be absent" check could match a later, coincidentally-identical byte run elsewhere in the ring — the test failed even though the ring buffer implementation was correct.
- **Fix:** Rewrote the test to write uniquely-numbered ASCII labels per chunk (`|chunk%05d|`) padded with a neutral filler byte, so the oldest label is guaranteed to be a substring that cannot reappear later; kept the exact-length and exact-equality-to-trailing-bytes assertions as the primary proof and added explicit first-label-absent / last-label-present checks for clarity.
- **Files modified:** internal/session/window_test.go
- **Verification:** `go test ./internal/session/... -run TestWindowRingBound -v` passes
- **Committed in:** d338ada (Task 1 commit)

**2. [Rule 1 - Bug] Fixed a Go compile error: byte constant overflow**
- **Found during:** Task 03-01-01
- **Issue:** `byte(len(written)+i) % 256` applies `% 256` after the conversion to `byte`, and the untyped constant `256` does not fit in `byte`, so the build failed (`256 (untyped int constant) overflows byte`).
- **Fix:** Reordered to `byte((len(written) + i) % 256)`, computing the modulo before the conversion. (Superseded entirely by the labeled-chunk rewrite above, but noted as its own compile-error fix since it was found first.)
- **Files modified:** internal/session/window_test.go
- **Verification:** `go build ./...` then passed
- **Committed in:** d338ada (Task 1 commit)

**3. [Rule 3 - Blocking] Removed a stray `stripTelnetIAC` mention from window.go's doc comment**
- **Found during:** Task 03-01-01, running the acceptance-criteria source assertion
- **Issue:** The acceptance criterion `grep -c "stripTelnetIAC" internal/session/window.go` must return 0 (the telnet stripper must not be duplicated or even named in this file), but the doc comment on `ansiCSI` referenced `stripTelnetIAC` by name, making the grep return 1.
- **Fix:** Reworded the comment to refer to "the existing telnet IAC stripper in websocket.go" without naming the function.
- **Files modified:** internal/session/window.go
- **Verification:** `grep -c "stripTelnetIAC" internal/session/window.go` now returns 0
- **Committed in:** d338ada (Task 1 commit)

---

**Total deviations:** 3 auto-fixed (2 test-correctness bugs in my own new test code, 1 acceptance-criteria wording fix). No scope creep — all three are within the two files this plan's tasks already touch.
**Impact on plan:** None on the shipped behavior; all three were caught and fixed before the task's commit, so every commit in this plan's history has passing tests and satisfies its acceptance criteria as committed.

## Issues Encountered

- **`-race` cannot run on this Windows dev machine.** `go env` shows `CGO_ENABLED=0` and no `gcc` is on `PATH` anywhere on the box (checked `where gcc`, common MinGW locations). `go test ./internal/session/... -race` fails immediately with `go: -race requires cgo; enable cgo by setting CGO_ENABLED=1`, before any test runs — this is an environment property, not a code defect. All verification in this plan ran as plain `go test` (no `-race`); every test passes, including the concurrency-adjacent `TestWindowSurvivesDisconnect`, which drives the real `Manager.Disconnect` under its own locking. The new map (`outputWindow`) uses the exact same `m.mu` guard, `RLock`/`Lock` discipline already proven race-free for the sibling `autopilot` map — no new locking primitive was introduced. Logged in `.planning/phases/03-one-ai-decision/deferred-items.md` so plan 03-13's evidence capture (which the plan's own `<verification>` section says will run `-race -v` into `evidence/01-test-report.txt`) knows to run it wherever cgo is actually available (e.g., a Linux CI runner or the Railway staging build image).
- **Pre-existing, out-of-scope test failure in `internal/icm`.** `go test ./...` (full-repo baseline) shows `FAIL: TestHandlerRegistration/CANCEL` in `internal/icm/icm_test.go:790`. This package was never touched by this plan (`git status --short` before and after every commit shows only `internal/session/*` files). Per the executor's scope boundary, this was not fixed — logged in `deferred-items.md` for whichever later Phase 3 plan owns `internal/icm` changes.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The Go contract plan 03-08 depends on is in place and proven by test: `const RecentOutputWindowBytes = 8192`, `func stripANSI(b []byte) []byte`, `func (m *Manager) RecentOutputSnapshot(userID string) string`.
- `internal/session` package passes its full test suite (`go test ./internal/session/...`, all green, no `-race` available locally — see Issues Encountered).
- `go build ./...` and `git diff --stat go.mod frontend/package.json` (empty) both confirmed at the plan's final commit.
- No blockers for plan 03-08 (the driver) or any sibling wave-1 plan; this plan has no `depends_on` and modified only the three files in its `files_modified` list.

---
*Phase: 03-one-ai-decision*
*Completed: 2026-09-15*

## Self-Check: PASSED

- FOUND: internal/session/window.go
- FOUND: internal/session/window_test.go
- FOUND: internal/session/manager.go
- FOUND: .planning/phases/03-one-ai-decision/deferred-items.md
- FOUND: d338ada (Task 1 commit)
- FOUND: 9c2f24c (Task 2 commit)
- FOUND: 253b2b8 (deferred-items docs commit)
