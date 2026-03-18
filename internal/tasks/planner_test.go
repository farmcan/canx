package tasks

import (
	"context"
	"testing"
)

func TestSingleTaskPlannerReturnsSinglePendingTask(t *testing.T) {
	t.Parallel()

	planner := SingleTaskPlanner{}
	tasks, err := planner.Plan(context.Background(), "ship canx mvp")
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if got, want := len(tasks), 1; got != want {
		t.Fatalf("Plan() len = %d, want %d", got, want)
	}

	task := tasks[0]
	if task.Status != StatusPending {
		t.Fatalf("task status = %q, want %q", task.Status, StatusPending)
	}
	if task.Goal == "" {
		t.Fatal("expected task goal")
	}
	if task.ID == "task-1" {
		t.Fatal("expected non-static task id")
	}
}
