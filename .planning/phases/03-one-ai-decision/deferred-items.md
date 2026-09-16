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

Note for 03-07: on this worktree's base (`a9563ce`), the `internal/icm` `TestHandlerRegistration/CANCEL` failure above is already resolved (a wave-1 commit, `dcb671d`, aligned the test with the retired `#CANCEL` handler) — `go test ./...` is fully green as of plan 03-07's execution. Leaving this entry in place for the historical record; no action needed by the verifier on this specific item.

## Plan 03-07

**Out-of-scope finding, documented in full in `03-07-SUMMARY.md` rather than fixed here:** `internal/auth/handler.go`'s `DEV MODE - OTP for %s: %s` lines (`Register` and `SendOTP`, the `else` branch of `if h.emailSender != nil && h.emailSender.IsConfigured()`) still print the one-time sign-in code directly. This branch only executes when SMTP is entirely unconfigured (`SMTPHost`/`SMTPUser`/`SMTPPass` all unset) — never true on staging or production, both of which require SMTP for email delivery to function at all. The plan's `<action>` and `<context>` named only the two `"STAGING: OTP sent to user, code: %s"` call sites (now routed through `logOTPIssued`, gated by `AUTH_LOG_OTP`); the DEV MODE lines were not among them and were left untouched. Flagged here for the Phase 3 security review (plan 03-13's agenda item for DR-2-01 closure) in case the owner wants this local-dev convenience closed off too — a one-line follow-up (route it through `logOTPIssued` or an equivalent gated helper).
