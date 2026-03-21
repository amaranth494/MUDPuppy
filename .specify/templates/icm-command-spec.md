# ICM Command Specification Template

> Use this template to document a new ICM command

---

## Command Overview

| Field | Value |
|-------|-------|
| **Command Name** | `COMMANDNAME` |
| **Operator Family** | `#` (Structured Directive) / `@` (Alias) / `$` (User Variable) / `%` (System Variable) |
| **Version** | 1.0.0 |
| **Status** | Draft / Review / Approved / Implemented |
| **Created** | YYYY-MM-DD |
| **Author** | Name/Team |

---

## 1. Purpose

> One sentence describing what this command does.

[Extended description of the command's purpose and use cases.]

---

## 2. Syntax Specification

### 2.1 Grammar

```
#COMMANDNAME <required_arg> [optional_arg]
```

### 2.2 Formal Syntax (BNF)

```bnf
<command> ::= "#COMMANDNAME" <space> <required_arg> [<space> <optional_arg>]
<required_arg> ::= <string>
<optional_arg> ::= <string>
```

### 2.3 Argument Specifications

| Argument | Required | Type | Default | Description |
|----------|----------|------|---------|-------------|
| `required_arg` | Yes | string | N/A | Description of required argument |
| `optional_arg` | No | string | `"default"` | Description of optional argument |

### 2.4 Examples

```mud
#COMMANDNAME myvalue
#COMMANDNAME myvalue myoptional
#COMMANDNAME "value with spaces"
```

---

## 3. Execution Contexts

| Context | Supported | Authority Required |
|---------|-----------|-------------------|
| Preview | Yes/No | CanValidate |
| Submission | Yes/No | CanExecute |
| Automation | Yes/No | CanExecute |
| Operational | Yes/No | CanExecute |

---

## 4. Normalization

### 4.1 Canonical Form

```
#COMMANDNAME <required_arg> [optional_arg]
```

### 4.2 Transformation Rules

- Arguments are trimmed of leading/trailing whitespace
- Single-word arguments are kept as-is
- Multi-word arguments are preserved with spaces

---

## 5. Validation Rules

| Rule | Error Code | Error Message |
|------|------------|---------------|
| At least 1 argument required | E2001 | "COMMANDNAME requires at least 1 argument" |
| Argument must be alphanumeric | E2002 | "Argument must contain only letters and numbers" |
| Optional argument max length | E2002 | "Optional argument exceeds maximum length of 100" |

---

## 6. Effects

### 6.1 State Effects

| Effect Type | Key | Value | Scope |
|-------------|-----|-------|-------|
| `variable` | `varname` | string | Profile/Session |
| `timer` | `timername` | TimerDef | Session |
| `trigger` | `triggername` | TriggerDef | Profile |
| `log` | N/A | LogEntry | N/A |

### 6.2 Output

> Description of what this command outputs, if anything.

---

## 7. Safety Integration

| Safety Check | Applied | Notes |
|--------------|---------|-------|
| Recursion Limit | Yes/No | |
| Circuit Breaker | Yes/No | |
| Rate Limit | Yes/No | |
| Queue Depth | Yes/No | |
| Timeout | Yes/No | Default: Xms |

---

## 8. Error Handling

### 8.1 Error Codes

| Code | Condition |
|------|-----------|
| E2001 | MissingArgument |
| E2002 | InvalidArgument |
| E4001 | PermissionDenied |
| E4002 | ContextNotAllowed |

### 8.2 Error Messages

| Code | Message |
|------|---------|
| E2001 | `#COMMANDNAME requires at least 1 argument` |
| E2002 | `Invalid argument: {details}` |

---

## 9. Help Documentation

### 9.1 Short Help

`#COMMANDNAME - Does something useful`

### 9.2 Long Help

```
#COMMANDNAME <required> [optional]

Description of what this command does.

Arguments:
  required  - What this required argument does
  optional  - What this optional argument does (default: default_value)

Examples:
  #COMMANDNAME myarg
  #COMMANDNAME myarg myoptional

See Also:
  #RELATED_COMMAND
```

---

## 10. Related Commands

| Command | Relationship |
|---------|--------------|
| `#EXISTING` | Similar functionality / Used together |
| `@ALIAS` | Example alias using this command |

---

## 11. Implementation Notes

### 11.1 Files Modified

| File | Change |
|------|--------|
| `internal/icm/dispatcher.go` | Add handler registration |
| `internal/icm/recognizer.go` | Add tokenization |
| `internal/icm/validator.go` | Add validation rules |
| `internal/icm/normalizer.go` | Add normalization |
| `internal/icm/icm_test.go` | Add tests |

### 11.2 API Changes

| Endpoint | Change |
|----------|--------|
| None | Required / Added |

---

## 12. QA Scenarios

| Scenario | Input | Expected Output |
|----------|-------|-----------------|
| Basic execution | `#COMMANDNAME value` | Success with effects |
| Missing argument | `#COMMANDNAME` | E2001 error |
| Invalid argument | `#COMMANDNAME !@#` | E2002 error |
| Preview context | `#COMMANDNAME value` | Preview result, no execution |
| Automation context | `#COMMANDNAME value` | Executed from alias |

---

## 13. Revision History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0.0 | YYYY-MM-DD | Name | Initial specification |
