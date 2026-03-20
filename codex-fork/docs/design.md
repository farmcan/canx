# Codex Fork Design

## Intent

This is an alternative path to the current CanX direction.

Instead of building a heavier supervisor around Codex sessions, this experiment assumes:
- the parent Codex already has the useful context
- child Codex instances do not need rich peer-to-peer messaging
- file-based handoff is enough for most subtask delegation

## Main Idea

Use a parent session jsonl file as the inheritance source.

The wrapper reads:
- session id
- parent cwd
- a short snapshot of recent assistant-visible outputs

Then it writes a task packet and hands control back to native Codex by printing or running:

```bash
codex fork <session-id> "<task>"
```

For the preferred local UX on this machine, the experiment also supports a Ghostty-first launcher:

```bash
open -na Ghostty.app --args -e /bin/sh -lc <launch.sh>
```

The Ghostty launcher sets a window title containing:
- the parent session id
- the delegated task text

The wrapper also creates two coordination files in the run directory:
- `workspace/`
- `status.json`
- `result.md`

## Why Keep It Isolated

This experiment intentionally avoids CanX mainline modules:
- no `internal/loop`
- no `internal/sessions`
- no dashboard or runlog integration

Reason:
- the hypothesis being tested is whether a much thinner workflow is good enough
- mixing it into the mainline too early would hide that comparison

## Boundaries

What is inherited:
- session id
- cwd
- recent visible text outputs from the parent session file

What is not inherited:
- private model state
- guaranteed full thread fidelity
- structured task ownership
- conflict resolution between child writers

So the correct term is not "true process fork".
It is closer to "session-guided child launch".

## Interactive Confirmation Convention

Because `codex fork` remains interactive, the generated prompt tells the child agent to use the standard choices:
- trust the isolated workspace
- choose the current directory when Codex asks between session directory and current directory
- approve the edits that write `result.md` and `status.json`

## MVP Boundary

The MVP is complete when:
- a shell script can parse a real Codex session file
- it writes a task packet
- it prepares an isolated child workspace
- it creates `status.json` and `result.md`
- it prints a valid `codex fork` launch command
- it can open a new Ghostty window that executes the generated launch script
- optional auto-launch works through one environment variable

Anything beyond that is phase two:
- parent-child index files
- multiple concurrent children
