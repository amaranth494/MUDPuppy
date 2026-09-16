# Risk Register — accepted and deferred security risks, by phase

This is the project-wide audit record of every security risk the owner has decided on at a phase security review. Each phase's own `NN-SECURITY.md` holds the full threat register and the audit trail for that phase; this file holds only the decisions, so an auditor can read the whole history in one place.

Rules (owner, 2026-09-15):

- **Accept Risk** records the acceptance, the phase it was accepted in, and the assessed criticality. Closed for good unless the owner reopens it. Accepted risks are not re-presented at later reviews.
- **Defer Until Next Review** temporarily accepts the risk so the phase can close. It is raised again at the next phase's review with the same three choices. The project cannot be considered closed while any deferral is active.
- **Remediate Now** reopens the phase with an emergency plan; when the remediation is proven and the phase re-approved, the entry here records the remediation and the closing security check.

Decisions are made on the phase's Risk Register artifact and read back from it; the phase security record is written from them, then this file is appended.

## Status summary

| Phase | Accepted | Deferred (active) | Remediated | Review date |
|-------|----------|-------------------|------------|-------------|
| 01 Profile Foundation and Policy Gate | 5 | 0 (2 raised again at the Phase 2 review) | 0 | 2026-09-15 |
| 02 Autopilot Switch | 11 | 0 (2 closed at the Phase 3 review) | 0 | 2026-09-15 |
| 03 One AI Decision | 14 | 7 | 0 | 2026-09-16 |

Active deferrals block project close. As of 2026-09-16 7 deferrals are active, all from Phase 3 (DR-3-01 to DR-3-07), to be raised at the Phase 4 security review; the two Phase 2 deferrals (DR-2-01, DR-2-02) were closed at the Phase 3 review. The Phase 1 deferrals were resolved at the Phase 2 review.

## Phase 01 — Profile Foundation and Policy Gate (reviewed 2026-09-15)

