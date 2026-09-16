---
phase: 03-one-ai-decision
plan: 06
subsystem: api
tags: [go, net-http, session, config, engage-gate]

# Dependency graph
requires:
  - phase: 03-one-ai-decision
    plan: 02
    provides: "internal/config.Config.AIConfigured() — the nil-safe check on the env-sourced AI model registry"
provides:
  - "internal/session.Handler.Autopilot's second engage gate: refused-not-configured when h.config == nil || !h.config.AIConfigured()"
affects: [03-08-driver, 03-13-evidence-harness]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Two independent engage gates composed in series inside the same switch branch: the Phase 1 policy gate (gateAllowed) checked first, the Phase 3 AI-configuration gate second, each with its own fixed refusal string, neither composing the other's wording"

key-files:
  created: []
  modified:
    - internal/session/handler.go
    - internal/session/handler_test.go

key-decisions:
  - "Deviation from RESEARCH Pattern 6: no new HandlerCallbacks.AIConfigured callback was added. session.Handler already holds config *config.Config (handler.go:16-20), and plan 03-02 put a nil-safe AIConfigured() directly on *config.Config. Reading it in the handler is the same check with one less indirection, needs no cmd/server/main.go edit, and keeps this plan's files (internal/session/handler.go, internal/session/handler_test.go) disjoint from plan 03-05's and 03-07's files in the same wave, exactly as the plan's context section directed."
  - "The refusal reuses AutopilotResponse.GateMessage (not a new response field) for the D-20 sentence, so the browser's existing Phase 2 notice-rendering path needs no change — it already brackets whatever string arrives in gate_message."
  - "newAutopilotHandler's test fixture (used by every pre-existing Phase 1/2 gate/engage test) was switched from an empty &config.Config{} to a new configuredConfig() helper whose AIConfigured() reports true, so the new second gate does not turn every pre-existing engage-path test into a refused-not-configured result. TestAutopilotHandler_AIConfigRefusal builds its own handlers directly with configuredConfig()/unconfiguredConfig()/nil to vary AI configuration explicitly."

requirements-completed: [REQ-env-config]

duration: 20min
completed: 2026-09-16
---

# Phase 3 Plan 06: AI-Configuration Engage Refusal Summary

**A second gate in the `Autopilot` handler's `"on"` branch — checked only after the Phase 1 policy gate allows — refuses `#AUTO ON` with the locked sentence `Autopilot refused: AI is not configured on this server` whenever `h.config` is nil or `AIConfigured()` is false, returning before `EngageAutopilot` is ever called.**

## Performance

- **Duration:** 20 min
- **Started:** 2026-09-15 (post worktree-base reset to a9563ce)
- **Completed:** 2026-09-16T02:07:15Z
- **Tasks:** 1
- **Files modified:** 2

## Accomplishments

