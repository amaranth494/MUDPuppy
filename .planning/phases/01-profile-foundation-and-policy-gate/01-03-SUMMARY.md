---
phase: 01-profile-foundation-and-policy-gate
plan: 03
subsystem: api
tags: [go, net-http, httptest, rest-sub-resource, policy-gate]

# Dependency graph
requires:
  - phase: 01-profile-foundation-and-policy-gate (plan 01)
    provides: "store.AISettings, Profile/ProfileUpdate AI fields, store.AcceptPolicy, store.EngageGateAllowed, store.EngageGateRefusalMessage"
  - phase: 01-profile-foundation-and-policy-gate (plan 02)
    provides: "internal/policy.Text() and internal/policy.Version()"
provides:
  - "Five HTTP endpoints under /api/v1/profiles/{connection_id}/{ai-settings,policy,policy/accept,engage-gate} exercising the whole phase's capability over the wire"
  - "Four fixed [AI-PLAYER] log format strings that are the phase's evidence surface for plan 01-05"
  - "internal/profiles/handler_test.go — eight httptest-based tests proving round-trip, blank round-trip, validation, acceptance-forgery refusal, one-time acceptance, both gate outcomes, and log-line emission without data leakage"
affects: [01-04, 01-05, 01-06]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "profileStorage interface in internal/profiles/handler.go: an unexported interface over the four store methods this package calls, satisfied by *store.ProfileStore, so the handler is testable with a hand-written fake and no live Postgres — the first such boundary in this package"
    - "Server-derived-only write: AcceptPolicy never decodes the request body; the version comes solely from policy.Version(), closing the acceptance-forgery threat by construction rather than by validation"
    - "wasAccepted-before-write pattern: AcceptPolicy captures profile.PolicyAcceptedAt != nil before calling the store, so the first-acceptance and already-accepted log lines are mutually exclusive by construction"

key-files:
  created:
    - internal/profiles/handler_test.go
  modified:
    - internal/profiles/handler.go
    - cmd/server/main.go

key-decisions:
  - "profileStorage interface introduced solely for testability; NewHandler's signature (*store.ProfileStore) is unchanged so cmd/server/main.go needed no constructor-call changes"
  - "AISettingsResponse doubles as both the GET response and the PUT request body, matching the existing Timers/Aliases/Triggers pattern in this file"
  - "The settings-saved log line placement stays inside PutAISettings (added in task 01-03-01, filled in task 01-03-02) rather than as a separate helper, keeping the four evidence lines co-located with the handlers that emit them"

patterns-established:
  - "Every new handler resolves through h.getProfileByConnectionID(r) exactly like the four pre-existing sub-resources — no ad hoc query, closing the IDOR threat (T-1-03) by reuse rather than new code"
  - "[AI-PLAYER] log lines carry only ids, a policy version, a timestamp, and booleans — never conduct rules, approach guidance, model names, or policy text (T-1-11), enforced here and asserted by TestAIPlayerLogLinesAreEmitted"

requirements-completed: [REQ-profile-ai-fields, REQ-policy-gate]

# Metrics
duration: ~20min
completed: 2026-09-15
---

# Phase 1 Plan 3: AI Settings, Policy Gate and Engage Gate Go Live Over HTTP Summary

**Five endpoints (`GET/PUT ai-settings`, `GET policy`, `POST policy/accept`, `GET engage-gate`) route through the existing session-scoped mux with server-derived-only acceptance and four fixed `[AI-PLAYER]` log lines, proven by eight new `httptest` tests running against a hand-written fake store.**

## Performance

- **Duration:** ~20 min
- **Tasks:** 3 completed
- **Files modified:** 3 (1 new test file, 2 modified source files)

## Accomplishments

