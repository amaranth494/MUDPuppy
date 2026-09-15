# Phase 1: Profile Foundation and Policy Gate - Pattern Map

**Mapped:** 2026-09-15
**Files analyzed:** 11
**Analogs found:** 9 exact/role-match / 11 (2 have no direct precedent — `internal/policy` package and both `_test.go` handler-level files; concrete stdlib/idiom patterns given instead)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `migrations/010_add_ai_fields.up.sql` / `.down.sql` | migration | batch (schema DDL) | `migrations/009_add_timers.up.sql` / `.down.sql` | exact |
| `internal/store/profile.go` (modify) | model / store | CRUD | itself — timers/automation column handling already in this file | exact (self-extension) |
| `internal/store/profile_test.go` (new) | test | transform (pure function) | `internal/icm/icm_test.go` | role-match (only test file in repo) |
| `internal/policy/policy.go` + `internal/policy/safety-and-abuse-policy-v1.md` (new package) | service / config | file-I/O (embed, read once at init) → request-response | `internal/help/handler.go` (content-serving shape); no `go:embed` precedent anywhere in repo | partial — serving shape matches, embedding mechanism is new-in-kind |
| `internal/policy/policy_test.go` (new) | test | transform | `internal/icm/icm_test.go` | role-match |
| `internal/profiles/handler.go` (modify: `GetAISettings`/`PutAISettings`, `GetPolicy`, `AcceptPolicy`, `GetEngageGate`) | controller | request-response / CRUD | `GetTimers`/`PutTimers` in the same file | exact |
| `internal/profiles/handler_test.go` (new) | test | request-response | none in repo | no analog — stdlib `httptest` pattern given below |
| `cmd/server/main.go` (modify: route registration) | route / config | request-response | the `timers` `mux.HandleFunc` block (same file) | exact |
| `frontend/src/types/index.ts` (modify) | model (types) | transform | `Timer`/`ProfileSettings`/`TimersResponse` interfaces | exact |
| `frontend/src/services/api.ts` (modify) | service (API client) | request-response | `getTimers`/`putTimers` | exact |
| `frontend/src/pages/SettingsPage.tsx` (modify: `SECTIONS` entry + `activeSection === 'ai-player'` block) | component (page) | request-response | the `timers`/`environment` section entries + inline JSX blocks | exact |
| `frontend/src/components/AIPlayerPanel.tsx` (new) | component | request-response | `frontend/src/components/EnvironmentPanel.tsx` | role-match — only existing standalone (non-inlined) sub-resource panel in the codebase |

## Pattern Assignments

### `migrations/010_add_ai_fields.up.sql` / `.down.sql` (migration, batch)

**Analog:** `migrations/009_add_timers.up.sql` (verbatim, 4 lines) and `.down.sql`

```sql
-- Source: migrations/009_add_timers.up.sql (verbatim)
-- +migrate Up
-- Add timers column to profiles table for time-based automation
ALTER TABLE profiles
ADD COLUMN IF NOT EXISTS timers JSONB NOT NULL DEFAULT '{"items": []}'::jsonb;
```
```sql
-- Source: migrations/009_add_timers.down.sql (verbatim)
-- +migrate Down
-- Remove timers column from profiles table
ALTER TABLE profiles
DROP COLUMN IF EXISTS timers;
```

