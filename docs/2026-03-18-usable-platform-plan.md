# CanX 可用平台实施计划

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标：** 让 CanX 能够运行一个真实的开发任务（包括开发 CanX 自身），产生可见的、可验证的输出。

**当前状态快照：** 骨架完整，测试全绿，持久化已有，docs 注入已有。核心缺口是：ExecRunner 接口未经真实 Codex 验证；loop 没有验证失败反馈；没有 AI 任务分解；每轮 prompt 对 worker 的指令不够精确。

**参照系统：** Magentic-One（动态重规划）、Claude Code subagents（隔离上下文 + 精确任务范围）、OpenHands（action-observation 循环）

**技术栈：** Go 1.24+，标准库，table-driven 测试

---

## 优先级说明

能用是第一要务。按以下顺序：
1. **修执行层**（ExecRunner 接口 + 验证输出捕获）——不修这个其他全白搭
2. **让 loop 更聪明**（验证失败反馈 + 更精确的 worker prompt）
3. **AI 任务分解**（CodxPlanner——这是从"有界重试"到"真正 supervisor"的核心跳跃）
4. **多任务顺序执行**（任务列表逐条调度）
5. **自托管演示**（用 CanX 开发 CanX）

---

## 先进系统差距分析

### 当前 CanX vs 先进实现的核心差距

**Magentic-One 做的，CanX 没做：**
- 每步之后 Orchestrator 重新评估计划，不是简单 retry
- 失败之后重规划（re-plan），而不是用同一 prompt 再跑一遍
- 每个 worker agent 有专化角色（Coder、Tester、Reviewer 是不同配置）

**Claude Code 做的，CanX 没做：**
- subagent 获得精确的任务范围：「你负责 X 文件，不要碰 Y，完成后输出 JSON summary」
- Parent agent 注入给 subagent 的 context 是精心裁剪的，不是整个 README
- 验证失败的错误信息直接作为 subagent 下一轮的输入

**OpenHands 做的，CanX 没做：**
- agent 的每一步 action 都被记录（edit file、run test、read file）
- observation（文件内容、测试输出）直接作为下一步的输入
- CanX 只有 runner output 字符串，不知道 worker 改了哪些文件

**核心结论：** CanX 目前是「把 prompt 喂给 Codex，看 stdout，决定继续还是停止」。先进系统是「知道 agent 做了什么，把做的结果（diff、测试输出）作为下一轮的结构化输入」。

---

## Chunk 1：验证并修复执行层

> **这是其他一切的前提。** ExecRunner 的接口没有经过真实 Codex 验证。

### Task 1：验证 ExecRunner 与真实 Codex 的接口

**背景：** `exec.CommandContext(ctx, r.bin, "exec", req.Prompt)` 把整个 prompt 作为第三个 CLI 参数传给 Codex。这在 prompt 含有换行符、引号时可能出问题。Codex CLI 的 `exec` 子命令实际接口需要验证。

**文件：**
- 修改：`internal/codex/exec_runner.go`
- 修改：`internal/codex/runner_test.go`

- [ ] **Step 1：验证 Codex CLI 接口**

手动运行：
```bash
echo "print hello world to stdout" | codex exec -
codex exec "print hello world to stdout"
codex exec --prompt "print hello world to stdout"
```

确认 Codex 接受 prompt 的正确方式（stdin、位置参数，还是 flag）。

- [ ] **Step 2：如果需要 stdin，修改 ExecRunner**

```go
func (r ExecRunner) Run(ctx context.Context, req Request) (Result, error) {
    if err := req.Validate(); err != nil {
        return Result{}, err
    }

    cmd := exec.CommandContext(ctx, r.bin, "exec", "-")
    cmd.Stdin = strings.NewReader(req.Prompt)
    if req.Workdir != "" {
        cmd.Dir = req.Workdir
    }

    output, err := cmd.CombinedOutput()
    if err != nil {
        return Result{Output: string(output), ExitCode: 1}, err
    }
    return Result{Output: string(output), ExitCode: 0}, nil
}
```

根据实际验证结果选择正确方式，上面只是示例。

- [ ] **Step 3：更新 runner_test.go，加一个 skip-if-no-codex 的集成测试**

