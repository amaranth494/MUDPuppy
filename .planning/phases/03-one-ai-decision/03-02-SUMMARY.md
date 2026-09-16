---
phase: 03-one-ai-decision
plan: 02
subsystem: api
tags: [go, net-http, gemini, config, env-vars, httptest]

# Dependency graph
requires:
  - phase: 01-profile-foundation-and-policy-gate
    provides: "Phase 1 D-09 (a blank profile model name means the server default) and the AISettings shape ResolveAISettings resolves against"
provides:
  - "internal/config.AIModelEntry, Config.AIDefaultModelSlug, Config.AIModels, AIConfigured(), ResolveModelEntry() — the env-sourced model registry and its resolution rules"
  - "internal/gemini.Client / NewClient / GenerateContent — a hand-written net/http client for the Gemini generateContent endpoint, returning {Reasoning, Command} or a typed *Error/Kind"
affects: [03-06-refusal, 03-08-driver, 03-13-evidence-harness]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Env-scanned registry: one AI_MODEL_<SLUG>_NAME variable per entry drives a single os.Environ() scan; sibling _ENDPOINT/_KEY/_PROVIDER variables are read by direct os.Getenv lookup keyed on the discovered slug"
    - "Warn-and-default, never fatal, for all new config fields — the one errors.New(...) fail-fast case in internal/config/config.go remains SESSION_SECRET only"
    - "Typed vendor error kinds (Kind + *Error + ErrorKind(err)) so callers branch without a type switch, mirrored on the internal/gemini side of the D-13 failure boundary"

key-files:
  created:
    - internal/config/config_test.go
    - internal/gemini/client.go
    - internal/gemini/client_test.go
    - .planning/phases/03-one-ai-decision/deferred-items.md
  modified:
    - internal/config/config.go

key-decisions:
  - "Env variable names (Claude's discretion per D-18): AI_MODEL_DEFAULT, AI_MODEL_<SLUG>_NAME, AI_MODEL_<SLUG>_ENDPOINT, AI_MODEL_<SLUG>_KEY, AI_MODEL_<SLUG>_PROVIDER (defaults to 'gemini' when blank)"
  - "ResolveModelEntry's unmatched-non-blank case returns the default entry's endpoint/key with ModelName replaced by the requested string, so a retired or unknown model name reaches the vendor and comes back as a named D-13 failure instead of a silent substitution"
  - "Gemini request/response bodies are typed Go structs (not map[string]any) for compile-time shape checking, while still producing the exact JSON RESEARCH.md's Gemini API Reference specifies"

requirements-completed: [REQ-env-config, REQ-single-decision]

duration: 8min
completed: 2026-09-15
---

# Phase 3 Plan 02: Model Registry and Gemini Client Summary

**Environment-only Gemini model registry (`AI_MODEL_*` vars) plus a hand-written `net/http` client for Gemini's `generateContent` endpoint that returns a typed `{reasoning, command}` answer or one of five named error kinds, with the API key traveling only in the `x-goog-api-key` header.**

## Performance

- **Duration:** 8 min
- **Started:** 2026-09-15T18:46:47-07:00 (post worktree-base reset)
- **Completed:** 2026-09-15T18:54:39-07:00
- **Tasks:** 2
- **Files modified:** 4 (2 created source files, 2 test files; 1 config file modified)

## Accomplishments

- `internal/config` now scans `os.Environ()` once for `AI_MODEL_<SLUG>_NAME` variables, builds a registry of `AIModelEntry` (model name, endpoint, key, provider), and exposes `AIConfigured()` and `ResolveModelEntry(requested)` for the refusal (plan 03-06) and the driver (plan 03-08) to call. Absence of every Gemini variable is not fatal — `Load()` still returns a valid `*Config` and `nil` error.
- `internal/gemini` is a brand-new package: a hand-written `net/http` client with zero new `go.mod` entries, posting to `{endpoint}/v1beta/models/{model}:generateContent` with the API key in the `x-goog-api-key` header (never a `?key=` query string), a `responseSchema` constraining the model to `{reasoning, command}`, and a double JSON decode of the inner answer.
- Every Gemini failure shape (transport error, 400, 401, 403, 429, empty candidates, unparseable inner JSON) returns as a typed `*gemini.Error` with a `Kind` a caller can branch on via `ErrorKind(err)`, and none of them panics.
- No model identifier or vendor host is a literal anywhere in the new or modified Go source; both live only as parameters supplied by the environment-sourced registry.

## Task Commits

1. **Task 03-02-01: Model registry from environment, absence non-fatal** - `942bf55` (feat)
2. **Task 03-02-02: Gemini generateContent client with typed errors** - `25629e8` (feat)
3. **Deferred-item logging (out-of-scope pre-existing failure)** - `c9933bd` (docs)

_No plan-metadata commit yet — this executor does not update STATE.md/ROADMAP.md (parallel worktree run); the orchestrator commits those after all wave agents complete._

## Files Created/Modified

- `internal/config/config.go` - Adds `AIModelEntry`, `Config.AIDefaultModelSlug`, `Config.AIModels`, the `AI_MODEL_*` env scan in `Load()`, `AIConfigured()`, `ResolveModelEntry()`
- `internal/config/config_test.go` - First test file for this package: `TestLoadAIRegistry`, `TestLoadAIRegistryIgnoresIncompleteEntries`, `TestAIConfigured`, `TestResolveModelEntry`
- `internal/gemini/client.go` - `Client`, `NewClient`, `GenerateContent`, `Answer`, `Kind` constants, `Error`, `ErrorKind`
- `internal/gemini/client_test.go` - `httptest`-backed `TestGenerateContent` (5 subtests) and `TestGenerateContentErrors` (7-case table)
- `.planning/phases/03-one-ai-decision/deferred-items.md` - Logs one pre-existing, out-of-scope test failure found while verifying `go test ./...`

