# CanX MVP Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first local single-machine `CanX` orchestrator that can load repository context, define bounded tasks, invoke Codex through an `exec` runner, apply a lightweight review gate, and stop with a clear loop decision.

**Architecture:** The MVP stays deliberately thin. `CanX` owns task models, loop control, workspace context loading, review gates, and run logging, while Codex remains the execution engine behind a `Runner` interface. The first usable path is `ExecRunner`, with interfaces shaped so an `AppServerRunner` can be added later without rewriting the control flow.

**Tech Stack:** `Go 1.25+`, standard library, table-driven tests

---

## Chunk 1: Core Models

### Task 1: Add task model

**Files:**
- Create: `internal/tasks/task.go`
- Test: `internal/tasks/task_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestTaskValidateRequiresIDAndGoal(t *testing.T) {
	task := Task{}
	if err := task.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tasks -run TestTaskValidateRequiresIDAndGoal -v`
Expected: FAIL with missing package or symbol errors

- [ ] **Step 3: Write minimal implementation**

```go
type Task struct {
	ID   string
	Goal string
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/tasks -run TestTaskValidateRequiresIDAndGoal -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tasks/task.go internal/tasks/task_test.go
git commit -m "feat: add canx task model"
```

### Task 2: Add loop config and decision model

**Files:**
- Create: `internal/loop/model.go`
- Test: `internal/loop/model_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestConfigValidateRequiresGoalAndMaxTurns(t *testing.T) {
	cfg := Config{}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/loop -run TestConfigValidateRequiresGoalAndMaxTurns -v`
Expected: FAIL with missing package or symbol errors

- [ ] **Step 3: Write minimal implementation**

```go
type Config struct {
	Goal     string
	MaxTurns int
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/loop -run TestConfigValidateRequiresGoalAndMaxTurns -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/loop/model.go internal/loop/model_test.go
git commit -m "feat: add canx loop model"
```

## Chunk 2: Workspace and Runner Boundaries

### Task 3: Add workspace context loader

**Files:**
- Create: `internal/workspace/context.go`
- Test: `internal/workspace/context_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestLoadReadsReadmeAndAgents(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "README.md"), []byte("readme"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("agents"), 0o644)

	ctx, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if ctx.Readme == "" || ctx.Agents == "" {
		t.Fatal("expected readme and agents content")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/workspace -run TestLoadReadsReadmeAndAgents -v`
Expected: FAIL with missing package or symbol errors

- [ ] **Step 3: Write minimal implementation**

```go
type Context struct {
	Readme string
	Agents string
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/workspace -run TestLoadReadsReadmeAndAgents -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/workspace/context.go internal/workspace/context_test.go
git commit -m "feat: add workspace context loader"
```

### Task 4: Add Codex runner interface and exec runner

**Files:**
- Create: `internal/codex/runner.go`
- Create: `internal/codex/exec_runner.go`
- Test: `internal/codex/runner_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestRequestValidateRequiresPrompt(t *testing.T) {
	req := Request{}
	if err := req.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/codex -run TestRequestValidateRequiresPrompt -v`
Expected: FAIL with missing package or symbol errors

- [ ] **Step 3: Write minimal implementation**

```go
type Request struct {
	Prompt string
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/codex -run TestRequestValidateRequiresPrompt -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/codex/runner.go internal/codex/exec_runner.go internal/codex/runner_test.go
git commit -m "feat: add codex runner boundary"
```

## Chunk 3: Review and Logging

### Task 5: Add review gate

**Files:**
- Create: `internal/review/gate.go`
- Test: `internal/review/gate_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestGateRejectsMissingValidation(t *testing.T) {
	result := Evaluate(Result{})
	if result.Approved {
		t.Fatal("expected review rejection")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/review -run TestGateRejectsMissingValidation -v`
Expected: FAIL with missing package or symbol errors

- [ ] **Step 3: Write minimal implementation**

```go
type Result struct {
	Validated bool
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/review -run TestGateRejectsMissingValidation -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/review/gate.go internal/review/gate_test.go
git commit -m "feat: add review gate"
```

### Task 6: Add run log model

**Files:**
- Create: `internal/runlog/log.go`
- Test: `internal/runlog/log_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestEntryValidateRequiresGoalAndDecision(t *testing.T) {
	entry := Entry{}
	if err := entry.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/runlog -run TestEntryValidateRequiresGoalAndDecision -v`
Expected: FAIL with missing package or symbol errors

- [ ] **Step 3: Write minimal implementation**

```go
type Entry struct {
	Goal     string
	Decision string
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/runlog -run TestEntryValidateRequiresGoalAndDecision -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/runlog/log.go internal/runlog/log_test.go
git commit -m "feat: add run log entry model"
```

## Chunk 4: End-to-End Skeleton

### Task 7: Wire minimal canxd flow

**Files:**
- Modify: `cmd/canxd/main.go`
- Test: `cmd/canxd/main_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestRunRejectsInvalidConfig(t *testing.T) {
	err := run(Config{})
	if err == nil {
		t.Fatal("expected validation error")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/canxd -run TestRunRejectsInvalidConfig -v`
Expected: FAIL with missing function or symbol errors

- [ ] **Step 3: Write minimal implementation**

```go
func run(cfg loop.Config) error {
	return cfg.Validate()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/canxd -run TestRunRejectsInvalidConfig -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/canxd/main.go cmd/canxd/main_test.go
git commit -m "feat: wire minimal canxd flow"
```

Plan complete and saved to `docs/2026-03-17-canx-mvp-plan.md`. Ready to execute?
