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

- [x] **Task:** Design the ICM command lifecycle: ingress → tokenize → validate → normalize → resolve → dispatch
- **Acceptance:** Documented lifecycle with clear phase definitions
- **Commit:** "PR02PH01T01: Define command lifecycle"
- **Design Document:** [.specify/plans/PR02PH01-design.md](.specify/plans/PR02PH01-design.md) - Section 1

### PR02PH01T02 — Define Frontend/Backend Contract

- [x] **Task:** Define API contract between frontend and backend for command processing
- **Acceptance:** Documented contract with request/response shapes
- **Commit:** "PR02PH01T02: Define frontend/backend contract"
- **Design Document:** [.specify/plans/PR02PH01-design.md](.specify/plans/PR02PH01-design.md) - Section 2

### PR02PH01T03 — Define Operator Families

- [x] **Task:** Catalog all operator families: #, @, %, $, and equivalent forms
- **Acceptance:** Documented operator family registry format
- **Commit:** "PR02PH01T03: Define operator families"
- **Design Document:** [.specify/plans/PR02PH01-design.md](.specify/plans/PR02PH01-design.md) - Section 3

### PR02PH01T03B — Define % Namespace Resolution Precedence

- [x] **Task:** Define % namespace resolution precedence: numeric %<n> is always session-scoped, named %NAME is always system-scoped. Define collision policy and error/malformed reference behavior.
- **Acceptance:** Documented precedence rules with numeric vs named resolution, collision handling, and pass-through behavior
- **Commit:** "PR02PH01T03B: Define % namespace resolution precedence"
- **Design Document:** [.specify/plans/PR02PH01-design.md](.specify/plans/PR02PH01-design.md) - Section 4

### PR02PH01T04 — Define Execution Contexts

- [x] **Task:** Define execution contexts: Preview, Submission, Automation, Operational
- **Acceptance:** Documented context definitions with authority levels
- **Commit:** "PR02PH01T04: Define execution contexts"
- **Design Document:** [.specify/plans/PR02PH01-design.md](.specify/plans/PR02PH01-design.md) - Section 5

### PR02PH01T05 — Define Normalization Format

- [x] **Task:** Define canonical command normalization format
- **Acceptance:** Documented normalization output shape
- **Commit:** "PR02PH01T05: Define normalization format"
- **Design Document:** [.specify/plans/PR02PH01-design.md](.specify/plans/PR02PH01-design.md) - Section 6

### PR02PH01T06 — Define Error Model

- [x] **Task:** Define error types, codes, and user-facing messages
- **Acceptance:** Documented error model
- **Commit:** "PR02PH01T06: Define error model"
- **Design Document:** [.specify/plans/PR02PH01-design.md](.specify/plans/PR02PH01-design.md) - Section 7

### PR02PH01T07 — Define Safety Hooks

- [x] **Task:** Define integration points for recursion limits, circuit breaker, rate limits
- **Acceptance:** Documented safety hook interface
- **Commit:** "PR02PH01T07: Define safety hooks"
- **Design Document:** [.specify/plans/PR02PH01-design.md](.specify/plans/PR02PH01-design.md) - Section 8

### PR02PH01T08 — Define Escape/Literal Grammar

- [x] **Task:** Define escape sequences, literal pass-through rules, and unknown operator handling
- **Acceptance:** Documented grammar for escape/literal behavior
- **Commit:** "PR02PH01T08: Define escape/literal grammar"
- **Design Document:** [.specify/plans/PR02PH01-design.md](.specify/plans/PR02PH01-design.md) - Section 9

### PR02PH01T09 — Define Pass-Through Behavior

- [x] **Task:** Define behavior for ordinary MUD commands that are not internal commands
- **Acceptance:** Documented pass-through classification rules
- **Commit:** "PR02PH01T09: Define pass-through behavior"
- **Design Document:** [.specify/plans/PR02PH01-design.md](.specify/plans/PR02PH01-design.md) - Section 10

### PR02PH01T10 — Define #LOG Governance

- [x] **Task:** Define #LOG directive visibility, permissions, output format, routing
- **Acceptance:** Documented #LOG specification
- **Commit:** "PR02PH01T10: Define #LOG governance"
- **Design Document:** [.specify/plans/PR02PH01-design.md](.specify/plans/PR02PH01-design.md) - Section 11

---

## Phase 2: ICM Core Implementation (PR02PH02)

### PR02PH02T01 — Create Backend ICM Core Structure

