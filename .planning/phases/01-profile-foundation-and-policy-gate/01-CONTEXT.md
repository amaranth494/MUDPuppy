# Phase 1: Profile Foundation and Policy Gate - Context

**Gathered:** 2026-09-15
**Status:** Ready for planning

<domain>
## Phase Boundary

The connection profile becomes the AI's per-game home. It stores conduct rules, approach guidance, and AI settings (one model name, a per-session call cap, a disengage threshold), all editable from the browser, with defined behaviour when any of them is blank. A one-time Safety and Abuse policy acceptance is recorded on the profile: the policy is shown the first time the owner opens AI configuration for that profile, accepted once, never expires, and is never re-asked. A server-side engage gate answers "may the AI be engaged on this profile" from that record; Phase 2's `#AUTO ON` will call it. No autopilot, no AI calls, no session log in this phase.

</domain>

<decisions>
## Implementation Decisions

### Where the owner edits and accepts
- **D-01:** AI configuration lives in a new **AI Player** section of the existing Settings page, beside Key Bindings, Aliases, Triggers, and Timers. The older ProfileModal is untouched.
- **D-02:** The section is reached per connection profile and uses the same per-connection GET/PUT sub-resource pattern as aliases, triggers, environment, and timers.
- **D-03:** The policy is presented inside the AI Player section. On a profile with no acceptance, opening the section shows the full policy text, its version, and a single **Accept** button, and nothing else is editable until accepted. After acceptance the section shows "Accepted v1.0 on <date>" and the settings editor.
- **D-04:** Accepting is a single button press. Accepting the policy text is the owner's confirmation under policy section 1; no extra checkbox, no scroll-to-end requirement, nothing else recorded.
- **D-05:** A refused engage (Phase 2) shows a clear message and links the owner to the AI Player section for that profile.

### Policy acceptance record and lifetime
- **D-06:** Acceptance is **one-time per profile** (per MUD connection). It does not expire. A later change to the policy text or version does **not** require re-acceptance. Deleting the profile discards the acceptance, so a new profile for the same game is asked once again.
- **D-07:** Acceptance is stored as columns on the `profiles` row: the policy version string accepted and an accepted-at timestamp. No history table.
- **D-08:** The engage gate is a server-side check on those columns: accepted means engageable. The gate is the only thing Phase 2 needs from this phase.

### AI settings shape and blank behaviour
- **D-09:** A profile holds **one** model name. Blank means the server's configured default model (an environment variable, added in Phase 3 when Gemini is wired; Phase 1 only stores and returns the field).
- **D-10:** Blank call cap means **no cap**. There is no default cap number.
- **D-11:** Blank disengage threshold means the engine's built-in error handling applies. The rule the owner set: any AI failure must fail with an informative error shown in the play screen, disengage the AI, and leave regular play untouched. **It can never crash** the server, the session, or the browser.
- **D-12:** Conduct rules and approach guidance are free text (textarea), handed to the model verbatim in later phases.
- **D-13:** There is no reconnect field in AI settings (locked in PROJECT.md).

### Claude's Discretion
- Policy text delivery: a markdown file embedded in the Go binary (`go:embed`) and served by one small endpoint; the version string is parsed from the file's "Policy version:" header at startup and is what gets written into the acceptance columns. Nothing compares versions afterwards.
- The numeric disengage-threshold default used when the field is blank (a small count of consecutive transient failures; non-transient failures such as a missing key or auth error disengage on the first occurrence). Phase 1 only stores and resolves the value; the loop that uses it is Phase 4.
- Text length limits for conduct rules and approach guidance, and validation messages, following the existing `validateUpdate` style in `internal/profiles/handler.go`.
- Exact JSONB shape of `ai_settings` and the Go struct/defaults that resolve blank fields, following `ProfileSettings` and `DefaultProfileSettings` in `internal/store/profile.go`.
- Endpoint paths under `/api/v1/profiles/{connection_id}/...` for the AI sub-resource, the policy text, the accept action, and the engage-gate check (a GET the browser and Phase 2 can both call; it doubles as the diagnostic surface for this phase).
- Migration numbering (`010`) and whether acceptance columns and AI fields share one migration.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Product definition
- `.specify/specs/ai-game-player-design-v3.md` — Deliverable D1 (profile foundation and policy gate) and Definition of complete items 1 and 6. Amended 2026-09-15 for one-time acceptance and blank-setting behaviour.
- `.specify/specs/safety-and-abuse-policy-v1.md` — The policy text (version 1.0) that is served and accepted; intro paragraph amended 2026-09-15 (no re-acceptance on change).

