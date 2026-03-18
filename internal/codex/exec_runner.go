package codex

import (
	"context"
	"os/exec"
	"strings"
)

type ExecRunner struct {
	bin              string
	skipGitRepoCheck bool
}

func NewExecRunner(bin string) ExecRunner {
	return ExecRunner{bin: bin}
}

// NewExecRunnerInDir creates an ExecRunner with the git-repo check evaluated
// once at construction time, avoiding a subprocess on every Run call.
func NewExecRunnerInDir(bin, workdir string) ExecRunner {
	return ExecRunner{bin: bin, skipGitRepoCheck: shouldSkipGitRepoCheck(workdir)}
}

func (r ExecRunner) Run(ctx context.Context, req Request) (Result, error) {
	if err := req.Validate(); err != nil {
		return Result{}, err
	}

	args := []string{"exec", "-"}
	if r.skipGitRepoCheck || shouldSkipGitRepoCheck(req.Workdir) {
		args = append(args, "--skip-git-repo-check")
	}
	cmd := exec.CommandContext(ctx, r.bin, args...)
	cmd.Stdin = strings.NewReader(req.Prompt)
	if req.Workdir != "" {
		cmd.Dir = req.Workdir
	}

	output, err := cmd.CombinedOutput()
	parsed := parseExecOutput(string(output))
	if err != nil {
		return Result{
			Output:   parsed.Output,
			ExitCode: 1,
			Runtime:  parsed.Runtime,
		}, RunError{Err: err, Output: strings.TrimSpace(parsed.Output)}
	}

	return Result{
		Output:   parsed.Output,
		ExitCode: 0,
		Runtime:  parsed.Runtime,
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

func parseExecOutput(raw string) Result {
	result := Result{Output: strings.TrimSpace(raw)}
	lines := strings.Split(raw, "\n")
	inCodexBlock := false
	skipNextExecTrace := false
	cleaned := make([]string, 0, len(lines))

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "model: "):
			result.Runtime.Model = strings.TrimPrefix(trimmed, "model: ")
		case strings.HasPrefix(trimmed, "provider: "):
			result.Runtime.Provider = strings.TrimPrefix(trimmed, "provider: ")
		case strings.HasPrefix(trimmed, "approval: "):
			result.Runtime.Approval = strings.TrimPrefix(trimmed, "approval: ")
		case strings.HasPrefix(trimmed, "sandbox: "):
			result.Runtime.Sandbox = strings.TrimPrefix(trimmed, "sandbox: ")
		case strings.HasPrefix(trimmed, "session id: "):
			result.Runtime.SessionID = strings.TrimPrefix(trimmed, "session id: ")
		case trimmed == "codex":
			inCodexBlock = true
			continue
		case trimmed == "user":
			inCodexBlock = false
			continue
		case strings.HasPrefix(trimmed, "tokens used"):
			inCodexBlock = false
			continue
		}

		if inCodexBlock {
			if trimmed == "exec" {
				skipNextExecTrace = true
				continue
			}
			if skipNextExecTrace && strings.HasPrefix(trimmed, "/bin/") {
				skipNextExecTrace = false
				continue
			}
			skipNextExecTrace = false
			cleaned = append(cleaned, line)
		}
	}

	if len(cleaned) > 0 {
		result.Output = strings.TrimSpace(strings.Join(cleaned, "\n"))
	}
	return result
}
