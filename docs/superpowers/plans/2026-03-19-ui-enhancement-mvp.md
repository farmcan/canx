# UI Enhancement MVP Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend the local CanX dashboard so humans and agents can inspect runs, tasks, sessions, and repo context from one place.

**Architecture:** Keep the current file-backed storage model. Add read-only JSON endpoints for task/session/context views, then upgrade the static dashboard UI to consume them. Do not add a frontend framework or real-time transport in this step.

**Tech Stack:** Go standard library HTTP, embedded static HTML/CSS/JS, existing `.canx` run/session files.

---

### Task 1: Add read-only inspect endpoints

**Files:**
- Modify: `cmd/canxd/server.go`
- Test: `cmd/canxd/server_test.go`
- Modify: `cmd/canxd/main.go`

- [ ] Write failing tests for session and context endpoints
- [ ] Add `GET /api/sessions/:id`
- [ ] Add `GET /api/context`
- [ ] Add `GET /api/runs/:id/tasks/:task_id`
- [ ] Run `go test ./cmd/canxd -v`

### Task 2: Improve dashboard layout and task/session views

**Files:**
- Modify: `cmd/canxd/ui/index.html`
- Modify: `cmd/canxd/ui/app.js`
- Modify: `cmd/canxd/ui/styles.css`

- [ ] Render runs as cards/list with clear status badges
- [ ] Render task list and selected task detail
- [ ] Render session metadata panel
- [ ] Render context viewer for `README.md`, `AGENTS.md`, and docs list
- [ ] Keep raw events visible for debugging

### Task 3: Document the UI flow and verify end to end

**Files:**
- Modify: `README.md`
- Modify: `docs/runbook.md`
- Modify: `docs/testing-methods.md`

- [ ] Document the new endpoints and panels
- [ ] Run `go test ./...`
- [ ] Run `go build ./...`
- [ ] Run one mock dashboard smoke and inspect API output
