package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/farmcan/canx/internal/codex"
	"github.com/farmcan/canx/internal/loop"
	"github.com/farmcan/canx/internal/rooms"
	"github.com/farmcan/canx/internal/runlog"
	"github.com/farmcan/canx/internal/tasks"
	"github.com/farmcan/canx/internal/workspace"
)

func main() {
	cfg, opts, command, args := parseFlags()
	output, err := dispatch(command, cfg, opts, args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "canx: %v\n", err)
		os.Exit(1)
	}

	if output != "" {
		fmt.Println(output)
	}
}

type Options struct {
	RepoPath    string
	CodexBin    string
	RunnerMode  string
	PlannerMode string
	TurnTimeout time.Duration
	Validations []string
}

func parseFlags() (loop.Config, Options, string, []string) {
	var (
		goal     = flag.String("goal", "bootstrap canx", "high-level goal for this run")
		maxTurns = flag.Int("max-turns", 1, "maximum number of loop turns")
		budget   = flag.Int("budget-seconds", 0, "total run budget in seconds (0 means disabled)")
		repoPath = flag.String("repo", ".", "target repository path")
		codexBin = flag.String("codex-bin", "codex", "codex binary path")
		runner   = flag.String("runner", "exec", "runner mode: exec or mock")
		planner  = flag.String("planner", "single", "planner mode: single or codx")
		timeout  = flag.Duration("turn-timeout", 30*time.Second, "timeout per loop turn")
	)
	var validations multiFlag
	flag.Var(&validations, "validate", "validation command to run after each turn (repeatable)")

	flag.Parse()

	command, args := defaultCommand(flag.Args())
	return loop.Config{
			Goal:          *goal,
			MaxTurns:      *maxTurns,
			BudgetSeconds: *budget,
		}, Options{
			RepoPath:    *repoPath,
			CodexBin:    *codexBin,
			RunnerMode:  *runner,
			PlannerMode: *planner,
			TurnTimeout: *timeout,
			Validations: validations,
		}, command, args
}

func defaultCommand(args []string) (string, []string) {
	if len(args) == 0 {
		return "run", nil
	}
	return args[0], args[1:]
}

func run(cfg loop.Config, opts Options) (string, error) {
	switch opts.RunnerMode {
	case "", "exec":
		return runWithRunner(cfg, opts, codex.NewExecRunnerInDir(opts.CodexBin, opts.RepoPath))
	case "mock":
		return runWithRunner(cfg, opts, codex.NewMockRunner(codex.Result{
			Output: "mock worker progress [canx:stop]",
		}))
	default:
		return "", fmt.Errorf("unknown runner mode: %s", opts.RunnerMode)
	}
}

