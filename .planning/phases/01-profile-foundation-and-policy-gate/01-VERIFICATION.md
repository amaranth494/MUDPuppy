---
phase: 01-profile-foundation-and-policy-gate
verified: 2026-09-15T00:00:00Z
status: passed
score: 4/4 must-haves verified
overrides_applied: 0
---

# Phase 1: Profile Foundation and Policy Gate Verification Report

**Phase Goal:** The connection profile is the single per-game home for the AI, and no profile can have the AI configured or engaged until its owner has accepted the Safety and Abuse policy on it, once.
**Verified:** 2026-09-15
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (ROADMAP Success Criteria 1-4)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | A profile stores and returns conduct rules, approach guidance, and AI settings (model name, call cap, disengage threshold); blank model name = server default, blank call cap = no cap, blank threshold = engine's built-in error handling (informative error, disengage, no crash); no reconnect field | ✓ VERIFIED | Code: `internal/store/profile.go` `AISettings` struct (3 fields, no reconnect), `ResolveAISettings` (lines 548-573), `DefaultDisengageThreshold = 3` (line 527). Live test run: `go test ./internal/store/... -run TestResolveAISettings -v` PASS (verified independently, matches `evidence/01-test-report.txt` lines 293-302, 332-342). Staging proof: `evidence/03-canned-report.txt` lines 28-31 (blank round-trip, `call_cap`/`disengage_threshold` come back `null` not `0`) and lines 36-37 (over-length conduct rules → HTTP 400 with exact message). Screenshot `evidence/09-blank-fields.png` visually confirmed: Model Name/Call Cap/Disengage Threshold show placeholder text (blank), Conduct Rules/Approach Guidance retain their separately-set values, matching the test's scope exactly. |
| 2 | Owner can read/edit conduct rules, approach guidance, AI settings from the browser; values survive page reload and a new session | ✓ VERIFIED | Code: `frontend/src/components/AIPlayerPanel.tsx` (274 lines) wired into `SettingsPage.tsx` line 1691-1692 via `activeSection === 'ai-player'`. Screenshots visually confirmed: `evidence/06-accepted-line.png` (editor appears post-accept), `evidence/07-values-after-reload.png`/`evidence/08-values-new-session.png` (cited, consistent naming with the verified set), `evidence/10-after-timers-save.png` (visually confirmed: AI fields identical after an unrelated Timers save — "No player killing..." / "Level cautiously..." persist). Staging proof: `evidence/03-canned-report.txt` lines 71-75 (round-trip) and 84-88 (post-timers unchanged), `C2: PASS` at line 94. |
| 3 | First time owner opens AI Player config, policy is presented and must be accepted before AI settings can be edited; server-side engage gate refuses any profile without recorded acceptance, with a clear message | ✓ VERIFIED | Code: `internal/profiles/handler.go` `GetEngageGate` (line 670) calls `store.EngageGateAllowed` and returns `store.EngageGateRefusalMessage` verbatim when refused. Screenshots visually confirmed: `evidence/05-policy-first.png` (policy text, Version 1.0, single "ACCEPT POLICY" button, zero settings fields rendered) and `evidence/06-accepted-line.png` ("✓ Accepted v1.0 on 2026-09-15" plus full editor, no reload). Staging proof: `evidence/03-canned-report.txt` line 14 (accepted=false pre-accept), lines 20-21 (gate refuses with exact message), line 63-64 (gate allows post-accept, no message). Log proof: `evidence/04-staging-ai-player.log` line 1 (`allowed=false`) and line 6 (`allowed=true`), same `connection_id`. |
| 4 | Acceptance recorded once per profile with timestamp and policy version (1.0); never expires; later policy change does not require re-acceptance; deleting the profile discards it | ✓ VERIFIED | Code: `internal/store/profile.go` `AcceptPolicy` (line 501) — guarded `UPDATE ... WHERE policy_accepted_at IS NULL`, no version-comparison logic anywhere in `internal/policy` package (confirmed: package contains no `expire`/`compare` reference). Staging proof: `evidence/03-canned-report.txt` lines 44-45 (forged acceptance via ai-settings PUT rejected), 50-52 (first accept), 57-58 (repeat accept echoes identical `accepted_at`). Log proof: `evidence/04-staging-ai-player.log` line 4 (`policy accepted ... version=1.0 accepted_at=2026-09-15T16:31:38.095576Z`) and line 5 (`policy already accepted` with the identical timestamp). Screenshot `evidence/11-recreated-profile-policy-again.png` visually confirmed: recreated connection for the same game shows the policy-first state again (deletion discards acceptance). |

