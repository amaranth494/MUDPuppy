---
phase: 02-autopilot-switch
verified: 2026-09-15T23:11:04Z
status: passed
score: 4/4 roadmap success criteria verified (7/7 plan-level must-have truth clusters verified)
overrides_applied: 0
---

# Phase 2: Autopilot Switch Verification Report

**Phase Goal:** The owner can engage and disengage autopilot from the terminal and always see the true state; taking the wheel is instant and lossless; a disconnect never turns autopilot off and never issues commands, only `#AUTO OFF` turns it off. No AI intelligence exists yet.
**Verified:** 2026-09-15T23:11:04Z
**Status:** passed
**Re-verification:** No — initial verification

## Method

This verification did not trust `02-07-SUMMARY.md`'s claims at face value. For every claim it: (1) read the actual source files implementing the behavior, (2) independently re-ran `go build`, `go vet`, `go test` for the four in-scope packages, `tsc --noEmit`, and both `verify-phase2.sh` self-test modes from a clean shell, (3) cross-checked every line-number citation in `02-07-SUMMARY.md`'s Phase Validation table against the actual byte offsets in `evidence/01-test-report.txt`, `evidence/03-canned-report.txt`, and `evidence/04-staging-ai-player.log`, and (4) opened all nine evidence screenshots with the image reader and visually compared each to what the summary and the 02-UI-SPEC.md copy contract claim it shows.

## Goal Achievement

