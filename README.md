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

## In one line

- automatically reads the repository context
- still needs a clear goal prompt from you
- works best today for small, practical, fast-iteration development loops

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
- `cmd/canxd serve`: local dashboard and read-only API
- `internal/codex`: Codex CLI / app-server adapters
- `internal/loop`: bounded supervisor loop
- `internal/tasks`: task graph, ownership, status
- `internal/review`: review and gate policies
- `internal/runlog`: session reports, run records, and event streams

## Reference docs

- Start here: `START_HERE.md`
- Docs index: `docs/README.md`
- Product intent: `docs/2026-03-18-product-intent.md`
- Runbook: `docs/runbook.md`
- Testing methods: `docs/testing-methods.md`
- Prompt templates: `docs/prompt-templates.md`
- Collaboration room design: `docs/2026-03-19-collab-room-design.md`
- Evaluation landscape: `docs/research/2026-03-18-evaluation-landscape.md`
- Codex/OpenClaw/Ralph/ACP notes: `docs/research/2026-03-18-codex-openclaw-ralph-acp.md`

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
4. `docs/README.md`
5. `docs/runbook.md`
6. `docs/prompt-templates.md`

## Run the MVP

```bash
go run ./cmd/canxd -goal "ship canx mvp" -max-turns 2 -repo .
```

Expected output shape:

```text
canx decision=... reason=... turns=... tasks=... session=... workspace=... docs=...
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

Inspect persisted runs and events through the local dashboard:

```bash
go run ./cmd/canxd serve -repo .
```

Then open `http://127.0.0.1:8090`.

## Current MVP2 shape

The current local MVP now includes:

- Ralph-lite bounded loop control
- lightweight task planning wired into the live engine
- Codex runner abstraction
- lightweight session registry inspired by ACP/session models
- file-backed run/event persistence under `.canx/runs/`
- minimal local dashboard for runs, tasks, and event streams
- fast smoke evals

## Fast eval

Run the lightweight smoke suite:

```bash
go test ./evals/smoke -v
```

Run the local agentic eval suite:

```bash
go test ./evals/agentic -v
```

Run the real Codex eval smoke:

```bash
make eval-real
```

Generate local report files:

```bash
make report
make report-real
```
