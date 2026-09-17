---
phase: 04-continuous-play
fixed_at: 2026-09-17T13:34:38Z
review_path: .planning/phases/04-continuous-play/04-REVIEW.md
iteration: 1
findings_in_scope: 14
fixed: 14
skipped: 0
status: all_fixed
---

# Phase 4: Code Review Fix Report

**Fixed at:** 2026-09-17T13:34:38Z
**Source review:** `.planning/phases/04-continuous-play/04-REVIEW.md`
**Iteration:** 1
**Base:** started from `6849195` on `ai-player`; rebased onto `209d293` before landing (see next paragraph)

**The branch moved while this pass ran.** The pass was told nothing else was writing to the tree. One commit landed on `ai-player` meanwhile: `209d293 fix(store): close game sessions a previous process left open, at server start`. The fast-forward was therefore refused, and nothing was forced. The fix commits lived on an isolated temp branch in a separate worktree, so they were rebased there onto `209d293` (16 of 16 clean, no conflicts; the one shared file, `cmd/server/main.go`, was touched 130 lines apart). Every gate below was then run AGAIN on the combined tree before the fast-forward, because the two changes had never been built or tested together. The hashes in this report are the rebased ones. `209d293` itself is untouched.

**Summary:**
- Findings in scope: 14 (2 Critical, 12 Warning). Info findings were out of scope unless a one-line fix sat inside code already being changed.
- Fixed: 14
- Skipped: 0
- Commits: 15 (one per finding, plus one test-only follow-up to WR-02 found by the flake gate)
- No migration, no new dependency (`go.mod`, `go.sum`, `frontend/package.json`, `frontend/package-lock.json` unchanged), no corpus item, target, window or fixture touched, no harness fixture touched.

Every finding below is a logic change, not a syntax change. Syntax and the test suite prove the code does what the tests say; they do not prove the tests say the right thing. **All 14 are marked "fixed: requires human verification"** for that reason, and the ones that change what the owner sees or what staging does are called out under "To re-verify on staging".

## Finding → commit → what changed → test

| Id | Commit | Status |
|----|--------|--------|
| CR-01 | `18f5562` | fixed: requires human verification |
| CR-02 | `d5cfd27` | fixed: requires human verification |
| WR-01 | `b1f4132` | fixed: requires human verification |
| WR-02 | `0a299d8`, follow-up `63395aa` | fixed: requires human verification |
| WR-03 | `1ed580c` | fixed: requires human verification |
| WR-04 | `7d96d6d` | fixed: requires human verification |
| WR-05 | `005b9ee` | fixed: requires human verification |
| WR-06 | `9199696` | fixed: requires human verification |
| WR-07 | `c5b58b3` | fixed: requires human verification |
| WR-08 | `7f3d3a0` | fixed: requires human verification |
| WR-09 | `1cbcec5` | fixed: requires human verification |
| WR-10 | `38004b9` | fixed: requires human verification |
| WR-11 | `990b45b` | fixed: requires human verification |
| WR-12 | `5fa30c0` | fixed: requires human verification |

Commit order was CR-01, CR-02, WR-01, WR-10, WR-02, WR-03 … WR-12. WR-10 went before WR-02 on purpose: WR-02 registers the stint's context earlier, which would have widened WR-10's window for the length of one commit.

## Fixed Issues

### CR-01: A quick re-engage sends the previous stint's stale decision and skips the reassess decision