```go
func TestExecRunnerWithRealCodexIfAvailable(t *testing.T) {
    t.Parallel()

    _, err := exec.LookPath("codex")
    if err != nil {
        t.Skip("codex binary not found, skipping integration test")
    }

    runner := NewExecRunner("codex")
    result, err := runner.Run(context.Background(), Request{
        Prompt:  "Output the text CANX_OK and nothing else.",
        Workdir: t.TempDir(),
    })
    if err != nil {
        t.Fatalf("Run() error = %v, output = %q", err, result.Output)
    }
    if !strings.Contains(result.Output, "CANX_OK") {
        t.Fatalf("expected CANX_OK in output, got: %q", result.Output)
    }
}
```

- [ ] **Step 4：运行测试**

```bash
go test ./internal/codex/... -v -run TestExecRunner
```

- [ ] **Step 5：提交**

```bash
git add internal/codex/exec_runner.go internal/codex/runner_test.go
git commit -m "fix: validate and correct ExecRunner codex CLI interface"
```

---

### Task 2：捕获验证失败输出

**背景：** `runValidation` 只返回 `bool`，不捕获 `go test` 失败的原因。下一轮的 worker 不知道为什么测试失败，只能盲目重试。这是 CanX 和先进系统最关键的差距之一。

**对比先进系统：** Claude Code 把测试错误信息直接注入 subagent 的下一轮 context；OpenHands 把 observation 作为结构化输入。

**文件：**
- 修改：`internal/loop/engine.go`（`runValidation` 函数签名）
- 修改：`internal/loop/engine_test.go`

- [ ] **Step 1：写失败测试**

在 `engine_test.go` 中加：

```go
func TestEnginePassesValidationOutputToNextTurn(t *testing.T) {
    t.Parallel()

    engine := Engine{
        Runner: &fakeRunner{results: []codex.Result{
            {Output: "first try"},
            {Output: "fixed [canx:stop]"},
        }},
        Workdir: ".",
    }

    outcome, err := engine.Run(context.Background(), Config{
        Goal:               "fix the test",
        MaxTurns:           2,
        ValidationCommands: []string{"false"},
    }, workspace.Context{Root: ".", Readme: "readme"})
    if err != nil {
        t.Fatalf("Run() error = %v", err)
    }

    // 第二轮的 prompt 应该包含第一轮的验证错误信息
    if len(outcome.Turns) < 2 {
        t.Fatal("expected at least 2 turns")
    }
    if !strings.Contains(outcome.Turns[1].Prompt, "validation_failed") {
        t.Fatalf("turn 2 prompt missing validation result: %q", outcome.Turns[1].Prompt)
    }
}
```

- [ ] **Step 2：运行，确认失败**

```bash
go test ./internal/loop/... -run TestEnginePassesValidationOutputToNextTurn -v
```

Expected: FAIL（`turn 2 prompt missing validation result`）

- [ ] **Step 3：修改 `runValidation` 返回 `(bool, string)`**

```go
func runValidation(ctx context.Context, workdir string, commands []string) (bool, string) {
    if len(commands) == 0 {
        return false, ""
    }

    var failOutput strings.Builder
    for _, command := range commands {
        cmd := exec.CommandContext(ctx, "sh", "-c", command)
        if workdir != "" {
            cmd.Dir = workdir
        }
        out, err := cmd.CombinedOutput()
        if err != nil {
            failOutput.WriteString(command)
            failOutput.WriteString(":\n")
            // 截断：最多 500 字符
            s := strings.TrimSpace(string(out))
            if len(s) > 500 {
                s = s[:500] + "\n...(truncated)"
            }
            failOutput.WriteString(s)
            failOutput.WriteString("\n")
            return false, failOutput.String()
        }
    }
    return true, ""
}
```

- [ ] **Step 4：更新 Engine.Run 传递 validationOutput**

```go
validationPassed, validationOutput := runValidation(turnCtx, e.Workdir, cfg.ValidationCommands)
```

在 `Turn` struct 里加 `ValidationOutput string`，在 `buildPrompt` 里用它替换现有的 `summarizeTurn`：

