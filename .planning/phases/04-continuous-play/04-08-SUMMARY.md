---
phase: 04-continuous-play
plan: 08
subsystem: driver
tags: [go, gemini, postgres, react, typescript, memory, prompt-injection]

# Dependency graph
requires:
  - phase: 04-continuous-play
    provides: "04-07's promptContext (Profile, Goal, QuestBullets, SessionMemory), the untrusted <QUEST_MEMORY>/<SESSION_MEMORY> markers and D-12 clampBullets read path, and internal/store/transcripts.go's SessionMemoryFor read; 04-06's QuestStore.UpdateBullets (written but unused) and profiles.SessionGoal"
provides:
  - "internal/gemini/client.go: session_memory/quest_memory optional array-of-string fields on the decision answer schema (propertyOrdering reasoning, command, session_memory, quest_memory), decoded tolerantly via decodeAnswer so a malformed memory field never costs the command"
  - "internal/driver/memory.go: truncateBullets (D-12 ceilings on the way out, trims and drops empty/whitespace-only entries, always non-nil) and bulletBytes for the memory log line"
  - "internal/driver/driver.go: persistMemory applies the replace-or-leave-alone rule once per decision, right after the answer is decoded -- covers every outcome sharing that answer (sent, blocked, or a shape failure that still parsed); Quests/Memory interfaces widened with UpdateBullets/UpdateSessionMemory; Event.SessionMemory rides every notified event via decorateEvent/currentSessionMemory"
  - "internal/store/transcripts.go: UpdateSessionMemory (single-statement write) and SessionMemoryForConnection (reads the most recent open game session's memory, for the read endpoint)"
  - "GET /api/v1/profiles/{connection_id}/ai-memory (internal/profiles/handler.go GetSessionMemory), GET-only, ownership via getProfileByConnectionID, fails closed 503 with no transcripts store wired"
  - "The read-only, collapsible Session Memory section in AIAssistPanel.tsx: header with disclosure glyph and count, disc-bulleted list, 'No session memory yet.' empty state, loads once via getSessionMemory and replaced wholesale from every ai message carrying session_memory"
  - "cmd/server/main.go wires aiDriver.SetQuests(questStore)/SetMemory(transcriptStore) and maps Event.SessionMemory onto the websocket AIDecisionPayload"
affects: [04-continuous-play plan 04-10 (harness ownership/not-owned-connection evidence for ai-memory), plan 04-11 (staging walkthrough screenshots evidence/10-session-memory-expanded.png, evidence/11-session-memory-after-refresh.png; Phase 4 security review carries memory poisoning as a new active risk, T-4-03)]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Memory rides the same answer JSON the command already carries (flat string[] rewritten in full, not an add/update/remove op-list) -- no extra model call, no entry-id bookkeeping; absent means leave memory alone, present means replace wholesale"
    - "persistMemory runs once, immediately after a successful answer decode, before any downstream branching (shape validation, Never-issue, reviewer, dispatch) -- one call site covers every outcome that shares that answer instead of threading memory fields through recordFailure/recordBlocked/recordSuccess"
    - "Event.SessionMemory is populated by decorateEvent on every notified event (decision or system), reading back through the same Memory collaborator persistMemory just wrote, so the browser always receives the current stored list, never a value computed only in memory"
    - "A malformed schema field (session_memory sent as a string instead of an array) is isolated with json.RawMessage in an intermediate wire struct so it can be dropped without failing the outer decode of reasoning/command"

key-files:
  created: []
  modified:
    - internal/gemini/client.go
    - internal/gemini/client_test.go
    - internal/driver/driver.go
    - internal/driver/memory.go
    - internal/driver/driver_test.go
    - internal/store/transcripts.go
    - internal/session/websocket.go
    - internal/session/websocket_test.go
    - internal/profiles/handler.go
    - internal/profiles/logs_test.go
    - cmd/server/main.go
    - frontend/src/services/api.ts
    - frontend/src/types/index.ts
    - frontend/src/components/AIAssistPanel.tsx
    - frontend/src/index.css
    - public/index.html
    - public/assets/index-Cj_-mmeJ.js
    - public/assets/index-Cj_-mmeJ.js.map
    - public/assets/index-UwP9oyAs.css

