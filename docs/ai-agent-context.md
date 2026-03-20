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
| 编排引擎 | `internal/loop` | ✅ Engine.Run，支持 bounded 多 task 调度、静态并行、受控动态 spawn、stop/escalate 信号 |
| Codex 接入 | `internal/codex` | ✅ ExecRunner、MockRunner、最小版 AppServerRunner（`approval=never`） |
| Validation gate | `internal/loop` | ✅ `sh -c` 命令，失败输出反馈到下一轮 prompt |
| Session 持久化 | `internal/runlog` + `internal/sessions` | ✅ JSON 写入 `.canx/sessions/` |
| CLI | `cmd/canxd` | ✅ `run` + `sessions list/show` 子命令 |
| Eval suite | `evals/agentic` | ✅ mock 套件 + 真实 Codex smoke |
| Eval report | `cmd/canx-eval-report` | ✅ `make report` 生成 JSON + Markdown |
| 工作区加载 | `internal/workspace` | ✅ README + AGENTS.md + docs/*.md |

### 已验证可用

```
make test           → 11 包全绿
make eval           → 4 个 agentic eval case 全 pass（mock，含 spawn_child_task）
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
    ├── Scheduler 选择可运行 task（受 MaxWorkers / 冲突控制约束）
    ├── for each runnable Task:
    │       ├── buildPrompt(goal, repo, tasks, turns)
    │       ├── Runner.Run(prompt) ──→ codex.Result
    │       │       ├── ExecRunner   → codex exec -（真实）
    │       │       └── MockRunner   → 内存（测试）
    │       ├── parse stop/spawn markers
    │       ├── runValidation(commands) → bool + output
    │       └── supervisor 更新 task / child task / stop-escalate
    │
    └── runlog.WriteSessionReport → .canx/sessions/<id>.json
```

---

## 下一步优先级

完整分析见 `docs/2026-03-18-framework-comparison.md`（含 LangGraph / Codex App Server / OpenClaw 深度比对）。

### 当前缺口说明

`EventStore`、`Rooms`、`server.go`（Dashboard）已在代码里，`P0 实时事件流` 和 `P0.5 SSE 长连接 tail` 已完成：`Engine` 每轮会即时写入 `session_started`、`task_state`、`turn_completed`，同步刷新 `run.json`，`/api/runs/:id/events/stream` 也会持续跟随新事件。当前剩余的主要缺口是 **session detail 的结构化 turns 还没有在 UI 中做成更友好的卡片视图**，以及 **review verdict 还没有接入更强的策略执行与 UI 呈现**。

### 优先级表

| 优先级 | 方向 | 说明 |
|---|---|---|
| **P0** | 实时事件流 | ✅ 已完成：`Engine` 运行中实时写事件，并同步刷新 `run.json`。 |
| **P0.5** | SSE 跟随新事件 | ✅ 已完成：`/api/runs/:id/events/stream` 现在是长连接 tail，不再只是一次性读历史。 |
| **P1** | 角色分化上下文注入 | ✅ 已完成最小版：Planner 使用轻量上下文，Worker 使用完整上下文，Reviewer 现在也有独立 prompt builder；ReviewRunner 仍是可选。 |
| **P2** | 结构化 stop payload | ✅ 已完成最小版：支持 `[canx:stop:{"summary":"...","files_changed":[...]}]`，Engine 会写回 `task.Summary` / `task.FilesChanged`，并在后续 worker prompt 里注入 completed task 结论。 |
| **P3** | 错误模式持久化 | ✅ 已完成最小版：validation 失败会去重追加到 `.canx/patterns.md`，`workspace.Load` 会加载它，worker prompt 注入 `Known failure patterns`。 |
| **P4** | AppServerRunner | ✅ 已完成最小版：支持 `codex app-server`、`SessionKey -> ThreadID` 复用、`approval=never`。下一步是 approval 事件处理与更完整的 item/delta 暴露。 |
| **P5** | Turn Checkpointing | 每轮写检查点，支持 resume，参考 LangGraph checkpointing。 |
| **P6** | 并发 Worker | ✅ 已完成最小版：支持 `MaxWorkers`、文件级冲突控制、worker 独立 session；下一步是增强调度策略与 resume。 |
| **P7** | Reviewer Worker | 第二个 Runner 调用做 AI review，替换当前纯规则的 `review.Evaluate`。 |

### 建议执行顺序

```
近期（可并行，代码不重叠）：
  P1  角色分化 prompt      → ✅ 已完成 planner/worker/reviewer 三种 role 的最小边界
  P0.75 UI 自动刷新        → ✅ 已完成：SSE 事件会驱动 runs/tasks/actions/session 面板刷新

随后：
  P3.5 session 增量持久化    → ✅ 已完成：session report 会在 `session_started` 和每次 `turn_completed` 后即时刷新，并持久化结构化 turn 详情
  P7 AI reviewer policy      → ✅ 已完成最小版：ReviewRunner 支持稳定 JSON verdict schema（approved/reason/warnings），自由文本仍可回退

再往后（复杂度高）：
  P4  AppServerRunner      → internal/codex 新增实现，不改 Engine 控制流
```

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
