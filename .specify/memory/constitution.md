# MUDPuppy Constitution

## Core Principles

### I. Service-First Architecture
The browser is a presentation layer, not the engine. All heavy lifting—socket connections, protocol negotiation, compression handling (MCCP), metadata channels (GMCP), session management, and persistence—lives at the service layer. The service layer performs protocol negotiation and stream normalization. The frontend owns presentation and visual interpretation. This separation ensures scalability, security, and the ability to evolve client capabilities without destabilizing the backend.

### II. Not a Terminal Emulator
We explicitly reject the "terminal emulator wrapped in a web page" approach. The frontend must not rely on raw ANSI rendering libraries. ANSI compatibility is achieved through server-side normalization into structured render events. While the MVP supports ANSI-equivalent rendering for compatibility, the architecture must not constrain the front end to retro terminal paradigms indefinitely. Future phases must be able to introduce richer rendering, modern UI constructs, and enhanced interaction paradigms without requiring a foundational rewrite.

**II.a MVP Terminal Rendering Exception (Time-Boxed)**
For MVP only, the frontend may render ANSI/VT output using a terminal rendering library (e.g., xterm.js) as an implementation accelerator, provided:
- The service-first architecture remains intact (service owns sockets, protocol negotiation, session management, persistence)
- The backend still performs protocol negotiation (MCCP, GMCP, etc.) and security controls
- A clear migration plan exists to replace raw ANSI rendering with structured render events post-MVP
- Specs must treat xterm.js rendering as temporary, not a guaranteed permanent UX

This exception is a bootstrap mechanism, not a product direction. The long-term goal remains structured render events for rich UI capabilities.

### III. MVP Playability Over Feature Bloat
The MVP focuses on playability, stability, and persistence—not feature bloat. We avoid overengineering with deep desktop-era features (full automapping, advanced script debugging) that are not required to establish a playable baseline. Each feature must prove its worth before inclusion.

### IV. Automation Scalability
Automation engines (aliases, triggers, timers, variables) must be designed to scale from simple rule-based MVP functionality to more advanced sandboxed scripting in later phases. The architecture must support this growth without breaking early implementations.

### V. Modern Web Application Standards
Beyond MUD-specific functionality, this is a modern web application. Authentication, account management, profile persistence, administrative tooling, logging, monitoring, and security must be first-class citizens. User data, configurations, and session artifacts must persist reliably in the cloud and follow users across devices.

### V.a Zero Local Storage Dependency
The product must not require persistent browser-local storage (LocalStorage, IndexedDB, filesystem APIs) for core functionality. A user must be able to log in and play fully from a fresh browser session on any device. Optional ephemeral client caching (in-memory only) is allowed for responsiveness, but must not be required for correctness, persistence, or recovery.

### VI. Multi-Tenant Security by Default
The product must be multi-tenant by design, secure by default, and operationally observable. Performance and reliability must be measurable and continuously improvable through proper monitoring and logging.

## Technology & Architecture

### Hosting Model
**MVP Hosting Standard (Required):** Railway is the authoritative MVP deployment platform and reference environment for CI/CD, environment variables, and managed services.

**Production Hosting Standard (Required):** The architecture must remain portable to a scalable production hosting target (e.g., AWS/GCP/Azure or equivalent container platform) without rewriting core systems.

**Portability Requirements:**
- All services run as containers (or container-compatible build artifacts)
- Configuration via environment variables and secrets management abstractions
- No Railway-unique runtime assumptions in core service logic
- Data stores must have migration paths (Postgres/Redis portable variants)

### Backend Composition
The service layer is comprised of discrete, focused components:

**API Gateway / WebSocket Server**
- Terminates HTTPS/WSS connections from clients
- Handles authentication, rate limiting, and request routing
- Manages client session lifecycle

**Session Manager**
- Maintains active MUD server connections (telnet/SSH)
- Handles connection pooling and reconnection logic
- Manages session state, heartbeats, and timeouts

**Protocol Handler**
- Processes inbound/outbound byte streams
- Negotiates protocols (MCCP, GMCP, MSDP, MTTS)
- Parses ANSI control sequences
- Normalizes output into a structured, presentation-agnostic event stream
- Removes raw escape sequences before transmission to the frontend
- Manages compression/decompression

