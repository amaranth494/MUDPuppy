---
phase: 01-profile-foundation-and-policy-gate
reviewed: 2026-09-15T17:30:02Z
depth: standard
files_reviewed: 14
files_reviewed_list:
  - cmd/server/main.go
  - internal/policy/policy.go
  - internal/policy/policy_test.go
  - internal/profiles/handler.go
  - internal/profiles/handler_test.go
  - internal/store/profile.go
  - internal/store/profile_test.go
  - migrations/010_add_ai_fields.up.sql
  - migrations/010_add_ai_fields.down.sql
  - frontend/src/components/AIPlayerPanel.tsx
  - frontend/src/pages/SettingsPage.tsx
  - frontend/src/services/api.ts
  - frontend/src/types/index.ts
  - scripts/verify-phase1.sh
findings:
  critical: 1
  warning: 3
  info: 1
  total: 5
status: issues_found
---

# Phase 01: Code Review Report

**Reviewed:** 2026-09-15T17:30:02Z
**Depth:** standard
**Files Reviewed:** 14
**Status:** issues_found

## Summary

Reviewed the diff introduced since `61e970b` for the profile-foundation-and-policy-gate phase: the five new `ai-settings` / `policy` / `policy/accept` / `engage-gate` HTTP endpoints, their `store.ProfileStore` backing methods, the `internal/policy` embed package, migration 010, and the AI Player settings UI.

Per-user scoping is sound: all five new endpoints route through `getProfileByConnectionID`, which resolves the profile via `GetProfileByConnection(userID, connectionID)` — a query filtered by both `user_id` and `connection_id` — so there is no IDOR path to another user's profile. The one-time policy-acceptance write is correctly guarded at the SQL level (`WHERE ... AND policy_accepted_at IS NULL`), making concurrent double-accept safe (first writer wins, second is a verified no-op). The `[AI-PLAYER]` log lines were checked against the actual code and a dedicated test (`TestAIPlayerLogLinesAreEmitted`) and do not print conduct-rules/approach-guidance prose or policy text — only booleans, IDs, and the version/timestamp. `go build ./...`, `go vet`, and `go test ./internal/policy/... ./internal/profiles/... ./internal/store/...` all pass.

