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
>
> **Evidence rule (owner-directed, binding).** Every success criterion and acceptance criterion is proven by one of exactly two artifact types: a **canned report** (a repeatable script or test run whose output is captured verbatim into a file) or a **screenshot** (an image of the browser showing the behaviour). A server log excerpt captured to a file counts as a canned report. **Running a database query is not evidence** and no row in this document may cite one. Every row's Evidence column names a file under `.planning/phases/01-profile-foundation-and-policy-gate/evidence/`.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go stdlib `testing` (only pre-existing test: `internal/icm/icm_test.go`); no frontend test tooling exists |
| **Config file** | none — `go test` needs none |
| **Quick run command** | `go test ./internal/store/... ./internal/policy/... ./internal/profiles/... -v` |
| **Full suite command** | `go test ./...` (same command CI runs in `.github/workflows/ci.yml`) |
| **Canned report harness** | `scripts/verify-phase1.sh` (plan 01-06) — drives the five endpoints and prints PASS/FAIL per ROADMAP criterion C1..C4 with response bodies |
| **Estimated runtime** | ~10 seconds for `go test`; ~15 seconds for the harness against staging |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/store/... ./internal/policy/... ./internal/profiles/... -v`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd:verify-work`:** Full suite green and captured to `evidence/01-test-report.txt`; `scripts/verify-phase1.sh` run against staging and captured to `evidence/03-canned-report.txt`; the staging startup and `[AI-PLAYER]` log excerpts captured to `evidence/02-staging-startup.log` and `evidence/04-staging-ai-player.log`; the seven end-user browser screenshots captured to `evidence/05-*.png` … `evidence/11-*.png`
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | Evidence File | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|---------------|--------|
| 01-01-01 | 01 | 1 | REQ-profile-ai-fields | T-1-10, T-1-11 | Startup logs the applied migration version and the AI-column fallback result, so the schema is provable from the log; no profile text is logged | source assertion + log excerpt | `grep -c 'Migrations completed successfully (version=%d, dirty=%v)' cmd/server/main.go && grep -c 'AI Player columns ensured' cmd/server/main.go` | `evidence/02-staging-startup.log` | ⬜ pending |
| 01-01-03 | 01 | 1 | REQ-profile-ai-fields | — | Blank model name, call cap, and disengage threshold resolve to Go-side defaults; non-blank values round-trip unchanged | unit | `go test ./internal/store/... -run TestResolveAISettings -v` | `evidence/01-test-report.txt` | ⬜ pending |
| 01-01-03 | 01 | 1 | REQ-policy-gate | T-1-05 | Engage gate returns refuse when no acceptance is recorded, allow when version and timestamp are set | unit | `go test ./internal/store/... -run TestEngageGate -v` | `evidence/01-test-report.txt` | ⬜ pending |
| 01-02-02 | 02 | 1 | REQ-policy-gate | T-1-02 | Embedded policy text serves with parsed version `1.0` | unit | `go test ./internal/policy/... -run TestParseVersion -v` | `evidence/01-test-report.txt` | ⬜ pending |
| 01-03-03 | 03 | 2 | REQ-profile-ai-fields | T-1-04 | `PUT` then `GET` of the AI sub-resource returns the stored values; validation rejects over-length text with a specific message | handler (`httptest`) | `go test ./internal/profiles/... -run TestAISettings -v` | `evidence/01-test-report.txt` | ⬜ pending |
| 01-03-03 | 03 | 2 | REQ-policy-gate | T-1-05 | Gate endpoint on an un-accepted profile returns the refusal message; on an accepted profile it passes | handler (`httptest`) | `go test ./internal/profiles/... -run TestEngageGateHandler -v` | `evidence/01-test-report.txt` | ⬜ pending |
| 01-03-03 | 03 | 2 | REQ-policy-gate | T-1-01, T-1-02 | A `PUT` to ai-settings carrying acceptance fields writes neither column; accept records the server-parsed version once | handler (`httptest`) | `go test ./internal/profiles/... -run "TestAISettingsCannotSetAcceptance\|TestPolicyAcceptUsesServerVersion" -v` | `evidence/01-test-report.txt` | ⬜ pending |
| 01-03-03 | 03 | 2 | REQ-policy-gate | T-1-11 | Accept, repeat-accept, both gate outcomes and a settings save each emit exactly one `[AI-PLAYER]` line, and the log carries no conduct rules, approach guidance or policy prose | handler (`httptest`, log capture) | `go test ./internal/profiles/... -run TestAIPlayerLogLines -v` | `evidence/01-test-report.txt` | ⬜ pending |
| 01-06-01 | 06 | 3 | REQ-profile-ai-fields, REQ-policy-gate | T-1-12 | The canned-report harness exists, is executable, and labels every check with the ROADMAP criterion C1..C4 it proves | shell lint + source assertion | `bash -n scripts/verify-phase1.sh && test -x scripts/verify-phase1.sh && grep -c 'C1\|C2\|C3\|C4' scripts/verify-phase1.sh` | `evidence/01-test-report.txt` | ⬜ pending |
| 01-06-02 | 06 | 3 | REQ-profile-ai-fields, REQ-policy-gate | T-1-13 | The harness passes on correct fixtures and FAILs (non-zero exit) on a wrong one, so a clean report carries information | fixture self-test | `bash scripts/verify-phase1.sh --self-test /tmp/phase1-selftest.txt && ! bash scripts/verify-phase1.sh --self-test-negative /tmp/phase1-negative.txt` | `evidence/01-test-report.txt` | ⬜ pending |
| 01-04-01..03 | 04 | 3 | REQ-profile-ai-fields, REQ-policy-gate | T-1-09, T-1-11 | The AI Player section type-checks and builds; no `dangerouslySetInnerHTML`, no console logging of profile text | build | `cd frontend && npm run build` | `evidence/01-test-report.txt` | ⬜ pending |
| 01-05-01 | 05 | 4 | REQ-profile-ai-fields, REQ-policy-gate | T-1-SC | Full suite, policy diff gate, harness self-test and frontend build captured verbatim with no dependency drift | canned report | `go test ./... -v` + `cd frontend && npm run build`, both captured | `evidence/01-test-report.txt` | ⬜ pending |
| 01-05-02 | 05 | 4 | REQ-profile-ai-fields | T-1-10 | Migration 010 is live on staging, proven by the startup log naming the applied version and the fallback result | log excerpt | `grep -c 'Migrations completed successfully (version=10' evidence/02-staging-startup.log && grep -c 'AI Player columns ensured' evidence/02-staging-startup.log` | `evidence/02-staging-startup.log` | ⬜ pending |
| 01-05-02 | 05 | 4 | REQ-profile-ai-fields, REQ-policy-gate | T-1-01, T-1-04, T-1-05, T-1-06 | All four ROADMAP criteria hold over HTTP against staging: blanks round-trip, over-length rejected, acceptance forgery refused, one-time acceptance, gate refuses then allows, timers save leaves AI fields intact | canned report | `BASE_URL=... SESSION_COOKIE=... CONNECTION_ID=... scripts/verify-phase1.sh evidence/03-canned-report.txt` | `evidence/03-canned-report.txt` | ⬜ pending |
| 01-05-02 | 05 | 4 | REQ-policy-gate | T-1-05, T-1-11 | The staging log shows one acceptance, one already-accepted echo with the same timestamp, both gate outcomes and a settings save — and no profile text | log excerpt | `grep -c '\[AI-PLAYER\] policy accepted' evidence/04-staging-ai-player.log && grep -c 'allowed=true' evidence/04-staging-ai-player.log` | `evidence/04-staging-ai-player.log` | ⬜ pending |
| 01-05-03 | 05 | 4 | REQ-policy-gate | T-1-05 | Fresh profile shows policy first with no editor; one Accept reveals the acceptance line and editor | player-observable (end-user screenshots) | n/a — image evidence | `evidence/05-policy-first.png`, `evidence/06-accepted-line.png` | ⬜ pending |
| 01-05-03 | 05 | 4 | REQ-profile-ai-fields | T-1-06 | Values persist across hard reload and a new session; cleared fields come back blank; an unrelated timers save leaves AI fields intact | player-observable (screenshots) | n/a — image evidence | `evidence/07-values-after-reload.png`, `evidence/08-values-new-session.png`, `evidence/09-blank-fields.png`, `evidence/10-after-timers-save.png` | ⬜ pending |
| 01-05-03 | 05 | 4 | REQ-policy-gate | — | Deleting and recreating the profile for the same game asks for the policy again | player-observable (screenshot) | n/a — image evidence | `evidence/11-recreated-profile-policy-again.png` | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/store/profile_test.go` — task 01-01-03 (plan 01, wave 1); blank-setting resolution and engage-gate decision — REQ-profile-ai-fields, REQ-policy-gate
- [ ] `internal/policy/policy_test.go` — task 01-02-02 (plan 02, wave 1); version parsing from the embedded policy markdown — REQ-policy-gate
- [ ] `internal/profiles/handler_test.go` — task 01-03-03 (plan 03, wave 2); `httptest` coverage of the AI sub-resource round-trip, validation, acceptance-forgery refusal, the gate endpoint, and the `[AI-PLAYER]` log-line contract — REQ-profile-ai-fields, REQ-policy-gate
- [ ] `scripts/verify-phase1.sh` plus `scripts/fixtures/phase1/` and `scripts/fixtures/phase1-negative/` — tasks 01-06-01 and 01-06-02 (plan 06, wave 3); the canned-report harness and its self-test — REQ-profile-ai-fields, REQ-policy-gate
- [ ] Framework install: none — `testing` is stdlib and already in use; the harness uses `curl` plus optional `jq` behind a `command -v` guard

---

## Manual-Only Verifications

Each row names the evidence file it produces. All files live under `.planning/phases/01-profile-foundation-and-policy-gate/evidence/`.

| Behavior | Requirement | Why Manual | Test Instructions | Evidence File |
|----------|-------------|------------|-------------------|---------------|
| Fresh profile shows the policy first with a single Accept button and no editor | REQ-policy-gate | No frontend test tooling in repo; the owner accepts an image for a visual claim | Open Settings → AI Player on a profile with no acceptance; confirm policy text, `Version 1.0`, single Accept button, zero settings inputs; screenshot the window including the nav | `evidence/05-policy-first.png` |
| One Accept press reveals "✓ Accepted v1.0 on <date>" and the editor without a reload | REQ-policy-gate | Visual state transition | Press Accept once; screenshot immediately, without reloading | `evidence/06-accepted-line.png` |
| Edited conduct rules, approach guidance and AI settings survive a hard reload and a new session | REQ-profile-ai-fields | Browser persistence is player-observable | Edit all five fields, save, hard-reload, screenshot; then log out and back in, screenshot again | `evidence/07-values-after-reload.png`, `evidence/08-values-new-session.png` |
| Cleared model name, call cap and threshold come back blank with hint text | REQ-profile-ai-fields | Visual; blank-vs-zero is only distinguishable on screen | Clear the three fields, save, reload, screenshot with hints legible | `evidence/09-blank-fields.png` |
| An unrelated timers save leaves the AI fields intact | REQ-profile-ai-fields | Cross-section integrity (T-1-06) | Save Timers for the same connection, reopen AI Player, screenshot | `evidence/10-after-timers-save.png` |
| Delete profile, recreate, policy asked again | REQ-policy-gate | Involves the existing profile delete UI | Delete the connection profile, create a new one for the same game, open AI Player, screenshot | `evidence/11-recreated-profile-policy-again.png` |
| Engage-gate refuses before acceptance and allows after | REQ-policy-gate | No end-user surface for the gate until Phase 2; proven by the canned report and the staging log, not a screenshot | Run `scripts/verify-phase1.sh` against staging; cite the `C3` lines and the `[AI-PLAYER] engage gate` log lines | `evidence/03-canned-report.txt`, `evidence/04-staging-ai-player.log` |
| Migration 010 applied on staging and the `[AI-PLAYER]` lines emitted | REQ-profile-ai-fields, REQ-policy-gate | Requires a deploy; captured from the platform's log viewer, not queried from the database | After deploy, capture the startup excerpt and the `[AI-PLAYER]` lines with the Railway MCP log tool, `railway logs --environment staging`, or the dashboard log pane | `evidence/02-staging-startup.log`, `evidence/04-staging-ai-player.log` |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] Every Per-Task and Manual-Only row cites an evidence file, and no row cites a database query
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
