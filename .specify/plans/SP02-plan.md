# SP02 — Session Proxy & Live Connectivity

## Plan

> **Constitutional Reference:** This plan implements SP02 and is governed by Constitution VIII. Deployment Governance & Environment Integrity. All branch promotions must follow the defined flow.

This plan outlines the execution path for SP02, introducing the first live MUD connection capability: an authenticated user can connect to, interact with, and disconnect from a real MUD server through a secure backend proxy.

---

## Phase 0: Branch Creation (PH00)

### Tasks:
- [ ] Create branch: `sp02-session-proxy`
- [ ] Ensure spec document SP02.md exists
- [ ] Commit to branch
- [ ] Push to origin
- [ ] Verify CI passes on branch

---

## Phase 1: Backend Connection Manager (PH01)

### SP02PH01T00 — Create saved_connections table
- [ ] **Task:** Create migration for mud_saved_connections table (id, user_id, name, host, port, created_at, updated_at)
- **Storage:** Postgres (not Redis)
- [ ] **Commit:** "SP02PH01T00: Saved connections table created"

### SP02PH01T01 — TCP Dialer Implementation
- [ ] **Task:** Implement outbound TCP connection to arbitrary host:port
- **Acceptance:** Backend can connect to test MUD server
- [ ] **Commit:** "SP02PH01T01: TCP dialer implemented"

### SP02PH01T02 — Port Whitelist Enforcement
- [ ] **Task:** Implement port whitelist (default: 23)
- **Reject:** Ports not in whitelist with clear error
- [ ] **Commit:** "SP02PH01T02: Port whitelist enforced"

### SP02PH01T03 — Private IP Detection
- [ ] **Task:** Implement private IP detection and blocking
- **Block:** 10.x.x.x, 172.16-31.x.x, 192.168.x.x, 127.x.x.x, localhost
- [ ] **Commit:** "SP02PH01T03: Private IP blocking implemented"

### SP02PH01T04 — Per-User Session Binding
- [ ] **Task:** Bind each MUD connection to authenticated user session
- **Acceptance:** One connection per user enforced
- [ ] **Commit:** "SP02PH01T04: Per-user session binding"

### SP02PH01T05 — Idle Timer Enforcement
- [ ] **Task:** Implement 30-minute idle timeout
- **Behavior:** No activity for 30 min → disconnect
- [ ] **Commit:** "SP02PH01T05: Idle timeout enforced"

### SP02PH01T06 — Hard Session Cap
- [ ] **Task:** Implement 24-hour absolute session limit
- **Behavior:** 24 hours max → forced disconnect
- [ ] **Commit:** "SP02PH01T06: Hard session cap enforced"

### SP02PH01T07 — Connection Metadata Logging
- [ ] **Task:** Log connection metadata (connect time, host, port, duration, disconnect reason)
- **Acceptance:** Logs visible, no PII logged
- [ ] **Commit:** "SP02PH01T07: Connection metadata logging"

### SP02PH01T08 — Local Test Verification
- [ ] **Task:** Test backend connecting to real MUD server locally
- [ ] **Acceptance:** Connect, send command, receive output, disconnect
- [ ] **Commit:** "SP02PH01T08: Local test verification"

---

## Phase 2: WebSocket Bridge (PH02)

### SP02PH02T01 — WSS Endpoint Implementation
- [ ] **Task:** Implement WebSocket upgrade endpoint at /api/v1/session/stream
- **Acceptance:** Endpoint accepts WebSocket connections from authenticated users
- [ ] **Commit:** "SP02PH02T01: WSS endpoint implemented"

### SP02PH02T02 — Bidirectional Stream Relay
- [ ] **Task:** Implement stream relay: MUD → client, client → MUD
- [ ] **Acceptance:** Data flows both directions in real-time
- [ ] **Commit:** "SP02PH02T02: Bidirectional stream relay"

### SP02PH02T03 — Message Size Enforcement
- [ ] **Task:** Implement 64KB message size limit per message
- **Reject:** Messages exceeding limit, close connection
- [ ] **Commit:** "SP02PH02T03: Message size limit enforced"

