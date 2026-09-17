---
phase: 05-coaching-channel
plan: 08
subsystem: ui
tags: [react, typescript, window.open, createPortal, react-dom, ai-assist-panel]

# Dependency graph
requires:
  - phase: 05-coaching-channel
    provides: "05-07's split panel (AI-player top+body, AI-chatter .ai-assist-chat strip), all chat/coaching/pause state living in AIAssistPanel.tsx itself, and the .ai-assist-chat-header/.ai-assist-popout-placeholder CSS classes 05-07 already landed"
provides:
  - "frontend/src/services/popout.ts: openPopout(key, title, width, height) -> Window | null (window.open plus stylesheet/style cloning, returns null on block) and closePopout(win)"
  - "frontend/src/components/AIAssistPanel.tsx: AIPlayerView and AIChatterView extracted as sibling components, each rendered inline or via createPortal into a same-origin child window; one Pop out button per view; Bring back docked placeholders; beforeunload-driven auto-return; unmount/connection-change cleanup that closes any open child window"
affects: [05-09-logs-page-conversation, 05-11-security-review-and-close]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "A browser-window helper (popout.ts) with zero component knowledge: it opens a blank same-origin window, clones this document's <link rel=stylesheet>/<style> nodes into it, and hands back the raw Window handle — the caller owns all further lifecycle (no registry, no postMessage)"
    - "A view extracted from a parent component keeps 100% of its state in the parent and receives everything through props, so the exact same JSX tree can render inline (a child of the parent's own return) or through createPortal(<View {...props} />, childWindow.document.body) without any duplicated subscription, fetch, or local state"
    - "A ref mirrors state that a cleanup closure needs to read at its LATEST value (poppedOutRef mirrors poppedOut) — the unmount/connectionId-change effect closes whatever is popped out NOW, not whatever was popped out when the effect was first created"
    - "Every locked copy string used in two attributes (aria-label and title) is hoisted to a single module-level constant and referenced twice by name, so the literal text itself appears exactly once in the file — this is what makes a one-string-one-place acceptance check pass without duplicating prose"

key-files:
  created:
    - frontend/src/services/popout.ts
  modified:
    - frontend/src/components/AIAssistPanel.tsx

key-decisions:
  - "connectionName (needed for the window title '{View} — {connection name}') is not exposed anywhere in SessionContext or as a prop AIAssistPanel already received — it only takes a connectionId. Fixed by calling the already-exported getConnection(connectionId) once per connection, the same silent-fallback shape every other per-connection load in this component already uses (goal, session memory, coaching, conversation). No change to api.ts or SessionContext.tsx (both off-limits per this worktree's file boundaries) — this is a new call to an existing, already-exported function."
  - "The AI-player view's Pop out button sits in .ai-assist-panel-header as a third flex child (title, Pop out, Minimize) rather than in a new wrapper div, since the header's existing display:flex + justify-content:space-between already spaces three items without any new CSS class or inline style — 05-UI-SPEC.md's own instruction not to reopen index.css unless a class is genuinely missing is honored (none was missing)."
  - "The spec's illustrative `popped` boolean prop was not added to either view's prop interface. Both views already know whether they're docked or popped from a prop that has actual behavioral meaning: AIChatterView's optional onPopOut (undefined when popped, since a popped window has nothing further to pop out to) and, for AIPlayerView, the parent's own ternary (the header's Pop-out button, not the view, is what differs between the two states). Adding an unused `popped` field would have tripped noUnusedLocals under this project's strict tsconfig for no behavioral gain."
  - "The pop-up-blocked notice and every aria-label/title copy string are hoisted to five module-level constants (AI_PLAYER_POPOUT_LABEL, AI_CHATTER_POPOUT_LABEL, AI_PLAYER_BRING_BACK_LABEL, AI_CHATTER_BRING_BACK_LABEL, POPUP_BLOCKED_NOTICE) instead of being written as JSX string literals twice (once for aria-label, once for title) per button, so each locked string's literal text exists in exactly one place in the file."

requirements-completed: [REQ-coaching-chat]

# Metrics
duration: ~15min (commit-to-commit: 14:02:11 to 14:05:54 local, task 1 to task 2; substantially longer reading PROJECT.md, STATE.md, 05-UI-SPEC.md §7, 05-CONTEXT.md, 05-PATTERNS.md, 05-07-SUMMARY.md and the current AIAssistPanel.tsx/index.css before the first commit, plus one npm ci to restore node_modules in this worktree)
completed: 2026-09-17
---

