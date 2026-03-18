package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/farmcan/canx/internal/codex"
	"github.com/farmcan/canx/internal/loop"
	"github.com/farmcan/canx/internal/runlog"
	"github.com/farmcan/canx/internal/sessions"
)

func TestRunRejectsInvalidConfig(t *testing.T) {
	t.Parallel()

	_, err := runWithRunner(loop.Config{}, Options{}, codex.NewMockRunner())
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestRunProducesSummaryForValidConfig(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	writeFile(t, tmp+"/README.md", "readme")
	writeFile(t, tmp+"/AGENTS.md", "agents")

	output, err := runWithRunner(loop.Config{
		Goal:     "ship mvp",
		MaxTurns: 2,
	}, Options{
		RepoPath: tmp,
	}, codex.NewMockRunner(codex.Result{
		Output: "[canx:stop] done",
	}))
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	if !strings.Contains(output, "decision=stop") {
		t.Fatalf("run() output = %q, want stop decision included", output)
	}
}

func TestRunIncludesWorkspaceSummaryWhenAvailable(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	writeFile(t, tmp+"/README.md", "readme")
	writeFile(t, tmp+"/AGENTS.md", "agents")

	output, err := runWithRunner(loop.Config{
		Goal:     "ship mvp",
		MaxTurns: 2,
	}, Options{
		RepoPath: tmp,
	}, codex.NewMockRunner(codex.Result{
		Output: "[canx:stop] done",
	}))
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	if !strings.Contains(output, "docs=0") {
		t.Fatalf("run() output = %q, want docs summary", output)
	}
}

func TestInspectSessionsListAndShow(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	sessionsDir := filepath.Join(tmp, ".canx", "sessions")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	report := runlog.SessionReport{
		Session: sessions.Session{
			ID:    "session-123",
			Label: "main",
		},
		Decision: "stop",
	}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(sessionsDir, "session-123.json"), data, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	listOutput, err := inspectSessions(Options{RepoPath: tmp}, []string{"list"})
	if err != nil {
		t.Fatalf("inspectSessions(list) error = %v", err)
	}
	if !strings.Contains(listOutput, "session-123.json") {
		t.Fatalf("list output = %q", listOutput)
	}

	showOutput, err := inspectSessions(Options{RepoPath: tmp}, []string{"show", "session-123"})
	if err != nil {
		t.Fatalf("inspectSessions(show) error = %v", err)
	}
	if !strings.Contains(showOutput, "\"session-123\"") {
		t.Fatalf("show output = %q", showOutput)
	}
}

func TestInspectSessionsListReturnsEmptyWhenDirectoryMissing(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	output, err := inspectSessions(Options{RepoPath: tmp}, []string{"list"})
	if err != nil {
		t.Fatalf("inspectSessions(list) error = %v", err)
	}
	if output != "(no sessions)" {
		t.Fatalf("output = %q", output)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}
