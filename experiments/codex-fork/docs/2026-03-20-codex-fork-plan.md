# Codex Fork Experiment Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build an isolated shell prototype that forks Codex work from an existing session file and writes a task packet plus launch command.

**Architecture:** Keep everything under `experiments/codex-fork/`. Use a small shell library for session parsing and packet generation, then a thin CLI wrapper to create packet files and optionally call `codex fork`.

**Tech Stack:** POSIX shell, `bash`, `jq`, Codex CLI

---

## Chunk 1: Shell Prototype

### Task 1: Add failing shell tests

**Files:**
- Create: `experiments/codex-fork/test/test.sh`
- Create: `experiments/codex-fork/docs/2026-03-20-codex-fork-plan.md`

- [ ] **Step 1: Write the failing test**
- [ ] **Step 2: Run test to verify it fails**
- [ ] **Step 3: Implement minimal shell library and CLI wrapper**
- [ ] **Step 4: Run test to verify it passes**

### Task 2: Document boundaries and usage

**Files:**
- Create: `experiments/codex-fork/README.md`
- Create: `experiments/codex-fork/docs/design.md`

- [ ] **Step 1: Describe intent, non-goals, and isolation rules**
- [ ] **Step 2: Document data flow and command usage**
- [ ] **Step 3: Link test command and expected behavior**
