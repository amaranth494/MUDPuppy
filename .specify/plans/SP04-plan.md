# SP04 Plan — Per-Connection Profiles & Input Customization

> **Spec ID:** SP04  
> **Status:** Draft  
> **Branch:** sp04-connection-profiles  
> **Constitutional Reference:** Constitution VIII (Deployment Governance), Constitution X (Pre-Push Build Requirements)  

---

## Overview

This plan implements SP04: Per-Connection Profiles & Input Customization. The spec introduces a per-connection profile layer with keybindings and settings, establishing the foundation for future automation features.

---

## Architecture Notes

> **Constitutional Note:** Per Constitution X, all staging pushes must be preceded by `npm ci && npm run build` (frontend) and `go build ./...` (backend).

### Database

- New `profiles` table with 1:1 relationship to `connections`
- JSONB columns for `keybindings` and `settings`
- Cascade delete from connections
- **pgcrypto extension required** for `gen_random_uuid()`

### Backend

- New `/api/profiles` endpoints (GET, PUT)
- Profile auto-creation in same transaction as connection
- Validation for JSONB structures
- Max binding count enforcement (50)

### Frontend

- Input interception layer in terminal component
- Keybinding lookup before command submission
- **Single `submitCommand(source, text)` function** for both typing and keybindings
- Settings panel in profile modal
- No localStorage usage

---

## Phase 0: Branch Creation (SP04PH00)

### Objectives

- Create feature branch from staging
- Commit spec document
- Verify CI passes

### Steps

1. **T00: Create feature branch**
   ```bash
   git checkout -b sp04-connection-profiles staging
   ```

2. **T01: Verify branch created correctly**
   ```bash
   git log --oneline -3
   ```

3. **T02: Verify spec document committed**
   ```bash
   git diff HEAD~1 --name-only | grep SP04
   ```

4. **T03: Push branch to origin**
   ```bash
   git push origin sp04-connection-profiles
   ```

5. **T04: Verify CI passes**
   - Check GitHub Actions run for sp04-connection-profiles
   - Ensure build passes before proceeding

---

## Phase 1: Migration + Profile Schema (SP04PH01)

### Objectives

- Create database migration for profiles table
- Verify schema applied correctly
- Test cascade delete behavior
- Migrate legacy connection data

### Steps

6. **T05: Create migration script** (`migrations/006_create_profiles.up.sql`)
   - Create `profiles` table
   - Add foreign key constraints
   - Add unique constraint on `connection_id`
   - Add indexes on `user_id` and `connection_id`

7. **T06: Create down migration** (`migrations/006_create_profiles.down.sql`)
   - Drop `profiles` table

8. **T07: Apply migration locally**
   ```bash
   # Using migrate tool or direct SQL execution
   ```

9. **T08: Verify schema**
   - Check table exists
   - Check constraints applied
   - Test cascade behavior (connection delete → profile delete)

10. **T09: Legacy data migration**
    - Create data migration for existing connections
    - Insert profile rows with default JSONB values
    - See SP04 Section 4.6 for SQL logic
    - **Verify pgcrypto extension installed** (for gen_random_uuid)

11. **T10: Verify staging database schema**
    - Verify staging database schema before merging
    - Ensure no schema mismatch between local and staging

---

## Phase 2: Backend CRUD + Validation (SP04PH02)

### Objectives

- Create profile store functions
- Implement API endpoints
- Add validation logic
- **Ensure transactional binding with connection creation**

### Steps

12. **T11: Create profile store** (`internal/store/profile.go`)
    - `CreateProfile(userID, connectionID string) (*Profile, error)`
    - `GetProfile(userID, profileID string) (*Profile, error)`
    - `GetProfileByConnection(userID, connectionID string) (*Profile, error)`
    - `UpdateProfile(userID, profileID string, updates *ProfileUpdate) (*Profile, error)`
    - `DeleteProfile(userID, profileID string) error`

13. **T12: Add profile model** (`internal/store/profile.go`)
    - Profile struct with JSONB tags
    - ProfileUpdate struct for partial updates