**Score:** 4/4 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `migrations/010_add_ai_fields.up.sql` / `.down.sql` | Five AI columns, reversible | ✓ VERIFIED | Confirmed on disk: exactly 5 `ADD COLUMN IF NOT EXISTS` / `DROP COLUMN IF EXISTS`, correct types/defaults, nullable acceptance columns with no DEFAULT |
| `cmd/server/main.go` | Version-logging migrate bootstrap + AI column fallback | ✓ VERIFIED | `m.Version()` read, `Migrations completed successfully (version=%d, dirty=%v)`, `AI Player columns ensured` all present and non-fatal on error |
| `internal/store/profile.go` | AISettings, Profile/ProfileUpdate fields, AcceptPolicy, ResolveAISettings, EngageGateAllowed | ✓ VERIFIED | All exports present, `ProfileUpdate` deliberately excludes acceptance fields (T-1-01 mitigation confirmed by source read) |
| `internal/store/profile_test.go` | Table tests for blank resolution and gate | ✓ VERIFIED / WIRED | `go test ./internal/store/... -v` run live: `TestResolveAISettings`, `TestEngageGateAllowed`, `TestEngageGateRefusalMessageIsPhase2Contract` all PASS |
| `internal/policy/policy.go` + embedded markdown | Text()/Version() served from binary, version parsed from header | ✓ VERIFIED | `go:embed` directive present; `diff -u` against `.specify/specs/safety-and-abuse-policy-v1.md` run live, empty output (byte-identical) |
| `internal/policy/policy_test.go` | Version-parsing and text-integrity tests | ✓ VERIFIED / WIRED | `go test ./internal/policy/... -v` run live: `TestParseVersion`, `TestVersionOfEmbeddedPolicy`, `TestTextOfEmbeddedPolicy` all PASS |
| `internal/profiles/handler.go` | 5 endpoints + 4 `[AI-PLAYER]` log lines | ✓ VERIFIED / WIRED | `GetAISettings`, `PutAISettings`, `GetPolicy`, `AcceptPolicy`, `GetEngageGate` all present, each calls `getProfileByConnectionID` (T-1-03 mitigation), each writes its `[AI-PLAYER]` log line |
| `internal/profiles/handler_test.go` | httptest coverage | ✓ VERIFIED / WIRED | `go test ./internal/profiles/... -v` run live: 8 subtests including `TestEngageGateHandlerRefusesWithoutAcceptance`/`Allows...`, `TestAIPlayerLogLinesAreEmitted`, all PASS |
| `cmd/server/main.go` routes | 4 new route paths registered | ✓ VERIFIED | `ai-settings`, `policy`, `policy/accept`, `engage-gate` all registered under `/api/v1/profiles/{connection_id}/...` |
| `frontend/src/types/index.ts`, `services/api.ts` | Types + 5 API functions | ✓ VERIFIED / WIRED | `AISettings`, `AISettingsResponse`, `PolicyResponse`, `EngageGateResponse` interfaces present; `getAISettings`/`putAISettings`/`getPolicy`/`acceptPolicy`/`getEngageGate` present, each uses `credentials: 'include'` |
| `frontend/src/components/AIPlayerPanel.tsx` | Two-state panel (gate, then editor) | ✓ VERIFIED / WIRED | 274 lines; imported and rendered in `SettingsPage.tsx`; confirmed against 4 screenshots showing both states correctly |
| `scripts/verify-phase1.sh` + fixtures | Canned-report harness, self-test proving it can fail | ✓ VERIFIED | 510 lines, executable; `scripts/fixtures/phase1/` (12 files) and `scripts/fixtures/phase1-negative/` (15 files) both present on disk |
| `evidence/` directory (11 files) | Canned reports, log excerpts, screenshots | ✓ VERIFIED | All 11 files present on disk; every cited line number in `01-05-SUMMARY.md`'s Phase Validation table resolves to the claimed content (checked directly against `03-canned-report.txt`, `04-staging-ai-player.log`, `02-staging-startup.log`, `01-test-report.txt`); 4 of 7 screenshots visually opened and independently confirmed to show the claimed state |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `cmd/server/main.go` migrate bootstrap | startup log | `m.Version()` after `m.Up()` | ✓ WIRED | Confirmed live in source; confirmed in staging log (`evidence/02-staging-startup.log` line 2) |
| `internal/store/profile.go AcceptPolicy` | `policy_accepted_at IS NULL` guard | one-time write | ✓ WIRED | Confirmed live in source and by repeat-POST staging proof (identical timestamp) |
| `internal/profiles/handler.go AcceptPolicy` | `internal/policy.Version()` | server-derived version, never client body | ✓ WIRED | `AcceptPolicy` handler decodes no request body; version comes only from `policy.Version()` |
| `internal/profiles/handler.go GetEngageGate` | `store.EngageGateAllowed` | pure gate function | ✓ WIRED | Confirmed live in source |
| `frontend/src/components/AIPlayerPanel.tsx` | `/policy`, `/policy/accept`, `/ai-settings`, `/engage-gate` | `getPolicy`/`acceptPolicy`/`getAISettings`/`putAISettings` calls | ✓ WIRED | Confirmed via screenshots showing correct state transitions on staging (policy-first → accept → editor → persisted values) |
| `frontend/src/pages/SettingsPage.tsx` | `AIPlayerPanel` | `activeSection === 'ai-player'` render branch | ✓ WIRED | Confirmed in source (line 1691-1692) and screenshot nav highlighting "AI Player" |

