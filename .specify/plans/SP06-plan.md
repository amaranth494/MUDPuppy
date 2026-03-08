# SP06 — Documentation & UX Refinement Plan

> **Plan ID:** SP06-plan  
> **Spec Reference:** SP06 — Documentation & UX Refinement  
> **Status:** Draft  
> **Constitutional Reference:** Constitution II, Constitution III  

---

## Phase Overview

| Phase | Focus | Description |
|-------|-------|-------------|
| SP06PH00 | Setup | Create branch, extend schema (no migration) |
| SP06PH01 | Help System Backend | Help content data, search API |
| SP06PH02 | Help Frontend | Help page UI with sections |
| SP06PH03 | Contextual Guidance | Add inline help to editors |
| SP06PH04 | Navigation | Connection Settings in nav when connected |
| SP06PH05 | Templates | Automation example templates |
| SP06PH06 | Ordering | Reorder controls for aliases/triggers |
| SP06PH07 | Status Indicators | Circuit breaker UI banner |
| SP06PH08 | Command History | Up/down arrow recall |

---

## Phase 0: Setup (SP06PH00)

### Objectives

- Create feature branch
- No schema changes (ordering via array position)
- No database migration required

### Steps

1. **T01: Create feature branch**
   - Branch from staging
   - Name: `sp06-docs-ux-refinement`

2. **T02: No schema changes needed**
   - Ordering via array position, no field required

3. **T03: No frontend type changes needed**
   - Array order is the source of truth

---

## Phase 1: Help System Backend (SP06PH01)

### Objectives

- Create help content files in repo
- Implement read-only API over help content
- Serve help via versioned endpoints

### Steps