One real defect was found: `AcceptPolicy` (both the store method and the handler that calls it) has an unguarded nil-dereference path when the underlying profile row disappears between the initial fetch and the accept write (e.g., the connection/profile is deleted in a second tab while the Accept Policy button's request is in flight). This is not covered by any existing test. Three lower-severity items round out the review: no upper bound on the AI settings numeric fields (weakens their stated purpose as safety-relevant cost/failure caps), a frontend edge case where invalid numeric input for those same fields is silently coerced to "blank" (no cap) rather than surfaced as a validation error, and an unbounded request body read shared with the rest of the codebase but newly exercised by two new large free-text fields.

## Critical Issues

### CR-01: Nil-pointer dereference in AcceptPolicy when the profile row disappears mid-request

**File:** `internal/profiles/handler.go:633-653` (reads `updatedProfile.PolicyVersionAccepted`/`PolicyAcceptedAt` unconditionally) and `internal/store/profile.go:501-512` (`ProfileStore.AcceptPolicy`)

**Issue:** `store.ProfileStore.AcceptPolicy` performs a guarded `UPDATE ... WHERE id = $2 AND user_id = $3 AND policy_accepted_at IS NULL` via `db.Exec`, which returns **no error** when zero rows match (e.g., the profile was deleted, or the connection_id/user_id pair no longer exists). It then unconditionally calls `s.GetProfile(userID, profileID)`, whose documented behavior (see `internal/store/profile.go:245-247`, `if err == sql.ErrNoRows { return nil, nil }`) is to return `(nil, nil)` — not an error — when the row is missing. So `AcceptPolicy` can legitimately return `(nil, nil)`.

The handler at `internal/profiles/handler.go:633` only guards on `err != nil`:
```go
updatedProfile, err := h.profileStore.AcceptPolicy(userUUID, profile.ID, policy.Version())
if err != nil {
    log.Printf("[PH0103] Accept policy failed: %v", err)
    h.sendError(w, "Failed to record acceptance")
    return
}
...
if updatedProfile.PolicyVersionAccepted != nil {   // <-- panics if updatedProfile == nil
    version = *updatedProfile.PolicyVersionAccepted
}
if updatedProfile.PolicyAcceptedAt != nil {        // <-- and again here
    acceptedAt = *updatedProfile.PolicyAcceptedAt
}
...
h.sendJSON(w, PolicyResponse{
    ...
    Accepted:        updatedProfile.PolicyAcceptedAt != nil,   // <-- and again here
    AcceptedAt:      updatedProfile.PolicyAcceptedAt,
    AcceptedVersion: updatedProfile.PolicyVersionAccepted,
})
```
This is reachable: `getProfileByConnectionID` a few lines earlier confirms the profile exists at fetch time, but if the connection/profile is deleted (e.g. via `DeleteProfile`, reachable from a connection-delete flow in another tab or a retried request) between that fetch and the `AcceptPolicy` call, the goroutine handling this request panics on the nil dereference. Go's `net/http` server recovers per-connection panics so the process itself survives, but the client gets a broken/empty response and a panic is logged — on the specific endpoint that records legal/safety policy acceptance, which is exactly where an unexplained failure is worst. This path has no test coverage: `TestPolicyAcceptUsesServerVersion` in `internal/profiles/handler_test.go` only exercises the profile-exists case, and the fake store's `AcceptPolicy` never returns `(nil, nil)`.

**Fix:** Add the same nil-guard `GetAISettings`/`GetPolicy`/`GetEngageGate` already have via `getProfileByConnectionID`:
```go
updatedProfile, err := h.profileStore.AcceptPolicy(userUUID, profile.ID, policy.Version())
if err != nil {
    log.Printf("[PH0103] Accept policy failed: %v", err)
    h.sendError(w, "Failed to record acceptance")
    return
}
if updatedProfile == nil {
    h.sendError(w, "Profile not found")
    return
}
```
(Equivalently, have `store.ProfileStore.AcceptPolicy` return a distinguishable "not found" error instead of delegating to `GetProfile`'s nil-on-not-found contract.)

## Warnings

### WR-01: No upper bound on Call Cap / Disengage Threshold

**File:** `internal/profiles/handler.go:775-796` (`validateAISettings`)
**Issue:** `validateAISettings` rejects `CallCap`/`DisengageThreshold` values below 1 but accepts any positive value up to `math.MaxInt`. These fields exist specifically to bound AI spend and failure tolerance (per the doc comments on `store.AISettings` and `store.DefaultDisengageThreshold`); an unbounded call cap (e.g. `999999999`) is functionally indistinguishable from "no cap" and defeats the field's purpose, whether entered by mistake or intentionally to bypass review-visible limits.
**Fix:** Add a sane upper bound, e.g.:
```go
const maxCallCap = 100000
if req.AISettings.CallCap != nil && (*req.AISettings.CallCap < 1 || *req.AISettings.CallCap > maxCallCap) {
    return &ValidationError{Message: "Call cap must be between 1 and 100000 when set"}
}
```

### WR-02: Invalid numeric input silently becomes "no cap" / "default threshold" instead of a validation error

**File:** `frontend/src/components/AIPlayerPanel.tsx:80-88` (`handleSave`)
**Issue:**
```ts
call_cap: callCapStr.trim() === '' ? null : Number(callCapStr),
disengage_threshold: disengageThresholdStr.trim() === '' ? null : Number(disengageThresholdStr),
```
If the user's typed value is non-empty but not a valid number (e.g. a lone `-` or `e`, which `<input type="number">` can still leave in `.value` in some browsers/input paths), `Number(...)` returns `NaN`. `JSON.stringify(NaN)` serializes to `null` (per the JSON spec's treatment of non-finite numbers), so the request silently sends `call_cap: null` / `disengage_threshold: null` — i.e. "no cap" / "use the default threshold" — instead of surfacing an error to the user. Since these fields are safety/cost-relevant caps, a mistyped value quietly removing the cap (rather than failing loudly) is a meaningful UX/safety regression, not just a cosmetic one.
**Fix:** Validate before building the request body and block the save with a visible error when the trimmed string is non-empty but not a valid finite number:
```ts
const parsedCallCap = callCapStr.trim() === '' ? null : Number(callCapStr);
if (parsedCallCap !== null && !Number.isFinite(parsedCallCap)) {
  setError('Call cap must be a valid number');
  return;
}
```
(same for `disengageThresholdStr`).

### WR-03: No request body size limit before validating conduct_rules/approach_guidance length

**File:** `internal/profiles/handler.go:556-560` (`PutAISettings`, `json.NewDecoder(r.Body).Decode(&req)`)
**Issue:** `validateAISettings` caps `ConductRules`/`ApproachGuidance` at 20000 characters, but that check happens only *after* the entire request body has been read into memory by `json.NewDecoder(r.Body).Decode`. There is no `http.MaxBytesReader` (or equivalent) anywhere in the codebase (`grep -rn "MaxBytesReader\|LimitReader" internal/ cmd/` returns nothing), so an authenticated caller can send an arbitrarily large body (e.g. hundreds of MB) that is fully buffered before the length check ever fires. This is a pre-existing gap shared by every other PUT endpoint in this handler, but this phase adds two new large free-text fields explicitly designed to hold long-form prose, making the gap more attractive to trigger here specifically.
**Fix:** Wrap the body reader once, e.g. in the router or per-handler:
```go
r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB, well above the 20000-char cap plus JSON overhead
```

## Info

### IN-01: Migration header comments use sql-migrate directive syntax under golang-migrate

**File:** `migrations/010_add_ai_fields.up.sql:1`, `migrations/010_add_ai_fields.down.sql:1`
**Issue:** Both files open with `-- +migrate Up` / `-- +migrate Down`, which is `rubenv/sql-migrate` directive syntax. The project uses `golang-migrate` (per `cmd/server/main.go`'s `m.Version()` / `.up.sql`/`.down.sql` file-name-based invocation), which does not read or need these directives — they are inert comments. This exactly matches the pre-existing convention in `migrations/009_add_timers.up.sql`/`.down.sql`, so it is not a regression introduced by this phase, just an inherited harmless inconsistency worth cleaning up if the migration files are ever touched again.
**Fix:** No action required for this phase; if migrations are revisited, drop the `-- +migrate` lines or switch tooling consistently.

---

_Reviewed: 2026-09-15T17:30:02Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
