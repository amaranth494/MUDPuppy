# SP00 Tasks Tracker

**Spec:** SP00 — Environment & Infrastructure Foundation  
**Reference:** [.specify/specs/SP00.md](../specs/SP00.md)  
**Reference:** [.specify/plans/SP00-plan.md](../plans/SP00-plan.md)

---

## Phase 0: Branch Creation (PH00)

### SP00PH00T01 — Create GitHub repository
- [x] **Task:** Create new repository on GitHub
- [x] **Commit:** N/A (remote creation)
- [x] **Status:** Completed

### SP00PH00T02 — Create spec branch
- [x] **Task:** Create branch `sp00-environment-foundation` from master
- [x] **Commit:** "SP00PH00: Branch created"
- [x] **Status:** Completed

### SP00PH00T03 — Commit initial files
- [x] **Task:** Commit README.md and SP00.md to branch
- [x] **Commit:** "SP00PH00: Initial files committed"
- [x] **Status:** Completed

---

## Phase 1: Local Environment (PH01)

### SP00PH01T01 — Clone repository
- [x] **Task:** Clone repository to local machine
- [x] **Commit:** N/A
- [x] **Status:** Completed

### SP00PH01T02 — Install and pin Node.js
- [x] **Task:** Install Node.js LTS, create .nvmrc file
- [x] **Commit:** "SP00PH01T02: Node version pinned"
- [x] **Status:** Completed

### SP00PH01T03 — Document Go version
- [x] **Task:** Document Go version requirement in README
- [x] **Commit:** "SP00PH01T03: Go version documented"
- [x] **Status:** Completed

### SP00PH01T04 — Initialize Go module
- [x] **Task:** Run `go mod init mudpuppy`, create basic server
- [x] **Acceptance:** Server reads PORT from env, defaults to 8080 locally
- [x] **Acceptance:** /health endpoint returns { status: "ok" }
- [x] **Commit:** "SP00PH01T04: Go module initialized, server created"
- [x] **Status:** Completed

### SP00PH01T05 — Create frontend structure
- [x] **Task:** Create /src directory, package.json
- [x] **Acceptance:** npm run dev works locally
- [x] **Acceptance:** npm run build produces build output
- [x] **Acceptance:** npm start serves built output
- [x] **Commit:** "SP00PH01T05: Frontend structure created"
- [x] **Status:** Completed

### SP00PH01T06 — Verify local execution
- [x] **Task:** Run frontend and backend locally
- [x] **Acceptance:** npm run dev starts dev server
- [x] **Acceptance:** go run main.go starts backend
- [x] **Commit:** "SP00PH01T06: Local execution verified"
- [x] **Status:** Completed

---

## Phase 2: GitHub + Railway Setup (PH02)

### SP00PH02T01 — Push code to GitHub
- [ ] **Task:** Push local repo to GitHub origin
- [ ] **Commit:** "SP00PH02T01: Code pushed to GitHub"
- [ ] **Status:** Pending

### SP00PH02T02 — Create Railway project
- [ ] **Task:** Create Railway account and new project
- [ ] **Commit:** N/A (remote creation)
- [ ] **Status:** Pending

### SP00PH02T03 — Link GitHub to Railway
- [ ] **Task:** Connect GitHub repo in Railway dashboard
- [ ] **Commit:** N/A
- [ ] **Status:** Pending

### SP00PH02T04 — Enable automatic deploys
- [ ] **Task:** Configure automatic deployment in Railway
- [ ] **Commit:** N/A
- [ ] **Status:** Pending

### SP00PH02T05 — Create CI workflow
- [x] **Task:** Create .github/workflows/ci.yml
- [x] **Acceptance:** Uses Node version from .nvmrc
- [x] **Acceptance:** Uses specified Go version
- [x] **Commit:** "SP00PH02T05: CI workflow created"
- [x] **Status:** Completed

---

## Phase 3: Data Stores (PH03)

### SP00PH03T01 — Provision PostgreSQL
- [ ] **Task:** Add PostgreSQL plugin in Railway
- [ ] **Commit:** N/A
- [ ] **Status:** Pending

### SP00PH03T02 — Provision Redis
- [ ] **Task:** Add Redis plugin in Railway
- [ ] **Commit:** N/A
- [ ] **Status:** Pending

### SP00PH03T03 — Configure environment variables
- [ ] **Task:** Add DATABASE_URL, REDIS_URL to Railway
- [ ] **Commit:** N/A
- [ ] **Status:** Pending

### SP00PH03T04 — Update backend for DB connection
- [ ] **Task:** Modify main.go to read DATABASE_URL, REDIS_URL
- [ ] **Acceptance:** Backend fails startup if DB connection fails
- [ ] **Acceptance:** Backend logs explicit success message
- [ ] **Commit:** "SP00PH03T04: Database connections configured"
- [ ] **Status:** Pending

### SP00PH03T05 — Verify database connection
- [ ] **Task:** Deploy and check logs for success messages
- [ ] **Acceptance:** Logs show "Connected to Postgres"
- [ ] **Acceptance:** Logs show "Connected to Redis"
- [ ] **Commit:** "SP00PH03T05: Database connections verified"
- [ ] **Status:** Pending

