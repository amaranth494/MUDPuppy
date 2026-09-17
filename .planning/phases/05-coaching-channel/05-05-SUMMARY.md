---
phase: 05-coaching-channel
plan: 05
subsystem: api
tags: [go, gemini, websocket, driver, session-manager, ai-chatter]

# Dependency graph
requires:
  - phase: 05-coaching-channel
    provides: "05-01's PausedByOwner/ConnectionLost waiting reasons and the [AI-PLAYER]-adjacent notifier wiring; 05-02's fail-closed reviewer verdict and ordinary-play exceptions; 05-04's per-user AI send rate limiter and RateLimitPerSecond AI setting"
  - phase: 04-continuous-play
    provides: "buildSystemInstruction/buildReviewSystemInstruction/untrustedDataParagraph/promptContext, clampBullets/renderBullets/neutraliseLine/wrapQuestMemory/wrapSessionMemory/wrapModelReasoning, tryReserveCall, the epoch-carrying EngageHook/DisengageHook pair on Manager, ResolveAISettings, and internal/store/decisions.go/quests.go/transcripts.go's store-per-concern shape"
provides:
  - "migration 015: conversation_lines (append-only, per game session) and game_sessions.coaching_suggestions (used by plan 05-06)"
  - "internal/store/conversation.go: ConversationStore with AppendChatLine, ConversationFor (login-scoped), ConversationForSession (Logs page), RecentConversation (bounded, newest-first SQL reversed to oldest-first in Go)"
  - "internal/gemini/client.go: Client.Chat and ChatAnswer{Reply string}, a third structured-output entry point, failing closed (KindMalformed) on a missing/empty reply"
  - "internal/driver/chat.go: HandleChat, a sibling of the decision path with no stint identity, its own in-flight guard, a shared call-cap reservation, and buildChatSystemInstruction carrying everything AI-player knows plus a bounded <CONVERSATION> tail"
  - "driver.go: Models.Chat, Decisions.ListForConnection, the Conversation collaborator + SetConversation, ChatEvent, and Notifier.NotifyChat (with a ChatNotifierFunc adapter alongside NotifierFunc)"
  - "internal/session/websocket.go: MsgTypeChat, ChatPayload, WSMessage.Chat, PushChat; case MsgTypeChat applies the ingress rate limiter but never calls applyWheelGrab, never requires a live game connection, never writes to clientToMUD"
  - "internal/session/manager.go: ChatHook/SetChatHook/FireChatHook, invoked directly by the websocket read loop (it has no autopilot state transition of its own to fire from)"
  - "cmd/server/main.go: conversationStore construction, SetChatHook(aiDriver.HandleChat), aiDriver.SetConversation, and NotifyChat wired through to wsHandler.PushChat"
affects: [05-06-ai-chatter-coaching-channel, 05-11-security-review-and-close]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "A driver-package sibling decision path (HandleChat) that reuses every prompt-assembly, untrusted-delimiting and call-cap primitive an existing decision path already proved, without inheriting that path's stint/epoch identity"
    - "A Manager-owned hook with no state transition of its own (ChatHook), invoked directly by the caller via an exported Fire method rather than fired internally from a locked state-mutating method, because the trigger is an inbound message, not a transition"
    - "Composing two single-method function-adapter types (NotifierFunc + ChatNotifierFunc) via anonymous-struct embedding to satisfy one widened two-method interface with one concrete value"

key-files:
  created:
    - migrations/015_add_coaching_and_conversation.up.sql
    - migrations/015_add_coaching_and_conversation.down.sql
    - internal/store/conversation.go
    - internal/store/conversation_test.go
    - internal/driver/chat.go
    - internal/driver/chat_test.go
  modified:
    - internal/gemini/client.go
    - internal/gemini/client_test.go
    - internal/driver/driver.go
    - internal/driver/driver_test.go
    - internal/driver/corpus_live_test.go
    - internal/session/websocket.go
    - internal/session/websocket_test.go
    - internal/session/manager.go
    - cmd/server/main.go

