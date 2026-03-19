# PR02 — Internal Command Module Tasks

> **Task Tracker:** PR02  
> **Spec Reference:** PR02 — Internal Command Module (ICM)  
> **Plan Reference:** PR02-plan  
> **Status:** Draft  
> **Pre-Release Version:** 0.2.0-pr2  
> **Total Tasks:** 60

---

## Phase 0: Setup (PR02PH00)

### PR02PH00T01 — Create Feature Branch

- [x] **Task:** Create branch `pr02-internal-command-module` from staging
- **Commit:** "PR02PH00T01: Create feature branch"
- **Status:** Completed - Branch created and committed

### PR02PH00T02 — Review Existing Command Processing + Early Audit

- [x] **Task:** Examine current automation engine, input handlers, keybinding system, and identify existing command processing paths. Survey for ad hoc prefix checks, inline alias expansion, variable replacement, or directive parsing that should route through ICM.
- **Acceptance:** Documented integration points, execution flow, and audit findings informing migration scope
- **Commit:** "PR02PH00T02: Review existing command processing code and audit scattered operator logic"
- **Audit Document:** [.specify/plans/PR02PH00-audit.md](.specify/plans/PR02PH00-audit.md) - Contains detailed findings on:
  - Current architecture and ingress routes
  - Integration points for ICM (primary: AutomationEngine.processUserInput)
  - Current operator implementation (#, @, $, %)
  - Gaps: escape handling, unknown operators, pass-through, backend processing
  - Safety systems present (circuit breaker, loop detection, rate limiting)
  - Migration scope and file modifications required
- **Status:** Completed

---

## Phase 1: ICM Design (PR02PH01)

### PR02PH01T01 — Define Command Lifecycle

- [ ] **Task:** Design the ICM command lifecycle: ingress → tokenize → validate → normalize → resolve → dispatch
- **Acceptance:** Documented lifecycle with clear phase definitions
- **Commit:** "PR02PH01T01: Define command lifecycle"

### PR02PH01T02 — Define Frontend/Backend Contract

- [ ] **Task:** Define API contract between frontend and backend for command processing
- **Acceptance:** Documented contract with request/response shapes
- **Commit:** "PR02PH01T02: Define frontend/backend contract"

### PR02PH01T03 — Define Operator Families

- [ ] **Task:** Catalog all operator families: #, @, %, $, and equivalent forms
- **Acceptance:** Documented operator family registry format
- **Commit:** "PR02PH01T03: Define operator families"

### PR02PH01T03B — Define % Namespace Resolution Precedence

- [ ] **Task:** Define % namespace resolution precedence: numeric %<n> is always session-scoped, named %NAME is always system-scoped. Define collision policy and error/malformed reference behavior.
- **Acceptance:** Documented precedence rules with numeric vs named resolution, collision handling, and pass-through behavior
- **Commit:** "PR02PH01T03B: Define % namespace resolution precedence"

### PR02PH01T04 — Define Execution Contexts

- [ ] **Task:** Define execution contexts: Preview, Submission, Automation, Operational
- **Acceptance:** Documented context definitions with authority levels
- **Commit:** "PR02PH01T04: Define execution contexts"

### PR02PH01T05 — Define Normalization Format

- [ ] **Task:** Define canonical command normalization format
- **Acceptance:** Documented normalization output shape
- **Commit:** "PR02PH01T05: Define normalization format"

### PR02PH01T06 — Define Error Model

- [ ] **Task:** Define error types, codes, and user-facing messages
- **Acceptance:** Documented error model
- **Commit:** "PR02PH01T06: Define error model"

### PR02PH01T07 — Define Safety Hooks

- [ ] **Task:** Define integration points for recursion limits, circuit breaker, rate limits
- **Acceptance:** Documented safety hook interface
- **Commit:** "PR02PH01T07: Define safety hooks"

### PR02PH01T08 — Define Escape/Literal Grammar

- [ ] **Task:** Define escape sequences, literal pass-through rules, and unknown operator handling
- **Acceptance:** Documented grammar for escape/literal behavior
- **Commit:** "PR02PH01T08: Define escape/literal grammar"

### PR02PH01T09 — Define Pass-Through Behavior

- [ ] **Task:** Define behavior for ordinary MUD commands that are not internal commands
- **Acceptance:** Documented pass-through classification rules
- **Commit:** "PR02PH01T09: Define pass-through behavior"

### PR02PH01T10 — Define #LOG Governance

- [ ] **Task:** Define #LOG directive visibility, permissions, output format, routing
- **Acceptance:** Documented #LOG specification
- **Commit:** "PR02PH01T10: Define #LOG governance"

---

## Phase 2: ICM Core Implementation (PR02PH02)

### PR02PH02T01 — Create Backend ICM Core Structure

- [ ] **Task:** Create backend ICM module structure with registry and handler resolution
- **Acceptance:** Backend module skeleton with component interfaces
- **Commit:** "PR02PH02T01: Create backend ICM core structure"

### PR02PH02T02 — Implement Recognizer

- [ ] **Task:** Implement tokenization and operator recognition
- **Acceptance:** Recognizer correctly identifies #, @, %, $ operators
- **Commit:** "PR02PH02T02: Implement recognizer"

### PR02PH02T03 — Implement Validator

- [ ] **Task:** Implement syntax validation and argument checking
- **Acceptance:** Validator rejects invalid commands with clear errors
- **Commit:** "PR02PH02T03: Implement validator"

### PR02PH02T04 — Implement Normalizer

- [ ] **Task:** Implement command normalization to canonical form
- **Acceptance:** Normalizer produces consistent output
- **Commit:** "PR02PH02T04: Implement normalizer"

### PR02PH02T05 — Implement Registry

- [ ] **Task:** Implement command registry for handler registration
- **Acceptance:** Registry stores command definitions and metadata
- **Commit:** "PR02PH02T05: Implement registry"

### PR02PH02T06 — Implement Dispatcher

- [ ] **Task:** Implement dispatcher with safety enforcement
- **Acceptance:** Dispatcher routes commands to handlers with safety checks
- **Commit:** "PR02PH02T06: Implement dispatcher"

### PR02PH02T07 — Implement Escape/Literal Handling

- [ ] **Task:** Implement escape sequence processing and literal pass-through
- **Acceptance:** Escape and literal behaviors work correctly
- **Commit:** "PR02PH02T07: Implement escape/literal handling"

### PR02PH02T08 — Implement Unknown Operator Handling

- [ ] **Task:** Implement handling for unrecognized operators
- **Acceptance:** Unknown operators handled according to spec
- **Commit:** "PR02PH02T08: Implement unknown operator handling"

### PR02PH02T09 — Implement Pass-Through Classification

- [ ] **Task:** Implement classification of commands as internal vs MUD commands
- **Acceptance:** Non-internal commands pass through correctly
- **Commit:** "PR02PH02T09: Implement pass-through classification"

### PR02PH02T10 — Create Frontend Adapter

- [ ] **Task:** Create frontend adapter for validation, preview, and submission integration
- **Acceptance:** Frontend adapter communicates with backend ICM
- **Commit:** "PR02PH02T10: Create frontend adapter"

### PR02PH02T11 — Backend Core Unit Tests

- [ ] **Task:** Write unit tests for ICM recognizer, validator, normalization, and registry components
- **Acceptance:** Core ICM components have automated unit test coverage before frontend integration
- **Commit:** "PR02PH02T11: Backend core unit tests"

---

## Phase 3: Frontend Integration (PR02PH03)

### PR02PH03T01 — Integrate Typed Input

- [ ] **Task:** Route typed command input through ICM adapter
- **Acceptance:** Commands processed through ICM pipeline
- **Commit:** "PR02PH03T01: Integrate typed input"

### PR02PH03T02 — Integrate Keybindings

- [ ] **Task:** Route keybinding dispatch through ICM adapter
- **Acceptance:** Keybindings use canonical command pipeline
- **Commit:** "PR02PH03T02: Integrate keybindings"

### PR02PH03T03 — Integrate History Replay

- [ ] **Task:** Route history recall through ICM adapter
- **Acceptance:** History commands use full submission path
- **Commit:** "PR02PH03T03: Integrate history replay"

### PR02PH03T04 — Integrate Editors

- [ ] **Task:** Route editor validation through ICM adapter
- **Acceptance:** Editors use ICM for syntax validation
- **Commit:** "PR02PH03T04: Integrate editors"

### PR02PH03T05 — Preserve Submission Behavior

- [ ] **Task:** Ensure canonical submitCommand() behavior preserved
- **Acceptance:** No regression in command submission
- **Commit:** "PR02PH03T05: Preserve submission behavior"

---

## Phase 4: Backend Integration (PR02PH04)

### PR02PH04T01 — Implement Authoritative Execution

- [ ] **Task:** Implement backend pathways for session-state commands
- **Acceptance:** Backend executes commands authoritatively
- **Commit:** "PR02PH04T01: Implement authoritative execution"

### PR02PH04T02 — Implement #LOG Directive

- [ ] **Task:** Implement standardized logging directive per governance spec
- **Acceptance:** #LOG command routes through ICM with proper governance
- **Commit:** "PR02PH04T02: Implement #LOG directive"

### PR02PH04T03 — Implement Command-State Effects

- [ ] **Task:** Implement stateful command effects (variables, timers)
- **Acceptance:** State changes go through ICM
- **Commit:** "PR02PH04T03: Implement command-state effects"

### PR02PH04T04 — Maintain Service-First Boundaries

- [ ] **Task:** Ensure service-first architecture maintained
- **Acceptance:** Backend remains authoritative for session-state
- **Commit:** "PR02PH04T04: Maintain service-first boundaries"

### PR02PH04T05 — Backend Integration Tests

- [ ] **Task:** Write integration tests for ICM handler registration, safety boundaries, and end-to-end command flows
- **Acceptance:** Core ICM components have automated test coverage
- **Commit:** "PR02PH04T05: Add backend unit/integration tests"

---

## Phase 5: Known Command Migration (PR02PH05)

### PR02PH05T01 — Migrate Structured Directives

- [ ] **Task:** Move #IF, #SET, #TIMER, etc. to ICM
- **Acceptance:** Directives registered and processed through ICM
- **Commit:** "PR02PH05T01: Migrate structured directives"

### PR02PH05T02 — Migrate User Variable References

- [ ] **Task:** Move user variable references ($name, ${name}) to ICM
- **Acceptance:** User variables resolved through ICM
- **Commit:** "PR02PH05T02: Migrate user variable references"

### PR02PH05T03 — Migrate System Variables (Numeric)

- [ ] **Task:** Move numeric system variable references (%1, %2, etc.) to ICM
- **Acceptance:** Session variables resolved through ICM, cleared on disconnect
- **Commit:** "PR02PH05T03: Migrate session variables"

### PR02PH05T04 — Migrate Alias References

- [ ] **Task:** Move @ alias references to ICM
- **Acceptance:** Alias references resolved through ICM
- **Commit:** "PR02PH05T04: Migrate alias references"

### PR02PH05T05 — Migrate System Variables (Named)

- [ ] **Task:** Move named system variables (%TIME, %CHARACTER) to ICM
- **Acceptance:** System variables handled through ICM
- **Commit:** "PR02PH05T05: Migrate system variables"

### PR02PH05T06 — Verify Compatibility

- [ ] **Task:** Verify existing alias chains and reference behavior compatible
- **Acceptance:** No behavioral regression
- **Commit:** "PR02PH05T06: Verify compatibility"

---

## Phase 6: Legacy Code Audit (PR02PH06)

### PR02PH06T01 — Audit Scattered Operator Logic

- [ ] **Task:** Search codebase for ad hoc operator handling, including service-side operational helpers that should become ICM-managed commands
- **Acceptance:** Documented list of audit findings
- **Commit:** "PR02PH06T01: Audit scattered operator logic"

### PR02PH06T02 — Convert Eligible Code

- [ ] **Task:** Convert scattered logic to ICM usage
- **Acceptance:** Reduced scattered implementations
- **Commit:** "PR02PH06T02: Convert eligible code"

### PR02PH06T03 — Document Conversion Decisions

- [ ] **Task:** Document why certain code was or wasn't converted
- **Acceptance:** Clear audit trail
- **Commit:** "PR02PH06T03: Document conversion decisions"

---

## Phase 7: New-Command Authoring Standard (PR02PH07)

### PR02PH07T01 — Create Process Documentation

- [ ] **Task:** Document repeatable process for future command additions
- **Acceptance:** Process documented and reviewed
- **Commit:** "PR02PH07T01: Create process documentation"

### PR02PH07T02 — Create Templates

- [ ] **Task:** Create templates for new command definitions with all required fields
- **Acceptance:** Templates available for future use
- **Commit:** "PR02PH07T02: Create templates"

### PR02PH07T03 — Create Checklist

- [ ] **Task:** Create checklist for command addition
- **Acceptance:** Checklist available for future specs
- **Commit:** "PR02PH07T03: Create checklist"

---

## Phase 8: Documentation (PR02PH08)

### PR02PH08T01 — Update Help Files

- [ ] **Task:** Update help documentation for ICM model
- **Acceptance:** Help reflects new command architecture
- **Commit:** "PR02PH08T01: Update help files"

### PR02PH08T02 — Create Release Notes

- [ ] **Task:** Document PR02 changes in release notes
- **Acceptance:** Release notes created
- **Commit:** "PR02PH08T02: Create release notes"

### PR02PH08T03 — Update Developer Guidance

- [ ] **Task:** Document ICM architecture for developers
- **Acceptance:** Developer guidance updated
- **Commit:** "PR02PH08T03: Update developer guidance"

---

## Phase 9: QA (PR02PH09)

### PR02PH09T01 — Deterministic Execution QA

- [ ] **Task:** Test deterministic execution order: user input → alias → variable → logic → queue → trigger
- **Acceptance:** Execution order verified
- **Commit:** "PR02PH09T01: Deterministic execution QA"

### PR02PH09T02 — Ingress Route QA

- [ ] **Task:** Test typed input, keybindings, history replay through ICM
- **Acceptance:** All ingress routes verified
- **Commit:** "PR02PH09T02: Ingress route QA"

### PR02PH09T03 — Safety Protection QA

- [ ] **Task:** Test recursion limits, max commands, queue backpressure, circuit breaker
- **Acceptance:** All safety protections fire correctly
- **Commit:** "PR02PH09T03: Safety protection QA"

### PR02PH09T04 — Literal Behavior QA

- [ ] **Task:** Test undefined variable preservation, escape sequences, unknown operators
- **Acceptance:** Literal behaviors verified
- **Commit:** "PR02PH09T04: Literal behavior QA"

### PR02PH09T04B — % Namespace QA

- [ ] **Task:** Test % namespace resolution: numeric %<n> session variables, named %NAME system variables, malformed % references, ambiguous inputs
- **Acceptance:** % namespace precedence rules verified, error/pass-through behaviors correct
- **Commit:** "PR02PH09T04B: % namespace QA"

### PR02PH09T05 — #LOG Governance QA

- [ ] **Task:** Test #LOG output format, routing, metadata
- **Acceptance:** #LOG governance verified
- **Commit:** "PR02PH09T05: #LOG governance QA"

### PR02PH09T06 — Regression Testing

- [ ] **Task:** Test all command paths for regressions
- **Acceptance:** No regressions found
- **Commit:** "PR02PH09T06: Regression testing"

---

## Phase 10: Commit to Production (PR02PH10)

### PR02PH10T01 — Final Build Verification

- [ ] **Task:** Build frontend and backend
- **Acceptance:** Both builds succeed
- **Commit:** "PR02PH10T01: Final build verification"

### PR02PH10T02 — Push to Staging

- [ ] **Task:** Push to staging branch
- **Acceptance:** CI/CD passes
- **Commit:** "PR02PH10T02: Push to staging"

### PR02PH10T03 — Master Promotion (Deferred)

- [ ] **Task:** Master promotion deferred to separate PR workflow
- **Acceptance:** N/A - handled separately
- **Note:** Production deployment to master occurs via separate PR, not this spec

### PR02PH10T04 — Branch Cleanup

- [ ] **Task:** Delete working branch
- **Acceptance:** Branch cleaned up
- **Commit:** "PR02PH10T04: Branch cleanup"

---

## Dependencies & Execution Order

### Phase Dependencies

- **PR02PH00 (Setup)**: No dependencies - can start immediately
- **PR02PH01 (Design)**: Depends on PH00 - BLOCKS core implementation
- **PR02PH02 (Core)**: Depends on PH01 - BLOCKS integration
- **PR02PH03 (Frontend)**: Depends on PH02 - frontend consumes contract
- **PR02PH02 (Core)**: Also blocks PH04 - backend authoritative execution
- **PR02PH04 (Backend)**: Depends on PH02 - backend execution contracts
- **PR02PH05 (Migration)**: Depends on PH02, PH03, PH04 - all integrations complete
- **PR02PH06 (Audit)**: Depends on PH05 - BLOCKS standards
- **PR02PH07 (Standards)**: Depends on PH06 - BLOCKS documentation
- **PR02PH08 (Docs)**: Depends on PH07 - can run in parallel with QA
- **PR02PH09 (QA)**: Depends on PH05, PH07 - may overlap with docs
- **PR02PH10 (Deploy)**: Depends on PH09 - FINAL PHASE

> **Note:** QA may overlap with Docs. QA validates implementation; Docs may refine after QA feedback.
