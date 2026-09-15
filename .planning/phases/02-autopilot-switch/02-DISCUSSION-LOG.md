# Phase 2: Autopilot Switch - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-09-15
**Phase:** 02-autopilot-switch
**Areas discussed:** Indicator and notices, Wheel-grab boundaries, #AUTO directive behaviour

---

## Pending todo cross-reference

| Option | Description | Selected |
|--------|-------------|----------|
| Note for the security review | Record as a reviewed item so the Phase 2 security review raises R-02 and R-04 | ✓ |
| Fold into Phase 2 decisions | Treat remediating R-02 and R-04 as Phase 2 work | |

**User's choice:** Note for the security review.

---

## Indicator and notices

### Where the indicator lives

| Option | Description | Selected |
|--------|-------------|----------|
| Header badge plus terminal notices | Badge beside the connection badge, re-synced from the server, plus a terminal line per change | ✓ |
| Header badge only | Terminal silent on state changes | |
| Terminal only | No badge | |

**User's choice:** Header badge plus terminal notices.

### Terminal notice style

| Option | Description | Selected |
|--------|-------------|----------|
| One short bracketed line per change | Same style as [Disconnected], via the #ECHO local path | ✓ |
| Verbose lines with reason and next step | Each line also says what to do next | |
| You decide | | |

**User's choice:** One short bracketed line per change.

### Connection drops while engaged

| Option | Description | Selected |
|--------|-------------|----------|
| Disconnected line plus an autopilot line | [Disconnected] then [Autopilot disengaged: connection lost] | |
| Badge flips only | | |
| You decide | | |

**User's choice (free text):** `[Disconnected]  [Autopilot waiting for Reconnect]`

### Connection returns

| Option | Description | Selected |
|--------|-------------|----------|
| No reminder; the OFF badge is enough | | |
| One reminder line on reconnect | [Autopilot is off; #AUTO ON to re-engage] | |
| You decide | | |

**User's choice (free text):** `[Reconnected] [Autopilot resuming]`

### Follow-up: the two free-text answers conflicted with the locked "lands on disengaged" rule

| Option | Description | Selected |
|--------|-------------|----------|
| Keep the design: lands on disengaged | No documents change; owner re-engages with #AUTO ON | |
| Change the design: waiting then resume | Autopilot waits across the drop and re-engages by itself; design, requirements, project, roadmap amended | ✓ (via free text, then confirmed) |
| Waiting state, but resume needs confirmation | Reminder on reconnect, no self-resume | |

**User's choice (free text):** "#AUTO ON is a response to an off state. Off state can be engaged with #AUTO OFF." Claude restated it as: ON entered only by #AUTO ON, OFF only by #AUTO OFF, a dropped connection is not #AUTO OFF, autopilot waits and resumes on return. Owner: "This is correct."
**Notes:** Claude flagged the policy section 2 supervision angle and that no reconnect toggle exists (resume happens when the owner reconnects by hand). Owner proceeded. Amendments committed as `c0a8319`; shelf copy untouched pending approval.

---

## Wheel-grab boundaries

### Alias expansion of a typed command

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, it counts as human | Only trigger- and timer-fired commands are exempt | ✓ |
| No, alias output is automation | | |

**User's choice:** Yes, it counts as human.

### Typed local-only # directive

| Option | Description | Selected |
|--------|-------------|----------|
| No, only commands that reach the game | | ✓ (final, after reversal below) |
| Yes, any typed line takes the wheel | | (initial answer) |

**User's choice:** Initially "Yes, any typed line takes the wheel." Reversed in the #AUTO area when Claude pointed out #AUTO ON would toggle itself: "# commands should not interrupt the AI then. Those internal commands should be ignored."

### Blank Enter

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, a blank line is a game command | | ✓ |
| No, ignore blank lines | | |

**User's choice (free text):** "A blank line is a Client command and should be considered human interaction and AI would disengage."

---

## #AUTO directive behaviour

### Does #AUTO trip the wheel-grab?

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, #AUTO is the switch, not a game command | #AUTO exempt | |
| No, keep it uniform | | |

**User's choice (free text):** Broadened the exemption to every # line (see reversal above).

### #AUTO ON while ON / #AUTO OFF while OFF

| Option | Description | Selected |
|--------|-------------|----------|
| No-op with a short notice | [Autopilot is already on] / [Autopilot is already off] | ✓ |
| Treat as a fresh engage or disengage | | |

### #AUTO ON with no connected game

| Option | Description | Selected |
|--------|-------------|----------|
| Refuse with a notice | Switch stays OFF | ✓ |
| Arm it: turns ON as waiting | | |

### #AUTO with no argument

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, #AUTO alone prints the state | [Autopilot: ON / WAITING / OFF] plus gate result; the diagnostic surface | ✓ |
| No, only ON and OFF exist | | |

---

## Claude's Discretion

Badge visuals and placement; websocket message shape for the state broadcast; endpoint paths; where the WAITING → ON resume hooks in; state-machine package and tests; source-flag field name and its default (absent = human); log line format; handling of a typed # directive that emits game commands.

## Deferred Ideas

- Security review: WAITING-then-resume against policy section 2 (possible bounded WAITING lifetime).
- Shelf copy of the amended design doc awaits the owner's approval.
- Reviewed, not folded: deferred security risks R-02 and R-04 (to the Phase 2 security review).