- `internal/session/handler.go`'s `Autopilot` handler now composes two independent engage gates in its `"on"` branch: the existing Phase 1 policy gate (`gateAllowed`, unchanged, still returns `refused-gate` with `store.EngageGateRefusalMessage` first) and a new Phase 3 configuration gate (`h.config == nil || !h.config.AIConfigured()`) checked only when the policy gate allows. The configuration refusal sets `Outcome: "refused-not-configured"`, `GateMessage: "Autopilot refused: AI is not configured on this server"` (locked, byte-for-byte, appears exactly once in the file), and returns before `EngageAutopilot` is called — the switch cannot move and no driver can be invoked on this path.
- One `[AI-PLAYER] autopilot user_id=%s connection_id=%s old=%s new=%s cause=refused-not-configured` log line is emitted per refusal, carrying no model name, endpoint, key or registry detail (verified by grep, see Self-Check).
- `#AUTO OFF` and the `"status"` action are untouched — neither reads `h.config` — so ordinary play and the badge do not depend on the AI registry.
- `AutopilotResponse`'s `Outcome` doc comment now lists `refused-not-configured` alongside the existing enum values.
- `TestAutopilotHandler_AIConfigRefusal` adds five subtests: `refuses_on_when_unconfigured` (also asserts the manager's state is still `AutopilotOff` after the call — "the switch stays off" mechanically), `nil_config_refuses` (no panic on a nil `*config.Config`), `gate_refusal_takes_precedence` (an unconfigured AI plus a refusing policy gate still returns `refused-gate` with the Phase 1 string, proving the two refusals never blur), `engages_when_configured` (a complete registry plus an allowing gate and a seeded connected session reaches `engaged`), and `off_is_unaffected_when_unconfigured` (`#AUTO OFF` with an unconfigured AI returns `disengaged`/`already-off`, never `refused-not-configured`).

## Task Commits

1. **Task 03-06-01: #AUTO ON is refused when the AI is not configured, and everything else about the switch is untouched** - `dda4bb4` (feat)

_No plan-metadata commit yet — this executor does not update STATE.md/ROADMAP.md (parallel worktree run); the orchestrator commits those after all wave agents complete._

## Files Created/Modified

- `internal/session/handler.go` - Adds the second engage gate to the `"on"` branch (`h.config == nil || !h.config.AIConfigured()`), the `refused-not-configured` outcome and its log line, and the updated `AutopilotResponse.Outcome` doc comment.
- `internal/session/handler_test.go` - Adds `configuredConfig()`/`unconfiguredConfig()` fixture helpers and `TestAutopilotHandler_AIConfigRefusal` (5 subtests); updates `newAutopilotHandler` to use `configuredConfig()` instead of an empty `&config.Config{}` so pre-existing Phase 1/2 gate/engage tests are unaffected by the new gate.

## Decisions Made

- Deviation from RESEARCH Pattern 6, as directed by the plan's context section: read `Config.AIConfigured()` directly from the `config *config.Config` field `session.Handler` already holds, instead of adding a new `HandlerCallbacks.AIConfigured` callback. One less indirection, no `cmd/server/main.go` edit, and keeps this plan's files disjoint from plan 03-05's and 03-07's in the same wave.
- Reused `AutopilotResponse.GateMessage` for the D-20 refusal sentence rather than adding a new response field, so the browser's existing Phase 2 bracketed-notice rendering needs no change.
- Switched `newAutopilotHandler`'s config fixture to a configured registry (see Deviations below) so the new gate does not regress every pre-existing gate/engage test that never cared about AI configuration.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Pre-existing Phase 1/2 autopilot handler tests failed after the new gate was added**
- **Found during:** Task 03-06-01 verification (`go test ./internal/session/...`)
- **Issue:** `TestAutopilotHandler` and `TestEngageRefusedForAnotherProfile` construct their handler via `newAutopilotHandler(m, gate)`, which built the handler with an empty `&config.Config{}`. Once the new configuration gate was added, that empty config's `AIConfigured()` correctly returns `false`, so every pre-existing "engages when gate passes and session connected" style test started returning `refused-not-configured` instead of `engaged`/`already-on`/etc. — a real regression the plan's own verification step ("the Phase 2 autopilot handler tests still pass") requires catching.
- **Fix:** Added a `configuredConfig()` test helper (a `*config.Config` whose registry has a default slug with a non-empty model name, endpoint and key) and switched `newAutopilotHandler`'s fixture to it, so tests that are about gate/engage mechanics — not about AI configuration — are unaffected by the new gate. `TestAutopilotHandler_AIConfigRefusal` builds its own handlers directly with `configuredConfig()`/`unconfiguredConfig()`/`nil` to vary AI configuration explicitly.
- **Files modified:** internal/session/handler_test.go
- **Verification:** `go test ./internal/session/...` passes with no failures (see Self-Check).
- **Committed in:** dda4bb4 (Task 1 commit — fixed before commit, not a separate commit)

---

**Total deviations:** 1 auto-fixed (1 bug, caught during self-verification before commit)
**Impact on plan:** None on the shipped behavior — the fix only corrects test fixtures to keep pre-existing coverage green; the production gate logic is exactly as specified.

## Issues Encountered

- **`-race` unavailable in this environment:** consistent with plan 03-02's note, this Windows Git Bash environment has `CGO_ENABLED=0` with no `gcc` on `PATH`, so `go test -race` cannot run. Ran `go test ./internal/session/...` and `go test ./internal/session/... -run TestAutopilotHandler_AIConfigRefusal -v` without `-race` instead; both pass with no `DATA RACE` tooling available to check. Flagging so plan 03-13's evidence-capture step (if it runs where cgo is available) can attempt `-race` there.

## User Setup Required

None — this plan makes no environment-configuration changes. The `AI_MODEL_*` environment variables plan 03-02 introduced still need to be set on Railway staging before `#AUTO ON` can actually engage there; until then, the refusal built in this plan is exactly what staging will exhibit, and is one of the few Phase 3 behaviors verifiable on staging before those variables are set.

## Next Phase Readiness

- The `refused-not-configured` outcome and its locked message are ready for plan 03-13's evidence harness to capture as `evidence/08-refused-not-configured.png` and the `TestAutopilotHandler_AIConfigRefusal` PASS lines in `evidence/01-test-report.txt`.
- No blockers for downstream Phase 3 plans introduced by this plan. This plan's files (`internal/session/handler.go`, `internal/session/handler_test.go`) do not overlap with plan 03-05 (`internal/session/manager.go`, `transcript.go`, `internal/store`, `internal/profiles`, migrations, `cmd/server/main.go`) or plan 03-07 (`internal/config/config.go`, `internal/auth/*`).

---
*Phase: 03-one-ai-decision*
*Completed: 2026-09-16*

## Self-Check: PASSED

- `internal/session/handler.go` FOUND (modified, verified on disk)
- `internal/session/handler_test.go` FOUND (modified, verified on disk)
- Commit `dda4bb4` FOUND in `git log --oneline --all`
- `grep -c "Autopilot refused: AI is not configured on this server" internal/session/handler.go` = 1
- `grep "\[AI-PLAYER\] autopilot" internal/session/handler.go | grep -ciE "model|key|endpoint|registry"` = 0
- `git diff --stat go.mod frontend/package.json` = empty
- `go build ./...` exits 0; `go vet ./...` exits 0
- `go test ./internal/session/...` passes with no failures (all 5 `TestAutopilotHandler_AIConfigRefusal` subtests PASS, plus all pre-existing session package tests)
- No unexpected file deletions or untracked files after commit
