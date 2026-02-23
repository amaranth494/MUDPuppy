# SP03 Tasks Tracker

> **Constitutional Reference:** This task tracker is governed by Constitution VIII. Deployment Governance & Environment Integrity.

**Spec:** SP03 — Persistent App Shell & Connections Manager  
**Reference:** [.specify/specs/SP03.md](../specs/SP03.md)  
**Reference:** [.specify/plans/SP03-plan.md](../plans/SP03-plan.md)

---

## Phase 0: Branch Creation (PH00)

### SP03PH00T01 — Create spec branch
- [x] **Task:** Create branch `sp03-persistent-shell-connections` from master
- [x] **Commit:** "SP03PH00: Branch created"
- [x] **Status:** Complete

### SP03PH00T02 — Commit spec document
- [x] **Task:** Commit SP03.md, SP03-plan.md, SP03-tasks.md to branch
- [x] **Commit:** "SP03PH00: Branch created"
- [x] **Status:** Complete

### SP03PH00T03 — Push to origin
- [x] **Task:** Push branch to origin
- [x] **Commit:** "SP03PH00: Branch pushed"
- [x] **Status:** Complete

### SP03PH00T04 — Verify CI passes
- [x] **Task:** Verify CI passes on branch
- [x] **Acceptance:** CI green
- [x] **Commit:** N/A
- [x] **Status:** Complete

### SP03PH00T05 — Phase 0 complete
- [x] **Task:** Confirm Phase 0 complete before feature work
- [x] **Status:** Complete

---

## Phase 1: Persistent App Shell (PH01)

### SP03PH01T01 — App Root Restructure
- [x] **Task:** Restructure App.tsx to mount Play at root level
- **Acceptance:** Session persists across route changes
- [x] **Commit:** "SP03PH01T01: App shell restructured"
- [x] **Status:** Complete

### SP03PH01T02 — Session State Lift
- [x] **Task:** Lift session state to application root context
- **Acceptance:** SessionContext available globally
- [x] **Commit:** "SP03PH01T02: Session state lifted to root"
- [x] **Status:** Complete

### SP03PH01T03 — Session Badge Component
- [x] **Task:** Create persistent session status badge
- **States:** Connected, Connecting, Disconnected
- **Source of Truth:** Must derive from GET /api/v1/session/status (Constitution IX)
- **Must NOT:** Infer solely from WebSocket open/close events
- **Acceptance:** Badge visible in all views
- [x] **Commit:** "SP03PH01T03: Session badge component created"
- [x] **Status:** Complete

---

## Phase 2: Sidebar Drawer (PH02)

### SP03PH02T01 — Sidebar Component
- [x] **Task:** Create collapsible sidebar component
- **Structure:** Logo, Play, Connections, Help, User Menu, Account, Sign Off
- [x] **Commit:** "SP03PH02T01: Sidebar component created"
- [x] **Status:** Complete

### SP03PH02T02 — Drawer Toggle Logic
- [x] **Task:** Implement open/close drawer state management
- **Acceptance:** Toggle has no session impact
- [x] **Commit:** "SP03PH02T02: Drawer toggle logic implemented"
- [x] **Status:** Complete

### SP03PH02T03 — Collapsed Icon Mode
- [x] **Task:** Implement collapsed state showing icons only
- **Storage:** Ephemeral (in-memory only); must NOT persist to localStorage (per Constitution V.a)
- [x] **Commit:** "SP03PH02T03: Collapsed icon mode implemented"
- [x] **Status:** Complete

### SP03PH02T04 — Header Removal
- [x] **Task:** Remove old header component
- **Prerequisite:** Relocate SessionBadge into Sidebar/AppShell before removing Header (SessionBadge currently imported by Header.tsx)
- **Acceptance:** Header component removed; SessionBadge remains visible in Sidebar/AppShell
- [x] **Commit:** "SP03PH02T04: Header component removed"
- [x] **Status:** Complete

---

## Phase 3: Modal Workspace Container (PH03)

### SP03PH03T01 — Modal Container Component
- [ ] **Task:** Create reusable modal overlay component
- **Features:** Full-height, close button, ESC support
- [ ] **Commit:** "SP03PH03T01: Modal container created"
- [ ] **Status:** Pending

### SP03PH03T02 — Input Lock Mechanism
- [ ] **Task:** Implement terminal input lock when modal open
- **UX Discipline:** Only command submission is locked; terminal rendering and text selection (Ctrl+C) must still work
- **Overlay Copy Behavior:** While modal overlay is open, allow selection/copy from terminal if overlay doesn't cover it; keep copy behavior sane
- **Acceptance:** No keystrokes leak to MUD; copy/selection still works
- [ ] **Commit:** "SP03PH03T02: Input lock mechanism implemented"
- [ ] **Status:** Pending

