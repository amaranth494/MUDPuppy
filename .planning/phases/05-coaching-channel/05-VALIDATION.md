---
phase: 5
slug: coaching-channel
status: planned
nyquist_compliant: true
wave_0_complete: false
created: 2026-09-17
updated: 2026-09-17
---

# Phase 5 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution. Derived from `05-RESEARCH.md` § Validation Architecture. The Per-Task Verification Map's Task ID, Plan and Wave columns were filled in when the eleven Phase 5 plans were written, and extended on 2026-09-17 when the checker's two blockers added the pause/resume stream notices, the chat conversation tail, the coaching-in-effect block in AI-chatter's prompt and the chat send-failure notice.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go standard library `testing` (backend). No frontend test framework exists or is added this phase; browser-only behaviour (panel split, pop-out windows, Help and Logs pages) is proven by end-user staging screenshots. |
| **Config file** | none — `go.mod` at repo root |
| **Quick run command** | `go test ./internal/driver/... ./internal/session/... ./internal/gemini/... ./internal/config/... -count=1` |
| **Full suite command** | `go test ./internal/... -count=1` |
| **Frontend gate** | `npm run build` in `frontend/` (type-check and bundle; no unit tests) |
| **Estimated runtime** | ~8 seconds (full Go suite measured at ~7.7s on 2026-09-17) |

---

## Sampling Rate

- **After every task commit:** Run the quick run command (and `npm run build` in `frontend/` when the task touched frontend files)
- **After every plan wave:** Run `go test ./internal/... -count=1`
- **Before `/gsd:verify-work`:** Full suite green, `go build ./...` clean, frontend build clean, plus the flagged live corpus rerun for DR-4-01 and D-28 (spends real model quota, run once at the phase gate by plan 05-11)
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

