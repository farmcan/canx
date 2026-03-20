# Codex Fork

`codex-fork/` is a first-class thin workflow inside this repository for fast multi-Codex delegation.

The core idea is:
- do not extend CanX main orchestration
- reuse native `codex fork`
- inherit context from an existing Codex session file
- pass work between Codex instances through files

## Scope

This workflow intentionally does a small set of things:
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
- `bin/codex-fork-here`: single-command Ghostty launcher for the current directory
- `lib/codex_fork.sh`: shell helpers
- `test/test.sh`: shell tests
- `docs/design.md`: design notes and boundaries
- `docs/2026-03-20-codex-fork-plan.md`: implementation plan

## Run

```bash
bash codex-fork/test/test.sh
```

```bash
bash codex-fork/bin/codex-fork \
  ~/.codex/sessions/2026/03/13/rollout-xxxx.jsonl \
  "Inspect reviewer path and propose a smaller design"
```

```bash
bash codex-fork/bin/codex-fork latest \
  "Inspect reviewer path and propose a smaller design"
```

```bash
bash codex-fork/bin/codex-fork pick \
  "Inspect reviewer path and propose a smaller design"
```

```bash
bash codex-fork/bin/codex-fork-ghostty \
  ~/.codex/sessions/2026/03/13/rollout-xxxx.jsonl \
  "Inspect reviewer path and propose a smaller design"
```

```bash
bash codex-fork/bin/codex-fork-here \
  "Inspect reviewer path and propose a smaller design"
```

Set `CODEX_FORK_AUTO_LAUNCH=1` to run the generated `codex fork` command immediately.
Set `CODEX_FORK_GHOSTTY_DRY_RUN=1` to print the Ghostty launcher command without opening a new window.
Set `CODEX_FORK_ENABLE_BYPASS=0` to turn off the default zero-interaction launch mode.

## Default Launch Mode

By default, this workflow generates `codex fork` launch commands with:

- `--dangerously-bypass-approvals-and-sandbox`
- `-C <isolated-workspace>`

The intent is simple: make child-session launch as close to zero-interaction as possible.

That means the generated child session will skip Codex confirmation prompts and start directly in the isolated workspace prepared by `codex-fork`.

If you want the normal Codex confirmation flow instead, disable the bypass flag for a run:

```bash
CODEX_FORK_ENABLE_BYPASS=0 bash codex-fork/bin/codex-fork \
  ~/.codex/sessions/2026/03/13/rollout-xxxx.jsonl \
  "Inspect reviewer path and propose a smaller design"
```

## Session Selection

You now have three ways to choose the parent session:

- pass an explicit session jsonl path
- use `latest` to fork the most recent local session file
- use `pick` to choose from the 5 most recent local session files, shown as `session-id | cwd | timestamp`

`latest` and `pick` resolve sessions from:

- `CODEX_FORK_SESSIONS_DIR` if set
- otherwise `$CODEX_HOME/sessions` if `CODEX_HOME` is set
- otherwise `~/.codex/sessions`

## Run Status

After preparing or launching a child run, inspect the handoff state with:

```bash
bash codex-fork/bin/codex-fork status \
  codex-fork/runs/20260320-120000
```

The status command prints:

- current run status from `status.json`
- session id and delegated task
- workspace path
- whether `result.md` is still pending or already written
- the next suggested action

## UX Notes

- `codex-fork-ghostty` extracts the parent session id from the session file automatically.
- `codex-fork-here` auto-detects the most recent session whose `cwd` matches your current directory.
- The Ghostty window title includes the session id and delegated task text.
- Default launch mode is optimized for zero interaction. If you disable bypass mode, Codex may ask for workspace trust, directory choice, and edit confirmations.
- The generated `codex fork` prompt still includes explicit instructions for those confirmations so the delegated child session stays on the handoff path.

## Handoff Files

Each run directory contains:
- `task-packet.md`: delegated task plus inherited parent context snapshot
- `workspace/`: isolated child work area, preferably a git worktree
- `status.json`: parent-visible state for the delegated run
- `result.md`: child Codex result handoff target
- `launch.sh`: exact command used to start the child session
