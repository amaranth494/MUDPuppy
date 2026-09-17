---
phase: 04-continuous-play
plan: 06
subsystem: api
tags: [go, postgres, migration, react, typescript, quest-memory]

# Dependency graph
requires:
  - phase: 04-continuous-play
    provides: "04-04's .ai-assist-panel-top region and standing status line, which the goal box now shares; the AIDecisionPayload/outcome union already carrying 'goal' and the .ai-system-line.state-goal CSS rule, both added ahead of this plan"
provides:
  - "Migration 013: profiles.session_goal (TEXT NOT NULL DEFAULT ''), game_sessions.session_memory (JSONB, unused until plan 04-08), and the quests table with a partial unique index on (connection_id, goal_text_normalized) WHERE status = 'active' making reactivate-or-create one atomic upsert"
  - "internal/store/quests.go: QuestStore with NormalizeGoal (the one definition of goal identity), EnsureActiveQuest, ActiveQuestFor, UpdateBullets (unused until plan 04-08); no method anywhere in the file ever sets status to anything but 'active'"
  - "SessionGoal as the fourth sibling of the ConductRules/ApproachGuidance/NeverIssueList chain on internal/store/profile.go's Profile/ProfileUpdate/GetProfile/GetProfileByConnection/UpdateProfile"
  - "GET/PUT /api/v1/profiles/{connection_id}/ai-goal (internal/profiles/handler.go GetGoal/PutGoal), reusing getProfileByConnectionID's ownership check, a 1000-character cap, and a nil-safe AINotifier hook (SetAINotifier) pushing the locked 'Goal changed: ...'/'Goal cleared' system line through the same wsHandler.PushAI path the driver's own notifications use"
  - "The Session Goal box at the top of the AI Assist panel: loads on mount/connection change, commits on blur/Enter, no Save button, failed saves surfaced through the panel's existing system-line mechanism without clearing the owner's typing"
affects: [04-continuous-play plans 04-07 (puts the goal and Quest bullets in the prompt), 04-08 (Session Memory, first user of session_memory and UpdateBullets), 04-10 (harness goal round-trip evidence), 04-11 (staging walkthrough screenshot evidence/06-goal-set.png)]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Quest reactivate-or-create is a single INSERT ... ON CONFLICT (connection_id, goal_text_normalized) WHERE status = 'active' DO UPDATE round trip against a partial unique index -- the first ON CONFLICT usage in this codebase's migrations, avoiding a SELECT-then-INSERT race"
    - "Quest identity is create-vs-reactivate detected in the handler by comparing Quest.CreatedAt.Equal(Quest.UpdatedAt) after EnsureActiveQuest returns, rather than widening the store's locked (Quest, error) signature"
    - "The goal endpoint's browser push (AINotifier) mirrors internal/driver.NotifierFunc's shape exactly, so cmd/server/main.go routes both the driver's decision events and the handler's goal-changed event through the identical wsHandler.PushAI call -- one message-delivery mechanism, not two"
    - "Store-level SQL statements are declared as named constants (ensureActiveQuestSQL, activeQuestForSQL, updateBulletsSQL) specifically so no-database tests can assert on their exact text (conflict target, and that status is never set to a terminal word), following the same stripped-comment-line technique migrations_test.go already uses on migration files"

key-files:
  created:
    - migrations/013_add_goal_memory_and_quests.up.sql
    - migrations/013_add_goal_memory_and_quests.down.sql
    - internal/store/quests.go
    - internal/store/quests_test.go
  modified:
    - internal/store/profile.go
    - internal/profiles/handler.go
    - internal/profiles/handler_test.go
    - cmd/server/main.go
    - frontend/src/services/api.ts
    - frontend/src/types/index.ts
    - frontend/src/components/AIAssistPanel.tsx
    - public/index.html
    - public/assets/index-V0ekvq_f.js
    - public/assets/index-V0ekvq_f.js.map

key-decisions:
  - "Goal length cap shipped at exactly 1000 characters (04-CONTEXT.md's locked discretion), enforced in PutGoal's validator before any store write"
  - "The created-vs-reactivated word in the [AI-PLAYER] goal log line is derived by comparing Quest.CreatedAt.Equal(Quest.UpdatedAt) in the handler, since EnsureActiveQuest's signature is locked to (Quest, error) by the plan and carries no explicit created flag; both timestamps come from the same statement's NOW() evaluation on insert, so they are exactly equal only on a fresh row"
  - "PutGoal saves the goal text even when no QuestStore is wired (h.quests == nil): the goal box's own persistence never depends on Quest bookkeeping succeeding, matching the handler's other nil-safe dependency guards (transcripts, decisions)"
  - "index.css needed no changes this plan: 04-UI-SPEC.md's .form-group/.form-label/.form-input/.form-hint reuse is verbatim and .ai-assist-panel-top already existed from plan 04-04"

patterns-established:
  - "A goal save's failure path reuses the panel's existing system-line list (kind: 'system', outcome: 'failed') rather than a new toast/banner component -- consistent with the project's 'no speed bumps' plumbing-decision guidance"

requirements-completed: [REQ-continuous-loop, REQ-doc-continuous-visible-play]

# Metrics
duration: 9min
completed: 2026-09-16
---

# Phase 4 Plan 06: Session Goal and Quest Memory Storage Summary

**A one-line, server-owned session goal (profiles.session_goal) that survives refresh/reconnect/#AUTO OFF-ON, a Quest record it names via an atomic partial-unique-index upsert, and a goal box at the top of the AI Assist panel that saves on blur/Enter with the server's own "Goal changed: ..." line as its only confirmation.**

## Performance

