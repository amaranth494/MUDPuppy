# SP02 Tasks Tracker

> **Constitutional Reference:** This task tracker is governed by Constitution VIII. Deployment Governance & Environment Integrity.

**Spec:** SP02 — Session Proxy & Live Connectivity  
**Reference:** [.specify/specs/SP02.md](../specs/SP02.md)  
**Reference:** [.specify/plans/SP02-plan.md](../plans/SP02-plan.md)

---

## Phase 0: Branch Creation (PH00)

### SP02PH00T01 — Create spec branch
- [x] **Task:** Create branch `sp02-session-proxy` from staging
- [x] **Commit:** "SP02PH00: Branch created"
- [x] **Status:** Completed

### SP02PH00T02 — Commit spec document
- [x] **Task:** Commit SP02.md, SP02-plan.md, SP02-tasks.md to branch
- [x] **Commit:** "SP02PH00: Branch created"
- [x] **Status:** Completed

### SP02PH00T03 — Push branch
- [x] **Task:** Push branch to origin
- [x] **Commit:** N/A
- [x] **Status:** Completed

### SP02PH00T04 — Verify CI passes
- [x] **Task:** Verify CI passes on sp02-session-proxy branch before proceeding
- [x] **Commit:** N/A
- [x] **Status:** Completed

---

## Phase 1: Backend Connection Manager (PH01)

### SP02PH01T00 — Create saved_connections table
- [x] **Task:** Create migration for mud_saved_connections table (id, user_id, name, host, port, created_at, updated_at)
- **Storage:** Postgres (not Redis)
- [x] **Commit:** "SP02PH01T00: Saved connections table created"
- [x] **Status:** Completed

### SP02PH01T01 — TCP Dialer Implementation
- [x] **Task:** Implement outbound TCP connection to arbitrary host:port
- **Acceptance:** Backend can connect to test MUD server
- [x] **Commit:** "SP02PH01T01: TCP dialer implemented"
- [x] **Status:** Completed

### SP02PH01T02 — Port Whitelist Enforcement
- [x] **Task:** Implement port whitelist (default: 23)
- **Reject:** Ports not in whitelist with clear error
- [x] **Commit:** "SP02PH01T02: Port whitelist enforced"
- [x] **Status:** Completed

### SP02PH01T03 — Private IP Detection
- [x] **Task:** Implement private IP detection and blocking
- **Block:** 10.x.x.x, 172.16-31.x.x, 192.168.x.x, 127.x.x.x, localhost
- [x] **Commit:** "SP02PH01T03: Private IP blocking implemented"
- [x] **Status:** Completed

### SP02PH01T04 — Per-User Session Binding
- [x] **Task:** Bind each MUD connection to authenticated user session
- **Acceptance:** One connection per user enforced
- [x] **Commit:** "SP02PH01T04: Per-user session binding"
- [x] **Status:** Completed

### SP02PH01T05 — Idle Timer Enforcement
- [x] **Task:** Implement 30-minute idle timeout
- **Behavior:** No activity for 30 min → disconnect
- [x] **Commit:** "SP02PH01T05: Idle timeout enforced"
- [x] **Status:** Completed

### SP02PH01T06 — Hard Session Cap
- [x] **Task:** Implement 24-hour absolute session limit
- **Behavior:** 24 hours max → forced disconnect
- [x] **Commit:** "SP02PH01T06: Hard session cap enforced"
- [x] **Status:** Completed

### SP02PH01T07 — Connection Metadata Logging
- [x] **Task:** Log connection metadata (connect time, host, port, duration, disconnect reason)
- **Acceptance:** Logs visible, no PII logged
- [x] **Commit:** "SP02PH01T07: Connection metadata logging"
- [x] **Status:** Completed

### SP02PH01T08 — Local Test Verification
- [x] **Task:** Test backend connecting to real MUD server locally
- [x] **Acceptance:** Connect, send command, receive output, disconnect
- [x] **Commit:** "SP02PH01T08: Local test verification"
- [x] **Status:** Completed

---

## Phase 2: WebSocket Bridge (PH02)

