# CanX MVP Code Review

**第一版：** 2026-03-18（初始骨架）
**本版：** 2026-03-18（第二轮，反映当前代码状态）
**范围：** 全仓库，对照 `docs/2026-03-18-product-intent.md` 和 `docs/2026-03-17-canx-mvp-design.md`
**测试状态：** `go test ./...` 全部通过

---

## 第一轮修复进度

第一轮发现的问题大多已在后续提交中修复，具体如下：

| 问题 | 状态 |
|---|---|
| `tasks` 包是孤岛，未接入 Engine | ✅ 已修复：加了 `Planner` 接口，Engine 调用 `Planner.Plan()` |
| 没有总时间预算 | ✅ 已修复：`Config.BudgetSeconds` + Engine 的 `context.WithTimeout` |
| `workspace.Load` 强制要求 AGENTS.md | ✅ 已修复：改为 `os.IsNotExist` 容错 |
| runlog 不持久化 | ✅ 已修复：`store.go` 写入 `.canx/sessions/<id>.json` |
| `runValidation` 硬编码 `zsh -lc` | ✅ 已修复：改为 `sh -c` |
| `main()` 用 `panic` | ✅ 已修复：改为 `fmt.Fprintf(os.Stderr)` + `os.Exit(1)` |
| `buildPrompt` 塞原始输出 | ✅ 已修复：改为 `summarizeTurn` 摘要 |
| `canxd` binary 被 git 追踪 | ✅ 已修复：`.gitignore` 加了 `/canxd` 和 `/.canx/` |
| Session 无时间戳 | ✅ 已修复：加了 `CreatedAt`、`UpdatedAt`、`LastSummary` |
| CLI 无 session 查看命令 | ✅ 新增：`canxd sessions list` / `canxd sessions show <id>` |

---

## 当前状态评估

代码整体质量明显提升。任务模型已接入运行路径，持久化已实现，CLI 变成了一个真正可用的工具。以下是当前仍然存在的问题。

---

## 重要问题

### 1. `StaticPlanner` 命名误导，且有 ID 碰撞风险

`StaticPlanner.Plan()` 总是把整个 goal 包装成一个 task，ID 固定为 `"task-1"`：

```go
func (StaticPlanner) Plan(goal string) ([]Task, error) {
    task := Task{
        ID:     "task-1",
        // ...
    }
    return []Task{task}, nil
}
```

两个问题：
1. **命名**：`StaticPlanner` 这个名字暗示"配置静态的"，而不是"永远只产生一个 task"。更准确的名字是 `SingleTaskPlanner` 或 `IdentityPlanner`。
2. **ID 固定**：如果将来有多个并发 Engine 实例，或者 Planner 被复用，`"task-1"` 会冲突。ID 应该基于 goal 做 hash，或者引入一个 ID 生成器。

---

### 2. `updateTaskStatuses` 只更新第一个 task

```go
func updateTaskStatuses(items []tasks.Task, done bool) []tasks.Task {
    // ...
    if done {
        next[0].Status = tasks.StatusDone
    } else {
        next[0].Status = tasks.StatusInProgress
    }
    return next
}
```

当前 `StaticPlanner` 只返回一个 task，所以没问题。但这个函数是为多 task 场景设计的（接受 `[]Task`），却只更新 `index 0`。一旦 `Planner` 返回多个任务，这里会静默地忽略其他任务。

**建议：** 要么明确注释"只跟踪第一个活跃任务"，要么改为按状态查找第一个 `pending/in_progress` 的 task 来更新，而不是硬用 `[0]`。

---

### 3. `Planner` 接口缺少 `context.Context`

```go
type Planner interface {
    Plan(goal string) ([]Task, error)
}
```

未来的 `CodxPlanner`（调用 Codex 生成任务列表）需要能够响应超时和取消。现在的接口签名不支持这个。

**建议：** 改为 `Plan(ctx context.Context, goal string) ([]Task, error)`。这是一个破坏性变更，越早做代价越低。

---

### 4. `buildPrompt` 加载了 `repo.Docs` 但从不使用

`workspace.Load` 收集了 `docs/` 下所有 Markdown 文档，存在 `repo.Docs []Document` 里，但 `buildPrompt` 完全忽略了它：

```go
func buildPrompt(goal string, repo workspace.Context, plannedTasks []tasks.Task, turns []Turn) string {
    // ...
    builder.WriteString(repo.Readme)  // 用了
    builder.WriteString(repo.Agents) // 用了
    // repo.Docs 从未出现在这里
}
```

设计文档明确要求工作区加载包括"a small number of high-signal docs under docs/"，目的是让 worker 理解项目背景。现在这些 docs 白加载了，worker 看不到它们。

**建议：** 在 `buildPrompt` 里加入 docs 注入，同时建议加 token 预算截断（不是所有 docs 都要注入，有些 target repo 的 docs/ 可能很大）。

---

### 5. `review.Evaluate` 的 `InScope` 仍然硬编码为 `true`

```go
reviewResult := review.Evaluate(review.Result{
    Validated: validationPassed,
    InScope:   true,  // 永远 true
})
```

