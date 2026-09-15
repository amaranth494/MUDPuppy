---
phase: 01-profile-foundation-and-policy-gate
plan: 04
subsystem: ui
tags: [react, typescript, vite, settings-page, policy-gate]

# Dependency graph
requires:
  - phase: 01-profile-foundation-and-policy-gate (plan 03)
    provides: "Five HTTP endpoints under /api/v1/profiles/{connection_id}/{ai-settings,policy,policy/accept,engage-gate}"
provides:
  - "frontend/src/types/index.ts AISettings, AISettingsResponse, PolicyResponse, EngageGateResponse interfaces"
  - "frontend/src/services/api.ts getAISettings, putAISettings, getPolicy, acceptPolicy, getEngageGate client functions"
  - "frontend/src/components/AIPlayerPanel.tsx — the two-state (policy gate, then editor) AI Player section"
  - "AI Player nav entry in SettingsPage.tsx, delegated to AIPlayerPanel"
affects: [01-05, 01-06]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "AIPlayerPanel.tsx follows EnvironmentPanel.tsx's load/save/error state shape (isLoading/isSaving/error/successMessage, useCallback loader + useEffect, try/catch/finally save) — the second standalone (non-inlined) sub-resource panel component in the codebase"
    - "Two-phase load: getPolicy() always runs first; getAISettings() only runs when the returned accepted flag is true, so no AI settings network call happens for an unaccepted profile"
    - "Numeric fields (call_cap, disengage_threshold) bind to string state in the browser so an empty input round-trips as JSON null rather than being coerced to a number client-side — blank stays a first-class value all the way to the server"
    - ".ai-player-accepted-line::first-letter CSS colors only the leading checkmark character green, keeping the rest of the accepted-status line dim, while keeping the raw JSX text '✓ Accepted v{version}...' as one contiguous literal (no span boundary between the checkmark and the following text)"

key-files:
  created:
    - frontend/src/components/AIPlayerPanel.tsx
  modified:
    - frontend/src/types/index.ts
    - frontend/src/services/api.ts
    - frontend/src/pages/SettingsPage.tsx
    - frontend/src/index.css

key-decisions:
  - "AIPlayerPanel is a standalone component (like EnvironmentPanel.tsx) rather than inlined into SettingsPage.tsx, per the plan's explicit recommendation — the two-state UI (policy gate vs. five-field editor) would have added ~150 lines to an already-large page file"
  - "Policy panel scrollbox uses a new .policy-panel CSS class (background/border/radius identical to .contextual-help, plus max-height: calc(100vh - 380px) and overflow-y: auto matching .keybindings-list) rather than reusing .contextual-help directly, since .contextual-help has no built-in scroll behavior and the UI-SPEC requires the heading, version line, and Accept button to stay visible together without page scroll"
  - "The accepted-status checkmark is colored via a CSS ::first-letter pseudo-element (.ai-player-accepted-line) instead of wrapping the ✓ in its own <span>, so the raw source text stays a contiguous '✓ Accepted v{version}...' literal (satisfying the plan's source-grep acceptance criterion) while still only accent-coloring the checkmark per the UI-SPEC Layout section"

patterns-established:
  - "Frontend AI Player sub-resource client functions in api.ts follow the getTimers/putTimers fetch-wrapper shape exactly: credentials: 'include', handleAuthError(response) before the response.ok check, throw new Error(data.error || '<UI-SPEC fallback string>') on failure"

requirements-completed: [REQ-profile-ai-fields, REQ-policy-gate]

# Metrics
duration: ~35min
completed: 2026-09-15
---

# Phase 1 Plan 4: AI Player Settings Panel (Policy Gate + Editor) Summary

**A new AI Player section in Settings shows the Safety and Abuse policy and a single Accept button on an unaccepted profile, then reveals a five-field AI settings editor (conduct rules, approach guidance, model name, call cap, disengage threshold) after one press, with blank numeric fields round-tripping as JSON null through `frontend/src/components/AIPlayerPanel.tsx`.**

## Performance

- **Duration:** ~35 min
- **Tasks:** 3 completed
- **Files modified:** 4 (1 new component, 3 modified: types, api client, SettingsPage, plus index.css for two new CSS rules)

## Accomplishments

