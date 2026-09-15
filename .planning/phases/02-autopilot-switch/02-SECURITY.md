---
phase: 02
slug: autopilot-switch
status: pending-owner-decisions
threats_open: 10
asvs_level: 1
created: 2026-09-15
---

# Phase 02 — Autopilot Switch — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.
> Register authored at plan time (register_authored_at_plan_time: true) across 02-01-PLAN.md through 02-07-PLAN.md. Verified against implemented code, not documentation, on 2026-09-15. Three critical code-review findings (CR-01, CR-02, CR-03) were fixed post-review in commit `b5c8bf5`; the fixes are verified in code below, not merely cited from the fix commit's message.
>
> **This audit closes every threat whose plan disposition is `mitigate` and finds the declared mitigation present in code.** It does **not** close threats dispositioned `accept` or `defer to security review` — those, plus the code-review's still-open warnings and the two carried-forward Phase 1 risks, are listed as **open-for-decision** for the owner at the Phase 2 security review, per `02-SECURITY-AGENDA.md`. Nothing below pre-decides them.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|----------------|
| Websocket/REST goroutines → `Manager` maps | Several goroutines read and write the autopilot map concurrently with `Connect`/`Disconnect` | autopilot state, connection id |
| Session lifecycle → autopilot lifecycle | The session map is deleted on every disconnect; autopilot state must not share that lifetime | autopilot state |
| Go process → server log | `[AI-PLAYER] autopilot` lines and the canned-report/staging log excerpt are readable by anyone with deploy-log access | user id, connection id, old/new state, cause |
| Browser HTTP request → session handler | Client JSON reaches `AutopilotRequest`; any field present is client-controlled | action, connection_id |
| Session handler → profile store | The engage-gate decision depends on a lookup that must be scoped to the caller and to the live session's own connection | user id, connection id, policy acceptance |
| Browser websocket → session websocket handler | Every field on an inbound `data` message, including `source`, is client-controlled and untrusted | source flag, command text |
| Websocket handler → browser | The outbound `autopilot` push reaches whoever holds that connection | state, cause |
| Owner's typed line → automation engine | Anything typed, including `#` directives, is owner input the engine classifies as human or automation | command text, source label |
| Server response → terminal output | The gate refusal sentence and outcome copy are server-supplied text rendered in the owner's terminal | refusal sentence, outcome |
| Server status and push → badge | The badge's only inputs; the browser must not substitute its own guess | autopilot_state, autopilot_connection_id |
| Operator shell → deployed server | `scripts/verify-phase2.sh` carries a live session cookie and drives authenticated writes against whatever `BASE_URL` names | SESSION_COOKIE, BASE_URL |
| Repository → Railway staging | The deployed binary may diverge from the repo; the canned report and the log are what reconcile them | git SHA, deploy state |
| Report/log/screenshot files → shared evidence | Evidence is committed or filed into the repo; anything it prints or shows is disclosed | request/response bodies, terminal state |

---

## Threat Register

