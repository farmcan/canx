package runlog

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/farmcan/canx/internal/sessions"
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