- `frontend/src/types/index.ts` gained `AISettings`, `AISettingsResponse`, `PolicyResponse`, `EngageGateResponse` — sub-resource-only types, never added to `Profile`/`UpdateProfileRequest`, with no reconnect member (D-13).
- `frontend/src/services/api.ts` gained `getAISettings`, `putAISettings`, `getPolicy`, `acceptPolicy`, `getEngageGate`, each following the `getTimers`/`putTimers` fetch-wrapper shape with the UI-SPEC's exact fallback error strings.
- `frontend/src/components/AIPlayerPanel.tsx` (new) renders the policy gate first (heading, version line, full policy text via a re-themed `renderContent`/`renderInline` pair, single Accept Policy button — no AI settings field rendered at all in this state) and, once `accepted` is true, the acceptance status line plus the five-field editor, all inside a single `.settings-section` wrapper matching every other Settings section.
- `frontend/src/pages/SettingsPage.tsx` gained the `ai-player` section (nav entry positioned after Environment, per D-01) delegated entirely to `AIPlayerPanel`; the diff is 9 lines, confirming the section is delegated, not inlined.
- `frontend/src/index.css` gained two small, scoped rules: `.policy-panel` (scrollable bordered box, identical properties to `.contextual-help` plus the `.keybindings-list` scroll pattern) and `.ai-player-accepted-line::first-letter` (accent-colors only the checkmark character).

## Task Commits

Each task was committed atomically:

1. **Task 01-04-01: Types and API client functions for the AI sub-resource, policy and gate** - `cb9036e` (feat)
2. **Task 01-04-02: AIPlayerPanel presents the policy first and the editor only after acceptance** - `04ee8d6` (feat)
3. **Task 01-04-03: The AI Player section appears in Settings for the selected connection** - `d11b21e` (feat)

## Files Created/Modified

- `frontend/src/types/index.ts` - `AISettings`, `AISettingsResponse`, `PolicyResponse`, `EngageGateResponse` interfaces
- `frontend/src/services/api.ts` - `getAISettings`, `putAISettings`, `getPolicy`, `acceptPolicy`, `getEngageGate`
- `frontend/src/components/AIPlayerPanel.tsx` (new) - two-state AI Player section component
- `frontend/src/pages/SettingsPage.tsx` - `SettingsSection` union + `SECTIONS` entry for `ai-player`; render branch delegating to `AIPlayerPanel`
- `frontend/src/index.css` - `.policy-panel`, `.ai-player-accepted-line::first-letter`

## Decisions Made

- Standalone `AIPlayerPanel.tsx` component (not inlined in `SettingsPage.tsx`), per the plan's explicit structural recommendation, matching `EnvironmentPanel.tsx` as the closest analog.
- `.policy-panel` is a new, minimal CSS class rather than reusing `.contextual-help` verbatim, because the UI-SPEC requires the policy panel's own internal scroll (heading/version/button must stay visible together without scrolling the page) and `.contextual-help` has no scroll behavior defined.
- The accepted-status checkmark's accent color is applied via `::first-letter` rather than a wrapping `<span>`, keeping the literal source text `✓ Accepted v{version}...` contiguous (required by the plan's source-grep acceptance check) while still satisfying the UI-SPEC's "checkmark only" coloring rule.

## Deviations from Plan

None - plan executed exactly as written. All source assertions in the plan's `<acceptance_criteria>` blocks (required literals, hint strings verbatim, mutually-exclusive branches, zero reconnect/hex-color/dangerouslySetInnerHTML/console.log matches, `git diff --stat` line count for `SettingsPage.tsx`) were verified to hold post-implementation.

## Issues Encountered

None. `cd frontend && npm run build` (tsc + vite build) passed clean after every task. Build output files under `public/assets/*` and the timestamp in `public/index.html` were regenerated by each `npm run build` verification run and reverted/removed before each commit, per this plan's instruction not to commit built output.

## User Setup Required

None - no external service configuration required. This plan adds no new npm dependency (`frontend/package.json` untouched, confirmed via `git diff --stat`).

## Next Phase Readiness

- Plan 01-05 (staging evidence capture) can now screenshot every named state from a live browser: `evidence/05-policy-first.png` (policy gate, unaccepted profile), `evidence/06-accepted-line.png` (accepted line + editor, no reload), `evidence/07-values-after-reload.png` / `evidence/08-values-new-session.png` (saved values persisting), `evidence/09-blank-fields.png` (cleared numeric fields round-tripping as blank).
- Plan 01-06 (canned report script) is unaffected by this plan — it drives the backend endpoints directly and does not depend on the frontend.
- No blockers.

---
*Phase: 01-profile-foundation-and-policy-gate*
*Completed: 2026-09-15*

## Self-Check: PASSED

All created/modified files verified present (frontend/src/types/index.ts, frontend/src/services/api.ts, frontend/src/components/AIPlayerPanel.tsx, frontend/src/pages/SettingsPage.tsx, frontend/src/index.css). All three task commit hashes (cb9036e, 04ee8d6, d11b21e) verified present in git log.
