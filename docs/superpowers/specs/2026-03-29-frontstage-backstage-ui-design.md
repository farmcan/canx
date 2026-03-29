# CanX Frontstage / Backstage UI Design

**Date:** 2026-03-29

## Goal

为 `CanX` 设计一套双模式前端：

- **Frontstage（前台演示模式）**：大场景、拟人化、适合直播/演示/状态感知
- **Backstage（后台控制台模式）**：高密度工程观测，适合调试、接管、回看

两种模式必须共用同一套运行事实来源，不能演化成两套系统。

## Why this direction

当前本地 dashboard 已经能展示 `run / task / session / context / events / rooms`，但偏工程调试视图，缺少：

- 面向非工程视角的阶段表达
- 对多代理协作的空间化表达
- 对外演示时的视觉记忆点

另一方面，直接复用 `Star-Office-UI` 不合适，因为其核心模型是 `agent -> state -> room area`，而 `CanX` 的核心模型是：

- `Run`
- `Task`
- `Session`
- `Turn`
- `Review`
- `Event`
- `Room / Message`

因此本设计选择：**复用 CanX 数据模型，自建前端表现层**。

## Product shape

## Design philosophy

### AI anthropomorphism is part of the product, not decoration

本项目的前台设计，不把 AI 当作抽象状态点来展示，而是把 AI 作为**可感知的角色**来表达。这里的“拟人化”不是做卡通包装，而是把复杂的 agent 行为转译成用户更容易理解的产品语言：

- 谁在处理当前工作
- 它此刻正在做什么
- 它是在推进、检查、交接，还是遇到了问题
- 一次 LLM call、一次 Tool Use、一次 review，到底意味着什么

因此前台的目标不是把 JSON 漂亮地排版，而是把运行中的系统行为，转成一段段可读、可感知、可记忆的角色动作。

### Every important flow becomes a personified beat

拟人化不只服务于“大阶段”，也要服务于更小粒度的真实交互。也就是说：

- 一次 `LLM call`
- 一次 `Tool Use`
- 一轮 `turn completed`
- 一次 `review verdict`
- 一次 `validation result`

都应该能被抽象成一个 **beat**，并进一步映射为一个 **frame** 或一段短动画。

这样 CanX 的前台长期就不是“任务看板”，而是：

- **单个 agent 可视化器**
- **单轮交互可视化器**
- **当前 run / 集群协作可视化器**

三个层级复用同一套设计语言。

### Mode 1: Frontstage

目标问题：

- 现在在推进哪个阶段
- 哪些 worker 正在工作
- 当前卡在 review、validation 还是等待人工
- 这次运行整体是顺畅、阻塞还是失败

前台不追求展示全部细节，而追求：

- 状态可读
- 阶段可感知
- 人物/区域/动作一眼能看懂

### Mode 2: Backstage

目标问题：

- 当前有哪些 run
- 当前 task 的真实状态是什么
- 哪个 session 在处理哪个 task
- 最新 turn、validation、review 到底说了什么
- 人类是否发过 instruction，系统如何响应

后台继续承担工程调试和接管功能。

## Shared data model

前后台必须共用以下共享读模型：

- `RunSummary`
- `TaskView`
- `SessionView`
- `TurnView`
- `ReviewView`
- `EventView`
- `RoomView`

### Derived presentation fields

为了支持前台表达，需要在前端或后端轻量派生以下字段：

- `phase`
  - `planning`
  - `working`
  - `validating`
  - `reviewing`
  - `syncing`
  - `blocked`
  - `done`
- `actor_role`
  - `supervisor`
  - `worker`
  - `reviewer`
  - `human`
- `scene_zone`
  - `command`
  - `workbench`
  - `test_lab`
  - `review_gate`
  - `sync_port`
  - `incident_zone`
- `display_status`
  - 适合前台展示的人类可读摘要

## Frontstage interaction model

### Layout

前台采用大场景布局：

- 顶部：当前 run 概览、goal、decision、reason
- 中央：拟人化场景
- 底部或侧边：简短阶段栏和当前 task 摘要
- 右上角：切换到 Backstage

### Generic interaction-to-frame model

Frontstage 不应只服务于当前 CanX 的固定阶段，而应抽象成一个更通用的：

- **interaction -> beat -> frame**

其中：

- `interaction`
  - 一次用户输入
  - 一次 agent 回复
  - 一次工具调用
  - 一轮 turn 完成
  - 一个 review / validation verdict
