# Deferred Items -- Phase 4 (continuous-play)

Out-of-scope discoveries logged during plan execution, per the executor's
scope-boundary rule (only auto-fix issues directly caused by the current
task's own changes).

## 04-10: Flaky test observed in `internal/driver` (out of scope)

- **Found during:** Task 04-10-01/02 verification (`go test ./internal/... -count=1`)
- **Symptom:** One run of the full suite showed `FAIL github.com/amaranth494/MudPuppy/internal/driver` with `[AI-PLAYER] decision ... model=test-model-not-a-real-name outcome= failure= ...` log lines suggesting a timing-sensitive assertion in the loop/pacing suite (plans 04-03/04-04). Three immediate reruns of `go test ./internal/driver/... -count=1` and one full-suite rerun were all green.
- **Why deferred:** This plan (04-10) touches only `scripts/verify-phase4.sh`, its two fixture directories, and (as a Rule 2 deviation) `internal/profiles/handler.go`/`handler_test.go` for the goal endpoint's new `quest` field. Nothing in this plan's diff touches `internal/driver`, so the flake is pre-existing infrastructure from an earlier Wave 2/3 plan, out of this task's scope to chase down.
- **Action:** Not fixed. Recorded here for the Phase 4 security/close review and for plan 04-11's full-suite run to watch for a repeat; if it recurs there, it should be investigated as part of that plan's own verification rather than patched blind.
