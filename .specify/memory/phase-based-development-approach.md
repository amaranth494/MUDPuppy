# Phase-Based Software Development Approach

## Purpose

Development work is organized into a hierarchy of **Phases, Waves, Plans, and Tasks**.

The purpose of this structure is to ensure that development is driven by measurable changes to the game rather than by activity alone. Every significant body of work should ultimately produce something that can be demonstrated, tested, observed by the player, or verified through the game's diagnostic systems.

The hierarchy is:

**Phase → Wave → Plan → Task**

Each level answers a different question:

- **Phase:** What consequential capability are we delivering?
- **Wave:** In what dependency order can the work be performed?
- **Plan:** What measurable system change moves us toward the Phase goal?
- **Task:** What individual work must be completed to implement the Plan?

---

# 1. Phase

A **Phase** is the highest-level development unit.

A Phase represents the delivery of a consequential capability, feature, system, or player experience that does not exist—or does not adequately exist—before the Phase begins.

A Phase should never be defined merely as a collection of engineering activity.

For example:

> "Work on the inventory system"

is not a sufficient Phase.

Instead:

> "Establish a functional inventory data system capable of creating, storing, retrieving, and validating inventory items."

defines an actual deliverable.

## Phase Goal

Every Phase must have a clearly defined **Goal**.

The Goal describes the tangible state of the game or codebase that should exist when the Phase is complete.

A Phase Goal should answer:

> **What will be true after this Phase that is not true before it?**

The Phase Goal should describe a meaningful capability rather than the implementation steps required to create it.

### Example

**Phase: Inventory Data Foundation**

**Goal:**  
Establish a persistent inventory data system capable of defining, storing, retrieving, and validating inventory items for use by future gameplay systems.

---

# 2. Phase Success Criteria

Every Phase must include explicit **Success Criteria**.

Success Criteria determine whether the Phase has achieved its Goal.

Success must be independently verifiable through one or both of the following mechanisms:

### Player-Observable Verification

The player can directly experience or interact with the completed capability.

Examples include:

- The player can pick up an item.
- The item appears in the inventory.
- The player can equip an item.
- A door correctly reacts to possession of a key.
- An enemy reacts to player proximity.

### Diagnostic Verification

The system can demonstrate that the capability functions correctly even when no player-facing interface exists yet.

Examples include:

- An inventory database query returns expected records.
- A diagnostic command successfully creates and retrieves an item.
- Automated validation reports no invalid item definitions.
- A debug interface displays database contents.
- Runtime telemetry confirms that an event was successfully processed.

Diagnostic verification is particularly important for foundational development where player-facing systems may not yet exist.

A Phase is not complete simply because its Tasks are finished.

A Phase is complete when its **Success Criteria have been demonstrated**.

---

# 3. Plans

A Phase is divided into one or more **Plans**.

A Plan represents a meaningful and measurable change to the system that contributes directly toward the Phase Goal.

Plans are larger than individual engineering assignments but smaller than the overall Phase.

Each Plan should produce a state change that can be evaluated independently.

A good Plan answers:

> **What new capability or measurable condition will exist after this work is completed?**

Examples might include:

- Define the inventory data schema.
- Implement inventory repository services.
- Add runtime inventory validation.
- Add inventory diagnostic commands.
- Connect inventory persistence to save data.

Plans should not simply describe categories of work such as:

> "Database work"

or:

> "Backend programming"

Instead, the Plan should describe the capability being created.

---

# 4. Plan Acceptance Criteria

Every Plan must have **Acceptance Criteria**.

Acceptance Criteria define the measurable conditions that demonstrate that the Plan has been successfully implemented.

Acceptance Criteria should describe observable results rather than merely confirming that code was written.

For example:

### Weak Acceptance Criteria

> Inventory database code is complete.

### Strong Acceptance Criteria

> A valid inventory item definition can be written to the inventory database and retrieved by its unique identifier with all required fields intact.

