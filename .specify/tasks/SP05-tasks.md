# SP05 — Automation Foundations (Aliases & Triggers) Tasks

> **Task Tracker:** SP05  
> **Spec Reference:** SP05 — Automation Foundations  
> **Status:** Draft  
> **Total Tasks:** 59

---

## Phase 0: Setup (SP05PH00)

### SP05PH00T01 — Create feature branch

- [ ] **Task:** Create branch `sp05-automation-foundations` from staging
- **Commit:** "SP05PH00T01: Create feature branch"
- [ ] **Status:** Pending

### SP05PH00T02 — Create migration 008

- [ ] **Task:** Add aliases, triggers, variables JSONB columns to profiles table
- **Acceptance:** Migration adds three columns with defaults `{"items": []}`
- **Commit:** "SP05PH00T02: Add automation fields migration"
- [ ] **Status:** Pending

### SP05PH00T03 — Update profile store

- [ ] **Task:** Add automation fields to Profile struct
- **Acceptance:** Go struct includes aliases, triggers, variables as jsonb
- **Commit:** "SP05PH00T03: Update profile store struct"
- [ ] **Status:** Pending

### SP05PH00T04 — Add profile automation handler

- [ ] **Task:** Create handlers for GET/PUT aliases, triggers, environment
- **Acceptance:** Handlers exist in profiles package
- **Commit:** "SP05PH00T04: Add automation handlers"
- [ ] **Status:** Pending

### SP05PH00T05 — Register automation routes

- [ ] **Task:** Add routes to profile router with ownership validation
- **Acceptance:** Routes respond to /api/v1/profiles/:connection_id/{aliases,triggers,environment}
- **Commit:** "SP05PH00T05: Register automation routes"
- [ ] **Status:** Pending

---

## Phase 1: Backend API (SP05PH01)

### SP05PH01T01 — Implement GET aliases endpoint

- [ ] **Task:** Fetch aliases from profile, return JSON structure
- **Acceptance:** Endpoint returns `{"items": [...]}`
- **Commit:** "SP05PH01T01: Implement GET aliases endpoint"
- [ ] **Status:** Pending

### SP05PH01T02 — Implement PUT aliases endpoint

- [ ] **Task:** Validate aliases, enforce max 200 limit, update profile
- **Acceptance:** Rejects if >200 aliases, accepts valid updates
- **Commit:** "SP05PH01T02: Implement PUT aliases endpoint"
- [ ] **Status:** Pending

### SP05PH01T03 — Implement GET triggers endpoint

- [ ] **Task:** Fetch triggers from profile, return JSON structure
- **Acceptance:** Endpoint returns `{"items": [...]}`
- **Commit:** "SP05PH01T03: Implement GET triggers endpoint"
- [ ] **Status:** Pending

### SP05PH01T04 — Implement PUT triggers endpoint

- [ ] **Task:** Validate triggers, enforce max 200 limit, validate cooldown_ms
- **Acceptance:** Rejects if >200 triggers or invalid cooldown, accepts valid updates
- **Commit:** "SP05PH01T04: Implement PUT triggers endpoint"
- [ ] **Status:** Pending

### SP05PH01T05 — Implement GET environment endpoint

- [ ] **Task:** Fetch variables from profile, return JSON structure
- **Acceptance:** Endpoint returns `{"items": [...]}`
- **Commit:** "SP05PH01T05: Implement GET environment endpoint"
- [ ] **Status:** Pending

### SP05PH01T06 — Implement PUT environment endpoint

- [ ] **Task:** Validate variables, enforce max 100 limit, validate names
- **Acceptance:** Rejects if >100 vars or invalid names, accepts valid updates
- **Commit:** "SP05PH01T06: Implement PUT environment endpoint"
- [ ] **Status:** Pending

### SP05PH01T07 — Add ownership validation

- [ ] **Task:** Ensure connection_id belongs to user at SQL level
- **Acceptance:** Users cannot access other users' automation
- **Commit:** "SP05PH01T07: Add ownership validation"
- [ ] **Status:** Pending

---

## Phase 2: Frontend Types (SP05PH02)