**Files modified:** `internal/session/autopilot.go`, `internal/session/manager.go`, `internal/session/handler.go`, `internal/driver/driver.go`, `internal/driver/loop.go`, tests, `internal/driver/corpus_live_test.go` (double only)
**Commit:** `18f5562`
**Applied fix:** The session manager is the source of truth for the stint. `AutopilotRecord.Epoch` goes up by one every time the state becomes On (an engage AND a resume), never repeats and survives the record being replaced. `EngageAutopilotEpoch` and `AutopilotEpochFor` expose it; the engage hook carries it. The driver keys its in-flight guard by user AND epoch, so a previous stint's slow model call no longer makes the new stint skip its D-08 reassess decision. Each iteration re-checks "state On and epoch unchanged" after the player call (which is before the reviewer reservation), after the reviewer call (which is before the ICM dispatch), and at the send. A stale iteration is dropped as `stage=dropped-stale` or `dropped-disengaged`: no send, no dispatch, no reviewer call, no retry, no counter, no notice, no memory write, no `lastDecisionAt` stamp. No decision row is written for it: no existing outcome value describes it honestly, so it is log-only and migration 012's CHECK is untouched. The send enforces the epoch in the manager through the new sibling `SendAICommand(userID, command, epoch)`; the human path is not overloaded. A duplicate or superseded engage hook starts nothing (`stage=engage-ignored`). The corpus double answers a constant On / epoch 1.
**Tests:** `TestStint_QuickReengageDropsTheOldDecision` (old call held, new stint's first decision runs while it is still in flight and carries the reassess paragraph; the old command is never sent or dispatched, not charged to the new stint's call count, writes no memory), `TestStint_StaleIterationIsNotAFailure`, `TestStint_DuplicateEngageStartsNothing`, `TestManager_StintEpoch`, `TestSendAICommand_RefusedOnEpochMismatch`.

### CR-02: Memory bullets and model reasoning can close their own untrusted block

