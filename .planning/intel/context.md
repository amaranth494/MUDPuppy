# Context

Synthesized by gsd-doc-synthesizer on 2026-09-14 (MODE: new).

No DOC-typed documents were ingested. This file carries the narrative and process context from the two
SPECs and from the owner's ingest instruction that does not fit as a decision, requirement, or constraint.

---

## Topic: Ingest scope and exclusions

source: owner instruction (ingest-docs prompt, 2026-09-14)

Exactly two documents are ingested: the AI Game Player design v3 (SPEC, precedence 0) and the Safety and
Abuse Policy v1 (SPEC, precedence 1). The design doc references the policy; the policy is what the design's
D1 policy gate enforces. All earlier specs in `.specify/` (SP00-SP06, PR01, PR02, constitution.md) are
deliberately excluded and treated as null. The existing MUDPuppy product is taken as-is; new work is
net-new.

---

## Topic: Intended goal (verbatim)

source: .specify/specs/ai-game-player-design-v3.md (Intended goal)

"Extend MUDPuppy so a game character can be driven either by its human owner or by a reasoning AI, switched
as easily as engaging and disengaging a car's autopilot. The AI takes instructions on how to approach a
game, accepts a small achievable goal per session, plays toward it using a language model (Gemini) rather
than scripted triggers, and gets measurably better over time using the game's own progression numbers.
Everything game-specific lives on the connection profile, so the same engine adapts to Alter Aeon, the
owner's own MUD, or any other text game, with no game knowledge built into the plumbing and no intent
detection in the engine."

---

## Topic: Delivery ordering and dependencies (verbatim)

source: .specify/specs/ai-game-player-design-v3.md (Order and dependencies)

"D1 through D4 are strictly sequential; each is a working, testable increment. D5 and D6 both build on D4
and can proceed in either order, though D5 first matches the collaboration-heavy early phase. D7 needs D6's
notes. D8 needs everything and is the finish line."

Deliverable titles for reference:
- D1: Profile foundation and policy gate
- D2: Autopilot switch
- D3: One AI decision
- D4: Continuous play
- D5: Coaching channel
- D6: Measurement and memory
- D7: Study loader
- D8: Acceptance run on Alter Aeon

---

## Topic: Supervision model

source: owner instruction (ingest-docs prompt, 2026-09-14); .specify/specs/safety-and-abuse-policy-v1.md (section 2); .specify/specs/ai-game-player-design-v3.md (Out of scope)

The AI driver runs server-side in Go and issues commands through the ICM automation context, but the
browser is the supervision surface and server-side does not mean unattended. The policy expects the owner
to be reachable and checking in at reasonable intervals, and the design lists autonomous unattended
operation as out of scope. The coaching pane (D5), live reasoning display (D3), status indicator (D2), and
wheel-grab (D2) are the concrete supervision affordances.

---

## Topic: Acceptance narrative

source: .specify/specs/ai-game-player-design-v3.md (D8; Definition of complete)

The proof runs on Alter Aeon from staging under the owner's supervision, in two phases. First,
reconnaissance on a fresh profile with the owner coaching, so the AI discovers and records how Alter Aeon
measures progress. Then goal sessions: the owner hand-plays through character creation and the mud school,
engages autopilot outside restricted areas, and the AI completes small session goals measured honestly by
the tally. Completion also requires that across at least three consecutive goal sessions the tally
improves while required coaching declines.

---

## Topic: Document linkage

source: .specify/specs/ai-game-player-design-v3.md (Related documents); .specify/specs/safety-and-abuse-policy-v1.md (header)

The design's "Related documents" section points at `.specify/specs/safety-and-abuse-policy-v1.md`
(version 1.0) and states per-profile acceptance is required before AI engagement, referenced by D1. The
policy has no outgoing references. The cross-reference graph is a single edge, design -> policy, and is
acyclic.

---

## Topic: Classifier notes carried forward

source: .planning/intel/classifications/ai-game-player-design-v3-specify1.json; .planning/intel/classifications/safety-and-abuse-policy-v1.json

- Both documents were typed SPEC by manifest override (`manifest_override: true`), confidence high.
- The classifier observed that the design document's acceptance-criteria style has PRD flavor; this
  synthesis extracted those criteria into requirements.md while keeping the SPEC precedence.
- The classifier observed that the policy reads as a user-facing acceptable-use policy but imposes
  enforceable system behaviors; those behaviors are recorded in constraints.md with CON-policy-* IDs.
- Classification output filenames use fallback identifiers rather than SHA-256 path hashes because no
  hashing tool was available to the classifier. This has no effect on synthesis.
