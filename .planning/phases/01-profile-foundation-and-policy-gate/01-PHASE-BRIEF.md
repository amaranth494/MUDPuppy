# Phase 1 Brief — Profile Foundation and Policy Gate

**Purpose:** Owner review before execution. Everything here is lifted from `.planning/ROADMAP.md` §Phase 1 and the five PLAN.md files; nothing is new. Approve this, and Phase 1 goes to `/gsd-execute-phase 1`.

---

## Phase Goal

The connection profile is the single per-game home for the AI, and no profile can have the AI configured or engaged until its owner has accepted the Safety and Abuse policy on it, once.

## Phase Success Criteria (what must be TRUE afterwards)

| # | Criterion | How it is demonstrated |
|---|-----------|------------------------|
| 1 | A profile stores and returns conduct rules, approach guidance, and AI settings (model name, call cap per session, disengage threshold). Blank model name = server default; blank call cap = no cap; blank threshold = engine built-in error handling (any AI failure yields an informative error in the play screen and disengages without crashing or interrupting play). No reconnect field. | `go test ./internal/store/...` passes the blank-resolution table tests; `SELECT` on a migrated staging `profiles` row shows the five new columns. |
| 2 | The owner can read and edit conduct rules, approach guidance, and AI settings for a profile from the browser, and the values survive a page reload and a new session. | Browser walkthrough on staging: edit, save, hard reload, log out and in, values unchanged. |
| 3 | The first time the owner opens the AI Player configuration for a profile, the policy is presented and must be accepted before AI settings can be edited; the server-side engage gate refuses any profile without a recorded acceptance, with a clear message. | Browser: fresh profile shows policy first, editor only after Accept. HTTP: `GET .../engage-gate` returns `allowed:false` plus the refusal message before acceptance and `allowed:true` after. |
| 4 | Acceptance is recorded once per profile with timestamp and policy version 1.0; it never expires, a later policy change does not require re-acceptance, and deleting the profile discards it. | `SELECT policy_version_accepted, policy_accepted_at` shows `1.0` and a timestamp; a second Accept leaves them unchanged; delete and recreate the profile, and the policy is asked again. |

**Phase Validation line (from ROADMAP):** Diagnostic: `go test` covers default resolution for blank AI settings and the engage-gate decision; a database inspection of a migrated profile shows the new fields and the acceptance columns carrying version 1.0 and a timestamp. Player-observable: open AI Player on a fresh profile and the policy appears first; accept once, then the settings editor is usable; edit, reload, values persist; call the engage gate on an un-accepted profile and receive the refusal; on the accepted profile it passes; delete the profile, recreate it, and the policy is asked again.

---

## Plans, Waves, and Acceptance Criteria

Waves are dependency order only. Wave 1 plans touch no common files and can run in parallel.

### Wave 1

#### Plan 01-01 — Profile row carries AI fields and a one-time acceptance record, with blanks resolved in Go

Capability created: the `profiles` table and the Go store know about the AI fields and the acceptance record, and Go alone decides what blank means and whether the gate is open.

| Acceptance criterion | Demonstrable artifact |
|----------------------|-----------------------|
| A migrated `profiles` row has `conduct_rules`, `approach_guidance`, `ai_settings`, `policy_version_accepted`, `policy_accepted_at`; a new profile has blank AI fields and NULL acceptance columns. | `migrations/010_add_ai_fields.up.sql` / `.down.sql`; `SELECT` output on a migrated database. |
| Blank model name resolves to the server default, blank call cap to no cap, blank disengage threshold to 3 consecutive transient failures. | `go test ./internal/store/... -run TestResolveAISettings` exits 0. |
| `store.EngageGateAllowed` returns false while the acceptance columns are unset and true once both are set. | `go test ./internal/store/... -run TestEngageGate` exits 0. |
| Recording acceptance a second time leaves the first timestamp and version untouched. | `AcceptPolicy` SQL contains `WHERE ... policy_accepted_at IS NULL`; store test asserts the guard. |
| The acceptance columns cannot be written through the general profile update path. | `ProfileUpdate` struct has no acceptance fields; only `store.AcceptPolicy` writes them. |
| Saving timers, aliases, triggers, or environment leaves the AI fields intact. | `UpdateProfile` carries SET clause, argument, and nil-fallback branch for each new column. |
| `AISettings` has exactly `model_name`, `call_cap`, `disengage_threshold`. No reconnect field. | Source assertion on `internal/store/profile.go`. |

