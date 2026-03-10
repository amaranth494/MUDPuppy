# PR01 — Post-MVP Foundation & Advanced Automation Logic Plan

> **Plan ID:** PR01-plan  
> **Spec Reference:** PR01 — Post-MVP Foundation & Advanced Automation Logic  
> **Status:** Draft  
> **Pre-Release Version:** 0.1.0-pr1  
> **Constitutional Reference:** Constitution III.a (Pre-Release Feature Scope Evolution), Constitution III.b (Automation Engine)

---

## Phase Overview

| Phase | Focus | Description |
|-------|-------|-------------|
| PR01PH00 | Setup | Create branch, extend schema for timer storage |
| PR01PH01 | Parser Foundation | # command parser, tokenization, syntax validation |
| PR01PH02 | Logic Engine | #IF/#ELSE/#ENDIF evaluation, variable substitution |
| PR01PH03 | Variable Operations | #SET command, variable storage in profile |
| PR01PH04 | Timer System | #TIMER, #ENDTIMER, #CANCEL, async scheduler |
| PR01PH05 | Safety Controls | Timer limits, nested logic limits, evaluation timeout |
| PR01PH06 | Editor UI | Multi-line editing, syntax validation UI |
| PR01PH07 | Integration | Connect logic to existing triggers/aliases |
| PR01PH08 | QA Verification | Manual QA test scenarios for all features |
| PR01PH09 | Documentation | Update help files, add Release Notes |
| PR01PH10 | Commit to Production | Build, push to staging, push to master, cleanup branch |

---

## Phase 0: Setup (PR01PH00)

### Objectives

- Create feature branch for PR01
- No schema changes required (timer state is session-only, variables use existing profile storage)
- No frontend type changes required initially

### Steps

1. **T01: Create feature branch**
   - Branch from staging
   - Name: `pr01-automation-logic`

2. **T02: Review existing automation code**
   - Examine current automation engine (`frontend/src/services/automation.ts`)
   - Identify integration points for new logic parser
   - Document current execution flow

3. **T03: No schema changes needed**
   - Timer state is session-only (not persisted)
   - Variables use existing profile storage (already in schema)

---

## Phase 1: Parser Foundation (PR01PH01)

### Objectives

- Implement tokenizer for # commands
- Validate # command syntax
- Parse automation action text into executable AST

### Steps

1. **T01: Create parser module**
   - Create `automation/parser.ts` in frontend
   - Define token types: COMMAND, TEXT, CONDITION, VARIABLE

2. **T02: Implement tokenizer**
   - Tokenize `#IF`, `#ELSE`, `#ENDIF`, `#SET`, `#TIMER`, `#ENDTIMER`, `#CANCEL`
   - Handle variable substitution `${...}`
   - Preserve plain text commands

3. **T03: Implement syntax validation**
   - Validate # command structure
   - Check matching #IF/#ENDIF pairs
   - Report syntax errors with line numbers

4. **T04: Add unit tests**
   - Test tokenizer for each # command
   - Test syntax validation errors

---

## Phase 2: Logic Engine (PR01PH02)

### Objectives

- Implement #IF condition evaluation
- Support comparison operators: > < >= <= ==
- Support logical operators: AND, OR
- Implement variable substitution in conditions

### Steps

1. **T01: Implement condition parser**
   - Parse `${variable}` references
   - Parse comparison expressions
   - Parse logical AND/OR expressions

2. **T02: Implement evaluation engine**
   - Evaluate conditions against variable values
   - Handle type coercion (string vs number comparison)
   - Return true/false for #IF decisions

3. **T03: Implement #IF/#ELSE/#ENDIF execution**
   - Execute commands based on condition results
   - Support nested #IF blocks (up to 3 levels)
   - Track execution depth for safety

4. **T04: Add unit tests**
   - Test condition evaluation
   - Test variable substitution
   - Test #IF/#ELSE execution

---

## Phase 3: Variable Operations (PR01PH03)

### Objectives

- Implement #SET command
- Store variables in profile
- Support variable types: string, number, boolean

### Steps

1. **T01: Implement #SET command parser**
   - Parse `#SET variable_name value`
   - Handle quoted strings
   - Handle numeric values

2. **T02: Implement variable storage**
   - Store variables in SessionContext
   - Persist variables to profile via API
   - Load variables on session connect

3. **T03: Implement variable substitution**
   - Replace ${variable_name} with value
   - Handle undefined variables (empty string)
   - Support variable in commands and conditions

4. **T04: Add unit tests**
   - Test #SET command parsing
   - Test variable persistence
   - Test variable substitution

---

## Phase 4: Timer System (PR01PH04)

### Objectives

- Implement #TIMER command with delayed execution
- Implement #TIMER ... REPEAT for repeating timers
- Implement #CANCEL command
- Manage timer lifecycle (create, execute, cancel)

### Steps

1. **T01: Define timer types**
   - Create Timer interface: id, name, duration, repeat, commands
   - Create TimerManager class

