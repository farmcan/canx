package runlog

import (
	"testing"
	"time"

	"github.com/farmcan/canx/internal/tasks"
)

func TestEventStoreSavesAndLoadsRunAndEvents(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewEventStore(root)
	record := RunRecord{
		ID:        "run-1",
		Goal:      "ship dashboard",
		RepoRoot:  root,
		Status:    "running",
		TaskCount: 1,
		Tasks:     []tasks.Task{{ID: "task-1", Goal: "do work", Status: tasks.StatusPending}},
		StartedAt: time.Now(),
	}

	if err := store.SaveRun(record); err != nil {
		t.Fatalf("SaveRun() error = %v", err)
	}
	if err := store.AppendEvent("run-1", Event{Kind: "run_started", Message: "started"}); err != nil {
		t.Fatalf("AppendEvent() error = %v", err)
	}
	if err := store.AppendEvent("run-1", Event{Kind: "task_started", TaskID: "task-1"}); err != nil {
		t.Fatalf("AppendEvent() error = %v", err)
	}

	loaded, err := store.LoadRun("run-1")
	if err != nil {
		t.Fatalf("LoadRun() error = %v", err)
	}
	if loaded.Goal != record.Goal {
		t.Fatalf("goal = %q, want %q", loaded.Goal, record.Goal)
	}

	events, err := store.LoadEvents("run-1")
	if err != nil {
		t.Fatalf("LoadEvents() error = %v", err)
	}
	if got, want := len(events), 2; got != want {
		t.Fatalf("events len = %d, want %d", got, want)
	}
}

func TestEventStoreListsRunsNewestFirst(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewEventStore(root)
	first := time.Now().Add(-time.Hour)
	second := time.Now()
	if err := store.SaveRun(RunRecord{ID: "run-1", Goal: "old", RepoRoot: root, Status: "done", StartedAt: first}); err != nil {
		t.Fatalf("SaveRun(run-1) error = %v", err)
	}
	if err := store.SaveRun(RunRecord{ID: "run-2", Goal: "new", RepoRoot: root, Status: "done", StartedAt: second}); err != nil {
		t.Fatalf("SaveRun(run-2) error = %v", err)
	}

	runs, err := store.ListRuns()
	if err != nil {
		t.Fatalf("ListRuns() error = %v", err)
	}
	if got, want := len(runs), 2; got != want {
		t.Fatalf("runs len = %d, want %d", got, want)
	}
	if runs[0].ID != "run-2" {
		t.Fatalf("first run = %q, want run-2", runs[0].ID)
	}
}
