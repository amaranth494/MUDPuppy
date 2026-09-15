# Phase 1: Profile Foundation and Policy Gate - Research

**Researched:** 2026-09-15
**Domain:** Go backend (stdlib `database/sql` + `lib/pq` + `golang-migrate`), REST sub-resource endpoints, React/TypeScript settings UI — brownfield extension of an existing pattern
**Confidence:** HIGH (this phase is almost entirely "do exactly what the last four sub-resources did"; the only genuinely new mechanism is `go:embed`, which is stdlib and stable)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Where the owner edits and accepts**
- **D-01:** AI configuration lives in a new **AI Player** section of the existing Settings page, beside Key Bindings, Aliases, Triggers, and Timers. The older ProfileModal is untouched.
- **D-02:** The section is reached per connection profile and uses the same per-connection GET/PUT sub-resource pattern as aliases, triggers, environment, and timers.
- **D-03:** The policy is presented inside the AI Player section. On a profile with no acceptance, opening the section shows the full policy text, its version, and a single **Accept** button, and nothing else is editable until accepted. After acceptance the section shows "Accepted v1.0 on <date>" and the settings editor.
- **D-04:** Accepting is a single button press. Accepting the policy text is the owner's confirmation under policy section 1; no extra checkbox, no scroll-to-end requirement, nothing else recorded.
- **D-05:** A refused engage (Phase 2) shows a clear message and links the owner to the AI Player section for that profile.

**Policy acceptance record and lifetime**
- **D-06:** Acceptance is **one-time per profile** (per MUD connection). It does not expire. A later change to the policy text or version does **not** require re-acceptance. Deleting the profile discards the acceptance, so a new profile for the same game is asked once again.
- **D-07:** Acceptance is stored as columns on the `profiles` row: the policy version string accepted and an accepted-at timestamp. No history table.
- **D-08:** The engage gate is a server-side check on those columns: accepted means engageable. The gate is the only thing Phase 2 needs from this phase.

**AI settings shape and blank behaviour**
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

### Deferred Ideas (OUT OF SCOPE)
- A diagnostic `#AI STATUS` directive reporting engaged state, call count, gate result, and last decision. Useful from Phase 2 onward; noted for the Phase 2 discussion.
- Policy section 7 enforcement (disabling AI features after a violation) stays v2 (ENF-01).
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| REQ-profile-ai-fields | The connection profile stores conduct rules (text, handed to the model), approach guidance (text), and AI settings (model name, call cap per session, disengage threshold). Reconnect is not an AI setting. Blank model name = server default; blank call cap = no cap; blank disengage threshold = engine's built-in error handling (informative error, disengage, no crash, regular play continues). | `Pattern 2` (JSONB/TEXT column scan/marshal extension) and `Pattern 4` (pure-function `ResolveDisengageThreshold`, no store-layer normalization of blank values — see `Anti-Patterns to Avoid`) give the exact extension points in `internal/store/profile.go`. `Code Examples` gives the `ProfileSettings`/`DefaultProfileSettings` template and its deliberate deviation for `AISettings`. `Pitfall 1` and `Pitfall 3` cover the two concrete ways this requirement's "store and return" guarantee could silently break. |
| REQ-policy-gate | The Safety and Abuse policy must be presented and accepted before AI settings can be edited or the AI engaged; engagement on a profile without recorded acceptance is refused with a clear message. Acceptance is recorded once per profile with timestamp and version; never expires, no re-acceptance on policy change, discarded on profile delete. | `Pattern 3` (`go:embed` policy delivery, exact header line and version-parsing approach, verified) and `Pattern 4` (`EngageGateAllowed` pure function) give the mechanism. `Anti-Patterns to Avoid` explicitly forbids version-comparison re-acceptance logic and a history table, matching D-06/D-07. Profile-delete-discards-acceptance is satisfied for free by `DeleteProfile`'s plain row `DELETE` (existing code, verified) since acceptance lives as columns on that same row — no cascade or extra cleanup logic needed. `Security Domain` covers gate-bypass and IDOR threats specific to this requirement. |
</phase_requirements>

## Summary

This phase adds no new external dependency and no new architectural pattern. It is a fifth instance of a pattern the codebase already implements four times over (aliases, triggers, environment, timers): a `profiles` table gains new JSONB/text columns, `internal/store/profile.go` gains matching struct fields and scan/marshal lines, `internal/profiles/handler.go` gains a paired `Get*/Put*` handler using the existing `getProfileByConnectionID` helper, `cmd/server/main.go` registers one more `mux.HandleFunc("/api/v1/profiles/{connection_id}/...")` block with the same method-switch shape, and `frontend/src/pages/SettingsPage.tsx` gains one more section entry. The two things that are new in kind rather than degree are (1) `go:embed` for the policy markdown — nothing in this repo uses `go:embed` yet, so the pattern must be introduced from scratch — and (2) a pure-function "engage gate" and "resolve blank AI settings" design, because this repo has almost no test scaffolding and the phase's diagnostic verification is `go test`.

