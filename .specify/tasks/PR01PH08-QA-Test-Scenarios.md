# PR01PH08 — QA Test Scenarios

> **Spec Reference:** PR01 — Post-MVP Foundation & Advanced Automation Logic  
> **Phase:** PR01PH08 (QA Verification)  
> **Version:** 0.1.0-pr1  

---

## 1. Parser Foundation Tests (PR01PH01)

### Test PR01PH01T01: Tokenization
| ID | Test Case | Input | Expected Output |
|----|-----------|-------|-----------------|
| P01 | Tokenize #IF | `#IF ${hp} < 30` | COMMAND: #IF with condition |
| P02 | Tokenize #ELSE | `#ELSE` | COMMAND: #ELSE |
| P03 | Tokenize #ENDIF | `#ENDIF` | COMMAND: #ENDIF |
| P04 | Tokenize #SET | `#SET target goblin` | COMMAND: #SET with args |
| P05 | Tokenize #TIMER | `#TIMER heal 10s` | COMMAND: #TIMER with args |
| P06 | Tokenize #ENDTIMER | `#ENDTIMER` | COMMAND: #ENDTIMER |
| P07 | Tokenize #CANCEL | `#CANCEL heal` | COMMAND: #CANCEL with timer name |
| P08 | Variable substitution | `${hp}` ${gold} | VARIABLE tokens |
| P09 | Mixed content | `cast ${target}` | TEXT + VARIABLE tokens |

### Test PR01PH01T02: Syntax Validation
| ID | Test Case | Input | Expected Result |
|----|-----------|-------|-----------------|
| V01 | Valid #IF | `#IF ${hp} < 30\ncast heal\n#endif` | No errors |
| V02 | Missing #ENDIF | `#IF ${hp} < 30\ncast heal` | Error: Unclosed #IF block |
| V03 | #ELSE without #IF | `#ELSE\ncast heal` | Error: #ELSE without matching #IF |
| V04 | #ENDIF without #IF | `#ENDIF` | Error: #ENDIF without matching #IF |
| V05 | #SET without args | `#SET` | Error: #SET requires variable name and value |
| V06 | #TIMER without args | `#TIMER` | Error: #TIMER requires name and duration |
| V07 | #CANCEL without args | `#CANCEL` | Error: #CANCEL requires timer name |
| V08 | Unknown command | `#FOO bar` | Error: Unknown command: #FOO |

---

## 2. Logic Engine Tests (PR01PH02)

### Test PR01PH02T01: Condition Parser
| ID | Test Case | Condition | Expected AST |
|----|-----------|-----------|--------------|
| C01 | Simple comparison | `${hp} < 30` | ComparisonNode: variable hp < value 30 |
| C02 | Greater than | `${gold} >= 1000` | ComparisonNode: variable gold >= value 1000 |
| C03 | String equality | `${target} == dragon` | ComparisonNode: variable target == value "dragon" |
| C04 | AND operator | `${hp} < 50 AND ${mana} > 10` | LogicalNode: AND of two comparisons |
| C05 | OR operator | `${hp} == 0 OR ${mana} == 0` | LogicalNode: OR of two comparisons |

### Test PR01PH02T02: Evaluation Engine
| ID | Test Case | Variables | Condition | Expected |
|----|-----------|-----------|-----------|----------|
| E01 | True comparison | hp: 20 | `${hp} < 30` | true |
| E02 | False comparison | hp: 50 | `${hp} < 30` | false |
| E03 | Numeric coercion | gold: "1000" | `${gold} >= 500` | true |
| E04 | String comparison | target: "goblin" | `${target} == goblin` | true |
| E05 | AND true | hp: 20, mana: 20 | `${hp} < 50 AND ${mana} > 10` | true |
| E06 | AND false (first) | hp: 60, mana: 20 | `${hp} < 50 AND ${mana} > 10` | false |
| E07 | AND false (second) | hp: 20, mana: 5 | `${hp} < 50 AND ${mana} > 10` | false |
| E08 | OR true (first) | hp: 10, mana: 5 | `${hp} < 50 OR ${mana} > 10` | true |
| E09 | OR true (second) | hp: 60, mana: 20 | `${hp} < 50 OR ${mana} > 10` | true |
| E10 | OR false | hp: 60, mana: 5 | `${hp} < 50 OR ${mana} > 10` | false |

