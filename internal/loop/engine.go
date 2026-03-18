package loop

import (
	"context"
	"errors"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/farmcan/canx/internal/codex"
	"github.com/farmcan/canx/internal/review"
	"github.com/farmcan/canx/internal/runlog"
	"github.com/farmcan/canx/internal/sessions"
	"github.com/farmcan/canx/internal/workspace"
)

const stopMarker = "[canx:stop]"

var ErrMissingRunner = errors.New("missing runner")

type Engine struct {
	Runner      codex.Runner
	Workdir     string
	TurnTimeout time.Duration
	Sessions    *sessions.Registry
}

type Outcome struct {
	Session  sessions.Session
	Turns    []Turn
	Decision Decision
	Logs     []runlog.Entry
}

type Turn struct {
	Number           int
	Prompt           string
	RunnerResult     codex.Result
	ValidationPassed bool
	Review           review.Result
}

func (e Engine) Run(ctx context.Context, cfg Config, repo workspace.Context) (Outcome, error) {
	if err := cfg.Validate(); err != nil {
		return Outcome{}, err
	}
	if e.Runner == nil {
		return Outcome{}, ErrMissingRunner
	}
	if e.Sessions == nil {
		e.Sessions = sessions.NewRegistry()
	}

	session, err := e.Sessions.Spawn(sessions.SpawnRequest{
		Label: "main",
		Mode:  sessions.ModePersistent,
		CWD:   e.Workdir,
	})
	if err != nil {
		return Outcome{}, err
	}

	outcome := Outcome{Session: session}
	for turn := 1; turn <= cfg.MaxTurns; turn++ {
		turnCtx := ctx
		cancel := func() {}
		if e.TurnTimeout > 0 {
			turnCtx, cancel = context.WithTimeout(ctx, e.TurnTimeout)
		}

		prompt := buildPrompt(cfg.Goal, repo, outcome.Turns)
		result, err := e.Runner.Run(turnCtx, codex.Request{
			Prompt:   prompt,
			Workdir:  e.Workdir,
			MaxTurns: 1,
		})
		if err != nil {
			cancel()
			return Outcome{}, err
		}

		validationPassed := runValidation(turnCtx, e.Workdir, cfg.ValidationCommands)
		cancel()
		reviewResult := review.Evaluate(review.Result{
			Validated: validationPassed,
			InScope:   true,
		})

		outcome.Turns = append(outcome.Turns, Turn{
			Number:           turn,
			Prompt:           prompt,
			RunnerResult:     result,
			ValidationPassed: validationPassed,
			Review:           reviewResult,
		})
		outcome.Logs = append(outcome.Logs, runlog.Entry{
			Goal:     cfg.Goal,
			Decision: reviewDecision(validationPassed, result.Output, turn, cfg.MaxTurns),
			Summary:  summarizeTurn(turn, result.Output, validationPassed),
		})
		session, err = e.Sessions.Steer(session.ID, summarizeTurn(turn, result.Output, validationPassed))
		if err != nil {
			return Outcome{}, err
		}
		outcome.Session = session

		switch {
		case strings.Contains(result.Output, stopMarker):
			session, _ = e.Sessions.Close(session.ID)
			outcome.Session = session
			outcome.Decision = Decision{Action: ActionStop, Reason: "runner requested stop"}
			return outcome, nil
		case reviewResult.Approved:
			session, _ = e.Sessions.Close(session.ID)
			outcome.Session = session
			outcome.Decision = Decision{Action: ActionStop, Reason: "validation passed"}
			return outcome, nil
		}
	}

	session, _ = e.Sessions.Close(session.ID)
	outcome.Session = session
	outcome.Decision = Decision{Action: ActionEscalate, Reason: "max turns reached"}
	return outcome, nil
}

func buildPrompt(goal string, repo workspace.Context, turns []Turn) string {
	var builder strings.Builder
	builder.WriteString("Goal:\n")
	builder.WriteString(goal)
	builder.WriteString("\n\nRepository context:\n")
	builder.WriteString(repo.Readme)
	if repo.Agents != "" {
		builder.WriteString("\n\nAgent rules:\n")
		builder.WriteString(repo.Agents)
	}
	if len(turns) > 0 {
		last := turns[len(turns)-1]
		builder.WriteString("\n\nPrevious turn summary:\n")
		builder.WriteString(last.RunnerResult.Output)
	}
	builder.WriteString("\n\nRespond with progress, and include [canx:stop] when the task is complete.")
	return builder.String()
}

func runValidation(ctx context.Context, workdir string, commands []string) bool {
	if len(commands) == 0 {
		return false
	}

	for _, command := range commands {
		cmd := exec.CommandContext(ctx, "zsh", "-lc", command)
		if workdir != "" {
			cmd.Dir = workdir
		}
		if err := cmd.Run(); err != nil {
			return false
		}
	}

	return true
}

func reviewDecision(validated bool, output string, turn, maxTurns int) string {
	switch {
	case strings.Contains(output, stopMarker):
		return ActionStop
	case validated:
		return ActionStop
	case turn >= maxTurns:
		return ActionEscalate
	default:
		return ActionContinue
	}
}

func summarizeTurn(turn int, output string, validated bool) string {
	summary := strings.TrimSpace(output)
	if summary == "" {
		summary = "no output"
	}

	status := "validation_failed"
	if validated {
		status = "validation_passed"
	}

	return "turn=" + strconv.Itoa(turn) + " " + status + " output=" + summary
}
