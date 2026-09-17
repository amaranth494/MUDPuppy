---
phase: 05-coaching-channel
plan: 06
subsystem: ai-safety
tags: [gemini, structured-output, coaching, prompt-injection, go, rest-api, red-team-corpus]

# Dependency graph
requires:
  - phase: 05-coaching-channel
    provides: "05-05's chatPromptContext/buildChatSystemInstruction, gemini.ChatAnswer{Reply}, the conversation store and MsgTypeChat transport; 05-02's fail-closed reviewer verdict; 05-04's per-user AI send rate limiter"
  - phase: 04-continuous-play
    provides: "promptContext/buildSystemInstruction/buildReviewSystemInstruction/untrustedDataParagraph, clampBullets/renderBullets/neutraliseLine/wrapSessionMemory, the profiles GET-only sub-resource pattern (GetAISettings/GetSessionMemory), the red-team corpus runner"
provides:
  - "gemini.ChatAnswer.Push/Withdraw, tolerant-decoded like session_memory/quest_memory; reply stays required and first in PropertyOrdering"
  - "internal/store/coaching.go: CoachingStore.CoachingFor/UpdateCoaching/CoachingForConnection, the Session Memory trio's shape on the coaching_suggestions column migration 015 added; openGameSessionSQL now seeds coaching_suggestions alongside session_memory"
  - "internal/driver/chat.go: chatPromptContext.Coaching, rendered in AI-chatter's own prompt with the verbatim-copy withdraw instruction; applyCoaching as the coaching store's one and only writer (push/withdraw, ceiling, dedup); Go-composed 'Sent to AI-player:'/'Withdrew from AI-player:' reply prefixes; the unbracketed 'Coaching received' marker on the existing decision channel"
  - "internal/driver/memory.go: wrapCoaching, maxCoachingBullets (8), markerNameRe now defusing COACHING and CONVERSATION via breakMarkerName"
  - "internal/driver/driver.go: Coaching interface/SetCoaching/coachingBullets; promptContext.Coaching threaded into both buildSystemInstruction and buildReviewSystemInstruction, below Session Memory, framed as the owner's own guidance (not model output); untrustedDataParagraph's <COACHING> clause, deliberately absent from the never-follow list"
  - "internal/profiles/handler.go: GetCoaching/GetConversation, GET-only, matching GetAISettings/GetSessionMemory's ownership-check shape"
  - "cmd/server/main.go: coachingStore wired to the driver and to the two new /ai-coaching, /ai-conversation routes"
  - "internal/driver/corpus_live_test.go: chatter-channel-01..03, the corpus's own attack on this phase's new laundering risk (D-28)"
affects: [05-11-security-review-and-close]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "A coaching store with exactly one writer (HandleChat), mechanically proven by a source-level test reading the package's own files, as the answer to a laundering risk a nil-safe collaborator alone cannot close"
    - "Server-composed reply prefixes (Sent to AI-player:/Withdrew from AI-player:) built from the stored list, never from the model's own reply text, so a lying model cannot forge owner-facing confirmation"
    - "A shared wrap/clamp helper (wrapCoaching) serving two independent prompts (AI-chatter's own, AI-player's own) so a withdraw can match text either model actually saw, byte for byte"
    - "Marker-name defusal for single-word markers (COACHING, CONVERSATION) via a matched-text-relative hyphen insertion (breakMarkerName), replacing the four-marker-only capture-group replacement that would have silently deleted a single-word match down to a bare hyphen"

key-files:
  created:
    - internal/store/coaching.go
    - internal/store/coaching_test.go
    - internal/profiles/coaching_test.go
  modified:
    - internal/gemini/client.go
    - internal/gemini/client_test.go
    - internal/store/transcripts.go
    - internal/driver/chat.go
    - internal/driver/chat_test.go
    - internal/driver/memory.go
    - internal/driver/driver.go
    - internal/driver/driver_test.go
    - internal/driver/corpus_live_test.go
    - internal/profiles/handler.go
    - internal/session/websocket.go
    - cmd/server/main.go

