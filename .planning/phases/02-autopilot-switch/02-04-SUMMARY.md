---
phase: 02-autopilot-switch
plan: 04
subsystem: automation
tags: [typescript, react, frontend, automation, directive-grammar, websocket, ai-player]

# Dependency graph
requires:
  - phase: 02-autopilot-switch (plan 02-02)
    provides: "POST /api/v1/session/autopilot (on/off/status) and the AutopilotRequest/AutopilotResponse wire contract; GET /api/v1/session/status carrying autopilot_state"
  - phase: 02-autopilot-switch (plan 02-03)
    provides: "WSMessage.Source (inbound data messages) and MsgTypeAutopilot on the websocket wire; IsHumanSource's allowlist (trigger/timer are automation, everything else human)"
provides:
  - "#AUTO / #AUTO ON / #AUTO OFF / #AUTO STATUS in the frontend directive grammar, calling the server and printing the owner's exact words per 02-UI-SPEC.md"
  - "CommandSource threaded from ProcessedCommand through setSubmitCommandCallback into the outbound websocket data message's source field (the field plan 02-03's server-side wheel-grab reads)"
  - "AutopilotAnswer type (state/outcome/gate_allowed/gate_message/policy_version) shared between evaluator.ts's autopilotControl and api.ts's setAutopilot()"
  - "onAutopilot/offAutopilot websocket handler pair and SessionStatus.autopilot_state for plan 02-06's badge wiring"
affects: [02-06-wheel-grab-wiring]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Trailing-optional-parameter threading (autopilotControl added to ExecutionContext/executeTokens/executeAutomationAction one position after source), mirroring the existing helpResolver precedent exactly"
    - "Directive-grammar two-file edit (CommandRegistry entry + executeTokenList switch case), the same shape already proven four times by ECHO/LOG/HELP/TIMER"

key-files:
  created: []
  modified:
    - frontend/src/services/automation/commands.ts
    - frontend/src/services/automation/evaluator.ts
    - frontend/src/services/automation.ts
    - frontend/src/services/api.ts
    - frontend/src/types/index.ts
    - public/index.html
    - public/assets/index-JQMLws06.js (bundle, generated)
    - public/assets/index-JQMLws06.js.map (bundle, generated)

key-decisions:
  - "D-08's second half (a typed # directive's own emitted commands are labelled automation) is implemented as a single check on the raw processUserInput input (trimmed-left startsWith '#'), applied only to the branch that already emits commands from an internal-command result — not a per-character or per-token classification, since the internal-command branch is the only place a # line's own executeAutomationAction result can produce commands"
  - "The discretionary unknown-argument line ([Autopilot: use #AUTO ON, #AUTO OFF, or #AUTO STATUS]) is not part of the UI contract; it exists because the grammar accepts any #AUTO argument and needs an answer for the case none of ON/OFF/STATUS/blank match"

requirements-completed: [REQ-autopilot-directives, REQ-wheel-grab]

# Metrics
duration: ~25min
completed: 2026-09-15
---

# Phase 2 Plan 4: Autopilot Directive Grammar and Source Threading Summary

**`#AUTO ON/OFF/STATUS` (and bare `#AUTO`) joins the frontend's `#` directive grammar exactly like `#HELP`, calling the server via a new `autopilotControl` callback and rendering every outcome verbatim from `02-UI-SPEC.md`; the `CommandSource` the engine already assigned at every origination point now survives the one call site that was dropping it (`processCommandQueue`) all the way to the outbound websocket `data` message's new `source` field.**

## Performance

- **Duration:** ~25 min
- **Completed:** 2026-09-15
- **Tasks:** 2 completed
- **Files modified:** 5 source files (2 tasks) + 3 generated bundle files (1 build commit)

