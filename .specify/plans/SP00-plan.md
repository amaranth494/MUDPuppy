# SP00 — Environment & Infrastructure Foundation

## Plan

This plan outlines the execution path for SP00, establishing the MVP runway.

---

## Phase 0: Branch Creation (PH00)

### Tasks:
- [ ] Create GitHub repository
- [ ] Create branch: `sp00-environment-foundation`
- [ ] Create initial README.md with project overview
- [ ] Commit SP00.md to branch
- [ ] Push branch to origin

---

## Phase 1: Local Environment (PH01)

### Tasks:
- [ ] Clone repository locally (if needed)
- [ ] Install Node.js (LTS)
- [ ] Create .nvmrc file with Node version
- [ ] Document Go version requirement
- [ ] Initialize Go project: `go mod init mudpuppy`
- [ ] **Backend:** Create basic server with PORT handling
  - Read PORT from environment variable
  - Default to 8080 locally
  - Create /health endpoint returning { "status": "ok" }
- [ ] **Frontend:** Create package.json with scripts
  - npm run dev (dev server)
  - npm run build (production build)
  - npm start (serve built output)
- [ ] Verify both can run locally
- [ ] Commit with message: "SP00PH01: Local environment initialized"

---

## Phase 2: GitHub + Railway Setup (PH02)

### Tasks:
- [ ] Push code to GitHub (if not already)
- [ ] Create Railway account
- [ ] Create Railway project
- [ ] Link GitHub repository to Railway
- [ ] Configure automatic deploys
- [ ] Create .github/workflows/ci.yml with basic build steps
- [ ] Commit with message: "SP00PH02: GitHub and Railway configured"

---

## Phase 3: Data Stores (PH03)

### Tasks:
- [ ] Provision PostgreSQL plugin in Railway
- [ ] Provision Redis plugin in Railway
- [ ] Add DATABASE_URL to Railway environment
- [ ] Add REDIS_URL to Railway environment
- [ ] Update backend to:
  - Read DATABASE_URL and REDIS_URL from environment
  - Fail startup if DB connection fails
  - Log explicit success message on connection
- [ ] Deploy and verify logs show successful connections
- [ ] Commit with message: "SP00PH03: Data stores provisioned"

---

## Phase 4: Deployment Verification (PH04)

### Tasks:
- [ ] Verify /health endpoint in backend (already created in PH01)
- [ ] Create simple index.html for frontend
- [ ] Deploy to Railway
- [ ] Verify production URL responds
- [ ] Verify /health endpoint returns JSON
- [ ] Verify Railway PORT handling works (Railway injects PORT)
- [ ] Commit with message: "SP00PH04: Deployment verified"

---

## Phase 5: CI Gate Verification (PH05)

### Tasks:
- [ ] Update .github/workflows/ci.yml to:
  - Use Node version from .nvmrc
  - Use specified Go version
  - Run on PR and push to master
- [ ] Add npm install, npm run build steps
- [ ] Add go build step
- [ ] Add placeholder test step
- [ ] Enable branch protection on master
- [ ] Create test PR to verify CI gate works
- [ ] Commit with message: "SP00PH05: CI gate verified"

---

## Phase 6: QA Validation (PH06)

### Tasks:
- [ ] Deploy latest to staging
- [ ] QA accesses production URL in browser
- [ ] QA verifies /health endpoint returns ok
- [ ] QA checks:
  - No console errors in browser
  - HTTPS certificate valid (padlock icon)
  - No mixed-content warnings
- [ ] QA reports: "Environment operational"

---

## Phase 7: Merge & Close (PH07)

### Tasks:
- [ ] Merge sp00-environment-foundation to master
- [ ] Verify master branch deploys successfully
- [ ] Mark SP00 as "Closed"
- [ ] Create SP01 — (next spec)

---

## Success Criteria

| Criterion | Verification |
|-----------|--------------|
| Node version pinned | .nvmrc exists |
| Go version documented | README or version file |
| Go module initialized | go.mod exists |
| PORT handling | Server reads PORT env, defaults to 8080 |
| Frontend scripts | npm run dev, npm run build, npm start work |
| Local run works | npm run dev + go run main.go |
| Auto-deploy works | git push triggers Railway deploy |
| Production URL | https://mudpuppy.up.railway.app |
| Health endpoint | /health returns { status: "ok" } |
| Postgres connected | Backend logs "Connected to Postgres" |
| Redis connected | Backend logs "Connected to Redis" |
| DB fail-fast | Backend fails to start if DB unavailable |
| CI uses pinned versions | CI uses .nvmrc Node, documented Go |
| CI builds pass | GitHub Actions shows green checkmark |
| QA checks pass | No console errors, HTTPS valid |

---

## Notes

- This plan produces no user-facing features
- Focus is on operational runway only
- Keep commits atomic and descriptive
- Each phase should be independently verifiable