| Threat ID | Category | Severity | Component | Disposition | Status | Evidence |
|-----------|----------|----------|-----------|-------------|--------|----------|
| T-2-01 | Spoofing / Tampering | High | client-supplied `source` flag; `IsHumanSource`/`applyWheelGrab` (`internal/session/websocket.go`); frontend labelling (`automation.ts`, `PlayScreen.tsx`) | mitigate | **CLOSED** | `IsHumanSource` (`websocket.go`) returns `false` only for `trigger`/`timer`, `true` for everything else including empty/garbled; `applyWheelGrab` contains exactly one `DisengageAutopilot` call and zero `EngageAutopilot`/resume calls. `TestWheelGrabSourceRule` (`websocket_test.go`) covers `off_stays_off`, `waiting_is_not_grabbed`, `second_user_untouched`, garbled-source — all PASS in `evidence/01-test-report.txt`. Frontend: blank-Enter path explicitly tags `'user'` (`PlayScreen.tsx`), directive-emitted commands tagged `'trigger'` (`automation.ts:523,551`, comment naming D-08). Live proof: `evidence/07-wheel-grab.png`, `evidence/08-trigger-keeps-on.png`, `cause=wheel-grab` in `evidence/04-staging-ai-player.log`. |
| T-2-02 | Elevation of Privilege (IDOR) | High | `Autopilot` handler, `EngageGate` closure (`cmd/server/main.go`), `applyWheelGrab` user arg | mitigate | **CLOSED** | `AutopilotRequest` has no user field; handler reads user id only from `r.Context().Value("user_id")` (`handler.go:309`). `EngageGate` closure calls `profileStore.GetProfileByConnection(userID, connectionID)`, scoped on both columns (`main.go`). `TestAutopilotHandler`'s `body_cannot_name_another_user` and `websocket_test.go`'s `second_user_untouched` PASS in `evidence/01-test-report.txt`. Live: `evidence/03-canned-report.txt` step 9 — not-owned connection id answers `refused-gate` (`PASS C4: a not-owned connection_id is refused`). |
| T-2-03 (mitigate scope: `StatusResponse.AutopilotState`, outbound `autopilot` push) | Information Disclosure | Medium | `handler.go Status()`, `websocket.go` outbound push | mitigate | **CLOSED** | Both are derived from `AutopilotStateFor(userIDStr)`/the authenticated connection only; no endpoint accepts a user id argument. `StatusResponse` field confirmed at `handler.go:88-90` (now also carries `AutopilotConnectionID`, `json:"autopilot_connection_id,omitempty"`, populated at `handler.go:290`). |
| T-2-03 (residual scope: the `autopilot` websocket message shape, plan 02-04) | Information Disclosure | Low | outbound `autopilot` websocket message | **accept** | **OPEN-FOR-DECISION** | Plan 02-04 accepted this rather than mitigating it. Carried on `02-SECURITY-AGENDA.md`'s carry-forward table for confirmation, not re-litigation: "the handler receives only a state string and a cause for a connection the browser already owns; no user identifier crosses this channel." Not closed by this audit — owner decision required (Accept / Defer / Remediate Now). |
| T-2-04 | Information Disclosure | Medium | `[AI-PLAYER] autopilot` log lines (`manager.go`, `handler.go`, `websocket.go`) | mitigate | **CLOSED** | `logAutopilotTransition` (`manager.go:381-384`) emits only `user_id=%s connection_id=%s old=%s new=%s cause=%s`; no `wsMsg.Data`, command text, conduct/guidance text interpolated anywhere in the three files. Staging log evidence: `evidence/04-staging-ai-player.log` — `grep -ci 'conduct_rules\|approach_guidance\|you took the wheel'` returns 0; all 9 expected causes present (`refused-gate`, `refused-no-session`, `engage`, `wheel-grab`, `disconnect`, `resume`, `already-off`, `already-on`, `disengage`). |
| T-2-05 | Tampering | Medium | `EngageAutopilot` on an already-on switch | mitigate | **CLOSED** | `EngageAutopilot` (`manager.go:421-459`): when `!changed` the existing record is not rewritten (only a log line with `curConnID` is emitted; no new `AutopilotRecord` allocated). `repeat_engage_is_a_no_op` (`manager_test.go`) and `repeat_on_is_already_on` (`handler_test.go`) PASS in `evidence/01-test-report.txt`. |
| T-2-06 | Spoofing (a badge that lies) | High | `AutopilotBadge.tsx`, `SessionContext.tsx` | mitigate | **CLOSED** | `AutopilotBadge.tsx` reads only `useSession().autopilotState`; `grep -c 'connectionState\|useState\|setInterval'` on the file returns 0 — no local derivation, no polling of its own. Live: `evidence/05-badge-after-refresh.png` shows the badge correct with no action taken after a hard reload. |
| T-2-07 | Repudiation (unfalsifiable evidence) | High | `scripts/verify-phase2.sh` and the staging canned report | mitigate | **CLOSED** | `--self-test-negative` produces `FAIL C1` (x2) and exit 1 — confirmed in `evidence/02-harness-selftest.txt` lines 140-141 (`FAIL C1: outcome is refused-gate (expected: refused-gate, got: engaged)`). Clean self-test shows zero `FAIL` lines and exit 0. |
| T-2-08 | Policy risk (not STRIDE) | Medium | waiting → on auto-resume with no time bound | **defer to security review** | **OPEN-FOR-DECISION** | `AutopilotRecord.WaitingSince` is stored on every WAITING transition (`autopilot.go`) so a bounded lifetime can be added without redesign, but no bound exists today. Written onto `02-SECURITY-AGENDA.md` Item 1 with all three dispositions unmarked. Not closed by this audit — owner decision required. |
| T-2-09 | Denial of Service (deadlock / data race) | Medium | the autopilot map under `m.mu` | mitigate | **CLOSED** | All access (`AutopilotStateFor`, `EngageAutopilot`, `DisengageAutopilot`, `parkAutopilotLocked`, `resumeAutopilotLocked`) is through methods that take `m.mu`; `EngageAutopilot` reads `m.sessions` inline rather than calling the RLock-taking `GetSession`. Package tested under `-race`: `evidence/01-test-report.txt`'s `-race` section contains no `DATA RACE` line. |
| T-2-10 | Spoofing (fail-open dependency) | Medium | nil/missing `EngageGate` callback | mitigate | **CLOSED** | `handler.go:341`: `if h.callbacks != nil && h.callbacks.EngageGate != nil { ... }` — `gateAllowed` stays at its zero value (`false`) otherwise, and the `"on"` branch checks `!gateAllowed` before ever calling `EngageAutopilot`, so a missing dependency fails closed. `nil_gate_callback_refuses` subtest PASS in `evidence/01-test-report.txt`. |
| T-2-11 | Tampering (input validation) | Medium | the `action` field | mitigate | **CLOSED** | `handler.go:328-332`: action lowercased and checked against the exact allowlist `on`/`off`/`status`; anything else is `http.Error(..., 400)` with no fallback branch. Live: `evidence/03-canned-report.txt` — `PASS C1: an invalid action is rejected with HTTP 400`. |
| T-2-12 | Denial of Service (lost keystrokes) | Medium | code between the guards and the `clientToMUD` send | mitigate | **CLOSED** | `applyWheelGrab`'s result only conditionally triggers a best-effort `writeJSON` push (`websocket.go:421-422`); the code review (`02-REVIEW.md`) found the `clientToMUD` select unconditional and unchanged on every path — confirmed structurally, no new blocking call on the command path. |
| T-2-13 | Tampering (self-cancelling directive) | Medium | a typed `#` directive that emits game commands | mitigate | **CLOSED** | `case 'AUTO':` (`evaluator.ts:1162-1164`) is side-effect only: "no command is ever emitted here, so a typed #AUTO line can never itself reach the MUD or disengage autopilot (D-08)." Any command a directive does emit is labelled `'trigger'` via `isDirectiveInput` (`automation.ts:523,551`). |
| T-2-14 | Spoofing (refusal wording drift) | Low | engage-gate refusal rendered in the terminal | mitigate | **CLOSED** | `grep -c 'AI Player has not been configured' frontend/src/services/automation/evaluator.ts` returns 0 — the frontend holds no copy of the sentence and renders only the server's `gate_message`. |
| T-2-15 | Repudiation (a skip read as a pass) | Medium | C2/C3 in the canned-report harness | mitigate | **CLOSED** | `_skip` maintains a separate counter from PASS/FAIL; `evidence/03-canned-report.txt` shows `SKIP C2` and `SKIP C3` each naming their screenshot/log evidence, and the closing summary reads `TOTAL CHECKS: 22  FAILURES: 0  SKIPPED: 2` — skips are never folded into the PASS count. |
| T-2-16 | Information Disclosure | Medium | canned-report contents | mitigate | **CLOSED** | `SESSION_COOKIE` appears in `scripts/verify-phase2.sh` only inside `curl -b "$SESSION_COOKIE"` invocations and usage/comment text; `grep -n 'echo.*SESSION_COOKIE\|printf.*SESSION_COOKIE'` returns no match — it is never echoed into the report. |
| T-2-17 | Tampering (wrong target) | Medium | `BASE_URL` pointed at production | mitigate | **CLOSED** | `scripts/verify-phase2.sh` records `git rev-parse --short HEAD` and `BASE_URL` in the report header; confirmed present in `evidence/03-canned-report.txt` lines 1-8 (`BASE_URL: https://mudpuppy-staging.up.railway.app`, `git rev-parse --short HEAD: 7a70ab8`). |
| T-2-18 | Repudiation (a notice printed twice or never) | Medium | the transition-notice printing paths (`PlayScreen.tsx`) | mitigate | **CLOSED** | The four literal notices (`[Autopilot disengaged: you took the wheel]`, `[Autopilot waiting for reconnect]`, `[Reconnected]`, `[Autopilot resuming]`) each appear exactly once in `PlayScreen.tsx`, each through `automationEngine.echoLocal`; on-to-off/waiting-to-off are left to the directive/wheel-grab paths per design. |
| T-2-19 | Information Disclosure (a notice reaching the game) | Medium | the bracketed transition lines | mitigate | **CLOSED** | Every added literal is written through `automationEngine.echoLocal`, the same local-only mechanism that already guarantees `[Disconnected]` never reaches the MUD — confirmed by source read of `PlayScreen.tsx`. |
| T-2-20 | Repudiation (carried-forward risks forgotten) | Medium | Phase 1's R-02 and R-04 | mitigate | **CLOSED** (agenda exists; risk *decisions* are open — see below) | `02-SECURITY-AGENDA.md` Items 2 and 3 reproduce R-02 and R-04 with their original remediation wording verbatim and state the project cannot close while either remains deferred. This threat's mitigation is "the risks are not forgotten," which the agenda satisfies; the risks *themselves* are not decided and are listed under Open Items below. |
| T-2-SC | Tampering (supply chain) | Low | `go.mod`, `frontend/package.json`, `jq` | **accept** | **OPEN-FOR-DECISION** | Verified true as a factual matter — `evidence/01-test-report.txt`'s `### DEPENDENCY DRIFT` section: `git diff --stat 628a038 -- go.mod frontend/package.json` produced empty output — but the disposition is `accept`, and per this audit's instructions an `accept` disposition is not closed by the auditor. Carried on `02-SECURITY-AGENDA.md`'s carry-forward table for confirmation, not re-litigation. Owner decision required. |

