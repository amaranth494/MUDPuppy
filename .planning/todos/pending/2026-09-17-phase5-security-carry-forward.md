---
created: 2026-09-17
source: Phase 4 security review (.planning/phases/04-continuous-play/04-SECURITY.md, Deferred Risks), Phase 4 Risk Register https://claude.ai/artifact/SvSdbXCa3YXmFDryiXpAaK
resolves_phase: 5
---

# Phase 5 carry-forward from the Phase 4 security review

Fold into Phase 5 planning. The four deferrals are recorded in `.planning/RISK-REGISTER.md` (DR-4-01 to DR-4-04) and must be raised again at the Phase 5 security review with the same three choices (Accept / Defer / Remediate Now). The project cannot close while any of them is active. Owner notes below are verbatim.

## Deferred risks the owner asked to fix in the next phase, per the recommended fix

1. **If the checker's answer arrives without its yes-or-no, the command is sent** (DR-4-02, register R-04, high; T-4-04 residual). Owner note: "Fix this in next Phase per the recommended fix". Recommended fix: treat a missing `blocked` field as a failed review, which already stops the command (`internal/gemini/client.go` `ReviewAnswer.Blocked` is a plain bool today, so an answer without the field decodes as not blocked and `internal/driver/driver.go` then sends); give the reviewer its own sentence saying the memory notes were written by the other model, not by itself. Add a test for each.
2. **The password-vault key check only runs on Railway** (DR-4-04, register R-14, low; T-4-06 residual, review note IN-07). Owner note: "Employ this fix next Phase with the recommended fix." Recommended fix: require the vault key everywhere unless a setting says this is local development (`internal/config/config.go` gates on `RAILWAY_ENVIRONMENT` alone today; `internal/crypto/crypto.go` still falls back to a random key on any other host).

## Deferred risk that is a feature request

3. **The last-resort speed limit on commands does not count the AI's commands** (DR-4-03, register R-13, high; T-4-02, OPEN in the audit and not mitigated). Owner note: "Yes, and the limiter needs to be a setting that can be configured.  This will have to be an added feature in subsequent Phases." Recommended fix: record every AI command with the ICM rate limiter before it is sent and refuse the send when the limiter says stop; add a flood test. The owner's added requirement: the limiter must be a setting that can be configured, built as an added feature in a subsequent phase. Note for whoever builds it, from the audit: `internal/icm/dispatcher.go` returns for a plain pass-through command before `RecordExecution`, and the dispatcher's circuit-breaker counter is never reset, so counting AI sends against it as it stands would eventually refuse every AI command in a long-running process. Until this is built, D-06 / T-4-02's "the ICM rate limit is the backstop" claim is not true for AI commands.

## Deferred risk addressed by later AI work

4. **The checker blocks ordinary fighting, and changes its mind** (DR-4-01, register R-01, high; agenda Part 2, staging walkthrough). Owner note: "This is going to be addressed by deeper AI analysis in later Phases." Recommended fix on the register: tell the checker plainly that fighting creatures and objects the game presents as targets is normal play and only attacking another player is off limits; add harmless fight examples to the corpus; rerun the corpus. Seen live as false blocks of `c chill touch golem` and `c static blast crystal`; these false blocks also feed the consecutive-block counter that switches autopilot off.

## At the Phase 5 security review

Re-present all four with what Phase 5 did or did not build, and record each outcome in an `Outcome` column on the Phase 04 Deferred table in `.planning/RISK-REGISTER.md`.
