---
created: 2026-09-16
source: Phase 3.1 security review (https://claude.ai/artifact/E8oVLdXwsPL6e5FABum3Fg) and code review (03.1-REVIEW.md)
resolves_phase: 4
---

# Phase 4 carry-forward from Phase 3.1

Fold into Phase 4 planning. The five deferrals are recorded in `.planning/RISK-REGISTER.md` (DR-3.1-01 to DR-3.1-05) and must be raised again at the Phase 4 security review; the owner's notes ask for three of them to be remediated in Phase 4. The backlog items are defects, not risks.

## Deferred risks the owner asked to remediate in Phase 4

1. **The checker trusts the first AI's explanation** (DR-3.1-01, reviewer reads the player model's reasoning outside the untrusted framing). Fix: wrap the reasoning in its own untrusted markers while keeping its current position ahead of the window; add a corpus item that attacks the reviewer through the reasoning channel; rerun the AFTER report on a fresh quota day before shipping. A first attempt (`9682f79`) was reverted (`87887ad`); its two corpus runs are `evidence/05e` and `05f`.
2. **The password vault invents a key when none is set** (DR-3.1-04, `internal/crypto/crypto.go` DefaultKeyStore). Fix: refuse to start when `ENCRYPTION_KEY_V1` is missing outside local development; document the variable as required; write the rotation procedure. Staging has a stable key set on 2026-09-16; V2/V3 and other environments untouched.
3. **The migration 012 down file cannot roll back once a blocked row exists** (DR-3.1-05). Fix: map blocked rows to failed, or stop with a clear message, before re-adding the constraint.

## Deferred risks to decide again at the Phase 4 review

4. **Two model calls per decision with no cap** (DR-3.1-02). Phase 4's call cap must count both calls (REQ-call-cap-and-error-disengage). Also: DR-3-04's single retry applies to both calls.
5. **Blocked decisions keep attack text forever** (DR-3.1-03). Owner note: "There needs to be a retention or cleanup policy enacted to fix this. Address in next Phase." Sits with DR-3-01's retention question.

## Phase 4 design inputs from this phase

- What a block means for the loop: this phase leaves the switch ON and idle after a block; Phase 4 decides skip / count toward the disengage threshold / back off.
- Alter Aeon dropped the socket about half a minute after every reconnect during the walkthroughs; the WAITING/resume path handled it, and each resume fires a fresh decision. A loop will feel this; investigate before building.
- Staging currently runs `gemini-3.5-flash-lite` (the `gemini-3.5-flash` daily free-tier quota was spent by the corpus runs on 2026-09-16). Decide which model staging should run on.

## Backlog (defects, not risks)

- WR-03: the Never-issue hint (locked copy in 03.1-UI-SPEC.md) and the system-instruction sentence describe the match more loosely than `matchNeverIssue` implements; align both with the test table.
- IN-02: `corpus_live_test.go` paces between items but not between an item's two calls; under rate pressure the reviewer call fails and the item reads as `failed-model`.
- `.claude/worktrees/agent-a084c785f3d4c84a8` could not be deleted (held open by a stale process); git no longer tracks it. Remove by hand.
