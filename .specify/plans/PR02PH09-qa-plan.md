# PR02PH09 — QA and Regression Plan

> **Plan Type:** QA Execution Plan  
> **Phase:** PR02PH09 — QA and Regression  
> **Status:** Draft

---

## Overview

PR02PH09 is the Quality Assurance phase for the Internal Command Module (ICM) implementation. This plan covers both automated test verification and browser-based manual testing to ensure the ICM meets all acceptance criteria defined in PR02 specification.

---

## QA Task Summary

| Task | Description | Type |
|------|-------------|------|
| PR02PH09T01 | Deterministic Execution QA | Automated + Manual |
| PR02PH09T02 | Ingress Route QA | Automated + Manual |
| PR02PH09T03 | Safety Protection QA | Automated + Manual |
| PR02PH09T04 | Literal Behavior QA | Automated + Manual |
| PR02PH09T04B | % Namespace QA | Automated + Manual |
| PR02PH09T05 | #LOG Governance QA | Automated + Manual |
| PR02PH09T06 | Regression Testing | Automated + Manual |

---

## Part 1: Automated Test Verification

### Prerequisites

1. **Run Backend Tests**
   ```bash
   cd internal/icm
   go test -v -cover ./...
   ```

2. **Expected Test Coverage Areas**
   - Recognizer (operator identification)
   - Validator (syntax validation)
   - Normalizer (canonical form conversion)
   - Registry (aliases, user variables, system variables)
   - Engine (command execution)
   - Classifier (internal vs pass-through)
   - EscapeHandler (escape sequences)
   - SafetyChecker (rate limits, timeouts)

### T01: Deterministic Execution QA (Automated)

**Existing Tests to Verify:**
- `TestEngine_Process` - Verifies execution order
- `TestEngine_ExecuteStructuredCommand` - Verifies directive processing

**Additional Test Cases Required:**

| Input | Expected Output | Notes |
|-------|-----------------|-------|
| `#echo hello` then `look` | Commands execute in order | Verify queue order |
| `#set var value` then `$var` | Variable set then resolved | Verify variable resolution order |
| `@alias` where alias contains `#if` | Alias expansion then directive | Verify alias→directive order |

**Verification Command:**
```bash
go test -v -run "TestEngine_Process|TestEngine_ExecuteStructuredCommand" ./internal/icm/
```

---

### T02: Ingress Route QA (Automated)

**Existing Tests to Verify:**
- All ICM component tests

**Key Integration Points:**
1. **Typed Input → ICM** - [`PlayScreen.tsx:378-384`](frontend/src/pages/PlayScreen.tsx:378)
2. **Keybindings → ICM** - Routes through `submitCommand()` 
3. **History Replay → ICM** - Routes through `submitCommand()`
4. **Settings Editors → ICM** - [`SettingsPage.tsx`](frontend/src/pages/SettingsPage.tsx)

**Verification Steps:**
1. Review code integration points match spec
2. Run full test suite to verify no regressions

---

### T03: Safety Protection QA (Automated)

**Existing Tests to Verify:**
- `TestDefaultSafetyChecker_CheckRateLimit`
- `TestDefaultSafetyChecker_CheckTimeout`

**Test Cases:**

| Safety Feature | Test Scenario | Expected Behavior |
|----------------|--------------|-------------------|
| Rate Limiting | Execute 100+ commands rapidly | Commands throttled after limit |
| Timeout | Long-running command | Timeout enforced |
| Circuit Breaker | Infinite loop detection | Circuit breaker triggers |

**Verification Command:**
```bash
go test -v -run "TestDefaultSafetyChecker" ./internal/icm/
```

**Manual Verification Required:**
- Trigger circuit breaker through UI
- Verify rate limiting message displayed

---

### T04: Literal Behavior QA (Automated)

**Existing Tests to Verify:**
- `TestEscapeHandler_ResolveEscapes`
- `TestEscapeHandler_HasEscapeSequences`
- `TestClassifier_Classify`

**Test Cases:**

