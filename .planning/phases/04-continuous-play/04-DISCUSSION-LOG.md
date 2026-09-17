# Phase 4: Continuous Play - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-09-16
**Phase:** 4-continuous-play
**Areas discussed:** Session goal; Loop rhythm and in-stint memory; Halts, thresholds, and blocks; Carry-forward asks (retention, policy wording, staging model); Memory model (owner-supplied)

Both pending todos (`2026-09-16-phase4-carry-forward-from-3.1.md`, `2026-09-16-phase4-security-carry-forward.md`) were folded without a question: each is the owner's own instruction marked to resolve in Phase 4.

---

## Session goal

| Option | Description | Selected |
|--------|-------------|----------|
| Goal box in the AI Assist panel | One-line box at the top of the panel; #AUTO ON as today | ✓ |
| #AUTO ON followed by the goal text | Typed in the terminal; bare #AUTO ON reuses the last goal | |
| Both: panel box is the source, directive fills it | Two ways to do one thing | |

**User's choice:** Goal box in the AI Assist panel.

| Option | Description | Selected |
|--------|-------------|----------|
| Engage anyway, play from the approach guidance | Panel shows "No session goal set" | ✓ |
| Refuse to engage until a goal is typed | Refusal line like the policy gate | |

**User's choice:** Engage anyway.

| Option | Description | Selected |
|--------|-------------|----------|
| Until you change or clear it | Server-side per profile; survives refresh, reconnect, OFF/ON | ✓ |
| Cleared every #AUTO OFF | Fresh goal per stint; wheel-grab clears it | |
| Lasts for the game connection only | Cleared at disconnect | |

**User's choice:** Until you change or clear it.

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, takes effect on the next decision | System line "Goal changed" | ✓ |
| No, locked while ON | #AUTO OFF to change it | |

**User's choice:** Yes, takes effect on the next decision.

---

## Loop rhythm and in-stint memory

| Option | Description | Selected |
|--------|-------------|----------|
| After new game output settles, with a quiet-game fallback | Settle delay, floor interval, minimum spacing, ICM backstop; constants Claude's | ✓ |
| Only after new game output, never on silence | A silent game stalls the stint | |
| Fixed clock | One decision every N seconds | |

**User's choice:** After new game output settles, with a quiet-game fallback.

| Option | Description | Selected |
|--------|-------------|----------|
| Its last few decisions, cleared on re-engage | Short list of commands and reasons in the prompt | ✓ (superseded) |
| Nothing beyond the game window, as today | Each decision stands alone | |
| A running plan it writes and re-reads each turn | Scratchpad plan, cleared on re-engage | |

**User's choice:** Its last few decisions. **Notes:** Superseded later in the session by the owner's four-layer memory model (below): Session Memory, a curated fact list living for the game connection, replaces the recent-decisions list.

---

## Halts, thresholds, and blocks

| Option | Description | Selected |
|--------|-------------|----------|
| OFF, with a notice | Cap reached lands Off; badge in the same message | ✓ |
| Stays ON but halted | A fourth "capped" reading | |

**User's choice:** OFF, with a notice.

| Option | Description | Selected |
|--------|-------------|----------|
| One stint: #AUTO ON to OFF | Count resets on every #AUTO ON | ✓ |
| One game connection | Connect to disconnect across stints | |
| One calendar day per profile | Daily budget on the profile | |

**User's choice:** One stint.

| Option | Description | Selected |
|--------|-------------|----------|
| Consecutive failures, reset by a sent command | Phase 1's documented default; non-transient failures disengage at once | ✓ |
| Total failures in the stint | Never resets during a stint | |

**User's choice:** Consecutive failures, reset by a sent command.

| Option | Description | Selected |
|--------|-------------|----------|
| Skip the turn; repeated blocks in a row disengage | Blocks are not failures; same threshold number applied to consecutive blocks, own notice | ✓ |
| Skip the turn; blocks never disengage | Only the cap bounds a run of blocks | |
| Count a block like a failure | One counter for both | |