Tasks: 01-01-01 migration 010 and startup fallback; 01-01-02 store read/write/accept; 01-01-03 resolver, gate, and tests.

#### Plan 01-02 — Policy text and version 1.0 ship inside the server binary

Capability created: the server can produce the approved policy text and its version with no filesystem dependency, and the version written on acceptance can never drift from the text shown.

| Acceptance criterion | Demonstrable artifact |
|----------------------|-----------------------|
| The policy markdown is compiled into the binary via `go:embed`. | `internal/policy/policy.go` contains `//go:embed safety-and-abuse-policy-v1.md`. |
| `policy.Version()` returns the string `1.0`, parsed from the document's `Policy version:` header, not hard-coded. | `go test ./internal/policy/... -run TestParseVersion` exits 0. |
| The embedded copy is byte-identical to `.specify/specs/safety-and-abuse-policy-v1.md`. | Test compares the two files; `diff` exits 0. |
| The package contains no version comparison and no expiry path. | Source assertion. |

Tasks: 01-02-01 embed text and parse version; 01-02-02 tests.

### Wave 2 (after Wave 1)

#### Plan 01-03 — AI settings, the policy, one-time acceptance, and the engage gate are reachable over HTTP

Capability created: the browser now, and Phase 2's `#AUTO ON` later, can read and write AI settings, read the policy, accept it once, and ask the gate, all under existing session auth and per-user profile scoping. This is the phase's diagnostic surface.

Endpoints, all under `/api/v1/profiles/{connection_id}/`:

| Endpoint | Purpose |
|----------|---------|
| `GET` / `PUT ai-settings` | Read and write conduct rules, approach guidance, AI settings |
| `GET policy` | Policy text, version `1.0`, and this profile's acceptance state |
| `POST policy/accept` | Record acceptance once, from the server's version and clock |
| `GET engage-gate` | `{"allowed": bool, "message": string}` |

| Acceptance criterion | Demonstrable artifact |
|----------------------|-----------------------|
| PUT then GET of ai-settings returns the values unchanged; blank fields round-trip as blank. | `go test ./internal/profiles/... -run TestAISettings` exits 0; `curl` round-trip. |
| Over-length input is rejected with a specific message and nothing is written. Limits: conduct rules and approach guidance 20000 chars, model name 200 chars, call cap and threshold at least 1 when set. | `PUT` with 20001 chars returns HTTP 400 and `{"error":"Conduct rules must be 20000 characters or less"}`. |
| A PUT carrying `policy_accepted_at` or `policy_version_accepted` changes neither column. | `TestAISettingsCannotSetAcceptance` passes. |
| `POST policy/accept` records version `1.0` and the server clock once; a second POST returns the same values. | `TestPolicyAcceptUsesServerVersion` passes; handler never decodes a client-supplied version. |
| `GET engage-gate` on an unaccepted profile returns `allowed:false` with the message: "AI Player has not been configured for this connection. Accept the Safety and Abuse policy in AI Player settings before engaging autopilot." On an accepted profile it returns `allowed:true`. | `TestEngageGateHandlerRefusesWithoutAcceptance` and the allowed counterpart pass; `curl` shows the exact body. |
| Another user's `connection_id` yields "Profile not found", never their data. | Every handler resolves via `getProfileByConnectionID` (scoped by `user_id`). |
| The ai-settings body has no reconnect field. | Source assertion. |

Tasks: 01-03-01 ai-settings GET/PUT and validation; 01-03-02 policy, accept, gate handlers; 01-03-03 routes and `httptest` coverage.

### Wave 3 (after Wave 2)

#### Plan 01-04 — The owner accepts the policy and edits AI settings from the browser

Capability created: Settings gains an AI Player section per connection profile that shows the policy first, accepts it once, then exposes the editor. Follows `01-UI-SPEC.md`.

