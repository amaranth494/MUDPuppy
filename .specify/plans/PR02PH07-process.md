# PR02PH07 — New-Command Authoring Standard

> **Plan Type:** Process Documentation  
> **Spec Reference:** PR02 — Internal Command Module (ICM)  
> **Version:** 0.2.0-pr2  
> **Status:** Draft

---

## Overview

This document establishes the repeatable process for adding new internal commands to the MUDPuppy Internal Command Module (ICM). Following this process ensures consistency, maintains architectural integrity, and preserves the constitutional separation between frontend authoring and backend authority.

---

## 1. Command Lifecycle Review

Before adding a new command, ensure familiarity with the ICM command lifecycle:

```
ingress → tokenize/recognize → validate → normalize → resolve → dispatch/report
```

Each phase has distinct responsibilities:

| Phase | Responsibility | Location |
|-------|---------------|----------|
| **Recognize** | Tokenize input, identify operator families | [`internal/icm/recognizer.go`](internal/icm/recognizer.go) |
| **Validate** | Check syntax validity, validate arguments | [`internal/icm/validator.go`](internal/icm/validator.go) |
| **Normalize** | Convert to canonical form | [`internal/icm/normalizer.go`](internal/icm/normalizer.go) |
| **Resolve** | Resolve variables, aliases | [`internal/icm/engine.go`](internal/icm/engine.go) |
| **Dispatch** | Route to handler, enforce safety | [`internal/icm/dispatcher.go`](internal/icm/dispatcher.go) |

---

## 2. Adding a New Structured Directive (#COMMAND)

Structured directives use the `#` prefix (e.g., `#IF`, `#SET`, `#LOG`).

### 2.1 Determine Command Classification

| Classification | Description | Example |
|---------------|-------------|---------|
| **User-Facing** | Available to end users via command entry | `#IF`, `#SET`, `#TIMER` |
| **Operational** | Developer/sysadmin only | `#LOG`, `#DEBUG` |
| **Internal** | Used by automation engine only | `#ELSE`, `#ENDIF` |

### 2.2 Step-by-Step Process

#### Step 1: Define the Command Specification

Create a specification document (can be inline in code comments) containing:

```markdown
## COMMAND_NAME

- **Operator Family:** # (Structured)
- **Purpose:** [One sentence describing what this command does]
- **Syntax:** `#COMMAND <arg1> [arg2]`
- **Execution Contexts:** Preview, Submission, Automation, Operational
- **Authority Required:** CanExecute (for execution contexts)
- **Effects:** What state changes does this produce?
- **Safety Hooks:** Which safeguards apply?
```

#### Step 2: Implement the Handler

Create a new handler struct in [`internal/icm/dispatcher.go`](internal/icm/dispatcher.go) or a dedicated file:

```go
// MyCommandHandler handles the #MYCOMMAND directive
type MyCommandHandler struct{}

// Handle executes the #MYCOMMAND directive
func (h *MyCommandHandler) Handle(ctx *ExecutionContext, cmd *NormalizedCommand) (*CommandResult, *ICMError) {
    // Implementation
    return &CommandResult{
        Output:   "result",
        Effects:  []StateEffect{},
        Metadata: ResultMetadata{
            ExecutionTime: 0,
            Handler:      "MYCOMMAND",
        },
    }, nil
}

// Validate validates the command arguments
func (h *MyCommandHandler) Validate(cmd *NormalizedCommand) *ICMError {
    // Validation logic
    return nil
}

// Name returns the command name
func (h *MyCommandHandler) Name() string {
    return "MYCOMMAND"
}

