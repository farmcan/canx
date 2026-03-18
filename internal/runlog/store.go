package runlog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/farmcan/canx/internal/sessions"
)

type SessionReport struct {
	Session   sessions.Session `json:"session"`
	Decision  string           `json:"decision"`
	Reason    string           `json:"reason"`
	TurnCount int              `json:"turn_count"`
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
