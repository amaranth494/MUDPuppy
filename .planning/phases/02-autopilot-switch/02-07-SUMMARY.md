---
phase: 02-autopilot-switch
plan: 07
subsystem: testing
tags: [evidence, staging, railway, screenshots, canned-report, autopilot]

# Dependency graph
requires:
  - phase: 02-autopilot-switch (plan 01)
    provides: "AutopilotState (off/on/waiting), Manager.EngageAutopilot/DisengageAutopilot, the [AI-PLAYER] autopilot log line format"
  - phase: 02-autopilot-switch (plan 02)
    provides: "POST /api/v1/session/autopilot, StatusResponse.autopilot_state"
  - phase: 02-autopilot-switch (plan 03)
    provides: "Server-side wheel-grab at the single command-ingress point"
  - phase: 02-autopilot-switch (plan 04)
    provides: "#AUTO directive grammar, source-tagged commands"
  - phase: 02-autopilot-switch (plan 05)
    provides: "scripts/verify-phase2.sh canned-report harness"
  - phase: 02-autopilot-switch (plan 06)
    provides: "AutopilotBadge.tsx and the terminal notices"
provides:
  - "evidence/01-test-report.txt — verbatim twelve-command diagnostic capture (go build, go vet, full go test -v, seven targeted runs, npm run build, dependency-drift check)"
  - "evidence/02-harness-selftest.txt — the canned-report harness proving itself: clean self-test and a FAIL C1 negative self-test"
  - "evidence/03-canned-report.txt — scripts/verify-phase2.sh run live against staging, 22 PASS / 0 FAIL / 2 SKIP, exit 0"
  - "evidence/04-staging-ai-player.log — every [AI-PLAYER] autopilot line across four Phase 2 staging deployments, all six causes present"
  - "evidence/05-badge-after-refresh.png through evidence/12-hand-play-unchanged.png (plus 11a, extra) — nine end-user screenshots covering every player-observable state"
  - "02-SECURITY-AGENDA.md — the Phase 2 security review agenda: waiting-to-on auto-resume plus carried-forward R-02/R-04, nothing decided in advance"
  - "02-07-SUMMARY.md — the four-row ROADMAP criterion table with line-numbered citations, closing Phase 2's Phase Validation"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Evidence-as-file: every ROADMAP success criterion is proven by a citable file on disk (report line, log line, or screenshot), never a database query"
    - "Fix-rebuild-redeploy-recapture: bugs found mid-walkthrough are fixed, committed, rebuilt, redeployed, and the evidence is recaptured against the fixed build, not the buggy one"

