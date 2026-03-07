# SP01 Tasks Tracker

**Spec:** SP01 — Account Authentication & Session Foundation  
**Reference:** [.specify/specs/SP01.md](../specs/SP01.md)  
**Reference:** [.specify/plans/SP01-plan.md](../plans/SP01-plan.md)

---

## Phase 0: Branch Creation (PH00)

### SP01PH00T01 — Create spec branch
- [x] **Task:** Create branch `sp01-account-auth` from master
- [x] **Commit:** "SP01PH00: Branch created"
- [x] **Status:** Completed

### SP01PH00T02 — Commit spec document
- [x] **Task:** Commit SP01.md to branch
- [x] **Commit:** "SP01PH00: Spec committed"
- [x] **Status:** Completed

### SP01PH00T03 — Push branch
- [x] **Task:** Push branch to origin
- [x] **Commit:** N/A
- [x] **Status:** Completed

---

## Phase 1: Database Schema (PH01)

### SP01PH01T01 — Choose and wire migration tool
- [x] **Task:** Select migration tool (Goose / Atlas / sql-migrate / golang-migrate), wire into backend build
- [x] **Acceptance:** Migration tool integrated with build system
- [x] **Commit:** "SP01PH01T01: Migration tool selected and wired"
- [x] **Status:** Completed

### SP01PH01T02 — Create users table
- [x] **Task:** Create users table migration (id uuid pk, email citext unique, email_verified_at nullable, created_at, updated_at)
- [x] **Acceptance:** Migration created with unique constraint on email
- [x] **Commit:** "SP01PH01T02: Users table created"
- [x] **Status:** Completed

### SP01PH01T03 — Create otp_challenges table
- [x] **Task:** Create otp_challenges table migration (id uuid pk, email indexed, otp_hash, expires_at, attempts default 0, created_at)
- [x] **Acceptance:** Migration created with indexed email column
- [x] **Commit:** "SP01PH01T03: OTP challenges table created"
- [x] **Status:** Completed

### SP01PH01T04 — Add DB connectivity check
- [x] **Task:** Add startup check: fail-fast if DATABASE_URL missing/unreachable
- [x] **Acceptance:** App panics on missing DB at startup
- [x] **Commit:** "SP01PH01T04: DB connectivity check added"
- [x] **Status:** Completed

### SP01PH01T05 — Run migrations on Railway Staging
- [x] **Task:** Run migrations from local terminal against Railway Staging Postgres (DATABASE_URL points to Staging)
- [x] **Acceptance:** Tables created in Staging, unique constraint enforced
- [x] **Commit:** "SP01PH01T05: Migrations applied to Railway Staging"
- [x] **Status:** Completed

### SP01PH01T06 — Deploy to Staging
- [x] **Task:** Deploy feature branch to Railway Staging environment
- [x] **Acceptance:** Staging deploy succeeds after migrations
- [x] **Commit:** "SP01PH01T06: Deployed to Railway Staging"
- [x] **Status:** Completed (code pushed to sp01-account-auth, awaiting Railway configuration)

---

## Phase 2: Redis Configuration (PH02)

### SP01PH02T01 — Create Redis utility module
- [x] **Task:** Create Go module with key builders, TTL constants, getters/setters. Key formats: OTP otp:email:{sha256(lower(email))}, Session session:{sessionID}, Session idle session_idle:{sessionID}
- [x] **Acceptance:** Utility module exists, no auth routes yet
- [x] **Commit:** "SP01PH02T01: Redis utility module created"
- [x] **Status:** Completed

### SP01PH02T02 — Define OTP TTL
- [x] **Task:** Implement OTP storage with SET key EX 900 (atomic, 15 min)
- [x] **Acceptance:** OTP keys expire in 15 minutes
- [x] **Commit:** "SP01PH02T02: OTP TTL implemented"
- [x] **Status:** Completed

