package review

import "testing"

func TestGateRejectsMissingValidation(t *testing.T) {
	t.Parallel()

	result := Evaluate(Result{})
	if result.Approved {
		t.Fatal("expected review rejection")
	}
}

func TestGateApprovesValidatedScopedResult(t *testing.T) {
	t.Parallel()

	result := Evaluate(Result{
		Validated: true,
		InScope:   true,
	})
	if !result.Approved {
		t.Fatal("expected review approval")
	}
}
