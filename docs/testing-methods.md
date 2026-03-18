# CanX Testing Methods

按从快到慢的顺序执行：

## 1. Focused package tests

```bash
go test ./internal/runlog ./cmd/canxd -v
go test ./internal/loop -v
```

适用：

- event store
- dashboard API
- loop 行为

## 2. Full repository validation

```bash
go test ./...
go build ./...
```

适用：

- 提交前总体验证
- 防止 dashboard / runlog 改动影响既有 eval

## 3. Report generation

```bash
make report
```

适用：

- 检查本地 eval 报告仍可生成

## 4. Manual dashboard smoke

```bash
go run ./cmd/canxd -goal "test dashboard" -runner mock -repo . -max-turns 1
go run ./cmd/canxd serve -repo .
```

手动检查：

- `/api/runs`
- `/api/runs/<run-id>`
- `/api/runs/<run-id>/events`
- `http://127.0.0.1:8090`

## 5. Real Codex experiment

```bash
go run ./cmd/canxd \
  -goal "Inspect this repository and report one bottleneck. Do not modify files. Reply with 3 bullets and [canx:stop]." \
  -runner exec \
  -planner codx \
  -repo . \
  -max-turns 8 \
  -turn-timeout 120s \
  -budget-seconds 900
```

记录：

- `run`
- `decision`
- `reason`
- `tasks`
- `model / sandbox / approval / runtime_session`
- `events.jsonl`