# Phase 5 Plan 8: Pop-out Windows — AI-player and AI-chatter Summary

**Either view of the AI Assist panel can be dragged into its own same-origin browser window via `window.open` plus a React portal — no second websocket, no second sign-in, no new dependency — and returns to the panel by itself when the owner closes that window or clicks Bring back.**

## Performance

- **Duration:** ~15 min commit-to-commit (14:02:11 to 14:05:54 local)
- **Tasks:** 2
- **Files modified:** 2 (1 created, 1 modified)

## Accomplishments

- `frontend/src/services/popout.ts` is the one and only `window.open` in the codebase: it opens a blank, same-origin, named child window (`mudpuppy-ai-player` / `mudpuppy-ai-chatter`), sets its title, zeroes its body margin, sets the app's own background token, and clones every `<link rel="stylesheet">` and `<style>` node from the parent document into it — so the child window renders with the exact same tokens, typography and colour buckets with zero drift and zero second stylesheet. Returns `null` on a browser block; no `postMessage`, no window registry.
- `AIAssistPanel.tsx`'s two views — `AIPlayerView` (goal box, Pause/Resume, status line, Session Memory, Coaching in effect, the thinking stream) and `AIChatterView` (the conversation strip, now with its own `.ai-assist-chat-header` row) — are extracted as sibling components that read everything through props. All state (chat entries, coaching list, decision entries, goal, memory, counts) stays in `AIAssistPanel` itself; popping a view out changes only where it paints (`createPortal` into the child window's `document.body` instead of the panel's own JSX), never what it reads or writes — the exact same `onAI`/`onChat` subscriptions, the exact same `useSession()` call, counted once each.
- One Pop out button per view: AI-player's sits in the existing `.ai-assist-panel-header` beside Minimize (a third flex child, no new CSS); AI-chatter's sits in its own new one-line header row inside its own view. Clicking either calls `openPopout` synchronously inside the `onClick`, per the pop-up-blocker requirement.
- Bringing a view back — by clicking "Bring back" or by the owner closing the child window's own native close button (detected via a `beforeunload` listener on the child) — clears that entry and the view returns to inline rendering with nothing refetched or duplicated. A `useEffect` cleanup backed by a ref that always mirrors the latest `poppedOut` state closes any still-open child window on unmount or `connectionId` change (not a stale snapshot from when the effect was first created).
- A blocked pop-up shows the locked `Your browser blocked the new window. Allow pop-ups for this site and try again.` notice inline in the view that tried, styled with the existing `.form-error` class, and stays docked — no half-moved state.
- `npm run build` exits 0 after every task; `git diff --stat frontend/package.json frontend/src/index.css` is empty throughout; no build output was committed.

## Task Commits

1. **Task 05-08-01: A view can be handed a window of its own, with this page's own styling** - `e92bbd2` (feat)
2. **Task 05-08-02: Each view sits in the panel or in its own window, and comes back on its own when the window closes** - `48311ac` (feat)

_No plan-metadata commit is made by this executor: per this worktree's instructions, STATE.md/ROADMAP.md are the orchestrator's to update after all wave agents complete._

## Files Created/Modified

- `frontend/src/services/popout.ts` (new) - `openPopout(key, title, width, height): Window | null` and `closePopout(win)`; the only `window.open` in the codebase
- `frontend/src/components/AIAssistPanel.tsx` - `AIPlayerView`/`AIChatterView` extracted as sibling components; `poppedOut`/`popupBlocked`/`connectionName` state; `handlePopOut`/`bringBack`; two `beforeunload` effects; one ref-backed unmount/connection-change cleanup effect; five module-level copy constants

## Decisions Made

See `key-decisions` in the frontmatter above.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `connectionName` for the window title has no existing source in this component's props or context**
- **Found during:** Task 05-08-02, while assembling `handlePopOut`'s call to `openPopout` with the contract's `{View} — {connection name}` title
- **Issue:** The UI contract's own code sample uses a bare `connectionName` variable as if it already existed in scope, but `AIAssistPanel` only receives a `connectionId` prop, and `SessionContext.tsx` exposes no connection name field. Without a fix the window title would have no way to include the connection name at all.
- **Fix:** Added a `connectionName` state, loaded via the already-exported `getConnection(connectionId)` (from `frontend/src/services/api.ts`, unmodified) in a `useEffect` keyed on `connectionId`, with the same silent-fallback-on-error shape every other per-connection load in this component already uses (goal, session memory, coaching, conversation).
- **Files modified:** `frontend/src/components/AIAssistPanel.tsx`
- **Verification:** `npm run build` exits 0; the title is assembled as `` `${...} — ${connectionName}` `` exactly as the contract's template literal shows, with `connectionName` now a real, loaded value instead of an undefined identifier
- **Committed in:** `48311ac` (Task 05-08-02 commit)

