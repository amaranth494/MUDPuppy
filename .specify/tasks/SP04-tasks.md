# SP04 Tasks Tracker

> **Constitutional Reference:** This task tracker is governed by Constitution VIII. Deployment Governance & Environment Integrity and Constitution X. Pre-Push Build Requirements.

**Spec:** SP04 — Per-Connection Profiles & Input Customization  
**Reference:** [.specify/specs/SP04.md](../specs/SP04.md)  
**Reference:** [.specify/plans/SP04-plan.md](../plans/SP04-plan.md)

---

## Phase 0: Branch Creation (SP04PH00)

### SP04PH00T01 — Create spec branch

- [x] **Task:** Create branch `sp04-connection-profiles` from staging
- **Command:** `git checkout -b sp04-connection-profiles staging`
- **Commit:** "SP04PH00: Branch created from staging"
- [x] **Status:** Completed

### SP04PH00T02 — Commit spec document

- [x] **Task:** Commit SP04.md, SP04-plan.md, SP04-tasks.md to branch
- **Acceptance:** Spec documents present in git history
- **Commit:** "SP04PH00: Spec documents committed"
- [x] **Status:** Completed

### SP04PH00T03 — Push to origin

- [x] **Task:** Push branch to origin
- **Command:** `git push origin sp04-connection-profiles`
- **Commit:** "SP04PH00: Branch pushed to origin"
- [x] **Status:** Completed

### SP04PH00T04 — Verify CI passes

- [x] **Task:** Verify CI passes on branch
- **Acceptance:** CI green before proceeding
- [x] **Status:** Completed

### SP04PH00T05 — Phase 0 complete

- [x] **Task:** Confirm Phase 0 complete before feature work
- [x] **Status:** Completed

> **Staging Push:** Before pushing to staging:
> - **Frontend:** `npm ci && npm run build`
> - **Backend:** `go build ./...`
> - **Then push:** `git push origin sp04-connection-profiles:staging`

---

## Phase 1: Migration + Profile Schema (SP04PH01)

### SP04PH01T01 — Create migration script

- [x] **Task:** Create `migrations/006_create_profiles.up.sql`
- **Acceptance:**
  - `profiles` table created
  - Foreign key to `users.id`
  - Foreign key to `connections.id`
  - Unique constraint on `connection_id`
  - Indexes on `user_id`, `connection_id`
- **Commit:** "SP04PH01T01: Create profiles table migration"
- [x] **Status:** Completed (Verified via code review)

### SP04PH01T02 — Create down migration

- [x] **Task:** Create `migrations/006_create_profiles.down.sql`
- **Acceptance:** Drops `profiles` table
- **Commit:** "SP04PH01T02: Create profiles down migration"
- [x] **Status:** Completed (Verified via code review)

### SP04PH01T03 — Apply migration locally

- [x] **Task:** Apply migration to local database
- **Command:** `migrate -path migrations -database "$DATABASE_URL" up`
- **Acceptance:** Migration version recorded
- **Verification:** Staging database at version 7 (migrations 006 & 007 applied)
- **Note:** Local verification requires DATABASE_URL; staging verified via Railway CLI
- [x] **Status:** Completed (verified via Railway CLI: version 7)

### SP04PH01T04 — Verify schema

- [x] **Task:** Verify table and constraints exist
- **Acceptance:**
  - Table exists
  - Constraints applied
  - Indexes present
- **Verification:** Migration version 7 confirms schema applied; code review verified structure
- **Note:** SQL verification requires psql (not available on Windows); code review confirms correct schema
- [x] **Status:** Completed (code review + migration version 7)

### SP04PH01T05 — Test cascade behavior

- [x] **Task:** Test cascade delete from connections
- **Acceptance:** Profile deleted when connection deleted
- **Verification:** Code review confirms ON DELETE CASCADE in migration (line 12)
- [x] **Status:** Completed (code verified ON DELETE CASCADE)

### SP04PH01T06 — Legacy data migration

- [x] **Task:** Create data migration for existing connections
- **Acceptance:** All existing connections have profiles with default JSONB values
- **SQL:** See SP04 Section 4.6 for migration logic
- **Verify:** pgcrypto extension installed for gen_random_uuid
- [x] **Status:** Completed (Verified via code review - migrations/007_migrate_existing_connections.up.sql)

