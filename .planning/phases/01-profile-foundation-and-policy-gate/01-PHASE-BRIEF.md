# Phase 1 Brief — Profile Foundation and Policy Gate

**Purpose:** Owner review before execution. Everything here is lifted from `.planning/ROADMAP.md` §Phase 1 and the six PLAN.md files; nothing is new. Approve this, and Phase 1 goes to `/gsd-execute-phase 1`.

**Evidence rule (owner-directed, binding for this and every later phase):** every success criterion and acceptance criterion is proven by one of two artifact types, and nothing else.

1. **A canned report.** A repeatable script or test run whose output is captured verbatim to a file under `evidence/`. Server log excerpts captured to a file count here.
2. **A screenshot from the end-user perspective.** The MUDPuppy browser page as the owner sees it. No devtools pane, no terminal, no raw JSON in frame.

Database queries and inspections are not evidence anywhere in this phase.

---

## Phase Goal

The connection profile is the single per-game home for the AI, and no profile can have the AI configured or engaged until its owner has accepted the Safety and Abuse policy on it, once.

## Phase Success Criteria (what must be TRUE afterwards)

| # | Criterion | Proof (canned report or end-user screenshot) |
|---|-----------|----------------------------------------------|
| 1 | A profile stores and returns conduct rules, approach guidance, and AI settings (model name, call cap per session, disengage threshold). Blank model name = server default; blank call cap = no cap; blank threshold = engine built-in error handling (any AI failure yields an informative error in the play screen and disengages without crashing or interrupting play). No reconnect field. | `evidence/01-test-report.txt` PASS lines for the blank-resolution tests; `evidence/03-canned-report.txt` `C1` lines showing blanks round-trip; `evidence/02-staging-startup.log` showing migration 010 applied; screenshot `09-blank-fields.png` showing cleared fields back blank with hint text. |
| 2 | The owner can read and edit conduct rules, approach guidance, and AI settings for a profile from the browser, and the values survive a page reload and a new session. | Screenshots `07-values-after-reload.png` and `08-values-new-session.png`. |
| 3 | The first time the owner opens the AI Player configuration for a profile, the policy is presented and must be accepted before AI settings can be edited; the server-side engage gate refuses any profile without a recorded acceptance, with a clear message. | Screenshots `05-policy-first.png` and `06-accepted-line.png`; `evidence/03-canned-report.txt` `C3` lines showing the refusal message before acceptance and allowed after; `evidence/04-staging-ai-player.log` `engage gate ... allowed=false` then `allowed=true`. |
| 4 | Acceptance is recorded once per profile with timestamp and policy version 1.0; it never expires, a later policy change does not require re-acceptance, and deleting the profile discards it. | `evidence/04-staging-ai-player.log` `policy accepted ... version=1.0 accepted_at=...`; `evidence/03-canned-report.txt` `C4` lines showing a second Accept returns the same timestamp; screenshot `11-recreated-profile-policy-again.png`. |

**Phase Validation line (from ROADMAP):** Diagnostic: `go test` covers default resolution for blank AI settings and the engage-gate decision; server logs on staging show migration 010 applied at startup, one structured line per policy acceptance carrying the connection id, policy version 1.0 and the timestamp, and one line per engage-gate decision. Proof for every criterion is a UAT finding from the browser walkthrough or a server log excerpt; database queries are not accepted as evidence. Player-observable: open AI Player on a fresh profile and the policy appears first; accept once, then the settings editor is usable; edit, reload, values persist; call the engage gate on an un-accepted profile and receive the refusal; on the accepted profile it passes; delete the profile, recreate it, and the policy is asked again.

---

## Evidence set produced by this phase

All files live under `.planning/phases/01-profile-foundation-and-policy-gate/evidence/`.

