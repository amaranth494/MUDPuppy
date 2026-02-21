# SP02 Tasks Tracker

**Spec:** SP02 — Session Proxy & Live Connectivity  
**Reference:** [.specify/specs/SP02.md](../specs/SP02.md)  
**Reference:** [.specify/plans/SP02-plan.md](../plans/SP02-plan.md)

---

## Phase 0: Branch Creation (PH00)

### SP02PH00T01 — Create spec branch
- [ ] **Task:** Create branch `sp02-session-proxy` from staging
- [ ] **Commit:** "SP02PH00: Branch created"
- [ ] **Status:** Pending

### SP02PH00T02 — Commit spec document
- [ ] **Task:** Commit SP02.md, SP02-plan.md, SP02-tasks.md to branch
- [ ] **Commit:** "SP02PH00: Spec committed"
- [ ] **Status:** Pending

### SP02PH00T03 — Push branch
- [ ] **Task:** Push branch to origin
- [ ] **Commit:** N/A
- [ ] **Status:** Pending

### SP02PH00T04 — Verify CI passes
- [ ] **Task:** Verify CI passes on sp02-session-proxy branch before proceeding
- [ ] **Commit:** N/A
- [ ] **Status:** Pending

---

## Phase 1: Backend Connection Manager (PH01)

### SP02PH01T00 — Create saved_connections table
- [ ] **Task:** Create migration for mud_saved_connections table (id, user_id, name, host, port, created_at, updated_at)
- **Storage:** Postgres (not Redis)
- [ ] **Commit:** "SP02PH01T00: Saved connections table created"
- [ ] **Status:** Pending

### SP02PH01T01 — TCP Dialer Implementation
- [ ] **Task:** Implement outbound TCP connection to arbitrary host:port
- **Acceptance:** Backend can connect to test MUD server
- [ ] **Commit:** "SP02PH01T01: TCP dialer implemented"
- [ ] **Status:** Pending

### SP02PH01T02 — Port Whitelist Enforcement
- [ ] **Task:** Implement port whitelist (default: 23)
- **Reject:** Ports not in whitelist with clear error
- [ ] **Commit:** "SP02PH01T02: Port whitelist enforced"
- [ ] **Status:** Pending

### SP02PH01T03 — Private IP Detection
- [ ] **Task:** Implement private IP detection and blocking
- **Block:** 10.x.x.x, 172.16-31.x.x, 192.168.x.x, 127.x.x.x, localhost
- [ ] **Commit:** "SP02PH01T03: Private IP blocking implemented"
- [ ] **Status:** Pending

### SP02PH01T04 — Per-User Session Binding
- [ ] **Task:** Bind each MUD connection to authenticated user session
- **Acceptance:** One connection per user enforced
- [ ] **Commit:** "SP02PH01T04: Per-user session binding"
- [ ] **Status:** Pending

### SP02PH01T05 — Idle Timer Enforcement
- [ ] **Task:** Implement 30-minute idle timeout
- **Behavior:** No activity for 30 min → disconnect
- [ ] **Commit:** "SP02PH01T05: Idle timeout enforced"
- [ ] **Status:** Pending

### SP02PH01T06 — Hard Session Cap
- [ ] **Task:** Implement 24-hour absolute session limit
- **Behavior:** 24 hours max → forced disconnect
- [ ] **Commit:** "SP02PH01T06: Hard session cap enforced"
- [ ] **Status:** Pending

### SP02PH01T07 — Connection Metadata Logging
- [ ] **Task:** Log connection metadata (connect time, host, port, duration, disconnect reason)
- **Acceptance:** Logs visible, no PII logged
- [ ] **Commit:** "SP02PH01T07: Connection metadata logging"
- [ ] **Status:** Pending

### SP02PH01T08 — Local Test Verification
- [ ] **Task:** Test backend connecting to real MUD server locally
- [ ] **Acceptance:** Connect, send command, receive output, disconnect
- [ ] **Commit:** "SP02PH01T08: Local test verification"
- [ ] **Status:** Pending

---

## Phase 2: WebSocket Bridge (PH02)

### SP02PH02T01 — WSS Endpoint Implementation
- [ ] **Task:** Implement WebSocket upgrade endpoint at /api/v1/session/stream
- **Acceptance:** Endpoint accepts WebSocket connections from authenticated users
- [ ] **Commit:** "SP02PH02T01: WSS endpoint implemented"
- [ ] **Status:** Pending

### SP02PH02T02 — Bidirectional Stream Relay
- [ ] **Task:** Implement stream relay: MUD → client, client → MUD
- [ ] **Acceptance:** Data flows both directions in real-time
- [ ] **Commit:** "SP02PH02T02: Bidirectional stream relay"
- [ ] **Status:** Pending

### SP02PH02T03 — Message Size Enforcement
- [ ] **Task:** Implement 64KB message size limit per message
- **Reject:** Messages exceeding limit, close connection
- [ ] **Commit:** "SP02PH02T03: Message size limit enforced"
- [ ] **Status:** Pending

### SP02PH02T04 — Flood Protection
- [ ] **Task:** Implement rate limiting (max 10 commands/second per user)
- **Enforcement Location:** WebSocket ingress layer (before reaching TCP writer)
- [ ] **Acceptance:** Rapid commands throttled or rejected at ingress
- [ ] **Commit:** "SP02PH02T04: Flood protection implemented"
- [ ] **Status:** Pending

