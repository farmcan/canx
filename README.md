# CanX

AI-to-AI software delivery orchestration for bounded, multi-agent development.

## Purpose

`CanX` is a separate infrastructure repository for coordinating:

- one architect/supervisor agent
- multiple scoped implementation agents
- one or more review agents
- bounded execution loops with budget, test, and merge gates

It is designed to help projects such as `Tradex` iterate faster without mixing business logic and agent runtime logic in the same repository.

## Problem statement

Single-agent development is still too human-centric:

- one agent spends most of its time talking to a human instead of collaborating with other agents
- large repository context accumulates in one thread and slows execution down
- one agent is forced to juggle architecture, implementation, review, verification, and documentation at once

For larger projects, this makes AI-assisted development slower and less reliable than it should be.

`CanX` exists to turn AI development into an AI-to-AI delivery pipeline where humans define goals and approve important decisions, while agents handle the main execution loop.

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

- Start here: `START_HERE.md`
- Product intent: `docs/2026-03-18-product-intent.md`
- Project context: `docs/2026-03-17-project-context.md`
- Requirements: `docs/2026-03-17-requirements.md`
- MVP design: `docs/2026-03-17-canx-mvp-design.md`
- MVP plan: `docs/2026-03-17-canx-mvp-plan.md`
- Landscape analysis: `docs/2026-03-18-landscape-analysis.md`
- Research landscape: `docs/research/2026-03-17-orchestrator-landscape.md`

## Initial principle

Reuse existing building blocks:

- Codex `app-server`
- Codex CLI execution surface
- existing multi-agent patterns from OpenAI Agents, LangGraph, AutoGen, and similar projects

Build only the thin supervisor layer that is missing for your workflow.

## Agent quickstart

If you are a fresh agent session, read these files first:

1. `START_HERE.md`
2. `README.md`
3. `AGENTS.md`

## Run the MVP

```bash
go run ./cmd/canxd -goal "ship canx mvp" -max-turns 2 -repo .
```

Expected output shape:

```text
canx decision=... reason=... turns=... session=... workspace=... docs=...
```

For a fast local demo without invoking real Codex:

```bash
go run ./cmd/canxd -goal "ship canx mvp" -max-turns 2 -repo . -runner mock
```

Inspect persisted sessions after a run:

```bash
go run ./cmd/canxd -repo . sessions list
go run ./cmd/canxd -repo . sessions show <session-id>
```

## Current MVP2 shape

The current local MVP now includes:

- Ralph-lite bounded loop control
- lightweight task planning wired into the live engine
- Codex runner abstraction
- lightweight session registry inspired by ACP/session models
- fast smoke evals

## Fast eval

Run the lightweight smoke suite:

```bash
go test ./evals/smoke -v
```