```go
// 在 buildPrompt 里，如果上一轮有 validation 输出：
if last.ValidationOutput != "" {
    builder.WriteString("\n\nValidation errors from last turn:\n")
    builder.WriteString(last.ValidationOutput)
}
```

- [ ] **Step 5：运行所有测试**

```bash
make test
```

- [ ] **Step 6：提交**

```bash
git add internal/loop/engine.go internal/loop/engine_test.go
git commit -m "feat: capture validation output and feed to next turn"
```

---

### Task 3：捕获 git diff（know what changed）

**背景：** 每轮执行后，CanX 不知道 worker 改了哪些文件。先进系统（OpenHands）把文件变更作为 observation 注入下一轮。

**文件：**
- 修改：`internal/loop/engine.go`（加 `captureGitDiff` 函数）
- 修改：`internal/loop/engine_test.go`

- [ ] **Step 1：写失败测试**

```go
func TestCaptureGitDiff(t *testing.T) {
    t.Parallel()

    dir := t.TempDir()
    // init a git repo
    run := func(args ...string) {
        cmd := exec.Command(args[0], args[1:]...)
        cmd.Dir = dir
        _ = cmd.Run()
    }
    run("git", "init")
    run("git", "config", "user.email", "test@test.com")
    run("git", "config", "user.name", "Test")
    if err := os.WriteFile(filepath.Join(dir, "foo.go"), []byte("package main\n"), 0o644); err != nil {
        t.Fatal(err)
    }
    run("git", "add", ".")
    run("git", "commit", "-m", "init")

    // make a change
    if err := os.WriteFile(filepath.Join(dir, "foo.go"), []byte("package main\n\nfunc f() {}\n"), 0o644); err != nil {
        t.Fatal(err)
    }

    diff := captureGitDiff(dir)
    if !strings.Contains(diff, "foo.go") {
        t.Fatalf("expected foo.go in diff, got: %q", diff)
    }
}
```

- [ ] **Step 2：运行，确认失败**

```bash
go test ./internal/loop/... -run TestCaptureGitDiff -v
```

- [ ] **Step 3：实现 `captureGitDiff`**

```go
func captureGitDiff(workdir string) string {
    cmd := exec.Command("git", "diff", "--stat", "HEAD")
    if workdir != "" {
        cmd.Dir = workdir
    }
    out, err := cmd.Output()
    if err != nil {
        return ""
    }
    s := strings.TrimSpace(string(out))
    if len(s) > 300 {
        s = s[:300] + "\n...(truncated)"
    }
    return s
}
```

- [ ] **Step 4：在 engine.go 里 Turn 结构加 `GitDiff string`，每轮执行后填充**

在 `buildPrompt` 里加：

```go
if last.GitDiff != "" {
    builder.WriteString("\n\nFiles changed last turn:\n")
    builder.WriteString(last.GitDiff)
}
```

- [ ] **Step 5：运行所有测试**

```bash
make test
```

- [ ] **Step 6：提交**

```bash
git add internal/loop/engine.go internal/loop/engine_test.go
git commit -m "feat: capture git diff per turn and inject into next prompt"
```

---

## Chunk 2：让 worker prompt 更精确

> 当前 prompt 结尾是「Respond with progress, and include [canx:stop] when the task is complete.」这对 Codex 来说指令太模糊。先进系统给 worker 的指令是精确的任务范围、约束、预期输出格式。

### Task 4：精确化 worker prompt 模板

**背景对比：**
- 当前 CanX：「Goal: X. Repository context: README. Respond with progress.」
- Claude Code subagent：「Your task is X. Relevant files: A, B. Do NOT modify C. When done, output a JSON summary with fields: files_changed, tests_run, status.」
- Magentic-One Coder：系统 prompt 包含角色定义、工具列表、输出格式约束

**文件：**
- 修改：`internal/loop/engine.go`（`buildPrompt` 函数）
- 修改：`internal/loop/engine_test.go`

- [ ] **Step 1：写测试，验证 prompt 包含精确指令**