// Description returns the command description
func (h *MyCommandHandler) Description() string {
    return "Description of what this command does"
}
```

#### Step 3: Register the Handler

Register the handler in the dispatcher's `registerBuiltInHandlers()` method:

```go
func (d *Dispatcher) registerBuiltInHandlers() {
    // Existing handlers...
    d.RegisterHandler(OperatorStructured, "MYCOMMAND", &MyCommandHandler{})
}
```

#### Step 4: Update the Recognizer

If the command has unique syntax requirements, update [`internal/icm/recognizer.go`](internal/icm/recognizer.go) to properly tokenize the command.

#### Step 5: Add Validation Rules

If the command requires specific argument validation, update [`internal/icm/validator.go`](internal/icm/validator.go).

#### Step 6: Define Normalization

Update [`internal/icm/normalizer.go`](internal/icm/normalizer.go) to convert input to canonical form.

#### Step 7: Add Tests

Add unit tests in [`internal/icm/icm_test.go`](internal/icm/icm_test.go):

```go
func TestMyCommand(t *testing.T) {
    // Test cases for the new command
}
```

#### Step 8: Update Frontend Adapter (if needed)

If the command needs frontend validation or preview, update [`frontend/src/services/icm-adapter.ts`](frontend/src/services/icm-adapter.ts).

---

## 3. Adding a New Alias Reference (@alias)

Aliases are user-defined shortcuts that expand to full commands.

### 3.1 Step-by-Step Process

#### Step 1: Define Alias Storage

Aliases are stored in the Registry ([`internal/icm/registry.go`](internal/icm/registry.go)):

```go
type AliasDefinition struct {
    Name        string
    Expansion   string
    Description string
    CreatedAt   int64
}
```

#### Step 2: Registration Methods

Use existing registry methods:

```go
// Register a new alias
registry.RegisterAlias("aliasname", "expansion text")

// Retrieve an alias
alias, exists := registry.GetAlias("aliasname")
```

#### Step 3: Recognition

The recognizer ([`internal/icm/recognizer.go`](internal/icm/recognizer.go)) identifies `@` prefix and extracts the alias name.

#### Step 4: Resolution

The engine ([`internal/icm/engine.go`](internal/icm/engine.go)) resolves aliases through the registry.

---

## 4. Adding a New User Variable ($name)

User variables are profile-scoped, user-editable variables.

### 4.1 Step-by-Step Process

#### Step 1: Define Variable Storage

```go
type VariableDefinition struct {
    Name       string
    Value      string
    IsReadOnly bool
}
```

#### Step 2: Use Existing Registry Methods

```go
// Set a user variable
registry.SetUserVariable("varname", "value")

// Get a user variable
value, exists := registry.GetUserVariable("varname")
```

#### Step 3: Recognition

The recognizer identifies `$name` or `${name}` syntax.

#### Step 4: The #SET Directive

User variables are typically created/modified via the `#SET` directive, which is already implemented in [`internal/icm/dispatcher.go`](internal/icm/dispatcher.go) (`SetHandler`).

---

## 5. Adding a New System Variable (%name)

System variables are session-scoped or read-only client data.

### 5.1 Types of System Variables

| Type | Description | Example |
|------|-------------|---------|
| **Numeric** | Session-scoped temporary arguments | `%1`, `%2`, `%42` |
| **Named** | Read-only client data | `%TIME`, `%CHARACTER` |

### 5.2 Step-by-Step Process

#### Step 1: Define System Variable

```go
type SystemVariableDefinition struct {
    Name        string
    Description string
    IsReadOnly  bool
    Getter      func() string
}
```

#### Step 2: Register the Variable

In [`internal/icm/registry.go`](internal/icm/registry.go), add to `registerDefaultSystemVariables()`:

```go
r.systemVariables["VARIABLENAME"] = &SystemVariableDefinition{
    Name:        "VARIABLENAME",
    Description: "Description of what this variable provides",
    IsReadOnly:  true,
    Getter:      func() string { /* return value */ },
}
```

Or register dynamically:

```go
registry.RegisterSystemVariable("NAME", &SystemVariableDefinition{
    Name:        "NAME",
    Description: "Description",
    IsReadOnly:  true,
    Getter:      func() string { return "value" },
})
```

---

## 6. Execution Contexts and Authority

Each command must specify which execution contexts it supports:

| Context | Description | Use Case |
|---------|-------------|----------|
| **Preview** | Local validation, help display | Editor validation, help text |
| **Submission** | User command entry | Direct user input |
| **Automation** | Alias/trigger/timer execution | Automation engine |
| **Operational** | Service tooling | Admin commands |

### 6.1 Defining Authority

In [`internal/icm/types.go`](internal/icm/types.go), authority is defined per context:

```go
var AuthorityLevels = map[ExecutionContext]Authority{
    ContextPreview: {
        CanValidate:   true,
        CanPreview:    true,
        CanExecute:    false,
        CanModifyState: false,
    },
    ContextSubmission: {
        CanValidate:   true,
        CanPreview:    true,
        CanExecute:    true,
        CanModifyState: true,
    },
    // ... etc
}
```

---

## 7. Safety Integration

All commands must integrate with ICM safety systems:

