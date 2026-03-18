package loop

import (
	"context"
	"testing"
	"time"

	"github.com/farmcan/canx/internal/codex"
	"github.com/farmcan/canx/internal/sessions"
	"github.com/farmcan/canx/internal/workspace"
)

func TestEngineStopsWhenValidationPasses(t *testing.T) {
	t.Parallel()

	engine := Engine{
		Runner: &fakeRunner{
			results: []codex.Result{{Output: "implemented change", ExitCode: 0}},
		},
		Workdir: ".",
	}

	outcome, err := engine.Run(context.Background(), Config{
		Goal:               "ship mvp",
		MaxTurns:           3,
		ValidationCommands: []string{"true"},
	}, workspace.Context{Root: ".", Readme: "readme"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got, want := len(outcome.Turns), 1; got != want {
		t.Fatalf("Run() turns = %d, want %d", got, want)
	}
	if got, want := outcome.Decision.Action, ActionStop; got != want {
		t.Fatalf("Run() decision = %q, want %q", got, want)
	}
	if outcome.Session.ID == "" {
		t.Fatal("expected session to be created")
	}
}

func TestEngineContinuesUntilMaxTurnsWhenValidationFails(t *testing.T) {
	t.Parallel()

	engine := Engine{
		Runner: &fakeRunner{
			results: []codex.Result{
				{Output: "first try", ExitCode: 0},
				{Output: "second try", ExitCode: 0},
			},
		},
		Workdir: ".",
	}

	outcome, err := engine.Run(context.Background(), Config{
		Goal:               "ship mvp",
		MaxTurns:           2,
		ValidationCommands: []string{"false"},
	}, workspace.Context{Root: ".", Readme: "readme"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got, want := len(outcome.Turns), 2; got != want {
		t.Fatalf("Run() turns = %d, want %d", got, want)
	}
	if got, want := outcome.Decision.Reason, "max turns reached"; got != want {
		t.Fatalf("Run() reason = %q, want %q", got, want)
	}
}

func TestEngineStopsOnStopMarker(t *testing.T) {
	t.Parallel()

	engine := Engine{
		Runner: &fakeRunner{
			results: []codex.Result{{Output: "done [canx:stop]", ExitCode: 0}},
		},
		Workdir: ".",
	}

	outcome, err := engine.Run(context.Background(), Config{
		Goal:     "ship mvp",
		MaxTurns: 3,
	}, workspace.Context{Root: ".", Readme: "readme"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got, want := outcome.Decision.Action, ActionStop; got != want {
		t.Fatalf("Run() decision = %q, want %q", got, want)
	}
	if got, want := outcome.Decision.Reason, "runner requested stop"; got != want {
		t.Fatalf("Run() reason = %q, want %q", got, want)
	}
}

func TestEngineHonorsTurnTimeout(t *testing.T) {
	t.Parallel()

	engine := Engine{
		Runner:      slowRunner{delay: 100 * time.Millisecond},
		Workdir:     ".",
		TurnTimeout: 10 * time.Millisecond,
	}

	_, err := engine.Run(context.Background(), Config{
		Goal:     "ship mvp",
		MaxTurns: 1,
	}, workspace.Context{Root: ".", Readme: "readme"})
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestEngineWritesTurnSummariesToSession(t *testing.T) {
	t.Parallel()

	registry := sessions.NewRegistry()
	engine := Engine{
		Runner:   &fakeRunner{results: []codex.Result{{Output: "first turn"}, {Output: "second turn [canx:stop]"}}},
		Workdir:  ".",
		Sessions: registry,
	}

	outcome, err := engine.Run(context.Background(), Config{
		Goal:     "ship mvp",
		MaxTurns: 2,
	}, workspace.Context{Root: ".", Readme: "readme"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	session, err := registry.Get(outcome.Session.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got, want := len(session.Turns), 2; got != want {
		t.Fatalf("session turns = %d, want %d", got, want)
	}
}

type fakeRunner struct {
	results []codex.Result
	index   int
}

func (r *fakeRunner) Run(_ context.Context, _ codex.Request) (codex.Result, error) {
	result := r.results[r.index]
	if r.index < len(r.results)-1 {
		r.index++
	}
	return result, nil
}

type slowRunner struct {
	delay time.Duration
}

func (r slowRunner) Run(ctx context.Context, _ codex.Request) (codex.Result, error) {
	select {
	case <-ctx.Done():
		return codex.Result{}, ctx.Err()
	case <-time.After(r.delay):
		return codex.Result{Output: "late"}, nil
	}
}