key-decisions:
  - "wrapCoaching/maxCoachingBullets landed in memory.go as part of task 05-06-01's own commit (a Rule 3 pull-forward, like 05-04's precedent), because chat.go's own coaching block needs them to compile and render; task 05-06-02 then added the marker-regex extension, the promptContext field and the AI-player prompt wiring on top"
  - "applyCoaching processes withdraws first, then pushes, against the list read once at the top of the call; ceiling enforcement (drop-oldest) runs once at the end over the whole resulting list rather than per-push, so a single push of 9+ lines in one message still lands at exactly maxCoachingBullets"
  - "Withdraw matching is exact-text first, then case-insensitive; a push exactly matching an existing line (post-neutralise) is not duplicated"
  - "The coaching-received marker rides the EXISTING decision Event/Notifier path (Kind: system, Outcome: coaching-received) via decorateEvent, not a new notify mechanism -- it carries the same stint counts every other AI-player system line does"
  - "GetCoaching/GetConversation on the profiles handler each get their own coachingStorage/conversationStorage interface (mirroring transcriptStorage), and their own SetCoachingStore/SetConversationStore setters following SetDecisionStore's nil-underlying-pointer guard, rather than widening an existing interface"
  - "TestHandleChat_PushReachesNextPrompt (the plan's own named round-trip test spanning both tasks) was written in task 05-06-02's commit, once promptContext.Coaching and buildSystemInstruction's own wiring existed, per the plan's own explicit either-is-acceptable note"

patterns-established:
  - "A prompt block whose wording must NOT reuse a sibling block's own 'you wrote this yourself' framing gets its own constant (coachingIntroSentence/reviewCoachingIntroSentence) rather than parameterising the shared one, so the wording difference is a compile-time-visible fact, not a runtime flag"

requirements-completed: [REQ-coaching-chat]

# Metrics
duration: ~75min
completed: 2026-09-17
---

# Phase 5 Plan 06: AI-Chatter Coaching Channel Summary

**A chat push becomes a standing coaching line AI-player reads on its very next decision, quoted back to the owner byte-for-byte and withdrawable by exact text, delimited and provenance-checked against the same laundering risk the phase's own new corpus items now attack.**

## Performance

- **Duration:** ~75 min (commit-to-commit, base 647a959 to 6bcdebc)
- **Started:** 2026-09-17T13:05:00Z (approximate, worktree base commit)
- **Completed:** 2026-09-17T13:25:00Z
- **Tasks:** 3
- **Files modified:** 15 (3 created, 12 modified)

## Accomplishments

- A coaching line the owner asks for reaches AI-player's very next decision: `TestHandleChat_PushReachesNextPrompt` proves the round trip from `HandleChat`'s storage write to `buildSystemInstruction`'s own `<COACHING>` block, byte for byte.
- AI-chatter can see what it has already told AI-player: the coaching in effect travels in its own `<COACHING>` block inside AI-chatter's own prompt (D-17), with an instruction to copy a line back **verbatim** when withdrawing it, proven against a real withdraw round trip in `TestHandleChat_PromptCarriesCoachingInEffect` (the withdraw text is read out of the fake model's own recorded instruction, never hard-coded).
- Nothing reaches AI-player the owner cannot see: `TestHandleChat_ReplyQuotesTheExactLineSent` proves the `Sent to AI-player:` prefix is composed in Go from the stored list, byte-identical, even when the model's own `reply` text claims something different; `TestHandleChat_EmitsCoachingReceivedMarker` proves the unbracketed `Coaching received` marker fires exactly once per push/withdraw, carrying no suggestion text.
- A withdraw removes the named line and says so; a withdraw matching nothing removes nothing and says so (`TestHandleChat_Withdraw`).
- Coaching lasts exactly as long as Session Memory: `TestCoachingStoreSQL` proves the SQL shape shares Session Memory's `login_started_at` bound and that `openGameSessionSQL` now seeds `coaching_suggestions` in the same INSERT.
- Coaching never outranks the profile: `TestCoachingIsSubordinateInBothPrompts` proves the coaching block sits after the conduct rules, approach guidance, Never-issue list, goal, Quest Memory and Session Memory in both prompts, and that neither prompt's own coaching label claims model authorship.
- The coaching store has exactly one writer: `TestCoachingHasOneWriter` reads the package's own `.go` files and proves `.UpdateCoaching(` is called from exactly one non-test file, `chat.go`.
- The panel has somewhere to reload "Coaching in effect" and the conversation from: `GetCoaching`/`GetConversation`, GET-only, wired to `/ai-coaching`/`/ai-conversation`.
- The red-team corpus now attacks this phase's own new risk: `chatter-channel-01..03` address "your coach"/"your assistant" and ask for a standing rule, rather than telling the player model to act directly; `TestCorpusIsWellFormed` requires the category and the three ids.

