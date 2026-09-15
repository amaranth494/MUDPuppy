---
title: Deferred security risks to re-raise at the Phase 2 security review
area: security
created: 2026-09-15
source: .planning/phases/01-profile-foundation-and-policy-gate/01-SECURITY.md
phase_origin: 01
raise_at: phase-02-security-review
status: pending
---

# Deferred security risks (carry-forward)

Owner decision on 2026-09-15 at the Phase 1 security review: these two risks are temporarily accepted so Phase 1 can close, and MUST be raised again at the Phase 2 security review with the same three choices (Accept / Defer / Remediate Now). The project cannot be considered closed while either remains deferred.

| ID | Criticality | Risk | Proposed remediation |
|----|-------------|------|----------------------|
| R-02 (WR-02) | Medium | Invalid numeric input in the AI Player panel silently saves as blank ("no cap" / "default") instead of a validation error. `frontend/src/components/AIPlayerPanel.tsx` handleSave. | Parse each numeric field before building the request; if non-empty and not a finite number, block the save with an inline error naming the field. |
| R-04 (OBS-01) | Medium | Staging prints the one-time login code in the deploy log (`STAGING: OTP sent to user, code: NNNNNN`); anyone with staging log access can sign in as any staging user. Pre-existing, staging only. | Gate the code-logging line behind an explicit env flag that is off by default, or log a hash; confirm production has no equivalent; review Railway log access on the staging project. |

Accepted at this review (closed, recorded in 01-SECURITY.md): R-01 WR-01 upper bounds, R-03 WR-03 body size limit, R-05 supply chain, R-06 staging test account, R-07 migration directive comments.
