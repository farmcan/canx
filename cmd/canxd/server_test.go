package main

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/farmcan/canx/internal/runlog"
)

func TestServeExposesRunAPI(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := runlog.NewEventStore(root)
	if err := store.SaveRun(runlog.RunRecord{ID: "run-1", Goal: "ship", RepoRoot: root, Status: "done"}); err != nil {
		t.Fatalf("SaveRun() error = %v", err)
	}
	if err := store.AppendEvent("run-1", runlog.Event{Kind: "run_finished"}); err != nil {
		t.Fatalf("AppendEvent() error = %v", err)
	}

	mux, err := newServerMux(store)
	if err != nil {
		t.Fatalf("newServerMux() error = %v", err)
	}

	req := httptest.NewRequest("GET", "/api/runs", nil)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)

	var runs []runlog.RunRecord
	if err := json.Unmarshal(resp.Body.Bytes(), &runs); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got, want := len(runs), 1; got != want {
		t.Fatalf("runs len = %d, want %d", got, want)
	}
}
