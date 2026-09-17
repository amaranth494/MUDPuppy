# MUDPuppy Environment Variables

This document tracks all environment variables used across MUDPuppy's staging and production environments.

---

## Comparison: Staging vs Production

| Variable | Staging | Production | Backend/Frontend | Notes |
|----------|---------|------------|------------------|-------|
| **DATABASE_URL** (Public) | `postgresql://postgres:JnAnBqLIFBjkheWDeUlXPfGpeJIwqQDL@turntable.proxy.rlwy.net:54782/railway` | `postgresql://postgres:BmoCEzLZwfvCcKmCKbnFaRYtisIITfqa@yamabiko.proxy.rlwy.net:17074/railway` | Backend | PostgreSQL public URL |
| **DATABASE_URL** (Internal) | `postgresql://postgres:JnAnBqLIFBjkheWDeUlXPfGpeJIwqQDL@postgres.railway.internal:5432/railway` | `postgresql://postgres:BmoCEzLZwfvCcKmCKbnFaRYtisIITfqa@postgres.railway.internal:5432/railway` | Backend | PostgreSQL internal (Railway only) |
| **REDIS_URL** | `redis://default:nCLQMpLGeJHEGQcRRfrtRjmBOTnwfaDZ@redis.railway.internal:6379` | `redis://default:PfnViYrQUyQbnCYbVJoAMpylkKseMRsR@redis.railway.internal:6379` | Backend | Redis session store |
| **SESSION_SECRET** | `mudPuppy2026SecureSessionSecret32!` | `mudPuppy2026SecureSessionSecret32!` | Backend | Session authentication (shared) |
| **SMTP_HOST** | `smtp.gmail.com` | `smtp.gmail.com` | Backend | Email server (shared) |
| **SMTP_PORT** | `465` | `465` | Backend | Email port (shared) |
| **SMTP_USER** | `castle.and.clark@gmail.com` | `castle.and.clark@gmail.com` | Backend | Email credentials (shared) |
| **SMTP_PASS** | `esvb secr zohp hbyf` | `esvb secr zohp hbyf` | Backend | Email credentials (shared) |
| **EMAIL_FROM_ADDRESS** | `castle.and.clark@gmail.com` | `castle.and.clark@gmail.com` | Backend | From address (shared) |

---

## Backend-Only Variables (Not Environment-Specific)

These variables are configured in code but not currently tracked in Railway:

| Variable | Default | Required | Notes |
|----------|---------|----------|-------|
| **PORT** | `8080` | No | Server listen port |
| **OTP_EXPIRY_MINUTES** | `15` | No | OTP challenge expiry |
| **MUD_PROXY_PORT_WHITELIST** | `23` | No | Allowed MUD proxy ports |
| **IDLE_TIMEOUT_MINUTES** | `30` | No | Session idle timeout |
| **HARD_SESSION_CAP_HOURS** | `24` | No | Maximum session duration |
| **MAX_MESSAGE_SIZE_BYTES** | `65536` | No | Max WebSocket message size |
| **COMMAND_RATE_LIMIT_PER_SECOND** | `10` | No | Server-side rate limiting |
| **MUD_PORT_DENYLIST** | `25,465,587,110,143,993,995,53,80,443,1433,1521,3306,5432,6379,27017,22,3389,5900,445,139,2049` | No | Blocked ports for MUD proxy |
| **MUD_PORT_ALLOWLIST** | (empty) | No | Override whitelist - if set, ONLY these ports allowed |
| **ENCRYPTION_KEY_V1** | (none) | Yes (outside local development) | Credential encryption key v1. Outside local development (`RAILWAY_ENVIRONMENT` set) the server refuses to start without it. A missing key means the credential vault generates a new random key on every process start, and every previously stored credential becomes unreadable. On the owner's own machine (`RAILWAY_ENVIRONMENT` unset) the server still starts with this variable absent. |
| **ENCRYPTION_KEY_V2** | (none) | No | Credential encryption key v2 (optional rotation slot) |
| **ENCRYPTION_KEY_V3** | (none) | No | Credential encryption key v3 (optional rotation slot) |
| **ADMIN_METRICS_SECRET** | (none) | Yes (if used) | Secret for /api/v1/admin/metrics |

---

## Rotating the credential vault key

The credential vault (`ENCRYPTION_KEY_V1`/`V2`/`V3`) encrypts every saved MUD connection's stored credentials at rest. Rotate it with these steps:

1. Generate a new 32-byte key and base64-encode it (any offline tool that produces cryptographically random bytes is fine).
2. Set the new value as `ENCRYPTION_KEY_V2` (or the next free slot after V1/V2 are both in use) on the target environment. Do not overwrite `ENCRYPTION_KEY_V1` yet — reads must still succeed against credentials encrypted under the current key.
3. Redeploy so the new key is loaded into the running process's key store.
4. Re-save each stored connection's credentials through the app so they are re-encrypted under the newest key.
5. On a later deploy, once nothing is left encrypted under the retired key, remove that slot's environment variable.

Key values are never pasted into a plan, a summary, an evidence file, a log line, or a commit message — only the variable name and slot number are ever written down.

---

## Frontend Variables

The frontend is built as a static SPA and does not have runtime environment variables. API endpoints are configured in `frontend/src/services/api.ts`:

| Variable | Value | Notes |
|----------|-------|-------|
| **API_BASE** | `/api/v1` | Proxied through backend |

---

## Summary

### Shared Between Staging & Production (4 variables):
- SESSION_SECRET
- SMTP_HOST
- SMTP_PORT
- SMTP_USER
- SMTP_PASS
- EMAIL_FROM_ADDRESS

### Different Per Environment (2 variables):
- DATABASE_URL
- REDIS_URL

### Backend-Only (13 variables):
- PORT
- OTP_EXPIRY_MINUTES
- MUD_PROXY_PORT_WHITELIST
- IDLE_TIMEOUT_MINUTES
- HARD_SESSION_CAP_HOURS
- MAX_MESSAGE_SIZE_BYTES
- COMMAND_RATE_LIMIT_PER_SECOND
- MUD_PORT_DENYLIST
- MUD_PORT_ALLOWLIST
- ENCRYPTION_KEY_V1
- ENCRYPTION_KEY_V2
- ENCRYPTION_KEY_V3
- ADMIN_METRICS_SECRET

### Frontend-Only (0 variables):
- All API calls use relative paths proxied through the backend