- **Duration:** 9 min (26c9b86 to 3e0d5f0)
- **Started:** 2026-09-16T22:11:40-07:00
- **Completed:** 2026-09-16T22:20:36-07:00
- **Tasks:** 3 completed
- **Files modified:** 13 (10 source + 3 rebuilt production bundle files)

## Accomplishments

- Migration 013 gives the goal (D-01), Session Memory (D-10, unused until plan 04-08), and Quest Memory (D-11) their storage: `profiles.session_goal`, `game_sessions.session_memory`, and the `quests` table with a partial unique index on `(connection_id, goal_text_normalized) WHERE status = 'active'` that makes reactivate-or-create a single atomic upsert (T-4-07) — proven by `TestQuestStore_EnsureActiveReactivates` and the migration's own `CREATE UNIQUE INDEX` source assertion.
- `internal/store/quests.go`'s `QuestStore` can only ever leave a Quest `'active'` — no method sets any other status, and `TestQuestStore_NeverCloses` mechanically checks that `succeeded`, `failed`, `abandoned`, and `invalidated` appear nowhere in the file's SQL (D-04: Phase 4 closes nothing).
- `GET`/`PUT /api/v1/profiles/{connection_id}/ai-goal` is a full sibling of the `ai-settings` sub-resource: same ownership check (`getProfileByConnectionID`, T-4-09), same ValidationError shape, this time capped at 1000 characters (T-4-10). Saving a non-blank goal reactivates-or-creates its Quest; a blank goal clears the field and touches no Quest.
- Saving a goal pushes exactly one system line through the same `wsHandler.PushAI` path the driver's decisions already use: `Goal changed: {goal}` for a non-blank goal, `Goal cleared` for a blank one, both byte-identical to `04-UI-SPEC.md` — proven by source assertion and `TestGoalRoundTrip`.
- The owner can now type a session goal into a one-line box at the top of the AI Assist panel, above the status line; it loads on mount and on connection change, commits on blur or Enter with no Save button, and a failed save leaves the typed text in place and reports itself through the panel's existing system-line mechanism instead of silently reverting.

## Task Commits

Each task was committed atomically:

1. **Task 04-06-01: The goal and the Quest have somewhere to live** - `26c9b86` (feat)
2. **Task 04-06-02: The goal is set and read over HTTP, and saving it says so on screen** - `30bd4e7` (feat)
3. **Task 04-06-03: The goal box sits at the top of the panel and is editable while the AI plays** - `3ad375e` (feat)

**Production bundle rebuild:** `3e0d5f0` (build)

**Plan metadata:** (this commit)

## Files Created/Modified

- `migrations/013_add_goal_memory_and_quests.up.sql` / `.down.sql` — session_goal, session_memory, the quests table and its partial unique index; down drops the table before the two columns
- `internal/store/profile.go` — `SessionGoal` added as the fourth sibling field/column throughout the existing standing-text chain
- `internal/store/quests.go` — `QuestStore`, `NormalizeGoal`, `EnsureActiveQuest`, `ActiveQuestFor`, `UpdateBullets`; SQL statements declared as named constants for no-database test assertions
- `internal/store/quests_test.go` — `TestNormalizeGoal`, `TestQuestStore_EnsureActiveReactivates`, `TestQuestStore_NeverCloses`
- `internal/profiles/handler.go` — `GoalResponse`, `GetGoal`, `PutGoal`, `questStorage` interface, `AIEvent`/`AINotifier`, `SetQuestStore`, `SetAINotifier`; the `[AI-PLAYER] goal` log line
- `internal/profiles/handler_test.go` — `fakeQuestStore`; `TestGoalRoundTrip`, `TestGoalRefusesNotOwnedConnection`; `fakeProfileStore.UpdateProfile` extended for `SessionGoal`
- `cmd/server/main.go` — `QuestStore` constructed beside the decision store, wired to the handler; `ai-goal` route registered; the goal notifier routed through `wsHandler.PushAI`
- `frontend/src/services/api.ts` — `getGoal`/`putGoal`
- `frontend/src/types/index.ts` — `GoalResponse`
- `frontend/src/components/AIAssistPanel.tsx` — the goal box as the first child of `.ai-assist-panel-top`; load/commit effects; failed-save system-line fallback
- `public/index.html`, `public/assets/index-V0ekvq_f.js[.map]` — rebuilt production bundle (`npm run build`)

## Decisions Made

See `key-decisions` in the frontmatter above.

## Deviations from Plan

None — plan executed exactly as written. `EnsureActiveQuest`'s signature was kept exactly as the plan specified `(Quest, error)`; the created-vs-reactivated distinction the log line needs was derived in the handler from the returned `Quest`'s own timestamps rather than widening the store's contract, which is a within-plan implementation choice, not a deviation from any locked interface.

## Issues Encountered

None.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- The goal does not yet reach the model's prompt — that is plan 04-07's job, as the plan's own objective states. Nothing here is a stub: the goal is fully stored, served, and displayed; it simply is not yet read by the driver.
- `session_memory` and `UpdateBullets` are present in the schema/store but unused until plan 04-08 curates Session Memory and Quest Memory during play, per this plan's own scope.
- No blockers. `go build ./...`, `go vet ./internal/store/... ./internal/profiles/... ./cmd/...`, `go test ./internal/... -count=1`, `go test ./internal/... -race -count=1`, and `cd frontend && npm run build` all pass on this machine; `git diff --stat go.mod frontend/package.json` is empty (T-4-SC).
- `internal/driver`, `internal/session/manager.go`, and `internal/session/window.go` were not touched by this plan, confirmed by `git diff --stat` over this plan's commit range — leaving them untouched for sibling wave-4 plans.

---
*Phase: 04-continuous-play*
*Completed: 2026-09-16*
