package agentic_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/farmcan/canx/internal/codex"
	"github.com/farmcan/canx/internal/loop"
	"github.com/farmcan/canx/internal/tasks"
	"github.com/farmcan/canx/internal/workspace"
)

type evalResult struct {
	Name       string `json:"name"`
	Success    bool   `json:"success"`
	Decision   string `json:"decision"`
	Reason     string `json:"reason"`
	Turns      int    `json:"turns"`
	Tasks      int    `json:"tasks"`
	DoneTasks  int    `json:"done_tasks"`
	DurationMS int64  `json:"duration_ms"`
	PromptDocs int    `json:"prompt_docs"`
	MultiTask  bool   `json:"multi_task"`
}

func TestAgenticQuickSuite(t *testing.T) {
	t.Parallel()

	results := []evalResult{
		runEvalCase(t, "stop_signal", loop.Engine{
			Runner:  codex.NewMockRunner(codex.Result{Output: "done [canx:stop]"}),
			Workdir: makeRepo(t),
		}, loop.Config{Goal: "finish one task", MaxTurns: 2}),
		runEvalCase(t, "validation_feedback", loop.Engine{
			Runner:  codex.NewMockRunner(codex.Result{Output: "first try"}, codex.Result{Output: "fixed [canx:stop]"}),
			Workdir: makeRepo(t),
		}, loop.Config{Goal: "fix failing validation", MaxTurns: 2, ValidationCommands: []string{"echo FAIL && false"}}),
		runEvalCase(t, "multi_task_sequence", loop.Engine{
			Runner:  codex.NewMockRunner(codex.Result{Output: "task 1 [canx:stop]"}, codex.Result{Output: "task 2 [canx:stop]"}),
			Workdir: makeRepo(t),
			Planner: fixedPlanner{tasks: []tasks.Task{
				{ID: "t1", Title: "Task 1", Goal: "do first thing", Status: tasks.StatusPending},
				{ID: "t2", Title: "Task 2", Goal: "do second thing", Status: tasks.StatusPending},
			}},
		}, loop.Config{Goal: "do both things", MaxTurns: 4}),
		runEvalCase(t, "spawn_child_task", loop.Engine{
			Runner: &promptEvalRunner{responses: map[string][]codex.Result{
				"Parent Task": {
					{Output: `need help [canx:spawn:{"title":"Child Task","goal":"write regression test","planned_files":["internal/loop/engine_test.go"]}]`},
					{Output: `parent done [canx:stop]`},
				},
				"Child Task": {
					{Output: `child done [canx:stop]`},
				},
			}},
			Workdir: makeRepo(t),
			Planner: fixedPlanner{tasks: []tasks.Task{
				{ID: "parent", Title: "Parent Task", Goal: "implement parent", Status: tasks.StatusPending, PlannedFiles: []string{"internal/loop/engine.go"}},
			}},
		}, loop.Config{Goal: "ship scheduler", MaxTurns: 3, MaxWorkers: 2, MaxSpawnDepth: 1, MaxChildrenPerTask: 2}),
	}

	for _, result := range results {
		if !result.Success {
			t.Fatalf("eval case failed: %+v", result)
		}
		payload, err := json.Marshal(result)
		if err != nil {
			t.Fatalf("Marshal() error = %v", err)
		}
		t.Log(string(payload))
	}
}

func TestAgenticRealExecSmokeIfEnabled(t *testing.T) {
	if os.Getenv("CANX_EVAL_REAL") != "1" {
		t.Skip("set CANX_EVAL_REAL=1 to run real codex eval")
	}
	if _, err := execLookPath("codex"); err != nil {
		t.Skip("codex binary not found")
	}

	repo := makeRepo(t)
	ctx, err := workspace.Load(repo)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	engine := loop.Engine{
		Runner:  codex.NewExecRunner("codex"),
		Workdir: repo,
	}

	start := time.Now()
	outcome, err := engine.Run(context.Background(), loop.Config{
		Goal:     "Do not modify files. Reply with CANX_REAL_EVAL [canx:stop].",
		MaxTurns: 1,
	}, ctx)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	result := evalResult{
		Name:       "real_exec_smoke",
		Success:    outcome.Decision.Action == loop.ActionStop,
		Decision:   outcome.Decision.Action,
		Reason:     outcome.Decision.Reason,
		Turns:      len(outcome.Turns),
		Tasks:      len(outcome.Tasks),
		DoneTasks:  doneTasks(outcome.Tasks),
		DurationMS: time.Since(start).Milliseconds(),
		PromptDocs: outcome.PromptDocsUsed,
		MultiTask:  len(outcome.Tasks) > 1,
	}
	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	t.Log(string(payload))
	if !result.Success {
		t.Fatalf("real exec eval failed: %+v", result)
	}
}