### SP05PH02T01 — Define Alias type

- [ ] **Task:** Create TypeScript Alias interface
- **Acceptance:** Includes id, pattern, type, replacement, enabled
- **Commit:** "SP05PH02T01: Define Alias type"
- [ ] **Status:** Pending

### SP05PH02T02 — Define Trigger type

- [ ] **Task:** Create TypeScript Trigger interface
- **Acceptance:** Includes id, match, type, action, cooldown_ms, enabled
- **Commit:** "SP05PH02T02: Define Trigger type"
- [ ] **Status:** Pending

### SP05PH02T03 — Define Variable type

- [ ] **Task:** Create TypeScript Variable interface
- **Acceptance:** Includes id, name, value
- **Commit:** "SP05PH02T03: Define Variable type"
- [ ] **Status:** Pending

### SP05PH02T04 — Define Automation response types

- [ ] **Task:** Create AliasesResponse, TriggersResponse, VariablesResponse
- **Acceptance:** Types wrap items arrays
- **Commit:** "SP05PH02T04: Define Automation response types"
- [ ] **Status:** Pending

### SP05PH02T05 — Update Profile type

- [ ] **Task:** Add aliases, triggers, variables to Profile type
- **Acceptance:** Profile includes automation fields
- **Commit:** "SP05PH02T05: Update Profile type"
- [ ] **Status:** Pending

### SP05PH02T06 — Update API service

- [ ] **Task:** Add methods for aliases, triggers, environment endpoints
- **Acceptance:** API service has getAliases, putAliases, etc.
- **Commit:** "SP05PH02T06: Update API service"
- [ ] **Status:** Pending

---

## Phase 3: Automation Engine (SP05PH03)

### SP05PH03T01 — Create automation engine module

- [ ] **Task:** Create `frontend/src/services/automation.ts`
- **Acceptance:** Module exports functions to process aliases and triggers
- **Commit:** "SP05PH03T01: Create automation engine module"
- [ ] **Status:** Pending

### SP05PH03T02 — Implement alias processing

- [ ] **Task:** Process input through alias engine, handle exact and prefix
- **Acceptance:** "l" → "look", "k goblin" → "kill goblin"
- **Commit:** "SP05PH03T02: Implement alias processing"
- [ ] **Status:** Pending

### SP05PH03T03 — Implement variable substitution

- [ ] **Task:** Replace ${name} with variable values
- **Acceptance:** "cast ${target}" with target=goblin → "cast goblin"
- **Commit:** "SP05PH03T03: Implement variable substitution"
- [ ] **Status:** Pending

### SP05PH03T04 — Implement trigger processing

- [ ] **Task:** Monitor output lines, check substring matches
- **Acceptance:** "You are hungry" triggers action
- **Commit:** "SP05PH03T04: Implement trigger processing"
- [ ] **Status:** Pending

### SP05PH03T05 — Implement internal expansion loop protection

- [ ] **Task:** Enforce max_recursion_depth for @alias chains, enforce max_commands_per_dispatch (200), detect cycles via visited alias IDs
- **Acceptance:** Internal expansion loops are detected and halted; MUD output-driven triggers fire indefinitely
- **Commit:** "SP05PH03T05: Implement internal expansion loop protection"
- [ ] **Status:** Pending

### SP05PH03T06 — Implement disconnection handling

- [ ] **Task:** Disable triggers when disconnected, clear timers
- **Acceptance:** No triggers fire after disconnect
- **Commit:** "SP05PH03T06: Implement disconnection handling"
- [ ] **Status:** Pending

### SP05PH03T07 — Add automation to session context

- [ ] **Task:** Load automation engine on connect, wire to input/output
- **Acceptance:** Automation active during session
- **Commit:** "SP05PH03T07: Add automation to session context"
- [ ] **Status:** Pending

### SP05PH03T08 — Integration with submitCommand

- [ ] **Task:** Process input through alias engine, output through trigger engine
- **Acceptance:** Full pipeline integration working
- **Commit:** "SP05PH03T08: Integration with submitCommand"
- [ ] **Status:** Pending

### SP05PH03T09 — Implement command separation parser

