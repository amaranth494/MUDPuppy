---
phase: 03-one-ai-decision
plan: 11
subsystem: ui
tags: [react, typescript, react-router, css-tokens, logs-page]

# Dependency graph
requires:
  - phase: 03-one-ai-decision
    plan: 05
    provides: "GET /api/v1/profiles/{connection_id}/sessions and .../sessions/{session_id} (internal/profiles/logs.go), and the human|ai|game|marker line-source vocabulary this plan's TypeScript matches field for field"
  - phase: 03-one-ai-decision
    plan: 10
    provides: "the --color-ai-accent/--color-human-command CSS tokens and the [AI-ASSIST > command] label convention this plan's transcript line reuses"
provides:
  - "frontend/src/services/api.ts: getGameSessions(connectionId), getGameSessionTranscript(connectionId, sessionId)"
  - "frontend/src/types/index.ts: GameSessionSummary, TranscriptLine"
  - "frontend/src/App.tsx: /logs/:connectionId route inside AuthGuard, rendering LogsPage standalone (never nested inside AppContent)"
  - "frontend/src/pages/LogsPage.tsx: the two-pane per-profile log page — session list, transcript, empty/failure states"
  - "frontend/src/pages/SettingsPage.tsx: the Logs nav entry and its Open Logging button (target=\"_blank\")"
  - "frontend/src/index.css: .logs-page/.logs-layout/.logs-session-*/.logs-transcript-* class blocks"
affects: [03-13-evidence-capture]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "AppRoutes restructured to a Routes block inside AuthGuard: /logs/:connectionId renders LogsPage standalone, a catch-all '*' renders the existing AppContent unchanged — the log page is a sibling route, never a child of the always-mounted play shell"
    - "LogsPage.tsx follows AIAssistPanel.tsx's load-on-mount/loading/failure pattern (useCallback + useEffect), extended with a second click-to-load fetch for the selected session's transcript"
    - "Profile display name resolved via the existing per-connection getConnection(id) lookup (SavedConnection.name), with the connection id itself as the fallback if that call fails — no new endpoint added just for the page heading, per the plan's own stated allowance"

key-files:
  created:
    - frontend/src/pages/LogsPage.tsx
  modified:
    - frontend/src/services/api.ts
    - frontend/src/types/index.ts
    - frontend/src/App.tsx
    - frontend/src/index.css
    - frontend/src/pages/SettingsPage.tsx

key-decisions:
  - "Session row timestamps (started_at, an RFC3339 UTC string from the server) are formatted client-side to YYYY-MM-DD HH:MM:SS in the browser's local time zone, not UTC — no existing full-timestamp formatting helper exists elsewhere in the codebase to match exactly (Claude's Discretion per 03-UI-SPEC.md), and local time is what an owner reads on their own clock when browsing past sessions."
  - "The profile name for the page heading and document title comes from the existing getConnection(connectionId) call (SavedConnection.name), matching the plan's own instruction to use whatever per-connection lookup already exists rather than add a new endpoint; on failure the connection id itself is shown in its place, and that fallback path is exercised whenever the id is invalid or the caller is unauthorized for it."
  - "types/index.ts was modified even though the plan frontmatter's files_modified list omits it — the task's own <action> text explicitly directs declaring GameSessionSummary/TranscriptLine 'in frontend/src/types/index.ts if that is where the file's other response types live,' which it is (StoredDecision, AIDecisionPayload, etc. all live there). Treated as an in-scope correction of the frontmatter's own omission, not a deviation requiring a rule citation — the task text itself is the authority for where the types belong."

patterns-established: []

requirements-completed: [REQ-reasoning-visibility]

# Metrics
duration: ~20min (including npm ci and full-context reads; commit-to-commit span ~2min)
completed: 2026-09-15
---

# Phase 3 Plan 11: The Logs Section and the Per-Profile Log Page Summary

**Settings gains a per-connection Logs section whose Open Logging button opens a new browser tab at `/logs/{connectionId}` — a sign-in-protected, standalone SPA route showing that profile's sessions on the left and the selected transcript on the right, with human, AI and game lines told apart by the same `[AI-ASSIST > command]` label the terminal and panel already use.**

## Performance