func runWithRunner(cfg loop.Config, opts Options, runner codex.Runner) (string, error) {
	if err := cfg.Validate(); err != nil {
		return "", err
	}

	repoPath := opts.RepoPath
	if repoPath == "" {
		repoPath = "."
	}

	absRepoPath, err := filepath.Abs(repoPath)
	if err != nil {
		return "", err
	}

	repo, err := workspace.Load(absRepoPath)
	if err != nil {
		return "", err
	}

	planner, err := selectPlanner(opts, absRepoPath)
	if err != nil {
		return "", err
	}

	cfg.ValidationCommands = opts.Validations
	eventStore := runlog.NewEventStore(absRepoPath)
	roomStore := rooms.NewStore(absRepoPath)
	runID := runlog.NewRunID()
	initialRun := runlog.RunRecord{
		ID:        runID,
		Goal:      cfg.Goal,
		RepoRoot:  absRepoPath,
		Status:    "running",
		StartedAt: time.Now(),
	}
	if err := eventStore.SaveRun(initialRun); err != nil {
		return "", err
	}
	if err := roomStore.SaveRoom(rooms.Room{
		ID:       "room-" + runID,
		Title:    "Main Room",
		RunID:    runID,
		RepoRoot: absRepoPath,
	}); err != nil {
		return "", err
	}
	if err := eventStore.AppendEvent(runID, runlog.Event{
		Kind:      "run_started",
		Message:   cfg.Goal,
		Timestamp: initialRun.StartedAt,
	}); err != nil {
		return "", err
	}
	engine := loop.Engine{
		Runner:      runner,
		Planner:     planner,
		Workdir:     absRepoPath,
		TurnTimeout: opts.TurnTimeout,
		EventSink: func(event runlog.Event) error {
			if err := eventStore.AppendEvent(runID, event); err != nil {
				return err
			}
			return updateRunProgress(eventStore, initialRun, runID, cfg.Goal, absRepoPath, event)
		},
	}

	outcome, err := engine.Run(context.Background(), cfg, repo)
	if err != nil {
		return "", err
	}
	finishedAt := time.Now()
	record := runlog.RunRecord{
		ID:         runID,
		Goal:       cfg.Goal,
		RepoRoot:   absRepoPath,
		Status:     string(outcome.Decision.Action),
		Reason:     outcome.Decision.Reason,
		SessionID:  outcome.Session.ID,
		TurnCount:  len(outcome.Turns),
		TaskCount:  len(outcome.Tasks),
		Tasks:      outcome.Tasks,
		StartedAt:  initialRun.StartedAt,
		FinishedAt: &finishedAt,
	}
	if err := eventStore.SaveRun(record); err != nil {
		return "", err
	}
	for _, event := range outcomeEvents(runID, outcome) {
		if err := eventStore.AppendEvent(runID, event); err != nil {
			return "", err
		}
	}
	if _, err := runlog.WriteSessionReport(absRepoPath, runlog.SessionReport{
		Session:   outcome.Session,
		RunID:     runID,
		Runtime:   latestRuntime(outcome),
		Decision:  outcome.Decision.Action,
		Reason:    outcome.Decision.Reason,
		TurnCount: len(outcome.Turns),
		Tasks:     outcome.Tasks,
	}); err != nil {
		return "", err
	}

	return fmt.Sprintf(
		"canx run=%s decision=%s reason=%s turns=%d tasks=%d session=%s workspace=%s docs=%d model=%s sandbox=%s approval=%s runtime_session=%s",
		runID,
		outcome.Decision.Action,
		outcome.Decision.Reason,
		len(outcome.Turns),
		len(outcome.Tasks),
		outcome.Session.ID,
		absRepoPath,
		len(repo.Docs),
		latestRuntime(outcome).Model,
		latestRuntime(outcome).Sandbox,
		latestRuntime(outcome).Approval,
		latestRuntime(outcome).SessionID,
	), nil
}

func latestRuntime(outcome loop.Outcome) codex.Runtime {
	if len(outcome.Turns) == 0 {
		return codex.Runtime{}
	}
	return outcome.Turns[len(outcome.Turns)-1].RunnerResult.Runtime
}

func selectPlanner(opts Options, workdir string) (tasks.Planner, error) {
	switch opts.PlannerMode {
	case "", "single":
		return tasks.SingleTaskPlanner{}, nil
	case "codx":
		return tasks.CodxPlanner{
			Runner: plannerRunnerAdapter{
				runner:  codex.NewExecRunnerInDir(opts.CodexBin, workdir),
				workdir: workdir,
			},
			PromptBuilder: func(goal string) string {
				return plannerPrompt(goal, workdir)
			},
		}, nil
	default:
		return nil, fmt.Errorf("unknown planner mode: %s", opts.PlannerMode)
	}
}

func plannerPrompt(goal, workdir string) string {
	repo, err := workspace.Load(workdir)
	if err != nil {
		return tasks.DefaultPlannerPrompt(goal)
	}
	var builder strings.Builder
	builder.WriteString("You are a software delivery supervisor. Given a goal, output a JSON array of tasks.\n\n")
	builder.WriteString("Each task must have: id (string), title (string, max 40 chars), goal (string), status (\"pending\").\n\n")
	builder.WriteString("Prefer 2-5 tasks when the goal naturally contains multiple steps such as inspect + test + implement.\n")
	builder.WriteString("Use a single task only when the goal is truly atomic.\n\n")
	builder.WriteString("Repository context:\n")
	builder.WriteString(repo.Readme)
	if repo.Agents != "" {
		builder.WriteString("\n\nAgent rules:\n")
		builder.WriteString(repo.Agents)
	}
	builder.WriteString("\n\nOutput ONLY valid JSON, no explanation. Maximum 5 tasks. Example:\n")
	builder.WriteString("[{\"id\":\"task-1\",\"title\":\"Add failing test\",\"goal\":\"write a failing test for X\",\"status\":\"pending\"}]\n\n")
	builder.WriteString("Goal: ")
	builder.WriteString(goal)
	return builder.String()
}

