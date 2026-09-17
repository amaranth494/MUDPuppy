---
phase: 05-coaching-channel
plan: 04
subsystem: ai-safety
tags: [rate-limiting, token-bucket, go, react, ai-settings]

# Dependency graph
requires:
  - phase: 05-coaching-channel
    provides: "05-02's gemini.ReviewAnswer.Blocked *bool fail-closed verdict and the driver's failure-kind/notice table shape this plan's rate-limited-ai kind joins"
provides:
  - "internal/driver/ratelimit.go: allowAISend/resetAISendLimiter, a per-user token bucket (session.RateLimiter) checked immediately before every AI Dispatch, independent of icm.Dispatcher's own never-reset circuit breaker"
  - "failureRateLimitedAI joins driver.go's transient failure-kind table with its own locked notice, so a refused send takes the repeated-failure road instead of disengaging on its first occurrence"
  - "store.AISettings.RateLimitPerSecond (*int, blank means DefaultAIRateLimitPerSecond=2, never unlimited) and its ResolveAISettings resolution, validated to 1-20 in internal/profiles/handler.go"
  - "AI Command Rate Limit (per second) field on AIPlayerPanel.tsx, a fourth sibling of Call Cap/Disengage Threshold"
affects: [05-11-security-review-and-close]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "A driver-owned per-user rate limiter (internal/driver package) that reuses an existing primitive (session.RateLimiter) from a different package, deliberately sharing no state with the ICM dispatcher's own circuit breaker, for a safety-limit fix that must not touch the dispatcher"
    - "A fourth blank-means-server-default AI setting following the identical *int / ResolvedAISettings.<X>Set pattern CallCap and DisengageThreshold already established"

key-files:
  created:
    - internal/driver/ratelimit.go
    - internal/driver/ratelimit_test.go
  modified:
    - internal/driver/driver.go
    - internal/driver/driver_test.go
    - internal/driver/loop.go
    - internal/driver/loop_test.go
    - internal/store/profile.go
    - internal/store/profile_test.go
    - internal/profiles/handler.go
    - internal/profiles/handler_test.go
    - frontend/src/components/AIPlayerPanel.tsx
    - frontend/src/types/index.ts

key-decisions:
  - "Server default 2 commands/second (store.DefaultAIRateLimitPerSecond); accepted range 1-20 when set; blank never means unlimited, unlike CallCap"
  - "The limiter lives entirely in internal/driver/ratelimit.go, reusing session.NewRateLimiter(n, time.Second); internal/icm/dispatcher.go is untouched (confirmed by an empty git diff --stat)"
  - "rate-limited-ai notice: \"AI decision failed: the AI's commands were coming too fast and this one was held back. Autopilot disengaged.\" (full form); the sub-threshold running-count form reuses kindSentences exactly like every other transient kind"
  - "resetAISendLimiter is called from StopLoop (loop.go) so a new stint starts with a full bucket, per the plan's own 'wherever a stint ends' instruction"
  - "The store.ResolvedAISettings/AISettings/DefaultAIRateLimitPerSecond scaffolding that task 05-04-02 describes was built in task 05-04-01's commit instead, as a Rule 3 blocking-issue fix -- ratelimit.go's allowAISend signature could not compile without it. Task 05-04-02's own commit then only adds validateAISettings' bounds check, the saved-log line's rate_limit_blank flag, and the two tests the plan's action text calls for."
  - "testProfile() (internal/driver/driver_test.go) now sets a generous rate limit (1,000,000/sec) so every pre-05-04 test's tight decision loop keeps exercising call cap/disengage threshold/pacing rather than incidentally tripping the new default 2/sec limiter"

patterns-established:
  - "When a plan's own tasks are internally dependency-ordered in a way the file-list split doesn't reflect (task N's new file needs a struct field task N+1 is nominally responsible for), pull the minimal compiling scaffolding forward into task N as a Rule 3 fix and document the split explicitly in both commits and the SUMMARY, rather than leaving task N uncompilable until task N+1 lands"

requirements-completed: [REQ-safety-limits-hold]

# Metrics
duration: ~17min
completed: 2026-09-17
---

# Phase 5 Plan 04: AI Command Rate Limit Summary

**A per-user token bucket in the driver refuses a flood of AI-issued commands before they reach icm.Dispatcher, backed by a fourth configurable AI setting (blank = server default 2/sec, range 1-20) surfaced on the AI Player settings panel**

## Performance

