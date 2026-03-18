# CanX Testing Methods

`CanX` 当前的测试分三层，按从快到慢排序：

## 1. Unit / package tests

目标：验证局部行为是否正确。

常用命令：

```bash
go test ./internal/...
go test ./cmd/...
go test ./...
```

适用场景：

- parser
- planner
- loop 行为
- session / report 持久化

## 2. Local agentic eval

目标：验证 orchestration 行为，而不是只测单个函数。

常用命令：

```bash
make eval
go test ./evals/agentic -v
make report
```

核心指标：

- `success`
- `turns`
- `tasks`
- `done_tasks`
- `duration_ms`
- `multi_task`

## 3. Real Codex eval / experiment

目标：验证真实 `codex exec` 集成、planner 分解能力、runtime 表现。

常用命令：

```bash
make eval-real
make report-real
```

或单独运行：

```bash
CANX_EVAL_REAL=1 go test ./evals/agentic -run TestAgenticRealExecSmokeIfEnabled -v
CANX_EVAL_REAL=1 go test ./evals/agentic -run TestPlannerRealSmokeIfEnabled -v
```

## 4. Isolated experiment

目标：做实验性 prompt / loop / runtime 验证，不污染主工作区。

推荐方式：

- 使用独立 git worktree
- 只跑只读任务或明确边界任务
- 记录 runtime 和 session 结果

示例：

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

记录内容：

- CLI 输出
- `session` JSON
- runtime metadata
- wall time / memory

## Recommended order

每次迭代建议按这个顺序：

1. focused `go test`
2. `go test ./...`
3. `make report`
4. 必要时 `make eval-real` 或实验性 worktree 运行
