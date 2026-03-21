# PR02 Release Notes — Internal Command Module (ICM)

> **Release Version:** 0.2.0-pr2  
> **Date:** 2026-03-21  
> **Feature Branch:** pr02-internal-command-module

---

## Overview

PR02 introduces the Internal Command Module (ICM), a unified backend-authoritative system for recognizing, parsing, validating, normalizing, and dispatching all internal command forms in MUDPuppy.

This release consolidates previously scattered operator handling into a single, maintainable architecture with clear separation between frontend (authoring/validation) and backend (execution/authority).

---

## New Features

### 1. Unified Command Processing

- **ICM Engine**: New backend service for processing all internal commands
- **Command Lifecycle**: Ingress → Tokenize → Validate → Normalize → Resolve → Dispatch
- **Frontend Adapter**: TypeScript adapter for validation, preview, and submission

### 2. Operator Families

All four operator families are now handled through ICM:

| Operator | Purpose | Examples |
|----------|---------|----------|
| `#` | Structured directives | #IF, #SET, #TIMER, #ELSE, #ENDIF |
| `@` | Alias references | @attack |
| `$` | User variables | $target, ${target} |
| `%` | System variables | %1, %TIME, %CHARACTER |

### 3. Structured Directives

Full support for automation control directives:

- **#IF / #ELSE / #ENDIF**: Conditional command execution
- **#SET**: Create and update user variables
- **#TIMER**: Create/update timer definitions
- **#START / #STOP**: Control timer execution
- **#CHECK**: Inspect timer status
- **#CANCEL**: Delete timer definitions

### 4. #LOG Directive

New operational directive for standardized logging:

- **Governed Access**: Restricted to Automation and Operational contexts
- **Structured Output**: JSON format with timestamp, level, correlation ID
- **Backend Routing**: Logs to backend logging service

### 5. System Variables

Pre-registered system variables:

- **Numeric** (%1, %2, etc.): Session-scoped arguments, cleared on disconnect
- **Named** (%TIME, %DATE, %CHARACTER, %SERVER, %SESSIONID, %HOST, %PORT)

### 6. Escape Sequences

Literal operator characters supported:

| Escape | Result |
|--------|--------|
| `\#` | Literal # |
| `\@` | Literal @ |
| `\$` | Literal $ |
| `\%` | Literal % |

---

## Architecture Changes

### Backend (Authoritative)

New ICM components in `internal/icm/`:

- **types.go**: Core type definitions
- **errors.go**: Error types and codes
- **engine.go**: Main ICM engine
- **handler.go**: HTTP handler for ICM endpoints
- **recognizer.go**: Tokenization and operator recognition
- **validator.go**: Syntax validation
- **normalizer.go**: Canonical command formatting
- **registry.go**: Command registration
- **dispatcher.go**: Handler routing with safety enforcement
- **escape.go**: Escape sequence processing
- **pass_through.go**: Unknown operator handling

### Frontend Integration

Updated files:

- **icm-adapter.ts**: New service for ICM communication
- **PlayScreen.tsx**: Integrated ICM for command submission
- **SettingsPage.tsx**: Integrated ICM for alias/trigger/timer editors
- **automation.ts**: Refactored to use ICM for operator detection
- **evaluator.ts**: Refactored to use ICM for variable resolution

---

## Breaking Changes

### None

This release maintains full backward compatibility:

- All existing aliases, triggers, timers, and variables continue to work
- Command processing order preserved
- Pass-through behavior unchanged for non-internal commands

---

## Deprecations

### None

No features have been deprecated in this release.

---

## Migration Notes

### For Users

No user action required. All existing automation continues to work.

### For Developers

If you have custom code that processes operators (#, @, $, %), it should now use the ICM adapter:

```typescript
import { recognizeCommand, validateCommand } from '@/services/icm-adapter';

// Classify a command
const result = recognizeCommand('kill $target');
// result: { isInternal: true, operator: '$', command: 'variable', ... }

// Validate before submission
const validation = await validateCommand('kill $target');
// validation: { valid: true, ... }
```

---

## Bug Fixes

- **Improved error messages**: Invalid commands now return clear error messages
- **Variable resolution**: Undefined variables now correctly pass through literally
- **Operator validation**: Unknown operators in reserved families now properly rejected

---

## QA Verification

All command entry paths tested:

- ✅ Typed command input through ICM
- ✅ Keybinding dispatch through ICM  
- ✅ History replay through ICM
- ✅ Editor validation through ICM
- ✅ Alias expansion through ICM
- ✅ Trigger action through ICM
- ✅ Timer fire through ICM

Safety systems verified:

- ✅ Recursion limits enforced
- ✅ Circuit breaker protection
- ✅ Rate limiting

---

## Future Considerations

### Planned Enhancements

- Extended variable syntax
- Regex-based pattern operators
- Additional system variables
- Enhanced debugging tools

### Developer Commands (Future)

- `#ECHO`: Debug echo commands
- `#DEBUG`: Debug mode controls
- `#PROFILE`: Performance profiling

---

## Credits

- **PR02**: Internal Command Module implementation
- **Architecture**: Backend-authoritative command processing
- **Integration**: Frontend ICM adapter

---

## Related Documentation

- [PR02 Specification](../specs/PR02.md)
- [PR02 Design Document](../plans/PR02PH01-design.md)
- [Help: Internal Commands](../../help/internal-commands.json)
- [Help: Variables](../../help/variables.json)
- [Help: Aliases](../../help/aliases.json)
- [Help: Triggers](../../help/triggers.json)