**User's choice:** Skip the turn; repeated blocks in a row disengage.

---

## Carry-forward asks

| Option | Description | Selected |
|--------|-------------|----------|
| 30 days | Nightly delete of transcript lines and decision window text | |
| 90 days | Longer window | |
| No time limit, delete path only | Nothing expires | |
| I need to discuss this further | | ✓ |

**User's choice:** Opened a discussion. Asked whether "logs or some other kind of retained material" was meant; answered that two stores exist (log-page transcripts and hidden per-decision snapshots). The owner then supplied the memory model (below). Retention re-asked with memory separated out:

| Option | Description | Selected |
|--------|-------------|----------|
| Snapshots 7 days, transcripts 30 days | Per-profile delete-now path for both | ✓ |
| Snapshots 7 days, transcripts 90 days | | |
| Snapshots 7 days, transcripts never expire | | |

**User's choice:** Snapshots 7 days, transcripts 30 days.

| Option | Description | Selected |
|--------|-------------|----------|
| Approve the wording, version 1.1 | Section 2 bullet rewritten to state auto-resume; no re-acceptance | |
| Leave the policy at 1.0 as is | Contradiction stays; risk already accepted | ✓ |

**User's choice:** Leave the policy at 1.0.

| Option | Description | Selected |
|--------|-------------|----------|
| Stay on gemini-3.5-flash-lite | Separate daily quota; cheaper per call | ✓ |
| Back to gemini-3.5-flash | Quota was spent by one day of corpus runs | |

**User's choice:** Stay on gemini-3.5-flash-lite.

---

## Memory model (owner-supplied, free text)

The owner supplied a three-layer memory model (Immediate Context, Session Memory, Historical Memory), then a Quest Memory addendum, then a spoken addendum that detail is inversely proportional to age. Saved verbatim to `.specify/specs/ai-memory-model-v1.md`. Follow-up questions:

| Question | Options | Selected |
|----------|---------|----------|
| Placement | Immediate Context and Session Memory in Phase 4, Historical in Phase 6 / All three in Phase 4 | Phase 4 + Phase 6 split |
| Session Memory visibility | Panel read-only / Panel editable / Not shown | Panel read-only |
| Goal box names the active Quest? | Yes / No, separate | Yes |
| Phase 4 / Phase 6 split with Quests | Quest Memory in Phase 4, closure and Historical in Phase 6 / Quest Memory waits for Phase 6 | Quest Memory in Phase 4 |
| Session Memory lifetime | The game connection across stints / One stint, cleared at #AUTO OFF | The game connection |
| Quest Memory in the panel? | Read-only beside Session Memory / Hidden | Free text: "It should be stored separately, but can be recalled through the AI chat interface." |

One AskUserQuestion round was dismissed by the owner ("I skipped over your questions") and re-asked on request.

---

## Claude's Discretion

Memory update mechanism (fields in the decision answer JSON, no extra call); Immediate Context seconds and retained screenful; memory size ceilings, storage shape, migration numbers, goal-text matching on reactivation; loop constants and goroutine lifecycle; exact notice wording in the UI-SPEC voice; panel layout for the goal box and Session Memory section; where the delete-captured-text action sits; `ai` message fields and REST routes; nightly job implementation; `[AI-PLAYER]` log lines; test scaffold organisation; whether icm-refused counts as transient; the 503 retry delay.

## Deferred Ideas

Quest Memory recall through chat (Phase 5); editing Session Memory by hand (Phase 5); end-of-session evaluation, quest closure, Historical Memory (Phase 6; roadmap Phase 6 wording to amend); policy v1.1 rewording declined; per-profile retention settings; fake game fixture on staging (declined in 3.1); backlog defects WR-03, IN-02, stale worktree directory; AR-3-08 production note.
