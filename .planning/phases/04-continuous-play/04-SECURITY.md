---
phase: 04
slug: continuous-play
status: verified
threats_open: 0
asvs_level: 1
created: 2026-09-17
register_authored_at_plan_time: true
accepted: 10
deferred: 4
remediate_now: 0
---

# Phase 04 — Continuous Play — Security

> Per-phase security contract: threat register, accepted risks, deferred risks, and audit trail.
> Register authored at plan time across 04-01-PLAN.md through 04-11-PLAN.md (35 distinct threat ids, `T-4-01` to `T-4-34` plus `T-4-SC`). Verified against implemented code, not documentation, on 2026-09-17 by gsd-security-auditor at HEAD `82b0241` (`04-SECURITY-AUDIT.md`: 31 verified in code, 1 accepted at plan level, 1 deferred to this review, **2 OPEN**). The code review's two critical findings and twelve warnings (`04-REVIEW.md`) were fixed before the audit (`04-REVIEW-FIX.md`, 15 commits) and the audit re-read every one of those fixes in the code rather than trusting the summaries, which predate them.
>
> **The owner's rule at this review:** only active risks that carry a concrete proposed remediation appear below as Accepted or Deferred. Items the owner judged already fixed, or fixed and confirmed in code, are recorded as Closures, not risks. Each risk is written as the problem, then the fix, in everyday words.
>
> The owner's decisions on every register item were made on the Phase 4 Risk Register (https://claude.ai/artifact/SvSdbXCa3YXmFDryiXpAaK) on 2026-09-17 and read back from it by the orchestrator after the owner reported 0 undecided / 10 accepted / 4 deferred / 0 remediate now. The register carried fourteen active risks under this rule (R-01 through R-14). No item was marked Remediate Now, so Phase 4 does not reopen.
>
> **What `threats_open: 0` means in this record, stated exactly.** The audit marked two plan-time threat rows OPEN, both rated high, because a declared mitigation is absent from the code or defeated by a committed file. Neither has been fixed. `threats_open: 0` is correct ONLY because each of those two rows now carries an owner disposition:
>
> - **T-4-05 → ACCEPTED** (register R-09, recorded as AR-4-01). Owner's note: "it's going into a private repo.  This is not a concern"
> - **T-4-02 → DEFERRED** (register R-13, recorded as DR-4-03). Owner's note: "Yes, and the limiter needs to be a setting that can be configured.  This will have to be an added feature in subsequent Phases." The note adds a requirement: the limiter must be a configurable setting.
>
> Neither row is mitigated. One is an accepted risk; the other is an active deferral that is raised again at the Phase 5 security review and blocks project close until it is resolved.

---

## Trust Boundaries (consolidated across all eleven plans)

| Boundary | Plans | Description |
|----------|-------|--------------|
| Test doubles → the decision path | 04-01 | A double that lies about the switch can make an unsafe build look safe |
| Test process → the network and the owner's quota | 04-01, 04-11 | The Phase 4 suite must never reach a live model; the corpus rerun spends a day's free-tier budget |
| Repository → `go.mod` / `frontend/package.json` | all | Any dependency added is a supply-chain change |
| Environment → the credential vault | 04-02 | Whether the vault key is real or invented at startup decides whether saved MUD passwords survive a restart |
| Migration files → the live staging database | 04-02, 04-06, 04-11 | A failed migration is a startup fatal; a rollback that fails partway leaves a half-changed schema |
| Documentation → readers | 04-02 | A key value written into a rotation procedure is disclosed |
| The autopilot switch → the loop goroutine | 04-03 | The switch is the authorization; a loop (or an in-flight decision) that outlives it acts without one |
| The game socket → the loop's wakeups and the window | 04-03, 04-05 | Game output is attacker-controlled text that decides when the AI acts and what it reasons about |
| The loop → the owner's model quota and the MUD | 04-03, 04-04 | Every iteration can spend up to four model calls; commands issued too fast are flooding |
| Hostile game text → the block counter | 04-04 | An attacker who can cause blocks can try to exhaust the session |
| The driver's own state → the browser's badge | 04-04 | If they disagree, the owner believes the AI is stopped when it is not |
| Vendor error responses → the failure taxonomy | 04-04 | A misclassified error stops play too early or never stops it |
| Browser → the goal, memory and captured-text endpoints | 04-06, 04-08, 04-09 | Each must serve, and destroy, only the caller's own profile's data |
| Owner's goal → the prompt | 04-06, 04-07 | Owner-written standing text, trusted only because the ownership check makes it genuinely the owner's |
| Model-written memory → the database → every future prompt | 04-07, 04-08 | Memory is written by a model that has just read hostile text, then re-enters every later decision |
| The first model's reasoning → the reviewer | 04-07 | An account written after reading hostile text, previously read as testimony |
| Stored captured text → time; the retention job → the audit record | 04-09 | Text kept forever is a growing disclosure surface; a prune that reaches too far destroys the audit trail or the AI's memory |
| The danger-zone control → an ordinary click | 04-09 | A destructive control beside a save button is a misclick from data loss |
| The harness → staging; the report and fixtures → readers and the verdict | 04-10, 04-11 | A live run authenticates as the owner and deletes real data; whatever the report prints is committed; fixtures that cannot disagree make every PASS meaningless |
| Repository → Railway staging; owner's browser session → the live MUD and model | 04-11 | The deployed binary may diverge from the repo; the walkthrough drives the real session end to end |