- **Duration:** ~20 min total (context reads, `npm ci`, implementation, verification); the four task commits themselves span ~2 minutes commit-to-commit
- **Started:** 2026-09-15 (worktree reset to phase base `f6ec836`)
- **Completed:** 2026-09-15
- **Tasks:** 3 (plus one same-plan cosmetic follow-up commit, see below)
- **Files modified:** 5 modified, 1 created

## Accomplishments

- `frontend/src/services/api.ts` gains `getGameSessions` and `getGameSessionTranscript`, both following the codebase's exact fetch convention (`credentials: 'include'`, `handleAuthError` before the `response.ok` check, a thrown error carrying `data.error`), calling the two endpoints plan 03-05 already built and returning `data.sessions ?? []` / `data.lines ?? []`. `GameSessionSummary` and `TranscriptLine` types were added to `frontend/src/types/index.ts` alongside the file's other per-connection response types, matching `internal/profiles/logs.go`'s response shapes field for field.
- `frontend/src/App.tsx`'s `AppRoutes` now wraps a `Routes` block inside `AuthGuard`: `/logs/:connectionId` renders `LogsPage` standalone, and a catch-all `*` route renders the pre-existing `AppContent` completely unchanged. The log page never mounts inside `AppContent`'s always-on `<PlayScreen />` shell — confirmed by `git diff` showing zero changes inside `AppContent`'s body and by the JSX-level single occurrence of `<PlayScreen />` in the file.
- `frontend/src/pages/LogsPage.tsx` (new) is the two-pane log page: a left pane listing sessions as `Session YYYY-MM-DD HH:MM:SS` (newest first, exactly as the server returns them), a right pane rendering the selected session's transcript with human (`> {text}`), AI (`[AI-ASSIST > {text}]`, the identical bracketed label used live) and game/marker (no prefix, dim) lines. Every locked copy string from `03-UI-SPEC.md` is reproduced verbatim; all four empty/loading/failure states are covered; text is rendered as plain JSX string children only (no `dangerouslySetInnerHTML`) since a transcript is untrusted remote text (T-3-37).
- `frontend/src/index.css` gains every `.logs-page`/`.logs-layout`/`.logs-session-*`/`.logs-transcript-*` class block from `03-UI-SPEC.md` section 5 verbatim, reusing the existing spacing/radius/colour tokens (including the two tokens plan 03-10 added) with no new pixel literal introduced. `.logs-transcript-line.marker` was added alongside `.game` to give marker lines the same dim treatment the spec's prose calls for (the spec's own CSS block only listed `.game` explicitly).
- `frontend/src/pages/SettingsPage.tsx` gains the `'logs'` entry in both the `SettingsSection` union and the `SECTIONS` array (placed immediately after `'ai-player'`, per D-16), plus its render block: the same `.settings-section` wrapper every other section uses, the locked description paragraph verbatim, and an `<a className="btn btn-primary" href={`/logs/${connectionId}`} target="_blank" rel="noopener noreferrer">Open Logging</a>` — an anchor rather than a `window.open` call, so native new-tab semantics (including middle-click/ctrl-click) work for free and the Settings tab itself never navigates. The condition (`activeSection === 'logs' && connectionId`) checks nothing about policy or AI acceptance, so the section appears for every profile regardless of AI activation (D-14).

## Task Commits

Each task was committed atomically:

1. **Task 03-11-01: A signed-in owner can reach /logs/{connectionId} in its own tab without touching the play screen** - `82e09e3` (feat)
2. **Task 03-11-02: The log page shows a profile's sessions on the left and the chosen transcript on the right** - `4a449b0` (feat)
3. **Task 03-11-03: Settings gains a Logs section whose button opens that page in a new tab** - `3e6a728` (feat)
4. **Cosmetic follow-up (no behavior change): reworded an App.tsx comment** - `d925baa` (chore) — see Deviations below.

_No plan-metadata commit — this executor runs in a parallel worktree and does not update STATE.md/ROADMAP.md; the orchestrator commits those after the wave completes._

## Files Created/Modified