> **Design Constraints:**
> - WebSocket must NOT dial TCP directly — must call into Session Manager
> - No new connection logic in handler layer — Handler → Manager only
> - Rate limiting at WebSocket ingress before manager write
> - Message size enforcement before forwarding to TCP
> - WebSocket close triggers proper session teardown
> - No cross-user leakage — session lookup uses authenticated user context only

### SP02PH02T01 — WSS Endpoint Implementation
- [x] **Task:** Implement WebSocket upgrade endpoint at /api/v1/session/stream
- **Acceptance:** Endpoint accepts WebSocket connections from authenticated users
- [x] **Commit:** "SP02PH02T01: WSS endpoint implemented"
- [x] **Status:** Completed

### SP02PH02T02 — Bidirectional Stream Relay
- [x] **Task:** Implement stream relay: MUD → client, client → MUD
- **Design Constraint:** WebSocket handler MUST call Session Manager methods — no direct TCP dialing
- **Acceptance:** Data flows both directions in real-time
- [x] **Commit:** "SP02PH02T02: Bidirectional stream relay"
- [x] **Status:** Completed

### SP02PH02T03 — Message Size Enforcement
- [x] **Task:** Implement 64KB message size limit per message
- **Design Constraint:** Enforcement must occur BEFORE forwarding to TCP (at WebSocket ingress)
- **Reject:** Messages exceeding limit, close connection
- [x] **Commit:** "SP02PH02T03: Message size limit enforced"
- [x] **Status:** Completed

### SP02PH02T04 — Flood Protection
- [x] **Task:** Implement rate limiting (max 10 commands/second per user)
- **Design Constraint:** Rate limiting enforced at WebSocket ingress before manager write
- **Enforcement Location:** WebSocket ingress layer (before reaching TCP writer)
- [x] **Acceptance:** Rapid commands throttled or rejected at ingress
- [x] **Commit:** "SP02PH02T04: Flood protection implemented"
- [x] **Status:** Completed

### SP02PH02T05 — Session Teardown
- [x] **Task:** Implement proper cleanup on WebSocket close
- **Design Constraint:** WebSocket close must trigger proper session teardown via Manager
- **Behavior:** MUD connection closed, resources freed
- [x] **Commit:** "SP02PH02T05: Session teardown implemented"
- [x] **Status:** Completed

### SP02PH02T06 — Multi-User Isolation Test
- [x] **Task:** Test with two simultaneous users
- **Design Constraint:** No cross-user leakage — session lookup must use authenticated user context only
- [x] **Acceptance:** No session crossover, independent connections
- [x] **Commit:** "SP02PH02T06: Multi-user isolation verified"
- [x] **Status:** Completed

---

## Phase 3: UI Implementation (PH03)

> **Guardrails:**
> - UI must implement contract in SP02.md (App shell + Play screen + state machine + errors)
> - NO feature creep: no macros, triggers, panels
> - Use REST control plane for connect/disconnect/status
> - Use WSS data plane for stream + command send
> - No browser-local persistence required for correctness

### SP02PH03T00 — Frontend Scaffold (React + TypeScript)
- [x] **Task:** Set up React + TypeScript frontend project
- **Framework:** React 18+, TypeScript 5+
- **Build Tool:** Vite (required for MVP speed, no Next.js for MVP)
- [x] **Commit:** "SP02PH03T00: Frontend scaffold created"
- [x] **Status:** Completed

### SP02PH03T01 — Global App Shell Components
- [x] **Task:** Implement top app bar, session status pill, account menu, primary navigation
- **Status Pill States:** Disconnected (gray), Connecting (yellow), Connected (green), Error (red)
- [x] **Commit:** "SP02PH03T01: App shell implemented"
- [x] **Status:** Completed

### SP02PH03T02 — Play Screen Connection Panel
- [x] **Task:** Implement connection panel with host, port (default 23), Connect/Disconnect buttons
- **Guardrail:** Use REST API for connect/disconnect (not WSS)
- **Guardrail:** No browser-local persistence — server-side only
- **State Management:**
  - GET /api/v1/session/status is authoritative for connection state
  - On page load: call GET /status
  - After Connect/Disconnect: call REST, then re-fetch /status
  - WSS starts only when status = "connected"
