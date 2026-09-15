---
title: Deferred security risks to re-raise at the Phase 3 security review (one marked MUST FIX)
area: security
created: 2026-09-15
source: .planning/phases/02-autopilot-switch/02-SECURITY.md
phase_origin: 02
raise_at: phase-03-security-review
status: pending
---

# Deferred security risks (carry-forward to Phase 3)

Owner decisions on 2026-09-15 at the Phase 2 security review (https://claude.ai/artifact/TiX2brwsPbgWVABBwAUFEj). The project cannot be considered closed while either remains deferred. Both must be raised at the Phase 3 security review with the same three choices (Accept / Defer / Remediate Now).

| ID | Criticality | Risk | Proposed remediation | Owner note |
|----|-------------|------|----------------------|------------|
| DR-2-01 (was DR-1-02 / R-04) | Medium | Staging prints the one-time sign-in code in the deploy log; anyone with staging log access can sign in as any staging user. Pre-existing, staging only. Deferred at Phase 1 and again at Phase 2. | Gate the code-logging line behind an env flag that is off by default, or log a hash; confirm production has no equivalent; review Railway log access on the staging project. | "Mark this as MUST FIX in next Phase." Plan the remediation into Phase 3, not merely re-present it. |
| DR-2-02 | Low | Six throwaway connection profiles remain on the owner's staging account (Phase 2 Harness, Harness B, Walkthrough, Refusal, Hand Play; Phase 1 Evidence Walkthrough). | Delete them from the Connections list, or keep as Phase 3 fixtures and delete after. | "Clean this up in Phase 3." |

Accepted at this review (closed, recorded in 02-SECURITY.md and .planning/RISK-REGISTER.md): AR-2-01 to AR-2-11.
