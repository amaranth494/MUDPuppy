# SP06 — Documentation & UX Refinement Tasks

> **Task Tracker:** SP06  
> **Spec Reference:** SP06 — Documentation & UX Refinement  
> **Status:** Draft  
> **Total Tasks:** 40

---

## Phase 0: Setup (SP06PH00)

### SP06PH00T01 — Create feature branch

- [ ] **Task:** Create branch `sp06-docs-ux-refinement` from staging
- **Commit:** "SP06PH00T01: Create feature branch"
- [ ] **Status:** Pending

### SP06PH00T02 — No schema changes needed

- [ ] **Task:** Ordering via array position - no field required
- **Acceptance:** N/A - task skipped
- **Commit:** N/A
- [ ] **Status:** Skipped

### SP06PH00T03 — No type changes needed

- [ ] **Task:** Array order is the source of truth - no type changes
- **Acceptance:** N/A - task skipped
- **Commit:** N/A
- [ ] **Status:** Skipped

---

*Note: Ordering via array position. No schema, migration, or type changes needed.*

---

## Phase 1: Help System Backend (SP06PH01)

### SP06PH01T01 — Define help content structure

- [ ] **Task:** Create help content JSON schema, define file locations (/help/*.json)
- **Acceptance:** Schema defined, 9 file paths planned
- **Commit:** "SP06PH01T01: Define help content schema and file structure"
- [ ] **Status:** Pending

### SP06PH01T02 — Create help content files (sections 1-4)

- [ ] **Task:** Create JSON files for Getting Started, Connecting, Terminal, Key Bindings
- **Acceptance:** 4 help files created in /help/ directory
- **Commit:** "SP06PH01T02: Create help content files (1-4)"
- [ ] **Status:** Pending

### SP06PH01T03 — Create help content files (sections 5-9)

- [ ] **Task:** Create JSON files for Aliases, Triggers, Variables, Safety, Troubleshooting
- **Acceptance:** All 9 help files created
- **Commit:** "SP06PH01T03: Create help content files (5-9)"
- [ ] **Status:** Pending

### SP06PH01T04 — Implement help API endpoints

- [ ] **Task:** Backend loads JSON at startup, serves via GET /api/v1/help, /api/v1/help/:slug, /api/v1/help/search
- **Acceptance:** Three endpoints functional, no DB, substring search works
- **Commit:** "SP06PH01T04: Implement read-only help API endpoints"
- [ ] **Status:** Pending

### SP06PH01T05 — Add automation status API

- [ ] **Task:** Add GET /api/v1/sessions/:id/automation/status, POST /resume, POST /disable endpoints
- **Acceptance:** Server-side endpoints per Constitution I
- **Commit:** "SP06PH01T05: Add automation status API endpoints"
- [ ] **Status:** Pending

---

## Phase 2: Help Frontend (SP06PH02)

### SP06PH02T01 — Redesign HelpPage layout

- [ ] **Task:** Transform HelpPage with sidebar nav, search box, content areas
- **Acceptance:** New layout matches design spec
- **Commit:** "SP06PH02T01: Redesign HelpPage layout"
- [ ] **Status:** Pending

### SP06PH02T02 — Implement help content display

- [ ] **Task:** Consume /api/v1/help and /api/v1/help/:slug, render all 9 sections
- **Acceptance:** All sections display correctly from API data
- **Commit:** "SP06PH02T02: Implement help content display from API"
- [ ] **Status:** Pending

### SP06PH02T03 — Implement help search

- [ ] **Task:** Fetch all help via GET /api/v1/help, search client-side
- **Acceptance:** Search filters locally, no backend call needed
- **Commit:** "SP06PH02T03: Implement client-side help search"
- [ ] **Status:** Pending

### SP06PH02T04 — Add examples section

- [ ] **Task:** Create Examples page in Help with template library
- **Acceptance:** All templates displayed
- **Commit:** "SP06PH02T04: Add examples section to Help"
- [ ] **Status:** Pending

### SP06PH02T05 — Style Help page

- [ ] **Task:** Apply design system styling to Help page
- **Acceptance:** Matches existing UI patterns
- **Commit:** "SP06PH02T05: Style Help page"
- [ ] **Status:** Pending

---

## Phase 3: Contextual Guidance (SP06PH03)

### SP06PH03T01 — Update AliasEditor

- [ ] **Task:** Add helper text above alias list and tooltip on pattern field
- **Acceptance:** Guidance text visible
- **Commit:** "SP06PH03T01: Add guidance to AliasEditor"
- [ ] **Status:** Pending

### SP06PH03T02 — Update TriggerEditor

- [ ] **Task:** Add note about substring matching and tooltip on match field
- **Acceptance:** Note visible
- **Commit:** "SP06PH03T02: Add guidance to TriggerEditor"
- [ ] **Status:** Pending

### SP06PH03T03 — Update EnvironmentPanel

- [ ] **Task:** Add inline syntax help and example in variable editor
- **Acceptance:** Help visible
- **Commit:** "SP06PH03T03: Add guidance to EnvironmentPanel"
- [ ] **Status:** Pending

### SP06PH03T04 — Update KeybindingEditor

- [ ] **Task:** Add description and conflict warning to keybinding editor
- **Acceptance:** Guidance visible
- **Commit:** "SP06PH03T04: Add guidance to KeybindingEditor"
- [ ] **Status:** Pending

### SP06PH03T05 — Test contextual guidance

- [ ] **Task:** Verify all guidance text displays correctly
- **Acceptance:** All tests pass
- **Commit:** "SP06PH03T05: Test contextual guidance"
- [ ] **Status:** Pending

---

## Phase 4: Navigation (SP06PH04)

### SP06PH04T01 — Update navigation state

- [ ] **Task:** Track connection state in app and expose to navigation
- **Acceptance:** State available for conditional rendering
- **Commit:** "SP06PH04T01: Update navigation state"
- [ ] **Status:** Pending

### SP06PH04T02 — Modify Sidebar

- [ ] **Task:** Add Connection Settings to nav items when connected
- **Acceptance:** Nav item appears when session active
- **Commit:** "SP06PH04T02: Add Connection Settings to Sidebar"
- [ ] **Status:** Pending

### SP06PH04T03 — Update ConnectionSettingsPage

- [ ] **Task:** Auto-detect active connection and load its settings
- **Acceptance:** Settings load for active connection
- **Commit:** "SP06PH04T03: Auto-load active connection settings"
- [ ] **Status:** Pending

### SP06PH04T04 — Test navigation flow

- [ ] **Task:** Verify nav appears/hides correctly and settings load
- **Acceptance:** All tests pass
- **Commit:** "SP06PH04T04: Test navigation flow"
- [ ] **Status:** Pending

---

## Phase 5: Templates (SP06PH05)

### SP06PH05T01 — Define templates

- [ ] **Task:** Create template data file with 3+ alias and 3+ trigger templates
- **Acceptance:** Templates defined in code
- **Commit:** "SP06PH05T01: Define automation templates"
- [ ] **Status:** Pending

### SP06PH05T02 — Template UI in AliasEditor

- [ ] **Task:** Add "Add Example" button and modal to AliasEditor
- **Acceptance:** UI opens and imports template
- **Commit:** "SP06PH05T02: Add template UI to AliasEditor"
- [ ] **Status:** Pending

### SP06PH05T03 — Template UI in TriggerEditor

- [ ] **Task:** Add "Add Example" button and modal to TriggerEditor
- **Acceptance:** UI opens and imports template
- **Commit:** "SP06PH05T03: Add template UI to TriggerEditor"
- [ ] **Status:** Pending

### SP06PH05T04 — Templates in Help

- [ ] **Task:** Display templates in Examples section with copy button
- **Acceptance:** All templates visible with copy functionality
- **Commit:** "SP06PH05T04: Add templates to Help examples"
- [ ] **Status:** Pending

### SP06PH05T05 — Test template workflow

- [ ] **Task:** Verify button, modal, and import work end-to-end
- **Acceptance:** All tests pass
- **Commit:** "SP06PH05T05: Test template workflow"
- [ ] **Status:** Pending

---

## Phase 6: Ordering (SP06PH06)

### SP06PH06T01 — Reorder in AliasEditor

- [ ] **Task:** Add Up/Down buttons and order number display to AliasEditor
- **Acceptance:** Reorder controls visible per alias
- **Commit:** "SP06PH06T01: Add reorder controls to AliasEditor"
- [ ] **Status:** Pending

### SP06PH06T02 — Reorder in TriggerEditor

- [ ] **Task:** Add Up/Down buttons and order number display to TriggerEditor
- **Acceptance:** Reorder controls visible per trigger
- **Commit:** "SP06PH06T02: Add reorder controls to TriggerEditor"
- [ ] **Status:** Pending

### SP06PH06T03 — Integrate with existing PUT endpoints

- [ ] **Task:** Use PUT /aliases and PUT /triggers to save reordered array
- **Acceptance:** Reordering persists to server
- **Commit:** "SP06PH06T03: Integrate reorder with existing PUT endpoints"
- [ ] **Status:** Pending

### SP06PH06T04 — Test reordering

- [ ] **Task:** Verify buttons, persistence, and execution priority
- **Acceptance:** All tests pass
- **Commit:** "SP06PH06T04: Test reordering functionality"
- [ ] **Status:** Pending

---

## Phase 7: Status Indicators (SP06PH07)

### SP06PH07T01 — Status banner

- [ ] **Task:** Create and display automation status banner when paused
- **Acceptance:** Banner shows with reason message
- **Commit:** "SP06PH07T01: Add automation status banner"
- [ ] **Status:** Pending

### SP06PH07T02 — Action buttons

- [ ] **Task:** Add Resume → POST /sessions/:id/automation/resume and Disable → POST /sessions/:id/automation/disable buttons
- **Acceptance:** Buttons call server-side API endpoints per Constitution I
- **Commit:** "SP06PH07T02: Add server-side API action buttons"
- [ ] **Status:** Pending

### SP06PH07T03 — Sidebar status

- [ ] **Task:** Add warning icon to Sidebar when automation paused
- **Acceptance:** Icon visible with tooltip
- **Commit:** "SP06PH07T03: Add status to Sidebar"
- [ ] **Status:** Pending

### SP06PH07T04 — Test circuit breaker UI

- [ ] **Task:** Verify banner, buttons, and sidebar indicator work
- **Acceptance:** All tests pass
- **Commit:** "SP06PH07T04: Test circuit breaker UI"
- [ ] **Status:** Pending

---

## Phase 8: Command History (SP06PH08)

### SP06PH08T01 — History state

- [ ] **Task:** Add command history array and position tracking to state
- **Acceptance:** State management in place
- **Commit:** "SP06PH08T01: Add command history state"
- [ ] **Status:** Pending

### SP06PH08T02 — Arrow key hooks

- [ ] **Task:** Intercept Up/Down arrows, update input field
- **Acceptance:** Arrows navigate history; recalled commands MUST go through submitCommand()
- **Commit:** "SP06PH08T02: Implement arrow key hooks with submitCommand pipeline"
- [ ] **Status:** Pending

### SP06PH08T03 — Boundary cases

- [ ] **Task:** Handle empty history and boundary positions
- **Acceptance:** No errors at boundaries
- **Commit:** "SP06PH08T03: Handle history boundary cases"
- [ ] **Status:** Pending

### SP06PH08T04 — Test command history

- [ ] **Task:** Verify up/down recall works end-to-end
- **Acceptance:** All tests pass
- **Commit:** "SP06PH08T04: Test command history"
- [ ] **Status:** Pending