## Environment Variables (for the owner to set on Railway staging — plan 03-13)

| Variable | Meaning |
|----------|---------|
| `AI_MODEL_DEFAULT` | The slug of the default registry entry, e.g. `GEMINI` |
| `AI_MODEL_<SLUG>_NAME` | The model identifier sent to the vendor, e.g. a Gemini model id |
| `AI_MODEL_<SLUG>_ENDPOINT` | The API base URL for that entry, e.g. `https://generativelanguage.googleapis.com` |
| `AI_MODEL_<SLUG>_KEY` | The API key for that entry |
| `AI_MODEL_<SLUG>_PROVIDER` | Optional; defaults to `gemini` when unset — the only implemented provider this phase |

`DEFAULT` is a reserved slug and is skipped with a warning if declared as an entry (e.g. `AI_MODEL_DEFAULT_NAME`). No key is committed anywhere in this repo; all test fixtures use the literal `test-key-not-a-real-credential`.

## Decisions Made

- Env variable naming scheme above (Claude's discretion, D-18) — a small set of `AI_MODEL_*` variables rather than one JSON blob, matching this codebase's existing per-field env-var convention in `config.go`.
- `ResolveModelEntry`'s three-branch resolution rule (blank → default; matched slug/model name → that entry; unmatched non-blank → default's endpoint/key with the requested name substituted) implements the plan's exact contract so an unknown/retired model surfaces as a vendor error (D-13) rather than a silent fallback.
- Gemini request/response JSON built from typed Go structs rather than `map[string]any`, for compile-time field-name safety, while producing byte-identical JSON shape to RESEARCH.md's reference.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Source-assertion self-match in a code comment**
- **Found during:** Task 03-02-02 verification
- **Issue:** A doc comment on the `x-goog-api-key` header line originally read `"never a ?key= query parameter"`, which itself matched the acceptance criterion's own `grep -cE "\?key=|..."` pattern intended to catch actual query-string construction, producing a false-positive count of 1 instead of 0.
- **Fix:** Reworded the comment to `"never as a URL query parameter"` — same meaning, no longer matches the literal grep pattern.
- **Files modified:** internal/gemini/client.go
- **Verification:** `grep -cE "\?key=|\"key\"|key=\" *\+" internal/gemini/client.go` now returns 0
- **Committed in:** 25629e8 (Task 2 commit — fixed before commit, not a separate commit)

---

**Total deviations:** 1 auto-fixed (1 bug, caught during self-verification before commit)
**Impact on plan:** Cosmetic only — no behavior change, source assertion now passes as specified.

## Issues Encountered

- **`-race` unavailable in this environment:** `go test ... -race` requires cgo, and this Windows Git Bash environment has `CGO_ENABLED=0` with no `gcc` on `PATH` (`go env CGO_ENABLED` → `0`, `where gcc` → not found). Ran `go test ./internal/gemini/... -v` and `go test ./internal/config/... -v` without `-race` instead; all tests pass. This is an environment limitation, not a code issue — flagging so the phase's evidence-capture step (plan 03-13, which runs on Railway/CI where cgo may be available) knows to attempt `-race` there if the harness environment supports it.
- **Pre-existing failing test outside this plan's scope:** `go test ./...` (full suite) shows `internal/icm` `TestHandlerRegistration/CANCEL` failing on the pre-plan baseline (commit `6e98595`), traced to a prior commit (`9066456`) unrelated to 03-02. `internal/icm` is not in this plan's `files_modified` list and was not touched. Logged to `.planning/phases/03-one-ai-decision/deferred-items.md` per the scope-boundary rule; not fixed here.

## User Setup Required

External services require manual configuration on Railway staging before plan 03-06's refusal and plan 03-08's driver can be exercised end-to-end:
- Set `AI_MODEL_DEFAULT`, `AI_MODEL_<SLUG>_NAME`, `AI_MODEL_<SLUG>_ENDPOINT`, `AI_MODEL_<SLUG>_KEY` (and optionally `_PROVIDER`) on the Railway `staging` environment, using the owner's Gemini free-tier API key. See the Environment Variables table above.
- No code or migration changes are needed to enable this — `AIConfigured()` reads these directly and turns `true` once they are set correctly.

## Next Phase Readiness

- `internal/config.AIConfigured()` and `ResolveModelEntry()` are ready for plan 03-06 to wire into the `#AUTO ON` engage-gate refusal (D-20).
- `internal/gemini.Client.GenerateContent` is ready for plan 03-08's driver to call with the ring-buffer snapshot (tier one) and the profile's conduct rules/approach guidance (tier two).
- No blockers for downstream Phase 3 plans introduced by this plan. The one pre-existing `internal/icm` test failure (see Issues Encountered) is unrelated and does not block 03-06/03-08, since neither touches the `CANCEL` handler path.

---
*Phase: 03-one-ai-decision*
*Completed: 2026-09-15*

## Self-Check: PASSED

All created files verified present on disk (internal/config/config.go, internal/config/config_test.go, internal/gemini/client.go, internal/gemini/client_test.go, .planning/phases/03-one-ai-decision/deferred-items.md). All three commit hashes (942bf55, 25629e8, c9933bd) verified present in `git log`.
