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
| 02 Autopilot Switch | pending | pending | pending | 2026-09-15, decisions pending |

Active deferrals block project close. As of 2026-09-15 the two Phase 1 deferrals are on the Phase 2 register awaiting a decision.

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
| DR-1-01 | R-02 / WR-02 | medium | Invalid numeric input in the AI Player panel silently saves as blank ("no cap" / "default"). | Parse each numeric field before the request; block the save with an inline error when non-empty and not a finite number. | Owner | 2026-09-15 | Phase 2 review, as R-02 | pending |
| DR-1-02 | R-04 / OBS-01 | medium | Staging prints the one-time sign-in code in the deploy log; anyone with staging log access can sign in as any staging user. | Gate the log line behind an env flag that is off by default, or log a hash; confirm production has no equivalent; review Railway log access. | Owner | 2026-09-15 | Phase 2 review, as R-03 | pending |

## Phase 02 — Autopilot Switch (reviewed 2026-09-15, decisions pending)

Record: `.planning/phases/02-autopilot-switch/02-SECURITY.md`. Decisions are being made on the "Phase 2 Risk Register" artifact (https://claude.ai/artifact/TiX2brwsPbgWVABBwAUFEj); this section is written from them once recorded.

Items awaiting decision: R-01 auto-resume with no time bound (T-2-08, medium); R-02 and R-03, the Phase 1 deferrals above; R-04 to R-08, the code review's open warnings and notes (low); R-09 and R-10, plan-level acceptances to confirm (low); R-11 to R-13, observations from execution (low). The three critical code-review findings were remediated before the review (commit b5c8bf5) and are closed in the phase record, not decided here.
