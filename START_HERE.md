# Start Here

If you are a fresh agent session, read these files in order:

1. `README.md`
2. `AGENTS.md`
3. `docs/2026-03-17-project-context.md`
4. `docs/2026-03-17-requirements.md`
5. `docs/2026-03-17-canx-mvp-design.md`
6. `docs/2026-03-17-canx-mvp-plan.md`

## Repository in one paragraph

`CanX` is a `Go` repository for orchestrating bounded, multi-agent software delivery loops around Codex. It is not a new model runtime and not a business app. It exists to move software development from long human-agent chat loops to AI-to-AI collaboration with supervisor, worker, reviewer, validation, and stop conditions.

## Current priority

Build the local single-machine MVP.

## Current implementation target

Start with:

- `internal/tasks`
- `internal/loop`

Then continue with:

- `internal/workspace`
- `internal/codex`
- `internal/review`
- `internal/runlog`
- `cmd/canxd`

## Rules of engagement

- Do not rebuild generic agent frameworks.
- Reuse Codex surfaces; own only orchestration logic.
- Keep interfaces small and explicit.
- Use TDD for behavior changes.
- Run `make test` and `make build` before claiming success.
