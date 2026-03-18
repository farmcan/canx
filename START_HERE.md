# Start Here

## 这是什么项目

**CanX** 是一个 Go 编排器，让多个 Codex worker 协作完成软件交付任务。

```
人类提供目标 → Planner 分解任务 → Codex Worker 执行 → validation（go test）→ stop/escalate
```

不是聊天机器人，不是业务应用，不是模型运行时。它的核心价值：

- **确定性 validation gate**（build + test 通过才继续，所有 Python 框架都没有）
- **有界循环**（max-turns + budget-seconds + stop-marker 防止无限循环）
- **仓库上下文注入**（README + AGENTS.md + docs 自动注入每个 prompt）
- **持久化 session report**（`.canx/sessions/` 保存每次运行记录）

---

## 当前状态：MVP 已可用

所有核心模块已实现并通过测试：

```bash
make test    # 11 包全绿
make eval    # 3 个 agentic eval case 全 pass
```

真实 Codex 集成已验证（`CANX_EVAL_REAL=1 make eval-real`）。

---

## 立刻能做的事

### 验证环境

```bash
make build && make test
```

### 跑一次 mock

```bash
go run ./cmd/canxd -goal "test mock run" -runner mock -repo . -max-turns 1
```

### 跑一次真实 Codex（只读，约 30s）

```bash
go run ./cmd/canxd \
  -goal "Read README.md and summarize what CanX does in 2 sentences. Do not modify any files. Reply with your summary then [canx:stop]." \
  -runner exec -repo . -max-turns 1 -turn-timeout 90s
```

---

## 必读文档（按顺序）

| 文档 | 内容 |
|---|---|
| `README.md` | 模块结构、构建方式 |
| `AGENTS.md` | 工程规则（必须遵守） |
| `docs/ai-agent-context.md` | 项目全局：架构图、当前状态、下一步优先级 |
| `docs/runbook.md` | 所有验证过的可运行命令 |

### 可选（背景分析）

| 文档 | 内容 |
|---|---|
| `docs/framework-comparison.md` | 外部框架对比（LangGraph / CrewAI / Codex App Server），CanX 演进方向 |
| `docs/prompt-templates.md` | 写 goal 的推荐模板 |

---

## 下一个最重要的工程任务

**AppServerRunner（P0）**：接入 Codex App Server JSON-RPC，替换当前 `codex exec -` subprocess 模式。每轮不再 fork 新进程，Thread 可跨 turn 复用上下文。

详见 `docs/ai-agent-context.md` → "下一步优先级" 和 `docs/framework-comparison.md` → "4.1 接入 Codex App Server"。

---

## 规则（每次改动前读一遍）

- 改动前写测试，改完跑 `make fmt` + `make build` + `make test`
- 不要重新实现 Codex、模型运行时、Docker 沙箱、通用 agent 框架
- 保持接口小，避免投机性抽象
- 一个 agent 一次只改一个文件范围
