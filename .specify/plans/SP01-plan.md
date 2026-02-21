# SP01 — Account Authentication & Session Foundation

## Plan

This plan outlines the execution path for SP01, introducing authenticated user accounts with persistent identity and server-managed session lifecycle.

---

## Phase 0: Branch Creation (PH00)

### Tasks:
- [ ] Create branch: `sp01-account-auth`
- [ ] Create spec document SP01.md
- [ ] Commit to branch
- [ ] Push to origin

---

## Phase 1: Database Schema (PH01)

### SP01PH01T01 — Create users table
- [ ] **Task:** Create users table migration
- **Fields:** id, email, created_at, updated_at
- [ ] **Commit:** "SP01PH01T01: Users table created"

### SP01PH01T02 — Verify migrations locally
- [ ] **Task:** Run migrations locally, confirm users table exists
- [ ] **Acceptance:** Table created without error
- [ ] **Commit:** "SP01PH01T02: Migrations verified"

---

## Phase 2: Redis Configuration (PH02)

### SP01PH02T01 — Create Redis utility module
- [ ] **Task:** Create Go module with key builders, TTL constants, getters/setters
- **Key formats:**
  - OTP: `otp:email:{sha256(lower(email))}`
  - Session: `session:{sessionID}` (hard cap)
  - Session idle: `session_idle:{sessionID}` (sliding)
- **Acceptance:** Utility module exists, no auth routes yet
- [ ] **Commit:** "SP01PH02T01: Redis utility module created"

### SP01PH02T02 — Define OTP TTL
- [ ] **Task:** Implement OTP storage with SET key EX 900 (atomic, 15 min)
- [ ] **Acceptance:** OTP keys expire in 15 minutes
- [ ] **Commit:** "SP01PH02T02: OTP TTL implemented"

### SP01PH02T03 — Define Session TTL (dual-key)
- [ ] **Task:** Implement session with two keys:
  - `session:{id}` with EX 86400 (24h hard cap)
  - `session_idle:{id}` with EX 1800 (30min sliding, updated on activity)
- [ ] **Acceptance:** Middleware checks both keys exist
- [ ] **Commit:** "SP01PH02T03: Session TTL implemented"

### SP01PH02T04 — Add Redis health check
- [ ] **Task:** Add startup check: fail-fast if REDIS_URL missing/unreachable
- [ ] **Acceptance:** App panics on missing Redis at startup
- [ ] **Commit:** "SP01PH02T04: Redis health check added"

---

## Phase 3: Authentication Backend (PH03)

### SP01PH03T01 — Implement email registration endpoint
- [ ] **Task:** POST /api/v1/register accepts email
- [ ] **Acceptance:** Returns 201 on success
- [ ] **Commit:** "SP01PH03T01: Registration endpoint created"

### SP01PH03T02 — Implement OTP generation and storage
- [ ] **Task:** Generate 6-digit OTP, store in Redis with 15-min TTL
- [ ] **Acceptance:** OTP stored in Redis (key: otp:{email}, TTL: 15 min). If SMTP fails, do NOT store OTP, return 503.
- [ ] **Commit:** "SP01PH03T02: OTP generation implemented"

### SP01PH03T03 — Implement OTP delivery (SMTP)
- [ ] **Task:** Send OTP via SMTP
- [ ] **Acceptance:** Email sent with OTP code
- [ ] **Commit:** "SP01PH03T03: OTP email delivery implemented"

### SP01PH03T04 — Implement login endpoint
- [ ] **Task:** POST /api/v1/login accepts email + OTP
- [ ] **Acceptance:** Returns session token on valid OTP
- [ ] **Commit:** "SP01PH03T04: Login endpoint created"

