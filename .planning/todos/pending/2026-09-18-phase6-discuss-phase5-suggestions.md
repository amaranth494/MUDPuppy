---
created: 2026-09-18
source: Phase 5 end-of-phase suggestions (.planning/phases/05-coaching-channel/05-PHASE-SUGGESTIONS.md, published at https://claude.ai/artifact/Nvz3BpyBwa4WiGD18tAnJB); owner instruction in chat, 2026-09-18
resolves_phase: 6
---

# Address the Phase 5 suggestions in the next discussion

Owner instruction, verbatim, 2026-09-18: "Make sure these suggestions are addressed in the next discussion."

This is binding on `/gsd-discuss-phase 6`. Every item below must be put to the owner in that discussion and leave it with an explicit outcome recorded in `06-CONTEXT.md`: taken into Phase 6, given to a named later phase or an inserted phase, done as a small standalone fix, or declined by the owner. None may be silently dropped. They are not security risks and must not appear on a Risk Register.

Raise first, before Phase 6's own scope is discussed, because the owner said it may change later phases:

- **b. How the AI models are used.** The owner said: "I want to make adjustments to how models are used in later Phases." Facts from Phase 5: the free daily quota for one model blocked testing for most of a day; the server knows only one provider (Gemini), so the OpenAI key the owner offered could not be used; AI-player, AI-chatter and the safety checker share one model setting. Ask what he wants changed, then decide whether it is part of Phase 6, its own inserted phase, or later.

The owner's own play-feature ideas: he said on 2026-09-17 "I have several updates in terms of play features that I think would be useful, but we need to close out PH5 first." Phase 5 is closed. Ask for them at the start of the same discussion and place each one the same way.

Then the rest, in the report's groups:

- **a. A standing suggestion made the AI repeat itself** (was register item R-14; the owner's note there: "We are going to address this in advanced logic for the AI-Player later."). Goes with his planned advanced AI-player logic; confirm where that work lives on the roadmap.
- **c. AI-chatter forgets the conversation when the game reconnects** (IN-03). Suggested for Phase 6, Measurement and Memory.
- **d. The play screen has no automated tests.** Needs the owner's decision on adding a test-runner package.
- **e. The Help page runs numbered steps together in every article.**
- **f. A hard refresh drops the live game view and needs a reconnect.**
- **h. Live model tests should run one at a time, spaced out** (the `-run TestLiveCorpus` filter starts both live tests).
- **i. Two code tidy-ups from the review** (IN-02, IN-07).
- **j. Developer-machine papercuts** (stale worktree base; one migration test and CRLF on Windows).
- **g. Run the staging harness once with autopilot on**, so its pause-and-resume check stops printing SKIP.
- **k. Staging leftovers**: the throwaway "Hand Play (no policy)" profile, and the Phase 4 walkthrough goal text still in the owner's goal box.

Also fold in at the same discussion: `.planning/todos/pending/2026-09-18-phase6-security-carry-forward.md` (DR-5-01 to DR-5-04, three of which the owner asked to be fixed in the next phase, including his requirement that AI-chatter and AI-player never become available on a profile that has not accepted the policy).
