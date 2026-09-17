---
phase: 05-coaching-channel
plan: 09
subsystem: ui
tags: [help, docs, react, typescript, go, rest-api, logs-page]

# Dependency graph
requires:
  - phase: 05-coaching-channel
    provides: "05-06's coaching/conversation stores and GetCoaching/GetConversation REST endpoints; 05-07's ChatLine/CoachingResponse/ConversationResponse types and the panel's Coaching in effect list, chat strip and Pause/Resume"
provides:
  - "help/ai-coaching.json: the server-authored Help article (slug ai-coaching) covering the two levels, how a message reaches AI-player, how long coaching lasts and how to take it back, what chat cannot do, the by-hand copy into Conduct rules/Approach guidance, Pause/Resume, and the two pop-out windows"
  - "frontend/src/pages/HelpPage.tsx: 'ai-coaching' in SECTION_ORDER between 'safety' and 'troubleshooting'"
  - "internal/store/conversation.go: ConversationForSession now takes (gameSessionID, connectionID), joined through game_sessions for defence in depth, matching GetSessionLines' precedent"
  - "internal/profiles: GetSessionConversation, GET-only, wired to /api/v1/profiles/{connection_id}/sessions/{session_id}/conversation"
  - "frontend/src/services/api.ts: getSessionConversation(connectionId, sessionId), reusing the ChatLine type"
  - "frontend/src/pages/LogsPage.tsx: the .logs-conversation section, its own fetch-on-select effect, rendered as a true sibling below .logs-transcript"
  - "confirmation that design v3, REQUIREMENTS.md, ROADMAP.md and PROJECT.md already state plain-text chat with no promotion and the help article (D-22) -- no document amendment was needed"
affects: [05-11-security-review-and-close]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "A plain layout-only wrapper div with an inline style (no index.css edit) restores a flex column's structure when a shared stylesheet another parallel plan depends on cannot be reopened -- the styled child's own className and markup stay byte-identical"
    - "A per-session read exposed over HTTP by adding the connection-scoping parameter to an already-declared-but-unused store method (ConversationForSession), rather than inventing a second method, mirroring the sibling transcript read's own defence-in-depth JOIN"

key-files:
  created:
    - help/ai-coaching.json
  modified:
    - frontend/src/pages/HelpPage.tsx
    - frontend/src/pages/LogsPage.tsx
    - frontend/src/services/api.ts
    - internal/store/conversation.go
    - internal/store/conversation_test.go
    - internal/profiles/handler.go
    - internal/profiles/logs.go
    - internal/profiles/coaching_test.go
    - cmd/server/main.go

key-decisions:
  - "The Logs page's per-session conversation read did not exist over HTTP (only the connection-scoped, login-bounded ConversationFor did, already used by the live panel) -- ConversationForSession existed in the store but was unused and unscoped. Widened it to take connectionID, added the JOIN defence-in-depth GetSessionLines already uses, and added GetSessionConversation/the new route, per the plan's own explicit instruction for this exact situation."
  - "Reused ChatLine (plan 05-07) for the Logs page conversation lines rather than declaring a second type, converting the numeric id to a string exactly as ChatLine's own doc comment anticipates for a reload."
  - "index.css was not reopened (owned by a parallel plan this wave). Since .logs-conversation's own CSS (margin-top/border-top/padding-top, no flex or overflow) already assumed a vertical stack, and .logs-transcript already carries flex:1 as a direct flex child of the row-flex .logs-layout, a plain wrapper div with an inline style (flex:1, column, minHeight:0) was added around both panes so the conversation section is a true DOM sibling below .logs-transcript -- .logs-transcript's own opening tag, closing tag and every line of its rendering logic are byte-identical to before this plan (confirmed by git diff)."
  - "All four documents (design v3, REQUIREMENTS.md, ROADMAP.md, PROJECT.md) already stated the D-22 amendment (plain-text chat, no promotion, help article) before this plan ran -- confirmed by direct grep with line numbers below. No document was amended."

requirements-completed: [REQ-promote-guidance, REQ-doc-coaching]

# Metrics
duration: ~90min
completed: 2026-09-17
---

# Phase 5 Plan 9: Help Article and Logs Page Conversation Summary

**A server-authored Help article ("AI-chatter & AI-player") explains the two levels and the by-hand copy into AI settings, and the Logs page gains its own read-only Coaching Conversation section below the transcript, backed by a newly-exposed per-session HTTP read.**

## Performance