- **Acceptance:** Connect enabled when disconnected, Disconnect visible when connected
- [x] **Commit:** "SP02PH03T02: Connection panel implemented"
- [x] **Status:** Completed

### SP02PH03T03 — Play Screen Output Panel
- [x] **Task:** Implement scrollable terminal rendering area
- **Terminal Renderer:** xterm.js (required for MVP ANSI rendering)
- **Guardrail:** Use WSS for real-time data stream
- **Guardrail:** No local storage for output history
- **Acceptance:** Output displays in real-time, scrollable
- [x] **Commit:** "SP02PH03T03: Output panel implemented"
- [x] **Status:** Completed

### SP02PH03T04 — Command Input Row
- [x] **Task:** Implement command input with Enter-to-send
- **Guardrail:** Send commands via WSS data plane
- **Guardrail:** Input disabled when disconnected
- **Acceptance:** Input enabled when connected, disabled when disconnected
- [x] **Commit:** "SP02PH03T04: Command input implemented"
- [x] **Status:** Completed

### SP02PH03T05 — Error Messaging
- [x] **Task:** Implement user-facing error messages
- **Error Mapping Table (backend error → user message):**
  | Backend Error | User Message |
  |--------------|-------------|
  | "port not allowed" | "Port not allowed by whitelist" |
  | "private IP address" | "Private addresses not allowed" |
  | "user already has active session" | "You already have an active connection" |
  | "connection refused" | "Connection refused by server" |
  | "no such host" | "Could not resolve host" |
  | "i/o timeout" | "Connection timed out" |
  | "session expired" | "Session expired, please reconnect" |
- **Guardrail:** No feature creep — macros, triggers, panels not included
- [x] **Commit:** "SP02PH03T05: Error messaging implemented"
- [x] **Status:** Completed

### SP02PH03T06 — Disabled States Enforcement
- [x] **Task:** Ensure UI correctly disables/enables controls per state machine
- **State Authority:** GET /api/v1/session/status is authoritative
- **Acceptance:** All states from Constitution V.3 table implemented
- [x] **Commit:** "SP02PH03T06: Disabled states enforced"
- [x] **Status:** Completed

### SP02PH03T07 — Connections Page
- [x] **Task:** Implement saved connections page for managing saved connections
- **API Requirement:** Backend CRUD endpoints required (POST/GET/PUT/DELETE /api/v1/connections)
- **Storage:** Postgres mud_saved_connections table
- **Status:** **DEFERRED** — Requires PH03 backend API tasks (not currently in plan)
- [x] **Commit:** "SP02PH03T07: Connections page deferred"
- [x] **Status:** Completed (Deferred)

---

## Phase 4: Abuse & Resource Enforcement (PH04)

### SP02PH04T01 — Connection Limit Hooks
- [ ] **Task:** Implement per-user connection limits
- **Acceptance:** One active connection per user enforced
- [ ] **Commit:** "SP02PH04T01: Connection limits enforced"
- [ ] **Status:** Pending

### SP02PH04T02 — Soft Backpressure Strategy
- [ ] **Task:** Implement soft throttling instead of immediate disconnect
- **Strategy:** Drop excess → Send warning → Escalate only on sustained abuse
- **Acceptance:** Legitimate heavy ANSI bursts don't kill sessions; fast typists not booted
- [ ] **Commit:** "SP02PH04T02: Soft backpressure implemented"
- [ ] **Status:** Pending

### SP02PH04T03 — Observability Metrics (PRIORITY)
- [ ] **Task:** Add metrics: avg throughput, bytes/sec per user, commands/sec, idle patterns
- **Rationale:** Establish baseline BEFORE enforcing limits
- [ ] **Commit:** "SP02PH04T03: Observability metrics added"
- [ ] **Status:** Pending

### SP02PH04T04 — Slow Client Test
- [ ] **Task:** Test with simulated slow browser (network throttling)
- **Rationale:** Ensure server doesn't panic on slow WebSocket consumer
- **Acceptance:** Server handles slow client gracefully
- [ ] **Commit:** "SP02PH04T04: Slow client scenario tested"
- [ ] **Status:** Pending