### SP02PH02T04 — Flood Protection
- [ ] **Task:** Implement rate limiting (max 10 commands/second per user)
- **Enforcement Location:** WebSocket ingress layer (before reaching TCP writer)
- **Acceptance:** Rapid commands throttled or rejected at ingress
- [ ] **Commit:** "SP02PH02T04: Flood protection implemented"

### SP02PH02T05 — Session Teardown
- [ ] **Task:** Implement proper cleanup on WebSocket close
- **Behavior:** MUD connection closed, resources freed
- [ ] **Commit:** "SP02PH02T05: Session teardown implemented"

### SP02PH02T06 — Multi-User Isolation Test
- [ ] **Task:** Test with two simultaneous users
- [ ] **Acceptance:** No session crossover, independent connections
- [ ] **Commit:** "SP02PH02T06: Multi-user isolation verified"

---

## Phase 3: UI Implementation (PH03)

### SP02PH03T00 — Frontend Scaffold (React + TypeScript)
- [ ] **Task:** Set up React + TypeScript frontend project
- **Framework:** React 18+, TypeScript 5+
- **Build Tool:** Vite (recommended) or Next.js
- [ ] **Commit:** "SP02PH03T00: Frontend scaffold created"

### SP02PH03T01 — Global App Shell Components
- [ ] **Task:** Implement top app bar, session status pill, account menu, primary navigation
- **Status Pill States:** Disconnected (gray), Connecting (yellow), Connected (green), Error (red)
- [ ] **Commit:** "SP02PH03T01: App shell implemented"

### SP02PH03T02 — Play Screen Connection Panel
- [ ] **Task:** Implement connection panel with host, port (default 23), Connect/Disconnect buttons
- **Acceptance:** Connect enabled when disconnected, Disconnect visible when connected
- [ ] **Commit:** "SP02PH03T02: Connection panel implemented"

### SP02PH03T03 — Play Screen Output Panel
- [ ] **Task:** Implement scrollable terminal rendering area
- **Acceptance:** Output displays in real-time, scrollable
- [ ] **Commit:** "SP02PH03T03: Output panel implemented"

### SP02PH03T04 — Command Input Row
- [ ] **Task:** Implement command input with Enter-to-send
- **Acceptance:** Input enabled when connected, disabled when disconnected
- [ ] **Commit:** "SP02PH03T04: Command input implemented"

### SP02PH03T05 — Error Messaging
- [ ] **Task:** Implement user-facing error messages
- **Errors:** Invalid host, connection refused, timeout, private IP, invalid port
- [ ] **Commit:** "SP02PH03T05: Error messaging implemented"

### SP02PH03T06 — Disabled States Enforcement
- [ ] **Task:** Ensure UI correctly disables/enables controls per state machine
- **Acceptance:** All states from Constitution V.3 table implemented
- [ ] **Commit:** "SP02PH03T06: Disabled states enforced"

### SP02PH03T07 — Connections Page (Minimal)
- [ ] **Task:** Implement saved connections page (optional if declared omitted)
- **Features:** Add, edit, delete, connect from saved
- [ ] **Commit:** "SP02PH03T07: Connections page implemented"

---

## Phase 4: Abuse & Resource Enforcement (PH04)

### SP02PH04T01 — Connection Limit Hooks
- [ ] **Task:** Implement per-user connection limits
- **Acceptance:** One active connection per user enforced
- [ ] **Commit:** "SP02PH04T01: Connection limits enforced"

### SP02PH04T02 — Soft Backpressure Strategy
- [ ] **Task:** Implement soft throttling instead of immediate disconnect
- **Strategy:** Drop excess → Send warning → Escalate only on sustained abuse
- [ ] **Acceptance:** Legitimate heavy ANSI bursts don't kill sessions; fast typists not booted
- [ ] **Commit:** "SP02PH04T02: Soft backpressure implemented"

### SP02PH04T03 — Observability Metrics (PRIORITY)
- [ ] **Task:** Add metrics for connection attempts, failures, durations
- **Acceptance:** Metrics visible in observability tooling
- [ ] **Commit:** "SP02PH04T03: Observability metrics added"

### SP02PH04T04 — Slow Client Test
- [ ] **Task:** Test with simulated slow browser (network throttling)
- **Rationale:** Ensure server doesn't panic on slow WebSocket consumer
- [ ] **Acceptance:** Server handles slow client gracefully
- [ ] **Commit:** "SP02PH04T04: Slow client scenario tested"

