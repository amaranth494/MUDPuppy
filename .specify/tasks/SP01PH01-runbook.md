# SP01PH01 - Database Schema & Migration Runbook

**Phase:** PH01 - Database Schema  
**Status:** Ready for Manual Execution

---

## Overview

This runbook documents the manual steps required to complete SP01PH01:
1. Run migrations on Railway Staging
2. Deploy to Railway Staging

---

## Prerequisites

Before proceeding, ensure you have:
- Railway CLI installed and authenticated (`railway login`)
- Access to the MUDPuppy Railway project
- DATABASE_URL for Railway Staging environment (from Railway dashboard)

---

## Step 1: Run Migrations on Railway Staging

### Option A: Using Railway CLI (Recommended)

1. **Login to Railway:**
   ```bash
   railway login
   ```

2. **Link to project:**
   ```bash
   railway link
   ```
   Select the MUDPuppy project and Staging environment.

3. **Install golang-migrate (if not already installed):**
   
   **macOS/Linux:**
   ```bash
   curl -L https://github.com/golang-migrate/migrate/releases/download/v4.18.1/migrate.linux-amd64.tar.gz | tar xz
   sudo mv migrate /usr/local/bin/
   ```
   
   **Windows (PowerShell):**
   ```powershell
   Invoke-WebRequest -Uri "https://github.com/golang-migrate/migrate/releases/download/v4.18.1/migrate.windows-amd64.zip" -OutFile "migrate.zip"
   Expand-Archive -Path "migrate.zip" -DestinationPath "C:\migrate"
   # Add C:\migrate to PATH
   ```

4. **Run migrations:**
   ```bash
   railway run migrate -path migrations -database "$DATABASE_URL" up
   ```
   
   Or set the DATABASE_URL first:
   ```bash
   export DATABASE_URL=$(railway variables get DATABASE_URL)
   migrate -path migrations -database "$DATABASE_URL" up
   ```

### Option B: Using npm script

1. **Set DATABASE_URL environment variable:**
   ```bash
   export DATABASE_URL="your-railway-staging-database-url"
   ```

2. **Run migrations:**
   ```bash
   npm run migrate:up
   ```

### Verification

After running migrations, verify the tables were created:

1. **Connect to the staging database:**
   ```bash
   railway run psql "$DATABASE_URL" -c "\dt"
   ```

2. **Expected output should show:**
   - `users` table
   - `otp_challenges` table

3. **Verify constraints:**
   ```bash
   railway run psql "$DATABASE_URL" -c "\d users"
   railway run psql "$DATABASE_URL" -c "\d otp_challenges"
   ```

---

## Step 2: Deploy to Railway Staging

### Method 1: Using Git Push (Automatic Deploy)

1. **Push changes to the feature branch:**
   ```bash
   git add .
   git commit -m "SP01PH01: Database schema and migrations"
   git push origin sp01-account-auth
   ```

2. **Railway will automatically deploy** when the branch is pushed. Check the Railway dashboard for deployment status.

### Method 2: Using Railway CLI (Manual Deploy)

1. **Deploy from current branch:**
   ```bash
   railway deploy
   ```

2. **Verify deployment:**
   ```bash
   railway open
   ```
   Or check the Railway dashboard.

---

## Verification Checklist

After completing both steps, verify:

- [ ] Migrations ran successfully (no errors)
- [ ] `users` table exists in staging database
- [ ] `otp_challenges` table exists in staging database
- [ ] Unique constraint on users.email works
- [ ] Indexes are created on email columns
- [ ] Staging app starts successfully
- [ ] `/health` endpoint returns 200 OK

---

## Rollback Procedure

If something goes wrong:

1. **Rollback migrations:**
   ```bash
   migrate -path migrations -database "$DATABASE_URL" down
   ```

2. **Redeploy previous version** from Railway dashboard or git history.

---

## Environment Variables Required

Ensure these are set in Railway Staging:

| Variable | Required | Description |
|----------|----------|-------------|
| `DATABASE_URL` | Yes | PostgreSQL connection string |
| `REDIS_URL` | Yes | Redis connection string |
| `PORT` | No | Server port (default: 8080) |

---

## Troubleshooting

### "FATAL: DATABASE_URL environment variable is required"
- Ensure DATABASE_URL is set in Railway Staging environment variables

### Migration fails with "relation already exists"
- Migrations may have already been applied. Check: `migrate -path migrations -database "$DATABASE_URL" version`

### Deployment fails
- Check Railway logs: `railway logs`
- Ensure all environment variables are set

---

## References

- [golang-migrate GitHub](https://github.com/golang-migrate/migrate)
- [Railway CLI Documentation](https://docs.railway.app/reference/cli)
- [SP01 Specification](../specs/SP01.md)
- [SP01 Tasks Tracker](./SP01-tasks.md)