| Input | Expected Output | Notes |
|-------|-----------------|-------|
| `\#echo` | `#echo` | Escape preserved |
| `\@alias` | `@alias` | Escape preserved |
| `\$var` | `$var` | Escape preserved |
| `\%sys` | `%sys` | Escape preserved |
| `${undefined}` | `${undefined}` | Undefined variable preserved |
| `look` | `look` (pass-through) | Non-internal command passes |
| `&unknown` | `&unknown` (pass-through) | Unknown operator passes |

**Verification Command:**
```bash
go test -v -run "TestEscapeHandler|TestClassifier" ./internal/icm/
```

---

### T04B: % Namespace QA (Automated)

**Existing Tests to Verify:**
- `TestRecognizer_RecognizeSystemVariable`

**Test Cases:**

| Input | Expected Classification | Notes |
|-------|----------------------|-------|
| `%1` | Numeric system variable | Session-scoped |
| `%42` | Numeric system variable | Session-scoped |
| `%TIME` | Named system variable | Read-only |
| `%CHARACTER` | Named system variable | Read-only |
| `%` | Malformed → passes through | Backward compatible |
| `%999` | Numeric system variable | High number |

**Verification Command:**
```bash
go test -v -run "TestRecognizer_RecognizeSystemVariable" ./internal/icm/
```

---

### T05: #LOG Governance QA (Automated)

**Existing Tests to Verify:**
- `TestEngine_LogCommand`
- `TestEngine_LogCommandPreviewDenied`

**Test Cases:**

| Context | Input | Expected Behavior |
|---------|-------|-------------------|
| Preview | `#log test` | Command validated but not executed |
| Submission | `#log test` | Command executes (if authorized) |
| Automation | `#log test` | Command executes (if authorized) |
| Operational | `#log test` | Command executes with full metadata |

**Verification Command:**
```bash
go test -v -run "TestEngine_LogCommand" ./internal/icm/
```

**Manual Verification Required:**
- Check log output format (JSON with timestamp, level, correlation ID)
- Verify routing to backend logging service

---

### T06: Regression Testing (Automated)

**Full Test Suite:**
```bash
go test -v ./internal/icm/
```

**Expected Results:**
- All existing tests pass
- No new test failures
- Code coverage maintained above 70%

---

## Part 2: Browser-Based Manual QA

### Prerequisites

1. **Start Backend Server**
   ```bash
   cd cmd/server
   go run main.go
   ```

2. **Start Frontend Dev Server**
   ```bash
   cd frontend
   npm run dev
   ```

3. **Open Browser** to http://localhost:5173 (or configured port)

### Browser QA Environment

| Element | Value |
|---------|-------|
| Backend URL | http://localhost:8080 |
| Frontend URL | http://localhost:5173 |
| Test Account | Use existing test account |

---

### T01: Deterministic Execution QA (Browser)

**Test Scenario 1: Typed Command Execution Order**

1. Connect to a test MUD server
2. Enter: `#set testvar hello`
3. Enter: `$testvar`
4. **Expected:** Variable set, then resolved to "hello"

**Test Scenario 2: Alias Expansion Order**

1. Create alias `@test` with expansion `#echo expanded`
2. Enter: `@test`
3. **Expected:** Alias expanded, directive executed

**Test Scenario 3: Trigger Execution Order**

1. Create trigger: "Welcome" → `#echo trigger fired`
2. Receive text "Welcome" in terminal
3. **Expected:** Trigger action executes after message display

---

### T02: Ingress Route QA (Browser)

**Test Scenario 1: Typed Input**

1. Type `#echo typed test`
2. Press Enter
3. **Expected:** Command processed through ICM, output displayed

**Test Scenario 2: Keybinding**

1. Configure a keybinding (e.g., Ctrl+1 → `#echo keybinding`)
2. Press the keybinding
3. **Expected:** Command processed through ICM

**Test Scenario 3: History Replay**

1. Enter several commands
2. Use Up arrow to navigate history
3. Select a previous command
4. Press Enter
5. **Expected:** History command processed through ICM