### SP02PH04T05 — Fail-Fast Invalid Handling
- [ ] **Task:** Ensure invalid connections fail fast with clear errors
- [ ] **Acceptance:** No hanging connections, immediate feedback
- [ ] **Commit:** "SP02PH04T05: Fail-fast handling implemented"

### SP02PH04T06 — Port Denylist Policy
- [ ] **Task:** Implement port denylist (always block dangerous ports)
- **Deny-list:** Email (25,465,587,110,143,993,995), DNS (53), Web (80,443), Databases (1433,1521,3306,5432,6379,27017), Remote admin (22,3389,5900), File sharing (445,139,2049)
- **Allow:** 23, 2525, and other non-deny-listed ports
- [ ] **Acceptance:** Blocked ports rejected with clear error; allowed ports work
- [ ] **Commit:** "SP02PH04T06: Port denylist implemented"

### SP02PH04T07 — Protocol Plausibility Check
- [ ] **Task:** Implement protocol mismatch detection
- **Behavior:** Check first N seconds/X bytes of server output
- **Disconnect if:** TLS handshake, HTTP headers, obvious binary protocol
- **Error code:** protocol_mismatch
- [ ] **Acceptance:** Non-MUD protocols rejected with clear message
- [ ] **Commit:** "SP02PH04T07: Protocol plausibility check added"

### SP02PH04T08 — Blocked Port Metrics
- [ ] **Task:** Add logging/metrics for blocked_port and protocol_mismatch
- **Metrics:** Count of blocked ports, count of protocol mismatches
- [ ] **Acceptance:** Metrics visible in observability tooling
- [ ] **Commit:** "SP02PH04T08: Port block metrics added"

### SP02PH04T09 — Port Policy Tests
- [ ] **Task:** Test denylist behavior
- **Tests:** Connect to blocked port (expect rejection), connect to allowed port (expect success)
- [ ] **Acceptance:** All denylist tests pass
- [ ] **Commit:** "SP02PH04T09: Port policy tests completed"

---

## Phase 5: QA Phase (PH05)

### SP02PH05T01 — Browser-Only Test Suite
- [ ] **Task:** QA validates all functionality from browser only
- **No database/log access required**

### Required Tests:
- [ ] Successful connect to test MUD
- [ ] Failed connect (invalid host)
- [ ] Failed connect (connection refused)
- [ ] Failed connect (timeout)
- [ ] Idle timeout at 30 minutes
- [ ] Hard cap at 24 hours
- [ ] Disconnect button functionality
- [ ] Rapid connect/disconnect sequences
- [ ] Two users simultaneously (isolation)
- [ ] Invalid host error message displayed
- [ ] Invalid port error message displayed
- [ ] Private IP attempt blocked with error
- [ ] Blocked port (e.g., 3306) rejected with clear error
- [ ] Allowed port (e.g., 2525) connects successfully

### SP02PH05T02 — QA Sign-Off
- [ ] **Task:** Produce QA pass report
- [ ] **Acceptance:** All tests pass, browser-only validation confirmed

---

## Phase 6: Final Phase - Merge & Deployment (PH06)

> **Constitutional Note (VIII):** All promotions must follow: sp02-session-proxy → staging → master. No direct Railway CLI deployment except emergency.

> **Idle Timeout Clarification:**
> - WebSocket read-deadline-based idle disconnect: REMOVED (per Constitution IX, backend session policy is authoritative)
> - Session hard cap (24 hours): REMAINS in effect
> - Activity-based idle timeout (if implemented): Must be defined as "activity-based only" - inbound MUD OR outbound user activity resets timer

### SP02PH06T01 — Confirm Branch State and Targets
- [ ] **Task:** Verify you are on the correct branch locally
- **Commands:** `git status`, `git branch --show-current` (should be sp02-session-proxy), `git remote -v`
- **Acceptance:** No uncommitted changes; branch is correct; CI is green

