# Frontstage / Backstage UI Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a dual-mode CanX UI with a Frontstage demo view and a Backstage engineering console, both powered by the existing CanX run/task/session/event data.

**Architecture:** Keep the current embedded static dashboard as the base. First refactor the existing dashboard into an explicit Backstage mode, then add a Frontstage mode that derives phase and zone presentation from the same read model. Evolve Frontstage toward a generic interaction-to-frame player so each turn, tool call, or key event can become a visual beat. Add an explicit anthropomorphism layer so every important interaction can become a character action, not just a text badge. Use placeholder art first so the UI can ship before final generated assets are ready.

**Tech Stack:** Go standard library HTTP, embedded static HTML/CSS/JS, existing `.canx` persisted files, optional generated PNG/WebP sprite assets.

---

## Chunk 1: Lock the shared presentation model

### Task 1: Define the UI read-model contract

**Files:**
- Modify: `cmd/canxd/server.go`
- Test: `cmd/canxd/server_test.go`
- Reference: `internal/runlog/events.go`
- Reference: `internal/runlog/store.go`
- Reference: `internal/tasks/task.go`

- [ ] Step 1: List the current fields already exposed by `/api/runs`, `/api/sessions`, `/api/rooms`, and `/api/context`
- [ ] Step 2: Identify the minimum derived fields needed for Frontstage: `phase`, `actor_role`, `scene_zone`, `display_status`
- [ ] Step 3: Write failing API tests for one consolidated read model or minimal additional fields on existing payloads
- [ ] Step 4: Implement the derived presentation fields in the server layer without changing persisted file formats
- [ ] Step 5: Run `go test ./cmd/canxd -v`
- [ ] Step 6: Commit

### Task 1B: Add a generic beat model on paper before deepening implementation

**Files:**
- Modify: `docs/superpowers/specs/2026-03-29-frontstage-backstage-ui-design.md`
- Reference: `internal/runlog/events.go`
- Reference: `internal/loop/engine.go`

- [ ] Step 1: Define a beat vocabulary: `briefing`, `tool_use`, `build`, `inspect`, `review`, `handoff`, `incident`, `complete`
- [ ] Step 2: Map existing CanX run/task/event semantics into those beats
- [ ] Step 3: Treat current `phase` as a compressed beat projection, not the long-term canonical model
- [ ] Step 4: Commit

### Task 2: Document the shared mapping rules

**Files:**
- Modify: `docs/ai-agent-context.md`
- Modify: `docs/runbook.md`

- [ ] Step 1: Add the Frontstage / Backstage concept and shared read-model note
- [ ] Step 2: Document the first-pass phase mapping rules
- [ ] Step 3: Run `go test ./...`
- [ ] Step 4: Commit

## Chunk 2: Refactor the existing dashboard into Backstage mode

### Task 3: Add mode-level shell layout

**Files:**
- Modify: `cmd/canxd/ui/index.html`
- Modify: `cmd/canxd/ui/styles.css`
- Modify: `cmd/canxd/ui/app.js`
- Test: `cmd/canxd/ui/live.test.mjs`

- [ ] Step 1: Add a top-level mode toggle shell: `Frontstage` / `Backstage`
- [ ] Step 2: Keep the current dashboard panels under a `Backstage` container
- [ ] Step 3: Preserve current refresh and selection behavior when Backstage is active
- [ ] Step 4: Extend or add a small JS test for mode-switch state if practical
- [ ] Step 5: Run `node --test cmd/canxd/ui/live.test.mjs`
- [ ] Step 6: Commit

### Task 4: Improve Backstage information density without changing scope

**Files:**
- Modify: `cmd/canxd/ui/app.js`
- Modify: `cmd/canxd/ui/styles.css`

- [ ] Step 1: Group existing panels into clear sections: overview, inspect, timeline, collaboration
- [ ] Step 2: Make selected task/session states more obvious
- [ ] Step 3: Keep raw JSON detail panels available for debugging
- [ ] Step 4: Verify the existing dashboard still loads and live refresh remains intact
- [ ] Step 5: Commit

## Chunk 3: Add Frontstage MVP with placeholder art

### Task 5: Build the Frontstage page structure

**Files:**
- Modify: `cmd/canxd/ui/index.html`
- Modify: `cmd/canxd/ui/styles.css`
- Modify: `cmd/canxd/ui/app.js`

- [ ] Step 1: Add a `Frontstage` container with sections for summary, scene, and compact phase/task strip
- [ ] Step 2: Render fixed scene zones: command deck, workbench, test lab, review gate, sync port, incident zone
- [ ] Step 3: Render one main avatar placeholder tied to the selected run or active task
- [ ] Step 4: Map derived `phase` and `scene_zone` fields to visible zone highlighting
- [ ] Step 5: Run manual smoke by opening `go run ./cmd/canxd serve -repo .`
- [ ] Step 6: Commit

### Task 6: Add lightweight motion and state cues

**Files:**
- Modify: `cmd/canxd/ui/styles.css`
- Modify: `cmd/canxd/ui/app.js`

- [ ] Step 1: Add CSS-only motion for active zone pulse, blocked warning, and sync glow
- [ ] Step 2: Add a short display-status bubble near the avatar
- [ ] Step 3: Ensure Frontstage updates from the same SSE flow already used by Backstage
- [ ] Step 4: Verify no polling duplication or obvious flicker
- [ ] Step 5: Commit