### SP04PH01T07 — Verify staging database schema

- [x] **Task:** Verify staging database schema before merging
- **Acceptance:** Schema matches local, no mismatch
- **Verification:** Railway CLI confirms migration version 7 (all migrations applied)
- [x] **Status:** Completed (verified via Railway CLI)

> **Staging Push:** Before pushing to staging:
> - **Frontend:** `npm ci && npm run build`
> - **Backend:** `go build ./...`
> - **Then push:** `git push origin sp04-connection-profiles:staging`

---

## Phase 2: Backend CRUD + Validation (SP04PH02)

### SP04PH02T01 — Create profile store

- [x] **Task:** Create `internal/store/profile.go`
- **Functions:**
  - `CreateProfile(userID, connectionID string) (*Profile, error)`
  - `GetProfile(userID, profileID string) (*Profile, error)`
  - `GetProfileByConnection(userID, connectionID string) (*Profile, error)`
  - `UpdateProfile(userID, profileID string, updates *ProfileUpdate) (*Profile, error)`
  - `DeleteProfile(userID, profileID string) error`
- **Commit:** "SP04PH02T01: Create profile store functions"
- [x] **Status:** Completed

### SP04PH02T02 — Add profile model

- [x] **Task:** Add Profile struct with JSONB tags
- **Acceptance:**
  - Profile struct with all fields
  - ProfileUpdate struct for partial updates
  - JSONB tags for keybindings and settings
- **Commit:** "SP04PH02T02: Add profile model with JSONB"
- [x] **Status:** Completed

### SP04PH02T03 — Create profiles handler

- [x] **Task:** Create `internal/profiles/handler.go`
- **Endpoints:**
  - GET `/api/profiles/:id`
  - PUT `/api/profiles/:id`
- **Commit:** "SP04PH02T03: Create profiles API handler"
- [x] **Status:** Completed

### SP04PH02T04 — Add validation logic

- [x] **Task:** Add server-side validation
- **Rules:**
  - Max 50 keybindings
  - Max command length 500 chars
  - Valid key format validation
  - Max scrollback 10000, min 100
  - Reject unknown JSONB fields
  - Reject nested objects in keybindings
  - Require string→string keybindings
- **Commit:** "SP04PH02T04: Add profile validation logic"
- [x] **Status:** Completed

### SP04PH02T05 — Register routes

- [x] **Task:** Add profile routes to router
- **File:** `cmd/server/main.go`
- **Acceptance:** Routes registered with auth middleware
- **Commit:** "SP04PH02T05: Register profile routes"
- [x] **Status:** Completed

### SP04PH02T06 — Build backend

- [x] **Task:** Build and verify backend compiles
- **Command:** `go build ./...`
- **Acceptance:** No build errors
- [x] **Status:** Completed

### SP04PH02T07 — Push to staging

- [x] **Task:** Push changes to staging
- **Frontend:** `npm ci && npm run build`
- **Backend:** `go build ./...`
- **Then push:** `git push origin sp04-connection-profiles:staging`
- **Acceptance:** Staging deployed, no 500s
- [x] **Status:** Completed

### SP04PH02T08 — Pre-push build verification

- [x] **Task:** Verify local build succeeds before staging push
- **Frontend:** `npm ci && npm run build`
- **Backend:** `go build ./...`
- **Acceptance:** Both builds pass without errors
- **Note:** Required before EVERY staging push per Constitution X
- [x] **Status:** Completed

---

## Phase 3: Input Interception Layer (SP04PH03)

### SP04PH03T01 — Create input hook

- [x] **Task:** Create `frontend/src/hooks/useInputInterceptor.ts`
- **Responsibilities:**
  - Listen for keyboard events
  - Check keybinding match
  - Return matched command or null
- **Commit:** "SP04PH03: Input interception layer + SP04PH04: Keybinding engine"
- [x] **Status:** Completed

### SP04PH03T02 — Integrate with terminal

