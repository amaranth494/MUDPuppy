---
phase: 03-one-ai-decision
reviewed: 2026-09-16T14:26:48Z
depth: standard
files_reviewed: 42
files_reviewed_list:
  - cmd/server/main.go
  - frontend/src/App.tsx
  - frontend/src/components/AIAssistPanel.tsx
  - frontend/src/index.css
  - frontend/src/pages/LogsPage.tsx
  - frontend/src/pages/PlayScreen.tsx
  - frontend/src/pages/SettingsPage.tsx
  - frontend/src/services/api.ts
  - frontend/src/services/automation/evaluator.ts
  - frontend/src/services/icm-adapter.ts
  - frontend/src/types/index.ts
  - internal/auth/handler.go
  - internal/auth/handler_test.go
  - internal/config/config.go
  - internal/config/config_test.go
  - internal/driver/driver.go
  - internal/driver/driver_test.go
  - internal/gemini/client.go
  - internal/gemini/client_test.go
  - internal/icm/dispatcher_test.go
  - internal/icm/handler.go
  - internal/icm/icm_test.go
  - internal/profiles/decisions.go
  - internal/profiles/decisions_test.go
  - internal/profiles/handler.go
  - internal/profiles/handler_test.go
  - internal/profiles/logs.go
  - internal/profiles/logs_test.go
  - internal/session/handler.go
  - internal/session/handler_test.go
  - internal/session/manager.go
  - internal/session/transcript.go
  - internal/session/transcript_test.go
  - internal/session/websocket.go
  - internal/session/websocket_test.go
  - internal/session/window.go
  - internal/session/window_test.go
  - internal/store/decisions.go
  - internal/store/transcripts.go
  - migrations/011_add_ai_session_tables.down.sql
  - migrations/011_add_ai_session_tables.up.sql
  - scripts/verify-phase3.sh
findings:
  critical: 2
  warning: 2
  info: 2
  total: 6
status: issues_found
---

# Phase 3: Code Review Report

**Reviewed:** 2026-09-16T14:26:48Z
**Depth:** standard
**Files Reviewed:** 42
**Status:** issues_found

## Summary

This review covered the full Phase 3 slice: the recent-text window and ring buffer, the Gemini client, the ICM wiring, the AI driver's decision loop, the session transcript tap, the decisions/sessions read endpoints, the DR-2-01 OTP-logging fix, the frontend AI Assist panel and Logs page, migrations, and the bash verification harness.

The security properties called out in the review brief hold up well under inspection: `[AI-PLAYER]` log lines are consistently limited to ids, stages, lengths, kinds and HTTP statuses (T-3-07/T-3-02 — verified in `internal/driver/driver.go`, `internal/session/transcript.go`, `internal/session/manager.go`, and `internal/gemini/client.go`); the decisions and transcript endpoints resolve ownership through `GetProfileByConnection` before touching a row, with a second, independent filter at the SQL layer as defence in depth (T-3-03/T-3-04 — verified in `internal/profiles/decisions.go`, `internal/profiles/logs.go`, `internal/store/decisions.go`, `internal/store/transcripts.go`, and their test files); the Gemini API key travels only in a header, never a URL (T-3-02); the DR-2-01 sign-in code is suppressed by default and gated behind `AUTH_LOG_OTP` (`internal/auth/handler.go`); and the frontend renders all model- and transcript-derived text as JSX children with no `dangerouslySetInnerHTML` anywhere (T-3-37).

However, two concurrency defects in the new session-transcript tap (`internal/session/manager.go` / `internal/session/transcript.go`) are serious enough to block: one is a crash (send on a closed channel), the other freezes every connected user's session for the duration of a database round trip. Both are reachable through ordinary connect/disconnect traffic, not just adversarial input, and neither is caught by `go test -race` (the race detector does not flag a logically-unsynchronized closed-channel send, and lock contention is not a data race). See Critical Issues below.

## Critical Issues

### CR-01: Transcript open/close runs synchronous database I/O while holding the global session-manager mutex

**File:** `internal/session/manager.go:534-583` (`openTranscriptLocked`, `closeTranscriptLocked`), called from `Connect` (line 280) and `Disconnect` (line 399)

**Issue:** `Manager.mu` is a single `sync.RWMutex` shared by every connected user (it guards `sessions`, `conns`, `autopilot`, `outputWindow`, and `transcripts` for the whole process). `Connect` and `Disconnect` both hold `m.mu.Lock()` for their entire body, and inside that locked section they call:

