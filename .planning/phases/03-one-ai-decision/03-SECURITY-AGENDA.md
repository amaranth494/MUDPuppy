---
phase: 03-one-ai-decision
document: security-review-agenda
status: pending-review
created: 2026-09-16
---

# Phase 3 Security Review Agenda

This document is the input to the Phase 3 security review, not its output. It decides nothing. Six open items carried from planning, twelve items raised live during today's staging walkthrough, and two closures for the owner to confirm are listed below, each open item with the same three dispositions available: **Accept**, **Defer**, **Remediate Now**. None is marked as chosen and none is recommended — that choice belongs to the owner at the review. The project cannot close while any item on this agenda remains deferred.

---

## Part A — Open items identified during planning

### Item 1: Transcript and window retention against the policy's data handling

**What is now stored, stated plainly:** every saved-profile connection's full game traffic from connect to disconnect is stored, including other players' speech and including hand play with no AI involved (D-14); and, separately, the game-text window the model saw on each decision is stored on the decision row (`ai_decisions.window_text`, D-12).

**Policy section 6, quoted verbatim** (`.specify/specs/safety-and-abuse-policy-v1.md`):

> ## 6. Credentials and data
>
> - Game credentials stored in a profile are used only to connect that profile. The AI does not use stored credentials for any other purpose.
> - Game text captured during play, including other players' words, is stored in session logs and learned notes for this tool's operation. Do not use it to profile or target other players.

**The tension, in one sentence:** the policy anticipates game text being stored "for this tool's operation" but says nothing about how long, and this phase stores it — including third parties' words captured incidentally — with no retention limit, no deletion path, and no export, for every connection indefinitely.

**What exists today:** no retention limit, no deletion path, and no export exist for either the transcript tables or the decision rows. The decisions endpoint deliberately does not serve `window_text` to the browser (it is written to the row but never read back over the REST reload path), so the retention question is about storage, not about display.

**Candidate remediation, should Remediate Now be chosen:** a retention window after which transcript and decision-window lines are deleted. The window's length itself is left to the owner; nothing in this phase assumes a value.

**Cross-reference:** T-3-09, T-3-15 (Information Disclosure (policy)) — dispositioned "defer to security review" in the plan-level threat registers.

**Dispositions:**
- [ ] **Accept** — full, unbounded retention stands as built.
- [ ] **Defer** — carry this item to a later security review without deciding now.
- [ ] **Remediate Now** — build the retention window described above; becomes its own plan or its own phase.

---

### Item 2: A resume now acts

**What changed, stated plainly:** Phase 2's security review accepted AR-2-01, an unbounded WAITING state that returns to ON by itself on reconnect, when nothing acted on it but a badge flip. From this phase, a WAITING-to-ON resume fires a real decision and a real command (D-02) — the same unbounded, un-timed resume now has a live consequence rather than a cosmetic one.

**Policy section 2, quoted verbatim** (`.specify/specs/safety-and-abuse-policy-v1.md`):

> ## 2. Supervise the AI
>
> - The AI plays under your supervision. You are expected to be reachable while it plays and to check on its behavior at reasonable intervals.
> - Do not leave the AI running unattended for extended periods. If you need to step away for long, disengage it.
> - If the AI is disconnected or kicked from a game while engaged, it will not reconnect on its own, and you must not re-engage it until you understand why it was disconnected.

**What was deliberately not built:** no bounded WAITING lifetime exists. `AutopilotRecord.WaitingSince` (`internal/session/autopilot.go`) is still stored on every WAITING transition, so a bounded waiting lifetime remains available without redesign should the owner want one.

**Cross-reference:** T-3-14 (Policy risk) — dispositioned "defer to security review"; Phase 2's own T-2-08, which that phase's agenda (Item 1) already raised once as AR-2-01 and asked to be re-raised here in concrete form now that an actual AI decision, not just a badge, fires on resume.