- **Duration:** ~90 min (commit-to-commit, base 87b52d3 to 1de6ad3)
- **Tasks:** 2
- **Files modified:** 10 (1 created, 9 modified)

## Accomplishments

- `help/ai-coaching.json` carries the locked slug (`ai-coaching`), title (`AI-chatter & AI-player`) and description, in the exact `HelpSection`/`Section` shape `help/safety.json` uses, requiring zero Go change (`internal/help`'s directory scan needs only the new file) — confirmed by `git diff --stat -- internal/help/` being empty.
- The article names every shipped surface it describes: `Sent to AI-player:`, `Coaching in effect`, `[Coaching received]`, `Conduct rules`, `Approach guidance`, `Paused by owner`, `Pop out` and `Bring back` all appear exactly once; a case-insensitive grep for promotion, promote, a clear button, `#AUTO PAUSE` or editing Session Memory returns zero matches — none of the withdrawn or non-existent features are described.
- `'ai-coaching'` sits in `HelpPage.tsx`'s `SECTION_ORDER` between `'safety'` and `'troubleshooting'`; `renderContent`/`renderInline` are untouched (`git diff` shows zero lines mentioning either function).
- All four canonical documents (design v3, REQUIREMENTS.md, ROADMAP.md, PROJECT.md) already stated the D-22 amendment — plain-text chat, no promotion, a help article explaining the by-hand copy — before this plan ran. No document needed amendment. Line numbers recorded below.
- The Logs page shows a past session's coaching conversation in its own labelled section below the transcript pane, never woven into it: `Coaching Conversation`, the empty state `No coaching conversation recorded for this session.`, and `You: {text}` / `AI-chatter: {text}` lines colored by the same human/chatter convention the transcript already uses.
- The per-session conversation read did not exist over HTTP before this plan (only the connection-scoped, login-bounded read the live panel uses did). Exposed it as `GET /api/v1/profiles/{connection_id}/sessions/{session_id}/conversation`, following `GetSessionTranscript`'s exact ownership-check shape, with the same connection-scoped defence-in-depth JOIN `GetSessionLines` already uses (a session id from another connection now returns an empty array, proven by `TestGetSessionConversation/session_id_from_another_connection_returns_no_lines`).
- `.logs-transcript`'s own markup is byte-identical to before this plan (`git diff` shows zero removed lines mentioning `logs-transcript`) — the conversation section was added as a true sibling below it via a plain layout-only wrapper `div` with an inline style, so the shared `index.css` (owned by a parallel plan this wave) was never reopened.
- `npm run build` exits 0 after each task; `go build ./...`, `go vet ./...` and `go test ./internal/store/... ./internal/profiles/...` are green except the one pre-existing, unrelated `TestMigration014AddsLoginStartedAtColumn` CRLF/LF artifact of this worktree checkout (documented in 05-06-SUMMARY.md, left untouched per that same standing note). `git diff --stat frontend/package.json frontend/package-lock.json go.mod` is empty throughout.

## Task Commits

1. **Task 05-09-01: The Help page explains the two levels and how to make a suggestion permanent by hand** - `3b0fe2c` (docs)
2. **Task 05-09-02: A past session's conversation can be read on the Logs page, in its own section under the transcript** - `1de6ad3` (feat)

_No plan-metadata commit is made by this executor: per this worktree's instructions, STATE.md/ROADMAP.md are the orchestrator's to update after all wave agents complete._

## Files Created/Modified

- `help/ai-coaching.json` - The Help article: seven sections (the two levels, how a message reaches AI-player, how long it lasts and how to take it back, what chat cannot do, making a suggestion permanent by hand, Pause and Resume, two windows if you want them)
- `frontend/src/pages/HelpPage.tsx` - `'ai-coaching'` added to `SECTION_ORDER`; nothing else changed
- `internal/store/conversation.go` - `ConversationForSession(gameSessionID, connectionID uuid.UUID)`: the SQL now joins `game_sessions` and filters on `connection_id`, matching `GetSessionLines`' defence-in-depth precedent
- `internal/store/conversation_test.go` - New subtest `session_read_is_scoped_to_the_connection_as_defence_in_depth`
- `internal/profiles/handler.go` - `conversationStorage` interface widened with `ConversationForSession(gameSessionID, connectionID uuid.UUID) ([]store.ConversationLine, error)`
- `internal/profiles/logs.go` - `GetSessionConversation`, GET-only, mirroring `GetSessionTranscript`'s ownership-check and response-building shape; `getSessionIDFromPath`'s doc comment notes it also serves the new one-segment-longer route unmodified
- `internal/profiles/coaching_test.go` - `fakeConversationStore.ConversationForSession` (keyed by `[2]uuid.UUID{session, connection}`); `TestGetSessionConversation` (returns stored lines, empty array, not-owned connection refused, session id from another connection returns no lines, no store wired fails closed, method not allowed)
- `cmd/server/main.go` - New route `/api/v1/profiles/{connection_id}/sessions/{session_id}/conversation` wired to `GetSessionConversation`
- `frontend/src/services/api.ts` - `getSessionConversation(connectionId, sessionId)`, following the transcript's own per-session fetch shape, returning `ChatLine[]` (numeric id converted to string)
- `frontend/src/pages/LogsPage.tsx` - `conversation`/`conversationLoading`/`conversationError` state; a fetch-on-select `useEffect` keyed on `[connectionId, selectedSessionId]`; the `.logs-conversation` section rendered as a sibling below `.logs-transcript`, inside a new layout-only wrapper `div` (inline style, no CSS file change)

## Decisions Made

See `key-decisions` in the frontmatter above.

## Document check (D-22, recorded with line numbers)

All four documents already stated the amendment before this plan ran; none required a change.

- `.specify/specs/ai-game-player-design-v3.md` — line 19 (Definition of complete item 4, amended) and line 68 (D5's third accepted-when bullet, amended): "Chat is only ever plain text. There is no promotion... a help article in the app explains AI-chatter, AI-player and how to do this."
- `.planning/REQUIREMENTS.md` — lines 38-39 (REQ-pause-resume, REQ-promote-guidance as amended) and line 63 (REQ-doc-coaching, item 4 amended), plus lines 19, 111 and 137.
- `.planning/ROADMAP.md` — line 27 (Phase 5 one-line description), lines 271-278 (Phase 5 Goal and success criterion 3), lines 318/320 (Phase Validation and implementation notes), line 402 (requirements table row).
- `.planning/PROJECT.md` — line 26 (Active D5 line, amended 2026-09-17).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] The Logs page's per-session conversation read was not yet exposed over HTTP**
- **Found during:** Task 05-09-02, reading `internal/profiles/handler.go` and `internal/store/conversation.go` per the task's own `read_first` instruction
- **Issue:** The connection-scoped, login-bounded `GetConversation`/`ConversationFor` the live panel already uses cannot answer "this one session's conversation" — its `ConversationLineResponse` carries no session id to filter by client-side, so the Logs page had nothing to call. `ConversationForSession(gameSessionID)` existed in the store but was unused, unscoped to a connection, and not wired to any route — exactly the situation the plan's own action text anticipated and pre-authorized a fix for.
- **Fix:** Widened `ConversationForSession` to `(gameSessionID, connectionID uuid.UUID)`, joining `game_sessions` and filtering on `connection_id` in the SQL — the same defence-in-depth shape `GetSessionLines` already uses (T-3-04/T-5-42). Added `GetSessionConversation` to `internal/profiles/logs.go`, mirroring `GetSessionTranscript`'s exact ownership-check and response shape, and registered `GET /api/v1/profiles/{connection_id}/sessions/{session_id}/conversation` in `cmd/server/main.go`. Widened the `conversationStorage` interface and its test fake accordingly.
- **Files modified beyond this plan's declared set:** `internal/store/conversation.go`, `internal/store/conversation_test.go`, `internal/profiles/handler.go`, `internal/profiles/logs.go`, `internal/profiles/coaching_test.go`, `cmd/server/main.go` — listed here per the plan's own explicit instruction to document this addition and explain why it was unavoidable (the alternative, client-side filtering, is not possible without a session id field the server does not send).
- **Verification:** `go build ./...`, `go vet ./...` clean; `TestConversationStoreSQL` and the new `TestGetSessionConversation` (six subtests, including a same-session-different-connection case) all pass.
- **Committed in:** `1de6ad3` (Task 05-09-02)

**2. [Rule 3 - Blocking] `.logs-conversation` could not be placed as a true DOM sibling of `.logs-transcript` without either reopening `index.css` or rewriting `.logs-transcript`'s own markup**
- **Found during:** Task 05-09-02, verifying the acceptance criterion that the conversation section's line number must be greater than `.logs-transcript`'s closing tag ("below, not inside")
- **Issue:** `.logs-layout` is `display: flex` (row) with `.logs-transcript` carrying `flex: 1` directly; `.logs-conversation`'s own CSS (margin-top/padding-top/border-top, no flex or overflow) assumes a vertical stack. Adding `.logs-conversation` as a third direct flex child of the row-flex `.logs-layout` would place it beside, not below, the transcript. The correct fix (a flex-column wrapper around both panes) needs a CSS class this phase's stylesheet does not have, and `index.css` is owned by a parallel plan this wave (machine notes forbid touching it).
- **Fix:** Added a plain wrapper `div` with layout properties set inline (`flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0`) around `.logs-transcript` and the new `.logs-conversation` section, so the conversation section is a genuine sibling below the transcript pane without any `index.css` edit. `.logs-transcript`'s own opening tag, closing tag and every line of its rendering logic are preserved byte-for-byte (confirmed: `git diff` on this file shows zero removed lines mentioning `logs-transcript`, only the two changed import lines and pure additions).
- **Files modified:** `frontend/src/pages/LogsPage.tsx` only; `frontend/src/index.css` untouched (`git diff --stat` empty)
- **Verification:** `npm run build` exits 0; `grep -c "logs-conversation" frontend/src/pages/LogsPage.tsx` returns 5; the conversation section's line number (190) is after `.logs-transcript`'s closing tag (189)
- **Committed in:** `1de6ad3` (Task 05-09-02)

---

**Total deviations:** 2 auto-fixed (both Rule 3 - blocking)
**Impact on plan:** Both were necessary for the plan's own stated acceptance criteria to be reachable at all. No architectural change: the new route follows an existing sibling pattern exactly, and the wrapper div is layout-only with no new dependency or component library.

## Issues Encountered

While recovering from an unrelated tooling misstep during this task (a prohibited `git stash` was run and immediately reverted per the sanctioned recovery procedure — apply by exact SHA, verify, then drop that one entry by SHA, leaving two unrelated pre-existing stash entries from other sessions untouched), no work was lost and no other worktree's state was affected. Recorded here for transparency; it did not change any file this plan touches.

## User Setup Required

None — no external service configuration required. No package was added to `go.mod` or `frontend/package.json` (confirmed empty `git diff --stat` after every task).

## Known Stubs

None. Both the Help article and the Logs page conversation section are wired to real data: the article describes only shipped surfaces (05-06/05-07's coaching channel), and the Logs page's conversation section reads from the real, newly-exposed per-session store method — no placeholder text, no hardcoded empty array feeding real UI.

## Threat Flags

None beyond what the plan's own threat model already registers (T-5-41 through T-5-44, T-5-SC). The new route (`GET .../sessions/{session_id}/conversation`) is covered by T-5-42 (Information Disclosure): it reuses the existing `getProfileByConnectionID` ownership check unchanged and adds a second, independent layer (the SQL-level connection_id JOIN) beyond what T-5-42's own mitigation text describes, proven by `TestGetSessionConversation/session_id_from_another_connection_returns_no_lines`.

## Next Phase Readiness

- Plan 05-11 (security review and close) can cite `TestGetSessionConversation`'s six subtests and `TestConversationStoreSQL`'s new defence-in-depth subtest directly as evidence for T-5-42's mitigation being real, not just declared.
- Evidence capture for `evidence/14-help-article.png` and `evidence/16-logs-conversation-section.png` can be taken directly against this build; the article's exact section titles and the Logs page's exact heading/empty-state strings are locked and verified above.
- No blockers. The one pre-existing, unrelated test failure (`TestMigration014AddsLoginStartedAtColumn`, the CRLF-vs-LF line-ending artifact of this worktree checkout documented in 05-06-SUMMARY.md) was left untouched, as instructed.

---
*Phase: 05-coaching-channel*
*Completed: 2026-09-17*

## Self-Check: PASSED

- FOUND: help/ai-coaching.json
- FOUND: frontend/src/pages/HelpPage.tsx
- FOUND: frontend/src/pages/LogsPage.tsx
- FOUND: frontend/src/services/api.ts
- FOUND: internal/store/conversation.go
- FOUND: internal/store/conversation_test.go
- FOUND: internal/profiles/handler.go
- FOUND: internal/profiles/logs.go
- FOUND: internal/profiles/coaching_test.go
- FOUND: cmd/server/main.go
- FOUND: commit 3b0fe2c (Task 05-09-01)
- FOUND: commit 1de6ad3 (Task 05-09-02)