- `openTranscriptLocked` → `m.transcriptSink.OpenGameSession(...)`, a synchronous `INSERT ... RETURNING id` (see `internal/store/transcripts.go:64-74`)
- `closeTranscriptLocked` → `t.close(sink)` (`internal/session/transcript.go:161-182`), which closes the line channel, then **blocks on `<-t.done`** until the writer goroutine has drained every batched line (including a synchronous `AppendGameLines` call for the final partial batch) and only then calls the synchronous `CloseGameSession` UPDATE.

Because every other Manager method that touches a live session — `ReadOutput`, `SendCommandAs`, `EngageAutopilot`, `DisengageAutopilot`, `AutopilotStateFor`, `RecentOutputSnapshot`, `GetSession` — also takes `m.mu` (read or write), a single saved-profile connect or disconnect stalls **every other user's** session for as long as the database call(s) take. A slow Postgres round trip, connection-pool exhaustion, or a stuck transaction turns one user's ordinary disconnect into a process-wide freeze: no other user's game output can be read, no command can be sent, and autopilot state cannot even be queried, until the blocking call returns. This is exactly the class of problem T-3-26 ("a database write on the MUD read path must not stall the game connection") was written to prevent, but the mitigation (`transcriptChannelCapacity` + a non-blocking `enqueue`) only covers the per-line hot path — it does not cover session open/close, which run synchronously under the manager-wide lock.

**Fix:** Move `OpenGameSession` and the blocking half of `close()` (channel close, drain-wait, `CloseGameSession`) outside the critical section. For example, release `m.mu` before calling into the transcript sink, and re-acquire only to update the map:

```go
// Connect (sketch)
gameSessionID, err := m.transcriptSink.OpenGameSession(userUUID, connUUID) // no lock held
m.mu.Lock()
m.transcripts[userID] = &transcriptSession{gameSessionID: gameSessionID, ...}
m.mu.Unlock()
```

```go
// Disconnect (sketch)
m.mu.Lock()
t, ok := m.transcripts[userID]
delete(m.transcripts, userID)
m.mu.Unlock()
if ok {
    t.close(m.transcriptSink) // blocking work happens with no lock held
}
```
Both call sites need to keep the "resolve-then-remove-then-close" ordering so a second caller can never observe a half-torn-down transcript, but none of that requires holding the session-wide lock for the duration of the I/O.

---

### CR-02: Send-on-closed-channel panic race between transcript `enqueue` and `close`

**File:** `internal/session/transcript.go:69-79` (`enqueue`), `:161-182` (`close`); reached via `internal/session/manager.go:611-618` (`feedTranscriptOutput`) and `:588-595` (`enqueueTranscriptLine`)

**Issue:** `feedTranscriptOutput` and `enqueueTranscriptLine` both look up the `*transcriptSession` under `m.mu.RLock()`, **release the lock**, and only then call a method on the object that ends in a channel send:

```go
func (m *Manager) feedTranscriptOutput(userID string, p []byte) {
    m.mu.RLock()
    t, ok := m.transcripts[userID]
    m.mu.RUnlock()
    if ok {
        t.feedGameOutput(p) // -> t.enqueue(...) -> t.lines <- ...
    }
}
```

Meanwhile, `closeTranscriptLocked` (called from `Disconnect`, itself reachable from a different goroutine — the hard-cap timer in `startTimers`, a REST `POST /api/v1/session/disconnect`, or a websocket read-error path — while the reading goroutine `readMUDOutput` is still delivering bytes) does:

```go
delete(m.transcripts, userID)
t.close(m.transcriptSink)   // closes t.lines
```

If `readMUDOutput`'s goroutine has already fetched the `*transcriptSession` pointer (before `RUnlock`) but has not yet reached the `select` inside `enqueue` when `close(t.lines)` runs, the send in `enqueue`'s `select` statement targets an already-closed channel. **Sending on a closed channel panics unconditionally** — Go's `select`/`default` construct does not protect against this, because a send to a closed channel is always "ready" and is chosen over `default`. An unrecovered panic in a goroutine terminates the entire process, taking down every connected user's session, not just the one that disconnected.

This is a realistic, not merely theoretical, race: an owner disconnecting while the MUD is still flushing a final burst of text (a `quit` message, a prompt, a `goodbye` banner) is the common case, not an edge case. `go test -race` will not catch this: the channel close and the channel send are each individually valid operations from the Go memory-model's point of view, so there is no unsynchronized memory access for the race detector to flag — the bug is a violation of the channel's logical close-once-no-more-senders protocol, which `-race` does not model.

