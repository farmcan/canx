# Multi-Codex Scheduler Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a practical multi-worker scheduler to `CanX` so a single goal can run independent Codex tasks in parallel and allow controlled worker-requested child task spawning.

**Architecture:** Upgrade `internal/loop.Engine` from a single-active-task loop into a bounded scheduler that manages multiple task sessions under supervisor control. Keep the existing `codex.Runner` abstraction, extend task/config models and marker parsing, and persist parent-child scheduling events into the existing runlog/session model.

**Tech Stack:** Go, existing `codex.Runner` abstraction, `internal/loop`, `internal/tasks`, `internal/runlog`, `cmd/canxd`, Go tests, mock runner evals

---

## File Map

### Existing files to modify

- `internal/loop/model.go`
  - Add scheduler-related config fields and validation defaults.
- `internal/tasks/task.go`
  - Extend task metadata for parent-child ownership, spawn depth, and planned file scopes.
- `internal/loop/engine.go`
  - Replace single-task progression with bounded scheduling, spawn handling, conflict checks, and new events.
- `internal/loop/engine_test.go`
  - Add focused scheduler tests, spawn-limit tests, and conflict-control tests.
- `internal/tasks/planner.go`
  - Normalize planned file metadata if needed for planner-produced tasks.
- `internal/tasks/codx_planner.go`
  - Update planner prompt/output expectations so tasks can optionally declare file scope.
- `internal/runlog/events.go`
  - Extend event payloads and persisted run/task metadata for parent-child scheduling.
- `internal/runlog/events_test.go`
  - Cover new event serialization and persisted task metadata.
- `cmd/canxd/main.go`
  - Add new CLI flags and wire defaults into `loop.Config`.
- `cmd/canxd/main_test.go`
  - Cover new CLI defaults, explicit flags, and persisted run summaries.
- `evals/agentic/suite_test.go`
  - Add smoke coverage for parallel task execution and controlled spawn flow.
- `docs/runbook.md`
  - Add commands for the new scheduler-related smoke paths.
- `docs/ai-agent-context.md`
  - Update current priority/state wording once implementation lands.

### New files to create

- `internal/loop/scheduler.go`
  - Focused scheduling helpers: runnable task selection, concurrency gating, spawn decisions, conflict checks.
- `internal/loop/markers.go`
  - Parse structured `stop` and `spawn` markers cleanly outside `engine.go`.
- `internal/loop/scheduler_test.go`
  - Unit tests for scheduling helpers and spawn/conflict logic.

## Chunk 1: Config And Task Model

### Task 1: Extend loop config for bounded concurrency

**Files:**
- Modify: `internal/loop/model.go`
- Test: `internal/loop/model_test.go`

- [ ] **Step 1: Write the failing config validation tests**

Add tests for:
- default-safe config with `MaxWorkers=2`, `MaxSpawnDepth=1`, `MaxChildrenPerTask=2`
- rejecting `MaxWorkers <= 0`
- rejecting `MaxSpawnDepth < 0`
- rejecting `MaxChildrenPerTask < 0`

- [ ] **Step 2: Run the focused test**

Run: `go test ./internal/loop -run 'TestConfig' -v`
Expected: FAIL because the new fields and validation rules do not exist yet.

- [ ] **Step 3: Implement minimal config changes**

Update `internal/loop/model.go`:
- add `MaxWorkers int`
- add `MaxSpawnDepth int`
- add `MaxChildrenPerTask int`
- keep existing validation intact
- add validation for new fields

- [ ] **Step 4: Re-run focused config tests**

Run: `go test ./internal/loop -run 'TestConfig' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/loop/model.go internal/loop/model_test.go
git commit -m "feat: add bounded scheduler config"
```

### Task 2: Extend task metadata for scheduler ownership and conflict control

**Files:**
- Modify: `internal/tasks/task.go`
- Test: `internal/tasks/task_test.go`

- [ ] **Step 1: Write failing task normalization/validation tests**

Add tests covering:
- parent/child metadata survives normalization
- empty `PlannedFiles` is allowed
- existing validation behavior remains unchanged

- [ ] **Step 2: Run the focused test**

Run: `go test ./internal/tasks -run 'TestTask' -v`
Expected: FAIL because the new fields do not exist.

- [ ] **Step 3: Implement minimal task model changes**

Add fields:
- `ParentTaskID string`
- `SpawnDepth int`
- `OwnerSessionID string`
- `DependsOn []string`
- `PlannedFiles []string`

Keep backward compatibility with existing `FilesChanged`, `Summary`, and status fields.

- [ ] **Step 4: Re-run task tests**

Run: `go test ./internal/tasks -run 'TestTask' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tasks/task.go internal/tasks/task_test.go
git commit -m "feat: extend task metadata for scheduler"
```

## Chunk 2: Structured Marker Parsing

### Task 3: Parse structured stop and spawn markers outside engine

**Files:**
- Create: `internal/loop/markers.go`
- Create: `internal/loop/markers_test.go`
- Modify: `internal/loop/engine.go`

- [ ] **Step 1: Write failing parser tests**

