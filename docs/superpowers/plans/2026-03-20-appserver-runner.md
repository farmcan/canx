# AppServerRunner Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a minimal `AppServerRunner` for CanX that reuses persistent Codex app-server threads per task session while keeping the existing scheduler and runner interface stable.

**Architecture:** Introduce a small JSON-RPC protocol layer, a stdio app-server connection manager, and a `Runner` implementation that maps `SessionKey` to persistent thread IDs. Keep first scope narrow: `approval=never`, final-output aggregation only, no interactive approval handling or UI streaming.

**Tech Stack:** Go, JSON-RPC over stdio, `internal/codex`, `cmd/canxd`, Go tests, fake stdio protocol server, optional real Codex smoke

---

## File Map

### Existing files to modify

- `internal/codex/runner.go`
  - Extend `Request` with optional `SessionKey`.
- `internal/codex/exec_runner.go`
  - Keep compatibility with the new request field.
- `internal/codex/runner_test.go`
  - Add tests for `SessionKey` compatibility and shared runner behavior where relevant.
- `cmd/canxd/main.go`
  - Add `-runner appserver` and instantiate `AppServerRunner`.
- `cmd/canxd/main_test.go`
  - Verify runner-mode selection includes `appserver`.
- `internal/loop/engine.go`
  - Pass `OwnerSessionID` into `codex.Request.SessionKey` when invoking the worker runner.
- `internal/loop/engine_test.go`
  - Verify worker requests include stable `SessionKey` values.
- `docs/runbook.md`
  - Document `-runner appserver` usage and minimal smoke commands.
- `docs/ai-agent-context.md`
  - Update runner status once implementation lands.

### New files to create

- `internal/codex/appserver_protocol.go`
  - JSON-RPC message types and payload structs.
- `internal/codex/appserver_conn.go`
  - stdio process lifecycle, request/response correlation, notification fan-out.
- `internal/codex/appserver_runner.go`
  - `Runner` implementation with `SessionKey -> ThreadID` reuse.
- `internal/codex/appserver_test.go`
  - Focused fake-server tests for protocol flow and thread reuse.

## Chunk 1: Request Surface

### Task 1: Extend codex request with optional session key

**Files:**
- Modify: `internal/codex/runner.go`
- Modify: `internal/codex/runner_test.go`

- [ ] **Step 1: Write failing tests for the new request field**

Cover:
- `Request.Validate()` still only requires prompt
- `SessionKey` does not affect validation
- `ExecRunner` ignores `SessionKey` and still validates/runs normally

- [ ] **Step 2: Run focused codex tests**

Run: `go test ./internal/codex -run 'TestRequest|TestExecRunner' -v`
Expected: FAIL because `SessionKey` is not present yet.

- [ ] **Step 3: Implement minimal request change**

Add:

```go
type Request struct {
    Prompt     string
    Workdir    string
    MaxTurns   int
    SessionKey string
}
```

Do not change `Validate()` semantics.

- [ ] **Step 4: Re-run focused codex tests**

Run: `go test ./internal/codex -run 'TestRequest|TestExecRunner' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/codex/runner.go internal/codex/runner_test.go
git commit -m "feat: add session key to codex requests"
```

## Chunk 2: Protocol Layer

### Task 2: Add protocol structs for app-server JSON-RPC

**Files:**
- Create: `internal/codex/appserver_protocol.go`
- Create: `internal/codex/appserver_test.go`

- [ ] **Step 1: Write failing protocol serialization tests**

Cover:
- `initialize` request encoding
- `thread/start` request encoding
- `turn/start` request encoding
- notification decoding for `thread/started`, `item/completed`, `turn/completed`

- [ ] **Step 2: Run focused protocol tests**

Run: `go test ./internal/codex -run 'TestAppServerProtocol' -v`
Expected: FAIL because protocol types do not exist.

- [ ] **Step 3: Implement minimal protocol types**

Add JSON-RPC and payload structs only for the first-scope methods/events:
- generic request/response envelope
- initialize
- thread/start
- turn/start
- thread/started
- item/completed
- turn/completed

Do not implement unused protocol surface.

- [ ] **Step 4: Re-run protocol tests**

Run: `go test ./internal/codex -run 'TestAppServerProtocol' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/codex/appserver_protocol.go internal/codex/appserver_test.go
git commit -m "feat: add app server protocol types"
```

## Chunk 3: Connection Layer

### Task 3: Build stdio connection manager with request correlation

**Files:**
- Create: `internal/codex/appserver_conn.go`
- Modify: `internal/codex/appserver_test.go`

- [ ] **Step 1: Write failing connection tests with a fake app-server**

Cover:
- starts child process / fake server
- sends `initialize` once
- correlates response by request id
- receives and buffers notifications
- handles malformed server output as an error

- [ ] **Step 2: Run focused connection tests**

Run: `go test ./internal/codex -run 'TestAppServerConn' -v`
Expected: FAIL because connection manager does not exist.

- [ ] **Step 3: Implement minimal connection manager**

Add:
- process spawn wrapper
- single reader goroutine for stdout
- mutex-protected writes
- request-id generation
- pending response map
- notification channel fan-out

Do not implement reconnect or process restart.

- [ ] **Step 4: Re-run focused connection tests**

Run: `go test ./internal/codex -run 'TestAppServerConn' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/codex/appserver_conn.go internal/codex/appserver_test.go
git commit -m "feat: add app server stdio connection manager"
```