*Status: CLOSED (mitigation verified in code) · OPEN (mitigation absent — none found this audit) · OPEN-FOR-DECISION (accept/defer disposition; owner has not yet recorded a decision)*
*Disposition: mitigate (implementation required) · accept (documented risk) · defer to security review*

### Fixes verified for the three CR-0x critical code-review findings (commit `b5c8bf5`)

These are not separate T-2 register entries — they are code-review findings against the T-2-02/T-2-06/T-2-01 mitigation surface, fixed after the review and verified here directly in code, not from the fix commit's message:

- **CR-01** (parked switch resumed onto any future connection, no gate re-check) — `resumeAutopilotLocked(userID, connectionID)` (`manager.go:521-541`) now refuses to resume unless the reconnecting `connectionID` matches `rec.ConnectionID`; a mismatch (including empty, i.e. quick connect) lands the switch on `off` with `cause=connection-changed`, logged. `Connect` (`manager.go:228`) passes the dialled `connectionID` through.
- **CR-02** (`#AUTO ON`'s gate not bound to the live session's connection) — `Session` gained `ConnectionID` (`manager.go:37-45`, set at `manager.go:197`); `EngageAutopilot` (`manager.go:441-444`) refuses with `ErrWrongConnection`/`refused-wrong-connection` when `req.ConnectionID` does not match `session.ConnectionID`. Wired through `handler.go:368-369` and rendered by `evaluator.ts:1214` (`case 'refused-wrong-connection':`).
- **CR-03** (`#AUTO OFF` unreachable after refresh while waiting) — `AutopilotConnectionIDFor` (`manager.go:390-397`) added; `StatusResponse.AutopilotConnectionID` (`handler.go:88-90,290`) carries it while the switch is on or waiting; `SessionContext.tsx:236-237` seeds `parkedConnectionIdRef` from it on every status refresh.

