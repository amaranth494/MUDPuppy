---
created: 2026-09-18
source: Phase 5 security review (.planning/phases/05-coaching-channel/05-SECURITY.md, Deferred Risks), Phase 5 Risk Register https://claude.ai/artifact/Ep5B1NwSKXUJ3eAYsUJf8n
resolves_phase: 6
---

# Phase 6 carry-forward from the Phase 5 security review

Fold into Phase 6 planning. The four deferrals are recorded in `.planning/RISK-REGISTER.md` (DR-5-01 to DR-5-04) and must be raised again at the Phase 6 security review with the same three choices (Accept / Defer / Remediate Now). The project cannot close while any of them is active. Owner notes below are verbatim.

## Deferred risks the owner asked to fix in the next phase, per the recommended approach

1. **The local-development switch would also work on Railway** (DR-5-01, register R-06, low; T-5-NEW-07 / IN-06 / T-5-52). Owner note: "fix this in next phase per the recommended approach". Recommended fix: make the server refuse the local-development opt-out (`MUDPUPPY_LOCAL_DEV`) when it can see it is running on a hosted platform such as Railway, and say why (`internal/config/config.go` no longer looks at `RAILWAY_ENVIRONMENT` at all today, so the switch is honoured everywhere it is set). Add a test.
2. **Chats are kept forever, and "Delete Captured Text Now" does not remove them** (DR-5-04, register R-11, low; T-5-NEW-09). Owner note: "Fix this in the next Phase per the recommended approach". Recommended fix: give conversation lines the same 30-day retention life as transcripts and include them in the owner's "Delete Captured Text Now" control (`internal/store/retention.go` currently names only `window_text` and `game_session_lines`; nothing prunes `conversation_lines`).

## Deferred risk that is a bug fix with an added requirement

3. **AI-chatter will answer for a profile that never accepted the policy** (DR-5-03, register R-10, low; T-5-NEW-11). Owner note: "AI-Chat and AI-Player should not ever become available unless the AI policy is accepted in the game profile.  Fix this as a bug in the next Phase." Recommended fix: have the server ask the same policy-acceptance gate before it answers a chat message that `#AUTO ON` already uses, and refuse with the same plain notice (`internal/driver/chat.go` and the chat case in `internal/session/websocket.go` currently consult no policy gate at all). **The owner's note adds a requirement, not just a fix: AI-Chat and AI-Player must never become available in a game profile unless that profile's AI policy has been accepted.** This requirement should be confirmed as holding for both surfaces, not only patched for chat.

## Deferred risk addressed by later AI-player logic work

4. **A hostile room could try to get a sentence to the AI through the chat, and the check on that can be got round** (DR-5-02, register R-08, high; T-5-26 deferred to the Phase 5 review + T-5-NEW-02 found by the audit). Owner note: "We are going to address this in advanced logic for the AI-Player later." Recommended fix on the register: close the two push-gate holes found by the audit — count every word that is not a filler word whatever its length (today only words of four letters or more count as real words), and run the provenance check on exactly the text that gets stored, not the longer text that is checked and then cut to 200 characters. Add both cases to the tests. This narrows the gap; per the register, it cannot remove it entirely, since neither gate can tell a genuinely sensible-sounding suggestion from a planted one.

## At the Phase 6 security review

Re-present all four with what Phase 6 did or did not build, and record each outcome in an `Outcome` column on the Phase 05 Deferred table in `.planning/RISK-REGISTER.md`.
