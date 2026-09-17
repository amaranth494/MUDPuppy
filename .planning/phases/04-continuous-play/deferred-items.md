# Deferred Items -- Phase 4 (continuous-play)

Out-of-scope discoveries logged during plan execution, per the executor's
scope-boundary rule (only auto-fix issues directly caused by the current
task's own changes).

## 04-10: Flaky test observed in `internal/driver` (out of scope)

- **Found during:** Task 04-10-01/02 verification (`go test ./internal/... -count=1`)
- **Symptom:** One run of the full suite showed `FAIL github.com/amaranth494/MudPuppy/internal/driver` with `[AI-PLAYER] decision ... model=test-model-not-a-real-name outcome= failure= ...` log lines suggesting a timing-sensitive assertion in the loop/pacing suite (plans 04-03/04-04). Three immediate reruns of `go test ./internal/driver/... -count=1` and one full-suite rerun were all green.
- **Why deferred:** This plan (04-10) touches only `scripts/verify-phase4.sh`, its two fixture directories, and (as a Rule 2 deviation) `internal/profiles/handler.go`/`handler_test.go` for the goal endpoint's new `quest` field. Nothing in this plan's diff touches `internal/driver`, so the flake is pre-existing infrastructure from an earlier Wave 2/3 plan, out of this task's scope to chase down.
- **Resolved 2026-09-16 (orchestrator, between waves 8 and 9):** Reproduced at about 1 failure in 30 runs: `TestLoop_Pacing/output_driven_decisions_follow_each_signal_after_settling`, "expected exactly 3 sent commands, got 2". Cause was in the test, not the loop: `waitForCalls(t, models, 3)` returns when the third model call starts, and the test read `sendCalls()` before that decision's review and send had finished. Fix: `waitForSendCount(t, sessions, 3)` before `StopLoop`. After the fix `go test ./internal/driver/... -count=120` and `-race -count=30` are green. No production code changed.
- **Original action:** Not fixed. Recorded here for the Phase 4 security/close review and for plan 04-11's full-suite run to watch for a repeat; if it recurs there, it should be investigated as part of that plan's own verification rather than patched blind.
