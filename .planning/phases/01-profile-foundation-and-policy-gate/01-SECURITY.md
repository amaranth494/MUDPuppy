---
phase: 01
slug: profile-foundation-and-policy-gate
status: verified
threats_open: 0
asvs_level: 1
created: 2026-09-15
---

# Phase 01 — Profile Foundation and Policy Gate — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.
> Register authored at plan time (register_authored_at_plan_time: true) across 01-01-PLAN.md through 01-06-PLAN.md. Verified against implemented code, not documentation, on 2026-09-15.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| HTTP request body → store | Client JSON reaches `ProfileUpdate`; anything present in that struct is writable by the client | conduct_rules, approach_guidance, ai_settings |
| Go process → Postgres | The single `UPDATE profiles SET ...` statement rewrites every mutable column on every write | profile row columns |
| Go process → server log | Startup and request logs are readable by anyone with deploy-log access; whatever is logged is disclosed | ids, version, timestamp, booleans |
| Approved document → running binary | The text the owner accepts must be the approved text; embedding removes the filesystem as a tampering surface at runtime | policy markdown |
| Browser → `/api/v1/profiles/{connection_id}/*` | Untrusted JSON body and untrusted `connection_id` path segment cross here | AI settings body, connection_id |
| Session cookie → user identity | `sessionMiddleware` establishes `user_id` in the request context; every handler must scope to it | user_id |
| Browser state → server | The panel may only send `conduct_rules`, `approach_guidance` and `ai_settings`; acceptance is a separate, bodyless POST | form field values |
| Server response → rendered policy | Policy text is server-supplied markdown rendered into React elements | policy text |
| Repository → Railway staging | The deployed binary and the deployed schema may diverge from the repo; the startup log is what reconciles them | migration version |
| Operator shell → deployed server | The script carries a live session cookie and drives authenticated writes against whatever `BASE_URL` names | SESSION_COOKIE, BASE_URL |
| Report file → shared evidence | The report is committed or pasted into a SUMMARY; anything it prints is disclosed | request/response bodies |

---

## Threat Register

| Threat ID | Category | Component | Disposition | Mitigation | Status |
|-----------|----------|-----------|-------------|------------|--------|
| T-1-01 | Tampering | `store.ProfileUpdate` / `UpdateProfile`, `PutAISettings`, `putAISettings` client | mitigate | Acceptance columns absent from `ProfileUpdate` struct and `UPDATE ... SET` clause; only `AcceptPolicy` writes them; `AISettingsResponse` carries no acceptance field | closed |
| T-1-02 | Spoofing | `store.AcceptPolicy`, `policy.Version()`, `AcceptPolicy` handler | mitigate | Version parsed at init from embedded document; handler passes only `policy.Version()`, never decodes the request body | closed |
| T-1-03 | Elevation of Privilege / Disclosure (IDOR) | `{connection_id}` on all five new endpoints | mitigate | Every handler resolves through `getProfileByConnectionID` → `GetProfileByConnection(userID, connectionID)` scoped by both `user_id` and `connection_id` | closed |
| T-1-04 | DoS + Tampering | `validateAISettings` | mitigate | Exact literal limits present: 20000/20000/200 char caps and `< 1` floor on call cap and disengage threshold, each with the exact declared error message | closed (see residual-risk note) |
| T-1-05 | Tampering | `store.EngageGateAllowed`, `GetEngageGate`, client gate rendering | mitigate | Gate takes only server-fetched row values, no client input, no version comparison; frontend branch driven only by server's `accepted` field | closed |
| T-1-06 | Tampering (integrity) | `store.UpdateProfile` | mitigate | Nil-fallback branches present for `ConductRules`, `ApproachGuidance`, `AISettings`; confirmed end to end by canned-report step 10 and screenshot `10-after-timers-save.png` | closed |
| T-1-07 | Tampering | policy text delivery | mitigate | `go:embed` directive on `policyText`; missing/renamed file is a build failure, not a runtime 404; `diff -u` against source is empty | closed |
| T-1-08 | Elevation of Privilege | verb guards on write handlers | mitigate | All five handlers reject the wrong HTTP method with 405; accept is POST-only, ai-settings GET/PUT-only | closed |
| T-1-09 | XSS | policy rendering in `AIPlayerPanel.tsx` | mitigate | Policy rendered via `renderContent`/`renderInline` React elements; `grep -c dangerouslySetInnerHTML` returns 0 | closed |
| T-1-10 | DoS (schema drift) | staging deploy path / startup log | mitigate | `cmd/server/main.go` logs `Migrations completed successfully (version=%d, dirty=%v)` and, from the fallback, `AI Player columns ensured`; both literals confirmed present in `evidence/02-staging-startup.log` | closed |
| T-1-11 | Information Disclosure | server logs, browser console, committed evidence | mitigate | No `[AI-PLAYER]` format string interpolates conduct/guidance/policy text; `policy` package has zero `log.` calls; `AIPlayerPanel.tsx` has zero `console.log`; staging log evidence files grepped clean; canned report truncates policy `text` to 80 chars | closed |
| T-1-12 | Tampering (wrong target) | `BASE_URL` pointed at production | mitigate | `verify-phase1.sh` header records `BASE_URL` and `git rev-parse --short HEAD`; confirmed present in `evidence/03-canned-report.txt` header | closed |
| T-1-13 | Repudiation (unfalsifiable evidence) | canned-report harness | mitigate | `--self-test-negative` flag and `scripts/fixtures/phase1-negative/` exist; empirically re-run during this audit: produced `FAIL C2` and exit code 1 | closed |
| T-1-SC | Tampering (supply chain) | `go.mod`, `frontend/package.json`, `jq` | accept | See Accepted Risks Log below | closed |

