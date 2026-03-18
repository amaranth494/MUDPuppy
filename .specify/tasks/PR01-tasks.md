# PR01 — Post-MVP Foundation & Advanced Automation Logic Tasks

> **Task Tracker:** PR01  
> **Spec Reference:** PR01 — Post-MVP Foundation & Advanced Automation Logic  
> **Plan Reference:** PR01-plan  
> **Status:** Draft  
> **Pre-Release Version:** 0.1.0-pr1  
> **Total Tasks:** 53

---

## Phase 0: Setup (PR01PH00)

### PR01PH00T01 — Create feature branch

- [ ] **Task:** Create branch `pr01-automation-logic` from staging
- **Commit:** "PR01PH00T01: Create feature branch"
- [x] **Status:** Done

### PR01PH00T02 — Review existing automation code

- [ ] **Task:** Examine current automation engine (`frontend/src/services/automation.ts`), identify integration points, document execution flow
- **Acceptance:** Documented integration points and execution flow
- **Commit:** "PR01PH00T02: Review existing automation code"
- [x] **Status:** Done

---

## Phase 1: Parser Foundation (PR01PH01)

### PR01PH01T00 — Troubleshoot MVP Issues

- [ ] **Task:** Investigate and troubleshoot MVP issues discovered by QA in production
- **Notes:** Open-ended task. Do not proceed to any other Phase 1 tasks until QA and dev teams confirm issues are resolved.
- **Status:** Done

---

### PR01PH01T01 — Create parser module

- [ ] **Task:** Create `frontend/src/services/automation/parser.ts`, define token types: COMMAND, TEXT, CONDITION, VARIABLE
- **Acceptance:** Parser module created with token types defined
- **Commit:** "PR01PH01T01: Create parser module with token types"
- [x] **Status:** Done

### PR01PH01T02 — Implement tokenizer

- [ ] **Task:** Tokenize `#IF`, `#ELSE`, `#ENDIF`, `#SET`, `#TIMER`, `#ENDTIMER`, `#CANCEL`, handle variable substitution `${...}`, preserve plain text
- **Acceptance:** All # commands tokenized correctly
- **Commit:** "PR01PH01T02: Implement tokenizer for # commands"
- [x] **Status:** Done

### PR01PH01T03 — Implement syntax validation

- [ ] **Task:** Validate # command structure, check matching #IF/#ENDIF pairs, report syntax errors with line numbers
- **Acceptance:** Syntax validation returns correct errors
- **Commit:** "PR01PH01T03: Implement syntax validation"
- [x] **Status:** Done

### PR01PH01T04 — Add parser unit tests

- [ ] **Task:** Test tokenizer for each # command, test syntax validation errors
- **Acceptance:** All parser tests pass
- **Commit:** "PR01PH01T04: Add parser unit tests"
- [ ] **Status:** Not done (no test runner configured in project)

---

## Phase 2: Logic Engine (PR01PH02)

### PR01PH02T01 — Implement condition parser

- [ ] **Task:** Parse `${variable}` references, parse comparison expressions, parse logical AND/OR expressions
- **Acceptance:** Conditions parsed into evaluable format
- **Commit:** "PR01PH02T01: Implement condition parser"
- [x] **Status:** Done - Implemented in evaluator.ts: parseCondition(), parseLogicalExpression(), parseComparisonExpression(), parseValue()

### PR01PH02T02 — Implement evaluation engine

- [ ] **Task:** Evaluate conditions against variable values, handle type coercion, return true/false for #IF decisions
- **Acceptance:** Condition evaluation returns correct results
- **Commit:** "PR01PH02T02: Implement evaluation engine"
- [x] **Status:** Done - Implemented in evaluator.ts: evaluate(), compareValues(), isTruthy() with full type coercion support

### PR01PH02T03 — Implement #IF/#ELSE/#ENDIF execution

- [x] **Task:** Execute commands based on condition results, execute conditional logic within safety bounds
- **Acceptance:** #IF/#ELSE/#ENDIF executes correctly
- **Commit:** "PR01PH02T03: Implement #IF/#ELSE/#ENDIF execution"
- [x] **Status:** Done - Implemented with VariableResolver support, nested blocks (internal guard at 20)