这个问题从第一轮开始就存在，仍未修复。`review.Result.InScope` 字段的存在暗示有 scope 检查逻辑，但实际上 `Evaluate` 对 InScope 的判断毫无意义，因为调用方永远传 `true`。

**建议：** 二选一：要么实现真实的 scope 检查（比较 `cfg.FilesInScope` 和 runner 实际修改的文件），要么把 `InScope` 从 `Result` 里删掉，直到真正需要时再加回来。

---

## 中等问题

### 6. `canxd sessions list` 目录不存在时报错

```go
entries, err := os.ReadDir(sessionsDir)
if err != nil {
    return "", err
}
```

用户第一次运行 `canxd sessions list`（还没有跑过任何 run）时，`.canx/sessions` 不存在，命令会返回错误而不是空列表。

**建议：** 用 `os.IsNotExist` 判断，目录不存在时直接返回空字符串或 `"(no sessions)"`。

---

### 7. `sessions.Registry.List()` 与 CLI 的 sessions 路径不一致

`sessions.Registry` 有一个 `List()` 方法，但 `inspectSessions` 函数绕过了 Registry，直接用 `os.ReadDir` 读磁盘文件。这意味着：

- Registry.List() 只能列出当前进程内存中的会话（这个进程跑完就消失了）
- CLI 的 `sessions list` 从磁盘读历史记录

两套 "list" 的语义不同，容易让后来的开发者误用 `Registry.List()`。

**建议：** 考虑把 `Registry.List()` 删掉或标注为"仅用于测试"，因为持久化的历史 sessions 只能从磁盘读取。

---

### 8. `sessions.Registry` 仍然没有并发保护

`Registry.sessions` 是无锁的 `map`。当前引擎是单 goroutine 顺序执行，不会触发 race。但 Engine.Run 被多 goroutine 并发调用时（比如将来的并发 worker 场景），同一个 Registry 实例会产生数据竞争。

**建议：** 加 `sync.RWMutex`，或者在 `Registry` 的 godoc 里明确注明"非并发安全，每个 Engine.Run 调用应传入独立 Registry 实例"。

---

### 9. `go.mod` 声明 `go 1.25.0`，版本尚不存在

截至 2026-03，最新发布版为 1.24.x，`go 1.25.0` 尚未正式发布。

**建议：** 改为 `go 1.24`。

---

## 小问题

- **`ExecRunner` 传 prompt 作为位置参数**：`exec.CommandContext(ctx, r.bin, "exec", req.Prompt)` 把整个 prompt 字符串作为第三个 CLI 参数。Codex CLI 的 `codex exec` 实际接口需要验证——长 prompt（含换行符）作为 shell 参数可能有问题。考虑改为 stdin 或临时文件。

- **`Request.MaxTurns` 未被 `ExecRunner` 使用**：`codex.Request.MaxTurns` 字段存在，Engine 传了 `MaxTurns: 1`，但 ExecRunner 构建的命令行根本不包含这个参数。这个字段在 ExecRunner 路径下是死字段。

- **`ModeOneshot` 从未被使用**：Engine 始终创建 `ModePersistent` session，`sessions.ModeOneshot` 没有任何代码路径。

- **`Makefile` 的 `fmt` 目标**：`gofmt -w $(shell find . -type f -name '*.go' -not -path './vendor/*')` 可以简化为 `gofmt -w ./...`。

- **`fakeRunner`（engine_test.go）与 `codex.MockRunner` 行为近似重复**：两者的差异仅在 index 越界处理。可以统一，但当前影响可控。

---

## 整体评价（第二轮）

相比第一轮，代码成熟度提升明显：

- 任务模型有了真正的接入路径（Planner → Engine → Outcome.Tasks）
- 持久化从零到可用（`store.go` + `sessions show/list`）
- CLI 变成了一个有子命令结构的真实工具
- 时间预算、AGENTS.md 容错、`sh -c` 都已修复

**剩余的核心问题是两个"下一步必须做"：**

1. **`buildPrompt` 不注入 docs**：这是 workspace 加载存在的意义，现在是空转。
2. **`Planner` 接口缺 context**：越早改代价越低，一旦有了 AI Planner 就必须改。

其他问题（`InScope` 硬编码、`updateTaskStatuses` 只更新 index 0、`sessions list` 报错）都是在下一个迭代前应顺手修的。

---

## 下一步建议

优先级排序：

1. **给 `Planner` 接口加 `context.Context` 参数**（破坏性变更，越早越好）。
2. **在 `buildPrompt` 里注入 `repo.Docs`**，加 token 预算截断（比如总 docs 长度不超过 4000 字符）。
3. **`updateTaskStatuses` 改为找第一个 pending/in_progress 的 task**，而不是硬用 `[0]`。
4. **`canxd sessions list` 目录不存在时返回空列表**（5 分钟内可完成）。
5. **明确 `review.InScope`**：实现真实判断，或者删掉这个字段。
6. **`StaticPlanner` 改名为 `SingleTaskPlanner`**，并修复固定 ID 问题。
7. **`go.mod` 改为 `go 1.24`**。
