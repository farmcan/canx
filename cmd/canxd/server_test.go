package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestSelectPreferredFrontstageRunPrefersPlayableStoppedRun(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := runlog.NewEventStore(root)
	running := runlog.RunRecord{ID: "run-running", Goal: "running", RepoRoot: root, Status: "running"}
	stopped := runlog.RunRecord{ID: "run-stopped", Goal: "stopped", RepoRoot: root, Status: "stop"}
	if err := store.SaveRun(running); err != nil {
		t.Fatalf("SaveRun(running) error = %v", err)
	}
	if err := store.SaveRun(stopped); err != nil {
		t.Fatalf("SaveRun(stopped) error = %v", err)
	}
	if err := store.AppendEvent("run-running", runlog.Event{Kind: "run_started"}); err != nil {
		t.Fatalf("AppendEvent(running) error = %v", err)
	}
	if err := store.AppendEvent("run-stopped", runlog.Event{Kind: "run_started"}); err != nil {
		t.Fatalf("AppendEvent(stopped start) error = %v", err)
	}
	if err := store.AppendEvent("run-stopped", runlog.Event{Kind: "turn_completed"}); err != nil {
		t.Fatalf("AppendEvent(stopped turn) error = %v", err)
	}
	if err := store.AppendEvent("run-stopped", runlog.Event{Kind: "run_finished"}); err != nil {
		t.Fatalf("AppendEvent(stopped finish) error = %v", err)
	}

	selected, err := selectPreferredFrontstageRun(store, []runlog.RunRecord{running, stopped})
	if err != nil {
		t.Fatalf("selectPreferredFrontstageRun() error = %v", err)
	}
	if selected.ID != "run-stopped" {
		t.Fatalf("selected run = %q", selected.ID)
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
	if err := os.MkdirAll(filepath.Join(root, "docs", "superpowers", "specs"), 0o755); err != nil {
		t.Fatalf("MkdirAll(specs) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs", "superpowers", "plans"), 0o755); err != nil {
		t.Fatalf("MkdirAll(plans) error = %v", err)
	}
	writeFile(t, filepath.Join(root, "docs", "superpowers", "specs", "2026-03-19-spec.md"), "latest spec")
	writeFile(t, filepath.Join(root, "docs", "superpowers", "plans", "2026-03-19-plan.md"), "latest plan")

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
	if contextPayload["latest_spec_path"] != "docs/superpowers/specs/2026-03-19-spec.md" {
		t.Fatalf("latest_spec_path = %#v", contextPayload["latest_spec_path"])
	}
	if contextPayload["latest_plan_path"] != "docs/superpowers/plans/2026-03-19-plan.md" {
		t.Fatalf("latest_plan_path = %#v", contextPayload["latest_plan_path"])
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

	presentationReq := httptest.NewRequest("GET", "/api/runs/run-1/presentation", nil)
	presentationResp := httptest.NewRecorder()
	mux.ServeHTTP(presentationResp, presentationReq)
	if presentationResp.Code != 200 {
		t.Fatalf("presentation status = %d body=%s", presentationResp.Code, presentationResp.Body.String())
	}
	var presentation map[string]any
	if err := json.Unmarshal(presentationResp.Body.Bytes(), &presentation); err != nil {
		t.Fatalf("presentation decode error = %v", err)
	}
	if presentation["phase"] != "done" {
		t.Fatalf("presentation phase = %#v", presentation["phase"])
	}
	if presentation["scene_zone"] != "sync_port" {
		t.Fatalf("presentation scene_zone = %#v", presentation["scene_zone"])
	}
	if presentation["display_status"] == "" {
		t.Fatal("expected display_status in presentation payload")
	}

	beatsReq := httptest.NewRequest("GET", "/api/runs/run-1/beats", nil)
	beatsResp := httptest.NewRecorder()
	mux.ServeHTTP(beatsResp, beatsReq)
	if beatsResp.Code != 200 {
		t.Fatalf("beats status = %d body=%s", beatsResp.Code, beatsResp.Body.String())
	}
	var beats []map[string]any
	if err := json.Unmarshal(beatsResp.Body.Bytes(), &beats); err != nil {
		t.Fatalf("beats decode error = %v", err)
	}
	if got, want := len(beats), 1; got != want {
		t.Fatalf("beats len = %d, want %d", got, want)
	}
	if beats[0]["type"] != "complete" {
		t.Fatalf("beat type = %#v", beats[0]["type"])
	}

	latestReq := httptest.NewRequest("GET", "/api/frontstage/latest", nil)
	latestResp := httptest.NewRecorder()
	mux.ServeHTTP(latestResp, latestReq)
	if latestResp.Code != 200 {
		t.Fatalf("frontstage latest status = %d body=%s", latestResp.Code, latestResp.Body.String())
	}
	var latest frontstagePayload
	if err := json.Unmarshal(latestResp.Body.Bytes(), &latest); err != nil {
		t.Fatalf("frontstage latest decode error = %v", err)
	}
	if latest.Run.ID != "run-1" {
		t.Fatalf("frontstage latest run id = %q", latest.Run.ID)
	}
	if got, want := len(latest.Beats), 1; got != want {
		t.Fatalf("frontstage latest beats len = %d, want %d", got, want)
	}
	if got, want := len(latest.Timeline), 1; got != want {
		t.Fatalf("frontstage latest timeline len = %d, want %d", got, want)
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

	ts := httptest.NewServer(mux)
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/api/runs/run-1/events/stream", nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext() error = %v", err)
	}
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("sse status = %d", resp.StatusCode)
	}

	buffer := make([]byte, 512)
	n, err := resp.Body.Read(buffer)
	if err != nil && err != io.EOF {
		t.Fatalf("Read() error = %v", err)
	}
	cancel()
	body := buffer[:n]
	if !bytes.Contains(body, []byte("event: run_finished")) {
		t.Fatalf("unexpected sse body: %s", string(body))
	}
}

