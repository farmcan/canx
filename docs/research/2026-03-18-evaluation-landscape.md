# CanX Evaluation Landscape

**Date:** 2026-03-18

## Why evaluation is a first-class concern

`CanX` is an orchestration system, not just a model wrapper. The primary risk is not only "can the model write code", but also:

- does the loop stop cleanly
- does the planner produce useful task decomposition
- does validation feedback reach the next turn
- does the system avoid empty retries
- does the system keep iteration fast enough to matter in practice

For that reason, `CanX` should use both:

- **local harness evaluation** for fast iteration
- **public benchmark alignment** for external comparison

## Recommended external references

### `Terminal-Bench`

- Repository: `https://github.com/harbor-framework/terminal-bench`
- Why it matters: evaluates agents on complicated terminal tasks, which is close to `CanX`'s operating mode.
- Best use for `CanX`: compare command execution, repo navigation, and stop/continue behavior.

### `Multi-SWE-bench`

- Repository: `https://github.com/multi-swe-bench/multi-swe-bench`
- Why it matters: multi-language benchmark family; more relevant than Python-only evaluation for a Go-heavy repo like `CanX`.
- Best use for `CanX`: small-sample comparisons using `flash` or `mini` style subsets before running larger suites.

### `SWE-bench` datasets

- Guide: `https://www.swebench.com/SWE-bench/guides/datasets/`
- Why it matters: still the most recognizable software engineering benchmark family.
- Caution: use `Lite` / `Pro`-style evaluation paths where possible; do not treat legacy `Verified` as the only serious metric.

### OpenAI note on `SWE-bench Verified`

- Article: `https://openai.com/index/why-we-no-longer-evaluate-swe-bench-verified/`
- Why it matters: explains why `Verified` has become less trustworthy as a frontier benchmark.
- Implication for `CanX`: do not optimize solely for a contaminated public benchmark.

### `OpenHands` benchmarks

- Repo area: `https://github.com/All-Hands-AI/OpenHands/tree/main/evaluation/benchmarks`
- Why it matters: useful reference for evaluation harness structure, especially around agent runs and repeatable tasks.

## CanX-specific metric model

Public benchmarks alone are not enough. `CanX` should track its own orchestrator metrics:

- `task_success_rate`
- `done_tasks_per_run`
- `planner_multi_task_rate`
- `avg_turns_to_stop`
- `clean_stop_rate`
- `validation_feedback_used_rate`
- `wall_clock_time_ms`

These are the metrics that tell us whether the orchestrator is improving, not just whether Codex can solve isolated tasks.

## Recommended evaluation ladder

### Layer 1: Local quick harness

Run on every iteration:

- mock stop-signal case
- validation-feedback case
- multi-task sequence case

Target:

- finishes in under one minute
- stable enough for daily iteration

### Layer 2: Real local Codex smoke

Run often, but not every tiny commit:

- one real `codex exec` smoke case
- one real planner smoke case

Target:

- verifies real CLI integration
- catches prompt, encoding, or transport regressions

### Layer 3: Public benchmark samples

Run less frequently:

- `Terminal-Bench` small sample
- `Multi-SWE-bench` mini/flash sample

Target:

- compare `CanX` with external agent systems
- measure relative progress, not just internal regressions

## Practical first step

For the current repository state, the highest-value immediate practice is:

1. keep `go test ./evals/... -v` green
2. keep one real `codex exec` smoke runnable
3. track whether `CodxPlanner` starts producing `2+` tasks more consistently

That is the shortest path to a practical, comparable evaluation loop.