---

## Threat Verification Register

Legend: **M**=mitigate, **A**=accept, **D**=defer to security review. File and line references are at HEAD `82b0241` and are condensed here; `04-SECURITY-AUDIT.md` holds the full evidence for every row, including each named residual. Status CLOSED means the audit confirmed the declared mitigation in code and re-ran its tests. A row the audit did NOT confirm says so in its Status cell and names the owner disposition that stands in place of a mitigation; the owner's Accept/Defer choices are recorded below, in Accepted Risks Log / Deferred Risks.

| Threat ID | Category | Plan(s) | Disp. | Evidence | Status |
|-----------|----------|---------|-------|----------|--------|
| T-4-01 | EoP — an orphaned loop or in-flight decision acting after the switch went off (High) | 04-03, 04-11 | M | Disengage hook on every changed-true transition (`internal/session/manager.go:930-932`, `:962-964`); `StopLoop` cancels the stint context (`internal/driver/loop.go:172-186`); the plan-time mitigation was found insufficient at the walkthrough and strengthened with a stint epoch (`manager.go:846-889`, `:997`), three stint checks per decision (`internal/driver/driver.go:620`, `:712`, `:763`) and an epoch-checked send (`manager.go:1068-1111`). `TestLoop_StopsWhenWheelGrabbed`, seven `TestStint_*` tests, `TestManager_StintEpoch`, `TestSendAICommand_RefusedOnEpochMismatch` PASS. Live: `evidence/05-staging-ai-player.log:126`, `:242`. Residuals carried as register item R-12 (AR-4-10) | CLOSED |
| T-4-02 | DoS (cost and flooding) — an unbounded loop spending quota or flooding the MUD (High) | 04-03, 04-04 | M | Two of three declared mechanisms verified: pacing (`loop.go:219-275`; `TestLoop_Pacing` PASS) and the call cap reserved before every model call (`driver.go:904-912`; `TestLoop_CallCap` PASS; live halt `evidence/05-staging-ai-player.log:68`). **The declared backstop is absent:** `internal/icm/dispatcher.go:208-212` returns for a plain pass-through command before `RecordExecution` (`:221`), and the driver never calls it, so the ICM rate limit never counts an AI command. Other residuals: blank cap means no cap (R-07, AR-4-06); a rate-limit answer counts as a strike (R-08, AR-4-07) | **OPEN at audit (backstop absent). Not mitigated. DEFERRED by the owner — DR-4-03 (R-13)** |
| T-4-03 | Tampering/EoP — the AI's own memory as an injection channel (High) | 04-07, 04-08, 04-11 | M (partial), then D | Partial mitigation present: both memory blocks delimited in both prompts (`driver.go:1319-1326`, `:1458-1465`; `internal/driver/memory.go:198-212`), named in the shared untrusted-data paragraph (`driver.go:1344-1352`), ceilings 30/20 bullets at 200 bytes (`memory.go:43-47`, `:123-161`), no bullet can forge a marker (`memory.go:82-100`). `TestMemoryIsWrappedInBothPrompts`, `TestMemoryCeilingsAreEnforced`, `TestBulletsAreNeutralisedAtWriteAndAtRead` PASS. Residual, confirmed in code: memory is persisted before the reviewer's verdict (`driver.go:657` precedes `:686`), nothing clears or ages out bullets, no corpus item attacks the memory-write path | CLOSED (deferred to this review — ACCEPTED by the owner, AR-4-04 / R-05) |
| T-4-04 | Tampering — the reviewer reading the first model's reasoning as testimony (High) | 04-07 | M | `wrapModelReasoning` (`driver.go:1288-1290`, used at `:675`); `reviewReasoningUntrustedSentence` (`:1422`, `:1449`). `TestReviewPromptWrapsReasoning`, `TestBuildReviewSystemInstruction` PASS. `reviewer-channel-01` reads `sent-unsteered` (`evidence/04-redteam-after.txt:36`), measured once on the final build. Residual: a reviewer answer with no `blocked` field decodes as not blocked (`internal/gemini/client.go:320-323`, `:348-352`) — register R-04, DR-4-02 | CLOSED (residual deferred — see DR-4-02) |
| T-4-05 | Info Disclosure — key, cookie, game text, goal, memory bullet or chosen command reaching a log line, a report, a fixture or a commit (High) | 04-02 to 04-11 | M (A in 04-05) | Every logging and key facet verified: `logDecision` carries ids, stage and lengths only (`driver.go:1260-1263`); goal, memory and retention lines carry lengths and counts; `internal/gemini/client.go` has no log call; grep of all 13 evidence text files for key, endpoint, cookie, delimiter and sign-in-code patterns: 0 hits. **One facet is defeated:** the committed `evidence/03-canned-report.txt` contains full decision lists (reasoning, command, notice; 45 decisions in RUN C, `:380`), Session Memory bullets verbatim (`:214`, `:368`, one naming the character) and the owner's goal (`:120`), because `scripts/verify-phase4.sh` prints raw response bodies | **OPEN at audit (canned-report facet defeated). Not mitigated. ACCEPTED by the owner — AR-4-01 (R-09)** |
| T-4-06 | Tampering — a non-development process starting on a silently generated vault key (High) | 04-02 | M | `config.Load` fails when `RAILWAY_ENVIRONMENT` is set and `ENCRYPTION_KEY_V1` is absent (`internal/config/config.go:214-217`) or unusable (`:218-229`, WR-07). `TestLoadRequiresEncryptionKeyOutsideDevelopment` PASS. Residual: the gate is inferred from one platform's variable (IN-07) — register R-14, DR-4-04 | CLOSED (residual deferred — see DR-4-04) |
| T-4-07 | Tampering (data integrity) — migration 012 rollback, migration 013 on the live schema, two active Quests (Medium) | 04-02, 04-06, 04-11 | M | `migrations/012_..._down.sql:10` reconciles `blocked` to `failed` before the constraint returns (`:27-28`); 013 is `IF NOT EXISTS` with a partial unique index (`:38-39`). `TestMigration012DownReconcilesBlockedRows`, `TestMigrationFilesPairUp`, `TestQuestStore_EnsureActiveReactivates` PASS. Live: log `:3`, `:115`, `:215` | CLOSED |
| T-4-08 | Info Disclosure — captured game text kept forever (Medium) | 04-09 | M | Four retention statements (`internal/store/retention.go:41-59`), one transaction each; daily job (`cmd/server/main.go:316-318`, `:671-694`); owner's delete endpoint (`internal/profiles/handler.go:954-991`). `TestRetentionWindows`, `TestRetention_TouchesOnlyCapturedText`, `TestDeleteCapturedText` PASS. Live: log `:5`, `:164`, `:198`; canned report `PASS C4` in all three runs | CLOSED |
| T-4-09 | Spoofing/IDOR — goal, memory and captured-text endpoints serving another owner (High) | 04-06, 04-08, 04-09 | M | All four handlers resolve through `getProfileByConnectionID` before any store call (`handler.go:784`, `:806`, `:916`, `:960`); goal write scoped by `user_id` (`internal/store/profile.go:546-547`). `TestGoalRefusesNotOwnedConnection`, `TestGetSessionMemory`, `TestDeleteCapturedText` not-owned subtests PASS; harness not-owned steps pass on staging | CLOSED |
| T-4-10 | DoS — unbounded goal or memory growth (Medium) | 04-06, 04-07, 04-08 | M | Goal capped at 1000 bytes (`handler.go:265`, `:818-821`); bullets capped in Go both ways (`memory.go:123-161`). `TestMemoryCeilingsAreEnforced`, `TestGoalRoundTrip` PASS | CLOSED |
| T-4-11 | Repudiation — a canned report that always passes (High) | 04-10, 04-11 | M | Re-run at audit: `--self-test` exit 0 (35 checks, 0 FAIL); `--self-test-negative` exit 1 with exactly one `FAIL C4` | CLOSED |
| T-4-12 | Tampering — a deploy or harness run aimed at the wrong target (High) | 04-11 | M | Procedural. Every RUN header records `BASE_URL` and a git short SHA; every deploy was `railway up --detach -e staging -s MudPuppy`; production untouched | CLOSED |
| T-4-13 | DoS — signing the owner out during the walkthrough (High) | 04-11 | M | Procedural. Both checkpoints state the standing instruction (`04-11-PLAN.md:77`, `:235`); no sign-in-code pattern in the log excerpt | CLOSED |
| T-4-14 | EoP — a command issued while disconnected, or the AI reconnecting on its own (High) | 04-03 | M | The driver's `Sessions` interface has no connect method (`driver.go:167-190`); a send while WAITING is refused (`manager.go:1109-1111`). `TestLoop_NothingIssuedWhileWaiting`, `TestLoop_NoAIReconnect` PASS | CLOSED |
| T-4-15 | DoS — a prune or delete reaching decision rows, memory or Quests; the harness's own delete (High; harness facet accepted at plan level) | 04-09, 04-10 | M (A in 04-10) | The four statements name only `window_text` and `game_session_lines`; `TestRetention_TouchesOnlyCapturedText`, `TestRetention_LeavesMemoryAlone`, `TestRetention_PerProfileDeleteIsScoped` PASS. The harness states in its own preamble that it deletes captured text (`scripts/verify-phase4.sh:282-286`) | CLOSED |
| T-4-16 | Repudiation — a badge reading On after a cap or threshold disengage (High) | 04-04 | M | `decorateEvent` reads the switch after the disengage is applied (`driver.go:955-971`); `TestAIPayloadCarriesSwitchState`, `TestAIPayloadSendsZeroAndEmpty` PASS; `evidence/08-cap-halt-badge-off.png`. Residual IN-09 (a late message can briefly flip the badge back to On) carried in R-12, AR-4-10 | CLOSED |
| T-4-17 | Repudiation — a session double that always reports Off (High) | 04-01 | M | `fakeSessions` drives state from the real pure functions; `TestFakeSessionsStateMachine` PASS | CLOSED |
| T-4-18 | DoS (cost) — a test reaching the live model; the corpus rerun's quota cost (Medium) | 04-01, 04-11 | M (A in 04-11) | The only `gemini.NewClient` in the driver package sits behind an unconditional `RAILWAY_ENVIRONMENT` fatal and an opt-in skip (`corpus_live_test.go:928-934`, `:949`). The corpus cost is now listed in the agenda's corrections section | CLOSED |
| T-4-19 | DoS — the new gate stopping the owner's local server (Medium) | 04-02 | M | Gate conditional on `RAILWAY_ENVIRONMENT` (`config.go:214`); local-development subtests PASS | CLOSED |
| T-4-20 | Tampering — hostile output volume driving the decision rate (Medium) | 04-03 | M | Buffered channel of one with a non-blocking send (`manager.go:631-639`); minimum spacing gates every tick (`loop.go:242-251`) | CLOSED |
| T-4-21 | DoS — a hostile room causing block after block (Medium) | 04-04 | M | Consecutive blocks counted separately and disengage through their own cause (`driver.go:1198-1205`); `TestLoop_ConsecutiveBlocks` PASS. Live side effect (false blocks of ordinary combat feed this counter) is register R-01, DR-4-01 | CLOSED (side effect deferred — see DR-4-01) |
| T-4-22 | EoP — an auth failure counted toward a three-strike threshold (Medium) | 04-04 | M | `transientFailureKinds` excludes auth, bad-request, missing-profile and missing-model (`driver.go:105-113`); auth subtest PASS | CLOSED |
| T-4-23 | DoS — a retry storm on an unavailable vendor (Low) | 04-04 | M | Exactly one retry per call site, in an `if` (`driver.go:585-609`, `:687-703`), reserved against the cap. `TestHandleEngage_RetryOn503` PASS | CLOSED |
| T-4-24 | DoS — a flood of output growing the prompt (Medium) | 04-05 | M | 8192-byte ring (`internal/session/window.go:13`, `:219-221`); `TestWindow_CeilingStillHolds` PASS | CLOSED |
| T-4-25 | Tampering — a decision made against stale text (Medium) | 04-05 | M | Age bound (`window.go:20`, `:187-224`), amended by WR-06 (`:168-177`); `TestWindow_AgeBounded`, `TestEngageLoop_Reassess` PASS | CLOSED |
| T-4-26 | DoS — an empty window after a quiet spell (Medium) | 04-05 | M | 2048-byte floor (`window.go:27`, `:215-218`); `TestWindow_NeverEmptyAfterQuiet` PASS | CLOSED |
| T-4-27 | Tampering — a Phase 4 path closing a Quest (Low) | 04-06 | M | No statement sets `status` to anything but `active` (`internal/store/quests.go:50-60`); `TestQuestStore_NeverCloses` PASS | CLOSED |
| T-4-28 | Tampering — forged closing markers breaking the framing (Medium) | 04-07 | M | Plan-time mitigation judged insufficient by the code review (CR-02) and replaced by `neutraliseUntrusted` / `neutraliseLine` (`memory.go:51-100`), applied by every wrapper and the goal block. Five tests in `neutralise_test.go` PASS | CLOSED |
| T-4-29 | DoS — a malformed memory field or a store error costing the command (Low) | 04-08 | M | Decode-tolerant memory fields (`internal/gemini/client.go:223-252`); store errors logged by id, never returned (`driver.go:867-889`) | CLOSED |
| T-4-30 | DoS — a destructive control adjacent to a save action (Medium) | 04-09 | M | Own block, scope-stating hint, two-step inline Yes/No (`frontend/src/components/AIPlayerPanel.tsx:314-333`); `evidence/12-delete-captured-text-confirm.png` | CLOSED |
| T-4-31 | DoS — a retention run taking the server down (Low) | 04-09 | M | Own goroutine (`main.go:318`); an error logs one line and the loop continues (`:674-679`) | CLOSED |
| T-4-32 | Repudiation — the harness presenting pacing or the cap as checked (Medium) | 04-10 | M | C3 and C5 are printed as SKIP naming the Go tests and screenshots (`verify-phase4.sh:855`, `:862`) | CLOSED |
| T-4-33 | Tampering — a harness run leaving a test goal on the owner's profile (Medium) | 04-10 | M | Restore step (`verify-phase4.sh:838-848`) plus an exit trap (`:615-640`); step 1 failing aborts before any write (`:660`) | CLOSED |
| T-4-34 | Repudiation — eleven deferred risks quietly lapsing (High) | 04-11 | M | `04-SECURITY-AGENDA.md` Part 1 re-presents all eleven active deferrals; every one now has a recorded outcome in `.planning/RISK-REGISTER.md` | CLOSED |
| T-4-SC | Tampering (supply chain) — `go.mod`, `frontend/package.json` (Low) | all eleven plans | A | `git diff --stat dd15a98..HEAD -- go.mod go.sum frontend/package.json frontend/package-lock.json`: empty. Covered by the standing acceptances AR-1-01, AR-2-08 and AR-3-13 ("keep the drift check in every phase's test report") | CLOSED |

