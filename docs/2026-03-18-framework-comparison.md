# 多 Agent 框架横向对比与 CanX 演进方向

**日期：** 2026-03-18
**背景：** 结合当前框架调研和 CanX 实际代码，分析 CanX 的定位与演进路径

---

## 一、现有多 Agent 框架概况

### 1.1 Python 生态三大框架

#### LangGraph（LangChain，生产级）

- 工作流是有状态的有向图（DAG），支持环路
- `create_supervisor` 一行创建中心路由节点，子 agent 可以是任意 LLM
- 内置状态持久化、检查点（配合 LangSmith 可观测）、人工介入点
- **核心优势：成熟度最高，最适合复杂分支、有状态、长时运行工作流**

#### CrewAI（社区最大，100K+ 开发者）

- 角色驱动（Role / Backstory / Goal），人体组织隐喻
- 支持层级流程：自动生成 Manager agent
- **核心优势：低代码上手，定义角色和任务最直观**

#### AutoGen / AG2（Microsoft）

- 对话驱动：agent 之间互相"说话"来协作
- 2026-03-05 新增 Intelligent Handoffs，AG2 做路由决策
- **核心优势：对话模式自然，适合探索性、协商性任务**

#### smolagents（HuggingFace）

- 极简：agent 直接写 Python 代码执行（Code-first）
- 适合研究原型，生产环境风险高（任意代码执行）

---

### 1.2 Codex 生态的最新演进（对 CanX 影响最大）

#### Codex App Server（2026 年初正式发布）

OpenAI 发布了完整的 JSON-RPC over stdio 协议，核心三层原语：

```
Item   → 原子输入/输出单位（有 started / delta / completed 生命周期）
Turn   → 一次 agent 工作单元（由用户输入触发）
Thread → 持久化会话容器（支持 create / resume / fork / archive）
```

架构组件：Stdio reader → Codex message processor → Thread manager → Core threads。
多 Thread 支持已在 TUI 中实现，Thread Manager 管理多个并发 Core 线程。
与 CLI、VS Code 插件、Web App、macOS 桌面、JetBrains / Xcode 共享同一 API。

**这直接影响 CanX 的 AppServerRunner 路径设计。**

#### Claude Code Agent Teams（v2.1.32+ 可用）

- Teammate 之间直接点对点消息（不需要 lead 中转）
- 共享 task list，可自协调
- 17 个 Hooks 生命周期事件：PreToolUse、PostToolUse、SubagentStart、TeammateIdle、TaskCompleted...
- 支持 split-pane 模式（每个 teammate 独立终端）
- Delegate 模式：Lead 限制为协调工具，禁止自己拿实现任务

#### OpenAI Agents SDK Handoffs

- `handoff()` 函数把 agent 间委托暴露为工具，LLM 决定何时触发
- 支持 `inputFilter`（控制历史传递范围）和 `onHandoff` 回调
- **注意：路由是 LLM 驱动的，不是确定性调度**

---

## 二、CanX 当前的差距（诚实版）

| 维度 | 外部框架 | CanX 现状 |
|---|---|---|
| 状态持久化 | LangGraph：检查点、可从任意 turn 恢复 | `.canx/sessions/*.json`，只写最终结果，无恢复机制 |
| 工作流建模 | LangGraph：图 + 条件边，任意依赖 | 线性循环，无分支表达能力 |
| Agent 通信 | Claude Teams：Teammate 点对点消息 | 无：单 Worker，只有 stop / escalate 两种信号 |
| 传输协议 | App Server：JSON-RPC / JSONL，持久 Thread | `codex exec -` subprocess，每轮新进程，上下文丢失 |
| 可观测性 | LangGraph + LangSmith：完整 trace | 只有 session JSON，无 trace |
| 多 agent 并发 | 各框架都有 | 不存在 |
| Reviewer agent | Claude Teams / LangGraph 可配置 | `review.Evaluate` 是纯规则，无 AI 判断 |

---

## 三、CanX 的真实优势