- **Duration:** ~17 min (commit-to-commit, base e5a5083 to 11a85d1)
- **Started:** 2026-09-17T12:03:26-07:00 (worktree base commit)
- **Completed:** 2026-09-17T12:19:39-07:00
- **Tasks:** 3
- **Files modified:** 12 (2 created, 10 modified; 4 of those 10 -- driver_test.go, loop.go, loop_test.go, handler_test.go -- were touched beyond each task's own `<files>` list; see Deviations)

## Accomplishments

- D-25/DR-4-03 closed in code: every AI-issued command is now counted against a per-user speed limit (`internal/driver/ratelimit.go`'s `allowAISend`) immediately before `d.commands.Dispatch`, so a flood is refused before it ever reaches the game -- proven by `TestAllowAISend_FloodRefused` PASS.
- The fix deliberately shares no state with `icm.Dispatcher`'s own circuit breaker (whose pass-through return before `RecordExecution` and never-reset counter are exactly why the old limiter never caught AI sends) -- proven by an empty `git diff --stat -- internal/icm/` across all three commits.
- A refused send is transient, not a first-strike disengage: `failureRateLimitedAI` ("rate-limited-ai") joins `transientFailureKinds` with its own notice, taking the same repeated-failure road as every other transient kind -- proven by `TestDriverRateLimitRefusalIsTransient` PASS (one command sent, one refused, zero disengage calls, the locked running-count notice).
- The limit is the owner's own AI setting: `store.AISettings.RateLimitPerSecond` (*int), blank resolving to `store.DefaultAIRateLimitPerSecond` (2/sec, never unlimited) via `ResolveAISettings`, validated to 1-20 in `internal/profiles/handler.go`, and round-tripping through the existing `ai_settings` JSONB column with zero migration -- proven by `TestResolveAISettings_RateLimitBlankMeansServerDefault` and `TestAISettingsRateLimitBounds` PASS.
- The owner can type the limit into AI Player settings: a fourth `AI Command Rate Limit (per second)` field on `AIPlayerPanel.tsx`, the three 05-UI-SPEC.md strings shipped verbatim, no existing field's markup, order or styling touched, no new CSS class, `npm run build` exits 0.

## Task Commits

1. **Task 05-04-01: The AI's own sends are counted, and a flood is refused before it reaches the game** - `cb6e5db` (feat)
2. **Task 05-04-02: The speed limit is an AI setting on the profile, blank meaning the server's own default** - `4b2b8b4` (feat)
3. **Task 05-04-03: The owner can type the limit into the AI settings he already uses** - `11a85d1` (feat)

_No plan-metadata commit is made by this executor: per this worktree's instructions, STATE.md/ROADMAP.md are the orchestrator's to update after all wave agents complete._

## Files Created/Modified

- `internal/driver/ratelimit.go` - `allowAISend`/`resetAISendLimiter`, a per-user `aiSendLimiter` map guarded by `d.mu`, lazily built/rebuilt on `session.NewRateLimiter(limit, time.Second)`
- `internal/driver/ratelimit_test.go` - `TestAllowAISend_FloodRefused`, `TestAllowAISend_BlankMeansServerDefault`, `TestAllowAISend_LimitChangeRebuildsBucket`, `TestDriverRateLimitRefusalIsTransient`
- `internal/driver/driver.go` - `failureRateLimitedAI` constant; entries in `failureNotices`/`kindSentences`/`transientFailureKinds`; doc-comment notes on `failureEventOutcome`/`failureDisengageCause` explaining its deliberate absence there; `allowAISend` call immediately before `d.commands.Dispatch`; `aiSendLimiters` field and its `New()` initialization
- `internal/driver/loop.go` - `StopLoop` now calls `resetAISendLimiter` so the next stint starts with a full bucket
- `internal/driver/driver_test.go` - `testProfile()` now sets a generous rate limit so pre-existing tests are unaffected by the new default limiter (see Deviations)
- `internal/driver/loop_test.go` - one comment corrected to stop claiming `testProfile()`'s `AISettings{}` is a zero value (it no longer is, for the rate limit field only)
- `internal/store/profile.go` - `AISettings.RateLimitPerSecond *int` (json `rate_limit_per_second`); `DefaultAIRateLimitPerSecond = 2`; `ResolvedAISettings.RateLimitSet`/`RateLimitPerSecond`; `ResolveAISettings` extended with the same three-line shape `CallCap` uses
- `internal/store/profile_test.go` - `TestResolveAISettings_RateLimitBlankMeansServerDefault`
- `internal/profiles/handler.go` - `validateAISettings` rejects a set rate limit outside 1-20; the `[AI-PLAYER] ai settings saved` log line gains `rate_limit_blank=%t`
- `internal/profiles/handler_test.go` - `TestAISettingsRateLimitBounds` (0, 21, negative rejected; nil, 1, 20 accepted)
- `frontend/src/types/index.ts` - `AISettings.rate_limit_per_second: number | null`
- `frontend/src/components/AIPlayerPanel.tsx` - `rateLimitStr` state, load/save wiring identical to `callCapStr`'s, and the fourth `.form-group` field with the three locked strings

## Decisions Made

- Server default 2 commands/second, accepted range 1-20 when set, blank never means unlimited (D-25's explicit condition, Claude's Discretion in the plan).
- The limiter lives entirely in `internal/driver/ratelimit.go`; `internal/icm/dispatcher.go` is untouched, confirmed mechanically by an empty `git diff --stat -- internal/icm/` at every commit in this plan.
- `resetAISendLimiter` is wired from `StopLoop` (a stint ending), matching the plan's "called wherever a stint ends" instruction, so a fresh `#AUTO ON` or a resume never inherits tokens the prior stint already spent.
- The `store.ResolvedAISettings`/`AISettings`/`DefaultAIRateLimitPerSecond` scaffolding task 05-04-02's own action text describes was built in task 05-04-01's commit instead (see Deviations) -- `ratelimit.go`'s `allowAISend(userID string, resolved store.ResolvedAISettings) bool` signature and its `store.DefaultAIRateLimitPerSecond` reference could not compile or be tested otherwise, and task 05-04-01's own acceptance criteria require `go test ./internal/driver/...` to pass before task 05-04-02 ever runs.
- `testProfile()`'s AI settings now carry an explicit, very high rate limit (1,000,000/sec) rather than leaving the field blank, so every pre-existing test's tight loop of `HandleEngage`/`EngageLoop` calls (which previously assumed no cap existed at all) keeps testing what it always tested rather than incidentally tripping the new default 2/sec limiter within the same wall-clock second.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `internal/store/profile.go`'s rate-limit scaffolding built one task early**
- **Found during:** Task 05-04-01, while writing `internal/driver/ratelimit.go`
- **Issue:** `allowAISend`'s signature and body reference `store.ResolvedAISettings.RateLimitSet`/`RateLimitPerSecond` and `store.DefaultAIRateLimitPerSecond`, none of which exist yet -- those additions are task 05-04-02's action, but task 05-04-01's own file list (`internal/driver/ratelimit.go, internal/driver/ratelimit_test.go, internal/driver/driver.go, internal/driver/driver_test.go`) does not include `internal/store/profile.go`, and task 05-04-01's acceptance criteria require the driver package to build and its tests to pass before task 05-04-02 ever begins.
- **Fix:** Added the minimal compiling scaffolding to `internal/store/profile.go` in task 05-04-01's commit: `AISettings.RateLimitPerSecond *int` (with its json tag), `DefaultAIRateLimitPerSecond = 2`, `ResolvedAISettings.RateLimitSet`/`RateLimitPerSecond`, and `ResolveAISettings`'s three-line resolution logic for the new field -- i.e. the full field-and-resolution half of task 05-04-02's own action text. Task 05-04-02's commit then only added what remained: `validateAISettings`' bounds check, the `rate_limit_blank` log flag, and the plan's two named tests.
- **Files modified:** `internal/store/profile.go` (in task 05-04-01's commit, not task 05-04-02's)
- **Verification:** `go build ./...`, `go vet ./internal/driver/... ./internal/icm/...`, and `go test ./internal/driver/... ./internal/icm/... -count=1 -race` all green after the fix; task 05-04-02's own acceptance-criteria greps (`json:"rate_limit_per_second"`, `DefaultAIRateLimitPerSecond` counts, etc.) all still pass regardless of which commit introduced the strings
- **Committed in:** `cb6e5db` (Task 05-04-01 commit); noted explicitly in `4b2b8b4`'s (Task 05-04-02) commit message

**2. [Rule 1 - Bug] Pre-existing driver tests broken by the new default rate limiter**
- **Found during:** Task 05-04-01, first `go test ./internal/driver/...` run after wiring `allowAISend` into the Dispatch call site
- **Issue:** `TestHandleEngage/icm_refusal_sends_nothing`, `TestScriptedStint`, `TestLoop_Pacing/output_driven_decisions_follow_each_signal_after_settling` and `TestLoop_BlankSettings/blank_cap_runs_past_many_calls` all fire several `HandleEngage`/`EngageLoop` decisions within the same wall-clock second, using `testProfile()`'s blank (zero-value) `AISettings`. Once `allowAISend` was wired in, that blank setting resolved to the new server default of 2 commands/second, so the third-and-later decision in each of these tests was refused by the new limiter instead of reaching the behaviour the test actually meant to exercise (an ICM refusal, a scripted three-command stint, output-driven pacing, and an unbounded call cap, respectively) -- an out-of-scope regression these tests never anticipated.
- **Fix:** `testProfile()` (the one shared fixture all four affected tests use, directly or via `newPacedDriver`) now sets `AISettings.RateLimitPerSecond` to a deliberately generous 1,000,000/sec, restoring every pre-existing test's original assumption of no practical cap on AI sends, while `TestDriverRateLimitRefusalIsTransient` builds its own profile with an explicit low limit (1/sec) to exercise the limiter on purpose. One comment in `loop_test.go` that described `testProfile()`'s `AISettings{}` as a zero value was corrected to stay accurate.
- **Files modified:** `internal/driver/driver_test.go`, `internal/driver/loop_test.go`
- **Verification:** `go test ./internal/driver/... -count=1 -race` green (all four previously-failing tests pass; no other test's behaviour changed)
- **Committed in:** `cb6e5db` (Task 05-04-01 commit)

**3. [Rule 2 - Missing Critical] `internal/profiles/handler_test.go` gains the plan's own named validation test**
- **Found during:** Task 05-04-02
- **Issue:** The plan's action text explicitly calls for "a handler-level test beside the existing AI-settings validation tests asserting that 0, 21 and a negative value are rejected... and that nil and 1 and 20 are accepted," which only `internal/profiles/handler_test.go` can host -- but that file is absent from task 05-04-02's `<files>` list (`internal/store/profile.go, internal/store/profile_test.go, internal/profiles/handler.go`).
- **Fix:** Added `TestAISettingsRateLimitBounds` to `handler_test.go`, exactly as the plan's action text describes, following the file's existing `fakeProfileStore`/`newTestRequest`/`newTestProfile` idiom.
- **Files modified:** `internal/profiles/handler_test.go`
- **Verification:** `go test ./internal/profiles/... -count=1 -v -run TestAISettingsRateLimitBounds` green; no other test in the package affected
- **Committed in:** `4b2b8b4` (Task 05-04-02 commit)

---

**Total deviations:** 3 auto-fixed (1 blocking, 1 bug, 1 missing-critical)
**Impact on plan:** All three were necessary for the plan's own stated acceptance criteria to be reachable at all (the driver package would not compile, four pre-existing tests would regress, and the plan's own named validation test would have nowhere to live). No scope creep beyond what each task's action text already called for; no behaviour outside D-25/DR-4-03 was touched.

## Issues Encountered

- The rate-limit hint string in `AIPlayerPanel.tsx` was first written wrapped across two JSX lines (matching the neighbouring Disengage Threshold hint's own wrapping); this made the plan's literal single-line `grep -c` acceptance check for the full sentence return 0. Rewrapped onto one line; the other hints' wrapping was left untouched (Regression Invariant: no existing field's markup changed).
- `npm run build` regenerates `public/assets/index-*.js`/`.css` and `public/index.html` with a new content hash; per this machine's standing instructions, that generated output was reverted after each build (`git checkout -- public/assets/index-DdCmptsN.css public/index.html`, deleting the new hashed `.js`/`.map`) since this plan does not call for shipping a rebuilt bundle.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- D-25/DR-4-03 is closed in code and covered by unit tests on both the driver and store/handler sides; the plan's own three named tests (`TestAllowAISend_FloodRefused`, `TestAllowAISend_BlankMeansServerDefault`, `TestResolveAISettings_RateLimitBlankMeansServerDefault`) all PASS, plus the fourth (`TestDriverRateLimitRefusalIsTransient`) and the handler-level bounds test.
- The staging screenshot proving the field's presence and behaviour (`evidence/12-ai-settings-rate-limit.png`) is plan 05-11's to capture, per this plan's own `<output>` instruction; no live model call and no deploy were made here.
- `internal/icm/dispatcher.go` is confirmed untouched by this plan (empty `git diff --stat`), so hand-typed play's dispatcher behaviour is unaffected.
- `go.mod` and `frontend/package.json` are untouched; no dependency was added.

## Self-Check: PASSED

- FOUND: internal/driver/ratelimit.go
- FOUND: internal/driver/ratelimit_test.go
- FOUND: internal/driver/driver.go
- FOUND: internal/driver/driver_test.go
- FOUND: internal/driver/loop.go
- FOUND: internal/driver/loop_test.go
- FOUND: internal/store/profile.go
- FOUND: internal/store/profile_test.go
- FOUND: internal/profiles/handler.go
- FOUND: internal/profiles/handler_test.go
- FOUND: frontend/src/components/AIPlayerPanel.tsx
- FOUND: frontend/src/types/index.ts
- FOUND: commit cb6e5db
- FOUND: commit 4b2b8b4
- FOUND: commit 11a85d1

---
*Phase: 05-coaching-channel*
*Completed: 2026-09-17*
