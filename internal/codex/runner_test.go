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