### Task 6B: Add a generic frame player path

**Files:**
- Create: `cmd/canxd/ui/frontstage-core.js`
- Create: `cmd/canxd/ui/frontstage.test.mjs`
- Create: `cmd/canxd/ui/frontstage.html`
- Create: `cmd/canxd/ui/frontstage-app.js`

- [ ] Step 1: Define a demo frame sequence independent of CanX-specific task statuses
- [ ] Step 2: Add a `Start Demo` path that plays beat-like frames in order
- [ ] Step 3: Reserve a realtime interaction window in the layout
- [ ] Step 4: Keep the interaction window ready for future room/message and tool-call triggers
- [ ] Step 5: Run `node --test cmd/canxd/ui/frontstage.test.mjs`
- [ ] Step 6: Commit

### Task 6C: Add a self-contained anthropomorphic sample page

**Files:**
- Create: `cmd/canxd/ui/frontstage-sample.html`
- Create: `cmd/canxd/ui/frontstage-sample-app.js`
- Create: `cmd/canxd/ui/frontstage-sample-core.js`
- Create: `cmd/canxd/ui/frontstage-sample.test.mjs`

- [ ] Step 1: Build a no-dependency sample page that always shows visible animation
- [ ] Step 2: Make one AI character move across command / forge / lab / incident / sync zones
- [ ] Step 3: Treat each sample frame as a personified beat, not just a raw status
- [ ] Step 4: Auto-play on load so the page demonstrates itself without user setup
- [ ] Step 5: Link the sample page from the main UI for easy review
- [ ] Step 6: Run `node --test cmd/canxd/ui/frontstage-sample.test.mjs`
- [ ] Step 7: Commit

### Task 6D: Evolve the sample into a multi-agent anthropomorphic stage

**Files:**
- Modify: `cmd/canxd/ui/frontstage-sample.html`
- Modify: `cmd/canxd/ui/frontstage-sample-app.js`
- Modify: `cmd/canxd/ui/frontstage-sample-core.js`
- Test: `cmd/canxd/ui/frontstage-sample.test.mjs`

- [ ] Step 1: Define a visible cast for supervisor, worker, reviewer, and ops roles
- [ ] Step 2: Render those roles as on-stage persona characters instead of a side-only list
- [ ] Step 3: Keep one active role moving while collaborators remain visible at their stations
- [ ] Step 4: Make the active role speak the current intermediate result or action in a speech bubble
- [ ] Step 5: Run `node --test cmd/canxd/ui/frontstage-sample.test.mjs`
- [ ] Step 6: Commit

## Chunk 4: Prepare the asset pipeline for generated art

### Task 7: Add an explicit asset manifest and fallbacks

**Files:**
- Create: `cmd/canxd/ui/assets/README.md`
- Create: `cmd/canxd/ui/assets/frontstage-manifest.json`
- Modify: `cmd/canxd/ui/app.js`

- [ ] Step 1: Define the expected art slots: scene background, zones, avatar states, effect layers
- [ ] Step 2: Add code paths that use CSS placeholders when an asset is missing
- [ ] Step 3: Document recommended file naming and dimensions
- [ ] Step 4: Commit

### Task 8: Write AI asset prompt templates

**Files:**
- Create: `docs/2026-03-29-frontstage-asset-prompts.md`
- Reference: `docs/superpowers/specs/2026-03-29-frontstage-backstage-ui-design.md`

- [ ] Step 1: Write one master character prompt
- [ ] Step 2: Write one background prompt
- [ ] Step 3: Write one prompt per phase: planning, working, validating, reviewing, syncing, blocked
- [ ] Step 4: Add dimensions, style constraints, and transparent-background guidance
- [ ] Step 5: Commit

### Task 8B: Extend prompts for generic beats

**Files:**
- Modify: `docs/2026-03-29-frontstage-asset-prompts.md`
- Modify: `docs/frontstage-gemini-prompts.md`

- [ ] Step 1: Add prompts for `briefing`, `tool_use`, `inspect`, `review`, `handoff`, and `incident`
- [ ] Step 2: Keep prompts reusable outside CanX-specific naming
- [ ] Step 3: Commit

## Chunk 5: Verify and document end to end

### Task 9: Run focused verification

**Files:**
- Modify: `README.md`
- Modify: `docs/runbook.md`

- [ ] Step 1: Document the new dual-mode UI and the expected asset pipeline
- [ ] Step 2: Run `node --test cmd/canxd/ui/live.test.mjs`
- [ ] Step 3: Run `node --test cmd/canxd/ui/frontstage.test.mjs`
- [ ] Step 4: Run `go test ./cmd/canxd -v`
- [ ] Step 5: Run `go build ./...`
- [ ] Step 6: Launch `go run ./cmd/canxd serve -repo .` and manually inspect both modes
- [ ] Step 7: Commit

### Task 10: Ship the MVP before final art

**Files:**
- No new files required beyond previous tasks

- [ ] Step 1: Confirm Frontstage is usable with placeholders only
- [ ] Step 2: Confirm Backstage still exposes raw engineering detail
- [ ] Step 3: Defer Phaser/Rive/game-style upgrades until the placeholder MVP proves useful
- [ ] Step 4: Commit