14. **T13: Create profiles handler** (`internal/profiles/handler.go`)
    - GET `/api/profiles/:id` - get profile
    - PUT `/api/profiles/:id` - update profile

15. **T14: Add validation logic**
    - Max 50 keybindings
    - Max command length 500 chars
    - Valid key format (basic validation)
    - Settings schema validation
    - Max scrollback 10000, min 100
    - Reject unknown JSONB fields
    - Reject nested objects in keybindings
    - Require string→string keybindings

16. **T15: Register routes** (`cmd/server/main.go`)
    - Add profile routes to router
    - Add auth middleware

17. **T16: Build and test backend**
    ```bash
    go build ./...
    ```

18. **T17: Push to staging**
    ```bash
    git push origin sp04-connection-profiles:staging
    ```

---

## Phase 3: Input Interception Layer (SP04PH03)

### Objectives

- Create keyboard event handler
- Implement keybinding lookup
- Create command dispatch flow

### Steps

19. **T18: Create input hook** (`frontend/src/hooks/useInputInterceptor.ts`)
    - Listen for keyboard events in terminal
    - Check for keybinding match
    - Return matched command or null

20. **T19: Integrate with terminal component** (`frontend/src/components/Terminal.tsx`)
    - Add input interceptor hook
    - Modify onSubmit handler to check keybindings first

21. **T20: Add modal lock check**
    - Ensure keybindings don't fire when modal active
    - Reuse SP03 input lock pattern

22. **T21: Add connection state check**
    - Don't dispatch keybindings when disconnected
    - Handle edge cases gracefully

23. **T22: Test input interception**
    - Verify keypress captured
    - Verify keybinding lookup works
    - Verify fallback to normal input

24. **T23: Push to staging**
    ```bash
    git push origin sp04-connection-profiles:staging
    ```

---

## Phase 4: Keybinding Engine (SP04PH04)

### Objectives

- Implement keybinding dispatch
- Connect to WebSocket
- Handle dispatch response

### Steps

25. **T24: Create keybinding service** (`frontend/src/services/keybindings.ts`)
    - Fetch profile keybindings on connect
    - Cache in session state
    - Lookup function

26. **T25: Implement dispatch logic**
    - Convert key event to binding key
    - Send command via WebSocket
    - Match dispatch model (same as typed command)
    - **Must use single submitCommand(source, text) function**
    - Both keybinding and typing handlers route through same function

27. **T26: Add rate limiting compliance**
    - Ensure keybindings respect server rate limits
    - No bypass of existing limits
    - Server rate limiting is authoritative

28. **T27: Add key repeat prevention**
    - Implement keydown/keyup tracking
    - Prevent auto-repeat on key hold
    - Require keyup before re-trigger

29. **T28: Test keybinding dispatch**
    - Map F1 to test command
    - Verify command sent via WebSocket
    - Verify response displayed correctly

30. **T29: Push to staging**
    ```bash
    git push origin sp04-connection-profiles:staging
    ```

---

## Phase 5: Settings Implementation (SP04PH05)

### Objectives

- Implement scrollback limit
- Implement echo input setting
- Implement timestamp input echo (locally-echoed commands only)
- Implement word wrap

### Steps

31. **T30: Add settings to profile**
    - Update Profile type with settings fields
    - Add default values

32. **T31: Implement scrollback limit** (`frontend/src/components/Terminal.tsx`)
    - Read scrollback_limit from profile
    - Apply to xterm instance on connect
    - **Enforce min 100, max 10000**

33. **T32: Implement echo input**
    - Add local echo logic
    - Toggle based on settings.echo_input

34. **T33: Implement timestamp input echo**
    - Add timestamp prefix to locally-echoed commands only
    - Must NOT timestamp server output
    - Hook into input echo pipeline, not xterm write pipeline
    - Frontend-only rendering
    - **Clarification:** This setting timestamps user input when echoed locally. Server output remains untimestamped for MVP.

35. **T34: Implement word wrap**
    - Configure xterm wrap mode
    - Toggle based on settings.word_wrap

36. **T35: Test all settings**
    - Verify each setting applies correctly
    - Verify settings persist across sessions

37. **T36: Push to staging**
    ```bash
    git push origin sp04-connection-profiles:staging
    ```

