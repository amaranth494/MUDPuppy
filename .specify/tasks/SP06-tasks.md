# SP06 — Documentation & UX Refinement Tasks

> **Task Tracker:** SP06  
> **Spec Reference:** SP06 — Documentation & UX Refinement  
> **Status:** Draft  
> **Total Tasks:** 66

---

## Phase 0: Setup (SP06PH00)

### SP06PH00T01 — Create feature branch

- [x] **Task:** Create branch `sp06-docs-ux-refinement` from staging
- **Commit:** "SP06PH00T01: Create feature branch"
- [x] **Status:** Completed

---

## Phase 1: Help System Backend (SP06PH01)

### SP06PH01T01 — Define help content structure

- [x] **Task:** Create help content JSON schema, define file locations (/help/*.json)
- **Acceptance:** Schema defined, 9 file paths planned
- **Commit:** "SP06PH01T01: Define help content schema and file structure"
- [x] **Status:** Completed

### SP06PH01T02 — Create help content files (sections 1-4)

- [x] **Task:** Create JSON files for Getting Started, Connecting, Terminal, Key Bindings
- **Acceptance:** 4 help files created in /help/ directory
- **Commit:** "SP06PH01T02: Create help content files (1-4)"
- [x] **Status:** Completed

### SP06PH01T03 — Create help content files (sections 5-9)

- [x] **Task:** Create JSON files for Aliases, Triggers, Variables, Safety, Troubleshooting
- **Acceptance:** All 9 help files created
- **Commit:** "SP06PH01T03: Create help content files (5-9)"
- [x] **Status:** Completed

### SP06PH01T04 — Implement help API endpoints

- [x] **Task:** Backend loads JSON at startup, serves via GET /api/v1/help, /api/v1/help/:slug
- **Acceptance:** Two endpoints functional, no DB
- **Commit:** "SP06PH01T04: Implement read-only help API endpoints"
- [x] **Status:** Completed

### SP06PH01T05 — Add automation status API

- [ ] **Task:** Expose automation status in SessionContext (client-side per SP06 spec)
- **Acceptance:** Frontend context has paused/reason/canResume
- **Commit:** "SP06PH01T05: Add automation status to frontend context"
- [ ] **Status:** Not done - per spec, automation status lives in frontend SessionContext, not backend

---

## Phase 2: Help Frontend (SP06PH02)

### SP06PH02T01 — Redesign HelpPage layout

- [x] **Task:** Transform HelpPage with sidebar nav, search box, content areas
- **Acceptance:** New layout matches design spec
- **Commit:** "SP06PH02T01: Redesign HelpPage layout"
- [x] **Status:** Completed

### SP06PH02T02 — Implement help content display

- [x] **Task:** Consume /api/v1/help and /api/v1/help/:slug, render all 9 sections
- **Acceptance:** All sections display correctly from API data
- **Commit:** "SP06PH02T02: Implement help content display from API"
- [x] **Status:** Completed

### SP06PH02T03 — Implement help search

- [x] **Task:** Fetch all help via GET /api/v1/help, search client-side
- **Acceptance:** Search filters locally, no backend call needed
- **Commit:** "SP06PH02T03: Implement client-side help search"
- [x] **Status:** Completed

### SP06PH02T04 — Add examples section

- [x] **Task:** Create Examples page in Help with template library
- **Acceptance:** Examples section added and ready for Phase 5 template integration
- **Commit:** "SP06PH02T04: Add examples section to Help"
- [x] **Status:** Completed

### SP06PH02T05 — Style Help page

- [x] **Task:** Apply design system styling to Help page
- **Acceptance:** Matches existing UI patterns
- **Commit:** "SP06PH02T05: Style Help page"
- [x] **Status:** Completed

---

## Phase 3: Contextual Guidance (SP06PH03)

### SP06PH03T01 — Update AliasEditor

- [x] **Task:** Add helper text above alias list and tooltip on pattern field
- **Acceptance:** Guidance text visible
- **Commit:** "SP06PH03T01: Add guidance to AliasEditor"
- [x] **Status:** Completed

### SP06PH03T02 — Update TriggerEditor

- [x] **Task:** Add note about substring matching and tooltip on match field
- **Acceptance:** Note visible
- **Commit:** "SP06PH03T02: Add guidance to TriggerEditor"
- [x] **Status:** Completed

### SP06PH03T03 — Update EnvironmentPanel

- [x] **Task:** Add inline syntax help and example in variable editor
- **Acceptance:** Help visible
- **Commit:** "SP06PH03T03: Add guidance to EnvironmentPanel"
- [x] **Status:** Completed

### SP06PH03T04 — Update KeybindingEditor

- [x] **Task:** Add description and conflict warning to keybinding editor
- **Acceptance:** Guidance visible
- **Commit:** "SP06PH03T04: Add guidance to KeybindingEditor"
- [x] **Status:** Completed

### SP06PH03T05 — Test contextual guidance

- [x] **Task:** Verify all guidance text displays correctly
- **Acceptance:** All tests pass
- **Commit:** "SP06PH03T05: Test contextual guidance"
- [x] **Status:** Completed

---

## Phase 4: Navigation (SP06PH04)

### SP06PH04T01 — Update navigation state

- [x] **Task:** Track connection state in app and expose to navigation
- **Acceptance:** State available for conditional rendering
- **Commit:** "SP06PH04T01: Update navigation state"
- [x] **Status:** Completed

### SP06PH04T02 — Modify Sidebar

- [x] **Task:** Add Connection Settings to nav items when connected
- **Acceptance:** Nav item appears when session active
- **Commit:** "SP06PH04T02: Add Connection Settings to Sidebar"
- [x] **Status:** Completed

### SP06PH04T03 — Update ConnectionSettingsPage

- [x] **Task:** Auto-detect active connection and load its settings
- **Acceptance:** Settings load for active connection
- **Commit:** "SP06PH04T03: Auto-load active connection settings"
- [x] **Status:** Completed

### SP06PH04T04 — Test navigation flow

- [x] **Task:** Verify nav appears/hides correctly and settings load
- **Acceptance:** All tests pass
- **Commit:** "SP06PH04T04: Test navigation flow"
- [x] **Status:** Completed

---

## Phase 5: Templates (SP06PH05)

### SP06PH05T01 — Define templates

- [x] **Task:** Create template data file with 3+ alias and 3+ trigger templates
- **Acceptance:** Templates defined in code
- **Commit:** "SP06PH05T01: Define automation templates"
- [x] **Status:** Completed

### SP06PH05T02 — Template UI in AliasEditor

- [x] **Task:** Add "Add Example" button and modal to AliasEditor
- **Acceptance:** UI opens and imports template
- **Commit:** "SP06PH05T02: Add template UI to AliasEditor"
- [x] **Status:** Completed

### SP06PH05T03 — Template UI in TriggerEditor

- [x] **Task:** Add "Add Example" button and modal to TriggerEditor
- **Acceptance:** UI opens and imports template
- **Commit:** "SP06PH05T03: Add template UI to TriggerEditor"
- [x] **Status:** Completed

### SP06PH05T04 — Templates in Help

- [x] **Task:** Display templates in Examples section with copy button
- **Acceptance:** All templates visible with copy functionality
- **Commit:** "SP06PH05T04: Add templates to Help examples"
- [x] **Status:** Completed

### SP06PH05T05 — Test template workflow

- [x] **Task:** Verify button, modal, and import work end-to-end
- **Acceptance:** All tests pass
- **Commit:** "SP06PH05T05: Test template workflow"
- [x] **Status:** Completed

---

## Phase 6: Ordering (SP06PH06)

### SP06PH06T01 — Reorder in AliasEditor

- [x] **Task:** Add Up/Down buttons and order number display to AliasEditor
- **Acceptance:** Reorder controls visible per alias
- **Commit:** "SP06PH06T01: Add reorder controls to AliasEditor"
- [x] **Status:** Completed

### SP06PH06T02 — Reorder in TriggerEditor

- [x] **Task:** Add Up/Down buttons and order number display to TriggerEditor
- **Acceptance:** Reorder controls visible per trigger
- **Commit:** "SP06PH06T02: Add reorder controls to TriggerEditor"
- [x] **Status:** Completed

### SP06PH06T03 — Integrate with existing PUT endpoints

- [x] **Task:** Use PUT /aliases and PUT /triggers to save reordered array
- **Acceptance:** Reordering persists to server
- **Commit:** "SP06PH06T03: Integrate reorder with existing PUT endpoints"
- [x] **Status:** Completed

### SP06PH06T04 — Test reordering

- [x] **Task:** Verify buttons, persistence, and execution priority
- **Acceptance:** All tests pass
- **Commit:** "SP06PH06T04: Test reordering functionality"
- [x] **Status:** VERIFIED (PH09)

---

## Phase 7: Status Indicators (SP06PH07)

### SP06PH07T01 — Status banner

- [x] **Task:** Create and display automation status banner when paused
- **Acceptance:** Banner shows with reason message
- **Commit:** "SP06PH07T01: Add automation status banner"
- [x] **Status:** Completed

### SP06PH07T02 — Action buttons

- [x] **Task:** Add Resume → POST /sessions/:id/automation/resume and Disable → POST /sessions/:id/automation/disable buttons
- **Acceptance:** Buttons call server-side API endpoints per Constitution I
- **Commit:** "SP06PH07T02: Add server-side API action buttons"
- [x] **Status:** Completed

> **Note on implementation:** Per SP06PH01T05 notes, automation status is client-side (frontend SessionContext). However, the Disable/Enable actions now persist to the server via ProfileSettings.automation_enabled, making it connection-persistent as required by the spec.

### SP06PH07T03 — Sidebar status

- [x] **Task:** Add warning icon to Sidebar when automation paused
- **Acceptance:** Icon visible with tooltip
- **Commit:** "SP06PH07T03: Add status to Sidebar"
- [x] **Status:** Completed

### SP06PH07T04 — Test circuit breaker UI

- [x] **Task:** Verify banner appears in PlayScreen and Connection Settings, verify buttons work, verify sidebar indicator
- **Acceptance:** All tests pass
- **Commit:** "SP06PH07T04: Test circuit breaker UI"
- [x] **Status:** VERIFIED (PH09)

---

## Phase 8: Command History (SP06PH08)

### SP06PH08T01 — History state

- [x] **Task:** Add command history array and position tracking to state
- **Acceptance:** State management in place
- **Commit:** "SP06PH08T01: Add command history state"
- [x] **Status:** Completed

### SP06PH08T02 — Arrow key hooks

- [x] **Task:** Intercept Up/Down arrows, update input field
- **Acceptance:** Arrows navigate history; recalled commands MUST go through submitCommand()
- **Commit:** "SP06PH08T02: Implement arrow key hooks with submitCommand pipeline"
- [x] **Status:** Completed

### SP06PH08T03 — Boundary cases

- [x] **Task:** Handle empty history and boundary positions
- **Acceptance:** No errors at boundaries
- **Commit:** "SP06PH08T03: Handle history boundary cases"
- [x] **Status:** Completed

### SP06PH08T04 — Test command history

- [x] **Task:** Verify up/down recall works end-to-end
- **Acceptance:** All tests pass
- **Commit:** "SP06PH08T04: Test command history"
- [x] **Status:** VERIFIED (PH09)

---

## Phase 9: QA Verification (SP06PH09)

> **Purpose:** Manual QA test scenarios for verification checklist. QA team to execute and check off each scenario.

[P] = PASS
[F] = FAIL
[I] = IGNORE

### SP06PH09T01 — Help System Tests

- [P] **SP06PH09T01a:** Navigate to Help page via sidebar - Verify sidebar loads and clicking Help navigates to /help
- **SP06PH09T01b:** Verify all 9 help sections accessible - Getting Started, Connecting, Terminal, Key Bindings, Aliases, Triggers, Variables, Safety, Troubleshooting
- **SP06PH09T01c:** Test help search - Type "alias" in search box, verify relevant results appear
- [P] **SP06PH09T01d:** Click help section from sidebar - Verify content loads without page refresh
- **SP06PH09T01e:** Check Examples section - Navigate to Examples, verify templates display with copy buttons

### SP06PH09T02 — Contextual Guidance Tests

- [P] **SP06PH09T02a:** Open AliasEditor - Verify helper text appears above alias list
- [P] **SP06PH09T02b:** Open TriggerEditor - Verify note about substring matching is visible
- [P] **SP06PH09T02c:** Open EnvironmentPanel variable editor - Verify inline syntax help and examples display
- [P] **SP06PH09T02d:** Open KeybindingEditor - Verify description and conflict warnings display

### SP06PH09T03 — Navigation Tests

- [P] **SP06PH09T03a:** Connect to a MUD server - Verify Connection Settings appears in sidebar nav
- [P] **SP06PH09T03b:** Click Connection Settings while connected - Verify settings page loads with active connection data
- [I] **SP06PH09T03c:** Disconnect - Verify Connection Settings disappears from sidebar - Settings is always visible in the navbar, per design.
- [P] **SP06PH09T03d:** Navigate to different pages while connected - Verify connection persists

### SP06PH09T04 — Status Indicator Tests

- [P] **SP06PH09T04a:** Trigger automation pause (e.g., too many rapid triggers) - Verify status banner appears in PlayScreen
- [P] **SP06PH09T04b:** Verify banner shows pause reason message
- [P] **SP06PH09T04c:** Click Resume button in banner - Verify automation resumes, banner disappears
- [P] **SP06PH09T04d:** Click Disable button in banner - Verify automation stays disabled
- [P] **SP06PH09T04e:** Navigate to Connection Settings while paused - Verify status indicator visible there too
- [P] **SP06PH09T04f:** Check Sidebar - Verify warning icon appears with tooltip when automation paused

### SP06PH09T05 — Command History Tests

- [P] **SP06PH09T05a:** Enter 5 unique commands in terminal (e.g., "look", "score", "inv", "who", "help")
- [P] **SP06PH09T05b:** Press Up arrow - Verify previous command appears in input field
- [P] **SP06PH09T05c:** Press Up arrow again - Verify earlier command appears
- [P] **SP06PH09T05d:** Press Down arrow - Verify next command appears
- [P] **SP06PH09T05e:** Press Down at end of history - Verify input clears or shows latest typed text
- [P] **SP06PH09T05f:** Press Enter on recalled command - Verify command executes (goes through full submit pipeline including alias expansion)
- [P] **SP06PH09T05g:** Verify no duplicates - Submit same command twice, recall with Up arrow - should only see command once in history
- [P] **SP06PH09T05h:** Press Up at beginning of history - Verify stays at oldest command (no wraparound errors)

### SP06PH09T06 — Integration Tests

- [I] **SP06PH09T06a:** Full user flow - Connect to MUD, create alias via template, test alias works, disconnect, reconnect, verify alias persists  - Templates are removed per operations. All other pieces work fine.
- [P] **SP06PH09T06b:** Cross-session persistence - Create automation (aliases/triggers/variables), disconnect, reconnect on different device/browser, verify automation loads
- [P] **SP06PH09T06c:** Multiple sessions - Open two sessions to different MUDs, verify automation status independent per session

---

## Sign-Off

| Role | Name | Date | Signature |
|------|------|------|------------|
| QA Lead | | | |
| Product Owner | | | |