**Dispositions:**
- [ ] **Accept** — the auto-resume-now-acts behaviour stands as built, with no time bound.
- [ ] **Defer** — carry this item to a later security review without deciding now.
- [ ] **Remediate Now** — build the bounded WAITING lifetime described above; becomes its own plan or its own phase.

---

### Item 3: The log page's authorization scope

**What is enforced today:** every transcript and decision read resolves through `ProfileStore.GetProfileByConnection(userID, connectionID)`, and the transcript read also filters on the connection id, so a guessed session id returns nothing. This is the same IDOR mitigation pattern Phase 1 and Phase 2 already established (T-1-03, T-2-02).

**The open question the owner asked to have raised:** whether per-profile, signed-in scoping is the right boundary for stored game text at all, given the log page is a plain SPA route that anyone with the owner's own signed-in session can open — there is no separate re-authentication, share-link restriction, or additional confirmation step beyond the session cookie every other route already trusts.

**Cross-reference:** T-3-04 (Access Control / IDOR).

**Dispositions:**
- [ ] **Accept** — per-profile, signed-in scoping stands as the sole boundary for stored game text.
- [ ] **Defer** — carry this item to a later security review without deciding now.
- [ ] **Remediate Now** — build a stronger boundary (for example a separate share-link mechanism with its own expiry, or a re-confirmation step before the log page opens); becomes its own plan or its own phase.

---

### Item 4: The Gemini key on staging and the free-tier limits

**What is enforced today:** the key lives only in Railway staging environment variables, travels only in an `x-goog-api-key` header, is never logged, and never appears in the repository or in any evidence file.

**What is not known:** Google does not publish fixed free-tier numeric rate limits (RESEARCH Assumption A1) — any specific number would be third-hand and unverified. A rate-limit answer (HTTP 429) is treated as an ordinary D-13 failure with no retry (D-19). The per-session call cap, which would be the owner's own cost and rate protection at a higher level than a single 429, is Phase 4's mechanism, not this phase's.

**Cross-reference:** T-3-02 (Information Disclosure — the key), T-3-19 (cost and free-tier consumption).

**Dispositions:**
- [ ] **Accept** — key handling and the no-retry-on-429 behaviour stand as built, with no call cap until Phase 4.
- [ ] **Defer** — carry this item to a later security review without deciding now.
- [ ] **Remediate Now** — build a stopgap (for example an early, phase-3-level call cap) ahead of Phase 4; becomes its own plan or its own phase.

---

### Item 5: The driver sends without passing the human wheel-grab

**The mechanism, stated exactly:** the driver reaches the MUD through `SendCommandAs(..., "ai")` rather than the browser's websocket ingress, so the wheel-grab check that disengages autopilot on human input never sees the driver's own command — by construction, since the driver's send is not human input. ICM's dispatcher and safety checker do gate the command before it is sent, and the transcript marks the line `ai` so it is distinguishable after the fact.

**The residual, stated for the owner:** a server-side send path exists that no browser-side control mediates. The wheel-grab is a human-input interrupt; it was never designed to intercept the AI's own sends, and nothing in this phase changes that design.

**Cross-reference:** T-3-06 (Elevation of Privilege) and Phase 2's T-2-01, which Phase 2's own agenda (Closing Note) required this phase's review to re-confirm now that an engaged autopilot issues real commands rather than merely holding a badge.

**Dispositions:**
- [ ] **Accept** — the server-side send path with no browser-side mediation stands as built.
- [ ] **Defer** — carry this item to a later security review without deciding now.
- [ ] **Remediate Now** — build a browser-side control that can interrupt an in-flight driver send; becomes its own plan or its own phase.

---

### Item 6: Prompt injection through game text

