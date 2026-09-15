# Deferred Items — Phase 01 (Profile Foundation and Policy Gate)

Items discovered during execution that are out of scope for the current plan and were not fixed (scope boundary rule).

| Category | Item | Found during | Status |
|----------|------|---------------|--------|
| Pre-existing test failure | `go test ./...` shows `TestHandlerRegistration/CANCEL` failing in `internal/icm/icm_test.go` ("Handler \"CANCEL\" not registered"). Unrelated to `internal/policy`; pre-exists this plan's changes. | Plan 01-02, Task 01-02-02 verification (`go test ./...`) | Deferred — not touched, out of scope for this plan |
