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
	"github.com/farmcan/canx/internal/tasks"
	"github.com/farmcan/canx/internal/workspace"
)

const stopMarker = "[canx:stop]"

var ErrMissingRunner = errors.New("missing runner")

type Engine struct {
	Runner      codex.Runner
	Workdir     string
	TurnTimeout time.Duration
	Sessions    *sessions.Registry
	Planner     tasks.Planner
}

type Outcome struct {
	Session        sessions.Session
	Tasks          []tasks.Task
	Turns          []Turn
	Decision       Decision
	Logs           []runlog.Entry
	PromptDocsUsed int
}

type Turn struct {
	Number           int
	Prompt           string
	RunnerResult     codex.Result
	ValidationPassed bool
	ValidationOutput string
	Review           review.Result
}

func (e Engine) Run(ctx context.Context, cfg Config, repo workspace.Context) (Outcome, error) {
	if err := cfg.Validate(); err != nil {
		return Outcome{}, err
	}
	if cfg.BudgetSeconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(cfg.BudgetSeconds)*time.Second)
		defer cancel()
	}
	if e.Runner == nil {
		return Outcome{}, ErrMissingRunner
	}
	if e.Sessions == nil {
		e.Sessions = sessions.NewRegistry()
	}
	if e.Planner == nil {
		e.Planner = tasks.SingleTaskPlanner{}
	}

	session, err := e.Sessions.Spawn(sessions.SpawnRequest{
		Label: "main",
		Mode:  sessions.ModePersistent,
		CWD:   e.Workdir,
	})
	if err != nil {
		return Outcome{}, err
	}

	plannedTasks, err := e.Planner.Plan(ctx, cfg.Goal)
	if err != nil {
		return Outcome{}, err
	}

	outcome := Outcome{
		Session: session,
		Tasks:   plannedTasks,
	}
	for turn := 1; turn <= cfg.MaxTurns; turn++ {
		turnCtx := ctx
		cancel := func() {}
		if e.TurnTimeout > 0 {
			turnCtx, cancel = context.WithTimeout(ctx, e.TurnTimeout)
		}

		prompt, docsUsed := buildPrompt(cfg.Goal, repo, outcome.Tasks, outcome.Turns)
		outcome.PromptDocsUsed = docsUsed
		result, err := e.Runner.Run(turnCtx, codex.Request{
			Prompt:   prompt,
			Workdir:  e.Workdir,
			MaxTurns: 1,
		})
		if err != nil {
			cancel()
			return Outcome{}, err
		}

		validationPassed, validationOutput := runValidation(turnCtx, e.Workdir, cfg.ValidationCommands)
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
			ValidationOutput: validationOutput,
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
		taskDone := reviewResult.Approved || strings.Contains(result.Output, stopMarker)
		outcome.Tasks = updateTaskStatuses(outcome.Tasks, taskDone)

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

const promptDocsBudget = 4000
const promptDocSnippetLimit = 800

func buildPrompt(goal string, repo workspace.Context, plannedTasks []tasks.Task, turns []Turn) (string, int) {
	var builder strings.Builder
	docsUsed := 0
	builder.WriteString("Goal:\n")
	builder.WriteString(goal)
	if len(plannedTasks) > 0 {
		builder.WriteString("\n\nTasks:\n")
		for _, task := range plannedTasks {
			builder.WriteString("- [")
			builder.WriteString(task.Status)
			builder.WriteString("] ")
			builder.WriteString(task.Title)
			builder.WriteString(": ")
			builder.WriteString(task.Goal)
			builder.WriteString("\n")
		}
	}
	builder.WriteString("\n\nRepository context:\n")
	builder.WriteString(repo.Readme)
	if repo.Agents != "" {
		builder.WriteString("\n\nAgent rules:\n")
		builder.WriteString(repo.Agents)
	}
	if len(repo.Docs) > 0 {
		builder.WriteString("\n\nReference docs:\n")
		usedChars := 0
		for _, doc := range repo.Docs {
			if usedChars >= promptDocsBudget {
				break
			}
			content := strings.TrimSpace(doc.Content)
			if content == "" {
				continue
			}
			content = truncateUTF8(content, promptDocSnippetLimit)
			remaining := promptDocsBudget - usedChars
			content = truncateUTF8(content, remaining)
			builder.WriteString("\n")
			builder.WriteString(doc.Path)
			builder.WriteString(":\n")
			builder.WriteString(content)
			builder.WriteString("\n")
			usedChars += len(content)
			docsUsed++
		}
	}
	if len(turns) > 0 {
		last := turns[len(turns)-1]
		builder.WriteString("\n\nPrevious turn summary:\n")
		builder.WriteString(summarizeTurn(last.Number, last.RunnerResult.Output, last.ValidationPassed))
		if last.ValidationOutput != "" {
			builder.WriteString("\n\nValidation errors from last turn:\n")
			builder.WriteString(last.ValidationOutput)
		}
	}
	builder.WriteString("\n\nRespond with progress, and include [canx:stop] when the task is complete.")
	return builder.String(), docsUsed
}

func updateTaskStatuses(items []tasks.Task, done bool) []tasks.Task {
	if len(items) == 0 {
		return items
	}

	next := make([]tasks.Task, len(items))
	copy(next, items)
	for index, item := range next {
		if item.Status != tasks.StatusPending && item.Status != tasks.StatusInProgress {
			continue
		}
		if done {
			next[index].Status = tasks.StatusDone
		} else {
			next[index].Status = tasks.StatusInProgress
		}
		return next
	}
	return next
}

func runValidation(ctx context.Context, workdir string, commands []string) (bool, string) {
	if len(commands) == 0 {
		return false, ""
	}

	for _, command := range commands {
		cmd := exec.CommandContext(ctx, "sh", "-c", command)
		if workdir != "" {
			cmd.Dir = workdir
		}
		output, err := cmd.CombinedOutput()
		if err != nil {
			return false, formatValidationFailure(command, string(output))
		}
	}

	return true, ""
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

func formatValidationFailure(command, output string) string {
	output = strings.TrimSpace(output)
	if len(output) > 500 {
		output = output[:500] + "\n...(truncated)"
	}
	if output == "" {
		output = "(no output)"
	}

	var builder strings.Builder
	builder.WriteString(command)
	builder.WriteString(":\n")
	builder.WriteString(output)
	return builder.String()
}

func truncateUTF8(input string, limit int) string {
	if limit <= 0 || len(input) <= limit {
		if limit <= 0 {
			return ""
		}
		return input
	}

	runes := 0
	for index := range input {
		if index > limit {
			return input[:runes]
		}
		runes = index
	}
	return input[:runes]
}