type plannerRunnerAdapter struct {
	runner  codex.Runner
	workdir string
}

func (a plannerRunnerAdapter) Run(ctx context.Context, prompt string) (string, error) {
	result, err := a.runner.Run(ctx, codex.Request{
		Prompt:   prompt,
		Workdir:  a.workdir,
		MaxTurns: 1,
	})
	if err != nil {
		return "", err
	}
	return result.Output, nil
}

func dispatch(command string, cfg loop.Config, opts Options, args []string) (string, error) {
	switch command {
	case "run":
		return run(cfg, opts)
	case "sessions":
		return inspectSessions(opts, args)
	case "serve":
		return serve(opts)
	default:
		return "", fmt.Errorf("unknown command: %s", command)
	}
}

func inspectSessions(opts Options, args []string) (string, error) {
	repoPath := opts.RepoPath
	if repoPath == "" {
		repoPath = "."
	}

	root, err := filepath.Abs(repoPath)
	if err != nil {
		return "", err
	}
	sessionsDir := filepath.Join(root, ".canx", "sessions")

	if len(args) == 0 || args[0] == "list" {
		entries, err := os.ReadDir(sessionsDir)
		if err != nil {
			if os.IsNotExist(err) {
				return "(no sessions)", nil
			}
			return "", err
		}
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
				continue
			}
			names = append(names, entry.Name())
		}
		sort.Strings(names)
		return strings.Join(names, "\n"), nil
	}

	if args[0] == "show" && len(args) > 1 {
		data, err := os.ReadFile(filepath.Join(sessionsDir, args[1]+".json"))
		if err != nil {
			return "", err
		}
		var report runlog.SessionReport
		if err := json.Unmarshal(data, &report); err != nil {
			return "", err
		}
		formatted, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return "", err
		}
		return string(formatted), nil
	}

	return "", fmt.Errorf("usage: canxd sessions list | canxd sessions show <session-id>")
}

func outcomeEvents(runID string, outcome loop.Outcome) []runlog.Event {
	return []runlog.Event{{
		RunID:     runID,
		Kind:      "run_finished",
		SessionID: outcome.Session.ID,
		Decision:  string(outcome.Decision.Action),
		Reason:    outcome.Decision.Reason,
		Tasks:     outcome.Tasks,
	}}
}

func summarizePrompt(prompt string) string {
	prompt = strings.TrimSpace(prompt)
	if len(prompt) > 400 {
		return prompt[:400] + "...(truncated)"
	}
	return prompt
}

func updateRunProgress(store runlog.EventStore, initial runlog.RunRecord, runID, goal, repoRoot string, event runlog.Event) error {
	record, err := store.LoadRun(runID)
	if err != nil {
		record = initial
	}
	record.ID = runID
	record.Goal = goal
	record.RepoRoot = repoRoot
	record.Status = "running"
	record.UpdatedAt = time.Now()
	if record.StartedAt.IsZero() {
		record.StartedAt = initial.StartedAt
	}
	if event.SessionID != "" {
		record.SessionID = event.SessionID
	}
	if event.Turn > record.TurnCount {
		record.TurnCount = event.Turn
	}
	if len(event.Tasks) > 0 {
		record.Tasks = event.Tasks
		record.TaskCount = len(event.Tasks)
	}
	return store.SaveRun(record)
}

func init() {
	flag.CommandLine.SetOutput(os.Stderr)
}

type multiFlag []string

func (m *multiFlag) String() string {
	return fmt.Sprintf("%v", []string(*m))
}

func (m *multiFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}
