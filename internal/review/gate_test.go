package review

import "testing"

func TestGateRejectsMissingValidation(t *testing.T) {
	t.Parallel()

	result := Evaluate(Result{})
	if result.Approved {
		t.Fatal("expected review rejection")
	}
}

func TestGateApprovesValidatedResult(t *testing.T) {
	t.Parallel()

	result := Evaluate(Result{Validated: true})
	if !result.Approved {
		t.Fatal("expected review approval")
	}
}