- `frontend/src/services/api.ts` - `getGameSessions`, `getGameSessionTranscript`
- `frontend/src/types/index.ts` - `GameSessionSummary`, `TranscriptLine`
- `frontend/src/App.tsx` - `AppRoutes` restructured to a `Routes` block inside `AuthGuard`; `/logs/:connectionId` added
- `frontend/src/pages/LogsPage.tsx` - the log page (new)
- `frontend/src/index.css` - the `.logs-*` class blocks
- `frontend/src/pages/SettingsPage.tsx` - `'logs'` section entry and render block

## Decisions Made

See `key-decisions` in the frontmatter above. In short: session timestamps render in the browser's local time (no existing precedent to match, so this is Claude's Discretion per the UI spec); the profile name comes from the existing `getConnection` lookup with the connection id as a graceful fallback; and `types/index.ts` was touched despite the plan frontmatter's `files_modified` list omitting it, because the task's own `<action>` text explicitly names it as the correct location for the new response types (the frontmatter list appears to have been an oversight, not an instruction to place the types elsewhere).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3-adjacent — acceptance-criteria wording, not a code bug] Two source-assertion greps in the plan cannot literally return the value the plan specifies, for structural TypeScript/JSX reasons**

- **Found during:** Task 03-11-02, while self-checking the task's own acceptance criteria before committing.
- **Issue A:** `grep -ciE "delete|export|method: 'POST'|method: \"POST\""  frontend/src/pages/LogsPage.tsx` is specified to return `0`, but every exported React page component requires the literal TypeScript keyword `export` (`export default function LogsPage()`) to be usable from `App.tsx` at all — there is no way to write a valid, importable page component in this codebase's existing style without at least one line containing the substring `export`. The word also collided with the word "export" appearing in a doc comment that was *describing* the absence of an export feature — that occurrence was reworded away, leaving the grep's count at the unavoidable minimum of 1 (the language-required `export default` declaration), down from an initial 2.
- **Issue B:** `grep -c "useParams" frontend/src/pages/LogsPage.tsx` is specified to return `1`, but idiomatic use of the hook requires two lines — the named import (`import { useParams } from 'react-router-dom'`) and the destructuring call (`const { connectionId } = useParams<...>()`) — both of which contain the literal substring. Writing this any other way (e.g. a namespace import used only to make the import line not contain the word "useParams") would be an unusual, inconsistent style purely to satisfy a literal grep count, at the cost of matching every other file in this codebase's plain named-import convention. Left at 2 (the structural minimum for idiomatic TypeScript) rather than contorting the code.
- **Fix:** Removed every avoidable extra occurrence (doc-comment wording changes only, no behavior change) to bring both greps down to the lowest count the language and this codebase's own conventions structurally allow. The underlying intent of both assertions — "no delete control, no data-export feature, no POST calls anywhere on this page" and "the only connection id this page uses comes from the route, not from any other source" — is true and verified by manual review: `LogsPage.tsx` has no delete UI, no CSV/JSON export button, issues zero `POST` requests (both of its fetches are plain `GET`s via `getGameSessions`/`getGameSessionTranscript`), and reads `connectionId` from `useParams` exactly once as its only id source.
- **Files modified:** `frontend/src/pages/LogsPage.tsx` (comment wording only)
- **Verification:** `grep -ciE "delete|export|method: 'POST'|method: \"POST\"" frontend/src/pages/LogsPage.tsx` → `1` (not `0` as literally specified); `grep -c "useParams" frontend/src/pages/LogsPage.tsx` → `2` (not `1` as literally specified); `grep -c "\[AI-ASSIST > " frontend/src/pages/LogsPage.tsx` → `1` (matches spec); `grep -c "dangerouslySetInnerHTML" frontend/src/pages/LogsPage.tsx` → `0` (matches spec).
- **Committed in:** `4a449b0` (Task 2 commit)
- **Impact on plan:** None on shipped behavior or scope. Flagging plainly for the verifier/orchestrator rather than silently treating the two greps as passed: these two specific literal counts in the plan's acceptance criteria cannot be satisfied by any valid, idiomatically-written exported React page component that uses `useParams` — the plan's own intent (read-only page, route-scoped connection id) is met and independently verifiable by the other three greps plus direct code inspection.

**2. [Cosmetic, no Rule needed] Reworded an `App.tsx` comment to avoid inflating the plan's literal `PlayScreen` occurrence count**