| File | Type | Produced by |
|------|------|-------------|
| `01-test-report.txt` | canned report: `go test ./... -v`, policy `diff`, `npm run build`, dependency-drift check, verbatim | 01-05-01 (automated) |
| `02-staging-startup.log` | log excerpt: `Migrations completed successfully (version=10, dirty=false)` and `AI Player columns ensured` | 01-05-02 (checkpoint) |
| `03-canned-report.txt` | canned report: `scripts/verify-phase1.sh` run against staging, every request and response, one PASS/FAIL per criterion `C1`..`C4` | 01-05-02 (checkpoint) |
| `04-staging-ai-player.log` | log excerpt: the `[AI-PLAYER]` lines for accept, repeat accept, gate refuse, gate allow, settings saved | 01-05-02 (checkpoint) |
| `05-policy-first.png` | end-user screenshot: fresh profile, policy text, version, single Accept button, no editor | 01-05-03 (checkpoint) |
| `06-accepted-line.png` | end-user screenshot: "✓ Accepted v1.0 on <date>" and the editor, no reload | 01-05-03 |
| `07-values-after-reload.png` | end-user screenshot: five edited fields after a hard reload | 01-05-03 |
| `08-values-new-session.png` | end-user screenshot: same after logout and login | 01-05-03 |
| `09-blank-fields.png` | end-user screenshot: cleared model, cap, threshold back blank with hint text | 01-05-03 |
| `10-after-timers-save.png` | end-user screenshot: AI fields intact after saving Timers | 01-05-03 |
| `11-recreated-profile-policy-again.png` | end-user screenshot: deleted and recreated profile shows the policy again | 01-05-03 |
| `01-05-SUMMARY.md` | one PASS/FAIL row per success criterion, each citing the files above by name and line | 01-05-03 |

The engage-gate refusal has no end-user surface until Phase 2 wires `#AUTO ON`, so it is proven by the canned report and the log, not a screenshot.

---

## Plans, Waves, and Acceptance Criteria

Waves are dependency order only. Plans in the same wave touch no common files.

### Wave 1

#### Plan 01-01 — Profile row carries AI fields and a one-time acceptance record, with blanks resolved in Go

Capability: the `profiles` table and the Go store know about the AI fields and the acceptance record, and Go alone decides what blank means and whether the gate is open.

| Acceptance criterion | Proof |
|----------------------|-------|
| Migration 010 applies and the five columns exist. | `02-staging-startup.log` line `Migrations completed successfully (version=10, dirty=false)` and `AI Player columns ensured`. |
| Blank model name resolves to the server default, blank call cap to no cap, blank disengage threshold to 3 consecutive transient failures. | `01-test-report.txt` PASS line for `TestResolveAISettings`. |
| `store.EngageGateAllowed` returns false while the acceptance columns are unset and true once both are set. | `01-test-report.txt` PASS line for `TestEngageGate`. |
| Recording acceptance a second time leaves the first timestamp and version untouched. | `03-canned-report.txt` `C4` line: second POST returns the same `accepted_at`. |
| The acceptance columns cannot be written through the general profile update path. | `01-test-report.txt` PASS line for `TestAISettingsCannotSetAcceptance`; `03-canned-report.txt` `C4` line: PUT with acceptance fields leaves `accepted=false`. |
| Saving timers, aliases, triggers, or environment leaves the AI fields intact. | `03-canned-report.txt` `C1` line after the timers PUT; screenshot `10-after-timers-save.png`. |
| `AISettings` has exactly `model_name`, `call_cap`, `disengage_threshold`. No reconnect field. | `03-canned-report.txt` response bodies show only those three keys; no reconnect control in any screenshot. |
| No log line ever carries conduct rules, approach guidance, or policy text. | `01-test-report.txt` PASS line for `TestAIPlayerLogLinesAreEmitted`; `04-staging-ai-player.log` contains ids, version, timestamps, and booleans only. |

Tasks: 01-01-01 migration 010, version log line, startup fallback; 01-01-02 store read, write, one-time accept; 01-01-03 resolver, gate, tests.

#### Plan 01-02 — Policy text and version 1.0 ship inside the server binary

Capability: the server produces the approved policy text and its version with no filesystem dependency, so the version written on acceptance cannot drift from the text shown.