1. **Go 原生，零 Python 依赖**：可以 `go install` 到任何 CI/CD 流水线，直接嵌入 Go 服务，启动快，无运行时依赖。

2. **确定性 validation gate**：`go test ./...` 的结果是调度决策的一等公民。所有 Python 框架都没有原生的"构建/测试通过才继续"语义。

3. **有界循环语义**：max-turns + budget-seconds + stop-marker 的组合防止 LLM 无限循环。Codex 内置多 agent 和 Claude Code subagent 都不保证这个。

4. **AGENTS.md / workspace-aware**：`workspace.Load` 把仓库上下文（README + AGENTS.md + docs）精确注入每个 prompt，专为 Codex 工作流优化，Python 框架没有对应设计。

5. **Runner 接口抽象**：ExecRunner → AppServerRunner 路径是可以平滑迁移的，不用重写控制流。

6. **自用驱动**：CanX 用来构建 Tradex，并最终用来构建 CanX 本身。需求是真实的，反馈循环紧密。

---

## 四、演进方向：取长补短，不造轮子

### 4.1 最高优先级：接入 Codex App Server

当前 `codex exec -` 每轮 fork 一个进程，丢失上下文，无法复用 Thread。App Server 的 Thread 原语完全匹配 CanX 的 `sessions.Registry` 设计意图。

目标接口（不改变 Engine 控制流）：

```go
type AppServerRunner struct {
    threadID string       // 映射到 App Server 的 Thread
    conn     *AppServerConn  // JSON-RPC over stdio
}

func (r *AppServerRunner) Run(ctx context.Context, req Request) (Result, error) {
    // 发送 Turn，流式接收 Items，组装 Result
}
```

**不要自己造 JSON-RPC 客户端**——直接对接 App Server 的 stdio 协议规范。
`sessions.Registry` 的 Spawn / Steer / Close 最终可以映射到 Thread create / send / archive，Registry 退化为代理层而非自建状态。

**取自 App Server**：持久化 Thread、流式输出、多 Thread 并发
**CanX 保留**：validation gate、stop-marker 解析、task scheduling

### 4.2 从 LangGraph 借鉴检查点，不引入 Python

LangGraph 的 checkpointing 思想值得借鉴，但不需要 Python。做法是每轮 turn 后写一次检查点，而不是只写最终结果：

```go
type Checkpoint struct {
    SessionID string
    Turn      int
    Tasks     []tasks.Task
    Turns     []Turn
    CreatedAt time.Time
}
// Engine.Run 每轮写 checkpoint → 支持从任意 turn resume
```

这让 CanX 获得 LangGraph 的可恢复语义，实现是 Go 的 JSON 文件，无需任何 Python 依赖。

### 4.3 两类 agent 角色分工（不是"任意图"）

不需要引入 LangGraph 的完整图模型。CanX 的场景只需要把当前单一 Worker 扩展到三种角色：

```
CodxPlanner（已有）    → 生成 tasks
Implementer Worker（已有）→ 执行每个 task
Reviewer Worker（待建）  → review 实现结果，输出 approved / rejected / comment
```

Reviewer Worker 就是另一个 `codex.Runner` 调用，prompt 是 worker 的 diff + 任务目标：

```go
reviewResult, _ := e.ReviewRunner.Run(ctx, codex.Request{
    Prompt:  buildReviewPrompt(task, workerOutput, diff),
    Workdir: e.Workdir,
})
// 解析 approved / rejected / comment
```

**取自 Claude Code 的 Delegate 模式**：Planner 产生 tasks 后，Engine 本身不写代码，只调度 Worker。

### 4.4 并发调度（参考 OpenClaw hub-and-spoke 模式）

Go 的 goroutine 天然适合多 Worker 并发，不需要外部框架：

```go
results := make(chan taskResult, len(tasks))
for _, t := range independentTasks {
    go func(task tasks.Task) {
        outcome, _ := e.runTask(ctx, task)
        results <- taskResult{task: task, outcome: outcome}
    }(t)
}
```

前提是 Task 之间没有依赖。有依赖的 task 仍然顺序执行（当前实现已是如此）。

