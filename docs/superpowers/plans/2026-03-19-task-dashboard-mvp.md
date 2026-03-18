# Task Dashboard MVP Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a practical run/task/session observability MVP to CanX with persisted events, inspectable APIs, and a minimal UI, while fixing the current loop regression.

**Architecture:** Persist run-level and event-level state under `.canx/`, expose it through small HTTP JSON endpoints, and render a static HTML dashboard from the same binary. Keep the storage model file-backed and append-only enough for local use. Fix the loop stop regression first so the dashboard reflects correct state.

**Tech Stack:** Go standard library, existing `cmd/canxd`, JSON files under `.canx/`, static HTML/CSS/JS embedded in Go.

---

## File structure

- Modify: `internal/loop/engine.go` — fix validation/stop regression and emit structured events
- Modify: `internal/loop/engine_test.go` — regression coverage for validation-stop and event emission
- Modify: `evals/smoke/smoke_test.go` — restore smoke expectations
- Create: `internal/runlog/events.go` — run/task/session/event model and file store
- Create: `internal/runlog/events_test.go` — store/read/query coverage
- Modify: `internal/runlog/store.go` — extend session report with run linkage
- Modify: `cmd/canxd/main.go` — add run IDs, HTTP server command, inspect endpoints
- Create: `cmd/canxd/server.go` — lightweight API and static UI serving
- Create: `cmd/canxd/server_test.go` — API handler coverage
- Create: `cmd/canxd/ui/index.html` — minimal dashboard shell
- Create: `cmd/canxd/ui/app.js` — fetch runs/tasks/events and render
- Create: `cmd/canxd/ui/styles.css` — minimal readable styling
- Modify: `docs/runbook.md` — dashboard run instructions
- Modify: `docs/testing-methods.md` — add API/UI verification path
- Modify: `README.md` — mention dashboard and event log

## Chunk 1: Fix loop regression and define event model

### Task 1: Restore correct stop behavior

**Files:**
- Modify: `internal/loop/engine.go`
- Test: `internal/loop/engine_test.go`
- Test: `evals/smoke/smoke_test.go`

- [ ] Write/adjust failing tests for validation stop behavior
- [ ] Make validation pass mark the active task done
- [ ] Ensure decision becomes `validation passed` when final active task completes
- [ ] Run `go test ./internal/loop ./evals/smoke -v`

### Task 2: Add run/event store

**Files:**
- Create: `internal/runlog/events.go`
- Test: `internal/runlog/events_test.go`
- Modify: `internal/runlog/store.go`

- [ ] Add `RunRecord`, `TaskEvent`, `TurnEvent`, and `EventStore`
- [ ] Persist run summary and append event stream under `.canx/runs/<run-id>/`
- [ ] Link session report to `run_id`
- [ ] Run `go test ./internal/runlog -v`

## Chunk 2: API and dashboard

### Task 3: Add inspect API

**Files:**
- Modify: `cmd/canxd/main.go`
- Create: `cmd/canxd/server.go`
- Test: `cmd/canxd/server_test.go`

- [ ] Add `serve` subcommand
- [ ] Add JSON endpoints for runs, run detail, and event stream
- [ ] Keep handlers read-only
- [ ] Run `go test ./cmd/canxd -v`

### Task 4: Add minimal static dashboard

**Files:**
- Create: `cmd/canxd/ui/index.html`
- Create: `cmd/canxd/ui/app.js`
- Create: `cmd/canxd/ui/styles.css`
- Modify: `cmd/canxd/server.go`

- [ ] Serve a single-page dashboard from embedded assets
- [ ] Show runs list, selected run summary, tasks, and raw event stream
- [ ] Keep it dependency-free and local-only
- [ ] Run `go test ./cmd/canxd -v`

## Chunk 3: Wire engine output and docs

### Task 5: Persist events from live runs

**Files:**
- Modify: `internal/loop/engine.go`
- Modify: `cmd/canxd/main.go`
- Modify: `internal/runlog/events.go`

- [ ] Create run record at run start
- [ ] Append task/session/turn/validation/decision events during execution
- [ ] Persist final run status and reason
- [ ] Run `go test ./internal/loop ./internal/runlog ./cmd/canxd -v`

### Task 6: Update docs and full verification

**Files:**
- Modify: `README.md`
- Modify: `docs/runbook.md`
- Modify: `docs/testing-methods.md`

- [ ] Document event log locations and dashboard usage
- [ ] Document quick verification commands
- [ ] Run `go test ./...`
- [ ] Run `go build ./...`
- [ ] Run `go run ./cmd/canxd serve -repo .` as a smoke check
