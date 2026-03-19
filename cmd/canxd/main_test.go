package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestRunRejectsUnknownPlannerMode(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	writeFile(t, tmp+"/README.md", "readme")

	_, err := runWithRunner(loop.Config{
		Goal:     "ship mvp",
		MaxTurns: 1,
	}, Options{
		RepoPath:    tmp,
		PlannerMode: "unknown",
	}, codex.NewMockRunner(codex.Result{
		Output: "[canx:stop] done",
	}))
	if err == nil {
		t.Fatal("expected planner mode error")
	}
}

func TestRunPersistsTurnProgressBeforeCompletion(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	writeFile(t, filepath.Join(tmp, "README.md"), "readme")
	writeFile(t, filepath.Join(tmp, "AGENTS.md"), "agents")

	runner := &stagedRunner{release: make(chan struct{})}
	done := make(chan error, 1)
	go func() {
		_, err := runWithRunner(loop.Config{
			Goal:               "ship mvp",
			MaxTurns:           2,
			ValidationCommands: []string{"false"},
		}, Options{RepoPath: tmp}, runner)
		done <- err
	}()

	runID := waitForRunID(t, tmp)
	waitForCondition(t, 2*time.Second, func() bool {
		events, err := os.ReadFile(filepath.Join(tmp, ".canx", "runs", runID, "events.jsonl"))
		if err != nil {
			return false
		}
		if !strings.Contains(string(events), "\"kind\":\"turn_completed\"") {
			return false
		}

		data, err := os.ReadFile(filepath.Join(tmp, ".canx", "runs", runID, "run.json"))
		if err != nil {
			return false
		}
		var record runlog.RunRecord
		if err := json.Unmarshal(data, &record); err != nil {
			return false
		}
		return record.Status == "running" && record.TurnCount == 1
	})

	close(runner.release)
	if err := <-done; err != nil {
		t.Fatalf("run() error = %v", err)
	}
}

func TestRunPersistsSessionProgressBeforeCompletion(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	writeFile(t, filepath.Join(tmp, "README.md"), "readme")
	writeFile(t, filepath.Join(tmp, "AGENTS.md"), "agents")

	runner := &stagedRunner{release: make(chan struct{})}
	done := make(chan error, 1)
	go func() {
		_, err := runWithRunner(loop.Config{
			Goal:               "ship mvp",
			MaxTurns:           2,
			ValidationCommands: []string{"false"},
		}, Options{RepoPath: tmp}, runner)
		done <- err
	}()

	runID := waitForRunID(t, tmp)
	waitForCondition(t, 2*time.Second, func() bool {
		runData, err := os.ReadFile(filepath.Join(tmp, ".canx", "runs", runID, "run.json"))
		if err != nil {
			return false
		}
		var record runlog.RunRecord
		if err := json.Unmarshal(runData, &record); err != nil {
			return false
		}
		if record.SessionID == "" {
			return false
		}

		sessionData, err := os.ReadFile(filepath.Join(tmp, ".canx", "sessions", record.SessionID+".json"))
		if err != nil {
			return false
		}
		var report runlog.SessionReport
		if err := json.Unmarshal(sessionData, &report); err != nil {
			return false
		}
		return report.RunID == runID && report.TurnCount == 1 && len(report.Session.Turns) == 1 && len(report.Turns) == 1 && report.Turns[0].Number == 1 && !report.Session.Closed
	})

	close(runner.release)
	if err := <-done; err != nil {
		t.Fatalf("run() error = %v", err)
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

type stagedRunner struct {
	calls   int
	release chan struct{}
}

func (r *stagedRunner) Run(_ context.Context, _ codex.Request) (codex.Result, error) {
	r.calls++
	switch r.calls {
	case 1:
		return codex.Result{Output: "first turn"}, nil
	default:
		<-r.release
		return codex.Result{Output: "second turn [canx:stop]"}, nil
	}
}

func waitForRunID(t *testing.T, root string) string {
	t.Helper()
	var runID string
	waitForCondition(t, 2*time.Second, func() bool {
		entries, err := os.ReadDir(filepath.Join(root, ".canx", "runs"))
		if err != nil {
			return false
		}
		for _, entry := range entries {
			if entry.IsDir() {
				runID = entry.Name()
				return true
			}
		}
		return false
	})
	return runID
}

func waitForCondition(t *testing.T, timeout time.Duration, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not met before timeout")
}