---

## 五、明确不做的事

| 想做但不要做 | 理由 |
|---|---|
| 自建 agent 间消息协议 | App Server 的 Thread / Turn / Item 已经是这个协议，直接用 |
| 引入 LangGraph 作为 Go 依赖 | Python 依赖破坏 Go-native 优势；CanX 场景不需要通用图 |
| 自建 Docker 沙箱 | Codex 内核级沙箱（macOS Seatbelt / Linux Landlock）已足够 |
| 自建模型运行时 | 永远不要 |
| 做成通用 agent 框架 | 会让 CanX 变成第 N+1 个 LangGraph，失去 Codex 专用编排层的价值 |
| 接入 LLM 的 Handoffs 路由 | OpenAI Agents SDK 的 handoffs 是 LLM 决定路由，引入会稀释 CanX 确定性调度的核心差异 |
| 自建 SWE-bench 评估集 | 用 OpenHands 的 benchmarks 作为参考即可 |

---

## 六、CanX 的正确定位

```
┌──────────────────────────────────────────────────────────┐
│  CanX（Go 确定性编排层）                                    │
│                                                           │
│  Validation Gate  ← 核心差异：所有 Python 框架无原生支持     │
│  Task Scheduler   ← 确定性调度，不是 LLM 路由               │
│  App Server 客户端 ← 不造协议，复用 Codex 原生 Thread        │
│  Workspace-aware  ← AGENTS.md + docs 注入，Python框架无    │
│  Go 原生          ← 嵌入 CI/CD，零 Python 依赖              │
├──────────────────────────────────────────────────────────┤
│  Codex App Server（JSON-RPC 传输层，OpenAI 维护）           │
│  Thread / Turn / Item 原语                                 │
└──────────────────────────────────────────────────────────┘
```

CanX 不应该是"Go 版 LangGraph"，而应该是"Codex 的确定性调度和验证层"。

**当 Codex 做一次调用时它是 AI，CanX 决定什么时候调用、调用什么、结果是否够好。**

这一层是所有现有框架都没有针对 Go + Codex 工作流特化过的地方，也是 CanX 不应该放弃的核心。

---

## 七、OpenClaw 深度分析：对 CanX 下一迭代的启发

**参考来源：** OpenClaw 源码深度阅读（2026-03-19）
**筛选原则：** 不重造 OpenClaw 的轮子，只提取 CanX 自己缺失的设计思路。

OpenClaw 把多 Agent 分成两层：静态 Channel 路由（不同渠道对应不同专家）+ 运行时动态 Spawn（父子 Agent 关系）。CanX 不是 Bot 平台，Channel 路由不适用，但第二层的工程设计有很多可借鉴的地方。

### 7.1 角色分化的上下文注入（可立刻改）

OpenClaw 给子 Agent 加载的是精简版 workspace：去掉 HEARTBEAT / MEMORY，只保留 5 个核心文件。理由是子 Agent 只处理当前子任务，不需要背负主 Agent 的全部历史——**最小知识原则**。

CanX 的现状：`buildPrompt` 对所有 Runner（Planner、Worker、Reviewer）注入同样的 workspace 上下文。

**改进方向：** 按角色区分注入内容。

```go
type PromptRole string
const (
    RolePlanner    PromptRole = "planner"    // 只需要 README + goal，不需要 docs 细节
    RoleWorker     PromptRole = "worker"     // 需要完整 workspace + 当前 task + 前几轮历史
    RoleReviewer   PromptRole = "reviewer"   // 需要 task goal + worker diff，不需要其他 docs
)
```

Planner 调用用最精简的 prompt（不注入 docs），Worker 用完整上下文，Reviewer 只注入 diff + task 目标。这样可以节省 token 并让每个角色更专注。

---

### 7.2 结构化的 stop 信号（低成本，高收益）

OpenClaw 的子任务结束时发送 `task_completion` 事件，包含：
- 完成状态
- 结果内容
- 消耗 token 数
- `replyInstruction`：`SILENT`（父 Agent 静默消化）或具体指令文本

