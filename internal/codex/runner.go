package codex

import (
	"context"
	"errors"
)

var ErrMissingPrompt = errors.New("missing prompt")

type Runner interface {
	Run(ctx context.Context, req Request) (Result, error)
}

type Request struct {
	Prompt   string
	Workdir  string
	MaxTurns int
}

func (r Request) Validate() error {
	if r.Prompt == "" {
		return ErrMissingPrompt
	}
	return nil
}

type Result struct {
	Output   string
	ExitCode int
}