### Method
- `.specify/memory/phase-based-development-approach.md` — Binding Phase → Wave → Plan → Task rules. Plans are capabilities with observable acceptance criteria.
- `CLAUDE.md` — Distilled method rules and project facts every agent must honor.

### Planning state
- `.planning/ROADMAP.md` §Phase 1 — Goal, success criteria, Phase Validation line, implementation notes.
- `.planning/REQUIREMENTS.md` — REQ-profile-ai-fields, REQ-policy-gate.
- `.planning/PROJECT.md` — Locked decisions (server-side driver, reconnect, prior specs excluded, DEC-policy-accept-once) and constraints.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/store/profile.go`: `Profile`, `ProfileSettings`, `DefaultProfileSettings`, `normalizeSettings`, `ProfileUpdate`, and per-column JSONB scan/marshal in `GetProfile`/`UpdateProfile`. New AI fields follow this exactly.
- `internal/profiles/handler.go`: paired `GetAliases`/`PutAliases`, `GetTriggers`/`PutTriggers`, `GetEnvironment`/`PutEnvironment`, `GetTimers`/`PutTimers`, plus `getProfileByConnectionID`, `validateUpdate`, `ValidationError`, `sendJSON`, `sendError`. The AI sub-resource is one more pair.
- `cmd/server/main.go`: route registration under `/api/v1/profiles/{connection_id}/{aliases|triggers|environment|timers}` with method switching inside the handler func; auth middleware wraps them.
- `internal/help/handler.go`: loads content files at startup and serves them as JSON; the model for serving the policy text (embedding rather than reading the filesystem is the discretion choice).
- `migrations/008_add_automation.up.sql`, `009_add_timers.up.sql`: the `ALTER TABLE profiles ADD COLUMN IF NOT EXISTS ... JSONB NOT NULL DEFAULT` pattern; next number is `010`.
- `frontend/src/pages/SettingsPage.tsx`: `SettingsSection` union and the section list (`general | keybindings | aliases | triggers | timers | environment`); each section renders its own editor and saves via `frontend/src/services/api.ts` functions like `getAliases`/`putAliases`.
- `frontend/src/types/index.ts`: `Profile`, `ProfileSettings`, `UpdateProfileRequest`; add the AI settings and acceptance status types here.

### Established Patterns
- Profiles are 1:1 with saved connections and addressed by `connection_id` in sub-resource routes.
- Validation lives server-side in the handler with specific messages; the frontend shows the returned error string.
- Defaults are resolved in Go, not in the browser; blank JSONB fields are normalized on read.
- No test scaffolding exists beyond `internal/icm/icm_test.go`; Phase 1 tests for default resolution and the gate will be the first store/handler tests.

### Integration Points
- New routes in `cmd/server/main.go` next to the existing profile sub-resources.
- New `SettingsSection` value and nav entry in `SettingsPage.tsx`; new API functions in `api.ts`.
- The engage-gate check must be callable from Go (for Phase 2's server-side `#AUTO ON` handling) and over HTTP (for the browser and for this phase's diagnostic verification).

</code_context>

<specifics>
## Specific Ideas

- Owner's words on defaults: "no cap, but error handling is in place that needs to fail with an informative error, but still allow regular play. It can't CRASH."
- Owner's words on acceptance: "it pops up the first time you ever try to activate AI configurations within the MUD that you're on, you accept it and then you move on... it doesn't expire... if it changes, it doesn't make you re-check... stored at the profile level... you're only asked once per game."
- The owner does not want implementation plumbing (file locations, version parsing, audit tables) turned into decisions; pick the simplest thing and move on.

</specifics>

<deferred>
## Deferred Ideas

- A diagnostic `#AI STATUS` directive reporting engaged state, call count, gate result, and last decision. Useful from Phase 2 onward; noted for the Phase 2 discussion.
- Policy section 7 enforcement (disabling AI features after a violation) stays v2 (ENF-01).

</deferred>

---

*Phase: 01-profile-foundation-and-policy-gate*
*Context gathered: 2026-09-15*
