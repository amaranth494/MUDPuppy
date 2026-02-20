# SP00PH02 Railway Setup Instructions

This document provides step-by-step instructions to complete SP00PH02T02, SP00PH02T03, and SP00PH02T04.

---

## Prerequisites

- GitHub account with access to the MudPuppy repository
- Railway account (sign up at https://railway.app)

---

## SP00PH02T02 — Create Railway Project

### Steps:

1. **Navigate to Railway**
   - Open https://railway.app in your browser
   - Log in to your Railway account

2. **Create New Project**
   - Click the **"New Project"** button
   - Select **"Empty Project"** (or "Create New" → "Empty Project")
   - Name the project: `mudpuppy`
   - Click **"Create Project"**

3. **Note the Project**
   - Once created, note the project URL or keep the tab open
   - You will link the GitHub repo in the next step

---

## SP00PH02T03 — Link GitHub to Railway

### Steps:

1. **Access Project Settings**
   - In your Railway project dashboard, click the **"Settings"** tab (gear icon)
   - Or navigate to Project Settings

2. **Connect GitHub Repository**
   - Under **"GitHub"** section, click **"Connect GitHub Repo"**
   - Search for `MudPuppy` in the repository list
   - Click on `amaranth494/MudPuppy` to connect it

3. **Verify Connection**
   - The repository should now appear as connected
   - You should see the branch `sp00-environment-foundation` available

---

## SP00PH02T04 — Enable Automatic Deploys

### Steps:

1. **Navigate to Deployments**
   - In Railway project dashboard, click the **"Deploy"** tab

2. **Configure Automatic Deployments**
   - Under **"GitHub"** settings in Railway:
     - Ensure the branch `sp00-environment-foundation` is selected
     - Toggle **"Automatic Deploys"** to ON
   - Or: When you push to the branch, Railway should automatically detect and deploy

3. **Verify First Deployment**
   - Trigger a deployment by pushing a commit (if not already triggered)
   - Watch the deployment progress in the Railway dashboard
   - Ensure the build completes successfully

4. **Note Production URL**
   - After deployment, Railway provides a URL (e.g., `https://mudpuppy-production.up.railway.app`)
   - This URL will be needed for later phases

---

## Verification Checklist

After completing all three tasks, verify:

- [ ] Railway project `mudpuppy` exists
- [ ] GitHub repository `amaranth494/MudPuppy` is linked
- [ ] Automatic deploys are enabled
- [ ] Deployment to Railway succeeded
- [ ] Production URL is accessible

---

## Next Steps

Once Railway setup is complete, proceed to:

- **SP00PH03** — Data Stores (PostgreSQL + Redis provisioning)
- The backend code is already configured to read `DATABASE_URL` and `REDIS_URL` environment variables
