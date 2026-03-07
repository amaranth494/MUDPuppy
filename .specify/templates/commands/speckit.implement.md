---
name: speckit.implement
description: Execute the implementation tasks
agent: speckit
workflow: ../.kilocode/workflows/speckit.implement.md
---

## Usage

/speckit.implement [task list]

## Description

Execute the implementation tasks. This command:

1. Loads the task list from speckit.tasks
2. Executes each task according to the implementation plan
3. Validates implementation against requirements
4. Reports progress and any blockers
5. Updates task status as completed

## Example

/speckit.implement Start implementing user authentication