*Status: open · closed*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

### Residual-risk note on T-1-04

The declared mitigation for T-1-04 (length caps and a `>= 1` floor on `call_cap`/`disengage_threshold`) is fully present and verified in `internal/profiles/handler.go:783-800` and exercised end-to-end by the canned report's over-length PASS line. Code review `01-REVIEW.md` flags two adjacent, undeclared gaps that were not part of this threat's committed mitigation scope and remain open as review follow-ups rather than phase blockers:
- **WR-01** — no upper bound on `call_cap`/`disengage_threshold` (only a floor), so an absurdly large value is functionally "no cap."
- **WR-03** — no `http.MaxBytesReader` before `json.NewDecoder(r.Body).Decode`, so the 20000-char check fires only after the full body is buffered.

Both are unresolved in the current code (`grep -c MaxBytesReader internal/ cmd/` returns 0) and are recommended for follow-up, but they do not invalidate the mitigation this threat's register entry actually commits to (a floor and explicit length caps), so T-1-04 is recorded CLOSED against its declared disposition, with this note as the auditable caveat.

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-1-01 | T-1-SC (register R-05) | No package was added to `go.mod` or `frontend/package.json` in any of the six plans comprising this phase. Verified via `evidence/01-test-report.txt` `### DEPENDENCY DRIFT` section: `git diff --stat 61e970b -- go.mod frontend/package.json` produced empty output (confirmed empty on re-check during this audit). `jq` is used only behind a `command -v jq` guard in `scripts/verify-phase1.sh` with a `grep`/`sed` fallback; nothing is installed. | Owner, Accept Risk at the Phase 1 review (criticality: low) | 2026-09-15 |
| AR-1-02 | WR-01 (register R-01) | No upper bound on `call_cap` / `disengage_threshold` in `validateAISettings`; only a `>= 1` floor. An absurdly large cap is functionally "no cap". Proposed remediation (not taken): add maximums, tests, and a matching hint in the panel. | Owner, Accept Risk at the Phase 1 review (criticality: medium) | 2026-09-15 |
| AR-1-03 | WR-03 (register R-03) | Request body buffered in full before the 20000-character checks run; no `http.MaxBytesReader` anywhere in the server. Pre-existing pattern, newly exercised by two long free-text fields. Proposed remediation (not taken): 1 MiB `MaxBytesReader` on JSON endpoints with a 413 response. | Owner, Accept Risk at the Phase 1 review (criticality: medium) | 2026-09-15 |
| AR-1-04 | OBS-02 (register R-06) | Throwaway test user (example.com address) and two connection profiles created on staging by the canned-report run remain as test data. Proposed remediation (not taken): delete via the app using the owner's session. | Owner, Accept Risk at the Phase 1 review (criticality: low) | 2026-09-15 |
| AR-1-05 | IN-01 (register R-07) | `migrations/010_add_ai_fields.*.sql` carry inert `-- +migrate` directive comments from another tool, matching migration 009. No runtime effect. | Owner, Accept Risk at the Phase 1 review (criticality: low) | 2026-09-15 |

*Accepted risks do not resurface in future audit runs.*

---

## Deferred Risks

Temporarily accepted so Phase 1 can close. Each MUST be raised again at the Phase 2 security review with the same three choices (Accept Risk / Defer Until Next Review / Remediate Now). The project cannot be considered closed while any row here remains deferred. Carry-forward record: `.planning/todos/pending/2026-09-15-deferred-security-risks.md`.

| Risk ID | Ref | Criticality | Risk | Proposed remediation | Deferred By | Date | Raise at |
|---------|-----|-------------|------|----------------------|-------------|------|----------|
| DR-1-01 | WR-02 (register R-02) | medium | Invalid numeric input in `AIPlayerPanel.tsx` `handleSave` serialises as `null` and silently saves as "no cap" / "default" instead of a validation error. | Parse each numeric field before building the request; if non-empty and not a finite number, block the save with an inline error naming the field. | Owner | 2026-09-15 | Phase 2 security review |
| DR-1-02 | OBS-01 (register R-04) | medium | Staging prints the one-time login code in the deploy log (`STAGING: OTP sent to user, code: NNNNNN`); anyone with staging log access can sign in as any staging user. Pre-existing and staging-only; it is the mechanism the Phase 1 evidence run used. | Gate the code-logging line behind an env flag that is off by default, or log a hash; confirm production has no equivalent; review Railway log access on the staging project. | Owner | 2026-09-15 | Phase 2 security review |

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-09-15 | 14 | 14 | 0 | gsd-security-auditor |
| 2026-09-15 | 7 register items (WR-01..03, IN-01, T-1-SC, OBS-01..02) | 5 accepted | 2 deferred to Phase 2 | Owner, via the Phase 1 Risk Register artifact (decisions read back from its store) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter
- [x] Owner risk decisions recorded: 5 accepted (AR-1-01..05), 2 deferred (DR-1-01..02), 0 remediate-now — phase remains closed
- [ ] Deferred risks DR-1-01 and DR-1-02 re-raised at the Phase 2 security review (open until then)

**Approval:** verified 2026-09-15; owner risk decisions recorded 2026-09-15
