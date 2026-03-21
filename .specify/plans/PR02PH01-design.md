# PR02PH01 — ICM Design Document

> **Phase:** PR02PH01 (ICM Design)  
> **Task:** PR02PH01T01-T10  
> **Version:** 0.2.0-pr2  
> **Status:** Draft

---

## 1. Command Lifecycle (PR02PH01T01)

The ICM defines a single authoritative command lifecycle that all internal commands must follow:

```
ingress → tokenize → validate → normalize → resolve → dispatch
```

### 1.1 Phase Definitions

| Phase | Description | Responsibility |
|-------|-------------|----------------|
| **Ingress** | Raw input enters the ICM from any ingress route (typed input, keybinding, history, automation, operational) | Input capture and initial routing |
| **Tokenize** | Input is broken into tokens, operator families are identified | Recognizer component |
| **Validate** | Syntax is checked, arguments are validated, grammar rules enforced | Validator component |
| **Normalize** | Command is converted to canonical form | Normalizer component |
| **Resolve** | Variable references, aliases, expressions are resolved | Resolver component |
| **Dispatch** | Command is routed to appropriate handler with safety checks | Dispatcher component |

### 1.2 Constitutional Execution Order

The ICM lifecycle operates within the broader constitutional execution order:

```
user input → alias expansion → variable substitution → logic evaluation → queue dispatch → trigger evaluation
```

- ICM handles recognition and normalization phases
- The constitutional order governs when each phase executes in the broader automation pipeline
- ICM preserves this order by integrating with existing automation engine

### 1.3 Ingress Routes

Commands may enter ICM from any of these sources:

1. **Typed command input** - User types in play screen
2. **Keybinding dispatch** - Keyboard shortcut triggers command
3. **History replay** - Previous command recalled and re-executed
4. **Alias expansion** - Alias triggers nested command
5. **Trigger action** - Trigger fires command
6. **Timer action** - Timer executes command
7. **Internal operational directive** - Backend service tooling
8. **Future service tooling** - API-driven commands

---

## 2. Frontend/Backend Contract (PR02PH01T02)

### 2.1 API Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/v1/icm/validate` | POST | Validate command syntax without execution |
| `/api/v1/icm/normalize` | POST | Normalize command to canonical form |
| `/api/v1/icm/execute` | POST | Execute command with full processing |
| `/api/v1/icm/preview` | POST | Preview command effects without execution |

### 2.2 Request Schema

```typescript
interface ICMRequest {
  // Input
  raw: string;
  
  // Execution context
  context: ExecutionContext;
  
  // Session context
  sessionId: string;
  userId?: string;
  
  // Optional: pre-resolved variables
  variables?: Record<string, string>;
  
  // Options
  options?: {
    skipSafety?: boolean;
    dryRun?: boolean;
    trace?: boolean;
  };
}
```

### 2.3 Response Schema

```typescript
interface ICMResponse {
  // Command classification
  isInternal: boolean;
  operator?: OperatorFamily;
  command?: string;
  
  // Processed form
  normalized?: string;
  resolved?: string;
  
  // Result
  result?: CommandResult;
  error?: ICMError;
  
  // Metadata
  processingTime: number;
  trace?: ProcessingTrace;
}
```

### 2.4 Execution Context Enum

```typescript
type ExecutionContext = 
  | 'preview'      // Local validation, help display, syntax check (frontend-only)
  | 'submission'   // User command entry, keybinding, history (Frontend → Backend)
  | 'automation'   // Alias expansion, trigger action, timer fire (Backend-authoritative)
  | 'operational'; // Service tooling, admin commands (Backend-only)
```

### 2.5 Shared Schema Summary