- `beat`
  - 对 interaction 的展示层归类，例如：
    - `briefing`
    - `tool_use`
    - `build`
    - `inspect`
    - `review`
    - `handoff`
    - `incident`
    - `complete`
- `frame`
  - 前台实际看到的一帧或一段短动画

这意味着 Frontstage 后续可以面向更通用的 agent 系统，而不只绑定 CanX 的当前 task 状态机。

### Personification layer

在 `interaction -> beat -> frame` 之上，再加一层明确的**拟人化映射**：

- `interaction`
  - 系统真实发生的事
- `beat`
  - 产品层归类
- `persona action`
  - 角色在场景中的动作语义
- `frame`
  - 用户最终看到的画面

例如：

| Interaction | Beat | Persona action | Frame |
|---|---|---|---|
| LLM call started | `briefing` / `inspect` | 指挥员看图纸 / 检查员比对面板 | 角色移动到对应区域并播放动作 |
| Tool call running | `tool_use` | 工匠调工具、搬运部件 | 工坊区域点亮，角色做工动作 |
| Validation passed | `inspect` / `complete` | 检查员确认通过 | 仪表变绿，角色点亮确认 |
| Review rejected | `review` / `incident` | 审查员打回，值班员介入 | Review Gate 或告警区触发反馈 |
| Turn completed | `handoff` | 归档员交接结果 | Sync Port 出现归档动作 |

### Why this abstraction matters

如果未来接入不同类型 agent：

- 代码 agent
- research agent
- tool-calling agent
- multi-step UI agent

那么前台不能只理解 `planning / working / done` 这些窄状态，而应该能表达：

- 这一轮发生了什么
- 调了什么工具
- 结果是推进、回退、等待还是异常

因此建议将 Frontstage 的真实输入定义为一组 **beats**，而不是只读最终 task status。

### Scene zones

建议的固定区域：

- `Command Deck`：Supervisor / planner
- `Workbench`：worker 执行中
- `Test Lab`：validation / build / test
- `Review Gate`：reviewer 或 gate verdict
- `Sync Port`：session 持久化 / actions / completed task 归档
- `Incident Zone`：blocked / error / escalate

### Multi-agent stage expression

前台最终不应只有一个抽象主角，而应逐步演进成**多 agent 协同舞台**：

- `Supervisor` 常驻 `Command Deck`
- `Worker` 常驻 `Workbench`
- `Reviewer` 常驻 `Review Gate` 或 `Test Lab`
- `Ops / Sync` 常驻 `Sync Port`

活跃角色负责移动、说话、执行当前 beat；其他角色保持驻场和待命状态。这样用户看到的不是“一个小人在跑”，而是“多个 AI 角色在协同交付”。

### State mapping

建议的第一版阶段映射：

| Phase | Zone | 建议动作 |
|---|---|---|
| `planning` | `Command Deck` | 看图纸、分发任务 |
| `working` | `Workbench` | 搬砖、敲击、调工具 |
| `validating` | `Test Lab` | 跑仪表、冒测试条 |
| `reviewing` | `Review Gate` | 检视、放行、打回 |
| `syncing` | `Sync Port` | 搬运文件、归档、同步 |
| `blocked` | `Incident Zone` | 报警、维修、卡住 |
| `done` | `Command Deck` or `Sync Port` | 归档完成、亮绿灯 |

### Beat-to-scene mapping

建议增加一个更通用的 beat 映射层：

| Beat | Zone | 建议动作 |
|---|---|---|
| `briefing` | `Command Deck` | 接收目标、展示任务卡 |
| `tool_use` | `Workbench` | 调工具、操作台、搬运部件 |
| `build` | `Workbench` / `Test Lab` | 组装、编排、执行 |
| `inspect` | `Test Lab` | 检查、测量、比对 |
| `review` | `Review Gate` | 审查、放行、驳回 |
| `handoff` | `Sync Port` | 传递产物、归档、同步 |
| `incident` | `Incident Zone` | 警报、维修、等待 |
| `complete` | `Sync Port` | 完成收尾 |

第一版仍可继续使用 `phase`，但设计上应明确：**phase 只是 beat 的一种压缩视图**。

## Backstage interaction model

后台基于现有 dashboard 增强，不推翻重做：

- 左侧：runs / tasks / sessions
- 中间：task detail / session detail
- 右侧：events / review / actions / room messages
- 顶部：mode toggle、filters、status badges

