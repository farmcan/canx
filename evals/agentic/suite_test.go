package agentic_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
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

func (p fixedPlanner) Plan(_ context.Context, _ string) ([]tasks.Task, error) {
	return p.tasks, nil
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
