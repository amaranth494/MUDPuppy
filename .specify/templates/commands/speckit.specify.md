---
name: speckit.specify
description: Create or update the feature specification from a natural language feature description
agent: speckit
workflow: ../.kilocode/workflows/speckit.specify.md
handoffs:
  - label: Build Technical Plan
    agent: speckit.plan
    prompt: Create a plan for the spec. I am building with...
  - label: Clarify Spec Requirements
    agent: speckit.clarify
    prompt: Clarify specification requirements
    send: true
---

## Usage

/speckit.specify [feature description]

## Description

Create or update a feature specification based on a natural language description. This command:

1. Generates a short branch name from the feature description
2. Creates a new feature branch and specification file
3. Fills in the specification template with requirements, user scenarios, and success criteria
4. Validates the specification quality

## Example

/speckit.specify Add user authentication with OAuth2 login