2. **T02: Implement #TIMER parsing**
   - Parse `#TIMER name duration` (delayed)
   - Parse `#TIMER name duration REPEAT` (repeating)
   - Extract commands between #TIMER and #ENDTIMER

3. **T03: Implement timer execution**
   - Use setTimeout/setInterval in frontend
   - Execute commands when timer fires
   - Handle repeating timers (auto-reset)

4. **T04: Implement #CANCEL**
   - Parse `#CANCEL timer_name`
   - Remove timer from active timers
   - Handle invalid timer name gracefully

5. **T05: Add unit tests**
   - Test timer creation
   - Test timer execution
   - Test timer cancellation

---

## Phase 5: Safety Controls (PR01PH05)

### Objectives

- Implement timer limit (max 10 active timers)
- Implement nested logic limit (max 3 levels)
- Implement evaluation timeout (max 500ms)
- Integrate with existing MVP safety (circuit breaker, rate limits)

### Steps

1. **T01: Implement timer limit**
   - Track active timer count
   - Reject new timers when limit reached
   - Return error to user

2. **T02: Implement nested logic limit**
   - Track #IF execution depth
   - Reject execution beyond 3 levels
   - Track depth in execution context

3. **T03: Implement evaluation timeout**
   - Wrap condition evaluation in timeout
   - Abort if evaluation exceeds 500ms
   - Log timeout events

4. **T04: Integrate with MVP safety**
   - Connect to existing circuit breaker
   - Connect to existing rate limiting
   - Coordinate safety system responses

5. **T05: Add unit tests**
   - Test timer limit enforcement
   - Test nested logic limit
   - Test evaluation timeout

---

## Phase 6: Editor UI (PR01PH06)

### Objectives

- Enhance automation editors for multi-line input
- Add syntax validation display
- Support indentation

### Steps

1. **T01: Update AliasEditor component**
   - Change action input to textarea
   - Add line numbers
   - Add placeholder showing # syntax examples

2. **T02: Update TriggerEditor component**
   - Change action input to textarea
   - Add line numbers
   - Add placeholder showing # syntax examples

3. **T03: Add syntax validation UI**
   - Validate on blur/change
   - Show inline errors
   - Color-code # commands

4. **T04: Test editor enhancements**
   - Manual testing of multi-line input
   - Verify syntax highlighting works
   - Verify error messages display

---

## Phase 7: Integration (PR01PH07)

### Objectives

- Connect new parser to existing automation execution
- Ensure triggers can use # commands
- Ensure aliases can use # commands

### Steps

1. **T01: Integrate parser with trigger execution**
   - Pass trigger action through parser
   - Execute parsed commands
   - Handle variable substitution

2. **T02: Integrate parser with alias execution**
   - Pass alias replacement through parser
   - Execute parsed commands
   - Handle variable substitution

3. **T03: Integrate timer execution**
   - Timer commands also parsed
   - Timers execute parsed commands
   - Timers share variable context

4. **T04: End-to-end testing**
   - Test complete trigger with #IF
   - Test complete alias with #SET
   - Test complete timer with #TIMER

---

## Phase 8: QA Verification (PR01PH08)

### Objectives

- Manual QA testing for all features
- Verify safety controls work
- Document test scenarios

### Steps

1. **T01: Create QA test scenarios**
   - Document all test cases from spec
   - Include edge cases
   - Include error scenarios