### Test PR01PH02T03: #IF/#ELSE/#ENDIF Execution
| ID | Test Case | Input | Expected Commands |
|----|-----------|-------|-------------------|
| IF01 | True branch | `#IF ${hp} < 30\ncast heal\n#endif` | [cast heal] |
| IF02 | False branch | `#IF ${hp} < 30\ncast heal\n#endif` (hp: 50) | [] |
| IF03 | If-else true | `#IF ${hp} < 30\ncast heal\n#else\nattack\n#endif` (hp: 20) | [cast heal] |
| IF04 | If-else false | `#IF ${hp} < 30\ncast heal\n#else\nattack\n#endif` (hp: 50) | [attack] |

---

## 3. Variable Operations Tests (PR01PH03)

### Test PR01PH03T01: Variable Resolution Precedence
| ID | Test Case | Setup | Variable | Expected Value |
|----|-----------|-------|----------|----------------|
| R01 | Profile variable | ${target}: "goblin" | ${target} | "goblin" |
| R02 | Session variable | %1: "orc" | %1 | "ororc" |
| R03 | Session overrides profile | ${target}: "goblin", %target: "dragon" | ${target} | "dragon" (session takes precedence) |
| R04 | System variable | %TIME: "12:00" | %TIME | "12:00" |

### Test PR01PH03T02: #SET Persistence
| ID | Test Case | Action | Expected Behavior |
|----|-----------|--------|-------------------|
| S01 | #SET profile variable | `#SET gold 1000` | Variable stored in profile |
| S02 | #SET persists to backend | `#SET gold 1000` | API call to persist variable |
| S03 | Backend failure handling | Backend fails | Variable remains in memory, no error thrown |

### Test PR01PH03T03: System Variable Protection
| ID | Test Case | Input | Expected Result |
|----|-----------|-------|-----------------|
| SY01 | Set %TIME | `#SET %TIME noon` | Error: Cannot modify system variable |
| SY02 | Set %CHARACTER | `#SET %CHARACTER newname` | Error: Cannot modify system variable |
| SY03 | Set %DATE | `#SET %DATE 2024-01-01` | Error: Cannot modify system variable |

### Test PR01PH03T04: Session Variables
| ID | Test Case | Action | Expected Result |
|----|-----------|--------|-----------------|
| SS01 | Set session variable | `#SET %1 goblin` | %1 set to "goblin" |
| SS02 | Session variable cleared | Disconnect | Session variables cleared |
| SS03 | Session variable cleared | New connection | Session variables cleared |

---

## 4. Timer System Tests (PR01PH04)

### Test PR01PH04T01: Timer Creation
| ID | Test Case | Input | Expected Result |
|----|-----------|-------|-----------------|
| T01 | Delayed timer | `#TIMER heal 10s\ncast heal\n#ENDTIMER` | Timer created, fires after 10s |
| T02 | Repeating timer | `#TIMER regen 5s REPEAT\ncast regen\n#ENDTIMER` | Timer created, fires every 5s |
| T03 | Duration parsing (seconds) | `10s` | 10000ms |
| T04 | Duration parsing (minutes) | `5m` | 300000ms |
| T05 | Duration parsing (hours) | `1h` | 3600000ms |
| T06 | Invalid duration | `10x` | Error: Invalid duration |

### Test PR01PH04T04: #CANCEL Command
| ID | Test Case | Action | Expected Result |
|----|-----------|--------|-----------------|
| C01 | Cancel existing timer | Create timer, then `#CANCEL heal` | Timer removed |
| C02 | Cancel non-existent timer | `#CANCEL nonexistent` | Error or no-op |

---

## 5. Safety Controls Tests (PR01PH05)

### Test PR01PH05T03: Evaluation Timeout
| ID | Test Case | Condition | Expected Behavior |
|----|-----------|-----------|-------------------|
| TO01 | Fast evaluation | Simple condition | Completes < 500ms |
| TO02 | Timeout enforcement | Complex condition | Aborted after 500ms |

### Test PR01PH05T02: Safety Systems Integration
| ID | Test Case | Condition | Expected Behavior |
|----|-----------|-----------|-------------------|
| S01 | Circuit breaker | Loop detected | Automation pauses |
| S02 | Rate limiting | >10 triggers/sec | Triggers blocked |
| S03 | Command queue limit | >100 commands queued | Backpressure applied |

---

