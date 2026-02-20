---
name: speckit.analyze
description: Cross-artifact consistency and alignment report
agent: speckit
workflow: ../.kilocode/workflows/speckit.analyze.md
handoffs:
  - label: Generate Tasks
    agent: speckit.tasks
    prompt: Generate tasks based on the analyzed artifacts. I am building with...
---

## Usage

/speckit.analyze [artifacts to analyze]

## Description

Generate a cross-artifact consistency and alignment report. This command:

1. Analyzes specification, plan, and tasks for consistency
2. Identifies gaps or conflicts between artifacts
3. Reports alignment with project constitution
4. Provides recommendations for fixes

## Example

/speckit.analyze Check alignment between spec and implementation tasks
