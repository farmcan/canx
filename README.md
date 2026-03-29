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
- Project context: `docs/ai-agent-context.md`
- Runbook: `docs/runbook.md`
- Testing methods: `docs/testing-methods.md`
- Prompt templates: `docs/prompt-templates.md`
- Framework comparison: `docs/2026-03-18-framework-comparison.md`
- Collaboration room design: `docs/2026-03-19-collab-room-design.md`
- UI observability design: `docs/2026-03-19-ui-observability-design.md`
- Scheduler design: `docs/superpowers/specs/2026-03-20-multi-codex-scheduler-design.md`
- App-server runner design: `docs/superpowers/specs/2026-03-20-appserver-runner-design.md`

## Initial principle

Reuse existing building blocks:

- Codex `app-server`
- Codex CLI execution surface
- existing multi-agent patterns from OpenAI Agents, LangGraph, AutoGen, and similar projects

Build only the thin supervisor layer that is missing for your workflow.

## Thin Workflow

`codex-fork/` is a first-class path in this repository for a much thinner delegation model:

- reuse native `codex fork`
- inherit a small amount of context from an existing session file
- hand work off through `task-packet.md`, `status.json`, and `result.md`

Use it when you want fast subtask delegation without pulling in the full CanX supervisor loop.

## Agent quickstart

If you are a fresh agent session, read these files first:

1. `START_HERE.md`
2. `README.md`
3. `AGENTS.md`
4. `docs/ai-agent-context.md`
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

The dashboard currently shows:

- runs with status and reason
- task list plus task detail
- session list plus session detail
- session metadata and report detail
- docs content viewer
- raw event stream
- repo context (`README.md`, `AGENTS.md`, docs inventory)
- room/message panel for human instructions

The local UI now has two modes:

- `Backstage`: the engineering console for runs, tasks, sessions, events, and rooms
- `Frontstage`: a presentation-oriented scene view that maps the active run into a staged control-room layout

Frontstage currently ships with CSS placeholders first, so it works before final art assets are ready. Asset naming guidance and prompt templates live in:

- `cmd/canxd/ui/assets/README.md`
- `docs/2026-03-29-frontstage-asset-prompts.md`

## Current MVP2 shape

The current local MVP now includes:

- Ralph-lite bounded loop control
- lightweight task planning wired into the live engine
- Codex runner abstraction with `exec`, `mock`, and minimal `appserver` modes
- bounded multi-worker scheduler with task-level parallelism
- supervisor-approved dynamic child task spawning
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
