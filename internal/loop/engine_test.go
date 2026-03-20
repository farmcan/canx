package loop

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/farmcan/canx/internal/codex"
	"github.com/farmcan/canx/internal/runlog"
	"github.com/farmcan/canx/internal/sessions"
	"github.com/farmcan/canx/internal/tasks"
	"github.com/farmcan/canx/internal/workspace"
)

func TestEngineStopsWhenValidationPasses(t *testing.T) {
	t.Parallel()

	engine := Engine{
		Runner: &fakeRunner{
			results: []codex.Result{{Output: "implemented change", ExitCode: 0}},
		},
		Workdir: ".",
	}

	outcome, err := engine.Run(context.Background(), Config{
		Goal:               "ship mvp",
		MaxTurns:           3,
		ValidationCommands: []string{"true"},
	}, workspace.Context{
		Root:   ".",
		Readme: "readme",
		Docs:   []workspace.Document{{Path: "docs/intent.md", Content: "high signal context"}},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got, want := len(outcome.Turns), 1; got != want {
		t.Fatalf("Run() turns = %d, want %d", got, want)
	}
	if got, want := outcome.Decision.Action, ActionStop; got != want {
		t.Fatalf("Run() decision = %q, want %q", got, want)
	}
	if outcome.Session.ID == "" {
		t.Fatal("expected session to be created")
	}
	if got, want := len(outcome.Tasks), 1; got != want {
		t.Fatalf("tasks len = %d, want %d", got, want)
	}
	if got, want := outcome.Tasks[0].Status, "done"; got != want {
		t.Fatalf("task status = %q, want %q", got, want)
	}
	if outcome.PromptDocsUsed == 0 {
		t.Fatal("expected prompt docs to be used")
	}
}

func TestEngineContinuesUntilMaxTurnsWhenValidationFails(t *testing.T) {
	t.Parallel()

	engine := Engine{
		Runner: &fakeRunner{
			results: []codex.Result{
				{Output: "first try", ExitCode: 0},
				{Output: "second try", ExitCode: 0},
			},
		},
		Workdir: ".",
	}

	outcome, err := engine.Run(context.Background(), Config{
		Goal:               "ship mvp",
		MaxTurns:           2,
		ValidationCommands: []string{"false"},
	}, workspace.Context{Root: ".", Readme: "readme"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got, want := len(outcome.Turns), 2; got != want {
		t.Fatalf("Run() turns = %d, want %d", got, want)
	}
	if got, want := outcome.Decision.Reason, "max turns reached"; got != want {
		t.Fatalf("Run() reason = %q, want %q", got, want)
	}
}

func TestEngineStopsOnStopMarker(t *testing.T) {
	t.Parallel()

	engine := Engine{
		Runner: &fakeRunner{
			results: []codex.Result{{Output: "done [canx:stop]", ExitCode: 0}},
		},
		Workdir: ".",
	}

	outcome, err := engine.Run(context.Background(), Config{
		Goal:     "ship mvp",
		MaxTurns: 3,
	}, workspace.Context{Root: ".", Readme: "readme"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got, want := outcome.Decision.Action, ActionStop; got != want {
		t.Fatalf("Run() decision = %q, want %q", got, want)
	}
	if got, want := outcome.Decision.Reason, "runner requested stop"; got != want {
		t.Fatalf("Run() reason = %q, want %q", got, want)
	}
}

func TestEngineParsesStructuredStopPayloadIntoTask(t *testing.T) {
	t.Parallel()

	engine := Engine{
		Runner: &fakeRunner{
			results: []codex.Result{{Output: `done [canx:stop:{"summary":"implemented healthz","files_changed":["cmd/tradexd/main.go","internal/httpapi/health.go"]}]`}},
		},
		Workdir: ".",
	}

	outcome, err := engine.Run(context.Background(), Config{
		Goal:     "ship mvp",
		MaxTurns: 1,
	}, workspace.Context{Root: ".", Readme: "readme"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got, want := outcome.Tasks[0].Summary, "implemented healthz"; got != want {
		t.Fatalf("task summary = %q, want %q", got, want)
	}
	if got, want := len(outcome.Tasks[0].FilesChanged), 2; got != want {
		t.Fatalf("files_changed len = %d, want %d", got, want)
	}
}

func TestEngineHonorsTurnTimeout(t *testing.T) {
	t.Parallel()

	engine := Engine{
		Runner:      slowRunner{delay: 100 * time.Millisecond},
		Workdir:     ".",
		TurnTimeout: 10 * time.Millisecond,
	}

	_, err := engine.Run(context.Background(), Config{
		Goal:     "ship mvp",
		MaxTurns: 1,
	}, workspace.Context{Root: ".", Readme: "readme"})
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestEngineWritesTurnSummariesToSession(t *testing.T) {
	t.Parallel()

	registry := sessions.NewRegistry()
	engine := Engine{
		Runner:   &fakeRunner{results: []codex.Result{{Output: "first turn"}, {Output: "second turn [canx:stop]"}}},
		Workdir:  ".",
		Sessions: registry,
	}

	outcome, err := engine.Run(context.Background(), Config{
		Goal:     "ship mvp",
		MaxTurns: 2,
	}, workspace.Context{Root: ".", Readme: "readme"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	session, err := registry.Get(outcome.Session.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got, want := len(session.Turns), 2; got != want {
		t.Fatalf("session turns = %d, want %d", got, want)
	}
}

func TestEngineUsesFirstActiveTaskNotJustIndexZero(t *testing.T) {
	t.Parallel()

	engine := Engine{
		Runner:  &fakeRunner{results: []codex.Result{{Output: "done [canx:stop]"}}},
		Workdir: ".",
		Planner: fixedPlanner{tasks: []tasks.Task{{ID: "t1", Goal: "done", Status: tasks.StatusDone}, {ID: "t2", Goal: "active", Status: tasks.StatusPending}}},
	}

	outcome, err := engine.Run(context.Background(), Config{
		Goal:     "ship mvp",
		MaxTurns: 1,
	}, workspace.Context{Root: ".", Readme: "readme", Docs: []workspace.Document{{Path: "docs/x.md", Content: "doc"}}})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got, want := outcome.Tasks[1].Status, tasks.StatusDone; got != want {
		t.Fatalf("second task status = %q, want %q", got, want)
	}
}

func TestEnginePassesValidationOutputToNextTurn(t *testing.T) {
	t.Parallel()

	engine := Engine{
		Runner: &fakeRunner{
			results: []codex.Result{
				{Output: "first try"},
				{Output: "fixed [canx:stop]"},
			},
		},
		Workdir: ".",
	}

	outcome, err := engine.Run(context.Background(), Config{
		Goal:               "fix the test",
		MaxTurns:           2,
		ValidationCommands: []string{"echo TEST_FAILED && false"},
	}, workspace.Context{Root: ".", Readme: "readme"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(outcome.Turns) < 2 {
		t.Fatal("expected at least 2 turns")
	}
	if !strings.Contains(outcome.Turns[1].Prompt, "TEST_FAILED") {
		t.Fatalf("turn 2 prompt missing validation output: %q", outcome.Turns[1].Prompt)
	}
}

func TestBuildPromptKeepsUTF8ValidWhenTruncatingDocs(t *testing.T) {
	t.Parallel()

	doc := strings.Repeat("中", promptDocSnippetLimit+1)
	prompt, _ := buildPrompt(promptRoleWorker, "ship mvp", workspace.Context{
		Root:   ".",
		Readme: "readme",
		Docs:   []workspace.Document{{Path: "docs/utf8.md", Content: doc}},
	}, nil, nil, -1)

	if !utf8.ValidString(prompt) {
		t.Fatal("expected prompt to remain valid utf-8")
	}
}

func TestBuildPromptOmitsDocsForPlannerRole(t *testing.T) {
	t.Parallel()

	prompt, docsUsed := buildPrompt(promptRolePlanner, "inspect repo", workspace.Context{
		Root:   ".",
		Readme: "readme",
		Agents: "agents",
		Docs:   []workspace.Document{{Path: "docs/high-signal.md", Content: "extra docs"}},
	}, nil, nil, -1)

	if strings.Contains(prompt, "Reference docs:") {
		t.Fatalf("planner prompt should omit docs: %q", prompt)
	}
	if docsUsed != 0 {
		t.Fatalf("planner docsUsed = %d, want 0", docsUsed)
	}
	if !strings.Contains(prompt, "Agent rules:") {
		t.Fatalf("planner prompt missing agent rules: %q", prompt)
	}
}

func TestBuildReviewPromptOmitsDocsAndIncludesValidation(t *testing.T) {
	t.Parallel()

	prompt := buildReviewPrompt(
		tasks.Task{ID: "t1", Title: "Task 1", Goal: "add healthz"},
		"worker changed files",
		"make test:\nFAIL",
	)

	if !strings.Contains(prompt, "Review task:") {
		t.Fatalf("review prompt missing task section: %q", prompt)
	}
	if !strings.Contains(prompt, "make test:\nFAIL") {
		t.Fatalf("review prompt missing validation output: %q", prompt)
	}
	if !strings.Contains(prompt, `{"approved":true,"reason":"approved","warnings":[]}`) {
		t.Fatalf("review prompt missing structured schema example: %q", prompt)
	}
	if strings.Contains(prompt, "Reference docs:") {
		t.Fatalf("review prompt should omit docs: %q", prompt)
	}
}

func TestEngineUsesReviewRunnerWhenConfigured(t *testing.T) {
	t.Parallel()

	reviewer := &fakeRunner{results: []codex.Result{{Output: `{"approved":false,"reason":"review says reject","warnings":["missing tests"]}`}}}
	engine := Engine{
		Runner:       &fakeRunner{results: []codex.Result{{Output: "worker output [canx:stop]"}}},
		ReviewRunner: reviewer,
		Workdir:      ".",
	}

	outcome, err := engine.Run(context.Background(), Config{
		Goal:     "ship mvp",
		MaxTurns: 1,
	}, workspace.Context{Root: ".", Readme: "readme", Docs: []workspace.Document{{Path: "docs/x.md", Content: "doc"}}})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if reviewer.lastPrompt == "" {
		t.Fatal("expected reviewer prompt to be captured")
	}
	if !strings.Contains(reviewer.lastPrompt, "worker output") {
		t.Fatalf("reviewer prompt missing worker output: %q", reviewer.lastPrompt)
	}
	if strings.Contains(reviewer.lastPrompt, "Reference docs:") {
		t.Fatalf("reviewer prompt should omit docs: %q", reviewer.lastPrompt)
	}
	if got, want := outcome.Turns[0].Review.Reason, "review says reject"; got != want {
		t.Fatalf("review reason = %q, want %q", got, want)
	}
	if outcome.Turns[0].Review.Approved {
		t.Fatal("expected reviewer verdict to override approval to false")
	}
	if got, want := len(outcome.Turns[0].Review.Warnings), 1; got != want {
		t.Fatalf("warnings len = %d, want %d", got, want)
	}
}

func TestBuildPromptKeepsDocsForWorkerRole(t *testing.T) {
	t.Parallel()

	prompt, docsUsed := buildPrompt(promptRoleWorker, "ship mvp", workspace.Context{
		Root:   ".",
		Readme: "readme",
		Docs:   []workspace.Document{{Path: "docs/high-signal.md", Content: "extra docs"}},
	}, []tasks.Task{{ID: "t1", Title: "Task 1", Goal: "do thing", Status: tasks.StatusPending}}, nil, 0)

	if !strings.Contains(prompt, "Reference docs:") {
		t.Fatalf("worker prompt missing docs: %q", prompt)
	}
	if docsUsed != 1 {
		t.Fatalf("worker docsUsed = %d, want 1", docsUsed)
	}
}

func TestBuildPromptIncludesKnownFailurePatternsForWorker(t *testing.T) {
	t.Parallel()

	prompt, _ := buildPrompt(promptRoleWorker, "ship mvp", workspace.Context{
		Root:     ".",
		Readme:   "readme",
		Patterns: "- make test:\nFAIL",
	}, []tasks.Task{{ID: "t1", Title: "Task 1", Goal: "do thing", Status: tasks.StatusPending}}, nil, 0)

	if !strings.Contains(prompt, "Known failure patterns:") {
		t.Fatalf("worker prompt missing patterns: %q", prompt)
	}
	if !strings.Contains(prompt, "FAIL") {
		t.Fatalf("worker prompt missing pattern content: %q", prompt)
	}
}

func TestEnginePersistsValidationFailurePattern(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	engine := Engine{
		Runner:  &fakeRunner{results: []codex.Result{{Output: "first try"}, {Output: "stop [canx:stop]"}}},
		Workdir: dir,
	}

	_, err := engine.Run(context.Background(), Config{
		Goal:               "fix failing validation",
		MaxTurns:           2,
		ValidationCommands: []string{"echo TEST_FAILED && false"},
	}, workspace.Context{Root: dir, Readme: "readme"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".canx", "patterns.md"))
	if err != nil {
		t.Fatalf("ReadFile(patterns) error = %v", err)
	}
	if !strings.Contains(string(data), "TEST_FAILED") {
		t.Fatalf("patterns file missing validation failure: %q", string(data))
	}
}

func TestSummarizeTurnTruncatesLongOutput(t *testing.T) {
	t.Parallel()

	summary := summarizeTurn(1, strings.Repeat("x", 1200), true)
	if !strings.Contains(summary, "...(truncated)") {
		t.Fatalf("summary missing truncation marker: %q", summary)
	}
}

func TestEngineRunsMultipleTasksInSequence(t *testing.T) {
	t.Parallel()

	engine := Engine{
		Runner: &fakeRunner{results: []codex.Result{
			{Output: "task 1 done [canx:stop]"},
			{Output: "task 2 done [canx:stop]"},
		}},
		Workdir: ".",
		Planner: fixedPlanner{tasks: []tasks.Task{
			{ID: "t1", Title: "Task 1", Goal: "do first thing", Status: tasks.StatusPending},
			{ID: "t2", Title: "Task 2", Goal: "do second thing", Status: tasks.StatusPending},
		}},
	}

	outcome, err := engine.Run(context.Background(), Config{
		Goal:     "do both things",
		MaxTurns: 4,
	}, workspace.Context{Root: ".", Readme: "readme"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	doneCount := 0
	for _, task := range outcome.Tasks {
		if task.Status == tasks.StatusDone {
			doneCount++
		}
	}
	if got, want := doneCount, 2; got != want {
		t.Fatalf("done tasks = %d, want %d", got, want)
	}
	if got, want := len(outcome.Turns), 2; got != want {
		t.Fatalf("turns = %d, want %d", got, want)
	}
}

func TestEngineRunsIndependentTasksInParallel(t *testing.T) {
	t.Parallel()

	runner := &parallelRunner{
		results: []codex.Result{
			{Output: "task 1 done [canx:stop]"},
			{Output: "task 2 done [canx:stop]"},
		},
		release: make(chan struct{}),
	}
	events := []string{}
	var eventsMu sync.Mutex
	engine := Engine{
		Runner:  runner,
		Workdir: ".",
		Planner: fixedPlanner{tasks: []tasks.Task{
			{ID: "t1", Title: "Task 1", Goal: "do first thing", Status: tasks.StatusPending, PlannedFiles: []string{"a.go"}},
			{ID: "t2", Title: "Task 2", Goal: "do second thing", Status: tasks.StatusPending, PlannedFiles: []string{"b.go"}},
		}},
		EventSink: func(event runlog.Event) error {
			eventsMu.Lock()
			defer eventsMu.Unlock()
			events = append(events, event.Kind)
			return nil
		},
	}

	done := make(chan struct{})
	var outcome Outcome
	var err error
	go func() {
		outcome, err = engine.Run(context.Background(), Config{
			Goal:       "do both things",
			MaxTurns:   1,
			MaxWorkers: 2,
		}, workspace.Context{Root: ".", Readme: "readme"})
		close(done)
	}()

	runner.waitForConcurrentCalls(t, 2)
	close(runner.release)
	<-done

	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got, want := len(outcome.Turns), 2; got != want {
		t.Fatalf("turns = %d, want %d", got, want)
	}
	for _, task := range outcome.Tasks {
		if task.Status != tasks.StatusDone {
			t.Fatalf("task %s status = %q, want done", task.ID, task.Status)
		}
		if task.OwnerSessionID == "" {
			t.Fatalf("task %s missing owner session id", task.ID)
		}
	}
	if outcome.Tasks[0].OwnerSessionID == outcome.Tasks[1].OwnerSessionID {
		t.Fatalf("owner session ids should differ, got %q", outcome.Tasks[0].OwnerSessionID)
	}
	turnCompleted := 0
	for _, kind := range events {
		if kind == "turn_completed" {
			turnCompleted++
		}
	}
	if got, want := turnCompleted, 2; got != want {
		t.Fatalf("turn_completed events = %d, want %d", got, want)
	}
}

func TestEngineCreatesChildTaskFromApprovedSpawnRequest(t *testing.T) {
	t.Parallel()

	engine := Engine{
		Runner: &promptTaskRunner{responses: map[string][]codex.Result{
			"Parent Task": {
				{Output: `need help [canx:spawn:{"title":"Child Task","goal":"write regression test","reason":"parallelize test work","planned_files":["internal/loop/engine_test.go"]}]`},
				{Output: `parent done [canx:stop:{"summary":"implemented parent","files_changed":["internal/loop/engine.go"]}]`},
			},
			"Child Task": {
				{Output: `child done [canx:stop:{"summary":"implemented child","files_changed":["internal/loop/engine_test.go"]}]`},
			},
		}},
		Workdir: ".",
		Planner: fixedPlanner{tasks: []tasks.Task{
			{ID: "parent", Title: "Parent Task", Goal: "implement parent logic", Status: tasks.StatusPending, PlannedFiles: []string{"internal/loop/engine.go"}},
		}},
	}

	outcome, err := engine.Run(context.Background(), Config{
		Goal:               "ship scheduler",
		MaxTurns:           3,
		MaxWorkers:         2,
		MaxSpawnDepth:      1,
		MaxChildrenPerTask: 2,
	}, workspace.Context{Root: ".", Readme: "readme"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got, want := len(outcome.Tasks), 2; got != want {
		t.Fatalf("tasks len = %d, want %d", got, want)
	}
	child := outcome.Tasks[1]
	if child.ParentTaskID != "parent" {
		t.Fatalf("child parent task id = %q, want parent", child.ParentTaskID)
	}
	if child.SpawnDepth != 1 {
		t.Fatalf("child spawn depth = %d, want 1", child.SpawnDepth)
	}
	if child.Status != tasks.StatusDone {
		t.Fatalf("child status = %q, want done", child.Status)
	}
	if child.Summary != "implemented child" {
		t.Fatalf("child summary = %q, want implemented child", child.Summary)
	}
}

func TestEnginePassesTaskOwnerSessionIDAsRequestSessionKey(t *testing.T) {
	t.Parallel()

	runner := &capturingRunner{result: codex.Result{Output: "done [canx:stop]"}}
	engine := Engine{
		Runner:  runner,
		Workdir: ".",
	}

	outcome, err := engine.Run(context.Background(), Config{
		Goal:     "ship session binding",
		MaxTurns: 1,
	}, workspace.Context{Root: ".", Readme: "readme"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(runner.requests) != 1 {
		t.Fatalf("requests len = %d, want 1", len(runner.requests))
	}
	if got, want := runner.requests[0].SessionKey, outcome.Tasks[0].OwnerSessionID; got != want {
		t.Fatalf("request session key = %q, want %q", got, want)
	}
}

func TestEngineRejectsSpawnRequestBeyondDepthLimit(t *testing.T) {
	t.Parallel()

	engine := Engine{
		Runner: &promptTaskRunner{responses: map[string][]codex.Result{
			"Parent Task": {
				{Output: `can't spawn more [canx:spawn:{"title":"Too Deep","goal":"do child","planned_files":["child.go"]}] [canx:stop]`},
			},
		}},
		Workdir: ".",
		Planner: fixedPlanner{tasks: []tasks.Task{
			{ID: "parent", Title: "Parent Task", Goal: "implement parent logic", Status: tasks.StatusPending, SpawnDepth: 1, PlannedFiles: []string{"internal/loop/engine.go"}},
		}},
	}

	outcome, err := engine.Run(context.Background(), Config{
		Goal:               "ship scheduler",
		MaxTurns:           2,
		MaxWorkers:         1,
		MaxSpawnDepth:      1,
		MaxChildrenPerTask: 2,
	}, workspace.Context{Root: ".", Readme: "readme"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got, want := len(outcome.Tasks), 1; got != want {
		t.Fatalf("tasks len = %d, want %d", got, want)
	}
}

func TestBuildPromptIncludesCompletedTaskSummaries(t *testing.T) {
	t.Parallel()

	prompt, _ := buildPrompt(promptRoleWorker, "ship mvp", workspace.Context{
		Root:   ".",
		Readme: "readme",
	}, []tasks.Task{
		{ID: "t1", Title: "Task 1", Goal: "first", Status: tasks.StatusDone, Summary: "implemented healthz", FilesChanged: []string{"cmd/tradexd/main.go"}},
		{ID: "t2", Title: "Task 2", Goal: "second", Status: tasks.StatusPending},
	}, nil, 1)

	if !strings.Contains(prompt, "Completed tasks:") {
		t.Fatalf("prompt missing completed tasks section: %q", prompt)
	}
	if !strings.Contains(prompt, "implemented healthz") {
		t.Fatalf("prompt missing completed task summary: %q", prompt)
	}
	if !strings.Contains(prompt, "cmd/tradexd/main.go") {
		t.Fatalf("prompt missing files_changed: %q", prompt)
	}
}

type fixedPlanner struct {
	tasks []tasks.Task
}

func (p fixedPlanner) Plan(_ context.Context, _ string) ([]tasks.Task, error) {
	return p.tasks, nil
}

type fakeRunner struct {
	results    []codex.Result
	index      int
	lastPrompt string
	mu         sync.Mutex
}

func (r *fakeRunner) Run(_ context.Context, req codex.Request) (codex.Result, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastPrompt = req.Prompt
	result := r.results[r.index]
	if r.index < len(r.results)-1 {
		r.index++
	}
	return result, nil
}

type parallelRunner struct {
	results     []codex.Result
	release     chan struct{}
	mu          sync.Mutex
	index       int
	inFlight    int
	maxInFlight int
}

func (r *parallelRunner) Run(_ context.Context, _ codex.Request) (codex.Result, error) {
	r.mu.Lock()
	current := r.index
	if current >= len(r.results) {
		current = len(r.results) - 1
	}
	r.index++
	r.inFlight++
	if r.inFlight > r.maxInFlight {
		r.maxInFlight = r.inFlight
	}
	r.mu.Unlock()

	<-r.release

	r.mu.Lock()
	r.inFlight--
	result := r.results[current]
	r.mu.Unlock()
	return result, nil
}

func (r *parallelRunner) waitForConcurrentCalls(t *testing.T, want int) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		r.mu.Lock()
		maxInFlight := r.maxInFlight
		r.mu.Unlock()
		if maxInFlight >= want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("runner max in-flight calls did not reach %d", want)
}

type promptTaskRunner struct {
	mu        sync.Mutex
	responses map[string][]codex.Result
}

type capturingRunner struct {
	requests []codex.Request
	result   codex.Result
	mu       sync.Mutex
}

func (r *capturingRunner) Run(_ context.Context, req codex.Request) (codex.Result, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.requests = append(r.requests, req)
	return r.result, nil
}

func (r *promptTaskRunner) Run(_ context.Context, req codex.Request) (codex.Result, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	activeTitle := activeTaskTitleFromPrompt(req.Prompt)
	for title, queue := range r.responses {
		if activeTitle != title {
			continue
		}
		if len(queue) == 0 {
			return codex.Result{}, nil
		}
		result := queue[0]
		r.responses[title] = queue[1:]
		return result, nil
	}
	return codex.Result{}, nil
}

func activeTaskTitleFromPrompt(prompt string) string {
	marker := "Active task:\n- ["
	start := strings.Index(prompt, marker)
	if start == -1 {
		return ""
	}
	line := prompt[start+len(marker):]
	statusEnd := strings.Index(line, "] ")
	if statusEnd == -1 {
		return ""
	}
	line = line[statusEnd+2:]
	titleEnd := strings.Index(line, ": ")
	if titleEnd == -1 {
		return ""
	}
	return line[:titleEnd]
}

type slowRunner struct {
	delay time.Duration
}

func (r slowRunner) Run(ctx context.Context, _ codex.Request) (codex.Result, error) {
	select {
	case <-ctx.Done():
		return codex.Result{}, ctx.Err()
	case <-time.After(r.delay):
		return codex.Result{Output: "late"}, nil
	}
}