**Fix:** Do not close `t.lines` from `close()` while other goroutines may still hold a reference and attempt to send. Two standard approaches:
1. Never close the data channel; signal shutdown with a separate `stop chan struct{}` that `writeLoop`'s `select` also watches, and let `enqueue` do a non-blocking `select` against `stop` as well as `t.lines`. `close()` closes `stop`, not `lines`.
2. Guard `enqueue` with the same `sync.RWMutex` that guards `m.transcripts` lookups, held across both the map lookup and the send (not just the lookup), and have `close()` take the map's write lock for the same span before closing the channel — trading the current lock-free hot path for one that cannot race with teardown.

Given the plan's own emphasis on "never block the MUD read/write path" (T-3-26), option 1 is more consistent with the phase's design.

## Warnings

### WR-01: React key collision when multiple system entries lack a stored decision id

**File:** `frontend/src/components/AIAssistPanel.tsx:154-158` (`key={entry.id}`)

**Issue:** `Driver.recordFailure` (`internal/driver/driver.go:292-330`) sets `decisionID := ""` whenever `d.decisions` is nil or `InsertDecision` returns an error, and still calls `d.notify(...)` with `Event{ID: decisionID, ...}` regardless. The websocket push carries that empty `ID` straight through to `AIDecisionPayload.id`. In `AIAssistPanel.tsx`, every live-pushed `system` entry is rendered with `key={entry.id}`; two or more failure/refusal notices arriving with `id === ""` (e.g., after two separate `#AUTO ON` attempts on a server whose decision store is briefly unavailable) produce duplicate React keys. React's reconciliation is undefined for duplicate keys in a list — the wrong DOM node can be reused or the wrong entry's content can appear to "not update" on a subsequent render.

**Fix:** Fall back to a client-generated identifier when the server-supplied id is empty, e.g.:
```tsx
const key = entry.id || `system-${index}-${Date.now()}`;
```
or have the driver assign a lightweight non-persisted id (e.g. a UUID generated in-process) to every `Event` even when the decision row could not be written, so `ID` is never empty on the wire.

### WR-02: Internal error text returned verbatim to the client

**File:** `internal/profiles/handler.go:223-228` (`GetByConnection`)

**Issue:** Unlike every other handler method in this file (`Get`, `Update`, `GetAliases`, etc., which all return a fixed string like `"Failed to get profile"`), `GetByConnection` does:
```go
h.sendError(w, "Failed to get profile: "+err.Error())
```
`err` here comes straight from `store.ProfileStore.GetProfileByConnection`, which can wrap a raw `database/sql` / `lib/pq` error (connection strings, constraint names, query fragments) depending on the failure mode. This is the one call site in the reviewed files that leaks an internal error string to an HTTP client, inconsistent with the rest of the package's discipline and with the "informative but not leaky" failure standard the phase's other new endpoints follow (D-13, D-20).

**Fix:** Log `err` (as the method already does one line above) and return the same fixed string the rest of the file uses:
```go
log.Printf("[SP04PH02T03] Get profile by connection failed: %v", err)
h.sendError(w, "Failed to get profile")
```

## Info

### IN-01: Dead statement in `EngageAutopilot`

**File:** `internal/session/manager.go:652` (`_ = curConnID`)

**Issue:** `curConnID` is assigned at the top of `EngageAutopilot` and is genuinely used four lines later (`m.logAutopilotTransition(userID, curConnID, cur, newState, "already-on")`), so the `_ = curConnID` statement in between the wrong-connection check and that use is a no-op leftover from an earlier refactor. It does not affect behavior but is confusing to a reader trying to determine whether `curConnID` is actually consumed.

**Fix:** Delete the line; the existing "already-on" use already keeps the variable alive.

### IN-02: Non-jq JSON field fallback in the verification harness can match the wrong field

**File:** `scripts/verify-phase3.sh:200-217` (`_get_field`), used for nested lookups like `sessions.0.id` at line 473

**Issue:** When `jq` is unavailable, `_get_field` reduces a dotted path like `sessions.0.id` to just its leaf (`id`) and greps the whole raw JSON body for the first `"id": ...` occurrence. If the response body contains any other `"id"` field before the first session's id (for example, if a future change adds a top-level `"request_id"` — no, that wouldn't match — but any additional object with an `id` key nested earlier in document order would), the harness would silently pick up the wrong value rather than the intended `sessions[0].id`. This is a latent fragility in a diagnostic tool, not a product defect, and is already scoped by comment as a fallback path.

**Fix:** Either require `jq` for this specific check (fail loudly with a clear message when absent) or narrow the non-jq fallback to search only within the `"sessions":[...]` substring before applying the leaf-key grep.

---

_Reviewed: 2026-09-16T14:26:48Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
