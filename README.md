# CanX

Supervisor-driven multi-agent orchestration for Codex-based software development.

## Purpose

`CanX` is a separate infrastructure repository for coordinating:

- one architect/supervisor agent
- multiple scoped implementation agents
- one or more review agents
- bounded execution loops with budget, test, and merge gates

It is designed to help projects such as `Tradex` iterate faster without mixing business logic and agent runtime logic in the same repository.

## Non-goals

- not a replacement for Codex itself
- not a general-purpose workflow engine
- not a marketplace/business app

## Why a separate repo

- keeps product code isolated from agent infrastructure
- makes the orchestration layer reusable across projects
- avoids overfitting the runtime to `Tradex`
- keeps docs, prompts, and loop control easier to evolve

## Architecture direction

- `cmd/canxd`: orchestrator entrypoint
- `internal/codex`: Codex CLI / app-server adapters
- `internal/loop`: bounded supervisor loop
- `internal/tasks`: task graph, ownership, status
- `internal/review`: review and gate policies
- `internal/gitops`: branch, patch, validate, merge helpers

## Reference docs

- Project context: `docs/2026-03-17-project-context.md`
- Requirements: `docs/2026-03-17-requirements.md`
- Name and scope: `docs/2026-03-17-naming-and-positioning.md`
- Research landscape: `docs/research/2026-03-17-orchestrator-landscape.md`

## Initial principle

Reuse existing building blocks:

- Codex `app-server`
- Codex CLI execution surface
- existing multi-agent patterns from OpenAI Agents, LangGraph, AutoGen, and similar projects

Build only the thin supervisor layer that is missing for your workflow.

## Agent quickstart

If you are a fresh agent session, read these files first:

1. `README.md`
2. `AGENTS.md`
3. `docs/2026-03-17-project-context.md`
4. `docs/2026-03-17-requirements.md`
5. `docs/2026-03-17-naming-and-positioning.md`