```go
func TestBuildPromptIncludesStopInstructions(t *testing.T) {
    t.Parallel()

    prompt, _ := buildPrompt("fix bug", workspace.Context{Readme: "readme"}, nil, nil)

    checks := []string{
        "fix bug",
        "[canx:stop]",
        "make test",
        "git diff",
    }
    for _, check := range checks {
        if !strings.Contains(prompt, check) {
            t.Errorf("prompt missing %q\nprompt:\n%s", check, prompt)
        }
    }
}
```

- [ ] **Step 2：运行，确认当前状态**

```bash
go test ./internal/loop/... -run TestBuildPromptIncludesStopInstructions -v
```

- [ ] **Step 3：替换 prompt 结尾为精确指令模板**

```go
const workerInstructions = `

Instructions:
- Work inside the repository only.
- Run validation commands (if any) after making changes.
- If all changes are complete and tests pass, output [canx:stop] on its own line.
- If you are blocked or cannot proceed, output [canx:escalate] with a one-line reason.
- Keep your response concise: what you did, what changed, what remains.`

// buildPrompt 末尾替换为：
builder.WriteString(workerInstructions)
```

- [ ] **Step 4：运行所有测试**

```bash
make test
```

- [ ] **Step 5：提交**

```bash
git add internal/loop/engine.go internal/loop/engine_test.go
git commit -m "feat: add precise worker instructions to prompt template"
```

---

### Task 5：支持 `[canx:escalate]` 作为显式退出信号

**背景：** worker 遇到无法继续的情况（缺少权限、需要人工决策）时，目前只能等到 max turns。`[canx:escalate]` 让 worker 可以主动请求人工介入，这是 Magentic-One 和 AutoGen 里的标准模式。

**文件：**
- 修改：`internal/loop/engine.go`
- 修改：`internal/loop/engine_test.go`

- [ ] **Step 1：写失败测试**

```go
func TestEngineEscalatesOnEscalateMarker(t *testing.T) {
    t.Parallel()

    engine := Engine{
        Runner:  &fakeRunner{results: []codex.Result{{Output: "blocked [canx:escalate] need database credentials"}}},
        Workdir: ".",
    }

    outcome, err := engine.Run(context.Background(), Config{
        Goal:     "migrate database",
        MaxTurns: 3,
    }, workspace.Context{Root: ".", Readme: "readme"})
    if err != nil {
        t.Fatalf("Run() error = %v", err)
    }

    if got, want := outcome.Decision.Action, ActionEscalate; got != want {
        t.Fatalf("decision = %q, want %q", got, want)
    }
    if !strings.Contains(outcome.Decision.Reason, "worker requested escalation") {
        t.Fatalf("reason = %q, expected worker requested escalation", outcome.Decision.Reason)
    }
}
```

- [ ] **Step 2：运行，确认失败**

```bash
go test ./internal/loop/... -run TestEngineEscalatesOnEscalateMarker -v
```

- [ ] **Step 3：在 engine.go 加 escalate marker 处理**

```go
const escalateMarker = "[canx:escalate]"

// 在 Run() 的 switch 里加：
case strings.Contains(result.Output, escalateMarker):
    session, _ = e.Sessions.Close(session.ID)
    outcome.Session = session
    outcome.Decision = Decision{Action: ActionEscalate, Reason: "worker requested escalation"}
    return outcome, nil
```

- [ ] **Step 4：运行所有测试**

```bash
make test
```

- [ ] **Step 5：提交**

```bash
git add internal/loop/engine.go internal/loop/engine_test.go
git commit -m "feat: support [canx:escalate] marker for worker-initiated escalation"
```

---

## Chunk 3：AI 任务分解（CodxPlanner）

> 这是从「有界重试器」到「真正 supervisor」的核心跳跃。没有这个，CanX 对每个 goal 只能产生一个任务，无法分解复杂需求。

### Task 6：定义 CodxPlanner 及其输出格式

**背景：** Magentic-One 的 Orchestrator 做的第一步是生成一个 plan（任务列表）。LangGraph 的 supervisor 在每一步决定路由到哪个 worker。CanX 的 `CodxPlanner` 应该调用 Codex，要求它以 JSON 格式输出任务列表，然后 parse 成 `[]tasks.Task`。

**文件：**
- 新建：`internal/tasks/codx_planner.go`
- 新建：`internal/tasks/codx_planner_test.go`

