# PR02 Developer Guidance — Internal Command Module (ICM)

> **For:** MUDPuppy Developers  
> **Version:** 0.2.0-pr2  
> **Date:** 2026-03-21

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Core Components](#core-components)
3. [Command Lifecycle](#command-lifecycle)
4. [Execution Contexts](#execution-contexts)
5. [Adding New Commands](#adding-new-commands)
6. [Frontend Integration](#frontend-integration)
7. [Safety Integration](#safety-integration)
8. [Testing](#testing)
9. [Troubleshooting](#troubleshooting)

---

## Architecture Overview

The Internal Command Module (ICM) is the **backend-authoritative** system for processing all internal commands in MUDPuppy. It provides:

- **Single source of truth** for operator semantics
- **Clear separation** between frontend (authoring) and backend (execution)
- **Consistent processing** across all command entry paths

### Constitutional Reconciliation

Per the Constitution (Sections IV.0 and IV.a):

| Layer | Responsibility |
|-------|----------------|
| **Frontend/Client** | Authoring, preview, validation messaging, help display |
| **Backend** | Command semantics, session state, safety enforcement, dispatch |

The ICM serves as the authoritative bridge between these layers.

---

## Core Components

### Backend (`internal/icm/`)

| File | Purpose |
|------|---------|
| [`types.go`](internal/icm/types.go) | Core type definitions (Command, Operator, Context) |
| [`errors.go`](internal/icm/errors.go) | Error types and codes |
| [`engine.go`](internal/icm/engine.go) | Main ICM engine orchestrator |
| [`handler.go`](internal/icm/handler.go) | HTTP handler for ICM endpoints |
| [`recognizer.go`](internal/icm/recognizer.go) | Tokenization and operator detection |
| [`validator.go`](internal/icm/validator.go) | Syntax validation and argument checking |
| [`normalizer.go`](internal/icm/normalizer.go) | Canonical command formatting |
| [`registry.go`](internal/icm/registry.go) | Command/handler registration |
| [`dispatcher.go`](internal/icm/dispatcher.go) | Handler routing with safety enforcement |
| [`escape.go`](internal/icm/escape.go) | Escape sequence processing |
| [`pass_through.go`](internal/icm/pass_through.go) | Unknown operator handling |

### Frontend (`frontend/src/services/`)

| File | Purpose |
|------|---------|
| [`icm-adapter.ts`](frontend/src/services/icm-adapter.ts) | TypeScript adapter for ICM communication |

---

## Command Lifecycle

```
User Input → Ingress → Tokenize → Validate → Normalize → Resolve → Dispatch → Result
```

### Phase Details

1. **Ingress**: Command enters via typed input, keybinding, history, or automation
2. **Tokenize/Recognize**: Identify operator family (#, @, $, %), extract command
3. **Validate**: Check syntax validity, required arguments
4. **Normalize**: Convert to canonical form (e.g., `$name` → `${name}`)
5. **Resolve**: Look up aliases, resolve variables
6. **Dispatch**: Route to handler, enforce safety
7. **Result**: Return execution result or error

---

## Execution Contexts

ICM commands execute in different contexts with varying authority levels:

| Context | Description | Authority |
|---------|-------------|-----------|
| **Preview** | Local validation, help display | Frontend-only |
| **Submission** | User command entry, keybinding | Frontend → Backend |
| **Automation** | Alias expansion, trigger action | Backend-authoritative |
| **Operational** | Service tooling, admin commands | Backend-only |

### Context Usage

```go
// In handler.go
func (h *Handler) ProcessCommand(cmd *Command, ctx Context) (*Result, error) {
    switch ctx {
    case Preview:
        return h.preview(cmd)
    case Submission:
        return h.submit(cmd)
    case Automation:
        return h.automate(cmd)
    case Operational:
        return h.operate(cmd)
    }
}
```

---

## Adding New Commands

### Step 1: Define the Command

Add to [`registry.go`](internal/icm/registry.go):

```go
func (r *Registry) registerCommands() {
    r.Register(CommandDefinition{
        Name:        "MYCOMMAND",
        Operator:    Hash,  // # operator
        Description: "My new command",
        Syntax:      "#MYCOMMAND <arg1> [arg2]",
        Handler:     MyCommandHandler{},
        Contexts:    []Context{Automation, Operational},
    })
}
```

### Step 2: Implement the Handler

```go
type MyCommandHandler struct{}

func (h MyCommandHandler) Execute(ctx *ExecutionContext, args []string) (*CommandResult, error) {
    // Validate arguments
    if len(args) < 1 {
        return nil, errors.New(ErrInvalidArguments, "expected at least 1 argument")
    }
    
    // Process command
    // ...
    
    return &CommandResult{
        Output: "command executed",
    }, nil
}
```

### Step 3: Register System Variables (if applicable)

```go
func (r *Registry) registerSystemVariables() {
    r.RegisterSystemVariable(SystemVariable{
        Name:        "MYVAR",
        Description: "My custom variable",
        Getter: func(ctx *ExecutionContext) string {
            return "value"
        },
    })
}
```

### Step 4: Update Frontend Adapter (if needed)

Add to [`icm-adapter.ts`](frontend/src/services/icm-adapter.ts):

```typescript
export interface MyCommandResult {
  output: string;
}

export async function executeMyCommand(
  args: string[],
  context: ExecutionContext
): Promise<ICMResponse<MyCommandResult>> {
  // Call backend ICM endpoint
}
```

### Step 5: Add Tests

```go
// In icm_test.go
func TestMyCommand(t *testing.T) {
    engine := NewEngine()
    
    result, err := engine.Execute("#MYCOMMAND arg1 arg2", Automation)
    require.NoError(t, err)
    assert.Equal(t, "command executed", result.Output)
}
```

---

## Frontend Integration

### Recognizing Commands

```typescript
import { recognizeCommand } from '@/services/icm-adapter';

// Classify input
const result = recognizeCommand('kill $target');
// result: {
//   isInternal: true,
//   operator: '$',
//   command: 'variable',
//   normalized: 'kill ${target}'
// }
```

### Validating Commands

```typescript
import { validateCommand } from '@/services/icm-adapter';

// Validate before submission
const validation = await validateCommand('#SET target=goblin');
// validation: {
//   valid: true,
//   errors: []
// }
```

### Error Handling

```typescript
const validation = await validateCommand('#INVALID args');
if (!validation.valid) {
  // Show error to user
  console.error(validation.errors);
}
```

---

## Safety Integration

### Required Safety Hooks

The ICM integrates with existing constitutional safeguards:

| Safety System | Integration Point |
|---------------|-------------------|
| Recursion limits | Dispatcher checks expansion depth |
| Command dispatch limits | Dispatcher counts commands |
| Circuit breaker | Dispatcher monitors for loops |
| Rate limiting | Dispatcher enforces minimum interval |

### Implementing Safety Checks

```go
func (d *Dispatcher) dispatch(cmd *Command) (*Result, error) {
    // Check circuit breaker
    if d.circuitBreaker.IsOpen() {
        return nil, errors.New(ErrCircuitBreakerOpen)
    }
    
    // Check rate limit
    if !d.rateLimiter.Allow() {
        return nil, errors.New(ErrRateLimited)
    }
    
    // Dispatch to handler
    return d.handlers[cmd.Name].Execute(cmd)
}
```

---

## Testing

### Unit Tests

Run backend unit tests:

```bash
go test -v ./internal/icm/...
```

### Integration Tests

The test file includes:

- Command recognition
- Validation
- Normalization
- Dispatch
- Safety enforcement
- #LOG governance
- End-to-end pipelines

### Frontend Tests

```bash
cd frontend
npm test
```

---

## Troubleshooting

### Common Issues

#### 1. Command Not Recognized

**Symptom**: Command returns "unknown command" error

**Solution**: 
- Check the operator is correct (#, @, $, %)
- Verify command is registered in registry.go
- Check spelling and case

#### 2. Variable Not Resolved

**Symptom**: `${variablename}` passes through literally

**Solution**:
- Verify variable exists in registry
- Check variable name case (case-sensitive)
- Ensure execution context has variable scope

#### 3. Validation Errors

**Symptom**: "Invalid arguments" error

**Solution**:
- Check required arguments
- Verify argument types
- Review syntax specification

#### 4. Safety Errors

**Symptom**: "Rate limited" or "Circuit breaker" errors

**Solution**:
- Wait before retrying
- Check for infinite loops
- Review automation configuration

---

## API Reference

### Backend Endpoint

```
POST /api/icm/execute
Content-Type: application/json

{
  "raw": "#SET target=goblin",
  "context": "submission"
}
```

### Response

```json
{
  "success": true,
  "output": "Variable target set to goblin",
  "normalized": "#SET target=goblin",
  "operator": "#",
  "command": "SET"
}
```

---

## Related Documents

- [PR02 Specification](../specs/PR02.md)
- [Design Document](./PR02PH01-design.md)
- [Audit Document](./PR02PH00-audit.md)
- [Release Notes](./PR02-release-notes.md)
- [Process Documentation](./PR02PH07-process.md)
- [Command Templates](../templates/)
