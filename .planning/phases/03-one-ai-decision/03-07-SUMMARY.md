---
phase: 03-one-ai-decision
plan: 07
subsystem: auth
tags: [go, security, logging, env-vars, DR-2-01]

# Dependency graph
requires:
  - phase: 03-one-ai-decision
    provides: "03-02's warn-and-default optional-env convention in internal/config/config.go, extended in place for AUTH_LOG_OTP"
provides:
  - "internal/config.Config.AuthLogOTP — off-by-default env flag (AUTH_LOG_OTP) gating one-time sign-in code logging"
  - "internal/auth.Handler.logOTPIssued — the single gated helper both OTP-issuance call sites now route through"
  - "internal/auth's first test file, proving the code is absent from the log unless the flag is explicitly on"
affects: [03-13-evidence-harness]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Warn-and-default, never fatal, for AUTH_LOG_OTP — matches config.go's existing majority pattern (strconv.ParseBool, unparseable value warns and stays at the false default), the SESSION_SECRET fail-fast case remains the sole exception"
    - "One gated logging helper per security-sensitive value: logOTPIssued is the only place in internal/auth allowed to print the code, both former call sites route through it"

key-files:
  created:
    - internal/auth/handler_test.go
  modified:
    - internal/config/config.go
    - internal/auth/handler.go

key-decisions:
  - "AUTH_LOG_OTP parsed with strconv.ParseBool rather than a hand-rolled '1'/'true' check, so it also accepts Go's other canonical boolean spellings (t/T/TRUE/0/f/F/FALSE); any unparseable value warns and the flag stays at its false default, matching the plan's off-by-default, never-fatal requirement"
  - "The DEV MODE branch (internal/auth/handler.go, the else side of the emailSender-configured check) still prints the code directly via log.Printf(\"DEV MODE - OTP for %s: %s\", email, otp) and was intentionally left untouched — see Deviations/Issues Encountered below for why this satisfies rather than contradicts the todo's 'confirm production has no equivalent' instruction"

requirements-completed: [REQ-env-config]

# Metrics
duration: ~9min
completed: 2026-09-15
---

# Phase 3 Plan 07: Sign-in Code Stops Reaching the Log (DR-2-01) Summary

**One gated helper (`logOTPIssued`) and one off-by-default env flag (`AUTH_LOG_OTP`) close DR-2-01: the one-time sign-in code no longer reaches staging's deploy log by default, the issuance is still auditable, and `internal/auth`'s first test file fails if anyone reintroduces the code or an email address into that line.**

## Performance

- **Duration:** ~9 min
- **Started:** 2026-09-15T19:00:00-07:00 (approx, post worktree-base reset)
- **Completed:** 2026-09-15T19:08:46-07:00
- **Tasks:** 1
- **Files modified:** 3 (1 config file modified, 1 handler file modified, 1 test file created)

## Accomplishments

- `internal/config.Config` gained `AuthLogOTP bool`, read from `AUTH_LOG_OTP` with the file's existing warn-and-default optional-env pattern (`strconv.ParseBool`, default `false`, unparseable values warn and stay `false`). It does not become a second fail-fast case; `SESSION_SECRET` remains the file's only `errors.New(...)`.
- `internal/auth.Handler` gained a `logOTPCode bool` field, set from `cfg.AuthLogOTP` in `NewHandler`, and a single new helper `func (h *Handler) logOTPIssued(otp string)` that both former `log.Printf("STAGING: OTP sent to user, code: %s", otp)` call sites (in `Register` and `SendOTP`) now call instead of printing directly.
  - When the flag is off (the default, and what staging runs): `[AUTH] one-time sign-in code issued (code suppressed; set AUTH_LOG_OTP=1 to print)` — the audit fact survives, the code, email address, user id, code length, and any prefix of it do not.
  - When the flag is on: the original `STAGING: OTP sent to user, code: %s` line, unchanged, for local debugging.
- `internal/auth/handler_test.go` is the first test file this package has had. `TestOTPNotLogged` has three subtests, all passing: `logs_issuance_without_the_code`, `logs_the_code_only_when_flag_is_on`, and `suppressed_line_carries_no_identifiers` (asserts no `@` and no 4+ digit run in the suppressed line — the value `"482913"` used throughout is a placeholder for assertions only, never a real code).

## Task Commits

Each task was committed atomically:

1. **Task 03-07-01: The sign-in code stops reaching the log, and a test proves it** - `bdb76aa` (fix)

_No plan-metadata commit — this executor does not update STATE.md/ROADMAP.md (parallel worktree run); the orchestrator commits those after all wave agents complete._

## Files Created/Modified

