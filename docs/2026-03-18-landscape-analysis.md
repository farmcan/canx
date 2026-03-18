# CanX 与主流工具的差异分析

**日期：** 2026-03-18
**参考：** Codex CLI 文档、Claude Code 文档、OpenClaw ACP、LangGraph Supervisor、AutoGen Magentic-One、OpenHands

---

## 一、各工具定位速览

### Codex CLI（OpenAI）

- 内置多 agent（experimental）：`multi_agent = true` 或 `/experimental`
- 三种内置角色：`default`、`worker`、`explorer`，可自定义
- 内置并发：最多 6 个 agent 线程，用户或 LLM 决定何时分叉
- 云端委托：`codex cloud exec`，fire-and-forget，返回 diff
- 沙箱：**内核级**（macOS Seatbelt / Linux Landlock + seccomp），隔离强
- 上下文：1M token
- AGENTS.md 是 Linux Foundation 开放标准

**关键限制：**
- 多 agent 是 LLM 自主决定分叉，不是确定性的任务调度
- 没有外部 validation gate（构建/测试结果不会反馈到调度决策）
- 没有 stop-on-validation-pass 的有界循环语义
- 没有跨会话的结构化任务状态

### Claude Code（Anthropic）

- 内置 subagent：Task 工具产生独立 context window 的子 agent
- 实验性 Agent Teams：多个 Claude Code 实例点对点协作
- 应用层 hooks：17 个生命周期事件，可注入任意校验逻辑
- 上下文：200K token
- CLAUDE.md 分层 hierarchy（5 层）

**关键限制：**
- Subagent 在同一次会话内部，不是独立进程
- Agent Teams 是实验性，没有确定性的任务路由
- hooks 与 agent 共享进程边界，不是内核级隔离
- 没有构建/测试验证与调度逻辑的原生集成

### OpenClaw / ACP

- ACP（Agent Communication Protocol）：结构化 agent 间消息协议
- WebSocket 驱动的有状态会话（`sessions_spawn`、`sessions_send`）
- `acpx`：无头 CLI 客户端，持久化多轮会话
- 支持 Discord/Telegram 线程绑定、hub-and-spoke、pipeline、swarm 模式
- 解决的核心问题：消息格式不统一、无 confidence 建模、无 deadline 约束、无信誉追踪、难以调试

**定位：** 协议层 + 平台，不是面向软件交付工作流的编排器

### LangGraph Supervisor（LangChain）

- `create_supervisor` 函数：中心节点路由到专化子 agent
- `Command(goto="next_node")` API 做确定性路由
- `@task`/`@entrypoint` 装饰器定义图
- Python 生态，有 cycle 支持，但需要手写 stop condition
- 强项：通用工作流图，弱点：无 coding-specific 验证/评审集成

### AutoGen Magentic-One（Microsoft）

- Orchestrator + 5 个专化 agent（WebSurfer、FileSurfer、Coder、Terminal 等）
- Orchestrator 做动态计划、进度追踪、错误恢复
- 模块化，agent 可以插拔不需要 retrain
- 强项：通用任务（web + file + code）；弱项：与特定代码工具（Codex）的深度集成

### OpenHands

- Docker 沙箱 + REST Agent Server（Python/Kubernetes）
- actions-observations 通信模型
- SWE-bench 优化，单 agent 擅长从 issue 到 PR 的全流程
- 强项：标准化 sandbox；弱项：多 repo 跨项目协调

---

## 二、CanX 在哪里有差异

诚实的差异分析：**不是"别人都没有的功能"，而是"别人组合起来不方便做的事情"。**

### 2.1 Codex 内置多 agent 的根本局限

Codex 的多 agent 是 LLM 驱动的自主决策。这意味着：

```
用户说"并行做这两件事" → Codex 大模型判断是否 spawn → spawn 了你不知道 → 结果回来你不知道过程
```

CanX 想做的是确定性的编排：