key-decisions:
  - "ChatAnswer carries exactly one field, Reply -- no push/withdraw fields ship in this plan (plan 05-06 extends the schema and the type when it adds that behaviour); grep -c \"push\" internal/gemini/client.go returns 0"
  - "AI-player's recent decisions read for chat reuse Decisions.ListForConnection with limit 0 (the store's own default page, oldest-first) and keep only the last maxRecentDecisionsForChat=5 in Go, because ListForConnection's own contract orders oldest-first with the limit taken from the front -- requesting a small limit directly would return the OLDEST decisions, not the most recent ones"
  - "The chat hook lives on internal/session.Manager (ChatHook/SetChatHook/chatHook field), not on WebSocketHandler, mirroring EngageHook/DisengageHook's storage location exactly (05-05-02's read_first pointed at Manager's pair as the type/setter convention to copy); unlike those two, ChatHook has no autopilot state transition to fire from, so it is invoked through a new exported FireChatHook(userID, connectionID, message) that takes its own lock, called directly by the websocket read loop's case MsgTypeChat: in a goroutine"
  - "Inbound chat text travels in WSMessage.Data (mirroring MsgTypeData's own convention for free text); the new WSMessage.Chat *ChatPayload field is populated on every OUTBOUND push only (PushChat), mirroring how Decision *AIDecisionPayload is outbound-only on MsgTypeAI -- this is Claude's Discretion since 05-CONTEXT.md is silent on which field carries the inbound text"
  - "The game text window travels inside AI-chatter's system instruction (wrapped in <GAME_TEXT>), not as the Chat call's user-text argument -- the user-text slot is reserved for the owner's own current message, the one live instruction in the call, per this plan's context note"
  - "Widening Notifier to two methods (NotifyDecision, NotifyChat) broke NotifierFunc's single-method satisfaction of the interface at cmd/server/main.go's one call site; fixed there (not deferred to a later plan) by adding a sibling ChatNotifierFunc adapter and composing both via anonymous-struct embedding -- documented below as a Rule 3 pull-forward, matching 05-04's precedent for a task's own compile dependency on a file outside its nominal list"
  - "TestWebSocket_ChatIsNotAWheelGrab lives in internal/session/websocket_test.go (not internal/driver/chat_test.go) and exercises Manager.FireChatHook directly rather than a live *websocket.Conn round trip: this package has no harness for one anywhere (TestWheelGrabSourceRule, the closest existing precedent, tests applyWheelGrab the same way), and clientToMUD is a channel local to HandleWebSocket's own call frame, not a Manager field, so \"nothing written to the MUD channel\" is proven by source inspection (the case block contains no reference to clientToMUD) rather than by a runtime assertion"

patterns-established:
  - "A prompt-context struct built once per call, threading a caller-supplied ID plus a handful of nil-safe collaborator reads (activeQuestBullets/sessionMemoryBullets reused verbatim) into a single builder function, kept easy to extend with one more field later (chatPromptContext's own doc comment names the field plan 05-06 will add)"

requirements-completed: [REQ-coaching-chat]

# Metrics
duration: ~37min
completed: 2026-09-17
---

# Phase 5 Plan 05: AI-Chatter Conversation, Storage and Chat Transport Summary

**HandleChat answers every owner chat message with a Gemini call carrying everything AI-player knows (goal, conduct rules, Quest/Session Memory, recent game text, recent decisions and a bounded conversation tail), stored in a new conversation_lines table and delivered over a new MsgTypeChat websocket channel that never touches the wheel-grab or the game.**

## Performance

- **Duration:** ~37 min (commit-to-commit, base 8f43a20 to 97ae816; first commit c4b69f6 at 12:28:54-07:00, last commit 97ae816 at 12:51:57-07:00, plus read/verification time before the first commit)
- **Started:** 2026-09-17T12:15:00-07:00 (approximate, worktree base commit)
- **Completed:** 2026-09-17T12:51:57-07:00
- **Tasks:** 3
- **Files modified:** 15 (6 created, 9 modified)

## Accomplishments