const plannerRealTimeout = 60 * time.Second

func TestPlannerRealSmokeIfEnabled(t *testing.T) {
	if os.Getenv("CANX_EVAL_REAL") != "1" {
		t.Skip("set CANX_EVAL_REAL=1 to run real codex planner eval")
	}
	if _, err := execLookPath("codex"); err != nil {
		t.Skip("codex binary not found")
	}

	planner := tasks.CodxPlanner{Runner: plannerEvalRunner{}}
	goals := []string{
		"Inspect README and test setup, then propose implementation steps.",
		"Break this repo work into tasks for adding a small CLI flag and tests.",
		"Plan a TDD change for loop behavior and validation handling.",
	}

	multiTaskCount := 0
	for _, goal := range goals {
		ctx, cancel := context.WithTimeout(context.Background(), plannerRealTimeout)
		items, err := planner.Plan(ctx, goal)
		cancel()
		if err != nil {
			t.Fatalf("Plan(%q) error = %v", goal, err)
		}
		if len(items) > 1 {
			multiTaskCount++
		}
		payload, err := json.Marshal(evalResult{
			Name:      "planner_real_smoke",
			Success:   len(items) > 0,
			Tasks:     len(items),
			MultiTask: len(items) > 1,
		})
		if err != nil {
			t.Fatalf("Marshal() error = %v", err)
		}
		t.Log(string(payload))
	}

	rate := float64(multiTaskCount) / float64(len(goals))
	t.Logf("planner_multi_task_rate=%.2f", rate)
	if rate == 0 {
		t.Fatal("expected planner to decompose at least one goal into multiple tasks")
	}
}

func runEvalCase(t *testing.T, name string, engine loop.Engine, cfg loop.Config) evalResult {
	t.Helper()

	repo := engine.Workdir
	ctx, err := workspace.Load(repo)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	start := time.Now()
	outcome, err := engine.Run(context.Background(), cfg, ctx)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	return evalResult{
		Name:       name,
		Success:    outcome.Decision.Action == loop.ActionStop || outcome.Decision.Action == loop.ActionEscalate,
		Decision:   outcome.Decision.Action,
		Reason:     outcome.Decision.Reason,
		Turns:      len(outcome.Turns),
		Tasks:      len(outcome.Tasks),
		DoneTasks:  doneTasks(outcome.Tasks),
		DurationMS: time.Since(start).Milliseconds(),
		PromptDocs: outcome.PromptDocsUsed,
		MultiTask:  len(outcome.Tasks) > 1,
	}
}

func doneTasks(items []tasks.Task) int {
	count := 0
	for _, item := range items {
		if item.Status == tasks.StatusDone {
			count++
		}
	}
	return count
}

type fixedPlanner struct {
	tasks []tasks.Task
}

type plannerEvalRunner struct{}

type promptEvalRunner struct {
	mu        sync.Mutex
	responses map[string][]codex.Result
}

func (plannerEvalRunner) Run(ctx context.Context, prompt string) (string, error) {
	result, err := codex.NewExecRunner("codex").Run(ctx, codex.Request{
		Prompt:   prompt,
		Workdir:  ".",
		MaxTurns: 1,
	})
	if err != nil {
		return "", err
	}
	return result.Output, nil
}

func (r *promptEvalRunner) Run(_ context.Context, req codex.Request) (codex.Result, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	title := evalActiveTaskTitle(req.Prompt)
	queue := r.responses[title]
	if len(queue) == 0 {
		return codex.Result{}, nil
	}
	result := queue[0]
	r.responses[title] = queue[1:]
	return result, nil
}

func (p fixedPlanner) Plan(_ context.Context, _ string) ([]tasks.Task, error) {
	return p.tasks, nil
}

func evalActiveTaskTitle(prompt string) string {
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

func makeRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "repo readme")
	writeFile(t, filepath.Join(dir, "AGENTS.md"), "repo agents")
	if err := os.Mkdir(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatalf("Mkdir(docs) error = %v", err)
	}
	writeFile(t, filepath.Join(dir, "docs", "one.md"), "doc one")
	return dir
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}

var execLookPath = func(file string) (string, error) {
	return exec.LookPath(file)
}
