# SP01PH07 — QA Validation Testing Guide

**Phase:** QA Validation (PH07)  
**Spec:** SP01 — Account Authentication & Session Foundation  
**Purpose:** Detailed guide for manual browser-based QA testing

---

## Prerequisites

### Test Environment
- **Staging URL:** `https://mudpuppy-staging.up.railway.app` (or your configured Railway staging URL)
- **Browser:** Use Chrome/Firefox in Incognito mode for each test to ensure clean state
- **Developer Tools:** Open browser DevTools (F12) → Network tab to inspect API requests

### Pre-Test Setup
1. Clear all cookies for the staging site before starting each test
2. Use Incognito/Private browser window for each test scenario
3. Note down the staging URL before beginning

---

## Test 1: SP01PH07T01 — Login from Clean Browser

**Objective:** Verify login works from a fresh/incognito browser session

### Steps
1. Open a new **Incognito/Private** browser window
2. Navigate to the staging URL: `https://mudpuppy-staging.up.railway.app`
3. You should see the login page with:
   - Email input field
   - "Register" link/button
4. Enter your test email address (e.g., `qa-test@example.com`)
5. Click "Send OTP" or "Register" button
6. **Check your email** for the OTP code (or check server logs in staging for the OTP)
7. Enter the 6-digit OTP in the provided field
8. Click "Login" or "Verify"
9. **Expected Result:** You should be logged in and see:
   - A welcome message or user email displayed
   - Logout button visible
   - Session cookie set (check DevTools → Application → Cookies)

### Validation Checklist
- [x] Login page loads without errors
- [x] Email submission triggers OTP send
- [x] OTP input field appears after email submission
- [x] Successful login shows authenticated state
- [x] No console errors in browser

---

## Test 2: SP01PH07T02 — Logout Invalidates Session

**Objective:** Verify that logging out immediately invalidates the session

### Prerequisites
- Complete Test 1 first (be logged in)

### Steps
1. After logging in, verify you can access `/api/v1/me`:
   - Open DevTools → Network tab
   - Click any authenticated action or refresh the page
   - Confirm you receive user data (not 401)
2. Click the **Logout** button in the UI
3. **Expected Result:**
   - You should be redirected to login page
   - Session cookie should be cleared (check DevTools → Application → Cookies)
4. Try to access an authenticated endpoint:
   - Open DevTools → Console
   - Run: `fetch('/api/v1/me').then(r => console.log(r.status))`
   - Or refresh the page
5. **Expected Result:** Returns `401 Unauthorized`

### Validation Checklist
- [x] Logout button is visible when logged in
- [x] Clicking logout shows login page
- [x] Session cookie is removed after logout
- [x] `/api/v1/me` returns 401 after logout
- [x] Cannot access protected pages after logout -- There are no pages

---

## Test 3: SP01PH07T03 — Expired OTP Rejection

**Objective:** Verify that an expired OTP is rejected with a clear error message

### Steps
1. Open a new Incognito window
2. Navigate to the staging site
3. Enter your test email and request an OTP
4. **Do NOT enter the OTP immediately**
5. **Wait 16 minutes** (OTP TTL is 15 minutes, so wait for it to expire)
6. After 16 minutes, enter the OTP that was sent
7. Click Login
8. **Expected Result:** Error message such as:
   - "Invalid or expired OTP"
   - "OTP has expired"
   - HTTP status 401

### Validation Checklist
- [x] OTP request succeeds initially
- [x] After waiting, OTP is rejected
- [FAIL] Error message is clear and user-friendly - Error is "Network error. Please try again." should be "OTP has expired, please request a new one."
- [x] No 500 Internal Server Error

---

## Test 4: SP01PH07T04 — Unauthorized Access Returns 401

**Objective:** Verify that accessing protected endpoints without authentication returns 401

### Steps
1. Open a new Incognito window (no login)
2. Try to access `/api/v1/me` directly:
   - In DevTools Console: `fetch('/api/v1/me').then(r => console.log(r.status, r.statusText))`
3. **Expected Result:** Returns `401 Unauthorized`
4. Try other protected endpoints (if available):
   - Any `/api/v1/*` endpoint that requires authentication