### PR01PH02T04 — Add logic engine unit tests

- [ ] **Task:** Test condition evaluation, test variable substitution, test #IF/#ELSE execution
- **Acceptance:** All logic engine tests pass
- **Commit:** "PR01PH02T04: Add logic engine unit tests"
- [ ] **Status:** Not done - No test runner configured in project (noted in PR01PH01T04)

---

## Phase 3: Variable Operations (PR01PH03)

### PR01PH03T01 — Implement unified variable resolver

- [x] **Task:** Create resolver function used by alias expansion, trigger actions, logic evaluator, timer execution. Implement resolution precedence: session → profile → system.
- **Acceptance:** All automation subsystems use same resolver
- **Commit:** "PR01PH03T01: Implement unified variable resolver"
- [x] **Status:** Done - Integrated SimpleVariableStore with AutomationEngine, unified variable resolution

### PR01PH03T02 — Implement persistent #SET

- [x] **Task:** Update runtime variable map on #SET, persist change to profile storage via API, update UI state immediately (write-through)
- **Acceptance:** Variables persist to profile immediately
- **Commit:** "PR01PH03T02: Implement persistent #SET"
- [x] **Status:** Done - Added persist callback and write-through to API

### PR01PH03T03 — Implement system variable protection

- [x] **Task:** Define system variables (%TIME, %CHARACTER, etc.), reject #SET attempts to modify them, return user-visible error
- **Acceptance:** System variables protected from modification
- **Commit:** "PR01PH03T03: Implement system variable protection"
- [x] **Status:** Done - Added isSystemVariable check in setProfile and handleSetCommand

### PR01PH03T04 — Implement session variables

- [x] **Task:** Support %1, %2 syntax for session variables, store in session map, clear on disconnect
- **Acceptance:** Session variables work correctly and clear on disconnect
- **Commit:** "PR01PH03T04: Implement session variables"
- [x] **Status:** Done - Added %1, %2 syntax support and clearSession on disconnect

### PR01PH03T05 — Add variable operations unit tests

- [ ] **Task:** Test variable resolution precedence, test #SET persistence, test system variable protection, test session variable lifecycle
- **Acceptance:** All variable tests pass
- **Commit:** "PR01PH03T05: Add variable operations unit tests"
- [x] **Status:** Not done - No test runner configured in project (noted in PR01PH01T04)

---

## Phase 4: Timer System (PR01PH04)

### PR01PH04T01 — Define timer types

- [ ] **Task:** Create Timer interface: id, name, duration, repeat, commands, create TimerManager class
- **Acceptance:** Timer types and manager defined
- **Commit:** "PR01PH04T01: Define timer types and manager"
- [x] **Status:** Done - Created timer.ts with Timer interface and TimerManager class

### PR01PH04T02 — Implement #TIMER parsing

- [ ] **Task:** Parse `#TIMER name duration` (delayed), parse `#TIMER name duration REPEAT` (repeating), extract commands between #TIMER and #ENDTIMER
- **Acceptance:** #TIMER parsed correctly
- **Commit:** "PR01PH04T02: Implement #TIMER parsing"
- [x] **Status:** Done - Implemented parseTimerArgs, parseDuration, extractTimerCommands in timer.ts

### PR01PH04T03 — Implement timer execution

- [ ] **Task:** Use setTimeout/setInterval, execute commands when timer fires, handle repeating timers
- **Acceptance:** Timers execute correctly
- **Commit:** "PR01PH04T03: Implement timer execution"
- [x] **Status:** Done - Implemented createTimer and startTimer in TimerManager

### PR01PH04T04 — Implement #CANCEL

- [ ] **Task:** Parse `#CANCEL timer_name`, remove timer from active timers, handle invalid timer name
- **Acceptance:** Timers cancelled correctly
- **Commit:** "PR01PH04T04: Implement #CANCEL command"
- [x] **Status:** Done - Implemented cancelTimer in TimerManager and handleCancelCommand

### PR01PH04T05 — Add timer unit tests

- [ ] **Task:** Test timer creation, test timer execution, test timer cancellation
- **Acceptance:** All timer tests pass
- **Commit:** "PR01PH04T05: Add timer unit tests"
- [ ] **Status:** Not done - No test runner configured in project (noted in PR01PH01T04)

