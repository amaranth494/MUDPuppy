---
phase: 05
slug: coaching-channel
status: verified
threats_open: 0
asvs_level: 1
created: 2026-09-18
register_authored_at_plan_time: true
accepted: 6
deferred: 4
remediate_now: 0
---

# Phase 05 — Coaching Channel — Security

> Per-phase security contract: threat register, accepted risks, deferred risks, and audit trail.
> Register authored at plan time across `05-01-PLAN.md` through `05-11-PLAN.md` (58 distinct threat ids, `T-5-01` to `T-5-57` plus `T-5-SC`). Verified against implemented code, not documentation, on 2026-09-18 by gsd-security-auditor at HEAD `0e47f98` (`05-SECURITY-AUDIT.md`: 52 verified in code, 3 accepted at plan level, 1 deferred to this review, **2 OPEN**; plus 11 unregistered items found during execution, `T-5-NEW-01` to `T-5-NEW-11`: 2 verified, 9 open or partial). The code review's two critical findings and fifteen warnings (`05-REVIEW.md`) were fixed before the audit (`05-REVIEW-FIX.md`, 23 commits) and the audit re-read every one of those fixes in the code rather than trusting the summaries, which predate them.
>
> **The owner's rule at this review:** only active risks that carry a concrete proposed remediation appear below as Accepted or Deferred. Items the owner judged already fixed, or fixed and confirmed in code, are recorded as Closures, not risks. Items the owner judged not to be security risks at all are recorded in their own section below, not in the Accepted or Deferred tables, and are carried forward in a separate, non-security report. Each risk is written as the problem, then the fix, in everyday words.
>
> The owner's decisions on every register item were made on the Phase 5 Risk Register (https://claude.ai/artifact/Ep5B1NwSKXUJ3eAYsUJf8n) on 2026-09-18 and read back from it by the orchestrator after the owner decided all 18 items: 13 accepted, 5 deferred, 0 remediate now. The register carried eighteen active risk items under this rule (R-01 through R-18). No item was marked Remediate Now, so Phase 5 does not reopen.
>
> **What `threats_open: 0` means in this record, stated exactly.** The audit marked two plan-time threat rows OPEN (both rated low) and one DEFERRED (rated high), because a declared mitigation is absent, partial, or explicitly left for this review. None of the three was mitigated by the audit's own read of the code at the time it ran. `threats_open: 0` is correct ONLY because each of those three rows now carries an owner disposition:
>
> - **T-5-46 → ACCEPTED** (register R-01, recorded as AR-5-01). Owner's note: "Push as is."
> - **T-5-41 → FIXED, same day** (register R-13). The help article's missing read-it-first sentence was added in commit `6fb1c9b`, the same day the audit found it missing. Owner's note: "This is not a security risk.  Update the guide as needed." This row is a Closure, not an accepted or deferred risk.
> - **T-5-26 → DEFERRED** (register R-08, recorded as DR-5-02). Owner's note: "We are going to address this in advanced logic for the AI-Player later."
>
> Nothing is hidden by the zero. One row is an accepted risk, one row is fixed and closed, and one row is an active deferral that is raised again at the Phase 6 security review and blocks project close until it is resolved.

---

## Trust Boundaries (consolidated across all eleven plans)

| Boundary | Plans | Description |
|----------|-------|--------------|
| Owner's Pause/Resume click, a reconnect → the autopilot switch | 05-01 | Two independent reasons to wait; clearing one must never clear the other or start the AI unasked |
| Reviewer's answer → the send path | 05-02 | A verdict with no yes-or-no must stop the command, not pass it |
| Environment → the credential vault | 05-03, 05-11 | Whether the vault key is real or invented at startup decides whether saved MUD passwords survive a restart |
| Signed-out browser → the log page | 05-03, 05-09 | A pasted log link must land on sign-in, never on a transcript |
| The driver → the MUD | 05-04 | Every AI command must be counted by a limiter of its own |
| Owner's current chat message → AI-chatter | 05-05 | The one live instruction in AI-chatter's prompt |
| Game text, memory, AI-player's reasoning, the conversation tail → AI-chatter's prompt | 05-05 | All attacker-influenceable; all delimited as data |
| Chat websocket message → the game and the switch | 05-05, 05-07 | A chat message must never become a command or a wheel-grab |
| AI-chatter's answer → the coaching store → AI-player's and the reviewer's prompts | 05-06 | The one place a model's output becomes guidance another model reads; the laundering path this phase creates |
| Browser → the coaching and conversation reads | 05-06, 05-07, 05-09 | Three new GET sub-resources exposing the owner's own text |
| Play screen → pop-out windows | 05-08 | A second document painted by the same page; must not become a second authenticated surface or outlive the page |
| Help article → the owner's own settings | 05-09 | The owner is taught to paste model-written text into trusted profile text |
| The harness → staging; the report and fixtures → readers and the verdict | 05-10, 05-11 | A live run authenticates as the owner and rewrites a setting; whatever the report prints is committed |
| Repository → Railway staging; the corpus rerun → the owner's quota | 05-11 | The deployed binary and model may diverge from what was measured |

---

## Threat Verification Register

