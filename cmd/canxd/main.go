package main

import (
	"flag"
	"fmt"
	"os"

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
	RepoPath string
}

func parseFlags() (loop.Config, Options) {
	var (
		goal     = flag.String("goal", "bootstrap canx", "high-level goal for this run")
		maxTurns = flag.Int("max-turns", 1, "maximum number of loop turns")
		repoPath = flag.String("repo", "", "target repository path")
	)

	flag.Parse()

	return loop.Config{
			Goal:     *goal,
			MaxTurns: *maxTurns,
		}, Options{
			RepoPath: *repoPath,
		}
}

func run(cfg loop.Config, opts Options) (string, error) {
	if err := cfg.Validate(); err != nil {
		return "", err
	}

	workspaceSummary := "workspace=unloaded"
	if opts.RepoPath != "" {
		ctx, err := workspace.Load(opts.RepoPath)
		if err != nil {
			return "", err
		}

		workspaceSummary = fmt.Sprintf("workspace=%s docs=%d", opts.RepoPath, len(ctx.Docs))
	}

	return fmt.Sprintf(
		"canx loop ready: goal=%s max_turns=%d %s",
		cfg.Goal,
		cfg.MaxTurns,
		workspaceSummary,
	), nil
}

func init() {
	flag.CommandLine.SetOutput(os.Stderr)
}