Whenever practical, Acceptance Criteria should be executable through:

- player interaction,
- developer tools,
- automated tests,
- debug interfaces,
- diagnostic commands,
- telemetry,
- logs,
- database inspection,
- runtime validation.

The completion of every Plan should therefore create a measurable difference between the state of the system before and after the Plan.

---

# 5. Waves

Plans within a Phase are organized into **Waves**.

A Wave represents the dependency order in which Plans can be executed.

The purpose of Waves is to make dependencies explicit while allowing independent work to proceed in parallel whenever possible.

## Wave 1

Wave 1 contains Plans that have no dependencies on other Plans within the Phase.

These Plans can begin immediately.

Multiple Plans may exist within Wave 1 if they can be performed independently.

## Wave 2

Wave 2 contains Plans that require the successful completion of one or more Plans from Wave 1.

## Wave 3 and Beyond

Subsequent Waves follow the same model.

A Plan belongs to the earliest Wave in which all of its dependencies have been satisfied.

This creates a dependency structure such as:

**Wave 1**
- Plan A
- Plan B
- Plan C

↓

**Wave 2**
- Plan D, dependent on A
- Plan E, dependent on B and C

↓

**Wave 3**
- Plan F, dependent on D and E

Plans within the same Wave should be considered parallelizable unless an explicit dependency exists between them.

If Plan B requires Plan A to be complete, they should not be placed in the same Wave.

---

# 6. Tasks

Plans are implemented through **Tasks**.

Tasks are the smallest planned units of assigned work.

Tasks represent specific actions performed by developers, designers, artists, engineers, or other contributors.

Examples include:

- Create the inventory item data class.
- Add the `item_id` field.
- Create the inventory database table.
- Implement the item repository.
- Add a debug command for item lookup.
- Write test inventory records.
- Add validation for duplicate item IDs.
- Update developer documentation.

Tasks do not necessarily require independent Acceptance Criteria.

Their purpose is to track the individual assignments necessary to complete a Plan.

A Task should normally be specific enough that:

1. someone can be assigned responsibility for it,
2. its completion can be tracked,
3. its relationship to a Plan is clear.

The completion of Tasks contributes to completion of the Plan.

The completion of Plans contributes to completion of the Wave.

The completion of all required Waves contributes to achievement of the Phase Goal.

---

# 7. Development Hierarchy

The development structure can therefore be represented as:

```text
PHASE
│
│ Goal
│ Success Criteria
│
├── WAVE 1
│   │
│   ├── PLAN 1
│   │   ├── Acceptance Criteria
│   │   ├── Task
│   │   ├── Task
│   │   └── Task
│   │
│   └── PLAN 2
│       ├── Acceptance Criteria
│       ├── Task
│       └── Task
│
├── WAVE 2
│   │
│   ├── PLAN 3
│   │   ├── Depends on Plan 1
│   │   ├── Acceptance Criteria
│   │   ├── Task
│   │   └── Task
│   │
│   └── PLAN 4
│       ├── Depends on Plan 1 and Plan 2
│       ├── Acceptance Criteria
│       └── Tasks
│
└── WAVE 3
    │
    └── PLAN 5
        ├── Depends on Plan 3 and Plan 4
        ├── Acceptance Criteria
        └── Tasks
```

---

# 8. Example: Inventory Database Phase

## Phase: Inventory Data Foundation

### Goal

Create a functional inventory data foundation capable of defining, storing, retrieving, and validating game inventory items.

### Success Criteria

The Phase is successful when:

- Inventory items can be created using the defined data structure.
- Inventory items can be stored and retrieved by unique identifier.
- Invalid inventory records are detected.
- Duplicate identifiers are rejected.
- The inventory database can be queried through an in-game diagnostic interface or developer command.
- A diagnostic test demonstrates successful creation, storage, retrieval, and validation of test inventory records.

No player-facing inventory screen is required for this Phase because the capability can be verified through diagnostics.