### SP01PH02T03 — Define Session TTL (dual-key)
- [x] **Task:** Implement session with two keys: session:{id} with EX 86400 (24h hard cap), session_idle:{id} with EX 1800 (30min sliding)
- [x] **Acceptance:** Middleware checks both keys exist
- [x] **Commit:** "SP01PH02T03: Session TTL implemented"
- [x] **Status:** Completed

### SP01PH02T04 — Add Redis health check
- [x] **Task:** Add startup check: fail-fast if REDIS_URL missing/unreachable
- [x] **Acceptance:** App panics on missing Redis at startup
- [x] **Commit:** "SP01PH02T04: Redis health check added"
- [x] **Status:** Completed

---

## Phase 3: Authentication Backend (PH03)

### SP01PH03T01 — Implement registration endpoint
- [x] **Task:** POST /api/v1/register accepts email
- [x] **Acceptance:** Returns 201 on success
- [x] **Commit:** "SP01PH03T01: Registration endpoint created"
- [x] **Status:** Completed

### SP01PH03T02 — Implement OTP generation
- [x] **Task:** Generate 6-digit OTP, store in Redis with 15-min TTL. If SMTP fails, do NOT store OTP.
- [x] **Acceptance:** OTP stored in Redis (key: otp:{email}, TTL: 15 min)
- [x] **Commit:** "SP01PH03T02: OTP generation implemented"
- [x] **Status:** Completed

### SP01PH03T03 — Implement OTP delivery (SMTP)
- [x] **Task:** Send OTP via SMTP
- [x] **Acceptance:** Email sent with OTP code
- [x] **Commit:** "SP01PH03T03: OTP email delivery implemented"
- [x] **Status:** Completed

### SP01PH03T04 — Implement login endpoint
- [x] **Task:** POST /api/v1/login accepts email + OTP
- [x] **Acceptance:** Returns session token on valid OTP
- [x] **Commit:** "SP01PH03T04: Login endpoint created"
- [x] **Status:** Completed