All three are also exercised by tests per `internal/session/manager_test.go` and `handler_test.go` (wrong-profile engage, quick-connect/other-profile reconnect landing off, and the status field), confirmed present in `evidence/01-test-report.txt`.

---

## Accepted Risks Log

*No risks have been formally accepted yet.* This phase's `accept`- and `defer`-dispositioned threats (T-2-SC, T-2-03 residual, T-2-08) and the items in "Open items for the owner's decision" below await the owner's Accept / Defer / Remediate Now choice at the Phase 2 security review (`02-SECURITY-AGENDA.md`). This auditor does not pre-decide them. Once decided, populate this log the way `01-SECURITY.md`'s Accepted Risks Log was populated after the Phase 1 review.

---

## Open Items for the Owner's Decision

These are not closed by this audit. Each carries the disposition options **Accept / Defer / Remediate Now** (register items) or is a code-review finding awaiting a remediate-or-accept decision. None is recommended here.

### From the STRIDE register (accept / defer dispositions, see table above for evidence)

| Threat ID | Criticality | One-line risk | Proposed remediation |
|-----------|-------------|----------------|------------------------|
| T-2-08 | Medium | An autopilot parked by a dropped connection resumes to ON by itself on reconnect, with no time bound and no owner confirmation, adjacent to but not squarely inside Safety-and-Abuse-Policy §2's "you must not re-engage until you understand why it was disconnected." | Bounded WAITING lifetime (owner sets the bound) after which the switch lands OFF instead of resuming; `AutopilotRecord.WaitingSince` already exists to support this with no redesign. |
| T-2-SC | Low | Confirm no package was added anywhere in Phase 2, so the supply-chain surface is unchanged. | None needed if confirmed; re-verify `git diff --stat <phase-base> -- go.mod frontend/package.json` is empty (it is, per `evidence/01-test-report.txt`). |
| T-2-03 (residual) | Low | The outbound `autopilot` websocket message carries a state and cause for a connection the browser already owns; confirm no user identifier ever crosses this channel, including if the message shape changes later. | None needed if confirmed true today; re-confirm at any future change to the websocket message shape. |

