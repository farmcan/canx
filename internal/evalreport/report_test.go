package evalreport

import (
	"strings"
	"testing"
)

func TestParseGoTestJSONExtractsEvalResults(t *testing.T) {
	t.Parallel()

	input := strings.NewReader(`{"Time":"2026-03-19T00:00:00Z","Action":"output","Package":"github.com/farmcan/canx/evals/agentic","Test":"TestAgenticQuickSuite","Output":"    suite_test.go:60: {\"name\":\"stop_signal\",\"success\":true,\"decision\":\"stop\",\"reason\":\"runner requested stop\",\"turns\":1,\"tasks\":1,\"done_tasks\":1,\"duration_ms\":4,\"prompt_docs\":1,\"multi_task\":false}\n"}
{"Time":"2026-03-19T00:00:01Z","Action":"output","Package":"github.com/farmcan/canx/evals/agentic","Test":"TestPlannerRealSmokeIfEnabled","Output":"    suite_test.go:152: planner_multi_task_rate=1.00\n"}`)

	report, err := ParseGoTestJSON(input)
	if err != nil {
		t.Fatalf("ParseGoTestJSON() error = %v", err)
	}
	if got, want := len(report.Results), 1; got != want {
		t.Fatalf("results len = %d, want %d", got, want)
	}
	if report.PlannerMultiTaskRate == nil || *report.PlannerMultiTaskRate != 1.0 {
		t.Fatalf("planner_multi_task_rate = %v", report.PlannerMultiTaskRate)
	}
}

func TestRenderMarkdownIncludesTableAndMermaid(t *testing.T) {
	t.Parallel()

	rate := 1.0
	report := Report{
		Results: []Result{
			{Name: "stop_signal", Success: true, Turns: 1, Tasks: 1, DoneTasks: 1, DurationMS: 4},
			{Name: "multi_task_sequence", Success: true, Turns: 2, Tasks: 2, DoneTasks: 2, DurationMS: 12, MultiTask: true},
		},
		PlannerMultiTaskRate: &rate,
	}

	out := RenderMarkdown(report)
	if !strings.Contains(out, "| stop_signal |") {
		t.Fatalf("markdown missing table row: %q", out)
	}
	if !strings.Contains(out, "```mermaid") {
		t.Fatalf("markdown missing mermaid block: %q", out)
	}
	if !strings.Contains(out, "planner_multi_task_rate") {
		t.Fatalf("markdown missing planner rate: %q", out)
	}
}