---

## Wave 1

### Plan 1: Define Inventory Data Model

**Objective:**  
Establish the canonical structure used to represent an inventory item.

**Acceptance Criteria:**

- A defined inventory item schema exists.
- Required and optional fields are documented.
- Invalid item definitions can be distinguished from valid definitions.

**Tasks:**

- Define item identifier.
- Define item name.
- Define item type.
- Define item properties.
- Define serialization format.
- Document required fields.
- Create representative test items.

---

### Plan 2: Establish Inventory Storage

**Objective:**  
Create the persistence mechanism used to store inventory records.

**Acceptance Criteria:**

- Inventory records can be written to storage.
- Stored records persist in the expected format.
- Records can be inspected through development tooling.

**Tasks:**

- Create storage structure.
- Configure inventory database.
- Create seed data.
- Implement database initialization.
- Add storage diagnostics.

---

# Wave 2

## Plan 3: Implement Inventory Repository

**Dependencies:**  
Plan 1 and Plan 2.

**Objective:**  
Provide the game with a controlled interface for accessing inventory data.

**Acceptance Criteria:**

- Items can be created.
- Items can be retrieved by ID.
- Missing items return a predictable result.
- Duplicate IDs are rejected.

**Tasks:**

- Implement create operation.
- Implement lookup operation.
- Implement update operation if required.
- Implement duplicate detection.
- Implement error handling.
- Add repository tests.

---

## Plan 4: Implement Inventory Validation

**Dependency:**  
Plan 1.

**Objective:**  
Prevent invalid item data from entering or remaining within the inventory system.

**Acceptance Criteria:**

- Required fields are validated.
- Invalid records generate diagnostic errors.
- Duplicate identifiers are detected.
- A validation report can identify all invalid test records.

**Tasks:**

- Implement schema validation.
- Implement identifier validation.
- Implement duplicate detection.
- Add diagnostic logging.
- Create invalid test records.

---

# Wave 3

## Plan 5: Implement Inventory Diagnostics

**Dependencies:**  
Plan 3 and Plan 4.

**Objective:**  
Provide developers with a mechanism for proving that the inventory data system is functioning.

**Acceptance Criteria:**

A diagnostic tool can:

- list inventory records,
- retrieve a specific inventory item,
- report invalid records,
- demonstrate successful database access,
- report failures in a reproducible manner.

**Tasks:**

- Add inventory diagnostic command.
- Add record listing.
- Add item lookup.
- Add validation reporting.
- Add test execution command.
- Document diagnostic usage.

---

# Phase Completion

The Inventory Data Foundation Phase is complete when all required Plans have satisfied their Acceptance Criteria and the Phase Success Criteria can be demonstrated.

For example, a developer should be able to execute:

```text
inventory_test
```

and receive a result demonstrating that:

```text
Inventory Database: PASS
Records Loaded: 25
Lookup Test: PASS
Write Test: PASS
Duplicate ID Test: PASS
Validation Test: PASS
Invalid Records: 0
```

The existence of this measurable result demonstrates that the Phase delivered its intended capability even though the player may not yet have an inventory interface.

---

# 9. Feature Decomposition Process

When a major feature is proposed, development planning should proceed from the top down.

## Step 1 — Define the Desired Capability

Determine what meaningful new capability should exist.

This becomes the **Phase**.

Ask:

> What will the game be capable of doing after this work that it cannot do now?

---

## Step 2 — Define Phase Success

Determine how the team will prove that the capability exists.

Success must be demonstrated through either:

- player-observable behavior,
- diagnostic verification,
- or both.

If there is no clear way to prove the Phase succeeded, the Phase is not yet sufficiently defined.

---

## Step 3 — Identify Intermediate System Changes

Determine which measurable changes must occur to reach the Phase Goal.

These become **Plans**.

Each Plan should create a meaningful intermediate capability.

---

## Step 4 — Identify Dependencies