- [ ] **Task:** Split commands by semicolon (;), trim whitespace, ignore empty commands
- **Acceptance:** "look;inv;draw sword" produces three separate commands
- **Commit:** "SP05PH03T09: Implement command separation parser"
- [ ] **Status:** Pending

### SP05PH03T10 — Implement automation loop detection

- [ ] **Task:** Track rolling history of dispatched commands, detect rapid repetition of same command signature, halt if loop detected
- **Acceptance:** Detects and halts infinite automation loops
- **Commit:** "SP05PH03T10: Implement automation loop detection"
- [ ] **Status:** Pending

### SP05PH03T11 — Implement command queue with backpressure

- [ ] **Task:** Create client-side command queue, process sequentially, limit in-flight commands, use server output as flow control, implement fallback timeout
- **Acceptance:** Commands queue and dispatch with pacing; no flooding to server
- **Commit:** "SP05PH03T11: Implement command queue with backpressure"
- [ ] **Status:** Pending

### SP05PH03T12 — Implement trigger self-activation protection

- [ ] **Task:** Store lastTriggerCommand with dispatch cycle ID, track cycle when processing output, ignore triggers for output from same cycle, clear after cycle
- **Acceptance:** Triggers do not re-fire on their own output
- **Commit:** "SP05PH03T12: Implement trigger self-activation protection"
- [ ] **Status:** Pending

### SP05PH03T13 — Implement automation circuit breaker

- [ ] **Task:** Pause automation dispatch when loop detected, stop queue, show UI notification, provide disable/resume options
- **Acceptance:** User notified of loop, can resume manually
- **Commit:** "SP05PH03T13: Implement automation circuit breaker"
- [ ] **Status:** Pending

### SP05PH03T14 — Implement memory safety safeguards

- [ ] **Task:** Use streaming command parsing, bounded queue structures, add queue size limits
- **Acceptance:** Client does not allocate unbounded memory
- **Commit:** "SP05PH03T14: Implement memory safety safeguards"
- [ ] **Status:** Pending

---

## Phase 4: Connection Settings UI (SP05PH04)

### SP05PH04T01 — Add Connection Settings to Sidebar

- [ ] **Task:** Add menu item in sidebar with icon
- **Acceptance:** Sidebar shows "Connection Settings"
- **Commit:** "SP05PH04T01: Add Connection Settings to Sidebar"
- [ ] **Status:** Pending

### SP05PH04T02 — Create ConnectionSettingsPage

- [ ] **Task:** Create page component with route `/connections/:id/settings`
- **Acceptance:** Page renders at correct route
- **Commit:** "SP05PH04T02: Create ConnectionSettingsPage"
- [ ] **Status:** Pending

### SP05PH04T03 — Create Settings workspace layout

- [ ] **Task:** Sidebar navigation within page, sections list
- **Acceptance:** Shows General, Key Bindings, Aliases, Triggers, Environment
- **Commit:** "SP05PH04T03: Create Settings workspace layout"
- [ ] **Status:** Pending

### SP05PH04T04 — Add General section

- [ ] **Task:** Link to existing terminal settings
- **Acceptance:** Section present (placeholder OK)
- **Commit:** "SP05PH04T04: Add General section"
- [ ] **Status:** Pending

### SP05PH04T05 — Add Key Bindings section

- [ ] **Task:** Link to existing keybindings UI
- **Acceptance:** Section present (placeholder OK)
- **Commit:** "SP05PH04T05: Add Key Bindings section"
- [ ] **Status:** Pending

### SP05PH04T06 — Implement section navigation

- [ ] **Task:** Click section to show content, maintain active state
- **Acceptance:** Navigation works between sections
- **Commit:** "SP05PH04T06: Implement section navigation"
- [ ] **Status:** Pending

### SP05PH04T07 — Remove legacy Profile modal

- [ ] **Task:** Remove Profile modal component, Connections > Profile menu item, related routing
- **Acceptance:** Profile editing only exists under Connection Settings
- **Commit:** "SP05PH04T07: Remove legacy Profile modal"
- [ ] **Status:** Pending

---

## Phase 5: Aliases UI (SP05PH05)

