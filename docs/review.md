# CanX MVP Code Review

**日期:** 2026-03-18
**范围:** 全仓库，对照 `docs/2026-03-18-product-intent.md` 和 `docs/2026-03-17-canx-mvp-design.md`
**测试状态:** `go test ./...` 全部通过

---

## 总体评价

骨架搭得扎实，TDD 执行到位，所有核心模块都有测试覆盖，零外部依赖（纯标准库），模块边界清晰。对于一个"证明流程可行"的 MVP，当前代码已经达到了设计文档所描述的最低目标。

但有几个明显的结构性偏差和若干中小问题，需要在进入下一个迭代前修正或明确。

---

## 与设计意图的符合度

| 设计要求 | 实现状态 | 说明 |
|---|---|---|
| 有界控制循环（max turns / timeout / stop marker）| ✅ 已实现 | engine.go 全部覆盖 |
| Runner 接口 + ExecRunner + 预留 AppServerRunner | ✅ 已实现 | 接口拆分合理 |
| Workspace 加载（README / AGENTS / docs）| ✅ 已实现 | 但有脆弱性，见下文 |
| Review gate | ✅ 已实现 | 但 InScope 硬编码，见下文 |
| Run log | ⚠️ 部分实现 | Entry 有模型，但不持久化 |
| Task 模型 | ⚠️ 模型存在但完全未接入 | tasks 包是孤岛 |
| 时间预算（time budget）| ❌ 未实现 | Config 只有 MaxTurns，无总时间上限 |
| Session 管理 | ✅ 已实现 | 内存态，合理 |
| 结构化日志 | ⚠️ 内存态 | 无持久化，无 timestamp |

---

## 重要问题（建议在下一步之前解决）

### 1. `internal/tasks` 是孤岛

任务模型定义齐全，但 `loop/engine.go` 完全没有使用它。当前 Engine 直接运行 Prompt，没有任何任务分解。

设计文档明确要求：
> supervisor defines or updates task list → dispatch one or more worker tasks

实际上 Engine 只是循环调用 Runner，完全没有"supervisor 把 Goal 分解成 Task，再按 Task 调度"的逻辑。

**影响：** 这是最大的结构性偏差。设计的核心价值（AI-to-AI 任务拆解和分配）目前不存在于运行路径中。

**建议：** 明确这是 MVP 的有意简化（记录在文档里），或者在下一步把 Task 接入 Engine，让 supervisor 用 Codex 生成初始任务列表。

---

### 2. `review.Evaluate` 的 `InScope` 永远为 `true`

`engine.go` 第 85-88 行：

```go
reviewResult := review.Evaluate(review.Result{
    Validated: validationPassed,
    InScope:   true,
})
```

`InScope` 被硬编码，review gate 对"worker 是否跑偏了"没有任何实际判断。Review gate 目前等价于：`Approved = validationPassed`。

**建议：** 如果当前不打算做真正的 scope 检查，直接把 `InScope` 字段从 `Evaluate` 中移除，或者在文档里注明这是 stub。保留字段但永远传 `true` 会让读者以为有实际逻辑。

---

### 3. `workspace.Load` 要求 AGENTS.md 必须存在

```go
agents, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
if err != nil {
    return Context{}, err
}
```

对于没有 AGENTS.md 的目标仓库（比如 Tradex 还没接入 CanX 规范），整个加载直接失败。设计文档说 AGENTS.md 是"如果存在则加载"，而非必选。

**建议：** 对 AGENTS.md 使用 `os.IsNotExist` 容错，README.md 保持必选。

---

### 4. runlog 没有持久化

`runlog.Entry` 被创建并追加到 `Outcome.Logs`，但整个 Outcome 只存在于内存中，进程退出即消失。

设计文档要求：
> creates durable memory without relying on long prompt history

**建议：** 在 MVP 阶段至少将 Logs append 写入一个 JSONL 文件（比如 `.canx/runs/<timestamp>.jsonl`）。不需要复杂，`encoding/json` + `os.OpenFile` 即可。

---

### 5. 没有总时间预算（time budget）

设计文档的 Inputs 列表包含：
> time budget