- [ ] **Step 1：写失败测试**

```go
package tasks

import (
    "context"
    "testing"
)

type fakePlannerRunner struct {
    output string
}

func (r fakePlannerRunner) Run(_ context.Context, goal string) (string, error) {
    return r.output, nil
}

func TestCodxPlannerParsesJSONOutput(t *testing.T) {
    t.Parallel()

    runner := fakePlannerRunner{output: `[
        {"id":"task-1","title":"Add test","goal":"add a failing test for X","status":"pending"},
        {"id":"task-2","title":"Implement X","goal":"implement X to pass the test","status":"pending"}
    ]`}

    planner := CodxPlanner{Runner: runner}
    tasks, err := planner.Plan(context.Background(), "implement feature X with TDD")
    if err != nil {
        t.Fatalf("Plan() error = %v", err)
    }

    if got, want := len(tasks), 2; got != want {
        t.Fatalf("Plan() len = %d, want %d", got, want)
    }
    if tasks[0].ID != "task-1" {
        t.Fatalf("task 0 id = %q, want task-1", tasks[0].ID)
    }
}

func TestCodxPlannerFallsBackOnInvalidJSON(t *testing.T) {
    t.Parallel()

    runner := fakePlannerRunner{output: "I'll create two tasks: first add a test, then implement"}

    planner := CodxPlanner{Runner: runner}
    tasks, err := planner.Plan(context.Background(), "implement feature X")
    if err != nil {
        t.Fatalf("Plan() error = %v", err)
    }

    // fallback: single task wrapping the goal
    if got, want := len(tasks), 1; got != want {
        t.Fatalf("Plan() fallback len = %d, want %d", got, want)
    }
}
```

- [ ] **Step 2：运行，确认失败**

```bash
go test ./internal/tasks/... -run TestCodxPlanner -v
```

- [ ] **Step 3：实现 CodxPlanner**

```go
package tasks

import (
    "context"
    "encoding/json"
    "strings"
)

// PlannerRunner is the execution interface CodxPlanner uses.
// It is separate from codex.Runner to avoid a circular import.
type PlannerRunner interface {
    Run(ctx context.Context, prompt string) (string, error)
}

type CodxPlanner struct {
    Runner PlannerRunner
}

const plannerPrompt = `You are a software delivery supervisor. Given a goal, output a JSON array of tasks.

Each task must have: id (string), title (string, max 40 chars), goal (string), status ("pending").

Output ONLY valid JSON, no explanation. Maximum 5 tasks. Example:
[{"id":"task-1","title":"Add failing test","goal":"write a failing test for X","status":"pending"}]

Goal: `

func (p CodxPlanner) Plan(ctx context.Context, goal string) ([]Task, error) {
    output, err := p.Runner.Run(ctx, plannerPrompt+goal)
    if err != nil {
        return nil, err
    }

    tasks, parseErr := parsePlanJSON(output)
    if parseErr != nil || len(tasks) == 0 {
        // fallback to single task
        return SingleTaskPlanner{}.Plan(ctx, goal)
    }

    for i := range tasks {
        tasks[i].Normalize()
    }
    return tasks, nil
}

func parsePlanJSON(output string) ([]Task, error) {
    // 找第一个 '[' 和最后一个 ']'，提取 JSON 数组
    start := strings.Index(output, "[")
    end := strings.LastIndex(output, "]")
    if start == -1 || end == -1 || end <= start {
        return nil, ErrMissingTaskID // reuse as "parse failed"
    }

    var tasks []Task
    if err := json.Unmarshal([]byte(output[start:end+1]), &tasks); err != nil {
        return nil, err
    }
    return tasks, nil
}
```

- [ ] **Step 4：运行所有测试**

```bash
go test ./internal/tasks/... -v
```

- [ ] **Step 5：提交**

```bash
git add internal/tasks/codx_planner.go internal/tasks/codx_planner_test.go
git commit -m "feat: add CodxPlanner with JSON task decomposition and fallback"
```

---

### Task 7：把 CodxPlanner 接入 CLI