### SP05PH05T01 — Create AliasesPanel component

- [ ] **Task:** Display list of aliases with buttons
- **Acceptance:** Shows all aliases for connection
- **Commit:** "SP05PH05T01: Create AliasesPanel component"
- [ ] **Status:** Pending

### SP05PH05T02 — Implement Add Alias form

- [ ] **Task:** Form with pattern, type, replacement, enabled
- **Acceptance:** Can create new alias
- **Commit:** "SP05PH05T02: Implement Add Alias form"
- [ ] **Status:** Pending

### SP05PH05T03 — Implement Edit Alias form

- [ ] **Task:** Pre-fill form, update on save
- **Acceptance:** Can edit existing alias
- **Commit:** "SP05PH05T03: Implement Edit Alias form"
- [ ] **Status:** Pending

### SP05PH05T04 — Implement Delete Alias

- [ ] **Task:** Confirm before delete, remove from list
- **Acceptance:** Can delete alias
- **Commit:** "SP05PH05T04: Implement Delete Alias"
- [ ] **Status:** Pending

### SP05PH05T05 — Add validation

- [ ] **Task:** Validate pattern and replacement required
- **Acceptance:** Shows errors for invalid input
- **Commit:** "SP05PH05T05: Add validation"
- [ ] **Status:** Pending

### SP05PH05T06 — Connect to API

- [ ] **Task:** Fetch on load, save changes to server
- **Acceptance:** Aliases persist to database
- **Commit:** "SP05PH05T06: Connect to API"
- [ ] **Status:** Pending

### SP05PH05T07 — Integrate with Connection Settings

- [ ] **Task:** Embed AliasesPanel in Aliases section
- **Acceptance:** Shows in Connection Settings page
- **Commit:** "SP05PH05T07: Integrate with Connection Settings"
- [ ] **Status:** Pending

---

## Phase 6: Triggers UI (SP05PH06)

### SP05PH06T01 — Create TriggersPanel component

- [ ] **Task:** Display list of triggers with buttons
- **Acceptance:** Shows all triggers for connection
- **Commit:** "SP05PH06T01: Create TriggersPanel component"
- [ ] **Status:** Pending

### SP05PH06T02 — Implement Add Trigger form

- [ ] **Task:** Form with match, type, action, cooldown, enabled
- **Acceptance:** Can create new trigger
- **Commit:** "SP05PH06T02: Implement Add Trigger form"
- [ ] **Status:** Pending

### SP05PH06T03 — Implement Edit Trigger form

- [ ] **Task:** Pre-fill form, update on save
- **Acceptance:** Can edit existing trigger
- **Commit:** "SP05PH06T03: Implement Edit Trigger form"
- [ ] **Status:** Pending

### SP05PH06T04 — Implement Delete Trigger

- [ ] **Task:** Confirm before delete, remove from list
- **Acceptance:** Can delete trigger
- **Commit:** "SP05PH06T04: Implement Delete Trigger"
- [ ] **Status:** Pending

### SP05PH06T05 — Add validation

- [ ] **Task:** Validate match, action required, cooldown positive
- **Acceptance:** Shows errors for invalid input
- **Commit:** "SP05PH06T05: Add validation"
- [ ] **Status:** Pending

### SP05PH06T06 — Connect to API

- [ ] **Task:** Fetch on load, save changes to server
- **Acceptance:** Triggers persist to database
- **Commit:** "SP05PH06T06: Connect to API"
- [ ] **Status:** Pending

### SP05PH06T07 — Integrate with Connection Settings

- [ ] **Task:** Embed TriggersPanel in Triggers section
- **Acceptance:** Shows in Connection Settings page
- **Commit:** "SP05PH06T07: Integrate with Connection Settings"
- [ ] **Status:** Pending

---

## Phase 7: Environment UI (SP05PH07)

### SP05PH07T01 — Create EnvironmentPanel component

- [ ] **Task:** Display list of variables with buttons
- **Acceptance:** Shows all variables for connection
- **Commit:** "SP05PH07T01: Create EnvironmentPanel component"
- [ ] **Status:** Pending

### SP05PH07T02 — Implement Add Variable form