**Files modified:** `internal/driver/memory.go`, `internal/driver/driver.go`, tests
**Commit:** `d5cfd27`
**Applied fix:** One helper, `neutraliseUntrusted` (and `neutraliseLine` for single-line content), used by `wrapWindow`, `wrapQuestMemory`, `wrapSessionMemory`, `wrapModelReasoning` and `goalBlock` at prompt-assembly time, so bullets poisoned before this fix are covered, and by `truncateBullets` at write time, so the store and the panel never hold a forged marker. Angle brackets and six families of Unicode look-alikes become square brackets; marker names lose their underscore (`</SESSION_MEMORY>` reaches the model as `[/SESSION-MEMORY]`); a bullet or goal is collapsed to one line with no control characters. The length ceiling applies after neutralising and no longer splits a multi-byte character (IN-03, same lines). Stored window text and stored/displayed reasoning are untouched.
**Pinned test changed on purpose:** `TestReviewPromptWrapsReasoning/forged_closing_markers…` asserted the forged markers appeared INTACT between the real ones. That is the hole. It now asserts each real closing marker appears exactly once and the words survive defanged. `TestBuildReviewSystemInstruction` and the other pinned-wording tests are unchanged.
**Tests:** `TestNeutraliseUntrusted`, `TestNeutraliseLine`, `TestWrappersCannotBeClosedByTheirContent`, `TestPromptsHoldEachMarkerOncePerBlock`, `TestBulletsAreNeutralisedAtWriteAndAtRead`.
**Not done (reviewer's "consider"):** moving the two memory blocks out of the system instruction into the user turn, and new corpus items. Both change the accepted prompt shape (D-13) and the corpus, which the task ruled out.

### WR-01: The AI send holds the manager lock across an unbounded socket write

**Files modified:** `internal/session/manager.go`, `internal/driver/driver.go` (one comment), tests
**Commit:** `b1f4132`
**Applied fix:** `sendCommand` reads the socket AND the autopilot record (state, epoch) under ONE read lock, releases it, then writes with a 5 s write deadline that is cleared afterwards, for human sends too. `m.mu` is never held across `conn.Write`. A failed write disconnects only the session it belonged to: `disconnect()` compares the socket written to with the current one under the teardown lock. The doc comments drop the claim of strict atomicity and state the remaining window (nanoseconds between unlock and the write syscall, meaning the AI command was issued just before the wheel-grab took effect).
**Tests:** `TestSendCommand_BlockedPeer` (ai and human: a `net.Pipe` nobody reads fails within the deadline; a concurrent `DisengageAutopilot` returns within 300 ms), `TestSendCommand_FailedWriteOnAReplacedSocketLeavesTheNewSessionAlone`; `TestSendCommandAs_AIRefusedUnlessAutopilotOn` comment updated, assertions unchanged.

### WR-02: `StopLoop` does not stop an in-flight iteration; the comments say it does

**Files modified:** `internal/driver/loop.go`, `internal/driver/driver.go`, `internal/session/manager.go` (comment), tests
**Commits:** `0a299d8`, follow-up `63395aa`
**Applied fix:** The gemini client already builds its request with `http.NewRequestWithContext`, so no client change was needed. The stint's context is created and registered BEFORE the stint's first decision and passed to `GenerateContent`, `ReviewCommand` and both 503 retries; the retry delay is a cancellable wait. A cancelled context reads as `dropped-disengaged` in the stint check, so the error a cancelled call returns is never counted or shown as a model failure. `StopLoop`'s and the manager's `disengageHook` comments now say exactly what is and is not interrupted. `HandleEngage` (tests, corpus harness) has no stint and keeps `context.Background()`.
**Tests:** `TestStint_StopLoopInterruptsAnInFlightModelCall` (a loop tick and the stint's first decision; the held call is never released), `TestStint_RetryDelayIsCancellable`.
**Follow-up:** the `-race -count=10` gate failed once in ten rounds in `TestLoop_Pacing/a_quiet_game…`. Cause: that sub-test called `StopLoop` straight after `waitForCalls(2)` (which returns when the call STARTS) and then asserted 2 sends; since this fix `StopLoop` drops a decision in flight, so a `StopLoop` that won the race left 1 send. It now waits for the send first, as its sibling sub-test already did. Test-only. 60 further rounds clean.

### WR-03: A dead or parked socket is reported as "the model could not be reached"

**Files modified:** `internal/session/manager.go`, `internal/driver/driver.go`, tests
**Commit:** `1ed580c`
**Applied fix:** The manager checks the switch before the connection for an AI send and returns typed `ErrNoConnection` / `ErrSendFailed`. The driver routes those to `recordSendFailure`: no D-15 count, no `DisengageAutopilot` (the WAITING the manager parked the switch at is never turned into OFF, D-19), existing not-a-halt event outcome. Audit row: existing `failed` outcome, `failure_kind = send-failed`.
**Tests:** `TestSendFailureIsNotAModelFailure` (count at threshold minus one stays put, nothing disengaged, state waiting), `TestSendAICommand_LostConnectionLeavesTheSwitchWaiting`.
**New owner-visible wording — needs the owner's eye:** "AI command was not sent: the game connection was lost."

### WR-04: On a resume the first decision runs before the game session exists

**Files modified:** `internal/session/manager.go`, tests
**Commit:** `7d96d6d`
**Applied fix:** Connect's tail is `openTranscriptThenResumeLocked`: release `m.mu`, open the transcript (the `OpenGameSession` round trip still runs with no lock held, per Phase 3 code review CR-01), re-lock, and only then resume, which fires the engage hook. The resume is skipped if the socket is no longer the one Connect dialled. Side effect, also a fix: the `[AI-ASSIST resumed]` transcript marker, lost the same way, now lands.
**Tests (fake sink with a slow open):** `TestResumeFiresAfterTheGameSessionExists`, `TestResumeSkippedWhenTheConnectionDroppedDuringTheOpen`.

### WR-05: Zero counts and an emptied memory list are dropped from the `ai` message

**Files modified:** `internal/session/websocket.go`, `cmd/server/main.go`, `frontend/src/components/AIAssistPanel.tsx`, tests, `public/` bundle
**Commit:** `005b9ee`
**Applied fix:** `calls`, `call_cap`, `failures`, `blocks`, `threshold` and `session_memory` are pointers built by `WithStint`: every driver event sends them even when 0 or `[]` (a nil list is sent as `[]`, never `null`); a message with no stint data (the goal-changed line) still omits them so it cannot zero the panel. The panel applies any carried number and any carried array.
**Tests:** `TestAIPayloadSendsZeroAndEmpty` (wire JSON), `TestAIPayloadCarriesSwitchState/the_event_after_a_cleared_streak_carries_zero_not_nothing`.

### WR-06: The 10-second Immediate Context is shorter than the gap between snapshots

**Files modified:** `internal/session/window.go`, `internal/session/manager.go`, `internal/driver/loop.go`, tests
**Commit:** `9199696`
**Applied fix:** `snapshotForDecision`: age bound = max(`windowMaxAge`, time since this user's previous snapshot), still capped by the 8 KB ring and the never-empty floor. `RecentOutputSnapshot` takes the write lock since the ring now records when it was last snapshotted.
**Found while fixing:** `runLoop` took a second, log-only snapshot an instant before each iteration's real one. With gap tracking that would have reset the memory and silently defeated the fix. Removed; the iteration's `stage=request` line already logs `snapshot_bytes`. The `loop-tick` line now logs `snapshot_bytes=0`.
**Tests (injectable clock):** `TestWindow_CoversTheGapSinceThePreviousDecision`, including a sub-test pinning that the fixed bound alone loses the text.

### WR-07: The D-25 startup gate only rejects an empty key

**Files modified:** `internal/crypto/crypto.go`, `internal/config/config.go`, tests
**Commit:** `c5b58b3`
**Applied fix:** `crypto.ParseKey` is the one definition of a usable key; `DefaultKeyStore` and `config.Load`'s gate both use it. Outside local development V1 is required and must parse; V2/V3 are optional but must parse when set. The error names the variable, never the value or its length.
**Pinned test changed on purpose:** "staging with the key set succeeds" used the literal `test-key-not-a-real-credential`, an unusable key, and expected success. It pinned the hole. It now uses a well-formed fake key.
**Review detail corrected:** a trailing newline is NOT unusable. Go's base64 decoder ignores CR/LF, the key store loads such a key, and the gate (same parser) accepts it. Pinned by a test. Quotes, a trailing space, URL-safe base64 and a wrong length are rejected.

### WR-08: The goal save is a whole-row read-modify-write, and two saves can be in flight

**Files modified:** `internal/store/profile.go`, `internal/profiles/handler.go`, `frontend/src/components/AIAssistPanel.tsx`, tests, `public/` bundle
**Commit:** `7f3d3a0`
**Applied fix:** `ProfileStore.UpdateSessionGoal` runs `UPDATE profiles SET session_goal = $1, updated_at = NOW() WHERE id = $2 AND user_id = $3`; `PutGoal` uses it. For the OTHER direction of the same race, `UpdateProfile`'s whole-row statement no longer names `session_goal` and `ProfileUpdate` has no `SessionGoal` field, so a settings/alias/trigger save cannot write an old goal back either. Only `PutGoal` ever used that field. Panel: one goal PUT in flight; when it finishes the loop re-reads what the box holds now and saves again if different, so the last typed value wins.
**Tests:** `TestUpdateSessionGoalTouchesOnlyTheGoal`, `TestUpdateProfileLeavesTheGoalAlone` (pinned SQL), `TestGoalSaveTouchesOnlyTheGoal` (handler).

### WR-09: A failed Quest upsert leaves a goal with no Quest

**Files modified:** `internal/profiles/handler.go`, `internal/driver/driver.go`, `frontend/src/services/api.ts`, `frontend/src/components/AIAssistPanel.tsx`, tests, `public/` bundle
**Commit:** `1cbcec5`
**Applied fix:** `PutGoal` answers 500 with `{error, goal, quest:"failed"}` instead of 200 / `quest:"none"`; the panel shows the server's words and leaves its last-saved marker alone so Enter re-sends. Driver: `activeQuest` looks the Quest up and, only on a miss for a non-blank goal, calls the same ensure-or-create upsert; `activeQuestBullets` and `persistMemory` both use it. An existing Quest is never upserted; a failed repair never fails the decision. New log line `quest-repair` (ids and outcome only).
**Tests:** `TestGoalQuestFailureIsNotSilent`, `TestMissingQuestIsRepairedAtTheNextDecision` (four cases).
**New owner-visible wording — needs the owner's eye:** "Your goal was saved, but its Quest could not be prepared. Press Enter in the goal box to try again."

### WR-10: A late `StopLoop` goroutine can cancel the next stint's loop

**Files modified:** `internal/session/manager.go`, `internal/driver/loop.go`, `internal/driver/driver.go`, tests
**Commit:** `38004b9`
**Applied fix:** The disengage hook carries the epoch of the stint that ended. `StopLoop(userID, epoch)` cancels only a loop of that stint or older; an earlier stint never replaces a later one in the registry; a loop that stops by itself removes its own entry if still its own.
**Tests:** `TestStint_LateStopLoopLeavesTheNextStintRunning`, `TestStint_LoopThatStopsItselfLeavesNoEntry`; `TestManager_DisengageHookFires` asserts the epoch on a disengage and on a park.

### WR-11: The harness's live run irreversibly deletes captured text and can blank the owner's goal

**Files modified:** `scripts/verify-phase4.sh`
**Commit:** `990b45b`
**Applied fix:** The live delete STAYS unconditional (C4's proof; T-4-15 accepted). Header, usage and the live report's own preamble say plainly that a live run permanently deletes the connection's captured text, what is kept, and that two Quest rows remain; the false "leaves the profile exactly as it found it" is gone. Step 1 failing (anything but 200) aborts with exit 2 before anything is changed. An EXIT/INT/TERM/HUP trap restores the goal once it has been read, only when there is a real goal to restore, at most once, never in self-test modes, never printing the cookie. Every curl call is bounded by `--max-time` (`HTTP_MAX_TIME`, default 30): bash runs a trap only after the command it waits on returns, so an unanswered request would otherwise hang the trap too. Usage cookie name corrected to `session_token` (IN-10, same lines).
**Verified against a localhost stub only:** step 1 → 500: exit 2 after one GET, zero writes. Step 16's PUT failing: the trap's second PUT restores. TERM during a hung step 4: no further steps (no delete), goal restored, exit 143. Cookie absent from every report.
**Not done (reviewer's suggestions the task overrode or did not ask for):** `--allow-delete` opt-in (the task says the delete stays unconditional); refusing to run while autopilot is On.

### WR-12: The harness's ownership checks pass on any failure at all

**Files modified:** `scripts/verify-phase4.sh`
**Commit:** `5fa30c0`
**Applied fix:** `_check_refused` requires status 400/403/404, a JSON body with a non-empty `error`, and the absence of the field that would carry the other owner's data (`goal`, `session_memory`, `snapshots_cleared`). 000, 401, 5xx and 200 each FAIL with a reason. The flawed `_check_not_status` is removed.
**Verified:** `--self-test` exit 0; `--self-test-negative` exit 1 with exactly one `FAIL C4`; localhost stub answering 502 to the not-owned calls → C1, C2, C4 all FAIL. No fixture changed.

## Skipped Issues

None. Info findings IN-01, IN-02, IN-04 to IN-09 were out of scope and untouched. IN-03 (byte truncation splitting a rune) and two items of IN-10 (curl `--max-time`, the usage text's cookie name) were fixed because they sat inside lines already being changed.

## Gate results (final tree)

Run twice: once on the branch as written, and once more on the tree rebased onto `209d293`. Both runs gave the results below; the flake row's numbers are totals.

| Gate | Result |
|------|--------|
| `go build ./...` after every commit's change set | pass |
| `go vet ./...` | pass |
| `go test ./... -race -count=1` | pass, all 9 packages |
| `go test ./internal/driver/... ./internal/session/... -race -count=10` | 1 failure in the first 10 rounds (the WR-02 follow-up above, test-only, fixed in `63395aa`); then 70 rounds clean, the last 10 on the rebased tree |
| `bash scripts/verify-phase4.sh --self-test` | exit 0 |
| `bash scripts/verify-phase4.sh --self-test-negative` | exit 1, exactly one `FAIL C4` |
| `cd frontend && npm run build` | pass; a clean build reproduces the committed bundle (`index-CHzRoMVx.js`), tree clean |
| `git diff --stat go.mod frontend/package.json` (plus `go.sum`, `package-lock.json`) | empty |

Not run, by instruction: no deploy, nothing against Railway or staging, no live corpus test, no SQL against any remote database, no sign-out. The harness's live mode was exercised only against a throwaway stub on `127.0.0.1`.

## To re-verify on staging

1. **Before deploying — WR-07 can stop the server.** If staging's `ENCRYPTION_KEY_V1` (or a set `V2`/`V3`) is not standard base64 of exactly 32 bytes, the server will now refuse to start, by design. Until now such a key was silently ignored and a random key used. Confirm the key is well formed before `railway up`, without printing it. If the server refuses to start, that also means stored credentials on staging were being encrypted under a throwaway key.
2. **CR-01 / WR-10 / WR-02, the walkthrough that found the original bug:** autopilot ON, take the wheel for one command while a decision is visibly pending, `#AUTO ON` again at once. Expect: no command from before the wheel-grab is sent; the first decision after re-engaging states the current situation (reassess); decisions keep coming. In the `[AI-PLAYER]` log expect `stage=dropped-disengaged` or `dropped-stale`, never `failed` for that decision. Repeat with a page refresh (park → resume).
3. **WR-04:** refresh the play screen with autopilot ON. The first decision after the resume should show Session Memory in use, the panel's memory list should survive, and the log page should show the `[AI-ASSIST resumed]` marker.
4. **WR-03:** let Alter Aeon drop the socket with autopilot ON. Expect the badge WAITING (not OFF), the new line "AI command was not sent: the game connection was lost." at most once, no "(n of 3)" count for it, and a resume on reconnect.
5. **WR-05:** force one transient failure, then a sent command: the status line must read "Consecutive failures: 0 of 3".
6. **WR-06:** in a busy room, check the AI's reasoning reacts to the game's answer to its own previous command.
7. **WR-08 / WR-09:** edit the goal quickly twice (blur then Enter) and confirm the box and a refresh agree; save AI Settings and the goal close together and confirm neither reverts the other.
8. **CR-02 changes what the model reads:** game text and MUD prompts containing `<` or `>` (for example `<100hp 50m>`) now reach the model as `[100hp 50m]`. This should be harmless, but it touches every prompt, so the D-24 AFTER red-team corpus rerun planned for a fresh-quota day should be done on this build before the staging deploy is accepted.
9. **WR-11:** the harness's live mode still deletes captured text. Its report now says so at the top.

## Owner decisions carried out of this pass

- Two new owner-visible sentences (WR-03, WR-09), quoted above. They follow the voice of the UI-SPEC's Copywriting Contract but were not in it.
- A send that fails because the connection was lost now stores a decision row (`failed` / `send-failed`) and shows one warning-coloured line. The alternative is log-only.

## Observations outside the findings (not changed)

- **`TestMigration014AddsLoginStartedAtColumn` is line-ending sensitive.** A fresh checkout with `core.autocrlf=true` gives `migrations/*.sql` CRLF endings and the test, which compares against bare `\n`, fails. It passes in the main checkout only because those files were authored there with LF. A `.gitattributes` line (`migrations/*.sql text eol=lf`), as already exists for `scripts/*.sh`, would fix it. Seen because this pass ran in a fresh worktree.
- **Residual window in `recordFailure`:** it counts, inserts the decision row (a database round trip) and only then disengages, by user id. A stint that ends and restarts inside that round trip would have the new stint disengaged by the old one's failure. Far narrower than CR-01 and needs an epoch-scoped disengage on the manager; not attempted here.
- **`Connect` still holds `m.mu` across `net.DialTimeout` (5 s)**, which stalls every user for the length of a slow dial. Same family as WR-01, pre-existing, outside the reviewed findings.
- **`TestLoop_Pacing/a_quiet_game…`** can still, in principle, see a third floor-interval tick if the test goroutine is descheduled for more than ~30 ms between the second send and `StopLoop`. Pre-existing; not observed in 60 rounds.

---

_Fixed: 2026-09-17T13:34:38Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
