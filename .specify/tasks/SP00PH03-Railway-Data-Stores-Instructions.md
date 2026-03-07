# SP00PH03 Railway Data Stores Instructions

This document provides step-by-step instructions to complete SP00PH03T01, SP00PH03T02, SP00PH03T03, and SP00PH03T05.

---

## Prerequisites

- Railway account with the `mudpuppy` project already created (from SP00PH02)
- GitHub repository linked to Railway
- Backend code deployed with DATABASE_URL and REDIS_URL connection logic

---

## SP00PH03T01 — Provision PostgreSQL

### Steps:

1. **Navigate to Railway Dashboard**
   - Open https://railway.app in your browser
   - Log in to your Railway account
   - Select the `mudpuppy` project

2. **Add PostgreSQL Plugin**
   - In the project dashboard, click the **"New"** button (or "+" icon)
   - Select **"Database"** → **"PostgreSQL"**
   - Click **"Add Plugin"** to confirm

3. **Wait for Provisioning**
   - Railway will provision a new PostgreSQL database
   - Wait for the plugin status to show as "Active" (green indicator)
   - This typically takes 1-2 minutes

4. **Verify DATABASE_URL**
   - Click on the PostgreSQL plugin in the plugins list
   - Go to the **"Variables"** tab
   - Verify that `DATABASE_URL` is automatically populated
   - The format should be: `postgres://username:password@hostname:port/database`

---

## SP00PH03T02 — Provision Redis

### Steps:

1. **Add Redis Plugin**
   - In the Railway project dashboard, click the **"New"** button (or "+" icon)
   - Select **"Database"** → **"Redis"**
   - Click **"Add Plugin"** to confirm

2. **Wait for Provisioning**
   - Railway will provision a new Redis instance
   - Wait for the plugin status to show as "Active" (green indicator)
   - This typically takes 1-2 minutes

3. **Verify REDIS_URL**
   - Click on the Redis plugin in the plugins list
   - Go to the **"Variables"** tab
   - Verify that `REDIS_URL` is automatically populated
   - The format should be: `redis://username:password@hostname:port`

---

## SP00PH03T03 — Configure Environment Variables

### Steps:

1. **Access Project Variables**
   - In Railway project dashboard, click the **"Variables"** tab
   - You should see both `DATABASE_URL` and `REDIS_URL` listed

2. **Add to Backend Service**
   - Ensure both variables are added to the main backend service:
     - Click on the backend service (the Go application)
     - Go to the **"Variables"** section
     - Add the following variables if not already present:
       - `DATABASE_URL` — should auto-populate from PostgreSQL plugin
       - `REDIS_URL` — should auto-populate from Redis plugin

3. **Important: Do NOT hardcode credentials**
   - All credentials should come from Railway's plugin system
   - Never commit database credentials to GitHub

---

## SP00PH03T05 — Verify Database Connection

### Steps:

1. **Deploy the Updated Code**
   - Ensure the backend code with database connection logic is pushed to GitHub
   - If automatic deploys are enabled, Railway will automatically deploy
   - Alternatively, manually trigger a deployment in the Railway dashboard

2. **Check Deployment Logs**
   - In Railway dashboard, go to **"Deployments"**
   - Click on the latest deployment
   - View the **"Logs"** tab

3. **Verify Success Messages**
   - Look for the following log messages:
     ```
     Connecting to PostgreSQL...
     Connected to PostgreSQL
     ```
     ```
     Connecting to Redis...
     Connected to Redis
     ```

4. **Verify Health Endpoint**
   - Once deployment completes, access the health endpoint:
     ```
     https://mudpuppy-production.up.railway.app/health
     ```
   - Should return: `{"status":"ok"}`

---

## Troubleshooting

### If PostgreSQL connection fails:

1. Check that DATABASE_URL is correctly set
2. Verify PostgreSQL plugin shows "Active" status
3. Check deployment logs for specific error messages
4. Redeploy the backend after confirming DATABASE_URL is set

### If Redis connection fails:

1. Check that REDIS_URL is correctly set
2. Verify Redis plugin shows "Active" status
3. Check deployment logs for specific error messages
4. Redeploy the backend after confirming REDIS_URL is set

### If server fails to start:

- The backend is designed to fail-fast if database connection fails
- Check the deployment logs for "Failed to connect to" errors
- Ensure DATABASE_URL and REDIS_URL are properly configured before deployment

---

## Verification Checklist

After completing all tasks, verify:

- [ ] PostgreSQL plugin is "Active" in Railway
- [ ] Redis plugin is "Active" in Railway
- [ ] DATABASE_URL is available in backend service variables
- [ ] REDIS_URL is available in backend service variables
- [ ] Deployment completes successfully
- [ ] Logs show "Connected to PostgreSQL"
- [ ] Logs show "Connected to Redis"
- [ ] /health endpoint returns {"status":"ok"}

---

## Next Steps

Once data stores are verified, proceed to:

- **SP00PH04** — Deployment Verification
  - Verify frontend landing page
  - Verify Railway PORT handling
  - Verify production URL