key-decisions:
  - "Ceilings shipped exactly as 04-07 already fixed them: 30 Session Memory bullets / 20 Quest bullets, each truncated to 200 characters (D-12), enforced again on the way out by truncateBullets independently of clampBullets' inbound enforcement"
  - "The replace-or-leave-alone rule is implemented as a single call site (persistMemory, right after the answer decodes) rather than threading memory fields through recordFailure/recordBlocked/recordSuccess -- persists on every outcome downstream of a successful answer (sent, blocked, shape failure), never on outcomes with no answer at all (missing profile/model, a transport/auth error, a cap halt before the call)"
  - "The route path is GET /api/v1/profiles/{connection_id}/ai-memory, matching the ai-goal/ai-settings sub-resource naming convention; response shape is {\"session_memory\": string[]}"
  - "The memory update log line is one combined '[AI-PLAYER] memory' line per decision that changed memory, carrying user_id/connection_id/decision_id (blank, matching the pre-storage stage-log convention) plus session/quest bullet counts and byte totals -- never bullet text"
  - "Quest Memory is looked up by ActiveQuestFor(connID, goal) rather than threading a Quest id through the answer path; a blank goal or no active Quest skips the write silently, never an error, and the command still sends"

patterns-established:
  - "A schema field that must tolerate a wrong JSON type (array vs string) decodes through an intermediate json.RawMessage wire struct rather than failing the whole payload -- the pattern this package now uses for any future optional structured field on the decision answer"

requirements-completed: [REQ-continuous-loop, REQ-doc-continuous-visible-play]

# Metrics
duration: 15min
completed: 2026-09-16
---

# Phase 4 Plan 08: Session Memory and Quest Memory Curation Summary

**The model rewrites a flat curated bullet list on the same JSON answer that already carries its command (no extra model call), Go truncates it to D-12's ceilings before storing it against the game session or the active Quest, and the owner watches the current list live in a collapsible, read-only Session Memory section of the AI Assist panel.**

## Performance

- **Duration:** 15 min (75fe5b2 to ff38f2c)
- **Started:** 2026-09-16T22:50:59-07:00
- **Completed:** 2026-09-16T23:05:59-07:00
- **Tasks:** 3 completed
- **Files modified:** 18 (14 source/test + 4 rebuilt production bundle files)

## Accomplishments

- **Schema and decode (D-10):** `internal/gemini/client.go`'s decision-answer schema gained `session_memory`/`quest_memory` optional string-array fields with `propertyOrdering: [reasoning, command, session_memory, quest_memory]`; `Required` still names only `reasoning`/`command`. A malformed shape (e.g. `session_memory` sent as a plain string) is isolated via an intermediate `json.RawMessage` wire struct so it decodes to nil without ever failing the command's own decode — proven by `TestAnswerCarriesMemoryFields`'s four cases (schema shape, both present, neither present, malformed).
- **Curation and persistence (D-10, D-11, D-12, T-4-10):** `truncateBullets` (new in `internal/driver/memory.go`) trims, drops empty/whitespace-only entries, and caps at 30 Session Memory / 20 Quest bullets and 200 characters each, always returning a non-nil slice so "replace with nothing" stays expressible. `persistMemory` (new in `driver.go`) applies the replace-or-leave-alone rule once per decision, immediately after the answer decodes and before any downstream branching — so it runs on every outcome sharing that answer (sent, blocked, or a shape failure that still parsed), never on an outcome with no answer at all. Session Memory writes through the `Memory` collaborator (`TranscriptStore.UpdateSessionMemory`, a new single-statement write); Quest Memory writes through `Quests.ActiveQuestFor` + `UpdateBullets`, skipped silently when the goal is blank. A storage error is logged (`[AI-PLAYER] memory-store-error`) and never blocks the command — proven by `TestDriverPersistsCuratedMemory`'s four cases.
- **Live push (D-10):** `Event` and `AIDecisionPayload` both gained a `SessionMemory`/`session_memory` field carrying the full curated list (never a diff), populated by `decorateEvent`/`currentSessionMemory` on every notified event, decision or system — so the panel's section always reflects what was just stored, proven by the "both present" test case asserting the pushed event's list matches the stored one.
- **Read endpoint (D-10):** `GET /api/v1/profiles/{connection_id}/ai-memory` (`GetSessionMemory`), GET-only, ownership resolved through `getProfileByConnectionID` exactly like every other sub-resource (T-4-09), reading `TranscriptStore.SessionMemoryForConnection`'s most-recent-open-game-session lookup. No PUT counterpart. `TestGetSessionMemory` covers stored bullets, no-open-session (empty array, never an error), not-owned-connection refusal, and a fail-closed 503 with no transcripts store wired.
- **Panel (D-10):** the collapsible Session Memory section is the last child of `.ai-assist-panel-top`, below the status line: `▸`/`▾ Session Memory ({count})` header, a disc-bulleted list when expanded and non-empty, `No session memory yet.` when expanded and empty. Defaults collapsed, component-only state (resets on remount). Loads once via `getSessionMemory` on mount/connection-change and is replaced wholesale from every `ai` message carrying `session_memory`. No textarea, input, or per-item control anywhere in the section.