### SP01PH03T05 — Implement session validation middleware
- [x] **Task:** Middleware validates session from Redis only, protects all /api/v1/* routes by default
- [x] **Acceptance:** Returns 401 without valid session
- [x] **Commit:** "SP01PH03T05: Session middleware created"
- [x] **Status:** Completed

### SP01PH03T06 — Implement logout endpoint
- [x] **Task:** DELETE /api/v1/logout invalidates session
- [x] **Acceptance:** Session removed, returns 200
- [x] **Commit:** "SP01PH03T06: Logout endpoint created"
- [x] **Status:** Completed

### SP01PH03T07 — Implement /api/v1/me endpoint
- [x] **Task:** GET /api/v1/me returns user info
- [x] **Acceptance:** Returns user email when authenticated
- [x] **Commit:** "SP01PH03T07: User info endpoint created"
- [x] **Status:** Completed

### SP01PH03T08 — Add environment variable handling
- [x] **Task:** Add SESSION_SECRET, SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASS, EMAIL_FROM_ADDRESS
- [x] **Acceptance:** Startup fails if SESSION_SECRET missing
- [x] **Commit:** "SP01PH03T08: Env vars configured"
- [x] **Status:** Completed

---

## Phase 4: Frontend UI (PH04)

### SP01PH04T01 — Create login page
- [x] **Task:** Login page with email input
- [x] **Acceptance:** Email field, submit button visible
- [x] **Commit:** "SP01PH04T01: Login page created"
- [x] **Status:** Completed

### SP01PH04T02 — Create registration flow
- [x] **Task:** Registration page with OTP entry
- [x] **Acceptance:** OTP input field appears after email submit
- [x] **Commit:** "SP01PH04T02: Registration flow created"
- [x] **Status:** Completed

### SP01PH04T03 — Implement auth state management
- [x] **Task:** Store session in HTTP-only cookie
- [x] **Acceptance:** Cookie set on login, cleared on logout
- [x] **Commit:** "SP01PH04T03: Auth state management implemented"
- [x] **Status:** Completed

### SP01PH04T04 — Add logged-in indicator
- [x] **Task:** Show user email when logged in
- [x] **Acceptance:** Displays logged-in state in UI
- [x] **Commit:** "SP01PH04T04: Logged-in indicator added"
- [x] **Status:** Completed

### SP01PH04T05 — Add logout button
- [x] **Task:** Logout button in UI
- [x] **Acceptance:** Clicking logout clears session
- [x] **Commit:** "SP01PH04T05: Logout button added"
- [x] **Status:** Completed

---

## Phase 5: Security & Rate Limiting (PH05)

### SP01PH05T01 — Implement OTP rate limiting
- [x] **Task:** Max 5 OTP requests per email per hour
- [x] **Acceptance:** Returns 429 when exceeded
- [x] **Commit:** "SP01PH05T01: OTP rate limiting implemented"
- [x] **Status:** Completed

### SP01PH05T02 — Implement login rate limiting
- [x] **Task:** Max 10 login attempts per IP per 10 min
- [x] **Acceptance:** Returns 429 when exceeded
- [x] **Commit:** "SP01PH05T02: Login rate limiting implemented"
- [x] **Status:** Completed

### SP01PH05T03 — Add security headers
- [x] **Task:** Add CSRF, security headers
- [x] **Acceptance:** Headers present in responses
- [x] **Commit:** "SP01PH05T03: Security headers added"
- [x] **Status:** Completed

### SP01PH05T04 — Add abuse logging
- [x] **Task:** Log rate-limit violations (IP, endpoint, timestamp) without PII
- [x] **Acceptance:** Logs include violation details for later analysis
- [x] **Commit:** "SP01PH05T04: Abuse logging added"
- [x] **Status:** Completed

---

## Phase 6: Deployment to Staging (PH06)

### SP01PH06T01 — Run migrations on Railway Staging
- [x] **Task:** Apply migrations to Railway Staging Postgres
- [x] **Acceptance:** Tables created in staging
- [x] **Commit:** "SP01PH06T01: Migrations applied to Railway Staging"
- [x] **Status:** Completed (golang-migrate runs programmatically on server startup)

### SP01PH06T02 — Configure environment variables
- [x] **Task:** Add SESSION_SECRET, SMTP_* to Railway Staging
- [x] **Acceptance:** Variables set, app starts
- [x] **Commit:** "SP01PH06T02: Environment configured"
- [x] **Status:** Completed (DATABASE_URL, REDIS_URL, SESSION_SECRET, SMTP_HOST=smtp.gmail.com, SMTP_PORT=465, SMTP_USER, SMTP_PASS, EMAIL_FROM_ADDRESS)

### SP01PH06T03 — Verify staging auth flow
- [x] **Task:** Test full registration/login flow in staging
- [x] **Acceptance:** All endpoints work in staging
- [x] **Commit:** "SP01PH06T03: Staging auth verified"
- [x] **Status:** Completed (Registration returns 201, Email sent successfully, Login works, Session validation works)

---

## Phase 7: QA Validation (PH07)

### SP01PH07T00 — QA Testing Guide Created
- [x] **Task:** Create detailed manual QA testing guide
- [x] **Acceptance:** Guide document created at .specify/tasks/SP01PH07-QA-Testing-Guide.md
- [x] **Status:** Completed

### SP01PH07T01 — QA login from clean browser
- [x] **Task:** QA tests from incognito browser
- [x] **Acceptance:** Login works from fresh session
- [x] **Status:** Completed

### SP01PH07T02 — QA logout invalidates session
- [x] **Task:** QA logs out, tries to access /api/v1/me
- [x] **Acceptance:** Returns 401 after logout
- [x] **Status:** Completed

### SP01PH07T03 — QA expired OTP rejection
- [x] **Task:** QA tests with expired OTP
- [x] **Acceptance:** Rejected with clear message
- [x] **Status:** Completed

### SP01PH07T04 — QA unauthorized access returns 401
- [x] **Task:** QA accesses /api/v1/me without session
- [x] **Acceptance:** Returns 401
- [x] **Status:** Completed

### SP01PH07T05 — QA rate limiting
- [x] **Task:** QA triggers rate limit
- [x] **Acceptance:** Returns 429
- [x] **Status:** Completed

### SP01PH07T06 — QA OTP reuse fails
- [x] **Task:** QA attempts to use same OTP twice sequentially
- [x] **Acceptance:** Second attempt fails
- [x] **Status:** Completed

### SP01PH07T07 — QA 24-hour hard cap enforced
- [x] **Task:** QA confirms session expires after 24h regardless of activity
- [x] **Acceptance:** Session invalid after hard cap
- [x] **Status:** Completed (code verified, not time-tested)

### SP01PH07T08 — QA middleware coverage
- [x] **Task:** QA attempts access to various /api/v1/* endpoints without auth
- [x] **Acceptance:** All return proper status codes
- [x] **Status:** Completed

### SP01PH07T09 — QA OTP concurrent reuse
- [x] **Task:** Fire two concurrent login attempts with same OTP
- [x] **Acceptance:** Only one succeeds, other fails cleanly (not 500)
- [x] **Status:** Completed

### SP01PH07T10 — QA sign-off
- [x] **Task:** QA produces pass report
- [x] **Acceptance:** All scenarios pass including hardening validations
- [x] **Status:** Completed

---

## Phase 8: Merge & Close (PH08)

### SP01PH08T01 — Final CI check
- [x] **Task:** Ensure CI passes
- [x] **Acceptance:** All tests green
- [x] **Status:** Completed

### SP01PH08T02 — Merge to master
- [x] **Task:** Merge sp01-account-auth to master
- [x] **Commit:** "SP01PH08T02: Merged to master"
- [x] **Status:** Completed

### SP01PH08T03 — Verify production
- [x] **Task:** Confirm production works post-merge
- [x] **Acceptance:** Production URL stable
- [x] **Status:** Completed

### SP01PH08T04 — Mark spec closed
- [x] **Task:** Update SP01.md status to Closed
- [x] **Commit:** "SP01PH08T04: Spec closed"
- [x] **Status:** Completed

### SP01PH08T05 — Establish Persistent Staging Branch
- [x] **Task:** Convert sp01-account-auth into long-lived staging branch: Merge to master, create staging from master, delete feature branch, configure Railway
- [x] **Acceptance:** master = production, staging = integration buffer, Railway: master→Prod, staging→Staging, future specs branch from staging
- [x] **Commit:** "SP01PH08T05: Staging branch established"
- [x] **Status:** Completed

---

## Summary

| Phase | Tasks | Completed |
|-------|-------|----------|
| PH00 | 3 | 3/3 |
| PH01 | 6 | 6/6 |
| PH02 | 4 | 4/4 |
| PH03 | 8 | 8/8 |
| PH04 | 5 | 5/5 |
| PH05 | 4 | 4/4 |
| PH06 | 3 | 3/3 |
| PH07 | 11 | 11/11 |
| PH08 | 5 | 5/5 |
| **Total** | **48** | **48/48** |

---

## Definition of Done (per Constitution)

Each Task is complete when:
- [ ] Code implemented
- [ ] Unit tests written (where applicable)
- [ ] CI passes
- [ ] Acceptance criteria validated locally
- [ ] No TODO placeholders remain
- [ ] No console errors
- [ ] Documentation updated (if applicable)

Each Phase is complete when:
- [ ] All Tasks complete
- [ ] Phase acceptance criteria satisfied

Each Spec is complete when:
- [ ] QA passes
- [ ] Merged to master
- [ ] Deployment verified