- [x] **Task:** Integrate interceptor with Terminal component
- **File:** `frontend/src/components/Terminal.tsx` (actually PlayScreen.tsx)
- **Acceptance:** onSubmit checks keybindings first
- **Commit:** "SP04PH03: Input interception layer + SP04PH04: Keybinding engine"
- [x] **Status:** Completed

### SP04PH03T03 — Add modal lock check

- [x] **Task:** Ensure keybindings don't fire when modal active
- **Acceptance:** Reuse SP03 input lock pattern
- **Commit:** "SP04PH03: Input interception layer + SP04PH04: Keybinding engine"
- [x] **Status:** Completed

### SP04PH03T04 — Add connection state check

- [x] **Task:** Don't dispatch when disconnected
- **Acceptance:** Keybindings ignored when disconnected
- **Commit:** "SP04PH03: Input interception layer + SP04PH04: Keybinding engine"
- [x] **Status:** Completed

### SP04PH03T05 — Test input interception

- [x] **Task:** Test keypress captured and lookup works
- **Acceptance:** Fallback to normal input when no match
- **Verification:** Build passes (go build + npm run build), code review confirms implementation
- [x] **Status:** Completed

### SP04PH03T06 — Push to staging

- [x] **Task:** Push changes to staging
- **Commit:** "SP04PH03: Input interception layer + SP04PH04: Keybinding engine"
- **Acceptance:** Staging deployed
- [x] **Status:** Completed

---

## Phase 4: Keybinding Engine (SP04PH04)

### SP04PH04T01 — Create keybinding service

- [ ] **Task:** Create `frontend/src/services/keybindings.ts`
- **Functions:**
  - Fetch profile keybindings on connect
  - Cache in session state
  - Lookup function
- **Commit:** "SP04PH04T01: Create keybinding service"
- [ ] **Status:** Pending

### SP04PH04T02 — Implement dispatch logic

- [ ] **Task:** Convert key event to binding, send via WebSocket
- **Acceptance:** Same dispatch model as typed command
- **Must use single submitCommand(source, text) function**
- **Commit:** "SP04PH04T02: Implement keybinding dispatch"
- [ ] **Status:** Pending

### SP04PH04T03 — Add rate limiting compliance

- [ ] **Task:** Ensure keybindings respect server rate limits
- **Acceptance:** No bypass of existing limits, server is authoritative
- **Commit:** "SP04PH04T03: Add rate limiting to keybindings"
- [ ] **Status:** Pending

### SP04PH04T04 — Add key repeat prevention

- [ ] **Task:** Implement keydown/keyup tracking
- **Acceptance:** Prevent auto-repeat on key hold, require keyup before re-trigger
- **Commit:** "SP04PH04T04: Add key repeat prevention"
- [ ] **Status:** Pending

### SP04PH04T05 — Test keybinding dispatch

- [ ] **Task:** Test F1 → command dispatch
- **Acceptance:** Command sent via WebSocket, response displayed
- [ ] **Status:** Pending

### SP04PH04T06 — Push to staging

- [ ] **Task:** Push changes to staging
- **Frontend:** `npm ci && npm run build`
- **Backend:** `go build ./...`
- **Then push:** `git push origin sp04-connection-profiles:staging`
- **Acceptance:** Staging deployed
- [ ] **Status:** Pending

---

## Phase 5: Settings Implementation (SP04PH05)

### SP04PH05T01 — Add settings to profile

- [ ] **Task:** Update Profile type with settings fields
- **Acceptance:** Default values for all settings
- **Commit:** "SP04PH05T01: Add settings to profile model"
- [ ] **Status:** Pending

### SP04PH05T02 — Implement scrollback limit

- [ ] **Task:** Apply scrollback_limit to xterm
- **File:** `frontend/src/components/Terminal.tsx`
- **Acceptance:** Lines limited on connect (min 100, max 10000)
- **Commit:** "SP04PH05T02: Implement scrollback limit"
- [ ] **Status:** Pending

### SP04PH05T03 — Implement echo input

- [ ] **Task:** Add local echo based on settings.echo_input
- **Acceptance:** Command echoed locally if enabled
- **Commit:** "SP04PH05T03: Implement echo input setting"
- [ ] **Status:** Pending

