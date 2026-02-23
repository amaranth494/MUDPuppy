# SP03 — Persistent App Shell & Connections Manager

## Plan

> **Constitutional Reference:** This plan implements SP03 and is governed by Constitution VIII (Deployment Governance).

This plan outlines the execution path for SP03, establishing a persistent application shell and Connections Manager MVP.

---

## Phase 0: Branch Creation (PH00)

### Tasks:
- [ ] Create branch: `sp03-persistent-shell-connections`
- [ ] Ensure spec document SP03.md exists
- [ ] Commit to branch
- [ ] Push to origin
- [ ] Verify CI passes on branch

---

## Phase 1: Persistent App Shell (PH01)

### SP03PH01T01 — App Root Restructure
- [ ] **Task:** Restructure App.tsx to mount Play at root level
- **Acceptance:** Session persists across route changes
- [ ] **Commit:** "SP03PH01T01: App shell restructured"

### SP03PH01T02 — Session State Lift
- [ ] **Task:** Lift session state to application root context
- **Acceptance:** SessionContext available globally
- [ ] **Commit:** "SP03PH01T02: Session state lifted to root"

### SP03PH01T03 — Session Badge Component
- [ ] **Task:** Create persistent session status badge
- **States:** Connected, Connecting, Disconnected
- **Source of Truth:** Must derive from GET /api/v1/session/status (Constitution IX)
- **Must NOT:** Infer solely from WebSocket open/close events
- **Acceptance:** Badge visible in all views
- [ ] **Commit:** "SP03PH01T03: Session badge component created"

---

## Phase 2: Sidebar Drawer (PH02)

### SP03PH02T01 — Sidebar Component
- [ ] **Task:** Create collapsible sidebar component
- **Structure:** Logo, Play, Connections, Help, User Menu, Account, Sign Off
- [ ] **Commit:** "SP03PH02T01: Sidebar component created"

### SP03PH02T02 — Drawer Toggle Logic
- [ ] **Task:** Implement open/close drawer state management
- **Acceptance:** Toggle has no session impact
- [ ] **Commit:** "SP03PH02T02: Drawer toggle logic implemented"

### SP03PH02T03 — Collapsed Icon Mode
- [ ] **Task:** Implement collapsed state showing icons only
- **Storage:** Ephemeral (in-memory only); must NOT persist to localStorage (per Constitution V.a)
- [ ] **Commit:** "SP03PH02T03: Collapsed icon mode implemented"

### SP03PH02T04 — Header Removal
- [ ] **Task:** Remove old header component
- [ ] **Commit:** "SP03PH02T04: Header component removed"

---

## Phase 3: Modal Workspace Container (PH03)

### SP03PH03T01 — Modal Container Component
- [ ] **Task:** Create reusable modal overlay component
- **Features:** Full-height, close button, ESC support
- [ ] **Commit:** "SP03PH03T01: Modal container created"

### SP03PH03T02 — Input Lock Mechanism
- [ ] **Task:** Implement terminal input lock when modal open
- **Acceptance:** No keystrokes leak to MUD
- [ ] **Commit:** "SP03PH03T02: Input lock mechanism implemented"

### SP03PH03T03 — Focus Restoration
- [ ] **Task:** Restore terminal focus on modal close
- [ ] **Commit:** "SP03PH03T03: Focus restoration implemented"

### SP03PH03T04 — High Load Handling
- [ ] **Task:** Ensure modal handles high ANSI bursts without crashing
- [ ] **Commit:** "SP03PH03T04: High load handling tested"

---

## Phase 4: Quick Connect (PH04)

### SP03PH04T01 — Quick Connect Modal UI
- [ ] **Task:** Create Quick Connect form (Host, Port fields)
- **Acceptance:** Modal opens from Play screen
- [ ] **Commit:** "SP03PH04T01: Quick Connect modal UI created"

### SP03PH04T02 — Connect Flow Integration
- [ ] **Task:** Integrate Quick Connect with existing session API
- **Acceptance:** Uses backend control-plane flow
- [ ] **Commit:** "SP03PH04T02: Connect flow integrated"

### SP03PH04T03 — Disconnect Integration
- [ ] **Task:** Add disconnect button to Quick Connect/Play
- [ ] **Commit:** "SP03PH04T03: Disconnect integration completed"

---

## Phase 5: Backend Migrations + Connections CRUD (PH05)

### SP03PH05T01 — Connections Table Migration
- [ ] **Task:** Create migration for connections table
- **Fields:** id, user_id, name, host, port, protocol, created_at, updated_at, last_connected_at
- **Storage:** Postgres
- [ ] **Commit:** "SP03PH05T01: Connections table migration created"

### SP03PH05T02 — Connections API Endpoints
- [ ] **Task:** Implement CRUD API endpoints for connections
- **Endpoints:** POST/GET/PUT/DELETE /api/v1/connections
- **Acceptance:** No secrets stored
- [ ] **Commit:** "SP03PH05T02: Connections API endpoints implemented"

### SP03PH05T03 — Last Connected Tracking
- [ ] **Task:** Update last_connected_at on successful connection
- [ ] **Commit:** "SP03PH05T03: Last connected tracking added"

