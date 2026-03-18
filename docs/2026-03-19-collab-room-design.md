# CanX Collaboration Room Design

**Date:** 2026-03-19

## Goal

让多个 agent 和人类可以围绕同一个 `run` / `task` 协作，而不是只有最终 summary。第一阶段先用文件和事件流实现，后续再升级成实时消息。

## Why this exists

当前 `CanX` 已经有：

- run
- task
- session
- event stream

但还缺一个“群”的抽象。实战里需要：

- 人可以说：`把策略改一下`
- reviewer agent 可以说：`这个 task 先 blocked`
- implementer agent 可以说：`我需要更多上下文`
- supervisor 可以把这些输入继续分发回具体 task/session

没有这一层，系统仍然偏单向执行，而不是协作开发。

## External patterns to borrow

### OpenClaw

借用点：

- `agentId` 代表一个独立 brain
- 每个 agent 有自己的 workspace / state / session store
- routing/binding 决定消息归到哪个 agent
- agent-to-agent messaging 默认关闭，必须显式启用

对 `CanX` 的启发：

- 每个 worker/reviewer/supervisor 都应该有明确 `participant_id`
- 群消息不能默认广播到所有 agent；要有明确路由和 allowlist

### ACP

借用点：

- `Run`
- `Session`
- `Message`
- `Await / resume`
- 流式事件

对 `CanX` 的启发：

- “群聊”底层不该是纯文本聊天记录，而应是结构化 message/event
- 人类输入和 agent 输入都应该走统一消息模型
- 系统要支持等待人工输入，再恢复执行

### Codex app-server

借用点：

- `thread/start`
- `turn/start`
- 流式 delta
- `turn/completed`

对 `CanX` 的启发：

- room 不要直接等同 session
- room 更像一个上层协作 thread
- 下面可以挂多个 worker session / turn

## Proposed model

### Room

一个 `room` 对应一次协作空间，可以绑定：

- 一个 `run`
- 一个或多个 `task`
- 多个 `participant`

Room 是“群”的抽象，不直接等同某个 Codex session。

### Participant

参与者统一建模：

- `human`
- `supervisor`
- `implementer`
- `reviewer`
- `observer`

每个 participant 有：

- `participant_id`
- `role`
- `session_id`（可选）
- `agent_id`（可选）

### Message

第一阶段建议的最小消息结构：

- `room_id`
- `message_id`
- `timestamp`
- `participant_id`
- `role`
- `kind`
- `body`
- `task_id`（可选）

`kind` 示例：

- `comment`
- `instruction`
- `proposal`
- `status`
- `question`
- `answer`
- `decision`

## MVP implementation path

### Phase 1: Markdown-backed room

先不用重型消息系统，直接文件化：

- `.canx/rooms/<room-id>/room.md`
- `.canx/rooms/<room-id>/messages.jsonl`

规则：

- `room.md` 给人看，做高层摘要
- `messages.jsonl` 给机器读，做结构化消息流

这样可以支持：

- 人工直接改 `room.md` 或追加一条消息文件
- agent 读取 room 状态并继续执行

### Phase 2: API-backed room

在现有 dashboard API 上增加：

- `GET /api/rooms`
- `GET /api/rooms/:id`
- `GET /api/rooms/:id/messages`
- `POST /api/rooms/:id/messages`

这个阶段人类和 agent 都可以通过统一 API 发消息。

### Phase 3: Live transport

再升级到：

- SSE
- WebSocket

用于实时显示：

- agent thought / delta
- reviewer comments
- human instructions
- task reassignment

## Practical recommendation

当前最合理的落地顺序：

1. 先把 `run/task/session/event` 大盘做稳
2. 再加 file-backed `room/message` 模型
3. 再给 dashboard 增加 room 面板
4. 最后再做实时 transport

不要一开始就直接上复杂 IM 系统。

## Bottom line

`CanX` 的“群”不应被建模成一个聊天窗口，而应被建模成：

- 上层 `room`
- 中层 `message/event`
- 下层 `session/turn`

这样既能让人类参与，也能让多个 agent 真正围绕 task 协作。