5. **Expected Result:** All return 401 when no session exists

### Validation Checklist
- [x] `/api/v1/me` returns 401 without session
- [x] Error response includes appropriate message
- [x] No sensitive data leaked in response

---

## Test 5: SP01PH07T05 — Rate Limiting

**Objective:** Verify that rate limiting returns 429 when limits are exceeded

### Test 5a: OTP Rate Limiting
**Limit:** 5 OTP requests per email per hour

### Steps
1. Open Incognito window
2. Navigate to login page
3. Enter email and request OTP **6 times in quick succession**
4. On the 6th attempt:
   - **Expected Result:** Returns `429 Too Many Requests`
   - Error message: "Too many OTP requests. Please try again later."

### Test 5b: Login Rate Limiting
**Limit:** 10 login attempts per IP per 10 minutes

### Steps
1. Use the same IP (or simulate different attempts)
2. Enter email + wrong OTP **11 times** in quick succession
3. On the 11th attempt:
   - **Expected Result:** Returns `429 Too Many Requests`
   - Error message: "Too many login attempts. Please try again later."

### Validation Checklist
- [x] OTP rate limit returns 429 after 5 requests
- [x] Login rate limit returns 429 after 10 attempts
- [x] Error messages are clear
- [x] Wait period allows requests again after limit expires

---

## Test 6: SP01PH07T06 — OTP Reuse Fails

**Objective:** Verify that an OTP cannot be used twice

### Steps
1. Open Incognito window
2. Navigate to login page
3. Enter email and request a new OTP
4. Note the OTP code (from email or logs)
5. Enter the OTP and successfully log in
6. **Now try to log in again with the same OTP:**
   - Click logout first
   - Enter same email
   - Enter the **same OTP** that was just used
7. **Expected Result:** 
   - Returns `401 Unauthorized`
   - Error message: "Invalid or expired OTP"

### Validation Checklist
- [x] First login with OTP succeeds
- [x] Second attempt with same OTP fails
- [x] Error message is clear
- [x] No 500 error

---

## Test 7: SP01PH07T07 — 24-Hour Hard Cap Enforced

**Objective:** Verify session expires after 24 hours regardless of activity

### Note for Testing
The 24-hour hard cap cannot be tested in real-time. This test requires code modification:

### Alternative Verification Steps
1. **Code Review:** Verify in [`internal/redis/ttl.go`](internal/redis/ttl.go:13) that `SessionTTL = 86400 * time.Second` (24 hours)
2. **Code Review:** Verify in [`internal/redis/client.go`](internal/redis/client.go:116) that session is stored with hard cap TTL
3. **Functional Test (simulated):**
   - Check DevTools → Application → Cookies
   - Note the `session_token` cookie has `Max-Age: 86400` (24 hours)
   - This confirms the cookie matches the server-side hard cap

### Validation Checklist
- [NOT TESTING] Session hard cap is 24 hours (86400 seconds) in code
- [NOT TESTING] Cookie Max-Age matches server TTL
- [NOT TESTING] Dual-key approach implemented (session + session_idle)

---

## Test 8: SP01PH07T08 — Middleware Coverage

**Objective:** Verify all `/api/v1/*` endpoints are protected by auth middleware

### Steps
1. **Test /logout without session:**
   - Open Incognito (no login)
   - Run: `fetch('/api/v1/logout', {method: 'DELETE'}).then(r => console.log(r.status))`
   - **Expected:** Returns `401 Unauthorized`

2. **Test /login with malformed payload:**
   - Open DevTools Console
   - Run: `fetch('/api/v1/login', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: '{}'}).then(r => console.log(r.status))`
   - **Expected:** Returns `400 Bad Request` (validation error)

3. **Test any stubbed routes:**
   - Try accessing any other `/api/v1/*` endpoint without auth
   - **Expected:** All return `401 Unauthorized`

### Validation Checklist
- [x] `/api/v1/logout` returns 401 without session
- [x] `/api/v1/login` with bad payload returns 400
- [x] Other protected routes return 401 without session

---

## Test 9: SP01PH07T09 — OTP Concurrent Reuse

**Objective:** Verify only one concurrent OTP login succeeds, other fails cleanly