| Field | Type | Description |
|-------|------|-------------|
| `raw` | string | Original raw input |
| `normalized` | string | Canonical command form |
| `operator` | OperatorFamily | Detected operator family (#, @, %, $) |
| `command` | string | Command identifier |
| `args` | string[] | Parsed arguments |
| `context` | ExecutionContext | Execution context |
| `result` | CommandResult | Execution result or error |

---

## 3. Operator Families (PR02PH01T03)

### 3.1 Family Registry

| Family | Operators | Description |
|--------|-----------|-------------|
| **Structured Directives** | `#` | Automation control: `#IF`, `#ELSE`, `#ENDIF`, `#SET`, `#TIMER`, `#START`, `#STOP`, `#CHECK`, `#CANCEL`, `#LOG` |
| **User Variables** | `$` | Profile variables: `${name}` or `$name` syntax |
| **System Variables** | `%` | Session-scoped: numeric (`%1`, `%2`) and named (`%TIME`, `%CHARACTER`) |
| **Alias References** | `@` | Alias invocation: `@aliasname` |

### 3.2 Operator Detection Rules

```
Operator Character Detection Order:
1. # - Structured directive (highest priority)
2. @ - Alias reference
3. $ - User variable
4. % - System variable

Detection is greedy - once an operator family is identified,
remaining processing follows that family's grammar.
```

### 3.3 Registry Entry Format

```typescript
interface OperatorFamilyRegistry {
  [family: string]: {
    prefix: string;           // e.g., "#", "@", "$", "%"
    name: string;             // e.g., "structured", "alias", "user_variable", "system_variable"
    commands: CommandRegistry;
    variablePattern?: RegExp;
    escapeSequence: string;
  };
}
```

---

## 4. % Namespace Resolution Precedence (PR02PH01T03B)

### 4.1 Resolution Rules

| Rule | Specification |
|------|---------------|
| **Recognition Order** | Numeric patterns (`%` followed by digits) are always resolved as numeric system variables. Named patterns are resolved against the registered system variable registry. |
| **Numeric Form** | `%<digits>` is always session-scoped and cleared on disconnect. |
| **Named Form** | `%<identifier>` is always a system variable and read-only. |
| **Collision Policy** | Named system variables take precedence over numeric forms when the name matches a registered system variable. Unknown names pass through literally. |
| **Error Behavior** | Malformed references (e.g., `%` alone with no digits or identifier) pass through literally to preserve backward compatibility. |

### 4.2 Examples

| Input | Resolution | Type |
|-------|------------|------|
| `%1` | Session variable 1 | Numeric session |
| `%42` | Session variable 42 | Numeric session |
| `%TIME` | System time | Named system |
| `%CHARACTER` | Current character name | Named system |
| `%UNKNOWN` | `%UNKNOWN` (literal pass-through) | Unknown |
| `%` | `%` (literal pass-through) | Malformed |

### 4.3 Session vs System Variable Scope

**Session Variables (Numeric %n):**
- Scope: Current session only
- Lifecycle: Created on connect, cleared on disconnect
- Persistence: None
- Population: Alias arguments, trigger captures

**System Variables (Named %NAME):**
- Scope: Read-only client data
- Lifecycle: Updated by client state changes
- Persistence: Memory only (not persisted)
- Examples: `%TIME`, `%CHARACTER`, `%SERVER`, `%SESSIONID`

---

## 5. Execution Contexts (PR02PH01T04)

### 5.1 Context Definitions

| Context | Description | Authority | Allowed Operations |
|---------|-------------|-----------|-------------------|
| **Preview** | Local validation, help display, syntax check | Frontend-only | Validation, normalization, syntax checking |
| **Submission** | User command entry, keybinding, history replay | Frontend → Backend | Full execution, session state changes |
| **Automation** | Alias expansion, trigger action, timer fire | Backend-authoritative | Full execution with automation context |
| **Operational** | Service tooling, admin commands | Backend-only | Administrative operations, diagnostics |

### 5.2 Authority Levels

```typescript
const AuthorityLevels = {
  PREVIEW: {
    canExecute: false,
    canModifyState: false,
    requiresAuth: false,
  },
  SUBMISSION: {
    canExecute: true,
    canModifyState: true,
    requiresAuth: true,
  },
  AUTOMATION: {
    canExecute: true,
    canModifyState: true,
    requiresAuth: true,
    isAutomation: true,
  },
  OPERATIONAL: {
    canExecute: true,
    canModifyState: true,
    requiresAuth: true,
    requiresRole: 'admin',
  },
};
```

### 5.3 Context Transitions

- Preview → Submission: After validation passes, user confirms
- Automation: Triggered by system events, no user confirmation
- Operational: Only accessible via authenticated API

---

## 6. Normalization Format (PR02PH01T05)

### 6.1 Canonical Command Form

All internal commands are normalized to a canonical form:

```
<OPERATOR><COMMAND> <ARG1> <ARG2> ... <ARGN>
```

### 6.2 Normalization Rules

| Input | Normalized Output |
|-------|-------------------|
| `#if $foo == bar` | `#IF $FOO == BAR` |
| `#  set  myvar  =  value` | `#SET myvar = value` |
| `@myalias arg1 arg2` | `@myalias arg1 arg2` |
| `${myvar}` | `$myvar` |
| `%session42` | `%42` (numeric form) |

### 6.3 Output Schema

```typescript
interface NormalizedCommand {
  // Canonical form
  canonical: string;
  
  // Decomposed parts
  operator: OperatorFamily;
  command: string;
  args: string[];
  
  // Metadata
  originalInput: string;
  transformations: string[];
  
  // Classification
  isInternal: boolean;
  requiresExecution: boolean;
}
```

### 6.4 Transformation Pipeline

1. **Trim** - Remove leading/trailing whitespace
2. **Normalize case** - Convert commands to uppercase (# family)
3. **Collapse whitespace** - Multiple spaces → single space
4. **Expand shorthand** - `$name` → `${name}`
5. **Resolve escapes** - `\#` → `#` (for storage, not display)

---

## 7. Error Model (PR02PH01T06)

### 7.1 Error Code Hierarchy

```typescript
enum ICMErrorCode {
  // Recognition errors (1xxx)
  E1000_UNRECOGNIZED_OPERATOR = 'E1000',
  E1001_AMBIGUOUS_INPUT = 'E1001',
  
  // Validation errors (2xxx)
  E2000_SYNTAX_ERROR = 'E2000',
  E2001_MISSING_ARGUMENT = 'E2001',
  E2002_INVALID_ARGUMENT = 'E2002',
  E2003_ARGUMENT_OUT_OF_RANGE = 'E2003',
  E2004_UNKNOWN_COMMAND = 'E2004',
  
  // Resolution errors (3xxx)
  E3000_UNDEFINED_VARIABLE = 'E3000',
  E3001_UNDEFINED_ALIAS = 'E3001',
  E3002_CIRCULAR_REFERENCE = 'E3002',
  E3003_RESOLUTION_TIMEOUT = 'E3003',
  
  // Execution errors (4xxx)
  E4000_EXECUTION_FAILED = 'E4000',
  E4001_PERMISSION_DENIED = 'E4001',
  E4002_RATE_LIMITED = 'E4002',
  E4003_CIRCUIT_BREAKER = 'E4003',
  
  // System errors (5xxx)
  E5000_INTERNAL_ERROR = 'E5000',
  E5001_SERVICE_UNAVAILABLE = 'E5001',
}
```

### 7.2 Error Response Schema

```typescript
interface ICMError {
  code: ICMErrorCode;
  message: string;
  userMessage: string;
  details?: Record<string, any>;
  context?: {
    phase: string;
    position?: number;
    input?: string;
  };
}
```

### 7.3 User-Facing Messages

| Code | User Message |
|------|--------------|
| E1000 | "Unknown operator in command" |
| E2000 | "Syntax error in command" |
| E2001 | "Missing required argument" |
| E2004 | "Unknown command: {command}" |
| E3000 | "Variable not found: {variable}" |
| E3001 | "Alias not found: {alias}" |
| E4001 | "You don't have permission to use this command" |
| E4002 | "Too many commands. Please wait before trying again." |
| E4003 | "Command execution paused due to repeated patterns" |

---

## 8. Safety Hooks (PR02PH01T07)

### 8.1 Required Safety Hooks

The ICM integrates with existing constitutional safeguards:

| Safety Feature | Hook Point | Configuration |
|----------------|-----------|---------------|
| **Recursion limits** | Resolve phase | `maxAliasDepth: 3` |
| **Command dispatch limits** | Dispatch phase | `maxCommandsPerDispatch: 100` |
| **Trigger rate limiting** | Dispatch phase | `minTriggerInterval: 100ms` |
| **Automation circuit breaker** | Dispatch phase | `maxLoopCount: 5` |
| **Evaluation timeout** | All phases | `maxExecutionTime: 5000ms` |
| **Queue backpressure** | Queue phase | `maxQueueDepth: 100` |

### 8.2 Safety Interface

```typescript
interface SafetyHooks {
  // Before execution
  checkRecursionDepth(context: ExecutionContext): boolean;
  checkCircuitBreaker(sessionId: string): boolean;
  checkRateLimit(sessionId: string, command: string): boolean;
  
  // During execution
  recordExecution(sessionId: string, command: string): void;
  checkTimeout(startTime: number): boolean;
  
  // Queue management
  checkQueueDepth(sessionId: string): boolean;
  getQueueBackpressureLevel(sessionId: string): 'normal' | 'warning' | 'critical';
}
```

### 8.3 Safety Enforcement Points

| Phase | Safety Check |
|-------|--------------|
| Validate | Syntax validation, argument limits |
| Normalize | Variable resolution limits |
| Resolve | Expression complexity limits |
| Dispatch | Circuit breaker, rate limits, quotas |

### 8.4 Default Limits

```typescript
const SafetyLimits = {
  maxAliasDepth: 3,
  maxCommandsPerDispatch: 100,
  minTriggerInterval: 100,      // milliseconds
  maxLoopCount: 5,
  maxExecutionTime: 5000,       // milliseconds
  maxQueueDepth: 100,
  maxVariableResolutionDepth: 10,
  maxExpressionComplexity: 50,
};
```

---

## 9. Escape/Literal Grammar (PR02PH01T08)

### 9.1 Escape Sequences

The ICM handles escape sequences to allow literal representation of operator characters:

| Escape | Literal Output | Example |
|--------|---------------|---------|
| `\#` | `#` | `\#echo` outputs `#echo` |
| `\@` | `@` | `\@mention` outputs `@mention` |
| `\$` | `$` | `\$var` outputs `$var` |
| `\%` | `%` | `\%time` outputs `%time` |
| `\\` | `\` | `\\n` outputs `\n` |

### 9.2 Escape Processing Rules

1. **Recognition**: Escape sequences are processed during the normalize phase
2. **Storage**: Escapes are preserved in normalized form for round-trip accuracy
3. **Display**: Escapes are shown as literals in UI
4. **Execution**: Escapes are resolved before dispatch

### 9.3 Literal Pass-Through Rules

| Condition | Behavior | Example |
|-----------|----------|---------|
| **Undefined variables** | Pass through literally | `${undefined}` → `${undefined}` |
| **Unknown operators** | Pass through to MUD if not reserved family | `&somecommand` → `&somecommand` |
| **Non-operator input** | Pass through to MUD unchanged | `look` → `look` |

### 9.4 Unknown Operator Handling

**Rule 1 — Unknown operator family (not reserved):**
- If the operator character is not a known reserved family (#, $, %, @), pass through to MUD server unchanged
- This preserves compatibility with non-standard prefixes

**Rule 2 — Unknown command inside reserved family:**
- If the operator is a reserved family (#, $, %, @) but the command is not recognized, return a validation error
- Do NOT pass through to MUD server — this would create ambiguity
- Log for debugging if enabled

### 9.5 Examples

| Input | Result | Reason |
|-------|--------|--------|
| `&somecommand` | Pass through | Unknown family `&` |
| `#WHATEVER` | Error | Reserved `#` but unknown command |
| `@undefined` | Error | Reserved `@` but unknown alias |
| `${undefined}` | Pass through | Unknown variable - pass through |
| `#echo hello` | Execute | Known directive |

---

## 10. Pass-Through Behavior (PR02PH01T09)

### 10.1 Classification Algorithm

```
1. If input starts with reserved operator (#, @, $):
   a. If recognized command/alias/variable → Process as internal
   b. If unrecognized command/alias → Error (reserved family)
   c. If unknown variable → Pass through literally

2. If input starts with unknown operator:
   → Pass through to MUD

3. If input has no operator prefix:
   → Pass through to MUD
```

### 10.2 Pass-Through Conditions

| Input Type | Classification | Action |
|------------|---------------|--------|
| Plain MUD command | External | Pass to MUD server |
| Unknown operator family | External | Pass to MUD server |
| Undefined variable reference | Literal | Pass through as-is |
| Malformed reference | Literal | Pass through as-is |

### 10.3 Pass-Through Schema

```typescript
interface PassThroughResult {
  classifiedAs: 'internal' | 'external' | 'literal';
  operator?: OperatorFamily;
  command?: string;
  rawOutput: string;
  shouldExecute: boolean;
  shouldForward: boolean;
}
```

---

## 11. #LOG Governance (PR02PH01T10)

### 11.1 Command Specification

The `#LOG` directive is a developer-facing operational command with the following governance:

| Aspect | Specification |
|--------|--------------|
| **Visibility** | Developer/Operational only (not end-user facing in help) |
| **Execution Contexts** | Automation, Operational |
| **Permission Model** | Permission-gated (admin/developer role) or trusted backend operational context |
| **Output Format** | Structured JSON with timestamp, level, message, correlation ID |
| **Routing Destinations** | Backend logging service, observability platform |
| **Required Metadata** | session_id, user_id, timestamp, level, correlation_id |
| **Help Documentation** | Not surfaced in user-facing help |

### 11.2 Command Syntax

```
#LOG <level> <message>
#LOG DEBUG Something happened
#LOG INFO User action completed
#LOG WARN Potential issue detected
#LOG ERROR Error occurred
```

### 11.3 Permission Model

```typescript
interface LogPermission {
  requiredRole: 'admin' | 'developer';
  requiredContext: 'automation' | 'operational';
  bypassForInternal: boolean;  // Internal services can bypass
}
```

### 11.4 Output Format

```typescript
interface LogEntry {
  // Required metadata
  timestamp: string;           // ISO 8601
  level: 'DEBUG' | 'INFO' | 'WARN' | 'ERROR';
  message: string;
  correlationId: string;
  
  // Session context
  sessionId: string;
  userId?: string;
  
  // Source context
  source: 'icm' | 'automation' | 'trigger' | 'timer' | 'manual';
  
  // Optional
  metadata?: Record<string, any>;
}
```

### 11.5 Routing

- **DEBUG**: Debug log sink (if enabled)
- **INFO**: Info log sink
- **WARN**: Warning log sink + alerts
- **ERROR**: Error log sink + alerts + monitoring

---

## 12. Design Summary

### 12.1 Phase Completion Status

| Task | Design Element | Status |
|------|---------------|--------|
| PR02PH01T01 | Command Lifecycle | Defined |
| PR02PH01T02 | Frontend/Backend Contract | Defined |
| PR02PH01T03 | Operator Families | Defined |
| PR02PH01T03B | % Namespace Resolution | Defined |
| PR02PH01T04 | Execution Contexts | Defined |
| PR02PH01T05 | Normalization Format | Defined |
| PR02PH01T06 | Error Model | Defined |
| PR02PH01T07 | Safety Hooks | Defined |
| PR02PH01T08 | Escape/Literal Grammar | Defined |
| PR02PH01T09 | Pass-Through Behavior | Defined |
| PR02PH01T10 | #LOG Governance | Defined |

---

## Appendix A: Component Interfaces

### A.1 Recognizer Interface

```typescript
interface Recognizer {
  tokenize(input: string): Token[];
  identifyOperator(input: string): OperatorFamily | null;
  extractCommand(input: string, operator: OperatorFamily): CommandCandidate;
}
```

### A.2 Validator Interface

```typescript
interface Validator {
  validate(command: CommandCandidate): ValidationResult;
  checkSyntax(operator: OperatorFamily, command: string, args: string[]): boolean;
  getAllowedCommands(operator: OperatorFamily): string[];
}
```

### A.3 Normalizer Interface

```typescript
interface Normalizer {
  normalize(input: string, context: ExecutionContext): NormalizedCommand;
  toCanonical(form: CommandCandidate): string;
  transform(input: string): Transformation[];
}
```

### A.4 Resolver Interface

```typescript
interface Resolver {
  resolve(command: NormalizedCommand, context: ExecutionContext): ResolvedCommand;
  resolveVariable(name: string): string | null;
  resolveAlias(name: string): string | null;
  resolveSystem(name: string): string | null;
}
```

### A.5 Dispatcher Interface

```typescript
interface Dispatcher {
  dispatch(command: ResolvedCommand, context: ExecutionContext): Promise<DispatchResult>;
  route(command: ResolvedCommand): Handler;
  enforceSafety(command: ResolvedCommand): SafetyCheckResult;
}
```

---

## Appendix B: Type Definitions

### B.1 Complete Type Summary

```typescript
// Operator families
type OperatorFamily = 'structured' | 'alias' | 'user_variable' | 'system_variable';

// Execution contexts
type ExecutionContext = 'preview' | 'submission' | 'automation' | 'operational';

// Error codes (abbreviated)
type ICMErrorCode = 
  | 'E1000' | 'E1001'
  | 'E2000' | 'E2001' | 'E2002' | 'E2003' | 'E2004'
  | 'E3000' | 'E3001' | 'E3002' | 'E3003'
  | 'E4000' | 'E4001' | 'E4002' | 'E4003'
  | 'E5000' | 'E5001';

// Log levels
type LogLevel = 'DEBUG' | 'INFO' | 'WARN' | 'ERROR';
```

---

> **End of Design Document**
> 
> This document defines all required design elements for Phase 1 (ICM Design) of the Internal Command Module (PR02).