## Chunk 4: Runner Reuse

### Task 4: Implement AppServerRunner with SessionKey thread reuse

**Files:**
- Create: `internal/codex/appserver_runner.go`
- Modify: `internal/codex/appserver_test.go`

- [ ] **Step 1: Write failing runner tests**

Cover:
- first `Run()` initializes connection and creates thread
- second `Run()` with same `SessionKey` reuses same thread
- two different `SessionKey` values create different threads
- unsupported approval/interaction event returns a clear error
- final output aggregation returns text in `Result.Output`

- [ ] **Step 2: Run focused runner tests**

Run: `go test ./internal/codex -run 'TestAppServerRunner' -v`
Expected: FAIL because runner does not exist.

- [ ] **Step 3: Implement minimal AppServerRunner**

Behavior:
- lazy-initialize `AppServerConn`
- require non-empty `SessionKey` for stable reuse; if empty, create an ephemeral key internally
- map `SessionKey -> ThreadID`
- create thread on first use
- start turn on each run
- aggregate final text from completed items or turn completion payload
- return explicit error on approval-required style events

- [ ] **Step 4: Re-run focused runner tests**

Run: `go test ./internal/codex -run 'TestAppServerRunner' -v`
Expected: PASS

- [ ] **Step 5: Run broader codex package tests**

Run: `go test ./internal/codex -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/codex/appserver_runner.go internal/codex/appserver_test.go
git commit -m "feat: add minimal app server runner"
```

## Chunk 5: Engine And CLI Wiring

### Task 5: Pass task session identity down into codex requests

**Files:**
- Modify: `internal/loop/engine.go`
- Modify: `internal/loop/engine_test.go`

- [ ] **Step 1: Write failing loop tests**

Cover:
- worker request uses `OwnerSessionID` as `SessionKey`
- same task across multiple turns keeps stable session key
- parallel tasks use different session keys

- [ ] **Step 2: Run focused loop tests**

Run: `go test ./internal/loop -run 'TestEngine.*SessionKey' -v`
Expected: FAIL because engine does not set request session keys.

- [ ] **Step 3: Implement minimal engine wiring**

When invoking `Runner.Run`, pass:

```go
codex.Request{
    Prompt:     prompt,
    Workdir:    e.Workdir,
    MaxTurns:   1,
    SessionKey: outcome.Tasks[index].OwnerSessionID,
}
```

- [ ] **Step 4: Re-run focused loop tests**

Run: `go test ./internal/loop -run 'TestEngine.*SessionKey' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/loop/engine.go internal/loop/engine_test.go
git commit -m "feat: wire task sessions into codex requests"
```

### Task 6: Expose appserver runner mode in CLI

**Files:**
- Modify: `cmd/canxd/main.go`
- Modify: `cmd/canxd/main_test.go`

- [ ] **Step 1: Write failing CLI tests**

Cover:
- `-runner appserver` is accepted
- unknown runner still errors
- appserver mode constructs the correct runner path

- [ ] **Step 2: Run focused CLI tests**

Run: `go test ./cmd/canxd -run 'TestRun|TestParseFlags' -v`
Expected: FAIL because appserver mode is unsupported.

- [ ] **Step 3: Implement minimal CLI runner selection**

Add a constructor path for:
- `codex.NewAppServerRunner(...)`

Keep default runner mode as `exec`.

- [ ] **Step 4: Re-run focused CLI tests**

Run: `go test ./cmd/canxd -run 'TestRun|TestParseFlags' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/canxd/main.go cmd/canxd/main_test.go
git commit -m "feat: add appserver runner mode"
```

## Chunk 6: Smoke And Docs

### Task 7: Add optional real smoke coverage and update docs

**Files:**
- Modify: `internal/codex/appserver_test.go`
- Modify: `docs/runbook.md`
- Modify: `docs/ai-agent-context.md`

- [ ] **Step 1: Add optional real app-server smoke test**

Cover:
- skip if `codex` missing
- skip unless explicit env is set
- run a single `approval=never` turn
- assert non-empty output and stable runtime/thread metadata

- [ ] **Step 2: Run focused codex smoke tests**

Run: `go test ./internal/codex -run 'TestAppServerRunner' -v`
Expected: PASS for fake-server tests; real smoke may skip.

- [ ] **Step 3: Update docs**

Document:
- `-runner appserver`
- first-scope limitation: `approval=never`
- recommended smoke command

- [ ] **Step 4: Run formatting**

Run: `gofmt -w internal/codex/*.go internal/loop/*.go cmd/canxd/*.go`
Expected: files formatted with no output

- [ ] **Step 5: Run focused verification**

Run: `go test ./internal/codex ./internal/loop ./cmd/canxd -v`
Expected: PASS

- [ ] **Step 6: Run full repository verification**

Run: `make build`
Expected: PASS

Run: `make test`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add docs/runbook.md docs/ai-agent-context.md internal/codex/appserver_test.go
git commit -m "docs: describe appserver runner"
```

## Execution Notes

- Keep the first implementation strictly scoped to `approval=never`.
- Do not let app-server protocol details leak into `internal/loop`.
- Prefer fake protocol servers over shell-script fixtures for deterministic tests.
- If real `codex app-server` behavior differs from fake tests, adjust the protocol layer, not the scheduler.
- Preserve `ExecRunner` unchanged so users can still fall back explicitly.

Plan complete and saved to `docs/superpowers/plans/2026-03-20-appserver-runner.md`. Ready to execute?
