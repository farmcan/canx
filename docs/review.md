# CanX MVP Code Review

**第一版：** 2026-03-18（初始骨架）
**第二版：** 2026-03-18（Planner + 持久化）
**第三版：** 2026-03-18（多任务 + eval suite + ExecRunner）
**本版（第四版）：** 2026-03-19（eval report + parsePlanJSON 改进 + escalate marker）
**范围：** 全仓库，HEAD `e4df041`
**测试状态：** `go test ./...` 全部通过（11 包）

---

## 第三轮修复进度

| 问题 | 状态 |
|---|---|
| `summarizeTurn` output 无截断 | ✅ 已修复：1000 字符上限，加 `...(truncated)` |
| `[canx:escalate]` marker 未实现 | ✅ 已实现：Engine switch 里处理，提前退出 |
| 非零退出码 + stop marker 导致崩溃 | ✅ 已修复：先检查 marker 再决定是否返回 error |
| `parsePlanJSON` 只找第一个 `[...]` | ✅ 已改进：`allIndexes` 穷举所有组合，取最后有效的 JSON 数组 |
| `review.InScope` 硬编码 `true` | ❌ **第五次出现，未修** |
| `go.mod go 1.25.0` | ❌ **第五次出现，未修** |
| `sessions.Registry` 无并发保护 | ❌ 持续存在 |

---

## 更正上一版

**第三版 review 中对 `truncateUTF8` 的 Bug 判断是错误的。**

我之前声称「函数会丢弃最后一个 rune」，经过本轮仔细验证，该函数逻辑是正确的：

- `runes = index` 记录的是「当前 rune 的字节起始位置」
- 当 `index > limit` 触发提前返回时，`input[:runes]` 正确截断到前一个 rune 的结束
- 当循环正常结束（所有 rune 起始都 ≤ limit）时，`input[:runes]` 丢弃最后一个 rune——这是正确行为，因为该 rune 从 `runes` 处开始但其结束超过了 limit

现有 `TestBuildPromptKeepsUTF8ValidWhenTruncatingDocs` 验证了 UTF-8 有效性，函数实现是正确的。

---

## 当前问题

### 重要

#### 1. `review.InScope` 硬编码 `true`（第五轮仍未修）

```go
reviewResult := review.Evaluate(review.Result{
    Validated: validationPassed,
    InScope:   true,   // 永远 true
})
```

`review.Result` 有 4 个字段，其中 `InScope` 和 `Approved` 都应该是计算结果，但 `InScope` 被静态传入，导致 `Evaluate` 中的 scope 分支永远不会触发。这个字段已经在五轮 review 里出现。它要么应该被删掉，要么应该有实际计算逻辑。

---

#### 2. `CodxPlanner.Plan` 在 runner 报错时静默 fallback

```go
output, err := p.Runner.Run(ctx, plannerPrompt+goal)
if err != nil {
    return nil, err  // ← 这是正确的，会传播 error
}
items, err := parsePlanJSON(output)
if err != nil || len(items) == 0 {
    return SingleTaskPlanner{}.Plan(ctx, goal)  // ← 这里静默 fallback
}
```

前半段（runner 报错）会传播错误，是正确的。但后半段（JSON 解析失败）静默退化到单任务规划器，没有任何日志或信号。如果 Codex 返回了非 JSON 格式的长文本（比如 Codex 输出了代码而不是 JSON），调用方无法区分「规划成功」和「规划失败后降级」。在多任务场景下，这意味着原本应该分解成 5 个任务的工作可能被当作 1 个任务执行，且没有任何警告。

---

#### 3. Escalate 时 active task 被留在 `in_progress` 状态

```go
taskDone := reviewResult.Approved || strings.Contains(result.Output, stopMarker)
outcome.Tasks = updateTaskStatuses(outcome.Tasks, activeIndex, taskDone)

switch {
case strings.Contains(result.Output, escalateMarker):
    // 直接返回，但 taskDone=false → active task 已被标为 in_progress
    session, _ = e.Sessions.Close(session.ID)
    outcome.Decision = Decision{Action: ActionEscalate, ...}
    return outcome, nil
```

worker 输出 `[canx:escalate]` 时，`taskDone = false`（output 不含 stopMarker，validation 也没 pass），所以 `updateTaskStatuses` 把该任务标为 `in_progress`。然后立刻 return。结果：session report 里 active task 的 status 是 `in_progress`，而不是 `blocked` 或 `escalated`。消费 session report 的工具无法区分「任务在进行中」和「任务因人工介入需求而中止」。

---

#### 4. `TestPlannerRealSmokeIfEnabled` 导致 `make report-real` 失败

实测（上次运行记录）：该测试在第二个 goal 时会让 Codex 进入真实代码执行模式而挂起，最终 timeout。`cmd/canx-eval-report` 在 `report-real` 模式下会运行 `TestPlannerRealSmokeIfEnabled`，导致整个 `make report-real` 命令失败。

