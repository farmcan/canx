package codex

import "context"

type MockRunner struct {
	Results []Result
	index   int
}

func NewMockRunner(results ...Result) *MockRunner {
	return &MockRunner{Results: results}
}

func (r *MockRunner) Run(_ context.Context, _ Request) (Result, error) {
	if len(r.Results) == 0 {
		return Result{}, nil
	}

	result := r.Results[r.index]
	if r.index < len(r.Results)-1 {
		r.index++
	}

	return result, nil
}
