---
phase: 02-autopilot-switch
document: security-review-agenda
status: pending-review
created: 2026-09-15
---

# Phase 2 Security Review Agenda

This document is the input to the Phase 2 security review, not its output. It decides nothing. It carries one new policy question this phase raised and two risks carried forward from the Phase 1 security review, each with the same three dispositions available: **Accept**, **Defer**, **Remediate Now**. None is marked as chosen and none is recommended — that choice belongs to the owner at the review. The project cannot close while any item on this agenda remains deferred.

---

## Item 1: Waiting-to-on auto-resume (new this phase)

**Behaviour, stated plainly:** an autopilot switch parked at WAITING by a dropped connection returns to ON by itself when the connection comes back — by any means, not only a deliberate reconnect click — with no owner action and no time bound. (`DEC-autopilot-waits-across-disconnect`, D-01.)

**Policy section 2, quoted verbatim** (`.specify/specs/safety-and-abuse-policy-v1.md`):

> ## 2. Supervise the AI
>
> - The AI plays under your supervision. You are expected to be reachable while it plays and to check on its behavior at reasonable intervals.
> - Do not leave the AI running unattended for extended periods. If you need to step away for long, disengage it.
> - If the AI is disconnected or kicked from a game while engaged, it will not reconnect on its own, and you must not re-engage it until you understand why it was disconnected.

**The tension, in one sentence:** the policy addresses the owner re-engaging after a disconnect, and the system now does something adjacent automatically — resuming the switch to ON on reconnect without the owner ever re-typing `#AUTO ON` or otherwise confirming they understand why the disconnect happened.

**What was deliberately not built:** Phase 2 built no time bound on the WAITING state. `AutopilotRecord.WaitingSince` (`internal/session/autopilot.go`) is already stored on every WAITING transition, so a bounded waiting lifetime can be added later without redesign — this was a deliberate scoping choice, not an oversight (see `02-RESEARCH.md` Open Question 1, resolved by deferring the decision to this review).

**Candidate remediation, should Remediate Now be chosen:** a bounded WAITING lifetime, after which the switch lands on OFF instead of resuming on reconnect. The bound itself (minutes, hours, or some other unit) is left to the owner to set; nothing in Phase 2 assumes a value.

**Cross-reference:** T-2-08 (Policy risk, not a STRIDE category) — `02-07-PLAN.md` threat register, disposition "defer to security review."

**Dispositions:**
- [ ] **Accept** — the auto-resume behaviour stands as built, with no time bound.
- [ ] **Defer** — carry this item to a later security review without deciding now.
- [ ] **Remediate Now** — build the bounded WAITING lifetime described above; becomes its own plan or its own phase.

---

## Item 2: R-02 — invalid numeric input silently saves as blank (carried forward from Phase 1)

**Risk, reproduced verbatim from `.planning/todos/pending/2026-09-15-deferred-security-risks.md`:**

> Invalid numeric input in the AI Player panel silently saves as blank ("no cap" / "default") instead of a validation error. `frontend/src/components/AIPlayerPanel.tsx` handleSave.

**Proposed remediation, reproduced verbatim:**

> Parse each numeric field before building the request; if non-empty and not a finite number, block the save with an inline error naming the field.

**Status this phase:** Phase 2 did not touch `frontend/src/components/AIPlayerPanel.tsx` and built no fix for this risk.

**Dispositions:**
- [ ] **Accept** — the silent-blank-save behaviour stands as built.
- [ ] **Defer** — carry this item to a later security review without deciding now.
- [ ] **Remediate Now** — build the remediation described above; becomes its own plan or its own phase.

---

## Item 3: R-04 — staging deploy log prints the one-time sign-in code (carried forward from Phase 1)

**Risk, reproduced verbatim from `.planning/todos/pending/2026-09-15-deferred-security-risks.md`:**

> Staging prints the one-time login code in the deploy log (`STAGING: OTP sent to user, code: NNNNNN`); anyone with staging log access can sign in as any staging user. Pre-existing, staging only.

**Proposed remediation, reproduced verbatim:**

> Gate the code-logging line behind an explicit env flag that is off by default, or log a hash; confirm production has no equivalent; review Railway log access on the staging project.

**Status this phase:** Phase 2 built no fix for this risk. Notably, this plan's own evidence capture (Task 02-07-02) read the staging deploy log via the Railway CLI to produce `evidence/04-staging-ai-player.log` — that is exactly the access path this risk describes. No OTP code was disclosed by this phase's evidence (the log was filtered to `[AI-PLAYER]` lines only), but the access path itself — anyone with `railway logs --environment staging` access can also see any OTP line printed in the same stream — remains open.

**Dispositions:**
- [ ] **Accept** — the OTP-in-log behaviour stands as built, staging only.
- [ ] **Defer** — carry this item to a later security review without deciding now.
- [ ] **Remediate Now** — build the remediation described above; becomes its own plan or its own phase.

---

## Carry-Forward Table: Threats Accepted at Plan Level (confirmation only, not re-litigation)

These Phase 2 threats were dispositioned as `accept` inside their originating plans rather than mitigated. They are listed here so the review sees them in one place. The review's job for this table is to confirm the acceptance still holds, not to re-decide it from scratch.

| Threat ID | Component | Accepted at | What to confirm |
|-----------|-----------|-------------|------------------|
| T-2-SC | `go.mod`, `frontend/package.json` (supply chain) | Plan 02-01, re-verified every subsequent plan | No package was installed anywhere in Phase 2 — confirm `git diff --stat <phase-base> -- go.mod frontend/package.json` is empty. `evidence/01-test-report.txt`'s `### DEPENDENCY DRIFT` section (empty) is the citable proof. |
| T-2-03 (residual, plan 02-04) | The new `autopilot` websocket handler (Information Disclosure, Low) | Plan 02-04 | The handler receives only a state string and a cause for a connection the browser already owns; no user identifier crosses this channel. Confirm this remains true if the websocket message shape changes in a later phase. |

---

## Closing Note

**Phase 3 must re-confirm T-2-01** (the wheel-grab source flag, spoofing/tampering) at its own security review. Phase 2 has no AI decisions yet — the actual harm of a forged source flag in Phase 2 alone is limited to "autopilot stays ON when the owner expected it to disengage." Starting Phase 3, an engaged autopilot begins issuing real commands, and the consequence of a forged flag changes materially: the same spoof could then mean an unwanted command actually executes rather than merely failing to interrupt an idle switch. This is not decided here; it is a flag for the next phase's own review to pick up.

---

**The project cannot close while any item on this agenda remains marked Defer.**