| Acceptance criterion | Proof |
|----------------------|-------|
| `policy.Version()` returns `1.0`, parsed from the document's `Policy version:` header. | `01-test-report.txt` PASS line for `TestParseVersion`. |
| The embedded copy is byte-identical to `.specify/specs/safety-and-abuse-policy-v1.md`. | `01-test-report.txt` `diff` section empty and PASS line for `TestTextOfEmbeddedPolicy`. |
| The owner sees that text and version in the browser. | Screenshot `05-policy-first.png`. |
| No version comparison and no expiry path exist. | `03-canned-report.txt` `C4` line: repeat accept returns the original version and timestamp. |

Tasks: 01-02-01 embed text and parse version; 01-02-02 tests.

### Wave 2 (after Wave 1)

#### Plan 01-03 — AI settings, the policy, one-time acceptance, and the engage gate are reachable over HTTP

Capability: the browser now, and Phase 2's `#AUTO ON` later, can read and write AI settings, read the policy, accept it once, and ask the gate, under existing session auth and per-user profile scoping. Every call that changes or decides something writes one log line.

Endpoints under `/api/v1/profiles/{connection_id}/`: `GET`/`PUT ai-settings`, `GET policy`, `POST policy/accept`, `GET engage-gate`.

Log contract (standard `log` package, `[AI-PLAYER]` prefix): `policy accepted connection_id= user_id= version= accepted_at=`; `policy already accepted connection_id= version= accepted_at=`; `engage gate connection_id= allowed=`; `ai settings saved connection_id= model_name_blank= call_cap_blank= threshold_blank=`.

| Acceptance criterion | Proof |
|----------------------|-------|
| PUT then GET of ai-settings returns the values unchanged; blanks round-trip as blank. | `03-canned-report.txt` `C1` lines with both response bodies. |
| Over-length input is rejected with a specific message and nothing is written. Limits: 20000 chars for conduct rules and approach guidance, 200 for model name, cap and threshold at least 1 when set. | `03-canned-report.txt` `C1` line: HTTP 400 with `Conduct rules must be 20000 characters or less`; `01-test-report.txt` PASS line for `TestAISettingsRejectsOverLengthText`. |
| A PUT carrying acceptance fields changes nothing about acceptance. | `03-canned-report.txt` `C4` line; `01-test-report.txt` PASS line for `TestAISettingsCannotSetAcceptance`. |
| `POST policy/accept` records version 1.0 and the server clock once; a second POST returns the same values. | `04-staging-ai-player.log` `policy accepted` then `policy already accepted` with equal `accepted_at`; `03-canned-report.txt` `C4`. |
| `GET engage-gate` returns `allowed:false` with the message "AI Player has not been configured for this connection. Accept the Safety and Abuse policy in AI Player settings before engaging autopilot." before acceptance, and `allowed:true` after. | `03-canned-report.txt` `C3` lines; `04-staging-ai-player.log` `engage gate ... allowed=false` then `allowed=true`. |
| Another user's connection id yields "Profile not found", never their data. | `01-test-report.txt` PASS line for the ownership test in `handler_test.go`. |
| The four log lines are emitted and contain no profile text. | `01-test-report.txt` PASS line for `TestAIPlayerLogLinesAreEmitted`. |

Tasks: 01-03-01 ai-settings GET/PUT and validation; 01-03-02 policy, accept, gate handlers with log lines; 01-03-03 routes and `httptest` coverage.

### Wave 3 (after Wave 2)

#### Plan 01-04 — The owner accepts the policy and edits AI settings from the browser

Capability: Settings gains an AI Player section per connection profile that shows the policy first, accepts it once, then exposes the editor. Follows `01-UI-SPEC.md`.

| Acceptance criterion | Proof |
|----------------------|-------|
| Settings shows an AI Player entry beside Key Bindings, Aliases, Triggers, Timers, Environment. | Every screenshot `05` to `11` shows the nav entry. |
| On an unaccepted profile the section shows the policy text, its version, and a single Accept Policy button; no settings field is rendered. | Screenshot `05-policy-first.png`. |
| One press of Accept Policy shows "✓ Accepted v1.0 on <date>" and the editor without a reload. | Screenshot `06-accepted-line.png`. |
| Saved conduct rules, approach guidance, and AI settings survive a hard reload and a new session. | Screenshots `07-values-after-reload.png`, `08-values-new-session.png`. |
| Cleared model name, call cap, and threshold save as blank and come back blank with hint text stating what blank means. | Screenshot `09-blank-fields.png`. |
| No reconnect control anywhere in the section. | Screenshots `06` to `10`. |
| Frontend builds clean. | `01-test-report.txt` `npm run build` section exits 0. |

