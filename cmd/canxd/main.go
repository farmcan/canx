package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/farmcan/canx/internal/codex"
	"github.com/farmcan/canx/internal/loop"
	"github.com/farmcan/canx/internal/workspace"
)

func main() {
	cfg, opts := parseFlags()
	output, err := run(cfg, opts)
	if err != nil {
		panic(err)
	}

	fmt.Println(output)
}

type Options struct {
	RepoPath    string
	CodexBin    string
	RunnerMode  string
	TurnTimeout time.Duration
	Validations []string
}

func parseFlags() (loop.Config, Options) {
	var (
		goal     = flag.String("goal", "bootstrap canx", "high-level goal for this run")
		maxTurns = flag.Int("max-turns", 1, "maximum number of loop turns")
		repoPath = flag.String("repo", ".", "target repository path")
		codexBin = flag.String("codex-bin", "codex", "codex binary path")
		runner   = flag.String("runner", "exec", "runner mode: exec or mock")
		timeout  = flag.Duration("turn-timeout", 30*time.Second, "timeout per loop turn")
	)
	var validations multiFlag
	flag.Var(&validations, "validate", "validation command to run after each turn (repeatable)")

	flag.Parse()

	return loop.Config{
			Goal:     *goal,
			MaxTurns: *maxTurns,
		}, Options{
			RepoPath:    *repoPath,
			CodexBin:    *codexBin,
			RunnerMode:  *runner,
			TurnTimeout: *timeout,
			Validations: validations,
		}
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

	cfg.ValidationCommands = opts.Validations
	engine := loop.Engine{
		Runner:      runner,
		Workdir:     absRepoPath,
		TurnTimeout: opts.TurnTimeout,
	}

	outcome, err := engine.Run(context.Background(), cfg, repo)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(
		"canx decision=%s reason=%s turns=%d session=%s workspace=%s docs=%d",
		outcome.Decision.Action,
		outcome.Decision.Reason,
		len(outcome.Turns),
		outcome.Session.ID,
		absRepoPath,
		len(repo.Docs),
	), nil
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