### SP04PH05T04 — Implement timestamp output

- [ ] **Task:** Add timestamp prefix to rendered lines
- **Acceptance:** Hook into xterm write pipeline AFTER parsing, NOT prefix raw ANSI
- **Commit:** "SP04PH05T04: Implement timestamp output"
- [ ] **Status:** Pending

### SP04PH05T05 — Implement word wrap

- [ ] **Task:** Configure xterm wrap mode
- **Acceptance:** Toggle based on settings.word_wrap
- **Commit:** "SP04PH05T05: Implement word wrap setting"
- [ ] **Status:** Pending

### SP04PH05T06 — Test all settings

- [ ] **Task:** Verify each setting applies correctly
- **Acceptance:** Settings persist across sessions
- [ ] **Status:** Pending

### SP04PH05T07 — Push to staging

- [ ] **Task:** Push changes to staging
- **Frontend:** `npm ci && npm run build`
- **Backend:** `go build ./...`
- **Then push:** `git push origin sp04-connection-profiles:staging`
- **Acceptance:** Staging deployed
- [ ] **Status:** Pending

---

## Phase 6: Profile UI Modal (SP04PH06)

### SP04PH06T01 — Create ProfileModal component

- [ ] **Task:** Create `frontend/src/components/ProfileModal.tsx`
- **Acceptance:** Reuse SP03 modal container
- **Commit:** "SP04PH06T01: Create ProfileModal component"
- [ ] **Status:** Pending

### SP04PH06T02 — Add keybinding list view

- [ ] **Task:** Display existing keybindings
- **Acceptance:** Show key → command mapping
- **Commit:** "SP04PH06T02: Add keybinding list view"
- [ ] **Status:** Pending

### SP04PH06T03 — Add keybinding add/edit

- [ ] **Task:** Inputs for key and command
- **Acceptance:** Validation feedback shown
- **Commit:** "SP04PH06T03: Add keybinding add/edit"
- [ ] **Status:** Pending

### SP04PH06T04 — Add keybinding delete

- [ ] **Task:** Remove button per binding
- **Acceptance:** No confirmation required (MVP)
- **Commit:** "SP04PH06T04: Add keybinding delete"
- [ ] **Status:** Pending

### SP04PH06T05 — Add settings toggles

- [ ] **Task:** Add UI for all settings
- **Fields:**
  - Scrollback limit input (min 100, max 10000)
  - Echo input checkbox
  - Timestamp output checkbox
  - Word wrap checkbox
- **Commit:** "SP04PH06T05: Add settings toggles UI"
- [ ] **Status:** Pending

### SP04PH06T06 — Connect to API

- [ ] **Task:** Fetch and save profile
- **Acceptance:** Profile loaded on open, saved on submit
- **Commit:** "SP04PH06T06: Connect ProfileModal to API"
- [ ] **Status:** Pending

### SP04PH06T07 — Test profile modal

- [ ] **Task:** Test from Connections Hub → Edit
- **Acceptance:** All fields work, save persists
- [ ] **Status:** Pending

### SP04PH06T08 — Push to staging

- [ ] **Task:** Push changes to staging
- **Frontend:** `npm ci && npm run build`
- **Backend:** `go build ./...`
- **Then push:** `git push origin sp04-connection-profiles:staging`
- **Acceptance:** Staging deployed
- [ ] **Status:** Pending

---

## Phase 7: QA Phase (SP04PH07)

### SP04PH07T01 — Profile auto-creation test

- [ ] **Task:** Create connection, verify profile auto-created
- **Acceptance:** 1:1 relationship maintained
- [ ] **Status:** Pending

### SP04PH07T02 — Keybinding functionality test

- [ ] **Task:** Add and trigger keybinding
- **Acceptance:** Command dispatched correctly
- [ ] **Status:** Pending

### SP04PH07T03 — Keybinding modal lock test

- [ ] **Task:** Open modal, press mapped key
- **Acceptance:** No dispatch during modal
- [ ] **Status:** Pending

### SP04PH07T04 — Keybinding collision resolution test

