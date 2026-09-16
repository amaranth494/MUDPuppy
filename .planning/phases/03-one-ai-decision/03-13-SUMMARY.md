---
phase: 03-one-ai-decision
plan: 13
subsystem: testing
tags: [evidence, staging, railway, screenshots, canned-report, gemini, autopilot]

# Dependency graph
requires:
  - phase: 03-one-ai-decision (plan 01)
    provides: "the rolling ANSI-free window the driver reads"
  - phase: 03-one-ai-decision (plan 02)
    provides: "AI model registry from the environment, AIConfigured, the Gemini client"
  - phase: 03-one-ai-decision (plan 03)
    provides: "the live ICM engine and TestDispatch_AutomationPassThrough"
  - phase: 03-one-ai-decision (plan 05)
    provides: "session transcripts and their read endpoints"
  - phase: 03-one-ai-decision (plan 06)
    provides: "the D-20 not-configured refusal"
  - phase: 03-one-ai-decision (plan 08)
    provides: "the driver: one decision per engage through Dispatch in the automation context"
  - phase: 03-one-ai-decision (plan 09)
    provides: "the ai websocket push and the decisions endpoint"
  - phase: 03-one-ai-decision (plan 10)
    provides: "the AI Assist panel and the [AI-ASSIST > command] terminal line"
  - phase: 03-one-ai-decision (plan 11)
    provides: "the Logs section and the two-pane log page"
  - phase: 03-one-ai-decision (plan 12)
    provides: "scripts/verify-phase3.sh canned-report harness"
provides:
  - "evidence/01-test-report.txt -- verbatim eighteen-command diagnostic capture, zero failures, zero data races, empty DEPENDENCY DRIFT and MODEL LITERALS sections, GO TEST EXIT: 0"
  - "evidence/02-harness-selftest.txt -- the harness proving itself: clean self-test (exit 0) and a FAIL C3 negative self-test (exit 1), re-captured after the live-found harness fix"
  - "evidence/03-canned-report.txt -- RUN A with the Gemini variables unset (20 PASS / 0 FAIL / 2 SKIP) and RUN B after the walkthrough (0 FAIL, the newest sent decision carrying reasoning and command), plus the walkthrough session's own transcript read"
  - "evidence/04-staging-ai-player.log -- startup excerpts for both worlds (migration 011 both times) and every [AI-PLAYER] line from the six walkthrough deployments"
  - "evidence/05 through 14 -- ten end-user screenshots covering every player-observable claim, captured on the fixed build"
  - "03-SECURITY-AGENDA.md -- the Phase 3 security review agenda, nothing decided in advance"
  - "03-13-SUMMARY.md -- the four-row ROADMAP criterion table with line-numbered citations, closing Phase 3's Phase Validation"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Evidence-as-file: every ROADMAP success criterion is proven by a citable file on disk (report line, log line, or screenshot), never a database query"
    - "Fix-rebuild-redeploy-recapture: bugs found mid-walkthrough are fixed, committed, rebuilt, redeployed, and the evidence is recaptured against the fixed build"
    - "Staging deploys are `railway up` uploads of the local checkout; the Railway dashboard's Deploy rebuilds from GitHub, where the branch has never been pushed"