### SP02PH06T02 — Merge to Staging via PR
- [ ] **Task:** Open PR: sp02-session-proxy → staging
- **Requirements:** Ensure required CI checks pass; Merge using normal PR merge method (no bypass)
- **Constitutional Verification (VIII.4):** `git branch` → confirm on sp02-session-proxy; `railway status` → confirm staging
- **Acceptance:** staging contains SP02 commits; Railway staging deploy triggers automatically

### SP02PH06T03 — Staging Validation
- [ ] **Task:** Validate in staging URL end-to-end

#### Functional Validation:
- [ ] Login via OTP; Connect to at least two MUDs (include non-23 port); Send commands, receive output
- [ ] Confirm telnet IAC stripping still clean; Confirm copy behavior works (Ctrl/Cmd+C and context menu)

#### Policy / Safety Validation:
- [ ] Private IP blocked; Denylisted port blocked with clear error; Protocol mismatch disconnect works

#### Observability Validation:
- [ ] /api/v1/admin/metrics: Requires secret; Counters increment during session; No user identifiers leak

- **Acceptance:** Staging behaves exactly like PH05 QA results; logs show no new errors

### SP02PH06T04 — Railway CLI Deployment Discipline
- [ ] **Task:** Document Railway CLI usage requirements
- **Directive:** Any Railway CLI usage must be preceded by explicit environment verification command and documented output
- **Acceptance:** PH06 writeup includes exact commands used and explicit verification output that it was staging

### SP02PH06T05 — Merge to Master via PR
- [ ] **Task:** Open PR: staging → master
- **Requirements:** CI must pass; Merge via PR (no CLI push to prod)
- **Acceptance:** Production deploy triggers from master automatically

### SP02PH06T06 — Production Verification
- [ ] **Task:** Repeat smaller version of staging validation on production
- **Validation:** Login works; Connect to one MUD; Denylist + private IP block work; Metrics endpoint secret-protected
- **Acceptance:** Production is healthy; no regressions; logs clean

### SP02PH06T07 — Spec Closed
- [ ] **Task:** Update SP02.md status to Closed
- **Idle Timeout Note:** WebSocket read-deadline idle disconnect: REMOVED; Session hard cap (24h): REMAINS
- [ ] **Commit:** "SP02PH06T07: Spec closed"

---

## Summary

| Phase | Tasks | Description |
|-------|-------|-------------|
| PH00 | 5 | Branch creation and CI verification |
| PH01 | 9 | Backend connection manager + DB migration |
| PH02 | 6 | WebSocket bridge |
| PH03 | 8 | UI implementation + frontend scaffold |
| PH04 | 9 | Abuse prevention + port policy |
| PH05 | 2 | QA validation |
| PH06 | 7 | Merge and deployment |
| **Total** | **45** | |

---

## Deployment Governance (Constitution VIII)

### Branch-to-Environment Mapping

| Branch | Environment | Purpose |
|--------|-------------|---------|
| sp02-session-proxy | Feature branch | Spec implementation |
| staging | Staging | Integration & QA |
| master | Production | Live system |

### Promotion Flow

**Step 1: Verify Environment (VIII.4)**
Before any promotion, verify:

1. `git branch` → confirm local branch is `sp02-session-proxy`
2. `git status` → confirm clean working tree
3. `git branch -a` → confirm remote staging exists
4. `railway status` → confirm linked to **staging** (never production)

**Step 2: Promote to Staging**
```
git push origin sp02-session-proxy:staging
```

**Step 3: Verify Staging Deployment**
- Confirm CI passes on staging
- Run QA in staging environment

**Step 4: Promote to Production** (after QA complete)
```
git push origin staging:master
```

### Railway CLI Rules (VIII.3)

- **DO NOT** use `railway up` for routine deployments
- Railway CLI only for: `railway status`, `railway logs`, emergency
- If uncertain about environment: `railway unlink` → `railway link` → `railway status`

### Emergency Deployment Protocol (VIII.6)

If Railway CLI deployment required:
1. Document justification in this plan
2. Run `railway status` and verify environment
3. Preserve output in this plan
4. After fix: promote via git to restore normal process

---

## Dependencies

- **SP00** — Environment & Infrastructure Foundation (complete)
- **SP01** — Account Authentication & Session Foundation (complete)
- Railway hosting
- Redis
- PostgreSQL
