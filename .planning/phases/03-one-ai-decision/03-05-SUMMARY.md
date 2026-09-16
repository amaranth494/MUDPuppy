---
phase: 03-one-ai-decision
plan: 05
subsystem: session
tags: [go, postgres, session-transcript, websocket-tap, sql, http-api]

# Dependency graph
requires:
  - phase: 03-one-ai-decision (wave 1, plan 03-01)
    provides: "internal/session/window.go's stripANSI, and the Manager map-under-m.mu convention the transcript tap reuses"
  - phase: 03-one-ai-decision (wave 1, plan 03-03)
    provides: "the ICM engine wired into cmd/server/main.go, confirming main.go's store/handler construction block this plan extends"
provides:
  - "migrations/011: game_sessions, game_session_lines, ai_decisions (one up/down pair, all three of this phase's tables)"
  - "internal/store/transcripts.go: TranscriptStore (open, batched append, close, owner-scoped list and read)"
  - "internal/session/transcript.go: TranscriptSink interface, TranscriptLine, transcriptSession (non-blocking tap, batching writer goroutine)"
  - "internal/session/manager.go: transcripts map + transcriptSink field; Connect/Disconnect hooks; SendCommandAs(userID, command, source); autopilot engage/disengage/park/resume marker lines"
  - "internal/profiles/logs.go: ListSessions and GetSessionTranscript, owner-scoped over GET /api/v1/profiles/{connection_id}/sessions[/{session_id}]"
  - "cmd/server/main.go: transcriptStore constructed once, wired to both the session tap (via transcriptSinkAdapter) and the profiles handler"
affects: [03-08-driver-single-decision, 03-11-log-page]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "TranscriptSink interface declared in the consuming package (internal/session), never importing internal/store; cmd/server/main.go bridges the two shapes with a thin transcriptSinkAdapter"
    - "Sixth userID-keyed map on Manager under the existing m.mu, same 'lives here not on Session' discipline as autopilot and outputWindow"
    - "Buffered channel (512) + non-blocking enqueue + single batching writer goroutine per open transcript, so a database problem never blocks the MUD read/write path (T-3-26)"
    - "transcriptStorage interface in internal/profiles mirrors the existing profileStorage interface, so logs_test.go needs no live Postgres connection"

key-files:
  created:
    - migrations/011_add_ai_session_tables.up.sql
    - migrations/011_add_ai_session_tables.down.sql
    - internal/store/transcripts.go
    - internal/session/transcript.go
    - internal/session/transcript_test.go
    - internal/profiles/logs.go
    - internal/profiles/logs_test.go
  modified:
    - internal/session/manager.go
    - internal/profiles/handler.go
    - internal/profiles/handler_test.go
    - cmd/server/main.go

key-decisions:
  - "internal/session declares its own TranscriptLine type rather than importing internal/store's GameSessionLine (per the plan's own stated interface contract); cmd/server/main.go's transcriptSinkAdapter converts between the two structurally-identical-but-differently-named shapes, since Go interface satisfaction requires exact declared types, not structural equivalence, across package boundaries"
  - "closeTranscriptLocked blocks synchronously (flush + CloseGameSession) inside Disconnect's held m.mu, rather than spawning a goroutine — this keeps the open/close lifecycle deterministic and testable (closes_on_disconnect asserts the fake sink recorded a close immediately after Disconnect returns) and is a deliberate, occasional lifecycle event, not the hot per-line enqueue path that T-3-26 protects"
  - "enqueueTranscriptLineLocked (no locking) vs enqueueTranscriptLine (RLock-then-release) are two separate helpers because EngageAutopilot/DisengageAutopilot/parkAutopilotLocked/resumeAutopilotLocked already hold m.mu (sync.RWMutex is not reentrant), while SendCommandAs and feedTranscriptOutput do not"
  - "Added marker lines to parkAutopilotLocked ('[AI-ASSIST waiting]'), the resumeAutopilotLocked success path ('[AI-ASSIST resumed]'), and its connection-changed force-off path ('[AI-ASSIST disengaged: connection-changed]') in addition to the plan's explicitly-tested engage/disengage markers, so a WAITING stretch and an unexpected forced disengage both stay visible inside the transcript rather than only the two paths TestTranscriptStintMarkers checks"