**文件：**
- 修改：`cmd/canxd/main.go`（加 `--planner` flag）
- 修改：`cmd/canxd/main_test.go`

- [ ] **Step 1：在 Options 加 PlannerMode，在 runWithRunner 里接入**

```go
// Options 加：
PlannerMode string  // "single" (default) or "codx"

// parseFlags 加：
planner = flag.String("planner", "single", "planner mode: single or codx")
```

在 `runWithRunner` 里：

```go
var planner tasks.Planner
switch opts.PlannerMode {
case "", "single":
    planner = tasks.SingleTaskPlanner{}
case "codx":
    planner = tasks.CodxPlanner{Runner: codexPlannerRunner{bin: opts.CodexBin}}
default:
    return "", fmt.Errorf("unknown planner mode: %s", opts.PlannerMode)
}

engine := loop.Engine{
    Runner:      runner,
    Planner:     planner,
    Workdir:     absRepoPath,
    TurnTimeout: opts.TurnTimeout,
}
```

`codexPlannerRunner` 是一个简单的适配器：

```go
type codexPlannerRunner struct{ bin string }

func (r codexPlannerRunner) Run(ctx context.Context, prompt string) (string, error) {
    cmd := exec.CommandContext(ctx, r.bin, "exec", prompt)
    out, err := cmd.CombinedOutput()
    return string(out), err
}
```

（根据 Task 1 的验证结果选择正确的接口方式）

- [ ] **Step 2：运行所有测试**

```bash
make test
```

- [ ] **Step 3：提交**

```bash
git add cmd/canxd/main.go cmd/canxd/main_test.go
git commit -m "feat: wire CodxPlanner into canxd via --planner flag"
```

---

## Chunk 4：多任务顺序执行

> 当前 Engine 把一个 goal 当单个任务运行 N 轮。接入 CodxPlanner 后，goal 会被分解成多个任务，Engine 需要按顺序调度它们。

### Task 8：Engine 支持多任务顺序调度

**背景：** Magentic-One 的 Orchestrator 每步选择最合适的 agent 处理当前任务。LangGraph 通过 `Command(goto=...)` 显式路由。CanX 的简化版本：按顺序执行任务列表，每个任务完成（stop marker 或 validation pass）后推进到下一个。

**文件：**
- 修改：`internal/loop/engine.go`（抽出 `runTask` 函数）
- 修改：`internal/loop/engine_test.go`

- [ ] **Step 1：写失败测试**

```go
func TestEngineRunsMultipleTasksInSequence(t *testing.T) {
    t.Parallel()

    engine := Engine{
        Runner: &fakeRunner{results: []codex.Result{
            {Output: "task 1 done [canx:stop]"},
            {Output: "task 2 done [canx:stop]"},
        }},
        Workdir: ".",
        Planner: &fakePlanner{tasks: []tasks.Task{
            {ID: "t1", Title: "Task 1", Goal: "do first thing", Status: tasks.StatusPending},
            {ID: "t2", Title: "Task 2", Goal: "do second thing", Status: tasks.StatusPending},
        }},
    }

    outcome, err := engine.Run(context.Background(), Config{
        Goal:     "do both things",
        MaxTurns: 4,
    }, workspace.Context{Root: ".", Readme: "readme"})
    if err != nil {
        t.Fatalf("Run() error = %v", err)
    }

    doneCount := 0
    for _, task := range outcome.Tasks {
        if task.Status == tasks.StatusDone {
            doneCount++
        }
    }
    if got, want := doneCount, 2; got != want {
        t.Fatalf("done tasks = %d, want %d", got, want)
    }
}

type fakePlanner struct{ tasks []tasks.Task }

func (p *fakePlanner) Plan(_ context.Context, _ string) ([]tasks.Task, error) {
    return p.tasks, nil
}
```

- [ ] **Step 2：运行，确认失败**

```bash
go test ./internal/loop/... -run TestEngineRunsMultipleTasksInSequence -v
```

- [ ] **Step 3：重构 Engine.Run，引入 activeTaskIndex**