```
Supervisor 分析 goal → 确定性生成 task 列表 → 按依赖关系调度 → 每轮 validation → 有界停止
```

这两种模式在以下场景的差异是实质性的：

| 场景 | Codex 内置多 agent | CanX 目标 |
|---|---|---|
| "修一个 bug" | 合适，简单任务 | 过重 |
| "给 Tradex 实现一个新模块，测试通过为止" | 可能多次失败不自知 | Validation gate 阻止回归 |
| "同时调度 3 个 worker，合并结果" | 实验性，不稳定 | 确定性 goroutine 并发 |
| "每轮检查 `go test ./...`，失败则继续" | 不支持 | 核心功能 |
| "跨多个 repo session 的协调" | 不支持 | AppServerRunner 路径 |

### 2.2 与 Claude Code 的差异

Claude Code 的 subagent 是**会话内部**的（同一 context window 的分叉），不是进程级隔离的独立 Codex 实例。CanX 的目标是**进程级独立的 Codex worker**，每个 worker 有自己的上下文、自己的 sandbox 和自己的生命周期。

这对于代码交付工作流的意义在于：
- 不同 worker 可以针对不同仓库或不同文件集
- Worker 失败不会污染 supervisor 的上下文
- Reviewer worker 和 implementer worker 完全隔离

### 2.3 与 LangGraph 的差异

LangGraph 是 Python 的通用工作流图。CanX 的价值在于：
- **Go 生态**：直接运行在 Go 服务器上，无需 Python 运行时
- **Codex-native**：直接对接 `codex exec` / `app-server`，不需要 LangChain 层
- **Coding-specific**：validation gate 是第一公民（`go test ./...`、`go build ./...`）

---

## 三、多 agent 设计合理性评估

### 3.1 当前实现 vs 开源 pattern

当前代码实际上是**单 agent + 有界重试循环**，不是真正的多 agent：

```
Engine.Run() → Runner.Run() → [validation] → [stop or continue]
```

这和 Ralph-lite 的目标是一致的，但与设计文档中的架构图（supervisor + multiple workers + reviewer）还有距离。

对照开源实现：

**Magentic-One 的 Orchestrator 做什么：**
1. 接收 goal
2. 生成 plan（任务列表）
3. 选择最合适的 agent 执行当前步骤
4. 观察结果，更新 plan
5. 检测 stall/error，重新规划

CanX 现在缺少步骤 2 和 3 的实现。`tasks` 包存在但从未接入 Engine。

**LangGraph Supervisor 的核心模式：**
```python
supervisor = create_supervisor(
    agents=[coder_agent, reviewer_agent, tester_agent],
    model=model,
)
```

这对应 CanX 中应有的：
```go
engine := loop.Engine{
    Supervisor: codex.NewSupervisorRunner(...), // 分解 goal → tasks
    Workers:    []codex.Runner{impl_runner, review_runner},
    Validator:  validation.Commands{...},
}
```

### 3.2 合理的多 agent 设计建议（参考开源，不造轮子）

以下是对照开源项目总结的合理 CanX 多 agent 架构：

#### 阶段一（当前 Ralph-lite，已基本完成）
```
Human goal → Supervisor prompt → Single Codex worker → Validation → Stop/Continue
```
**这是合理的起点。** 不要急于扩展。

#### 阶段二（下一个里程碑）
参考 Magentic-One 的 Orchestrator 模式：

```
Human goal
  → Supervisor agent（via Codex）: 生成结构化 task list → JSON
  → task 写入 tasks.Task 队列
  → Engine 按依赖顺序调度 worker
  → 每个 worker 返回 summary
  → Validation gate（build + test）
  → Supervisor 重新评估：done / continue / re-plan
```

关键点：Supervisor 本身也是一个 Codex 调用，不是硬编码逻辑。它输出结构化 JSON（task list），CanX 负责解析和调度。

