---
phase: 03-one-ai-decision
plan: 10
subsystem: ui
tags: [react, typescript, websocket, css-tokens, ai-assist-panel]

# Dependency graph
requires:
  - phase: 03-one-ai-decision
    plan: 09
    provides: "MsgTypeAI/AIDecisionPayload on the Go WSMessage, GET /api/v1/profiles/{connection_id}/decisions, and the wire contract this plan's TypeScript matches field for field"
provides:
  - "frontend/src/types/index.ts: AIDecisionPayload, StoredDecision, decision? on WSMessage, 'ai' added to WSMessageType"
  - "frontend/src/services/api.ts: case 'ai' in handleMessage's switch, onAI/offAI handler pair, getDecisions(connectionId, limit?)"
  - "frontend/src/index.css: --color-ai-accent and --color-human-command tokens, every ai-assist-panel/ai-assist-tab/ai-decision/ai-system-line class block from 03-UI-SPEC.md section 1"
  - "frontend/src/components/AIAssistPanel.tsx: the floating, minimizable AI Assist panel — decision cards, system lines, empty/loading states, REST reload on mount, live updates via onAI"
  - "frontend/src/pages/PlayScreen.tsx: the panel's gated mount (D-07) and the [AI-ASSIST > command] terminal echo (D-09) plus the D-13 failure/refusal terminal line"
affects: [03-13-evidence-capture]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "PanelEntry union (DecisionEntry | SystemEntry) inside AIAssistPanel.tsx: reload rows are always decision cards (a failed/refused row's stored notice fills the command slot so the existing outcome-failed/outcome-refused CSS override applies), while live kind==='system' pushes render as a separate .ai-system-line — the two visual treatments 03-UI-SPEC.md's reload paragraph and markup sketch each specify"
    - "AutopilotBadge-style 'server owns truth, derive nothing': the panel's minimized-tab dot reads autopilotState straight from SessionContext, no local poll"
    - "Per-connection load-on-mount (AIPlayerPanel.tsx's pattern): PlayScreen loads the policy-acceptance signal once per currentConnectionId change to gate the panel's mount"

key-files:
  created:
    - frontend/src/components/AIAssistPanel.tsx
  modified:
    - frontend/src/types/index.ts
    - frontend/src/services/api.ts
    - frontend/src/index.css
    - frontend/src/pages/PlayScreen.tsx

key-decisions:
  - "The panel's live 'system line' entries are scoped to exactly what the 'ai' websocket message can carry per plan 03-09's wire contract: AIDecisionPayload's outcome is 'sent'|'refused'|'failed' only, and internal/driver/driver.go's recordFailure is the only caller that ever sends Kind: 'system' (a D-13 failure/refusal notice with 'Autopilot disengaged' already worded into its text). The general engaged/waiting/resuming/disengaged switch-position notices D-08 also lists remain terminal-only, delivered by the pre-existing 'autopilot' websocket message and PlayScreen's own previousAutopilotStateRef transition effect (untouched by this plan) — there is no wire path for those transitions to reach the panel in this phase's shipped backend, so wiring the panel to expect them would be inventing a message the driver never sends. The panel's own state-engaged/state-waiting/state-resuming/state-disengaged CSS classes from 03-UI-SPEC.md are added for completeness and forward compatibility but are only reachable today via state-refused/state-failed on the two outcomes the wire contract actually delivers."
  - "Both the panel's terminal echo and its own system-line rendering bracket-wrap the raw payload.message ([${payload.message}]) rather than relying on the driver to send pre-bracketed text — the driver's stored notice strings (e.g. 'AI decision failed: the model could not be reached. Autopilot disengaged.') are deliberately plain per 03-09's own code, matching the terminal's existing bracketed-local-line convention (echoLocal callers always supply their own brackets) and 03-UI-SPEC.md's Copywriting Contract table, which lists the bracketed form as the locked, verbatim string for both surfaces."
  - "The AI Assist panel's outer mount gate in PlayScreen.tsx checks currentConnectionId, connectionState === 'connected', and a locally-loaded aiActivated flag (from getPolicy(connectionId).accepted, loaded once per connection exactly like AIPlayerPanel.tsx) — matching the task's literal instruction ('a connected saved-profile connection... and that profile has the AI activated') rather than gating on policy acceptance alone, which would show the panel on a still-connecting or disconnected saved-profile tab."