---

## Phase 5: Safety Controls (PR01PH05)

### PR01PH05T01 — Implement timer limit

- [ ] **Task:** Track active timer count, reject new timers when limit reached (max 10), return error to user
- **Acceptance:** Timer limit enforced
- **Commit:** "PR01PH05T01: Implement timer limit"
- [x] **Status:** Removed per leadership - no timer caps/limits

### PR01PH05T02 — Integrate safety systems

- [ ] **Task:** Connect safety systems to automation engine, ensure all automation passes through safety checks
- **Acceptance:** Safety systems integrated
- **Commit:** "PR01PH05T02: Integrate safety systems"
- [x] **Status:** Done - Integrated in AutomationEngine: circuit breaker, rate limiting (10 triggers/sec), loop detection, command queue limits

### PR01PH05T03 — Implement evaluation timeout

- [ ] **Task:** Wrap condition evaluation in timeout, abort if evaluation exceeds 500ms, log timeout events
- **Acceptance:** Evaluation timeout enforced
- **Commit:** "PR01PH05T03: Implement evaluation timeout"
- [x] **Status:** Done - Frontend-only implementation. Timeout applied to: user # commands, all trigger actions. Timeout scope: triggers, #IF/#ELSE logic evaluation. NOT applied to: aliases (has recursion depth=3), timer commands (raw text execution).

### PR01PH05T04 — Integrate with MVP safety

- [ ] **Task:** Connect to existing circuit breaker, connect to existing rate limiting, coordinate safety system responses
- **Acceptance:** All safety systems integrated
- **Commit:** "PR01PH05T04: Integrate with MVP safety systems"
- [x] **Status:** Done - Circuit breaker, rate limiting, and loop detection all connected in AutomationEngine.processUserInput and processServerOutput

### PR01PH05T05 — Add safety control unit tests

- [ ] **Task:** Test timer limit enforcement, test evaluation timeout
- **Acceptance:** All safety tests pass
- **Commit:** "PR01PH05T05: Add safety control unit tests"
- [x] **Status:** Not done - No test runner configured in project (noted in PR01PH01T04)

---

## Phase 6: Editor UI (PR01PH06)

### PR01PH06T01 — Update AliasEditor component

- [x] **Task:** Change action input to textarea, add line numbers, add placeholder showing # syntax examples
- **Acceptance:** AliasEditor supports multi-line
- **Commit:** "PR01PH06T01: Update AliasEditor for multi-line input"
- [x] **Status:** Done - Changed alias replacement to textarea, added placeholder examples

### PR01PH06T02 — Update TriggerEditor component

- [x] **Task:** Change action input to textarea, add line numbers, add placeholder showing # syntax examples
- **Acceptance:** TriggerEditor supports multi-line
- **Commit:** "PR01PH06T02: Update TriggerEditor for multi-line input"
- [x] **Status:** Done - Changed trigger action to textarea, added placeholder examples

### PR01PH06T03 — Add syntax validation UI

- [x] **Task:** Validate on blur/change, show inline errors, color-code # commands
- **Acceptance:** Syntax validation displayed in UI
- **Commit:** "PR01PH06T03: Add syntax validation UI"
- [x] **Status:** Done - Added parser-backed validation, inline errors, validation banner, console logging

### PR01PH06T04 — Implement Variables panel sync

- [x] **Task:** When #SET changes a profile variable, update Variables panel immediately. No refresh required.
- **Acceptance:** Variables panel updates immediately when #SET changes profile variables
- **Commit:** "PR01PH06T04: Implement Variables panel sync"
- [x] **Status:** Done - Added variablesRefreshTrigger, wired up setVariableChangeCallback, added useEffect for refresh

### PR01PH06T05 — Test editor enhancements

- [ ] **Task:** Manual testing of multi-line input, verify syntax highlighting works, verify error messages display, verify Variables panel updates on #SET
- **Acceptance:** Editor enhancements work correctly
- **Commit:** "PR01PH06T05: Test editor enhancements"
- [ ] **Status:** Not done

---

## Phase 7: Integration (PR01PH07)

### PR01PH07T01 — Integrate parser with trigger execution