Every row is claimed by a task in a plan. Rows marked ❌ W0 need their test file created or extended by the task that owns them; in this phase every Wave 0 gap is closed by the same task that makes the behaviour true, because each plan ships its own tests.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 05-06-01 / 05-06-02 | 05-06 | 4 | REQ-coaching-chat | T-5-26 coaching laundering | Only the owner's own chat message, through `HandleChat`, can write coaching; the pushed line reaches the next decision's prompt | unit | `go test ./internal/driver/... -run TestHandleChat_PushReachesNextPrompt -v` | ❌ W0 → 05-06 | ⬜ pending |
| 05-06-01 | 05-06 | 4 | REQ-coaching-chat (D-10, D-17) | T-5-56 echo laundering | AI-chatter's own prompt carries the coaching currently in effect in a delimited `<COACHING>` block, with the instruction to copy a withdrawn line verbatim from it — so a withdraw matches against a real model, not only a fake one | unit | `go test ./internal/driver/... -run TestHandleChat_PromptCarriesCoachingInEffect -v` | ❌ W0 → 05-06 | ⬜ pending |
| 05-06-01 | 05-06 | 4 | REQ-coaching-chat | — | Withdraw removes the suggestion from the list and the next prompt, and the reply names the exact line removed | unit | `go test ./internal/driver/... -run TestHandleChat_Withdraw -v` | ❌ W0 → 05-06 | ⬜ pending |
| 05-06-01 | 05-06 | 4 | REQ-coaching-chat | T-5-27 false quotation | The `Sent to AI-player:` prefix is composed in Go from the stored line, so a lying model cannot fake it | unit | `go test ./internal/driver/... -run TestHandleChat_ReplyQuotesTheExactLineSent -v` | ❌ W0 → 05-06 | ⬜ pending |
| 05-06-01 | 05-06 | 4 | REQ-coaching-chat (D-05) | — | The `[Coaching received]` marker is emitted once per push or withdraw, carries no suggestion text, and is put on the wire unbracketed because the panel supplies the brackets | unit | `go test ./internal/driver/... -run TestHandleChat_EmitsCoachingReceivedMarker -v` | ❌ W0 → 05-06 | ⬜ pending |
| 05-06-01 | 05-06 | 4 | REQ-coaching-chat | T-5-26 | `UpdateCoaching` has exactly one non-test caller and it is `chat.go` | unit (source assertion) | `go test ./internal/driver/... -run TestCoachingHasOneWriter -v` | ❌ W0 → 05-06 | ⬜ pending |
| 05-06-02 | 05-06 | 4 | REQ-coaching-chat | T-5-28 rule override | Coaching sits below the conduct rules, guidance, Never-issue list and goal in both prompts, with an explicit subordination sentence | unit | `go test ./internal/driver/... -run "TestCoachingIsSubordinateInBothPrompts\|TestCoachingIsWrappedInBothPrompts" -v` | ❌ W0 → 05-06 | ⬜ pending |
| 05-06-02 | 05-06 | 4 | REQ-coaching-chat | T-5-26 | A coaching line cannot forge a marker, `COACHING` and `CONVERSATION` are both in the marker-name regex, and the ceiling of 8 bullets at 200 chars is enforced in Go | unit | `go test ./internal/driver/... -run "TestCoachingCannotForgeAMarker\|TestCoachingCeilingIsEnforced\|TestUntrustedParagraphNamesEveryMarker" -v` | ❌ W0 → 05-06 | ⬜ pending |
| 05-06-01 | 05-06 | 4 | REQ-coaching-chat | — | Coaching's lifetime is Session Memory's: the same `login_started_at` inheritance | unit | `go test ./internal/store/... -run TestCoachingStoreSQL -v` | ❌ W0 → 05-06 | ⬜ pending |
| 05-05-02 | 05-05 | 3 | REQ-coaching-chat | T-5-21 cost abuse | Cap reached: no model call, owner message still stored, locked notice, autopilot untouched | unit | `go test ./internal/driver/... -run TestHandleChat_CapReached -v` | ❌ W0 → 05-05 | ⬜ pending |
| 05-05-02 | 05-05 | 3 | REQ-coaching-chat | T-5-19 | AI-chatter's prompt carries everything AI-player knows, every untrusted block delimited and neutralised | unit | `go test ./internal/driver/... -run "TestHandleChat_ReplyCarriesEverythingAIPlayerKnows\|TestHandleChat_UntrustedBlocksCannotForgeAMarker" -v` | ❌ W0 → 05-05 | ⬜ pending |
| 05-05-02 | 05-05 | 3 | REQ-coaching-chat (D-01, D-11) | T-5-55 stale instruction replay | The stateless chat call carries a bounded 10-line `<CONVERSATION>` tail, oldest first, excluding the current message, so a follow-up makes sense without the tail becoming an instruction | unit | `go test ./internal/driver/... -run TestHandleChat_PromptCarriesRecentConversation -v` | ❌ W0 → 05-05 | ⬜ pending |
| 05-05-02 | 05-05 | 3 | REQ-coaching-chat | T-5-20 | AI-chatter dispatches nothing, writes no memory, goal or profile and never touches the switch | unit | `go test ./internal/driver/... -run "TestHandleChat_TouchesNothingElse\|TestHandleChat_WorksInEveryAutopilotState" -v` | ❌ W0 → 05-05 | ⬜ pending |
| 05-05-03 | 05-05 | 3 | REQ-coaching-chat | T-5-22 wheel-grab bypass | A chat message never reaches the MUD and never takes the wheel, and is accepted while disconnected | unit | `go test ./internal/session/... ./internal/driver/... -run TestWebSocket_ChatIsNotAWheelGrab -v` | ❌ W0 → 05-05 | ⬜ pending |
| 05-05-01 | 05-05 | 3 | REQ-coaching-chat | — | The conversation is stored against the game session, read back login-scoped, and readable as a bounded newest-first tail | unit | `go test ./internal/store/... -run TestConversationStoreSQL -v` | ❌ W0 → 05-05 | ⬜ pending |
| 05-01-01 | 05-01 | 1 | REQ-pause-resume | — | Pause stops the loop through the disengage hook while the game keeps filling the reaction window (replaces the research's provisional `TestLoop_PauseNoDecisions`, which assumed the check lived in the driver; the window is fed by the session manager, so the test lives there) | unit | `go test ./internal/session/... -run TestManager_PauseStopsTheLoopAndKeepsReading -v` | ❌ W0 → 05-01 | ⬜ pending |
| 05-01-01 | 05-01 | 1 | REQ-pause-resume | T-5-01 state race | Resume only when both "paused by owner" and "connection lost" are clear; a reconnect never lifts a pause | unit | `go test ./internal/session/... -run "TestManager_ResumeRequiresBothReasonsClear\|TestManager_PauseNeverResumesByItself\|TestAutopilotRecordCarriesBothWaitingReasons" -v` | ❌ W0 → 05-01 | ⬜ pending |
| 05-01-01 | 05-01 | 1 | REQ-pause-resume | — | Taking the wheel while paused still lands the switch OFF | unit | `go test ./internal/session/... -run TestManager_WheelGrabWhilePausedLandsOff -v` | ❌ W0 → 05-01 | ⬜ pending |
| 05-01-02 | 05-01 | 1 | REQ-pause-resume | T-5-03 | The pause and resume actions answer with the state and both waiting reasons | unit | `go test ./internal/session/... -run TestAutopilotHandler_PauseAndResumeActions -v` | ❌ W0 → 05-01 | ⬜ pending |
| 05-01-03 | 05-01 | 1 | REQ-pause-resume (D-15, UI-SPEC Copywriting Contract) | T-5-54 false notice | The thinking stream gets `Autopilot paused` on a real pause and `Autopilot resumed` only once every waiting reason has cleared, as system-kind AI events with outcomes `paused` and `resumed`, unbracketed on the wire | unit | `go test ./internal/session/... -run TestAutopilotHandler_EmitsPausedAndResumedNotices -v` | ❌ W0 → 05-01 | ⬜ pending |
| 05-01-01 | 05-01 | 1 | REQ-pause-resume | — | The first decision after resume reassesses (Phase 4 path, unchanged) | unit | `go test ./internal/driver/... -run TestEngageLoop_Reassess -v` | ✅ (Phase 4, regression) | ⬜ pending |
| 05-02-01 | 05-02 | 1 | D-24 / DR-4-02 | T-5-05 reviewer fail-open | A reviewer answer missing `blocked` is a failed review; the command is not sent | unit | `go test ./internal/gemini/... -run "TestReviewCommand_MissingBlockedField\|TestReviewCommand_BlockedFalseIsStillAVerdict" -v` and `go test ./internal/driver/... -run TestDriverTreatsMissingVerdictAsFailedReview -v` | ❌ W0 → 05-02 | ⬜ pending |
| 05-02-02 | 05-02 | 1 | D-24 / DR-4-02 | T-5-07 false authorship | The reviewer is told the memory notes were written by the other model, and the player prompt is not | unit | `go test ./internal/driver/... -run TestBuildReviewSystemInstruction_MemoryAuthorship -v` | ❌ W0 → 05-02 | ⬜ pending |
| 05-02-02 | 05-02 | 1 | D-23 / DR-4-01 | T-5-06 over-blocking | Fighting a game-presented target is named as ordinary play; only attacking another player is off limits | unit | `go test ./internal/driver/... -run TestBuildReviewSystemInstruction_FightingIsOrdinaryPlay -v` | ❌ W0 → 05-02 | ⬜ pending |
| 05-02-02 → 05-11-02 | 05-02 / 05-11 | 1 / 7 | D-23 / DR-4-01 | T-5-06, T-5-08 | The two real false blocks are benign controls sampled three times each and are not blocked | live (flagged, gated) | `MUDPUPPY_LIVE_CORPUS=1 MUDPUPPY_LIVE_CORPUS_STRICT=1 go test ./internal/driver/... -run TestLiveCorpus_HostileText -v` (well-formedness by `go test ./internal/driver/... -run TestCorpusIsWellFormed -v`) | ❌ W0 → 05-02 | ⬜ pending |
| 05-06-03 → 05-11-02 | 05-06 / 05-11 | 4 / 7 | D-28 | T-5-26 coaching laundering | Attacks aimed at the coaching channel push no suggestion and steer no command | live (flagged, gated) | same corpus command, items `chatter-channel-01` to `-03` | ❌ W0 → 05-06 | ⬜ pending |
| 05-04-01 | 05-04 | 2 | D-25 / DR-4-03 | T-5-14 rate-limit bypass | AI sends over the per-second limit are refused before dispatch; a flood test proves it, and the refusal is transient | unit | `go test ./internal/driver/... -run "TestAllowAISend_\|TestDriverRateLimitRefusalIsTransient" -v` | ❌ W0 → 05-04 | ⬜ pending |
| 05-04-02 | 05-04 | 2 | D-25 / DR-4-03 | T-5-16 | Blank means the server's own default, never unlimited; out-of-range values are rejected | unit | `go test ./internal/store/... ./internal/profiles/... -run "TestResolveAISettings\|ValidateAISettings" -v` | ❌ W0 → 05-04 | ⬜ pending |
| 05-04-01 | 05-04 | 2 | D-25 / DR-4-03 | T-5-15 | The ICM dispatcher's pass-through and circuit breaker are untouched | source assertion | `git diff --stat -- internal/icm/` is empty, captured in `evidence/01-test-report.txt` § ICM UNCHANGED | ✅ existing | ⬜ pending |
| 05-03-01 | 05-03 | 1 | D-26 / DR-4-04 | T-5-10 vault key | The server refuses to start without a vault key on any host unless the local-development setting is on, and announces the opt-out | unit | `go test ./internal/config/... -run "TestLoadRequiresEncryptionKeyEverywhereUnlessLocalDev\|TestLocalDevOptOutIsAnnounced" -v` | ❌ W0 → 05-03 | ⬜ pending |
| 05-03-02 → 05-11-03 | 05-03 / 05-11 | 1 / 7 | D-27 / AR-4-08 | T-5-11 | `/logs/:connectionId` sends a signed-out visitor to sign-in | source assertion + manual screenshot | `AuthGuard` line-range grep in `frontend/src/App.tsx`, captured in `evidence/01-test-report.txt` § LOGS ROUTE GUARD; screenshot `evidence/13-logs-signed-out.png` | ✅ existing code path | ⬜ pending |
| 05-07-01 / 05-07-02 / 05-07-03 | 05-07 | 5 | REQ-coaching-chat, REQ-pause-resume | T-5-32 wrong input path | The panel compiles, renders every locked string, and never calls the command path from the chat box | build + source assertion | `cd frontend && npm run build` plus the plan's greps | n/a (no FE framework) | ⬜ pending |
| 05-07-02 | 05-07 | 5 | REQ-pause-resume (D-15) | T-5-54 | `.ai-system-line.state-paused` (warning yellow) and `.state-resumed` (neutral `#888`) exist and match the server's two `Outcome` strings, so the locked pause and resume notices render in their assigned colour buckets with no new rendering code | build + source assertion | `cd frontend && npm run build` plus `grep -c "\.ai-system-line\.state-paused\|\.ai-system-line\.state-resumed" frontend/src/index.css` | n/a (no FE framework) | ⬜ pending |
| 05-07-03 | 05-07 | 5 | REQ-coaching-chat (UI-SPEC "Chat — send failed") | T-5-57 silent loss | `sendChat` returns a truthful boolean and a false answer keeps the draft and renders `Message failed to send — try again` as a red system chat line | build + source assertion | `cd frontend && npm run build` plus the plan's `sendChat` readiness/catch and call-site-branch greps | n/a (no FE framework) | ⬜ pending |
| 05-08-01 / 05-08-02 | 05-08 | 6 | REQ-coaching-chat | T-5-37, T-5-38 | The only `window.open` is `popout.ts`; one subscription for the whole panel; every D-29 string present | build + source assertion | `cd frontend && npm run build` plus the plan's greps | n/a (no FE framework) | ⬜ pending |
| 05-09-01 | 05-09 | 6 | REQ-promote-guidance, REQ-doc-coaching | T-5-44 doc drift | The help article parses, carries the locked slug/title/description, names every shipped surface, and mentions no promotion | build + JSON parse | `cd frontend && npm run build` and the article parse in the plan's verify block | n/a | ⬜ pending |
| 05-09-02 | 05-09 | 6 | REQ-doc-coaching | T-5-42 | The Logs page shows the conversation in its own section below the transcript | build + source assertion | `cd frontend && npm run build` plus the plan's greps | n/a | ⬜ pending |
| 05-10-01 | 05-10 | 5 | REQ-coaching-chat, REQ-pause-resume | T-5-45 unfalsifiable evidence | The canned report can fail: the negative self-test yields `FAIL C3` and a non-zero exit | script self-test | `bash scripts/verify-phase5.sh --self-test` and `--self-test-negative` | ❌ W0 → 05-10 | ⬜ pending |
| 05-11-01 | 05-11 | 7 | all four | T-5-SC | No package added anywhere; no model literal in source; the coaching store has one writer | canned report | `evidence/01-test-report.txt` §§ DEPENDENCY DRIFT, MODEL LITERALS, COACHING WRITER | n/a | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

**Deviation recorded against `05-RESEARCH.md` § Phase Requirements → Test Map:** that document proposed `go test ./internal/driver/... -run TestLoop_PauseNoDecisions`. Planning found that the Immediate Context window is filled by `internal/session/manager.go`'s MUD-read path (`appendOutputWindow`), entirely independent of the driver's loop, and that a pause stops the loop through the existing disengage hook. Both halves of D-14 are therefore observable in one `internal/session` test, and the row above names it. No coverage is lost; the test moved packages.

---

## Wave 0 Requirements

Each gap is closed by the plan named beside it, in the same task that makes the behaviour true.

- [ ] `internal/session/manager_test.go` / `autopilot_test.go` — extend; the pause/resume pair, the two waiting reasons, the window-still-fed rule and the two locked stream notices (**plan 05-01**, wave 1)
- [ ] `internal/gemini/client_test.go` — extend; the missing-verdict regression test (**plan 05-02**, wave 1), then the `Chat` decode tests (**plan 05-05**, wave 3) and the push/withdraw decode tests (**plan 05-06**, wave 4)
- [ ] `internal/driver/testdata/` and `corpus_live_test.go` — new corpus items: the two repeat-sampled benign fight items (**plan 05-02**, wave 1) and the three chatter-channel attacks (**plan 05-06**, wave 4)
- [ ] `internal/config/config_test.go` — extend; the everywhere-unless-local-dev rule (**plan 05-03**, wave 1)
- [ ] `internal/driver/ratelimit_test.go` — new; the flood test (**plan 05-04**, wave 2)
- [ ] `internal/store/profile_test.go` — extend; blank-means-server-default for the rate limit (**plan 05-04**, wave 2)
- [ ] `internal/driver/chat_test.go` — new; the `HandleChat` suite including the conversation-tail test (**plan 05-05**, wave 3), extended with the coaching push/withdraw suite and the coaching-in-effect prompt test (**plan 05-06**, wave 4)
- [ ] `internal/driver/driver_test.go` — extend `fakeModels` with a `Chat` method, add a `fakeConversation` double (recording appends and answering `RecentConversation`) and a `NotifyChat` recorder (**plan 05-05**, wave 3), then a `fakeCoaching` double (**plan 05-06**, wave 4)
- [ ] `internal/store/conversation_test.go` and `internal/store/coaching_test.go` — new; SQL-shape tests that start no database (**plans 05-05 and 05-06**, waves 3 and 4)
- [ ] `scripts/verify-phase5.sh` and its fixtures — new; the canned-report harness with both self-test modes (**plan 05-10**, wave 5)
- [ ] Framework install: **none**. Nothing is added to `go.mod` or `frontend/package.json` anywhere in this phase.

---

## Manual-Only Verifications

Evidence for these is an end-user screenshot or a canned report / log excerpt from staging, filed by plan 05-11. Database queries never count.

| Behavior | Requirement | Plan / Task | Evidence file | Test Instructions |
|----------|-------------|-------------|---------------|-------------------|
| A coaching message sent while the AI plays shows `[Coaching received]` in the stream and is reflected in the next decision's logged reasoning | REQ-coaching-chat (criterion 1) | 05-11 / 05-11-04 | `06-coaching-in-next-decision.png`, `07-chat-reply.png`, `05-staging-ai-player.log` | Engage autopilot on staging, send a coaching message, capture the panel and the log lines showing the push and the next decision; check the marker reads with exactly one pair of brackets |
| Taking a suggestion back in words removes the exact line, proving AI-chatter can see the coaching in effect | REQ-coaching-chat (D-10, D-17) | 05-11 / 05-11-04 | recorded in `05-11-SUMMARY.md` (step 8) | Say "forget what I said about ..."; the reply must begin `Withdrew from AI-player: ` with the exact line and the line must leave the list; "nothing matched" while the line is still listed is a finding |
| Pause from the AI-player button: WAITING with `Paused by owner`, `[Autopilot paused]` in the stream, the game still read, no model calls and no commands for that stretch; resume shows `[Autopilot resumed]` and reassesses | REQ-pause-resume (criterion 2) | 05-11 / 05-11-04 | `18-paused-waiting-reason.png`, `05-staging-ai-player.log` | Pause on staging, wait through several rooms of game output, capture the panel and the bracketed quiet stretch in the log; resume and capture the first decision |
| One panel, two views; the pushed suggestion quoted in the reply, listed under Coaching in effect and marked in the stream; the list survives a refresh | criterion 4 | 05-11 / 05-11-04 | `17-panel-two-views.png`, `09-coaching-after-refresh.png` | Staging screenshots of the docked panel before and after a hard refresh |
| The message box works with autopilot off and typing in it never takes the wheel | REQ-coaching-chat (D-06, D-16) | 05-11 / 05-11-04 | `08-chat-while-off.png` | Send a chat message with the badge reading Off; confirm the badge does not change and the terminal receives nothing |
| Both pop-out windows live beside the play screen, working the same as docked; closing one returns it to the panel | D-29 (criterion 4) | 05-11 / 05-11-04 | `10-popouts-both-open.png`, `11-popout-brought-back.png` | Real multi-window browser behaviour cannot be exercised in jsdom; staging screenshots only |
| The help article opens and its steps lead the owner to copy a suggestion into AI settings by hand | REQ-doc-coaching, REQ-promote-guidance (criterion 3) | 05-11 / 05-11-04 | `14-help-article.png`, `15-guidance-pasted-by-hand.png` | Open the article, follow it for real, capture the saved settings field |
| The Logs page shows the Coaching Conversation section below the transcript | D-04, UI-SPEC §6 | 05-11 / 05-11-04 | `16-logs-conversation-section.png` | Select the walkthrough's session on the Logs page |
| The AI Command Rate Limit field sits with the other AI settings with its locked label, placeholder and hint | D-25 | 05-11 / 05-11-03 | `12-ai-settings-rate-limit.png` | Open AI Player settings; leave the field blank |
| `/logs/:connectionId` sends a signed-out visitor to sign-in | D-27 / AR-4-08 | 05-11 / 05-11-03 | `13-logs-signed-out.png` | Private window or separate browser profile only. **Never sign the owner out.** |
| Hand play on a profile that never accepted the policy is untouched by this phase | UI-SPEC Regression Invariant | 05-11 / 05-11-04 | `19-hand-play-unchanged.png` | Connect on a non-accepted profile and play by hand; no panel, no chat strip, no Pause button, no pop-out control |

---

## Validation Sign-Off

- [x] All tasks have an `<automated>` verify block or a named Wave 0 dependency closed within their own plan
- [x] Sampling continuity: no three consecutive tasks without an automated verify — every plan's tasks carry either a `go test`/`go vet` command, an `npm run build` gate with source assertions, or (for the two staging checkpoints) a `<human-check>` backed by grep-able acceptance criteria
- [x] Wave 0 covers all MISSING references, each assigned to the plan that makes the behaviour true
- [x] No watch-mode flags
- [x] Feedback latency < 15s (full Go suite ~7.7s; frontend build is the only slower gate and runs only on frontend tasks)
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** planner, 2026-09-17, on completion of plans 05-01 through 05-11; revised the same day after the checker's review added four rows (pause/resume stream notices, the chat conversation tail, the coaching-in-effect chat block, the chat send-failure notice) and one CSS row.
</content>
