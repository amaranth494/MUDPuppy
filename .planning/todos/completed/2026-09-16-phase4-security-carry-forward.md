---
title: Phase 4 carry-forward from the Phase 3 security review
created: 2026-09-16
source: .planning/phases/03-one-ai-decision/03-SECURITY.md (Deferred Risks), Phase 3 Risk Register https://claude.ai/artifact/HauUDyWUkqutt7LuXWse1m
resolves_phase: 4
status: pending
---

# What Phase 4 must carry, in the owner's words

Seven risks were deferred at the Phase 3 security review (2026-09-16). Each must be raised again at the Phase 4 security review with the same three choices, and the owner's notes below are instructions for Phase 4's planning, not suggestions. The project cannot close while any of them stays deferred.

## Before Phase 4 starts: an emergency security task (owned by Phase 3.1, inserted 2026-09-16)

| ID | Risk | Owner note (verbatim) | What it asks for |
|----|------|-----------------------|------------------|
| DR-3-02 (R-06, T-3-01, **high**) | Prompt injection through game text steers which command the AI issues; shape validation bounds the command's form, not its content. | "I don't want to hold up this Phase with this, but I want to dig into this as a emergency security task prior to the next Phase." | A dedicated investigation and, if warranted, a mitigation plan (second model pass reviewing the chosen command against the room text, a denylist of high-risk verbs, or both) run and reviewed with the owner **before** Phase 4 planning is approved. Owned by the inserted Phase 3.1 (Prompt Injection Review), whose security review re-decides DR-3-02; Phase 4 only confirms the outcome. |

## Build into Phase 4

| ID | Risk | Owner note (verbatim) | What Phase 4 builds |
|----|------|-----------------------|---------------------|
| DR-3-01 (R-01, T-3-09/T-3-15, medium) | Every session transcript and every decision window is kept forever; no retention limit, deletion path or export. | "Make a retention policy standard in next Phase" | A retention policy as a standard: a retention window for transcript lines and decision windows (length chosen by the owner at discuss-phase), a scheduled delete, and a per-profile delete path, with evidence. |
| DR-3-03 (R-08, D-13, medium) | The sidebar badge reads On for a few seconds after an AI-failure disengage until the status poll catches up. | "Proposed remediation: have the ai message that carries a failure notice also carry the new switch state, so the badge updates in the same message. Do this in next Phase" | The `ai` websocket message carries the switch state alongside a failure notice; the badge updates from it; a screenshot proves no lag. |
| DR-3-04 (R-10, D-03/D-13, medium) | A transient vendor 503 counts as a failed decision and disengages autopilot; no retry by design. | "A single retry is fine, but it needs to inform the user while establishing a connection." | Exactly one automatic retry on a vendor 503 (distinct from the other failure kinds), with a visible notice in the panel and terminal while the retry is in flight; a second 503 lands as today's failure. The model name stays an owner-set environment variable. |
| DR-3-07 (R-20, IN-01, low) | A dead statement in `EngageAutopilot` (`internal/session/manager.go`). | "Fix this in next Phase." | Delete the statement in Phase 4's first plan that touches the session manager. |

## Process and ownership items to raise again

| ID | Risk | Owner note (verbatim) | What to do |
|----|------|-----------------------|------------|
| DR-3-05 (R-13, medium) | The Railway dashboard's Deploy rebuilds staging from GitHub, where the branch had never been pushed; such a build crashed staging. | "Please ensure that as Phases close that we're pushing to github appropriately." | `ai-player` was pushed at the Phase 3 close and CLAUDE.md now carries the rule: push at every phase close. Raise again at the Phase 4 review to confirm the rule held. |
| DR-3-06 (R-18, DR-2-01 carry-forward / T-3-34, medium) | Who can read the staging log (`railway logs` on project `mudpuppy`) has never been reviewed; user ids, connection ids and decision stages appear there. | — | An owner admin task: review project members and tokens on Railway; record the outcome at the Phase 4 review. |

## Accepted with an instruction attached

| ID | Risk | Owner note (verbatim) | What it asks for |
|----|------|-----------------------|------------------|
| AR-3-01 (R-02, T-3-14, medium) | A resume from Waiting now acts, with no time bound; policy section 2 asks the owner not to re-engage after a disconnect until they understand why. | "I've already accepted this risk in previous Phases. If the policy is the reason this keeps coming up, then adjust the policy." | Adjust `.specify/specs/safety-and-abuse-policy-v1.md` section 2 so the auto-resume behaviour is stated as intended rather than contradicted; a policy text change with a version bump, presented to the owner in Phase 4's discuss step. Do not raise the resume itself again. |
| AR-3-08 (R-12, medium) | `GOOGLE_API_KEY` was set on production and production redeployed; the project reads `AI_MODEL_GEMINI_KEY`. | "We're going to need it in Prod. Leave it there." | Leave it. When production is configured for the AI (final acceptance), the five `AI_MODEL_*` variables are what the server reads; `GOOGLE_API_KEY` alone does nothing. Note for the production cut-over checklist. |
