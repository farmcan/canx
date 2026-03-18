package codex

import (
	"context"
	"os/exec"
	"strings"
	"testing"
)

func TestRequestValidateRequiresPrompt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		req     Request
		wantErr bool
	}{
		{
			name: "valid request",
			req: Request{
				Prompt: "add task model",
			},
		},
		{
			name:    "missing prompt",
			req:     Request{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExecRunnerRejectsMissingBinary(t *testing.T) {
	t.Parallel()

	runner := NewExecRunner("definitely-not-a-real-codex-binary")
	_, err := runner.Run(context.Background(), Request{Prompt: "hello"})
	if err == nil {
		t.Fatal("expected exec runner error")
	}
}

func TestRunErrorIncludesOutput(t *testing.T) {
	t.Parallel()

	err := RunError{Err: exec.ErrNotFound, Output: "runner failed"}
	if !strings.Contains(err.Error(), "runner failed") {
		t.Fatalf("Error() = %q", err.Error())
	}
}

func TestExecRunnerWithRealCodexIfAvailable(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("codex"); err != nil {
		t.Skip("codex binary not found, skipping integration test")
	}

	runner := NewExecRunner("codex")
	result, err := runner.Run(context.Background(), Request{
		Prompt:  "Output the text CANX_OK and nothing else.",
		Workdir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("Run() error = %v, output = %q", err, result.Output)
	}
	if !strings.Contains(result.Output, "CANX_OK") {
		t.Fatalf("expected CANX_OK in output, got: %q", result.Output)
	}
}

func TestParseExecOutputExtractsRuntimeAndFinalMessage(t *testing.T) {
	t.Parallel()

	raw := `OpenAI Codex v0.111.0 (research preview)
--------
workdir: /tmp/repo
model: gpt-5.4
provider: openai
approval: never
sandbox: read-only
reasoning effort: medium
reasoning summaries: none
session id: 019d01c7-404e-7cd2-91e5-b421f62c6d09
--------
user
Goal: test

codex
- first point
- second point [canx:stop]
tokens used
3,971
`

	result := parseExecOutput(raw)
	if got, want := result.Runtime.Model, "gpt-5.4"; got != want {
		t.Fatalf("model = %q, want %q", got, want)
	}
	if got, want := result.Runtime.Sandbox, "read-only"; got != want {
		t.Fatalf("sandbox = %q, want %q", got, want)
	}
	if strings.Contains(result.Output, "OpenAI Codex") {
		t.Fatalf("expected cleaned output, got raw banner: %q", result.Output)
	}
	if !strings.Contains(result.Output, "[canx:stop]") {
		t.Fatalf("expected final output to keep stop marker: %q", result.Output)
	}
}

func TestParseExecOutputDropsExecTraceNoise(t *testing.T) {
	t.Parallel()

	raw := `codex
先检查仓库。
exec
/bin/zsh -lc "sed -n '1,200p' README.md" in /tmp/repo
真正结论 [canx:stop]
`

	result := parseExecOutput(raw)
	if strings.Contains(result.Output, "/bin/zsh -lc") {
		t.Fatalf("expected exec trace to be removed: %q", result.Output)
	}
	if !strings.Contains(result.Output, "真正结论 [canx:stop]") {
		t.Fatalf("expected conclusion to remain: %q", result.Output)
	}
}