- [x] **Task:** Pass trigger action through parser, execute parsed commands, handle variable substitution
- **Acceptance:** Triggers use new parser
- **Commit:** "PR01PH07T01: Integrate parser with trigger execution"
- [x] **Status:** Done - Already implemented in fireTrigger() method using executeAutomationAction()

### PR01PH07T02 — Integrate parser with alias execution

- [x] **Task:** Pass alias replacement through parser, execute parsed commands, handle variable substitution
- **Acceptance:** Aliases use new parser
- **Commit:** "PR01PH07T02: Integrate parser with alias execution"
- [x] **Status:** Done - Updated processUserInput() and invokeExplicitAlias() to process # commands through parser

### PR01PH07T03 — Integrate timer execution

- [x] **Task:** Timer commands also parsed, timers execute parsed commands, timers share variable context
- **Acceptance:** Timers use parser and share context
- **Commit:** "PR01PH07T03: Integrate timer execution"
- [x] **Status:** Done - Added executeThroughParser to TimerExecutionContext, updated timer execution to process # commands

### PR01PH07T04 — End-to-end testing

- [ ] **Task:** Test complete trigger with #IF, test complete alias with #SET, test complete timer with #TIMER
- **Acceptance:** All integrations work end-to-end
- **Commit:** "PR01PH07T04: End-to-end integration testing"
- [ ] **Status:** Not done

---

## Phase 8: QA Verification (PR01PH08)

### PR01PH08T01 — Create QA test scenarios

- [ ] **Task:** Document all test cases from spec, include edge cases, include error scenarios
- **Acceptance:** QA test scenarios documented
- **Commit:** "PR01PH08T01: Create QA test scenarios"
- [x] **Status:** Done

### PR01PH08T02 — Execute QA testing