key-files:
  created:
    - .planning/phases/03-one-ai-decision/evidence/01-test-report.txt
    - .planning/phases/03-one-ai-decision/evidence/02-harness-selftest.txt
    - .planning/phases/03-one-ai-decision/evidence/03-canned-report.txt
    - .planning/phases/03-one-ai-decision/evidence/04-staging-ai-player.log
    - .planning/phases/03-one-ai-decision/evidence/05-decision-in-panel.png
    - .planning/phases/03-one-ai-decision/evidence/06-ai-assist-terminal-line.png
    - .planning/phases/03-one-ai-decision/evidence/07-panel-after-refresh.png
    - .planning/phases/03-one-ai-decision/evidence/08-refused-not-configured.png
    - .planning/phases/03-one-ai-decision/evidence/09-auto-unknown-option.png
    - .planning/phases/03-one-ai-decision/evidence/10-decision-failure-disengage.png
    - .planning/phases/03-one-ai-decision/evidence/11-logs-section.png
    - .planning/phases/03-one-ai-decision/evidence/12-log-page-two-pane.png
    - .planning/phases/03-one-ai-decision/evidence/13-hand-play-unchanged.png
    - .planning/phases/03-one-ai-decision/evidence/14-connections-cleaned.png
    - .planning/phases/03-one-ai-decision/03-SECURITY-AGENDA.md
  modified:
    - frontend/src/services/automation/evaluator.ts (fix found in task 03-13-02)
    - internal/driver/driver.go (diagnostic line found necessary in task 03-13-03)
    - cmd/server/main.go (client timeout found necessary in task 03-13-03)
    - scripts/verify-phase3.sh (fix found in task 03-13-03)
    - public/ (production bundle rebuilt for the deploys)
    - .railwayignore (new)

key-decisions:
  - "Screenshots are Win32 PrintWindow captures of the Chrome window cropped to the page viewport (the extension's save-to-disk produced no file on this machine); they are the page as the owner sees it, no browser chrome, no devtools"
  - "The walkthrough decision was made on the Phase 2 Hand Play profile, whose panel had no history, after the Phase 2 Walkthrough profile had accumulated failure entries during model troubleshooting"
  - "The Gemini model is an environment variable and was changed twice on staging (gemini-2.5-flash -> gemini-3.8-flash -> gemini-3.5-flash) without any source change; the client timeout is code (120s)"

patterns-established:
  - "A model-error log line carries only kind and HTTP status, never the vendor message, so staging failures can be told apart without breaking T-3-07"

requirements-completed: [REQ-single-decision, REQ-reasoning-visibility, REQ-env-config]

# Metrics
duration: ~4h across the four tasks (test report capture; deploy, RUN A and two screenshots; the walkthrough with four staging redeploys, RUN B, the log excerpt and the cleanup; the agenda)
completed: 2026-09-16
---

# Phase 03 Plan 13: Phase 3 demonstrated on staging and filed as evidence

**The ROADMAP Phase Validation was performed on staging on 2026-09-16: with autopilot engaged, Gemini read the game, made one decision, the command went through the ICM automation context to the game, the reasoning appeared in the panel as it happened and came back after a refresh, the server refused to drive while unconfigured, and every one of those claims points at a file.**

## Performance

- **Duration:** ~4 hours wall clock, most of it the staging walkthrough (four redeploys while the Gemini model and client timeout were sorted out, one browser session expiry, one browser-extension reconnect).
- **Started:** 2026-09-16T03:20Z (task 03-13-01 by the executor agent)
- **Completed:** 2026-09-16T14:30Z (this summary)
- **Tasks:** 4/4 (03-13-01 executor; 03-13-02 and 03-13-03 performed by the orchestrator against live staging with the owner supplying the key and two session cookies; 03-13-04 executor)

## Accomplishments