Cover:
- plain `[canx:stop]`
- structured `[canx:stop:{...}]`
- structured `[canx:spawn:{...}]`
- invalid JSON payload rejected cleanly
- multiple markers in a single output prefer valid structured payloads

- [ ] **Step 2: Run focused parser tests**

Run: `go test ./internal/loop -run 'TestParse.*Marker' -v`
Expected: FAIL because `markers.go` does not exist.

- [ ] **Step 3: Implement minimal parser**

Add helpers in `internal/loop/markers.go`:
- `parseStopPayload(output string) *stopPayload`
- `parseSpawnRequests(output string) []spawnRequest`
- `hasStopSignal(output string) bool`
- `hasEscalateSignal(output string) bool`

Move marker parsing out of `engine.go` so scheduler logic can reuse it.

- [ ] **Step 4: Re-run parser tests**

Run: `go test ./internal/loop -run 'TestParse.*Marker' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/loop/markers.go internal/loop/markers_test.go internal/loop/engine.go
git commit -m "feat: parse structured scheduler markers"
```

## Chunk 3: Scheduler Helpers

### Task 4: Add reusable scheduling and conflict helpers

**Files:**
- Create: `internal/loop/scheduler.go`
- Create: `internal/loop/scheduler_test.go`
- Modify: `internal/loop/engine.go`

- [ ] **Step 1: Write failing scheduler-helper tests**

Cover:
- selecting runnable tasks up to `MaxWorkers`
- blocking tasks with overlapping `PlannedFiles`
- allowing tasks with disjoint `PlannedFiles`
- rejecting spawn when depth exceeds `MaxSpawnDepth`
- rejecting spawn when child count exceeds `MaxChildrenPerTask`

- [ ] **Step 2: Run focused helper tests**

Run: `go test ./internal/loop -run 'TestScheduler' -v`
Expected: FAIL because helper code does not exist.

- [ ] **Step 3: Implement minimal helper layer**

Add helpers such as:
- `selectRunnableTasks(tasks []tasks.Task, maxWorkers int) []int`
- `tasksConflict(a, b tasks.Task) bool`
- `canApproveSpawn(parent tasks.Task, tasks []tasks.Task, cfg Config) (bool, string)`
- `childCount(tasks []tasks.Task, parentID string) int`

Keep the helpers pure where possible so they are cheap to test.

- [ ] **Step 4: Re-run helper tests**

Run: `go test ./internal/loop -run 'TestScheduler' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/loop/scheduler.go internal/loop/scheduler_test.go internal/loop/engine.go
git commit -m "feat: add scheduler conflict and spawn helpers"
```

## Chunk 4: Parallel Initial Task Execution

### Task 5: Upgrade engine to run initial independent tasks in parallel

**Files:**
- Modify: `internal/loop/engine.go`
- Modify: `internal/loop/engine_test.go`
- Modify: `internal/runlog/events.go`
- Modify: `internal/runlog/events_test.go`

- [ ] **Step 1: Write failing engine tests for parallel execution**

Add tests covering:
- two independent tasks both progress within one scheduler cycle
- each running task gets its own `OwnerSessionID`
- run/task events reflect multiple task starts

Use a staged mock runner so the test can observe overlapping execution.

- [ ] **Step 2: Run focused engine tests**

Run: `go test ./internal/loop -run 'TestEngine.*Parallel' -v`
Expected: FAIL because the engine is still single-active-task.

- [ ] **Step 3: Implement minimal parallel scheduler**

Refactor `internal/loop/engine.go`:
- replace `firstActiveTaskIndex` loop with scheduler selection
- spawn a worker goroutine per runnable task, bounded by `MaxWorkers`
- collect results and update task/session state deterministically
- keep validation/review gate per task result
- preserve budget and turn timeout semantics

Do not add dynamic spawn yet in this task.

- [ ] **Step 4: Re-run focused engine tests**

Run: `go test ./internal/loop -run 'TestEngine.*Parallel' -v`
Expected: PASS

- [ ] **Step 5: Re-run broader loop package tests**

Run: `go test ./internal/loop -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/loop/engine.go internal/loop/engine_test.go internal/runlog/events.go internal/runlog/events_test.go
git commit -m "feat: schedule independent tasks in parallel"
```

## Chunk 5: Controlled Dynamic Spawn

### Task 6: Accept worker spawn requests through supervisor approval

**Files:**
- Modify: `internal/loop/engine.go`
- Modify: `internal/loop/engine_test.go`
- Modify: `internal/tasks/task.go`
- Modify: `internal/runlog/events.go`
- Modify: `internal/runlog/events_test.go`

- [ ] **Step 1: Write failing engine tests for spawn approval**

Cover:
- worker emits a valid `spawn request`
- supervisor creates a child task with `ParentTaskID` and incremented `SpawnDepth`
- rejected spawn feeds a reason back into the parent task state/prompt path
- child task respects concurrency and conflict rules

- [ ] **Step 2: Run focused spawn tests**

Run: `go test ./internal/loop -run 'TestEngine.*Spawn' -v`
Expected: FAIL because spawn requests are not handled.

- [ ] **Step 3: Implement minimal spawn flow**

