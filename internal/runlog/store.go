package runlog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/farmcan/canx/internal/codex"
	"github.com/farmcan/canx/internal/review"
	"github.com/farmcan/canx/internal/sessions"
	"github.com/farmcan/canx/internal/tasks"
)

type SessionTurn struct {
	Number           int           `json:"number"`
	Summary          string        `json:"summary,omitempty"`
	Output           string        `json:"output,omitempty"`
	ValidationPassed bool          `json:"validation_passed"`
	ValidationOutput string        `json:"validation_output,omitempty"`
	Review           review.Result `json:"review,omitempty"`
	Runtime          codex.Runtime `json:"runtime,omitempty"`
}

type SessionReport struct {
	Session   sessions.Session `json:"session"`
	RunID     string           `json:"run_id,omitempty"`
	Runtime   codex.Runtime    `json:"runtime,omitempty"`
	Decision  string           `json:"decision"`
	Reason    string           `json:"reason"`
	TurnCount int              `json:"turn_count"`
	Turns     []SessionTurn    `json:"turns,omitempty"`
	Tasks     []tasks.Task     `json:"tasks,omitempty"`
	WrittenAt time.Time        `json:"written_at"`
}

func WriteSessionReport(root string, report SessionReport) (string, error) {
	report.WrittenAt = time.Now()

	dir := filepath.Join(root, ".canx", "sessions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	path := filepath.Join(dir, report.Session.ID+".json")
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}

	return path, nil
}