## Task Commits

Each task was committed atomically:

1. **Task 04-08-01: The answer that carries the command also carries the curated memory** - `75fe5b2` (feat)
2. **Task 04-08-02: What the model chooses to remember is truncated, stored and pushed (part 1: driver/store)** - `f8133a0` (feat)
2b. **Task 04-08-02: What the model chooses to remember is truncated, stored and pushed (part 2: wire the real stores)** - `7eef6c7` (feat)
3. **Task 04-08-03: The owner can open the AI's memory and read it** - `ff38f2c` (feat)

## Files Created/Modified

- `internal/gemini/client.go` / `client_test.go` — `schemaProperty.Items`, the two memory fields, `decodeAnswer`'s tolerant decode; `TestAnswerCarriesMemoryFields`
- `internal/driver/memory.go` — `truncateBullets`, `bulletBytes`
- `internal/driver/driver.go` / `driver_test.go` — `Quests.UpdateBullets`, `Memory.UpdateSessionMemory`, `Event.SessionMemory`, `currentSessionMemory`, `persistMemory`; `TestDriverPersistsCuratedMemory`, extended `TestMemoryCeilingsAreEnforced`; fake `fakeQuestStore`/`fakeMemoryStore`
- `internal/store/transcripts.go` — `UpdateSessionMemory`, `SessionMemoryForConnection`
- `internal/session/websocket.go` / `websocket_test.go` — `AIDecisionPayload.SessionMemory`; `TestPushAI`'s equality check moved to `reflect.DeepEqual` (Rule 1 — a `[]string` field broke the existing `!=` struct comparison)
- `internal/profiles/handler.go` / `logs_test.go` — `SessionMemoryResponse`, `GetSessionMemory`, `transcriptStorage.SessionMemoryForConnection`; `TestGetSessionMemory`; `fakeTranscriptStore.SessionMemoryForConnection`
- `cmd/server/main.go` — `aiDriver.SetQuests`/`SetMemory` wiring, `ai-memory` route, `SessionMemory` in the notifier mapping
- `frontend/src/services/api.ts`, `types/index.ts` — `getSessionMemory`, `SessionMemoryResponse`, `AIDecisionPayload.session_memory`
- `frontend/src/components/AIAssistPanel.tsx` — the Session Memory section, its load effect, and the `handleAI` wholesale-replace branch
- `frontend/src/index.css` — the four `.ai-assist-memory*` classes
- `public/index.html`, `public/assets/index-Cj_-mmeJ.js[.map]`, `index-UwP9oyAs.css` — rebuilt production bundle (`npm run build`)

## Decisions Made