| Safety System | Purpose | Check Location |
|---------------|---------|-----------------|
| **Recursion Limits** | Maximum alias expansion depth | Dispatcher |
| **Circuit Breaker** | Pause when loops detected | Dispatcher |
| **Rate Limits** | Commands per second | Dispatcher |
| **Queue Depth** | Command queue limits | Dispatcher |
| **Timeout** | Maximum execution time | Dispatcher |

### 7.1 Implementing Safety Checks

The dispatcher ([`internal/icm/dispatcher.go`](internal/icm/dispatcher.go)) automatically enforces safety checks before executing any command. Handlers receive the `ExecutionContext` which includes safety metadata.

---

## 8. Error Handling

### 8.1 Error Codes

Defined in [`internal/icm/errors.go`](internal/icm/errors.go):

| Code | Description |
|------|-------------|
| E1001 | InvalidOperator |
| E2001 | MissingArgument |
| E2002 | InvalidArgument |
| E3001 | UnknownCommand |
| E4001 | PermissionDenied |
| E4002 | ContextNotAllowed |
| E4003 | CircuitBreaker |
| E5000 | InternalError |

### 8.2 Returning Errors

```go
return nil, NewICMError(E2001MissingArgument, "Command requires argument", nil)
```

---

## 9. Frontend Integration

### 9.1 ICM Adapter

The frontend uses [`frontend/src/services/icm-adapter.ts`](frontend/src/services/icm-adapter.ts) to communicate with the backend ICM.

### 9.2 Adding Frontend Validation

For commands that need frontend validation:

```typescript
// In icm-adapter.ts
export async function validateMyCommand(command: string): Promise<ValidationResult> {
  return await apiValidateCommand(command, 'preview');
}
```

---

## 10. Testing Requirements

### 10.1 Unit Tests

All new commands must have unit tests in [`internal/icm/icm_test.go`](internal/icm/icm_test.go):

```go
func TestMyCommand(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        want     *NormalizedCommand
        wantErr  bool
    }{
        // Test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### 10.2 Integration Tests

Test the full pipeline:

```go
func TestEndToEnd_MyCommand(t *testing.T) {
    engine := NewEngine(registry, dispatcher)
    result, err := engine.ProcessCommand("#MYCOMMAND arg1 arg2", ContextSubmission)
    // Assertions
}
```

---

## 11. Documentation Requirements

### 11.1 Code Documentation

Each handler must document:

- Command purpose
- Syntax specification
- Arguments
- Execution contexts supported
- Effects produced

### 11.2 Help System

If the command is user-facing, add help content in [`help/`](help/) directory.

---

## 12. Checklist Summary

Before considering a new command complete, verify:

- [ ] Handler implemented in dispatcher or dedicated file
- [ ] Handler registered in dispatcher
- [ ] Recognizer handles command syntax
- [ ] Validator enforces argument rules
- [ ] Normalizer produces canonical form
- [ ] Authority levels defined
- [ ] Safety integration verified
- [ ] Error codes defined
- [ ] Unit tests added
- [ ] Integration tests added
- [ ] Frontend adapter updated (if needed)
- [ ] Help documentation added (if user-facing)

---

## Appendix: Related Files

| File | Purpose |
|------|---------|
| [`internal/icm/types.go`](internal/icm/types.go) | Core type definitions |
| [`internal/icm/errors.go`](internal/icm/errors.go) | Error types and codes |
| [`internal/icm/engine.go`](internal/icm/engine.go) | Main ICM engine |
| [`internal/icm/recognizer.go`](internal/icm/recognizer.go) | Tokenization and recognition |
| [`internal/icm/validator.go`](internal/icm/validator.go) | Syntax validation |
| [`internal/icm/normalizer.go`](internal/icm/normalizer.go) | Command normalization |
| [`internal/icm/registry.go`](internal/icm/registry.go) | Command registry |
| [`internal/icm/dispatcher.go`](internal/icm/dispatcher.go) | Command dispatch |
| [`internal/icm/escape.go`](internal/icm/escape.go) | Escape handling |
| [`internal/icm/pass_through.go`](internal/icm/pass_through.go) | Pass-through logic |
| [`frontend/src/services/icm-adapter.ts`](frontend/src/services/icm-adapter.ts) | Frontend adapter |
| [`internal/icm/icm_test.go`](internal/icm/icm_test.go) | Unit tests |
