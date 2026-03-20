package loop

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
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

type taskExecution struct {
	Index       int
	TaskSession sessions.Session
	Prompt      string
	Result      codex.Result
	Err         error
}

func (e Engine) Run(ctx context.Context, cfg Config, repo workspace.Context) (Outcome, error) {
	cfg = cfg.WithDefaults()
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
		Turns:     snapshotSessionTurns(outcome.Turns),
		Tasks:     cloneTasks(outcome.Tasks),
	}); err != nil {
		return Outcome{}, err
	}
	for turn := 1; turn <= cfg.MaxTurns; turn++ {
		if firstActiveTaskIndex(outcome.Tasks) == -1 {
			session, _ = e.Sessions.Close(session.ID)
			outcome.Session = session
			outcome.Decision = Decision{Action: ActionStop, Reason: "all tasks complete"}
			return outcome, nil
		}

		runnable := selectRunnableTasks(outcome.Tasks, cfg.MaxWorkers)
		if len(runnable) == 0 {
			activeIndex := firstActiveTaskIndex(outcome.Tasks)
			if activeIndex != -1 {
				runnable = []int{activeIndex}
			}
		}
		if len(runnable) == 0 {
			break
		}

		executions := make([]taskExecution, 0, len(runnable))
		executionCh := make(chan taskExecution, len(runnable))
		var wg sync.WaitGroup
		for _, activeIndex := range runnable {
			taskSessionID := outcome.Tasks[activeIndex].OwnerSessionID
			taskSession := sessions.Session{}
			var err error
			if taskSessionID == "" {
				taskSession, err = e.Sessions.Spawn(sessions.SpawnRequest{
					Label: outcome.Tasks[activeIndex].ID,
					Mode:  sessions.ModePersistent,
					CWD:   e.Workdir,
				})
				if err != nil {
					return Outcome{}, err
				}
				outcome.Tasks[activeIndex].OwnerSessionID = taskSession.ID
			} else {
				taskSession, err = e.Sessions.Get(taskSessionID)
				if err != nil {
					return Outcome{}, err
				}
			}

			prompt, docsUsed := buildPrompt(promptRoleWorker, cfg.Goal, repo, outcome.Tasks, outcome.Turns, activeIndex)
			if outcome.PromptDocsUsed == 0 && docsUsed > 0 {
				outcome.PromptDocsUsed = docsUsed
			}

			wg.Add(1)
			go func(index int, prompt string, taskSession sessions.Session) {
				defer wg.Done()
				turnCtx := ctx
				cancel := func() {}
				if e.TurnTimeout > 0 {
					turnCtx, cancel = context.WithTimeout(ctx, e.TurnTimeout)
				}
				defer cancel()

				result, err := e.Runner.Run(turnCtx, codex.Request{
					Prompt:     prompt,
					Workdir:    e.Workdir,
					MaxTurns:   1,
					SessionKey: taskSession.ID,
				})
				if err != nil && !hasStopSignal(result.Output) && !hasEscalateSignal(result.Output) {
					executionCh <- taskExecution{Index: index, TaskSession: taskSession, Prompt: prompt, Result: result, Err: err}
					return
				}
				executionCh <- taskExecution{Index: index, TaskSession: taskSession, Prompt: prompt, Result: result, Err: nil}
			}(activeIndex, prompt, taskSession)
		}
		wg.Wait()
		close(executionCh)
		for execution := range executionCh {
			if execution.Err != nil {
				return Outcome{}, execution.Err
			}
			executions = append(executions, execution)
		}
		sort.Slice(executions, func(i, j int) bool {
			return executions[i].Index < executions[j].Index
		})

		roundSawStop := false
		roundSawValidationApproval := false
		roundSawEscalation := false
		for _, execution := range executions {
			validationPassed, validationOutput := runValidation(ctx, e.Workdir, cfg.ValidationCommands)
			if validationOutput != "" {
				_ = persistFailurePattern(e.Workdir, validationOutput)
				if repo.Root == e.Workdir {
					repo.Patterns = loadPatternsFile(e.Workdir)
				}
			}

			reviewResult := review.Evaluate(review.Result{
				Validated: validationPassed,
			})
			if e.ReviewRunner != nil {
				reviewPrompt := buildReviewPrompt(outcome.Tasks[execution.Index], execution.Result.Output, validationOutput)
				reviewRun, reviewErr := e.ReviewRunner.Run(ctx, codex.Request{
					Prompt:   reviewPrompt,
					Workdir:  e.Workdir,
					MaxTurns: 1,
				})
				if reviewErr == nil {
					if verdict, ok := review.ParseVerdict(reviewRun.Output); ok {
						reviewResult.Approved = verdict.Approved
						reviewResult.Reason = verdict.Reason
						reviewResult.Warnings = verdict.Warnings
					} else {
						reviewResult.Reason = strings.TrimSpace(reviewRun.Output)
					}
				}
			}

			turnNumber := len(outcome.Turns) + 1
			outcome.Turns = append(outcome.Turns, Turn{
				Number:           turnNumber,
				Prompt:           execution.Prompt,
				RunnerResult:     execution.Result,
				ValidationPassed: validationPassed,
				ValidationOutput: validationOutput,
				Review:           reviewResult,
			})
			outcome.Logs = append(outcome.Logs, runlog.Entry{
				Goal:     cfg.Goal,
				Decision: reviewDecision(validationPassed, execution.Result.Output, turnNumber, cfg.MaxTurns),
				Summary:  summarizeTurn(turnNumber, execution.Result.Output, validationPassed),
			})
			session, err = e.Sessions.Steer(session.ID, summarizeTurn(turnNumber, execution.Result.Output, validationPassed))
			if err != nil {
				return Outcome{}, err
			}
			outcome.Session = session
			if err := e.emitSession(runlog.SessionReport{
				Session:   outcome.Session,
				Runtime:   execution.Result.Runtime,
				Decision:  "running",
				Reason:    summarizeTurn(turnNumber, execution.Result.Output, validationPassed),
				TurnCount: len(outcome.Turns),
				Turns:     snapshotSessionTurns(outcome.Turns),
				Tasks:     cloneTasks(outcome.Tasks),
			}); err != nil {
				return Outcome{}, err
			}

			payload := parseStopPayload(execution.Result.Output)
			for _, request := range parseSpawnRequests(execution.Result.Output) {
				if ok, _ := canApproveSpawn(outcome.Tasks[execution.Index], outcome.Tasks, request, cfg); ok {
					outcome.Tasks = append(outcome.Tasks, tasks.Task{
						ID:           buildChildTaskID(outcome.Tasks[execution.Index], request, childCount(outcome.Tasks, outcome.Tasks[execution.Index].ID)+1),
						Title:        request.Title,
						Goal:         request.Goal,
						Status:       tasks.StatusPending,
						ParentTaskID: outcome.Tasks[execution.Index].ID,
						SpawnDepth:   outcome.Tasks[execution.Index].SpawnDepth + 1,
						PlannedFiles: append([]string(nil), request.PlannedFiles...),
					})
				}
			}
			taskDone := reviewResult.Approved || hasStopSignal(execution.Result.Output)
			outcome.Tasks = updateTaskStatuses(outcome.Tasks, execution.Index, taskDone)
			if payload != nil {
				outcome.Tasks[execution.Index].Summary = payload.Summary
				outcome.Tasks[execution.Index].FilesChanged = payload.FilesChanged
			}
			if err := e.emitEvent(runlog.Event{
				Kind:       "turn_completed",
				SessionID:  execution.TaskSession.ID,
				TaskID:     outcome.Tasks[execution.Index].ID,
				Turn:       turnNumber,
				Message:    summarizePrompt(execution.Prompt),
				Output:     execution.Result.Output,
				Validated:  validationPassed,
				Validation: validationOutput,
				Runtime: map[string]any{
					"model":      execution.Result.Runtime.Model,
					"provider":   execution.Result.Runtime.Provider,
					"sandbox":    execution.Result.Runtime.Sandbox,
					"approval":   execution.Result.Runtime.Approval,
					"session_id": execution.Result.Runtime.SessionID,
				},
			}); err != nil {
				return Outcome{}, err
			}
			if err := e.emitEvent(runlog.Event{
				Kind:      "task_state",
				SessionID: execution.TaskSession.ID,
				TaskID:    outcome.Tasks[execution.Index].ID,
				Message:   outcome.Tasks[execution.Index].Title,
				Tasks:     cloneTasks(outcome.Tasks),
			}); err != nil {
				return Outcome{}, err
			}

			if hasEscalateSignal(execution.Result.Output) {
				outcome.Tasks = blockActiveTask(outcome.Tasks, execution.Index)
				roundSawEscalation = true
			}
			if hasStopSignal(execution.Result.Output) {
				roundSawStop = true
			}
			if reviewResult.Approved {
				roundSawValidationApproval = true
			}
		}

		if roundSawEscalation {
			session, _ = e.Sessions.Close(session.ID)
			outcome.Session = session
			outcome.Decision = Decision{Action: ActionEscalate, Reason: "worker requested escalation"}
			return outcome, nil
		}
		if firstActiveTaskIndex(outcome.Tasks) == -1 {
			session, _ = e.Sessions.Close(session.ID)
			outcome.Session = session
			switch {
			case roundSawStop:
				outcome.Decision = Decision{Action: ActionStop, Reason: "runner requested stop"}
			case roundSawValidationApproval:
				outcome.Decision = Decision{Action: ActionStop, Reason: "validation passed"}
			default:
				outcome.Decision = Decision{Action: ActionStop, Reason: "all tasks complete"}
			}
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

func snapshotSessionTurns(turns []Turn) []runlog.SessionTurn {
	if len(turns) == 0 {
		return nil
	}
	snapshots := make([]runlog.SessionTurn, 0, len(turns))
	for _, turn := range turns {
		snapshots = append(snapshots, runlog.SessionTurn{
			Number:           turn.Number,
			Summary:          summarizeTurn(turn.Number, turn.RunnerResult.Output, turn.ValidationPassed),
			Output:           turn.RunnerResult.Output,
			ValidationPassed: turn.ValidationPassed,
			ValidationOutput: turn.ValidationOutput,
			Review:           turn.Review,
			Runtime:          turn.RunnerResult.Runtime,
		})
	}
	return snapshots
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
	builder.WriteString("\n\nReply with ONLY valid JSON using this schema:\n{\"approved\":true,\"reason\":\"approved\",\"warnings\":[]}\nDo not include markdown fences or extra text.")
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

func summarizePrompt(prompt string) string {
	prompt = strings.TrimSpace(prompt)
	if len(prompt) > 400 {
		return prompt[:400] + "...(truncated)"
	}
	return prompt
}

func buildChildTaskID(parent tasks.Task, request spawnRequest, ordinal int) string {
	title := strings.TrimSpace(strings.ToLower(request.Title))
	title = strings.ReplaceAll(title, " ", "-")
	if title == "" {
		title = "child"
	}
	return parent.ID + "-child-" + strconv.Itoa(ordinal) + "-" + title
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
