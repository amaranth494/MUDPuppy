---
title: Phase 1 code review warnings carried forward
area: profiles
created: 2026-09-15
source: .planning/phases/01-profile-foundation-and-policy-gate/01-REVIEW.md
phase_origin: 01
status: pending
---

# Phase 1 code review: warnings carried forward

Advisory findings from the Phase 1 code review, left open by owner decision on 2026-09-15 (only the Critical CR-01 was fixed before closing the phase). Address when a later phase touches the same code, or as a quick task.

- **WR-01** `internal/profiles/handler.go` `validateAISettings`: no server-side upper bound on `call_cap` or `disengage_threshold`. Add sane maximums so a typo cannot set an absurd cap.
- **WR-02** `frontend/src/components/AIPlayerPanel.tsx` (around lines 80-88): invalid numeric input for call cap or threshold becomes `null` via `JSON.stringify(NaN)` and silently saves as "no cap" / "default" instead of surfacing a validation error.
- **WR-03** request bodies are read without a size limit (pre-existing pattern, now exercised by two large free-text fields). Wrap with `http.MaxBytesReader` on the AI settings PUT at minimum.
- **Info** `migrations/010_add_ai_fields.*.sql` reuse `sql-migrate` directive comments that are inert under golang-migrate, matching migration 009. Cosmetic; align when migrations are next touched.