## Accomplishments
- `frontend/src/services/automation/commands.ts`: `'AUTO'` entry in `CommandRegistry` (`requiresArgs: false`, `category: CommandCategories.OUTPUT`), matching `'HELP'`'s exact shape.
- `frontend/src/services/automation/evaluator.ts`: new exported `AutopilotAnswer` type; `autopilotControl?: { setState }` added to `ExecutionContext` and threaded as a new trailing optional parameter through `executeTokens`, `executeAutomationAction`, and `executeWithTimeout` (mirrors the `helpResolver` precedent exactly). New `case 'AUTO':` in the same switch as `case 'HELP':` — parses the argument (empty/`STATUS`→status, `ON`→on, `OFF`→off, anything else→a discretionary usage line), refuses with `[Autopilot needs a connected game; connect first]` when no `autopilotControl` is wired yet, otherwise awaits the callback and renders `engaged`/`disengaged`/`already-on`/`already-off`/`refused-gate`/`refused-no-session`/`status` strictly from the outcome — every bracketed string character-for-character from `02-UI-SPEC.md`'s Copywriting Contract, the `refused-gate` branch rendering the server's `gate_message` with no hardcoded copy of Phase 1's refusal sentence. Side-effect only (advances the token index like every other output-only case) so `#AUTO` never emits a game command, closing D-08 by construction.
- `frontend/src/services/automation.ts`: `setAutopilotControl()` stores the callback and passes it as the new trailing argument at both `executeAutomationAction` call sites in `processUserInput` that already pass `'cli'`. D-08's second half: a new `isDirectiveInput` check (raw input, trimmed-left, starts with `#`) labels commands the internal-command branch emits as source `'trigger'` instead of `'user'` when the line itself was a typed directive, so a directive's own emitted commands can never take the wheel back from themselves. Separately (Task 2): `onSubmitCommand`/`setSubmitCommandCallback` widened to `(command, source) => void`; `processCommandQueue`'s single call site now passes `cmd.source` through instead of dropping it.
- `frontend/src/types/index.ts`: `SessionStatus.autopilot_state?: 'on' | 'waiting' | 'off'` (D-10's refresh-correctness field); `WSMessageType` gains `'autopilot'`; `WSMessage` gains outbound `source?: string`.
- `frontend/src/services/api.ts`: `sendCommand(command, source?)` includes `source` in the outbound `data` message payload (omitted key when `source` is `undefined`, so existing callers are byte-identical); `WebSocketManager` gains `case 'autopilot':` dispatching to a new `autopilotHandlers` array via `onAutopilot`/`offAutopilot`, mirroring `onStatus`/`offStatus`; new `setAutopilot(connectionId, action)` POSTs to `/api/v1/session/autopilot` following the established fetch-wrapper shape (`credentials: 'include'`, `handleAuthError` before the `response.ok` check), returning the shared `AutopilotAnswer` type.
- Public bundle rebuilt and committed (`public/index.html`, `public/assets/index-JQMLws06.js[.map]`) so the running server serves the new directive and source-threading logic, matching Phase 1's `build(...)` commit precedent.

## Task Commits

Each task was committed atomically:

1. **Task 02-04-01: #AUTO joins the directive grammar and prints the owner's words for every answer** - `9e15007` (feat)
2. **Task 02-04-02: Every command the browser sends says whether a person typed it** - `19e3839` (feat)
3. **Public bundle rebuild** - `695ddb5` (build)

**Plan metadata:** (this commit)

## Files Created/Modified
- `frontend/src/services/automation/commands.ts` - `'AUTO'` entry in `CommandRegistry`
- `frontend/src/services/automation/evaluator.ts` - `AutopilotAnswer` type, `autopilotControl` threaded through `ExecutionContext`/`executeTokens`/`executeAutomationAction`/`executeWithTimeout`, `case 'AUTO':`
- `frontend/src/services/automation.ts` - `setAutopilotControl`, D-08 `isDirectiveInput` labelling, widened `onSubmitCommand`/`setSubmitCommandCallback`, `processCommandQueue` passes `cmd.source`
- `frontend/src/services/api.ts` - `sendCommand(command, source?)`, `case 'autopilot':`, `autopilotHandlers`/`onAutopilot`/`offAutopilot`, `setAutopilot()`
- `frontend/src/types/index.ts` - `SessionStatus.autopilot_state`, `WSMessageType`/`WSMessage` gain `autopilot`/`source`
- `public/index.html`, `public/assets/index-JQMLws06.js[.map]` - rebuilt production bundle

## Decisions Made
- `AutopilotAnswer` is defined once in `evaluator.ts` and imported by both `automation.ts` (typing `setAutopilotControl`'s parameter) and `api.ts` (`setAutopilot`'s return type) — one shared definition, per the plan's explicit instruction, rather than two structurally-identical types.
- The unknown-`#AUTO`-argument line is a discretionary addition (not in the UI contract) since the grammar's `requiresArgs: false` accepts any argument; it follows the same bracketed, plain-spoken voice as every other notice in this file.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- The worktree's git history was stale at spawn time (HEAD sat on an unrelated SP06 numpad-key-support commit chain, not the phase-02 wave-2 base). Corrected per the mandatory worktree-branch-check protocol: verified HEAD was on the `worktree-agent-*` namespace (not a protected ref), then hard-reset to the required base commit `1f444a64f091d24d34124d8bfab49afa60af0338` before any edits began. No repo files were touched by this correction; it is a worktree-setup issue, not a plan deviation.
- `frontend/node_modules` did not exist in the fresh worktree checkout; ran `npm ci` inside `frontend/` (not the main repo) before the first build, per the tooling note.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `context.autopilotControl` (evaluator.ts) and `api.ts`'s `setAutopilot`/`onAutopilot`/`offAutopilot` are the wiring seams plan 02-06 uses to connect `#AUTO` to the session context's connection id and to drive the header badge; neither this plan's `evaluator.ts` nor `automation.ts` needs to know about connection ids — the callback captures that.
- The outbound `data` message's `source` field now carries `CommandSource` end-to-end (queue → callback → `sendCommand` → JSON payload), ready for plan 02-03's server-side wheel-grab to read; `PlayScreen.tsx`'s `setSubmitCommandCallback` registration and its blank-command direct-send branch (Pitfall 2, RESEARCH.md) still need their own call-site updates in plan 02-06 to actually pass `source` through to `wsManager.sendCommand` — that plumbing is explicitly out of this plan's five-file scope (`PlayScreen.tsx` is not in `files_modified`).
- `SessionStatus.autopilot_state` and `WSMessage`'s `autopilot` type/`onAutopilot` handler are ready for `SessionContext.tsx`'s `refreshStatus()` extension and the new `AutopilotBadge.tsx` component, both plan 02-06 work per `02-UI-SPEC.md`.

## Self-Check: PASSED

- FOUND: frontend/src/services/automation/commands.ts (modified)
- FOUND: frontend/src/services/automation/evaluator.ts (modified)
- FOUND: frontend/src/services/automation.ts (modified)
- FOUND: frontend/src/services/api.ts (modified)
- FOUND: frontend/src/types/index.ts (modified)
- FOUND: commit 9e15007 (Task 1)
- FOUND: commit 19e3839 (Task 2)
- FOUND: commit 695ddb5 (public bundle rebuild)
- `cd frontend && npm run build` (tsc + vite build) exit 0, both after Task 1 and after Task 2
- `grep -c "'AUTO'" frontend/src/services/automation/commands.ts` = 2 (key + `name: 'AUTO'`)
- `grep -c "case 'AUTO':" frontend/src/services/automation/evaluator.ts` = 1
- `grep -c 'AI Player has not been configured' frontend/src/services/automation/evaluator.ts` = 0
- Every bracketed UI-contract string (`[Autopilot engaged]`, `[Autopilot disengaged]`, `[Autopilot is already on]`, `[Autopilot is already off]`, `[Autopilot needs a connected game; connect first]`) found character-for-character in `evaluator.ts`
- `case 'AUTO':` block contains only `context.outputMessage` calls — no `echoLocal`, no MUD-send call
- `grep -c 'onSubmitCommand(cmd.command, cmd.source)' frontend/src/services/automation.ts` = 1; `grep -c 'onSubmitCommand(cmd.command)'` = 0
- `grep -n "autopilot_state?: 'on' | 'waiting' | 'off';"` found in `frontend/src/types/index.ts`
- `sendCommand(command: string, source?: CommandSource)` found in `api.ts`; payload includes `source`
- `case 'autopilot':`, `autopilotHandlers`, `onAutopilot(`, `offAutopilot(` all found in `api.ts`
- `setAutopilot(` found, POSTs to `/api/v1/session/autopilot`, `credentials: 'include'`, `handleAuthError` called before the `response.ok` check
- `grep -c "export type CommandSource = 'user' | 'alias' | 'trigger'" frontend/src/services/automation.ts` = 1 (unchanged)
- `git diff --stat frontend/package.json` empty (T-2-SC) — no dependency added
- `go build ./...` exit 0
- `go test ./...` shows only the pre-existing, excused `internal/icm` `TestHandlerRegistration/CANCEL` baseline failure — no new failures
- `git diff --diff-filter=D --name-only` empty for every task commit — no unintended deletions

---
*Phase: 02-autopilot-switch*
*Completed: 2026-09-15*
