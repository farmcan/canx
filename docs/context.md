# CanX 项目上下文

> 一页纸：你需要知道的一切。

---

## 这是什么

**CanX** 是一个 Go 写的编排器，让多个 Codex worker 协作完成软件交付任务。

它不是聊天机器人，不是业务应用，不是模型运行时。它的唯一目的是：

```
人类提供目标 → Supervisor 分解任务 → Codex Worker 执行 → 验证（go test ./...）→ 停止/升级
```

相比直接用 Codex：
- 有**确定性的 validation gate**（build + test 通过才继续）
- 有**有界循环**（max-turns + budget-seconds + stop-marker）
- 有**仓库上下文注入**（README + AGENTS.md + docs 自动加载）
- 有**持久化 session report**（`.canx/sessions/`）

---

## 当前代码状态（2026-03-19）

### 已完成

| 模块 | 位置 | 状态 |
|---|---|---|
| 任务模型 | `internal/tasks` | ✅ Task + Planner 接口 + CodxPlanner |
| 编排引擎 | `internal/loop` | ✅ Engine.Run，多任务顺序执行，stop/escalate 信号 |
| Codex 接入 | `internal/codex` | ✅ ExecRunner（`codex exec -`），MockRunner |
| Validation gate | `internal/loop` | ✅ `sh -c` 命令，失败输出反馈到下一轮 prompt |
| Session 持久化 | `internal/runlog` + `internal/sessions` | ✅ JSON 写入 `.canx/sessions/` |
| CLI | `cmd/canxd` | ✅ `run` + `sessions list/show` 子命令 |
| Eval suite | `evals/agentic` | ✅ mock 套件 + 真实 Codex smoke |
| Eval report | `cmd/canx-eval-report` | ✅ `make report` 生成 JSON + Markdown |
| 工作区加载 | `internal/workspace` | ✅ README + AGENTS.md + docs/*.md |

### 已验证可用

```
make test           → 11 包全绿
make eval           → 3 个 agentic eval case 全 pass（mock）
make report         → 生成 evals/reports/latest.md
真实 Codex smoke    → decision=stop，约 22s
CLI mock run        → decision=stop，约 1.6s
CLI 真实只读任务    → worker 正确读 README，输出 [canx:stop]
```

---

## 架构图

```
canxd CLI
    │
    ▼
loop.Engine.Run(ctx, Config, workspace.Context)
    │
    ├── Planner.Plan(goal) ──→ []Task
    │       ├── SingleTaskPlanner  （默认）
    │       └── CodxPlanner        （-planner codx，调用 Codex 分解）
    │
    ├── for each active Task:
    │       ├── buildPrompt(goal, repo, tasks, turns)
    │       ├── Runner.Run(prompt) ──→ codex.Result
    │       │       ├── ExecRunner   → codex exec -（真实）
    │       │       └── MockRunner   → 内存（测试）
    │       ├── runValidation(commands) → bool + output
    │       └── switch: stop / escalate / continue
    │
    └── runlog.WriteSessionReport → .canx/sessions/<id>.json
```

---

## 下一步优先级

完整分析见 `docs/2026-03-18-framework-comparison.md`（含 LangGraph / Codex App Server / OpenClaw 深度比对）。

| 优先级 | 方向 | 说明 |
|---|---|---|
| **P0** | AppServerRunner | 替换 `codex exec -` subprocess，接入 Codex App Server JSON-RPC。每轮不再新建进程，Thread 跨 turn 复用上下文。 |
| **P1** | 角色分化上下文注入 | Planner / Worker / Reviewer 各用精简 prompt。Planner 最轻，Reviewer 只看 diff + task goal。参考 OpenClaw 最小知识原则。 |
| **P2** | 结构化 stop payload | `[canx:stop:{"summary":"...","files_changed":[...]}]`，Engine 解析写入 session report，供下一 task 引用。 |
| **P3** | Turn Checkpointing | 每轮写检查点，支持 resume，参考 LangGraph checkpointing。 |
| **P4** | 错误模式持久化 | Validation 失败写入 `.canx/patterns.md`，每次 run 时加载注入 Worker prompt 头部。自改进循环。 |
| **P5** | 并发 Worker | 先在 `Config` 加 `MaxConcurrentWorkers` / `MaxSpawnDepth` 预埋，再实现 goroutine 并发调度。 |
| **P6** | Reviewer Worker | 第二个 Runner 调用做 AI review，替换当前纯规则的 `review.Evaluate`。 |
| **P7** | 可观测性 | structured trace log（JSON），可接入外部分析工具。 |

---

## 工程规则（每次改动前读一遍）

- **不要重新实现** Codex、模型运行时、Docker 沙箱、通用 agent 框架。
- **优先复用** Codex CLI / App Server；CanX 只做编排逻辑。
- **每个循环必须有显式上限**：turn count、budget、timeout、exit 条件。
- **改动前写测试**：`go test ./...` 绿了才算完成。
- **改完跑 `make fmt` + `make build` + `make test`**。
- **并行修改同一文件**：禁止，单任务单文件范围。

---

## 关键文件快速索引

```
cmd/canxd/main.go           CLI 入口，flags 解析，Engine 组装
internal/loop/engine.go     核心控制流（最重要的文件）
internal/tasks/codx_planner.go   AI 任务分解
internal/codex/exec_runner.go    真实 Codex 进程接口
evals/agentic/suite_test.go      端到端评测套件
docs/runbook.md             验证过的可运行命令
docs/prompt-templates.md    写 goal 的推荐模板
docs/framework-comparison.md    外部框架对比与演进方向
```

---

## 沙箱说明

Codex 默认 `read-only` 沙箱，worker **无法修改文件**。要允许写入：

```toml
# ~/.codex/config.toml
sandbox = "workspace-write"
```

只读模式下 worker 会诚实输出 `[canx:stop]`，CanX 可以正确识别（不会崩溃）。