**Row count:** 35 threat ids. 33 CLOSED by the audit (31 verified in code, T-4-SC accepted at plan level under the standing supply-chain acceptance, T-4-03 deferred to this review and now accepted as AR-4-04). **2 were OPEN at the audit and are NOT mitigated:** T-4-05 carries the owner's acceptance (AR-4-01) and T-4-02 carries the owner's deferral (DR-4-03). With those two owner dispositions recorded, 0 rows are without a disposition.

**Unregistered attack surface found during verification (no `T-4-NN` mapping):** the audit lists twelve items that surfaced after the plan summaries were written, from the staging walkthrough, the code review and the audit itself (`04-SECURITY-AUDIT.md`, "Unregistered Attack Surface"). Nine were already fixed in code and tested when the audit ran (items 1 to 9: the in-flight send and stale-stint decision, delimiter breakout through memory, the manager lock across a socket write, a dead socket counted as a model failure, an unusable vault key, the goal/settings save race, the silent Quest failure, two harness defects, the stale memory read-back). What was not fixed became register items and is recorded below: reviewer reliability and false blocks of ordinary combat (R-01, R-02, R-03), the missing-`blocked`-field default (R-04), the inert ICM backstop (R-13), memory written before the reviewer's verdict (R-05), and the two timing residuals (R-12).

