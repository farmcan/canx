package review

import (
	"encoding/json"
	"strings"
)

type Result struct {
	Validated bool
	Approved  bool
	Reason    string
	Warnings  []string
}

type verdict struct {
	Approved *bool    `json:"approved"`
	Reason   string   `json:"reason"`
	Warnings []string `json:"warnings"`
}

func Evaluate(result Result) Result {
	if !result.Validated {
		result.Approved = false
		result.Reason = "missing validation"
		return result
	}
	result.Approved = true
	result.Reason = "approved"
	return result
}

func ParseVerdict(output string) (Result, bool) {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return Result{}, false
	}
	trimmed = extractJSONBlock(trimmed)
	var parsed verdict
	if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
		return Result{}, false
	}
	if parsed.Approved == nil {
		return Result{}, false
	}
	return Result{Approved: *parsed.Approved, Reason: parsed.Reason, Warnings: parsed.Warnings}, true
}

func extractJSONBlock(output string) string {
	output = strings.TrimSpace(output)
	if strings.HasPrefix(output, "```") {
		lines := strings.Split(output, "\n")
		if len(lines) >= 3 {
			return strings.Join(lines[1:len(lines)-1], "\n")
		}
	}
	return output
}
