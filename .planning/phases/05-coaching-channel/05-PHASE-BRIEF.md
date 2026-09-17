# Phase 5 Brief — Coaching Channel

**Status:** Approved by the owner for execution, 2026-09-17 (artifact https://claude.ai/artifact/9ruK4eP1XcrS62dXtzyKsB).

**Purpose:** Owner review before execution. Everything here is lifted from `.planning/ROADMAP.md` §Phase 5, `05-CONTEXT.md` (D-01 to D-29), `05-UI-SPEC.md` and the eleven PLAN.md files; nothing is new. Approve this, and Phase 5 goes to `/gsd-execute-phase 5`.

**Evidence rule (owner-directed, binding):** every success criterion and acceptance criterion is proven by a canned report captured verbatim to a file under `evidence/` (test run, harness run, red-team report, or server log excerpt) or by a screenshot from the end-user perspective. Database queries are not evidence; the roadmap's "can be queried" diagnostic is proven through the Logs page's Coaching Conversation section and the application's own read endpoint. Every new `[AI-PLAYER]` and `[AI-CHATTER]` log line carries ids, stages, outcomes and lengths only, never chat text, a reply, game text, reasoning, a command, the goal, a memory or coaching line, or a key.

---

## Phase Goal

The owner can coach the AI from a chat pane while it plays, see the guidance take effect on the next decision, and pause and resume it. The conversation (AI-chatter) sits above the play loop (AI-player): it chats in plain text and pushes suggestions down mid-stream, and has no other permissions. Making guidance permanent is the owner's own copy into AI settings, explained by a help article. Each panel view can be popped out into its own window (D-29).

Four things are built: the conversation (AI-chatter answers every message with one capped model call, reading everything AI-player knows), coaching (standing suggestions for the login, pushed and withdrawn in words, visible three ways), pause and resume (one button on AI-player, WAITING with a reason, only the owner lifts it), and the screen (one panel, two views, pop-out windows). Folded in from the Phase 4 security review: DR-4-01, DR-4-02, DR-4-03 (as an owner setting), DR-4-04 and the AR-4-08 log-page check.

## Phase Success Criteria (what must be TRUE afterwards)

| # | Criterion | Proof |
|---|-----------|-------|
| 1 | A chat message sent while the AI plays is reflected in its next decision, verifiable from the logged reasoning | `evidence/06-coaching-in-next-decision.png`, `07-chat-reply.png`; `05-staging-ai-player.log` `stage=coaching-pushed`; `01-test-report.txt` PASS `TestHandleChat_PushReachesNextPrompt` |
| 2 | Pause and resume work from the AI-player part of the panel (not chat); pausing stops commands but keeps reading, shows WAITING with its reason, never resumes by itself | `evidence/18-paused-waiting-reason.png`; the quiet stretch between `pause` and `resume-owner` in `05-staging-ai-player.log`; `01-test-report.txt` PASS `TestManager_ResumeRequiresBothReasonsClear`, `TestManager_PauseStopsTheLoopAndKeepsReading`, `TestAutopilotHandler_EmitsPausedAndResumedNotices` |
| 3 | Chat is only ever plain text, AI-chatter changes no configuration; a help article explains the two levels and the by-hand copy | `evidence/14-help-article.png`, `15-guidance-pasted-by-hand.png`; `01-test-report.txt` PASS `TestHandleChat_TouchesNothingElse` |
| 4 | One panel with two views; every pushed suggestion visible (quoted reply, Coaching in effect, stream marker); each view can be popped out and works the same | `evidence/17-panel-two-views.png`, `09-coaching-after-refresh.png`, `10-popouts-both-open.png`, `11-popout-brought-back.png` |

## Plans by Wave (11 plans, 29 tasks, 2 owner checkpoints)

| Wave | Plan | Capability | Depends on |
|------|------|------------|------------|
| 1 | 05-01 | The owner can stop AI-player where it stands, and the stream says so (D-13 to D-16) | — |
| 1 | 05-02 | The safety checker stops blocking ordinary fights and stops failing open (DR-4-01, DR-4-02) | — |
| 1 | 05-03 | The vault key is required everywhere, and the log page stays behind sign-in (DR-4-04, AR-4-08) | — |
| 2 | 05-04 | A real speed limit sits behind AI-player's pacing, and the owner sets it (DR-4-03) | 02 |
| 3 | 05-05 | The owner has someone to ask: AI-chatter answers, and the conversation is kept (migration 015) | 01, 02, 04 |
| 4 | 05-06 | The owner's words change what AI-player does next, and can be taken back (D-28 channel hardening) | 05 |
| 5 | 05-07 | The two levels are on screen: one panel, two views | 04, 06 |
| 5 | 05-10 | One command turns the reachable half of the phase into a canned PASS/FAIL report | 01, 04, 06 |
| 6 | 05-08 | The owner can put either view where he wants it: pop-out windows (D-29) | 07 |
| 6 | 05-09 | The app explains the two levels, and the Logs page keeps the conversation | 06, 07 |
| 7 | 05-11 | The phase is demonstrated on staging and every criterion points at filed evidence, plus the security agenda | all |

## Decisions Honoured

D-01 to D-29 from `05-CONTEXT.md` are each cited in at least one plan (checked by grep). Discretion calls recorded in the plans: two plain yes/no waiting reasons; pause/resume as two more actions on the existing autopilot endpoint; the terminal reuses `[AI-ASSIST waiting]` / `[AI-ASSIST resumed]` and invents no new line; rate limit default 2 per second, range 1 to 20, blank means the server default; one migration, `015_add_coaching_and_conversation`; chat as a new message type on the existing websocket; AI-chatter uses the profile's configured model; chat message cap 1,000 characters; conversation tail of 10 lines in AI-chatter's prompt; coaching capped at 8 lines of 200 characters; withdraw matches exact text then case-insensitive; the two new reads are GET-only; the wire carries unbracketed notice text and the panel adds the brackets; pop-outs 420×800 and 420×480, not remembered across page loads, both-popped-out leaves two placeholders; help article as `help/ai-coaching.json` with no Go change.

## Things the Owner Was Told Before Approving

1. After 05-03 the server refuses to start on any host without `ENCRYPTION_KEY_V1` unless `MUDPUPPY_LOCAL_DEV=true` is set on purpose; never on staging.
2. A reconnect no longer lifts a pause.
3. Every AI-chatter reply counts against the session call cap (D-11).
4. Blank Call Cap means no cap; blank AI Command Rate Limit means the server default of 2 per second.
5. The terminal prints the same `[AI-ASSIST waiting]` line for a pause and a disconnect; the panel shows the reason.
6. Chat and coaching text are not covered by the retention job or Delete Captured Text Now in this phase; no later home is named.
7. A verdict-less checker answer now fails the review, so a flaky vendor shows as failed reviews.
8. The corpus rerun needs a fresh-quota day and can stop the phase before staging if anything is steered or a benign fight is blocked.
9. AI-chatter sees blocked and failed decisions (D-17).
10. A pop-out is a second window onto the same session, not a security boundary.
11. The coaching-channel laundering risk (T-5-26) is handed to the owner open at the review.
12. UI defaults not asked about: plain-line chat, help article after "Safety", both-popped-out leaves two notes, pop-out size and state not remembered.

## What the Owner Supplies

1. The owner's Alter Aeon character on a profile that has accepted the policy, with a session goal saved, and the owner's own hands for the walkthrough.
2. A fresh-quota day for the corpus rerun.
3. A private browser window or separate browser profile for the signed-out log-page screenshot (the owner is never signed out).
4. A second profile that has not accepted the policy, for the hand-play-unchanged screenshot.
5. Confirmation that `ENCRYPTION_KEY_V1` stays set on staging.
6. Accept / Defer / Remediate Now decisions on every agenda item at the security review.

## Security

Fifty-seven numbered threats `T-5-01` to `T-5-57` plus `T-5-SC`, twenty-one rated high; each mitigated by a named task, accepted at plan level, or carried to the review. The security-review agenda (written by plan 05-11, register left untouched) carries DR-4-01 to DR-4-04 with what this phase built against each, AR-4-08 (code half done, `railway logs` access stays the owner's admin task), the new coaching-channel risk T-5-26 with what is not solved, and the plan-level acceptances (T-5-SC, T-5-13, T-5-09, T-5-41) for confirmation; three dispositions each, nothing pre-chosen. Closing notes: policy stays 1.0; whether `ai-player` has been pushed to GitHub at phase close.

## Out of Scope

Promotion (withdrawn, not deferred); per-message actions and a coaching clear button; a Session Memory editor (D-19); `#AUTO PAUSE` or pausing from chat; any write path to coaching or conversation other than chat; purging chat or coaching text; any change to the hand-play limiter or the policy text; a frontend test framework or any new package; debrief, end-of-session evaluation and Historical Memory (Phase 6); study loader (Phase 7); resizable docked chat, speech-bubble chat, remembered pop-out sizes.

## Planning Record

- Context gathered 2026-09-17 (commit 625d2b2); D-29 and four owner confirmations added the same day during the UI design contract.
- UI-SPEC verified 6/6 twice (commits ee9a219, b2b6f0f); research and validation strategy (commit 9e3f561).
- Plans created with the pattern map (commit 237c95a). Plan checker: first pass 2 blockers and 4 warnings (pause/resume stream notices undelivered; AI-chatter not shown the coaching in effect; send-failed notice unwired; file counts; research questions unmarked); one targeted revision closed all six and caught a doubled-bracket defect; second pass clean.
- Coverage: all four REQ ids and all 29 decisions in plans; `05-VALIDATION.md` filled per task, `nyquist_compliant: true`.

---

## Approval

Approved 2026-09-17. The owner approved the brief as published, with no changes, in chat at the end of `/gsd-plan-phase 5` ("Approved").