#### 阶段三（多 worker 并发）
参考 OpenClaw hub-and-spoke 模式：

```go
// 并发调度多个独立 worker
tasks := supervisor.Decompose(goal)
results := make(chan codex.Result, len(tasks))
for _, t := range tasks {
    go func(task tasks.Task) {
        result, _ := engine.RunTask(ctx, t)
        results <- result
    }(t)
}
```

Go 的 goroutine 天然适合这个模式，不需要引入外部 Python 框架。

#### 阶段四（AI reviewer，而不是规则 gate）
参考 Claude Code 的 deliberation pattern：

```
Implementer worker output → Reviewer worker（独立 Codex session）→ review.Result
```

当前 `review.Evaluate` 是纯规则的，真正有价值的 review 需要另一个 Codex 调用（reviewer prompt + worker output）返回 approved/rejected/comment。

---

## 四、哪些东西不应该自己做

| 功能 | 现状 | 建议 |
|---|---|---|
| Agent 间消息协议 | sessions 包自建 | 对接 OpenClaw ACP 的 `acpx`，或使用 Codex app-server JSON-RPC |
| 通用工作流图 | 无，也不需要 | 不引入 LangGraph，保持线性 + fork 模式即可 |
| Web/文件/浏览器 agent | 无 | 不需要，Codex 已经处理文件操作 |
| 模型运行时 | 无 | 永远不自建，依赖 Codex |
| Docker sandbox | 无 | 依赖 Codex 自带的 kernel sandbox |
| SWE-bench 评估 | 无 | 用 OpenHands benchmarks 作为参考，不自建 |

**当前 `sessions.Registry` 的风险：** 它的 Spawn/Steer/Close 语义和 OpenClaw ACP 的 `sessions_spawn`/`sessions_send`/`sessions_close` 高度重叠。一旦接入 `AppServerRunner`，这个包可能会变成冗余层。建议现阶段保留（ExecRunner 下够用），但在 AppServerRunner 接入时重新评估是否直接代理 ACP 会话。

---

## 五、CanX 的实际优势（诚实的版本）

CanX 当前没有"独一无二"的技术优势，但有几个**组合优势**：

1. **Go 原生**：无 Python 依赖，可以直接嵌入 Go 服务，启动快，部署简单。

2. **确定性 validation gate**：`go test ./...` 的结果决定是否继续，这是 Codex 内置多 agent 和 Claude Code subagent 都不原生支持的决策点。

3. **Codex-first 的仓库意识**：`workspace.Load` 把 README + AGENTS.md + docs 注入每个 worker prompt，这是针对 Codex 工作流优化的，其他框架是通用的。

4. **自用驱动**：CanX 用来构建 Tradex，并最终用来构建 CanX 本身。自用意味着反馈循环紧密，需求是真实的。

5. **架构灵活性**：Runner 接口允许 ExecRunner → AppServerRunner 的平滑迁移，而不需要重写控制流。

**最重要的一点：** CanX 的价值不在于替代 Codex 或 Claude Code，而在于**在它们之上增加一层确定性的编排和验证**，让 AI-to-AI 的软件交付循环变得可重复、可审计、可停止。

---

## 六、近期风险

- **Codex 内置多 agent 快速成熟**：如果 Codex 在未来 6 个月内原生支持 validation gate 和确定性任务调度，CanX 的 ExecRunner 路径会被边缘化。
  - **应对：** 尽快走 AppServerRunner 路径，深度对接 Codex 的 JSON-RPC 接口，做 Codex 的协调层而不是竞争层。

- **tasks 包继续是孤岛**：如果下一个迭代还没有把 Task 接入 Engine，CanX 就一直是"有界重试器"，而不是真正的多 agent 编排器。

- **review gate 没有 AI 判断**：当前 review 是纯规则的（validated + inScope），没有实际的 AI review。对 Tradex 这样的真实项目，规则 gate 不够用。
