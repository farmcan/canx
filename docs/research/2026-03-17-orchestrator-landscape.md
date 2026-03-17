# Orchestrator Landscape

**Date:** 2026-03-17

## Core judgment

There are already strong building blocks for multi-agent orchestration. The gap is not “can agents exist”, but “how to combine them cleanly for coding workflows with bounded loops, validation, and repo-aware coordination”.

## What to reuse

### Codex app-server

- Local repo reference: `../codex/codex-rs/app-server/README.md` in the existing workspace
- Why it matters:
  - JSON-RPC interface
  - turn lifecycle
  - event streaming
  - approval flow support
  - better long-term integration target than ad-hoc CLI scraping

### Codex CLI / exec surface

- Why it matters:
  - easiest short-term way to dispatch worker jobs
  - lower implementation overhead for MVP supervisor loops

## External references

### Codex-focused

- `codex-subagents-mcp`
  - `https://github.com/leonardsellem/codex-subagents-mcp`
  - useful as a reference for Codex subagent wrapping

- `agentpipe`
  - `https://github.com/kevinelliott/agentpipe`
  - useful for CLI-agent orchestration patterns and Codex execution ideas

### Multi-agent orchestration patterns

- OpenAI Agents handoffs
  - `https://openai.github.io/openai-agents-python/ref/handoffs/`
  - useful for explicit handoff semantics

- `openai-agents-go`
  - `https://github.com/nlpodyssey/openai-agents-go`
  - useful for Go-oriented agent structure and guardrails

- LangGraph Supervisor
  - `https://github.com/langchain-ai/langgraph-supervisor-py`
  - useful for manager-worker patterns

- AutoGen
  - `https://github.com/microsoft/autogen`
  - useful for multi-agent conversation orchestration

- OpenHands
  - `https://github.com/All-Hands-AI/OpenHands`
  - useful for coding-agent platform patterns

## What not to rebuild

- generic LLM runtime layers already handled by Codex
- broad agent framework semantics already solved in existing SDKs
- arbitrary workflow-engine features unrelated to coding tasks

## What this repo should own

- bounded supervisor loop
- task ownership and dispatch model
- Codex adapter interface
- validation and review gates
- Git-aware integration flow
