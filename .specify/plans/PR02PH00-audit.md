# PR02PH00 Audit Findings - Command Processing Review

> **Phase:** PR02PH00 (Setup)  
> **Task:** PR02PH00T01 + PR02PH00T02  
> **Date:** 2026-03-19

---

## 1. Current Architecture Overview

### 1.1 Command Ingress Routes

```mermaid
graph TD
    A[User Types Command] --> B[PlayScreen.submitCommand]
    C[Keybinding Press] --> D[useInputInterceptor Hook]
    D --> B
    B --> E[AutomationEngine.processUserInput]
    E --> F{Command Type?}
    F -->|# command| G[executeAutomationAction]
    F -->|Alias @name| H[evaluateAlias]
    F -->|Variable ${name}| I[substituteVariables]
    F -->|MUD command| J[wsManager.sendCommand]
    G --> K[Queue Commands]
    H --> K
    I --> K
    K --> J
    J --> L[MUD Server]
```

### 1.2 Current Operator Handling Locations

| Operator | Files | Functions |
|----------|-------|-----------|
| `#` (directives) | `automation.ts`, `parser.ts`, `evaluator.ts` | `processUserInput()`, `executeAutomationAction()` |
| `@` (aliases) | `automation.ts` | `evaluateAlias()`, `invokeExplicitAlias()` |
| `$` (user variables) | `automation.ts`, `evaluator.ts` | `substituteVariables()` |
| `%` (system/session) | `evaluator.ts` | `SimpleVariableStore.getSession()`, `getSystem()` |

---

## 2. Integration Points for ICM

### 2.1 Primary Frontend Integration Point

**File:** [`frontend/src/services/automation.ts`](frontend/src/services/automation.ts:419)  
**Function:** `AutomationEngine.processUserInput(input: string)`  
**Current Behavior:**
- Command separation (semicolons)
- # command detection and execution
- Variable substitution `${var}`
- Alias evaluation
- Queue management

**ICM Integration:** Replace internal processing with ICM adapter calls for validation, normalization, and dispatch.

### 2.2 Secondary Integration Points

| Location | File | Current Behavior | ICM Role |
|----------|------|------------------|----------|
| Keybindings | [`useInputInterceptor.ts`](frontend/src/hooks/useInputInterceptor.ts:63) | Returns command string | Preview validation |
| History Replay | [`PlayScreen.tsx`](frontend/src/pages/PlayScreen.tsx:544) | Routes through `submitCommand()` | Already covered by primary |
| Settings Timer | [`SettingsPage.tsx`](frontend/src/pages/SettingsPage.tsx:1453) | Calls `processUserInput()` | Already covered by primary |
| Timer Execution | [`automation/timer.ts`](frontend/src/services/automation/timer.ts:574) | Variable substitution | Backend execution |

### 2.3 Backend Integration Point

**File:** [`internal/session/websocket.go`](internal/session/websocket.go:344)  
**Handler:** `MsgTypeData` case  
**Current Behavior:**
- Rate limiting
- Message size validation
- Forward to MUD server

**ICM Integration:** Route through ICM for session-state commands before forwarding to MUD.

---

## 3. Current Operator Implementation Details

### 3.1 # Structured Directives

**Supported Commands:**
- `#IF`, `#ELSE`, `#ENDIF` - Conditional logic
- `#SET`, `#ADD`, `#SUB` - Variable operations
- `#TIMER`, `#START`, `#STOP`, `#CHECK`, `#CANCEL` - Timer management

**Implementation:**
- Tokenized in [`parser.ts`](frontend/src/services/automation/parser.ts:148)
- Executed in [`evaluator.ts`](frontend/src/services/automation/evaluator.ts:884)
- Validated in [`parser.ts:validateSyntax()`](frontend/src/services/automation/parser.ts:287)

**Migration Note:** Already has structure similar to ICM but needs normalization and registry.

### 3.2 @ Alias References

**Current Implementation:**
- Explicit invocation: `@aliasname` detected via `startsWith('@')`
- Implicit alias expansion via `evaluateAlias()`
- Located in [`automation.ts:645`](frontend/src/services/automation.ts:645)

**Migration Note:** Needs ICM registry integration and normalization.

### 3.3 $ User Variables

