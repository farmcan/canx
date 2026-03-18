package runlog

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/farmcan/canx/internal/sessions"
	"github.com/farmcan/canx/internal/tasks"
)

func TestWriteSessionReportPersistsJSONFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	report := SessionReport{
		Session: sessions.Session{
			ID:          "session-1",
			Label:       "main",
			LastSummary: "done",
		},
		Decision: "stop",
		Tasks: []tasks.Task{
			{ID: "task-1", Goal: "ship mvp", Status: tasks.StatusDone},
		},
	}

	path, err := WriteSessionReport(dir, report)
	if err != nil {
		t.Fatalf("WriteSessionReport() error = %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("Stat(%s) error = %v", path, err)
	}
	if filepath.Base(path) != "session-1.json" {
		t.Fatalf("unexpected filename %q", filepath.Base(path))
	}
}
