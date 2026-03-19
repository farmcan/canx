package loop

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
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
const escalateMarker = "[canx:escalate]"

const (
	promptRolePlanner  = "planner"
	promptRoleWorker   = "worker"
	promptRoleReviewer = "reviewer"
)

var ErrMissingRunner = errors.New("missing runner")

type Engine struct {
	Runner       codex.Runner
	ReviewRunner codex.Runner
	Workdir      string
	TurnTimeout  time.Duration
	Sessions     *sessions.Registry
	Planner      tasks.Planner
	EventSink    func(runlog.Event) error
	SessionSink  func(runlog.SessionReport) error
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

type stopPayload struct {
	Summary      string   `json:"summary"`
	FilesChanged []string `json:"files_changed"`
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
	if err := e.emitEvent(runlog.Event{
		Kind:      "session_started",
		SessionID: session.ID,
		Timestamp: session.CreatedAt,
		Runtime: map[string]any{
			"mode": session.Mode,
			"cwd":  session.CWD,
		},
	}); err != nil {
		return Outcome{}, err
	}
	if err := e.emitEvent(runlog.Event{
		Kind:      "task_state",
		SessionID: session.ID,
		Tasks:     cloneTasks(outcome.Tasks),
	}); err != nil {
		return Outcome{}, err
	}
	if err := e.emitSession(runlog.SessionReport{
		Session:   outcome.Session,
		Runtime:   codex.Runtime{},
		Decision:  "running",
		Reason:    "session started",
		TurnCount: len(outcome.Turns),
		Tasks:     cloneTasks(outcome.Tasks),
	}); err != nil {
		return Outcome{}, err
	}
	for turn := 1; turn <= cfg.MaxTurns; turn++ {
		activeIndex := firstActiveTaskIndex(outcome.Tasks)
		if activeIndex == -1 {
			session, _ = e.Sessions.Close(session.ID)
			outcome.Session = session
			outcome.Decision = Decision{Action: ActionStop, Reason: "all tasks complete"}
			return outcome, nil
		}

		turnCtx := ctx
		cancel := func() {}
		if e.TurnTimeout > 0 {
			turnCtx, cancel = context.WithTimeout(ctx, e.TurnTimeout)
		}

		prompt, docsUsed := buildPrompt(promptRoleWorker, cfg.Goal, repo, outcome.Tasks, outcome.Turns, activeIndex)
		if outcome.PromptDocsUsed == 0 {
			outcome.PromptDocsUsed = docsUsed
		}
		result, err := e.Runner.Run(turnCtx, codex.Request{
			Prompt:   prompt,
			Workdir:  e.Workdir,
			MaxTurns: 1,
		})
		if err != nil {
			// If the runner failed but the output contains a stop or escalate
			// marker, treat it as a partial success: the worker completed its
			// reasoning but the process exited non-zero (e.g. read-only sandbox
			// blocking writes).  Surface the output instead of dropping it.
			if !strings.Contains(result.Output, stopMarker) && !strings.Contains(result.Output, escalateMarker) {
				cancel()
				return Outcome{}, err
			}
		}

		validationPassed, validationOutput := runValidation(turnCtx, e.Workdir, cfg.ValidationCommands)
		if validationOutput != "" {
			_ = persistFailurePattern(e.Workdir, validationOutput)
			if repo.Root == e.Workdir {
				repo.Patterns = loadPatternsFile(e.Workdir)
			}
		}
		cancel()
		reviewResult := review.Evaluate(review.Result{
			Validated: validationPassed,
		})
		if e.ReviewRunner != nil {
			reviewPrompt := buildReviewPrompt(outcome.Tasks[activeIndex], result.Output, validationOutput)
			reviewRun, reviewErr := e.ReviewRunner.Run(turnCtx, codex.Request{
				Prompt:   reviewPrompt,
				Workdir:  e.Workdir,
				MaxTurns: 1,
			})
			if reviewErr == nil {
				reviewResult.Reason = strings.TrimSpace(reviewRun.Output)
			}
		}

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
		if err := e.emitSession(runlog.SessionReport{
			Session:   outcome.Session,
			Runtime:   result.Runtime,
			Decision:  "running",
			Reason:    summarizeTurn(turn, result.Output, validationPassed),
			TurnCount: len(outcome.Turns),
			Tasks:     cloneTasks(outcome.Tasks),
		}); err != nil {
			return Outcome{}, err
		}
		payload := parseStopPayload(result.Output)
		taskDone := reviewResult.Approved || hasStopSignal(result.Output)
		outcome.Tasks = updateTaskStatuses(outcome.Tasks, activeIndex, taskDone)
		if payload != nil {
			outcome.Tasks[activeIndex].Summary = payload.Summary
			outcome.Tasks[activeIndex].FilesChanged = payload.FilesChanged
		}
		if err := e.emitEvent(runlog.Event{
			Kind:       "turn_completed",
			SessionID:  session.ID,
			TaskID:     outcome.Tasks[activeIndex].ID,
			Turn:       turn,
			Message:    summarizePrompt(prompt),
			Output:     result.Output,
			Validated:  validationPassed,
			Validation: validationOutput,
			Runtime: map[string]any{
				"model":      result.Runtime.Model,
				"provider":   result.Runtime.Provider,
				"sandbox":    result.Runtime.Sandbox,
				"approval":   result.Runtime.Approval,
				"session_id": result.Runtime.SessionID,
			},
		}); err != nil {
			return Outcome{}, err
		}
		if err := e.emitEvent(runlog.Event{
			Kind:      "task_state",
			SessionID: session.ID,
			TaskID:    outcome.Tasks[activeIndex].ID,
			Message:   outcome.Tasks[activeIndex].Title,
			Tasks:     cloneTasks(outcome.Tasks),
		}); err != nil {
			return Outcome{}, err
		}

		switch {
		case strings.Contains(result.Output, escalateMarker):
			outcome.Tasks = blockActiveTask(outcome.Tasks, activeIndex)
			session, _ = e.Sessions.Close(session.ID)
			outcome.Session = session
			outcome.Decision = Decision{Action: ActionEscalate, Reason: "worker requested escalation"}
			return outcome, nil
		case hasStopSignal(result.Output):
			if firstActiveTaskIndex(outcome.Tasks) != -1 {
				continue
			}
			session, _ = e.Sessions.Close(session.ID)
			outcome.Session = session
			outcome.Decision = Decision{Action: ActionStop, Reason: "runner requested stop"}
			return outcome, nil
		case reviewResult.Approved:
			if firstActiveTaskIndex(outcome.Tasks) != -1 {
				continue
			}
			session, _ = e.Sessions.Close(session.ID)
			outcome.Session = session
			outcome.Decision = Decision{Action: ActionStop, Reason: "validation passed"}
			return outcome, nil
		}
	}

	outcome.Tasks = blockActiveTask(outcome.Tasks, firstActiveTaskIndex(outcome.Tasks))
	session, _ = e.Sessions.Close(session.ID)
	outcome.Session = session
	outcome.Decision = Decision{Action: ActionEscalate, Reason: "max turns reached"}
	return outcome, nil
}

func (e Engine) emitEvent(event runlog.Event) error {
	if e.EventSink == nil {
		return nil
	}
	return e.EventSink(event)
}

func (e Engine) emitSession(report runlog.SessionReport) error {
	if e.SessionSink == nil {
		return nil
	}
	return e.SessionSink(report)
}

func cloneTasks(items []tasks.Task) []tasks.Task {
	if len(items) == 0 {
		return nil
	}
	next := make([]tasks.Task, len(items))
	copy(next, items)
	return next
}

const promptDocsBudget = 4000
const promptDocSnippetLimit = 800

func buildPrompt(role, goal string, repo workspace.Context, plannedTasks []tasks.Task, turns []Turn, activeIndex int) (string, int) {
	var builder strings.Builder
	docsUsed := 0
	builder.WriteString("Goal:\n")
	builder.WriteString(goal)
	if len(plannedTasks) > 0 && activeIndex >= 0 && activeIndex < len(plannedTasks) {
		builder.WriteString("\n\nActive task:\n")
		builder.WriteString("- [")
		builder.WriteString(plannedTasks[activeIndex].Status)
		builder.WriteString("] ")
		builder.WriteString(plannedTasks[activeIndex].Title)
		builder.WriteString(": ")
		builder.WriteString(plannedTasks[activeIndex].Goal)
		builder.WriteString("\n")

		builder.WriteString("\nQueued tasks:\n")
		for index, task := range plannedTasks {
			if index == activeIndex {
				continue
			}
			builder.WriteString("- [")
			builder.WriteString(task.Status)
			builder.WriteString("] ")
			builder.WriteString(task.Title)
			builder.WriteString(": ")
			builder.WriteString(task.Goal)
			builder.WriteString("\n")
		}

		completed := completedTasks(plannedTasks)
		if len(completed) > 0 {
			builder.WriteString("\nCompleted tasks:\n")
			for _, task := range completed {
				builder.WriteString("- ")
				builder.WriteString(task.Title)
				if task.Summary != "" {
					builder.WriteString(": ")
					builder.WriteString(task.Summary)
				}
				if len(task.FilesChanged) > 0 {
					builder.WriteString(" (files: ")
					builder.WriteString(strings.Join(task.FilesChanged, ", "))
					builder.WriteString(")")
				}
				builder.WriteString("\n")
			}
		}
	}
	builder.WriteString("\n\nRepository context:\n")
	builder.WriteString(repo.Readme)
	if repo.Agents != "" {
		builder.WriteString("\n\nAgent rules:\n")
		builder.WriteString(repo.Agents)
	}
	if role == promptRoleWorker && len(repo.Docs) > 0 {
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
	if role == promptRoleWorker && strings.TrimSpace(repo.Patterns) != "" {
		builder.WriteString("\n\nKnown failure patterns:\n")
		builder.WriteString(strings.TrimSpace(repo.Patterns))
		builder.WriteString("\n")
	}
	if role == promptRoleWorker && len(turns) > 0 {
		last := turns[len(turns)-1]
		builder.WriteString("\n\nPrevious turn summary:\n")
		builder.WriteString(summarizeTurn(last.Number, last.RunnerResult.Output, last.ValidationPassed))
		if last.ValidationOutput != "" {
			builder.WriteString("\n\nValidation errors from last turn:\n")
			builder.WriteString(last.ValidationOutput)
		}
	}
	if role == promptRolePlanner {
		builder.WriteString("\n\nReturn a concise task-oriented response.")
	} else {
		builder.WriteString("\n\nRespond with progress, and include [canx:stop] when the task is complete.")
	}
	return builder.String(), docsUsed
}

func buildReviewPrompt(task tasks.Task, workerOutput, validationOutput string) string {
	var builder strings.Builder
	builder.WriteString("Review task:\n")
	builder.WriteString(task.Title)
	builder.WriteString(": ")
	builder.WriteString(task.Goal)
	builder.WriteString("\n\nWorker output:\n")
	builder.WriteString(strings.TrimSpace(workerOutput))
	if strings.TrimSpace(validationOutput) != "" {
		builder.WriteString("\n\nValidation output:\n")
		builder.WriteString(strings.TrimSpace(validationOutput))
	}
	builder.WriteString("\n\nReply with a concise review verdict and key issue summary.")
	return builder.String()
}

func updateTaskStatuses(items []tasks.Task, activeIndex int, done bool) []tasks.Task {
	if len(items) == 0 {
		return items
	}

	next := make([]tasks.Task, len(items))
	copy(next, items)
	if activeIndex < 0 || activeIndex >= len(next) {
		return next
	}
	if done {
		next[activeIndex].Status = tasks.StatusDone
	} else {
		next[activeIndex].Status = tasks.StatusInProgress
	}
	return next
}

func completedTasks(items []tasks.Task) []tasks.Task {
	var completed []tasks.Task
	for _, item := range items {
		if item.Status == tasks.StatusDone {
			completed = append(completed, item)
		}
	}
	return completed
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
	if len(summary) > 1000 {
		summary = summary[:1000] + "...(truncated)"
	}

	status := "validation_failed"
	if validated {
		status = "validation_passed"
	}

	return "turn=" + strconv.Itoa(turn) + " " + status + " output=" + summary
}

func hasStopSignal(output string) bool {
	return strings.Contains(output, stopMarker) || strings.Contains(output, "[canx:stop:")
}

func parseStopPayload(output string) *stopPayload {
	start := strings.Index(output, "[canx:stop:")
	if start == -1 {
		return nil
	}
	body := output[start+len("[canx:stop:"):]
	end := strings.LastIndex(body, "]")
	if end == -1 {
		return nil
	}
	body = body[:end]
	var payload stopPayload
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		return nil
	}
	return &payload
}

func summarizePrompt(prompt string) string {
	prompt = strings.TrimSpace(prompt)
	if len(prompt) > 400 {
		return prompt[:400] + "...(truncated)"
	}
	return prompt
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

func persistFailurePattern(root, pattern string) error {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return nil
	}
	dir := filepath.Join(root, ".canx")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, "patterns.md")
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if strings.Contains(string(existing), pattern) {
		return nil
	}
	var builder strings.Builder
	if len(existing) > 0 {
		builder.Write(existing)
		if !strings.HasSuffix(builder.String(), "\n") {
			builder.WriteString("\n")
		}
	}
	builder.WriteString("- ")
	builder.WriteString(strings.ReplaceAll(pattern, "\n", "\n  "))
	builder.WriteString("\n")
	return os.WriteFile(path, []byte(builder.String()), 0o644)
}

func loadPatternsFile(root string) string {
	data, err := os.ReadFile(filepath.Join(root, ".canx", "patterns.md"))
	if err != nil {
		return ""
	}
	return string(data)
}

func blockActiveTask(items []tasks.Task, activeIndex int) []tasks.Task {
	if len(items) == 0 || activeIndex < 0 || activeIndex >= len(items) {
		return items
	}
	next := make([]tasks.Task, len(items))
	copy(next, items)
	next[activeIndex].Status = tasks.StatusBlocked
	return next
}

func firstActiveTaskIndex(items []tasks.Task) int {
	for index, item := range items {
		if item.Status == tasks.StatusPending || item.Status == tasks.StatusInProgress {
			return index
		}
	}
	return -1
}
