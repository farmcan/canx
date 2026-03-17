# CanX Agent Guide

Scope: this file applies to the entire `canx/` repository.

## Communication

- 默认使用中文与用户沟通；若用户明确要求其他语言，再切换。
- 说明设计判断时，优先引用仓库文档、测试结果、协议文档、外部参考链接。

## Repository intent

- `CanX` is a reusable supervisor/orchestrator for Codex-driven development.
- Keep this repository infrastructure-focused; business logic belongs in downstream repos such as `Tradex`.
- Prefer thin orchestration over rebuilding existing agent frameworks from scratch.
- Treat the primary problem as **AI-to-AI collaboration**, not human-chat optimization.
- Humans should provide goals, approvals, and exception handling; agents should perform the main delivery loop.

## Start-here protocol

- On a fresh session, read these files first:
  1. `START_HERE.md`
  2. `README.md`
  3. `docs/2026-03-17-project-context.md`
  4. `docs/2026-03-17-requirements.md`
  5. `docs/2026-03-17-canx-mvp-design.md`
  6. `docs/2026-03-17-canx-mvp-plan.md`
- `docs/2026-03-17-naming-and-positioning.md` and `docs/research/` are reference material, not mandatory first-pass reads.

## Engineering rules

- Do not reimplement model runtimes that already exist in Codex or existing SDKs.
- Prefer adapters around `codex app-server` or `codex exec` over shell-script sprawl.
- Every loop must have explicit limits: turn count, budget, timeout, and exit criteria.
- Keep modules small and testable.
- Avoid speculative abstractions until at least two concrete call sites exist.

## Multi-agent rules

- Separate roles clearly: supervisor, implementer, reviewer.
- One agent owns one task scope at a time.
- Avoid parallel edits to the same files.
- Encode repeated agent mistakes into docs, tests, or lint rules.
- Keep agent context small; prefer passing curated task packets over sharing long conversation history.

## Validation

- Run `gofmt -w` on changed Go files.
- Prefer focused tests first, then `go test ./...`.
- Run `go build ./...` before claiming the repository is in a good state.
- Prefer `make fmt`, `make test`, and `make build` when available.

## Documentation

- Keep rationale in `docs/`.
- When external tools or projects influence design, add the reference under `docs/research/`.
- When changing architecture direction, update `README.md` and the relevant design doc together.