### SP03PH03T03 — Focus Restoration
- [ ] **Task:** Restore terminal focus on modal close
- [ ] **Commit:** "SP03PH03T03: Focus restoration implemented"
- [ ] **Status:** Pending

### SP03PH03T04 — High Load Handling
- [ ] **Task:** Ensure modal handles high ANSI bursts without crashing
- [ ] **Commit:** "SP03PH03T04: High load handling tested"
- [ ] **Status:** Pending

---

## Phase 4: Quick Connect (PH04)

### SP03PH04T01 — Quick Connect Modal UI
- [ ] **Task:** Create Quick Connect form (Host, Port fields)
- **Acceptance:** Modal opens from Play screen
- [ ] **Commit:** "SP03PH04T01: Quick Connect modal UI created"
- [ ] **Status:** Pending

### SP03PH04T02 — Connect Flow Integration
- [ ] **Task:** Integrate Quick Connect with existing session API
- **Acceptance:** Uses backend control-plane flow
- [ ] **Commit:** "SP03PH04T02: Connect flow integrated"
- [ ] **Status:** Pending

### SP03PH04T03 — Disconnect Integration
- [ ] **Task:** Add disconnect button to Quick Connect/Play
- [ ] **Commit:** "SP03PH04T03: Disconnect integration completed"
- [ ] **Status:** Pending

---

## Phase 5: Backend Migrations + Connections CRUD (PH05)

### SP03PH05T01 — Connections Table Migration
- [ ] **Task:** Create migration for connections table
- **Fields:** id, user_id, name, host, port, protocol, created_at, updated_at, last_connected_at
- **Storage:** Postgres
- [ ] **Commit:** "SP03PH05T01: Connections table migration created"
- [ ] **Status:** Pending

### SP03PH05T02 — Connections API Endpoints
- [ ] **Task:** Implement CRUD API endpoints for connections
- **Endpoints:** POST/GET/PUT/DELETE /api/v1/connections
- **Acceptance:** No secrets stored
- [ ] **Commit:** "SP03PH05T02: Connections API endpoints implemented"
- [ ] **Status:** Pending

### SP03PH05T03 — Last Connected Tracking
- [ ] **Task:** Update last_connected_at on successful connection
- [ ] **Commit:** "SP03PH05T03: Last connected tracking added"
- [ ] **Status:** Pending

### SP03PH05T04 — Recent Connections Query
- [ ] **Task:** Implement endpoint to fetch recent connections
- **Definition:** Recent connections = top 5 connections ordered by last_connected_at DESC where last_connected_at IS NOT NULL
- [ ] **Commit:** "SP03PH05T04: Recent connections query implemented"
- [ ] **Status:** Pending

---

## Phase 6: Connections Hub UI (PH06)

### SP03PH06T01 — Connections List View
- [ ] **Task:** Create saved connections list in modal
- **Acceptance:** Displays all user connections
- [ ] **Commit:** "SP03PH06T01: Connections list view created"
- [ ] **Status:** Pending

### SP03PH06T02 — Create Connection Form
- [ ] **Task:** Create form for new connection
- **Fields:** Name, Host, Port, Protocol (default telnet)
- [ ] **Commit:** "SP03PH06T02: Create connection form created"
- [ ] **Status:** Pending

### SP03PH06T03 — Edit Connection Form
- [ ] **Task:** Create form for editing existing connection
- [ ] **Commit:** "SP03PH06T03: Edit connection form created"
- [ ] **Status:** Pending

### SP03PH06T04 — Delete Connection
- [ ] **Task:** Implement delete with confirmation
- **Rule:** Does not affect active session
- [ ] **Commit:** "SP03PH06T04: Delete connection implemented"
- [ ] **Status:** Pending

### SP03PH06T05 — Connect from Hub
- [ ] **Task:** Add connect button to saved connections
- [ ] **Commit:** "SP03PH06T05: Connect from hub implemented"
- [ ] **Status:** Pending

### SP03PH06T06 — Recent Connections Display
- [ ] **Task:** Show recent connections from server
- [ ] **Commit:** "SP03PH06T06: Recent connections display added"
- [ ] **Status:** Pending

---

## Phase 7: QA Phase (PH07)

### SP03PH07T01 — Session Continuity Tests
- [ ] **Task:** QA validates session persists across navigation
- **Tests:** Open Help, Connections, Account - session stays connected
- **Acceptance:** Scrollback intact; no reconnect
- [ ] **Commit:** N/A
- [ ] **Status:** Pending

