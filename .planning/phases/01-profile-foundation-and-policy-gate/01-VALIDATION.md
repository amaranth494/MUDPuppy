---
phase: 1
slug: profile-foundation-and-policy-gate
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-09-15
---

# Phase 1 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Derived from `01-RESEARCH.md` §Validation Architecture. The planner fills the Per-Task map with real task IDs; the executor flips statuses.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go stdlib `testing` (only existing test: `internal/icm/icm_test.go`); no frontend test tooling exists |
| **Config file** | none — `go test` needs none |
| **Quick run command** | `go test ./internal/store/... ./internal/policy/... ./internal/profiles/... -v` |
| **Full suite command** | `go test ./...` (same command CI runs in `.github/workflows/ci.yml`) |
| **Estimated runtime** | ~10 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/store/... ./internal/policy/... ./internal/profiles/... -v`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd:verify-work`:** Full suite green, plus the staging database inspection and the browser walkthrough from the ROADMAP Phase Validation line
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 01-01-03 | 01 | 1 | REQ-profile-ai-fields | — | Blank model name, call cap, and disengage threshold resolve to Go-side defaults; non-blank values round-trip unchanged | unit | `go test ./internal/store/... -run TestResolveAISettings -v` | ❌ W0 (created by 01-01-03) | ⬜ pending |
| 01-05-02 | 05 | 4 | REQ-profile-ai-fields | T-1-10 | New columns present on a migrated `profiles` row with correct types and defaults | diagnostic (DB) | `SELECT conduct_rules, approach_guidance, ai_settings, policy_version_accepted, policy_accepted_at FROM profiles LIMIT 1;` | n/a | ⬜ pending |
| 01-03-03 | 03 | 2 | REQ-profile-ai-fields | T-1-04 | `PUT` then `GET` of the AI sub-resource returns the stored values; validation rejects over-length text with a specific message | handler (`httptest`) | `go test ./internal/profiles/... -run TestAISettings -v` | ❌ W0 (created by 01-03-03) | ⬜ pending |
| 01-01-03 | 01 | 1 | REQ-policy-gate | T-1-05 | Engage gate returns refuse when no acceptance is recorded, allow when version and timestamp are set | unit | `go test ./internal/store/... -run TestEngageGate -v` | ❌ W0 (created by 01-01-03) | ⬜ pending |
| 01-02-02 | 02 | 1 | REQ-policy-gate | T-1-02 | Embedded policy text serves with parsed version `1.0` | unit | `go test ./internal/policy/... -run TestParseVersion -v` | ❌ W0 (created by 01-02-02) | ⬜ pending |
| 01-03-03 | 03 | 2 | REQ-policy-gate | T-1-05 | Gate endpoint on an un-accepted profile returns the refusal message; on an accepted profile it passes | handler (`httptest`) | `go test ./internal/profiles/... -run TestEngageGateHandler -v` | ❌ W0 (created by 01-03-03) | ⬜ pending |
| 01-03-03 | 03 | 2 | REQ-policy-gate | T-1-01, T-1-02 | A `PUT` to ai-settings carrying acceptance fields writes neither column; accept records the server-parsed version once | handler (`httptest`) | `go test ./internal/profiles/... -run "TestAISettingsCannotSetAcceptance\|TestPolicyAcceptUsesServerVersion" -v` | ❌ W0 (created by 01-03-03) | ⬜ pending |
| 01-05-03 | 05 | 4 | REQ-profile-ai-fields, REQ-policy-gate | T-1-05, T-1-06 | Fresh profile shows policy first, accept reveals the editor, values persist across reload and session, an unrelated timers save leaves AI fields intact, delete and recreate re-asks | player-observable (browser) | manual walkthrough per the ROADMAP Phase Validation line | n/a | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/store/profile_test.go` — task 01-01-03 (plan 01, wave 1); blank-setting resolution and engage-gate decision — REQ-profile-ai-fields, REQ-policy-gate
- [ ] `internal/policy/policy_test.go` — task 01-02-02 (plan 02, wave 1); version parsing from the embedded policy markdown — REQ-policy-gate
- [ ] `internal/profiles/handler_test.go` — task 01-03-03 (plan 03, wave 2); `httptest` coverage of the AI sub-resource round-trip, validation, acceptance-forgery refusal and the gate endpoint — REQ-profile-ai-fields, REQ-policy-gate
- [ ] Framework install: none — `testing` is stdlib and already in use

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Fresh profile shows the policy first; Accept reveals the editor; "Accepted v1.0 on <date>" shown | REQ-policy-gate | No frontend test tooling in repo; ROADMAP Phase Validation specifies a browser walkthrough | Open Settings → AI Player on a profile with no acceptance; confirm policy text, version, single Accept button; press Accept; confirm editor appears |
| Edited conduct rules, approach guidance, and AI settings survive reload and a new session | REQ-profile-ai-fields | Browser persistence is player-observable | Edit all fields, save, hard-reload, log out and in, confirm values |
| Delete profile, recreate, policy asked again | REQ-policy-gate | Involves the existing profile delete UI | Delete the connection profile, create a new one for the same game, open AI Player, confirm policy is presented |
| Migrated `profiles` row shows new fields and acceptance columns on staging | REQ-profile-ai-fields, REQ-policy-gate | Railway staging DB inspection | Run the SELECT above against the staging database after deploy |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
