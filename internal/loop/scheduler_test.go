package loop

import (
	"testing"

	"github.com/farmcan/canx/internal/tasks"
)

func TestSchedulerSelectRunnableTasksRespectsConcurrencyAndConflicts(t *testing.T) {
	t.Parallel()

	t.Run("selects up to max workers", func(t *testing.T) {
		t.Parallel()

		items := []tasks.Task{
			{ID: "task-1", Goal: "a", Status: tasks.StatusPending, PlannedFiles: []string{"a.go"}},
			{ID: "task-2", Goal: "b", Status: tasks.StatusPending, PlannedFiles: []string{"b.go"}},
			{ID: "task-3", Goal: "c", Status: tasks.StatusPending, PlannedFiles: []string{"c.go"}},
		}

		indexes := selectRunnableTasks(items, 2)
		if len(indexes) != 2 {
			t.Fatalf("selectRunnableTasks() len = %d, want 2", len(indexes))
		}
		if indexes[0] != 0 || indexes[1] != 1 {
			t.Fatalf("selectRunnableTasks() = %#v, want [0 1]", indexes)
		}
	})

	t.Run("skips conflicting planned files", func(t *testing.T) {
		t.Parallel()

		items := []tasks.Task{
			{ID: "task-1", Goal: "a", Status: tasks.StatusPending, PlannedFiles: []string{"shared.go"}},
			{ID: "task-2", Goal: "b", Status: tasks.StatusPending, PlannedFiles: []string{"shared.go"}},
			{ID: "task-3", Goal: "c", Status: tasks.StatusPending, PlannedFiles: []string{"other.go"}},
		}

		indexes := selectRunnableTasks(items, 3)
		if len(indexes) != 2 {
			t.Fatalf("selectRunnableTasks() len = %d, want 2", len(indexes))
		}
		if indexes[0] != 0 || indexes[1] != 2 {
			t.Fatalf("selectRunnableTasks() = %#v, want [0 2]", indexes)
		}
	})

	t.Run("tasks without planned files run sequentially", func(t *testing.T) {
		t.Parallel()

		items := []tasks.Task{
			{ID: "task-1", Goal: "a", Status: tasks.StatusPending},
			{ID: "task-2", Goal: "b", Status: tasks.StatusPending, PlannedFiles: []string{"other.go"}},
		}

		indexes := selectRunnableTasks(items, 2)
		if len(indexes) != 1 || indexes[0] != 0 {
			t.Fatalf("selectRunnableTasks() = %#v, want [0]", indexes)
		}
	})
}

func TestSchedulerSpawnApprovalBounds(t *testing.T) {
	t.Parallel()

	baseCfg := Config{
		Goal:               "ship scheduler",
		MaxTurns:           2,
		MaxWorkers:         2,
		MaxSpawnDepth:      1,
		MaxChildrenPerTask: 1,
	}

	t.Run("rejects spawn depth overflow", func(t *testing.T) {
		t.Parallel()

		parent := tasks.Task{ID: "task-parent", Goal: "parent", SpawnDepth: 1}
		ok, reason := canApproveSpawn(parent, nil, spawnRequest{
			Title:        "child",
			Goal:         "do child",
			PlannedFiles: []string{"child.go"},
		}, baseCfg)
		if ok {
			t.Fatal("expected spawn to be rejected")
		}
		if reason == "" {
			t.Fatal("expected rejection reason")
		}
	})

	t.Run("rejects child count overflow", func(t *testing.T) {
		t.Parallel()

		parent := tasks.Task{ID: "task-parent", Goal: "parent", SpawnDepth: 0}
		items := []tasks.Task{
			parent,
			{ID: "child-1", Goal: "child", ParentTaskID: "task-parent"},
		}
		ok, reason := canApproveSpawn(parent, items, spawnRequest{
			Title:        "child-2",
			Goal:         "do child",
			PlannedFiles: []string{"child.go"},
		}, baseCfg)
		if ok {
			t.Fatal("expected spawn to be rejected")
		}
		if reason == "" {
			t.Fatal("expected rejection reason")
		}
	})

	t.Run("rejects conflicting planned files with active task", func(t *testing.T) {
		t.Parallel()

		parent := tasks.Task{ID: "task-parent", Goal: "parent", SpawnDepth: 0}
		items := []tasks.Task{
			parent,
			{ID: "task-2", Goal: "other", Status: tasks.StatusInProgress, PlannedFiles: []string{"shared.go"}},
		}
		ok, reason := canApproveSpawn(parent, items, spawnRequest{
			Title:        "child",
			Goal:         "do child",
			PlannedFiles: []string{"shared.go"},
		}, Config{
			Goal:               "ship scheduler",
			MaxTurns:           2,
			MaxWorkers:         3,
			MaxSpawnDepth:      1,
			MaxChildrenPerTask: 2,
		})
		if ok {
			t.Fatal("expected spawn to be rejected")
		}
		if reason == "" {
			t.Fatal("expected rejection reason")
		}
	})

	t.Run("approves spawn within bounds", func(t *testing.T) {
		t.Parallel()

		parent := tasks.Task{ID: "task-parent", Goal: "parent", SpawnDepth: 0}
		items := []tasks.Task{
			parent,
			{ID: "task-2", Goal: "other", Status: tasks.StatusInProgress, PlannedFiles: []string{"other.go"}},
		}
		ok, reason := canApproveSpawn(parent, items, spawnRequest{
			Title:        "child",
			Goal:         "do child",
			PlannedFiles: []string{"child.go"},
		}, Config{
			Goal:               "ship scheduler",
			MaxTurns:           2,
			MaxWorkers:         3,
			MaxSpawnDepth:      1,
			MaxChildrenPerTask: 2,
		})
		if !ok {
			t.Fatalf("expected spawn approval, got reason %q", reason)
		}
	})
}
