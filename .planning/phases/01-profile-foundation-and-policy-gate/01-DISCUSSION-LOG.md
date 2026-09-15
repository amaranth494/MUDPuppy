# Phase 1: Profile Foundation and Policy Gate - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-09-15
**Phase:** 01-profile-foundation-and-policy-gate
**Areas discussed:** Where the owner edits and accepts, Policy text and acceptance record, AI settings shape and defaults

---

## Where the owner edits and accepts

| Option | Description | Selected |
|--------|-------------|----------|
| New 'AI Player' section in Settings | Beside Key Bindings, Aliases, Triggers, Timers; same per-connection sub-resource pattern; ProfileModal untouched | ✓ |
| Tab in the ProfileModal | Reuse the older modal | |
| Both | Full editor in Settings plus read-only summary in the modal | |

**User's choice:** New 'AI Player' section in Settings

| Option | Description | Selected |
|--------|-------------|----------|
| Policy panel in the AI section; refusal links to it | Full text, version, Accept button; then "Accepted v1.0 on <date>"; refused engage links here | ✓ |
| Modal on refusal, also openable from the section | Policy as a modal at refusal time | |
| Section only | Acceptance only in settings; refusal just points there | |

**User's choice:** Policy panel in the AI section; refusal links to it

| Option | Description | Selected |
|--------|-------------|----------|
| Accept button plus a section-1 checkbox | Explicit "game permits automation" checkbox stored with acceptance | |
| Single Accept button | Accepting the text is the confirmation | ✓ |
| Scroll-to-end then Accept | Button enables after scrolling the text | |

**User's choice:** Single Accept button
**Notes:** Moved to next area without further questions.

---

## Policy text and acceptance record

| Option | Description | Selected |
|--------|-------------|----------|
| History table, one row per acceptance | Audit trail of every acceptance | |
| Two columns on the profile | Version accepted and timestamp on the profiles row | ✓ |
| Columns plus history | Both | |

**User's choice:** "It's stored in the connection profile, per MUD connection."

| Option | Description | Selected |
|--------|-------------|----------|
| Serve the spec file directly | Read `.specify/specs/...` at startup like help content | |
| Versioned copy under policy/ with a matching test | Deployable copy with drift test | |
| Embedded in Go | Policy text as a Go constant | (discretion) |

| Option | Description | Selected |
|--------|-------------|----------|
| Parsed from the policy file header | "Policy version: 1.0" parsed at startup | (discretion) |
| Go constant | Bumped by hand | |
| Database table | Editable without deploy | |

**User's choice:** Chose "discuss further" on both, then stated that these are not functional concerns and were a speed bump. Stated the required behaviour: the policy pops up the first time AI configuration is opened on a profile, is accepted once, never expires, is never re-asked when the policy changes, and is stored on the profile; deleting the profile means asking again. Text delivery and version handling were left to Claude (embedded markdown, version parsed from its header, recorded for the record only).
**Notes:** This changed the design doc (D1 and Definition of complete item 1), the policy intro paragraph, the Phase 1 roadmap entry, REQ-policy-gate, and the PROJECT.md decision table. All amended and committed.

---

## AI settings shape and defaults

| Option | Description | Selected |
|--------|-------------|----------|
| One model, blank means server default | Single field; blank uses the env-configured model | ✓ |
| Two: play model and debrief model | Separate names | |
| You decide | | |

**User's choice:** One model, blank means server default

| Option | Description | Selected |
|--------|-------------|----------|
| Call cap 100 per session, disengage after 3 consecutive errors | | |
| Call cap 250, disengage after 5 consecutive errors | | |
| You decide | | |

**User's choice:** Free text: "no cap, but error handling is in place that needs to fail with an informative error, but still allow regular play. It can't CRASH."
**Notes:** Blank call cap means no cap. Blank threshold means built-in error handling: informative error, disengage, regular play unaffected, never crash. The numeric threshold default is Claude's discretion. This amended the design doc (D1 acceptance and Definition of complete item 6), Phase 1 and Phase 4 roadmap criteria, REQ-profile-ai-fields, REQ-safety-limits-hold, and the PROJECT.md constraint.

---

## Claude's Discretion

- Policy text embedded in the Go binary and served by one endpoint; version parsed from the file header.
- Numeric disengage-threshold default when blank.
- Text length limits and validation messages for conduct rules and guidance.
- JSONB shape of `ai_settings`, Go struct and defaults.
- Endpoint paths for the AI sub-resource, policy text, accept action, and engage-gate check.
- Migration layout.

## Deferred Ideas

- `#AI STATUS` diagnostic directive (Phase 2 or later).
- Policy section 7 enforcement remains v2 (ENF-01).