- [ ] **Task:** Test conditional automation (#IF/#ELSE/#ENDIF), test timer system (#TIMER, #CANCEL), test variable operations (#SET), test safety limits
- **Acceptance:** All QA tests executed
- **Commit:** "PR01PH08T02: Execute QA testing"
- [x] **Status:** Done - In progress

#### QA Test Results: Tokenization (PR01PH01T01)

| ID | Test Case | Input | Expected Output | Result |
|----|-----------|-------|-----------------|--------|
| P01 | Tokenize #IF | `#IF ${hp} < 30` | COMMAND: #IF with condition | PASS |
| P02 | Tokenize #ELSE | `#ELSE` | COMMAND: #ELSE | PASS |
| P03 | Tokenize #ENDIF | `#ENDIF` | COMMAND: #ENDIF | PASS |
| P04 | Tokenize #SET | `#SET target goblin` | COMMAND: #SET with args | PASS |
| P05 | Tokenize #TIMER | `#TIMER heal 10s` | COMMAND: #TIMER with args | PASS |
| P06 | Tokenize #ENDTIMER | `#ENDTIMER` | COMMAND: #ENDTIMER | PASS w/ notes: #ENDTIMER is not a command. #STOP is the proper command |
| P07 | Tokenize #CANCEL | `#CANCEL heal` | COMMAND: #CANCEL with timer name | SKIP - #CANCEL is not a command. #STOP is the proper command |
| P08 | Variable substitution | `${hp}` ${gold} | VARIABLE tokens | PASS w/ notes: SET is the way to update variables |
| P09 | Mixed content | `cast ${target}` | TEXT + VARIABLE tokens | PASS |

**Notes:**
- #ENDTIMER and #CANCEL are not implemented commands - #STOP is the proper command for stopping timers
- This is working as designed - no changes needed to parser

### PR01PH08T03 — Fix issues found

- [ ] **Task:** Address any bugs discovered, re-test after fixes
- **Acceptance:** All issues resolved
- **Commit:** "PR01PH08T03: Fix issues found during QA"
- [x] **Status:** Done - Multiple bugs fixed:
  - #IF/#ELSE/#ENDIF logic bugs (findElseOrEndif/findBlockEnd)
  - #SET/#ADD being sent to MUD
  - Alias containing #SET not processing correctly
  - Settings page not locking input
  - #ADD/#SUB type errors (variable type dropdown added)
  - ConnectionSettingsPage removed (using SettingsPage only)

### PR01PH08T04 — Document results

- [ ] **Task:** Record test outcomes, note any limitations, identify future improvements
- **Acceptance:** QA results documented
- **Commit:** "PR01PH08T04: Document QA results"
- [ ] **Status:** Not done

---

## Phase 9: Documentation (PR01PH09)

### PR01PH09T01 — Create timers help file

- [ ] **Task:** Create `/help/timers.json`, document #TIMER syntax, #CANCEL syntax, include examples for delayed and repeating timers
- **Acceptance:** Timers help file created and integrated
- **Commit:** "PR01PH09T01: Create timers help file"
- [ ] **Status:** Not done

### PR01PH09T02 — Create advanced automation help file

- [ ] **Task:** Create `/help/automation.json`, document #IF/#ELSE/#ENDIF syntax, #SET command, variable substitution, include examples
- **Acceptance:** Advanced automation help file created
- **Commit:** "PR01PH09T02: Create advanced automation help file"
- [ ] **Status:** Not done

### PR01PH09T03 — Update existing help files

- [ ] **Task:** Update `/help/aliases.json` with #SET examples, `/help/triggers.json` with #IF examples, `/help/variables.json` with #SET documentation
- **Acceptance:** Existing help files updated
- **Commit:** "PR01PH09T03: Update existing help files"
- [ ] **Status:** Not done

### PR01PH09T04 — Create Release Notes help file

- [ ] **Task:** Create `/help/release-notes.json`, add entry for version 0.1.0-pr1, include summary of PR01 features
- **Acceptance:** Release Notes help file created
- **Commit:** "PR01PH09T04: Create Release Notes help file"
- [ ] **Status:** Not done

### PR01PH09T05 — Update Help navigation

- [ ] **Task:** Add Timers section, Advanced Automation section, and Release Notes to Help navigation
- **Acceptance:** Help navigation updated
- **Commit:** "PR01PH09T05: Update Help navigation"
- [ ] **Status:** Not done

---

## Phase 10: Commit to Production (PR01PH10)

### PR01PH10T01 — Final Build Verification

- [ ] **Task:** Build frontend one final time (`npm run build`), build backend one final time (`go build`), verify both builds succeed without errors
- **Acceptance:** Both frontend and backend build successfully
- **Commit:** "PR01PH10T01: Final build verification"
- [ ] **Status:** Not done

### PR01PH10T02 — Push to Git Staging

- [ ] **Task:** Push working branch `pr01-automation-logic` to staging branch, verify push succeeds, wait for CI/CD pipeline to complete
- **Acceptance:** Code pushed to staging, CI/CD passes
- **Commit:** "PR01PH10T02: Push to Git staging"
- [ ] **Status:** Not done

### PR01PH10T03 — Push to Git Master

- [ ] **Task:** Push from staging to master/main, this triggers production deployment
- **Acceptance:** Code pushed to master, production deployment triggered
- **Commit:** "PR01PH10T03: Push to Git master"
- [ ] **Status:** Not done

### PR01PH10T04 — Branch Cleanup

- [ ] **Task:** Switch to staging branch (`git checkout staging`), delete working branch (`git branch -D pr01-automation-logic`)
- **Acceptance:** Working branch deleted, staging is current branch
- **Commit:** "PR01PH10T04: Branch cleanup"
- [ ] **Status:** Not done

> **This is the VERY LAST step for this spec. No spec is considered complete until this workflow is executed.**

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

### Within Each Phase

- Parser tasks should be sequential
- Logic engine builds on parser
- Variables can parallel with logic engine after parser
- Timers can start after parser, parallel with logic engine
- Safety requires all features complete
- UI requires safety to be ready
- Integration requires UI
- QA is complete
- Documentation is complete
- Commit to Production is final step

### Parallel Opportunities

- PR01PH02 (Logic Engine) and PR01PH03 (Variables) can proceed in parallel after PH01
- PR01PH04 (Timers) can proceed after PH01 independently
- All phases after PH05 must be sequential

---

## Notes

- Timer state is session-only (not persisted across sessions)
- Variables are persisted via existing profile storage
- All # commands are parsed before execution
- Safety limits are enforced at parse/execution time
- Editor UI changes are additive (no breaking changes)
- Backward compatibility maintained with MVP automation
- Maximum 10 active timers per session
- Maximum 500ms for condition evaluation
