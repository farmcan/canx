package tasks

import (
	"context"
	"testing"
)

type fakePlannerRunner struct {
	output string
	err    error
}

func (r fakePlannerRunner) Run(_ context.Context, _ string) (string, error) {
	return r.output, r.err
}

func TestCodxPlannerParsesJSONOutput(t *testing.T) {
	t.Parallel()

	runner := fakePlannerRunner{output: `[
		{"id":"task-1","title":"Add test","goal":"add a failing test for X","status":"pending"},
		{"id":"task-2","title":"Implement X","goal":"implement X to pass the test","status":"pending"}
	]`}

	planner := CodxPlanner{Runner: runner}
	items, err := planner.Plan(context.Background(), "implement feature X with TDD")
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if got, want := len(items), 2; got != want {
		t.Fatalf("Plan() len = %d, want %d", got, want)
	}
	if got, want := items[0].ID, "task-1"; got != want {
		t.Fatalf("task 0 id = %q, want %q", got, want)
	}
}

func TestCodxPlannerFallsBackOnInvalidJSON(t *testing.T) {
	t.Parallel()

	runner := fakePlannerRunner{output: "I'll create two tasks: first add a test, then implement"}

	planner := CodxPlanner{Runner: runner}
	items, err := planner.Plan(context.Background(), "implement feature X")
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if got, want := len(items), 1; got != want {
		t.Fatalf("Plan() fallback len = %d, want %d", got, want)
	}
}