**The mitigation that exists:** Go-side validation rejects a multi-line command or a command prefixed `#`, `@`, `$`, or `%`, applied before the command ever reaches the dispatcher, with a test row per attack shape (D-13's "malformed answer" / "a `#` directive or other non-game line" categories).

**The residual:** a hostile room description or another player's speech can still steer *which* legitimate, single-line game command the model chooses to issue — the mechanical backstop constrains command *shape*, not command *content* or the reasoning that produced it.

**Cross-reference:** T-3-01 (Tampering / Elevation of Privilege).

**Dispositions:**
- [ ] **Accept** — shape-only validation stands as the sole mitigation for prompt injection via game text.
- [ ] **Defer** — carry this item to a later security review without deciding now.
- [ ] **Remediate Now** — build a content-level mitigation (for example a second model pass reviewing the chosen command against the room text, or a denylist of high-risk verbs); becomes its own plan or its own phase.

---

## Part B — Items raised during today's staging walkthrough (2026-09-16)

The orchestrator ran tasks 03-13-02 and 03-13-03 against Railway staging today. The following twelve items surfaced live during that walkthrough and were not anticipated by the plans below. Each is raised here in the same spirit as Part A — described, not decided.

### Item 7: A server-only plan's acceptance criterion was actually a browser behaviour, and it slipped through

**What happened:** plan 03-06 promised the D-20 refusal sentence would appear in the terminal, but its own scope only changed the server; the browser silently dropped the outcome instead of rendering it, until this was caught and fixed mid-walkthrough today in commit `e57e91a` (`frontend/src/services/automation/evaluator.ts`).

**What this raises:** whether the plan-checker step that approves a plan as "server-only" should also check whether any of its acceptance criteria describe a UI-visible outcome, since a server-only plan whose criterion is a browser behaviour can pass its own tests while the criterion itself goes unmet until an end-to-end walkthrough happens to notice.

**Dispositions:**
- [ ] **Accept** — the fix landed today (commit `e57e91a`) is sufficient; no process change needed.
- [ ] **Defer** — carry this item to a later security review without deciding now.
- [ ] **Remediate Now** — add a plan-checker rule that flags a server-only plan with a UI-visible acceptance criterion; becomes its own plan or its own phase.

---

### Item 8: The autopilot badge briefly misrepresents the switch position after an AI-failure disengage

**What happened:** after an AI-failure disengage the server turns the switch off immediately, but the sidebar badge kept reading "Autopilot: On" for several seconds until the next status poll caught up — the `ai` websocket message carries the failure notice, but not the switch state itself. Screenshot `10-decision-failure-disengage.png` was taken after the poll caught up, so it shows the badge correctly reading Off, but the lag itself was observed during the walkthrough.

**What this raises:** D-13 says the badge reads Off on a first failure; for several seconds after the failure the badge visibly disagreed with the actual switch position.

**Dispositions:**
- [ ] **Accept** — the badge-lag window stands as built; the poll interval is close enough.
- [ ] **Defer** — carry this item to a later security review without deciding now.
- [ ] **Remediate Now** — have the `ai` websocket message that carries a failure notice also carry the new switch state, so the badge updates in the same message; becomes its own plan or its own phase.

---

### Item 9: A hard refresh's "survives a refresh" claim depends on reconnecting the same profile

**What happened:** a hard page refresh drops the live game session — a pre-existing behaviour noted in Phase 2 — and the AI Assist panel only mounts when a connected profile is present. Proving D-12's "survives a refresh" therefore required reconnecting the same profile after the refresh, after which the panel rebuilt the decision from the server. This matched the plan's own expectation, but the mechanism (reconnect, then rebuild) is a step the D-12 wording does not spell out.

**What this raises:** whether the gap between "the decision survives a refresh" (as read in D-12 and the ROADMAP criterion) and "the decision survives a refresh once the same profile is reconnected" is a distinction that should be made explicit in future phase wording, so a later reader does not assume the panel persists through the refresh with no further action.

**Dispositions:**
- [ ] **Accept** — the reconnect-then-rebuild behaviour stands as built and as the correct reading of D-12.
- [ ] **Defer** — carry this item to a later security review without deciding now.
- [ ] **Remediate Now** — make the requirement's wording (or the UI itself) close this gap explicitly; becomes its own plan or its own phase.

---

### Item 10: Vendor availability and latency versus the no-retry design

**What happened:** the initial default model `gemini-2.5-flash` returned HTTP 404 for the owner's new key; `gemini-3.8-flash` exceeded the client's original 30-second timeout and then separately returned HTTP 503 (overloaded) twice; `gemini-3.5-flash` returned 503 once and then succeeded. The client timeout was raised from 30s to 120s in commit `ff4a958` (`cmd/server/main.go`) so the successful call could complete. The driver itself made no retries at any point, by design (D-03: one decision per engage).

**What this raises:** whether the no-retry design (correct for "the AI stays idle after one decision," D-03) is also the right answer for "the vendor returned a transient 503," which is a different failure class from "the model produced a bad answer" — today both land identically as a first D-13 failure that disengages autopilot. Also raised: whether the default model name should be an environment variable set per-deploy rather than a value chosen once and left, given three different model names were tried live today before one answered reliably.

**Dispositions:**
- [ ] **Accept** — the no-retry-on-503 behaviour and the raised 120s timeout stand as built.
- [ ] **Defer** — carry this item to a later security review without deciding now.
- [ ] **Remediate Now** — add a single automatic retry on a 503 specifically (distinct from D-13's other failure kinds), or formalize the default model name as an owner-set environment variable reviewed per deploy; becomes its own plan or its own phase.

---

### Item 11: The new model-error diagnostic log line's content

**What happened:** a new diagnostic log line was added today in commit `0e24dad` (`internal/driver/driver.go`), of the shape `[AI-PLAYER] model-error ... kind=<transport|auth|bad_request|rate_limited|malformed> http=<status>`. It carries a classification and an HTTP status but never the vendor's own error message text.

**What this raises:** confirmation that this line stays within T-3-07's "no content" rule (no game text, no reasoning, no key, no endpoint) — the line's fields today are limited to a fixed enum and a numeric status, so no vendor message text reaches the log, but this is worth the review confirming explicitly given it is new code added during today's walkthrough, outside any plan's original file list.

**Dispositions:**
- [ ] **Accept** — the `model-error` line's current field set (kind, http status only) stands as built and satisfies T-3-07.
- [ ] **Defer** — carry this item to a later security review without deciding now.
- [ ] **Remediate Now** — restrict or further redact the line's fields; becomes its own plan or its own phase.

---

### Item 12: A production environment variable was set and triggered an unplanned production redeploy

**What happened:** the owner set the Gemini key (`GOOGLE_API_KEY`) on the **production** environment of service `MudPuppy` first, before staging. Railway's own behaviour on an environment-variable change is to redeploy that environment from its last-deployed source, which triggered a production redeploy from GitHub at 06:23 PDT (deployment `11762ced`). The redeployed code is the old, pre-Phase-3 production build, which does not read `GOOGLE_API_KEY` at all, so the variable is set but has no effect on production's behavior.

**What this raises:** production was supposed to remain untouched until final acceptance (CLAUDE.md, this plan's own context section); an environment variable was set there and a redeploy occurred, even though the running code is unaffected. The variable itself (`GOOGLE_API_KEY`) is also not one of the five variables named in `03-02-SUMMARY.md` for this project's own model registry (`AI_MODEL_DEFAULT`, `AI_MODEL_GEMINI_NAME`, `AI_MODEL_GEMINI_ENDPOINT`, `AI_MODEL_GEMINI_KEY`, `AI_MODEL_GEMINI_PROVIDER`), so it is inert dead configuration on production even if a future production deploy were to include Gemini-reading code that happened to expect that exact name.

**Dispositions:**
- [ ] **Accept** — the stray `GOOGLE_API_KEY` variable and the unplanned redeploy stand as-is on production.
- [ ] **Defer** — carry this item to a later security review without deciding now.
- [ ] **Remediate Now** — remove `GOOGLE_API_KEY` from the production environment and document the incident in the deploy runbook; becomes its own plan or its own phase.

---

### Item 13: Railway dashboard deploys diverge from the branch actually running on staging

**What happened:** the Railway dashboard's Deploy button rebuilds a service from its connected GitHub repository. Branch `ai-player` has never been pushed to GitHub — `origin/ai-player` sits at the March commit `9066456`. A dashboard-triggered rebuild against that stale branch crashed on startup with "no migration found for version 11," because the stale branch's migration set does not include this phase's migration. Every working staging deploy to date has instead been a `railway up` upload of the local checkout, bypassing GitHub entirely.

**What this raises:** a process risk — anyone (the owner or a future collaborator) who clicks Deploy in the Railway dashboard, not knowing this, will crash staging. This is not a code defect; it is an undocumented deploy-path divergence.

**Dispositions:**
- [ ] **Accept** — continue deploying only via `railway up` from the local checkout, with no other change.
- [ ] **Defer** — carry this item to a later security review without deciding now.
- [ ] **Remediate Now** — either push branch `ai-player` to GitHub so dashboard deploys are safe, or document in the deploy runbook that dashboard deploys must not be used for this branch; becomes its own plan or its own phase.

---

### Item 14: Staging session tokens were pasted into the chat transcript

**What happened:** the owner pasted two staging `session_token` values into the chat during this walkthrough. The first has since expired along with its Redis session; the second is the current session's token.

**What this raises:** session tokens now exist in a chat transcript, which is a persistence location outside the application's own session store and outside this project's own log-content discipline (T-3-02/T-3-07's "never in a log or transcript" rule for secrets, applied here to a different class of secret than the ones those threats name). No credential was reused maliciously here, but the exposure pattern is the same class of risk DR-2-01 already treats seriously for the OTP code.

**Dispositions:**
- [ ] **Accept** — the exposure stands as a one-time incident with no further action.
- [ ] **Defer** — carry this item to a later security review without deciding now.
- [ ] **Remediate Now** — the owner rotates/signs out of the current staging session at their own convenience (never done by the executor — signing the owner out is the owner's own step), and future harness runs are performed either by the owner directly or via a short-lived token instead of a long-lived session cookie; becomes its own plan or its own phase.

---

### Item 15: A harness bug was found and fixed live, exposing a gap in the self-test's own coverage

**What happened:** during today's live run, `scripts/verify-phase3.sh` had two bugs, fixed in commit `987c22e`: step 1 rejected the legitimate `refused-not-configured` outcome and passed a string where an exit code was expected; step 4 read `decisions[0]` even though the decisions endpoint returns oldest-first and a failed decision carries no reasoning field, which the step assumed was always present.

**What this raises:** the harness's own self-test (`--self-test` / `--self-test-negative`, proven clean in `evidence/02-harness-selftest.txt` before either live run) exercises the harness against fixture data shaped by the harness author's own assumptions — it did not, and structurally cannot, catch a live-shape mismatch against the real deployed endpoints, because the fixture and the bug shared the same blind spot.

**Dispositions:**
- [ ] **Accept** — the harness stands as fixed today (commit `987c22e`), with no change to how future harnesses are validated.
- [ ] **Defer** — carry this item to a later security review without deciding now.
- [ ] **Remediate Now** — add a recorded-live fixture (a captured real response, replayed) to the self-test suite so future harness self-tests catch live-shape drift; becomes its own plan or its own phase.

---

### Item 16: DEV MODE OTP log lines still print the sign-in code when SMTP is unconfigured

**What happened:** plan 03-07 closed DR-2-01 for the two `"STAGING: OTP sent to user, code: %s"` call sites, gating them behind `AUTH_LOG_OTP` (off by default). Two other, pre-existing lines — `"DEV MODE - OTP for %s: %s"` in `internal/auth/handler.go`, in the `else` branch of `if h.emailSender != nil && h.emailSender.IsConfigured()` — still print the code directly and were out of that plan's named scope (documented in `03-07-SUMMARY.md` and `deferred-items.md`). This branch executes only when SMTP is entirely unconfigured, which is never true on staging or production today, both of which require SMTP for email delivery to function at all.

**What this raises:** this is not a live-staging exposure today, but it is a residual place the code can reach a log if SMTP were ever left unconfigured in a future environment (for example a demo environment). Raised here for DR-2-01 closure wording, so the closure records what was fixed and what was deliberately left as a local-developer convenience rather than silently treating DR-2-01 as fully closed everywhere.

**Dispositions:**
- [ ] **Accept** — the DEV MODE lines stand as-is; they cannot execute on staging or production as configured today.
- [ ] **Defer** — carry this item to a later security review without deciding now.
- [ ] **Remediate Now** — route the two DEV MODE lines through `logOTPIssued` or an equivalent gated helper; becomes its own plan or its own phase.

---

### Item 17: `-race` requires a user-scope MinGW GCC not present on the dev machine by default

**What happened:** every `-race` invocation named in the Phase 3 plans requires `CGO_ENABLED=1` and a C toolchain; this dev machine has neither by default. Phase 2's security review already accepted this as AR-2-11 (an executor installed MinGW-w64 GCC, user scope, without asking, so `-race` could run; no project dependency changed). `evidence/01-test-report.txt` for this phase was captured with that toolchain present, so its `-race` sections are real, but this remains a per-machine dependency, not a project one.

**What this raises:** whether CI (a shared, reproducible environment, unlike a developer's own machine) should run `-race` as a standing step, so the race detector's coverage does not depend on any one contributor's local machine having a C toolchain installed.

**Dispositions:**
- [ ] **Accept** — per-machine `-race` availability, as it stands today (present on this machine, not guaranteed elsewhere), is sufficient.
- [ ] **Defer** — carry this item to a later security review without deciding now.
- [ ] **Remediate Now** — add `-race` as a standing CI step with its own C-toolchain provisioning; becomes its own plan or its own phase.

---

## Part C — Closures (for the owner to confirm, not decide)

### DR-2-01 — closed by plan 03-07

**Owner's original wording, reproduced verbatim** (`.planning/RISK-REGISTER.md`, `.planning/todos/pending/2026-09-15-phase3-security-carry-forward.md`):

> Staging prints the one-time sign-in code in the deploy log; anyone with staging log access can sign in as any staging user. Pre-existing, staging only. Deferred at Phase 1 and again at Phase 2.
>
> Owner note: "Mark this as MUST FIX in next Phase."

**Proposed remediation, reproduced verbatim:**

> Gate the code-logging line behind an env flag that is off by default, or log a hash; confirm production has no equivalent; review Railway log access on the staging project.

**What was built:** `internal/config.Config` gained `AuthLogOTP bool` (env `AUTH_LOG_OTP`, off by default, warn-and-default parsing). `internal/auth.Handler` gained one gated helper, `logOTPIssued`, and both former `"STAGING: OTP sent to user, code: %s"` call sites now route through it. With the flag off — staging's actual configuration — the log line reads `[AUTH] one-time sign-in code issued (code suppressed; set AUTH_LOG_OTP=1 to print)`.

**Cited evidence:** `evidence/01-test-report.txt`, `TestOTPNotLogged` PASS lines (three subtests: `logs_issuance_without_the_code`, `logs_the_code_only_when_flag_is_on`, `suppressed_line_carries_no_identifiers`); `evidence/04-staging-ai-player.log`, no `code:` line present for the fresh sign-in captured during today's walkthrough.

**Carried forward, not closed by code:** the non-code part of the original remediation — "review Railway log access on the staging project" — remains open. It has its own three dispositions here, separate from the closure above:

- [ ] **Accept** — Railway staging log access, as currently granted, stands as-is.
- [ ] **Defer** — carry this item to a later security review without deciding now.
- [ ] **Remediate Now** — review and, if needed, restrict who has `railway logs --environment staging` access on project `mudpuppy`; becomes its own plan or its own phase.

Also see Part B, Item 16 (the DEV MODE OTP lines, out of plan 03-07's named scope) and Item 14 (session tokens pasted into chat during this walkthrough) — both bear on how completely "the sign-in code / session credentials never reach an unintended reader" holds today.

---

### DR-2-02 — closed by plan 03-13

**Owner's original wording, reproduced verbatim** (`.planning/RISK-REGISTER.md`, `.planning/todos/pending/2026-09-15-phase3-security-carry-forward.md`):

> Six throwaway connection profiles remain on the owner's staging account (Phase 2 Harness, Harness B, Walkthrough, Refusal, Hand Play; Phase 1 Evidence Walkthrough).
>
> Owner note: "Clean this up in Phase 3."

**Proposed remediation, reproduced verbatim:**

> Delete them from the Connections list, or keep as Phase 3 fixtures and delete after.

**What was done:** the six profiles were used as Phase 3 fixtures during today's walkthrough and then deleted through the app, using the owner's own signed-in session — the owner was never signed out.

**Cited evidence:** `evidence/14-connections-cleaned.png`, showing the Connections list with none of the six present.

This is a closure to confirm, not a decision — DR-2-02 requires no disposition.

---

## Carry-Forward Table: Threats Accepted at Plan Level (confirmation only, not re-litigation)

These Phase 3 threats were dispositioned `accept` inside their originating plans rather than mitigated or deferred. They are listed here so the review sees them in one place. The review's job for this table is to confirm the acceptance still holds, not to re-decide it from scratch.

| Threat ID | Component | Accepted at | What to confirm |
|-----------|-----------|-------------|------------------|
| T-3-SC | `go.mod`, `frontend/package.json` (supply chain) | Plan 03-01, re-verified every subsequent plan | No package was installed anywhere in Phase 3 — confirm `git diff --stat <phase-base-commit> -- go.mod frontend/package.json` is empty. `evidence/01-test-report.txt`'s `### DEPENDENCY DRIFT` section (empty) is the citable proof. |
| T-3-19 | Cost and free-tier consumption of the Gemini API | Plans 03-02, 03-08 | The owner's key is on the Gemini free tier for this test; the per-session call cap that would bound cost is Phase 4's mechanism, not built here. Confirm this remains an acceptable posture until Phase 4 ships it. See also Part A, Item 4. |
| T-3-25 | The retired `#AUTO STATUS` diagnostic line | Plan 03-04 | `#AUTO STATUS` no longer prints a status line; the badge and the panel are the only status surfaces now. Confirm no other diagnostic path silently depended on the retired line's output. |

---

## Forward Notes for Phase 4

1. **ICM's `RecordExecution` gap.** `Dispatcher.Dispatch`'s own circuit-breaker and rate-limit counters do not accumulate for the AI's pass-through commands, because `RecordExecution` only fires after a handler successfully executes and no handler is registered for a plain game command (RESEARCH Open Question 1). Harmless at one command per engage (this phase); not harmless once Phase 4 issues commands in a loop. Phase 4 must decide whether the driver calls `RecordExecution` explicitly after a successful send, or relies entirely on its own separate call-cap mechanism and treats ICM's counters as decorative for this command class.
2. **The seventh failure notice.** A failure notice exists for an ICM refusal — "the command was refused by the command safety limits" — that the owner has not yet seen in a screenshot or a live walkthrough, since no walkthrough step in this phase tripped ICM's safety checker. Its exact wording should be reviewed with the owner before Phase 4 makes ICM refusals a more frequent occurrence.

---

**The project cannot close while any item on this agenda remains marked Defer.**