### SP02PH04T05 — Fail-Fast Invalid Handling
- [ ] **Task:** Ensure invalid connections fail fast with clear errors
- [ ] **Acceptance:** No hanging connections, immediate feedback
- [ ] **Commit:** "SP02PH04T05: Fail-fast handling implemented"
- [ ] **Status:** Pending

---

## Phase 5: QA Phase (PH05)

### SP02PH05T01 — Browser-Only Test Suite
- [ ] **Task:** QA validates all functionality from browser only
- **No database/log access required**

#### Required Tests:
- [ ] Successful connect to test MUD
- [ ] Failed connect (invalid host)
- [ ] Failed connect (connection refused)
- [ ] Failed connect (timeout)
- [ ] Idle timeout at 30 minutes (may be simulated or documented)
- [ ] Hard cap at 24 hours (may be documented)
- [ ] Disconnect button functionality
- [ ] Rapid connect/disconnect sequences
- [ ] Two users simultaneously (isolation)
- [ ] Invalid host error message displayed
- [ ] Invalid port error message displayed
- [ ] Private IP attempt blocked with error

- [ ] **Commit:** "SP02PH05T01: Browser-only tests completed"
- [ ] **Status:** Pending

### SP02PH05T02 — QA Sign-Off
- [ ] **Task:** Produce QA pass report
- [ ] **Acceptance:** All tests pass, browser-only validation confirmed
- [ ] **Commit:** "SP02PH05T02: QA sign-off"
- [ ] **Status:** Pending

---

## Phase 6: Final Phase - Merge & Deployment (PH06)

> **Constitutional Note (VIII):** All promotions must follow: sp02-session-proxy → staging → master. No direct Railway CLI deployment except emergency.

### SP02PH06T01 — Code Review
- [ ] **Task:** Complete code review
- **Constitutional Check:** Verify implementation matches SP02.md spec
- [ ] **Acceptance:** All comments resolved
- [ ] **Commit:** N/A
- [ ] **Status:** Pending

### SP02PH06T02 — Merge to Staging
- [ ] **Task:** Merge sp02-session-proxy → staging
- **Constitutional Verification (VIII.4):**
  - [ ] `git branch` → confirm on sp02-session-proxy
  - [ ] `railway status` → confirm staging environment
- **Command:** `git push origin sp02-session-proxy:staging`
- [ ] **Acceptance:** CI passes on staging
- [ ] **Commit:** "SP02PH06T02: Merged to staging"
- [ ] **Status:** Pending

### SP02PH06T03 — Staging Validation
- [ ] **Task:** Verify SP02 features work in staging environment
- **QA Required:** All acceptance criteria verified in staging
- [ ] **Acceptance:** All acceptance criteria verified in staging
- [ ] **Commit:** N/A
- [ ] **Status:** Pending

### SP02PH06T04 — Merge to Master
- [ ] **Task:** Merge staging → master
- **Constitutional Requirements (VIII.5):**
  - [ ] All SP02 tasks complete
  - [ ] QA phase complete
  - [ ] Acceptance criteria verified
- **Command:** `git push origin staging:master`
- [ ] **Acceptance:** CI passes on master
- [ ] **Commit:** "SP02PH06T04: Merged to master"
- [ ] **Status:** Pending

### SP02PH06T05 — Production Verification
- [ ] **Task:** Verify SP02 features work in production
- [ ] **Acceptance:** All acceptance criteria verified in production
- [ ] **Commit:** N/A
- [ ] **Status:** Pending

### SP02PH06T06 — Spec Closed
- [ ] **Task:** Update SP02.md status to Closed
- [ ] **Commit:** "SP02PH06T06: Spec closed"
- [ ] **Status:** Pending

---

## Summary

| Phase | Tasks | Status |
|-------|-------|--------|
| PH00 | 5 | 5/5 |
| PH01 | 9 | 9/9 |
| PH02 | 6 | 6/6 |
| PH03 | 8 | 8/8 |
| PH04 | 4 | 0/4 |
| PH05 | 2 | 0/2 |
| PH06 | 6 | 0/6 |
| **Total** | **40** | **28/40** |