### Observable Truths (ROADMAP Phase 2 Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `#AUTO ON` engages only past the Phase 1 gate (refused with the gate's message otherwise); `#AUTO OFF` disengages; the indicator always matches the true server-held state, including after a page refresh | VERIFIED | `internal/session/handler.go` `Autopilot()` resolves the gate via `h.callbacks.EngageGate` before every action (line ~329-336), fails closed on nil callback; `cmd/server/main.go:192-206` wires `EngageGate` to `store.EngageGateAllowed` scoped by `GetProfileByConnection(userID, connectionID)`. Re-ran `evidence/03-canned-report.txt` equivalent live: self-test PASS 22/0/2, exit 0 (independently re-run, matches filed report). Screenshot `06-refused-before-acceptance.png` shows the exact refusal sentence in red with badge OFF. Screenshot `05-badge-after-refresh.png` shows badge WAITING post-refresh — verified this is the *true* server state (a page reload closes the websocket, which this app's pre-existing `Disconnect` path treats as a drop, correctly parking autopilot per criterion 3) and not a stale/incorrect read; `10-resume-after-reconnect.png` shows the same switch correctly resuming to ON on the next connect, closing the loop. Badge component (`AutopilotBadge.tsx`) reads only `useSession().autopilotState`, derives nothing locally — confirmed by source read. |
| 2 | Any game command typed by the human while engaged disengages autopilot before the command is sent, with no lost keystrokes; commands fired by browser triggers or timers do not trip the wheel-grab | VERIFIED (see documentation note below) | `internal/session/websocket.go` `applyWheelGrab` runs before `clientToMUD <- wsMsg.Data` unconditionally on every path (command always queued regardless of grab outcome — no lost keystrokes, confirmed by reading the code around the channel send). `IsHumanSource` classifies `trigger`/`timer` as automation and everything else (including absent/garbled) as human — matches RESEARCH Pitfall 3's fail-safe direction. `go test -run TestWheelGrabSourceRule -v` re-run clean: classifier and all 7 decision subtests PASS. Screenshot `07-wheel-grab.png` shows `[Autopilot disengaged: you took the wheel]` followed by the game's reply with badge OFF. Screenshot `08-trigger-keeps-on.png` shows repeated trigger/timer-fired commands with badge remaining ON throughout. Blank-Enter path (`PlayScreen.tsx` line ~482, `wsManager.sendCommand(command + '\n', 'user')`) is explicitly, unconditionally tagged `'user'` — confirmed by source read, satisfying D-07/RESEARCH Pitfall 2. |
| 3 | A disconnect while engaged moves autopilot to a waiting state that issues nothing; the AI never initiates a reconnect; when the connection comes back by any means autopilot resumes to ON by itself; `#AUTO OFF` while waiting lands on OFF and stays OFF after reconnecting | VERIFIED | `internal/session/manager.go` `Disconnect()` calls `m.parkAutopilotLocked(userID)` *before* `delete(m.sessions, userID)` (confirmed by reading lines 300-345) — the autopilot map is a separate userID-keyed map from `m.sessions`, so it survives the deletion (RESEARCH Pitfall 1 correctly closed). `Connect()` calls `m.resumeAutopilotLocked(userID)` only after a successful dial (line 220-221) — the server never dials on its own, satisfying "AI never initiates a reconnect." `go test -run TestWaitingSurvivesDisconnectAndResumesOnConnect -race -v` re-run clean, all 3 subtests PASS including `off_while_waiting_stays_off_after_reconnect`. Screenshot `09-waiting-after-drop.png` shows `[Disconnected]` then `[Autopilot waiting for reconnect]`, badge WAITING. Screenshot `10-resume-after-reconnect.png` shows `[Reconnected]` then `[Autopilot resuming]`, badge ON, nothing typed. Screenshots `11-off-while-waiting-stays-off.png` and `11a-off-typed-while-waiting.png` show `#AUTO OFF` typed while WAITING producing `[Autopilot disengaged]` with badge OFF while still disconnected, and badge still OFF after the subsequent hand reconnect (no resume line, matching `Resume()`'s off-stays-off transition). |
| 4 | The owner can create a game profile, accept the policy on it, and hand-play the character normally with the AI disengaged; nothing about ordinary play changes | VERIFIED | `evidence/03-canned-report.txt` steps 5, and the not-owned-connection IDOR check (step 9) re-verified line-for-line. Screenshot `12-hand-play-unchanged.png` shows ordinary typed commands with badge OFF, no bracketed autopilot lines, no banner — matches D-12 exactly. `internal/session/manager.go` `parkAutopilotLocked`/`resumeAutopilotLocked` are no-ops with no log line when there is no autopilot record for the user (confirmed by source read), so ordinary play never touches the `[AI-PLAYER] autopilot` log path. |

**Score:** 4/4 ROADMAP success criteria verified against the codebase (not merely against the summary's claims).

### Documentation Note (not a functional gap)

ROADMAP.md's Phase 2 Success Criterion 2 text reads: *"...commands fired by browser aliases, triggers, or timers do not trip the wheel-grab."* The actual built and tested behavior treats `alias`-sourced commands as **human** (they *do* trip the wheel-grab) — confirmed by `internal/session/websocket_test.go`'s `alias_source_disengages` subtest and by `IsHumanSource`'s explicit comment "owner typed the line that expanded (D-07)". This is a deliberate, owner-approved decision recorded verbatim in `02-CONTEXT.md` D-07/D-09 ("owner typed it... alias expansion of a typed command", "Yes, it counts as human"), made because in this codebase an "alias" is a substitution of something the owner typed (`automation.ts`: `processUserInput` handles "typed input, alias expansion"), not an automatically-firing macro like a trigger or timer. It fails in the safer direction (more disengages, never a silent bypass). Critically, the actual binding acceptance text for **REQ-wheel-grab** in `.planning/REQUIREMENTS.md` says only *"Any game command typed while engaged disengages before the command is sent, with no lost keystrokes"* — it contains no alias exemption at all, so the higher-precedence requirement is satisfied without qualification. Recommend correcting ROADMAP.md's Phase 2 Success Criterion 2 wording (drop "aliases," or clarify it means alias-expanded *output* commands beyond the first) in a documentation pass; this does not block the phase.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/session/autopilot.go` | Pure 3-state machine, 4 transitions | VERIFIED | 115 lines; `Engage`/`Disengage`/`EnterWaiting`/`Resume` match D-01/D-04 exactly; `WaitingSince` present for future bounded-lifetime remediation |
| `internal/session/autopilot_test.go` | Table tests for every transition pair | VERIFIED | `TestAutopilotTransitions`, 12 subtests, all PASS on re-run |
| `internal/session/manager.go` (extended) | 4th userID-keyed autopilot map, Disconnect/Connect hooks | VERIFIED | `parkAutopilotLocked` called before `delete(m.sessions, userID)`; `resumeAutopilotLocked` called after successful dial; both no-op silently when off (D-12) |
| `internal/session/manager_test.go` | End-to-end Disconnect→Connect proof | VERIFIED | `TestEngageAutopilot`, `TestWaitingSurvivesDisconnectAndResumesOnConnect`, 228 lines, all PASS under `-race` |
| `internal/session/handler.go` (extended) | POST /api/v1/session/autopilot, gate seam, autopilot_state on status | VERIFIED | `Autopilot()` handler present, user identity from context only, gate resolved first, off always allowed |
| `internal/session/handler_test.go` | httptest coverage of refusals/no-ops/IDOR | VERIFIED | 387 lines; `TestAutopilotHandler`, `TestStatusCarriesAutopilotState`, all PASS |
| `internal/session/websocket.go` (extended) | Source field, IsHumanSource, applyWheelGrab | VERIFIED | Wheel-grab runs before the channel send unconditionally; command always queued |
| `internal/session/websocket_test.go` | Classifier + wheel-grab decision tests | VERIFIED | 175 lines; `TestWheelGrabSourceRule`, classifier + 7 decision subtests, all PASS |
| `cmd/server/main.go` (wiring) | Route + EngageGate closure | VERIFIED | `/api/v1/session/autopilot` registered; `EngageGate` closure scoped by `GetProfileByConnection` |
| `frontend/src/services/automation/evaluator.ts` (`case 'AUTO':`) | Directive grammar entry, exact UI-SPEC copy | VERIFIED | All outcomes (`engaged`, `disengaged`, `already-on/off`, `refused-gate`, `refused-no-session`, `status`) produce the exact strings and ANSI colors specified in `02-UI-SPEC.md` |
| `frontend/src/services/automation.ts`, `api.ts` | Source tagging threaded to the websocket | VERIFIED | Blank Enter unconditionally tagged `'user'`; `CommandSource` threaded from `onSubmitCommand` through to `sendCommand` |
| `frontend/src/components/AutopilotBadge.tsx` | 3-state badge, server-derived only | VERIFIED | 54 lines (min 20); reads only `useSession().autopilotState`; no local derivation |
| `frontend/src/context/SessionContext.tsx` | autopilotState + autopilotControl wiring | VERIFIED | Status-poll and websocket-push both feed `autopilotState`; `parkedConnectionIdRef` fix lets `#AUTO OFF` reach the server while WAITING |
| `frontend/src/index.css` | 3 badge state classes | VERIFIED | `.autopilot-badge.state-on/.state-waiting/.state-off` present, structural copy of `.session-badge.status-*` |
| `scripts/verify-phase2.sh` | Canned-report harness, self-test modes | VERIFIED | 535 lines (min 180); independently re-run `--self-test` (22 PASS/0 FAIL, exit 0) and `--self-test-negative` (FAIL C1 x2, exit 1) from a clean shell — both match the filed evidence exactly |
| `scripts/fixtures/phase2/`, `scripts/fixtures/phase2-negative/` | Fixture sets | VERIFIED | Present; exercised successfully by both self-test modes above |
| Evidence files (13, 4 text + 9 PNG) | Per `02-07-PLAN.md` artifact list | VERIFIED | All present on disk; every text-evidence line citation in `02-07-SUMMARY.md` checked against actual file content and found accurate; all 9 PNGs opened and visually confirmed to show the claimed content |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `manager.go Disconnect` | `m.autopilot[userID]` | `parkAutopilotLocked` before `delete(m.sessions, userID)` | WIRED | Confirmed by direct source read, lines 300-345 |
| `manager.go Connect` | `m.autopilot[userID]` | `resumeAutopilotLocked` after successful dial | WIRED | Confirmed by direct source read, lines 190-225 |
| `handler.go Autopilot` | `store.EngageGateAllowed` | `HandlerCallbacks.EngageGate` | WIRED | Nil-safe, fails closed |
| `main.go EngageGate closure` | `profileStore.GetProfileByConnection` | ownership-scoped lookup | WIRED | IDOR-safe; confirmed against `evidence/03-canned-report.txt` step 9 (re-verified) |
| `websocket.go case MsgTypeData` | `manager.DisengageAutopilot` | `applyWheelGrab` before channel send | WIRED | Command queued unconditionally on every path |
| `evaluator.ts case 'AUTO'` | `POST /api/v1/session/autopilot` | `context.autopilotControl.setState` | WIRED | Threaded through `SessionContext.tsx` |
| `automation.ts processCommandQueue` | websocket data message | `onSubmitCommand(cmd.command, cmd.source)` | WIRED | Source no longer dropped |
| `api.ts sendCommand` | `websocket.go case MsgTypeData` | `source` field on the data message | WIRED | Confirmed both ends read/write the same field name |
| `SessionContext.tsx refreshStatus` | `GET /api/v1/session/status autopilot_state` | 15s poll + websocket push | WIRED | Badge re-syncs after refresh, confirmed by `05-badge-after-refresh.png` behavior |
| `Sidebar.tsx` | `<AutopilotBadge />` | direct render under `<SessionBadge />` | WIRED | **Fixed mid-phase** — `Header.tsx` (the plan's originally intended mount point) is dead code, never imported anywhere in the app (confirmed: `grep` for `import Header` / `<Header` returns nothing); the walkthrough correctly caught this and remounted the badge in `Sidebar.tsx`, which `App.tsx` does render |

### Behavioral Spot-Checks (independently re-run by this verifier, not copied from evidence)

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Full build compiles | `go build ./...` | exit 0 | PASS |
| Vet clean on in-scope packages | `go vet ./internal/session/... ./cmd/...` | exit 0, no output | PASS |
| Phase 2 + Phase 1 regression packages green | `go test ./internal/session/... ./internal/store/... ./internal/profiles/... ./internal/policy/...` | all `ok` | PASS |
| Pre-existing baseline failure still isolated to `icm` | `go test ./internal/icm/...` | `FAIL TestHandlerRegistration/CANCEL` only | PASS (matches documented, excused baseline) |
| Frontend type-checks clean | `cd frontend && npx tsc --noEmit` | exit 0, no output | PASS |
| Harness proves itself clean | `bash scripts/verify-phase2.sh --self-test <tmp>` | 22 PASS/0 FAIL/2 SKIP, exit 0 | PASS |
| Harness proves it can fail | `bash scripts/verify-phase2.sh --self-test-negative <tmp>` | FAIL C1 x2, TOTAL FAILURES: 2, exit 1 | PASS |

### Data-Flow Trace (Level 4)

`AutopilotBadge.tsx` → `useSession().autopilotState` → `SessionContext.tsx` state, populated by two real sources: (a) `GET /api/v1/session/status` poll response `autopilot_state` field (traced to `handler.go Status()` → `manager.AutopilotStateFor(userIDStr)`, a real map read, not a static value), and (b) the websocket `autopilot` push message (traced to `websocket.go`'s `MsgTypeAutopilot` writes at `parkAutopilotLocked`/`resumeAutopilotLocked`/`applyWheelGrab` call sites). No hardcoded/static fallback found. Status: FLOWING.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| REQ-autopilot-directives | 02-01, 02-02, 02-04, 02-05, 02-06 | `#AUTO ON/OFF/STATUS`, gate-gated engage, true-state indicator | SATISFIED | Criteria 1 above; `.planning/REQUIREMENTS.md` line 19 (currently unchecked/"Pending" in the traceability table — bookkeeping only, see note below) |
| REQ-wheel-grab | 02-03, 02-04, 02-06 | Typed command disengages before send, no lost keystrokes | SATISFIED | Criterion 2 above; `.planning/REQUIREMENTS.md` line 20 (already checked/"Complete") |
| REQ-no-auto-reconnect | 02-01, 02-02, 02-05, 02-06 | Disconnect parks, AI never reconnects, resume on connect, `#AUTO OFF` sticks | SATISFIED | Criterion 3 above; `.planning/REQUIREMENTS.md` line 21 (currently unchecked/"Pending" — bookkeeping only) |
| REQ-doc-hand-play-and-gate | 02-02, 02-05, 02-06 | Create profile, accept policy, hand-play unchanged; AI ungatable pre-acceptance | SATISFIED | Criterion 4 above; `.planning/REQUIREMENTS.md` line 60 maps this to REQ-policy-gate (Phase 1, already Complete) + REQ-autopilot-directives (this phase) |

No orphaned requirements found for Phase 2 — all four IDs declared in ROADMAP.md's `Requirements:` line appear in at least one plan's `requirements:` frontmatter field and are covered above.

**Documentation-bookkeeping note (not a functional gap):** `.planning/REQUIREMENTS.md`'s checkbox list (lines 19-21) and its Traceability table (lines 96-99) show REQ-autopilot-directives, REQ-no-auto-reconnect, and REQ-doc-hand-play-and-gate as unchecked/"Pending" despite this phase's evidence supporting all four IDs; REQ-wheel-grab alone is marked checked/"Complete". This is a stale-document artifact — REQUIREMENTS.md's status markers appear to be updated at phase-close time, which had not yet happened at verification time — not evidence of missing implementation. Recommend updating these markers when the phase is closed.

### Anti-Patterns Found

None. Scanned every file touched by this phase (`internal/session/*.go`, `cmd/server/main.go`, the five frontend files from 02-04, the five frontend files from 02-06, `scripts/verify-phase2.sh`) for `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER`/"coming soon"/"not yet implemented"/empty-return stubs. Zero matches beyond narrative code comments explaining design rationale (not markers of incomplete work). No hardcoded-empty-render patterns found in `AutopilotBadge.tsx` or `PlayScreen.tsx`'s autopilot code paths.

**Known, pre-existing quirk (documented in 02-07-SUMMARY.md, independently confirmed):** `internal/session/manager.go`'s `EngageAutopilot`/`DisengageAutopilot` log the *stored* `AutopilotRecord.ConnectionID` (`curConnID`), not the connection ID from the current request, for `refused-no-session` and `already-off` causes — confirmed at `manager.go` lines 406-409 and 441-444. This is a minor diagnostic-log-attribution issue, not a security or correctness defect (the returned state and HTTP response are always correct); the summary correctly flags it as a suggested future fix rather than papering over it.

### Human Verification Required

None. This verifier independently re-ran every automated check named in the phase's validation strategy and personally inspected all nine evidence screenshots against their claims (Steps required by the task). No player-observable behavior remains unconfirmed.

### Gaps Summary

No blocking gaps found. Two non-blocking documentation notes are recorded above (ROADMAP.md Success Criterion 2's imprecise "aliases...do not trip" phrasing vs. the deliberate, REQUIREMENTS.md-compliant, owner-approved built behavior; and REQUIREMENTS.md's stale Pending/Complete checkboxes). Neither affects the phase's actual delivered capability. All four ROADMAP Phase 2 success criteria are independently verified against running code, re-executed tests, and visually-confirmed screenshots — not merely against SUMMARY.md's narrative. The three mid-walkthrough bug fixes documented in `02-07-SUMMARY.md` (badge never mounted because `Header.tsx` is dead code; `[Disconnected]` not firing on user-initiated drop; `#AUTO OFF` unreachable while parked) were verified as genuinely fixed by reading the current source, not merely trusted from the summary's narrative.

---

*Verified: 2026-09-15T23:11:04Z*
*Verifier: Claude (gsd-verifier)*
