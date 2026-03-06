# SP05 — Automation Foundations (Aliases & Triggers) Plan

> **Plan ID:** SP05-plan  
> **Spec Reference:** SP05 — Automation Foundations  
> **Status:** Draft  
> **Constitutional Reference:** Constitution VIII, Constitution X  

---

## Phase Overview

| Phase | Focus | Description |
|-------|-------|-------------|
| SP05PH00 | Setup | Create branch, migration, backend types |
| SP05PH01 | Backend API | Profile automation endpoints |
| SP05PH02 | Frontend Types | TypeScript types for automation |
| SP05PH03 | Automation Engine | Core alias/trigger processing |
| SP05PH04 | Connection Settings UI | New workspace UI |
| SP05PH05 | Aliases UI | Alias CRUD interface |
| SP05PH06 | Triggers UI | Trigger CRUD interface |
| SP05PH07 | Environment UI | Variable CRUD interface |
| SP05PH08 | Integration | Connect automation to session |

---

## Phase 0: Setup (SP05PH00)

### Objectives
- Create feature branch
- Add automation fields to profile table
- Add backend types

### Steps

1. **T01: Create feature branch**
   - Branch from staging
   - Name: `sp05-automation-foundations`

2. **T02: Create migration 008**
   - Add aliases, triggers, variables JSONB columns
   - Set defaults to `{"items": []}`

3. **T03: Update profile store**
   - Add automation fields to Profile struct
   - Add JSONB handling for aliases, triggers, variables

4. **T04: Add profile automation handler**
   - Create handlers for GET/PUT aliases, triggers, environment

5. **T05: Register automation routes**
   - Add routes to profile router
   - Ensure ownership validation

---

## Phase 1: Backend API (SP05PH01)

### Objectives
- Implement REST endpoints for automation CRUD
- Enforce resource limits server-side

### Steps

1. **T01: Implement GET aliases endpoint**
   - Fetch aliases from profile
   - Return JSON structure

2. **T02: Implement PUT aliases endpoint**
   - Validate aliases structure
   - Enforce max_aliases limit (200)
   - Validate each alias has required fields
   - Update profile

3. **T03: Implement GET triggers endpoint**
   - Fetch triggers from profile
   - Return JSON structure

4. **T04: Implement PUT triggers endpoint**
   - Validate triggers structure
   - Enforce max_triggers limit (200)
   - Validate each trigger has required fields
   - Validate cooldown_ms is positive
   - Update profile

5. **T05: Implement GET environment endpoint**
   - Fetch variables from profile
   - Return JSON structure

6. **T06: Implement PUT environment endpoint**
   - Validate variables structure
   - Enforce max_variables limit (100)
   - Validate variable names (no ${} syntax)
   - Update profile

7. **T07: Add ownership validation**
   - Ensure connection_id belongs to user
   - SQL-level validation

---

## Phase 2: Frontend Types (SP05PH02)

### Objectives
- Define TypeScript types for automation
- Update API service types

### Steps

1. **T01: Define Alias type**
   ```typescript
   interface Alias {
     id: string;
     pattern: string;
     type: 'exact' | 'prefix';
     replacement: string;
     enabled: boolean;
   }
   ```

2. **T02: Define Trigger type**
   ```typescript
   interface Trigger {
     id: string;
     match: string;
     type: 'contains';
     action: string;
     cooldown_ms: number;
     enabled: boolean;
   }
   ```

3. **T03: Define Variable type**
   ```typescript
   interface Variable {
     id: string;
     name: string;
     value: string;
   }
   ```

4. **T04: Define Automation response types**
   ```typescript
   interface AliasesResponse {
     items: Alias[];
   }
   
   interface TriggersResponse {
     items: Trigger[];
   }
   
   interface VariablesResponse {
     items: Variable[];
   }
   ```

5. **T05: Update Profile type**
   - Add aliases, triggers, variables fields

6. **T06: Update API service**
   - Add methods for aliases, triggers, environment endpoints

---

## Phase 3: Automation Engine (SP05PH03)

### Objectives
- Implement alias processing
- Implement trigger processing
- Implement variable substitution
- Implement safeguards

### Steps

1. **T01: Create automation engine module**
   - `frontend/src/services/automation.ts`
   - Initialize with profile data

2. **T02: Implement alias processing**
   - Process input through alias engine
   - Handle exact and prefix types
   - Apply variable substitution
   - Prevent recursive expansion

3. **T03: Implement variable substitution**
   - Replace ${name} with variable values
   - Handle undefined variables gracefully

4. **T04: Implement trigger processing**
   - Monitor output lines
   - Check substring matches
   - Apply cooldown tracking

5. **T05: Implement internal expansion loop protection**
   - Enforce max_recursion_depth for explicit alias invocation (@alias)
   - Enforce max_commands_per_dispatch for single dispatch cycle (e.g., 200)
   - Detect cycles in @alias expansion via visited alias IDs

6. **T06: Implement disconnection handling**
   - Disable triggers when disconnected
   - Clear cooldown timers

7. **T07: Add automation to session context**
   - Load automation engine on connect
   - Wire into input/output pipeline

8. **T08: Integration with submitCommand**
   - Process input through alias engine before submit
   - Process output through trigger engine after receive

9. **T09: Implement command separation parser**
   - Split commands by semicolon (;)
   - Trim whitespace from each command
   - Ignore empty commands

10. **T10: Implement automation loop detection**
   - Track rolling history of dispatched commands
   - Record source, normalized_command, timestamp, trigger_id, alias_id
   - Detect rapid repetition of same command signature
   - Halt execution if loop detected

11. **T11: Implement command queue with backpressure**
   - Create client-side command queue
   - Process commands sequentially
   - Limit in-flight commands
   - Use server output as flow control signal
   - Implement fallback timeout for unresponsive servers