---

## Phase 6: Profile UI Modal (SP04PH06)

### Objectives

- Create profile settings modal
- Add keybinding editor
- Add settings toggles

### Steps

38. **T37: Create ProfileModal component** (`frontend/src/components/ProfileModal.tsx`)
    - Reuse SP03 modal container
    - Add profile form

39. **T38: Add keybinding list view**
    - Display existing keybindings
    - Show key → command mapping

40. **T39: Add keybinding add/edit**
    - Input for key combination
    - Input for command
    - Validation feedback

41. **T40: Add keybinding delete**
    - Remove button per binding
    - Confirmation not required (MVP)

42. **T41: Add settings toggles**
    - Scrollback limit input
    - Echo input checkbox
    - Timestamp output checkbox
    - Word wrap checkbox

43. **T42: Connect to API**
    - Fetch profile on modal open
    - Save profile on submit

44. **T43: Test profile modal**
    - Open from Connections Hub → Edit
    - Verify all fields work
    - Verify save persists

45. **T44: Push to staging**
    ```bash
    git push origin sp04-connection-profiles:staging
    ```

---

## Phase 7: QA Phase (SP04PH07)

### Objectives

- Validate all SP04 requirements
- Test edge cases
- Verify no regressions

### Steps

46. **T45: Profile auto-creation test**
    - Create new connection
    - Verify profile auto-created
    - Verify 1:1 relationship

47. **T46: Keybinding functionality test**
    - Add keybinding
    - Trigger keybinding
    - Verify command dispatched

48. **T47: Keybinding modal lock test**
    - Open modal
    - Press mapped key
    - Verify no dispatch

49. **T48: Keybinding collision resolution test**
    - Test F1 vs Ctrl+F1 precedence
    - **Most specific match wins**
    - Test case: Bind F1 to "score" and Ctrl+F1 to "help", verify each fires correct command

50. **T49: Settings persistence test**
    - Set scrollback limit
    - Logout
    - Login
    - Verify setting persisted

51. **T50: No localStorage test**
    - Check Application tab
    - Verify no profile data in localStorage

52. **T51: No mid-session profile mutation test**
    - Edit profile while connected
    - Verify behavior does NOT change until reconnect
    - Keybindings remain unchanged until reconnect

53. **T52: Rate limiting test**
    - Rapid keybinding presses
    - Verify rate limited (429 or dropped)

54. **T53: Regression tests**
    - Connection CRUD still works
    - Session management still works
    - No console errors

55. **T54: QA sign-off**
    - Review test results
    - Mark SP04PH07 complete

---

## Phase 8: Merge & Close (SP04PH08)

### Objectives

- Final staging validation
- Production deployment
- Git lifecycle closure

### Steps

56. **T55: Final staging push**
    ```bash
    git push origin sp04-connection-profiles:staging
    ```

57. **T56: Verify staging**
    - Test in incognito
    - Verify hashed assets
    - No 404s
    - No console errors
    - No credential leakage

58. **T57: Promote to master**
    ```bash
    git push origin staging:master
    ```

59. **T58: Verify production**
    - Test production deployment
    - Verify all features work

60. **T59: Close branch**
    - Delete feature branch (optional)
    - Update spec status to Closed

---

## Summary

| Phase | Tasks | Description |
|-------|-------|-------------|
| SP04PH00 | T00-T04 | Branch creation |
| SP04PH01 | T05-T10 | Migration + schema |
| SP04PH02 | T11-T17 | Backend CRUD |
| SP04PH03 | T18-T23 | Input interception |
| SP04PH04 | T24-T29 | Keybinding engine |
| SP04PH05 | T30-T36 | Settings implementation |
| SP04PH06 | T37-T44 | Profile UI modal |
| SP04PH07 | T45-T54 | QA validation |
| SP04PH08 | T55-T59 | Merge & close |

**Total Tasks:** 59

---

## Dependencies

- SP03 (Persistent App Shell) - Must be complete
- Railway deployment - For staging/production
- Existing WebSocket infrastructure - For command dispatch
- pgcrypto extension - For UUID generation in migrations
