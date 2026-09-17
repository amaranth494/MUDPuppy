# AI Memory Model v1

Owner's statement, supplied 2026-09-16 during the Phase 4 (Continuous Play) discussion. Precedence: sits beside `ai-game-player-design-v3.md` and refines its D3/D4/D6 memory language; where this document and design v3 differ on memory, this document governs. Text below is the owner's, verbatim; only headings were normalised.

---

For the purposes of AI decision-making, the agent should maintain three different layers of game memory rather than treating its entire play history as one continuously retained block of game text.

## 1. Immediate Context — Reaction Window

The Immediate Context is the short-lived window of game text the AI is actively reacting to right now.

This should contain approximately the last 10 seconds of relevant game activity, or whatever amount of text corresponds to the immediate interaction window for the game.

It exists to answer questions such as:

* What was just said to me?
* What appeared or changed on screen?
* Who just attacked me?
* What choice was I just offered?
* What happened as a direct result of my last action?

This is the layer that may contain relatively raw game text because precise wording, sequencing, and immediate state can matter to the next decision.

It should be ephemeral. As events move outside the reaction window, their raw text should normally be discarded rather than accumulated indefinitely.

## 2. Session Memory — Curated Working Memory

The Session Memory is a compact, curated representation of important things that have happened during the current play session.

It is not a transcript and should not contain full game logs.

Instead, it should be maintained as a structured or bulleted set of facts representing meaningful interactions, transactions, discoveries, and unresolved threads from the session.

Examples:

* Spoke with Captain Reyes about the missing supply convoy.
* Learned that the northern bridge is destroyed.
* Fought and defeated three bandits near the quarry.
* Acquired the brass observatory key.
* Promised Mira that we would look for her brother.
* Merchant Halden refused to sell ammunition.
* Discovered that the warehouse has a locked basement.
* Current objective is to reach the communications tower.

The AI should continually curate this information rather than blindly appending everything that occurs.

Items may be:

* added when something consequential happens;
* updated when circumstances change;
* merged when multiple events describe the same situation;
* removed when they are no longer relevant;
* promoted into long-term memory when they have lasting significance.

The purpose of Session Memory is to give the AI enough context to pursue goals coherently across an entire session without retaining the raw text of every interaction.

## 3. Historical Memory — Long-Term Episodic Memory

The Historical Memory represents significant events and knowledge accumulated across the AI's lifetime of play.

This is deeper historical context rather than a running record of everything the AI has ever seen.

It should consist of durable, summarized information such as:

* Major characters encountered and the AI's history with them.
* Important alliances, betrayals, promises, or conflicts.
* Major quests or objectives completed.
* Significant discoveries about the world.
* Recurring behavioral lessons.
* Important locations previously visited.
* Major victories or failures.
* Long-running unresolved story threads.

Historical Memory may be produced through periodic consolidation rather than written directly during every interaction.

For example, session logs and Session Memory could later be reviewed and condensed into something like:

* Helped Commander Voss defend Ardent Station during the pirate attack.
* Established a cooperative relationship with engineer Talia Marr.
* Learned that the Black Meridian organization is searching for pre-collapse navigation technology.
* Previously entered the Helios ruins but could not access the sealed lower level.

This layer should favor meaning and continuity over exact wording.

Raw game text should generally not survive into Historical Memory.

## Relationship Between the Three Layers

The intended information flow is:

Game Output → Immediate Context → Session Memory → Historical Memory

Each transition should reduce detail while increasing durability.

Immediate Context preserves enough raw information to react correctly.

Session Memory preserves what matters for the current experience.

Historical Memory preserves what matters to the AI's continuing understanding of the game world.

The system therefore should not treat memory as an ever-growing transcript.

A useful design principle is:

The older an event becomes, the less text we preserve and the more meaning we preserve.

This also separates memory from the independent AI-decision audit record.

A decision record may contain the model's reasoning, selected command, outcome, and other decision metadata. Those records serve auditing and analysis purposes.

They should not require indefinitely retaining the raw game-text window that accompanied the decision.

If a temporary game-text snapshot is captured alongside a decision so that the decision can initially be reconstructed or audited, that snapshot should be treated as a separate, pruneable artifact rather than as part of the AI's persistent memory.

---

# Addendum: Quest Memory — Persistent Goal-Oriented Memory

In addition to Immediate Context, Session Memory, and Historical Memory, the AI should maintain a fourth memory type: Quest Memory.

Quest Memory is a persistent, goal-oriented memory store for objectives that may span multiple play sessions.

A "quest" in this context does not need to be a formal game quest. It is any goal the character is actively trying to accomplish over time.

Examples include:

* Reach level 10.
* Acquire enough money to purchase a specific item.
* Find a particular NPC.
* Learn how to access a restricted location.
* Build a particular character capability.
* Complete a collection.
* Gain membership in a faction.
* Defeat a recurring enemy.
* Discover the cause of an ongoing problem.

The defining characteristic is that the goal may not be completed during the current session.

## Purpose

Quest Memory exists so that progress toward an active goal is not lost when a session ends.

Session Memory may contain information relevant to the goal while the current session is underway, but anything that remains important to accomplishing the goal must be promoted into Quest Memory before that session context is discarded.