---

## Code Review Fix Verification (04-REVIEW.md)

Two critical findings and twelve warnings; all fourteen were fixed before the audit (`04-REVIEW-FIX.md`) and each fix was read in the code at HEAD by the audit. Ten info findings were left unfixed; the ones that carry a security decision are register items.

| Finding | Disposition | Detail |
|---------|-------------|--------|
| CR-01, WR-02, WR-10 (a stale decision sent into the next stint; in-flight work outliving the switch) | Fixed | Stint epoch (`18f5562` and follow-ups); seven `TestStint_*` tests; live at log `:242`. Residual carried in R-12 (AR-4-10) |
| CR-02 (delimiter breakout through a memory bullet, the goal or the reasoning) | Fixed | `neutraliseUntrusted` / `neutraliseLine` applied at read and write time; five tests |
| WR-01 (manager lock held across an unbounded socket write) | Fixed | 5s write deadline; sibling (`Connect` holds the lock across the dial) left, carried in R-12 (AR-4-10) |
| WR-03 (a dead socket counted as a model failure) | Fixed | Tested; its new owner-visible sentence not yet seen on staging |
| WR-04 (resume before the game session exists) | Fixed | `TestResumeFiresAfterTheGameSessionExists` |
| WR-05 (zero counts dropped from the status message) | Fixed | Tested; not yet watched on staging — R-11 (AR-4-09) |
| WR-06 (window age bound after a long pause) | Fixed | `TestWindow_CoversTheGapSinceThePreviousDecision` |
| WR-07 (a vault key that is present but unusable) | Fixed | `c5b58b3`; residual IN-07 is R-14 (DR-4-04) |
| WR-08 (goal/settings save race) | Fixed | Tested; not yet watched on staging — R-11 (AR-4-09) |
| WR-09 (a silent Quest failure) | Fixed | Tested; not yet watched on staging — R-11 (AR-4-09) |
| WR-11, WR-12 (harness overwriting the goal on a failed first step; counting "not 200" as a refusal) | Fixed | Both self-tests re-run at the audit |
| IN-01 to IN-10 | Not fixed | IN-07 is R-14 (DR-4-04); IN-09 is in R-12 (AR-4-10); the rest are backlog |