Record: `.planning/phases/01-profile-foundation-and-policy-gate/01-SECURITY.md`. Decisions made on the "Phase 1 Risk Register" artifact (https://claude.ai/artifact/728XpbNvbQSoPgQeEUyoP5).

### Accepted

| ID | Register item | Criticality | Risk (short) | Proposed remediation not taken | Accepted by | Date |
|----|---------------|-------------|--------------|--------------------------------|-------------|------|
| AR-1-01 | R-05 / T-1-SC | low | No package added to `go.mod` or `frontend/package.json` in Phase 1; the guard is the empty dependency-drift section of the test report. | None; keep the drift check in every phase's test report. | Owner | 2026-09-15 |
| AR-1-02 | R-01 / WR-01 | medium | No upper bound on `call_cap` or `disengage_threshold`; only a floor of 1, so an absurd cap is functionally no cap. | Add maximums in `validateAISettings`, tests, and a matching hint in the panel. | Owner | 2026-09-15 |
| AR-1-03 | R-03 / WR-03 | medium | Request bodies are buffered in full before the length checks run; no `http.MaxBytesReader` on JSON endpoints. | 1 MiB `MaxBytesReader` on JSON endpoints with a 413 response. | Owner | 2026-09-15 |
| AR-1-04 | R-06 / OBS-02 | low | Throwaway test user and two connection profiles left on staging by the Phase 1 evidence run. | Delete them through the app with the owner's session. | Owner | 2026-09-15 |
| AR-1-05 | R-07 / IN-01 | low | Migrations 009 and 010 carry inert `-- +migrate` directive comments from another tool. | Drop the lines the next time migrations are touched. | Owner | 2026-09-15 |

### Deferred

| ID | Register item | Criticality | Risk (short) | Proposed remediation | Deferred by | Date | Raised again at | Outcome |
|----|---------------|-------------|--------------|----------------------|-------------|------|-----------------|---------|
| DR-1-01 | R-02 / WR-02 | medium | Invalid numeric input in the AI Player panel silently saves as blank ("no cap" / "default"). | Parse each numeric field before the request; block the save with an inline error when non-empty and not a finite number. | Owner | 2026-09-15 | Phase 2 review, as R-02 | Accepted at the Phase 2 review (AR-2-02), 2026-09-15 |
| DR-1-02 | R-04 / OBS-01 | medium | Staging prints the one-time sign-in code in the deploy log; anyone with staging log access can sign in as any staging user. | Gate the log line behind an env flag that is off by default, or log a hash; confirm production has no equivalent; review Railway log access. | Owner | 2026-09-15 | Phase 2 review, as R-03 | Deferred again at the Phase 2 review (DR-2-01, must fix in Phase 3), 2026-09-15 |

## Phase 02 — Autopilot Switch (reviewed 2026-09-15)

Record: `.planning/phases/02-autopilot-switch/02-SECURITY.md`. Decisions made on the "Phase 2 Risk Register" artifact (https://claude.ai/artifact/TiX2brwsPbgWVABBwAUFEj). The three critical code-review findings were remediated before the review (commit b5c8bf5) and are closed in the phase record, not decided here.

### Accepted

| Risk ID | Register item | Criticality | Risk | Proposed remediation (not taken) | Accepted By | Date |
|---------|---------------|-------------|------|----------------------------------|-------------|------|
| AR-2-01 | R-01 / T-2-08 | medium | Waiting-to-on auto-resume has no time bound; resume now happens only onto the same profile (b5c8bf5). | A bounded waiting lifetime after which the switch lands on Off; WaitingSince is already stored. | Owner, Accept Risk on the Phase 2 Risk Register | 2026-09-15 |
| AR-2-02 | R-02 / DR-1-01 / WR-02 (Phase 1) | medium | Invalid numeric input in the AI Player panel silently saves as blank. Deferred from Phase 1, now accepted. | Parse each numeric field before the request; block the save with an inline error when non-empty and not a finite number. | Owner, Accept Risk on the Phase 2 Risk Register | 2026-09-15 |
| AR-2-03 | R-04 / WR-01 | low | The wheel-grab reads the switch and disengages it under two separate locks; outcome correct, log cause can be misleading. | One Manager method that reads and disengages under a single lock. | Owner, Accept Risk on the Phase 2 Risk Register | 2026-09-15 |
| AR-2-04 | R-05 / WR-02 | low | The browser types the websocket source field as any string; the server treats unknown labels as human. | Type the field as the CommandSource union. | Owner, Accept Risk on the Phase 2 Risk Register | 2026-09-15 |
| AR-2-05 | R-06 / WR-03 | low | Typed directives relabel their output as automation per line; safe today because no CLI directive emits a game command. | A regression test, or scope the relabelling to directives known to emit nothing. | Owner, Accept Risk on the Phase 2 Risk Register | 2026-09-15 |
| AR-2-06 | R-07 / IN-01 | low | An unrecognised answer to #AUTO prints nothing. | Print a generic line and log the outcome in the default branch. | Owner, Accept Risk on the Phase 2 Risk Register | 2026-09-15 |
| AR-2-07 | R-08 / IN-02 | low | A user-initiated disconnect refreshes the status twice. | A comment marking the double call deliberate, or collapse it with ordering preserved. | Owner, Accept Risk on the Phase 2 Risk Register | 2026-09-15 |
| AR-2-08 | R-09 / T-2-SC | low | No package was added in Phase 2; the dependency-drift section of the test report is empty. | None; keep the drift check in every phase. | Owner, Accept Risk on the Phase 2 Risk Register | 2026-09-15 |
| AR-2-09 | R-10 / T-2-03 residual (plan 02-04) | low | The autopilot websocket push carries a state and a cause to the owner's own connection only. | None; re-confirm if the message gains fields. | Owner, Accept Risk on the Phase 2 Risk Register | 2026-09-15 |
| AR-2-10 | R-11 / observed | low | For refused-no-session and already-off the log line names the last engaged profile, not the requesting one. | Log the requested connection id for refusals and no-ops. | Owner, Accept Risk on the Phase 2 Risk Register | 2026-09-15 |
| AR-2-11 | R-13 / observed | low | An executor installed MinGW-w64 GCC on the build machine (user scope, winget) without asking so go test -race could run; no project dependency changed. | Keep and document as required tooling, or uninstall and drop -race. | Owner, Accept Risk on the Phase 2 Risk Register | 2026-09-15 |

### Deferred

| Risk ID | Register item | Criticality | Risk | Proposed remediation | Owner note | Deferred By | Date | Raised again at |
|---------|---------------|-------------|------|----------------------|------------|-------------|------|-----------------|
| DR-2-01 | R-03 / DR-1-02 / OBS-01 (Phase 1) | medium | Staging prints the one-time sign-in code in the deploy log; anyone with staging log access can sign in as any staging user. Deferred at Phase 1, deferred again here. | Gate the log line behind an env flag off by default, or log a hash; confirm production has no equivalent; review Railway log access. | Owner note: "Mark this as MUST FIX in next Phase." | Owner | 2026-09-15 | Phase 3 security review (must fix) |
| DR-2-02 | R-12 / observed | low | Six throwaway connection profiles remain on staging (Phase 2 Harness, Harness B, Walkthrough, Refusal, Hand Play; Phase 1 Evidence Walkthrough). | Delete them from the Connections list, or keep as Phase 3 fixtures. | Owner note: "Clean this up in Phase 3." | Owner | 2026-09-15 | Phase 3 security review |

## Phase 03 — One AI Decision (reviewed 2026-09-16)

Record: `.planning/phases/03-one-ai-decision/03-SECURITY.md`. Decisions made on the "Phase 3 Risk Register" artifact (https://claude.ai/artifact/HauUDyWUkqutt7LuXWse1m). The two critical and two warning code-review findings were remediated before the review (commits cbf7d97, 0b74ea7, 48165ab, 2a2f349, redeployed as dd8d7cab) and are not register items. DR-2-01 and DR-2-02 were closed at this review (plans 03-07 and 03-13); their residuals are register items R-16 (accepted) and R-18 (deferred).

### Accepted

| Risk ID | Register item | Criticality | Risk | Proposed remediation (not taken) | Owner note | Accepted by | Date |
|---------|---------------|-------------|------|----------------------------------|------------|-------------|------|
| AR-3-01 | R-02 / Item 2 / T-3-14, AR-2-01 | medium | A resume from Waiting now acts, and it has no time bound | A bounded Waiting lifetime after which the switch lands on Off instead of resuming. | I've already accepted this risk in previous Phases. If the policy is the reason this keeps coming up, then adjust the policy. | Owner | 2026-09-16 |
| AR-3-02 | R-03 / Item 3 / T-3-04 | low | The log page trusts the session cookie alone | A share link with its own expiry, or a confirmation step before the log page opens. | — | Owner | 2026-09-16 |
| AR-3-03 | R-04 / Item 4 / T-3-02, T-3-19 | medium | The Gemini key on staging, the free tier, and no call cap until Phase 4 | Nothing beyond Phase 4's call cap, or an early stopgap cap at the driver. | Having an option and we can set it whenever we want. This is not a Risk. | Owner | 2026-09-16 |
| AR-3-04 | R-05 / Item 5 / T-3-06, T-2-01 | medium | The driver sends to the game without passing the human wheel-grab | A browser-side control that can interrupt an in-flight driver send, or an explicit acceptance that the ICM safety checker is the mediation for this path. | This is by design. Not a risk. | Owner | 2026-09-16 |
| AR-3-05 | R-07 / Item 7 | low | A server-only plan promised a browser behaviour and the gap was only caught live | A plan-checker rule that flags a server-only plan whose acceptance criterion describes something visible in the browser. | — | Owner | 2026-09-16 |
| AR-3-06 | R-09 / Item 9 / D-12 | low | "Survives a refresh" means reconnect the same profile, then it comes back | State the reconnect in the requirement, or keep the panel mounted for the last-selected profile across a refresh. | — | Owner | 2026-09-16 |
| AR-3-07 | R-11 / Item 11 / T-3-07 | low | The new model-error log line carries a kind and an HTTP status | None if the field set stands; otherwise redact further. | — | Owner | 2026-09-16 |
| AR-3-08 | R-12 / Item 12 | medium | A key was set on production and production redeployed | Remove GOOGLE_API_KEY from production and note the incident in the deploy runbook. | We're going to need it in Prod. Leave it there. | Owner | 2026-09-16 |
| AR-3-09 | R-14 / Item 14 | medium | Two staging session tokens were pasted into the chat | The owner signs out or rotates the current session at their convenience; future harness runs by the owner or through a short-lived token. | — | Owner | 2026-09-16 |
| AR-3-10 | R-15 / Item 15 | low | The harness self-test could not see the live-shape bugs it shipped with | A recorded-live fixture in the self-test suite. | — | Owner | 2026-09-16 |
| AR-3-11 | R-16 / Item 16 / DR-2-01 residual | low | Two dev-mode lines still print the sign-in code when no mail server is configured | Route the two lines through the same gated helper. | — | Owner | 2026-09-16 |
| AR-3-12 | R-17 / Item 17 / AR-2-11 | low | The race detector depends on one machine's compiler | A standing CI step that runs -race with its own compiler. | — | Owner | 2026-09-16 |
| AR-3-13 | R-19 / T-3-SC | low | Supply chain: no new packages this phase | Nothing to remediate. Keep the drift check in every phase's test report. | — | Owner | 2026-09-16 |
| AR-3-14 | R-21 / IN-02 | low | The harness's no-jq fallback can match the wrong JSON field | Require jq for live mode, or parse with a small embedded script. | — | Owner | 2026-09-16 |

### Deferred

| Risk ID | Register item | Criticality | Risk | Proposed remediation | Owner note | Deferred by | Date | Raised again at | Outcome |
|---------|---------------|-------------|------|----------------------|------------|-------------|------|-----------------|---------|
| DR-3-01 | R-01 / Item 1 / T-3-09, T-3-15 | medium | Every session transcript and every decision window is kept forever | A retention window after which transcript lines and decision windows are deleted, with the length left to the owner, plus a per-profile delete path. | Make a retention policy standard in next Phase | Owner | 2026-09-16 | Phase 4 security review | |
| DR-3-02 | R-06 / Item 6 / T-3-01 | high | Prompt injection through game text steers which command the AI issues | A content-level mitigation: a second model pass reviewing the chosen command against the room text, or a denylist of high-risk verbs, or both. | I don't want to hold up this Phase with this, but I want to dig into this as a emergency security task prior to the next Phase. | Owner | 2026-09-16 | Phase 4 security review | |
| DR-3-03 | R-08 / Item 8 / D-13 | medium | The badge reads On for a few seconds after an AI-failure disengage | Have the ai message that carries a failure notice also carry the new switch state, so the badge updates in the same message. | Proposed remediation: have the ai message that carries a failure notice also carry the new switch state, so the badge updates in the same message. Do this in next Phase | Owner | 2026-09-16 | Phase 4 security review | |
| DR-3-04 | R-10 / Item 10 / D-03, D-13 | medium | A transient vendor 503 counts as a failed decision and disengages autopilot | One automatic retry on a 503 specifically, distinct from the other failure kinds, with a user-visible notice while it retries; the model name kept as an owner-set environment variable. | A single retry is fine, but it needs to inform the user while establishing a connection. | Owner | 2026-09-16 | Phase 4 security review | |
| DR-3-05 | R-13 / Item 13 | medium | A dashboard Deploy rebuilds staging from a branch that was never pushed | Push ai-player to GitHub so dashboard deploys are safe, or document that only railway up may deploy this branch. | Please ensure that as Phases close that we're pushing to github appropriately. | Owner | 2026-09-16 | Phase 4 security review | |
| DR-3-06 | R-18 / DR-2-01 carry-forward / T-3-34 | medium | Who can read the staging log has not been reviewed | Review and, if needed, restrict who can run railway logs against staging. | — | Owner | 2026-09-16 | Phase 4 security review | |
| DR-3-07 | R-20 / IN-01 | low | A dead statement in EngageAutopilot | Delete the statement. | Fix this in next Phase. | Owner | 2026-09-16 | Phase 4 security review | |
