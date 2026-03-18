package runlog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/farmcan/canx/internal/codex"
	"github.com/farmcan/canx/internal/sessions"
	"github.com/farmcan/canx/internal/tasks"
)

type SessionReport struct {
	Session   sessions.Session `json:"session"`
	Runtime   codex.Runtime    `json:"runtime,omitempty"`
	Decision  string           `json:"decision"`
	Reason    string           `json:"reason"`
	TurnCount int              `json:"turn_count"`
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