### SP02PH02T05 — Session Teardown
- [ ] **Task:** Implement proper cleanup on WebSocket close
- **Behavior:** MUD connection closed, resources freed
- [ ] **Commit:** "SP02PH02T05: Session teardown implemented"
- [ ] **Status:** Pending

### SP02PH02T06 — Multi-User Isolation Test
- [ ] **Task:** Test with two simultaneous users
- [ ] **Acceptance:** No session crossover, independent connections
- [ ] **Commit:** "SP02PH02T06: Multi-user isolation verified"
- [ ] **Status:** Pending

---

## Phase 3: UI Implementation (PH03)

### SP02PH03T00 — Frontend Scaffold (React + TypeScript)
- [ ] **Task:** Set up React + TypeScript frontend project
- **Framework:** React 18+, TypeScript 5+
- **Build Tool:** Vite (recommended) or Next.js
- [ ] **Commit:** "SP02PH03T00: Frontend scaffold created"
- [ ] **Status:** Pending

### SP02PH03T01 — Global App Shell Components
- [ ] **Task:** Implement top app bar, session status pill, account menu, primary navigation
- **Status Pill States:** Disconnected (gray), Connecting (yellow), Connected (green), Error (red)
- [ ] **Commit:** "SP02PH03T01: App shell implemented"
- [ ] **Status:** Pending

### SP02PH03T02 — Play Screen Connection Panel
- [ ] **Task:** Implement connection panel with host, port (default 23), Connect/Disconnect buttons
- **Acceptance:** Connect enabled when disconnected, Disconnect visible when connected
- [ ] **Commit:** "SP02PH03T02: Connection panel implemented"
- [ ] **Status:** Pending

### SP02PH03T03 — Play Screen Output Panel
- [ ] **Task:** Implement scrollable terminal rendering area
- **Acceptance:** Output displays in real-time, scrollable
- [ ] **Commit:** "SP02PH03T03: Output panel implemented"
- [ ] **Status:** Pending

### SP02PH03T04 — Command Input Row
- [ ] **Task:** Implement command input with Enter-to-send
- **Acceptance:** Input enabled when connected, disabled when disconnected
- [ ] **Commit:** "SP02PH03T04: Command input implemented"
- [ ] **Status:** Pending

### SP02PH03T05 — Error Messaging
- [ ] **Task:** Implement user-facing error messages
- **Errors:** Invalid host, connection refused, timeout, private IP, invalid port
- [ ] **Commit:** "SP02PH03T05: Error messaging implemented"
- [ ] **Status:** Pending

### SP02PH03T06 — Disabled States Enforcement
- [ ] **Task:** Ensure UI correctly disables/enables controls per state machine
- **Acceptance:** All states from Constitution V.3 table implemented
- [ ] **Commit:** "SP02PH03T06: Disabled states enforced"
- [ ] **Status:** Pending

### SP02PH03T07 — Connections Page
- [ ] **Task:** Implement saved connections page for managing saved connections
- **Features:** Add, edit, delete, connect from saved
- **Storage:** Postgres mud_saved_connections table
- [ ] **Commit:** "SP02PH03T07: Connections page implemented"
- [ ] **Status:** Pending

---

## Phase 4: Abuse & Resource Enforcement (PH04)

### SP02PH04T01 — Connection Limit Hooks
- [ ] **Task:** Implement per-user connection limits
- **Acceptance:** One active connection per user enforced
- [ ] **Commit:** "SP02PH04T01: Connection limits enforced"
- [ ] **Status:** Pending

### SP02PH04T02 — Throughput Limits
- [ ] **Task:** Implement throughput monitoring and limits
- [ ] **Acceptance:** Excessive throughput detected and throttled
- [ ] **Commit:** "SP02PH04T02: Throughput limits enforced"
- [ ] **Status:** Pending

### SP02PH04T03 — Observability Metrics
- [ ] **Task:** Add metrics for connection attempts, failures, durations
- **Acceptance:** Metrics visible in observability tooling
- [ ] **Commit:** "SP02PH04T03: Observability metrics added"
- [ ] **Status:** Pending

### SP02PH04T04 — Fail-Fast Invalid Handling
- [ ] **Task:** Ensure invalid connections fail fast with clear errors
- [ ] **Acceptance:** No hanging connections, immediate feedback
- [ ] **Commit:** "SP02PH04T04: Fail-fast handling implemented"
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

### SP02PH06T01 — Code Review
- [ ] **Task:** Complete code review
- [ ] **Acceptance:** All comments resolved
- [ ] **Commit:** N/A
- [ ] **Status:** Pending

### SP02PH06T02 — Merge to Staging
- [ ] **Task:** Merge sp02-session-proxy → staging
- [ ] **Acceptance:** CI passes on staging
- [ ] **Commit:** "SP02PH06T02: Merged to staging"
- [ ] **Status:** Pending

### SP02PH06T03 — Staging Validation
- [ ] **Task:** Verify SP02 features work in staging environment
- [ ] **Acceptance:** All acceptance criteria verified in staging
- [ ] **Commit:** N/A
- [ ] **Status:** Pending

### SP02PH06T04 — Merge to Master
- [ ] **Task:** Merge staging → master
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
| PH00 | 5 | 0/5 |
| PH01 | 9 | 0/9 |
| PH02 | 6 | 0/6 |
| PH03 | 8 | 0/8 |
| PH04 | 4 | 0/4 |
| PH05 | 2 | 0/2 |
| PH06 | 6 | 0/6 |
| **Total** | **40** | **0/40** |