- `internal/profiles/handler.go` gained a `profileStorage` interface (the package's first store-boundary abstraction) so the handler package can be tested without Postgres, plus `GetAISettings`/`PutAISettings` (length-bounded, blank-preserving), `GetPolicy`, `AcceptPolicy` (POST-only, server-version-derived, request body never decoded), and `GetEngageGate` (reads only the server-fetched row, never a client-supplied flag).
- Four `[AI-PLAYER]` log lines — policy accepted, policy already accepted, engage gate decision, AI settings saved — are emitted at the exact points plan 01-05 will grep out of the staging log, and are proven by source assertion and by `TestAIPlayerLogLinesAreEmitted` to never interpolate conduct rules, approach guidance, or policy text.
- `cmd/server/main.go` registers the four new route blocks (`ai-settings` GET/PUT, `policy` GET, `policy/accept` POST-only, `engage-gate` GET) inside the same `sessionMiddleware`-wrapped mux as every other profile sub-resource — no separate auth wiring was needed or added.
- `internal/profiles/handler_test.go` (new, the first `_test.go` file for this package) proves round-trip, blank round-trip, over-length rejection with the exact error body, acceptance-forgery refusal, one-time acceptance echoing the server version and original timestamp on a repeat POST, both engage-gate outcomes, and the four log lines' presence without leaking profile text — all eight tests pass.

## Task Commits

Each task was committed atomically:

1. **Task 01-03-01: The AI settings sub-resource reads and writes the three editable fields** - `011d920` (feat)
2. **Task 01-03-02: The policy is served, accepted once, the gate answers over HTTP, and each event logs one evidence line** - `bff1a7f` (feat)
3. **Task 01-03-03: Routes are registered and httptest proves round-trip, validation and the gate** - `b77ece3` (feat)

## Files Created/Modified

- `internal/profiles/handler.go` - `profileStorage` interface; `AISettingsResponse`, `PolicyResponse`, `EngageGateResponse`; `GetAISettings`/`PutAISettings`/`GetPolicy`/`AcceptPolicy`/`GetEngageGate`; `validateAISettings`; four `[AI-PLAYER]` log lines
- `cmd/server/main.go` - four new `mux.HandleFunc` route blocks for `ai-settings`, `policy`, `policy/accept`, `engage-gate`
- `internal/profiles/handler_test.go` - `fakeProfileStore` (hand-written, satisfies `profileStorage`); eight tests: `TestAISettingsRoundTrip`, `TestAISettingsBlankRoundTrip`, `TestAISettingsRejectsOverLengthText`, `TestAISettingsCannotSetAcceptance`, `TestPolicyAcceptUsesServerVersion`, `TestEngageGateHandlerRefusesWithoutAcceptance`, `TestEngageGateHandlerAllowsAfterAcceptance`, `TestAIPlayerLogLinesAreEmitted`

## Decisions Made

- `profileStorage` is declared as an unexported interface over exactly the four `*store.ProfileStore` methods this package calls (`GetProfile`, `GetProfileByConnection`, `UpdateProfile`, `AcceptPolicy`); `NewHandler`'s signature stays `*store.ProfileStore` so no caller needed to change.
- `AcceptPolicy` computes `wasAccepted` from the profile fetched *before* the store call, so the first-acceptance and already-accepted log branches are mutually exclusive by construction, not by re-checking after the write.
- The AI-settings-saved log line was written into `PutAISettings` as part of task 01-03-02 (per the plan's instruction to "leave a single place for it" in task 01-03-01), keeping the two tasks' commits atomic and independently buildable.

## Deviations from Plan

None — plan executed exactly as written. All source assertions in the plan's `<acceptance_criteria>` blocks (interface shape, exact log format strings, JSON tags, method-not-allowed guards, `grep` counts) were verified to hold post-implementation.

## Issues Encountered

`go test ./internal/icm/...` still fails on the pre-existing `TestHandlerRegistration/CANCEL` case at this plan's base commit — confirmed unrelated (no file this plan touches lives under `internal/icm`) and out of scope per the Scope Boundary rule; not fixed. `go test ./internal/profiles/... ./internal/store/... ./internal/policy/...` and `go build ./...` are both clean, and `git diff --stat go.mod` is empty (T-1-SC).

## User Setup Required

None — no external service configuration required. This plan adds no new dependency and no environment variable.

## Next Phase Readiness

- Plan 01-04 (frontend AI Player panel) can now call all five endpoints exactly as contracted in this plan's endpoint table.
- Plan 01-05 (staging evidence capture) has its four `[AI-PLAYER]` log format strings fixed and verified to carry ids, version, timestamp, and booleans only — safe to capture verbatim from the staging log.
- Plan 01-06 (canned report script) can drive the five endpoints directly; every error path still returns HTTP 400 with `{"error": "..."}`, matching every other profile sub-resource in this codebase (a known, pre-existing inconsistency, not changed here).

No blockers.

---
*Phase: 01-profile-foundation-and-policy-gate*
*Completed: 2026-09-15*

## Self-Check: PASSED

All created/modified files verified present (internal/profiles/handler.go, internal/profiles/handler_test.go, cmd/server/main.go, .planning/phases/01-profile-foundation-and-policy-gate/01-03-SUMMARY.md). All three task commit hashes (011d920, bff1a7f, b77ece3) verified present in git log.
