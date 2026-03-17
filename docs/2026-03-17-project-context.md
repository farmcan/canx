# CanX Project Context

**Date:** 2026-03-17

## Why this repo exists

The user wants a development accelerator where:

- one Codex instance acts as architect / supervisor
- that supervisor delegates to other Codex workers
- workers report back through a controlled loop
- reviews and validation gates happen before integration

This should be reusable across projects and not mixed into `Tradex` business code.

## Core pain point

The current single-agent workflow is too centered on human-agent interaction:

- a single agent carries too much context
- execution slows down as the conversation grows
- the same agent is expected to plan, implement, review, verify, and explain

The user wants the default workflow to become:

- AI agents collaborate with other AI agents
- humans define goals and step in for critical choices
- development moves through bounded loops with clear gates

## Desired outcome

Build a thin orchestration layer that can:

- read specs and plans from a target repository
- split work into bounded tasks
- dispatch worker Codex sessions
- collect results
- run review passes
- trigger validation commands
- stop automatically when success or abort conditions are met

## Design constraint

Avoid reinventing wheels:

- reuse Codex `app-server` or CLI surfaces
- reuse existing handoff/supervisor patterns
- focus custom code on orchestration, boundaries, and workflow control

## First downstream user

- `Tradex`

## Language choice

- primary language: `Go`

## Expected first milestone

- define orchestrator roles
- define task and loop models
- define a Codex adapter boundary
- support a minimal bounded supervisor loop