func TestServeStreamsFutureSSEEvents(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := runlog.NewEventStore(root)
	if err := store.SaveRun(runlog.RunRecord{ID: "run-1", Goal: "ship", RepoRoot: root, Status: "running"}); err != nil {
		t.Fatalf("SaveRun() error = %v", err)
	}
	if err := store.AppendEvent("run-1", runlog.Event{Kind: "run_started", Message: "start"}); err != nil {
		t.Fatalf("AppendEvent() error = %v", err)
	}

	mux, err := newServerMux(store)
	if err != nil {
		t.Fatalf("newServerMux() error = %v", err)
	}

	ts := httptest.NewServer(mux)
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/api/runs/run-1/events/stream", nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext() error = %v", err)
	}
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()

	received := make(chan string, 1)
	go func() {
		data, _ := io.ReadAll(resp.Body)
		received <- string(data)
	}()

	time.Sleep(100 * time.Millisecond)
	if err := store.AppendEvent("run-1", runlog.Event{Kind: "turn_completed", Message: "later"}); err != nil {
		t.Fatalf("AppendEvent(later) error = %v", err)
	}

	var body string
	select {
	case body = <-received:
	case <-time.After(500 * time.Millisecond):
		cancel()
		body = <-received
	}

	if !strings.Contains(body, "event: turn_completed") {
		t.Fatalf("expected streamed future event, got: %s", body)
	}
	cancel()
}

func TestRoomInstructionCreatesActionRecord(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := runlog.NewEventStore(root)
	if err := store.SaveRun(runlog.RunRecord{
		ID:       "run-1",
		Goal:     "ship ui",
		RepoRoot: root,
		Status:   "running",
		Tasks:    []tasks.Task{{ID: "task-1", Title: "Task 1", Goal: "ship ui", Status: tasks.StatusPending}},
	}); err != nil {
		t.Fatalf("SaveRun() error = %v", err)
	}
	roomStore := rooms.NewStore(root)
	if err := roomStore.SaveRoom(rooms.Room{ID: "room-1", Title: "Main Room", RunID: "run-1", RepoRoot: root}); err != nil {
		t.Fatalf("SaveRoom() error = %v", err)
	}

	mux, err := newServerMux(store)
	if err != nil {
		t.Fatalf("newServerMux() error = %v", err)
	}

	postBody := []byte(`{"participant_id":"human-1","role":"human","kind":"instruction","task_id":"task-1","body":"mark blocked"}`)
	postReq := httptest.NewRequest("POST", "/api/rooms/room-1/messages", bytes.NewReader(postBody))
	postResp := httptest.NewRecorder()
	mux.ServeHTTP(postResp, postReq)
	if postResp.Code != 200 {
		t.Fatalf("room post status = %d body=%s", postResp.Code, postResp.Body.String())
	}

	actionReq := httptest.NewRequest("GET", "/api/runs/run-1/actions", nil)
	actionResp := httptest.NewRecorder()
	mux.ServeHTTP(actionResp, actionReq)
	if actionResp.Code != 200 {
		t.Fatalf("actions status = %d", actionResp.Code)
	}
	var actions []map[string]any
	if err := json.Unmarshal(actionResp.Body.Bytes(), &actions); err != nil {
		t.Fatalf("actions decode error = %v", err)
	}
	if got, want := len(actions), 1; got != want {
		t.Fatalf("actions len = %d, want %d", got, want)
	}
	if actions[0]["task_id"] != "task-1" {
		t.Fatalf("action task_id = %#v", actions[0]["task_id"])
	}
}
