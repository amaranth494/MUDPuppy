---
name: speckit.constitution
description: Create or update the project constitution from interactive or provided principle inputs
agent: speckit
workflow: ../.kilocode/workflows/speckit.constitution.md
handoffs:
  - label: Build Specification
    agent: speckit.specify
    prompt: Implement the feature specification based on the updated constitution. I want to build...
---

## Usage

/speckit.constitution [principles]

## Description

Create or update the project constitution at `.specify/memory/constitution.md`. This file is a TEMPLATE containing placeholder tokens in square brackets (e.g. `[PROJECT_NAME]`, `[PRINCIPLE_1_NAME]`). The command:

1. Loads the existing constitution template
2. Collects/derives concrete values for placeholder tokens
3. Fills the template precisely with project-specific principles
4. Propagates any amendments across dependent artifacts
5. Produces a Sync Impact Report

## Example

/speckit.constitution Add principles for test-driven development and open source licensing