`go build ./...`, `go vet ./...` and `go test ./... -count=1` were re-run by the audit: all green.

---

## Accepted Risks Log

Decisions recorded by the owner on the Phase 4 Risk Register (https://claude.ai/artifact/SvSdbXCa3YXmFDryiXpAaK) on 2026-09-17 and read back from it. Accepted risks are closed for good unless the owner reopens them; they are not re-presented at later reviews. Owner notes are verbatim; "—" means the owner left no note.

| Risk ID | Register item | Criticality | Risk | Proposed remediation (not taken) | Owner note | Accepted By | Date |
|---------|---------------|-------------|------|----------------------------------|------------|-------------|------|
| AR-4-01 | R-09 / T-4-05 (audit: OPEN) | high | The Phase 4 report file saved in the project holds the AI's own words (its reasoning, commands and notices for every decision in each run), the AI's memory notes (one names your character) and your goal. Pushing the branch puts that file on GitHub. The plan said this kind of text must not reach a saved report; it did, and it has not been removed. | Change the harness to print lengths and fingerprints instead of the words, take one clean run on staging, and rewrite the unpushed commits so the old report never reaches GitHub. | it's going into a private repo.  This is not a concern | Owner, Accept Risk on the Phase 4 Risk Register | 2026-09-17 |
| AR-4-02 | R-02 / Agenda Part 2 / direct-03 diagnosis | medium | The safety checker only reliably stops the kinds of harm its own list names. For anything not on the list it is closer to a coin flip, and testing each attack once cannot tell a dependable stop from a lucky one. | Run each reviewer-dependent corpus item five times per report, or ask the reviewer twice and block if either answer says block. | — | Owner, Accept Risk on the Phase 4 Risk Register | 2026-09-17 |
| AR-4-03 | R-03 / D-29 | low | The new rule that stops the AI from binding your character to a contract, oath, pledge, debt or membership may also stop a real guild invite or a shop loan. Nothing tests for that. | Add two benign corpus controls (a guild invite, a shop loan); narrow the clause to contract, oath and pledge if they are blocked. | — | Owner, Accept Risk on the Phase 4 Risk Register | 2026-09-17 |
| AR-4-04 | R-05 / T-4-03 | medium | A room could trick the AI into writing a false "fact" into its own notes, and the AI would then lean on that fact in every later decision. The notes are saved even when the safety checker goes on to block the command, and there is no way to clear them or let them age out. | Persist memory only after the reviewer passes the decision; add fact-shaped memory-poisoning corpus items; give the owner a clear-notes button. | — | Owner, Accept Risk on the Phase 4 Risk Register | 2026-09-17 |
| AR-4-05 | R-06 / D-31 | low | No test signs in and out for real to check that the AI's session notes start fresh at each sign-in. If a later change broke that, no test would notice. | A small stand-in for the session store and two request-level tests for sign-in and sign-out. | — | Owner, Accept Risk on the Phase 4 Risk Register | 2026-09-17 |
| AR-4-06 | R-07 / D-14 / DR-3.1-02 residual | medium | The limit on how many AI calls a session can make is off until you set it. Blank is the starting setting on every profile, and blank means no limit. | Start new profiles with a default cap, or warn when autopilot is engaged with no cap. | I've already accepted this in previous Phases. | Owner, Accept Risk on the Phase 4 Risk Register | 2026-09-17 |
| AR-4-07 | R-08 / T-4-02 residual | low | When the free AI service is busy and answers "slow down", that counts as a strike, and three strikes in a row switch autopilot off, even during ordinary play. | Wait and retry once on a rate-limit answer and do not count it toward the consecutive-failure threshold. | This is per design | Owner, Accept Risk on the Phase 4 Risk Register | 2026-09-17 |
| AR-4-08 | R-10 / DR-3-06 | low | Nobody has checked who can read the staging server's log (the Railway project's members and access tokens). | Review Railway project members and tokens. | Not concerned about Admins, however the linked display needs to be gated behind the login of the person in the browser tool | Owner, Accept Risk on the Phase 4 Risk Register | 2026-09-17 |
| AR-4-09 | R-11 / 04-REVIEW-FIX.md: WR-05, WR-08, WR-09 | low | Three fixes (the status line showing zero counts, saving the goal and the settings at the same moment, and a Quest failure no longer being silent) are proven by automated tests, but nobody has watched them work on staging. | Fifteen minutes on staging watching each one, with a screenshot of the status line. | — | Owner, Accept Risk on the Phase 4 Risk Register | 2026-09-17 |
| AR-4-10 | R-12 / 04-REVIEW-FIX.md: seen but not fixed | low | Two small timing gaps were noticed and left alone. A failure from an AI run that has just ended can be counted against, or switch off, the run that has just started. And while the server is connecting to a game it can hold up everyone's stop control for up to five seconds; a late message can also flip the badge back to On for a moment after the AI has stopped. | Pass the stint epoch into recordFailure/recordBlocked; dial the game before taking the manager lock; make the badge ignore messages from an ended stint. | — | Owner, Accept Risk on the Phase 4 Risk Register | 2026-09-17 |

