# Phase 2 Brief — Autopilot Switch

**Status:** Awaiting owner approval for execution (planned 2026-09-15).

**Purpose:** Owner review before execution. Everything here is lifted from `.planning/ROADMAP.md` §Phase 2 and the seven PLAN.md files; nothing is new. Approve this, and Phase 2 goes to `/gsd-execute-phase 2`.

**Evidence rule (owner-directed, binding for this and every later phase):** every success criterion and acceptance criterion is proven by one of two artifact types, and nothing else.

1. **A canned report.** A repeatable script or test run whose output is captured verbatim to a file under `evidence/`. Server log excerpts captured to a file count here.
2. **A screenshot from the end-user perspective.** The MUDPuppy browser page as the owner sees it. No devtools pane, no terminal, no raw JSON in frame.

Database queries and inspections are not evidence anywhere in this phase.

---

## Phase Goal

The owner can engage and disengage autopilot from the terminal and always see the true state; taking the wheel is instant and lossless; a disconnect never turns autopilot off and never issues commands, only `#AUTO OFF` turns it off. No AI intelligence exists yet.

## Phase Success Criteria (what must be TRUE afterwards)

| # | Criterion | Proof (canned report or end-user screenshot) |
|---|-----------|----------------------------------------------|
| 1 | `#AUTO ON` engages only when the profile passes the Phase 1 gate (otherwise refused with the gate's message); `#AUTO OFF` disengages; the play-screen indicator always matches the true server-held state, including after a page refresh. | `evidence/01-test-report.txt` PASS lines for `TestEngageAutopilot`, `TestAutopilotHandler` and `TestStatusCarriesAutopilotState`; `evidence/03-canned-report.txt` `PASS C1` lines with the refusal body printed and the status endpoint agreeing with the autopilot endpoint; screenshots `evidence/06-refused-before-acceptance.png` and `evidence/05-badge-after-refresh.png`. |
| 2 | Any game command typed by the human while engaged disengages autopilot before the command is sent, with no lost keystrokes; commands fired by browser aliases, triggers, or timers do not trip the wheel-grab. | `evidence/01-test-report.txt` PASS lines for `TestWheelGrabSourceRule`; screenshots `evidence/07-wheel-grab.png` and `evidence/08-trigger-keeps-on.png`; `evidence/04-staging-ai-player.log` `cause=wheel-grab` line for the typed command and no such line for the trigger-fired one. |
| 3 | A disconnect while engaged moves autopilot to a waiting state that issues nothing; the AI never initiates a reconnect; when the connection comes back by any means autopilot resumes to ON by itself; `#AUTO OFF` while waiting lands on OFF and it stays OFF after the connection returns. (Owner amendment 2026-09-15.) | `evidence/01-test-report.txt` PASS lines for `TestWaitingSurvivesDisconnectAndResumesOnConnect`; screenshots `evidence/09-waiting-after-drop.png`, `evidence/10-resume-after-reconnect.png` and `evidence/11-off-while-waiting-stays-off.png`; `evidence/04-staging-ai-player.log` `cause=disconnect` then `cause=resume`. |
| 4 | The owner can create a game profile, accept the policy on it, and hand-play the character normally with the AI disengaged; nothing about ordinary play changes. | `evidence/03-canned-report.txt` `PASS C4` lines; screenshot `evidence/12-hand-play-unchanged.png`. |

**Phase Validation line (from ROADMAP):** Player-observable on staging against a live MUD: `#AUTO ON` is refused before acceptance and engages after it; the indicator matches the server state after a page refresh; typing a command while engaged shows the disengage notice and the command's game response; a trigger-fired command leaves autopilot engaged; dropping the connection shows the indicator as waiting and no command is sent while disconnected; reconnecting by hand shows it resuming to ON without typing `#AUTO ON`; `#AUTO OFF` while waiting lands on OFF and it stays OFF after reconnecting. Diagnostic: `go test` covers the engaged-state machine and the human-versus-automation source flag.

Two criteria cannot be reached over HTTP. The canned report prints `SKIP C2` and `SKIP C3` naming the screenshots and the log excerpt that prove them instead, and never reports them as PASS.

---

## Breakdown by Success Criterion (review view)

Each phase criterion, then every plan acceptance criterion that contributes to it, then the artifact that proves it.

### Criterion 1 — `#AUTO ON` only past the gate, `#AUTO OFF` disengages, the indicator always matches the server

| Plan | Plan acceptance criterion | Proof |
|------|---------------------------|-------|
| 02-01 | Autopilot reads on, off or waiting, and only Engage reaches on while only Disengage reaches off | `01-test-report.txt`: PASS `TestAutopilotTransitions`, one subtest line per transition pair |
| 02-01 | A second Engage while already on changes nothing: not the state, not the stored connection id, not the waiting timestamp | `01-test-report.txt`: PASS `repeat_engage_is_a_no_op` |
| 02-01 | Engaging without a connected game session is refused and the switch stays off | `01-test-report.txt`: PASS `refused_without_connected_session` |
| 02-02 | Engaging on a profile that has not accepted the policy is refused with Phase 1's exact sentence and the switch stays off | `03-canned-report.txt`: pre-acceptance `PASS C1` with the refusal body printed character for character; `01-test-report.txt`: PASS `TestAutopilotHandler` |
| 02-02 | Engaging with the gate passed but no connected game is refused with the no-connection outcome | `03-canned-report.txt`: `refused-no-session` step `PASS C1` |
| 02-02 | `#AUTO OFF` while already off answers already-off and changes nothing; turning off is always permitted, even while the gate refuses | `03-canned-report.txt`: `PASS C1` for that step; `01-test-report.txt`: PASS `off_is_allowed_even_when_gate_refuses` |
| 02-02 | `GET /api/v1/session/status` carries the server-held autopilot state on every response | `03-canned-report.txt`: paired `PASS C1` lines comparing the two endpoints; `01-test-report.txt`: PASS `TestStatusCarriesAutopilotState` |
| 02-02 | An unknown action is rejected with 400 rather than silently interpreted | `03-canned-report.txt`: invalid-action step `PASS C1` |
| 02-04 | `#AUTO ON`, `#AUTO OFF`, `#AUTO STATUS` and a bare `#AUTO` parse in the same directive grammar as `#ECHO` and `#HELP`, and each prints exactly the UI-SPEC line | `06-refused-before-acceptance.png` and `05-badge-after-refresh.png` |
| 02-04 | A refused `#AUTO ON` prints the engage gate's own sentence, bracket-wrapped, and the switch stays off | `06-refused-before-acceptance.png` |
| 02-04 | `#AUTO` with no argument prints one line carrying the state and the engage-gate result | `05-badge-after-refresh.png` showing the status line beside the badge |
| 02-06 | The badge reads `Autopilot: On`, `Autopilot: Waiting` or `Autopilot: Off`, and reads the same after a hard refresh with no action taken | `05-badge-after-refresh.png` |
| 02-06 | The badge renders the server-held value and infers nothing locally | `05-badge-after-refresh.png` and `09-waiting-after-drop.png` |
| 02-05 | The harness prints PASS, FAIL or SKIP per ROADMAP criterion with the response body under every step, and carries the full `go test` output and its exit status | `03-canned-report.txt`: `PASS C1`, `GO TEST EXIT` line |

### Criterion 2 — Typing takes the wheel with no lost keystrokes; triggers and timers do not

| Plan | Plan acceptance criterion | Proof |
|------|---------------------------|-------|
| 02-03 | A command the owner typed disengages autopilot before the command leaves for the game, and the command still goes through | `01-test-report.txt`: PASS `human_source_disengages`; `07-wheel-grab.png` showing the notice, the command and the game's reply in one frame |
| 02-03 | A command fired by a browser trigger or timer is sent with autopilot still on | `01-test-report.txt`: PASS `trigger_source_leaves_it_on`; `08-trigger-keeps-on.png` |
| 02-03 | A source flag that is missing, empty or unrecognised is treated as human | `01-test-report.txt`: PASS `blank_source_disengages` and the garbled-source subtest |
| 02-03 | The source flag can only keep an already-on autopilot on; it can never engage one | `01-test-report.txt`: PASS `off_stays_off` |
| 02-03 | The wheel-grab acts only on the autopilot of the authenticated user of that websocket | `01-test-report.txt`: PASS `second_user_untouched` |
| 02-03 | No keystroke is lost: the command is queued on every path | `07-wheel-grab.png` showing the game's reply after the notice |
| 02-04 | Every command the browser sends states whether a person typed it or automation fired it | `07-wheel-grab.png` and `08-trigger-keeps-on.png` differ only because of the source |
| 02-04 | A `#` line never disengages autopilot, because it produces no game command and is never sent to the MUD | `08-trigger-keeps-on.png`; `01-test-report.txt` `npm run build` section exits 0 |
| 02-06 | A blank Enter is labelled human, so pressing Enter on an empty line takes the wheel | `07-wheel-grab.png`, blank-Enter step |
| 02-06 | Typing while engaged prints `[Autopilot disengaged: you took the wheel]` and the game's reply follows | `07-wheel-grab.png` |
| 02-07 | Every wheel-grab is one `[AI-PLAYER] autopilot` line with `cause=wheel-grab` carrying no command text | `04-staging-ai-player.log` |

### Criterion 3 — A drop parks it at WAITING, the connection returning resumes it, `#AUTO OFF` while waiting sticks

| Plan | Plan acceptance criterion | Proof |
|------|---------------------------|-------|
| 02-01 | A disconnect while on lands on waiting and survives the deletion of the session map entry, driven through the real `Manager.Disconnect` | `01-test-report.txt`: PASS `TestWaitingSurvivesDisconnectAndResumesOnConnect` |
| 02-01 | A connection returning while waiting resumes to on with no owner action | `01-test-report.txt`: same test, resume subtest |
| 02-01 | `#AUTO OFF` while waiting lands on off and stays off after the connection returns | `01-test-report.txt`: PASS `off_while_waiting_stays_off_after_reconnect` |
| 02-01 | A disconnect while off changes nothing | `01-test-report.txt`: PASS `disconnect_while_off_changes_nothing` |
| 02-01 | The whole `internal/session` package passes under `-race` | `01-test-report.txt`: `go test ./internal/session/... -race -v` section exits 0 |
| 02-06 | A drop while engaged prints `[Disconnected]` then `[Autopilot waiting for reconnect]` on the next line and the badge reads Waiting | `09-waiting-after-drop.png` |
| 02-06 | The connection returning while waiting prints `[Reconnected]` then `[Autopilot resuming]` and the badge reads On without the owner typing anything | `10-resume-after-reconnect.png` |
| 02-07 | The staging log shows the parking and the resume as transitions, with no game text on the line | `04-staging-ai-player.log`: `cause=disconnect` and `cause=resume` |
| 02-07 | `#AUTO OFF` while waiting still reads Off after the connection is back | `11-off-while-waiting-stays-off.png` |

### Criterion 4 — Create a profile, accept the policy, hand-play normally with nothing changed

| Plan | Plan acceptance criterion | Proof |
|------|---------------------------|-------|
| 02-02 | A request aimed at a connection the caller does not own is refused by the same gate, because the profile is resolved by owner and connection together | `03-canned-report.txt`: not-owned-connection `PASS C4` |
| 02-02 | The gate refusal and the acceptance path behave on the deployed server exactly as Phase 1 defined them | `03-canned-report.txt`: `PASS C4` lines; `04-staging-ai-player.log`: `cause=refused-gate` |
| 02-06 | With autopilot off, ordinary play is untouched: no new line of any kind, no banner, no new control, and the only visible difference is a badge reading `Autopilot: Off` | `12-hand-play-unchanged.png` |
| 02-05 | No criterion in the report is established by a database query | `03-canned-report.txt` and `02-harness-selftest.txt` contain no SQL |

Plan 02-07 adds no capability. It runs the phase on staging, files the evidence above, writes `02-07-SUMMARY.md` with one PASS/FAIL row per criterion citing these files by name and line, and writes the security-review agenda.

---

## Plans by wave

Waves are dependency order only. Plans in the same wave touch no common files.

| Wave | Plan | Capability | Depends on | What the owner will be able to see afterwards |
|------|------|------------|------------|-----------------------------------------------|
| 1 | 02-01 — Autopilot holds a position on the server and a dropped connection parks it instead of flipping it off | A three-value state machine (on, off, waiting) held on `Manager` in a map of its own, with the disconnect path parking an engaged switch and the connect path resuming it, one `[AI-PLAYER]` line per call | Phase 1 | Nothing in the browser yet. The test report shows the switch surviving a real disconnect and coming back on its own. |
| 2 | 02-02 — The switch answers over HTTP and refuses without the policy gate or a connected game | One endpoint for on, off and status, behind the Phase 1 engage gate, scoped to the caller's own connection, with the autopilot state added to the 15-second status response | 02-01 | The canned report shows the refusal before acceptance, the no-connection refusal after it, and the status endpoint stating the switch position. |
| 2 | 02-03 — Typing takes the wheel; triggers and timers do not | The wheel-grab enforced server-side at the single point where a browser command crosses into the game, reading a source flag that fails safe to human | 02-01 | Nothing in the browser yet. The test report shows a typed command disengaging and a trigger-fired one not. |
| 3 | 02-04 — `#AUTO` is a directive the owner can type, and the browser labels every command human or automation | `#AUTO ON`, `#AUTO OFF`, `#AUTO STATUS` and a bare `#AUTO` in the same `#` grammar as `#ECHO` and `#HELP`, and the source the engine already assigns reaching the websocket | 02-02, 02-03 | The owner can type `#AUTO ON` and see it refused or accepted, and type `#AUTO` to read the state and the gate result on one line. |
| 3 | 02-05 — One command turns the Phase 2 HTTP sequence into a canned PASS/FAIL report per success criterion | `scripts/verify-phase2.sh` plus fixtures, with a self-test and a negative self-test so the report can be trusted | 02-02, 02-03 | A report file the owner can open, with a verdict per criterion and the response body under each step. |
| 4 | 02-06 — The badge and the notices tell the truth, including after a refresh | A header badge beside the connection badge reading On, Waiting or Off from the server, and one bracketed terminal line per transition through the local-echo path | 02-04 | The badge, the wheel-grab notice, the waiting and resuming lines, and ordinary play with nothing added when the switch is off. |
| 5 | 02-07 — Phase 2 demonstrated on staging and filed as evidence | The ROADMAP Phase Validation line performed against Railway staging, twelve evidence files on disk, a criterion table with verdicts, and the security-review agenda | 02-01 to 02-06 | The walkthrough itself, and afterwards a folder of reports, a log excerpt and eight screenshots. Two tasks are blocking human checkpoints. |

---

## Decisions honoured

- **D-01:** `#AUTO ON` is the only way into ON and `#AUTO OFF` the only way into OFF. A dropped connection moves ON to WAITING; the connection returning moves WAITING back to ON with no owner action. `#AUTO OFF` while WAITING lands on OFF and it stays OFF after reconnecting.
- **D-02:** `#AUTO ON` engages only when the profile passes the Phase 1 engage gate. A refusal prints the gate's own message and the switch stays OFF.
- **D-03:** `#AUTO ON` with no connected game is refused with `[Autopilot needs a connected game; connect first]` and the switch stays OFF. There is no arming before connecting.
- **D-04:** `#AUTO ON` while already ON and `#AUTO OFF` while already OFF are no-ops that print `[Autopilot is already on]` and `[Autopilot is already off]`. A repeated `#AUTO ON` never restarts or resets anything.
- **D-05:** `#AUTO` with no argument, and `#AUTO STATUS`, print one line: the current state and the engage-gate result for the current profile.
- **D-06:** The state is held server-side per user session and is the single truth. A server restart loses it and lands on OFF, which is the safe direction.
- **D-07:** Any line the owner types that is sent to the game takes the wheel, including alias expansions of a typed command, command-history recall, and a blank Enter. The owner's words: "A blank line is a Client command and should be considered human interaction and AI would disengage." Disengage happens before the line is sent and no keystroke is lost.
- **D-08:** Lines beginning with `#` are client-internal and never disengage autopilot. The owner's words: "# commands should not interrupt the AI; those internal commands should be ignored." Only `#AUTO OFF` turns autopilot off.
- **D-09:** Commands fired by browser triggers and timers pass through with autopilot still ON. The websocket `data` message gains a source flag so the server applies the wheel-grab only to human-sourced input.
- **D-10:** The indicator is a header badge beside the existing connection badge, reading ON, WAITING or OFF, driven by the server so it re-syncs after a page refresh and never shows a state the server does not hold.
- **D-11:** Every state change also prints one short bracketed line in the terminal through the local-echo path, never sent to the game: `[Autopilot engaged]`, `[Autopilot disengaged]`, `[Autopilot disengaged: you took the wheel]`, the gate's refusal message, `[Disconnected]` then `[Autopilot waiting for reconnect]`, and `[Reconnected]` then `[Autopilot resuming]`.
- **D-12:** Ordinary play with autopilot OFF is unchanged. No new lines, no new behaviour, no banner, no tooltip. The badge simply reads `Autopilot: Off`.

---

## What the owner will do during the walkthrough

All of this is on Railway staging, in the browser, signed in as the owner, against a live MUD. No terminal is needed. The staging session is never signed out, because signing back in is the owner's own step. Each step produces one screenshot; the file names are fixed because the plans cite them.

1. On a connection profile that has never accepted the policy, connect to the MUD and type `#AUTO ON`. The terminal prints the engage gate's refusal sentence in brackets and the header badge still reads `Autopilot: Off`. Screenshot: `evidence/06-refused-before-acceptance.png`.
2. Open AI Player settings for that profile, accept the policy, return to the play screen and type `#AUTO ON`. `[Autopilot engaged]` appears and the badge reads `Autopilot: On`. Hard-refresh the page and do nothing else. The badge still reads `Autopilot: On`, with the `#AUTO` status line visible in the terminal. Screenshot: `evidence/05-badge-after-refresh.png`.
3. While engaged, type a harmless game command such as `look`. The terminal shows `[Autopilot disengaged: you took the wheel]`, the command, the game's reply beneath it, and the badge now reading `Autopilot: Off`. Then engage again and press Enter on an empty line; that disengages too. Screenshot: `evidence/07-wheel-grab.png`.
4. Create a browser trigger that sends a command on a line the MUD reliably prints. Engage autopilot and let the trigger fire. The trigger's command and the game's reply appear and the badge still reads `Autopilot: On`. Screenshot: `evidence/08-trigger-keeps-on.png`.
5. With autopilot on, drop the connection. The terminal shows `[Disconnected]` and, on the next line, `[Autopilot waiting for reconnect]`. The badge reads `Autopilot: Waiting` and no command is sent while disconnected. Screenshot: `evidence/09-waiting-after-drop.png`.
6. Press Connect to reconnect by hand. The terminal shows `[Reconnected]` and, on the next line, `[Autopilot resuming]`. The badge reads `Autopilot: On` and `#AUTO ON` was never typed. Screenshot: `evidence/10-resume-after-reconnect.png`.
7. Engage again, drop the connection again, and while the badge reads Waiting type `#AUTO OFF`. `[Autopilot disengaged]` appears and the badge reads Off. Reconnect by hand. The badge still reads `Autopilot: Off` after the connection is back. Screenshot: `evidence/11-off-while-waiting-stays-off.png`.
8. Create a fresh profile, accept the policy on it, connect, and hand-play several commands with autopilot off. Ordinary play looks as it does today: no bracketed autopilot line anywhere, no banner, no new control, and a badge reading `Autopilot: Off`. Screenshot: `evidence/12-hand-play-unchanged.png`.

Around the walkthrough, the executor deploys the branch to staging, runs `scripts/verify-phase2.sh` against it into `evidence/03-canned-report.txt`, and captures the `[AI-PLAYER]` lines the run and the walkthrough produced into `evidence/04-staging-ai-player.log`. The local test report and the harness self-test are captured first, into `evidence/01-test-report.txt` and `evidence/02-harness-selftest.txt`. The owner confirms both checkpoints.

---

## Security

### Threat register summary

Twenty-one threats were identified across the seven plans. Each is mitigated by a named task, accepted at plan level, or deferred to the phase-close security review.

| Threat | What it is | Handled by |
|--------|------------|------------|
| T-2-01 | Spoofing or tampering with the client-supplied source flag on `data` messages, to bypass the wheel-grab | 02-03: only `trigger` and `timer` count as automation, everything else is human; the flag can never engage or resume, only keep an already-on switch on. Confirmed end to end by 02-07. |
| T-2-02 | Elevation of privilege: aiming the switch at another user's connection | 02-02: the user id comes only from the request context and the profile is resolved by owner and connection together. 02-03: the wheel-grab uses the websocket's authenticated user. 02-05: the harness asserts the refusal. |
| T-2-03 | Information disclosure through the autopilot state on the status response and the websocket push | 02-02 and 02-03: both derived from the authenticated user; the push goes only to the connection it answers. Residual accepted in 02-04. |
| T-2-04 | Information disclosure through `[AI-PLAYER]` log lines | 02-01, 02-02 and 02-03: the fixed format carries ids, states and a cause only. 02-07 greps the filed excerpt for game text and profile text. |
| T-2-05 | Tampering: a repeated engage resetting something | 02-01: a second engage leaves the record byte-identical. |
| T-2-06 | Spoofing: a badge that shows a state the server does not hold | 02-06: the badge reads only the server value, holds no local state and polls nothing. 02-07: the refresh screenshot is taken with no action after the reload. |
| T-2-07 | Repudiation: a canned report that always passes | 02-05 and 02-07: the negative self-test proves the harness reports `FAIL C1` and exits non-zero before the staging report is trusted. |
| T-2-08 | Policy risk: WAITING returns to ON by itself, with no time bound | Deferred to the security review. 02-01 records `WaitingSince` so a bound can be added later without redesign. On the agenda as item 1. |
| T-2-09 | Deadlock or data race on the new map | 02-01: all access through methods that take the lock, no reentrant `GetSession` call, package run under `-race`. |
| T-2-10 | Spoofing: a nil engage-gate callback failing open | 02-02: a nil callback is a refusal, proven by a subtest. |
| T-2-11 | Tampering: an unvalidated `action` field | 02-02: allowlisted to `on`, `off`, `status`, otherwise 400 with no silent fallback. |
| T-2-12 | Denial of service: a lost keystroke on the command path | 02-03: the channel send stays unchanged and unconditional and no new blocking call sits on the path. |
| T-2-13 | Tampering: a typed directive cancelling autopilot through the commands it emits | 02-04: `#AUTO` emits no game command, and commands a typed directive does emit are labelled automation. |
| T-2-14 | Spoofing: the refusal sentence drifting from Phase 1's wording | 02-04: the frontend renders the server's message and holds no copy of the sentence. |
| T-2-15 | Repudiation: a SKIP read as a PASS | 02-05: C2 and C3 print as `SKIP` with their own counter and each names the screenshot that proves it. |
| T-2-16 | Information disclosure through the report contents | 02-05: the session cookie is never echoed and no game or profile text is printed. |
| T-2-17 | Tampering: a run or a deploy aimed at production | 02-05 and 02-07: staging only, with `BASE_URL` and the git SHA recorded in the report header. |
| T-2-18 | Repudiation: a notice printed twice or not at all | 02-06: one printing owner per transition and a first-render guard. |
| T-2-19 | Information disclosure: a notice reaching the game | 02-06: every line goes through `echoLocal`, the path that already keeps `[Disconnected]` off the wire. |
| T-2-20 | Repudiation: the carried-forward Phase 1 risks being forgotten | 02-07: both are written onto the agenda with their original remediation wording. |
| T-2-SC | Supply chain: an unplanned dependency | Accepted in every plan. This phase installs nothing, and the `### DEPENDENCY DRIFT` section of `evidence/01-test-report.txt` must be empty. |

### Security-review agenda (decided by the owner at phase close)

Plan 02-07 writes `02-SECURITY-AGENDA.md` as the input to this review, not its output. Three items are on it. Each carries the same three dispositions, and **no disposition is pre-chosen and none is recommended**. The choice is the owner's at the review.

| # | Item | Accept | Defer | Remediate Now |
|---|------|--------|-------|---------------|
| 1 | **WAITING to ON automatic resume versus policy section 2.** An autopilot parked by a dropped connection returns to ON by itself when the connection comes back, with no owner action and no time bound. Policy section 2 addresses the owner re-engaging after a disconnection is understood; the system now does something adjacent automatically. Phase 2 deliberately built no time bound. The candidate remediation, should Remediate Now be chosen, is a bounded waiting lifetime after which the switch lands on OFF, with the bound itself left to the owner. Cross-references T-2-08. | open | open | open |
| 2 | **Carried forward R-02.** Invalid numeric input in the AI Player panel saves as blank rather than raising a validation error. Phase 2 did not touch that file and built no fix. The agenda reproduces the proposed remediation from the pending todo verbatim. | open | open | open |
| 3 | **Carried forward R-04.** The staging deploy log prints the one-time sign-in code, so anyone with staging log access can sign in as any staging user. Phase 2's own evidence capture reads the staging log, which is exactly the access path the risk describes. The agenda reproduces the proposed remediation verbatim. | open | open | open |

The project cannot close while any item on this agenda remains deferred. If the owner chooses Remediate Now on any item, that becomes its own plan or its own phase; nothing is remediated inside Phase 2.

The agenda also lists, for confirmation rather than re-litigation, the threats accepted at plan level: T-2-SC (no packages installed, both manifests unchanged) and the residual on T-2-03 from plan 02-04. It closes by noting that Phase 3 must re-confirm T-2-01, because from Phase 3 an engaged autopilot starts issuing real commands and the consequence of a forged source flag changes.

---

## Out of scope / deferred

- **No AI intelligence.** Nothing is driven while the switch reads ON. No model call, no decision, no reasoning, no session log. That is Phase 3.
- **No reconnect toggle.** The AI never opens a connection and there is no setting that would let it. Reconnecting is the owner's action, by any means they choose.
- **No bounded WAITING lifetime.** WAITING has no time limit in this phase, by design. A bound is built only if the owner chooses Remediate Now on agenda item 1. `WaitingSince` is recorded so that choice stays cheap.
- **Shelf copy of the design not yet updated.** `D:\Projects\ai-mud-player\documents\ai-game-player-design-v3.md` still holds the pre-amendment text. The repo copy carries the 2026-09-15 WAITING amendment. The shelf copy is updated only when the owner says the amended version is approved.

---

Approve this, and Phase 2 goes to `/gsd-execute-phase 2`.