- [ ] **Task:** Form with name and value inputs
- **Acceptance:** Can create new variable
- **Commit:** "SP05PH07T02: Implement Add Variable form"
- [ ] **Status:** Pending

### SP05PH07T03 — Implement Edit Variable form

- [ ] **Task:** Pre-fill form, update on save
- **Acceptance:** Can edit existing variable
- **Commit:** "SP05PH07T03: Implement Edit Variable form"
- [ ] **Status:** Pending

### SP05PH07T04 — Implement Delete Variable

- [ ] **Task:** Confirm before delete, remove from list
- **Acceptance:** Can delete variable
- **Commit:** "SP05PH07T04: Implement Delete Variable"
- [ ] **Status:** Pending

### SP05PH07T05 — Add validation

- [ ] **Task:** Validate name required, no ${} in name, unique name
- **Acceptance:** Shows errors for invalid input
- **Commit:** "SP05PH07T05: Add validation"
- [ ] **Status:** Pending

### SP05PH07T06 — Connect to API

- [ ] **Task:** Fetch on load, save changes to server
- **Acceptance:** Variables persist to database
- **Commit:** "SP05PH07T06: Connect to API"
- [ ] **Status:** Pending

### SP05PH07T07 — Integrate with Connection Settings

- [ ] **Task:** Embed EnvironmentPanel in Environment section
- **Acceptance:** Shows in Connection Settings page
- **Commit:** "SP05PH07T07: Integrate with Connection Settings"
- [ ] **Status:** Pending

---

## Phase 8: Integration (SP05PH08)

### SP05PH08T01 — Test alias flow

- [ ] **Task:** Create alias, type command, verify transformation
- **Acceptance:** Alias transforms input correctly
- **Commit:** "SP05PH08T01: Test alias flow"
- [ ] **Status:** Pending

### SP05PH08T02 — Test trigger flow

- [ ] **Task:** Create trigger, receive matching output, verify action fires
- **Acceptance:** Trigger responds to output
- **Commit:** "SP05PH08T02: Test trigger flow"
- [ ] **Status:** Pending

### SP05PH08T03 — Test variable substitution

- [ ] **Task:** Create variable, use in alias, verify substitution
- **Acceptance:** Variable substitution works
- **Commit:** "SP05PH08T03: Test variable substitution"
- [ ] **Status:** Pending

### SP05PH08T04 — Test trigger safeguards

- [ ] **Task:** Rapid output triggers, verify throttle works
- **Acceptance:** Throttle prevents runaway triggers
- **Commit:** "SP05PH08T04: Test trigger safeguards"
- [ ] **Status:** Pending

### SP05PH08T05 — Test disconnection handling

- [ ] **Task:** Disconnect session, verify triggers disabled
- **Acceptance:** No triggers fire after disconnect
- **Commit:** "SP05PH08T05: Test disconnection handling"
- [ ] **Status:** Pending

### SP05PH08T06 — Test persistence

- [ ] **Task:** Create automation, reconnect, verify loads
- **Acceptance:** Automation persists across sessions
- **Commit:** "SP05PH08T06: Test persistence"
- [ ] **Status:** Pending

### SP05PH08T07 — Run build checks

- [ ] **Task:** Run `npm run build` and `go build ./...`
- **Acceptance:** Both builds succeed
- **Commit:** "SP05PH08T07: Run build checks"
- [ ] **Status:** Pending

### SP05PH08T08 — Push to staging

- [ ] **Task:** Commit with message, push branch, create PR
- **Acceptance:** Code in staging environment
- **Commit:** "SP05PH08T08: Push to staging"
- [ ] **Status:** Pending

---

## Task Summary

| Phase | Tasks |
|-------|-------|
| SP05PH00 | T01-T05 |
| SP05PH01 | T01-T07 |
| SP05PH02 | T01-T06 |
| SP05PH03 | T01-T14 |
| SP05PH04 | T01-T07 |
| SP05PH05 | T01-T07 |
| SP05PH06 | T01-T07 |
| SP05PH07 | T01-T07 |
| SP05PH08 | T01-T08 |

**Total:** 61 tasks
