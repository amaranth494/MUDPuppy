# Phase 4: Continuous Play - Research

**Researched:** 2026-09-17
**Domain:** Go backend extension of a working single-decision AI driver into a paced, cancellable continuous loop (`internal/driver`, `internal/session`, `internal/gemini`, `internal/store`, `internal/config`, `internal/crypto`) plus the first three layers of a new memory model (Immediate Context, Session Memory, Quest Memory) and a nightly retention job; TypeScript frontend extension (goal box, Session Memory panel, badge-from-message). No new package, no new external dependency, no new go.mod/package.json entry — everything below is stdlib (`context`, `time`, `sync`) plus the existing hand-written Gemini client.
**Confidence:** HIGH for every code-level extension point (every claim below traces to a direct read this session of `driver.go`, `driver_test.go`, `manager.go`, `window.go`, `websocket.go`, `handler.go`, `client.go`, `profile.go`, `decisions.go`, `transcripts.go`, `config.go`, `crypto.go`, migrations 010-012, `AIAssistPanel.tsx`, `AutopilotBadge.tsx`, `AIPlayerPanel.tsx`, `SessionContext.tsx`, and a live `go build`/`go test`/`go test -race` run on this machine). MEDIUM for the Gemini structured-output extension (nested-array support and `propertyOrdering` confirmed via official docs and this project's own already-shipped, tested code, but exact token/latency cost is not measured — no live model call was made this session, per instruction). LOW/ASSUMED for the Alter Aeon socket-drop root cause (the MUD server's own behavior is not observable from this codebase; what is ruled out on our side is HIGH confidence, the actual remote cause is a documented open question) and for the recommended pacing/memory-size numeric constants (all explicitly Claude's Discretion per CONTEXT.md, offered as starting points for the staging walkthrough to tune, not verified against live gameplay timing).

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Session goal and the active Quest**
- **D-01:** The session goal is a one-line text box at the top of the AI Assist panel. It is stored server-side per connection profile, survives a page refresh, a reconnect, and `#AUTO OFF` then `#AUTO ON`, and lasts until the owner changes or clears it. No directive argument; `#AUTO ON` stays bare (Phase 3 D-11 unchanged).
- **D-02:** `#AUTO ON` with an empty goal box engages anyway. The AI plays from the profile's approach guidance as its standing direction and the panel shows "No session goal set". Blank means the profile's default direction, the same pattern as blank AI settings.
- **D-03:** The goal is editable while the AI plays. The next decision's prompt carries the new goal and the panel prints a system line in the family of "Goal changed: ...".
- **D-04:** The goal box names the active Quest. Typing a goal creates a Quest record for it or reactivates an open Quest whose goal text matches. The loop reads the active Quest's bullets into every prompt. Changing the goal switches which Quest is active; the previous one stays open for another day. Closing a Quest is Phase 6's job; nothing in Phase 4 closes one.

**The loop and its pacing**
- **D-05:** The loop is a server-side goroutine per engaged user session, started on every real engage (Phase 3's `HandleEngage` becomes the first iteration), stopped by a disengage of any cause, paused by WAITING, and restarted by a WAITING-to-ON resume.
- **D-06:** The next decision fires after new game output arrives and a short settle delay passes. If the game stays quiet, a floor interval fires a decision anyway. Decisions are never closer than a minimum spacing, and the ICM dispatcher's rate limit is the backstop. All three durations are server constants (Claude's discretion; starting points: settle about 1 to 2 seconds, floor about 20 seconds, minimum spacing a few seconds, tuned against Alter Aeon during the walkthrough).
- **D-07:** Per-decision display is unchanged from Phase 3.
- **D-08:** Reassessment on re-engage: every engage starts with a fresh Immediate Context snapshot, and the first decision's prompt carries an explicit instruction to re-check the current situation against the goal before acting and to say so in its reasoning. No plan object exists to carry over.

**Memory layers built in this phase**
- **D-09:** Immediate Context is the existing rolling window made time-bounded: roughly the last 10 seconds of game activity, with the current 8 KB byte size as the ceiling. It never goes empty: the most recent screenful is always retained. Exact seconds and screenful size are Claude's discretion.
- **D-10:** Session Memory is a curated bullet list of facts for the current game connection. Starts empty at connect, updated by the model as part of each decision (add, update, merge, remove; mechanism is Claude's discretion), has a size ceiling, lives for the game connection across stints/wheel-grabs/WAITING resumes, persisted, shown read-only in the AI Assist panel as a collapsible section that updates live.
- **D-11:** Quest Memory is a per-profile, per-goal store of terse bullets germane to the goal. Created by D-04, read into every prompt while its Quest is active, updated during play, persists across sessions, stored separately, not shown on the panel.
- **D-12:** Detail is inversely proportional to age (memory model Addendum 2).
- **D-13:** The prompt now carries, in order: profile's standing text, session goal, active Quest's bullets, Session Memory, Immediate Context window. Memory text is model-written from game text, so it is delimited as untrusted data in both the player prompt and the reviewer prompt, exactly as the window is.

**Halts, thresholds, and blocks**
- **D-14:** The call cap counts every model call in a stint: decision call, reviewer call, any retry. Count starts at zero on every `#AUTO ON`, including after a wheel-grab. When the next call would exceed the cap, the loop stops before making it and autopilot lands OFF with a locked notice. Blank cap means no cap.
- **D-15:** The error threshold counts consecutive transient failures (api-error, rate-limited, malformed, no-command, multi-command, non-game-line, icm-refused) and is reset by a sent command. A transient failure below the threshold sends nothing, shows its notice with the count, loop continues. Non-transient failures (missing configuration, auth error, missing profile) still disengage at once. Blank threshold resolves to 3.
- **D-16:** One automatic retry on a vendor 503: same call retried once after a short delay, with a visible notice while the retry is in flight. A second 503 is a transient failure counted under D-15. Applies to both the decision call and the reviewer call; the retry counts against the cap.
- **D-17:** A blocked command skips the turn: nothing sent, block shown/stored, loop waits for next tick. Blocks are not failures, don't feed D-15's counter. Consecutive blocks counted separately against the same threshold number; reaching it disengages with its own notice. A sent command resets the block count.
- **D-18:** Every `ai` websocket message carrying a disengage also carries the new switch state; badge updates from that message.
- **D-19:** A disconnect while engaged pauses the loop in WAITING; nothing issued while disconnected; AI never reconnects; resume restarts the loop per D-05/D-08. Alter Aeon dropped the socket ~30s after every reconnect in Phase 3.1 walkthroughs; researcher investigates before the loop is built.
- **D-20:** The Phase 4 test suite (`go test ./internal/...`, fakes, no network) covers: cap halt (set/blank), threshold disengage + notice, blank threshold resolving to 3, 503 retry, consecutive-block disengage, no commands while disconnected + resume, no AI-initiated reconnect, re-engage fresh window + reassess instruction, no crash on any failure.

**Retention standard and audit stores**
- **D-21:** Decision snapshots (window text) pruned after 7 days; session transcripts pruned after 30 days. Decision rows keep timestamp/reasoning/command/outcome/block reason forever. Per-profile "delete captured text now" action for both stores. Nightly server-side job, one `[AI-PLAYER]` log line per run with counts. Memory layers are not captured text and are not pruned by this job.
- **D-22:** Policy stays at version 1.0.
- **D-23:** Staging stays on `gemini-3.5-flash-lite` for this phase.

**Security carry-forward remediations**
- **D-24:** DR-3.1-01: reviewer prompt wraps the first model's stated reasoning in its own untrusted markers; add a reviewer-channel corpus item; rerun AFTER report on a fresh-quota day before staging deploy.
- **D-25:** DR-3.1-04: server refuses to start when `ENCRYPTION_KEY_V1` is missing outside local development; document as required; write a key-rotation procedure.
- **D-26:** DR-3.1-05: migration 012's down file maps blocked rows to failed (or stops with a clear message) before re-adding the constraint.
- **D-27:** DR-3-07: delete the dead statement in `EngageAutopilot` (`internal/session/manager.go`).
- **D-28:** Closures recorded at the Phase 4 review per the mapping in 04-CONTEXT.md.

### Claude's Discretion
- Session Memory/Quest Memory curation mechanism (extra fields in the decision answer JSON; researcher confirms schema fits Gemini's constrained output).
- Immediate Context seconds, retained screenful size, where timestamps live in the ring buffer.
- Session/Quest Memory size ceilings, storage shape, migration numbers (013+), goal-text matching (trim + case-fold).
- Loop constants (settle, floor, minimum spacing), goroutine lifecycle/cancellation/in-flight guard, building on `Driver.inFlight`.
- Exact wording of new notices, in `03-UI-SPEC.md`'s Copywriting Contract voice; locked strings, no vendor text.
- Panel layout for goal box and collapsible Session Memory section; where "delete captured text now" sits.
- `ai` websocket message fields for goal/switch-state/counts/memory; REST routes for goal get/set and memory read on attach.
- Nightly job implementation (in-process ticker is fine) and schedule.
- `[AI-PLAYER]` log lines for loop ticks, cap/threshold counts, retries, blocks, memory updates, retention runs, following existing `stage=` format.
- Which tests move into a shared scaffold and how the Phase 4 suite is organised.
- Whether icm-refused counts as transient (default: yes) and the retry delay for the 503.

### Deferred Ideas (OUT OF SCOPE)
- Quest Memory recall through chat (Phase 5). Editing Session Memory by hand (Phase 5). End-of-session Quest evaluation, quest closure, Historical Memory (Phase 6). Policy v1.1 rewording (declined, stays 1.0). Per-profile retention settings (server default only, for now). Fake game fixture / diagnostic endpoint on staging (declined). Backlog defects (WR-03 Never-issue hint wording, IN-02 corpus pacing, stale worktree directory). AR-3-08 `GOOGLE_API_KEY` on production (leave it, note for cut-over checklist).
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| REQ-continuous-loop | Owner sets a session goal; AI plays toward it until stopped, within safety limits; loop paces to game rhythm | Loop architecture (goroutine, `OutputSignal`, settle/floor/minSpacing) in Architecture Patterns 1-2; goal storage in Standard Stack/migration 013 |
| REQ-call-cap-and-error-disengage | Call cap halts with notice; repeated errors/malformed output disengage with notice | Pattern 3 (counters woven into `HandleEngage`); Common Pitfall 1 (transient vs non-transient failure-kind gap) |
| REQ-reengage-reassess | Re-engagement demonstrably reassesses, not a stale plan | Pattern 1 (`EngageLoop` first-iteration reassess instruction); D-08 code touch points in `buildSystemInstruction` |
| REQ-doc-continuous-visible-play | Every decision visible with reasoning as it happens, continuously | Existing Phase 3 display path unchanged (D-07); loop reuses `HandleEngage`'s notify/store calls verbatim each iteration |
| REQ-doc-wheel-grab-and-reengage | Typing a command instantly disengages; re-engaging reassesses | Pattern 2 (`DisengageHook` cancels loop mid-sleep, not just after an iteration completes) |
| REQ-safety-limits-hold | All four mechanical limits hold under automated test, including blank settings | Validation Architecture section; Pitfall 1; Pattern 4 (test-fake gaps to close) |

</phase_requirements>

## Summary

Phase 4 turns Phase 3's one-shot `Driver.HandleEngage` into a paced, cancellable loop without replacing it: `HandleEngage` becomes the loop's per-iteration body, unchanged in its display/storage contract, extended with per-user call/failure/block counters, a 503 retry, and a first-iteration reassessment flag. The loop itself is new: a goroutine per engaged user, started wherever `EngageHook`/`engageHook` fires today (`internal/session/manager.go:801-802` and `internal/session/handler.go:408-410`), paced by a new output-arrived signal from `internal/session.Manager` plus a floor timer, and stopped the instant autopilot leaves ON — which requires a new, symmetric `DisengageHook` fired from `DisengageAutopilot` and `parkAutopilotLocked` so a wheel-grab or a disconnect cancels the loop mid-sleep rather than only being noticed after the next decision completes.

The three new memory layers slot into the existing two-call structure with no new model call: Session Memory and Quest Memory become two more fields on the same JSON answer schema Gemini already returns (`internal/gemini/client.go`'s `responseSchema`/`propertyOrdering` mechanism, already proven in this codebase for the reviewer's `reason`-before-`blocked` ordering), confirmed by official Gemini docs to support nested arrays. The simplest schema shape — and the one recommended here over the CONTEXT.md-suggested add/update/remove op-list — is a flat array of strings that the model rewrites in full each decision it wants to change memory; no per-entry IDs, no Go-side reconciliation, and it is exactly as capable of "add, update, merge, remove" as an op-list, since removing an item just means it is absent from the next array. Storage reuses existing conventions: one new `TEXT` column on `profiles` for the goal, one new column on the existing `game_sessions` table for Session Memory (JSONB array), and one small new `quests` table for Quest Memory with a partial unique index enforcing D-04's "reactivate the open Quest with matching goal text" rule at the database level.

Two carry-forward security items have precise, narrow fixes already discoverable in the code: the migration 012 down-file gap (D-26) needs one `UPDATE ... SET outcome='failed'` statement before its existing (already-correct) constraint-lookup-and-recreate logic, and the `ENCRYPTION_KEY_V1` startup requirement (D-25) has a ready-made "are we outside local development" signal already used elsewhere in this exact codebase (`RAILWAY_ENVIRONMENT`), so no new environment-detection mechanism needs inventing. The Alter Aeon 30-second socket drop (D-19) cannot be root-caused from this codebase alone — Alter Aeon is an opaque remote server — but everything on our side that could explain a *self-inflicted* drop is ruled out with HIGH confidence (idle timeout is disabled by explicit prior instruction, no TCP keepalive is set either way, the 30-second ping ticker found in the code is for the *browser-facing* websocket, not the outbound MUD socket, and reconnection is manual, never automatic). The loop's WAITING-pause/resume design already tolerates this regardless of cause, per D-19's own instruction.

The existing test fakes (`internal/driver/driver_test.go`) are a strong foundation but have three concrete gaps for Phase 4's multi-iteration scenarios: `fakeModels` supports only one canned answer/error rather than a scripted sequence, the `Sessions` interface has no `AutopilotStateFor` method for the loop to self-check against, and `fakeSessions.DisengageAutopilot` always returns `(Off, true)` rather than modeling a real state machine. All three are small, additive fixes reusing the real `session.AutopilotState`/`Engage`/`Disengage`/`Resume` pure functions that already exist and are already tested.

**Primary recommendation:** Wrap `HandleEngage` in a goroutine-per-user loop paced by a new `Manager.OutputSignal` channel plus floor/settle/minSpacing durations as `Driver` struct fields (not global constants) so tests can inject millisecond-scale values with no fake-clock library; extend the existing decision-answer JSON schema with two flat string-array fields (`session_memory`, `quest_memory`) rather than an operation-based schema; store the goal as one `TEXT` column, Session Memory as one JSONB column on `game_sessions`, and Quest Memory in one new `quests` table with a partial unique index on `(connection_id, goal_text_normalized) WHERE status='active'`.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Loop pacing (settle/floor/minSpacing, goroutine lifecycle) | API/Backend (Go) | — | Server-side driver owns timing; browser only displays results (CON: server-side is not unattended, but pacing logic is inherently backend state) |
| Output-arrived signal | API/Backend (Go, `session.Manager`) | — | The relay path that already feeds the ring buffer is the only place that knows "new bytes arrived" |
| Session goal storage & edit | API/Backend (Go, `profiles` handler + `store`) | Browser (goal box UI) | Server owns truth (established pattern); browser reflects and edits via REST |
| Quest record lifecycle (create/reactivate, Phase 4 never closes) | API/Backend + Database | — | Matching/reactivation logic needs a DB uniqueness guarantee (partial unique index), not just application code |
| Session Memory curation | API/Backend (model call) + Database | Browser (read-only display) | Model proposes, Go persists and pushes; browser is read-only per D-10 |
| Quest Memory curation | API/Backend (model call) + Database | — | Not shown on panel in Phase 4 (D-11); pure backend/DB concern |
| Call cap / error / block counters | API/Backend (Go, in-memory per Driver instance) | — | Per-stint counters reset on engage; no need for DB persistence within a stint |
| 503 retry | API/Backend (Go, `driver` package) | — | Vendor-specific transient-error handling belongs beside the call site, not in the generic `gemini` client (kept vendor-status-agnostic there) |
| Retention job | API/Backend (Go ticker) + Database | — | No new infrastructure; in-process ticker per CLAUDE.md's "no speed bumps" instruction |
| Badge / switch-state display | Browser (React) | API/Backend (pushes state) | Server owns truth; browser derives nothing (established `AutopilotBadge.tsx` pattern) |
| ENCRYPTION_KEY_V1 startup gate | API/Backend (Go, `config`/`crypto`/`main`) | — | Startup-time fail-fast, no browser involvement |

## Standard Stack

No new libraries. Everything below is Go standard library (`context`, `time`, `sync`) plus the project's own existing `internal/gemini` client and `database/sql` + `golang-migrate` conventions. This mirrors every prior phase's "no new go.mod entry" pattern (Phase 1 AR-1-01, Phase 2 AR-2-08, Phase 3 AR-3-13).

### Package Legitimacy Audit

**Not applicable — no external packages are installed in this phase.** No `go.mod` or `frontend/package.json` change is anticipated. The plan-checker should verify this holds (empty dependency-drift section in the test report, matching every prior phase).

## Architecture Patterns

### System Architecture Diagram

```
                         ┌─────────────────────────────────────────────┐
                         │  Owner's browser (AI Assist panel)            │
                         │  - types/edits session goal                  │
                         │  - reads Session Memory (read-only, live)     │
                         │  - sees badge from `ai` message (D-18)        │
                         └───────────────┬───────────────────────────────┘
                                         │ REST: goal GET/PUT, memory GET
                                         │ WS: `ai` message (decision/system + goal/state/memory fields)
                         ┌───────────────▼───────────────────────────────┐
                         │  internal/profiles (Handler)                  │
                         │  internal/session (Handler, Manager)          │
                         └───────────────┬───────────────────────────────┘
                                         │ EngageHook / DisengageHook
                         ┌───────────────▼───────────────────────────────┐
                         │  internal/driver.Driver                       │
                         │  ┌──────────────────────────────────────────┐│
                         │  │ EngageLoop(userID, connID)                ││
                         │  │  - reset counters (D-14/D-15/D-17)        ││
                         │  │  - HandleEngage(first=true)  ─────────────┼┼──► reassess prompt
                         │  │  - runLoop goroutine:                     ││
                         │  │      select {                             ││
                         │  │        <-ctx.Done()        -> return      ││
                         │  │        <-OutputSignal(u)   -> settle wait ││
                         │  │        <-time.After(floor) -> tick        ││
                         │  │      }                                    ││
                         │  │      enforce minSpacing                   ││
                         │  │      HandleEngage(first=false)            ││
                         │  └──────────────────────────────────────────┘│
                         │  HandleEngage (per iteration, Phase 3 body +  │
                         │  cap/threshold/block counters + 503 retry +   │
                         │  memory field parse/persist)                 │
                         └───────┬─────────────────────┬────────────────┘
                                 │ GenerateContent /     │ Dispatch (ContextAutomation)
                                 │ ReviewCommand          │
                         ┌───────▼────────┐      ┌───────▼─────────┐
                         │ internal/gemini │      │ internal/icm    │
                         │ (schema now has │      │ Dispatcher      │
                         │ session_memory, │      │ (unchanged)     │
                         │ quest_memory)   │      └────────┬────────┘
                         └────────────────┘               │ SendCommandAs
                                                    ┌──────▼──────────┐
                                                    │ MUD (Alter Aeon)│
                                                    │ via net.Conn    │
                                                    └──────┬──────────┘
                                                           │ ReadOutput
                                                    ┌──────▼──────────┐
                                                    │ session.Manager │
                                                    │ ringBuffer      │
                                                    │ (time-bounded,  │
                                                    │  D-09) +        │
                                                    │ OutputSignal    │
                                                    └─────────────────┘
```

### Recommended Project Structure

No new top-level packages. New files inside existing packages:

```
internal/driver/
├── driver.go          # HandleEngage gains counters, retry, memory parse; new EngageLoop/runLoop
├── loop.go            # NEW: loop goroutine lifecycle, pacing select, counters (kept out of driver.go for readability)
├── memory.go          # NEW: prompt-context struct, memory schema fields, size-ceiling truncation
├── driver_test.go     # extended fakes: scripted fakeModels, real-state fakeSessions, OutputSignal fake
└── loop_test.go        # NEW: Phase 4 safety-limit suite (D-20)

internal/session/
├── manager.go         # OutputSignal map+method, DisengageHook, dead-statement removal (D-27)
├── window.go          # ringBuffer -> timestamped chunks, snapshotWithin(maxAge)
└── autopilot.go        # unchanged (Engage/Disengage/EnterWaiting/Resume pure functions reused as-is)

internal/store/
├── profile.go         # +SessionGoal field/column plumbing
├── quests.go          # NEW: QuestStore (create/reactivate/read/update bullets)
├── decisions.go        # +PruneWindowsOlderThan
└── transcripts.go       # +PruneOlderThan, +UpdateSessionMemory

migrations/
└── 013_add_goal_memory_and_quests.up.sql / .down.sql

cmd/server/main.go     # wire DisengageHook, wire retentionLoop ticker, wire EngageLoop instead of HandleEngage
```

### Pattern 1: The loop wraps `HandleEngage`, not the reverse — `EngageLoop` is the new engage-hook target

**What:** Today, both real-engage trigger points call `driver.HandleEngage` directly with `go`:
- `internal/session/manager.go:801-802` (`resumeAutopilotLocked`, a WAITING-to-ON resume)
- `internal/session/handler.go:408-410` (`Autopilot` HTTP handler, `#AUTO ON`'s "engaged" outcome)

Phase 4 changes both call sites to invoke a new `Driver.EngageLoop(userID, connectionID string)` instead. `EngageLoop` resets the three per-stint counters (call, consecutive-failure, consecutive-block — D-14/D-15/D-17 all say "starts at zero on every `#AUTO ON`, including after a wheel-grab"), calls `HandleEngage(userID, connectionID, isFirstIteration=true)` synchronously for the first decision (D-05: "Phase 3's `HandleEngage` becomes the first iteration"; D-08's reassess instruction only applies here), then — only if autopilot is still ON after that first call — starts `runLoop` as its own goroutine for iteration 2 onward.

**When to use:** This is the single required refactor of the engage-trigger wiring. `EngageHook`'s type signature (`func(userID, connectionID string)`) does not need to change — `EngageLoop` satisfies it exactly, so `SetEngageHook(aiDriver.EngageLoop)` in `cmd/server/main.go:274-275` is a one-line change from `aiDriver.HandleEngage`.

**Example (illustrative, not exact code):**
```go
// Source: this repo, internal/driver/driver.go:186 (existing HandleEngage signature)
// and internal/session/manager.go:98-102 (EngageHook type, unchanged)
func (d *Driver) EngageLoop(userID, connectionID string) {
    d.resetStintCounters(userID)
    d.HandleEngage(userID, connectionID, true) // first=true: D-08 reassess instruction
    if d.sessions.AutopilotStateFor(userID) != session.AutopilotOn {
        return // a first-decision failure/cap already disengaged; no loop to start
    }
    d.startLoop(userID, connectionID)
}
```

### Pattern 2: A symmetric `DisengageHook` is required so the loop can be cancelled *while sleeping*, not just noticed after an iteration

**What:** `EngageHook` (`internal/session/manager.go:98-102`) has no opposite number today — nothing tells the driver "stop" when autopilot leaves ON. Without one, a wheel-grab (`DisengageAutopilot`, `manager.go:710-736`) or a disconnect (`parkAutopilotLocked`, `manager.go:742-759`) would only be noticed by the loop the next time it wakes up from its `select` — which, with a ~20s floor interval and up to ~120s of in-flight model latency (see Common Pitfall 2), could leave a "ghost" loop goroutine alive for a long time after the owner believes they have taken the wheel back. REQ-doc-wheel-grab-and-reengage's "instantly disengages" would then be true for the *command path* (already true today, unchanged) but not for the *loop's own internal state* — a subtle miss that would not show up in Phase 3-style testing but would show up as a stray decision landing after a wheel-grab in a slow-model scenario.

**When to use:** Add `type DisengageHook func(userID string)` and `SetDisengageHook` to `session.Manager`, mirroring `EngageHook`/`SetEngageHook` exactly (`manager.go:98-110`). Fire it (with `go`, matching the existing `go m.engageHook(...)` discipline at `manager.go:802`) from the end of `DisengageAutopilot` when `changed` is true (`manager.go:721-736`) and from the end of `parkAutopilotLocked` when `changed` is true (`manager.go:742-759`). Wire it in `cmd/server/main.go` to `aiDriver.StopLoop`, which cancels the per-user loop's `context.CancelFunc` and deletes its map entry. This is safe to call even when no loop is running (a no-op map miss) and safe to call from inside the very goroutine it targets (a `context.CancelFunc` may be called from any goroutine, including the one whose context it cancels — it only marks `Done()`, it does not block).

**Anti-pattern to avoid:** Do not rely solely on `HandleEngage`'s own post-call check of `AutopilotStateFor` to end the loop. That check is still valuable as defense in depth (it is what ends the loop after a self-inflicted disengage from inside `recordFailure`, with no separate wiring needed since `DisengageAutopilot` already fires the new hook), but it cannot substitute for immediate cancellation of a loop that is *asleep in its pacing `select`*, which is exactly the wheel-grab scenario.

### Pattern 3: Cap/threshold/block counters must be woven into `HandleEngage` at its existing model-call sites, not bolted on around it

**What:** D-14 requires the cap check to happen *before* a call is made ("the loop stops before making it"), and it counts the decision call, the reviewer call, and any retry — three or four separate call sites inside the existing function body. The exact touch points, by line number in the current `internal/driver/driver.go`:

1. Before line 246 (`d.models.GenerateContent(...)`, the player call) — check-and-increment the cap; if it would be exceeded, notify+disengage with the cap notice and return, making *no* call.
2. Inside the 503-retry branch that wraps the above (new code, D-16) — the retry is itself a call and needs its own cap check before it fires.
3. Before line 287 (`d.models.ReviewCommand(...)`, the reviewer call) — same check-and-increment.
4. Inside that call's own 503-retry branch — same check.

**When to use:** A small helper `d.tryReserveCall(userID string, resolved store.ResolvedAISettings) bool` that locks `d.mu` (the same mutex already guarding `inFlight`, `driver.go:147-149`), compares `d.callCounts[userID]` against `resolved.CallCap`/`resolved.CallCapSet`, increments and returns `true` on success or returns `false` without incrementing. A `false` result at any of the four sites above must skip straight to the cap-reached notify+disengage path and make no further calls that iteration.

**Example — recommended notify/disengage shape (reuses existing `recordFailure` machinery, no new decision outcome needed):**
```go
// Illustrative. Reuses store's existing outcome CHECK values ('sent','refused','failed','blocked') —
// no new migration is needed for the cap-reached case; it is recorded as outcome "failed"
// with a distinct failure_kind so the D-20 test suite can assert on it precisely.
const failureCapReached = "call-cap"
var failureNotices = map[string]string{
    // ...existing six entries (driver.go:47-55)...
    failureCapReached: "Session call cap reached. Autopilot disengaged.", // exact wording is Claude's Discretion, D-14 family
}
```

### Pattern 4: The block-repeated-disengage path (D-17) is a *second* notification on top of the existing `recordBlocked` call, not a replacement for it

**What:** `recordBlocked` (`driver.go:391-426`) today deliberately never disengages ("D-06: deliberately no autopilot-disengage call here"). D-17 does not change that for a single block — it adds a *separate* consecutive-block counter that, on reaching the threshold, disengages with its *own* notice ("AI decisions were blocked repeatedly...") distinct from D-15's failure-threshold notice. The two call sites that call `recordBlocked` today (`driver.go:275-278` for Never-issue, `driver.go:304-310` for the reviewer) both need a follow-up check after the `recordBlocked` call: increment the block counter, and if it has reached the resolved threshold, fire a second notify (system kind, the D-17 wording) and call `sessions.DisengageAutopilot(userID, "ai-blocked-repeatedly")`. A sent command (`recordSuccess`, `driver.go:430-458`) resets both this counter and D-15's failure counter to zero.

**Don't hand-roll:** Reuse the exact same counter-map-plus-mutex pattern as the call cap (Pattern 3) — `d.blockCounts map[string]int`, `d.failureCounts map[string]int`, guarded by the existing `d.mu`. Three maps of the same shape, not three different mechanisms.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| "Has new output arrived for this user" signal | A polling loop that re-snapshots the window on a tight timer to detect changes | A non-blocking, buffered-size-1 `chan struct{}` per user (`Manager.OutputSignal`), signaled from the existing `appendOutputWindow` call site (`manager.go:489-502`) | Zero extra CPU when idle; reuses the exact point where new bytes are already known to have arrived; standard Go "wakeup channel" idiom |
| Deterministic timing tests for the loop | A fake-clock library (none exists in `go.mod`; `stretchr/testify` is present transitively but unused) | Make `settleDelay`/`floorInterval`/`minSpacing` `Driver` struct fields (not package constants), set to millisecond values in tests | Matches this project's "no new dependency" pattern (every prior phase's AR-*-SC row); the existing `waitForCalls` polling helper (`driver_test.go:305-315`) already proves this style works for timing-sensitive assertions |
| Quest goal-text matching | A fuzzy-match or NLP similarity function | `strings.TrimSpace` + `strings.ToLower` equality, stored as a second `goal_text_normalized` column with a DB-level partial unique index | D-04 explicitly says "trim and case-fold is fine"; a partial unique index (`WHERE status='active'`) makes reactivation-vs-create atomic and race-free without a `SELECT-then-INSERT` in application code |
| Session/Quest Memory diffing (add/update/merge/remove as distinct operations) | An operation-verb schema (`{op: "add"/"update"/"remove", id, text}`) requiring the model to track stable entry IDs across turns and Go to reconcile them | A flat `string[]` the model rewrites in full whenever it wants to change anything | Strictly simpler on both sides; "remove" is just "absent from next array", "merge" is just "write one bullet instead of two"; avoids an entire class of ID-tracking bugs the op-based approach invites for no behavioral gain, at the cost of slightly more output tokens per turn a memory update actually happens |
| Nightly retention scheduling | A cron library or external scheduler | `time.NewTicker` in a goroutine started once in `cmd/server/main.go`, mirroring `internal/session/manager.go:302-344`'s existing `startTimers` shape (the only precedent for a background ticker anywhere in this codebase) | No new infrastructure, per CLAUDE.md's "no speed bumps" instruction; this codebase already has exactly one ticker-in-goroutine pattern to copy |

**Key insight:** Every "don't hand-roll" item above is really the same instinct applied five times: Phase 4 adds *state machines and timing*, and the temptation is to reach for a general-purpose library or a clever abstraction. This codebase's own established style (hand-written HTTP client instead of an SDK, hand-written table tests instead of testify, hand-written ticker instead of cron) says: reuse the plainest primitive that is already proven to work here.

## Common Pitfalls

### Pitfall 1: D-15's "transient vs non-transient" split cannot be implemented with today's failure-kind taxonomy without adding new kinds

**What goes wrong:** D-15 lists `api-error` as one of the *transient* kinds that count toward the disengage threshold, but says "missing configuration, auth error, missing profile" must still disengage at once (non-transient). Today's code (`driver.go:219-226` for missing profile, `driver.go:247-263` for the Gemini call error) maps *all* of these — a missing profile, a Gemini `KindAuth` (401/403), a Gemini `KindBadRequest`, and a genuine transport failure — into the **same** `failureAPIError` kind. There is currently no way to tell "the key is wrong" (should disengage immediately, every time) apart from "the network hiccuped" (should count toward a 3-strikes threshold) once both have collapsed into `failureAPIError`.

**Why it happens:** Phase 3 only needed "any failure disengages" (D-13's single-strike rule); Phase 4 is the first phase where the distinction matters, and `gemini.ErrorKind` (`internal/gemini/client.go:75-80`) already *does* expose `KindAuth`/`KindBadRequest` separately from `KindTransport` — the information exists, it is just discarded at `driver.go:256-262`'s `switch` (which only branches on `KindRateLimited`/`KindMalformed`, defaulting everything else including `KindAuth` and `KindBadRequest` to `failureAPIError`).

**How to avoid:** Add distinct failure kinds — e.g. `failureAuth` (from `gemini.KindAuth`), `failureBadRequest` or fold it into `failureAuth`'s "non-transient" bucket (Claude's Discretion on the exact split), and treat the existing missing-profile/missing-model-entry defensive returns (`driver.go:219-226`, `driver.go:236-241`) as non-transient too. Then `recordFailure`'s unconditional `d.sessions.DisengageAutopilot(userID, "ai-failure")` call (`driver.go:373`) must become conditional: disengage immediately for non-transient kinds; for transient kinds, increment `d.failureCounts[userID]` and only disengage when it reaches the resolved threshold, otherwise show the "N of M" notice and let the loop continue.

**Warning signs:** A D-20 test asserting "three consecutive transient failures disengage, but a bare auth failure disengages on the first" will fail immediately against unmodified Phase 3 code, because both currently produce byte-identical `failureAPIError` records.

### Pitfall 2: Real Gemini round-trip latency (up to 120s, confirmed on staging) dwarfs the proposed floor interval

**What goes wrong:** `cmd/server/main.go:270-272` sets the Gemini client's HTTP timeout to 120 seconds, with a comment explaining it was raised from the 30s default specifically because "current Gemini flash models take well over the 30s default to return a structured answer (seen on staging 2026-09-16: transport timeout at exactly 30s)." Phase 4 now makes **two** calls per decision (player + reviewer, each individually up to 120s) plus up to one retry of each on a 503 (D-16) — a worst-case single iteration could legitimately take several minutes. D-06's suggested floor interval starting point ("about 20 seconds") is not wrong, but it only governs the *gap after a decision lands* before nudging the AI during a quiet game — it cannot and does not need to bound how long a single in-flight decision takes, because `Driver.inFlight` (`driver.go:147-149, 186-199`) already guarantees only one decision runs at a time per user.

**Why it happens:** It is easy to read "floor interval ~20s" as an upper bound on decision cadence and be surprised when real staging behavior is dominated by model latency, not by the loop's own pacing constants.

**How to avoid:** Document this expectation explicitly in the plan and the staging walkthrough script — actual decision cadence on `gemini-3.5-flash-lite` may be several seconds to (rarely) over a minute per decision, and the floor/settle/minSpacing constants matter most for a fast model or a quiet game, not as a hard ceiling on responsiveness.

**Warning signs:** A staging walkthrough that expects "a decision every ~20 seconds like clockwork" and instead sees bursts and gaps.

### Pitfall 3: `buildSystemInstruction`/`buildReviewSystemInstruction` both need new inputs, and the shared untrusted-data paragraph must grow to cover memory too

**What goes wrong:** Today both functions take only `profile *store.Profile` (`driver.go:496`, `driver.go:574`). D-13 requires the session goal, active Quest bullets, and Session Memory to appear in **both** prompts (the reviewer needs them too, so it can be told they are untrusted — "Memory text is model-written from game text, so it is delimited as untrusted data in both the player prompt and the reviewer prompt, exactly as the window is"). Neither function signature has anywhere to receive this new data, and the shared `untrustedDataParagraph()` (`driver.go:523-530`, explicitly documented as "the one place the wording lives; it must never be reworded independently in either caller") only names the `<GAME_TEXT>` marker today.

**How to avoid:** Introduce a small `promptContext` struct bundling `*store.Profile`, `Goal string`, `QuestBullets []string`, `SessionMemory []string`, and thread it through both builders in place of the bare `*store.Profile` parameter. Extend `untrustedDataParagraph()` with one more sentence naming new markers (e.g. `<SESSION_MEMORY>`/`<QUEST_MEMORY>`, parallel to `<GAME_TEXT>`) so the single-source-of-truth invariant holds for the new content too.

**Warning signs:** A plan that adds memory fields to the JSON *answer* schema (the easy, already-verified part) but forgets that memory must also flow *into* the next prompt as an *input* — these are two separate, both-required changes.

## Code Examples

### Existing model-call sites where cap checks and the 503 retry must be inserted (exact anchors)
```go
// Source: this repo, internal/driver/driver.go:246 and :287 (read this session)
// Site 1 — player call:
answer, genErr := d.models.GenerateContent(context.Background(), entry.Endpoint, entry.ModelName, entry.APIKey, systemInstruction, wrapped)
// Site 2 — reviewer call:
review, revErr := d.models.ReviewCommand(context.Background(), entry.Endpoint, entry.ModelName, entry.APIKey, reviewSystemInstruction, reviewUserText)
```
Both sites need, in order: (1) `tryReserveCall` before the call; (2) on a `*gemini.Error` with `Status == http.StatusServiceUnavailable`, a single retry after a short delay, itself gated by another `tryReserveCall`; (3) on any other error (or a second 503), fall through to the existing `recordFailure` path unchanged.

### Existing Status field already carries the 503 regardless of Kind
```go
// Source: this repo, internal/gemini/client.go:304-326 (errorFromResponse)
// 503 is not 400/401/403/429, so it falls into the `default: kind = KindTransport` branch —
// but Status is always set to resp.StatusCode regardless of which Kind bucket it lands in.
return &Error{Kind: kind, Status: resp.StatusCode, Message: message}
// A 503-specific retry check is therefore simply:
// if gerr, ok := err.(*gemini.Error); ok && gerr.Status == http.StatusServiceUnavailable { ... }
```

### Existing propertyOrdering mechanism, already shipped and tested — the pattern for adding memory fields
```go
// Source: this repo, internal/gemini/client.go:108-124 and :275-288 (ReviewCommand's schema)
type responseSchema struct {
    Type             string                    `json:"type"`
    Properties       map[string]schemaProperty `json:"properties"`
    Required         []string                  `json:"required"`
    PropertyOrdering []string                  `json:"propertyOrdering,omitempty"`
}
// ReviewCommand already proves nested-free arrays are not required to use PropertyOrdering
// correctly; GenerateContent's schema needs a new schemaProperty variant for `type: "array"`
// with an `items` sub-schema — see Gemini's own structured-output docs (Sources) for the
// {"type":"array","items":{"type":"string"}} shape.
```

### Existing ring buffer that needs to become time-aware (D-09)
```go
// Source: this repo, internal/session/window.go:33-80 (full file read this session)
// Today: pure byte ring, no timestamps at all. append(p []byte) and snapshot() string
// are the only two operations; RecentOutputWindowBytes = 8192 is the only ceiling.
// Phase 4 needs a second axis (age) alongside the existing byte ceiling, with an explicit
// "never below one screenful" floor — see Architecture Patterns / Standard Stack below.
```

### Existing engage/disengage hook pattern to mirror for the new DisengageHook
```go
// Source: this repo, internal/session/manager.go:98-110
type EngageHook func(userID, connectionID string)
func (m *Manager) SetEngageHook(h EngageHook) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.engageHook = h
}
// New, symmetric:
type DisengageHook func(userID string)
func (m *Manager) SetDisengageHook(h DisengageHook) { /* identical shape */ }
```

## State of the Art

| Old Approach (Phase 3) | New Approach (Phase 4) | When Changed | Impact |
|--------------------------|------------------------|---------------|--------|
| `HandleEngage` fires once per `#AUTO ON`/resume, then autopilot sits ON and idle (Phase 3 D-03) | `HandleEngage` becomes one iteration of a paced, cancellable loop | This phase | The single biggest behavioral change; every safety limit (cap, threshold, block-count) becomes meaningful only once iteration is unbounded |
| `recordFailure` always disengages, unconditionally, on any failure kind | Disengage is conditional on transient-vs-non-transient (D-15) and on reaching a threshold | This phase | Requires new failure-kind granularity (Pitfall 1) |
| One Gemini call per decision, no retry | Up to two calls (player + reviewer, unchanged from 3.1) each with up to one 503 retry | This phase (retry only) | Cap accounting must count every one of up to four calls per iteration |
| `ringBuffer` is a pure byte-count window | Ring buffer gains a time dimension (D-09) | This phase | New "never empty" floor logic needed to avoid an engage-after-quiet seeing nothing |

**Deprecated/outdated:** None — this phase only extends Phase 3 code, it does not retire anything Phase 3 built. The one dead statement being removed (D-27, `internal/session/manager.go`'s `EngageAutopilot`) is a pre-existing no-op (`_ = curConnID` at `manager.go:689`, left over from a refactor — `curConnID` is read at line 693 in the no-change branch but this line executes unconditionally after that branch has already returned, making it dead in the changed-state path).

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Recommended pacing constants (settle ~1.5s, floor ~20s, minSpacing ~3s) as starting points | Architecture Patterns / Pattern 1-2 | Low — CONTEXT.md D-06 already frames these as tunable starting points confirmed against Alter Aeon during the staging walkthrough, not locked values |
| A2 | Recommended Immediate Context window (~10s age, ~2-4KB retained-screenful floor, 8KB ceiling unchanged) | Standard Stack / Pattern discussion of `window.go` | Low — D-09 explicitly leaves exact numbers to Claude's Discretion; wrong numbers are a tuning issue, not a correctness issue, since the "never empty" floor is the only hard requirement |
| A3 | Flat string-array memory schema (full replacement) is simpler and equally capable versus the CONTEXT.md-suggested add/update/remove op-list | Don't Hand-Roll, Summary | Medium — this is a genuine design deviation from the owner's stated "intended mechanism" wording in CONTEXT.md's Claude's Discretion section; the discuss-phase or plan review should confirm the owner is comfortable with the simpler shape before implementation, since it does change what the model is asked to produce each turn |
| A4 | Session Memory lives as a JSONB column on the existing `game_sessions` table rather than a new dedicated table | Standard Stack / migration proposal | Low-Medium — reuses an existing table's lifecycle correctly (D-10's "lives for the game connection... at disconnect it is stored with the game session" matches `game_sessions`' own open/close lifecycle exactly), but a reviewer might prefer a dedicated table for future extensibility |
| A5 | Cap-reached and block-repeated-disengage are recorded as `outcome='failed'` with a new `failure_kind` value, not a new `outcome` enum value | Pattern 3 | Low — avoids repeating the exact migration-CHECK-constraint headache D-26 exists to fix; alternative (a 5th outcome value) is also viable but adds another CHECK-constraint migration risk for no clear benefit |
| A6 | `ENCRYPTION_KEY_V1` "outside local development" detection reuses `RAILWAY_ENVIRONMENT != ""`, the same signal already used in `internal/driver/corpus_live_test.go:810-812` | Runtime/Config findings (Q7) | Low — this is the only "are we not on the developer's own machine" signal that exists anywhere in this codebase today; if a future environment (a second Railway service, or a non-Railway host) is added, this check will need to grow with it, but that is true of the existing precedent too |
| A7 | Gemini's structured-output schema has no *documented* numeric depth/size limit but "very large or deeply nested schemas may be rejected" per official docs — treated as a reason to keep the memory schema to one level of array-of-strings nesting, not array-of-objects | Standard Stack, Sources | Low — the recommended flat-string-array design already avoids deep nesting entirely, so this assumption mainly justifies *not* choosing the deeper op-based alternative |
| A8 | Token/latency cost of adding `session_memory`/`quest_memory` fields to the existing answer schema is modest relative to existing reasoning+command output, but is not measured this session (no live model call was made, per instruction) | Summary, Q3 | Medium — if actual cost proves high on `gemini-3.5-flash-lite`, the recommended size ceiling (Claude's Discretion) may need to be tightened during the staging walkthrough; flagged as an Open Question below |

**If this table is empty:** N/A — see rows above.

## Open Questions

1. **What is the actual root cause of Alter Aeon dropping the socket ~30 seconds after every reconnect?**
   - What we know: nothing on the MUDPuppy side self-inflicts this — the idle timeout is explicitly disabled (`internal/session/manager.go:340`, comment "Idle timeout disabled - removed per user request"), the hard session cap is 24 hours checked once a minute (irrelevant at 30s scale), no TCP keepalive is configured either way on the outbound `net.Dial` connection (`manager.go:251`), the only 30-second ticker anywhere in the codebase is the *browser-facing* websocket's ping ticker (`internal/session/websocket.go:317`, unrelated to the outbound MUD TCP socket), and reconnection is manual only — no automatic reconnect logic exists in the frontend (`frontend/src/context/SessionContext.tsx` grep found only manual "Force reconnect" UI, no timer-driven retry). This codebase also sends **zero** telnet option-negotiation replies of any kind back to the MUD (`manager.go:261`'s comment: "Skip telnet negotiation consumption — let the WebSocket handler read all data"; `stripTelnetIAC` in `websocket.go:677-730` only *parses* IAC sequences for display, it never writes a negotiation response).
   - What's unclear: whether Alter Aeon's own server-side idle/negotiation policy (invisible to this codebase — Alter Aeon is a third-party MUD accessed over a plain TCP socket) is what closes the connection at ~30s, e.g. a "client never responded to our option negotiation, treat as non-interactive" timeout, or a "linkdead grace period" tied to the *previous* dead socket rather than the new one. This cannot be determined from code inspection alone and no live-Alter-Aeon test was run this session.
   - Recommendation: proceed with D-19's decision as-is — the loop's WAITING-pause/resume design (Pattern 2's `DisengageHook`/`StopLoop` on park, `EngageLoop`/`StartLoop` on resume) already tolerates this regardless of cause. If the drop proves disruptive during the staging walkthrough, a candidate follow-up (out of Phase 4's scope, a possible future decimal phase) is implementing minimal telnet negotiation replies (e.g., responding `WONT`/`DONT` to unsupported options) so the MUD sees an interactive-looking client — but this is speculative and not required by any Phase 4 success criterion.

2. **Actual token/latency cost of the extended answer schema on `gemini-3.5-flash-lite`**
   - What we know: nested arrays are supported by Gemini's structured output (confirmed via official docs), and this project's own client already successfully uses `propertyOrdering` for a two-field schema in production-adjacent testing (the reviewer's `reason`/`blocked` pair).
   - What's unclear: exactly how much additional output-token cost and latency a `session_memory`/`quest_memory` array adds per turn, since no live call was made this session (instruction: do not spend model quota during research).
   - Recommendation: measure during the plan's own test/staging work (which will make live calls anyway) rather than here; keep the size ceiling conservative (a handful of short bullets) as a starting point and tune if the staging walkthrough shows a latency regression.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | All backend work | Yes | go1.26.0 windows/amd64 | — |
| `go test -race` (requires cgo + a C compiler) | D-20 test suite, if race-checked | Yes | MinGW-w64 GCC at `C:\Users\johna\AppData\Local\Microsoft\WinGet\Packages\...\mingw64\bin\gcc.exe`, `CGO_ENABLED=1` | Same tooling already recorded as required in Phase 2's AR-2-11; no new decision needed |
| Gemini API reachability | Loop's real model calls (staging walkthrough only, not the D-20 unit suite) | Not tested this session (instruction: no live model calls during research) | `gemini-3.5-flash-lite`, confirmed working in Phase 3.1's own staging deploys | D-20's own test suite requires zero network access (fakes only), so this does not block Phase 4 development |
| Railway CLI / staging deploy | Staging walkthrough, not development | Assumed available per every prior phase's evidence trail | — | — |

**Missing dependencies with no fallback:** None.

**Missing dependencies with fallback:** None — the one untested item (live Gemini reachability) has no bearing on the required-by-D-20 unit test suite, which is fake-only by design.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go standard library `testing` (no testify, no fake-clock library — see Don't Hand-Roll) |
| Config file | none — `go.mod` at repo root, `go 1.26` |
| Quick run command | `go test ./internal/driver/... -run TestHandleEngage -count=1` (existing suite, ~0.7s) |
| Full suite command | `go test ./internal/... -count=1` |

Measured this session (`go clean -testcache` then a fresh run): full suite **2.2 seconds** without `-race`, **9.4 seconds** with `-race` (both green, zero failures, on the current Phase 3.1 build). `go build ./...` completes in ~8.5s cold.

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| REQ-continuous-loop | Loop fires multiple decisions, paced by output-arrived signal and floor interval | unit | `go test ./internal/driver/... -run TestLoop_Pacing -v` | ❌ Wave 0 |
| REQ-call-cap-and-error-disengage | Cap halts loop with notice, counting both calls; blank cap never halts | unit | `go test ./internal/driver/... -run TestLoop_CallCap -v` | ❌ Wave 0 |
| REQ-call-cap-and-error-disengage | Threshold disengage with "N of M" notice; blank threshold resolves to 3 | unit | `go test ./internal/driver/... -run TestLoop_ErrorThreshold -v` | ❌ Wave 0 |
| REQ-call-cap-and-error-disengage | 503 retry once, with notice, counted against cap | unit | `go test ./internal/driver/... -run TestHandleEngage_RetryOn503 -v` | ❌ Wave 0 |
| REQ-call-cap-and-error-disengage | Consecutive-block disengage with its own notice, reset by a sent command | unit | `go test ./internal/driver/... -run TestLoop_ConsecutiveBlocks -v` | ❌ Wave 0 |
| REQ-reengage-reassess | First post-engage decision's prompt carries the reassess instruction and a fresh window | unit | `go test ./internal/driver/... -run TestEngageLoop_Reassess -v` | ❌ Wave 0 |
| REQ-safety-limits-hold | No commands issued while disconnected (WAITING), resume restarts loop | unit | `go test ./internal/session/... -run TestManager_LoopStopsOnPark -v` | ❌ Wave 0 |
| REQ-safety-limits-hold | No AI-initiated reconnect (already proven by Phase 2/3's existing tests; re-run as regression) | unit | `go test ./internal/session/... -run TestAutopilot -v` | ✅ existing |
| REQ-safety-limits-hold | Blank settings: no cap, informative failure, no crash, regular play unaffected | unit | `go test ./internal/driver/... -run TestLoop_BlankSettings -v` | ❌ Wave 0 |
| REQ-doc-wheel-grab-and-reengage | DisengageHook cancels an in-flight-sleeping loop immediately | unit | `go test ./internal/session/... -run TestManager_DisengageHookFires -v` | ❌ Wave 0 |

### Sampling Rate

- **Per task commit:** `go test ./internal/driver/... ./internal/session/... -count=1` (targeted, fast)
- **Per wave merge:** `go test ./internal/... -count=1` (full suite, ~2.2s)
- **Phase gate:** Full suite green, plus `go test ./internal/... -race -count=1` (~9.4s) before `/gsd:verify-work`

### Wave 0 Gaps

- [ ] `internal/driver/loop_test.go` — new file, covers REQ-continuous-loop, REQ-call-cap-and-error-disengage, REQ-reengage-reassess
- [ ] `fakeModels` extension in `driver_test.go` — a scripted queue of answers/errors consumed in call order, needed by every multi-iteration test above (today it holds exactly one canned `answer`/`err` pair — `driver_test.go:42-56`)
- [ ] `fakeSessions` extension in `driver_test.go` — real `session.AutopilotState` field driven by the actual `Engage`/`Disengage`/`EnterWaiting`/`Resume` pure functions (`internal/session/autopilot.go:60-116`) instead of `DisengageAutopilot` unconditionally returning `(Off, true)` (today's `driver_test.go:198-203`); plus an `AutopilotStateFor` method, since the `Sessions` interface (`driver.go:87-92`) does not have one yet
- [ ] `fakeSessions.OutputSignal` — a controllable channel the test can send on to simulate "new game output arrived," since no such method exists on the real `Sessions` interface yet either
- [ ] `internal/session/manager_test.go` (or extend the existing autopilot test file) — covers the new `DisengageHook`/`EngageHook` interplay for REQ-doc-wheel-grab-and-reengage
- [ ] Framework install: none — everything above is additive to the existing `testing`-only setup

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V1 Architecture, Design and Threat Modeling | Yes | Goroutine-per-user loop with explicit cancellation (Pattern 2) prevents an orphaned loop from outliving its authorization state |
| V4 Access Control | Yes | Goal/memory REST endpoints reuse the existing `getProfileByConnectionID` ownership check pattern (`internal/profiles/handler.go`), unchanged from every prior AI-settings sub-resource |
| V5 Input Validation | Yes | Goal text length-capped (recommend ~500-1000 chars, matching the keybinding-command precedent at `internal/profiles/handler.go:823-825`, not the 20000-char free-text fields); memory arrays defensively truncated in Go even though the schema requests a bounded count, since Gemini's schema has no confirmed `maxItems` enforcement (see Assumption A7) |
| V6 Cryptography | Yes | `ENCRYPTION_KEY_V1` startup gate (D-25) — never hand-roll key generation as a "convenience" fallback outside local development; `crypto.KeyStore`'s existing AES-256-GCM implementation is otherwise unchanged |
| V8 Data Protection | Yes | Retention job (D-21) is the primary control here — pruning `window_text` and transcript lines on a schedule, with memory layers explicitly exempted per the owner's own instruction |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Memory poisoning: hostile game text convinces the model to write a false or instruction-laden "fact" into Session/Quest Memory, which then re-enters every future prompt as apparently-trusted standing context | Tampering / Elevation of Privilege | D-13's requirement that memory text be delimited as untrusted data in *both* prompts (Pitfall 3) is the direct mitigation; this is a genuinely new injection surface introduced by this phase and should be an explicit line item in the Phase 4 security review, distinct from Phase 3.1's game-text-only threat model |
| Orphaned loop goroutine surviving past its authorization window | Elevation of Privilege | `DisengageHook`/`StopLoop` (Pattern 2) plus the post-iteration `AutopilotStateFor` self-check as defense in depth |
| Unbounded call-cost from a runaway loop | Denial of Service (against the owner's own wallet/quota) | Cap-before-call discipline (Pattern 3) — checked before every one of up to four model calls per iteration, not just once per iteration |
| Startup with a silently-regenerated encryption key outside local development | Tampering (of the credential vault's own integrity) | D-25's fail-fast gate, reusing the existing `RAILWAY_ENVIRONMENT` signal (Assumption A6) |

## Project Constraints (from CLAUDE.md)

- Plans are capabilities with observable acceptance criteria (Phase → Wave → Plan → Task); a plan named for a layer ("backend loop work") is invalid — Phase 4's plans should be named for capabilities (e.g., "The loop paces itself to game output and a floor interval" not "Add loop goroutine").
- Evidence is a canned report, `[AI-PLAYER]` log excerpt, or end-user screenshot — never a database query. Every Phase 4 plan's acceptance criteria must specify one of these three forms.
- Push `ai-player` to GitHub at phase close, after security-review commits land, per the standing rule from the Phase 3 review (DR-3 R-13).
- Deploy staging only via `railway up`; the dashboard's Deploy rebuilds from GitHub and will crash staging if the branch is ever behind.
- Do not read or cite prior `.specify/` specs beyond `ai-game-player-design-v3.md`, `safety-and-abuse-policy-v1.md`, and (new this phase) `ai-memory-model-v1.md` — product is taken as-is.
- No package may be added to `go.mod` or `frontend/package.json` without it showing up in the test report's dependency-drift section (empty is expected and required for Phase 4).

## Sources

### Primary (HIGH confidence)
- Direct code read this session: `internal/driver/driver.go` (full file, 657 lines), `internal/driver/driver_test.go` (fakes and first 130 lines plus the `waitForCalls`/`assertBlockedDecision` helpers), `internal/session/manager.go` (full file, 1051 lines), `internal/session/window.go` (full file), `internal/session/websocket.go` (message types, IAC stripper, ping ticker), `internal/session/handler.go` (full file), `internal/session/autopilot.go` (state-machine functions), `internal/gemini/client.go` (full file, 327 lines), `internal/store/profile.go` (full file), `internal/store/decisions.go` (full file), `internal/store/transcripts.go` (full file), `internal/config/config.go` (full file), `internal/crypto/crypto.go` (full file), `migrations/010_add_ai_fields.up.sql`, `migrations/011_add_ai_session_tables.up.sql`, `migrations/012_add_never_issue_and_blocked_outcome.up.sql` + `.down.sql`, `internal/profiles/handler.go` (AI-settings sub-resource + validation), `cmd/server/main.go` (wiring section, lines 180-310), `frontend/src/components/AIAssistPanel.tsx` (full file), `frontend/src/components/AutopilotBadge.tsx` (full file), `frontend/src/components/AIPlayerPanel.tsx` (full file), `frontend/src/context/SessionContext.tsx` (lines 220-295).
- Live commands run this session: `go version` (go1.26.0 windows/amd64), `go build ./...` (~8.5s, clean), `go test ./internal/... -count=1` (2.2s, all green), `go test ./internal/... -race -count=1` (9.4s, all green), `where gcc` (MinGW-w64 present).
- `ai.google.dev/gemini-api/docs/structured-output` — WebFetch this session, confirming nested arrays of objects are supported in `responseSchema` and that "very large or deeply nested schemas may be rejected" (no numeric limit documented).
- This project's own prior research (`03-RESEARCH.md`, `03.1-RESEARCH.md`) for the Gemini REST contract and `propertyOrdering`'s documented behavior — not re-verified from scratch this session since it is already HIGH-confidence, code-proven (the reviewer's `reason`-before-`blocked` ordering ships and is tested today).

### Secondary (MEDIUM confidence)
- WebSearch this session for `propertyOrdering` recursive/nested behavior — corroborated qualitatively by multiple sources (Firebase AI Logic docs, Google Cloud Vertex AI docs) but not cross-checked against a single canonical ai.google.dev page for the recursive-nesting claim specifically; this project's own already-shipped flat-object usage is the stronger evidence and is HIGH confidence.

### Tertiary (LOW confidence)
- The Alter Aeon socket-drop root cause (Open Question 1) — no authoritative source exists; Alter Aeon's server behavior is not documented anywhere accessible to this research and was not tested live this session.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies, every extension point verified by direct code read
- Architecture (loop, hooks, counters): HIGH for the mechanism (all existing collaborators and call sites read directly), MEDIUM for the exact numeric pacing constants (explicitly Claude's Discretion, un-tuned against live gameplay)
- Memory schema (Gemini structured output): HIGH for feasibility (nested arrays confirmed by official docs and by this project's own already-shipped `propertyOrdering` usage), LOW for cost/latency (not measured, no live call made per instruction)
- Pitfalls: HIGH — Pitfall 1 (failure-kind taxonomy gap) and Pitfall 3 (prompt-builder signature gap) are both concrete, line-cited code facts, not speculation
- Socket-drop investigation: HIGH for what is ruled out on this codebase's side, LOW/unresolved for the actual remote cause (flagged as Open Question 1, consistent with D-19's own instruction that the loop must tolerate this "regardless")

**Research date:** 2026-09-17
**Valid until:** 14 days (short window, matching Phase 3's own precedent, because the Gemini model-naming/behavior landscape and the live Alter Aeon socket behavior are both moving targets within this project's timeline; re-verify the `gemini-3.5-flash-lite` model is still serving and re-check the socket-drop timing if more than two weeks elapse before the staging walkthrough)