### Steps
1. Open two different browser windows (or use two different browsers)
2. **In Window A:** Request OTP for test email
3. **In Window B:** Request OTP for same email (this will generate a new OTP)
4. **In Window A:** Try to login with the OTP you received
5. **In Window B:** Try to login with the OTP you received (from the second request)
6. **Expected Result:**
   - One login succeeds (200)
   - Other login fails with 401 (not 500)
   - No server errors

### Validation Checklist
- [x] Only one concurrent login attempt succeeds
- [x] Failed attempt returns 401 (not 500)
- [x] No server crashes or internal errors

---

## Test 10: SP01PH07T10 — QA Sign-Off

**Objective:** Final verification that all scenarios pass

### Final Checklist

| Scenario | Status | Notes |
|----------|--------|-------|
| Login from clean browser | [ ] | |
| Logout invalidates session | [ ] | |
| Expired OTP rejection | [ ] | |
| Unauthorized access 401 | [ ] | |
| OTP rate limiting (429) | [ ] | |
| Login rate limiting (429) | [ ] | |
| OTP reuse fails | [ ] | |
| 24-hour hard cap | [ ] | Code verified |
| Middleware coverage | [ ] | |
| Concurrent OTP reuse | [ ] | |

### Overall Acceptance
- [ ] All QA scenarios pass
- [ ] No console errors in browser
- [ ] No 500 Internal Server Errors
- [ ] All error messages are user-friendly
- [ ] Security headers present

---

## API Reference

### Endpoints

| Method | Path | Auth Required | Description |
|--------|------|---------------|-------------|
| POST | `/api/v1/register` | No | Register/Send OTP |
| POST | `/api/v1/login` | No | Login with OTP |
| DELETE | `/api/v1/logout` | Yes | Logout |
| GET | `/api/v1/me` | Yes | Get current user |
| GET | `/health` | No | Health check |

### Expected Response Codes

| Code | Meaning |
|------|---------|
| 200 | Success |
| 201 | Created |
| 400 | Bad Request (invalid input) |
| 401 | Unauthorized (not logged in) |
| 404 | Not Found |
| 429 | Too Many Requests (rate limited) |
| 500 | Internal Server Error |

### Error Response Format
```json
{
  "error": "Error message here"
}
```

---

## Troubleshooting

### OTP Not Received
- Check server logs for "STAGING: OTP sent" message
- In development mode, OTP is logged to console

### Session Not Working
- Check browser DevTools → Application → Cookies
- Verify `session_token` cookie exists and has correct attributes:
  - `HttpOnly: true`
  - `Secure: true` (on HTTPS)
  - `SameSite: Lax`

### Rate Limit Too Aggressive
- Wait for the TTL to expire (1 hour for OTP, 10 minutes for login)
- Use different email/IP to test

---

## Test Data

### Test Email
Use a disposable test email for QA:
- `qa-test@example.com` (replace with your test email)

### Staging Environment Variables (Reference)
```
DATABASE_URL=<provided>
REDIS_URL=<provided>
SESSION_SECRET=<must be set>
SMTP_HOST=smtp.gmail.com
SMTP_PORT=465
SMTP_USER=<provided>
SMTP_PASS=<provided>
EMAIL_FROM_ADDRESS=<provided>
```

---

## Report Template

### QA Test Results - SP01PH07

**Tester:** ________________  
**Date:** ________________  
**Environment:** Staging

| Test ID | Test Name | Pass/Fail | Notes |
|---------|-----------|-----------|-------|
| SP01PH07T01 | Login from clean browser | | |
| SP01PH07T02 | Logout invalidates session | | |
| SP01PH07T03 | Expired OTP rejection | | |
| SP01PH07T04 | Unauthorized access 401 | | |
| SP01PH07T05a | OTP rate limiting | | |
| SP01PH07T05b | Login rate limiting | | |
| SP01PH07T06 | OTP reuse fails | | |
| SP01PH07T07 | 24-hour hard cap | | |
| SP01PH07T08 | Middleware coverage | | |
| SP01PH07T09 | Concurrent OTP reuse | | |

**Overall Result:** ________________  
**Comments:** _______________________________________________________________
