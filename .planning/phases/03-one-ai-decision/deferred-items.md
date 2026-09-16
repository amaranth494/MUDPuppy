# Deferred Items — Phase 03 (plan 03-01)

Out-of-scope discoveries found while executing 03-01-PLAN.md, logged per the
executor's scope-boundary rule (only auto-fix issues directly caused by the
current task's own changes).

## Pre-existing failure: `internal/icm` `TestHandlerRegistration/CANCEL`

`go test ./...` shows `--- FAIL: TestHandlerRegistration/CANCEL` in
`internal/icm/icm_test.go:790` ("Handler \"CANCEL\" not registered"). This
package was not touched by plan 03-01 (only `internal/session/*` was
modified — confirmed via `git status --short` before and after this plan's
commits). The failure pre-dates this plan and is unrelated to the ring
buffer / ANSI strip / manager changes made here. Left unfixed per the
executor's scope boundary; flagged for whichever later Phase 3 plan owns
`internal/icm` changes (per 03-PATTERNS.md, several plans modify or add
tests in this package).

## Environment limitation: `-race` requires cgo, unavailable on this dev machine

This worktree's Go toolchain has `CGO_ENABLED=0` and no `gcc` on `PATH` (not
MinGW, not any other toolchain). `go test ./internal/session/... -race`
fails immediately with `go: -race requires cgo; enable cgo by setting
CGO_ENABLED=1` before any test runs — this is a machine/environment
property, not a code defect. All of plan 03-01's task verification steps
that specify `-race` were run instead as plain `go test` (no `-race` flag);
every test passes. The code was written to the same lock discipline as the
existing `AutopilotStateFor`/`autopilot` map (verified pattern, RLock/Lock
via `m.mu`, no additional mutex), so there is no known race condition, but
this has not been confirmed by the race detector on this machine. Whichever
environment captures `evidence/01-test-report.txt` for plan 03-13 should
run `-race` there if cgo is available (e.g., Railway staging build image or
a CI runner with a C toolchain), since it is a stronger check than what
this executor could produce locally.