**Data Layer**
- User accounts and authentication data
- Session history and logs
- User configurations and profiles
- Automation rules (aliases, triggers, timers, variables)
- Utilizes appropriate data stores (SQL for structured data, NoSQL for logs/sessions)

**Automation Engine**
- Rule-based processing for MVP (aliases, triggers, timers, variables)
- Sandboxed execution environment for advanced scripting
- Event-driven architecture for trigger firing

### Frontend Architecture
The browser-based client is a modern SPA (Single Page Application):
- Consumes WebSocket API for real-time communication
- Renders session data with flexibility for future rich UI
- Stores minimal local state; prefers server-side persistence
- Responsive design for cross-device access

**Render Abstraction Model**
The service layer is responsible for protocol negotiation and stream normalization. It must never emit raw ANSI or terminal escape sequences to the frontend. The frontend is responsible for presentation and visual interpretation of structured render events.

The service does not emit ANSI or HTML. It emits structured render events. For example, instead of transmitting raw escape sequences like `\x1b[31mYou are bleeding!\x1b[0m`, the backend produces a structured event model:

```json
{
  "type": "text",
  "segments": [
    { "content": "You are bleeding!", "style": { "fg": "red" } }
  ]
}
```

This ensures:
- No terminal escape codes reach the client
- No HTML is embedded server-side
- Future UI paradigms (animated UI, component-driven layouts, accessibility overlays, health bars, context panels) remain possible
- Non-text visualizations can be supported later without reworking the backend

**Render Events Roadmap:**
Post-MVP, the architecture must support migrating from xterm.js terminal rendering to structured render events. This enables:
- Semantic UI layers (health bars, overlay panels, contextual information)
- Component-driven layouts beyond terminal grid
- Accessibility overlays and screen reader support
- Rich media integration
The migration path from MVP (xterm.js) to Phase N (structured events) must be planned and documented.

### WebSocket-Based Communication
The browser connects securely over HTTPS (port 443) to the hosted service via WebSocket. The service maintains the actual telnet (and later optional SSH) connections to MUD servers, handling all protocol complexity.

### Protocol Support
The service layer handles:
- ANSI stream rendering
- Protocol negotiation
- Compression (MCCP)
- Metadata channels (GMCP)

### Local Development
Development occurs locally using Visual Studio Code, with code committed to a repository that feeds a CI/CD pipeline responsible for automated builds, testing, and deployment to the hosted environment.

### External Dependencies
The following external services and software are required dependencies:

**Hosting Platform:**
- Railway - Primary hosting platform (PaaS with Git push-to-deploy)
  - Handles container building, SSL, deployment
  - Provides managed PostgreSQL and Redis
  - Free tier available for development
  - Scales as needed with paid plans

**Version Control:**
- GitHub - Source code repository
  - GitHub Actions for CI/CD pipelines
  - Connected to Railway for automatic deployments

**Data Stores:**
- PostgreSQL (via Railway) - User accounts, configurations, automation rules
- Redis (via Railway) - Session caching, real-time state

**Development Tools:**
- Node.js (LTS) - Frontend runtime
- Go - Backend service development (optimal for network proxy services)
- Visual Studio Code - Primary IDE
- Git - Version control

## Development Workflow

### Constitution-Driven Development
Success comes from disciplined adherence to a constitution-driven development model. Every feature spec must trace back to this doctrine. We define clearly what the product is and is not.

### Phased Implementation
Each subsequent phase expands capability without invalidating earlier architectural decisions. The MVP establishes playable baseline functionality; future phases build upon that foundation deliberately.

### MVP Definition
The MVP establishes the minimum viable product that demonstrates core value. MVP must include:
- Account authentication + profile persistence
- Connect/reconnect to MUD (telnet) via service proxy
- ANSI-equivalent display (xterm.js allowed for MVP)
- Input history + basic keybind support
- Aliases + triggers + timers + variables (rule-based)
- Logging (server-side)

Post-MVP phases will introduce richer rendering, advanced automation scripting, SSH support, and enhanced UI constructs.

### CI/CD Pipeline
The deployment workflow uses Railway with GitHub integration for push-to-deploy functionality.

**Pipeline Flow (Local to Production):**
1. **Local Development** - Developer writes code in VS Code
2. **Commit & Push** - Code pushed to GitHub repository
3. **Railway Auto-Deploy** - Railway automatically detects the push and:
   - Builds the application from source
   - Runs build commands
   - Deploys to the Railway environment
