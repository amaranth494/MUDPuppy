---
phase: 5
slug: coaching-channel
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-09-17
---

# Phase 5 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution. Derived from `05-RESEARCH.md` § Validation Architecture.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go standard library `testing` (backend). No frontend test framework exists or is added this phase; browser-only behaviour (panel split, pop-out windows) is proven by end-user staging screenshots. |
| **Config file** | none — `go.mod` at repo root |
| **Quick run command** | `go test ./internal/driver/... ./internal/session/... ./internal/gemini/... ./internal/config/... -count=1` |
| **Full suite command** | `go test ./internal/... -count=1` |
| **Frontend gate** | `npm run build` in `frontend/` (type-check and bundle; no unit tests) |
| **Estimated runtime** | ~8 seconds (full Go suite measured at ~7.7s on 2026-09-17) |

---

## Sampling Rate

- **After every task commit:** Run the quick run command (and `npm run build` in `frontend/` when the task touched frontend files)
- **After every plan wave:** Run `go test ./internal/... -count=1`
- **Before `/gsd:verify-work`:** Full suite green, `go build ./...` clean, frontend build clean, plus the flagged live corpus rerun for DR-4-01 (spends real model quota, run once at the phase gate)
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

The planner assigns task IDs; every row below must be claimed by a task in a plan. Rows marked ❌ W0 need their test file created or extended before or with the task that makes them pass.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD | TBD | TBD | REQ-coaching-chat | coaching laundering | Only the owner's own chat message, through `HandleChat`, can write coaching; pushed line reaches the next decision's prompt | unit | `go test ./internal/driver/... -run TestHandleChat_PushReachesNextPrompt -v` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | REQ-coaching-chat | — | Withdraw removes the suggestion from the list and the next prompt | unit | `go test ./internal/driver/... -run TestHandleChat_Withdraw -v` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | REQ-coaching-chat | cost abuse | Cap reached: no model call, owner message still stored, locked notice | unit | `go test ./internal/driver/... -run TestHandleChat_CapReached -v` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | REQ-pause-resume | — | Pause stops decisions and sends; the game is still read | unit | `go test ./internal/driver/... -run TestLoop_PauseNoDecisions -v` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | REQ-pause-resume | — | Resume only when both "paused by owner" and "connection lost" are clear; never by itself | unit | `go test ./internal/session/... -run TestManager_ResumeRequiresBothReasonsClear -v` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | REQ-pause-resume | — | First decision after resume reassesses | unit | `go test ./internal/driver/... -run TestEngageLoop_Reassess -v` | ✅ (Phase 4, regression) | ⬜ pending |
| TBD | TBD | TBD | D-24 / DR-4-02 | reviewer fail-open | Reviewer answer missing `blocked` is a failed review; command not sent | unit | `go test ./internal/gemini/... -run TestReviewCommand_MissingBlockedField -v` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | D-23 / DR-4-01 | over-blocking | Benign fight commands pass the reviewer, sampled more than once | live (flagged) | `MUDPUPPY_LIVE_CORPUS=1 go test ./internal/driver/... -run TestLiveCorpus_HostileText -v` | ❌ W0 (new corpus items) | ⬜ pending |
| TBD | TBD | TBD | D-25 / DR-4-03 | rate-limit bypass | AI sends over the per-second limit are refused; flood test proves it | unit | `go test ./internal/driver/... -run TestAllowAISend_FloodRefused -v` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | D-26 / DR-4-04 | vault key regeneration | Server refuses to start without a vault key anywhere unless the local-development setting is on | unit | `go test ./internal/config/... -run TestLoadRequiresEncryptionKeyEverywhereUnlessLocalDev -v` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | D-28 | coaching laundering | AI-chatter channel attack items in the red-team corpus do not produce coaching or commands | live (flagged) | same corpus command, new AI-chatter items | ❌ W0 (new corpus items) | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/driver/chat_test.go` — new; the three REQ-coaching-chat rows
- [ ] `internal/driver/ratelimit_test.go` — new; DR-4-03 flood test
- [ ] `internal/session/manager_test.go` — extend; pause/resume pair and the both-reasons-clear rule
- [ ] `internal/gemini/client_test.go` — extend; DR-4-02 missing-field test
- [ ] `internal/config/config_test.go` — extend; DR-4-04 everywhere-unless-local-dev rule
- [ ] `internal/driver/driver_test.go` — extend `fakeModels` with a `Chat` method
- [ ] `internal/driver/testdata/` — new corpus items (benign fight items for D-23, AI-chatter channel items for D-28)
- [ ] Framework install: none

---

## Manual-Only Verifications

Evidence for these is an end-user screenshot or a canned report / `[AI-PLAYER]` log excerpt from staging. Database queries never count.

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| A coaching message sent while the AI plays shows "[Coaching received]" in the stream and is reflected in the next decision's logged reasoning | REQ-coaching-chat (criterion 1) | Needs the live model on staging | Engage autopilot on staging, send a coaching message, capture the panel screenshot and the log excerpt showing the next decision's reasoning |
| Pause from the AI-player button: WAITING with "Paused by owner", game still read, no model calls, no commands; resume reassesses | REQ-pause-resume (criterion 2) | Live loop behaviour over time | Pause on staging, wait through game output, capture screenshot and log excerpt; resume and capture the first decision |
| Panel is one panel with two views; pushed suggestions are quoted in the reply, listed under "Coaching in effect", marked in the stream | criterion 4 | Browser layout; no frontend test framework | Staging screenshot of the docked panel |
| Both pop-out windows live beside the play screen, working the same as docked; closing one returns it to the panel; blocked pop-up shows the plain notice | D-29 (criterion 4) | Real multi-window browser behaviour cannot be exercised in jsdom | Staging screenshots: both windows open and live; panel after closing one |
| Help article opens and its steps lead the owner to copy a suggestion into AI settings by hand | REQ-doc-coaching, REQ-promote-guidance (criterion 3) | End-user walkthrough | Staging screenshots of the article and the AI settings field after the copy |
| Logs page shows the "Coaching Conversation" section below the transcript | D-27 discretion, UI-SPEC §6 | Browser layout | Staging screenshot |
| `/logs/:connectionId` sends a signed-out visitor to sign-in | D-27 / AR-4-08 | Must be seen as a signed-out visitor | Private window or separate browser profile only. Never sign the owner out. |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