---

### T03: Safety Protection QA (Browser)

**Test Scenario 1: Rate Limiting**

1. Execute 100+ commands rapidly (e.g., `#echo test` repeated)
2. **Expected:** After rate limit, see throttling message or delay

**Test Scenario 2: Circuit Breaker**

1. Create trigger with recursive expansion
2. Trigger the loop
3. **Expected:** Circuit breaker pauses execution, notification displayed

---

### T04: Literal Behavior QA (Browser)

**Test Scenario 1: Escape Sequences**

1. Enter: `\#echo hello`
2. **Expected:** Outputs "#echo hello" (literal, not processed)

**Test Scenario 2: Undefined Variables**

1. Enter: `$undefinedvar`
2. **Expected:** Outputs "$undefinedvar" (preserved, no error)

**Test Scenario 3: Unknown Operators**

1. Enter: `&somecommand`
2. **Expected:** Passes through to MUD server unchanged

---

### T04B: % Namespace QA (Browser)

**Test Scenario 1: Numeric Session Variables**

1. Create alias: `@argtest %1`
2. Enter: `@argtest hello`
3. **Expected:** `%1` resolves to "hello"

**Test Scenario 2: Named System Variables**

1. Enter: `%TIME`
2. **Expected:** Displays current time value

**Test Scenario 3: Malformed % Reference**

1. Enter: `%`
2. **Expected:** Passes through literally (backward compatible)

---

### T05: #LOG Governance QA (Browser)

**Test Scenario 1: Preview Context**

1. Open command preview (if available)
2. Enter: `#log test message`
3. **Expected:** Validation occurs, but no log entry created

**Test Scenario 2: Operational Context**

1. In operational mode, enter: `#log info Test message`
2. Check backend logs
3. **Expected:** JSON log entry with timestamp, level, correlation ID

---

### T06: Regression Testing (Browser)

**Test Scenario 1: Ordinary Commands**

1. Enter: `look`, `who`, `inventory`, `say hello`
2. **Expected:** All commands pass through to MUD server

**Test Scenario 2: Existing Aliases**

1. Use existing alias definitions
2. Execute aliases
3. **Expected:** Aliases work as before migration

**Test Scenario 3: Existing Triggers**

1. Use existing trigger definitions
2. Fire triggers
3. **Expected:** Triggers work as before migration

**Test Scenario 4: Settings Editors**

1. Open Settings → Aliases
2. Create/edit/delete alias
3. **Expected:** Validation works, aliases save correctly

4. Open Settings → Triggers
5. Create/edit/delete trigger
6. **Expected:** Validation works, triggers save correctly

---

## Acceptance Criteria Summary

| Task | Criteria | Verification Method |
|------|----------|-------------------|
| T01 | Execution order: input→alias→variable→logic→queue→trigger verified | Run tests + manual |
| T02 | Typed input, keybindings, history replay all route through ICM | Code review + manual |
| T03 | Recursion limits, rate limits, circuit breaker fire correctly | Run tests + manual |
| T04 | Escape sequences, undefined variables, unknown operators work | Run tests + manual |
| T04B | % namespace resolution: numeric vs named verified | Run tests + manual |
| T05 | #LOG governance: contexts, output format verified | Run tests + manual |
| T06 | No regressions in existing command paths | Run tests + manual |

---

## Test Data

### Test Aliases
```
@test → #echo alias expansion
@args → %1 %2 %3
```

### Test Triggers
```
Pattern: "Welcome" → Action: #echo Welcome trigger fired
Pattern: "You die" → Action: #log error Player died
```

### Test Variables
```
#set testvar testvalue
#set counter 0
```

---

## Sign-Off

| Role | Name | Date | Status |
|------|------|------|--------|
| QA Engineer | | | Pending |
| Tech Lead | | | Pending |
| Product Owner | | | Pending |

---

## Notes

- QA may overlap with documentation refinements
- Browser-based tests require running server and frontend
- Use test MUD server (not production) for manual testing
- Document any discrepancies found during QA