patterns-established: []

requirements-completed: [REQ-reasoning-visibility]

# Metrics
duration: ~12min
completed: 2026-09-15
---

# Phase 3 Plan 10: The AI Assist Panel and the Terminal's AI-Issued Command Line Summary

**A floating, minimizable AI Assist panel now renders every AI decision as reasoning-then-command, rebuilds itself from the server after a refresh, and the terminal prints an AI-issued command as `[AI-ASSIST > command]` in a reserved magenta so it can never be mistaken for something the owner typed.**

## Performance

- **Duration:** ~12 min (commit-to-commit, base `b309e04` to final task commit `f31ea11`)
- **Started:** 2026-09-15T20:03:19-07:00
- **Completed:** 2026-09-15T20:08:33-07:00
- **Tasks:** 3
- **Files modified:** 5 (1 created, 4 modified)

## Accomplishments

- `frontend/src/types/index.ts` and `frontend/src/services/api.ts` carry the `ai` websocket message end to end: `AIDecisionPayload` matches `internal/session/websocket.go`'s struct field for field (no `window_text`), `case 'ai'` dispatches to a new `onAI`/`offAI` pair mirroring `onAutopilot`/`offAutopilot` exactly, and `getDecisions` follows the codebase's `credentials: 'include'` + `handleAuthError` fetch convention.
- `frontend/src/index.css` gains the two locked colour tokens (`--color-ai-accent` #ff66ff, `--color-human-command` #00ffff) and every `.ai-assist-panel`/`.ai-assist-tab`/`.ai-decision`/`.ai-system-line` class block verbatim from `03-UI-SPEC.md`.
- `frontend/src/components/AIAssistPanel.tsx` renders each decision as reasoning then a bold `→ {command}` line, reloads the connection's stored history once on mount (failed/refused rows show their stored notice in place of a command), stays live via `onAI`/`offAI`, autoscrolls to the newest entry, and contains no input element of any kind (D-08) — proven by `grep -cE "<input|<textarea|type=\"submit\"" frontend/src/components/AIAssistPanel.tsx` returning 0.
- `frontend/src/pages/PlayScreen.tsx` mounts the panel as a `position: fixed` sibling of `.output-panel-full` (the terminal keeps its full width by construction), gated on a connected saved-profile connection plus its policy-acceptance signal, and writes `[AI-ASSIST > {command}]` in `brightmagenta` for a sent decision and the D-13 failure/refusal notice as a bracketed red line — with zero changes to `submitCommand`, the wheel-grab, the autopilot transition effect, or the badge.

## Task Commits

Each task was committed atomically:

1. **Task 03-10-01: The browser can hear an AI message and ask for the decisions it missed** - `b3ae07d` (feat)
2. **Task 03-10-02: The AI Assist panel exists, shows a decision, and can be collapsed to a tab** - `00aa6c3` (feat)
3. **Task 03-10-03: The panel appears for AI-activated profiles only, and the AI's command is unmistakable in the terminal** - `f31ea11` (feat)

_No plan-metadata commit — this executor runs in a parallel worktree and does not update STATE.md/ROADMAP.md; the orchestrator commits those after the wave completes._

## Files Created/Modified

- `frontend/src/types/index.ts` - `AIDecisionPayload`, `StoredDecision`, `decision?` on `WSMessage`, `'ai'` added to `WSMessageType`
- `frontend/src/services/api.ts` - `aiHandlers`, `case 'ai':`, `onAI`/`offAI`, `getDecisions`
- `frontend/src/index.css` - two new `:root` tokens, the panel/tab/decision/system-line class blocks
- `frontend/src/components/AIAssistPanel.tsx` - the panel component (new)
- `frontend/src/pages/PlayScreen.tsx` - gated mount, policy-acceptance load-on-connection-change, `onAI` terminal-echo subscription

## Decisions Made

See `key-decisions` in the frontmatter above for full rationale. In short: the panel's live system-line rendering is scoped to exactly the failure/refusal notices the `ai` websocket message can actually carry per plan 03-09's shipped wire contract (no engaged/waiting/resuming/disengaged path exists into this message type in this phase's backend, so those remain terminal-only via the pre-existing `autopilot` message and transition effect); both the terminal and the panel bracket-wrap the driver's plain `message` text client-side so the two surfaces show the identical bracketed string; and the panel's outer mount gate in `PlayScreen.tsx` checks connection state, connection id, and policy acceptance together rather than acceptance alone.

## Deviations from Plan

None - plan executed exactly as written. The one interpretive judgment call (scoping the panel's live system lines to the failure/refusal outcomes the wire contract actually delivers) is documented above as a decision, not a deviation — it does not change any file the plan didn't already name, add scope, or contradict any acceptance criterion; every criterion in all three tasks passes as specified.

## Issues Encountered

None. `cd frontend && npm run build` (`tsc && vite build`) passed cleanly after every task with no TypeScript errors; build output (`public/index.html`, `public/assets/*`) was discarded after each build per the tooling notes and never committed. `git diff --stat frontend/package.json` was empty at every checkpoint (T-3-SC held).

## User Setup Required

None new. The Gemini environment variables plan 03-02 introduced still need to be set on Railway staging before `#AUTO ON` can reach a real model and produce a real decision for `evidence/05-decision-in-panel.png`; this plan's panel and terminal line are otherwise ready to render whatever the driver produces the moment it exists.

## Next Phase Readiness

- Everything plan 03-13's evidence capture needs from the frontend is in place: the panel opens automatically for an AI-activated, connected profile, shows nothing for a profile that never accepted the policy (Regression Invariant), reloads its history after a refresh, and the terminal's `[AI-ASSIST > command]` line is unmistakable from a typed command.
- No blockers for plan 03-13. This plan's `files_modified` list is exactly what the plan named (`frontend/src/types/index.ts`, `frontend/src/services/api.ts`, `frontend/src/components/AIAssistPanel.tsx`, `frontend/src/index.css`, `frontend/src/pages/PlayScreen.tsx`), with no overlap declared against plan 03-12's `scripts/verify-phase3.sh`/`scripts/fixtures/` work running in parallel.
- `cd frontend && npm run build` passes with no TypeScript errors at this plan's final commit; no build output was committed.

## Known Stubs

None. The panel renders whatever the server actually sends (live push or reload); there is no hardcoded empty/mock data path and no placeholder copy beyond 03-UI-SPEC.md's own locked empty/loading strings, which are the correct behavior when a connection genuinely has no decisions yet.

## Threat Flags

None beyond what this plan's own `<threat_model>` already covers (T-3-37, T-3-38, T-3-11, T-3-39 mitigated as specified; T-3-SC accepted and held — `git diff --stat frontend/package.json` confirmed empty throughout). No new network endpoints, auth paths, or schema changes were introduced outside that register: the panel renders model-authored text as JSX children only (no `dangerouslySetInnerHTML`, no `innerHTML`), and the mount/mismatch gate is read-only against the existing policy-acceptance endpoint.

---
*Phase: 03-one-ai-decision*
*Completed: 2026-09-15*

## Self-Check: PASSED

- FOUND: frontend/src/types/index.ts
- FOUND: frontend/src/services/api.ts
- FOUND: frontend/src/index.css
- FOUND: frontend/src/components/AIAssistPanel.tsx
- FOUND: frontend/src/pages/PlayScreen.tsx
- FOUND: b3ae07d (Task 1 commit)
- FOUND: 00aa6c3 (Task 2 commit)
- FOUND: f31ea11 (Task 3 commit)
