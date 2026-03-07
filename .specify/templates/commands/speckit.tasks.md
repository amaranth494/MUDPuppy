---
name: speckit.tasks
description: Generate actionable tasks from a technical plan
agent: speckit
workflow: ../.kilocode/workflows/speckit.tasks.md
handoffs:
  - label: Implement
    agent: speckit.implement
    prompt: Execute the implementation tasks. I am building with...
---

## Usage

/speckit.tasks [technical plan]

## Description

Generate actionable tasks from a technical implementation plan. This command:

1. Loads the technical plan
2. Breaks down implementation steps into discrete tasks
3. Estimates effort and assigns priorities
4. Creates task descriptions with acceptance criteria
5. Organizes tasks into categories and milestones

## Example

/speckit.tasks Generate tasks for user authentication implementation
