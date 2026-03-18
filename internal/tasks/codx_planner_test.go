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

func TestCodxPlannerParsesFencedJSONOutput(t *testing.T) {
	t.Parallel()

	runner := fakePlannerRunner{output: "Here is the plan:\n```json\n[\n  {\"id\":\"task-a\",\"title\":\"Inspect\",\"goal\":\"inspect the repo\"}\n]\n```"}

	planner := CodxPlanner{Runner: runner}
	items, err := planner.Plan(context.Background(), "inspect repo")
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if got, want := len(items), 1; got != want {
		t.Fatalf("Plan() len = %d, want %d", got, want)
	}
	if got, want := items[0].Status, StatusPending; got != want {
		t.Fatalf("status = %q, want %q", got, want)
	}
}

func TestCodxPlannerNormalizesMissingID(t *testing.T) {
	t.Parallel()

	runner := fakePlannerRunner{output: `[
		{"title":"Inspect","goal":"inspect the repo","status":"pending"},
		{"title":"Test","goal":"run tests","status":"pending"}
	]`}

	planner := CodxPlanner{Runner: runner}
	items, err := planner.Plan(context.Background(), "inspect repo")
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if items[0].ID == "" || items[1].ID == "" {
		t.Fatal("expected generated task ids")
	}
	if items[0].ID == items[1].ID {
		t.Fatal("expected unique generated task ids")
	}
}

func TestParsePlanJSONUsesLastValidJSONArray(t *testing.T) {
	t.Parallel()

	output := `user
Example:
[{"id":"task-1","title":"Example","goal":"example","status":"pending"}]

codex
[{"id":"task-a","title":"Inspect","goal":"inspect repo","status":"pending"},{"id":"task-b","title":"Test","goal":"run tests","status":"pending"}]`

	items, err := parsePlanJSON(output)
	if err != nil {
		t.Fatalf("parsePlanJSON() error = %v", err)
	}
	if got, want := len(items), 2; got != want {
		t.Fatalf("len = %d, want %d", got, want)
	}
	if got, want := items[0].ID, "task-a"; got != want {
		t.Fatalf("first id = %q, want %q", got, want)
	}
}