直接影响：`make report-real` 目前不可用。

根本原因：`plannerEvalRunner` 没有给 Codex 设置超时，且 planner prompt 对某些 goal 无法阻止 Codex 开始真实执行。

---

### 中等

#### 5. `shouldSkipGitRepoCheck` 每次 `Run` 都调用

```go
func (r ExecRunner) Run(ctx context.Context, req Request) (Result, error) {
    args := []string{"exec", "-"}
    if shouldSkipGitRepoCheck(req.Workdir) {  // ← 每次 Run 都跑一次 git
        args = append(args, "--skip-git-repo-check")
    }
```

`req.Workdir` 在同一个 Engine.Run 内不变，但每轮 turn 都会重新执行 `git rev-parse --is-inside-work-tree`。5 轮 = 5 次额外 git 子进程。影响不大，但属于不必要的重复。

---

#### 6. `Request.MaxTurns` 是完全死掉的字段

`codex.Request.MaxTurns` 存在，Engine 传 `MaxTurns: 1`，但 `ExecRunner.Run` 构建的命令行里从未使用这个字段。以后如果接入 `AppServerRunner` 这个字段有意义，但目前对 ExecRunner 完全无用，且没有注释说明。

---

#### 7. `evals/reports/` 生成文件应该被 gitignore

`evals/reports/latest.json`、`latest.jsonl`、`latest.md` 是 `make report` 的输出，每次运行都会变化。这类文件通常不应该进入版本控制（类似 `*.o`、`dist/`）。当前 `.gitignore` 只有 `/canxd` 和 `/.canx/`，没有排除 `evals/reports/`。

---

#### 8. `parsePlanJSON` 可能接受非 Task 的有效 JSON 数组

```go
var items []Task
if err := json.Unmarshal([]byte(output[start:end+1]), &items); err == nil {
    return items, nil
}
```

如果输出里包含 `[1, 2, 3]`（valid JSON array of ints），`json.Unmarshal` 到 `[]Task` 会成功，返回包含零值字段的 Task 列表（ID、Title、Goal 全空）。后续的 Normalize + ID 生成会补全 ID，但 Goal 为空的任务仍然会进入 Engine，最终在 prompt 里产生空目标的任务行。

实际发生概率低（Codex 不太可能输出纯整数数组），但技术上是个漏洞。

---

### 小问题（持续）

- **`go.mod go 1.25.0`**：五轮未修，1 分钟可改，该修了。
- **`sessions.Registry` 无并发保护**：Engine 目前单 goroutine，短期无害。但应在注释里明确"非并发安全"。
- **`ModeOneshot` 从未使用**：Engine 始终创建 `ModePersistent`，可以考虑删掉或加注释。
- **`evalreport.ParseGoTestJSON` 不处理 scanner.Bytes() 的错误**：`scanner.Err()` 只在循环后检查，循环内的 `json.Unmarshal` 错误被 `continue` 忽略，是正常设计（容忍部分行解析失败）。

---

## 新增内容评价

### `parsePlanJSON` 改进（`allIndexes` 穷举）

从「取第一个 `[` 到最后一个 `]`」改为「穷举所有组合，取最后一个有效的 JSON 数组」，解决了 Codex 在输出里先输出 planner example 再输出真实结果的场景。有对应测试 `TestParsePlanJSONUsesLastValidJSONArray`。改动合理，测试充分。

时间复杂度：O(n²) on the number of `[` and `]` characters，对实际 Codex 输出（几 KB）完全可接受。

### eval report 工具（`internal/evalreport` + `cmd/canx-eval-report`）

架构清晰：`ParseGoTestJSON` 解析 `go test -json` 输出，`RenderMarkdown` 生成报告。分离了解析和渲染，有单元测试，是正确的设计。

一个小观察：`RenderMarkdown` 在 `results` 为空时仍生成包含表头但无数据行的 Markdown 表格，以及包含空数组的 mermaid 图。这在空报告场景下渲染正常，不是 bug。

### escalate marker 实现

`[canx:escalate]` 被正确识别并在 switch 里优先处理（先于 stop marker）。这个优先级顺序是对的：worker 显式请求升级的意图应该比 stop 优先。实测验证了这个路径（Codex 在只读沙箱里诚实报告无法写入并发出 stop）。

---

## 整体评价

代码已经处于可以真实运行的状态，核心路径经过端到端验证。四轮下来累计修复了约 20 个问题。剩余最重要的事情是：

1. 删掉或实现 `review.InScope`（五轮未修，技术债）
2. 修 `make report-real` 的超时问题（功能破损）
3. `CodxPlanner` fallback 加日志或让上层感知降级
4. Escalate 时把 active task 标为 `blocked` 而不是 `in_progress`