12. **T12: Implement trigger self-activation protection**
   - Store lastTriggerCommand with dispatch cycle ID
   - Track cycle ID when processing server output
   - Ignore trigger matches for output from same cycle
   - Clear lastTriggerCommand after cycle completes

13. **T13: Implement automation circuit breaker**
   - Pause automation dispatch when loop detected
   - Stop command queue processing
   - Display UI notification to user
   - Provide options: disable triggers, disable rule, resume manually

14. **T14: Implement memory safety safeguards**
   - Use streaming command parsing
   - Implement bounded queue structures
   - Add queue size limits with backpressure

---

## Phase 4: Connection Settings UI (SP05PH04)

### Objectives
- Add Connection Settings to sidebar
- Create workspace navigation

### Steps

1. **T01: Add Connection Settings to Sidebar**
   - Add menu item in sidebar
   - Icon: settings/gear

2. **T02: Create ConnectionSettingsPage**
   - `frontend/src/pages/ConnectionSettingsPage.tsx`
   - Route: `/connections/:id/settings`

3. **T03: Create Settings workspace layout**
   - Sidebar navigation within page
   - Sections: General, Key Bindings, Aliases, Triggers, Environment

4. **T04: Add General section**
   - Link to existing terminal settings
   - Keep as placeholder for now

5. **T05: Add Key Bindings section**
   - Link to existing keybindings UI
   - Keep as placeholder for now

6. **T06: Implement section navigation**
   - Click section to show relevant content
   - Maintain active section state

7. **T07: Remove legacy Profile modal**
   - Remove Profile modal component
   - Remove Connections > Profile menu item
   - Remove any related routing

---

## Phase 5: Aliases UI (SP05PH05)

### Objectives
- Create alias management interface
- CRUD operations for aliases

### Steps

1. **T01: Create AliasesPanel component**
   - Display list of aliases
   - Add/Edit/Delete buttons

2. **T02: Implement Add Alias form**
   - Pattern input
   - Type selector (exact/prefix)
   - Replacement input
   - Enabled toggle

3. **T03: Implement Edit Alias form**
   - Pre-fill form with existing values
   - Update on save

4. **T04: Implement Delete Alias**
   - Confirmation before delete
   - Remove from list

5. **T05: Add validation**
   - Pattern required
   - Replacement required
   - Unique pattern within type

6. **T06: Connect to API**
   - Fetch aliases on load
   - Save changes to server

7. **T07: Integrate with Connection Settings**
   - Embed AliasesPanel in Aliases section

---

## Phase 6: Triggers UI (SP05PH06)

### Objectives
- Create trigger management interface
- CRUD operations for triggers

### Steps

1. **T01: Create TriggersPanel component**
   - Display list of triggers
   - Add/Edit/Delete buttons

2. **T02: Implement Add Trigger form**
   - Match input
   - Type selector (contains only for SP05)
   - Action input
   - Cooldown input (ms)
   - Enabled toggle

3. **T03: Implement Edit Trigger form**
   - Pre-fill form with existing values
   - Update on save

4. **T04: Implement Delete Trigger**
   - Confirmation before delete
   - Remove from list

5. **T05: Add validation**
   - Match required
   - Action required
   - Cooldown must be positive

6. **T06: Connect to API**
   - Fetch triggers on load
   - Save changes to server

7. **T07: Integrate with Connection Settings**
   - Embed TriggersPanel in Triggers section

---

## Phase 7: Environment UI (SP05PH07)

### Objectives
- Create variable management interface
- CRUD operations for environment variables

### Steps

1. **T01: Create EnvironmentPanel component**
   - Display list of variables
   - Add/Edit/Delete buttons

2. **T02: Implement Add Variable form**
   - Name input
   - Value input

3. **T03: Implement Edit Variable form**
   - Pre-fill form with existing values
   - Update on save

4. **T04: Implement Delete Variable**
   - Confirmation before delete
   - Remove from list

5. **T05: Add validation**
   - Name required
   - Name cannot contain ${}
   - Unique name

6. **T06: Connect to API**
   - Fetch variables on load
   - Save changes to server

7. **T07: Integrate with Connection Settings**
   - Embed EnvironmentPanel in Environment section

---

## Phase 8: Integration (SP05PH08)

### Objectives
- Full end-to-end testing
- Bug fixes and polish

### Steps

1. **T01: Test alias flow**
   - Create alias
   - Type command
   - Verify transformation

2. **T02: Test trigger flow**
   - Create trigger
   - Receive matching output
   - Verify action fires

3. **T03: Test variable substitution**
   - Create variable
   - Use in alias
   - Verify substitution

4. **T04: Test trigger safeguards**
   - Rapid output triggers
   - Verify throttle works

5. **T05: Test disconnection handling**
   - Disconnect session
   - Verify triggers disabled

6. **T06: Test persistence**
   - Create automation
   - Reconnect
   - Verify automation loads

7. **T07: Run build checks**
   - `npm run build`
   - `go build ./...`

8. **T08: Push to staging**
   - Commit with proper message
   - Push branch
   - Create PR to staging

---

## Task Summary

| Phase | Tasks |
|-------|-------|
| SP05PH00 | T01-T05 |
| SP05PH01 | T01-T07 |
| SP05PH02 | T01-T06 |
| SP05PH03 | T01-T10 |
| SP05PH04 | T01-T07 |
| SP05PH05 | T01-T07 |
| SP05PH06 | T01-T07 |
| SP05PH07 | T01-T07 |
| SP05PH08 | T01-T08 |

**Total:** 58 tasks

---

## Constitutional Compliance

- **Constitution VIII:** Branch prefix `sp05-*`, follow deployment flow
- **Constitution X:** Run build checks before staging push
