package loop

import "testing"

func TestParseStopMarkerPayload(t *testing.T) {
	t.Parallel()

	t.Run("plain stop marker has no payload", func(t *testing.T) {
		t.Parallel()

		if !hasStopSignal("done [canx:stop]") {
			t.Fatal("expected stop signal to be detected")
		}
		if payload := parseStopPayload("done [canx:stop]"); payload != nil {
			t.Fatalf("parseStopPayload() = %#v, want nil", payload)
		}
	})

	t.Run("structured stop marker returns payload", func(t *testing.T) {
		t.Parallel()

		output := `done [canx:stop:{"summary":"fixed issue","files_changed":["internal/loop/engine.go"]}]`
		payload := parseStopPayload(output)
		if payload == nil {
			t.Fatal("expected structured stop payload")
		}
		if payload.Summary != "fixed issue" {
			t.Fatalf("payload summary = %q, want fixed issue", payload.Summary)
		}
		if len(payload.FilesChanged) != 1 || payload.FilesChanged[0] != "internal/loop/engine.go" {
			t.Fatalf("payload files changed = %#v, want engine path", payload.FilesChanged)
		}
	})

	t.Run("invalid stop payload returns nil", func(t *testing.T) {
		t.Parallel()

		output := `done [canx:stop:{"summary":}]`
		if payload := parseStopPayload(output); payload != nil {
			t.Fatalf("parseStopPayload() = %#v, want nil", payload)
		}
	})
}

func TestParseSpawnMarkerRequests(t *testing.T) {
	t.Parallel()

	t.Run("structured spawn marker returns request", func(t *testing.T) {
		t.Parallel()

		output := `need help [canx:spawn:{"title":"Add test","goal":"write regression test","reason":"parallelize test work","planned_files":["internal/loop/engine_test.go"]}]`
		requests := parseSpawnRequests(output)
		if len(requests) != 1 {
			t.Fatalf("parseSpawnRequests() len = %d, want 1", len(requests))
		}
		if requests[0].Title != "Add test" {
			t.Fatalf("request title = %q, want Add test", requests[0].Title)
		}
		if len(requests[0].PlannedFiles) != 1 || requests[0].PlannedFiles[0] != "internal/loop/engine_test.go" {
			t.Fatalf("request planned files = %#v, want engine test path", requests[0].PlannedFiles)
		}
	})

	t.Run("invalid spawn payload is ignored", func(t *testing.T) {
		t.Parallel()

		output := `need help [canx:spawn:{"title":}]`
		requests := parseSpawnRequests(output)
		if len(requests) != 0 {
			t.Fatalf("parseSpawnRequests() len = %d, want 0", len(requests))
		}
	})

	t.Run("multiple markers returns all valid spawn requests", func(t *testing.T) {
		t.Parallel()

		output := `a [canx:spawn:{"title":"Task A","goal":"do a","planned_files":["a.go"]}] b [canx:spawn:{"title":"Task B","goal":"do b","planned_files":["b.go"]}]`
		requests := parseSpawnRequests(output)
		if len(requests) != 2 {
			t.Fatalf("parseSpawnRequests() len = %d, want 2", len(requests))
		}
	})
}

func TestParseEscalateMarkerSignal(t *testing.T) {
	t.Parallel()

	if !hasEscalateSignal("blocked [canx:escalate]") {
		t.Fatal("expected escalate signal to be detected")
	}
	if hasEscalateSignal("all good") {
		t.Fatal("did not expect escalate signal")
	}
}
