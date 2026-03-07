---
name: speckit.plan
description: Create a technical implementation plan from a feature specification
agent: speckit
workflow: ../.kilocode/workflows/speckit.plan.md
handoffs:
  - label: Generate Tasks
    agent: speckit.tasks
    prompt: Generate actionable tasks from the plan. I am building with...
---

## Usage

/speckit.plan [feature spec]

## Description

Create a technical implementation plan from a feature specification. This command:

1. Loads the feature specification
2. Analyzes requirements and success criteria
3. Creates a detailed technical plan with implementation steps
4. Identifies dependencies and risks
5. Defines milestones and checkpoints

## Example

/speckit.plan Create plan for user authentication feature