### Carried forward from the Phase 1 security review (`.planning/todos/pending/2026-09-15-deferred-security-risks.md`, wording verbatim)

| Risk ID | Criticality | One-line risk | Proposed remediation (verbatim) |
|---------|-------------|-----------------|-----------------------------------|
| R-02 | Medium | "Invalid numeric input in the AI Player panel silently saves as blank ('no cap' / 'default') instead of a validation error. `frontend/src/components/AIPlayerPanel.tsx` handleSave." | "Parse each numeric field before building the request; if non-empty and not a finite number, block the save with an inline error naming the field." |
| R-04 | Medium | "Staging prints the one-time login code in the deploy log (`STAGING: OTP sent to user, code: NNNNNN`); anyone with staging log access can sign in as any staging user. Pre-existing, staging only." | "Gate the code-logging line behind an explicit env flag that is off by default, or log a hash; confirm production has no equivalent; review Railway log access on the staging project." |

Phase 2 did not touch `AIPlayerPanel.tsx` and built no fix for R-02. Phase 2's own evidence capture (`evidence/04-staging-ai-player.log`) used `railway logs --environment staging` — exactly the access path R-04 describes; no OTP was disclosed in the filtered excerpt, but the access path itself remains open.

### From the Phase 2 code review (`02-REVIEW.md`) — not yet remediated

