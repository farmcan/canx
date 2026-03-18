# CanX Prompt Templates

适用于 `CanX` 当前阶段：会自动读 repo，但仍需要清晰目标；最适合小步快跑的实战迭代。

## 1. Small Feature

```text
Goal:
Add a small practical feature to CanX.

Requirements:
- keep the change minimal
- touch only the directly relevant files
- add or update tests
- stop when validation passes

Constraints:
- do not refactor unrelated packages
- prefer existing patterns in this repo

Validation:
- go test ./...
- go build ./...
```

## 2. Bugfix

```text
Goal:
Fix a specific CanX bug.

Requirements:
- reproduce the issue first
- add or update a regression test
- implement the smallest fix that resolves the bug
- stop when the regression test and full validation pass

Constraints:
- do not bundle unrelated cleanup
- explain the root cause briefly in the final output

Validation:
- go test ./...
- go build ./...
```

## 3. Refactor

```text
Goal:
Refactor a small part of CanX without changing external behavior.

Requirements:
- preserve behavior
- improve clarity or boundaries
- keep the diff focused
- stop when tests still pass

Constraints:
- no speculative abstraction
- no package-wide redesign

Validation:
- go test ./...
- go build ./...
```

## 4. Eval Improvement

```text
Goal:
Improve CanX evaluation or reporting in a practical way.

Requirements:
- keep output machine-readable
- improve measurability or comparability
- add or update tests
- stop when the new eval path works

Constraints:
- keep runtime reasonable
- do not weaken existing eval coverage

Validation:
- go test ./...
- go build ./...
- make report
```

## 5. Self-Hosting Iteration

```text
Goal:
Make a small improvement to CanX that helps CanX iterate on itself more effectively.

Requirements:
- choose one concrete bottleneck
- keep the change minimal and practical
- add tests or eval coverage
- stop when validation passes

Constraints:
- prefer improvements to planner, loop, eval, report, or runtime visibility
- avoid broad framework work

Validation:
- go test ./...
- go build ./...
- make eval
```

## Usage Notes

- 写 prompt 时，优先给：
  - `Goal`
  - `Requirements`
  - `Constraints`
  - `Validation`
- 不要把整个背景重复塞进 prompt；`CanX` 已经会读 `README.md`、`AGENTS.md` 和 `docs/`
- 当前最推荐：
  - 小 feature
  - bugfix
  - eval/report 改进
  - 自举式小迭代
