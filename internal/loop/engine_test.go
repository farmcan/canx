package loop

import (
	"context"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/farmcan/canx/internal/codex"
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

type fixedPlanner struct {
	tasks []tasks.Task
}

func (p fixedPlanner) Plan(_ context.Context, _ string) ([]tasks.Task, error) {
	return p.tasks, nil
}

type fakeRunner struct {
	results []codex.Result
	index   int
}

func (r *fakeRunner) Run(_ context.Context, _ codex.Request) (codex.Result, error) {
	result := r.results[r.index]
	if r.index < len(r.results)-1 {
		r.index++
	}
	return result, nil
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
