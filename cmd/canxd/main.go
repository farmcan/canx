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

	fmt.Println(output)
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
		return runWithRunner(cfg, opts, codex.NewExecRunner(opts.CodexBin))
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
	engine := loop.Engine{
		Runner:      runner,
		Planner:     planner,
		Workdir:     absRepoPath,
		TurnTimeout: opts.TurnTimeout,
	}

	outcome, err := engine.Run(context.Background(), cfg, repo)
	if err != nil {
		return "", err
	}
	if _, err := runlog.WriteSessionReport(absRepoPath, runlog.SessionReport{
		Session:   outcome.Session,
		Runtime:   latestRuntime(outcome),
		Decision:  outcome.Decision.Action,
		Reason:    outcome.Decision.Reason,
		TurnCount: len(outcome.Turns),
		Tasks:     outcome.Tasks,
	}); err != nil {
		return "", err
	}

	return fmt.Sprintf(
		"canx decision=%s reason=%s turns=%d tasks=%d session=%s workspace=%s docs=%d model=%s sandbox=%s approval=%s runtime_session=%s",
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
				runner:  codex.NewExecRunner(opts.CodexBin),
				workdir: workdir,
			},
		}, nil
	default:
		return nil, fmt.Errorf("unknown planner mode: %s", opts.PlannerMode)
	}
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