**Pattern:** `${variable_name}` (full syntax only, no $name shorthand)  
**Location:** [`automation.ts:625`](frontend/src/services/automation.ts:625)  
**Variable Store:** `SimpleVariableStore` in [`evaluator.ts`](frontend/src/services/automation/evaluator.ts:472)

**Migration Note:** ICM should normalize variable references and resolve against backend store.

### 3.4 % System Variables

**Numeric Form (%1, %2, etc.):**
- Session-scoped temporary variables
- Populated from alias arguments or trigger captures
- Cleared on disconnect

**Named Form (%TIME, %CHARACTER, %SERVER, etc.):**
- Read-only system variables
- Updated via `updateSystemVariable()` in [`automation.ts:244`](frontend/src/services/automation.ts:244)

**Location:** [`evaluator.ts:118`](frontend/src/services/automation/evaluator.ts:118) (SYSTEM_VARIABLES constant)

---

## 4. Gaps Identified

### 4.1 Escape Sequence Handling

**Current State:** NOT IMPLEMENTED  
- No handling for `\#`, `\@`, `\$`, `\%` escape sequences
- Variables only match `${...}` pattern

**Required for ICM:** Implement escape grammar per PR02 spec.

### 4.2 Unknown Operator Handling

**Current State:** Partial  
- Unknown # commands: Errors logged, command not executed
- Unknown @ aliases: Logged, passed through to MUD
- Unknown $ variables: Passed through literally

**Required for ICM:** Define clear behavior per PR02 spec (validation error for reserved families).

### 4.3 Pass-Through Classification

**Current State:** Implicit  
- Non-# commands passed to MUD after variable substitution
- No explicit classification as "internal" vs "MUD" commands

**Required for ICM:** Explicit classification and pass-through logic.

### 4.4 Backend Command Processing

**Current State:** Frontend-only  
- All command processing happens in frontend
- Backend just forwards to MUD server
- No session-state command handling in backend

**Required for ICM:** Backend-authoritative execution for session-state commands.

### 4.5 Command Normalization

**Current State:** Not centralized  
- Each component handles its own format
- No canonical command form defined

**Required for ICM:** Define and enforce normalization format.

---

## 5. Safety Systems Already Present

The current implementation has several safety systems that ICM should integrate with:

| Safety Feature | Location | Details |
|---------------|----------|---------|
| Circuit Breaker | [`automation.ts:118`](frontend/src/services/automation.ts:118) | Trips on loops |
| Loop Detection | [`automation.ts:115`](frontend/src/services/automation.ts:115) | Command history tracking |
| Queue Backpressure | [`automation.ts:110`](frontend/src/services/automation.ts:110) | Max 100 commands |
| Command Rate Limiting | [`websocket.go:110`](internal/session/websocket.go:110) | Backend rate limiter |
| Max Alias Depth | [`automation.ts:65`](frontend/src/services/automation.ts:65) | Max 3 levels |

---

## 6. Migration Scope Summary

### High Priority (Phase 2-3)
1. Create ICM core with registry
2. Implement command normalization
3. Add escape sequence handling
4. Create frontend adapter

### Medium Priority (Phase 4-5)
5. Backend ICM integration
6. Migrate # commands to ICM
7. Migrate variable references to ICM

### Lower Priority (Phase 6)
8. Audit and convert scattered operator logic

---

## 7. Files Requiring Modification

### Frontend
- `frontend/src/services/automation.ts` - Replace processing with ICM adapter
- `frontend/src/services/automation/parser.ts` - Integrate with ICM normalizer
- `frontend/src/services/automation/evaluator.ts` - Integrate with ICM registry
- `frontend/src/hooks/useInputInterceptor.ts` - Add ICM preview validation
- `frontend/src/pages/PlayScreen.tsx` - Ensure ICM integration

### Backend
- `internal/session/websocket.go` - Add ICM routing for session-state commands
- New: `internal/icm/` - ICM core module

---

## 8. Recommendations

1. **Start with ICM core creation** in backend before frontend integration
2. **Define normalization format** early to guide all component changes
3. **Preserve existing safety systems** - integrate rather than replace
4. **Maintain frontend-only for preview/validation** while moving execution to backend
5. **Add escape handling first** - enables testing ICM recognition without breaking existing commands