`loop.Config` 只有 `MaxTurns`，没有 `Deadline time.Time` 或 `Budget time.Duration`。单次 `TurnTimeout` 乘以 `MaxTurns` 不等同于总预算，因为 validation 时间不计入 TurnTimeout。

**建议：** `Config` 添加 `Budget time.Duration`，在 `Run()` 开始时用 `context.WithTimeout(ctx, cfg.Budget)` 包住整个循环。

---

## 中等问题（下一阶段处理即可）

### 6. `main()` 的错误处理用了 `panic`

```go
if err != nil {
    panic(err)
}
```

CLI 工具应该用 `fmt.Fprintf(os.Stderr, "canx: %v\n", err)` + `os.Exit(1)`，panic 会输出 goroutine stack，对终端用户来说噪音很大。

---

### 7. `runValidation` 硬编码 `zsh -lc`

```go
cmd := exec.CommandContext(ctx, "zsh", "-lc", command)
```

在 Linux CI / Docker 环境会失败（zsh 未安装或行为不同）。

**建议：** 改为 `sh -c`，或者把 shell 作为可配置参数。

---

### 8. `sessions.Registry` 没有并发保护

`Registry.sessions` 是普通 `map`，没有 mutex。当前引擎是单 goroutine，所以不会 race，但如果将来在同一 Registry 上并发操作（比如 appserver runner 场景），会触发 race detector。

**建议：** 加 `sync.RWMutex`，或者在注释里明确"非并发安全，每个 Run 调用应有独立 Registry"。

---

### 9. `buildPrompt` 把上一轮完整 Output 塞入 prompt

```go
builder.WriteString(last.RunnerResult.Output)
```

Codex 输出可能很长（diff、日志等），每一轮都追加会导致 prompt 快速膨胀。

**建议：** 用 `summarizeTurn` 的摘要版本，而不是原始输出。

---

### 10. `canxd` 编译产物被 git 追踪

仓库根目录存在 `canxd` 二进制文件。应加入 `.gitignore`。

---

### 11. `go.mod` 声明 `go 1.25.0`，版本不存在

截至 2026-03 最新发布版为 1.24.x，`go 1.25.0` 尚未发布。建议改为 `go 1.24`。

---

## 小问题（可以随手修或推后）

- **`Makefile` 的 `fmt` 目标**：`gofmt -w $(shell find ...)` 可改为 `gofmt -w ./...`，更简洁。
- **`Task.Status` 用字符串常量**：可以定义 `type Status string` 增加类型安全，但 MVP 阶段影响不大。
- **`runlog.Entry` 无时间戳**：`Entry` 缺少 `Timestamp time.Time`，事后无法追溯单条记录的时间。
- **`loop/engine_test.go` 中的 `fakeRunner` 与 `codex.MockRunner` 有微小重复**：两者行为几乎一致，区别仅在于 fakeRunner 到达末尾后停留在最后一个结果。可以考虑统一，但现阶段影响可控。
- **`sessions` 模块的 `ModeOneshot` 存在但从未使用**：Engine 始终创建 `ModePersistent` session，oneshot 分支代码没有覆盖路径。

---

## 有没有重复造轮子？

总体没有。代码保持了"薄封装"的原则，没有自建 agent 框架、没有自建模型运行时。

一个值得关注的点：`sessions.Registry` 提供了 Spawn / Steer / Close 生命周期，这和 Codex app-server 本身的 session 模型高度重叠。未来切换到 `AppServerRunner` 之后，这个 Registry 可能会成为冗余层。建议在接入 `AppServerRunner` 时重新评估 sessions 包是否还需要存在，或者限定 Registry 只用于 ExecRunner 场景。

---

## 下一步建议

优先级排序：

1. **明确 tasks 包的定位**：是刻意推迟还是遗漏？在 START_HERE.md 或 review 里说清楚。
2. **修复 workspace.Load 对 AGENTS.md 的强制要求**（10 分钟内可完成）。
3. **给 Config 添加 Budget 字段并接入 context**。
4. **runlog 持久化**：最小实现是写 JSONL 文件，不需要数据库。
5. **review.InScope 去除硬编码**：要么实现真实判断，要么移除该字段。
6. **main() panic → os.Exit(1)**。
7. **把 canxd 加入 .gitignore**。
