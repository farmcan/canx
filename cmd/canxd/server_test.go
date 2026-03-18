package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/farmcan/canx/internal/rooms"
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

	sessionsReq := httptest.NewRequest("GET", "/api/sessions", nil)
	sessionsResp := httptest.NewRecorder()
	mux.ServeHTTP(sessionsResp, sessionsReq)
	if sessionsResp.Code != 200 {
		t.Fatalf("sessions status = %d", sessionsResp.Code)
	}
	var sessionReports []runlog.SessionReport
	if err := json.Unmarshal(sessionsResp.Body.Bytes(), &sessionReports); err != nil {
		t.Fatalf("sessions decode error = %v", err)
	}
	if got, want := len(sessionReports), 1; got != want {
		t.Fatalf("sessions len = %d, want %d", got, want)
	}

	docReq := httptest.NewRequest("GET", "/api/context/docs/docs/one.md", nil)
	docResp := httptest.NewRecorder()
	mux.ServeHTTP(docResp, docReq)
	if docResp.Code != 200 {
		t.Fatalf("doc status = %d", docResp.Code)
	}
	var docPayload map[string]any
	if err := json.Unmarshal(docResp.Body.Bytes(), &docPayload); err != nil {
		t.Fatalf("doc decode error = %v", err)
	}
	if docPayload["content"] != "doc one" {
		t.Fatalf("doc content = %#v", docPayload["content"])
	}
}

func TestServeExposesRoomsAPI(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "README.md"), "readme body")
	store := runlog.NewEventStore(root)
	roomStore := rooms.NewStore(root)
	if err := roomStore.SaveRoom(rooms.Room{
		ID:       "room-1",
		Title:    "Main Room",
		RunID:    "run-1",
		RepoRoot: root,
	}); err != nil {
		t.Fatalf("SaveRoom() error = %v", err)
	}

	mux, err := newServerMux(store)
	if err != nil {
		t.Fatalf("newServerMux() error = %v", err)
	}

	listReq := httptest.NewRequest("GET", "/api/rooms", nil)
	listResp := httptest.NewRecorder()
	mux.ServeHTTP(listResp, listReq)
	if listResp.Code != 200 {
		t.Fatalf("rooms status = %d", listResp.Code)
	}
	var roomList []rooms.Room
	if err := json.Unmarshal(listResp.Body.Bytes(), &roomList); err != nil {
		t.Fatalf("rooms decode error = %v", err)
	}
	if got, want := len(roomList), 1; got != want {
		t.Fatalf("rooms len = %d, want %d", got, want)
	}

	postBody := []byte(`{"participant_id":"human-1","role":"human","kind":"instruction","task_id":"task-1","body":"adjust strategy"}`)
	postReq := httptest.NewRequest("POST", "/api/rooms/room-1/messages", bytes.NewReader(postBody))
	postResp := httptest.NewRecorder()
	mux.ServeHTTP(postResp, postReq)
	if postResp.Code != 200 {
		t.Fatalf("room post status = %d body=%s", postResp.Code, postResp.Body.String())
	}

	msgReq := httptest.NewRequest("GET", "/api/rooms/room-1/messages", nil)
	msgResp := httptest.NewRecorder()
	mux.ServeHTTP(msgResp, msgReq)
	if msgResp.Code != 200 {
		t.Fatalf("room messages status = %d", msgResp.Code)
	}
	var messages []rooms.Message
	if err := json.Unmarshal(msgResp.Body.Bytes(), &messages); err != nil {
		t.Fatalf("messages decode error = %v", err)
	}
	if got, want := len(messages), 1; got != want {
		t.Fatalf("messages len = %d, want %d", got, want)
	}
	if messages[0].Body != "adjust strategy" {
		t.Fatalf("message body = %q", messages[0].Body)
	}
	if messages[0].TaskID != "task-1" {
		t.Fatalf("message task_id = %q", messages[0].TaskID)
	}
}

func TestServeExposesSSEEvents(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := runlog.NewEventStore(root)
	if err := store.SaveRun(runlog.RunRecord{ID: "run-1", Goal: "ship", RepoRoot: root, Status: "done"}); err != nil {
		t.Fatalf("SaveRun() error = %v", err)
	}
	if err := store.AppendEvent("run-1", runlog.Event{Kind: "run_finished", Message: "done"}); err != nil {
		t.Fatalf("AppendEvent() error = %v", err)
	}

	mux, err := newServerMux(store)
	if err != nil {
		t.Fatalf("newServerMux() error = %v", err)
	}

	req := httptest.NewRequest("GET", "/api/runs/run-1/events/stream", nil)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	if resp.Code != 200 {
		t.Fatalf("sse status = %d", resp.Code)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if !bytes.Contains(body, []byte("event: run_finished")) {
		t.Fatalf("unexpected sse body: %s", string(body))
	}
}
