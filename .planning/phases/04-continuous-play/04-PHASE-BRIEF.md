# Phase 4 Brief — Continuous Play

**Status:** Pending owner approval for execution (artifact https://claude.ai/artifact/Su5CixwfRAJMNC1AkEts7t).

**Purpose:** Owner review before execution. Everything here is lifted from `.planning/ROADMAP.md` §Phase 4, `04-CONTEXT.md` and the eleven PLAN.md files; nothing is new. Approve this, and Phase 4 goes to `/gsd-execute-phase 4`.

**Evidence rule (owner-directed, binding):** every success criterion and acceptance criterion is proven by a canned report captured verbatim to a file under `evidence/` (test run, harness run, red-team report, or server log excerpt) or by a screenshot from the end-user perspective. Database queries are not evidence. Every new `[AI-PLAYER]` log line carries ids, stages, counts and lengths only, never game text, the goal, a memory bullet, a command or a key.

---

## Phase Goal

The owner sets a session goal and the AI plays toward it continuously until stopped, within safety limits that are proven under test; re-engaging after manual driving starts from the current situation, not a stale plan.

Three things are built: the loop (Phase 3's single decision becomes the first tick of a paced, cancellable server-side loop), the limits (call cap, failure threshold, block counter, the single 503 retry, badge from the disengage message), and the first three layers of the owner's memory model (time-bounded Immediate Context, curated Session Memory shown read-only, Quest Memory tied to the goal and persisting across sessions). Folded in from the Phase 3 and 3.1 reviews: the retention standard with a delete-now action, the badge fix, the 503 retry, the reviewer's untrusted-reasoning fix, the credential-vault key requirement, the migration 012 rollback fix, and the dead statement.

## Phase Success Criteria (what must be TRUE afterwards)

| # | Criterion | Proof |
|---|-----------|-------|
| 1 | The owner sets a session goal, and the loop runs read, decide, act continuously against a live game toward it, paced to the game's turn rhythm, every decision visible with its reasoning as it happens | `evidence/06-goal-set.png`, `07-paced-decisions.png`; `01-test-report.txt` PASS `TestLoop_Pacing`; `05-staging-ai-player.log` `stage=loop-tick` |
| 2 | The session call cap halts the loop with a visible notice; repeated errors or malformed output disengage with a visible notice | `evidence/08-cap-halt-badge-off.png`; `01-test-report.txt` PASS `TestLoop_CallCap`, `TestLoop_ErrorThreshold`, `TestLoop_ConsecutiveBlocks`; `05-staging-ai-player.log` `stage=cap` |
| 3 | Typing any game command instantly disengages and goes through; re-engagement demonstrably reassesses rather than resuming a stale plan | `evidence/09-wheel-grab-reengage-reasoning.png`; `01-test-report.txt` PASS `TestEngageLoop_Reassess`, `TestLoop_StopsWhenWheelGrabbed`; the wheel-grab transition in `05-staging-ai-player.log` |
| 4 | All mechanical safety limits hold under automated test (cap when set, no AI reconnect, disengage on repeated errors, nothing while disconnected with resume on return, blank settings safe) | `01-test-report.txt` PASS `TestLoop_CallCap`, `TestLoop_NoAIReconnect`, `TestLoop_ErrorThreshold`, `TestLoop_NothingIssuedWhileWaiting`, `TestManager_LoopStopsOnPark`, `TestLoop_BlankSettings`, with `-race`; `evidence/13-hand-play-unchanged.png` |

## Plans by Wave (11 plans, 32 tasks, 2 owner checkpoints)

| Wave | Plan | Capability | Depends on |
|------|------|------------|------------|
| 1 | 04-01 | A multi-decision stint can be driven and observed in tests, with no network | — |
| 1 | 04-02 | The server refuses to start without the credential-vault key, and migration 012 rolls back cleanly (DR-3.1-04, DR-3.1-05) | — |
| 2 | 04-03 | The AI keeps playing on its own, paced to the game's output, and stops the instant the wheel is grabbed (DR-3-07) | 01 |
| 3 | 04-04 | The mechanical limits stop the loop and the owner sees why (DR-3.1-02, DR-3-04, DR-3-03) | 03 |
| 3 | 04-05 | The AI reads only what just happened, and never an empty screen | 03 |
| 4 | 04-06 | The owner sets a session goal, and it names the active Quest (migration 013) | 04 |
| 5 | 04-07 | Every prompt carries the goal, the Quest's bullets and Session Memory, all framed as untrusted (DR-3.1-01) | 06 |
| 6 | 04-08 | The AI curates its Session Memory and Quest Memory as it plays, and the owner can watch it | 07 |
| 7 | 04-09 | Captured game text ages out on a schedule, and the owner can delete it now (DR-3-01, DR-3.1-03) | 08 |
| 8 | 04-10 | One command turns the Phase 4 HTTP surface into a canned PASS/FAIL report per criterion | 06, 08, 09 |
| 9 | 04-11 | The phase is demonstrated on staging and every criterion points at a filed piece of evidence, plus the security agenda (D-28) | all |

## Decisions Honoured

D-01 to D-28 from `04-CONTEXT.md` are each cited in at least one plan (checked by grep). Discretion calls recorded in the plans: pacing 1.5 s settle / 20 s floor / 3 s minimum spacing as injectable struct fields; window about 10 s with a 2 KB never-empty floor and the 8 KB ceiling; memory as two flat string arrays rewritten in full by the model, capped at 30 and 20 bullets of 200 characters in Go; goal as a 1,000-character column on `profiles`, Session Memory as JSONB on `game_sessions`, Quest Memory in a `quests` table with a partial unique index for reactivation; goal box saves on blur or Enter; cap and block halts stored as failed rows with new failure kinds; `icm-refused` transient; 503 retry after 2 s; retention as a 24-hour in-process ticker; no memory-updated system line.

## What the Owner Supplies

1. A session goal actually worth pursuing on the Alter Aeon character (checkpoint 1).
2. A small call cap for one test (checkpoint 2), cleared afterwards.
3. The five `AI_MODEL_*` variables on the dev machine for the corpus rerun on a fresh-quota day (about seventy calls on `gemini-3.5-flash-lite`).
4. Confirmation that `ENCRYPTION_KEY_V1` stays set on staging (the server will refuse to start on Railway without it after plan 04-02).
5. The owner's character on Alter Aeon for the walkthrough; socket drops are expected and recorded.

## Security

Thirty-four numbered threats `T-4-01` to `T-4-34` plus `T-4-SC`, ids phase-wide. High-severity items and their mitigating tasks are in the plans' `<threat_model>` blocks. The security-review agenda (written by plan 04-11) carries the eleven deferred Phase 3 and 3.1 risks with what this phase did about each, the one new active risk (the AI's own memory as an injection channel, T-4-03), and the plan-level acceptances for confirmation; three dispositions each, nothing pre-chosen. Closing notes: policy stays 1.0 (D-22); `GOOGLE_API_KEY` on production is a cut-over note (AR-3-08).

## Out of Scope

Coaching, chat pause/resume and Quest Memory recall (Phase 5); hand-editing Session Memory (Phase 5); end-of-session evaluation, quest closure, Historical Memory (Phase 6, roadmap wording to be amended first); study loader (Phase 7); any policy text change; per-profile retention settings; telnet negotiation work for the Alter Aeon drop; fake game or diagnostic endpoint; backlog defects WR-03, IN-02 and the stale worktree directory.

## Planning Record

- Context gathered 2026-09-16 (commit ecfa0b3).
- UI-SPEC verified 6/6 with two non-blocking notes (commit b8c6f2a); research and validation strategy (commit e76e762); pattern map (commit baf86b0).
- Plans created (commit 3713a1b). Plan checker: passed first time, no blockers, three INFO notes (research open questions marked resolved for planning; PATTERNS.md omits `internal/store/retention.go`, whose analog plan 04-09 cites itself; plans 04-04, 04-06 and 04-08 touch 10 to 14 files each).
- Coverage: all six REQ ids and all 28 decisions in plans; `04-VALIDATION.md` filled with 31 task rows, `nyquist_compliant: true`.

---

## Approval

Pending. The owner approves in chat against the published brief artifact; this section is then updated and committed before `/gsd-execute-phase 4`.