| Acceptance criterion | Demonstrable artifact |
|----------------------|-----------------------|
| Settings shows an AI Player entry beside Key Bindings, Aliases, Triggers, Timers, Environment. | `SettingsPage.tsx` contains an `ai-player` section; visible in the browser. |
| On an unaccepted profile the section shows the policy text, its version, and a single Accept Policy button; no settings field is rendered or editable. | Browser walkthrough on a fresh profile. |
| One press of Accept Policy replaces the policy panel with "✓ Accepted v1.0 on YYYY-MM-DD" and the editor, without a reload. | Browser walkthrough. |
| The editor edits conduct rules, approach guidance, model name, call cap, disengage threshold; save then reload shows the saved values. | Browser walkthrough. |
| Clearing model name, call cap, or threshold saves as blank and comes back blank; hint text states what blank means. | Browser walkthrough. |
| No reconnect control anywhere in the section. | Source assertion on `AIPlayerPanel.tsx`. |
| Load and save failures show the UI-SPEC error messages in a dismissible banner. | Browser: stop the server, observe the banner. |
| Frontend builds clean. | `cd frontend && npm run build` exits 0. |

Tasks: 01-04-01 types and API client; 01-04-02 `AIPlayerPanel.tsx`; 01-04-03 Settings nav and section wiring.

### Wave 4 (after Wave 3)

#### Plan 01-05 — Phase demonstrated on staging, evidence recorded

Capability created: the ROADMAP Phase Validation line has actually been performed against Railway staging and the evidence is written down. Under the binding method, Phase 1 is not complete until this plan passes. Two tasks are blocking human checkpoints.

| Acceptance criterion | Demonstrable artifact |
|----------------------|-----------------------|
| Full Go suite and frontend build green; no package added to `go.mod` or `frontend/package.json`. | `go test ./...` and `npm run build` output; `git diff --stat go.mod frontend/package.json` empty. |
| Migration 010 has run on staging; a `profiles` row shows all five new columns. | Staging `SELECT conduct_rules, approach_guidance, ai_settings, policy_version_accepted, policy_accepted_at FROM profiles LIMIT 1;` output pasted into the summary. Checkpoint 01-05-02. |
| Fresh profile: AI Player shows the policy first; engage-gate refuses with the exact message. | Walkthrough plus `curl` output. Checkpoint 01-05-03. |
| After one Accept: row carries `1.0` and a timestamp, editor usable, engage-gate allows. | Walkthrough plus `SELECT` and `curl` output. |
| Edited values survive a hard reload and a new login session. | Walkthrough. |
| Saving timers does not blank the AI fields. | Walkthrough plus `GET ai-settings`. |
| Delete the profile, recreate it for the same game, policy is asked again. | Walkthrough. |
| Evidence recorded per ROADMAP success criterion. | `01-05-SUMMARY.md` with a pass/fail line for criteria 1 through 4. |

Tasks: 01-05-01 suite and dependency check (automated); 01-05-02 staging migration and DB inspection (checkpoint); 01-05-03 staging walkthrough (checkpoint).

---

## Traceability

| Item | Where covered |
|------|---------------|
| REQ-profile-ai-fields | 01-01, 01-03, 01-04, 01-05 |
| REQ-policy-gate | 01-01, 01-02, 01-03, 01-04, 01-05 |
| Decisions D-01 to D-13 | each cited by at least one plan's `must_haves.truths` |
| Security threats T-1-01 to T-1-10 | each has a mitigating task; the three high-severity ones (acceptance forgery, ownership bypass, gate bypass) have dedicated tests in 01-03 |

## Decided under Claude's Discretion (plumbing, no owner action needed)

- Blank disengage threshold resolves to 3 consecutive transient failures. Phase 4 consumes it.
- Text limits: 20000 chars for conduct rules and approach guidance, 200 for model name.
- Policy markdown is copied into `internal/policy/` and embedded; version parsed from the `Policy version:` header once at startup.
- One migration file (`010_add_ai_fields`) carries both the AI fields and the acceptance columns.
- Endpoint names: `ai-settings`, `policy`, `policy/accept`, `engage-gate`.

## Explicitly out of this phase

No autopilot, no AI calls, no session log, no `#AI STATUS` directive, no policy section 7 enforcement, no reconnect field.