In `engine.go`:
- parse `spawn requests` from worker output
- approve/reject with scheduler helpers
- append child tasks when approved
- emit `task_spawn_requested`, `task_spawn_approved`, or `task_spawn_rejected`
- attach rejection reasons so the parent can adapt next turn

Keep worker-to-worker communication disallowed.

- [ ] **Step 4: Re-run focused spawn tests**

Run: `go test ./internal/loop -run 'TestEngine.*Spawn' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/loop/engine.go internal/loop/engine_test.go internal/runlog/events.go internal/runlog/events_test.go internal/tasks/task.go
git commit -m "feat: add supervisor-approved task spawn"
```

## Chunk 6: CLI Wiring And Planner Surface

### Task 7: Add scheduler flags and defaults to CLI

**Files:**
- Modify: `cmd/canxd/main.go`
- Modify: `cmd/canxd/main_test.go`

- [ ] **Step 1: Write failing CLI tests**

Cover:
- default `run` path sets safe scheduler defaults
- explicit `-max-workers`
- explicit `-max-spawn-depth`
- explicit `-max-children-per-task`

- [ ] **Step 2: Run focused CLI tests**

Run: `go test ./cmd/canxd -run 'TestRun|TestParseFlags' -v`
Expected: FAIL because the flags do not exist.

- [ ] **Step 3: Implement minimal CLI wiring**

Update `parseFlags()` and `Options`/`loop.Config` wiring so the new scheduler config reaches `Engine.Run`.

- [ ] **Step 4: Re-run focused CLI tests**

Run: `go test ./cmd/canxd -run 'TestRun|TestParseFlags' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/canxd/main.go cmd/canxd/main_test.go
git commit -m "feat: expose scheduler controls on cli"
```

### Task 8: Let planner tasks optionally carry file scope metadata

**Files:**
- Modify: `internal/tasks/codx_planner.go`
- Modify: `internal/tasks/codx_planner_test.go`

- [ ] **Step 1: Write failing planner tests**

Cover:
- planner JSON including `planned_files`
- missing `planned_files` remains valid
- normalization preserves backward compatibility

- [ ] **Step 2: Run focused planner tests**

Run: `go test ./internal/tasks -run 'TestCodxPlanner' -v`
Expected: FAIL because planner/task mapping does not cover this field.

- [ ] **Step 3: Implement minimal planner update**

Adjust planner prompt and parsing so `planned_files` may be returned and stored in `Task.PlannedFiles`.

- [ ] **Step 4: Re-run planner tests**

Run: `go test ./internal/tasks -run 'TestCodxPlanner' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tasks/codx_planner.go internal/tasks/codx_planner_test.go
git commit -m "feat: allow planner file scopes for scheduler"
```

## Chunk 7: Evals, Docs, And Final Verification

### Task 9: Add smoke coverage for multi-worker scheduler behavior

**Files:**
- Modify: `evals/agentic/suite_test.go`

- [ ] **Step 1: Write failing eval cases**

Add cases for:
- multi-task parallel run
- approved spawn creates child task
- conflicting planned files stay sequential

- [ ] **Step 2: Run focused eval tests**

Run: `go test ./evals/agentic -run 'TestAgenticQuickSuite' -v`
Expected: FAIL because the new cases are unimplemented.

- [ ] **Step 3: Implement minimal eval support**

Use mock/staged runners instead of real Codex so the suite stays fast and deterministic.

- [ ] **Step 4: Re-run focused eval tests**

Run: `go test ./evals/agentic -run 'TestAgenticQuickSuite' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add evals/agentic/suite_test.go
git commit -m "test: add scheduler smoke evals"
```

### Task 10: Update runbook/context docs and verify the whole repo

**Files:**
- Modify: `docs/runbook.md`
- Modify: `docs/ai-agent-context.md`

- [ ] **Step 1: Update docs**

Document:
- new CLI flags
- expected scheduler behavior
- new focused test commands

- [ ] **Step 2: Run formatting**

Run: `gofmt -w internal/loop/*.go internal/tasks/*.go internal/runlog/*.go cmd/canxd/*.go evals/agentic/*.go`
Expected: files formatted with no output

- [ ] **Step 3: Run focused package tests**

Run: `go test ./internal/loop ./internal/tasks ./internal/runlog ./cmd/canxd ./evals/agentic -v`
Expected: PASS

- [ ] **Step 4: Run full repository verification**

Run: `make build`
Expected: PASS

Run: `make test`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add docs/runbook.md docs/ai-agent-context.md
git commit -m "docs: describe multi-codex scheduler"
```

## Execution Notes

- Keep `engine.go` from becoming the new monolith; move pure scheduling and marker logic into focused helper files.
- Preserve current semantics where validation failure feeds back into later prompts.
- Avoid speculative abstractions for generic graphs or agent chats.
- Prefer deterministic tests using staged/mock runners over sleeps.
- If parallel scheduling causes nondeterministic test flakes, add explicit event synchronization in tests rather than relaxing assertions.

Plan complete and saved to `docs/superpowers/plans/2026-03-20-multi-codex-scheduler.md`. Ready to execute?
