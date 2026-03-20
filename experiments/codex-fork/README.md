# Codex Fork Experiment

This directory is an isolated experiment for a thinner multi-Codex workflow.

The core idea is:
- do not extend CanX main orchestration
- reuse native `codex fork`
- inherit context from an existing Codex session file
- pass work between Codex instances through files

## Scope

This experiment only does three things:
- parse a parent session jsonl file
- write a task packet with a small inherited context snapshot
- prepare an isolated child workspace
- create `status.json` and `result.md` handoff files
- print or launch a native `codex fork` command

It does not do:
- central session management
- task graphs
- event streaming
- review policy
- merge gates

## Layout

- `bin/codex-fork`: thin CLI wrapper
- `bin/codex-fork-ghostty`: Ghostty-first launcher
- `lib/codex_fork.sh`: shell helpers
- `test/test.sh`: shell tests
- `docs/design.md`: design notes and boundaries
- `docs/2026-03-20-codex-fork-plan.md`: implementation plan

## Run

```bash
bash experiments/codex-fork/test/test.sh
```

```bash
bash experiments/codex-fork/bin/codex-fork \
  ~/.codex/sessions/2026/03/13/rollout-xxxx.jsonl \
  "Inspect reviewer path and propose a smaller design"
```

```bash
bash experiments/codex-fork/bin/codex-fork-ghostty \
  ~/.codex/sessions/2026/03/13/rollout-xxxx.jsonl \
  "Inspect reviewer path and propose a smaller design"
```

Set `CODEX_FORK_AUTO_LAUNCH=1` to run the generated `codex fork` command immediately.
Set `CODEX_FORK_GHOSTTY_DRY_RUN=1` to print the Ghostty launcher command without opening a new window.

## UX Notes

- `codex-fork-ghostty` extracts the parent session id from the session file automatically.
- The Ghostty window title includes the session id and delegated task text.
- The generated `codex fork` prompt includes explicit instructions for the common interactive confirmations:
  - trust the isolated workspace
  - choose the current directory instead of the original session directory
  - approve the handoff-file edits

## Handoff Files

Each run directory contains:
- `task-packet.md`: delegated task plus inherited parent context snapshot
- `workspace/`: isolated child work area, preferably a git worktree
- `status.json`: parent-visible state for the delegated run
- `result.md`: child Codex result handoff target
- `launch.sh`: exact command used to start the child session