- [x] **Task:** Create backend ICM module structure with registry and handler resolution
- **Acceptance:** Backend module skeleton with component interfaces
- **Commit:** "PR02PH02T01: Create backend ICM core structure"
- **Files Created:**
  - `internal/icm/types.go` - Core type definitions
  - `internal/icm/errors.go` - Error types and codes
  - `internal/icm/engine.go` - Main ICM engine
  - `internal/icm/handler.go` - HTTP handler

### PR02PH02T02 — Implement Recognizer

- [x] **Task:** Implement tokenization and operator recognition
- **Acceptance:** Recognizer correctly identifies #, @, %, $ operators
- **Commit:** "PR02PH02T02: Implement recognizer"
- **File:** `internal/icm/recognizer.go`

### PR02PH02T03 — Implement Validator

- [x] **Task:** Implement syntax validation and argument checking
- **Acceptance:** Validator rejects invalid commands with clear errors
- **Commit:** "PR02PH02T03: Implement validator"
- **File:** `internal/icm/validator.go`

### PR02PH02T04 — Implement Normalizer

- [x] **Task:** Implement command normalization to canonical form
- **Acceptance:** Normalizer produces consistent output
- **Commit:** "PR02PH02T04: Implement normalizer"
- **File:** `internal/icm/normalizer.go`

### PR02PH02T05 — Implement Registry

- [x] **Task:** Implement command registry for handler registration
- **Acceptance:** Registry stores command definitions and metadata
- **Commit:** "PR02PH02T05: Implement registry"
- **File:** `internal/icm/registry.go`

### PR02PH02T06 — Implement Dispatcher

- [x] **Task:** Implement dispatcher with safety enforcement
- **Acceptance:** Dispatcher routes commands to handlers with safety checks
- **Commit:** "PR02PH02T06: Implement dispatcher"
- **File:** `internal/icm/dispatcher.go`

### PR02PH02T07 — Implement Escape/Literal Handling

- [x] **Task:** Implement escape sequence processing and literal pass-through
- **Acceptance:** Escape and literal behaviors work correctly
- **Commit:** "PR02PH02T07: Implement escape/literal handling"
- **File:** `internal/icm/escape.go`

### PR02PH02T08 — Implement Unknown Operator Handling

- [x] **Task:** Implement handling for unrecognized operators
- **Acceptance:** Unknown operators handled according to spec
- **Commit:** "PR02PH02T08: Implement unknown operator handling"
- **File:** `internal/icm/pass_through.go`

### PR02PH02T09 — Implement Pass-Through Classification

- [x] **Task:** Implement classification of commands as internal vs MUD commands
- **Acceptance:** Non-internal commands pass through correctly
- **Commit:** "PR02PH02T09: Implement pass-through classification"
- **File:** `internal/icm/pass_through.go`

### PR02PH02T10 — Create Frontend Adapter

- [x] **Task:** Create frontend adapter for validation, preview, and submission integration
- **Acceptance:** Frontend adapter communicates with backend ICM
- **Commit:** "PR02PH02T10: Create frontend adapter"
- **File:** `frontend/src/services/icm-adapter.ts`

### PR02PH02T11 — Backend Core Unit Tests

