# Git and Railway CLI Commands Reference

> **Constitutional Note (VIII.4):** Before any branch promotion, verify your environment with `railway status`. Environment ambiguity is prohibited.

## Part 1: Git Commands for Pushing to Different Branches

### Check Current Branch
```powershell
git branch
```
Shows which branch you're currently on (marked with *).

### View All Branches (Local and Remote)
```powershell
git branch -a
```

### Switch to a Branch
```powershell
git checkout staging
git checkout master
git checkout sp02-session-proxy
```

### Push Local Branch to Remote (Different Branch Names)

**Option A: Push to same-named branch on remote**
```powershell
git push origin staging
git push origin master
```
This pushes your current branch to a branch with the same name on origin.

**Option B: Push to different branch name on remote**
```powershell
git push origin local-branch:remote-branch
```
Examples:
```powershell
# Push sp02-session-proxy branch to staging branch on remote
git push origin sp02-session-proxy:staging

# Push staging branch to production branch on remote  
git push origin staging:production

# Push current branch to master on remote
git push origin HEAD:master
```

### Common Git Workflow
```powershell
# 1. Make sure you're on the right branch
git checkout sp02-session-proxy

# 2. Commit your changes
git add .
git commit -m "Description of changes"

# 3. Push to remote
git push origin sp02-session-proxy

# 4. If you need to update a different branch:
git push origin sp02-session-proxy:staging
```

---

## Part 2: Railway CLI Commands

### Check Current Environment
```powershell
railway status
```
Output shows:
```
Project: mudpuppy
Environment: production  (or staging)
Service: MudPuppy
```

### List All Environments
```powershell
railway environment list
```

### Switch to a Different Environment
```powershell
railway link
```
This will prompt you to select which environment to link to:
1. Select your project (mudpuppy)
2. Select the environment (production or staging)
3. Select the service

### Deploy (Trigger Deployment)
```powershell
# Deploys current directory to the linked environment
railway up
```

### View Deployments
```powershell
railway deployment list
```

### View Logs
```powershell
railway logs
```

### Switch Environment Example
```powershell
# 1. First check current environment
railway status

# 2. Unlink current
railway unlink

# 3. Link to different environment
railway link
# Follow prompts to select staging environment

# 4. Now deploy to staging
railway up
```

---

## Summary

### The Command Used in This Phase
```powershell
git push origin sp02-session-proxy:staging
```

This takes the local branch `sp02-session-proxy` and pushes it to the remote `origin`, but instead of using the same name, it updates the remote branch called `staging`. 

The syntax is: `git push origin SOURCE_BRANCH:TARGET_BRANCH`

### Key Insight
- **Git branches** are just pointers to commits
- When you push `local-branch:remote-branch`, you're telling Git "take the commit that local-branch points to and make remote-branch point to the same commit"
- Railway deployments are triggered by Git branch pushes to the linked branch