4. **Production** - Live on Railway with automatic SSL

**Workflow Simplicity:**
- No manual Docker builds required
- Railway handles containerization internally
- GitHub Actions can be added later for test automation before deploy
- Environment variables managed in Railway dashboard
- One-click rollback via Railway console

**Environment Tiers:**
- **Development** - Auto-deploy on feature branch merge
- **Staging** - Full integration testing mirror of production
- **Production** - Manual approval required for release

## Governance

### Constitution Supremacy
This constitution supersedes all other practices. Amendments require:
- Clear documentation of the change
- Approval from project leadership
- A migration plan if the amendment affects existing functionality

### Feature Validation
All features must demonstrate:
- Alignment with core principles
- Playability/stability value
- Extensibility without breaking existing architecture

### Quality Gates
- All code must pass automated tests before merge
- Security review required for authentication and multi-tenant features
- Performance benchmarks must be established and maintained

### Resource Governance
As a multi-tenant hosted service, resource limits must be enforceable. Exact numeric thresholds live in an operational SLO document that can be updated without constitutional amendment.

**The platform must enforce configurable limits for:**
- Concurrent sessions per user
- Concurrent outbound MUD connections per user
- Automation rule counts and execution rates
- Message size and throughput
- Connection duration and idle timeouts

**Limits are policy-driven and configurable by tier** (free/dev/staging/production), and must be adjustable without code changes.

### Abuse Prevention Doctrine
This product proxies arbitrary text streams and opens outbound connections. Abuse prevention is constitution-level, not implementation-level:

**Connection Restrictions:**
- Default deny all outbound ports
- Whitelist allowed destination ports: 23 (telnet), 22 (SSH), 80, 443
- No port scanning capabilities
- No proxy to internal/private networks

**Traffic Monitoring:**
- Detect and block flood behavior (threshold: 1000 messages/second)
- Detect unusual traffic patterns
- Log all outbound connections for abuse analysis

**Behavioral Limits:**
- Maximum message size: 64KB
- Maximum connection duration: 24 hours (user can reconnect)
- Automatic disconnection on idle (timeout: 30 minutes)

### Observability Requirements
Performance and reliability are constitutional values. Exact numeric thresholds live in an operational SLO document that can be updated without constitutional amendment.

**Define SLOs for:**
- Session startup latency
- Round-trip responsiveness
- Disconnect rate
- Automation execution time
- Error rates

**Metrics must be measurable:**
- Session metrics: startup latency, RTT, disconnect frequency, memory per session
- Automation metrics: trigger execution time, failure rate, rules engine throughput
- System metrics: active concurrent sessions, API response latency (p50, p95, p99), error rate by type

### Automation Sandbox Boundaries
Automation runs server-side and must be strictly sandboxed:

**Execution Constraints:**
- No filesystem access
- No arbitrary outbound network calls (MUD connections are exception)
- No subprocess execution
- Deterministic timeout boundaries (max 5 seconds per script)
- Memory-limited execution environment (10MB per automation)

**Isolation:**
- Per-user execution context
- No shared state between automation rules
- No access to other users' data
- Quota enforcement: max 1000 automation executions per minute per user

### Versioning Strategy
Frontend and backend evolve independently. Versioning ensures compatibility:

**API Versioning:**
- URL-based versioning: /api/v1/, /api/v2/
- Deprecation policy: 6 months notice before removing vN
- Backward compatibility within major version

**WebSocket Protocol Versioning:**
- Version header in WebSocket handshake
- Graceful protocol negotiation
- Client must declare supported protocol version
- Server rejects incompatible protocol versions

**Upgrade Path:**
- Clients must upgrade within 2 major versions of server
- Feature flags allow server-side enablement for newer API features
- Breaking changes only in major version bumps

**Version**: 1.7.1 | **Ratified**: 2026-02-20 | **Last Amended**: 2026-02-20

**Amendment v1.7.1:** Changed all references from 'main' to 'master' branch to match repository naming convention.

---

## Specification Governance & Development Lifecycle

### I. Spec-Driven Development Model
All feature work must be executed under a formally defined Specification ("Spec"). No engineering work may begin without an approved Spec document.

Each Spec represents a discrete, user-visible feature or cohesive set of features. Architectural refactors that materially affect behavior must also be treated as Specs.