- The owner can hold a conversation with AI-chatter and get a real answer: `HandleChat` calls `Client.Chat` once per owner message, with a system instruction carrying the session goal, the profile's conduct rules/approach guidance/Never-issue list, the recent game text window, Quest Memory, Session Memory, AI-player's recent decisions (wrapped as the other model's own account), and a bounded `<CONVERSATION>` tail — proven by `TestHandleChat_ReplyCarriesEverythingAIPlayerKnows`.
- A follow-up message makes sense: the last 10 conversation lines travel in the prompt, oldest first, read before the owner's current message is stored so it never appears twice — proven by `TestHandleChat_PromptCarriesRecentConversation`'s four subtests (5 lines all appear, only the last 10 of 15 appear, the current message never leaks into the block, a nil Conversation collaborator still replies with no block at all).
- AI-chatter works identically whether autopilot is On, Off or Waiting, consulting no epoch or stint machinery at all (`epochForCallCount` stays 0) — proven by `TestHandleChat_WorksInEveryAutopilotState`.
- Every reply draws on the same per-user call cap a decision draws on; when the cap is spent, the owner's message is still stored and a locked, unbracketed notice is shown, with no model call, no decision row and no disengage — proven by `TestHandleChat_CapReached`.
- AI-chatter reaches nothing but the conversation: zero Dispatch calls, zero memory writes, zero quest writes, zero disengages across a successful chat, a model error and a cap-reached chat — proven by `TestHandleChat_TouchesNothingElse`.
- Hostile text in the game window, a memory bullet and a stored conversation line cannot forge a marker (including a fake `<COACHING>` tag from a channel this plan doesn't even use yet) — proven by the comparative-count assertion in `TestHandleChat_UntrustedBlocksCannotForgeAMarker`.
- A second message sent before the first reply lands is refused with a system notice and produces no second model call; a message over 1000 characters is rejected before any model call or stored row — proven by `TestHandleChat_InFlightGuard` and `TestHandleChat_MessageTooLong`.
- Typing into the chat box is never a game command and never a wheel-grab, in every state including disconnected — proven by `TestWebSocket_ChatIsNotAWheelGrab`.
- The conversation is stored beside the decisions it was about: migration 015's `conversation_lines` table, an append-only, per-game-session store with a login-scoped read, a per-session read for the Logs page, and the bounded newest-first-then-reversed read the chat prompt uses — proven by `TestConversationStoreSQL`.

## Task Commits

1. **Task 05-05-01: The conversation has somewhere to live and the model has a third way in** - `c4b69f6` (feat)
2. **Task 05-05-02: AI-chatter answers, knowing everything AI-player knows and able to touch none of it** - `85fd837` (feat)
3. **Task 05-05-03: A chat message travels the connection the panel already has, and is never mistaken for a game command** - `97ae816` (feat)

_No plan-metadata commit is made by this executor: per this worktree's instructions, STATE.md/ROADMAP.md are the orchestrator's to update after all wave agents complete._

## Files Created/Modified

- `migrations/015_add_coaching_and_conversation.up.sql` / `.down.sql` - `conversation_lines` (BIGSERIAL id, `game_session_id` FK ON DELETE CASCADE, `seq BIGINT` ordering column with a unique `(game_session_id, seq)` index, `speaker` CHECK'd to owner/chatter/system, `text`, `created_at`) and `game_sessions.coaching_suggestions JSONB NOT NULL DEFAULT '[]'::jsonb`
- `internal/store/conversation.go` - `ConversationStore`: `AppendChatLine` (transaction-wrapped, assigns the next `seq` itself), `ConversationFor` (login-scoped, oldest-first), `ConversationForSession` (per-session, oldest-first), `RecentConversation` (newest-first SQL + `LIMIT`, reversed to oldest-first in Go)
- `internal/store/conversation_test.go` - `TestConversationStoreSQL`: SQL-constant shape assertions plus the two Go-side guards (unknown speaker, non-positive limit), no database touched
- `internal/gemini/client.go` - `ChatAnswer{Reply string}`, `Client.Chat`, `chatAnswerWire`/`decodeChatAnswer` (fails closed on a missing/empty reply)
- `internal/gemini/client_test.go` - `TestChat_DecodesReply`, `TestChat_MissingReplyIsMalformed`, `TestChat_SchemaShape`, `writeChatSuccess` helper
- `internal/driver/chat.go` - `HandleChat`, `chatPromptContext`, `buildChatSystemInstruction`, `renderRecentDecisions`, `beginChat`/`endChat` (in-flight guard), `appendChatLine`/`recentConversationTail`/`recentDecisionsForChat`/`buildChatPromptContext` (nil-safe collaborator reads), `notifyChat`/`notifyChatSystem`
- `internal/driver/chat_test.go` - the eight named `TestHandleChat_*` tests plus the `chatTestFixture` helper
- `internal/driver/driver.go` - `Models.Chat`, `Decisions.ListForConnection`, `Conversation` interface + `SetConversation`, `ChatEvent`, `Notifier.NotifyChat`, `ChatNotifierFunc`, `Driver.conversation`/`chatInFlight` fields
- `internal/driver/driver_test.go` - `fakeModels.Chat` (+ its own call counter and last-system-instruction/last-user-text recording, reusing the shared `block`/`ignoreCancel` fields), `fakeConversation`, `fakeDecisionsStore.ListForConnection`, `fakeNotifier.NotifyChat`, `fakeSessions.epochForCalls`/`epochForCallCount`
- `internal/driver/corpus_live_test.go` - `corpusDecisions.ListForConnection` (Rule 3: required for the package to compile once `Decisions` widened)
- `internal/session/websocket.go` - `MsgTypeChat`, `ChatPayload`, `WSMessage.Chat`, `PushChat`, `case MsgTypeChat:` in the read loop
- `internal/session/websocket_test.go` - `TestWebSocket_ChatIsNotAWheelGrab`
- `internal/session/manager.go` - `ChatHook`, `SetChatHook`, `FireChatHook`, `Manager.chatHook` field
- `cmd/server/main.go` - `conversationStore` construction, `sessionManager.SetChatHook(aiDriver.HandleChat)`, `aiDriver.SetConversation(conversationStore)`, `SetNotifier` now composes `NotifierFunc` + `ChatNotifierFunc` (the latter pushing through `wsHandler.PushChat`)

## Exact wire/log contract (for plan 05-06 and plan 05-11)

- `ChatAnswer{Reply string}` — plan 05-06 adds coaching-suggestion fields to this same type; nothing else is shipped here (`grep -c "push" internal/gemini/client.go` returns 0)
- `chatPromptContext{Profile, Goal, Window, QuestBullets, SessionMemory, RecentDecisions, ConversationTail}` — plan 05-06 adds one more field, `Coaching`
- Conversation tail size: `maxConversationTail = 10`, tail lines rendered as `"speaker: text"`, clamped through the same `clampBullets` pipeline every other memory layer uses
- Locked notice, unbracketed on the wire: `"AI-chatter has reached the session's call cap and can't reply right now. Your message was saved."` (`chatCapReachedNotice`); confirmed `grep -cE '"\[AI-chatter has reached' internal/driver/chat.go` returns 0
- Locked pointer sentence: `"Update this in AI Player settings."` (`chatUpdateSettingsSentence`)
- `[AI-CHATTER]` log lines carry `stage=received|reply|cap|failed`, ids and lengths only — 10 distinct call sites in `chat.go`, never the message or reply text
- The chat hook lives on `internal/session.Manager` (`ChatHook`/`SetChatHook`/`chatHook`), invoked via the exported `FireChatHook(userID, connectionID, message)` from the websocket read loop's `case MsgTypeChat:`, in a goroutine — not on `WebSocketHandler`
- Inbound chat text rides `WSMessage.Data` (matching `MsgTypeData`'s own convention); the new `WSMessage.Chat *ChatPayload` field is populated on outbound pushes only (`PushChat`)
- `ChatPayload{ID, Speaker, Text, State, Timestamp}`; `State` is `"cap"` or `"failed"` on a system line, empty otherwise
- `go test ./internal/driver/... ./internal/session/... ./internal/gemini/... ./internal/store/... -count=1` all green except the pre-existing, unrelated `TestMigration014AddsLoginStartedAtColumn` CRLF failure (machine-specific, documented below)

## Decisions Made

See `key-decisions` in the frontmatter above.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `Notifier`'s widened two-method interface broke `cmd/server/main.go`'s one `NotifierFunc` call site**
- **Found during:** Task 05-05-02, immediately after adding `NotifyChat` to the `Notifier` interface
- **Issue:** `cmd/server/main.go:330` passed a bare `aidriver.NotifierFunc(...)` value to `SetNotifier`; once `Notifier` required both `NotifyDecision` and `NotifyChat`, that single-method function type no longer satisfied the interface, breaking `go build ./...` for the whole repo — a build the plan's own Task 05-05-02 acceptance criteria require to pass.
- **Fix:** Added a sibling `ChatNotifierFunc` adapter (mirroring `NotifierFunc` exactly) and changed the one call site to compose both via anonymous-struct embedding, promoting both methods onto one concrete value. Task 05-05-02's commit wired the chat half as a documented no-op (MsgTypeChat/PushChat did not exist yet); Task 05-05-03's commit replaced that no-op with the real translation to `wsHandler.PushChat`.
- **Files modified:** `internal/driver/driver.go` (the `ChatNotifierFunc` type itself, needed for any caller), `cmd/server/main.go`
- **Verification:** `go build ./...` green after both commits; `go vet ./...` clean
- **Committed in:** `85fd837` (Task 05-05-02, no-op wiring), `97ae816` (Task 05-05-03, real wiring)

**2. [Rule 3 - Blocking] `Decisions` interface widened by `ListForConnection`, breaking two test-only doubles outside Task 05-05-02's own file list**
- **Found during:** Task 05-05-02, first `go vet ./internal/driver/...` after widening `Decisions`
- **Issue:** `internal/driver/corpus_live_test.go`'s `corpusDecisions` double (not in Task 05-05-02's `<files>` list) no longer satisfied the `Decisions` interface passed to `New` in its own live-corpus test, breaking `go vet`/`go test` for the whole `driver` package.
- **Fix:** Added `corpusDecisions.ListForConnection`, an unfiltered read of every stored row (the corpus runner never calls it itself, so no filtering/limiting logic was needed beyond satisfying the interface).
- **Files modified:** `internal/driver/corpus_live_test.go`
- **Verification:** `go vet ./internal/driver/...` and `go test ./internal/driver/... -count=1` both green after the fix
- **Committed in:** `85fd837` (Task 05-05-02 commit)

### Notes on acceptance-criteria text that does not literally hold against the pre-existing codebase (documented, not fixed — out of this plan's scope per the Scope Boundary rule)

- Task 05-05-03's acceptance criterion `grep -c "SetAINotifier" cmd/server/main.go returns 1` does not hold even at the pre-plan baseline: `cmd/server/main.go` already had two independent `SetAINotifier` call sites before this plan touched the file — `profilesHandler.SetAINotifier` (plan 04-06) and `sessionHandler.SetAINotifier` (plan 05-01) — confirmed by `git diff HEAD~1 -- cmd/server/main.go` for Task 3's own commit, which touched neither line. The intent the criterion is checking ("plan 05-01's wiring is intact") holds: both lines are untouched.
- The plan's literal verification command `awk '/case MsgTypeChat:/,/case Msg/' internal/session/websocket.go | grep -c "Allow()"` (expected to return 1) cannot return 1 for any implementation: the range's own start pattern, the line `case MsgTypeChat:`, itself contains the substring `case Msg`, so POSIX/awk range semantics close the range on that same single line (confirmed with a minimal control example: `printf 'a\ncase MsgTypeChat:\nrl.Allow()\ncase MsgTypeAI:\nb\n' | awk '/case MsgTypeChat:/,/case Msg/'` prints only the one label line). The same collapse makes the two zero-count checks (`applyWheelGrab`, `clientToMUD`) trivially true regardless of the case body's actual contents, so those two checks do not mechanically prove anything either, though they happen to also match the real intent. The underlying property this criterion aims to prove — a rate-limiter check present, no wheel-grab, no MUD-channel write — was confirmed by direct source reading of the `case MsgTypeChat:` block in `internal/session/websocket.go` and by the plan's own plain-English `<verification>` item 4, which states the same property without the malformed one-liner.

---

**Total deviations:** 2 auto-fixed (both Rule 3, blocking), plus 2 documented acceptance-criteria discrepancies against the pre-existing baseline or a self-defeating shell one-liner (neither a fix nor scope creep — both are informational).
**Impact on plan:** Both auto-fixes were required for the plan's own stated acceptance criteria (`go build ./...` exits 0) to be reachable at all. No scope creep beyond what compiling the widened interfaces required. The two documented discrepancies do not represent unmet intent — the actual behaviour each criterion aims to prove is confirmed true by direct code inspection and by the tests this plan added.

## Issues Encountered

- `buildChatSystemInstruction`'s original closing sentence unconditionally named `<CONVERSATION>` in prose even when the tail was empty, which broke `TestHandleChat_PromptCarriesRecentConversation`'s "nil Conversation collaborator produces no `<CONVERSATION>` block at all" subtest (the literal string `<CONVERSATION>` appeared in the instruction regardless of whether the block itself was rendered). Fixed by moving the `<GAME_TEXT>`/`<QUEST_MEMORY>`/`<SESSION_MEMORY>`-style block-naming sentence inside the tail's own `if len(ctx.ConversationTail) > 0` branch and making the final trust-boundary sentence generic (no marker names).
- The first draft of `TestHandleChat_UntrustedBlocksCannotForgeAMarker` asserted each closing marker (`</GAME_TEXT>`, `</SESSION_MEMORY>`, `</CONVERSATION>`) appeared exactly once — this failed, because `untrustedDataParagraph()` (reused verbatim from the existing player/reviewer prompts) already names every marker once in its own explanatory prose, on top of the real wrapped occurrence. Rewrote the test to use the same comparative pattern `TestPromptsHoldEachMarkerOncePerBlock` (`internal/driver/neutralise_test.go`) already established: compare marker counts between an otherwise-identical clean prompt and a poisoned one, since "the honest baseline is the comparison, not the number 1" (that file's own doc comment).

## User Setup Required

None — no external service configuration required. No package was added to `go.mod` or `frontend/package.json` (confirmed empty `git diff --stat go.mod frontend/package.json` after every task).

## Known Stubs

None. This plan touches only Go backend code (`internal/store`, `internal/gemini`, `internal/driver`, `internal/session`, `cmd/server`); no frontend rendering was added or left stubbed. The frontend chat strip, pop-out windows and their wiring to `MsgTypeChat`/`PushChat`/`ChatPayload` are a later plan's work (per the phase's wave plan); this plan's `<output>` only calls for the backend surface these later plans build on.

## Next Phase Readiness

- Plan 05-06 (AI-chatter coaching channel) can rely on: `gemini.ChatAnswer`'s exact shape (`Reply` only, ready for coaching-suggestion fields to join it), `chatPromptContext`'s exact field list (ready for a `Coaching []string` field), the `<CONVERSATION>` framing and the conversation-tail size (10), the locked cap notice and pointer-sentence strings, and the `[AI-CHATTER]` log format.
- Plan 05-11 (evidence capture) can quote `TestConversationStoreSQL`, `TestChat_DecodesReply`, `TestChat_MissingReplyIsMalformed`, `TestChat_SchemaShape`, the eight `TestHandleChat_*` tests and `TestWebSocket_ChatIsNotAWheelGrab` directly into `evidence/01-test-report.txt`, and can quote migration 015's exact object names into the staging startup log check.
- No blockers. The one pre-existing, unrelated test failure (`TestMigration014AddsLoginStartedAtColumn`, a CRLF-vs-LF line-ending artifact of this worktree checkout per this machine's own standing instructions) was left untouched, as instructed.

---
*Phase: 05-coaching-channel*
*Completed: 2026-09-17*