**Multi-column precedent** (`migrations/008_add_automation.up.sql`, verbatim — comma-separated `ADD COLUMN` list in one `ALTER TABLE`, the shape for `010`'s five new columns):
```sql
ALTER TABLE profiles
ADD COLUMN IF NOT EXISTS aliases JSONB NOT NULL DEFAULT '{"items": []}'::jsonb,
ADD COLUMN IF NOT EXISTS triggers JSONB NOT NULL DEFAULT '{"items": []}'::jsonb,
ADD COLUMN IF NOT EXISTS variables JSONB NOT NULL DEFAULT '{"items": []}'::jsonb;
```

Apply this shape to `010`: `conduct_rules TEXT`, `approach_guidance TEXT`, `ai_settings JSONB NOT NULL DEFAULT '{}'::jsonb` (blank/zero AISettings, per Anti-Pattern — do not default to concrete resolved values), `policy_version_accepted TEXT`, `policy_accepted_at TIMESTAMPTZ` (nullable — no `DEFAULT NOW()`, no `NOT NULL`; this is the deliberate deviation from every other column in this table). The `.down.sql` drops all five with `DROP COLUMN IF EXISTS`.

**Pitfall 2 (flagged, not settled):** `cmd/server/main.go` lines ~100-107 run a second, unconditional startup fallback `ALTER TABLE profiles ADD COLUMN IF NOT EXISTS timers ...` after `m.Up()`, commented "fallback for when migration 009 isn't in migrate.zip" — see excerpt under `cmd/server/main.go` below. Decide whether `010`'s columns need the same treatment; no build script producing `migrate.zip` was found in this repo.

---

### `internal/store/profile.go` (model/store, CRUD)

**Analog:** the file's own existing `Timers`/`ProfileSettings` handling — this is a self-extension, not a copy from elsewhere.

**Struct field pattern** (lines 10-23, `Profile` struct — add `ConductRules *string`, `ApproachGuidance *string`, `AISettings AISettings`, `PolicyVersionAccepted *string`, `PolicyAcceptedAt *string`):
```go
// Source: internal/store/profile.go:10-23 (verbatim)
type Profile struct {
	ID           uuid.UUID         `json:"id"`
	UserID       uuid.UUID         `json:"user_id"`
	ConnectionID uuid.UUID         `json:"connection_id"`
	Keybindings  map[string]string `json:"keybindings"`
	Settings     ProfileSettings   `json:"settings"`
	Aliases      Aliases           `json:"aliases"`
	Triggers     Triggers          `json:"triggers"`
	Variables    Variables         `json:"variables"`
	Timers       Timers            `json:"timers"`
	CreatedAt    string            `json:"created_at"`
	UpdatedAt    string            `json:"updated_at"`
}
```

**New struct template** (`ProfileSettings`/`DefaultProfileSettings`, lines 82-99 — the template for `AISettings`, but see the deliberate deviation below):
```go
// Source: internal/store/profile.go:82-99 (verbatim)
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
**Deliberate deviation:** do **not** write a `DefaultAISettings()`/`normalizeSettings`-equivalent that fills blank fields with concrete values on read (see Anti-Pattern below). `AISettings` zero value must round-trip as-is:
```go
type AISettings struct {
	ModelName          string `json:"model_name"`
	CallCap            *int   `json:"call_cap"`
	DisengageThreshold *int   `json:"disengage_threshold"`
}
```

**JSONB scan pattern** (`GetProfileByConnection`, lines 249-303, verbatim — this exact shape repeats 3x in the file: `GetProfile`, `GetProfileByConnection`, and implicitly `CreateProfile`'s insert):
```go
// Source: internal/store/profile.go:256-297 (verbatim)
var profile Profile
var keybindingsJSON, settingsJSON, aliasesJSON, triggersJSON, variablesJSON, timersJSON []byte

err := s.db.QueryRow(query, connectionID, userID).Scan(
	&profile.ID,
	&profile.UserID,
	&profile.ConnectionID,
	&keybindingsJSON,
	&settingsJSON,
	&aliasesJSON,
	&triggersJSON,
	&variablesJSON,
	&timersJSON,
	&profile.CreatedAt,
	&profile.UpdatedAt,
)
// ...
if err := json.Unmarshal(timersJSON, &profile.Timers); err != nil {
	return nil, err
}
```
Extend the `SELECT` column list and `Scan` args with `conduct_rules, approach_guidance, ai_settings, policy_version_accepted, policy_accepted_at` in both `GetProfile` and `GetProfileByConnection` (both must stay in sync — they currently duplicate the identical query/scan/unmarshal block). `conduct_rules`/`approach_guidance`/`policy_version_accepted` scan directly into `*string` (plain `TEXT`, no `json.Unmarshal`); `ai_settings` scans into `[]byte` then `json.Unmarshal`s like `timersJSON`.

**Pitfall 3 — nullable timestamp, the one genuinely new scan shape:** every existing timestamp (`CreatedAt`, `UpdatedAt`) is `NOT NULL DEFAULT NOW()` and scans into a plain `string`. `policy_accepted_at` is nullable. Use `sql.NullString` as the intermediate scan target, convert to `*string` on the struct:
```go
var policyAcceptedAtNS sql.NullString
// ... add &policyAcceptedAtNS to the Scan(...) call ...
if policyAcceptedAtNS.Valid {
	profile.PolicyAcceptedAt = &policyAcceptedAtNS.String
} else {
	profile.PolicyAcceptedAt = nil
}
```
`database/sql` is already imported (line 4); no new import needed.

**`UpdateProfile` partial-update fallback pattern** (lines 306-395, verbatim — the exact shape that must be replicated for every new column, this is Pitfall 1):
```go
// Source: internal/store/profile.go:354-358, 390-392 (verbatim, Timers as the template)
if updates.Timers != nil {
	timersJSON, _ = json.Marshal(*updates.Timers)
} else {
	timersJSON, _ = json.Marshal(existing.Timers)
}
// ...
query := `
	UPDATE profiles
	SET keybindings = $1, settings = $2, aliases = $3, triggers = $4, variables = $5, timers = $6, updated_at = NOW()
	WHERE id = $7 AND user_id = $8
	RETURNING updated_at
`
// ...
if updates.Timers != nil {
	existing.Timers = *updates.Timers
}
```
**Pitfall 1 (critical):** `UpdateProfile` marshals *every* column on *every* write via a single `UPDATE ... SET col1=$1, col2=$2, ...` statement (not a dynamic partial-column update). Every new field added to `Profile`/`ProfileUpdate` must get: (a) a `SET` clause entry, (b) a `QueryRow` arg, (c) an `if updates.X != nil { marshal updates.X } else { marshal/pass existing.X }` branch, (d) an `if updates.X != nil { existing.X = *updates.X }` branch in the return path. Miss any one and an unrelated `PUT /timers` silently wipes AI fields back to zero value. `ConductRules`/`ApproachGuidance`/`PolicyVersionAccepted`/`PolicyAcceptedAt` (all `*string`, nilable) need the same else-branch treatment but marshal as plain values, not JSON — pass the `*string` (or dereferenced value) directly as a `sql.DB` positional arg, no `json.Marshal` needed for `TEXT` columns.

**`ProfileUpdate` struct extension point** (lines 116-123, verbatim):
```go
type ProfileUpdate struct {
	Keybindings *map[string]string `json:"keybindings,omitempty"`
	Settings    *ProfileSettings   `json:"settings,omitempty"`
	Aliases     *Aliases           `json:"aliases,omitempty"`
	Triggers    *Triggers          `json:"triggers,omitempty"`
	Variables   *Variables         `json:"variables,omitempty"`
	Timers      *Timers            `json:"timers,omitempty"`
}
```
Add `ConductRules *string`, `ApproachGuidance *string`, `AISettings *AISettings` here. **Security note:** do NOT add `PolicyVersionAccepted`/`PolicyAcceptedAt` to `ProfileUpdate` — per RESEARCH's Security Domain, acceptance may only be written by a dedicated `AcceptPolicy` store method that derives the version from `policy.Version()` server-side, never from client-supplied `ProfileUpdate` JSON (prevents acceptance forgery).

**New pure functions (no existing precedent, sketch per RESEARCH Pattern 4 — place in this same file, no `*sql.DB` receiver so they're independently unit-testable):**
```go
const DefaultDisengageThreshold = 3

func ResolveDisengageThreshold(s AISettings) int {
	if s.DisengageThreshold != nil {
		return *s.DisengageThreshold
	}
	return DefaultDisengageThreshold
}

func EngageGateAllowed(policyVersionAccepted *string, policyAcceptedAt *string) bool {
	return policyVersionAccepted != nil && *policyVersionAccepted != "" && policyAcceptedAt != nil
}
```

**Anti-pattern to avoid (`normalizeSettings`, lines 101-113, verbatim — do NOT copy this shape for AI settings):**
```go
// Source: internal/store/profile.go:101-113 (verbatim) — DO NOT replicate for AISettings
func normalizeSettings(s ProfileSettings) ProfileSettings {
	if s.ScrollbackLimit == 0 {
		s.ScrollbackLimit = DefaultProfileSettings().ScrollbackLimit
	}
	// ...
	return s
}
```
This silently overwrites blank with a concrete default at read time — correct for `ScrollbackLimit` (no meaningful "blank"), wrong for `AISettings` (blank is a first-class, round-tripping, user-visible state per D-09/D-10/D-11). `ResolveDisengageThreshold` must be called explicitly only where the engine needs a concrete number (Phase 4), never inside `GetProfile`/`GetProfileByConnection`.

---

### `internal/store/profile_test.go` (test, new)

**Analog:** `internal/icm/icm_test.go` (only existing test file, plain `testing`, table-driven, no mocking library)

```go
// Source: internal/icm/icm_test.go:1-40 (verbatim shape)
package icm

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestRecognizer_RecognizeStructured(t *testing.T) {
	r := NewRecognizer()

	tests := []struct {
		input    string
		wantOp   OperatorFamily
		wantCmd  string
		wantArgs []string
	}{
		{"#echo hello", OperatorStructured, "ECHO", []string{"hello"}},
		// ...
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := r.Recognize(tt.input)
			if err != nil {
				t.Errorf("Recognize(%q) error = %v", tt.input, err)
				return
			}
			if got.Operator != tt.wantOp {
				t.Errorf("Recognize(%q).Operator = %v, want %v", tt.input, got.Operator, tt.wantOp)
			}
		})
	}
}
```
Apply this exact shape (`package store`, plain `testing`, table of struct literals, `t.Run` subtests) to `TestResolveDisengageThreshold` (nil → 3, set pointer → dereferenced value) and `TestEngageGateAllowed` (both nil/empty → false; both populated → true; only one populated → false). No `*sql.DB`, no fixtures — pure function tests only, per RESEARCH Pattern 4.

---

### `internal/policy/policy.go` + `safety-and-abuse-policy-v1.md` (service/config, new package)

**Analog for "serve loaded content" shape:** `internal/help/handler.go` (full file read, 160 lines) — filesystem-load-at-construction pattern, the closest existing precedent even though the embedding mechanism itself is new:

```go
// Source: internal/help/handler.go:33-52 (verbatim shape — load-once-at-construction)
type Handler struct {
	helpDir  string
	sections map[string]HelpSection
}