2. **T02: Execute QA testing**
   - Test conditional automation (#IF/#ELSE/#ENDIF)
   - Test timer system (#TIMER, #CANCEL)
   - Test variable operations (#SET)
   - Test safety limits

3. **T03: Fix issues found**
   - Address any bugs discovered
   - Re-test after fixes

4. **T04: Document results**
   - Record test outcomes
   - Note any limitations
   - Identify future improvements

---

## Phase 9: Documentation (PR01PH09)

### Objectives

- Update help files with new automation features
- Add Release Notes section to help system
- Ensure all new features are documented

### Steps

1. **T01: Create timers help file**
   - Create `/help/timers.json`
   - Document #TIMER syntax
   - Document #CANCEL syntax
   - Include examples for delayed and repeating timers

2. **T02: Create advanced automation help file**
   - Create `/help/automation.json`
   - Document #IF/#ELSE/#ENDIF syntax
   - Document #SET command
   - Document variable substitution
   - Include examples for conditional automation

3. **T03: Update existing help files**
   - Update `/help/aliases.json` with #SET examples
   - Update `/help/triggers.json` with #IF examples
   - Update `/help/variables.json` with #SET documentation

4. **T04: Create Release Notes help file**
   - Create `/help/release-notes.json`
   - Add entry for version 0.1.0-pr1
   - Include summary of PR01 features
   - Document all new automation capabilities

5. **T05: Update Help navigation**
   - Add Timers section to Help navigation
   - Add Advanced Automation section to Help navigation
   - Add Release Notes to Help navigation

---

## Phase 10: Commit to Production (PR01PH10)

### Objectives

- Final build verification for frontend and backend
- Proper Git workflow execution
- Branch cleanup after successful completion

### Steps

1. **T01: Final Build Verification**
   - Build frontend one final time: `cd frontend && npm run build`
   - Build backend one final time: `go build -o mudpuppy-server ./cmd/server`
   - Verify both builds succeed without errors

2. **T02: Push to Git Staging**
   - Push working branch `pr01-automation-logic` to staging branch
   - Verify push succeeds
   - Wait for CI/CD pipeline to complete successfully

3. **T03: Push to Git Master**
   - Push from staging to master/main
   - This triggers production deployment
   - Verify push succeeds

4. **T04: Branch Cleanup**
   - Switch to staging branch: `git checkout staging`
   - Delete working branch: `git branch -D pr01-automation-logic`
   - Verify cleanup completed

> **This is the VERY LAST step for this spec. No spec is considered complete until this workflow is executed.**

---

## Technical Context

**Language/Version:** TypeScript 5.x, Go 1.21  
**Primary Dependencies:** React 18, xterm.js (existing), Redis (existing), PostgreSQL (existing)  
**Storage:** Profile storage (existing), Session state (new - session-only)  
**Testing:** Jest (frontend), Go testing (backend)  
**Target Platform:** Web browser, Linux server  
**Project Type:** Web application with backend API  
**Performance Goals:** Timer precision within 100ms, parser under 10ms  
**Constraints:** Session-only timer state (not persisted), max 10 active timers  
**Scale/Scope:** Per-connection automation, single-user context

---

## Project Structure

### Documentation (this feature)

```
.specify/
├── specs/PR01.md              # This specification
├── plans/PR01-plan.md         # This plan
└── tasks/PR01-tasks.md       # Task tracker
```

### Source Code (repository root)

```
frontend/src/
├── services/
│   ├── automation.ts          # Existing automation engine
│   └── automation/
│       ├── parser.ts          # NEW: Parser module
│       ├── evaluator.ts       # NEW: Logic evaluation
│       ├── timer.ts           # NEW: Timer system
│       └── safety.ts          # NEW: Safety controls
├── components/
│   ├── AliasEditor.tsx        # EXISTING: Update for multi-line
│   └── TriggerEditor.tsx     # EXISTING: Update for multi-line
└── context/
    └── SessionContext.tsx     # EXISTING: Add timer management

backend/
└── internal/
    └── profiles/
        └── handler.go         # EXISTING: Variable persistence
```

---

## Complexity Tracking

> **No complexity violations for this specification.**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|---------------------------------------|
| None | N/A | N/A |

---

## Dependencies & Execution Order

### Phase Dependencies

- **PR01PH00 (Setup)**: No dependencies - can start immediately
- **PR01PH01 (Parser)**: Depends on PR01PH00 - BLOCKS logic engine
- **PR01PH02 (Logic Engine)**: Depends on PR01PH01 - BLOCKS variable operations
- **PR01PH03 (Variables)**: Depends on PR01PH01 - Can parallel with PH02
- **PR01PH04 (Timers)**: Depends on PR01PH01 - BLOCKS safety controls
- **PR01PH05 (Safety)**: Depends on PR01PH02, PH03, PH04 - BLOCKS editor UI
- **PR01PH06 (Editor UI)**: Depends on PR01PH05 - BLOCKS integration
- **PR01PH07 (Integration)**: Depends on PR01PH06 - BLOCKS QA
- **PR01PH08 (QA)**: Depends on PR01PH07 - BLOCKS Documentation
- **PR01PH09 (Documentation)**: Depends on PR01PH08 - BLOCKS Commit to Production
- **PR01PH10 (Commit to Production)**: Depends on PR01PH09 - FINAL PHASE

### Parallel Opportunities

- PR01PH02 (Logic Engine) and PR01PH03 (Variables) can proceed in parallel after PH01
- PR01PH04 (Timers) can proceed after PH01 independently
- All phases after PH05 must be sequential

---

## Implementation Strategy

### Sequential Delivery

1. Complete PR01PH00: Setup → Foundation ready
2. Complete PR01PH01: Parser → Core parsing ready
3. Complete PR01PH02: Logic Engine → Conditional execution ready
4. Complete PR01PH03: Variables → Variable operations ready
5. Complete PR01PH04: Timers → Timer system ready
6. Complete PR01PH05: Safety → Safety controls ready
7. Complete PR01PH06: Editor UI → UI enhancements ready
8. Complete PR01PH07: Integration → Full automation ready
9. Complete PR01PH08: QA → Release ready

### Each Phase Adds Value

- Parser alone enables syntax validation
- Logic engine enables conditional automation
- Variables enable dynamic behavior
- Timers enable scheduled actions
- Safety ensures protection
- Editor UI enables user-friendly editing
- Integration enables full feature
- QA ensures quality

---

## Notes

- Timer state is session-only (not persisted across sessions)
- Variables are persisted via existing profile storage
- All # commands are parsed before execution
- Safety limits are enforced at parse/execution time
- Editor UI changes are additive (no breaking changes)
- Backward compatibility maintained with MVP automation
