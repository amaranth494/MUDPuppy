# PR02PH09 — QA and Regression Plan

> **Plan Type:** QA Execution Plan  
> **Phase:** PR02PH09 — QA and Regression  
> **Status:** Draft

---

## Overview

This QA plan focuses on **individual command verification** first, then complex scenarios. Before testing execution order, safety systems, or regressions, we must verify each ICM command works correctly in the CLI/UX.

---

## ICM Command Inventory

### Structured Directives (#)
| Command | Purpose | Handler |
|---------|---------|---------|
| `#ECHO` | Output message | EchoHandler |
| `#LOG` | Structured logging | LogHandler |
| `#HELP` | Help display | HelpHandler |
| `#SET` | Set user variable | SetHandler |
| `#IF` | Conditional | IfHandler |
| `#ELSE` | Else branch | ElseHandler |
| `#ENDIF` | End conditional | EndIfHandler |
| `#TIMER` | Create/update timer | TimerHandler |
| `#START` | Start timer | StartTimerHandler |
| `#STOP` | Stop timer | StopTimerHandler |
| `#CHECK` | Check timer status | CheckTimerHandler |
| `#CANCEL` | Delete timer | CancelTimerHandler |

### Alias References (@)
- `@aliasname` - Invoke alias by name
- `@aliasname arg1 arg2` - Invoke alias with arguments

### User Variables ($)
- `$varname` - Reference user variable
- `${varname}` - Reference user variable (canonical form)

### System Variables (%)
- `%1`, `%2`, etc. - Numeric session variables (positional)
- `%TIME` - Current time
- `%DATE` - Current date
- `%CHARACTER` - Character name
- `%SERVER` - Server name
- `%SESSIONID` - Session ID
- `%HOST` - Host
- `%PORT` - Port

---

## Part 1: Individual Command Verification

### Phase 1A: Verify Structured Directives

Connect to a MUD server and test each directive individually in the terminal input:

#### Test: #ECHO
```
#echo Hello World
```
**Expected:** "Hello World" appears in terminal output

#### Test: #SET and $ Retrieval
```
#set myvar testvalue
$myvar
```
**Expected:** Variable set, then "$myvar" resolves to "testvalue"

#### Test: #IF / #ELSE / #ENDIF
```
#if 1 == 1
#echo true branch
#else
#echo false branch
#endif
```
**Expected:** "true branch" output

#### Test: #TIMER (Create Timer)
```
#timer testtimer 60 #echo timer fired
```
**Expected:** Timer created, no error

#### Test: #START (Start Timer)
```
#start testtimer
```
**Expected:** Timer started

#### Test: #CHECK (Check Timer Status)
```
#check testtimer
```
**Expected:** Timer status displayed (running/stopped)

#### Test: #STOP (Stop Timer)
```
#stop testtimer
```
**Expected:** Timer stopped

#### Test: #CANCEL (Delete Timer)
```
#cancel testtimer
```
**Expected:** Timer deleted

#### Test: #LOG (Operational)
```
#log info This is a test
```
**Expected:** Log entry created (context-dependent)

---

### Phase 1B: Verify Alias References

#### Test: Simple Alias Invocation
1. Create alias via Settings → Aliases: `@test → #echo alias works`
2. In terminal: `@test`
**Expected:** "alias works" output

#### Test: Alias with Arguments
1. Create alias: `@argtest → say %1`
2. In terminal: `@argtest hello`
**Expected:** "say hello" sent to MUD

---

### Phase 1C: Verify User Variables

#### Test: $varname (Short Form)
```
#set shortvar shortvalue
$shortvar
```
**Expected:** Variable value "shortvalue" displayed

#### Test: ${varname} (Canonical Form)
```
#set longvar longvalue
${longvar}
```
**Expected:** Variable value "longvalue" displayed

#### Test: Undefined Variable (Should Pass Through)
```
$undefinedvar
```
**Expected:** "$undefinedvar" appears literally (no error)

---

### Phase 1D: Verify System Variables

#### Test: Numeric Session Variables (%1, %2, etc.)
1. Create alias: `@numtest → echo %1 %2 %2`
2. Run: `@numtest hello world`
**Expected:** "hello world world" output

#### Test: %TIME
```
%TIME
```
**Expected:** Current time displayed (e.g., "14:30:00")

#### Test: %DATE
```
%DATE
```
**Expected:** Current date displayed (e.g., "2026-03-21")

#### Test: %CHARACTER (After Connection)
```
%CHARACTER
```
**Expected:** Character name displayed (if connected)

#### Test: %SERVER (After Connection)
```
%SERVER
```
**Expected:** Server name displayed (if connected)

---

### Phase 1E: Verify Escape Sequences

#### Test: \# (Literal Hash)
```
\#echo hello
```
**Expected:** "#echo hello" output (not processed as command)

#### Test: \@ (Literal At)
```
\@myalias
```
**Expected:** "@myalias" output (not processed as alias)

#### Test: \$ (Literal Dollar)
```
\$myvar
```
**Expected:** "$myvar" output (not processed as variable)

#### Test: \% (Literal Percent)
```
\%TIME
```
**Expected:** "%TIME" output (not processed as system variable)

---

### Phase 1F: Verify Pass-Through Behavior

#### Test: Ordinary MUD Commands
```
look
who
inventory
say hello
```
**Expected:** All commands pass through to MUD server unchanged

#### Test: Unknown Operator (Not Reserved)
```
&somecommand
```
**Expected:** Passes through to MUD server unchanged

---

## Part 2: Complex Scenario Testing

After all individual commands pass, proceed to complex scenarios:

### Phase 2A: Execution Order
- Verify input → alias → variable → logic → queue → trigger order

### Phase 2B: Ingress Routes
- Typed input through ICM
- Keybindings through ICM
- History replay through ICM

### Phase 2C: Safety Protections
- Rate limiting
- Circuit breaker
- Timeout handling

### Phase 2D: % Namespace Resolution
- Numeric %<n> precedence over named
- Malformed % handling

### Phase 2E: #LOG Governance
- Preview context denies
- Automation/Operational contexts allow

### Phase 2F: Regression Testing
- Existing aliases still work
- Existing triggers still work
- Settings editors work correctly

---

## Acceptance Criteria

| Phase | Criteria | Pass Condition |
|-------|----------|----------------|
| 1A | All 11 structured directives work | Each returns expected output |
| 1B | Alias references work | Aliases expand and execute |
| 1C | User variables work | Set/get/undefined cases |
| 1D | System variables work | Numeric and named cases |
| 1E | Escape sequences work | Each escape resolves correctly |
| 1F | Pass-through works | Non-ICM commands pass through |
| 2A | Execution order correct | Commands process in correct order |
| 2B | All ingress routes verified | ICM used for all entry points |
| 2C | Safety protections fire | Limits enforced correctly |
| 2D | % namespace verified | Precedence rules work |
| 2E | #LOG governance verified | Context restrictions work |
| 2F | No regressions | Existing features work |

---

## Sign-Off

| Role | Name | Date | Status |
|------|------|------|--------|
| QA Engineer | | | Pending |
| Tech Lead | | | Pending |
| Product Owner | | | Pending |

---

## Notes

- **Phase 1 must pass completely before Phase 2**
- Use a test MUD server (not production)
- Document any discrepancies found
- Test with both connected and disconnected states where applicable