- [x] **Task:** Write unit tests for ICM recognizer, validator, normalization, and registry components
- **Acceptance:** Core ICM components have automated unit test coverage before frontend integration
- **Commit:** "PR02PH02T11: Backend core unit tests"
- **File:** `internal/icm/icm_test.go"

---

## Phase 3: Frontend Integration (PR02PH03)

### PR02PH03T01 — Integrate Typed Input

- [x] **Task:** Route typed command input through ICM adapter
- **Acceptance:** Commands processed through ICM pipeline
- **Commit:** "PR02PH03T01: Integrate typed input"
- **Files Modified:** `frontend/src/pages/PlayScreen.tsx`
  - Added ICM adapter import
  - Added `recognizeCommand()` call in `submitCommand()` to classify commands
  - Added `validateCommand()` call to validate internal commands before processing
  - Shows ICM validation errors in terminal for invalid commands

### PR02PH03T02 — Integrate Keybindings

- [x] **Task:** Route keybinding dispatch through ICM adapter
- **Acceptance:** Keybindings use canonical command pipeline
- **Commit:** "PR02PH03T02: Integrate keybindings"
- **Files Modified:** `frontend/src/pages/PlayScreen.tsx`
  - Keybindings automatically route through `submitCommand()` which now includes ICM integration
  - No additional changes needed - already uses canonical command pipeline

### PR02PH03T03 — Integrate History Replay

- [x] **Task:** Route history recall through ICM adapter
- **Acceptance:** History commands use full submission path
- **Commit:** "PR02PH03T03: Integrate history replay"
- **Files Modified:** `frontend/src/pages/PlayScreen.tsx`
  - History replay automatically routes through `submitCommand()` which now includes ICM integration
  - No additional changes needed - already uses canonical command pipeline

### PR02PH03T04 — Integrate Editors

- [x] **Task:** Route editor validation through ICM adapter
- **Acceptance:** Editors use ICM for syntax validation
- **Commit:** "PR02PH03T04: Integrate editors"
- **Files Modified:** `frontend/src/pages/SettingsPage.tsx`
  - Added ICM adapter import
  - Added `validateAliasReplacement()` function with ICM validation
  - Added `validateTriggerAction()` function with ICM validation
  - Added `validateTimerCommands()` function with ICM validation
  - Enhanced validation for alias, trigger, and timer editors

### PR02PH03T05 — Preserve Submission Behavior

- [x] **Task:** Ensure canonical submitCommand() behavior preserved
- **Acceptance:** No regression in command submission
- **Commit:** "PR02PH03T05: Preserve submission behavior"
- **Files Modified:** `frontend/src/pages/PlayScreen.tsx`
  - Submission behavior preserved: still gates on `isInputLocked`
  - Still processes through automation engine
  - Still sends to WebSocket
  - Still handles echo based on profile settings
  - Added ICM validation as a pre-processing step

---

## Phase 4: Backend Integration (PR02PH04)

### PR02PH04T01 — Implement Authoritative Execution

- [x] **Task:** Implement backend pathways for session-state commands
- **Acceptance:** Backend executes commands authoritatively
- **Commit:** "PR02PH04T01: Implement authoritative execution"
- **Changes:**
  - Added `ExecutionSession` type with session metadata
  - Added session management methods: `CreateSession`, `GetSession`, `UpdateSession`, `ClearSession`
  - Added session variable storage and retrieval
  - Added command history tracking

### PR02PH04T02 — Implement #LOG Directive

- [x] **Task:** Implement standardized logging directive per governance spec
- **Acceptance:** #LOG command routes through ICM with proper governance
- **Commit:** "PR02PH04T02: Implement #LOG directive"
- **Changes:**
  - Enhanced `LogHandler` with governance controls
  - Restricted to Automation and Operational contexts only
  - Added structured logging with metadata (timestamp, level, correlation ID)
  - Added log level support (debug, info, warn, error)
  - Added `LogEntry` type with proper JSON serialization

### PR02PH04T03 — Implement Command-State Effects

- [x] **Task:** Implement stateful command effects (variables, timers)
- **Acceptance:** State changes go through ICM
- **Commit:** "PR02PH04T03: Implement command-state effects"
- **Changes:**
  - Added `StateEffectHandler` interface for processing effects
  - Added timer command handlers: `StartTimerHandler`, `StopTimerHandler`, `CheckTimerHandler`, `CancelTimerHandler`
  - Added conditional handlers: `ElseHandler`, `EndIfHandler`
  - Enhanced `TimerHandler` with proper effect types
  - Added effect processing in engine

### PR02PH04T04 — Maintain Service-First Boundaries

- [x] **Task:** Ensure service-first architecture maintained
- **Acceptance:** Backend remains authoritative for session-state
- **Commit:** "PR02PH04T04: Maintain service-first boundaries"
- **Changes:**
  - Authority levels properly restrict preview vs submission vs automation vs operational
  - Session state managed entirely on backend
  - Command execution gated by context authority
  - Safety checks enforced at dispatcher level

### PR02PH04T05 — Backend Integration Tests

- [x] **Task:** Write integration tests for ICM handler registration, safety boundaries, and end-to-end command flows
- **Acceptance:** Core ICM components have automated test coverage
- **Commit:** "PR02PH04T05: Add backend unit/integration tests"
- **Tests Added:**
  - `TestEngine_ExecuteStructuredCommand` - Tests command execution
  - `TestEngine_LogCommand` - Tests #LOG directive
  - `TestEngine_TimerCommands` - Tests timer commands
  - `TestEngine_SessionManagement` - Tests session CRUD operations
  - `TestEngine_CommandHistory` - Tests command history
  - `TestEngine_StateEffects` - Tests state effect handling
  - `TestDispatcher_SafetyEnforcement` - Tests circuit breaker and rate limiting
  - `TestLogHandler_Governance` - Tests #LOG governance
  - `TestAuthorityLevels` - Tests authority context checks
  - `TestHandlerRegistration` - Tests handler registration
  - `TestEndToEnd_CommandPipeline` - Tests full pipeline execution

---

## Phase 5: Known Command Migration (PR02PH05)

### PR02PH05T01 — Migrate Structured Directives

- [x] **Task:** Move #IF, #SET, #TIMER, etc. to ICM
- **Acceptance:** Directives registered and processed through ICM
- **Commit:** "PR02PH05T01: Migrate structured directives"
- **Changes:**
  - All structured directive handlers registered in dispatcher: ECHO, LOG, HELP, SET, IF, TIMER, ELSE, ENDIF, START, STOP, CHECK, CANCEL
  - Recognizer properly identifies # prefix commands
  - Normalizer converts to canonical form
  - Tests verify all directives work correctly

### PR02PH05T02 — Migrate User Variable References

- [x] **Task:** Move user variable references ($name, ${name}) to ICM
- **Acceptance:** User variables resolved through ICM
- **Commit:** "PR02PH05T02: Migrate user variable references"
- **Changes:**
  - Recognizer parses $name and ${name} syntax
  - Normalizer expands $name to ${name} canonical form
  - Engine resolves user variables from registry
  - Pass-through behavior for undefined variables (returns literal ${name})
  - Tests verify user variable recognition and resolution

### PR02PH05T03 — Migrate System Variables (Numeric)

- [x] **Task:** Move numeric system variable references (%1, %2, etc.) to ICM
- **Acceptance:** Session variables resolved through ICM, cleared on disconnect
- **Commit:** "PR02PH05T03: Migrate session variables"
- **Changes:**
  - Recognizer parses numeric system variables (%1, %42, etc.)
  - Engine resolves session variables from session context
  - Session variables cleared on disconnect via ClearSession
  - Tests verify numeric system variable recognition

### PR02PH05T04 — Migrate Alias References

- [x] **Task:** Move @ alias references to ICM
- **Acceptance:** Alias references resolved through ICM
- **Commit:** "PR02PH05T04: Migrate alias references"
- **Changes:**
  - Recognizer parses @aliasname syntax
  - Registry stores aliases with expansion text
  - Engine resolves aliases through registry
  - Error returned for undefined aliases
  - Tests verify alias recognition and resolution

### PR02PH05T05 — Migrate System Variables (Named)

- [x] **Task:** Move named system variables (%TIME, %CHARACTER) to ICM
- **Acceptance:** System variables handled through ICM
- **Commit:** "PR02PH05T05: Migrate system variables"
- **Changes:**
  - Registry pre-registered with default system variables: TIME, DATE, CHARACTER, SERVER, SESSIONID, HOST, PORT
  - Recognizer parses named system variables
  - Normalizer uppercases names for consistency
  - Engine resolves through registry getter functions
  - Tests verify named system variable recognition

### PR02PH05T06 — Verify Compatibility

- [x] **Task:** Verify existing alias chains and reference behavior compatible
- **Acceptance:** No behavioral regression
- **Commit:** "PR02PH05T06: Verify compatibility"
- **Verification:**
  - All backend ICM tests pass (29 tests)
  - Frontend ICM adapter integrated with PlayScreen and SettingsPage
  - Escape sequences handled correctly (\#, \@, \$, \%)
  - Pass-through behavior verified for unknown commands
  - Authority levels enforced for different execution contexts

---

## Phase 6: Legacy Code Audit (PR02PH06)

### PR02PH06T01 — Audit Scattered Operator Logic

- [x] **Task:** Search codebase for ad hoc operator handling, including service-side operational helpers that should become ICM-managed commands
- **Acceptance:** Documented list of audit findings
- **Commit:** "PR02PH06T01: Audit scattered operator logic"
- **Audit Document:** [.specify/plans/PR02PH06-audit.md](.specify/plans/PR02PH06-audit.md) - Contains detailed findings on:
  - Scattered # prefix checks in automation.ts (3 locations)
  - @ alias invocation handling in automation.ts and evaluator.ts
  - $ variable substitution in frontend (intentional client-side)
  - % session variable handling in evaluator.ts
  - Timer command implementations in timer.ts
  - Parser directive handling in parser.ts
  - ICM already integrated at PlayScreen entry point

### PR02PH06T02 — Convert Eligible Code

- [x] **Task:** Convert scattered logic to ICM usage
- **Acceptance:** Reduced scattered implementations - 6 redundant prefix checks converted to ICM
- **Commit:** "PR02PH06T02: Convert redundant operator checks to ICM usage"
- **Changes Made:**
  - Added ICM adapter import to automation.ts and evaluator.ts
  - Added `isInternalCommand()` helper function using ICM `recognizeCommand()`
  - Refactored 3 redundant `#` prefix checks in automation.ts to use ICM
  - Refactored 1 redundant `@` prefix check in automation.ts to use ICM
  - Refactored 2 redundant prefix checks in evaluator.ts to use ICM
  - All remaining paths classified as non-authoritative helpers