Tasks: 01-04-01 types and API client; 01-04-02 `AIPlayerPanel.tsx`; 01-04-03 Settings nav and section wiring.

#### Plan 01-06 — One command produces the Phase 1 canned report

Capability: `scripts/verify-phase1.sh` drives the whole Phase 1 HTTP sequence against any running server and writes a report with every request, every response body, and one PASS/FAIL line per success criterion `C1`..`C4`, then appends the full `go test ./... -v` output. This is the canned report used on staging.

| Acceptance criterion | Proof |
|----------------------|-------|
| A clean run prints `PASS C1` through `PASS C4` and exits 0. | `--self-test` against bundled fixtures, captured in `01-test-report.txt`. |
| A wrong response produces a `FAIL` line and a non-zero exit, so a clean report means something. | `--self-test-negative` exits 1 with `FAIL C2`, captured in `01-test-report.txt`. |
| The report never prints the session cookie and truncates the policy text. | Self-test report contains no cookie value. |

The session cookie is browser-issued (OTP login), so the caller supplies it; the fixture self-tests are the automated gate and the staging run in 01-05-02 is the live proof.

Tasks: 01-06-01 the script; 01-06-02 self-tests and `go test` capture.

### Wave 4 (after Wave 3)

#### Plan 01-05 — Phase demonstrated on staging, evidence filed

Capability: the ROADMAP Phase Validation line has actually been performed against Railway staging and the evidence is on disk. Under the binding method, Phase 1 is not complete until this plan passes. Two tasks are blocking human checkpoints.

| Task | What happens | Artifacts |
|------|--------------|-----------|
| 01-05-01 (automated) | Full Go suite, policy diff, self-tests, frontend build, dependency-drift check captured verbatim. | `01-test-report.txt` |
| 01-05-02 (checkpoint) | Deploy to staging. Capture the startup log. Run `scripts/verify-phase1.sh` against staging with a fresh profile. Capture the `[AI-PLAYER]` log lines. Log capture via the Railway MCP log tool, `railway logs`, or the dashboard log pane. | `02-staging-startup.log`, `03-canned-report.txt`, `04-staging-ai-player.log` |
| 01-05-03 (checkpoint) | Browser walkthrough on staging producing the seven end-user screenshots, then the summary table with a PASS/FAIL per criterion citing files and line numbers. Screenshots may be taken by the executor with browser automation or by the owner; the owner confirms at the checkpoint. | `05` to `11` PNGs, `01-05-SUMMARY.md` |

---

## Traceability

| Item | Where covered |
|------|---------------|
| REQ-profile-ai-fields | 01-01, 01-03, 01-04, 01-05, 01-06 |
| REQ-policy-gate | all six plans |
| Decisions D-01 to D-13 | each cited by at least one plan's `must_haves.truths` |
| Security threats T-1-01 to T-1-11 | each mitigated by a named task; high-severity ones (acceptance forgery, ownership bypass, gate bypass) have dedicated tests; T-1-11 keeps profile text out of logs |

## Decided under Claude's Discretion (plumbing, no owner action needed)

- Blank disengage threshold resolves to 3 consecutive transient failures. Phase 4 consumes it.
- Text limits: 20000 chars for conduct rules and approach guidance, 200 for model name.
- Policy markdown copied into `internal/policy/` and embedded; version parsed from the `Policy version:` header once at startup.
- One migration file (`010_add_ai_fields`) carries both the AI fields and the acceptance columns.
- Endpoint names: `ai-settings`, `policy`, `policy/accept`, `engage-gate`.
- Log lines use the existing standard `log` package with an `[AI-PLAYER]` prefix.

## Explicitly out of this phase

No autopilot, no AI calls, no session log, no `#AI STATUS` directive, no policy section 7 enforcement, no reconnect field, no end-user surface for the gate refusal (Phase 2).