## 6. Integration Tests (PR01PH07)

### Test PR01PH07T01: Trigger with #IF
| ID | Test Case | Trigger | Action | Server Output | Expected |
|----|-----------|---------|--------|---------------|----------|
| TR01 | Trigger with true condition | "You are hungry" | `#if ${food} > 0\neat bread\n#endif` | "You are hungry" | "eat bread" sent |
| TR02 | Trigger with false condition | "You are hungry" | `#if ${food} > 0\neat bread\n#endif` (food: 0) | "You are hungry" | Nothing sent |

### Test PR01PH07T02: Alias with #SET
| ID | Test Case | Alias Pattern | Replacement | Input | Expected |
|----|-----------|---------------|-------------|-------|----------|
| AL01 | Alias with #SET | "attack" | `#if ${target} == \"\"\n#SET target goblin\n#endif\nkill ${target}` | "attack" | "kill goblin" sent, target saved |

### Test PR01PH07T03: Timer Execution
| ID | Test Case | Timer | Expected Behavior |
|----|-----------|-------|-------------------|
| TE01 | Timer fires | `#TIMER heal 1s\ncast heal\n#ENDTIMER` | After 1s, "cast heal" sent |
| TE02 | Repeating timer | `#TIMER regen 1s REPEAT\ncast regen\n#ENDTIMER` | Every 1s, "cast regen" sent |

---

## 7. End-to-End Test Scenarios

### E2E01: Combat Healing Automation
**Setup:**
- Variable: hp (profile variable)
- Variable: mana (profile variable)
- Variable: food (profile variable)

**Test:**
1. Set hp=20, mana=50, food=5
2. Trigger: "You take damage"
3. Action: `#if ${hp} < 30\ncast heal\n#endif`
4. **Expected:** "cast heal" command sent

### E2E02: Gold Management Automation
**Setup:**
- Variable: gold (profile variable)

**Test:**
1. Set gold=1500
2. Alias pattern: "bank"
3. Replacement: `#if ${gold} > 1000\nbank deposit\n#else\nget coin\n#endif`
4. User types: "bank"
5. **Expected:** "bank deposit" command sent

### E2E03: Auto-Buff Timer
**Setup:**
- No variables needed

**Test:**
1. Create timer: `#TIMER buff 1s REPEAT\ncast armor\n#ENDTIMER`
2. Wait 3+ seconds
3. **Expected:** Multiple "cast armor" commands sent at ~1s intervals

### E2E04: Dynamic Target Selection
**Setup:**
- Variable: target (profile variable, default empty)

**Test:**
1. Set target="" (empty)
2. Alias pattern: "k"
3. Replacement: `#if ${target} == \"\"\n#SET target orc\n#endif\nkill ${target}`
4. User types: "k"
5. **Expected:** "SET target orc" executed, "kill orc" command sent

---

## 8. Error Scenario Tests

### ERR01: Invalid Syntax Handling
**Input:** `#IF hp < 30` (missing ${})
**Expected:** Parser error or condition evaluates to false

### ERR02: Missing Variable Handling
**Input:** `${nonexistent} > 5` (variable not defined)
**Expected:** Treats as empty string or 0, evaluates to false

### ERR03: Timer Command Inside Timer Block
**Input:** `#TIMER test 5s\n#TIMER nested 1s\n#ENDTIMER\n#ENDTIMER`
**Expected:** Nested timer parsed or error reported

---

## Test Execution Checklist

- [ ] All Parser tests pass
- [ ] All Syntax Validation tests pass
- [ ] All Condition Parser tests pass
- [ ] All Evaluation Engine tests pass
- [ ] All #IF/#ELSE/#ENDIF tests pass
- [ ] All Variable Resolution tests pass
- [ ] All #SET Persistence tests pass
- [ ] All System Variable Protection tests pass
- [ ] All Session Variable tests pass
- [ ] All Timer Creation tests pass
- [ ] All #CANCEL tests pass
- [ ] All Safety Control tests pass
- [ ] All Trigger Integration tests pass
- [ ] All Alias Integration tests pass
- [ ] All Timer Integration tests pass
- [ ] All End-to-End tests pass
- [ ] All Error Scenario tests pass

---

## Notes

- Tests marked as "Not done" in task file cannot be executed due to no test runner configured
- Manual testing required for full QA verification
- Build must succeed before QA testing can proceed