CanX 现在的 stop 信号只是纯文本 `[canx:stop]`，没有任何结构化 payload。

**改进方向：** 扩展 stop marker 携带 JSON payload：

```
[canx:stop:{"summary":"fixed the bug","files_changed":["internal/loop/engine.go"]}]
```

Engine 解析这个 payload，写入 session report，让下一个 task 的 prompt 可以引用上一个 task 的结构化结论，而不是从非结构化文本里猜。

---

### 7.3 Spawn 深度限制（并发 Worker 上线前必须加）

OpenClaw 的两道安全门：
- `maxSpawnDepth`：默认 1，必须显式配置才能超过一层，防止 Agent 指数级扩散
- `maxChildrenPerAgent`：默认 5，单个 Agent 最多同时 spawn 5 个子 Agent

CanX 准备在 P3 实现并发 Worker。在实现之前，Config 里应该先加这两个参数作为预埋：

```go
type Config struct {
    Goal                string
    MaxTurns            int
    BudgetSeconds       int
    ValidationCommands  []string
    MaxConcurrentWorkers int  // 新增，默认 1（顺序）
    MaxSpawnDepth        int  // 新增，默认 1（禁止子 Agent 再 spawn）
}
```

并发 Worker 功能没做完之前，这两个参数只做校验，不影响现有逻辑。

---

### 7.4 错误模式持久化（自改进机制）

OpenClaw 的 AGENTS.md 在 Agent 犯错时**自动更新**，把踩坑记录写入文件，下次启动时注入到 system prompt。这是最朴素的自改进循环。

CanX 当前的状态：Validation 失败的输出会作为下一 turn 的 prompt 输入（已实现），但这个反馈是**会话内的、一次性的**，跑完就消失了。

**改进方向：** 持久化失败模式到 `.canx/patterns.md`：

```
# CanX Error Patterns
## go test failure: missing import
- 出现次数：3
- 最近一次：session-abc123
- 处置：Worker 自动 fix，第二 turn 通过
```

下次 run 时，`workspace.Load` 把 patterns.md 一并加载，注入 Worker prompt 头部。Worker 在开始前就能看到历史上踩过的坑。

这是 CanX 自托管开发的核心机制——跑得越多，越知道怎么避坑。

---

### 7.5 不要借鉴的部分

| OpenClaw 机制 | 理由 |
|---|---|
| Channel 路由（不同群聊路由到不同 Agent） | CanX 是 CLI，不是 Bot 平台 |
| SOUL.md / IDENTITY.md / HEARTBEAT.md | Persona 和定时任务，与 CanX 的代码交付场景无关 |
| WebSocket / ACP 协议栈 | CanX 的传输层应该是 Codex App Server JSON-RPC，不是 WebSocket |
| agent-to-agent 消息（5 轮上限） | CanX 当前是父 → 子单向派发，暂不需要双向对话 |
| workspace git init | CanX 的工作目标仓库本身就是 git repo，`.canx/` 随之管理即可 |

---

## 八、演进优先级排序（含 OpenClaw 启发更新）

| 优先级 | 方向 | 内容 | 来源 |
|---|---|---|---|
| P0 | AppServerRunner | 接入 Codex App Server JSON-RPC，替换 subprocess 模式 | Codex 官方 |
| P1 | 角色分化上下文注入 | Planner / Worker / Reviewer 各自的精简 prompt | OpenClaw |
| P2 | 结构化 stop payload | `[canx:stop:{...}]` 携带 summary + files_changed | OpenClaw |
| P3 | Turn Checkpointing | 每轮写检查点，支持 resume | LangGraph |
| P4 | 错误模式持久化 | `.canx/patterns.md` + workspace.Load 注入 | OpenClaw |
| P5 | 并发 Worker | goroutine 并发调度独立 task（先加 Config 预埋） | 内部设计 |
| P6 | Reviewer Worker | 第二个 Runner 调用做 AI review | Claude Teams |
| P7 | 可观测性 | structured trace log，可接入外部分析工具 | LangGraph |