**2. [Rule 1 - Bug] Two comments produced accidental extra matches on the locked "Pop out" / "Bring back" copy strings**
- **Found during:** Task 05-08-02, running the acceptance criteria's `grep -c "Pop out"` and `grep -c "Bring back"` before committing
- **Issue:** A doc comment above `AIChatterView` referenced "this view's own Pop out button" and another above the `beforeunload` effect referenced clicking `"Bring back"` — both intended as prose, but both are exact substring matches of the locked button-text strings, bringing each count to 3 instead of the required 2 (one occurrence per actual button).
- **Fix:** Reworded both comments ("pop-out control" instead of "Pop out button"; "the docked placeholder's own button below" instead of quoting "Bring back") — no behavior change, counts restored to 2 each.
- **Files modified:** `frontend/src/components/AIAssistPanel.tsx`
- **Verification:** `grep -c "Pop out"` and `grep -c "Bring back"` both return 2; `npm run build` still exits 0
- **Committed in:** `48311ac` (Task 05-08-02 commit)

---

**Total deviations:** 2 auto-fixed (1 blocking, 1 bug)
**Impact on plan:** Both were necessary for the contract's own window-title requirement and the plan's own acceptance criteria to hold. No scope creep: no new dependency, no architecture change, `api.ts`/`index.css`/`package.json` untouched.

## Issues Encountered

This worktree had no `node_modules` at the start of execution; `npm ci` (foreground, per the machine notes) restored it before the first build. `npm run build` regenerates `public/index.html` and `public/assets/*` every time it runs; these were reverted with `git checkout -- public/index.html` and `rm` after every build, per the machine notes' instruction not to commit build output — confirmed via `git status --short` returning clean before each commit.

## User Setup Required

None — no external service configuration required. `git diff --stat frontend/package.json` and `git diff --stat frontend/package-lock.json` are both empty after every task; no dependency was added and no test framework was introduced (per the plan's own objective: `jsdom` cannot exercise real multi-window behaviour, so this capability's proof is a staging screenshot, not a unit test).

## Known Stubs

None. Both views render the exact same live component tree whether docked or popped out — no placeholder data path was introduced. The one intentionally minimal piece is the pop-up-blocked notice, which is locked, static copy by design (05-UI-SPEC.md Copywriting Contract), not a stub.

## Evidence Note (for the plan's own evidence trail)

A blocked pop-up could not be reproduced locally in this execution (no browser session was driven interactively in this worktree); the locked notice string and its `.form-error` (destructive-red) rendering are confirmed present in source and compile cleanly. Per the plan's own objective text, this capability's staging proof is `evidence/10-popouts-both-open.png` (both windows open at once, fed by the same live data) and `evidence/11-popout-brought-back.png` (Bring back / close-by-hand restoring the docked layout with its Phase 4 geometry intact), to be captured in the phase's evidence-gathering plan — not by this executor.

## Next Phase Readiness

- Plan 05-09 (Logs page conversation section) is unaffected: this plan touched only `popout.ts` (new) and `AIAssistPanel.tsx`; `frontend/src/pages/LogsPage.tsx`, `frontend/src/services/api.ts`, `frontend/src/types/index.ts` and `help/ai-coaching.json` were not opened or modified.
- Plan 05-11 (evidence capture / security review) can screenshot `evidence/10-popouts-both-open.png` (both views popped out simultaneously, each showing live decision/chat activity matching the docked panel's own last-known state before it was replaced by placeholders) and `evidence/11-popout-brought-back.png` (Bring back restoring the 380×760 docked geometry unchanged) directly against this build.
- No blockers. The two deviations above are both frontend-only, additive fixes with no backend or cross-plan-boundary changes.

---
*Phase: 05-coaching-channel*
*Completed: 2026-09-17*

## Self-Check: PASSED

- FOUND: frontend/src/services/popout.ts
- FOUND: frontend/src/components/AIAssistPanel.tsx
- FOUND: .planning/phases/05-coaching-channel/05-08-SUMMARY.md
- FOUND: commit e92bbd2 (Task 05-08-01)
- FOUND: commit 48311ac (Task 05-08-02)