### SP01PH03T05 — Implement session validation middleware
- [ ] **Task:** Middleware validates session from Redis only, protects all /api/v1/* routes by default
- [ ] **Acceptance:** Returns 401 without valid session
- [ ] **Commit:** "SP01PH03T05: Session middleware created"

### SP01PH03T06 — Implement logout endpoint
- [ ] **Task:** DELETE /api/v1/logout invalidates session
- [ ] **Acceptance:** Session removed, returns 200
- [ ] **Commit:** "SP01PH03T06: Logout endpoint created"

### SP01PH03T07 — Implement /api/v1/me endpoint
- [ ] **Task:** GET /api/v1/me returns user info
- [ ] **Acceptance:** Returns user email when authenticated
- [ ] **Commit:** "SP01PH03T07: User info endpoint created"

### SP01PH03T08 — Add environment variable handling
- [ ] **Task:** Add SESSION_SECRET, SMTP_* vars
- [ ] **Acceptance:** Startup fails if SESSION_SECRET missing
- [ ] **Commit:** "SP01PH03T08: Env vars configured"

---

## Phase 4: Frontend UI (PH04)

### SP01PH04T01 — Create login page
- [ ] **Task:** Login page with email input
- [ ] **Acceptance:** Email field, submit button
- [ ] **Commit:** "SP01PH04T01: Login page created"

### SP01PH04T02 — Create registration flow
- [ ] **Task:** Registration page with OTP entry
- [ ] **Acceptance:** OTP input field appears after email submit
- [ ] **Commit:** "SP01PH04T02: Registration flow created"

### SP01PH04T03 — Implement auth state management
- [ ] **Task:** Store session in HTTP-only cookie
- [ ] **Acceptance:** Cookie set on login, cleared on logout
- [ ] **Commit:** "SP01PH04T03: Auth state management implemented"

### SP01PH04T04 — Add logged-in indicator
- [ ] **Task:** Show user email when logged in
- [ ] **Acceptance:** Displays logged-in state in UI
- [ ] **Commit:** "SP01PH04T04: Logged-in indicator added"

### SP01PH04T05 — Add logout button
- [ ] **Task:** Logout button in UI
- [ ] **Acceptance:** Clicking logout clears session
- [ ] **Commit:** "SP01PH04T05: Logout button added"

---

## Phase 5: Security & Rate Limiting (PH05)

### SP01PH05T01 — Implement OTP rate limiting
- [ ] **Task:** Max 5 OTP requests per email per hour
- [ ] **Acceptance:** Returns 429 when exceeded
- [ ] **Commit:** "SP01PH05T01: OTP rate limiting implemented"

### SP01PH05T02 — Implement login rate limiting
- [ ] **Task:** Max 10 login attempts per IP per 10 min
- [ ] **Acceptance:** Returns 429 when exceeded
- [ ] **Commit:** "SP01PH05T02: Login rate limiting implemented"

### SP01PH05T03 — Add security headers
- [ ] **Task:** Add CSRF, security headers
- [ ] **Acceptance:** Headers present in responses
- [ ] **Commit:** "SP01PH05T03: Security headers added"

### SP01PH05T04 — Add abuse logging
- [ ] **Task:** Log rate-limit violations (IP, endpoint, timestamp) without PII
- [ ] **Acceptance:** Logs include violation details for later analysis
- [ ] **Commit:** "SP01PH05T04: Abuse logging added"

---

## Phase 6: Deployment & Testing (PH06)

### SP01PH06T01 — Run migrations on Railway
- [ ] **Task:** Apply migrations to Railway Postgres
- [ ] **Acceptance:** Tables created in production
- [ ] **Commit:** "SP01PH06T01: Migrations applied to Railway"

### SP01PH06T02 — Configure environment variables
- [ ] **Task:** Add SESSION_SECRET, SMTP_* to Railway
- [ ] **Acceptance:** Variables set, app starts
- [ ] **Commit:** "SP01PH06T02: Environment configured"

### SP01PH06T03 — Verify production auth flow
- [ ] **Task:** Test full registration/login flow in production
- [ ] **Acceptance:** All endpoints work in production
- [ ] **Commit:** "SP01PH06T03: Production auth verified"

---

## Phase 7: QA Validation (PH07)

### SP01PH07T01 — QA login from clean browser
- [ ] **Task:** QA tests from incognito browser
- [ ] **Acceptance:** Login works from fresh session

### SP01PH07T02 — QA logout invalidates session
- [ ] **Task:** QA logs out, tries to access /api/v1/me
- [ ] **Acceptance:** Returns 401 after logout

### SP01PH07T03 — QA expired OTP rejection
- [ ] **Task:** QA tests with expired OTP
- [ ] **Acceptance:** Rejected with clear message

### SP01PH07T04 — QA unauthorized access returns 401
- [ ] **Task:** QA accesses /api/v1/me without session
- [ ] **Acceptance:** Returns 401

### SP01PH07T05 — QA rate limiting
- [ ] **Task:** QA triggers rate limit
- [ ] **Acceptance:** Returns 429

### SP01PH07T06 — QA OTP reuse fails
- [ ] **Task:** QA attempts to use same OTP twice
- [ ] **Acceptance:** Second attempt fails

### SP01PH07T07 — QA 24-hour hard cap enforced
- [ ] **Task:** QA confirms session expires after 24h regardless of activity
  - Temporarily lower hard TTL to ~60 seconds in staging
  - Confirm session dies even if idle key keeps refreshing
  - Restore TTL constant afterward
- [ ] **Acceptance:** Session invalid after hard cap

### SP01PH07T08 — QA middleware coverage
- [ ] **Task:** QA attempts access to various /api/v1/* endpoints without auth:
  - /api/v1/logout without session → 401
  - /api/v1/login with malformed payload → 400
  - Any stubbed /api/v1/* routes → 401
- [ ] **Acceptance:** All return proper status codes

### SP01PH07T09 — QA OTP concurrent reuse
- [ ] **Task:** Fire two concurrent login attempts with same OTP
- [ ] **Acceptance:** Only one succeeds, other fails cleanly (not 500)

### SP01PH07T10 — QA sign-off

### SP01PH07T09 — QA sign-off
- [ ] **Task:** QA produces pass report
- [ ] **Acceptance:** All scenarios pass including hardening validations

---

## Phase 8: Merge & Close (PH08)

### SP01PH08T01 — Final CI check
- [ ] **Task:** Ensure CI passes
- [ ] **Acceptance:** All tests green

### SP01PH08T02 — Merge to master
- [ ] **Task:** Merge sp01-account-auth to master
- [ ] **Commit:** "SP01PH08T02: Merged to master"

### SP01PH08T03 — Verify production
- [ ] **Task:** Confirm production works post-merge
- [ ] **Acceptance:** Production URL stable

### SP01PH08T04 — Mark spec closed
- [ ] **Task:** Update SP01.md status to Closed
- [ ] **Commit:** "SP01PH08T04: Spec closed"

### SP01PH08T05 — Establish Persistent Staging Branch
- [ ] **Task:** Convert sp01-account-auth into long-lived staging branch:
  1. Ensure sp01-account-auth is up-to-date
  2. Merge sp01-account-auth → master
  3. Create staging branch from master: git checkout -b staging
  4. Push staging: git push -u origin staging
  5. Delete sp01-account-auth (local + remote)
- [ ] **Acceptance:**
  - master reflects production state
  - staging exists as long-lived integration branch
  - Railway configured: master → Production, staging → Staging
  - Future specs branch from staging, not master
- [ ] **Commit:** "SP01PH08T05: Staging branch established"

---

## Success Criteria

| Criterion | Verification |
|-----------|--------------|
| User registration | Email → OTP → account created |
| Login flow | Email + OTP → session token |
| Logout | Session invalidated |
| Session persistence | Refresh maintains login |
| Auth middleware | Protected routes require auth |
| Rate limiting | 429 when exceeded |
| Redis sessions | Session stored in Redis |
| HTTP-only cookie | No LocalStorage |
| Environment vars | SESSION_SECRET required |
| Production deploy | All flows work |

---

## Notes

- This spec introduces identity and session scaffolding only
- Does not include MUD connections, automation, or WebSocket streams
- Sessions are Redis-backed (ephemeral, acceptable for MVP)
- SMTP credentials provided by owner for MVP