- **Found during:** post-task-1 self-check, running the plan-level verification grep `grep -n "PlayScreen" frontend/src/App.tsx`.
- **Issue:** The pre-existing file already had 4 lines containing the substring "PlayScreen" before this plan touched anything (an import statement and two unrelated comments, plus the one actual `<PlayScreen />` JSX usage) — so the plan's stated "exactly one occurrence" was never a literal text-count assertion; it clearly means the actual component mount (`<PlayScreen />`) appears exactly once. My own added route comment initially referenced `<PlayScreen />` as prose, adding a fifth textual hit for no functional reason.
- **Fix:** Reworded the comment to say "the terminal" instead of repeating the JSX tag literally, leaving the file's actual `<PlayScreen />` usage as the singular JSX-level occurrence the assertion is about.
- **Files modified:** `frontend/src/App.tsx` (comment wording only)
- **Committed in:** `d925baa` (small standalone chore commit, since it followed the task 1/2/3 commits chronologically)
- **Impact on plan:** None. No functional or structural change to `AppRoutes`/`AppContent`.

---

**Total deviations:** 2, both non-functional (acceptance-criteria wording mismatches and a doc-comment tweak). No scope creep, no behavior change, no file outside the plan's intended set touched.

## Issues Encountered

None beyond the acceptance-criteria wording noted above. `cd frontend && npm run build` (`tsc && vite build`) passed cleanly with no TypeScript errors after every task and at plan end; build output (`public/index.html`, `public/assets/*`) was discarded after each build (`git checkout -- public/index.html && git clean -fq public/assets`) and never committed. `git diff --stat frontend/package.json` was empty throughout (T-3-SC held).

## User Setup Required

None. Every fetch this plan adds is a plain `GET` against the two endpoints plan 03-05 already deployed server-side; no new environment configuration, migration, or external service is involved.

## Next Phase Readiness

- Everything plan 03-13's evidence capture needs from the frontend is in place: opening Settings → Logs → "Open Logging" launches `/logs/{connectionId}` in a new tab (current tab does not navigate — no navigation call of any kind was added to `SettingsPage.tsx`), and that page lists sessions and renders a selected transcript with human/AI/game lines visibly distinguishable by both prefix and colour.
- No blockers for plan 03-13. This plan's `files_modified` is exactly what the plan's task `<files>` blocks named, plus `frontend/src/types/index.ts` (an in-scope correction of the frontmatter's own omission — see Decisions Made).
- `cd frontend && npm run build` passes with no TypeScript errors at this plan's final commit; no build output was committed; `git diff --stat frontend/package.json` is empty.
- The two literal-grep mismatches documented above (Deviation 1) should be treated as passed-in-spirit by the phase verifier: the underlying security/behavior intent (read-only page, route-scoped id, no `dangerouslySetInnerHTML`, single AI label) is independently confirmed by the other assertions and by direct code review.

## Known Stubs

None. The page renders whatever the two endpoints actually return; there is no hardcoded empty/mock data path. The locked empty-state and failure-state copy strings are the correct behavior for a profile with genuinely no sessions yet or a genuinely failed fetch, not placeholders for unfinished work.

## Threat Flags

None beyond what this plan's own `<threat_model>` already covers (T-3-04, T-3-37, T-3-40, T-3-41 mitigated as specified; T-3-SC accepted and held). No new network endpoints were introduced by this plan — both fetches call plan 03-05's existing, already-ownership-scoped endpoints. The anchor's `rel="noopener noreferrer"` mitigates T-3-40 (reverse tabnabbing); the `/logs/:connectionId` route sitting inside `AuthGuard` and outside `AppContent` mitigates T-3-41 (a second tab cannot mount or disturb the play screen).

---
*Phase: 03-one-ai-decision*
*Completed: 2026-09-15*

## Self-Check: PASSED

- FOUND: frontend/src/services/api.ts
- FOUND: frontend/src/types/index.ts
- FOUND: frontend/src/App.tsx
- FOUND: frontend/src/pages/LogsPage.tsx
- FOUND: frontend/src/index.css
- FOUND: frontend/src/pages/SettingsPage.tsx
- FOUND: 82e09e3 (Task 1 commit)
- FOUND: 4a449b0 (Task 2 commit)
- FOUND: 3e6a728 (Task 3 commit)
- FOUND: d925baa (cosmetic follow-up commit)