patterns-established:
  - "A session transcript's open/close lifecycle is looked up by userID under Manager.mu and driven synchronously from Connect/Disconnect; per-line enqueue is the only non-blocking surface"

requirements-completed: [REQ-reasoning-visibility]

# Metrics
duration: ~75min
completed: 2026-09-15
---

# Phase 3 Plan 05: Session Transcript Summary

**Every saved-profile game connection is recorded from connect to disconnect — human, AI and game lines tagged apart, autopilot stint markers included — in three new Postgres tables, tapped through a non-blocking buffered-channel writer on the existing Connect/Disconnect/SendCommand choke points, and readable back by its owner over two new HTTP endpoints.**

## Performance

- **Duration:** ~75 min
- **Started:** 2026-09-15T19:05:00Z (approx.)
- **Completed:** 2026-09-15T19:22:00Z (approx.)
- **Tasks:** 3
- **Files modified:** 12 (9 created, 3 modified across tasks; a 4th modified file, `internal/profiles/handler_test.go`, was touched as an in-scope bug fix during task 3)

## Accomplishments
- One migration (`011`) creates `game_sessions`, `game_session_lines` and `ai_decisions` together, with the `CHECK` constraints, cascade/set-null foreign keys and indexes the plan specified — `ai_decisions` is created here but not written to until plan 03-08.
- `internal/store/transcripts.go` gives the codebase `TranscriptStore`: open, one-multi-row-INSERT batched append (never a per-line loop), close, and two owner-scoped read methods, matching `ProfileStore`'s exact constructor shape.
- `internal/session/transcript.go` + `manager.go` changes give every saved-profile `Connect` an open transcript and every `Disconnect` a closed one, with `SendCommand`/`SendCommandAs` tagging human vs. AI lines, `ReadOutput` feeding game lines (ANSI stripped, CR trimmed), and autopilot engage/disengage/park/resume leaving `[AI-ASSIST ...]` marker lines — all proven by `go test -run TestTranscript -v` (4 tests, 3 named subtests, all PASS).
- `internal/profiles/logs.go` gives the owner `GET .../sessions` and `GET .../sessions/{session_id}`, both resolving ownership through `GetProfileByConnection` before any row is read, with a session id belonging to another connection returning an empty `lines` array rather than another profile's text.
- `cmd/server/main.go` wires one `TranscriptStore` instance to two consumers — the session manager's write-side tap (via a small adapter) and the profiles handler's read endpoints — confirmed by a single `store.NewTranscriptStore(db)` call site.

## Task Commits

Each task was committed atomically:

1. **Task 03-05-01: The tables exist and a store can open, append to, close, list and read a transcript** - `48dca29` (feat)
2. **Task 03-05-02: Connecting a profile starts a transcript, disconnecting ends it, and every line is marked human, ai or game** - `93179e5` (feat)
3. **Task 03-05-03: The owner can list a profile's sessions and read one back over HTTP, and nobody else can** - `7bf9ab7` (feat)