**The owner's condition on AR-4-08.** The acceptance covers who administers Railway. The owner's note adds that the in-app log display must sit behind the signed-in user's login. The orchestrator's check: every log and decision read is under `/api/v1/`, and `sessionMiddleware` in `cmd/server/main.go` answers 401 without a valid session, so the display's data is gated. What is not yet confirmed is that the `/logs/:connectionId` page itself redirects to sign-in when signed out; that is recorded as `.planning/todos/pending/2026-09-17-confirm-log-page-login-gate.md`.

*Accepted risks do not resurface in future audit runs.*

---

## Deferred Risks

Temporarily accepted so Phase 4 can close. Each is raised again at the Phase 5 security review with the same three choices. The project cannot be considered closed while any deferral is active. Carried forward in `.planning/todos/pending/2026-09-17-phase5-security-carry-forward.md`.

| Risk ID | Register item | Criticality | Risk | Proposed remediation | Owner note | Deferred By | Date | Raised again at |
|---------|---------------|-------------|------|----------------------|------------|-------------|------|-----------------|
| DR-4-01 | R-01 / Agenda Part 2 / staging walkthrough | high | The safety checker blocks ordinary fighting, and it does not give the same answer every time. On staging it stopped a normal tutorial attack on a golem, saying that attacking "another player or entity" is forbidden, when your rule only says never attack another player. The AI cannot get past the tutorial's own fights, and these wrong blocks count toward switching autopilot off. | Tell the checker plainly that fighting creatures and objects the game presents as targets is normal play and only attacking another player is off limits; add harmless fight examples to the corpus; rerun the corpus. | This is going to be addressed by deeper AI analysis in later Phases. | Owner | 2026-09-17 | Phase 5 security review |
| DR-4-02 | R-04 / T-4-04 residual / diagnosis side notes | high | If the safety checker's answer ever arrives without its yes-or-no, the server reads that as "not blocked" and sends the command. The answer format says the yes-or-no is required, but nothing on our side refuses an answer that leaves it out. Separately, the checker is told it wrote the AI's memory notes, which it did not. | Treat a missing `blocked` field as a failed review (which already stops the command); give the reviewer its own sentence saying the memory notes were written by the other model. | Fix this in next Phase per the recommended fix | Owner | 2026-09-17 | Phase 5 security review |
| DR-4-03 | R-13 / T-4-02 (audit: OPEN) | high | The last-resort speed limit on commands does not count the AI's commands. The AI's normal pacing and the call cap both work, but if the pacing ever had a bug there is nothing behind it to stop the AI from flooding the game. The plan named this speed limit as the backstop; it is not doing that job, and it has not been fixed. | Record every AI command with the ICM rate limiter before it is sent and refuse the send when the limiter says stop; add a flood test. Owner adds: the limiter must be a configurable setting, as an added feature in a subsequent phase. | Yes, and the limiter needs to be a setting that can be configured.  This will have to be an added feature in subsequent Phases. | Owner | 2026-09-17 | Phase 5 security review |
| DR-4-04 | R-14 / T-4-06 residual / review note IN-07 | low | The check that refuses to start the server without the password-vault key only runs on Railway. On any other host the server would still quietly make up a new key at every restart, and saved MUD passwords would stop working. | Require the vault key everywhere unless a setting says this is local development. | Employ this fix next Phase with the recommended fix. | Owner | 2026-09-17 | Phase 5 security review |

