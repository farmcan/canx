# Codex / OpenClaw / Ralph / ACP Notes

**Date:** 2026-03-18

## Why this note exists

`CanX` should not drift into inventing a custom agent framework without clear need.

This note captures the most relevant ideas to reuse from:

- Codex `app-server`
- OpenClaw ACP
- Ralph-style orchestration
- ACP session protocol

## Codex: what to borrow

### Source

- OpenAI article: `https://openai.com/index/unlocking-the-codex-harness/`
- Local repo doc: `codex-rs/app-server/README.md` in the workspace

### High-value ideas

- **thread / turn separation**
  - thread = durable conversation identity
  - turn = one active unit of work
- **streamed lifecycle**
  - start
  - delta/events
  - completed
- **explicit review flow**
  - `review/start` exists as a first-class action
- **command execution as separate primitive**
  - `command/exec` is distinct from normal conversation turns

### What CanX should do

- keep `session` and `turn` separate in the design
- keep validation separate from worker generation
- prepare for an `AppServerRunner` later instead of overfitting to shell parsing

### What CanX should not do

- do not re-implement a full JSON-RPC app-server
- do not duplicate Codex thread persistence
- do not create a parallel runtime abstraction larger than needed for `exec`/`app-server`

## OpenClaw ACP: what to borrow

### Sources

- `https://docs.openclaw.ai/tools/acp-agents`
- `https://docs.openclaw.ai/cli/acp`
- local docs in `/Users/levi/wrksp/openclaw/docs/tools/acp-agents.md`

### High-value ideas

- **session identity matters**
  - ACP sessions map to stable session keys
- **persistent vs oneshot mode**
  - this is a useful first-order distinction
- **spawn / steer / close lifecycle**
  - simple, operator-friendly control surface
- **runtime routing**
  - route by session/runtime rather than by ad-hoc prompt tricks

### What CanX should do

- keep lightweight local session registry
- preserve `spawn`, `steer`, and `close` semantics
- keep session inspection easy for humans and AI

### What CanX should not do

- do not implement full ACP transport now
- do not copy OpenClaw’s channel/thread binding system
- do not expand into a general bot platform

## ACP protocol itself: what matters now

### Source

- `https://agentclientprotocol.com/protocol/session-setup`

### High-value ideas

- initialization first
- protocol capabilities negotiation
- session replay/update semantics
- clear session lifecycle messages

### Practical takeaway for CanX

Even though the local MVP is not speaking ACP yet, the design should stay compatible with:

- stable session identity
- explicit lifecycle transitions
- replayable session state

This argues for persisted session reports and structured session inspection.

## Ralph-style orchestration: what to borrow

### Sources

- `https://ralphworkflow.com/`
- community example: `https://github.com/alfredolopez80/multi-agent-ralph-loop`

### High-value ideas

- keep orchestration simple
- use a durable loop instead of giant one-shot prompts
- emphasize progress and quality gates
- use different agent roles when helpful

### What CanX should do

- stay close to a bounded `while true` loop
- keep a small number of states and clear stop conditions
- add minimal structure only when it reduces failure or improves visibility

### What CanX should not do

- do not turn MVP into a heavyweight workflow engine
- do not add complexity before a local loop proves useful

## Session visibility: why it matters

Both humans and AI need to inspect:

- current session id
- last summaries
- turns so far
- decision / reason
- planned tasks

Without this, debugging and self-improvement loops are weak.

This is why `CanX` should keep:

- persisted session reports
- CLI inspection
- structured JSON output

## Recommended next borrowing path

### Near-term

- borrow more from Ralph:
  - simple supervisor loop
  - lightweight role split
- borrow more from OpenClaw ACP:
  - session lifecycle semantics

### Mid-term

- borrow more from Codex app-server:
  - thread-aware runner
  - review/start style review runner
  - richer event streaming

## Bottom line

`CanX` should be:

- **Ralph-lite in control flow**
- **ACP-inspired in session lifecycle**
- **Codex-native in execution surface**

It should **not** become:

- another general agent framework
- another protocol stack
- another bot platform
