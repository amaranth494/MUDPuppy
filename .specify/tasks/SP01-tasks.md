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
- [ ] **Task:** Deploy feature branch to Railway Staging environment
- [ ] **Acceptance:** Staging deploy succeeds after migrations
- [ ] **Commit:** "SP01PH01T06: Deployed to Railway Staging"
- [ ] **Status:** Pending

---

## Phase 2: Redis Configuration (PH02)

### SP01PH02T01 — Configure Redis key naming
- [ ] **Task:** Set Redis key format: `otp:{email}`, `session:{sessionID}`
- [ ] **Acceptance:** Keys defined in code
- [ ] **Commit:** "SP01PH02T01: Redis keys configured"
- [ ] **Status:** Pending

### SP01PH02T02 — Set OTP TTL
- [ ] **Task:** Configure OTP TTL to 15 minutes
- [ ] **Acceptance:** TTL set in Redis config
- [ ] **Commit:** "SP01PH02T02: OTP TTL configured"
- [ ] **Status:** Pending

### SP01PH02T03 — Set session TTL
- [ ] **Task:** Configure session TTL: Absolute max = 24 hours (hard cap), Idle timeout = 30 minutes (resets on activity)
- [ ] **Acceptance:** TTL set in Redis config
- [ ] **Commit:** "SP01PH02T03: Session TTL configured"
- [ ] **Status:** Pending

---

## Phase 3: Authentication Backend (PH03)

### SP01PH03T01 — Implement registration endpoint
- [ ] **Task:** POST /api/v1/register accepts email
- [ ] **Acceptance:** Returns 201 on success
- [ ] **Commit:** "SP01PH03T01: Registration endpoint created"
- [ ] **Status:** Pending

### SP01PH03T02 — Implement OTP generation
- [ ] **Task:** Generate 6-digit OTP, store in Redis with 15-min TTL. If SMTP fails, do NOT store OTP.
- [ ] **Acceptance:** OTP stored in Redis (key: otp:{email}, TTL: 15 min)
- [ ] **Commit:** "SP01PH03T02: OTP generation implemented"
- [ ] **Status:** Pending

### SP01PH03T03 — Implement OTP delivery (SMTP)
- [ ] **Task:** Send OTP via SMTP
- [ ] **Acceptance:** Email sent with OTP code
- [ ] **Commit:** "SP01PH03T03: OTP email delivery implemented"
- [ ] **Status:** Pending

### SP01PH03T04 — Implement login endpoint
- [ ] **Task:** POST /api/v1/login accepts email + OTP
- [ ] **Acceptance:** Returns session token on valid OTP
- [ ] **Commit:** "SP01PH03T04: Login endpoint created"
- [ ] **Status:** Pending

