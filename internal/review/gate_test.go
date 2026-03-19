package review

import "testing"

func TestParseVerdictStructuredJSON(t *testing.T) {
	t.Parallel()

	result, ok := ParseVerdict(`{"approved":false,"reason":"missing healthz","warnings":["no test update"]}`)
	if !ok {
		t.Fatal("expected structured verdict to parse")
	}
	if result.Approved {
		t.Fatal("expected approval=false")
	}
	if got, want := result.Reason, "missing healthz"; got != want {
		t.Fatalf("reason = %q, want %q", got, want)
	}
	if got, want := len(result.Warnings), 1; got != want {
		t.Fatalf("warnings len = %d, want %d", got, want)
	}
}

func TestParseVerdictFallsBackOnPlainText(t *testing.T) {
	t.Parallel()

	_, ok := ParseVerdict("review says reject")
	if ok {
		t.Fatal("expected plain text not to parse as structured verdict")
	}
}

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