### SP03PH05T04 — Recent Connections Query
- [ ] **Task:** Implement endpoint to fetch recent connections
- **Definition:** Recent connections = top 5 connections ordered by last_connected_at DESC where last_connected_at IS NOT NULL
- [ ] **Commit:** "SP03PH05T04: Recent connections query implemented"

---

## Phase 6: Connections Hub UI (PH06)

### SP03PH06T01 — Connections List View
- [ ] **Task:** Create saved connections list in modal
- **Acceptance:** Displays all user connections
- [ ] **Commit:** "SP03PH06T01: Connections list view created"

### SP03PH06T02 — Create Connection Form
- [ ] **Task:** Create form for new connection
- **Fields:** Name, Host, Port, Protocol (default telnet)
- [ ] **Commit:** "SP03PH06T02: Create connection form created"

### SP03PH06T03 — Edit Connection Form
- [ ] **Task:** Create form for editing existing connection
- [ ] **Commit:** "SP03PH06T03: Edit connection form created"

### SP03PH06T04 — Delete Connection
- [ ] **Task:** Implement delete with confirmation
- **Rule:** Does not affect active session
- [ ] **Commit:** "SP03PH06T04: Delete connection implemented"

### SP03PH06T05 — Connect from Hub
- [ ] **Task:** Add connect button to saved connections
- [ ] **Commit:** "SP03PH06T05: Connect from hub implemented"

### SP03PH06T06 — Recent Connections Display
- [ ] **Task:** Show recent connections from server
- [ ] **Commit:** "SP03PH06T06: Recent connections display added"

---

## Phase 7: QA Phase (PH07)

### SP03PH07T01 — Session Continuity Tests
- [ ] **Task:** QA validates session persists across navigation
- **Tests:** Open Help, Connections, Account - session stays connected
- [ ] **Commit:** N/A

### SP03PH07T02 — Connection CRUD Tests
- [ ] **Task:** QA validates create, edit, delete operations
- [ ] **Commit:** N/A

### SP03PH07T03 — Quick Connect Tests
- [ ] **Task:** QA validates connect/disconnect flow
- [ ] **Commit:** N/A

### SP03PH07T04 — Drawer Toggle Tests
- [ ] **Task:** QA validates drawer open/close behavior
- [ ] **Commit:** N/A

### SP03PH07T05 — Modal Input Lock Tests
- [ ] **Task:** QA validates no keystrokes leak while modal open
- [ ] **Commit:** N/A

### SP03PH07T06 — Error Handling Tests
- [ ] **Task:** QA validates invalid host/port handling
- [ ] **Commit:** N/A

### SP03PH07T07 — QA Sign-Off
- [ ] **Task:** Produce pass/fail report
- [ ] **Acceptance:** All tests pass; report references SP03 task IDs

---

## Phase 8: Final Phase - Merge & Deployment (PH08)

### SP03PH08T01 — Code Review
- [ ] **Task:** Complete code review
- **Constitutional Check:** Verify implementation matches SP03.md
- [ ] **Acceptance:** All comments resolved

### SP03PH08T02 — Merge to Staging
- [ ] **Task:** Merge sp03-persistent-shell-connections → staging
- **Constitutional Verification:**
  - `git branch` → confirm on sp03-persistent-shell-connections
  - `railway status` → confirm staging environment
- [ ] **Acceptance:** CI passes on staging

### SP03PH08T03 — Staging Validation
- [ ] **Task:** Validate SP03 features work in staging
- **Acceptance:** All acceptance criteria verified

### SP03PH08T04 — Merge to Master
- [ ] **Task:** Merge staging → master
- **Constitutional Requirements:**
  - All SP03 tasks complete
  - QA phase complete
  - Acceptance criteria verified
- [ ] **Acceptance:** CI passes; Railway auto-deploy triggers

### SP03PH08T05 — Production Verification
- [ ] **Task:** Verify SP03 features work in production
- **Acceptance:** Production healthy; no regressions

### SP03PH08T06 — Spec Closed
- [ ] **Task:** Update SP03.md status to Closed
- [ ] **Commit:** "SP03PH08T06: Spec closed"

---

## Summary

| Phase | Tasks | Description |
|-------|-------|-------------|
| PH00 | 5 | Branch creation and CI verification |
| PH01 | 3 | Persistent App Shell |
| PH02 | 4 | Sidebar Drawer |
| PH03 | 4 | Modal Workspace Container |
| PH04 | 3 | Quick Connect |
| PH05 | 4 | Backend Migrations + Connections CRUD |
| PH06 | 6 | Connections Hub UI |
| PH07 | 7 | QA Phase |
| PH08 | 6 | Merge and Deployment |
| **Total** | **42** | |

---

## Deployment Governance (Constitution VIII)

All promotions must follow:
1. `sp03-persistent-shell-connections` → `staging`
2. `staging` → `master`
3. No direct Railway CLI deployment except emergency
4. All changes must pass CI before merge

---

**Document Version:** 1.0.0  
**Last Updated:** 2026-02-23