**Primary recommendation:** Follow the Timers sub-resource pattern exactly for the new `ai-settings` sub-resource (store field additions, handler pair, route registration, frontend section); introduce a new `internal/policy` package that embeds a copy of the policy markdown via `go:embed` and exposes a parsed version string; keep the engage-gate decision and the blank-settings resolution as pure functions over plain Go values (no `*sql.DB` argument) so `go test ./...` can exercise them with zero test infrastructure, matching the only existing test file's style (`internal/icm/icm_test.go`, plain `testing`, no mocking library).

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Store conduct rules / approach guidance / AI settings | API / Backend (`internal/store`) | Database / Storage (`profiles` table) | Same tier that already owns `Aliases`, `Triggers`, `Timers` on the same row |
| Resolve blank AI settings to conservative defaults | API / Backend (pure Go function) | — | D-explicit: "Defaults are resolved in Go, not in the browser" (CONTEXT.md Established Patterns) |
| Serve policy text + version | API / Backend (`go:embed` + one handler) | — | Mirrors `internal/help` (server loads content, serves JSON), but embedded not filesystem-read, per CONTEXT.md discretion |
| Record policy acceptance | API / Backend (columns on `profiles` row) | Database / Storage | One-time write, no history table (D-07) |
| Engage-gate decision | API / Backend (pure function + HTTP GET) | — | Must be callable from Go directly (Phase 2's `#AUTO ON` handler) and over HTTP (browser + diagnostic) |
| Present policy / editor / accept button | Browser / Client (`SettingsPage.tsx` new section) | — | Matches existing AI Player-adjacent sections (Aliases, Triggers, Timers, Environment) which are all client-rendered forms over the same sub-resource pattern |
| Refusal message surface | Browser / Client (text only, no logic) | API / Backend (source of the exact string) | Phase 1 only guarantees the server returns the exact string (D-05); Phase 2 renders it |

## Standard Stack

No new packages are introduced by this phase. `go.mod` (module `github.com/amaranth494/MudPuppy`, Go 1.26) already contains everything needed:

| Library | Version (from go.mod) | Purpose | Why Standard |
|---------|------|---------|--------------|
| `database/sql` + `github.com/lib/pq` | `v1.11.2` [VERIFIED: go.mod] | Postgres access | Already the only DB driver in the codebase; no `pgx` present |
| `github.com/golang-migrate/migrate/v4` | `v4.18.0` [VERIFIED: go.mod] | Schema migrations | Already runs at server startup (`cmd/server/main.go:85-94`) |
| `github.com/google/uuid` | `v1.6.0` [VERIFIED: go.mod] | Profile/connection IDs | Already used throughout `store`/`profiles` |
| `embed` (stdlib) | Go 1.26 stdlib [VERIFIED: pkg.go.dev/embed via WebSearch] | Embed policy markdown into the binary | Zero-dependency, exactly what CONTEXT.md discretion specifies (`go:embed`) |
| `net/http` stdlib `ServeMux` | Go 1.26 stdlib | Route registration | Already the only router in the codebase — no `chi`/`gorilla/mux` for HTTP routing (gorilla is only used for websockets: `github.com/gorilla/websocket v1.5.3`) |

No new frontend packages either — `frontend/package.json` [VERIFIED: codebase] has no test framework, no component library, no HTTP client beyond the browser `fetch` used by `frontend/src/services/api.ts`.

**Installation:** none required.

## Package Legitimacy Audit

**Not applicable.** This phase installs zero new packages in either `go.mod` or `frontend/package.json`. The slopcheck/registry-verification gate is skipped per its own instructions ("Required whenever this phase installs external packages").

## Architecture Patterns

### System Architecture Diagram

```
Browser (SettingsPage.tsx, new "AI Player" section)
   │
   │ GET  /api/v1/profiles/{connection_id}/policy          (policy text + version + acceptance status)
   │ POST /api/v1/profiles/{connection_id}/policy/accept    (record acceptance, once)
   │ GET  /api/v1/profiles/{connection_id}/ai-settings      (conduct rules, approach guidance, ai_settings)
   │ PUT  /api/v1/profiles/{connection_id}/ai-settings
   │ GET  /api/v1/profiles/{connection_id}/engage-gate      (diagnostic + browser refusal-message source)
   ▼
sessionMiddleware (cmd/server/main.go) — requires session cookie for all /api/v1/* except /register,/send-otp,/login,/health
   ▼
profiles.Handler (internal/profiles/handler.go) — new Get/Put pair + 2 new small handlers
   │  reuses getProfileByConnectionID(r) → (userUUID, *store.Profile, error)
   ▼
store.ProfileStore (internal/store/profile.go)
   │  GetProfileByConnection / UpdateProfile — extended with new columns
   │  new pure functions: ResolveAISettings(...), EngageGateAllowed(...)
   ▼
Postgres `profiles` table (migration 010: conduct_rules TEXT, approach_guidance TEXT,
   ai_settings JSONB, policy_version_accepted TEXT, policy_accepted_at TIMESTAMPTZ NULL)

Separately, at process start:
internal/policy package (new) — go:embed safety-and-abuse-policy-v1.md → parses
"Policy version: X.Y" header line once → held in memory, served by the /policy GET handler
and written verbatim into policy_version_accepted on /policy/accept.
```

### Recommended Project Structure

```
internal/
├── policy/                    # NEW — embeds and parses the policy markdown
│   ├── policy.go              # go:embed directive, ParseVersion(), GetText()
│   └── safety-and-abuse-policy-v1.md   # COPY of .specify/specs/safety-and-abuse-policy-v1.md
├── store/
│   └── profile.go             # extended: AISettings struct, new Profile fields, gate/resolve funcs
└── profiles/
    └── handler.go              # extended: GetAISettings/PutAISettings, GetPolicy, AcceptPolicy, GetEngageGate

migrations/
├── 010_add_ai_fields.up.sql
└── 010_add_ai_fields.down.sql

frontend/src/
├── components/
│   └── AIPlayerPanel.tsx       # NEW — mirrors EnvironmentPanel.tsx's shape (own file + own load/save state)
├── pages/
│   └── SettingsPage.tsx        # extended: SECTIONS entry, renders <AIPlayerPanel connectionId=.../>
├── services/
│   └── api.ts                  # extended: getPolicy, acceptPolicy, getAISettings, putAISettings, getEngageGate
└── types/
    └── index.ts                 # extended: AISettings, PolicyStatus, EngageGateResult, Profile fields
```

**Why a standalone `AIPlayerPanel.tsx` rather than inlining into `SettingsPage.tsx`:** `EnvironmentPanel.tsx` [VERIFIED: codebase, `frontend/src/components/EnvironmentPanel.tsx`] is the one existing sub-resource section that already lives in its own component file (aliases/triggers/timers are inlined at 1692 lines total inside `SettingsPage.tsx`). Given this phase's section has two states (pre-acceptance gate vs. post-acceptance editor) plus a markdown-rendering policy panel, following `EnvironmentPanel.tsx`'s pattern (own `useState`/`useEffect`/`loadX`/`handleSave`, receiving `connectionId` as a prop) keeps `SettingsPage.tsx` from growing further and is the less-invasive option the UI-SPEC already permits at "Claude's Discretion" level (exact class name, textarea height, etc. are explicitly left open — component file boundary is a natural extension of that same discretion).

### Pattern 1: Sub-resource GET/PUT pair (existing, to be replicated)

**What:** Every profile-scoped resource other than the top-level profile itself (`keybindings`, `settings`) is exposed as `/api/v1/profiles/{connection_id}/<name>` with a `GET` returning the current value and a `PUT` replacing it wholesale, both going through `getProfileByConnectionID(r)` for auth+lookup and `profileStore.UpdateProfile(userUUID, profile.ID, updates)` for the write.

**When to use:** For `ai-settings` (conduct_rules + approach_guidance + ai_settings together, since the UI-SPEC shows them saved by a single "Save AI Settings" button).

**Example (existing code, Timers — the template to copy):**
```go
// Source: internal/profiles/handler.go (verbatim, GetTimers/PutTimers)
func (h *Handler) GetTimers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	_, profile, err := h.getProfileByConnectionID(r)
	if err != nil {
		h.sendError(w, err.Error())
		return
	}
	h.sendJSON(w, TimersResponse{Items: profile.Timers.Items})
}

func (h *Handler) PutTimers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userUUID, profile, err := h.getProfileByConnectionID(r)
	if err != nil {
		h.sendError(w, err.Error())
		return
	}
	var req TimersResponse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid request body")
		return
	}
	// ... validation ...
	updates := &store.ProfileUpdate{Timers: &store.Timers{Items: req.Items}}
	updatedProfile, err := h.profileStore.UpdateProfile(userUUID, profile.ID, updates)
	// ... error handling ...
	h.sendJSON(w, TimersResponse{Items: updatedProfile.Timers.Items})
}
```

**Route registration (existing code, to be copied for `ai-settings`, `policy`, and `engage-gate`):**
```go
// Source: cmd/server/main.go:319-327 (verbatim)
mux.HandleFunc("/api/v1/profiles/{connection_id}/timers", func(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		profilesHandler.GetTimers(w, r)
	case http.MethodPut:
		profilesHandler.PutTimers(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
})
```
Note: despite the `{connection_id}` syntax (Go 1.22+ `ServeMux` named-parameter pattern matching), the existing handlers do **not** use `r.PathValue("connection_id")` — they re-parse the path manually via `getConnectionIDFromPath` (`strings.Split(r.URL.Path, "/")`, taking `parts[4]`). Keep using that helper for consistency; do not introduce `r.PathValue` only for the new routes, since `getProfileByConnectionID` already calls the string-split helper internally and both must agree.

The path-split helper only requires `len(parts) >= 5` and reads `parts[4]`, so nested paths like `/api/v1/profiles/{connection_id}/policy/accept` (6 parts) work unmodified with the same helper — verified by reading `getConnectionIDFromPath` (`internal/profiles/handler.go`):
```go
// Source: internal/profiles/handler.go (verbatim)
func (h *Handler) getConnectionIDFromPath(r *http.Request) (uuid.UUID, error) {
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 5 {
		return uuid.Nil, &ValidationError{Message: "Connection ID not found"}
	}
	return uuid.Parse(parts[4])
}
```

### Pattern 2: JSONB column scan/marshal in `store.ProfileStore` (existing, to be replicated)

**What:** Every JSONB column is fetched as `[]byte` in `GetProfile`/`GetProfileByConnection`, then `json.Unmarshal`'d into the typed struct field; every write in `UpdateProfile` does the reverse with `json.Marshal`, falling back to marshaling the *existing* value when the update pointer is `nil` (partial-update semantics — a `PUT` to one sub-resource does not clobber others, because `UpdateProfile` always re-marshals every column, existing or new, on every write).

**Example (existing code, verbatim from `internal/store/profile.go`):**
```go
var profile Profile
var keybindingsJSON, settingsJSON, aliasesJSON, triggersJSON, variablesJSON, timersJSON []byte
err := s.db.QueryRow(query, profileID, userID).Scan(
    &profile.ID, &profile.UserID, &profile.ConnectionID,
    &keybindingsJSON, &settingsJSON, &aliasesJSON, &triggersJSON, &variablesJSON, &timersJSON,
    &profile.CreatedAt, &profile.UpdatedAt,
)
// ...
if err := json.Unmarshal(timersJSON, &profile.Timers); err != nil {
    return nil, err
}
```
**Applies to:** `ai_settings` (JSONB → `AISettings` struct). `conduct_rules` and `approach_guidance` are plain `TEXT` columns per the roadmap's implementation notes ("profiles gain conduct_rules, approach_guidance, ai_settings JSONB..." — note only `ai_settings` is called out as JSONB), so those two scan directly into `*string`, no `json.Unmarshal` needed. `policy_version_accepted` (TEXT) likewise scans directly into `*string`. `policy_accepted_at` is the one genuinely new scan shape in this codebase: every existing timestamp column (`created_at`, `updated_at`) is `NOT NULL DEFAULT NOW()` and scans into a plain `string` field; this phase's `policy_accepted_at` must be **nullable** (unset until acceptance) — recommend scanning into `sql.NullString` (simplest, matches the existing convention of representing timestamps as strings in the `Profile` struct) rather than introducing `time.Time`/`sql.NullTime`, which would be the first use of the `time` package in this file.

### Pattern 3: `go:embed` for the policy markdown (new to this codebase)

**What:** `grep -rn "go:embed"` across the repo returns **zero matches** [VERIFIED: ripgrep search, this session] — `internal/help/handler.go` instead reads `./help/*.json` from the filesystem at startup via `os.ReadDir`/`os.ReadFile` with a `helpDir` constructor argument. This is a real behavioral difference the planner should be aware of: help content can be edited on disk without a rebuild; embedded policy text cannot (rebuild required to change it, but re-acceptance is explicitly never required (D-06), so this is a non-issue for this phase).

**Constraint (why the file must be copied, not referenced in place):** `go:embed` patterns are resolved relative to the directory containing the `.go` file with the directive, and **cannot** contain `..` or reach outside that directory's subtree — confirmed via the official `embed` package docs: "Patterns must not match files outside the package's module... contain '.' or '..' path elements" [CITED: pkg.go.dev/embed]. The canonical source `.specify/specs/safety-and-abuse-policy-v1.md` lives outside any Go package directory (`.specify/` is not under `internal/` or `cmd/`), so it **must be copied** into a new package directory, e.g. `internal/policy/safety-and-abuse-policy-v1.md`, for `//go:embed safety-and-abuse-policy-v1.md` to reach it.

**Header line to parse (verbatim, confirmed by reading the file):**
```
Policy version: 1.0
```
This is line 3 of `.specify/specs/safety-and-abuse-policy-v1.md` [VERIFIED: file read this session]. A simple line-prefix scan (`strings.HasPrefix(line, "Policy version:")` then `strings.TrimSpace(strings.TrimPrefix(...))`) extracts `"1.0"` deterministically; no markdown/YAML frontmatter parser is needed or justified for one header line.

**Recommended shape:**
```go
// internal/policy/policy.go (new file, sketch — not existing code)
package policy

import (
	_ "embed"
	"strings"
)

//go:embed safety-and-abuse-policy-v1.md
var policyText string

var policyVersion = parseVersion(policyText)

func parseVersion(text string) string {
	for _, line := range strings.Split(text, "\n") {
		if v, found := strings.CutPrefix(strings.TrimSpace(line), "Policy version:"); found {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func Text() string    { return policyText }
func Version() string { return policyVersion }
```
Parsing once at package init (via a package-level `var`) matches the existing "load once at startup" shape used by `internal/help` (loaded in `NewHandler`, held in a map for the process lifetime) — no per-request file I/O either way.

### Pattern 4: Pure-function gate and default resolution (new to this codebase, required for testability)

**What:** This repo has one test file total, `internal/icm/icm_test.go` [VERIFIED: `find`/`ls` this session — no other `_test.go` files exist], and it tests pure functions with plain `testing.T` table tests, no mocking library, no `sqlmock`/`pgxmock` dependency in `go.mod`, no DB driver other than `lib/pq` (`database/sql`, not `pgx`, which rules out `pgxmock` even if it were added). The Phase Validation line requires `go test` to cover "default resolution for blank AI settings and the engage-gate decision (not accepted, accepted)" **without** a live Postgres being implied as a prerequisite for that specific test run (the separate "database inspection" diagnostic is what proves the migration itself).

**Recommendation:** Write both as pure functions taking only plain Go values (no `*sql.DB`, no `*store.ProfileStore` receiver) so they need zero fixtures:
```go
// internal/store/profile.go additions (sketch — not existing code)

// DefaultDisengageThreshold is the number of consecutive transient AI failures
// tolerated before disengage, used when a profile's disengage_threshold is blank.
const DefaultDisengageThreshold = 3

// ResolveDisengageThreshold returns the effective threshold: the profile's
// explicit value if set, otherwise the conservative default.
func ResolveDisengageThreshold(s AISettings) int {
	if s.DisengageThreshold != nil {
		return *s.DisengageThreshold
	}
	return DefaultDisengageThreshold
}

// EngageGateAllowed reports whether AI engagement may proceed for a profile,
// based solely on its recorded policy acceptance. Callable from Go directly
// (Phase 2's #AUTO ON handler) with no HTTP or DB dependency.
func EngageGateAllowed(policyVersionAccepted string, policyAcceptedAt *string) bool {
	return policyVersionAccepted != "" && policyAcceptedAt != nil
}
```
`go test ./internal/store/...` then exercises `ResolveDisengageThreshold` (blank → 3, set → the set value) and `EngageGateAllowed` (both columns empty/nil → false, both populated → true) with plain table tests, matching `icm_test.go`'s style, and with **zero** new test infrastructure — directly satisfying the roadmap's implementation note that "conservative defaults live in Go" and the Phase Validation's `go test` diagnostic, without requiring the sqlmock/pgxmock investigation the task brief anticipated as a fallback (it is not needed; the gate is naturally expressible as a pure function over two already-fetched column values).

### Anti-Patterns to Avoid

- **Normalizing blank AI settings at the store layer the way `normalizeSettings` does for `ProfileSettings`:** `normalizeSettings` (existing code) silently overwrites a zero `ScrollbackLimit` with `1000` on every read — appropriate there because "blank" has no meaning distinct from "default" for scrollback. For AI settings, blank is a **first-class, user-visible, round-tripping value** ("no cap", "server default", "engine default") that success criterion 1 requires the profile to "store and return" as blank — do not silently rewrite it to a concrete number at read time. Resolution (`ResolveDisengageThreshold` etc.) must be a separate, explicitly-called function used only where the *engine* needs a concrete number (Phase 4's loop), never inside `GetProfile`/`GetProfileByConnection`.
- **Adding a version-comparison / re-acceptance check anywhere:** D-06 and the policy file's own amended intro both explicitly forbid this ("a later policy change does not require re-acceptance... nothing compares versions afterwards" — CONTEXT.md Claude's Discretion). The engage gate checks only "is `policy_accepted_at` set", never "does `policy_version_accepted` match the current embedded version".
- **Building an acceptance history table:** D-07 is explicit — "columns on the `profiles` row... No history table."

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Loading a markdown file into the binary | A custom file-embedding build step or a runtime file-read-and-cache layer | `//go:embed` (stdlib, Go 1.16+) | Zero dependencies, compile-time safety (missing file = build error, not a runtime 404), already the discretion-chosen approach in CONTEXT.md |
| Nullable timestamp scanning | A hand-rolled `[]byte`-to-`*string` conversion with manual NULL checks | `database/sql.NullString` (stdlib) | Standard idiom for a nullable text/timestamp-as-text column; matches this repo's existing convention of representing `created_at`/`updated_at` as plain strings rather than `time.Time` |
| Markdown-to-HTML rendering for the policy panel | A markdown library dependency | Reuse `HelpPage.tsx`'s existing `renderContent`/`renderInline` paragraph/bold/list-splitting logic (already hand-rolled, already in the codebase, explicitly named as reusable in the UI-SPEC's Claude's Discretion section) | Introducing a markdown-parser npm package for one policy document (7 short sections, no tables, no nested lists) is disproportionate; the existing hand-rolled renderer already covers the policy text's actual markdown surface (headings, bold, bullet lists) |

**Key insight:** every "hand-roll or not" question in this phase resolves to "there's already exactly one file in this codebase that solved a smaller version of this problem — extend it, don't add a library."

## Common Pitfalls

### Pitfall 1: Treating `UpdateProfile`'s partial-update semantics as automatic for brand-new columns
**What goes wrong:** `UpdateProfile` (existing code) marshals *every* JSONB column on every write, falling back to the existing value when the corresponding `ProfileUpdate` pointer is `nil`. If the new `AISettings`/`ConductRules`/`ApproachGuidance`/policy columns are added to `Profile` and `ProfileUpdate` but the fallback branches (`if updates.X != nil { ... } else { ...marshal existing.X... }`) are forgotten in `UpdateProfile`, a `PUT /timers` call (unrelated to AI settings) would silently wipe the AI fields back to their zero value on every unrelated save, because the SQL `UPDATE` statement sets *all* columns in one statement.
**Why it happens:** `UpdateProfile`'s query is a single `UPDATE ... SET col1=$1, col2=$2, ...` covering every mutable column, not a dynamic partial-column `UPDATE`. Every new column added to the table must also be added to this one `UPDATE` statement and its existing-value fallback.
**How to avoid:** When extending `UpdateProfile`, add the new columns to the `SET` list, the `QueryRow` argument list, and both the `if updates.X != nil` and `else` branches, exactly mirroring how `timersJSON` was added in the same function for the previous sub-resource.
**Warning signs:** After implementing `ai-settings`, manually `PUT /timers` (or any other existing sub-resource) on a profile that already has AI settings saved, then re-`GET /ai-settings` — if it comes back blank, the fallback branch was missed.

### Pitfall 2: Forgetting the `010` migration must also patch the startup fallback pattern
**What goes wrong:** `cmd/server/main.go` (existing code, lines ~100-105) has a **second**, redundant `ALTER TABLE profiles ADD COLUMN IF NOT EXISTS timers ...` executed unconditionally at startup, commented "fallback for when migration 009 isn't in migrate.zip" — implying the deployed environment (Railway) may run from a bundled `migrate.zip` that can lag behind the `migrations/` directory in the repo. If a `010` migration is added to `migrations/` but this startup fallback pattern is not extended (or the deploy process's `migrate.zip` is not confirmed to include `010_*.sql`), the new columns could be present in one environment and absent in another.
**Why it happens:** The comment implies a known, already-worked-around deployment quirk, not a hypothetical.
**How to avoid:** Either (a) add the same defensive `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` fallback lines for the five new columns right after the `010` migration file, following the existing timers precedent exactly, or (b) confirm with the owner/CI that `migrate.zip` is rebuilt from `migrations/` on every deploy and the fallback is no longer needed. This is a **flag for validation**, not a settled fact — no build script that produces `migrate.zip` was found in this repo (`grep -rl "migrate.zip"` found only the one `main.go` comment referencing it) [LOW confidence — recommend the planner add a task to verify the Railway deploy process directly rather than assume].
**Warning signs:** Migration `010` runs clean locally but the staging startup log inspection (the phase's own diagnostic) shows the columns missing.

### Pitfall 3: Scanning `policy_accepted_at` as a plain (non-nullable) `string` like `created_at`
**What goes wrong:** Every existing timestamp field on `Profile` (`CreatedAt`, `UpdatedAt`) is `NOT NULL DEFAULT NOW()` and scans directly into a `string`. `policy_accepted_at` is the first nullable timestamp column on this table. Scanning a SQL `NULL` into a Go `*string` destination via `Scan(&profile.PolicyAcceptedAt)` where the field is a plain `string` will return a scan error ("converting NULL to string is unsupported") for every profile that hasn't accepted yet — i.e., every freshly migrated or freshly created profile, which is also exactly the state the phase's own Phase Validation walkthrough starts from ("open the AI Player section on a fresh profile").
**Why it happens:** Copy-pasting the `CreatedAt`/`UpdatedAt` scan pattern without noticing they're non-nullable by design (`DEFAULT NOW()`), whereas acceptance is deliberately absent until the owner clicks Accept.
**How to avoid:** Use `sql.NullString` (or `sql.NullTime`) as the intermediate scan target for `policy_accepted_at`, then convert to a `*string` (or `*time.Time`) on the `Profile` struct so `nil` cleanly represents "not accepted", matching how the frontend needs to distinguish "not accepted" (show policy) from "accepted" (show editor).
**Warning signs:** `GetProfileByConnection` returns a scan error only for profiles that have never accepted — i.e., it looks like it "works in testing" if the developer's test profile happens to have already accepted, and fails specifically for the fresh-profile case the phase is supposed to prove.

## Code Examples

### Existing `ProfileSettings`/`DefaultProfileSettings` pattern (verbatim, the template for `AISettings`)
```go
// Source: internal/store/profile.go
type ProfileSettings struct {
	ScrollbackLimit   int  `json:"scrollback_limit"`
	EchoInput         bool `json:"echo_input"`
	TimestampOutput   bool `json:"timestamp_output"`
	WordWrap          bool `json:"word_wrap"`
	AutomationEnabled bool `json:"automation_enabled"`
}

func DefaultProfileSettings() ProfileSettings {
	return ProfileSettings{
		ScrollbackLimit:   1000,
		EchoInput:         false,
		TimestampOutput:   false,
		WordWrap:          true,
		AutomationEnabled: true,
	}
}
```
**Deliberate deviation for `AISettings`:** unlike `ProfileSettings`, do **not** write a `DefaultAISettings()`/`normalizeSettings`-style function that fills in concrete values on read — see "Anti-Patterns to Avoid" above. `AISettings`'s zero value (`ModelName: ""`, `CallCap: nil`, `DisengageThreshold: nil`) **is** the correct, meaningful "blank" state and must round-trip as-is.

### Existing frontend fetch wrapper pattern (verbatim, the template for `getAISettings`/`putAISettings`/`getPolicy`/`acceptPolicy`/`getEngageGate`)
```typescript
// Source: frontend/src/services/api.ts (getTimers/putTimers, verbatim)
export async function getTimers(connectionId: string): Promise<TimersResponse> {
  const response = await fetch(`${API_BASE}/profiles/${connectionId}/timers`, {
    credentials: 'include',
  });
  handleAuthError(response);
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || 'Failed to get timers');
  }
  return await response.json();
}

export async function putTimers(connectionId: string, items: Timer[]): Promise<TimersResponse> {
  const response = await fetch(`${API_BASE}/profiles/${connectionId}/timers`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ items }),
  });
  handleAuthError(response);
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || 'Failed to update timers');
  }
  return await response.json();
}
```
The JSON error shape every handler returns is `{"error": "<message>"}` [VERIFIED: `internal/profiles/handler.go`, `ErrorResponse` struct + `sendError`], with HTTP status `400 Bad Request` for **every** error path including "not found" and "unauthorized" (`sendError` always calls `w.WriteHeader(http.StatusBadRequest)`) [VERIFIED: `internal/profiles/handler.go`, `sendError`] — this is a real existing inconsistency (404-shaped errors return HTTP 400), not something to "fix" in this phase; the frontend already only reads `data.error` and ignores the status code beyond `response.ok`, so new endpoints should follow the same `sendError` convention for consistency with every other sub-resource, not introduce differentiated status codes.

### Existing `EnvironmentPanel.tsx` load/save/error state shape (verbatim shape, template for `AIPlayerPanel.tsx`)
```typescript
// Source: frontend/src/components/EnvironmentPanel.tsx (verbatim shape)
const [isLoading, setIsLoading] = useState(true);
const [isSaving, setIsSaving] = useState(false);
const [error, setError] = useState<string | null>(null);
const [successMessage, setSuccessMessage] = useState<string | null>(null);

const loadX = useCallback(async () => {
  if (!connectionId) return;
  setIsLoading(true);
  setError(null);
  try {
    const data = await getX(connectionId);
    setX(data);
  } catch (err) {
    setError(err instanceof Error ? err.message : 'Failed to load X');
  } finally {
    setIsLoading(false);
  }
}, [connectionId]);
```
Error/success message markup: `<div className="message message-error">{error}<button onClick={() => setError(null)}>Dismiss</button></div>` and `<div className="message message-success">{successMessage}</div>` — exactly the classes the UI-SPEC's copywriting contract specifies for `Failed to load AI settings` / `Failed to save AI settings` / `Failed to load policy` / `Failed to record acceptance`.

## State of the Art

Not applicable in the "library version drift" sense — there is no external library whose API changed. The one relevant "state of the art" fact is that Go's `net/http.ServeMux` has supported method- and wildcard-aware routing patterns (`"GET /path/{id}/sub"`) natively since Go 1.22, and this codebase (Go 1.26) already uses the `{connection_id}` wildcard syntax in its route strings — but, as noted in Pattern 1, does not use the paired `r.PathValue()` extraction API, instead re-parsing the raw path manually. This is an existing inconsistency, not something introduced by this phase; new routes should match the existing manual-parse convention rather than mixing styles within `internal/profiles/handler.go`.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `conduct_rules` and `approach_guidance` should be plain `TEXT` columns (not JSONB-wrapped) | Standard Stack / Pattern 2 | Low — inferred directly from the roadmap implementation note's phrasing ("conduct_rules, approach_guidance, ai_settings JSONB" — only the third is tagged JSONB), but not stated as an explicit CONTEXT.md decision. If wrong, only the migration SQL and one struct tag change; no behavioral impact. |
| A2 | Default numeric disengage threshold = 3 consecutive transient failures | Pattern 4 / Code Examples | Low — CONTEXT.md explicitly delegates this exact number to Claude's Discretion ("a small count of consecutive transient failures"); 3 is a reasonable, arbitrary choice within that delegated range. No owner-visible behavior depends on the exact number in Phase 1 (the loop that uses it is Phase 4 per CONTEXT.md). |
| A3 | `policy_accepted_at` should scan via `sql.NullString`/`*string` rather than introducing `time.Time` into `internal/store/profile.go` | Pitfall 3 | Low-Medium — this is a genuine design choice, not a verified fact. Using `*string` matches the existing all-strings convention for timestamps on `Profile` but means the acceptance date must be string-parsed if it's ever needed as a real `time.Time` for comparison logic later. If Phase 2+ needs to do time arithmetic on `policy_accepted_at`, `*time.Time`/`sql.NullTime` would have been the better choice; a follow-up type change is low-cost since nothing depends on the acceptance timestamp's type across phase boundaries per the gate design (Pattern 4 only checks non-nil, never compares times). |
| A4 | Railway staging's deploy process rebuilds/repackages `migrate.zip` (or otherwise makes `migrations/010_*.sql` available) on every deploy, so the startup fallback pattern in `main.go` (Pitfall 2) is optional rather than required | Pitfall 2 | Medium — if false, migration 010 could silently not run on staging even though it runs locally. No Dockerfile, docker-compose, or Railway config file exists in this repo to confirm the deploy mechanism [VERIFIED: repo search found none]; deployment is inferred to be Railway's Nixpacks auto-detection of `go.mod`/`package.json`, not directly confirmed. Recommend the planner add an explicit staging-verification task (the phase's own "database inspection of a migrated profile" diagnostic already covers this if run against staging, not just local). |

## Open Questions (RESOLVED)

1. **[RESOLVED — settled empirically by checkpoint task 01-05-02, which captures the staging startup log showing `Migrations completed successfully (version=10` and `AI Player columns ensured`; no SQL is run] Does the deployed (Railway staging) process definitely run `migrate.Up()` from the same `migrations/` directory checked into this branch, or from a separately-packaged `migrate.zip` that could lag behind?**
   - What we know: `cmd/server/main.go` calls `migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)` at every startup [VERIFIED: `cmd/server/main.go:85-94`], which — if the working directory at runtime contains an up-to-date `migrations/` folder — would pick up `010_*.sql` automatically with no separate packaging step needed. The one comment referencing `migrate.zip` ("fallback for when migration 009 isn't in migrate.zip") is the only evidence of an alternate packaging path, and no script producing that zip was found in this repo.
   - What's unclear: whether `migrate.zip` is a leftover from a discontinued deploy mechanism (in which case the comment and its fallback `ALTER TABLE` line are dead-code-adjacent and can be ignored) or an active part of the current Railway build (in which case `010`'s columns need the same fallback treatment).
   - Recommendation: the plan's verification step should explicitly include deploying to Railway staging and capturing its startup log (not just a local run), which will settle this empirically regardless of which mechanism is actually active.

2. **[RESOLVED — locked at 20000 / 20000 / 200 characters in task 01-03-01] Exact validation limits for `conduct_rules`/`approach_guidance` text length and `model_name` length.**
   - What we know: CONTEXT.md explicitly delegates this to Claude's Discretion, "following the existing `validateUpdate` style." Existing precedent caps are small (keybinding commands: 500 chars; alias/trigger/timer counts: 50-200 items) because those are short structured fields, not free text meant to be "handed to the model verbatim."
   - What's unclear: there is no existing precedent in this codebase for a large free-text field, so any number is a genuine judgment call, not an inference from existing patterns.
   - Recommendation: a generous but bounded limit (e.g., 20,000 characters each for conduct_rules/approach_guidance — enough for substantial free text, small enough to bound request/row size; 200 characters for model_name, matching typical model identifier lengths) is a reasonable default for the planner to lock in; this is squarely inside the discretion CONTEXT.md already granted and does not need owner confirmation per the "no speed bumps" project instruction.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | Building/testing the whole backend | ✓ | go1.26.0 windows/amd64 [VERIFIED: `go version`, this session] | — |
| `migrate.exe` (golang-migrate CLI) | Local migration testing (`npm run migrate:up`) | ✓ | Binary present at repo root, `migrate.exe` [VERIFIED: file listing, this session] | — |
| Local Postgres / `DATABASE_URL` | Running the server locally | ✗ | — | No `.env`/`.env.example` found in repo; `DATABASE_URL` must be sourced from Railway staging or a locally-started Postgres neither of which is configured in-repo. The phase's "database inspection of a migrated profile" diagnostic should target Railway staging (per CLAUDE.md: "Environment: Railway project `mudpuppy`, environment `staging`") |
| Docker | N/A this phase | ✗ | — | Not used by this repo at all — no Dockerfile found; Railway deployment appears to be Nixpacks-based auto-detection, not confirmed further (see Open Question 1) |
| `psql` CLI | Manual DB inspection convenience | ✗ | — | Use `migrate.exe`, a one-off Go diagnostic, or Railway's own DB console/CLI instead |

**Missing dependencies with no fallback:** none — the phase's diagnostic (`go test`) needs no external service, and the "database inspection" diagnostic can run against Railway staging via the Railway CLI/console even without a local Postgres.

**Missing dependencies with fallback:** local Postgres (fallback: use Railway staging directly, which is the project's stated environment per CLAUDE.md anyway).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` [VERIFIED: only existing test file, `internal/icm/icm_test.go`, imports `"testing"` and nothing else test-related] |
| Config file | none — no `pytest.ini`/`jest.config`/`vitest.config` equivalent exists for Go; `go test` needs none |
| Quick run command | `go test ./internal/store/... ./internal/profiles/... -v` |
| Full suite command | `go test ./...` (this is exactly the command CI already runs — `.github/workflows/ci.yml`, step "Run backend tests" [VERIFIED: `.github/workflows/ci.yml`], currently with `continue-on-error: true`, i.e. CI does not yet gate merges on test success) |

**Frontend test tooling:** none exists. `frontend/package.json` [VERIFIED: read this session] has no `vitest`/`jest`/`@testing-library/*` dependency and no `test` script; the root `package.json`'s `"test": "echo 'No tests yet' && exit 0"` [VERIFIED: read this session] confirms this is a known, accepted gap, not an oversight to fix in this phase. Frontend verification for this phase is therefore player-observable browser walkthrough only, exactly as the roadmap's Phase Validation line specifies ("open the AI Player section on a fresh profile...").

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| REQ-profile-ai-fields | Blank model name / call cap / disengage threshold resolve to documented defaults; non-blank values round-trip unchanged | unit | `go test ./internal/store/... -run TestResolveDisengageThreshold -v` | ❌ Wave 0 |
| REQ-profile-ai-fields | Profile stores and returns conduct_rules/approach_guidance/ai_settings across a `PUT` then `GET` | unit (handler-level, no live DB — see below) or integration | `go test ./internal/profiles/... -run TestAISettingsRoundTrip -v` | ❌ Wave 0 |
| REQ-profile-ai-fields | New columns present on a migrated `profiles` row | diagnostic (staging startup log) | capture `Migrations completed successfully (version=10` and `AI Player columns ensured` from the Railway staging log into `evidence/02-staging-startup.log` | n/a (log evidence; DB queries are not accepted as evidence) |
| REQ-policy-gate | `EngageGateAllowed` returns false when `policy_accepted_at` is nil/empty, true when both columns are populated | unit | `go test ./internal/store/... -run TestEngageGateAllowed -v` | ❌ Wave 0 |
| REQ-policy-gate | Policy text served with the correct parsed version (`"1.0"`) | unit | `go test ./internal/policy/... -run TestParseVersion -v` | ❌ Wave 0 |
| REQ-policy-gate | Fresh profile: opening AI Player shows policy first, not editor; accept records acceptance; reload persists; profile delete+recreate re-asks | player-observable (browser) | manual walkthrough against a live connection, per roadmap's Phase Validation line | n/a |
| REQ-policy-gate | `GET .../engage-gate` on an un-accepted profile returns the refusal message; on accepted profile it passes | diagnostic (HTTP) | manual `curl`/browser devtools against the running server, or a `go test` using `httptest.NewRecorder()` against the handler directly | ❌ Wave 0 (optional `httptest`-based handler test) |

**Note on "unit test without a live Postgres" for the handler-level round-trip:** `store.ProfileStore` methods take a real `*sql.DB` and there is no interface/mock boundary in front of it today, and no `sqlmock`/`pgxmock` dependency exists in `go.mod` [VERIFIED: go.mod contents, this session]. Two genuinely test-without-DB options exist: (a) test only the pure functions (`ResolveDisengageThreshold`, `EngageGateAllowed`, `policy.ParseVersion`) which need no DB at all and fully cover the Phase Validation's stated `go test` diagnostic ("default resolution for blank AI settings and the engage-gate decision"), or (b) introduce a `ProfileStore` interface (e.g. `type ProfileReader interface { GetProfileByConnection(...) (*Profile, error) }`) purely for handler-level testing via a hand-written fake struct (no new dependency, ~10 lines). Recommendation: (a) alone satisfies the Phase Validation line as written; (b) is optional polish the planner may add as a stretch task, not required for the phase to be provable.

### Sampling Rate
- **Per task commit:** `go test ./internal/store/... ./internal/policy/... -v` (the two new pure-function packages)
- **Per wave merge:** `go test ./...` (full existing suite, matches CI)
- **Phase gate:** Full suite green (or at minimum, no new failures beyond the pre-existing `continue-on-error: true` baseline) before `/gsd:verify-work`, plus the seven end-user screenshots and the staging log captures described in the roadmap's Phase Validation line (database queries are not accepted as evidence).

### Wave 0 Gaps
- [ ] `internal/store/profile_test.go` — new file; covers `ResolveDisengageThreshold` (blank → default, set → value) and `EngageGateAllowed` (both unaccepted/accepted combinations) — REQ-profile-ai-fields, REQ-policy-gate
- [ ] `internal/policy/policy_test.go` — new file; covers `ParseVersion` against the embedded text, asserting it returns `"1.0"` — REQ-policy-gate
- [ ] Framework install: none — `testing` is stdlib, already used by `internal/icm/icm_test.go`

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No (unchanged this phase) | Existing session-cookie middleware (`sessionMiddleware` in `cmd/server/main.go`) already wraps every new route under `/api/v1/*` |
| V3 Session Management | No (unchanged this phase) | Same existing middleware; no new session semantics introduced |
| V4 Access Control | Yes | Every new handler must call `getProfileByConnectionID(r)`, which enforces `user_id` scoping via `GetProfileByConnection(userUUID, connectionID)` — a profile can only be read/written by the user who owns it, exactly as every existing sub-resource already enforces [VERIFIED: `internal/profiles/handler.go`] |
| V5 Input Validation | Yes | New `PUT ai-settings` handler must validate: `model_name` length bound, `call_cap`/`disengage_threshold` non-negative when provided, `conduct_rules`/`approach_guidance` length bound — following `validateUpdate`'s existing style (return `*ValidationError` with a specific message, surfaced verbatim to the frontend) |
| V6 Cryptography | No | No secrets/credentials touched by this phase (API keys are Phase 3) |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| IDOR via `connection_id` — a user supplying another user's `connection_id` in the URL to read/write their AI settings or policy acceptance | Elevation of Privilege / Information Disclosure | Already mitigated by the existing pattern: `GetProfileByConnection(userUUID, connectionID)` filters by both columns in its `WHERE` clause (`WHERE connection_id = $1 AND user_id = $2` [VERIFIED: `internal/store/profile.go`]), so a mismatched owner simply gets "Profile not found," never another user's data. New handlers must go through `getProfileByConnectionID` exactly like the existing four, never construct their own ad hoc query. |
| Unbounded free-text storage (`conduct_rules`/`approach_guidance`) used for resource exhaustion or, later, prompt-injection surface once handed to Gemini in Phase 3 | Tampering / Denial of Service | Length-bound validation at the handler layer (see V5 above); prompt-injection mitigation itself is explicitly out of scope for Phase 1 (no model calls exist yet) but the length bound set here is a reasonable first control that Phase 3 inherits for free |
| Policy acceptance forgery/bypass (crafting a request that sets `policy_accepted_at` without going through the accept endpoint, or an engage-gate check that trusts a client-supplied "accepted" flag) | Tampering | The gate must only ever read `policy_version_accepted`/`policy_accepted_at` from the server-fetched `Profile` row, never accept these as client-supplied request fields — `UpdateProfileRequest`/`ProfileUpdate` for the `ai-settings` PUT must **not** include acceptance fields; only the dedicated `POST .../policy/accept` handler may write them, and it must derive the version from the server's embedded `policy.Version()`, never from the request body |

## Sources

### Primary (HIGH confidence)
- `internal/store/profile.go` — full file read, this session (struct definitions, scan/marshal pattern, `UpdateProfile`, `DeleteProfile`)
- `internal/profiles/handler.go` — full file read, this session (handler pairs, `getProfileByConnectionID`, `validateUpdate`, `sendJSON`/`sendError`, `ErrorResponse`)
- `cmd/server/main.go` — route registration block (lines ~156-345) and migration bootstrap (lines ~60-105) read this session
- `migrations/006_create_profiles.up.sql`, `008_add_automation.up.sql`, `009_add_timers.up.sql`/`.down.sql` — read verbatim this session
- `internal/help/handler.go` — full file read, this session (confirms no `go:embed` usage, filesystem-load pattern)
- `frontend/src/pages/SettingsPage.tsx`, `frontend/src/services/api.ts`, `frontend/src/types/index.ts`, `frontend/src/components/EnvironmentPanel.tsx` — read this session
- `frontend/package.json`, root `package.json`, `go.mod` — read this session (dependency inventory, confirms no test framework, no new packages needed)
- `.github/workflows/ci.yml` — read this session (confirms `go test ./...` and `npm test` are the CI commands, both `continue-on-error: true`)
- `.specify/specs/safety-and-abuse-policy-v1.md` — read this session (exact "Policy version: 1.0" header text, full policy structure)
- `pkg.go.dev/embed` (via WebSearch, this session) — `go:embed` cannot traverse `..` or reach outside the package directory [CITED: pkg.go.dev/embed]

### Secondary (MEDIUM confidence)
- none required — every substantive claim traced to a file read this session or to the official `embed` package documentation

### Tertiary (LOW confidence)
- The `migrate.zip` deployment mechanism (Open Question 1 / Pitfall 2) — only evidence is a single code comment in `main.go`; no build script producing it was found in this repo. Flagged explicitly for validation, not stated as settled fact.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies; every existing dependency and its version confirmed directly from `go.mod`/`package.json`
- Architecture: HIGH — this phase is a structural repeat of an existing, four-times-proven pattern; the one new mechanism (`go:embed`) is stdlib and its constraints are documented and verified
- Pitfalls: HIGH for Pitfalls 1 and 3 (directly observed in the existing code's structure); MEDIUM for Pitfall 2 (inferred from a single comment, explicitly flagged as needing validation, not asserted as fact)

**Research date:** 2026-09-15
**Valid until:** 30 days (stable brownfield codebase, no fast-moving external dependencies involved)