---

## Phase 4: Deployment Verification (PH04)

### SP00PH04T01 — Create health endpoint
- [ ] **Task:** Verify /health endpoint exists (created in PH01)
- [ ] **Acceptance:** Returns { status: "ok" }
- [ ] **Commit:** N/A
- [ ] **Status:** Pending

### SP00PH04T02 — Create frontend landing
- [ ] **Task:** Create simple index.html
- [ ] **Commit:** "SP00PH04T02: Frontend landing created"
- [ ] **Status:** Pending

### SP00PH04T03 — Deploy to Railway
- [ ] **Task:** Push to trigger Railway deployment
- [ ] **Commit:** N/A
- [ ] **Status:** Pending

### SP00PH04T04 — Verify Railway PORT handling
- [ ] **Task:** Confirm Railway PORT variable works
- [ ] **Acceptance:** Server responds on Railway-assigned port
- [ ] **Commit:** N/A
- [ ] **Status:** Pending

### SP00PH04T05 — Verify production URL
- [ ] **Task:** Access production URL and /health endpoint
- [ ] **Acceptance:** Production URL returns frontend
- [ ] **Acceptance:** /health returns JSON ok status
- [ ] **Commit:** "SP00PH04T05: Production verified"
- [ ] **Status:** Pending

---

## Phase 5: CI Gate Verification (PH05)

### SP00PH05T01 — Configure CI workflow
- [ ] **Task:** Update ci.yml for PR + push triggers
- [ ] **Acceptance:** CI uses Node from .nvmrc
- [ ] **Acceptance:** CI uses documented Go version
- [ ] **Commit:** "SP00PH05T01: CI configured for version pinning"
- [ ] **Status:** Pending

### SP00PH05T02 — Add build steps
- [ ] **Task:** Add npm install, npm build, go build to ci.yml
- [ ] **Commit:** "SP00PH05T02: Build steps added"
- [ ] **Status:** Pending

### SP00PH05T03 — Enable branch protection
- [ ] **Task:** Protect master branch, require CI passing
- [ ] **Commit:** N/A
- [ ] **Status:** Pending

### SP00PH05T04 — Test CI gate
- [ ] **Task:** Create test PR, verify CI runs and blocks merge
- [ ] **Acceptance:** CI fails on broken code
- [ ] **Acceptance:** CI passes on valid code
- [ ] **Commit:** "SP00PH05T04: CI gate verified"
- [ ] **Status:** Pending

---

## Phase 6: QA Validation (PH06)

### SP00PH06T01 — Deploy to staging
- [ ] **Task:** Deploy latest to staging environment
- [ ] **Commit:** N/A
- [ ] **Status:** Pending

### SP00PH06T02 — QA browser verification
- [ ] **Task:** QA accesses production URL, verifies UI
- [ ] **Acceptance:** No console errors in browser
- [ ] **Commit:** N/A
- [ ] **Status:** Pending

### SP00PH06T03 — QA security checks
- [ ] **Task:** Verify HTTPS and no mixed-content
- [ ] **Acceptance:** HTTPS certificate valid (padlock icon)
- [ ] **Acceptance:** No mixed-content warnings
- [ ] **Commit:** N/A
- [ ] **Status:** Pending

### SP00PH06T04 — QA health endpoint
- [ ] **Task:** QA accesses /health, confirms ok status
- [ ] **Commit:** N/A
- [ ] **Status:** Pending

### SP00PH06T05 — QA sign-off
- [ ] **Task:** QA produces pass report referencing SP00
- [ ] **Commit:** N/A
- [ ] **Status:** Pending

---

## Phase 7: Merge & Close (PH07)

### SP00PH07T01 — Merge to master
- [ ] **Task:** Merge sp00-environment-foundation to master
- [ ] **Commit:** "SP00PH07T01: Merged to master"
- [ ] **Status:** Pending

### SP00PH07T02 — Verify master deploys
- [ ] **Task:** Confirm master branch deploys successfully
- [ ] **Commit:** N/A
- [ ] **Status:** Pending

### SP00PH07T03 — Mark spec closed
- [ ] **Task:** Update SP00.md status to "Closed"
- [ ] **Commit:** "SP00PH07T03: Spec closed"
- [ ] **Status:** Pending

---

## Summary

| Phase | Tasks | Completed |
|-------|-------|-----------|
| PH00 | 3 | 3/3 |
| PH01 | 6 | 6/6 |
| PH02 | 5 | 1/5 |
| PH03 | 5 | 0/5 |
| PH04 | 5 | 0/5 |
| PH05 | 4 | 0/5 |
| PH06 | 5 | 0/5 |
| PH07 | 3 | 0/3 |
| **Total** | **36** | **10/36** |

---

## Key Improvements Made

1. **PH00 now includes repo creation** - No more sequencing leak
2. **Frontend structure is specific** - dev/build/start scripts required
3. **PORT handling explicit** - Server must read PORT env, default to 8080
4. **DB verification strengthened** - Fails fast, explicit success logging
5. **QA checks expanded** - Console errors, HTTPS, mixed-content
6. **CI version pinning verified** - Uses .nvmrc and documented Go version

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
