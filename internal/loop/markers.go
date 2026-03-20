package loop

import (
	"encoding/json"
	"strings"
)

type spawnRequest struct {
	Title        string   `json:"title"`
	Goal         string   `json:"goal"`
	Reason       string   `json:"reason"`
	PlannedFiles []string `json:"planned_files"`
}

func hasStopSignal(output string) bool {
	return strings.Contains(output, stopMarker) || strings.Contains(output, "[canx:stop:")
}

func hasEscalateSignal(output string) bool {
	return strings.Contains(output, escalateMarker)
}

func parseStopPayload(output string) *stopPayload {
	body, ok := parseMarkerPayload(output, "[canx:stop:")
	if !ok {
		return nil
	}
	var payload stopPayload
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		return nil
	}
	return &payload
}

func parseSpawnRequests(output string) []spawnRequest {
	bodies := parseMarkerPayloads(output, "[canx:spawn:")
	if len(bodies) == 0 {
		return nil
	}
	requests := make([]spawnRequest, 0, len(bodies))
	for _, body := range bodies {
		var request spawnRequest
		if err := json.Unmarshal([]byte(body), &request); err != nil {
			continue
		}
		if strings.TrimSpace(request.Title) == "" || strings.TrimSpace(request.Goal) == "" {
			continue
		}
		requests = append(requests, request)
	}
	return requests
}

func parseMarkerPayload(output, prefix string) (string, bool) {
	bodies := parseMarkerPayloads(output, prefix)
	if len(bodies) == 0 {
		return "", false
	}
	return bodies[0], true
}

func parseMarkerPayloads(output, prefix string) []string {
	results := []string{}
	start := 0
	for {
		index := strings.Index(output[start:], prefix)
		if index == -1 {
			return results
		}
		index += start
		bodyStart := index + len(prefix)
		bodyEnd, ok := findJSONMarkerEnd(output, bodyStart)
		if !ok {
			return results
		}
		results = append(results, output[bodyStart:bodyEnd])
		start = bodyEnd + 1
	}
}

func findJSONMarkerEnd(input string, start int) (int, bool) {
	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(input); i++ {
		ch := input[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			switch ch {
			case '\\':
				escaped = true
			case '"':
				inString = false
			}
			continue
		}
		switch ch {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			if depth > 0 {
				depth--
			}
			if depth == 0 {
				if i+1 < len(input) && input[i+1] == ']' {
					return i + 1, true
				}
				return 0, false
			}
		}
	}
	return 0, false
}
