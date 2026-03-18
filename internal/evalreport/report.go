package evalreport

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type Result struct {
	Name       string `json:"name"`
	Success    bool   `json:"success"`
	Decision   string `json:"decision,omitempty"`
	Reason     string `json:"reason,omitempty"`
	Turns      int    `json:"turns"`
	Tasks      int    `json:"tasks"`
	DoneTasks  int    `json:"done_tasks"`
	DurationMS int64  `json:"duration_ms"`
	PromptDocs int    `json:"prompt_docs,omitempty"`
	MultiTask  bool   `json:"multi_task"`
}

type Report struct {
	Results              []Result `json:"results"`
	PlannerMultiTaskRate *float64 `json:"planner_multi_task_rate,omitempty"`
}

type goTestEvent struct {
	Action string `json:"Action"`
	Output string `json:"Output"`
}

func ParseGoTestJSON(input io.Reader) (Report, error) {
	var report Report
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		var event goTestEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			continue
		}
		if event.Action != "output" {
			continue
		}
		for _, line := range strings.Split(event.Output, "\n") {
			if result, ok := parseResultLine(line); ok {
				report.Results = append(report.Results, result)
				continue
			}
			if rate, ok := parsePlannerRate(line); ok {
				report.PlannerMultiTaskRate = &rate
			}
		}
	}
	return report, scanner.Err()
}

func RenderMarkdown(report Report) string {
	var builder strings.Builder
	builder.WriteString("# CanX Eval Report\n\n")
	if report.PlannerMultiTaskRate != nil {
		builder.WriteString("- `planner_multi_task_rate`: ")
		builder.WriteString(fmt.Sprintf("%.2f", *report.PlannerMultiTaskRate))
		builder.WriteString("\n")
	}
	builder.WriteString("\n| Case | Success | Turns | Tasks | Done | Duration (ms) |\n")
	builder.WriteString("| --- | --- | ---: | ---: | ---: | ---: |\n")
	for _, result := range report.Results {
		builder.WriteString("| ")
		builder.WriteString(result.Name)
		builder.WriteString(" | ")
		builder.WriteString(strconv.FormatBool(result.Success))
		builder.WriteString(" | ")
		builder.WriteString(strconv.Itoa(result.Turns))
		builder.WriteString(" | ")
		builder.WriteString(strconv.Itoa(result.Tasks))
		builder.WriteString(" | ")
		builder.WriteString(strconv.Itoa(result.DoneTasks))
		builder.WriteString(" | ")
		builder.WriteString(strconv.FormatInt(result.DurationMS, 10))
		builder.WriteString(" |\n")
	}
	builder.WriteString("\n```mermaid\nxychart-beta\n")
	builder.WriteString(`title "CanX Eval Durations (ms)"` + "\n")
	builder.WriteString("x-axis [")
	for index, result := range report.Results {
		if index > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString("\"")
		builder.WriteString(result.Name)
		builder.WriteString("\"")
	}
	builder.WriteString("]\n")
	builder.WriteString("y-axis \"ms\" 0 --> ")
	builder.WriteString(strconv.FormatInt(maxDuration(report.Results), 10))
	builder.WriteString("\nbar [")
	for index, result := range report.Results {
		if index > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(strconv.FormatInt(result.DurationMS, 10))
	}
	builder.WriteString("]\n```\n")
	return builder.String()
}

func parseResultLine(line string) (Result, bool) {
	start := strings.Index(line, "{")
	end := strings.LastIndex(line, "}")
	if start == -1 || end == -1 || end <= start {
		return Result{}, false
	}

	var result Result
	if err := json.Unmarshal([]byte(line[start:end+1]), &result); err != nil {
		return Result{}, false
	}
	if result.Name == "" {
		return Result{}, false
	}
	return result, true
}

func parsePlannerRate(line string) (float64, bool) {
	const prefix = "planner_multi_task_rate="
	index := strings.Index(line, prefix)
	if index == -1 {
		return 0, false
	}
	value := strings.TrimSpace(line[index+len(prefix):])
	rate, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, false
	}
	return rate, true
}

func maxDuration(results []Result) int64 {
	var max int64 = 1
	for _, result := range results {
		if result.DurationMS > max {
			max = result.DurationMS
		}
	}
	return max
}
