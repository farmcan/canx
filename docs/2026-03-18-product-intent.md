# CanX Product Intent

**Date:** 2026-03-18

## One sentence

`CanX` exists to move software delivery from slow, human-centric single-agent chat loops to fast, bounded, AI-to-AI collaboration loops led by multiple Codex workers.

## The real goal

The long-term goal is not simply “use Codex better”.

The real goal is:

- multiple Codex agents collaborate on one software task
- a supervisor coordinates them
- work keeps moving with minimal human intervention
- the system becomes good enough to help develop downstream repos like `Tradex`
- eventually, `CanX` should help improve `CanX` itself

## Why this repo exists

Single-agent workflows break down on larger projects:

- too much context sits in one thread
- one agent spends most of its time talking to a human
- architecture, coding, testing, and review get mixed together
- iteration slows down as context grows

`CanX` exists to make the default workflow:

- AI supervisor
- AI workers
- AI reviewer
- bounded validation loop
- human only for goals, approvals, and blockers

## Ralph-lite MVP

The MVP should borrow the useful part of `Ralph`:

- keep the control loop simple
- keep the loop moving
- avoid heavyweight protocol work too early

Practical interpretation for `CanX`:

- start with a bounded `while true` style loop
- load repo context
- build a prompt
- run Codex
- run validation
- decide continue / stop / escalate

But unlike a naive infinite shell loop, `CanX` must add:

- max turns
- timeout
- validation gates
- stop markers
- structured logs

## Borrow, don’t rebuild

`CanX` should reuse:

- Codex CLI / `codex exec`
- Codex `app-server` later
- OpenClaw / ACP ideas for session and runtime boundaries
- existing multi-agent patterns from LangGraph, OpenAI Agents, AutoGen, and similar systems

`CanX` should **not** try to rebuild:

- a whole generic agent framework
- a model runtime
- a large protocol stack before the local MVP works

## Fresh-agent expectation

When a new agent opens this repo, it should quickly understand:

- this is not a chatbot shell
- this is not a business app
- this is an AI-to-AI software delivery orchestrator
- the current implementation target is a Ralph-lite local MVP
- the end goal is multi-Codex collaboration with self-improving development loops
