package main

import (
	"os"
	"strings"
	"testing"

	"github.com/farmcan/canx/internal/loop"
)

func TestRunRejectsInvalidConfig(t *testing.T) {
	t.Parallel()

	_, err := run(loop.Config{}, Options{})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestRunProducesSummaryForValidConfig(t *testing.T) {
	t.Parallel()

	output, err := run(loop.Config{
		Goal:     "ship mvp",
		MaxTurns: 2,
	}, Options{})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	if !strings.Contains(output, "ship mvp") {
		t.Fatalf("run() output = %q, want goal included", output)
	}
}

func TestRunIncludesWorkspaceSummaryWhenAvailable(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	writeFile(t, tmp+"/README.md", "readme")
	writeFile(t, tmp+"/AGENTS.md", "agents")

	output, err := run(loop.Config{
		Goal:     "ship mvp",
		MaxTurns: 2,
	}, Options{
		RepoPath: tmp,
	})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	if !strings.Contains(output, "docs=0") {
		t.Fatalf("run() output = %q, want docs summary", output)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}