核心逻辑变化：
- 每次循环开始，找第一个 `pending` 的任务作为 active task
- `buildPrompt` 只注入 active task 的目标（不是所有任务）
- active task 收到 stop marker 或 validation pass 时，标为 done，进入下一个任务
- 所有任务 done → ActionStop（全部完成）
- max turns 到了但还有未完成任务 → ActionEscalate

关键实现点（不要全部重写，只改 turn 循环的状态推进逻辑）：

```go
// 在 turn 循环里，找第一个 pending task
activeTask := findFirstActive(outcome.Tasks)
if activeTask == nil {
    // 所有任务都 done
    outcome.Decision = Decision{Action: ActionStop, Reason: "all tasks complete"}
    break
}

prompt, docsUsed := buildPromptForTask(cfg.Goal, *activeTask, repo, outcome.Turns)
```

- [ ] **Step 4：运行所有测试**

```bash
make test
```

- [ ] **Step 5：提交**

```bash
git add internal/loop/engine.go internal/loop/engine_test.go
git commit -m "feat: multi-task sequential execution in engine"
```

---

## Chunk 5：自托管演示

> 这是验证整个系统的最终测试。用 CanX 来给 CanX 开发一个小功能。

### Task 9：端到端自托管 smoke test

**目标：** 跑一次真实的 `canxd` 命令，让 CanX 给自己添加一个简单的功能（比如修一个已知 bug 或加一个小 feature）。记录结果。

- [ ] **Step 1：构建 canxd**

```bash
make build
./canxd --help
```

确认 `--planner` flag 存在，`sessions list` 命令可用。

- [ ] **Step 2：用 mock runner 跑一次完整循环**

```bash
./canxd \
  -goal "Fix the canxd sessions list command to return empty list when directory does not exist" \
  -repo . \
  -runner mock \
  -validate "make test" \
  -max-turns 3
```

预期输出：`decision=stop reason=... turns=1 tasks=1`

确认 `.canx/sessions/` 里有 JSON 文件。

```bash
./canxd sessions list
./canxd sessions show <session-id>
```

- [ ] **Step 3：用 codx runner 跑一次真实任务（需要 Codex 可用）**

```bash
./canxd \
  -goal "Fix the canxd sessions list command to return empty list when .canx/sessions directory does not exist" \
  -repo . \
  -runner exec \
  -planner single \
  -validate "make test" \
  -max-turns 5 \
  -budget-seconds 300
```

观察：
- 每轮输出了什么
- 是否停止（stop marker 或 validation pass）
- `.canx/sessions/` 里的 JSON 记录了什么

- [ ] **Step 4：记录结果**

在 `docs/` 里新建一个简短的 `2026-03-18-first-self-run.md`，记录：
- 跑的 goal
- 几轮完成
- worker 做了什么（diff）
- 是否 validation pass

这个文档就是 CanX 能用的第一个证据。

- [ ] **Step 5：提交**

```bash
git add docs/2026-03-18-first-self-run.md
git commit -m "docs: record first canxd self-hosted run result"
```

---

## 实施顺序建议

```
Chunk 1（必须先做）
  Task 1：ExecRunner 验证 ← 其他一切的前提
  Task 2：验证输出捕获 ← 最大的质量跳跃
  Task 3：git diff 捕获 ← 让 loop 有视野

Chunk 2（并行可做）
  Task 4：精确 prompt 模板
  Task 5：escalate marker

Chunk 3（依赖 Chunk 1 完成）
  Task 6：CodxPlanner
  Task 7：接入 CLI

Chunk 4（依赖 Chunk 3）
  Task 8：多任务顺序执行

Chunk 5（最后）
  Task 9：自托管演示
```

估时：Chunk 1-2 约 1 天，Chunk 3-4 约 2 天，Chunk 5 约半天。总计 3-4 天可以看到第一个真实的自托管运行结果。

---

## 暂时不做的事

以下内容有价值但不在当前计划范围内：

- **并发 worker 调度**：顺序执行先跑通再说
- **AI review gate**：规则 gate 先用着，AI review 是第二阶段
- **AppServerRunner**：ExecRunner 先跑通，app-server 后续
- **Web UI / 仪表盘**：CLI + JSON 文件够用
- **分布式执行**：单机先跑
- **OpenClaw ACP 集成**：AppServerRunner 时再考虑