key-files:
  created:
    - .planning/phases/02-autopilot-switch/evidence/01-test-report.txt
    - .planning/phases/02-autopilot-switch/evidence/02-harness-selftest.txt
    - .planning/phases/02-autopilot-switch/evidence/03-canned-report.txt
    - .planning/phases/02-autopilot-switch/evidence/04-staging-ai-player.log
    - .planning/phases/02-autopilot-switch/evidence/05-badge-after-refresh.png
    - .planning/phases/02-autopilot-switch/evidence/06-refused-before-acceptance.png
    - .planning/phases/02-autopilot-switch/evidence/07-wheel-grab.png
    - .planning/phases/02-autopilot-switch/evidence/08-trigger-keeps-on.png
    - .planning/phases/02-autopilot-switch/evidence/09-waiting-after-drop.png
    - .planning/phases/02-autopilot-switch/evidence/10-resume-after-reconnect.png
    - .planning/phases/02-autopilot-switch/evidence/11-off-while-waiting-stays-off.png
    - .planning/phases/02-autopilot-switch/evidence/11a-off-typed-while-waiting.png
    - .planning/phases/02-autopilot-switch/evidence/12-hand-play-unchanged.png
    - .planning/phases/02-autopilot-switch/02-SECURITY-AGENDA.md
    - .planning/phases/02-autopilot-switch/02-07-SUMMARY.md
  modified:
    - frontend/src/components/Sidebar.tsx (fix: mount the badge)
    - frontend/src/services/api.ts (fix: synchronous disconnect notification)
    - frontend/src/context/SessionContext.tsx (fix: #AUTO OFF reachable while waiting)
    - frontend/src/pages/PlayScreen.tsx (fix: input accepts # directives while waiting)

key-decisions:
  - "Logs captured with the Railway CLI, railway logs --environment staging -d <deployment-id> | grep -F '[AI-PLAYER]', one section per deployment (four total), not the MCP tool or the dashboard"
  - "Screenshots captured with the Claude in Chrome extension's page screenshot (viewport capture of the play screen only: sidebar pill + badge, terminal, input; no devtools, no terminal window, no raw JSON), relayed to disk as JPEG and converted to PNG with Pillow (1317x896); owner reviewed the set at the checkpoint"
  - "05-badge-after-refresh.png deviates from the plan's expected image (badge reads Waiting, not On) because a page reload closes the websocket, which this app has always treated as a disconnect; this is documented as a deviation, not a defect, because it proves the badge re-syncs from the server rather than memory"
  - "Three bugs found during the walkthrough were fixed, committed, rebuilt, and redeployed before the affected evidence was captured, so every screenshot and log line reflects the fixed build, not the buggy one"

patterns-established: []

requirements-completed: [REQ-autopilot-directives, REQ-wheel-grab, REQ-no-auto-reconnect, REQ-doc-hand-play-and-gate]

# Metrics
duration: ~90min
completed: 2026-09-15
---

# Phase 2 Plan 7: Phase 2 Evidence Filing and Security Agenda Summary

**Phase 2's four ROADMAP success criteria are each proven by a named, line-numbered file on disk — a verbatim test report, a harness self-test, a live staging canned report, a staging log excerpt, and nine end-user screenshots — with zero database queries anywhere in the evidence, and the Phase 2 security review has a written agenda deciding nothing in advance.**

## Performance

- **Duration:** ~90 min across Tasks 02-07-01 through 02-07-04 (test report/harness capture, staging deploy and canned report/log, browser walkthrough with three mid-walkthrough fixes, and this written work)
- **Tasks:** 4 (2 checkpoints, both approved-pending at time of writing this summary — see below)
- **Files modified:** 13 evidence files + 2 written documents (this SUMMARY and the security agenda) + 4 source files fixed during the walkthrough

## Accomplishments

- Captured the full twelve-command diagnostic gate verbatim into `evidence/01-test-report.txt`: `go build ./...` (EXIT 0), `go vet` (EXIT 0), the full `go test ./... -v` (one pre-existing `internal/icm` `TestHandlerRegistration/CANCEL` baseline failure, annotated and excused, `internal/session`/`internal/store` fully green), `go test ./internal/session/... -race -v` (no data race), five targeted `go test -run` invocations (`TestAutopilotTransitions`, `TestEngageAutopilot`, `TestWaitingSurvivesDisconnectAndResumesOnConnect`, `TestWheelGrabSourceRule`, `TestAutopilotHandler|TestStatusCarriesAutopilotState`), the Phase 1 regression `TestEngageGate`, `npm run build` (EXIT 0), and an empty `### DEPENDENCY DRIFT` section confirming no package was added (T-2-SC). File ends `GO TEST EXIT: 1` (the full-suite exit code, carrying only the pre-existing `internal/icm` failure).
- Proved the canned-report harness falsifiable before trusting it: `evidence/02-harness-selftest.txt` shows a clean `--self-test` (22 checks, 0 FAIL, exit 0) and a `--self-test-negative` run that fails on purpose (`FAIL C1` twice, exit 1) — T-2-07.
- Deployed the branch to Railway staging four times across the walkthrough (deployments `d8552a5b`, `6ca35a4c`, `1d21cc21`, `e40d2c11`) and produced `evidence/03-canned-report.txt`: `scripts/verify-phase2.sh --no-tests` run live against `https://mudpuppy-staging.up.railway.app` at git SHA `7a70ab8` with never-accepted profile `eda71e2f-2574-4bfe-98e7-3af2c8852a56` — 22 PASS, 0 FAIL, 2 SKIP (C2 and C3, each naming their screenshot/log evidence), exit 0. A first run (profile `67a3c3ad…`, without `--no-tests`) also passed every HTTP check but exited 1 solely because the appended full Go suite carries the same pre-existing excused `internal/icm` failure documented in `evidence/01-test-report.txt`; the filed report is the `--no-tests` run.
- Captured `evidence/04-staging-ai-player.log` with the Railway CLI (`railway logs --environment staging -d <deployment-id> | grep -F '[AI-PLAYER]'`), one section per deployment, containing at least one line for every required cause: `refused-gate`, `refused-no-session`, `engage`, `wheel-grab`, `disconnect`, `resume` (plus `already-on`, `already-off`, `disengage` from the fuller walkthrough).
- Walked the play screen on staging against a live MUD (Alter Aeon) across five profiles and captured nine PNGs (eight required plus one supporting extra), covering every player-observable state in the ROADMAP Phase Validation line.
- Found and fixed three bugs mid-walkthrough (see Deviations below), rebuilt, redeployed, and recaptured the affected evidence against the fixed build.
- Wrote `02-SECURITY-AGENDA.md`: the Phase 2 security review agenda, deciding nothing in advance.

## Evidence File Inventory

| File | Size | Produced by |
|------|------|-------------|
| `evidence/01-test-report.txt` | 61,944 bytes | Task 02-07-01 |
| `evidence/02-harness-selftest.txt` | 13,310 bytes | Task 02-07-01 |
| `evidence/03-canned-report.txt` | 6,334 bytes | Task 02-07-02 |
| `evidence/04-staging-ai-player.log` | 10,037 bytes | Task 02-07-02 |
| `evidence/05-badge-after-refresh.png` | 74,609 bytes | Task 02-07-03 |
| `evidence/06-refused-before-acceptance.png` | 185,618 bytes | Task 02-07-03 |
| `evidence/07-wheel-grab.png` | 209,392 bytes | Task 02-07-03 |
| `evidence/08-trigger-keeps-on.png` | 361,397 bytes | Task 02-07-03 |
| `evidence/09-waiting-after-drop.png` | 177,120 bytes | Task 02-07-03 |
| `evidence/10-resume-after-reconnect.png` | 171,625 bytes | Task 02-07-03 |
| `evidence/11-off-while-waiting-stays-off.png` | 158,950 bytes | Task 02-07-03 |
| `evidence/11a-off-typed-while-waiting.png` (extra, supporting) | 175,931 bytes | Task 02-07-03 |
| `evidence/12-hand-play-unchanged.png` | 239,195 bytes | Task 02-07-03 |

**Log capture tool:** Railway CLI, `railway logs --environment staging -d <deployment-id> | grep -F '[AI-PLAYER]'` — not the MCP log tool, not the dashboard pane.

**Screenshot capture method:** the Claude in Chrome extension's page screenshot (viewport capture of the MUDPuppy play screen only — sidebar with the CONNECTED/DISCONNECTED pill and the AUTOPILOT badge, terminal, input; no devtools pane, no terminal window, no raw JSON in frame), relayed to disk as JPEG and converted to PNG with Pillow, 1317x896. The owner reviewed the resulting set of nine at the checkpoint.

**Profiles created on staging for evidence** (all alteraeon.com:3000): Phase 2 Harness (`67a3c3ad-93c5-4a38-923f-38b8f4f0cc7c`), Phase 2 Harness B (`eda71e2f-2574-4bfe-98e7-3af2c8852a56`), Phase 2 Walkthrough (`00daed67-7d1c-434d-8c61-0063feb6e672`), Phase 2 Refusal (`3239395c-0f51-4511-8f05-a355cf27c58e`), Phase 2 Hand Play (`89ea91f3-d4ec-47bc-b6b9-c8204868d332`).

No SQL was run anywhere in this phase's evidence capture.

## Phase Validation

ROADMAP's Phase 2 Phase Validation line: *"Player-observable on staging against a live MUD — `#AUTO ON` is refused before acceptance and engages after it; the indicator matches the server state after a page refresh; typing a command while engaged shows the disengage notice and the command's game response; a trigger-fired command leaves autopilot engaged; dropping the connection shows the indicator as waiting and no command is sent while disconnected; reconnecting by hand shows it resuming to ON without typing `#AUTO ON`; `#AUTO OFF` while waiting lands on OFF and it stays OFF after reconnecting. Diagnostic — `go test` covers the engaged-state machine and the human-versus-automation source flag."*

**Performed.** Every clause is demonstrated below with a citable file. No row cites a database query or a database inspection.

| Criterion | What it claims | Evidence | Verdict |
|-----------|----------------|----------|---------|
| 1 | `#AUTO ON` engages only past the gate, `#AUTO OFF` disengages, the indicator always matches the server including after a refresh | `evidence/01-test-report.txt` lines 366-382 (`TestEngageAutopilot`, all 4 subtests PASS), lines 553-571 (`-race` rerun, PASS), lines 675-692 (targeted rerun, PASS); `evidence/03-canned-report.txt` lines 22-23, 27-29, 34-42, 50-53, 56-61, 66-67, 77-78, 82-84 (`PASS C1` throughout, 12 assertions, zero FAIL); `evidence/06-refused-before-acceptance.png` (refusal sentence + badge Off); `evidence/05-badge-after-refresh.png` (see deviation note below — badge reads Waiting after a hard reload because the reload drops the websocket, which this app treats as a disconnect; the true server-held `autopilot_state` at that moment was `waiting`, proving the badge re-syncs from the server, not memory; `evidence/10-resume-after-reconnect.png` then shows the same parked switch resuming to On on the next manual connect) | PASS |
| 2 | Typing disengages before the command is sent with no lost keystrokes; aliases, triggers and timers do not trip it | `evidence/01-test-report.txt` lines 403-450 (`TestWheelGrabSourceRule`, classifier + 7 decision subtests, all PASS), lines 592-639 (`-race` rerun, PASS), lines 722-770 (targeted rerun, PASS); `evidence/07-wheel-grab.png` (typed `look` while On → `[Autopilot disengaged: you took the wheel]`, the game's reply, badge Off, one frame; empty-Enter-while-On also disengages per D-07, visible in session history); `evidence/08-trigger-keeps-on.png` (repeating timer + trigger fire while On, badge stays On; a typed `#AUTO ON` confirms `[Autopilot is already on]`; an alias-expanded command correctly DID trip the wheel-grab per D-09); `evidence/04-staging-ai-player.log` lines 15, 17, 32, 36 (`cause=wheel-grab`) | PASS |
| 3 | A drop parks it at waiting issuing nothing, the connection returning resumes it, `#AUTO OFF` while waiting stays off | `evidence/01-test-report.txt` lines 383-402 (`TestWaitingSurvivesDisconnectAndResumesOnConnect`, 3 subtests PASS), lines 572-591 (`-race` rerun, PASS), lines 697-717 (targeted `-race` rerun, PASS); `evidence/09-waiting-after-drop.png` (`[Disconnected]` then `[Autopilot waiting for reconnect]`, badge Waiting, pill Disconnected); `evidence/10-resume-after-reconnect.png` (`[Reconnected]` then `[Autopilot resuming]`, badge On, `#AUTO ON` never typed); `evidence/11-off-while-waiting-stays-off.png` + `evidence/11a-off-typed-while-waiting.png` (`#AUTO OFF` typed while Waiting produces `[Autopilot disengaged]` and badge Off while still Disconnected — 11a; badge still Off after reconnecting by hand, no resume line, because the terminal clears on a new connection — 11); `evidence/04-staging-ai-player.log` lines 13, 19, 22 (`cause=disconnect`) and lines 14, 23, 25 (`cause=resume`) | PASS |
| 4 | A profile can be created, the policy accepted, and the character hand-played with nothing about ordinary play changed | `evidence/03-canned-report.txt` lines 28-29, 46-47, 71-73 (`PASS C4`, policy accept + IDOR refusal); `evidence/12-hand-play-unchanged.png` (fresh profile, policy accepted, `look`/`help`/`who` typed by hand, no bracketed autopilot line, no banner, badge Off) | PASS |

No row above cites a SQL query or a database inspection as its evidence.

## Deviations from Plan

### Auto-fixed Issues (all Rule 1 — bug fixes found during the walkthrough)

**1. [Rule 1 - Bug] `Header.tsx` is never rendered, so plan 02-06's badge never appeared**
- **Found during:** Task 02-07-03, first screenshot attempt
- **Issue:** Plan 02-06 mounted `<AutopilotBadge />` in `frontend/src/components/Header.tsx`, but `Header.tsx` is not rendered anywhere in the running app — a gap in plan 02-06's delivered claim.
- **Fix:** `AutopilotBadge` is now rendered in `Sidebar.tsx` directly under the `SessionBadge` pill.
- **Files modified:** `frontend/src/components/Sidebar.tsx`
- **Commit:** `b5af164` (fix(02-06): mount the autopilot badge in the sidebar beside the live connection badge)

**2. [Rule 1 - Bug] `[Disconnected]` never printed for a user-clicked drop**
- **Found during:** Task 02-07-03, screenshot 09 attempt
- **Issue:** The sidebar disconnect control closed the socket after `PlayScreen` had unregistered its handlers (pre-existing race), so criterion 3's `[Disconnected]` line never printed for user-initiated drops.
- **Fix:** `WebSocketManager.notifyDisconnect()` now fires the handlers synchronously before close, with a guard against double firing.
- **Files modified:** `frontend/src/services/api.ts`
- **Commit:** `1d9e95f` (fix(02-06): print [Disconnected] on a user-initiated disconnect too)

**3. [Rule 1 - Bug] `#AUTO OFF` unreachable while autopilot was parked**
- **Found during:** Task 02-07-03, screenshot 11 attempt
- **Issue:** The input was disabled whenever disconnected and the `#AUTO` control was unbound once the connection id cleared, so criterion 3's "`#AUTO OFF` while waiting" could not be typed.
- **Fix:** While `autopilotState` is waiting, the input now accepts `#` directives (other text still prints `[Not connected]`), and the control stays bound to the parked connection id. The server already allowed `off` unconditionally.
- **Files modified:** `frontend/src/context/SessionContext.tsx`, `frontend/src/pages/PlayScreen.tsx`
- **Commit:** `6bdeccd` (fix(02-07): let #AUTO OFF reach the server while autopilot is parked)

All three fixes were committed on `ai-player`, rebuilt, redeployed to staging, and the affected evidence (badge screenshots, the disconnect line, and screenshots 11/11a) was captured on the fixed build.

### Documented Deviation (not a bug — a corrected expectation)

**05-badge-after-refresh.png shows Waiting, not On.** The plan's task text expected the badge to still read `Autopilot: On` after a hard reload. A page reload closes the websocket, and this app has always treated a closed websocket as a game disconnect (`internal/session/websocket.go`, pre-existing SP02-era behavior) — not new to this phase. The server therefore parked autopilot at `waiting` before the reload completed, and the badge, filled from `GET /api/v1/session/status` after the reload, correctly reads `Autopilot: Waiting` with the pill `Disconnected`. This is the true server state (the status API returned `autopilot_state: "waiting"` at that moment) and it proves the badge re-syncs from the server rather than memory (the pre-reload memory said On). The criterion's actual requirement — "the indicator always matches the true server-held state, including after a page refresh" — holds; the plan's guess assumed the connection survives a reload, which it does not. `evidence/10-resume-after-reconnect.png` then shows the same parked switch resuming to On on the next manual connect, closing the loop.

## Known Quirks (observed, no change made)

- **Log attribution:** `internal/session/manager.go`'s `[AI-PLAYER] autopilot` line prints the record's last-engaged `connection_id`, not the requesting one, for `refused-no-session` and `already-off`. In `evidence/04-staging-ai-player.log`, the two harness runs' causes for these therefore show `connection_id=00daed67…` (the Walkthrough profile) while `refused-gate` and `policy accepted` show the harness profile id. Minor diagnostic-clarity issue; suggest fixing in a later phase.
- **Host tooling, not a project dependency:** plan 02-01's executor installed a MinGW-w64 GCC via `winget` (user scope) so `go test -race` could run; `go.mod` and `frontend/package.json` are unchanged, confirmed by `evidence/01-test-report.txt`'s empty `### DEPENDENCY DRIFT` section.
- **Automation engine came up disabled on connect** despite `settings.automation_enabled=true` being stored (a stale profile closure at connect time; pre-existing SP06 behaviour) — worked around in-session with the "Enable Automation" button for screenshot 08.
- **Chrome's page reload drops the game session** — pre-existing, documented above as the reason for 05's content.

## Task Commits

1. **Task 02-07-01: Test report and harness self-test** — `f8bd07a` (test)
2. **Task 02-07-02: Staging canned report and log** — `19e59c0` (test, checkpoint)
3. **Task 02-07-03: Screenshots and walkthrough fixes** — `b5af164`, `1d9e95f`, `6bdeccd` (fix), `7a70ab8` (test, checkpoint)
4. **Task 02-07-04: Security agenda** — this commit (docs)

## Checkpoint Status

- **Task 02-07-02 (staging canned report + log):** performed by the orchestrator against live Railway staging. Owner approval pending at the time of writing this summary — the orchestrator presents both this summary and the underlying evidence files to the owner after this plan's tracking commit.
- **Task 02-07-03 (screenshot walkthrough):** performed by the orchestrator against live Railway staging with a live MUD connection. Owner approval pending at the time of writing this summary, for the same reason.

## Security Agenda

The Phase 2 security review agenda is filed at `.planning/phases/02-autopilot-switch/02-SECURITY-AGENDA.md`. It carries three items — the new waiting-to-on auto-resume question, and the two Phase 1 carried-forward risks R-02 and R-04 — each with Accept / Defer / Remediate Now and nothing chosen in advance.

## Issues Encountered

None beyond the three bugs documented above, all fixed within this plan's scope.

## User Setup Required

None — no external service configuration required beyond the existing Railway staging deployment used throughout Phase 2.

## Next Phase Readiness

Phase 2's ROADMAP Phase Validation line has been demonstrated end to end on staging with no database queries anywhere in the evidence trail. All four success criteria carry a PASS verdict with a citable file. The Phase 2 security review can begin using `02-SECURITY-AGENDA.md`; the project cannot close while any agenda item remains deferred.

## Self-Check

- FOUND: `.planning/phases/02-autopilot-switch/evidence/01-test-report.txt` (contains `GO TEST EXIT`)
- FOUND: `.planning/phases/02-autopilot-switch/evidence/02-harness-selftest.txt` (contains `FAIL C1`)
- FOUND: `.planning/phases/02-autopilot-switch/evidence/03-canned-report.txt` (contains `PASS C1`, `PASS C4`, `SKIP C2`, `SKIP C3`)
- FOUND: `.planning/phases/02-autopilot-switch/evidence/04-staging-ai-player.log` (contains `[AI-PLAYER] autopilot`, all six causes)
- FOUND: `.planning/phases/02-autopilot-switch/evidence/05-badge-after-refresh.png`
- FOUND: `.planning/phases/02-autopilot-switch/evidence/06-refused-before-acceptance.png`
- FOUND: `.planning/phases/02-autopilot-switch/evidence/07-wheel-grab.png`
- FOUND: `.planning/phases/02-autopilot-switch/evidence/08-trigger-keeps-on.png`
- FOUND: `.planning/phases/02-autopilot-switch/evidence/09-waiting-after-drop.png`
- FOUND: `.planning/phases/02-autopilot-switch/evidence/10-resume-after-reconnect.png`
- FOUND: `.planning/phases/02-autopilot-switch/evidence/11-off-while-waiting-stays-off.png`
- FOUND: `.planning/phases/02-autopilot-switch/evidence/11a-off-typed-while-waiting.png`
- FOUND: `.planning/phases/02-autopilot-switch/evidence/12-hand-play-unchanged.png`
- FOUND: commit `f8bd07a` (Task 1)
- FOUND: commit `19e59c0` (Task 2)
- FOUND: commit `b5af164`, `1d9e95f`, `6bdeccd`, `7a70ab8` (Task 3, including walkthrough fixes)
- FOUND: `.planning/phases/02-autopilot-switch/02-SECURITY-AGENDA.md`

## Self-Check: PASSED

---
*Phase: 02-autopilot-switch*
*Completed: 2026-09-15*

## Owner approval

Phase 2 was approved by the owner on 2026-09-15 through the Autopilot Switch Evidence Dossier (https://claude.ai/artifact/HoKNf4VDKDGX49r97K9Y6B), after the verifier passed 4 of 4, the code review's three critical findings were fixed in b5c8bf5 and redeployed, and the regression gate passed. Both 02-07 checkpoints are approved by that same act.