- Captured the eighteen-command diagnostic gate verbatim into `evidence/01-test-report.txt`: `go build` (EXIT 0), `go vet` (EXIT 0), the full `go test ./... -v` with zero `--- FAIL` lines (the Phase 2 baseline failure `TestHandlerRegistration/CANCEL` was fixed at this phase's wave 1 gate, commit `dcb671d`), the eleven targeted runs including seven `-race` runs with no `DATA RACE`, `npm run build` (EXIT 0), an empty `### DEPENDENCY DRIFT` section (line 1368) and an empty `### MODEL LITERALS` section (line 1372), ending `GO TEST EXIT: 0` (line 1376). `-race` ran with the user-scope MinGW GCC Phase 2 installed, prefixed inline before each command.
- Captured `evidence/02-harness-selftest.txt`: `--self-test` 14 PASS / 2 SKIP / 0 FAIL, exit 0; `--self-test-negative` four `FAIL C3` lines, exit 1. Re-captured after the harness fix below.
- Deployed the branch to Railway staging with the Gemini variables absent (deploy `8f6251fe`), confirmed migration 011 at startup, ran RUN A of the harness live: 20 PASS / 0 FAIL / 2 SKIP, exit 0, the `on` step answering `refused-not-configured` with the D-20 sentence character for character (`03-canned-report.txt` lines 26-28).
- The owner created a Gemini API key and set it on staging as `AI_MODEL_GEMINI_KEY`; the orchestrator set `AI_MODEL_DEFAULT=GEMINI`, `AI_MODEL_GEMINI_ENDPOINT`, `AI_MODEL_GEMINI_PROVIDER=gemini` and `AI_MODEL_GEMINI_NAME` (values never recorded here except the model name and the public endpoint). Production received no Phase 3 deploy.
- Watched the AI make one decision on staging (deploy `2f2fcd71`, model `gemini-3.5-flash`): request at 14:02:48Z, answer after six seconds, dispatch through the ICM automation context, sent (`04-staging-ai-player.log` lines 97-101). The panel showed the reasoning and `-> yes`; the terminal showed `[AI-ASSIST > yes]` and the game's reply; the badge read On.
- Proved the decision survives a refresh, forced a failure without touching the server, opened the Logs section and the two-pane log page, hand-played on a profile that never accepted the policy, ran RUN B (0 FAIL, exit 0, the newest sent decision carrying reasoning, command and outcome at line 224), captured every `[AI-PLAYER]` line of the walkthrough per deployment, and deleted the six DR-2-02 profiles through the app.
- Found and fixed three things mid-walkthrough, each redeployed before the affected evidence was captured (see Deviations).

## Evidence File Inventory

| File | Type | What it proves | Task |
|------|------|----------------|------|
| `01-test-report.txt` | canned report | build, vet, the whole Go suite, seven `-race` runs, the frontend build, no dependency drift, no model literal in source | 03-13-01 |
| `02-harness-selftest.txt` | canned report | the harness passes its own fixtures and fails the negative fixture set | 03-13-01, re-captured in 03-13-03 |
| `03-canned-report.txt` | canned report | RUN A (variables unset: the D-20 refusal), RUN B (variables set: decisions and sessions returned over HTTP), the walkthrough session's own lines | 03-13-02, 03-13-03 |
| `04-staging-ai-player.log` | log excerpt | migration 011 at both startups; engage, refusal, request/dispatch/sent, failed, open/close lines; nothing it must not contain | 03-13-02, 03-13-03 |
| `05-decision-in-panel.png` | screenshot | reasoning and command in the panel, badge On | 03-13-03 |
| `06-ai-assist-terminal-line.png` | screenshot | `[AI-ASSIST > yes]` and the game's reply in one frame (same frame as 05) | 03-13-03 |
| `07-panel-after-refresh.png` | screenshot | the decision rebuilt from the server in a fresh terminal | 03-13-03 |
| `08-refused-not-configured.png` | screenshot | the D-20 sentence in the terminal, badge Off | 03-13-02 |
| `09-auto-unknown-option.png` | screenshot | the D-11 line, badge unchanged | 03-13-02 |
| `10-decision-failure-disengage.png` | screenshot | the same failure sentence in panel and terminal, badge Off | 03-13-03 |
| `11-logs-section.png` | screenshot | the Logs section with its description and Open Logging | 03-13-03 |
| `12-log-page-two-pane.png` | screenshot | sessions left, transcript right, `> look` / `> help` in cyan, `[AI-ASSIST > yes]` in magenta, game output plain | 03-13-03 |
| `13-hand-play-unchanged.png` | screenshot | no panel, no tab, badge Off on an unaccepted profile | 03-13-03 |
| `14-connections-cleaned.png` | screenshot | the Connections list with none of the six DR-2-02 profiles | 03-13-03 |

## Phase Validation

| Criterion | What it claims | Evidence | Verdict |
|-----------|----------------|----------|---------|
| 1 | With autopilot engaged, the driver reads the game text, sends one request with conduct rules and approach guidance included, and issues the returned command through the ICM automation context so it passes the dispatcher and its safety limits | `evidence/01-test-report.txt` lines 86-92 (`TestHandleEngage` PASS with `dispatch_precedes_send` line 88 and `icm_refusal_sends_nothing` line 92) and line 115 (`TestHandleEngageFailures`); `evidence/05-decision-in-panel.png`; `evidence/06-ai-assist-terminal-line.png`; `evidence/04-staging-ai-player.log` lines 97-101 (`cause=engage`, `stage=request`, `stage=answer`, `stage=dispatch`, `stage=sent`, one decision, `cmd_len=3`) | PASS |
| 2 | The decision and the reasoning appear in the play screen as they happen and are stored, surviving a page refresh | `evidence/05-decision-in-panel.png`; `evidence/07-panel-after-refresh.png`; `evidence/03-canned-report.txt` line 224 (RUN B decisions step returning the walkthrough's reasoning, command `yes`, outcome `sent` over HTTP) and lines 362-370 (the walkthrough session read back with human, ai and game lines); `evidence/12-log-page-two-pane.png` | PASS |
| 3 | Model names and API key come from environment configuration; nothing is hard-coded; the server fails clearly if they are absent when engagement is attempted | `evidence/03-canned-report.txt` lines 26-28 (RUN A `PASS C3`, outcome `refused-not-configured`, the D-20 sentence character for character, state off) and line 39 (no model literal in source); `evidence/08-refused-not-configured.png`; `evidence/01-test-report.txt` lines 1372-1375 (`### MODEL LITERALS` empty) and lines 20 and 30 (`TestLoadAIRegistry`, `TestAIConfigured`); `evidence/04-staging-ai-player.log` lines 23-28 (`cause=refused-not-configured`) and lines 4 and 13 (the server started normally in both worlds) | PASS |
| 4 | An automation-context command is dispatched through the ICM dispatcher and safety checker, and the frontend adapter no longer falls back silently | `evidence/01-test-report.txt` lines 160-162 (`TestDispatch_AutomationPassThrough_GoesThroughSafetyChecker` PASS including `refused_when_rate_limit_tripped`) and line 1269 (the same under `-race`); `evidence/03-canned-report.txt` line 72 and line 250 (the ICM validate route answering 200 in both runs) and lines 160-161 (the two diagnostics exiting 0 live); `evidence/13-hand-play-unchanged.png` | PASS |

No row cites a database query. The owner was never signed out by the executor or the orchestrator; the staging session expired on its own (Redis idle expiry) once during the key setup and the owner signed back in.

**Step 5 observation (D-03):** after the single decision the badge read On and exactly one `[AI-ASSIST > yes]` line appeared; no second command was issued while autopilot stayed on (screenshot 05; log lines 97-101 show one request and one sent).

**Step 7 observation (D-16):** pressing Open Logging opened a new tab titled "Session Logs -- Phase 2 Hand Play (Alter Aeon)"; the Settings tab stayed on `/settings/logs`.

## Deviations from Plan

### Auto-fixed Issues (Rule 1 bug fixes found during the walkthrough, each redeployed before the affected evidence was captured)

1. **The D-20 refusal never reached the terminal.** Plan 03-06 promised `[Autopilot refused: AI is not configured on this server]` in the terminal but changed only the server; the `#AUTO` evaluator had no case for the `refused-not-configured` outcome, so `#AUTO ON` printed nothing while the badge stayed Off. Fixed in `e57e91a` (`frontend/src/services/automation/evaluator.ts`), bundle rebuilt, redeployed as `8f6251fe`; screenshots 08 and 09 and RUN A were captured on that build.
2. **A failed decision could not be told apart.** The first engage with the key set failed as `failure=api-error`, which covers transport, bad-request and auth alike. Added one log line, `[AI-PLAYER] model-error ... kind=<kind> http=<status>` (never the vendor message), in `0e24dad`. It showed HTTP 404 for `gemini-2.5-flash` on the owner's new key, a 30-second transport timeout for `gemini-3.8-flash`, then HTTP 503 (overloaded) twice on that model and once on `gemini-3.5-flash` before the decision succeeded.
3. **The client timeout was too short for a current model.** Raised from the 30-second default to 120 seconds in `ff4a958` (`cmd/server/main.go`).
4. **The harness rejected a legitimate live outcome and checked the wrong decision.** Found by the live RUN B: step 1 treated `refused-wrong-connection` (the browser connected to another profile) as a failure and passed a string where `_check` expects an exit code; step 4 read `decisions[0]`, but the endpoint lists oldest first and a failed decision has no reasoning, so the shape check never ran live. Fixed in `987c22e`; the fixture self-tests still pass and the negative set still fails; evidence 02 re-captured; RUN B re-run.

### Documented Deviations (not bugs -- corrected expectations)

- **Refresh.** A hard page refresh drops the live game session (pre-existing, documented in Phase 2), and the AI Assist panel only mounts with a connected profile. Step 4 was therefore: refresh, reconnect the same profile, take no engage action. The panel rebuilt the decision from the server into a fresh terminal (screenshot 07 shows no `[AI-ASSIST` line in the terminal and the decision in the panel); the badge read Off because the session had been re-established, not because autopilot was switched.
- **The walkthrough profile.** The decision evidence was captured on the Phase 2 Hand Play profile rather than the Phase 2 Walkthrough profile: the Walkthrough profile's panel had accumulated failure entries during the model troubleshooting, and the plan asks for one decision in a clean panel. RUN B therefore used the Hand Play connection id; RUN A had used the Walkthrough id.
- **The model name.** Neither the design document nor the context named a model. `gemini-2.5-flash` (404 on the owner's key), then `gemini-3.8-flash` (the vendor's current recommendation; timed out at 30 seconds, then 503), then `gemini-3.5-flash` (503 once, then success). Each change was an environment variable followed by a `railway up` restart; no source change carries a model name (`01-test-report.txt` line 1372).
- **The failure step used the settings endpoint.** Step 6's "set the profile's model name in Settings" was done through the app's own `PUT /api/v1/profiles/{id}/ai-settings` from the signed-in browser, not by clicking through the AI Player form, and reset to blank the same way afterwards.
- **RUN B's transcript step.** The harness reads the newest session, which was the post-refresh session and held only game and marker lines. The walkthrough session itself was read back over HTTP and appended to RUN B (`03-canned-report.txt` lines 362-371: human=3, ai=1, game=52, marker=4).

## Known Quirks (observed, no change made)

- After an AI-failure disengage the server is off immediately, but the sidebar badge kept reading On for a few seconds until the next status poll; screenshot 10 was taken after it caught up. On the agenda.
- A dashboard "Deploy" on Railway rebuilds from the GitHub repo, where `ai-player` has never been pushed; that build (`1a677b8a`) crashed on `no migration found for version 11`. Every working staging deploy in this phase is a `railway up` upload. On the agenda.
- The owner first placed the key on production as `GOOGLE_API_KEY`, which triggered a production redeploy from GitHub (`11762ced`, old code, variable unused). On the agenda with a recommendation to remove it.
- The Chrome extension's screenshot save produced no file on this machine; the screenshots are Win32 `PrintWindow` captures of the Chrome window cropped to the page viewport, at the window's native size (1278 x 1213 to 2277 x 1297 depending on how the owner had the window sized).

## Task Commits

- `6d6c342` test(03-13): capture Phase 3 test report and harness self-test as evidence (03-13-01)
- `e57e91a` fix(03-06): print the not-configured refusal in the terminal (found in 03-13-02)
- `30ca218` test(03-13): capture D-20 refusal and D-11 unknown-option screenshots on staging
- `567af77` test(03-13): file RUN A of the Phase 3 canned report (Gemini variables unset)
- `0e24dad` feat(03-08): log the model error kind and HTTP status on a failed decision (found in 03-13-03)
- `ff4a958` fix(03-08): give the Gemini client a 120s timeout (found in 03-13-03)
- `5e3854b` test(03-13): capture the decision, refresh, and forced-failure screenshots on staging
- `5f58865` test(03-13): file the Logs section and log page screenshots and the staging log excerpt
- `4cb96ee` test(03-13): capture hand play on a profile that never accepted the policy
- `987c22e` fix(03-12): harness accepts refused-wrong-connection and checks the newest sent decision (found in 03-13-03)
- `454e3fa` test(03-13): append the walkthrough session's human/ai/game lines to RUN B
- `a98e013` test(03-13): capture the Connections list after the six DR-2-02 profiles were deleted
- Supporting: `5de6fce` build(03): rebuild production bundle with the Phase 3 frontend; `ae54a31` chore: keep agent worktrees out of Railway uploads

Staging deployments, in order: `e0698b29` (first upload, variables absent), `8f6251fe` (refusal fix, variables absent: RUN A, 08, 09), `1a677b8a` (dashboard rebuild from GitHub, crashed), `c394504f` (variables set), `1421a68a` (model-error line), `4ef71070` (gemini-3.8-flash), `3edc4abd` (120s timeout), `2f2fcd71` (gemini-3.5-flash: the decision, 05-07, 10-14, RUN B).

## Checkpoint Status

- **Task 03-13-02 (human-verify):** performed by the orchestrator against live staging with the owner supplying the key. Owner approval of the resulting files is taken through the Phase 3 Evidence Dossier, as in Phases 1 and 2.
- **Task 03-13-03 (human-verify):** performed by the orchestrator against live staging with the owner supplying two session cookies (both expired or rotated since) and signing back in once after a Redis session expiry. Owner approval through the same dossier.

## Security Agenda

`03-SECURITY-AGENDA.md` carries the Phase 2 deferred risks, the CONTEXT and RESEARCH items, and twelve findings from this walkthrough, with nothing decided in advance. DR-2-01 (sign-in code in the log) is closed by plan 03-07 with one residual on the agenda (the dev-mode print when SMTP is wholly unconfigured). DR-2-02 (six throwaway profiles) is closed: screenshot 14.

## Issues Encountered

- Google AI Studio refused an automated key creation as "suspicious"; the owner created the key by hand. The orchestrator never handled the key value.
- The owner's staging session expired once during the key setup (Redis idle expiry); the owner signed back in.
- The Chrome extension disconnected twice; the owner re-opened it once, the second time it reconnected on its own.

## User Setup Required

None beyond what happened during the walkthrough. The staging environment now holds the five `AI_MODEL_*` variables. The production environment holds an unused `GOOGLE_API_KEY` the owner may want to remove.

## Next Phase Readiness

Phase 3's Phase Validation is performed and filed. The verifier, code review and regression gate run next; the Evidence Dossier is built from these files for the owner's acceptance; the security review takes the agenda.

## Self-Check: PASSED

- All ten PNGs exist with the exact names (listed above); each is the page as the owner sees it, no devtools, no terminal window, no raw JSON.
- `03-canned-report.txt` contains `### RUN A` and `### RUN B`; RUN B contains `PASS C2`, `PASS C3`, `PASS C4` and zero lines starting with `FAIL`.
- `04-staging-ai-player.log` contains `cause=engage`, `cause=refused-not-configured`, `stage=request`, `stage=dispatch`, `stage=sent`, `stage=failed`, `event=open`, `event=close` and the migration 011 startup lines; the T-3-07/T-3-02 grep returns 0; the DR-2-01 `code:` grep returns 0.
- The criterion table has four rows, each citing at least one evidence file, report and log citations carry line numbers, each row carries PASS.
- No SQL statement was run and none is cited.