## Files Created/Modified
- `migrations/011_add_ai_session_tables.up.sql` / `.down.sql` - the three tables, one pair
- `internal/store/transcripts.go` - `TranscriptStore`: `OpenGameSession`, `AppendGameLines`, `CloseGameSession`, `ListSessionsForConnection`, `GetSessionLines`
- `internal/session/transcript.go` - `TranscriptSink`, `TranscriptLine`, `transcriptSession` (enqueue/feedGameOutput/writeLoop/close)
- `internal/session/transcript_test.go` - `TestTranscriptOpensOnConnectAndClosesOnDisconnect`, `TestTranscriptLineSources`, `TestTranscriptStintMarkers`, `TestTranscriptDoesNotBlockReader`
- `internal/session/manager.go` - `transcripts`/`transcriptSink` fields, `SetTranscriptSink`, `openTranscriptLocked`/`closeTranscriptLocked`, `enqueueTranscriptLine`/`enqueueTranscriptLineLocked`, `feedTranscriptOutput`, `SendCommandAs` (with `SendCommand` as a wrapper), marker enqueues in the four autopilot transition functions
- `internal/profiles/logs.go` - `ListSessions`, `GetSessionTranscript`, `getSessionIDFromPath`
- `internal/profiles/logs_test.go` - `TestListSessionsScopedToOwner`, `TestGetSessionTranscript` (both with the two named subtests the plan specified)
- `internal/profiles/handler.go` - `transcriptStorage` interface, `Handler.transcripts` field, `NewHandlerWithTranscripts`
- `internal/profiles/handler_test.go` - one-line bug fix (see Deviations)
- `cmd/server/main.go` - `transcriptSinkAdapter`, `transcriptStore` construction, `SetTranscriptSink` wiring, `NewHandlerWithTranscripts` wiring, two new routes

## Decisions Made
- The plan's own `<context>` section anticipated that `internal/session`'s `TranscriptLine` and `internal/store`'s `GameSessionLine` might be "structurally identical," in which case no adapter would be needed. They are structurally identical but are two distinct named types in two different packages, and Go requires an interface's method signatures to match by declared type, not by structural shape, when the types involved are slices of named structs — so `*store.TranscriptStore` cannot satisfy `session.TranscriptSink` directly. `transcriptSinkAdapter` in `cmd/server/main.go` (the one file that already imports both packages) performs the one-line-per-field conversion.
- `closeTranscriptLocked` runs synchronously inside `Disconnect`'s held `m.mu`, blocking until the write loop drains and `CloseGameSession` returns, rather than being fired off in a goroutine. This was necessary for the `closes_on_disconnect` subtest to be deterministic (no sleep/poll needed) and is justified because open/close is a deliberate, once-per-connection lifecycle event, not the per-line hot path `T-3-26` is about.
- Two enqueue helpers (`enqueueTranscriptLine` takes `m.mu.RLock()` itself; `enqueueTranscriptLineLocked` assumes the caller already holds it) exist because `sync.RWMutex` is not reentrant and the four autopilot transition functions already hold `m.mu` when they need to drop a marker line.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Fixed a latent bug in `internal/profiles/handler_test.go` that panicked the new tests**
- **Found during:** Task 3, running `go test ./internal/profiles/...` (the full package, not just the new tests)
- **Issue:** `TestAIPlayerLogLinesAreEmitted` did `log.SetOutput(&buf); defer log.SetOutput(nil)`. Setting the global `log` package's output writer to `nil` is invalid — any subsequent `log.Printf` call in the same test binary panics with a nil-pointer dereference inside `log.(*Logger).output`. This was harmless before this plan because that test was the last one in `handler_test.go`, so nothing ran afterward that logged anything. This plan's new `internal/profiles/logs_test.go` runs after it (alphabetically, in the same package/test binary), and `ListSessions`/`GetSessionTranscript` both call `log.Printf` for the `[AI-PLAYER] transcript-read` line, so `go test ./internal/profiles/...` panicked.
- **Fix:** Captured the previous writer (`prevOutput := log.Writer()`) before redirecting, and restored it in the deferred call (`defer log.SetOutput(prevOutput)`) instead of `nil`.
- **Files modified:** internal/profiles/handler_test.go
- **Verification:** `go test ./internal/profiles/...` passes in full (previously panicked); `go test ./...` (whole repo) passes with no failures.
- **Committed in:** 7bf9ab7 (Task 3 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking test-infrastructure bug in a file outside this plan's `files_modified`, fixed because it directly blocked this plan's own task-3 verification step — `go test ./internal/profiles/...` — and the fix is a one-line, unambiguous correction of an invalid `log.SetOutput(nil)` call).
**Impact on plan:** None on shipped behavior; the fix only restores correct test isolation. No scope creep — the alternative (leaving it broken) would have meant this plan's own acceptance criterion ("go test ./internal/profiles/... -run ... exits 0") technically passes in isolation but `go test ./...` and any CI run of the full suite would panic.