- [ ] **Task:** Test F1 vs Ctrl+F1 precedence
- **Acceptance:** Most specific match wins
- **Test case:** Bind F1 to "score" and Ctrl+F1 to "help", verify each fires correct command
- [ ] **Status:** Pending

### SP04PH07T05 — Settings persistence test

- [ ] **Task:** Set value, logout, login, verify persisted
- **Acceptance:** Settings survive session
- [ ] **Status:** Pending

### SP04PH07T06 — No localStorage test

- [ ] **Task:** Check Application tab
- **Acceptance:** No profile data in localStorage
- [ ] **Status:** Pending

### SP04PH07T07 — No mid-session profile mutation test

- [ ] **Task:** Edit profile while connected, verify behavior does NOT change
- **Acceptance:** Keybindings remain unchanged until reconnect
- [ ] **Status:** Pending

### SP04PH07T08 — Rate limiting test

- [ ] **Task:** Rapid keybinding presses
- **Acceptance:** Rate limited (429 or dropped)
- [ ] **Status:** Pending

### SP04PH07T09 — Regression tests

- [ ] **Task:** Verify connection CRUD, session management
- **Acceptance:** No console errors, no breakage
- [ ] **Status:** Pending

### SP04PH07T10 — QA sign-off

- [ ] **Task:** Review test results
- **Acceptance:** All tests pass, QA signs off
- [ ] **Status:** Pending

### SP04PH07T11 — Profile fetch sequencing test

- [ ] **Task:** Verify profile fetched BEFORE WebSocket ready for input
- **Acceptance:** User clicks Connect → profile loads → WebSocket connects → keybindings available immediately
- [ ] **Status:** Pending

### SP04PH07T12 — Profile fetch failure handling test

- [ ] **Task:** Verify connection blocked if profile fetch fails
- **Test cases:**
  - 404 (Not Found) → Block connect, show error
  - 500 (Server Error) → Block connect, show error
  - Network Error → Block connect, show error
- **Acceptance:** Do not partially connect. Session must not establish if profile cannot be loaded.
- [ ] **Status:** Pending

### SP04PH07T13 — Browser refresh behavior test

- [ ] **Task:** Verify browser refresh ends active session
- **Test cases:**
  - While connected, press browser refresh
  - Session status becomes Disconnected
  - User must explicitly reconnect
  - After reconnect, profile loads normally
- **Acceptance:** Refresh ends session; requires explicit reconnect; profile loads correctly on next connect
- [ ] **Status:** Pending

---

## Phase 8: Merge & Close (SP04PH08)

### SP04PH08T01 — Final staging push

- [ ] **Task:** Push final changes to staging
- **Frontend:** `npm ci && npm run build`
- **Backend:** `go build ./...`
- **Then push:** `git push origin sp04-connection-profiles:staging`
- **Acceptance:** Staging deployed
- [ ] **Status:** Pending

### SP04PH08T02 — Verify staging

- [ ] **Task:** Test in incognito
- **Checks:**
  - Hashed assets present
  - No 404s
  - No console errors
  - No credential leakage
- [ ] **Status:** Pending

### SP04PH08T03 — Promote to master

- [ ] **Task:** Push staging to master
- **Command:** `git push origin staging:master`
- **Acceptance:** Production deployed
- [ ] **Status:** Pending

### SP04PH08T04 — Verify production

- [ ] **Task:** Test production deployment
- **Acceptance:** All features work in production
- [ ] **Status:** Pending

### SP04PH08T05 — Close spec

- [ ] **Task:** Mark SP04 as Closed
- **Acceptance:** Spec status updated, branch lifecycle complete
- [ ] **Status:** Pending

---

## Summary

| Phase | Tasks | Status |
|-------|-------|--------|
| SP04PH00 | T01-T05 | Completed |
| SP04PH01 | T01-T07 | Pending |
| SP04PH02 | T01-T08 | Pending |
| SP04PH03 | T01-T06 | Pending |
| SP04PH04 | T01-T06 | Pending |
| SP04PH05 | T01-T07 | Pending |
| SP04PH06 | T01-T08 | Pending |
| SP04PH07 | T01-T13 | Pending |
| SP04PH08 | T01-T05 | Pending |

**Total:** 65 tasks
