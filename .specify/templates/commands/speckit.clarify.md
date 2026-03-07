---
name: speckit.clarify
description: Ask structured questions to de-risk ambiguous areas before planning
agent: speckit
workflow: ../.kilocode/workflows/speckit.clarify.md
handoffs:
  - label: Build Technical Plan
    agent: speckit.plan
    prompt: Create a plan with clarified requirements. I am building with...
---

## Usage

/speckit.clarify [ambiguous area]

## Description

Ask structured questions to de-risk ambiguous areas before planning. This command:

1. Identifies unclear or ambiguous requirements
2. Presents options with implications for each
3. Guides user to make informed decisions
4. Documents clarifications for future reference

## Example

/speckit.clarify What authentication methods should we support?