Quest Memory should therefore preserve:

* The goal itself.
* Current progress toward the goal.
* Important discoveries related to the goal.
* Requirements or prerequisites that have been learned.
* Relevant people, places, objects, or systems.
* Failed approaches that should not be repeated.
* Unresolved leads.
* Intermediate milestones already completed.
* Constraints or blockers preventing further progress.

For example, a Quest Memory for Reach Level 10 might contain:

* Current level: 7.
* Forest enemies provide reliable experience with low risk.
* Arena matches provide significantly more experience but require an entry fee.
* Completed the trainer's qualification challenge.
* The eastern caves are currently too dangerous.
* Need approximately 18,000 additional experience.
* The merchant in Redhaven sells experience-boosting food.

The purpose is not to retain every interaction that occurred while pursuing the goal. It is to retain the information that remains useful for continuing pursuit of that goal.

## Quest Memory Is Multi-Session Working Memory

Quest Memory should persist independently of individual sessions.

The relationship between Session Memory and Quest Memory is therefore selective.

During play:

Immediate Context → Session Memory

When something occurs that materially affects an active goal:

Session Memory → Quest Memory

At the end of a session, the system should evaluate active goals and ensure that any important progress, discoveries, blockers, or next steps have been preserved in their corresponding Quest Memory entries.

This allows the AI to begin a future session with continuity:

* What am I trying to accomplish?
* How far have I gotten?
* What have I already learned?
* What should I try next?
* What should I avoid repeating?

## Quest Completion and Closure

Quest Memory is temporary in the sense that it should only exist while the goal remains active.

When a goal reaches a terminal state, its detailed Quest Memory should be closed.

A terminal state may be:

* Succeeded — the goal was accomplished.
* Failed — the goal can no longer be accomplished.
* Abandoned — the character intentionally stopped pursuing it.
* Invalidated — circumstances changed such that the goal no longer exists or makes sense.

At that point, the detailed working information is no longer necessary.

Instead, the system should summarize the outcome and promote that summary into Historical Memory.

For example:

Active Quest Memory

Goal: Gain membership in the Iron Guild.
Spoke with Guildmaster Renn.
Need sponsorship from two current members.
Talia agreed to sponsor me.
Borin will sponsor me if I recover his stolen tools.
Tools were traced to the abandoned mill.

After success, this might become:

Historical Memory

Joined the Iron Guild after earning sponsorship from Talia and Borin and recovering Borin's stolen tools.

The detailed Quest Memory can then be removed.

Likewise, if the goal becomes impossible:

Attempted to join the Iron Guild, but the guild dissolved following Guildmaster Renn's death.

That outcome becomes historical context, while the active quest state is discarded.

## Quest Memory Should Track Meaning, Not Logs

Like Session Memory and Historical Memory, Quest Memory should not become a transcript.

It should be continuously curated.

New information should only be retained when it changes one of the following:

* Understanding of the goal.
* Progress toward the goal.
* Available strategies.
* Known requirements.
* Known blockers.
* Relevant relationships.
* Likely next actions.

Repeated or obsolete information should be merged, updated, or removed.

For example:

Instead of retaining:

* Talked to guard about tower.
* Guard said tower needs key.
* Later talked to merchant about key.
* Merchant said mayor has key.
* Mayor said key was stolen.

Quest Memory should consolidate this into:

* Accessing the tower requires a key.
* The mayor originally possessed the key.
* The key has been stolen; current location unknown.

## Revised Memory Model

The complete model therefore contains four distinct forms of memory:

Immediate Context — "What is happening right now?"
Short-lived raw or near-raw information used for immediate reactions.

Session Memory — "What has happened during this session that still matters?"
Curated working memory for the current play session.

Quest Memory — "What do I need to remember until this goal is resolved?"
Persistent, multi-session working memory tied to active goals.

Historical Memory — "What important things have happened over my lifetime?"
Durable episodic summaries of completed experiences and resolved goals.

The information flow becomes:

Game Output → Immediate Context → Session Memory

From Session Memory, significant information can follow two paths:

Session Memory → Quest Memory
when the information remains relevant to an unresolved goal.

Session Memory → Historical Memory
when the information is independently significant to the character's long-term history.

When a quest reaches a terminal state:

Quest Memory → Historical Memory
and the detailed Quest Memory is then removed.

A useful design principle is:

Session Memory remembers what still matters today. Quest Memory remembers what still matters until the job is done. Historical Memory remembers what mattered after it was over.

---

# Addendum 2: Detail Is Inversely Proportional to Age

Owner's spoken remark, 2026-09-16, tidied from speech without changing its meaning.

The detail stored in each memory layer must be inversely proportional to the time since it was committed to memory.

* **Immediate Context** carries the most detail. The AI needs the most recent details in full, because any of them could be what it has to react to next.
* **Session Memory** carries some detail, but compacted or truncated so the full scope of the session fits.
* **Quest Memory** carries only the pieces of information germane to the quest. Those pieces may have details of their own, but they are mostly bullet points as they pertain to the quest.
* **Historical Memory** carries little detail. It is mostly footnotes in a list: this is what happened to me last month, three months ago, a year ago. Little snippets the AI can harken back to, not a lot of detail.