func NewHandler(helpDir string) *Handler {
	h := &Handler{
		helpDir:  helpDir,
		sections: make(map[string]HelpSection),
	}
	if err := h.loadHelpContent(); err != nil {
		log.Printf("Warning: Failed to load help content: %v", err)
	}
	return h
}
```
```go
// Source: internal/help/handler.go:99-116 (verbatim — GET handler shape: build response struct, set Content-Type, json.Encode, log on encode failure)
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	summaries := make([]HelpSummary, 0, len(h.sections))
	// ...
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(summaries); err != nil {
		log.Printf("Error encoding help summaries: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
```

**`go:embed` mechanism (no precedent in repo — `grep -rn "go:embed"` returns zero matches; this is the canonical stdlib pattern, not existing code):**
```go
// internal/policy/policy.go (new file)
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
**Constraint:** `go:embed` cannot traverse `..` or reach outside the package directory [pkg.go.dev/embed]. The canonical source `.specify/specs/safety-and-abuse-policy-v1.md` is outside any Go package — copy it verbatim into `internal/policy/safety-and-abuse-policy-v1.md`.

**Header line to parse (verbatim, line 3 of the source file):**
```
Policy version: 1.0
```
`parseVersion` must return exactly `"1.0"`.

**`internal/policy/policy_test.go`:** same table-test shape as `internal/store/profile_test.go` above — one test asserting `Version() == "1.0"` against the real embedded text (no fixtures needed, it's compiled in).

---

### `internal/profiles/handler.go` (controller, request-response — modify)

**Analog:** `GetTimers`/`PutTimers` (same file, lines 420-485, verbatim — the exact pair-shape to replicate for `GetAISettings`/`PutAISettings`):
```go
// Source: internal/profiles/handler.go:420-434 (verbatim)
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
```
```go
// Source: internal/profiles/handler.go:436-485 (verbatim, PUT half — validate, build store.ProfileUpdate, call UpdateProfile, respond)
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

	if len(req.Items) > 50 {
		h.sendError(w, "Maximum 50 timers allowed")
		return
	}
	for _, timer := range req.Items {
		if strings.TrimSpace(timer.Name) == "" {
			h.sendError(w, "Timer name cannot be empty")
			return
		}
		if timer.Duration <= 0 {
			h.sendError(w, "Timer duration must be greater than 0")
			return
		}
	}

	updates := &store.ProfileUpdate{Timers: &store.Timers{Items: req.Items}}
	updatedProfile, err := h.profileStore.UpdateProfile(userUUID, profile.ID, updates)
	if err != nil {
		log.Printf("[PR01PH06] Update timers failed: %v", err)
		h.sendError(w, "Failed to update timers")
		return
	}

	h.sendJSON(w, TimersResponse{Items: updatedProfile.Timers.Items})
}
```
`GetAISettings`/`PutAISettings` follow this exactly, bundling `ConductRules`, `ApproachGuidance`, and `AISettings` in one request/response struct (UI-SPEC: single "Save AI Settings" button saves all three together).

**`GetPolicy` (new small handler — combine `internal/policy.Text()`/`.Version()` with acceptance status read off the profile):**
```go
// shape: GET-only handler, getProfileByConnectionID for auth, then sendJSON — no PUT counterpart
func (h *Handler) GetPolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	_, profile, err := h.getProfileByConnectionID(r)
	if err != nil {
		h.sendError(w, err.Error())
		return
	}
	h.sendJSON(w, PolicyResponse{
		Text:         policy.Text(),
		Version:      policy.Version(),
		Accepted:     profile.PolicyAcceptedAt != nil,
		AcceptedAt:   profile.PolicyAcceptedAt,
		AcceptedVersion: profile.PolicyVersionAccepted,
	})
}
```

**`AcceptPolicy` (new — `POST`, not GET/PUT; writes acceptance columns only, never from client body, per Security Domain):**
```go
func (h *Handler) AcceptPolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userUUID, profile, err := h.getProfileByConnectionID(r)
	if err != nil {
		h.sendError(w, err.Error())
		return
	}
	updatedProfile, err := h.profileStore.AcceptPolicy(userUUID, profile.ID, policy.Version())
	if err != nil {
		log.Printf("Accept policy failed: %v", err)
		h.sendError(w, "Failed to record acceptance")
		return
	}
	h.sendJSON(w, PolicyResponse{ /* ... */ })
}
```
(`store.AcceptPolicy` is a small dedicated method, not routed through the general `UpdateProfile`/`ProfileUpdate` path, precisely so acceptance columns can never be set via the AI-settings PUT body.)

**`GetEngageGate` (new — GET, diagnostic + browser refusal-message source, calls the pure `store.EngageGateAllowed` function):**
```go
func (h *Handler) GetEngageGate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	_, profile, err := h.getProfileByConnectionID(r)
	if err != nil {
		h.sendError(w, err.Error())
		return
	}
	allowed := store.EngageGateAllowed(profile.PolicyVersionAccepted, profile.PolicyAcceptedAt)
	resp := EngageGateResponse{Allowed: allowed}
	if !allowed {
		resp.Message = "AI Player has not been configured for this connection. Accept the Safety and Abuse policy in AI Player settings before engaging autopilot."
	}
	h.sendJSON(w, resp)
}
```
(Refusal string copied verbatim from `01-UI-SPEC.md` Copywriting Contract — this is the exact string Phase 2 must surface for `#AUTO ON`.)

**Auth/scoping pattern reused unchanged, all new handlers must call this exact helper (lines 487-514, verbatim — do not construct ad hoc queries, per Security Domain IDOR mitigation):**
```go
// Source: internal/profiles/handler.go:487-514 (verbatim)
func (h *Handler) getProfileByConnectionID(r *http.Request) (uuid.UUID, *store.Profile, error) {
	userID := r.Context().Value("user_id")
	if userID == nil {
		return uuid.Nil, nil, &ValidationError{Message: "Unauthorized"}
	}
	userIDStr := userID.(string)
	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, nil, &ValidationError{Message: "Invalid user ID"}
	}

	connectionID, err := h.getConnectionIDFromPath(r)
	if err != nil {
		return uuid.Nil, nil, err
	}

	profile, err := h.profileStore.GetProfileByConnection(userUUID, connectionID)
	if err != nil {
		log.Printf("[SP05PH01T07] Get profile by connection failed: %v", err)
		return uuid.Nil, nil, &ValidationError{Message: "Failed to get profile"}
	}
	if profile == nil {
		return uuid.Nil, nil, &ValidationError{Message: "Profile not found"}
	}

	return userUUID, profile, nil
}
```

**Path-parsing note:** `getConnectionIDFromPath` (lines 518-525) only requires `len(parts) >= 5` and reads `parts[4]`, so nested paths like `/api/v1/profiles/{connection_id}/policy/accept` (6 segments) work unmodified with the same helper — do not introduce `r.PathValue()` for the new routes; the repo's existing routes deliberately re-parse the raw path instead.

**`sendJSON`/`sendError`/`ErrorResponse` (lines 599-612, verbatim — reuse unchanged, do not create new response helpers):**
```go
func (h *Handler) sendJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("[SP04PH02T03] Failed to encode JSON: %v", err)
	}
}

func (h *Handler) sendError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}
```
Note: every error path returns HTTP 400 (including "not found"/"unauthorized") — a known existing inconsistency, not to be "fixed" here; the frontend only reads `data.error`.

**`validateUpdate` style (lines 528-566, verbatim — the pattern for new `validateAISettings`/length-bound checks, e.g. 20,000 chars for conduct_rules/approach_guidance, 200 chars for model_name per RESEARCH Open Question 2):**
```go
// Source: internal/profiles/handler.go:555-563 (verbatim)
if req.Settings != nil {
	settings := *req.Settings
	if settings.ScrollbackLimit < 100 || settings.ScrollbackLimit > 10000 {
		return &ValidationError{Message: "Scrollback limit must be between 100 and 10000"}
	}
}
```

---

### `internal/profiles/handler_test.go` (test, new — no existing precedent)

**No analog exists** (`grep` confirms zero `_test.go` files under `internal/profiles/`). Canonical stdlib `net/http/httptest` pattern against the handler's existing constructor (`NewHandler(profileStore)`, line 20 of `handler.go`) — RESEARCH's option (b), optional stretch, not required to satisfy Phase Validation:
```go
package profiles

import (
	"net/http/httptest"
	"testing"
)

func TestGetEngageGate_NotAccepted(t *testing.T) {
	h := NewHandler(fakeStore /* hand-written fake satisfying only the methods used, ~10 lines, no mocking library */)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/profiles/"+connID+"/engage-gate", nil)
	req = req.WithContext(context.WithValue(req.Context(), "user_id", userID.String()))
	rec := httptest.NewRecorder()
	h.GetEngageGate(rec, req)
	// assert rec.Code, decode rec.Body into EngageGateResponse
}
```
Since `store.ProfileStore` has no interface boundary today, this requires either (a) skip this test file and rely on the pure-function tests in `profile_test.go` to satisfy the Phase Validation `go test` line (sufficient per RESEARCH), or (b) introduce a small `ProfileReader`/`ProfileStore`-shaped interface purely for this test. Not required for phase completion.

---

### `cmd/server/main.go` (route/config — modify)

**Route registration analog** (lines 319-328, verbatim, the `timers` block — the exact shape to copy for `ai-settings`, `policy`, `policy/accept`, `engage-gate`):
```go
// Source: cmd/server/main.go:319-328 (verbatim)
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
Register 4 new blocks after it: `.../ai-settings` (GET/PUT), `.../policy` (GET), `.../policy/accept` (POST only in the switch), `.../engage-gate` (GET only). All sit inside `protectedHandler := sessionMiddleware(redisClient, mux)` (line 361) automatically — no separate auth wiring needed, the whole `mux` is wrapped once.

**Startup fallback pattern (Pitfall 2 — decide whether to replicate, verbatim existing code, lines 100-107):**
```go
// Source: cmd/server/main.go:100-107 (verbatim)
// Ensure timers column exists (fallback for when migration 009 isn't in migrate.zip)
log.Println("Ensuring timers column exists...")
_, err = db.Exec(`ALTER TABLE profiles ADD COLUMN IF NOT EXISTS timers JSONB NOT NULL DEFAULT '{"items": []}'`)
if err != nil {
	log.Printf("Warning: Failed to ensure timers column: %v", err)
} else {
	log.Println("Timers column ensured")
}
```
If replicated for `010`'s five columns, this runs unconditionally at every startup right after `m.Up()` (line 94-98).

---

### `frontend/src/types/index.ts` (model/types — modify)

**Analog** (`Timer`/`TimersResponse`, lines 205-227, and `ProfileSettings`/`Profile`, lines 133-169, verbatim):
```typescript
// Source: frontend/src/types/index.ts:205-227 (verbatim)
export interface Timer {
  id: string;
  name: string;
  duration: number; // in milliseconds
  repeat: boolean;
  commands: string;
  enabled: boolean;
}

export interface TimersResponse {
  items: Timer[];
}
```
```typescript
// Source: frontend/src/types/index.ts:143-154, 166-169 (verbatim — Profile and UpdateProfileRequest do NOT carry sub-resource fields inline; aliases/triggers/variables are optional and separately fetched)
export interface Profile {
  id: string;
  user_id: string;
  connection_id: string;
  keybindings: Record<string, string>;
  settings: ProfileSettings;
  aliases?: AutomationAliases;
  triggers?: AutomationTriggers;
  variables?: AutomationVariables;
  created_at: string;
  updated_at: string;
}

export interface UpdateProfileRequest {
  keybindings?: Record<string, string>;
  settings?: ProfileSettings;
}
```
Add new interfaces following the `Timer`/`TimersResponse` shape: `AISettings` (`model_name: string`, `call_cap: number | null`, `disengage_threshold: number | null`), `AISettingsResponse` (`conduct_rules: string`, `approach_guidance: string`, `ai_settings: AISettings`), `PolicyResponse` (`text: string`, `version: string`, `accepted: boolean`, `accepted_at: string | null`, `accepted_version: string | null`), `EngageGateResponse` (`allowed: boolean`, `message?: string`). Do **not** add acceptance fields to `Profile`/`UpdateProfileRequest` — they're sub-resource-only, matching how `Timers` itself is never in the top-level `Profile` interface either.

---

### `frontend/src/services/api.ts` (service/API client — modify)

**Analog** (`getTimers`/`putTimers`, lines 558-584, verbatim):
```typescript
// Source: frontend/src/services/api.ts:558-568 (verbatim)
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
```
```typescript
// Source: frontend/src/services/api.ts:571-584 (verbatim, PUT shape)
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
`API_BASE = '/api/v1'` (line 4); `handleAuthError` (lines 7-14) redirects to `/login` on 401, reused unchanged. Add `getAISettings`/`putAISettings`, `getPolicy`/`acceptPolicy` (POST, no body needed beyond credentials), `getEngageGate` following this exact shape — error message strings match the UI-SPEC Copywriting Contract (`'Failed to load AI settings'`, `'Failed to save AI settings'`, `'Failed to load policy'`, `'Failed to record acceptance'`).

---

### `frontend/src/pages/SettingsPage.tsx` (component/page — modify)

**`SECTIONS` array analog** (lines 14-29, verbatim):
```typescript
// Source: frontend/src/pages/SettingsPage.tsx:14-29 (verbatim)
type SettingsSection = 'general' | 'keybindings' | 'aliases' | 'triggers' | 'timers' | 'environment';

interface Section {
  id: SettingsSection;
  label: string;
  icon: string;
}

const SECTIONS: Section[] = [
  { id: 'general', label: 'General', icon: '⚙' },
  { id: 'keybindings', label: 'Key Bindings', icon: '⌨' },
  { id: 'aliases', label: 'Aliases', icon: '⚡' },
  { id: 'triggers', label: 'Triggers', icon: '⚓' },
  { id: 'timers', label: 'Timers', icon: '⏱' },
  { id: 'environment', label: 'Environment', icon: '📦' },
];
```
Extend the union with `'ai-player'` and append `{ id: 'ai-player', label: 'AI Player', icon: '🤖' }` to `SECTIONS` (per UI-SPEC, positioned after `timers`/`environment`).

**Nav rendering** (lines 977-986, verbatim — unchanged, iterates `SECTIONS` automatically, no per-section nav code needed):
```typescript
{SECTIONS.map(section => (
  <button
    key={section.id}
    className={`settings-nav-item ${activeSection === section.id ? 'active' : ''}`}
    onClick={() => handleSectionChange(section.id)}
  >
    <span className="settings-nav-icon">{section.icon}</span>
    <span className="settings-nav-label">{section.label}</span>
  </button>
))}
```

**Content rendering — every other section is inlined directly in `SettingsPage.tsx`'s JSX** (e.g. `activeSection === 'environment'` at line 1599), but per RESEARCH's structural recommendation (and matching `EnvironmentPanel.tsx`'s existence as the one precedent for a standalone panel), render the new section as a single delegated component rather than inlining ~150 lines of two-state (policy-gate vs. editor) JSX:
```typescript
{activeSection === 'ai-player' && (
  <AIPlayerPanel connectionId={selectedConnectionId} />
)}
```

---

### `frontend/src/components/AIPlayerPanel.tsx` (component, new)

**Analog:** `frontend/src/components/EnvironmentPanel.tsx` (full file, 288 lines) — the only existing standalone sub-resource panel component (aliases/triggers/timers are inlined in `SettingsPage.tsx`; `EnvironmentPanel.tsx` itself is currently unimported/unused elsewhere in the codebase but is still the correct structural template per RESEARCH).

**Load/save/error state shape** (lines 1-19, verbatim):
```typescript
// Source: frontend/src/components/EnvironmentPanel.tsx:1-19 (verbatim)
import { useState, useEffect, useCallback } from 'react';
import { Variable, VariableType } from '../types';
import { getEnvironment, putEnvironment } from '../services/api';

interface EnvironmentPanelProps {
  connectionId: string;
  isConnectedToThis: boolean;
  onVariablesChange?: (variables: Variable[]) => void;
}

export default function EnvironmentPanel({ connectionId, isConnectedToThis, onVariablesChange }: EnvironmentPanelProps) {
  const [variables, setVariables] = useState<Variable[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);
```

**Load pattern** (lines 54-74, verbatim):
```typescript
const loadVariables = useCallback(async () => {
  if (!connectionId) return;
  setIsLoading(true);
  setError(null);
  try {
    const data = await getEnvironment(connectionId);
    setVariables(data.items || []);
    if (onVariablesChange) onVariablesChange(data.items || []);
  } catch (err) {
    setError(err instanceof Error ? err.message : 'Failed to load variables');
  } finally {
    setIsLoading(false);
  }
}, [connectionId, onVariablesChange]);

useEffect(() => {
  loadVariables();
}, [loadVariables]);
```

**Save pattern** (lines 77-110, verbatim shape — `AIPlayerPanel` needs two independent loads/saves: policy status first, then, only if accepted, AI settings):
```typescript
const handleSave = async () => {
  if (!connectionId) return;
  setIsSaving(true);
  setError(null);
  setSuccessMessage(null);
  try {
    await putEnvironment(connectionId, variables);
    setSuccessMessage('Variables saved successfully');
    await loadVariables();
  } catch (err) {
    setError(err instanceof Error ? err.message : 'Failed to save variables');
  } finally {
    setIsSaving(false);
  }
};
```

**Error/success message markup** (lines 177-189, verbatim — reuse exactly, matches UI-SPEC's copywriting contract classes):
```typescript
{error && (
  <div className="message message-error" style={{ marginBottom: '1rem' }}>
    {error}
    <button onClick={() => setError(null)} style={{ marginLeft: '1rem' }}>Dismiss</button>
  </div>
)}
{successMessage && (
  <div className="message message-success" style={{ marginBottom: '1rem' }}>
    {successMessage}
  </div>
)}
```

**Save button pattern** (lines 271-285, verbatim — `isSaving` disables + relabels):
```typescript
<div className="settings-actions">
  <button className="btn btn-primary" onClick={handleSave} disabled={isSaving}>
    {isSaving ? 'Saving...' : 'Save Variables'}
  </button>
</div>
```
For `AIPlayerPanel`: `{isSaving ? 'Saving...' : 'Save AI Settings'}` per UI-SPEC Copywriting Contract; a second, separate `.btn.btn-primary` "Accept Policy" button with its own `isAccepting` state for the pre-acceptance gate state.

**Markdown rendering for the policy text (State 1):** reuse `renderContent`/`renderInline` logic shape from `frontend/src/pages/HelpPage.tsx` (lines 126-219 and 222+), restyled with CSS custom properties instead of that file's hardcoded inline hex colors:
```typescript
// Source: frontend/src/pages/HelpPage.tsx:126-159 (shape, verbatim except style values — re-theme rgba/hex to var(--color-text)/var(--color-text-dim))
const renderContent = (content: string) => {
  const paragraphs = content.split('\n\n');
  return paragraphs.map((para, idx) => {
    if (para.startsWith('**') && para.includes(':**')) {
      const parts = para.split(':');
      const title = parts[0].replace(/\*\*/g, '');
      const rest = parts.slice(1).join(':');
      return (
        <div key={idx} style={{ marginBottom: '1rem' }}>
          <h4 style={{ marginBottom: '0.5rem', opacity: 0.9, fontWeight: 600 }}>{title}</h4>
          <div style={{ opacity: 0.8 }}>{renderInline(rest)}</div>
        </div>
      );
    }
    if (para.includes('\n- ') || para.startsWith('- ')) {
      const items = para.split('\n').map((line) => line.replace(/^-\s*/, ''));
      return (
        <ul key={idx} style={{ marginBottom: '1rem', paddingLeft: '1.5rem', opacity: 0.8 }}>
          {items.map((item, i) => <li key={i} style={{ marginBottom: '0.25rem' }}>{renderInline(item)}</li>)}
        </ul>
      );
    }
    return <p key={idx} style={{ marginBottom: '1rem', opacity: 0.8, lineHeight: 1.6 }}>{renderInline(para)}</p>;
  });
};
```
The policy markdown has no tables, so the table-handling branch (`HelpPage.tsx` lines 162-210) can be omitted.

## Shared Patterns

### Auth/scoping (IDOR mitigation)
**Source:** `internal/profiles/handler.go:487-514`, `getProfileByConnectionID`
**Apply to:** every new handler (`GetAISettings`, `PutAISettings`, `GetPolicy`, `AcceptPolicy`, `GetEngageGate`) — never construct an ad hoc query; always resolve `(userUUID, *store.Profile, error)` through this one helper, which enforces `WHERE connection_id = $1 AND user_id = $2` in `GetProfileByConnection`.

### Error/success response shape
**Source:** `internal/profiles/handler.go:599-612` (`sendJSON`/`sendError`/`ErrorResponse`)
**Apply to:** all new handlers — `{"error": "<message>"}` body, HTTP 400 for every error path (existing inconsistency, not to be changed).

### Partial-update-via-full-marshal
**Source:** `internal/store/profile.go:306-395` (`UpdateProfile`)
**Apply to:** every new `*string`/`AISettings` field added to `Profile`/`ProfileUpdate` — must appear in the `SET` list, `QueryRow` args, the `if updates.X != nil {...} else {...}` marshal branch, and the post-write `if updates.X != nil { existing.X = *updates.X }` branch. This is Pitfall 1 and is the single highest-risk copy-paste point in the whole phase.

### Frontend fetch wrapper
**Source:** `frontend/src/services/api.ts:558-584` (`getTimers`/`putTimers`), `handleAuthError` at lines 7-14
**Apply to:** all new `api.ts` functions — `credentials: 'include'`, `handleAuthError(response)` before the `response.ok` check, `throw new Error(data.error || '<fallback message>')` on failure.

### Load/save state shape for standalone panels
**Source:** `frontend/src/components/EnvironmentPanel.tsx:1-19, 54-110, 177-189, 271-285`
**Apply to:** `AIPlayerPanel.tsx` — `isLoading`/`isSaving`/`error`/`successMessage` state, `useCallback` load function + `useEffect`, try/catch/finally save handler, identical `.message.message-error`/`.message.message-success` markup with Dismiss button.

## No Analog Found

| File | Role | Data Flow | Reason |
|---|---|---|---|
| `internal/policy/policy.go` (the `go:embed` directive itself) | service/config | file-I/O | `grep -rn "go:embed"` returns zero matches anywhere in the repo — this is the first use of the stdlib `embed` package. `internal/help/handler.go` is the closest analog for the *serving* shape only (filesystem read at construction, not compile-time embed). The canonical stdlib pattern is given in full above; no further codebase search is useful. |
| `internal/profiles/handler_test.go` | test | request-response | No `_test.go` file exists anywhere under `internal/profiles/` or any other HTTP handler package in the repo (`internal/icm/icm_test.go` tests a pure parser, not an HTTP handler). The stdlib `httptest` pattern is given above; this file is optional (RESEARCH marks it stretch, not required for Phase Validation). |

## Metadata

**Analog search scope:** `internal/store/`, `internal/profiles/`, `internal/help/`, `internal/icm/`, `cmd/server/`, `migrations/`, `frontend/src/types/`, `frontend/src/services/`, `frontend/src/pages/`, `frontend/src/components/`
**Files scanned:** `internal/store/profile.go` (402 lines, full), `internal/profiles/handler.go` (625 lines, full), `cmd/server/main.go` (targeted: lines 60-110, 270-340), `internal/help/handler.go` (160 lines, full), `internal/icm/icm_test.go` (targeted: lines 1-60), `migrations/006_create_profiles.up.sql`, `008_add_automation.up.sql`, `009_add_timers.up.sql`/`.down.sql` (full, small files), `frontend/src/types/index.ts` (targeted: lines 128-227), `frontend/src/services/api.ts` (targeted: lines 1-20, 555-584), `frontend/src/pages/SettingsPage.tsx` (targeted: lines 970-1000, 1599-1650, plus grep for `SECTIONS`), `frontend/src/components/EnvironmentPanel.tsx` (288 lines, full), `frontend/src/pages/HelpPage.tsx` (targeted: lines 126-230), `.specify/specs/safety-and-abuse-policy-v1.md` (targeted: lines 1-10)
**Pattern extraction date:** 2026-09-15