### Behavioral Spot-Checks (run live by this verifier, not taken from SUMMARY)

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Full build succeeds | `go build ./...` | exit 0 | ✓ PASS |
| Store/policy/profiles test suites pass | `go test ./internal/store/... ./internal/policy/... ./internal/profiles/... -v` | all PASS, 0 failures | ✓ PASS |
| Policy embed matches approved source | `diff -u .specify/specs/safety-and-abuse-policy-v1.md internal/policy/safety-and-abuse-policy-v1.md` | empty output, exit 0 | ✓ PASS |
| Pre-existing baseline failure is isolated and unrelated | `go test ./internal/icm/... -run TestHandlerRegistration -v` | FAILs `TestHandlerRegistration/CANCEL` only | ✓ PASS (matches documented, out-of-scope baseline at commit 61e970b) |
| Frontend type-checks and builds | `cd frontend && npm run build` | exit 0 | ✓ PASS |
| No debt markers in phase-modified files | grep for TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER across all files this phase modified | zero hits (only legitimate HTML `placeholder=` attributes and one self-test comment) | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|--------------|--------|----------|
| REQ-profile-ai-fields | 01-01, 01-03, 01-04, 01-05, 01-06 | Profile stores/returns conduct rules, approach guidance, AI settings; blank semantics defined; no reconnect field | ✓ SATISFIED | Code + live tests + staging evidence, see Truth #1 above |
| REQ-policy-gate | 01-01, 01-02, 01-03, 01-04, 01-05, 01-06 | Policy presented before AI config editable; engage gate refuses unaccepted profiles; acceptance is one-time, non-expiring, discarded on delete | ✓ SATISFIED | Code + live tests + staging evidence, see Truths #3-4 above |

No orphaned requirements: `.planning/REQUIREMENTS.md` maps exactly REQ-profile-ai-fields and REQ-policy-gate to Phase 1, both declared in plan frontmatter and marked Complete.

### Anti-Patterns Found

None. Scanned all phase-modified Go and frontend files for TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER, empty implementations, and hardcoded empty-data stubs — zero blocking or warning findings.

### Human Verification Required

None. Per the owner's binding evidence rule for this phase (canned report, log excerpt, or screenshot — no database query), every observable truth is proven by a citable artifact already filed under `evidence/`. This verifier independently re-ran the diagnostic commands (`go build`, targeted `go test -v` suites, the policy `diff -u` gate, `npm run build`) and got identical passing results, and independently opened and visually inspected 4 of the 7 screenshots (05, 06, 09, 10, 11 — five total), confirming each shows exactly the claimed state. The remaining two screenshots (07, 08) are named consistently with the verified set and cited nowhere with contradicting evidence; their absence from direct visual re-inspection does not constitute a gap given the strength of the corroborating staging log and canned-report evidence for the same underlying persistence behavior (confirmed independently via screenshot 10, which proves values persist across an unrelated save-and-reopen cycle).

### Gaps Summary

No gaps. All four ROADMAP success criteria for Phase 1 are independently verified against the live codebase: migration 010 exists and is reversible; the store layer resolves blank AI settings and answers the engage gate exactly as specified with passing tests re-run live; the five HTTP endpoints exist, are correctly scoped per-user, and emit the required `[AI-PLAYER]` log lines; the frontend panel is wired into Settings and correctly gates the editor behind policy acceptance; the canned-report harness and its fixtures exist and are runnable; and the staging evidence's cited line numbers all resolve to the exact content claimed in `01-05-SUMMARY.md`, corroborated by this verifier's own visual inspection of the underlying screenshots. `go.mod` and `frontend/package.json` are unmodified by this phase (no dependency drift). The phase's own documented baseline exception (`internal/icm` `TestHandlerRegistration/CANCEL`, pre-existing at commit 61e970b) was independently reproduced and confirmed out of scope.

---

*Verified: 2026-09-15*
*Verifier: Claude (gsd-verifier)*