Only one Spec may be active/open at a time. No parallel feature development is permitted.

This ensures:
- Focused delivery
- Controlled scope
- Clear QA accountability
- No partial-feature drift
- Predictable merge cadence

### II. Spec Identification & Naming Convention
Each Spec must be uniquely numbered and tracked sequentially.

**Format:** SP##PH##T##
- SP## = Spec number (e.g., SP01)
- PH## = Phase number (e.g., PH01)
- T## = Task number (e.g., T01)

Example: SP01PH01T01 = Spec 01, Phase 01, Task 01

This naming format must be used in:
- Documentation
- Branch names
- Commit messages
- Issue tracking
- QA reports

### III. Structure of a Spec
Each Spec must contain:
- Problem Statement
- Scope Definition
- Non-Goals (Explicitly Stated)
- Architectural Impact
- User-Facing Behavior Description
- Acceptance Criteria
- Definition of Done

A Spec without explicit acceptance criteria may not be approved. Acceptance criteria must be:
- Observable
- Testable
- User-accessible (no internal-only validation)

### IV. Spec Lifecycle Phases
Each Spec is divided into Phases. Each Phase contains clearly defined Tasks.

**Phase 0 — Branch Creation (SP##PH00)**
- Purpose: Isolation and development boundary creation
- Create a new Git branch from master
- Branch must be named: sp##-feature-shortname
- CI must pass on branch creation
- Spec document must be committed to branch before development begins
- No feature development may begin before Phase 0 completes

**Development Phases (PH01 → PHNN)**
- Each Phase must define objective clearly
- Contain enumerated Tasks
- Include acceptance criteria per Task
- Define completion conditions

Each Task must:
- Have a Definition of Done
- Be independently testable
- Not depend on undeclared future work
- Tasks may not be vague ("implement backend changes" is invalid)

**QA Phase (Final Development Phase Before Merge)**
- Every Spec must include a dedicated QA Phase before merge (mandatory)
- Purpose: Validate UI behavior, user-visible functionality, acceptance criteria
- QA has access only to: web UI, user-level functionality, normal authentication paths
- QA has NO: database access, log access, admin console access, internal observability tools
- If QA cannot validate from UI, feature is not complete
- QA must test: positive cases, negative cases, edge cases, failure handling, UI regressions
- QA must produce pass/fail report referencing Spec IDs

**Final Phase — Merge & Commit (Last Phase)**
- Only after QA passes
- All Spec acceptance criteria verified
- CI must pass
- Code review completed
- Branch merged into master
- Deployment follows CI/CD process
- No merge before QA approval

Spec is "Closed" only after: successful merge, deployment, post-deploy validation

### V. Acceptance Criteria Requirements
Each Spec must define acceptance criteria at two levels:

**A. Feature-Level Acceptance Criteria**
Defines what the end user must experience:
- User can create X
- User can modify Y
- System responds within Z milliseconds
- Error condition displays defined message

**B. Task-Level Acceptance Criteria**
Each Task must have:
- Measurable output
- Observable behavior
- Clear Definition of Done

### VI. Definition of Done (DoD)
A Task is complete only when:
- Code implemented
- Unit tests written (where applicable)
- CI passes
- Acceptance criteria validated locally
- No TODO placeholders remain
- No console errors
- Documentation updated (if applicable)

A Phase is complete only when:
- All Tasks complete
- Phase acceptance criteria satisfied

A Spec is complete only when:
- QA passes
- Merged to master
- Deployment verified

### VII. Spec Concurrency Rule
Only one Spec may be active at any time.

Exceptions require:
- Formal amendment
- Leadership approval
- Justification for parallelization

This prevents:
- Architectural drift
- Incomplete integrations
- QA overload
- Hidden feature coupling

### VIII. QA Environment Constraints
Testing must reflect real user conditions.

QA must:
- Use normal user accounts
- Use production-like UI
- Test via browser only

QA must not:
- Directly modify database
- Inspect backend logs
- Access internal APIs directly

If testing requires backend inspection, the feature is improperly designed or improperly instrumented.

User-visible correctness is the constitutional standard.

### IX. Spec Closure & Audit Trail
Each Spec must produce:
- Final Spec document (versioned)
- QA report
- Merge reference (commit ID)
- Deployment reference
- Release notes

Specs must remain archived and traceable for audit purposes.
