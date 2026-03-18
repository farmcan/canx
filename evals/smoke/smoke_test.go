package smoke_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/farmcan/canx/internal/codex"
	"github.com/farmcan/canx/internal/loop"
	"github.com/farmcan/canx/internal/workspace"
)

func TestSmokeLoopStopsOnRunnerSignal(t *testing.T) {
	t.Parallel()

	repo := makeRepo(t)
	ctx, err := workspace.Load(repo)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	engine := loop.Engine{
		Runner:  codex.NewMockRunner(codex.Result{Output: "implemented feature [canx:stop]"}),
		Workdir: repo,
	}

	outcome, err := engine.Run(context.Background(), loop.Config{
		Goal:     "complete one smoke task",
		MaxTurns: 3,
	}, ctx)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	t.Logf("decision=%s reason=%s turns=%d", outcome.Decision.Action, outcome.Decision.Reason, len(outcome.Turns))
	if outcome.Decision.Action != loop.ActionStop {
		t.Fatalf("decision=%q, want stop", outcome.Decision.Action)
	}
}

func TestSmokeLoopStopsOnValidation(t *testing.T) {
	t.Parallel()

	repo := makeRepo(t)
	ctx, err := workspace.Load(repo)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	engine := loop.Engine{
		Runner:  codex.NewMockRunner(codex.Result{Output: "implemented feature"}),
		Workdir: repo,
	}

	outcome, err := engine.Run(context.Background(), loop.Config{
		Goal:               "pass validation",
		MaxTurns:           2,
		ValidationCommands: []string{"true"},
	}, ctx)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	t.Logf("decision=%s reason=%s turns=%d", outcome.Decision.Action, outcome.Decision.Reason, len(outcome.Turns))
	if outcome.Decision.Reason != "validation passed" {
		t.Fatalf("reason=%q, want validation passed", outcome.Decision.Reason)
	}
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
