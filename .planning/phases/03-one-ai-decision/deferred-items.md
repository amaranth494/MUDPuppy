# Deferred Items — Phase 3 (one-ai-decision)

Out-of-scope discoveries logged during plan execution, per the executor's scope boundary rule (only auto-fix issues directly caused by the current task's changes).

## From plan 03-02

- **Pre-existing failing test:** `internal/icm` `TestHandlerRegistration/CANCEL` fails (`icm_test.go:790: Handler "CANCEL" not registered`) on the pre-plan baseline (commit `6e98595`, before any 03-02 changes). Traced to commit `9066456` ("Debug: Remove #CANCEL from available commands per user request"), which predates Phase 3 execution. `internal/icm` is not in plan 03-02's `files_modified` list and was not touched by this plan. Not fixed here — out of scope for 03-02 (internal/config, internal/gemini only). Flagged for the phase verifier / whichever plan owns `internal/icm` in this phase.