See `key-decisions` in the frontmatter above.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `TestPushAI`'s struct equality check broke when `AIDecisionPayload` gained a slice field**
- **Found during:** Task 04-08-02, after wiring `AIDecisionPayload.SessionMemory` (`[]string`)
- **Issue:** `go vet` failed: `internal/session/websocket_test.go:83` compared `*got[0].Decision != payload` — Go structs containing a slice field cannot be compared with `==`/`!=`.
- **Fix:** Replaced the comparison with `!reflect.DeepEqual(*got[0].Decision, payload)`, adding the `reflect` import. No other comparison in the file used this pattern.
- **Files modified:** `internal/session/websocket_test.go`
- **Verification:** `go vet ./internal/session/...` and `go test ./internal/session/...` both pass.
- **Committed in:** `7eef6c7` (part of task 2's second commit)

**2. [Rule 2 - Missing critical] Added `TestGetSessionMemory` for the new read endpoint**
- **Found during:** Task 04-08-03
- **Issue:** The plan's task-3 acceptance criteria list only source assertions and a green build for `GetSessionMemory`; no Go test was named. A new HTTP endpoint touching ownership (T-4-09) and a fail-closed dependency (no transcripts store) without any test coverage would be a correctness/security gap.
- **Fix:** Added `TestGetSessionMemory` to `internal/profiles/logs_test.go`, mirroring the existing `TestListSessionsScopedToOwner`/`TestGetSessionTranscript` style: stored-bullets round-trip, no-open-session returns an empty (never nil) array, not-owned-connection refusal, and a fail-closed 503 with no transcripts store wired.
- **Files modified:** `internal/profiles/logs_test.go`
- **Verification:** `go test ./internal/profiles/... -run TestGetSessionMemory -v` — all four cases PASS.
- **Committed in:** `ff38f2c` (Task 3 commit)

---

**Total deviations:** 2 auto-fixed (1 bug, 1 missing critical test coverage)
**Impact on plan:** Both necessary for a passing build and for baseline correctness/security coverage of a new endpoint. No scope creep — no new behavior beyond what the plan specified.

## Issues Encountered

Task 04-08-02 was committed in two parts: the driver/store logic and tests landed first (`f8133a0`), then the remaining declared files for that task (`internal/session/websocket.go`, `cmd/server/main.go`'s `SetQuests`/`SetMemory` wiring and notifier mapping) were completed and committed separately (`7eef6c7`) after noticing the task's full file list had not yet been covered. Both commits are additive, `go build`/`go vet`/`go test` were green after each, and the task's full declared scope is covered across the two.

## User Setup Required

None — no external service configuration required. No live Gemini call was made; token/latency cost of the extended answer schema remains plan 04-11's staging-walkthrough measurement per 04-RESEARCH's Open Question 2.

## Threat Flags

None new beyond what `04-08-PLAN.md`'s own threat model already declares (T-4-03 memory poisoning, T-4-10 unbounded growth, T-4-09 IDOR, T-4-29 memory-loss-costs-a-command, T-4-05 information disclosure, T-4-SC supply chain) — every file touched is inside that register. T-4-03 (memory poisoning: a bullet written under hostile game text's influence re-entering every later prompt as apparent fact) is carried forward as this phase's new active risk to the Phase 4 security review, per the plan's own instruction, with three dispositions and nothing pre-chosen.

## Next Phase Readiness

- Session Memory and Quest Memory now flow end to end: model proposes on the existing answer → Go truncates and stores → the panel shows the current list live and after a refresh (server-side read, not a client cache) → the same bullets re-enter every later prompt as untrusted data (04-07's wrapping, unchanged).
- Quest Memory is stored and persists across sessions but remains deliberately unshown on the panel (D-11) and unclosed (Phase 4 never sets a Quest's status to anything but `active` — `internal/store/quests.go` still mechanically proves this via `TestQuestStore_NeverCloses`, untouched by this plan).
- No blockers. `go build ./...`, `go vet ./internal/... ./cmd/...`, `go test ./internal/... -count=1`, `go test ./internal/... -race -count=1`, and `cd frontend && npm run build` all pass on this machine; `git diff --stat go.mod frontend/package.json` is empty (T-4-SC).
- Plans 04-09 through 04-11 (the remaining Phase 4 waves) inherit a driver whose prompt context and memory persistence are both proven; nothing here closes a Quest, edits memory by hand, or measures live-model cost — all explicitly out of this plan's scope per the objective.

---
*Phase: 04-continuous-play*
*Completed: 2026-09-16*

## Self-Check: PASSED

- FOUND: internal/gemini/client.go
- FOUND: internal/gemini/client_test.go
- FOUND: internal/driver/driver.go
- FOUND: internal/driver/memory.go
- FOUND: internal/driver/driver_test.go
- FOUND: internal/store/transcripts.go
- FOUND: internal/session/websocket.go
- FOUND: internal/session/websocket_test.go
- FOUND: internal/profiles/handler.go
- FOUND: internal/profiles/logs_test.go
- FOUND: cmd/server/main.go
- FOUND: frontend/src/services/api.ts
- FOUND: frontend/src/types/index.ts
- FOUND: frontend/src/components/AIAssistPanel.tsx
- FOUND: frontend/src/index.css
- FOUND: public/assets/index-Cj_-mmeJ.js
- FOUND: public/assets/index-Cj_-mmeJ.js.map
- FOUND: public/assets/index-UwP9oyAs.css
- FOUND: .planning/phases/04-continuous-play/04-08-SUMMARY.md
- FOUND: commit 75fe5b2 (Task 04-08-01)
- FOUND: commit f8133a0 (Task 04-08-02, part 1)
- FOUND: commit 7eef6c7 (Task 04-08-02, part 2)
- FOUND: commit ff38f2c (Task 04-08-03)
- FOUND: commit 377b0a2 (docs: plan summary)
