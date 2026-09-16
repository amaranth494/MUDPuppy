# Deferred Items — Phase 03 (one-ai-decision)

Out-of-scope discoveries logged during plan execution, per the executor's scope-boundary rule (only auto-fix issues directly caused by the current task's own changes). Consolidated by the orchestrator after wave 1; plans 03-01, 03-02 and 03-03 each recorded the same two findings independently.

## Pre-existing failure: `internal/icm` `TestHandlerRegistration/CANCEL`

Reported by 03-01, 03-02 and 03-03.

`go test ./internal/icm/...` shows `--- FAIL: TestHandlerRegistration/CANCEL` (`internal/icm/icm_test.go:790`, "Handler \"CANCEL\" not registered"). `internal/icm/dispatcher.go:170` comments out registration of the `CANCEL` timer handler ("PR02PH09: User does not need #CANCEL - commented out") but the existing test still asserts it is registered. Traced to commit `9066456` ("Debug: Remove #CANCEL from available commands per user request"), which predates Phase 3 execution; confirmed present on the pre-plan baseline `6e98595` with zero Phase 3 files changed.

Not fixed by any wave 1 plan (out of each plan's `files_modified`). Flagged for the phase verifier and for the post-merge test gate; the orchestrator decides at wave close whether to fix it as a post-merge deviation or carry it to the phase security/verification agenda.

## Environment limitation: `-race` requires cgo, unavailable on this dev machine

Reported by 03-01, 03-02 and 03-03.

The local Go toolchain has `CGO_ENABLED=0` and no `gcc` on `PATH`, so every `go test ... -race` invocation named in the plans and in 03-VALIDATION.md fails immediately with `go: -race requires cgo; enable cgo by setting CGO_ENABLED=1`. All affected tests were run as plain `go test` instead and pass (03-01 window tests, 03-02 config and gemini tests, 03-03 `TestDispatch_AutomationPassThrough` with all three subtests). The new code follows the existing lock discipline (`m.mu` RLock/Lock on the session Manager), so there is no known race, but the race detector has not confirmed it on this machine.

Whichever environment captures `evidence/01-test-report.txt` for plan 03-13 should run `-race` there if cgo is available (Railway build image or a CI runner with a C toolchain); otherwise the report must state plainly that `-race` was not run.

## Plan 03-05

Reported by 03-05 (session transcript).

- **`-race` still unavailable.** Same environment limitation as above; `go test ./internal/session/... -run TestTranscript -race -v` and the plan's other `-race` invocations were run as plain `go test ... -v` instead. All pass, including the concurrency-adjacent `TestTranscriptDoesNotBlockReader` (enqueue under a permanently-blocked sink call) and the whole-package `go test ./internal/session/...`. The new `transcriptSession` type uses its own `sync.Mutex` (`muPending`) plus `sync/atomic` counters (`seq`, `dropped`), never touching `Manager`'s `m.mu` from inside the write-loop goroutine, so there is no new cross-lock ordering — but the race detector has not confirmed this on this machine.
- **Pre-existing `internal/icm` `TestHandlerRegistration/CANCEL` failure no longer reproduces.** `go test ./...` at this plan's base (`a9563ce`, post-wave-1) shows `internal/icm` passing cleanly; whichever wave 1 plan (03-02 or 03-03) touched the CANCEL handler registration appears to have resolved it before this plan started. Not re-verified in depth since `internal/icm` is outside this plan's `files_modified`; noting here so the orchestrator can drop the now-stale flag above if wave 2/3 confirms it independently.
- **Fixed in-scope (not deferred): a latent bug in `internal/profiles/handler_test.go`.** `TestAIPlayerLogLinesAreEmitted` did `defer log.SetOutput(nil)`, which corrupts the global `log` package output for every test that runs afterward in the same test binary — harmless while it was the last test in the file, but this plan's new `internal/profiles/logs_test.go` runs after it alphabetically and its handlers call `log.Printf`, so `go test ./internal/profiles/...` panicked with a nil-writer dereference until fixed. Corrected to capture and restore the previous writer (`prevOutput := log.Writer()` / `defer log.SetOutput(prevOutput)`) rather than deferring it out-of-scope per Rule 3 (blocking issue directly triggered by this task's new code in the same package/test binary). Committed alongside this plan's task 3 commit.
