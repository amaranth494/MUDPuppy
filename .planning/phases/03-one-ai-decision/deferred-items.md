# Deferred Items — Phase 03 (one-ai-decision)

Items discovered during execution that are out of scope for the current plan/task
and are logged here rather than fixed, per the executor's scope-boundary rule.

## Plan 03-03

- **Pre-existing test failure: `TestHandlerRegistration/CANCEL`** (`internal/icm/icm_test.go:790`).
  `internal/icm/dispatcher.go:170` comments out registration of the `CANCEL` timer
  handler ("PR02PH09: User does not need #CANCEL - commented out") but the existing
  test `TestHandlerRegistration` still asserts it is registered. Confirmed pre-existing
  by removing plan 03-03's new `dispatcher_test.go` and re-running `go test ./internal/icm/...`
  — the failure persists with zero files changed by this plan. Out of scope for 03-03-01
  (files_modified only lists `internal/icm/dispatcher_test.go`); not fixed here.

- **`-race` unavailable in this environment.** `go test ./internal/icm/... -run
  TestDispatch_AutomationPassThrough -race -v` (the exact command in 03-VALIDATION.md)
  fails with `go: -race requires cgo; enable cgo by setting CGO_ENABLED=1` because no
  C compiler (gcc) is present on this Windows/Git-Bash worktree. Verified the test
  passes without `-race`: `go test ./internal/icm/... -run TestDispatch_AutomationPassThrough -v`
  — all three subtests PASS. Flagging for the phase-level evidence capture step (plan
  03-13), which may run on a host with cgo available.
