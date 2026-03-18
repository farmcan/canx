# CanX Runbook

验证过的命令，复制即可运行。

---

## 前提

```bash
# 确认 codex 可用
codex --version

# 确认构建通过
make build

# 确认所有测试通过
make test
```

---

## 测试套件

更系统的测试分层见：`docs/testing-methods.md`

### 快速单元测试（无需 Codex）

```bash
make test
```

### Eval 套件（mock runner，无需 Codex，毫秒级）

```bash
go test ./evals/agentic/... -v -run TestAgenticQuickSuite
```

预期输出（三个 eval case 全部 pass）：

```
{"name":"stop_signal","success":true,"decision":"stop","turns":1,"tasks":1,"done_tasks":1,...}
{"name":"validation_feedback","success":true,"decision":"stop","turns":2,...}
{"name":"multi_task_sequence","success":true,"decision":"stop","turns":2,"tasks":2,"done_tasks":2,...}
```

### 真实 Codex 集成 smoke（需要 Codex，约 20s）

```bash
CANX_EVAL_REAL=1 go test ./evals/agentic/... -v -run TestAgenticRealExecSmokeIfEnabled -timeout 120s
```

预期：Codex 收到 prompt，回复 `CANX_REAL_EVAL [canx:stop]`，decision=stop。

---

## CLI 运行

### mock runner（不调用 Codex，用于管道验证）

```bash
go run ./cmd/canxd -goal "test mock run" -runner mock -repo . -max-turns 1
```

预期输出：

```
canx decision=stop reason=runner requested stop turns=1 tasks=1 session=session-xxxx workspace=... docs=10
```

### 真实 Codex：只读任务（约 30-60s）

```bash
go run ./cmd/canxd \
  -goal "Read README.md and summarize what CanX does in 2 sentences. Do not modify any files. Reply with your summary then [canx:stop]." \
  -runner exec \
  -repo . \
  -max-turns 1 \
  -turn-timeout 90s
```

### 真实 Codex：带验证的任务（需要 workspace-write 沙箱才能写文件）

```bash
go run ./cmd/canxd \
  -goal "YOUR GOAL HERE" \
  -runner exec \
  -repo . \
  -validate "make test" \
  -max-turns 5 \
  -turn-timeout 120s \
  -budget-seconds 600
```

> **沙箱说明：** Codex 默认以 `read-only` 模式运行，worker 无法修改文件。要允许写入，需要在 `~/.codex/config.toml` 设置 `sandbox = "workspace-write"`，或者在 Codex 配置里开启写权限。只读模式下 worker 会诚实报告无法完成写操作并输出 `[canx:stop]`，CanX 会正确识别这个信号（不会崩溃）。

### 使用 AI 规划器分解任务

```bash
go run ./cmd/canxd \
  -goal "YOUR GOAL HERE" \
  -runner exec \
  -planner codx \
  -repo . \
  -max-turns 5 \
  -turn-timeout 90s
```

> `-planner codx` 先调用 Codex 把 goal 分解成 2-5 个子任务，再逐一执行。每个子任务都会显示在 session report 里。

### 启动本地 dashboard

```bash
go run ./cmd/canxd serve -repo .
```

打开：`http://127.0.0.1:8090`

当前 dashboard 能看到：

- runs 列表
- 单次 run 的最终 task 状态
- 原始 event stream
- runtime 元数据（在 turn event 里）

---

## 查看运行历史

```bash
# 列出所有 session
go run ./cmd/canxd -repo . sessions list

# 查看某次运行的完整 JSON 报告
go run ./cmd/canxd -repo . sessions show <session-id>
```

session-id 是不带 `.json` 后缀的文件名，例如 `session-968503e2847fdad7`。

Run 和 event 文件默认保存在：

```text
.canx/runs/<run-id>/run.json
.canx/runs/<run-id>/events.jsonl
```

---

## 已验证的运行结果（2026-03-19）

| 测试 | 结果 | 耗时 | 备注 |
|---|---|---|---|
| `TestAgenticQuickSuite` | ✅ PASS | 0.02s | mock，3 个 case 全 pass |
| `TestAgenticRealExecSmokeIfEnabled` | ✅ PASS | 21.7s | 真实 Codex，decision=stop |
| CLI mock run | ✅ PASS | 1.6s | decision=stop，docs=10 |
| CLI 真实 Codex 只读任务 | ✅ PASS | 52s | 正确读 README，输出中文摘要，[canx:stop] |
| CLI 真实 Codex 写任务（沙箱限制） | ✅ PASS | 140s | worker 诚实报告无法写入，[canx:stop]，CanX 正确识别 |
| `TestPlannerRealSmokeIfEnabled` | ✅ PASS | 107s | 当前 3-goal smoke 样本得到 `planner_multi_task_rate=1.00` |

---

## 已知限制

### Planner eval 较慢但已可用

```bash
CANX_EVAL_REAL=1 go test ./evals/agentic/... -v -run TestPlannerRealSmokeIfEnabled -timeout 120s
```

当前状态：本机已经通过，但运行时间长，约 `107s`。

主要风险：

- planner 真实评测仍然偏慢
- 当前样本量很小，`planner_multi_task_rate=1.00` 只代表当前 smoke sample

优化方向：

- 给 planner runner 单独设置更短 timeout
- 增加更稳定的小样本 goal 集
- 继续改进 runtime 输出清洗

### 写权限需要配置

Codex 默认 read-only 沙箱，worker 无法修改文件。要让 CanX 真正自托管开发，需要在 Codex 配置里开启 `workspace-write`。

---

## 构建可分发的二进制

```bash
go build -o canxd ./cmd/canxd
./canxd -goal "test" -runner mock -repo .
```

## 实验性只读运行

用于验证 prompt、planner、runtime 信息，不修改仓库文件。

```bash
go run ./cmd/canxd \
  -goal "Inspect this repository's evaluation pipeline and propose one small next improvement. Do not modify any files. Break the work into tasks if useful. Reply with 3 bullets and [canx:stop]." \
  -runner exec \
  -planner codx \
  -repo . \
  -max-turns 2 \
  -turn-timeout 120s \
  -budget-seconds 300
```

实验时重点记录：

- `decision`
- `reason`
- `run`
- `tasks`
- `model / sandbox / approval / runtime_session`
- `session show <session-id>` 的输出

---

## 测试分层说明

每次迭代建议按这个顺序：

### 第一层：单元测试（无需 Codex，毫秒级）

```bash
go test ./internal/...
go test ./cmd/...
go test ./...          # 全部
```

适用：parser、planner 逻辑、loop 行为、session / report 持久化。

### 第二层：Agentic eval（mock runner，毫秒级）

```bash
make eval              # 等同 go test ./evals/... -v
make report            # 生成 evals/reports/latest.md
```

核心指标：`success` / `turns` / `tasks` / `done_tasks` / `duration_ms` / `multi_task`

### 第三层：真实 Codex eval（需要 codex 二进制，约 20-120s）

```bash
make eval-real         # 真实 codex exec smoke
make report-real       # 生成含真实 codex 结果的报告
```

或单独运行：

```bash
CANX_EVAL_REAL=1 go test ./evals/agentic -run TestAgenticRealExecSmokeIfEnabled -v -timeout 120s
CANX_EVAL_REAL=1 go test ./evals/agentic -run TestPlannerRealSmokeIfEnabled -v -timeout 200s
```

### 第四层：隔离实验（git worktree，不污染主工作区）

```bash
git worktree add ../canx-exp main
cd ../canx-exp
# 做实验，记录结果后删除 worktree
git worktree remove ../canx-exp
```
