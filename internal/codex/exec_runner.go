package codex

import (
	"context"
	"os/exec"
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

	cmd := exec.CommandContext(ctx, r.bin, "exec", req.Prompt)
	if req.Workdir != "" {
		cmd.Dir = req.Workdir
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return Result{
			Output:   string(output),
			ExitCode: 1,
		}, err
	}

	return Result{
		Output:   string(output),
		ExitCode: 0,
	}, nil
}