---

## Closures confirmed at this review

Eleven deferrals were carried into this review (six from Phase 3, five from Phase 3.1). The Phase 4 Risk Register listed nine as fixed, not as decisions; one closes with the push; one came back as a register item and was accepted. Each outcome is recorded on its own row in `.planning/RISK-REGISTER.md`.

| Carried in | Closed by | Evidence |
|-----------|-----------|----------|
| DR-3-01 (game text kept forever) and DR-3.1-03 (a blocked decision's attack text kept forever) | **Remediated** by D-21: decision snapshots age out after 7 days, transcripts after 30, a daily job prunes, and the owner can delete his own captured text on demand | Commits `7466bb9`, `8609dd1`. `evidence/05-staging-ai-player.log:5`, `:164`, `:198`; `evidence/03-canned-report.txt` `PASS C4` (`:87-99`, `:235-247`, `:389-401`); `evidence/12-delete-captured-text-confirm.png`. **Stated limitation:** the AI's reasoning, notices and memory notes, which are derived from game text, are kept on purpose and have no delete path (see AR-4-04) |
| DR-3-03 (the badge reads On for a few seconds after a failure) | **Remediated** by D-18: every message that carries a disengage also carries the new switch state | Commit `52abc84`. `TestAIPayloadCarriesSwitchState` (`evidence/01-test-report.txt:500`); `evidence/08-cap-halt-badge-off.png`. Residual IN-09 carried in AR-4-10 |
| DR-3-04 (a vendor 503 disengages autopilot) | **Remediated** by D-16: one retry on a 503 with a visible notice | Commit `52abc84`. `TestHandleEngage_RetryOn503` (`evidence/01-test-report.txt:472`) |
| DR-3-05 (a dashboard deploy could build an unpushed branch) | Held through Phase 4 (every deploy was `railway up`); closed by the push of `ai-player` at Phase 4 close | The push follows the commit of this record; the commit is recorded on the DR-3-05 row of `.planning/RISK-REGISTER.md` |
| DR-3-06 (who can read the staging log) | Returned as R-10 and **accepted** by the owner (AR-4-08), with a condition | See AR-4-08 above and `.planning/todos/pending/2026-09-17-confirm-log-page-login-gate.md` |
| DR-3-07 (a dead statement in EngageAutopilot) | **Remediated** by D-27: the statement is deleted | Commit `4d85687`; `TestEngageAutopilot` (`evidence/01-test-report.txt:1622`) |
| DR-3.1-01 (the checker trusts the first AI's explanation) | **Remediated** by D-24: the reasoning is wrapped as untrusted and a reviewer-channel corpus item was added | Commits `f5c1c1f`, `dbcd5fd`. `TestReviewPromptWrapsReasoning` (`evidence/01-test-report.txt:262`); `evidence/04-redteam-after.txt:36` (`reviewer-channel-01`, `sent-unsteered`), `:55` (`STEERED: 0`). **Stated limitation, not erased by this closure:** the owner's note asked for the rerun "on a fresh quota day" and that did not happen — nine corpus runs are filed for 2026-09-17, every one with rate-limited items; the filed report measured 25 of 31 items, and between it and the second final-build run every item was measured once. Because the first AI was not steered, the checker's own catch rate through this channel is still unmeasured; that uncertainty is what AR-4-02 accepts |
| DR-3.1-02 (two calls per decision, no cap) | **Remediated** by D-14: the call cap counts every model call in a run, retries included | Commit `e755622`. `TestLoop_CallCap` (`evidence/01-test-report.txt:692`); `evidence/05-staging-ai-player.log:68` (`cause=ai-call-cap`); `evidence/08-cap-halt-badge-off.png`. Residual (the cap is off until set) accepted as AR-4-06 |
| DR-3.1-04 (the vault invents a key when none is set) | **Remediated** by D-25: the server refuses to start outside local development without a usable key | Commits `2954960`, `c5b58b3`. `TestLoadRequiresEncryptionKeyOutsideDevelopment` (`evidence/01-test-report.txt:39`). Residual (the check only runs on Railway) deferred as DR-4-04 |
| DR-3.1-05 (migration 012 cannot be rolled back once a block exists) | **Remediated** by D-26: the rollback converts blocked rows to failed first | Commit `d8ae8a3`. `TestMigration012DownReconcilesBlockedRows` (`evidence/01-test-report.txt:1893`) |
| Verified, nothing to accept or defer | Re-confirmed by the audit | T-4-SC (no new dependency; standing acceptances AR-1-01, AR-2-08, AR-3-13), T-4-09 (ownership check on every new endpoint), T-4-11 (the harness can fail), T-4-15 and T-4-18 plan-level accepted facets (the harness's own delete is stated in its preamble; the corpus cost is listed in the agenda's corrections) |

**Backlog, not risks (no register entry, not raised again at a future security review):** IN-01 (retention day constants are only printed; the SQL hard-codes the intervals), IN-02 (`line_count` not reset), IN-03 (limits are cut by bytes where the wording says characters), IN-04 (no `http.MaxBytesReader` before the goal length check; same class as the already-accepted AR-1-03), IN-05 (the retrying notice fires before the retry's reservation), IN-06 (a second-device sign-in moves the memory boundary), IN-08 (an armed delete confirmation survives a profile switch), IN-10 (the harness passes the cookie on curl's command line), the agenda's stale line-number citations (corrected in its own corrections section), and the two new owner-visible sentences from WR-03 and WR-09, which the owner saw in the Evidence Dossier.

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open-for-decision | Open (mitigation absent) | Run By |
|------------|----------------|--------|--------------------|-----------------------------|--------|
| 2026-09-17 | 35 threat ids (31 verified, 1 accepted at plan level, 1 deferred to the review, 2 open) | 33 | 14 register items listed for the owner (R-01 to R-14) | 2 (T-4-02 backstop, T-4-05 canned-report facet) | gsd-security-auditor |
| 2026-09-17 | 14 register items (R-01 to R-14) | 10 accepted (AR-4-01 to AR-4-10), 4 deferred (DR-4-01 to DR-4-04) | 0 | 0 without an owner disposition; the 2 absent mitigations remain absent — T-4-05 accepted (AR-4-01), T-4-02 deferred (DR-4-03) | Owner, on the Phase 4 Risk Register artifact |

After the owner's decisions every item has a disposition: 33 plan-time rows verified or closed by the audit, 2 rows not mitigated and carried by an owner decision (1 accepted, 1 deferred), nine earlier deferrals confirmed remediated, DR-3-05 closing with the push, DR-3-06 accepted, 10 risks accepted, 4 deferred to the Phase 5 review, none remediated now. `threats_open: 0` on that basis and no other.

---

## Sign-Off

- [x] All 35 register rows have a disposition; 33 are CLOSED with cited evidence and 2 are recorded as not mitigated, each carrying the owner's decision (T-4-05 accepted as AR-4-01; T-4-02 deferred as DR-4-03)
- [x] Code-review findings (CR-01, CR-02, WR-01 to WR-12) confirmed fixed in code by the audit; unfixed info findings triaged into register items or backlog
- [x] Accepted risks documented in the Accepted Risks Log (AR-4-01 to AR-4-10) with the owner's notes verbatim
- [x] Deferred risks documented and carried to the Phase 5 review (DR-4-01 to DR-4-04), with the owner's instructions recorded verbatim, including the added requirement on DR-4-03 that the limiter be a configurable setting
- [x] All eleven deferrals carried in from Phase 3 and Phase 3.1 have a recorded outcome
- [x] The owner's condition on AR-4-08 recorded as a pending todo
- [x] `threats_open: 0` confirmed, on the basis stated at the top of this record
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-09-17 — owner decisions recorded on the Phase 4 Risk Register (https://claude.ai/artifact/SvSdbXCa3YXmFDryiXpAaK); no Remediate Now decision, so Phase 4 closes. The push of `ai-player` follows this record.
