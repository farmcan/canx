package codex

import (
	"context"
	"os/exec"
	"strings"
)

type ExecRunner struct {
	bin string
}

func NewExecRunner(bin string) ExecRunner {
	return ExecRunner{bin: bin}
}

func (r ExecRunner) Run(ctx context.Context, req Request) (Result, error) {
	if err := req.Validate(); err != nil {
		return Result{}, err
	}

	args := []string{"exec", "-"}
	if shouldSkipGitRepoCheck(req.Workdir) {
		args = append(args, "--skip-git-repo-check")
	}
	cmd := exec.CommandContext(ctx, r.bin, args...)
	cmd.Stdin = strings.NewReader(req.Prompt)
	if req.Workdir != "" {
		cmd.Dir = req.Workdir
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return Result{
			Output:   string(output),
			ExitCode: 1,
		}, RunError{Err: err, Output: strings.TrimSpace(string(output))}
	}

	return Result{
		Output:   string(output),
		ExitCode: 0,
	}, nil
}

func shouldSkipGitRepoCheck(workdir string) bool {
	if workdir == "" {
		return false
	}

	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = workdir
	return cmd.Run() != nil
}