Determine which Plans require other Plans to be completed first.

Use those dependencies to organize Plans into **Waves**.

Plans without dependencies enter Wave 1.

Plans dependent on Wave 1 enter Wave 2.

Continue until the dependency chain has been represented.

---

## Step 5 — Define Plan Acceptance Criteria

For every Plan, determine how its completion will be demonstrated.

Ask:

> What observable difference will exist when this Plan is finished?

If there is no measurable answer, the Plan may be too vague or may simply represent a collection of Tasks rather than an actual Plan.

---

## Step 6 — Break Plans into Tasks

Determine the individual assignments required to satisfy each Plan's Acceptance Criteria.

Tasks should be concrete, assignable units of work.

---

## Step 7 — Execute by Wave

Work begins with all eligible Plans in Wave 1.

Plans within the same Wave may proceed concurrently.

A subsequent Wave becomes eligible when its required predecessor Plans have met their Acceptance Criteria.

---

## Step 8 — Validate the Phase

Completing all Plans does not automatically complete the Phase.

The Phase's Success Criteria must be executed and demonstrated independently.

This final validation confirms that the collection of Plans actually produced the intended feature.

---

# 10. Core Rules

Several rules govern the development model.

### Rule 1 — Phases describe outcomes, not activity.

A Phase must create a consequential capability.

---

### Rule 2 — Every Phase must be provable.

The Phase must contain explicit Success Criteria that can be demonstrated through gameplay or diagnostics.

---

### Rule 3 — Plans must change the system.

A Plan is not simply a category of Tasks. Completing the Plan must result in an observable and measurable difference.

---

### Rule 4 — Every Plan requires Acceptance Criteria.

A Plan is complete only when its intended system change has been demonstrated.

---

### Rule 5 — Waves represent dependencies.

Wave ordering is determined by dependency, not convenience or arbitrary scheduling.

---

### Rule 6 — Work within a Wave should be parallelizable.

If one Plan depends upon another, they belong in different Waves.

---

### Rule 7 — Tasks are execution units.

Tasks describe the work necessary to implement Plans and can be individually assigned and tracked.

---

### Rule 8 — Task completion does not equal feature completion.

Tasks complete Plans.

Plans complete Waves.

Waves contribute toward Phases.

Only successful validation of the Phase Success Criteria completes the Phase.

---

# 11. Standard Phase Template

Each Phase should use the following structure:

```text
PHASE: <Phase Name>

GOAL:
<Describe the consequential capability that will exist when
the Phase is complete.>

SUCCESS CRITERIA:
- <Measurable Phase result>
- <Measurable Phase result>
- <Player or diagnostic verification>

WAVE 1

    PLAN 1: <Plan Name>

    OBJECTIVE:
    <System change created by the Plan>

    ACCEPTANCE CRITERIA:
    - <Measurable result>
    - <Measurable result>

    TASKS:
    - <Task>
    - <Task>
    - <Task>


    PLAN 2: <Plan Name>

    OBJECTIVE:
    <System change created by the Plan>

    ACCEPTANCE CRITERIA:
    - <Measurable result>

    TASKS:
    - <Task>
    - <Task>


WAVE 2

    PLAN 3: <Plan Name>

    DEPENDENCIES:
    - Plan 1
    - Plan 2

    OBJECTIVE:
    <System change created by the Plan>

    ACCEPTANCE CRITERIA:
    - <Measurable result>

    TASKS:
    - <Task>
    - <Task>


PHASE VALIDATION:

<Describe exactly how the Phase Success Criteria will
be demonstrated.>
```

---

# 12. Guiding Principle

The central principle of this development approach is:

> **Development progress is measured by demonstrated changes in capability, not by the amount of work performed.**

Tasks represent effort.

Plans represent measurable implementation.

Waves represent dependency.

Phases represent delivered capability.

Every Phase should end with the team being able to point to something concrete and say:

> **The game can now do this, and here is how we prove it.**