第一版保持现有静态 HTML/CSS/JS 技术栈，避免前端体系一次性升级过大。

## Visual direction

### Core principle

前台强调“拟人化阶段表达”，但不要求一开始就是复杂游戏场景。第一阶段可以先做：

- 区域背景
- 角色占位
- 状态动画
- 简短气泡
- 区域高亮和轻量动效

### Recommended art direction

优先推荐：**工坊 / 调度中心混合风格**

理由：

- 比纯办公室更适合表达 `build / validate / review / sync / blocked`
- 比纯工业搬砖更容易兼容软件编排语义
- 更适合后续做 AI agent 的拟人化角色

## Asset plan

第一阶段不追求大而全，只需要最小闭环素材。

### Required images for MVP

#### 1. Scene background

- `frontstage-scene-bg`
  - 1 张
  - 内容：调度中心 / 工坊俯视或 3/4 俯视大场景
  - 用途：前台主背景

#### 2. Zone markers

- `zone-command`
- `zone-workbench`
- `zone-test-lab`
- `zone-review-gate`
- `zone-sync-port`
- `zone-incident`

可选：

- 先用代码绘制占位块替代
- 之后再换成正式美术

#### 3. Main avatar

建议先只做 1 个主角色，后续再扩展 supervisor / worker / reviewer 变体。

- `avatar-idle`
- `avatar-planning`
- `avatar-working`
- `avatar-validating`
- `avatar-reviewing`
- `avatar-syncing`
- `avatar-blocked`

每个状态建议：

- 4 到 8 帧
- 同尺寸
- 同视角
- 透明背景

#### 4. Optional effect layers

- `fx-progress`
- `fx-warning`
- `fx-review-pass`
- `fx-review-reject`
- `fx-sync`

#### 5. Generic beat cards / overlays

为了支持“每次交互变成一个画面”，建议追加轻量覆盖层：

- `frame-briefing`
- `frame-tool-use`
- `frame-inspect`
- `frame-review`
- `frame-handoff`
- `frame-incident`

这些不一定是完整背景，可以只是：

- 区域覆盖图
- 小道具
- 卡片弹层
- 短暂特效

### Google AI generation guidance

可让 Google Gemini 先生成：

- 背景概念图
- 角色设定图
- 各状态关键帧
- 道具 / 机器 / 区域 icon

不建议第一步就要求它直接生成最终可用 sprite sheet。更稳的流程是：

1. 角色设定
2. 状态关键帧
3. 统一尺寸和锚点
4. 再切图或转 sprite sheet

## Technical approach

### Backstage

沿用现有：

- `cmd/canxd/server.go`
- `cmd/canxd/ui/index.html`
- `cmd/canxd/ui/app.js`
- `cmd/canxd/ui/styles.css`

新增 mode toggle 和更明确的数据分区。

### Frontstage

第一阶段建议仍使用原生前端，不引入大型框架。原因：

- 当前 UI 已是静态嵌入式资源
- Frontstage MVP 主要是状态映射与表现层
- 先验证信息架构和资产工作流，再决定是否升级为 React/Canvas 游戏式实现

推荐分两期：

- **Phase 1**：DOM/CSS + 轻动画 + 状态区块 + 角色帧图
- **Phase 2**：若交互复杂度上升，再引入 `Phaser` 或等价 2D 场景层

### Frame engine direction

长期建议 Frontstage 增加一个 `frame engine`：

- 输入：run / turn / tool call / review / room message
- 中间层：归一化为 `beats`
- 输出：按顺序播放的 `frames`

这样可以支持：

- 自动播放某次 run 的关键过程
- 一轮一帧的可回看模式
- 工具调用触发的即时小动画
- 实时互动窗口与场景联动

## Out of scope for MVP

- 多角色复杂路径寻路
- 复杂物理运动
- 完整资源编辑器
- 独立权限系统
- 真正的游戏玩法

## Success criteria

MVP 完成后应满足：

- 用户可以一键切换 Frontstage / Backstage
- Frontstage 能看懂当前阶段和系统状态
- Backstage 仍能完整查看 run/task/session/event
- 资产可逐步替换，不绑死某一套第三方素材
- 整套前端适合继续独立开源

下一阶段目标应补充：

- 单次 interaction / turn 能映射为可播放 frame
- 工具调用能触发对应区域动画或覆盖层
- 互动窗口能成为 frame trigger 的正式输入之一