Legend: **M**=mitigate, **A**=accept, **D**=defer to security review. File:line references are condensed here; `05-SECURITY-AUDIT.md` holds the full evidence for every row, including each named residual. Status CLOSED/VERIFIED means the audit confirmed the declared mitigation in code and re-ran its tests. Rows updated at this review to carry the owner's decision are marked in bold in their Status cell.

| Threat ID | Category | Plan(s) | Disp. | Evidence (condensed) | Status |
|-----------|----------|---------|-------|------------------------|--------|
| T-5-01 | Tampering — a reconnect cancelling an owner pause (High) | 05-01 | M | `resumeAutopilotLocked` clears only `ConnectionLost` while `PausedByOwner` stands; Off clears both. `TestManager_ResumeRequiresBothReasonsClear` PASS. Residual: no reconnect happened while paused on staging, so this rule is proven by the Go test only | VERIFIED |
| T-5-02 | EoP — resume bypassing the policy gate (Medium) | 05-01 | M | The resume arm now asks the engage gate about the connection the switch is parked on, fails closed on a missing gate. `TestAutopilotHandler_ResumeConsultsTheEngageGate` PASS. Residual: the reconnect-resume path still consults no gate, as in Phase 4; tests only | VERIFIED |
| T-5-03 | DoS — a switch that can never be resumed (Medium) | 05-01 | M | Both reasons on every handler arm and every push; a hand-typed command while paused lands Off. Screenshot `18-paused-waiting-reason.png` | VERIFIED |
| T-5-04 | Repudiation — a pause or resume leaving no trace (Low) | 05-01 | M | `logAutopilotTransition`, transcript markers, stream notices. Live log `:79-80` | VERIFIED |
| T-5-54 | Spoofing — a notice for a transition that did not happen (Medium) | 05-01 | M | Both notices sit on the changed branch only. `TestAutopilotHandler_EmitsPausedAndResumedNotices` PASS | VERIFIED |
| T-5-05 | Tampering — a missing verdict read as "not blocked" (High) | 05-02 | M | `ReviewAnswer.Blocked` is `*bool`; a nil verdict is `KindMalformed`. `TestReviewCommand_MissingBlockedField` PASS. Closes Phase 4's DR-4-02 (first half) | VERIFIED |
| T-5-06 | DoS — the checker blocking ordinary fights (High) | 05-02 | M | Fighting-is-ordinary-play sentence written into the reviewer prompt. Measured not blocked in 9 of 9 measured samples (3 unmeasured, transport failures). Residual: every sample was on `gemini-3.5-flash-lite`; staging now runs `gemini-3.1-flash-lite` — see T-5-NEW-08 / AR-5-04 | VERIFIED |
| T-5-07 | Spoofing — the reviewer told it wrote the memory (Medium) | 05-02 | M | Per-reader authorship sentence. `TestBuildReviewSystemInstruction_MemoryAuthorship` PASS | VERIFIED |
| T-5-08 | Repudiation — one sample hiding an unreliable verdict (Medium) | 05-02 | M | `Repeats: 3` on both fight items; an unmeasured sample never counted as "not blocked". Residual: only the two fight items are repeat-sampled | VERIFIED |
| T-5-09 | DoS (cost) — the corpus spending the day's quota (Medium) | 05-02 (M), 05-11 (A) | A | Both live tests SKIP without the flag, refuse under `RAILWAY_ENVIRONMENT`. **Correction to the agenda: the rerun was on a fresh-quota day, but 10 of 40 items in the kept run came back `failed-model` because three live tests shared the key within five minutes — the agenda's framing omits this rate** | ACCEPTED |
| T-5-10 | Tampering — a random vault key off Railway (Medium) | 05-03 | M | `config.Load` fails everywhere unless `MUDPUPPY_LOCAL_DEV`; the opt-out is announced. Residual: T-5-NEW-07, see DR-5-01 | VERIFIED |
| T-5-11 | Info Disclosure — a signed-out visitor reading a transcript (High) | 05-03 | M | `/logs/:connectionId` inside `AuthGuard` with a comment naming AR-4-08. Screenshot `13-logs-signed-out.png` | VERIFIED |
| T-5-12 | Info Disclosure — a key value in a log line, fixture or report (High) | 05-03 | M | Both error strings name the variable only. Evidence grep: 0 hits | VERIFIED |
| T-5-13 | DoS — the stricter gate stopping staging (Low) | 05-03 | A | Staging started cleanly on three filed deployments | ACCEPTED |
| T-5-14 | DoS — AI commands never counted by a limiter (High) | 05-04 | M | `allowAISend` checked and consumed immediately before the one `Dispatch`. Closes Phase 4's DR-4-03. Residual: IN-04, unfixed | VERIFIED |
| T-5-15 | DoS — reusing the dispatcher's never-reset counter (Medium) | 05-04 | M | `ratelimit.go` shares no state with `internal/icm`; diff re-run empty | VERIFIED |
| T-5-16 | Tampering — a blank limit read as unlimited (Medium) | 05-04 | M | Blank resolves to the server default. Two tests PASS | VERIFIED |
| T-5-17 | Tampering — an absurd limit (Low) | 05-04 | M | `validateAISettings` rejects anything outside 1 to 20. Staging confirmed | VERIFIED |
| T-5-18 | DoS — one refused send disengaging autopilot (Low) | 05-04 | M | A rate-limited refusal is a transient failure kind with its own notice | VERIFIED |
| T-5-19 | EoP — hostile game text treated as the owner's words to AI-chatter (High) | 05-05 | M | Every untrusted block wrapped; a per-reader security core; the owner's current message is the only trusted instruction | VERIFIED |
| T-5-20 | EoP — AI-chatter acquiring power beyond talking (High) | 05-05 | M | Grep of `chat.go`: no Dispatch, recordFailure, DisengageAutopilot or SendAICommand | VERIFIED |
| T-5-21 | DoS (cost) — unlimited chat messages each spending a model call (High) | 05-05 | M | Reservation before the call; cap branch makes no call; chat has its own count, reset at login and each stint. Cost consequence: T-5-NEW-04, see AR-5-03 | VERIFIED |
| T-5-22 | Tampering — a chat message reaching the MUD or acting as a wheel-grab (High) | 05-05 | M | The `MsgTypeChat` case has no dispatch and no wheel-grab path | VERIFIED |
| T-5-23 | Info Disclosure — chat text or the reply in a log line or report (Medium) | 05-05 | M | Every `[AI-CHATTER]` line carries ids, stages, counts and lengths only. See T-5-46 and T-5-NEW-06 for the artifacts | VERIFIED |
| T-5-24 | DoS — an enormous message or a burst (Medium) | 05-05 | M | 1000-character cap, 64 KB websocket cap, one message in flight, HTTP timeout | VERIFIED |
| T-5-25 | Tampering — migration 015 failing on the live schema (Medium) | 05-05, 05-11 | M | `IF NOT EXISTS`, constant default, cascade, reversing down file. Live: `version=15, dirty=false`. **Correction to the agenda: the filed log excerpt carries a startup line for only 3 of the 6 staging deployments; the walkthrough deployment's own header has nothing under it, and the final deployment is not in the file at all** | VERIFIED |
| T-5-55 | EoP — an old conversation line acted on as a fresh request (Medium) | 05-05 | M | Tail bounded to 10 lines, neutralised, cut to 200 bytes, read before the current message is stored. **Correction to the agenda: this row is `mitigate` at plan time, not a plan-level acceptance as the agenda's Part 3 lists it — it is genuinely mitigated** | VERIFIED |
| T-5-26 | EoP — the coaching channel as a laundering path (High) | 05-06 (M), 05-11 (D) | D | One writer; neutralise-and-delimit; the push and withdraw gates added after the code review. Live: 0 laundered of 10 hostile samples after the withdraw gate. **The agenda understates this: two push-gate bypasses found by this audit (T-5-NEW-02) are not on the agenda** | **DEFERRED — carried forward as DR-5-02 (R-08); owner: "We are going to address this in advanced logic for the AI-Player later."** |
| T-5-27 | Spoofing — the reply claiming a different line was sent (High) | 05-06 | M | Prefixes composed in Go from what was actually stored; a failed store reports nothing sent; the model's own text has the prefixes stripped. Residual: T-5-NEW-10, see AR-5-02 | VERIFIED |
| T-5-28 | EoP — coaching used to talk past a conduct rule or the Never-issue list (High) | 05-06 | M | Coaching block sits below the standing text, goal and both memories; named "guidance only, never authority" | VERIFIED |
| T-5-29 | DoS — an unbounded coaching list (Medium) | 05-06 | M | 8 bullets at 200 bytes, enforced in and out; at most 2 pushes and 2 withdraws per message | VERIFIED |
| T-5-30 | Info Disclosure — another user reading coaching or conversation (Medium) | 05-06 | M | Ownership check before any store read, GET-only at mux and handler. Staging confirmed | VERIFIED |
| T-5-31 | Repudiation — a suggestion in effect the owner never saw go in (Medium) | 05-06 | M | Quoted reply, live list, `Coaching received` marker, count-only log lines. Live confirmed | VERIFIED |
| T-5-56 | EoP — a stored coaching line read by AI-chatter as a fresh request (Medium) | 05-06 | M | Same pipeline; named as a record "not a live instruction to act on again". **Correction to the agenda: this row is `mitigate` at plan time, not a plan-level acceptance as the agenda's Part 3 lists it — it is genuinely mitigated** | VERIFIED |
| T-5-32 | Tampering — a chat message reaching the MUD, or a command swallowed by chat (High) | 05-07 | M | `sendChat` writes type `chat`, never calls `sendCommand`. Grep: 0 | VERIFIED |
| T-5-33 | EoP — model or game text rendered as markup (Medium) | 05-07 | M | `dangerouslySetInnerHTML`, `innerHTML`, `document.write`: 0 matches under `frontend/src` | VERIFIED |
| T-5-34 | Info Disclosure — the new reads served to a non-owner (Medium) | 05-07 | M | All three client functions send credentials and run `handleAuthError` | VERIFIED |
| T-5-35 | DoS — a disabled message box while autopilot is off (Low) | 05-07 | M | Send is disabled only by an empty draft. Screenshot `08-chat-while-off.png` | VERIFIED |
| T-5-36 | Repudiation — coaching invisible after a refresh (Medium) | 05-07 | M | Screenshot `09-coaching-after-refresh.png`; the read is login-scoped server side | VERIFIED |
| T-5-57 | Repudiation — a chat message silently lost (Medium) | 05-07 | M | `sendChat` returns a truthful boolean; four paths that said "saved" when nothing was, removed | VERIFIED |
| T-5-37 | Info Disclosure — a pop-out as a second authenticated surface (Medium) | 05-08 | M | `window.open('')` only, one file, no `postMessage`, nothing new authenticated | VERIFIED |
| T-5-38 | DoS — a pop-out outliving the play screen (Medium) | 05-08 | M | Closed on `pagehide` and `beforeunload`; a reused window is emptied before use. No automated test (no frontend runner); screenshots 10, 11 | VERIFIED |
| T-5-39 | Tampering — a blocked pop-up leaving a half-moved view (Low) | 05-08 | M | `openPopout` returns `null` on a block | VERIFIED |
| T-5-40 | Spoofing — an unstyled or look-alike window (Low) | 05-08 | M | Stylesheets cloned into the child head; the title names the view | VERIFIED |
| T-5-41 | EoP — the article teaching the owner to paste model-written text into Conduct rules (Medium) | 05-09 | M | Article names the two settings; the copy is manual. **OPEN at audit — the read-it-first sentence was missing, and the agenda's Part 3 wrongly listed this row as a plan-level acceptance; the plan's disposition is `mitigate`.** Fixed the same day, commit `6fb1c9b` (the read-it-first sentence added, stray backticks removed) | **CLOSED (mitigated). Owner (R-13): "This is not a security risk.  Update the guide as needed."** |
| T-5-42 | Info Disclosure — a conversation served to a non-owner via the Logs page (Medium) | 05-09 | M | See T-5-NEW-01; the page renders inside `AuthGuard` | VERIFIED |
| T-5-43 | EoP — chat text rendered as markup on the Logs page (Medium) | 05-09 | M | Lines are React text children, `white-space: pre-wrap` | VERIFIED |
| T-5-44 | Repudiation — a design document still promising promotion (Medium) | 05-09 | M | `PROJECT.md`, `REQUIREMENTS.md`, `ROADMAP.md` all state plain-text chat with promotion withdrawn | VERIFIED |
| T-5-45 | Repudiation — a canned report that always passes (High) | 05-10, 05-11 | M | Re-run: `--self-test` exit 0 (42 checks), `--self-test-negative` exit 1 with one `FAIL C3`. **Note: C1 (pause and resume) is SKIP in both staging runs because the switch was off each time — the harness never exercised pause or resume on staging; that criterion rests on the walkthrough log and screenshot 18 instead** | VERIFIED |
| T-5-46 | Info Disclosure — a key, cookie, chat text, reply, game text, goal or coaching line in a committed report, log excerpt or screenshot (High) | 05-10, 05-11 | M | All nine evidence text files grepped clean; the harness redacts coaching, conversation and settings bodies. **One facet defeated: commit `1d2e0c2` still holds the owner's conduct rules and Never-issue list seven times, in unpushed history** | **OPEN at audit (git-history facet only; every current file verified). Not mitigated. ACCEPTED by the owner — AR-5-01 (R-01); owner: "Push as is." `ai-player` is pushed after this record is committed, as-is.** |
| T-5-47 | Tampering — the harness engaging autopilot unattended (High) | 05-10 | M | The script sends only `status`, `pause` and `resume`; skips C1 with a stated reason when the switch is off | VERIFIED |
| T-5-48 | Tampering — the rate-limit setting left changed (Medium) | 05-10 | M | Raw settings body recorded before the first write, restored verbatim, compared byte for byte; EXIT/INT/TERM/HUP traps | VERIFIED |
| T-5-49 | DoS (cost) — a harness step spending a model call (Low) | 05-10 | M | No step sends a chat message or the `on` action | VERIFIED |
| T-5-50 | Tampering — a deploy, variable change or harness run aimed at the wrong target (High) | 05-11 | M | Both RUN headers record the staging base URL and a git SHA; production untouched. Deviation (model name changed on the owner's instruction) recorded | VERIFIED |
| T-5-51 | DoS — signing the owner out (High) | 05-11 | M | The owner was never signed out; screenshot 13 came from a private window; no sign-in-code pattern in any evidence file | VERIFIED |
| T-5-52 | Tampering — `MUDPUPPY_LOCAL_DEV` set on staging (High) | 05-11 | M | The plan forbids it; the SUMMARY states it was never set. Nothing in code refuses it on a hosted platform: see T-5-NEW-07, DR-5-01 | VERIFIED |
| T-5-53 | Repudiation — four deferred risks quietly lapsing (High) | 05-11 | M | `05-SECURITY-AGENDA.md` Part 1 re-presents DR-4-01 to DR-4-04 and AR-4-08; 36 checkboxes, 0 marked | VERIFIED |
| T-5-SC | Tampering (supply chain) — `go.mod`, `frontend/package.json` (Low) | all eleven plans | A | Dependency-drift diff re-run: empty | ACCEPTED |

**Unregistered attack surface found during verification (no `T-5-NN` mapping):** eleven items surfaced from the code review, the walkthrough or the audit itself, ids `T-5-NEW-01` onward. Each is now governed by the register item and decision that follow it.

| Id | What it is | Found by | Governing decision | Status |
|----|-----------|----------|---------------------|--------|
| T-5-NEW-01 | A new HTTP read, `GET .../sessions/{session_id}/conversation`, added by plan 05-09 outside its file list | walkthrough notes | GET-only, ownership resolved before any read, session-and-connection-filtered. No register item; sound as built, no owner decision needed | VERIFIED |
| T-5-NEW-02 | The push gate: what it stops and what still gets through | this audit | Stops a line sharing no words with the owner's message; bypassed by short words (under four letters, free) and by the gate reading the whole line while only 200 bytes are stored. Governed by R-08 → **DR-5-02**; owner: "We are going to address this in advanced logic for the AI-Player later." | PARTIAL |
| T-5-NEW-03 | The withdraw gate: what it stops and what still gets through | code review fix report | Stops a hostile withdraw during an unrelated question (0 of 10 after the gate); bypassed by a question merely mentioning the subject, a take-back phrase inside an ordinary question, and a four-letter stem over-match. Governed by R-09 → **AR-5-02** | PARTIAL |
| T-5-NEW-04 | AI-chatter's own call count as a cost control (WR-08) | this audit | Chat's own count is no longer bounded in total by the owner's Call Cap and restarts at every stint or Resume. Governed by R-12 → **AR-5-03**; owner (design decision): "AI-CHAT and AI-PLAYER are different in nature and the Call Cap should only affect the AI-PLAYER.  Keep as is." | PARTIAL |
| T-5-NEW-05 | The pop-out windows | code review | Same-origin `about:blank`, no URL, no `postMessage`, nothing new authenticated. No register item; sound as built | VERIFIED |
| T-5-NEW-06 | Owner text and the character's name in committed planning documents | this audit | Walkthrough notes, agenda and summary quote coaching lines, AI reasoning and the character's name. Governed by R-17 → **AR-5-06** | PARTIAL |
| T-5-NEW-07 | `MUDPUPPY_LOCAL_DEV` honoured on a hosted platform (IN-06) | code review | The opt-out is honoured everywhere, Railway included; only a written instruction prevents it being set there. Governed by R-06 → **DR-5-01**; owner: "fix this in next phase per the recommended approach" | OPEN |
| T-5-NEW-08 | Staging runs a model the safety checker has never been measured on | walkthrough notes | The 40-item hostile-text corpus ran on `gemini-3.5-flash-lite` only; staging runs `gemini-3.1-flash-lite`. Governed by R-15 → **AR-5-04** | PARTIAL |
| T-5-NEW-09 | The conversation is kept forever, and "Delete Captured Text Now" does not remove it | this audit | The retention job names only `window_text` and `game_session_lines`; nothing prunes `conversation_lines`. Governed by R-11 → **DR-5-04**; owner: "Fix this in the next Phase per the recommended approach" | OPEN |
| T-5-NEW-10 | A forged "Sent to AI-player:" line can still be written by the model in four forms | this audit | An invisible space, a no-break space, a "1." leader, and a look-alike letter slip past the cleaner. Governed by R-09 → **AR-5-02** (same acceptance as T-5-NEW-03) | PARTIAL |
| T-5-NEW-11 | AI-chatter answers for a profile that never accepted the policy | this audit | Nothing in the chat handler consults policy acceptance; the panel hides itself, the server does not check. Governed by R-10 → **DR-5-03**; owner: "AI-Chat and AI-Player should not ever become available unless the AI policy is accepted in the game profile.  Fix this as a bug in the next Phase." | OPEN |

Two rows on the main register were OPEN at the audit; both now carry a disposition above (T-5-46 accepted, T-5-41 fixed and closed). One row was DEFERRED at the audit and remains an active deferral (T-5-26).

---

## Register items that were not security risks

The owner gave a standing rule in chat on 2026-09-18, quoted verbatim: **"These are not security risks.  Don't treat this as a backlog review.  If you have end of Phase suggestions, that need a place in upcoming Phases, then make a separate report."** Two register items fall under that rule. Neither appears in the Accepted or Deferred tables below, and neither is a risk this record tracks or that blocks project close.

- **R-14 — A standing suggestion can make the AI play worse** (walkthrough observation, agenda item 11). During the walkthrough a standing coaching line ("always look at the room before moving to a new room") made AI-player issue `look` several times in a row at one spot before moving on. Nothing was blocked and nothing harmful was sent; this is a play-quality observation, not a security risk. Owner's note: "We are going to address this in advanced logic for the AI-Player later." Under the owner's own rule this is not a deferral and does not block project close. It is carried in `05-PHASE-SUGGESTIONS.md`, item (a), under the owner's planned advanced AI-player logic work.
- **R-18 — Small things left undone** (agenda item 12; IN-02, IN-03, IN-04, IN-07; harness C1). Owner's note: "Not a security risk." Carried in `05-PHASE-SUGGESTIONS.md`, items (c), (d), (g) and (i).

---

## Corrections to the agenda

Findings the audit made about `05-SECURITY-AGENDA.md` itself, carried into this record because the agenda is not the record of the owner's decisions — this file is:

1. **Three rows were mislabelled.** `05-SECURITY-AGENDA.md` Part 3 lists T-5-41, T-5-55 and T-5-56 as "dispositioned accept inside the plans." All three are `mitigate` in their plans, not `accept`. T-5-55 and T-5-56 are in fact fully mitigated; T-5-41 was only half mitigated at audit time (the read-it-first sentence was missing) and has since been fixed, commit `6fb1c9b`.
2. **The corpus rerun's cost was understated.** Item 9 states the rerun happened "on a fresh-quota day" without saying that three live runs shared the key within five minutes and the kept run lost 10 of 40 items to rate pressure (`failed-model`).
3. **Five items were missing from the agenda entirely:** the conversation's missing retention and delete path (register R-11); chat's own cost numbers no longer bounded by the Call Cap (register R-12); forged quoted "Sent to AI-player:" lines (part of register R-09); AI-chatter answering without a policy-acceptance check (register R-10); and quoted owner text and the character's name in committed planning notes (register R-17).
4. **The harness's C1 check (pause and resume) was SKIP in both staging runs**, because autopilot was off each time the harness ran and it never engages it itself. The pause-and-resume criterion rests entirely on the owner's own walkthrough log and screenshot, not on the harness.
5. **The filed staging log excerpt carries a startup line for only 3 of the 6 deployments.** The walkthrough's own deployment header has nothing under it, and the final deployment is not in the file at all.

---

## Accepted Risks Log

Decisions recorded by the owner on the Phase 5 Risk Register (https://claude.ai/artifact/Ep5B1NwSKXUJ3eAYsUJf8n) on 2026-09-18 and read back from it. Accepted risks are closed for good unless the owner reopens them; they are not re-presented at later reviews. Owner notes are verbatim; "—" means the owner left no note.

| Risk ID | Register item | Criticality | Risk | Proposed remediation (not taken) | Owner note | Accepted By | Date |
|---------|---------------|-------------|------|----------------------------------|------------|-------------|------|
| AR-5-01 | R-01 / T-5-46 (audit: OPEN) | low | An older, unredacted copy of the Phase 5 test report — printed because the harness had not yet been fixed to redact AI-settings bodies — is still in the branch's unpushed git history (commit `1d2e0c2`) and holds the owner's two conduct rules and the Never-issue list seven times. The current file on disk is clean. Pushing the branch puts that old history on GitHub too. It holds no key, password, goal, chat text or character name. | Rewrite the unpushed commits so the old version never reaches GitHub, then push. | Push as is. | Owner | 2026-09-18 |
| AR-5-02 | R-09 / T-5-NEW-03 + T-5-NEW-10 | low | Two smaller gaps in the chat checks. The withdraw gate can be tripped by a question that merely shares a word stem with a standing line, or by an ordinary question that happens to contain a take-back phrase, and its word match is on the first four letters only ("playing" matches "player"). Separately, a fooled AI-chatter can still write a fake "Sent to AI-player:" line using an invisible space, a no-break space, a numbered-list leader, or a look-alike letter, which the server's own cleaner does not strip. A faked line cannot hide a real one or change what the AI reads; it could only mislead the owner about what was sent. | Match on five letters instead of four, only honour a take-back phrase when it is genuinely what the message is about, and clean look-alike and invisible characters before checking for the prefix. | — | Owner | 2026-09-18 |
| AR-5-03 | R-12 / T-5-NEW-04 | low | The Call Cap no longer limits every AI call. AI-chatter now keeps its own call count so it can still answer after AI-player hits the cap, but that count uses the same cap number (or 100 when blank) and restarts at every new stint and every Resume, so total calls in a login are no longer bounded by the cap alone. Only the owner can send chat messages, so this is the owner's own spending; on the free tier the cost is quota, not money. | Keep chat's own count, but stop it restarting at each stint (once per login instead), and show it in the panel beside the play count. | AI-CHAT and AI-PLAYER are different in nature and the Call Cap should only affect the AI-PLAYER.  Keep as is. | Owner | 2026-09-18 |
| AR-5-04 | R-15 / T-5-NEW-08 + T-5-09 | medium | The safety checker has never been fully tested on the model staging now runs. Staging was switched to `gemini-3.1-flash-lite` when the free daily quota for `gemini-3.5-flash-lite` ran out, and stays there by the owner's choice. The 40-item hostile-text corpus ran on 3.5 only. The one live test on the staging model is the chat test, in which that model obeyed hostile text twice; the server's own check stopped it both times, not the model. Also, 10 of 40 items in the kept hostile-text run were lost to the free tier's per-minute limit because three live tests shared the key within five minutes; every item was measured once across the two attempts, none twice. | Run the full hostile-text test on `gemini-3.1-flash-lite` on a fresh-quota day, by itself with nothing else sharing the key, and file the report. If it does worse than 3.5, switch staging back. | — | Owner | 2026-09-18 |
| AR-5-05 | R-16 / found by the audit, bears on AR-4-08 | medium | Every command the owner types by hand is written to the server log word for word, including a password typed at a MUD's own login prompt (rather than using saved credentials). This predates Phase 5 (the code is from 2026-03-06) and nobody had flagged it before this audit. Chat text never reaches these lines and the AI's commands are logged only by length. Anyone with access to the Railway project's logs — the same access AR-4-08 is about — could read a hand-typed password there. | Log the length of a hand-typed command, not its words, the same way the AI's own lines already do. | — | Owner | 2026-09-18 |
| AR-5-06 | R-17 / T-5-NEW-06 | low | The evidence reports and the log excerpt are clean, but committed planning documents (`05-11-WALKTHROUGH-NOTES.md`, `05-11-SUMMARY.md`, the agenda) quote the owner's two coaching lines verbatim, a few sentences of the AI's reasoning, and the character name Ulwynn, because they describe what happened. Pushing the branch puts those on GitHub. All harmless in content. | Reword the notes to describe without quoting, before the push. | — | Owner | 2026-09-18 |

*Accepted risks do not resurface in future audit runs.*

---

## Deferred Risks

Temporarily accepted so Phase 5 can close. Each is raised again at the Phase 6 security review with the same three choices. The project cannot be considered closed while any deferral is active. Carried forward in `.planning/todos/pending/2026-09-18-phase6-security-carry-forward.md`.

| Risk ID | Register item | Criticality | Risk | Proposed remediation | Owner note | Deferred By | Date | Raised again at |
|---------|---------------|-------------|------|----------------------|------------|-------------|------|-----------------|
| DR-5-01 | R-06 / T-5-NEW-07 / IN-06 / T-5-52 | low | The local-development switch that lets a developer's own machine start without a vault key is honoured anywhere, including on Railway. If it were ever set on staging or production, the server would go back to inventing a key at every restart and saved passwords would break again. It is not set on staging today, and the rule against setting it is a written instruction, not something the code enforces. | Make the server refuse the switch when it can see it is running on Railway, and say why. | fix this in next phase per the recommended approach | Owner | 2026-09-18 | Phase 6 security review |
| DR-5-02 | R-08 / T-5-26 (deferred to this review, high) + T-5-NEW-02 | high | AI-chatter reads the game's own text and it can write a short line that AI-player then reads as guidance from the owner. A hostile room could try to talk AI-chatter into "relaying" a line. Two checks stand in the way and no hostile line got through in live tests (0 of 8, then 0 of 10), but the audit found two ways round the word-overlap check: short words (under four letters) do not count, and the check reads the whole line while only the first 200 characters are stored. Both need AI-chatter to have been fooled first, and every line is still quoted back to the owner. | Count every word that is not a filler word whatever its length, and run the check on exactly the text that gets stored. Add both cases to the tests. This narrows the gap; it cannot remove it. | We are going to address this in advanced logic for the AI-Player later. | Owner | 2026-09-18 | Phase 6 security review |
| DR-5-03 | R-10 / T-5-NEW-11 | low | AI-chatter will answer for a profile that never accepted the Safety and Abuse policy. The play screen hides the panel on such a profile, but the server itself does not check before answering a chat message sent by hand rather than through the page. It could not make the AI play — autopilot is still gated — but it spends an AI call and shows game text and notes for that profile. **The owner's note adds a requirement: AI-Chat and AI-Player must never become available unless the AI policy has been accepted in the game profile.** | Have the server ask the same policy gate before it answers a chat message, and refuse with the same plain notice. | AI-Chat and AI-Player should not ever become available unless the AI policy is accepted in the game profile.  Fix this as a bug in the next Phase. | Owner | 2026-09-18 | Phase 6 security review |
| DR-5-04 | R-11 / T-5-NEW-09 | low | The conversation with AI-chatter is kept forever, and the owner's "Delete Captured Text Now" button does not remove it. Phase 4 gave stored game text a 7-day life and transcripts 30 days, with a delete button covering both; the new conversation follows neither rule, even though AI-chatter's replies can repeat game text, including other players' words. | Give conversation lines the same 30-day life as transcripts and include them in Delete Captured Text Now. | Fix this in the next Phase per the recommended approach | Owner | 2026-09-18 | Phase 6 security review |

---

## Closures confirmed at this review

Four deferrals were carried into this review from Phase 4, plus one owner condition and one audit OPEN row. All six are closed here, not carried forward as risks.

| Carried in | Closed by | Evidence |
|-----------|-----------|----------|
| DR-4-01 (the checker blocked ordinary fighting inconsistently) | **Remediated** by Phase 5: the reviewer is now told plainly that fighting a target the game presents is ordinary play; the two real false blocks became repeat-sampled corpus items, measured not blocked in 9 of 9 measured samples across two attempts (3 of 9 unmeasured due to transport failure, never folded into "not blocked"). Register R-02, closed by the owner at this review. | `internal/driver/driver.go` `buildReviewSystemInstruction`; `evidence/04-redteam-after.txt:43-48`, `evidence/04-redteam-after-attempt1.txt:43-48`. **Stated residual:** every measured sample was on `gemini-3.5-flash-lite`; nobody has yet watched the AI get through the tutorial's fights on staging, and staging now runs `gemini-3.1-flash-lite` — carried forward as AR-5-04 (R-15) |
| DR-4-02 (a missing verdict read as "not blocked"; the reviewer told it wrote the memory) | **Remediated** by Phase 5: a missing `blocked` field is now a failed review, which never sends; the reviewer is told the memory notes were written by the other model. Register R-03, closed by the owner at this review. | `internal/gemini/client.go` `ReviewCommand`; `TestReviewCommand_MissingBlockedField`, `TestBuildReviewSystemInstruction_MemoryAuthorship` |
| DR-4-03 (the speed-limit backstop never counted the AI's commands) | **Remediated** by Phase 5: every AI command is now counted by its own per-user limiter immediately before it is sent, and per the owner's own condition it is a configurable profile setting (1 to 20 per second; blank means the server default of 2). Register R-04, closed by the owner at this review. | `internal/driver/ratelimit.go`; `internal/store/profile.go`; `evidence/12-ai-settings-rate-limit.png`. **Stated residual (IN-04):** the bucket can pass up to double the limit across a window edge, and a stored value of 0 would stall the AI; neither is reachable from game text |
| DR-4-04 (the vault-key requirement only ran on Railway) | **Remediated** by Phase 5: the key is now required on every host, with a named local-development switch as the only opt-out, announced in the log. Register R-05, closed by the owner at this review. | `internal/config/config.go`; `TestLoadRequiresEncryptionKeyEverywhereUnlessLocalDev`. The switch being honoured on Railway too is carried forward as DR-5-01 (R-06) |
| AR-4-08's condition (the in-app log display must sit behind the signed-in user's login) | **Met and confirmed.** The `/logs/:connectionId` route sits inside `AuthGuard` with a comment naming this condition; a private browser window lands on sign-in with no transcript. Register R-07, closed by the owner at this review. | `frontend/src/App.tsx`; `evidence/13-logs-signed-out.png`; the `LOGS ROUTE GUARD` section of the Phase 5 test report. **What remains the owner's own task:** reviewing who can run `railway logs` against staging |
| T-5-41 (the help article never told the owner to read a suggestion before pasting it) | **Fixed.** The read-it-first sentence was added, and stray backticks removed, the same day the audit found the gap: commit `6fb1c9b`. Register R-13, closed by the owner at this review, who also stated it was not a security risk in the first place. | `help/ai-coaching.json`; commit `6fb1c9b` |

---

## Security Audit Trail

| Audit Date | Threats Total | Verified/Closed | Open-for-decision | Open (mitigation absent or deferred) | Run By |
|------------|----------------|------------------|--------------------|----------------------------------------|--------|
| 2026-09-18 | 58 threat ids (52 verified, 3 accepted at plan level, 1 deferred to the review, 2 open) plus 11 unregistered items (2 verified, 9 open or partial) | 55 threat ids closed by the audit | 18 register items listed for the owner (R-01 to R-18) | 3 (T-5-46 accepted-risk facet, T-5-41 fixed same day, T-5-26 active deferral) | gsd-security-auditor |
| 2026-09-18 | 18 register items (R-01 to R-18) | 6 accepted (AR-5-01 to AR-5-06), 4 deferred (DR-5-01 to DR-5-04), 6 closures (DR-4-01 to DR-4-04 remediated and closed, AR-4-08's condition met, T-5-41 fixed) | 0 | 0 without an owner disposition; 2 items reclassified as not security (R-14, R-18) and carried to `05-PHASE-SUGGESTIONS.md` | Owner, on the Phase 5 Risk Register artifact |

After the owner's decisions every item has a disposition: 55 plan-time threat ids verified or closed by the audit, one accepted (T-5-46), one fixed the same day (T-5-41), one deferred (T-5-26); four earlier Phase 4 deferrals confirmed remediated and closed; one Phase 4 condition (AR-4-08) confirmed met; 6 risks accepted, 4 deferred to the Phase 6 review, none remediated now; 2 items reclassified as not security under the owner's own rule and carried in a separate report. `threats_open: 0` on that basis and no other.

**The policy stays at version 1.0.** No rewording of `.specify/specs/safety-and-abuse-policy-v1.md` was made or considered this phase.

**`ai-player` is pushed after this record is committed, as-is, per AR-5-01.** No history rewrite was performed; the owner's decision was to push the branch as it stands.

---

## Sign-Off

- [x] All 58 register threat rows plus 11 unregistered items have a disposition; 55 threat ids are CLOSED/VERIFIED with cited evidence, and the 3 remaining are recorded as not mitigated at audit time, each now carrying the owner's decision (T-5-46 accepted as AR-5-01; T-5-41 fixed same day as commit `6fb1c9b`; T-5-26 deferred as DR-5-02)
- [x] Code-review findings (CR-01, CR-02, WR-01 to WR-15) confirmed fixed in code by the audit; unfixed info findings triaged into register items or carried in `05-PHASE-SUGGESTIONS.md`
- [x] Accepted risks documented in the Accepted Risks Log (AR-5-01 to AR-5-06) with the owner's notes verbatim
- [x] Deferred risks documented and carried to the Phase 6 review (DR-5-01 to DR-5-04), with the owner's instructions recorded verbatim, including the added requirement on DR-5-03 that AI-Chat and AI-Player must never become available without policy acceptance
- [x] All four deferrals carried in from Phase 4 (DR-4-01 to DR-4-04) have a recorded outcome, and AR-4-08's condition is confirmed met
- [x] Two register items reclassified as not security (R-14, R-18) per the owner's own rule, and recorded in `05-PHASE-SUGGESTIONS.md` instead of this file's Accepted or Deferred tables
- [x] `threats_open: 0` confirmed, on the basis stated at the top of this record
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-09-18 — owner decisions recorded on the Phase 5 Risk Register (https://claude.ai/artifact/Ep5B1NwSKXUJ3eAYsUJf8n); no Remediate Now decision, so Phase 5 closes. The push of `ai-player` follows this record.