| Finding | Criticality | One-line risk | Proposed remediation |
|---------|-------------|-----------------|-------------------------|
| WR-01 | Medium | `applyWheelGrab` (`websocket.go:69-81`) reads `AutopilotStateFor` and separately calls `DisengageAutopilot` under two independent lock acquisitions — a check-then-act race against a concurrent REST `#AUTO OFF` or `#AUTO ON`. Worst case today is benign (a skipped push or a harmless no-op), but it is the one place in the package that departs from the single-lock read-modify-write pattern used everywhere else. Verified still present in current code (two separate calls, `websocket.go:73,77,79`). | Add a `Manager.WheelGrab(userID string) (grabbed bool, state AutopilotState)` performing the read-and-maybe-disengage atomically under one lock. |
| WR-02 | Low | `WSMessage.source` is typed as a bare `string` in `frontend/src/types/index.ts:62`, not the `CommandSource` union, even though the field gates a safety-relevant server decision (`IsHumanSource`). Verified still `source?: string` in current code. | Type it as `source?: CommandSource` imported from `services/automation`. |
| WR-03 | Low | D-08's "typed `#` directive relabels its own output as automation" rule (`automation.ts`) is scoped to the whole input line, not to specific commands; it is safe today only because `#IF` is blocked from CLI, with no regression test enforcing that invariant as new `#` commands are added. | Scope the relabelling to known side-effect-only commands, or add a regression test that fails if any future CLI-reachable directive emits a game command while `isDirectiveInput` is true. |
| IN-01 | Low | `#AUTO`'s outcome switch (`evaluator.ts`) silently no-ops on an unrecognised `outcome` value — a future contract drift would print nothing to the owner. | Emit a generic `[Autopilot: unexpected response]` line (and/or console log) in the `default` branch. |
| IN-02 | Low | `disconnect()` in `SessionContext.tsx` triggers `refreshStatus()` twice (once via the synchronous `notifyDisconnect` → `handleDisconnectRef` path, once directly) — harmless, but undocumented as intentional. | No functional fix required; add a comment noting the double-call is intentional so a future edit doesn't "simplify" it into a race. |

**The project cannot close while any item in this section, or in `02-SECURITY-AGENDA.md`, remains undecided/deferred** (per `02-SECURITY-AGENDA.md`'s closing line and the Phase 1 precedent).

---

## Security Audit Trail

| Audit Date | Threats Total (register rows) | Closed | Open-for-decision | Open (mitigation absent) | Run By |
|------------|-------------------------------|--------|--------------------|-----------------------------|--------|
| 2026-09-15 | 22 (21 unique T-2 IDs, T-2-03 split across two dispositions) | 19 | 3 (T-2-08, T-2-SC, T-2-03 residual) | 0 | gsd-security-auditor |

Items still awaiting an owner decision, counted in `threats_open` (10 total): 3 register items above + R-02 + R-04 + WR-01 + WR-02 + WR-03 + IN-01 + IN-02.

---

## Sign-Off

- [x] All 22 register rows have a disposition (mitigate / accept / defer to security review)
- [x] Every `mitigate` threat's declared mitigation was located in the implemented code, not inferred from documentation or code structure (19/19 closed with cited evidence)
- [x] The three post-review critical fixes (CR-01, CR-02, CR-03, commit `b5c8bf5`) were independently re-verified in current code, not accepted from the commit message
- [ ] Accepted risks documented in Accepted Risks Log — **pending**, awaiting owner decisions at the Phase 2 security review
- [ ] `threats_open: 0` — **not confirmed**; 10 items await an owner decision (see Open Items above and `02-SECURITY-AGENDA.md`)
- [ ] `status: verified` — **not set**; frontmatter reads `pending-owner-decisions`
- [ ] Deferred/accepted items re-raised and decided at the Phase 2 security review

**Approval:** pending — this document records verification of implemented mitigations only. It does not close the phase. The Phase 2 security review (owner) must record Accept / Defer / Remediate Now against every row in "Open Items for the Owner's Decision" and against T-2-08, T-2-SC and the T-2-03 residual before `status` can move to `verified` and `threats_open` can move to `0`.