## Issues Encountered

- **`-race` still unavailable on this dev machine.** Same `CGO_ENABLED=0`, no-gcc environment already logged by plans 03-01/03-02/03-03. `go test ./internal/session/... -run TestTranscript -race -v` was run as plain `go test ./internal/session/... -run TestTranscript -v` instead; all four tests and three named subtests pass. `TestTranscriptDoesNotBlockReader` specifically exercises the non-blocking guarantee (enqueue while the sink's `AppendGameLines` is permanently blocked) and passed. Logged in `.planning/phases/03-one-ai-decision/deferred-items.md` under a new "## Plan 03-05" section, alongside a note that the previously-flagged `internal/icm` `TestHandlerRegistration/CANCEL` failure no longer reproduces at this plan's base commit (fixed by an earlier wave-1 commit, `dcb671d`).
- **`uuid.Parse` requirement surfaced a test-fixture gap.** `openTranscriptLocked` parses both `userID` and `connectionID` as UUIDs before calling the sink (matching `internal/store`'s UUID-typed methods) and silently no-ops on a parse failure. `manager_test.go`'s existing convention of seeding fake string ids like `"user-1"`/`"conn-1"` (used for `EngageAutopilot` tests, which never parse UUIDs) does not work for transcript tests; `transcript_test.go` uses real `uuid.New().String()` values everywhere a transcript is expected to open, while still using `seedConnectedSession`/`EngageAutopilot` with those same UUID strings (those functions treat ids as opaque map keys, so this is not a special case, just a naming choice).

## User Setup Required

None - no external service configuration required. Gemini env vars remain the Phase 3 blocker noted in STATE.md for later plans (03-02/03-08), unrelated to this plan.

## Next Phase Readiness

- The Go contract plans 03-08 and 03-11 depend on is in place and proven by test: `TranscriptSink` interface, `Manager.SendCommandAs(userID, command, source string) error` (with `SendCommand` delegating `"human"`, confirmed by source assertion that `websocket.go` needs no edit), and the line-source vocabulary `human`/`ai`/`game`/`marker`.
- `evidence/03-canned-report.txt`'s sessions-list and transcript-read steps against staging, and `evidence/12-log-page-two-pane.png`, remain deferred to plan 03-13's evidence capture (this plan's endpoints are ready; staging has not been exercised from this worktree).
- No blockers for plan 03-08 (the driver, which will call `SendCommandAs(userID, command, "ai")`) or plan 03-11 (the log page, which will call the two new endpoints this plan built).
- `go build ./...`, `go vet ./internal/session/... ./internal/store/... ./internal/profiles/... ./cmd/...`, and `go test ./...` (whole repo) all pass at this plan's final commit. `git diff --stat go.mod frontend/package.json` is empty.

---
*Phase: 03-one-ai-decision*
*Completed: 2026-09-15*

## Self-Check: PASSED

- FOUND: migrations/011_add_ai_session_tables.up.sql
- FOUND: migrations/011_add_ai_session_tables.down.sql
- FOUND: internal/store/transcripts.go
- FOUND: internal/session/transcript.go
- FOUND: internal/session/transcript_test.go
- FOUND: internal/profiles/logs.go
- FOUND: internal/profiles/logs_test.go
- FOUND: internal/session/manager.go
- FOUND: internal/profiles/handler.go
- FOUND: internal/profiles/handler_test.go
- FOUND: cmd/server/main.go
- FOUND: .planning/phases/03-one-ai-decision/deferred-items.md
- FOUND: 48dca29 (Task 1 commit)
- FOUND: 93179e5 (Task 2 commit)
- FOUND: 7bf9ab7 (Task 3 commit)