- `internal/config/config.go` - Adds `AuthLogOTP bool` to `Config` (doc comment names DR-2-01) and its `AUTH_LOG_OTP` warn-and-default read in `Load()`
- `internal/auth/handler.go` - Adds `logOTPCode bool` to `Handler`, sets it from `cfg.AuthLogOTP` in `NewHandler`, adds `logOTPIssued(otp string)`, and replaces both `log.Printf("STAGING: OTP sent to user, code: %s", otp)` call sites (in `Register` and `SendOTP`) with `h.logOTPIssued(otp)`
- `internal/auth/handler_test.go` - New file. `TestOTPNotLogged` with the three required subtests, capturing `log` output via `log.SetOutput(&buf)` (restored via `defer log.SetOutput(os.Stderr)`) and `log.SetFlags(0)` (restored via `defer log.SetFlags(origFlags)`) so timestamp digits don't produce false positives in the digit-run assertion

## Repository-wide OTP logging confirmation (DR-2-01)

Command (exact, from the plan's `<action>`):

```
grep -rn "otp\|OTP" --include=*.go . | grep -i "log\.\|Printf\|Println"
```

Output (exact, re-verified against the committed file):

```
./internal/auth/handler.go:58:		log.Printf("STAGING: OTP sent to user, code: %s", otp)
./internal/auth/handler.go:61:	log.Printf("[AUTH] one-time sign-in code issued (code suppressed; set AUTH_LOG_OTP=1 to print)")
./internal/auth/handler.go:106:	return fmt.Sprintf("%06d", otp), nil
./internal/auth/handler.go:167:		log.Printf("ABUSE: OTP rate limit exceeded for email hash, endpoint: %s, timestamp: %s",
./internal/auth/handler.go:179:		log.Printf("Error generating OTP: %v", err)
./internal/auth/handler.go:186:		log.Printf("Error storing OTP: %v", err)
./internal/auth/handler.go:196:			log.Printf("Failed to send OTP email: %v", err)
./internal/auth/handler.go:204:		log.Printf("DEV MODE - OTP for %s: %s", email, otp)
./internal/auth/handler.go:256:		log.Printf("ABUSE: OTP rate limit exceeded for email hash, endpoint: %s, timestamp: %s",
./internal/auth/handler.go:268:		log.Printf("Error generating OTP: %v", err)
./internal/auth/handler.go:275:		log.Printf("Error storing OTP: %v", err)
./internal/auth/handler.go:285:			log.Printf("Failed to send OTP email: %v", err)
./internal/auth/handler.go:293:		log.Printf("DEV MODE - OTP for %s: %s", email, otp)
./internal/auth/handler.go:350:		log.Printf("Error verifying OTP: %v", err)
./internal/config/config.go:112:			log.Printf("Warning: Invalid OTP_EXPIRY_MINUTES '%s', using default 15", expiryStr)
./internal/config/config.go:250:			log.Printf("Warning: Invalid AUTH_LOG_OTP %q, using default false", otpLogStr)
```

No other file in the repository (including `internal/auth/handler_test.go`) matches — the new test's identifiers (`TestOTPNotLogged`, `logOTPIssued` calls, etc.) don't independently satisfy this grep's `log\.|Printf|Println` half.

**Reading this confirmation, exactly as the todo asks ("confirm production has no equivalent"):**

- Line 58 is the code-printing line, now reachable only from inside `logOTPIssued`'s `if h.logOTPCode` branch (flag explicitly on). Line 61 is the suppressed audit line, printed when the flag is off — the default, and what staging runs.
- Every other `handler.go` OTP-related log line is either a rate-limit/error-path message that never carries the code value (`ABUSE: ...`, `Error generating OTP`, `Error storing OTP`, `Failed to send OTP email`, `Error verifying OTP`), or the two `DEV MODE - OTP for %s: %s` lines (204, 293), addressed next. `internal/config/config.go`'s two matches are unrelated warn-and-default messages for numeric/boolean parsing, neither of which prints a code.
- The two former code-printing call sites this plan targeted (`Register` at the former line 181, `SendOTP` at the former line 270) now both route through `logOTPIssued`, which is gated by `h.logOTPCode` (from `cfg.AuthLogOTP`, off by default; staging's configuration does not set `AUTH_LOG_OTP`). This is the exact call path staging exercises when SMTP is configured — the branch DR-2-01 was raised against.
- The two remaining `DEV MODE - OTP for %s: %s` lines (204, 293) still print the code directly. Each sits in the `else` branch of `if h.emailSender != nil && h.emailSender.IsConfigured()`, i.e. it only executes when SMTP is entirely unconfigured. The plan's `<action>` and `<context>` both scope this task to the two `"STAGING: ..."` call sites verified this session (`internal/auth/handler.go:181` and `:270` pre-plan); the DEV MODE lines were not among them and were left as-is.
  - This line cannot execute in any environment where `SMTPHost`/`SMTPUser`/`SMTPPass` are set — staging and production both require these for email delivery to work at all, so a working deployment never takes this branch. It exists purely for a developer running the server locally with no SMTP credentials configured.
  - This is flagged here rather than silently left out of the confirmation, per Rule 2/threat-surface-scan discipline. It is not fixed in this plan because the plan's action explicitly named only the two `STAGING` call sites, and folding the DEV MODE branch in would be an unrequested behavior change to local-developer ergonomics rather than a change to what staging or production can print. If the owner wants the DEV MODE branch closed off too (e.g. for a future demo environment that runs without SMTP), that is a one-line follow-up: route it through `logOTPIssued` as well, or its own gated variant.
- Outside `internal/auth`, only `internal/config/config.go` matches at all, and both of its matches are unrelated warn-and-default parsing messages (`OTP_EXPIRY_MINUTES`, `AUTH_LOG_OTP` themselves) — neither prints a code. No other package in the repository (`internal/redis`, `internal/email`, `internal/store`, etc.) has any OTP-related `log.`/`Printf`/`Println` call at all — the grep above is the entire repository-wide surface.

## Decisions Made

- `AUTH_LOG_OTP` parsed with `strconv.ParseBool` (accepts `1`/`t`/`T`/`TRUE`/`true`/`True` as true and `0`/`f`/`F`/`FALSE`/`false`/`False` as false; anything else is unparseable) rather than a hand-rolled two-value check, matching this file's stdlib-first convention elsewhere (`strconv.Atoi` for the integer fields). The plan's "accept `1` and `true`" language is satisfied; the other canonical boolean spellings are accepted as a superset with no behavior change to the two named values, and any truly unparseable value warns and stays at the `false` default exactly as specified.
- The DEV MODE OTP-printing line was left untouched — see the Repository-wide confirmation section above for the full reasoning. Documented here rather than silently fixed because it is a real (if narrow) residual place a code can reach a log, and the plan's `<action>` explicitly did not name it.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Source-assertion self-match in the helper's own doc comment**
- **Found during:** Task 03-07-01 self-verification
- **Issue:** The first draft of the doc comment above `logOTPIssued` began "`logOTPIssued` is the single place in this package allowed to print a one-time sign-in code..." — the literal function name in its own comment made `grep -c "logOTPIssued" internal/auth/handler.go` return 4 instead of the acceptance criterion's expected 3 (one definition, two call sites).
- **Fix:** Reworded the comment's opening to "This is the single place in this package allowed to print a one-time sign-in code..." — same meaning, no longer contains the literal function name.
- **Files modified:** internal/auth/handler.go
- **Verification:** `grep -c "logOTPIssued" internal/auth/handler.go` now returns 3
- **Committed in:** bdb76aa (Task 1 commit — fixed before commit, not a separate commit)

---

**Total deviations:** 1 auto-fixed (1 bug, caught during self-verification before commit)
**Impact on plan:** Cosmetic only — no behavior change, source assertion now passes as specified.

## Issues Encountered

- **DEV MODE OTP print line, out of the named scope:** see "Repository-wide OTP logging confirmation" above. Not a bug in this plan's own work — the line predates this plan and was not one of the two call sites the plan's `<context>` verified and named for replacement. Documented in full rather than silently omitted from the grep confirmation.
- **`-race` unavailable in this environment:** consistent with 03-02's finding — `CGO_ENABLED=0`, no `gcc` on `PATH` in this Windows Git Bash environment. Ran `go test ./internal/auth/... ./internal/config/... -v` and the full `go test ./...` without `-race`; all packages pass, including `internal/icm` (previously flagged as a pre-existing failure in 03-02's deferred-items entry — it now passes on this worktree's base, so no new entry is needed here).

## User Setup Required

None — `AUTH_LOG_OTP` is optional and defaults to off; no Railway staging configuration change is required to close DR-2-01. The owner may set `AUTH_LOG_OTP=1` on a local/dev environment only, never on staging, if the code needs to be seen for debugging.

## Next Phase Readiness

- DR-2-01's code-side remediation (the env flag and the single gated helper) is complete and tested. The remaining piece from the todo — "review Railway log access on the staging project" — is an operational decision, out of scope for this plan, and plan 03-13 puts it on the Phase 3 security review agenda alongside the record of this closure.
- `evidence/01-test-report.txt` (plan 03-13) should capture `go test ./internal/auth/... -run TestOTPNotLogged -v`'s PASS lines as the proof artifact this plan's `must_haves.truths` names.
- `evidence/04-staging-ai-player.log` (plan 03-13's staging walkthrough) should be checked for the absence of a `code:` line on a fresh sign-in, confirming the deployed behavior matches this plan's test.
- No blockers for downstream Phase 3 plans. This plan touched only `internal/config/config.go` (additively, alongside 03-02's untouched AI registry block) and `internal/auth/handler.go`/`handler_test.go`, none of which any other Phase 3 plan modifies.

---
*Phase: 03-one-ai-decision*
*Completed: 2026-09-15*

## Self-Check: PASSED

All created/modified files verified present on disk (internal/config/config.go, internal/auth/handler.go, internal/auth/handler_test.go). Commit hash bdb76aa verified present in `git log`.