### SP03PH07T02 — Connection CRUD Tests
- [ ] **Task:** QA validates create, edit, delete operations
- **Tests:**
  - Create new connection
  - Edit existing connection
  - Delete connection
  - Verify persistence across login/logout
- [ ] **Commit:** N/A
- [ ] **Status:** Pending

### SP03PH07T03 — Quick Connect Tests
- [ ] **Task:** QA validates connect/disconnect flow
- **Tests:**
  - Connect via Quick Connect
  - Disconnect and reconnect
  - Session state matches backend
- [ ] **Commit:** N/A
- [ ] **Status:** Pending

### SP03PH07T04 — Drawer Toggle Tests
- [ ] **Task:** QA validates drawer open/close behavior
- **Tests:**
  - Open/close drawer
  - Verify no session impact
  - Verify collapsed state works
- [ ] **Commit:** N/A
- [ ] **Status:** Pending

### SP03PH07T05 — Modal Input Lock Tests
- [ ] **Task:** QA validates no keystrokes leak while modal open
- **Tests:**
  - Open modal
  - Type commands
  - Verify commands don't reach MUD
  - Close modal
  - Verify terminal input restored
- [ ] **Commit:** N/A
- [ ] **Status:** Pending

### SP03PH07T06 — Error Handling Tests
- [ ] **Task:** QA validates error handling
- **Tests:**
  - Invalid host error message displayed
  - Invalid port error message displayed
  - **Deny-listed port blocked with clear error**
  - **Private IP attempt blocked with error**
  - **Error messages distinguish: Invalid input, Deny-listed port, Private IP blocked, DNS failure, Timeout**
- [ ] **Commit:** N/A
- [ ] **Status:** Pending

### SP03PH07T07 — QA Sign-Off
- [ ] **Task:** Produce pass/fail report referencing SP03 task IDs
- **Acceptance:** All tests pass
- [ ] **Commit:** N/A
- [ ] **Status:** Pending

---

## Phase 8: Final Phase - Merge & Deployment (PH08)

> **Constitutional Note (VIII):** All promotions must follow: sp03-persistent-shell-connections → staging → master. No direct Railway CLI deployment except emergency.

### SP03PH08T01 — Code Review
- [ ] **Task:** Complete code review
- **Constitutional Check:** Verify implementation matches SP03.md
- [ ] **Acceptance:** All comments resolved
- [ ] **Commit:** N/A
- [ ] **Status:** Pending

### SP03PH08T02 — Merge to Staging
- [ ] **Task:** Merge sp03-persistent-shell-connections → staging
- **Constitutional Verification (VIII.4):**
  - [ ] `git branch` → confirm on sp03-persistent-shell-connections
  - [ ] `railway status` → confirm staging environment
- **Command:** `git push origin sp03-persistent-shell-connections:staging`
- [ ] **Acceptance:** CI passes on staging
- [ ] **Commit:** "SP03PH08T02: Merged to staging"
- [ ] **Status:** Pending

### SP03PH08T03 — Staging Validation
- [ ] **Task:** Verify SP03 features work in staging environment
- **QA Required:** All acceptance criteria verified in staging
- **Acceptance:** All acceptance criteria verified in staging
- [ ] **Commit:** N/A
- [ ] **Status:** Pending

### SP03PH08T04 — Merge to Master
- [ ] **Task:** Merge staging → master
- **Constitutional Requirements (VIII.5):**
  - [ ] All SP03 tasks complete
  - [ ] QA phase complete
  - [ ] Acceptance criteria verified
- **Command:** `git push origin staging:master`
- [ ] **Acceptance:** CI passes on master
- [ ] **Commit:** "SP03PH08T04: Merged to master"
- [ ] **Status:** Pending

### SP03PH08T05 — Production Verification
- [ ] **Task:** Verify SP03 features work in production
- **Acceptance:** All acceptance criteria verified in production
- [ ] **Commit:** N/A
- [ ] **Status:** Pending

### SP03PH08T06 — Spec Closed
- [ ] **Task:** Update SP03.md status to Closed
- [ ] **Commit:** "SP03PH08T06: Spec closed"
- [ ] **Status:** Pending

---

## Summary

| Phase | Tasks | Status |
|-------|-------|--------|
| PH00 | 5 | 5/5 |
| PH01 | 3 | 3/3 |
| PH02 | 4 | 4/4 |
| PH03 | 4 | 0/4 |
| PH04 | 3 | 0/3 |
| PH05 | 4 | 0/4 |
| PH06 | 6 | 0/6 |
| PH07 | 7 | 0/7 |
| PH08 | 6 | 0/6 |
| **Total** | **42** | **15/42** |

---

**Document Version:** 1.0.0  
**Last Updated:** 2026-02-23
