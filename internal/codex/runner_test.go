package codex

import (
	"context"
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
