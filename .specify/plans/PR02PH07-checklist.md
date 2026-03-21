# PR02PH07 — New Command Addition Checklist

> Use this checklist when adding new commands to the ICM system.

---

## Phase 1: Specification

- [ ] **1.1** Created command specification document
- [ ] **1.2** Defined command name and operator family
- [ ] **1.3** Documented command purpose and use cases
- [ ] **1.4** Specified syntax (BNF or informal)
- [ ] **1.5** Listed all arguments with types and defaults
- [ ] **1.6** Provided usage examples
- [ ] **1.7** Determined execution contexts (Preview/Submission/Automation/Operational)
- [ ] **1.8** Defined authority requirements
- [ ] **1.9** Documented expected state effects
- [ ] **1.10** Specified safety hooks to apply

---

## Phase 2: Backend Implementation

### 2.1 Handler Implementation

- [ ] **2.1.1** Created handler struct in [`internal/icm/dispatcher.go`](internal/icm/dispatcher.go) or dedicated file
- [ ] **2.1.2** Implemented `Handle()` method
- [ ] **2.1.3** Implemented `Validate()` method
- [ ] **2.1.4** Implemented `Name()` method returning command name
- [ ] **2.1.5** Implemented `Description()` method for help
- [ ] **2.1.6** Implemented context authorization checks
- [ ] **2.1.7** Implemented proper error handling with ICM error codes

### 2.2 Registration

- [ ] **2.2.1** Registered handler in dispatcher's `registerBuiltInHandlers()`
- [ ] **2.2.2** Used correct operator family (`OperatorStructured`, `OperatorAlias`, etc.)

### 2.3 Recognizer Updates

- [ ] **2.3.1** Updated recognizer to identify command syntax
- [ ] **2.3.2** Added token type for command (if unique)
- [ ] **2.3.3** Tested recognition of command in various contexts

### 2.4 Validator Updates

- [ ] **2.4.1** Added validation rules in [`internal/icm/validator.go`](internal/icm/validator.go)
- [ ] **2.4.2** Defined argument count validation
- [ ] **2.4.3** Defined argument format validation
- [ ] **2.4.4** Used appropriate error codes (E2xxx for validation errors)

### 2.5 Normalizer Updates

- [ ] **2.5.1** Updated normalizer to produce canonical form
- [ ] **2.5.2** Defined transformation rules
- [ ] **2.5.3** Tested normalization with various inputs

### 2.6 Registry Updates (for variables/aliases)

- [ ] **2.6.1** Added alias definition type (if `@` command)
- [ ] **2.6.2** Added variable definition type (if `$` or `%` command)
- [ ] **2.6.3** Registered in default variables (if system variable)
- [ ] **2.6.4** Implemented getter functions

---

## Phase 3: Safety Integration

- [ ] **3.1** Verified circuit breaker integration
- [ ] **3.2** Verified rate limiting integration
- [ ] **3.3** Verified queue depth integration
- [ ] **3.4** Verified timeout handling
- [ ] **3.5** Added recursion depth handling (if applicable)
- [ ] **3.6** Documented any command-specific safety behavior

---

## Phase 4: Testing

### 4.1 Unit Tests

- [ ] **4.1.1** Added unit tests for handler `Validate()` method
- [ ] **4.1.2** Added unit tests for handler `Handle()` method
- [ ] **4.1.3** Covered valid input cases
- [ ] **4.1.4** Covered error cases
- [ ] **4.1.5** Covered edge cases
- [ ] **4.1.6** Tested all execution contexts

### 4.2 Integration Tests

- [ ] **4.2.1** Added end-to-end pipeline test
- [ ] **4.2.2** Tested full recognition → validation → normalization → dispatch flow
- [ ] **4.2.3** Verified state effects are applied correctly
- [ ] **4.2.4** Verified safety checks are enforced

### 4.3 Regression Tests

- [ ] **4.3.1** Verified existing commands still work
- [ ] **4.3.2** Verified escape sequences still work
- [ ] **4.3.3** Verified pass-through behavior unchanged

---

## Phase 5: Frontend Integration

- [ ] **5.1** Added validation function to [`frontend/src/services/icm-adapter.ts`](frontend/src/services/icm-adapter.ts)
- [ ] **5.2** Added preview function (if applicable)
- [ ] **5.3** Added execute function (if applicable)
- [ ] **5.4** Updated API client if new endpoints needed
- [ ] **5.5** Added frontend tests

---

## Phase 6: Documentation

### 6.1 Code Documentation

- [ ] **6.1.1** Added godoc comments to handler
- [ ] **6.1.2** Documented execution contexts
- [ ] **6.1.3** Documented error conditions

### 6.2 Help System

- [ ] **6.2.1** Added help text to handler (`HelpText()` method)
- [ ] **6.2.2** Created help file in [`help/`](help/) directory (if user-facing)
- [ ] **6.2.3** Added command to help index

### 6.3 Specification

- [ ] **6.3.1** Updated command specification document with implementation notes
- [ ] **6.3.2** Added revision history entry

---

## Phase 7: Review

- [ ] **7.1** Code reviewed by another developer
- [ ] **7.2** Security implications reviewed
- [ ] **7.3** Performance implications reviewed
- [ ] **7.4** Backward compatibility verified
- [ ] **7.5** All acceptance criteria met

---

## Quick Reference

### Error Code Reference

| Range | Category |
|-------|----------|
| E1xxx | Recognition Errors |
| E2xxx | Validation Errors |
| E3xxx | Resolution Errors |
| E4xxx | Permission/Context Errors |
| E5xxx | Internal Errors |

### File Locations

| Component | Location |
|-----------|----------|
| Handler | [`internal/icm/dispatcher.go`](internal/icm/dispatcher.go) |
| Recognizer | [`internal/icm/recognizer.go`](internal/icm/recognizer.go) |
| Validator | [`internal/icm/validator.go`](internal/icm/validator.go) |
| Normalizer | [`internal/icm/normalizer.go`](internal/icm/normalizer.go) |
| Registry | [`internal/icm/registry.go`](internal/icm/registry.go) |
| Engine | [`internal/icm/engine.go`](internal/icm/engine.go) |
| Types | [`internal/icm/types.go`](internal/icm/types.go) |
| Errors | [`internal/icm/errors.go`](internal/icm/errors.go) |
| Tests | [`internal/icm/icm_test.go`](internal/icm/icm_test.go) |
| Frontend Adapter | [`frontend/src/services/icm-adapter.ts`](frontend/src/services/icm-adapter.ts) |

### Execution Contexts

| Context | Description | Use For |
|---------|-------------|---------|
| `ContextPreview` | Local validation | Editor validation, help |
| `ContextSubmission` | User command | Direct input |
| `ContextAutomation` | Automation | Alias/trigger/timer |
| `ContextOperational` | Service tooling | Admin commands |

---

## Sign-Off

| Role | Name | Date | Signature |
|------|------|------|-----------|
| Developer | | | |
| Reviewer | | | |
| QA | | | |
| Product Owner | | | |