### PR02PH06T03 — Document Conversion Decisions

- [x] **Task:** Document why certain code was or wasn't converted
- **Acceptance:** Clear audit trail with before/after reduction table
- **Commit:** "PR02PH06T03: Document conversion decisions"
- **Documentation:** Included in [.specify/plans/PR02PH06-audit.md](.specify/plans/PR02PH06-audit.md) - Contains:
  - Before/After reduction table (6 redundant checks removed)
  - Classification of remaining paths as "non-authoritative helpers"
  - Architectural compliance section
  - Files modified list

---

## Phase 7: New-Command Authoring Standard (PR02PH07)

### PR02PH07T01 — Create Process Documentation

- [x] **Task:** Document repeatable process for future command additions
- **Acceptance:** Process documented and reviewed
- **Commit:** "PR02PH07T01: Create process documentation"
- **Document:** [.specify/plans/PR02PH07-process.md](.specify/plans/PR02PH07-process.md) - Contains detailed process for:
  - Adding structured directives (#COMMAND)
  - Adding alias references (@alias)
  - Adding user variables ($name)
  - Adding system variables (%name)
  - Execution contexts and authority
  - Safety integration
  - Error handling
  - Frontend integration
  - Testing requirements
  - Documentation requirements

### PR02PH07T02 — Create Templates

- [x] **Task:** Create templates for new command definitions with all required fields
- **Acceptance:** Templates available for future use
- **Commit:** "PR02PH07T02: Create templates"
- **Templates Created:**
  - [.specify/templates/icm-command-go.template](.specify/templates/icm-command-go.template) - Go handler template
  - [.specify/templates/icm-system-variable-go.template](.specify/templates/icm-system-variable-go.template) - System variable template
  - [.specify/templates/icm-adapter-typescript.template](.specify/templates/icm-adapter-typescript.template) - Frontend adapter template
  - [.specify/templates/icm-command-spec.md](.specify/templates/icm-command-spec.md) - Command specification template

### PR02PH07T03 — Create Checklist

- [x] **Task:** Create checklist for command addition
- **Acceptance:** Checklist available for future specs
- **Commit:** "PR02PH07T03: Create checklist"
- **Checklist:** [.specify/plans/PR02PH07-checklist.md](.specify/plans/PR02PH07-checklist.md) - Contains:
  - Phase 1: Specification checklist
  - Phase 2: Backend implementation checklist
  - Phase 3: Safety integration checklist
  - Phase 4: Testing checklist
  - Phase 5: Frontend integration checklist
  - Phase 6: Documentation checklist
  - Phase 7: Review checklist
  - Quick reference (error codes, file locations, contexts)

---

## Phase 8: Documentation (PR02PH08)

### PR02PH08T01 — Update Help Files

- [x] **Task:** Update help documentation for ICM model
- **Acceptance:** Help reflects new command architecture
- **Commit:** "PR02PH08T01: Update help files"
- **Changes:**
  - Created `help/internal-commands.json` - Comprehensive guide to ICM operators
  - Updated `help/variables.json` - Added system variables (%)
  - Updated `help/aliases.json` - Added @ direct alias invocation
  - Updated `help/triggers.json` - Added variable/alias usage in triggers

### PR02PH08T02 — Create Release Notes

- [x] **Task:** Document PR02 changes in release notes
- **Acceptance:** Release notes created
- **Commit:** "PR02PH08T02: Create release notes"
- **Document:** [.specify/plans/PR02-release-notes.md](.specify/plans/PR02-release-notes.md)
- **Contents:**
  - Overview of ICM features
  - New features (operators, directives, system variables)
  - Architecture changes
  - Breaking changes (none)
  - Migration notes
  - QA verification

### PR02PH08T03 — Update Developer Guidance

- [x] **Task:** Document ICM architecture for developers
- **Acceptance:** Developer guidance updated
- **Commit:** "PR02PH08T03: Update developer guidance"
- **Document:** [.specify/plans/PR02PH08-developer-guidance.md](.specify/plans/PR02PH08-developer-guidance.md)
- **Contents:**
  - Architecture overview
  - Core components
  - Command lifecycle
  - Execution contexts
  - Adding new commands
  - Frontend integration
  - Safety integration
  - Testing
  - Troubleshooting

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