1. **T01: Define help content structure**
   - Create help content JSON schema
   - Define all 9 section structures
   - Plan file locations: /help/*.json

2. **T02: Create help content files**
   - Create /help/getting-started.json
   - Create /help/connecting.json
   - Create /help/terminal.json
   - Create /help/keybindings.json

3. **T03: Create help content (continued)**
   - Create /help/aliases.json
   - Create /help/triggers.json
   - Create /help/variables.json
   - Create /help/safety.json
   - Create /help/troubleshooting.json

4. **T04: Implement help API endpoints**
   - Backend loads JSON files at startup
   - `GET /api/v1/help` — list all sections
   - `GET /api/v1/help/:slug` — get section content
   - `GET /api/v1/help/search?q=...` — substring search
   - No database, no write endpoints

5. **T05: Expose automation status in frontend**
   - Add automation status to SessionContext
   - Frontend reads paused/reason/canResume from context
   - No backend endpoint required

---

## Phase 2: Help Frontend (SP06PH02)

### Objectives

- Transform HelpPage to full documentation hub
- Add navigation sidebar
- Implement search UI

### Steps

1. **T02T01: Redesign HelpPage layout**
   - Add sidebar navigation
   - Add search box
   - Create section content areas

2. **T02T02: Implement help content display**
   - Fetch help content from API
   - Render all 9 sections
   - Add smooth scroll navigation

3. **T02T03: Implement help search**
   - Add search input component
   - Filter sections by query
   - Highlight matching text

4. **T02T04: Add examples section**
   - Create examples page in Help
   - Add alias template library
   - Add trigger template library

5. **T02T05: Style Help page**
   - Match existing design system
   - Add responsive behavior
   - Ensure accessibility

---

## Phase 3: Contextual Guidance (SP06PH03)

### Objectives

- Add helper text to Alias Editor
- Add notes to Trigger Editor
- Add syntax help to Environment Variables

### Steps

1. **T03T01: Update AliasEditor component**
   - Add helper text above alias list
   - Add tooltip guidance on pattern field

2. **T03T02: Update TriggerEditor component**
   - Add note about substring matching
   - Add tooltip on match field

3. **T03T03: Update EnvironmentPanel**
   - Add inline variable syntax help
   - Add example in variable editor

4. **T03T04: Update KeybindingEditor**
   - Add description of keybinding behavior
   - Add conflict warning

5. **T03T05: Test contextual guidance**
   - Verify all help text displays
   - Check tooltips work

---

## Phase 4: Navigation (SP06PH04)

### Objectives

- Add Connection Settings to nav when connected
- Auto-load active connection settings

### Steps

1. **T04T01: Update navigation state**
   - Track connection state in app
   - Show Connection Settings nav item when connected

2. **T04T02: Modify Sidebar component**
   - Add Connection Settings to nav items
   - Add conditional rendering based on session

3. **T04T03: Update ConnectionSettingsPage**
   - Auto-detect active connection
   - Load connection settings automatically
   - Show connection name in header

4. **T04T04: Test navigation flow**
   - Verify nav appears when connected
   - Verify nav hidden when disconnected
   - Verify settings load correctly

---

## Phase 5: Templates (SP06PH05)

### Objectives

- Create template library
- Add "Add Example" buttons
- Implement one-click import

### Steps

1. **T05T01: Define template data structure**
   - Create `src/templates.ts` (TypeScript)
   - Define 3+ alias templates
   - Define 3+ trigger templates

2. **T05T02: Add template UI to AliasEditor**
   - Add "Add Example" button
   - Open template modal on click
   - One-click import to editor

3. **T05T03: Add template UI to TriggerEditor**
   - Add "Add Example" button
   - Open template modal on click
   - One-click import to editor

4. **T05T04: Add templates to Help**
   - Display all templates in Examples section
   - Add copy button for each

5. **T05T05: Test template workflow**
   - Verify button appears
   - Verify modal opens
   - Verify import works

---

## Phase 6: Ordering (SP06PH06)

### Objectives

- Add Move Up/Down buttons
- Add priority display
- Persist order to server

### Steps

1. **T06T01: Add reorder buttons to AliasEditor**
   - Add Up/Down buttons per alias
   - Add order number display

2. **T06T02: Add reorder buttons to TriggerEditor**
   - Add Up/Down buttons per trigger
   - Add order number display

3. **T06T03: Integrate with existing PUT endpoints**
   - Use `PUT /aliases` and `PUT /triggers` endpoints
   - Send entire reordered array
   - Update local state optimistically

4. **T06T04: Test reordering**
   - Verify buttons work
   - Verify order persists
   - Verify priority affects execution

---

## Phase 7: Status Indicators (SP06PH07)

### Objectives

- Display circuit breaker banner
- Add Resume/Disable buttons
- Show status in sidebar

### Steps

1. **T07T01: Add automation status banner**
   - Create Banner component
   - Display when automation paused
   - Show reason message

2. **T07T02: Add action buttons**
   - Add "Resume Automation" button → POST /api/v1/sessions/:id/automation/resume
   - Add "Disable Automation" button → POST /api/v1/sessions/:id/automation/disable
   - Consume GET /api/v1/sessions/:id/automation/status

3. **T07T03: Add status to Sidebar**
   - Add warning icon when paused
   - Show tooltip with status

4. **T07T04: Test circuit breaker UI**
   - Verify banner appears
   - Verify buttons work
   - Verify sidebar indicator

---

## Phase 8: Command History (SP06PH08)

### Objectives

- Implement up/down arrow recall
- Add session-scoped history
- Handle empty state

### Steps

1. **T08T01: Add history state**
   - Track command history array
   - Track current history position

2. **T08T02: Hook into input handler**
   - Intercept Up Arrow key
   - Intercept Down Arrow key
   - Update input field value
   - **CRITICAL:** Recalled command must go through `submitCommand()`, never direct socket send

3. **T08T03: Handle empty/boundary cases**
   - No-op at start of history (Up)
   - No-op at end of history (Down)
   - Clear input at end (Down)

4. **T08T04: Test command history**
   - Verify Up recalls previous
   - Verify Down goes forward
   - Verify boundaries work

---

## Implementation Checklist

| Phase | Task | Description | Status |
|-------|------|-------------|--------|
| SP06PH00 | T01 | Create feature branch | [ ] |
| SP06PH00 | T02 | No schema changes (array order) | [ ] |
| SP06PH00 | T03 | No type changes needed | [ ] |
| SP06PH01 | T01 | Define help content structure | [ ] |
| SP06PH01 | T02 | Create help content files (1-4) | [ ] |
| SP06PH01 | T03 | Create help content files (5-9) | [ ] |
| SP06PH01 | T04 | Implement help API endpoints | [ ] |
| SP06PH01 | T05 | Add automation status API endpoints | [ ] |
| SP06PH02 | T01 | Redesign HelpPage | [ ] |
| SP06PH02 | T02 | Implement help display from API | [ ] |
| SP06PH02 | T03 | Implement client-side help search | [ ] |
| SP06PH02 | T04 | Add examples section | [ ] |
| SP06PH02 | T05 | Style Help page | [ ] |
| SP06PH03 | T01 | Update AliasEditor | [ ] |
| SP06PH03 | T02 | Update TriggerEditor | [ ] |
| SP06PH03 | T03 | Update EnvironmentPanel | [ ] |
| SP06PH03 | T04 | Update KeybindingEditor | [ ] |
| SP06PH03 | T05 | Test guidance | [ ] |
| SP06PH04 | T01 | Update nav state | [ ] |
| SP06PH04 | T02 | Modify Sidebar | [ ] |
| SP06PH04 | T03 | Update ConnectionSettingsPage | [ ] |
| SP06PH04 | T04 | Test navigation | [ ] |
| SP06PH05 | T01 | Define templates | [ ] |
| SP06PH05 | T02 | Template UI AliasEditor | [ ] |
| SP06PH05 | T03 | Template UI TriggerEditor | [ ] |
| SP06PH05 | T04 | Templates in Help | [ ] |
| SP06PH05 | T05 | Test templates | [ ] |
| SP06PH06 | T01 | Reorder AliasEditor | [ ] |
| SP06PH06 | T02 | Reorder TriggerEditor | [ ] |
| SP06PH06 | T03 | Reorder API | [ ] |
| SP06PH06 | T04 | Test reorder | [ ] |
| SP06PH07 | T01 | Status banner | [ ] |
| SP06PH07 | T02 | Action buttons | [ ] |
| SP06PH07 | T03 | Sidebar status | [ ] |
| SP06PH07 | T04 | Test circuit breaker | [ ] |
| SP06PH08 | T01 | History state | [ ] |
| SP06PH08 | T02 | Arrow key hooks | [ ] |
| SP06PH08 | T03 | Boundary cases | [ ] |
| SP06PH08 | T04 | Test history | [ ] |

---

## Notes

- Help content can be served statically initially; API is future-proofing
- Command history is session-scoped only (not persisted)
- Automation status endpoint should handle race conditions
- Templates should be easy to extend in the future