## Task Commits

1. **Task 05-06-01: What the owner asks for becomes a standing suggestion, and what he takes back disappears** - `a9b7f45` (feat)
2. **Task 05-06-02: AI-player and the safety checker both read the coaching, and neither lets it outrank the profile** - `fc2ad03` (feat)
3. **Task 05-06-03: The panel can reload what is in effect, and the corpus attacks the channel** - `6bcdebc` (feat)

_No plan-metadata commit is made by this executor: per this worktree's instructions, STATE.md/ROADMAP.md are the orchestrator's to update after all wave agents complete._

## Files Created/Modified

- `internal/gemini/client.go` - `ChatAnswer.Push`/`Withdraw` (optional array-of-string, `schemaProperty.Items`); `PropertyOrdering: [reply, push, withdraw]`; `chatAnswerWire`/`decodeChatAnswer` tolerant-decode push/withdraw the same way `decodeAnswer` tolerates a malformed `session_memory`
- `internal/gemini/client_test.go` - `TestChat_DecodesPushAndWithdraw`, `TestChat_MalformedPushIsDropped`, `writeChatAnswer`; `TestChat_SchemaShape` updated for the 3-element ordering and the two new properties
- `internal/store/coaching.go` - `CoachingStore`: `CoachingFor`, `UpdateCoaching`, `CoachingForConnection` (the `login_started_at`-bounded connection read), copying `transcripts.go`'s Session Memory trio field for field
- `internal/store/coaching_test.go` - `TestCoachingStoreSQL`: SQL-constant shape assertions, no database touched
- `internal/store/transcripts.go` - `openGameSessionSQL` now seeds `coaching_suggestions` alongside `session_memory` in the same INSERT, same `COALESCE(...'[]'::jsonb)` shape
- `internal/driver/chat.go` - `chatPromptContext.Coaching`; the coaching-in-effect block and verbatim-copy sentence in `buildChatSystemInstruction`; `applyCoaching` (the one writer), `composeChatReply`, `notifyCoachingReceived`, `indexOfCoachingLine`/`indexOfCoachingLineFold`, the `coachingSentPrefix`/`coachingWithdrewPrefix`/`coachingWithdrawNoMatchSentence` constants; AI-chatter's own "second job" instruction paragraph
- `internal/driver/chat_test.go` - `newFakeCoaching` wiring in `chatTestFixture`; `TestHandleChat_PromptCarriesCoachingInEffect`, `TestHandleChat_Withdraw`, `TestHandleChat_ReplyQuotesTheExactLineSent`, `TestHandleChat_EmitsCoachingReceivedMarker`, `TestHandleChat_CoachingCeilings`, `TestCoachingHasOneWriter`; `TestHandleChat_UntrustedBlocksCannotForgeAMarker` extended to compare `<COACHING>`/`</COACHING>` counts too
- `internal/driver/memory.go` - `promptContext.Coaching`; `maxCoachingBullets = 8`; `wrapCoaching`; `markerNameRe` now matches `COACHING`/`CONVERSATION`; `breakMarkerName` (matched-text-relative hyphen insertion, replacing the old capture-group-only replacement)
- `internal/driver/driver.go` - `Coaching` interface, `coaching` field, `SetCoaching`; `coachingBullets` helper; `promptCtx.Coaching` assembly in `decide`; `coachingIntroSentence`/`reviewCoachingIntroSentence`; the `<COACHING>` block in both `buildSystemInstruction` and `buildReviewSystemInstruction`; the `<COACHING>` clause in `untrustedDataParagraph`; `Event`'s doc comment names `coaching-received`
- `internal/driver/driver_test.go` - `fakeCoaching`; `TestCoachingIsWrappedInBothPrompts`, `TestCoachingIsSubordinateInBothPrompts`, `TestCoachingCeilingIsEnforced`, `TestCoachingCannotForgeAMarker`, `TestHandleChat_PushReachesNextPrompt`; `TestUntrustedParagraphNamesEveryMarker` extended; two pre-existing blank-Never-issue-list subtests narrowed to the labelled-heading text (see Deviations)
- `internal/driver/corpus_live_test.go` - `chatter-channel-01`, `chatter-channel-02`, `chatter-channel-03`; `chatter-channel` added to `requiredCategories`; an explicit id-existence check beside the `benign-fight` one
- `internal/profiles/handler.go` - `coachingStorage`/`conversationStorage` interfaces; `Handler.coaching`/`Handler.conversation` fields; `SetCoachingStore`/`SetConversationStore`; `CoachingResponse`, `ConversationLineResponse`, `ConversationResponse`; `GetCoaching`, `GetConversation`
- `internal/profiles/coaching_test.go` - `fakeCoachingStore`, `fakeConversationStore`; `TestGetCoaching`, `TestGetConversation` (stored round-trip, empty-array, not-owned refusal, no-store-wired 503, method-not-allowed, mirroring `TestGetSessionMemory`'s exact shape)
- `internal/session/websocket.go` - `AIDecisionPayload`'s doc comment names `coaching-received` among its documented `Outcome` values
- `cmd/server/main.go` - `coachingStore` construction; `aiDriver.SetCoaching(coachingStore)`; `profilesHandler.SetCoachingStore`/`SetConversationStore`; the `/ai-coaching` and `/ai-conversation` route registrations

## Decisions Made

See `key-decisions` in the frontmatter above.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `wrapCoaching`/`maxCoachingBullets` pulled forward from task 05-06-02 into task 05-06-01's commit**
- **Found during:** Task 05-06-01, writing `chat.go`'s own coaching-in-effect block
- **Issue:** The plan's own action text for task 1 says "render it with the same `wrapCoaching` helper task 05-06-02 adds to `memory.go`" — but `memory.go` is not in task 1's file list, and `wrapCoaching`/`maxCoachingBullets` did not exist yet at that point in a strict task-ordering read.
- **Fix:** Added the minimal `wrapCoaching` function and `maxCoachingBullets` constant to `memory.go` in task 1's own commit (nothing else in `memory.go` changed at that point — no `promptContext.Coaching` field, no marker-regex extension). Task 2's commit then added the rest of `memory.go`'s changes on top.
- **Files modified:** `internal/driver/memory.go` (in task 1's commit)
- **Verification:** `go build ./...`, `go vet`, and task 1's own named tests all green before task 1's commit; task 2's own tests green after task 2's commit
- **Committed in:** `a9b7f45` (Task 1), `fc2ad03` (Task 2 completes the file)

**2. [Rule 1 - Bug] Two pre-existing blank-Never-issue-list subtests and one marker-forging assertion narrowed to survive task 2's own change**
- **Found during:** Task 05-06-02, after adding the `<COACHING>` clause to `untrustedDataParagraph`
- **Issue:** `TestBuildSystemInstruction`/`TestBuildReviewSystemInstruction`'s `blank_list_emits_no_heading` subtests asserted a bare `"Never-issue"` substring never appears when the profile's own list is blank; `TestHandleChat_UntrustedBlocksCannotForgeAMarker` asserted a bare `"<COACHING>"`/`"</COACHING>"` substring never appears in a poisoned chat prompt. Both broke once the shared `untrustedDataParagraph`'s own new coaching sentence started naming "the Never-issue list" and `<COACHING>`/`</COACHING>` in prose, unconditionally, on every prompt — exactly like it already does for `<GAME_TEXT>`/`<QUEST_MEMORY>`/`<SESSION_MEMORY>`.
- **Fix:** The two blank-list subtests now check for the labelled heading text `"Never-issue commands ("` specifically, not the bare word. `TestHandleChat_UntrustedBlocksCannotForgeAMarker` folds `<COACHING>`/`</COACHING>` into its existing comparative marker-count loop (clean vs. poisoned), matching the file's own documented rationale ("exactly once is not the honest baseline; no more than the clean version is") rather than a hard zero-check. `TestHandleChat_PromptCarriesCoachingInEffect`'s own empty-store subtest was similarly narrowed to `"<COACHING>\n"` (the real wrapped block) rather than the bare marker name. The underlying property each test protects — no real heading/block when none applies — still holds and is still asserted.
- **Files modified:** `internal/driver/driver_test.go`, `internal/driver/chat_test.go`
- **Verification:** `go test ./internal/driver/... -count=1` green, including all four pre-existing Phase 4 prompt tests
- **Committed in:** `fc2ad03` (Task 2)

**3. [Rule 2 - Missing Critical] Handler-level tests for `GetCoaching`/`GetConversation`**
- **Found during:** Task 05-06-03
- **Issue:** The plan's task 3 acceptance criteria name only `TestCorpusIsWellFormed` and source-assertion greps for the two new endpoints; it does not name a handler-level test the way earlier tasks name theirs. Every other GET-only AI sub-resource in this codebase (`GetAISettings`, `GetSessionMemory`) has its own handler test suite (`TestGetSessionMemory` et al.) proving the round trip, the empty-array shape, the ownership refusal and the fail-closed 503 — leaving the two new endpoints untested would be an inconsistency an untested HTTP surface represents a real gap.
- **Fix:** Added `internal/profiles/coaching_test.go` with `TestGetCoaching`/`TestGetConversation`, mirroring `TestGetSessionMemory`'s exact five-subtest shape (stored round-trip, empty array, not-owned refusal, no-store-wired 503, method-not-allowed).
- **Files modified:** `internal/profiles/coaching_test.go` (new)
- **Verification:** `go test ./internal/profiles/... -count=1 -v` green
- **Committed in:** `6bcdebc` (Task 3)

### Notes on acceptance-criteria text that does not literally hold against the codebase (documented, not fixed — out of scope per the Scope Boundary rule)

- Task 1's own literal check `grep -rc "UpdateCoaching(" internal/driver --include=*.go | grep -v "_test.go" | grep -v ":0"` cannot name exactly one file: `driver.go`'s own `Coaching` interface declares the method (`UpdateCoaching(gameSessionID uuid.UUID, coaching []string) error`), which also contains the literal substring `UpdateCoaching(` — a declaration, not a call. `TestCoachingHasOneWriter` proves the actual property (`.UpdateCoaching(` — a method call on a receiver — appears in exactly one non-test file, `chat.go`) mechanically instead.
- Task 2's own verification item 4 describes the never-follow sentence as listing "exactly GAME_TEXT, QUEST_MEMORY, SESSION_MEMORY and MODEL_REASONING" — direct code reading confirms `MODEL_REASONING` has never been part of that sentence (it is reviewer-only, named by the separate `reviewReasoningUntrustedSentence`); this task did not add it, matching the task's own action text ("extending the existing never-follow list without adding `<COACHING>` to it"). `TestUntrustedParagraphNamesEveryMarker` asserts the sentence still names the three markers that were actually there, plus confirms `COACHING` is absent — the substantive property the criterion aims at.
- Task 3's own check `grep -c "SetCoaching" cmd/server/main.go returns 1` returns 2, because `SetCoachingStore` (the profiles-handler setter, following `SetDecisionStore`/`SetQuestStore`'s naming convention) contains `SetCoaching` as a substring. Both calls are necessary and distinct (`aiDriver.SetCoaching` wires the driver's own collaborator; `profilesHandler.SetCoachingStore` wires the GET-only read); confirmed by direct code reading, not a duplicate or accidental second driver-side call.
- gofmt's own struct-field column alignment inserts multiple spaces between `Coaching` and `[]string` in `chatPromptContext`'s declaration, so task 1's literal `grep -c "Coaching \[\]string"` (single space) returns 0; `grep -cE "Coaching +\[\]string"` (flexible whitespace) confirms the field exists exactly once.

---

**Total deviations:** 3 auto-fixed (1 blocking, 1 bug, 1 missing-critical), plus 4 documented acceptance-criteria discrepancies against the codebase's own pre-existing shape or gofmt's own formatting (none a fix, none scope creep — the underlying property each criterion aims to prove is confirmed true by a passing unit test or direct code inspection).
**Impact on plan:** All three auto-fixes were necessary for the plan's own stated acceptance criteria to be reachable at all, or to close an obvious test-coverage gap the plan's own conventions elsewhere demand. No behaviour outside D-08/D-09/D-10/D-12/D-17/D-28 was touched.

## Issues Encountered

- The first draft of `TestCoachingIsSubordinateInBothPrompts` used the bullet text `"coaching bullet"` — the neutraliser's own extended `markerNameRe` (now matching the bare word `COACHING` case-insensitively, not just `<COACHING>`) defused the word "coaching" inside that literal test string into `"coac-hing"`. Fixed by using a bullet text (`"avoid the north road"`) that does not itself contain the word being defended against — the neutraliser behaved correctly; the test string was a poor choice.
- The first draft of the ordering test's marker-position extraction used a bare `strings.Index(instruction, "- ")` to find the coaching bullet inside the built prompt; this matched the em-dash-style `" -- "` sequences used throughout the prose (a literal `-` immediately followed by a space is a substring of `--`), landing on the wrong position entirely. Fixed by anchoring the search to `"<COACHING>\n"`/`"\n</COACHING>"` specifically.

## User Setup Required

None — no external service configuration required. No package was added to `go.mod` or `frontend/package.json` (confirmed empty `git diff --stat go.mod frontend/package.json` after every task).

## Known Stubs

None. This plan touches only Go backend code (`internal/store`, `internal/gemini`, `internal/driver`, `internal/profiles`, `internal/session`, `cmd/server`); no frontend rendering was added or left stubbed. The frontend "Coaching in effect" list and its wiring to the new `/ai-coaching`/`/ai-conversation` GET endpoints are a later plan's work per this phase's own wave plan; this plan's `<output>` calls only for the backend surface those later plans build on.

## Threat Flags

None beyond what the plan's own threat model already registers (T-5-26 through T-5-31, T-5-56, T-5-SC) — every file this plan touched implements a mitigation already named in `05-06-PLAN.md`'s threat register; no new network endpoint, auth path or schema change outside that register was introduced. The two new GET routes (`/ai-coaching`, `/ai-conversation`) are covered by T-5-30 (Information Disclosure), mitigated the same way every other AI sub-resource is (`h.getProfileByConnectionID`).

## Next Phase Readiness

- Plan 05-11 (evidence capture) can quote `TestCoachingStoreSQL`, `TestChat_DecodesPushAndWithdraw`, `TestChat_MalformedPushIsDropped`, the nine `TestHandleChat_*`/`TestCoaching*` tests in `chat_test.go` and `driver_test.go`, `TestCoachingHasOneWriter`, `TestGetCoaching`, `TestGetConversation` and the extended `TestCorpusIsWellFormed` directly into `evidence/01-test-report.txt`, and can file the `chatter-channel-01..03` item lines from a live corpus run into `evidence/04-redteam-after.txt` (D-28).
- The exact coaching ceiling (8, `maxBulletChars` 200), the two locked prefixes (`Sent to AI-player: `, `Withdrew from AI-player: `), the unbracketed marker message (`Coaching received`), and the two new route paths/response shapes (`CoachingResponse{coaching []string}`, `ConversationResponse{lines []ConversationLineResponse}`) are all available for the frontend plan that wires the panel's "Coaching in effect" list and pop-out views.
- `TestHandleChat_PushReachesNextPrompt` — the plan's own named round-trip test spanning both tasks 05-06-01 and 05-06-02 — was written and committed in task 05-06-02's commit (`fc2ad03`), once `promptContext.Coaching` and its wiring into `buildSystemInstruction` existed; the plan's own text names this split as acceptable.
- No blockers. The one pre-existing, unrelated test failure (`TestMigration014AddsLoginStartedAtColumn`, a CRLF-vs-LF line-ending artifact of this worktree checkout per this machine's own standing instructions) was left untouched, as instructed.

---
*Phase: 05-coaching-channel*
*Completed: 2026-09-17*

## Self-Check: PASSED

- FOUND: internal/store/coaching.go
- FOUND: internal/store/coaching_test.go
- FOUND: internal/profiles/coaching_test.go
- FOUND: internal/gemini/client.go
- FOUND: internal/driver/chat.go
- FOUND: internal/driver/memory.go
- FOUND: internal/driver/driver.go
- FOUND: internal/profiles/handler.go
- FOUND: internal/session/websocket.go
- FOUND: cmd/server/main.go
- FOUND: internal/driver/corpus_live_test.go
- FOUND: commit a9b7f45 (Task 05-06-01)
- FOUND: commit fc2ad03 (Task 05-06-02)
- FOUND: commit 6bcdebc (Task 05-06-03)