### SP01PH03T05 — Implement session validation middleware
- [ ] **Task:** Middleware validates session from Redis only, protects all /api/v1/* routes by default
- [ ] **Acceptance:** Returns 401 without valid session
- [ ] **Commit:** "SP01PH03T05: Session middleware created"
- [ ] **Status:** Pending

### SP01PH03T06 — Implement logout endpoint
- [ ] **Task:** DELETE /api/v1/logout invalidates session
- [ ] **Acceptance:** Session removed, returns 200
- [ ] **Commit:** "SP01PH03T06: Logout endpoint created"
- [ ] **Status:** Pending

### SP01PH03T07 — Implement /api/v1/me endpoint
- [ ] **Task:** GET /api/v1/me returns user info
- [ ] **Acceptance:** Returns user email when authenticated
- [ ] **Commit:** "SP01PH03T07: User info endpoint created"
- [ ] **Status:** Pending

### SP01PH03T08 — Add environment variable handling
- [ ] **Task:** Add SESSION_SECRET, SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASS, EMAIL_FROM_ADDRESS
- [ ] **Acceptance:** Startup fails if SESSION_SECRET missing
- [ ] **Commit:** "SP01PH03T08: Env vars configured"
- [ ] **Status:** Pending

---

## Phase 4: Frontend UI (PH04)

### SP01PH04T01 — Create login page
- [ ] **Task:** Login page with email input
- [ ] **Acceptance:** Email field, submit button visible
- [ ] **Commit:** "SP01PH04T01: Login page created"
- [ ] **Status:** Pending

### SP01PH04T02 — Create registration flow
- [ ] **Task:** Registration page with OTP entry
- [ ] **Acceptance:** OTP input field appears after email submit
- [ ] **Commit:** "SP01PH04T02: Registration flow created"
- [ ] **Status:** Pending

### SP01PH04T03 — Implement auth state management
- [ ] **Task:** Store session in HTTP-only cookie
- [ ] **Acceptance:** Cookie set on login, cleared on logout
- [ ] **Commit:** "SP01PH04T03: Auth state management implemented"
- [ ] **Status:** Pending

### SP01PH04T04 — Add logged-in indicator
- [ ] **Task:** Show user email when logged in
- [ ] **Acceptance:** Displays logged-in state in UI
- [ ] **Commit:** "SP01PH04T04: Logged-in indicator added"
- [ ] **Status:** Pending

### SP01PH04T05 — Add logout button
- [ ] **Task:** Logout button in UI
- [ ] **Acceptance:** Clicking logout clears session
- [ ] **Commit:** "SP01PH04T05: Logout button added"
- [ ] **Status:** Pending

---

## Phase 5: Security & Rate Limiting (PH05)

### SP01PH05T01 — Implement OTP rate limiting
- [ ] **Task:** Max 5 OTP requests per email per hour
- [ ] **Acceptance:** Returns 429 when exceeded
- [ ] **Commit:** "SP01PH05T01: OTP rate limiting implemented"
- [ ] **Status:** Pending

### SP01PH05T02 — Implement login rate limiting
- [ ] **Task:** Max 10 login attempts per IP per 10 min
- [ ] **Acceptance:** Returns 429 when exceeded
- [ ] **Commit:** "SP01PH05T02: Login rate limiting implemented"
- [ ] **Status:** Pending

### SP01PH05T03 — Add security headers
- [ ] **Task:** Add CSRF, security headers
- [ ] **Acceptance:** Headers present in responses
- [ ] **Commit:** "SP01PH05T03: Security headers added"
- [ ] **Status:** Pending

### SP01PH05T04 — Add abuse logging
- [ ] **Task:** Log rate-limit violations (IP, endpoint, timestamp) without PII
- [ ] **Acceptance:** Logs include violation details for later analysis
- [ ] **Commit:** "SP01PH05T04: Abuse logging added"
- [ ] **Status:** Pending

---

## Phase 6: Deployment & Testing (PH06)

### SP01PH06T01 — Run migrations on Railway
- [ ] **Task:** Apply migrations to Railway Postgres
- [ ] **Acceptance:** Tables created in production
- [ ] **Commit:** "SP01PH06T01: Migrations applied to Railway"
- [ ] **Status:** Pending

### SP01PH06T02 — Configure environment variables
- [ ] **Task:** Add SESSION_SECRET, SMTP_* to Railway
- [ ] **Acceptance:** Variables set, app starts
- [ ] **Commit:** "SP01PH06T02: Environment configured"
- [ ] **Status:** Pending

### SP01PH06T03 — Verify production auth flow
- [ ] **Task:** Test full registration/login flow in production
- [ ] **Acceptance:** All endpoints work in production
- [ ] **Commit:** "SP01PH06T03: Production auth verified"
- [ ] **Status:** Pending

---

## Phase 7: QA Validation (PH07)

### SP01PH07T01 — QA login from clean browser
- [ ] **Task:** QA tests from incognito browser
- [ ] **Acceptance:** Login works from fresh session
- [ ] **Status:** Pending

### SP01PH07T02 — QA logout invalidates session
- [ ] **Task:** QA logs out, tries to access /api/v1/me
- [ ] **Acceptance:** Returns 401 after logout
- [ ] **Status:** Pending

### SP01PH07T03 — QA expired OTP rejection
- [ ] **Task:** QA tests with expired OTP
- [ ] **Acceptance:** Rejected with clear message
- [ ] **Status:** Pending

### SP01PH07T04 — QA unauthorized access returns 401
- [ ] **Task:** QA accesses /api/v1/me without session
- [ ] **Acceptance:** Returns 401
- [ ] **Status:** Pending

### SP01PH07T05 — QA rate limiting
- [ ] **Task:** QA triggers rate limit
- [ ] **Acceptance:** Returns 429
- [ ] **Status:** Pending

### SP01PH07T06 — QA OTP reuse fails
- [ ] **Task:** QA attempts to use same OTP twice
- [ ] **Acceptance:** Second attempt fails
- [ ] **Status:** Pending

### SP01PH07T07 — QA 24-hour hard cap enforced
- [ ] **Task:** QA confirms session expires after 24h regardless of activity
- [ ] **Acceptance:** Session invalid after 24h
- [ ] **Status:** Pending

### SP01PH07T08 — QA middleware coverage
- [ ] **Task:** QA attempts access to various /api/v1/* endpoints without auth
- [ ] **Acceptance:** All return 401
- [ ] **Status:** Pending

### SP01PH07T09 — QA sign-off
- [ ] **Task:** QA produces pass report
- [ ] **Acceptance:** All scenarios pass including hardening validations
- [ ] **Status:** Pending

---

## Phase 8: Merge & Close (PH08)

### SP01PH08T01 — Final CI check
- [ ] **Task:** Ensure CI passes
- [ ] **Acceptance:** All tests green
- [ ] **Status:** Pending

### SP01PH08T02 — Merge to master
- [ ] **Task:** Merge sp01-account-auth to master
- [ ] **Commit:** "SP01PH08T02: Merged to master"
- [ ] **Status:** Pending

### SP01PH08T03 — Verify production
- [ ] **Task:** Confirm production works post-merge
- [ ] **Acceptance:** Production URL stable
- [ ] **Status:** Pending

### SP01PH08T04 — Mark spec closed
- [ ] **Task:** Update SP01.md status to Closed
- [ ] **Commit:** "SP01PH08T04: Spec closed"
- [ ] **Status:** Pending

---

## Summary

| Phase | Tasks | Completed |
|-------|-------|----------|
| PH00 | 3 | 3/3 |
| PH01 | 6 | 5/6 |
| PH02 | 3 | 0/3 |
| PH03 | 8 | 0/8 |
| PH04 | 5 | 0/5 |
| PH05 | 4 | 0/4 |
| PH06 | 3 | 0/3 |
| PH07 | 9 | 0/9 |
| PH08 | 4 | 0/4 |
| **Total** | **45** | **5/45** |

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
