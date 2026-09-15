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
| 02 Autopilot Switch | 11 | 2 | 0 | 2026-09-15 |

Active deferrals block project close. As of 2026-09-15 two deferrals are active, both to be raised at the Phase 3 security review: DR-2-01 (sign-in code in the staging log, marked must fix by the owner) and DR-2-02 (throwaway profiles on staging). The Phase 1 deferrals were both decided at the Phase 2 review: DR-1-01 accepted as AR-2-02, DR-1-02 deferred again as DR-2-01.

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
