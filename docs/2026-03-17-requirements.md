# CanX Requirements

**Date:** 2026-03-17

## One-paragraph brief

`CanX` is a `Go`-based orchestration repository for running a bounded, multi-agent software delivery loop around Codex. One supervisor agent reads a target repository’s specs and plans, delegates work to scoped worker agents, collects results, runs review and validation gates, and decides whether to continue, stop, or escalate.

## Basic user needs

The system should help the user:

- accelerate software iteration with multiple Codex agents
- keep architecture and product intent stable across fresh agent sessions
- avoid uncontrolled “infinite loops” by enforcing bounded execution
- separate infrastructure concerns from downstream business repositories
- reuse existing tools instead of rebuilding generic agent frameworks

## Agent-friendly repository goals

A fresh agent should be able to understand the repository quickly by reading a small number of files and seeing:

- what this repository is for
- what it is not for
- what the first milestone is
- which modules own which responsibilities
- which external projects are being reused
- how to validate changes

## Core functional requirements

- define a supervisor role
- define worker and reviewer roles
- define bounded loop state and stop conditions
- define a Codex integration boundary
- define task ownership and status tracking
- define validation and review gates before integration

## Non-functional requirements

- clear structure
- small interfaces
- low ceremony for iteration
- testable core logic
- repo-local documentation for new agents

## First milestone requirements

- minimal `Go` service skeleton
- repository docs that preserve intent
- documented external references
- module boundaries for supervisor loop, task model, Codex adapter, and review flow

## Out of scope for the first milestone

- distributed execution
- complex UI
- generalized workflow automation beyond coding tasks
- replacing Codex itself
