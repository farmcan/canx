package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"net/http/httptest"
	"testing"

	"github.com/farmcan/canx/internal/runlog"
	"github.com/farmcan/canx/internal/sessions"
	"github.com/farmcan/canx/internal/tasks"
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

func TestServeExposesTaskSessionAndContextAPI(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "README.md"), "readme body")
	writeFile(t, filepath.Join(root, "AGENTS.md"), "agents body")
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatalf("MkdirAll(docs) error = %v", err)
	}
	writeFile(t, filepath.Join(root, "docs", "one.md"), "doc one")

	store := runlog.NewEventStore(root)
	task := tasks.Task{ID: "task-1", Title: "Task 1", Goal: "ship ui", Status: tasks.StatusDone}
	if err := store.SaveRun(runlog.RunRecord{
		ID:       "run-1",
		Goal:     "ship ui",
		RepoRoot: root,
		Status:   "stop",
		Tasks:    []tasks.Task{task},
	}); err != nil {
		t.Fatalf("SaveRun() error = %v", err)
	}

	if _, err := runlog.WriteSessionReport(root, runlog.SessionReport{
		RunID: "run-1",
		Session: sessions.Session{
			ID:          "session-1",
			Label:       "main",
			LastSummary: "done",
		},
		Decision: "stop",
		Tasks:    []tasks.Task{task},
	}); err != nil {
		t.Fatalf("WriteSessionReport() error = %v", err)
	}

	mux, err := newServerMux(store)
	if err != nil {
		t.Fatalf("newServerMux() error = %v", err)
	}

	taskReq := httptest.NewRequest("GET", "/api/runs/run-1/tasks/task-1", nil)
	taskResp := httptest.NewRecorder()
	mux.ServeHTTP(taskResp, taskReq)
	if taskResp.Code != 200 {
		t.Fatalf("task status = %d", taskResp.Code)
	}
	var gotTask tasks.Task
	if err := json.Unmarshal(taskResp.Body.Bytes(), &gotTask); err != nil {
		t.Fatalf("task decode error = %v", err)
	}
	if gotTask.ID != "task-1" {
		t.Fatalf("task id = %q", gotTask.ID)
	}

	sessionReq := httptest.NewRequest("GET", "/api/sessions/session-1", nil)
	sessionResp := httptest.NewRecorder()
	mux.ServeHTTP(sessionResp, sessionReq)
	if sessionResp.Code != 200 {
		t.Fatalf("session status = %d", sessionResp.Code)
	}
	var report runlog.SessionReport
	if err := json.Unmarshal(sessionResp.Body.Bytes(), &report); err != nil {
		t.Fatalf("session decode error = %v", err)
	}
	if report.Session.ID != "session-1" {
		t.Fatalf("session id = %q", report.Session.ID)
	}

	contextReq := httptest.NewRequest("GET", "/api/context", nil)
	contextResp := httptest.NewRecorder()
	mux.ServeHTTP(contextResp, contextReq)
	if contextResp.Code != 200 {
		t.Fatalf("context status = %d", contextResp.Code)
	}
	var contextPayload map[string]any
	if err := json.Unmarshal(contextResp.Body.Bytes(), &contextPayload); err != nil {
		t.Fatalf("context decode error = %v", err)
	}
	if contextPayload["readme"] == "" {
		t.Fatal("expected readme in context payload")
	}
}
