package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/farmcan/canx/internal/rooms"
	"github.com/farmcan/canx/internal/runlog"
	"github.com/farmcan/canx/internal/tasks"
	"github.com/farmcan/canx/internal/workspace"
)

type runPresentation struct {
	Phase         string `json:"phase"`
	SceneZone     string `json:"scene_zone"`
	DisplayStatus string `json:"display_status"`
	ActorRole     string `json:"actor_role"`
}

type runBeat struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title"`
	Summary   string `json:"summary"`
	Zone      string `json:"zone"`
	ActorRole string `json:"actor_role"`
	CreatedAt string `json:"created_at,omitempty"`
}

type frontstagePayload struct {
	Run          runlog.RunRecord      `json:"run"`
	Presentation runPresentation       `json:"presentation"`
	Beats        []runBeat             `json:"beats"`
	Timeline     []runBeat             `json:"timeline"`
	Actions      []runlog.ActionRecord `json:"actions"`
	Messages     []rooms.Message       `json:"messages"`
}

//go:embed ui/*
var uiFiles embed.FS

func serve(opts Options) (string, error) {
	repoPath := opts.RepoPath
	if repoPath == "" {
		repoPath = "."
	}
	root, err := filepath.Abs(repoPath)
	if err != nil {
		return "", err
	}

	store := runlog.NewEventStore(root)
	mux, err := newServerMux(store)
	if err != nil {
		return "", err
	}

	addr := "127.0.0.1:8090"
	fmt.Printf("canx dashboard listening on http://%s\n", addr)
	return "", http.ListenAndServe(addr, mux)
}

func newServerMux(store runlog.EventStore) (*http.ServeMux, error) {
	roomStore := rooms.NewStore(store.Root)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/runs", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/runs" {
			http.NotFound(w, r)
			return
		}
		runs, err := store.ListRuns()
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, runs)
	})
	mux.HandleFunc("/api/frontstage/latest", func(w http.ResponseWriter, r *http.Request) {
		runs, err := store.ListRuns()
		if err != nil {
			writeError(w, err)
			return
		}
		if len(runs) == 0 {
			writeJSON(w, frontstagePayload{})
			return
		}
		selected, err := selectPreferredFrontstageRun(store, runs)
		if err != nil {
			writeError(w, err)
			return
		}
		payload, err := buildFrontstagePayload(store, roomStore, selected)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, payload)
	})
	mux.HandleFunc("/api/runs/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/runs/")
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) == 0 || parts[0] == "" {
			http.NotFound(w, r)
			return
		}
		runID := parts[0]
		switch {
		case len(parts) == 1:
			record, err := store.LoadRun(runID)
			if err != nil {
				writeError(w, err)
				return
			}
			writeJSON(w, record)
		case len(parts) == 2 && parts[1] == "presentation":
			record, err := store.LoadRun(runID)
			if err != nil {
				writeError(w, err)
				return
			}
			writeJSON(w, deriveRunPresentation(record))
		case len(parts) == 2 && parts[1] == "beats":
			record, err := store.LoadRun(runID)
			if err != nil {
				writeError(w, err)
				return
			}
			events, err := store.LoadEvents(runID)
			if err != nil {
				writeError(w, err)
				return
			}
			writeJSON(w, deriveRunBeats(record, events))
		case len(parts) == 3 && parts[1] == "tasks":
			record, err := store.LoadRun(runID)
			if err != nil {
				writeError(w, err)
				return
			}
			task, ok := findTask(record.Tasks, parts[2])
			if !ok {
				http.NotFound(w, r)
				return
			}
			writeJSON(w, task)
		case len(parts) == 2 && parts[1] == "events":
			events, err := store.LoadEvents(runID)
			if err != nil {
				writeError(w, err)
				return
			}
			writeJSON(w, events)
		case len(parts) == 2 && parts[1] == "actions":
			actions, err := store.ListActions(runID)
			if err != nil {
				writeError(w, err)
				return
			}
			writeJSON(w, actions)
		case len(parts) == 3 && parts[1] == "events" && parts[2] == "stream":
			if err := streamRunEvents(w, r, store, runID); err != nil {
				writeError(w, err)
				return
			}
		default:
			http.NotFound(w, r)
		}
	})
	mux.HandleFunc("/api/sessions/", func(w http.ResponseWriter, r *http.Request) {
		sessionID := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/sessions/"), "/")
		if sessionID == "" {
			http.NotFound(w, r)
			return
		}
		root := store.Root
		data, err := os.ReadFile(filepath.Join(root, ".canx", "sessions", sessionID+".json"))
		if err != nil {
			writeError(w, err)
			return
		}
		var report runlog.SessionReport
		if err := json.Unmarshal(data, &report); err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, report)
	})
	mux.HandleFunc("/api/sessions", func(w http.ResponseWriter, r *http.Request) {
		reports, err := listSessionReports(store.Root)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, reports)
	})
	mux.HandleFunc("/api/context", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := workspace.Load(store.Root)
		if err != nil {
			writeError(w, err)
			return
		}
		latestSpecPath := latestDocPath(store.Root, filepath.Join("docs", "superpowers", "specs"))
		latestPlanPath := latestDocPath(store.Root, filepath.Join("docs", "superpowers", "plans"))
		writeJSON(w, map[string]any{
			"root":             ctx.Root,
			"readme":           ctx.Readme,
			"agents":           ctx.Agents,
			"docs":             ctx.Docs,
			"latest_spec_path": latestSpecPath,
			"latest_plan_path": latestPlanPath,
		})
	})
	mux.HandleFunc("/api/context/docs/", func(w http.ResponseWriter, r *http.Request) {
		relPath := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/context/docs/"), "/")
		if relPath == "" {
			http.NotFound(w, r)
			return
		}
		data, err := os.ReadFile(filepath.Join(store.Root, relPath))
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, map[string]any{
			"path":    relPath,
			"content": string(data),
		})
	})
	mux.HandleFunc("/api/rooms", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/rooms" {
			http.NotFound(w, r)
			return
		}
		items, err := roomStore.ListRooms()
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, items)
	})
	mux.HandleFunc("/api/rooms/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/rooms/")
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) == 0 || parts[0] == "" {
			http.NotFound(w, r)
			return
		}
		roomID := parts[0]
		switch {
		case len(parts) == 1:
			room, err := roomStore.LoadRoom(roomID)
			if err != nil {
				writeError(w, err)
				return
			}
			writeJSON(w, room)
		case len(parts) == 2 && parts[1] == "messages" && r.Method == http.MethodGet:
			messages, err := roomStore.ListMessages(roomID)
			if err != nil {
				writeError(w, err)
				return
			}
			writeJSON(w, messages)
		case len(parts) == 2 && parts[1] == "messages" && r.Method == http.MethodPost:
			var message rooms.Message
			if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
				writeError(w, err)
				return
			}
			stored, err := roomStore.AppendMessage(roomID, message)
			if err != nil {
				writeError(w, err)
				return
			}
			room, err := roomStore.LoadRoom(roomID)
			if err == nil && room.RunID != "" && message.Kind == "instruction" {
				_ = store.AppendAction(room.RunID, runlog.ActionRecord{
					RunID:         room.RunID,
					RoomID:        roomID,
					TaskID:        message.TaskID,
					ParticipantID: message.ParticipantID,
					Role:          message.Role,
					Kind:          message.Kind,
					Body:          message.Body,
				})
			}
			writeJSON(w, stored)
		default:
			http.NotFound(w, r)
		}
	})

	staticRoot, err := fs.Sub(uiFiles, "ui")
	if err != nil {
		return nil, err
	}
	mux.Handle("/", http.FileServer(http.FS(staticRoot)))
	return mux, nil
}

func selectPreferredFrontstageRun(store runlog.EventStore, runs []runlog.RunRecord) (runlog.RunRecord, error) {
	if len(runs) == 0 {
		return runlog.RunRecord{}, nil
	}
	selected := runs[0]
	bestScore := -1
	for _, run := range runs {
		score := 0
		events, err := store.LoadEvents(run.ID)
		if err != nil {
			return runlog.RunRecord{}, err
		}
		score += len(events)
		if run.Status == "stop" {
			score += 10
		}
		if run.Status == "running" {
			score -= 2
		}
		if score > bestScore {
			selected = run
			bestScore = score
		}
	}
	return selected, nil
}

func buildFrontstagePayload(store runlog.EventStore, roomStore rooms.Store, record runlog.RunRecord) (frontstagePayload, error) {
	events, err := store.LoadEvents(record.ID)
	if err != nil {
		return frontstagePayload{}, err
	}
	actions, err := store.ListActions(record.ID)
	if err != nil {
		return frontstagePayload{}, err
	}
	roomItems, err := roomStore.ListRooms()
	if err != nil {
		return frontstagePayload{}, err
	}
	var latestMessages []rooms.Message
	for _, room := range roomItems {
		if room.RunID != record.ID {
			continue
		}
		messages, err := roomStore.ListMessages(room.ID)
		if err != nil {
			return frontstagePayload{}, err
		}
		if len(messages) > len(latestMessages) {
			latestMessages = messages
		}
	}
	return frontstagePayload{
		Run:          record,
		Presentation: deriveRunPresentation(record),
		Beats:        deriveRunBeats(record, events),
		Timeline:     deriveFrontstageTimeline(record, events, actions, latestMessages),
		Actions:      actions,
		Messages:     latestMessages,
	}, nil
}

func listSessionReports(root string) ([]runlog.SessionReport, error) {
	dir := filepath.Join(root, ".canx", "sessions")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var reports []runlog.SessionReport
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		var report runlog.SessionReport
		if err := json.Unmarshal(data, &report); err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}
	return reports, nil
}

func latestDocPath(root, relDir string) string {
	dir := filepath.Join(root, relDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	if len(names) == 0 {
		return ""
	}
	return filepath.ToSlash(filepath.Join(relDir, names[len(names)-1]))
}

func findTask(items []tasks.Task, taskID string) (tasks.Task, bool) {
	for _, item := range items {
		if item.ID == taskID {
			return item, true
		}
	}
	return tasks.Task{}, false
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	writeJSON(w, map[string]string{"error": err.Error()})
}

func deriveRunPresentation(record runlog.RunRecord) runPresentation {
	phase := "planning"
	sceneZone := "command_deck"
	displayStatus := strings.TrimSpace(record.Goal)
	actorRole := "supervisor"

	var activeTask *tasks.Task
	for index := range record.Tasks {
		task := &record.Tasks[index]
		switch task.Status {
		case tasks.StatusBlocked:
			activeTask = task
			phase = "blocked"
			sceneZone = "incident_zone"
			displayStatus = fmt.Sprintf("Blocked: %s", firstNonEmpty(task.Title, task.Goal, record.Goal))
			actorRole = "worker"
			return runPresentation{
				Phase:         phase,
				SceneZone:     sceneZone,
				DisplayStatus: displayStatus,
				ActorRole:     actorRole,
			}
		case tasks.StatusInProgress:
			activeTask = task
		}
	}

	if activeTask != nil {
		phase = "working"
		sceneZone = "workbench"
		actorRole = "worker"
		displayStatus = fmt.Sprintf("Working: %s", firstNonEmpty(activeTask.Title, activeTask.Goal, record.Goal))
	}

	if record.Status == "stop" || allTasksDone(record.Tasks) {
		phase = "done"
		sceneZone = "sync_port"
		actorRole = "supervisor"
		displayStatus = fmt.Sprintf("Completed: %s", firstNonEmpty(record.Reason, record.Goal, "run complete"))
	}

	if record.Status == "running" && activeTask == nil && len(record.Tasks) > 0 {
		phase = "planning"
		sceneZone = "command_deck"
		displayStatus = fmt.Sprintf("Planning: %s", firstNonEmpty(record.Tasks[0].Title, record.Goal, "starting run"))
	}

	return runPresentation{
		Phase:         phase,
		SceneZone:     sceneZone,
		DisplayStatus: displayStatus,
		ActorRole:     actorRole,
	}
}

func allTasksDone(items []tasks.Task) bool {
	if len(items) == 0 {
		return false
	}
	for _, item := range items {
		if item.Status != tasks.StatusDone {
			return false
		}
	}
	return true
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func deriveRunBeats(record runlog.RunRecord, events []runlog.Event) []runBeat {
	if len(events) == 0 {
		presentation := deriveRunPresentation(record)
		return []runBeat{{
			ID:        record.ID + "-beat-0",
			Type:      beatTypeFromPresentation(presentation),
			Title:     firstNonEmpty(record.Goal, record.ID),
			Summary:   presentation.DisplayStatus,
			Zone:      presentation.SceneZone,
			ActorRole: presentation.ActorRole,
		}}
	}

	beats := make([]runBeat, 0, len(events))
	for index, event := range events {
		beatType, zone, actorRole := classifyEventBeat(event)
		title := firstNonEmpty(event.Kind, record.Goal, record.ID)
		summary := firstNonEmpty(event.Message, event.Reason, event.Decision, title)
		beats = append(beats, runBeat{
			ID:        fmt.Sprintf("%s-beat-%d", record.ID, index),
			Type:      beatType,
			Title:     title,
			Summary:   summary,
			Zone:      zone,
			ActorRole: actorRole,
			CreatedAt: event.Timestamp.Format(time.RFC3339Nano),
		})
	}
	return beats
}

func beatTypeFromPresentation(presentation runPresentation) string {
	switch presentation.Phase {
	case "working":
		return "tool_use"
	case "blocked":
		return "incident"
	case "done":
		return "complete"
	default:
		return "briefing"
	}
}

func classifyEventBeat(event runlog.Event) (beatType string, zone string, actorRole string) {
	switch event.Kind {
	case "run_started", "session_started":
		return "briefing", "command_deck", "supervisor"
	case "task_state":
		for _, task := range event.Tasks {
			switch task.Status {
			case tasks.StatusBlocked:
				return "incident", "incident_zone", "worker"
			case tasks.StatusInProgress:
				return "tool_use", "workbench", "worker"
			case tasks.StatusDone:
				return "handoff", "sync_port", "worker"
			}
		}
		return "briefing", "command_deck", "supervisor"
	case "turn_completed":
		if event.Validated {
			return "inspect", "test_lab", "worker"
		}
		if strings.TrimSpace(event.Validation) != "" {
			return "incident", "incident_zone", "worker"
		}
		return "tool_use", "workbench", "worker"
	case "run_finished":
		if strings.EqualFold(event.Decision, "stop") {
			return "complete", "sync_port", "supervisor"
		}
		return "incident", "incident_zone", "supervisor"
	default:
		return "briefing", "command_deck", "supervisor"
	}
}

func deriveFrontstageTimeline(record runlog.RunRecord, events []runlog.Event, actions []runlog.ActionRecord, messages []rooms.Message) []runBeat {
	items := deriveRunBeats(record, events)
	for index, action := range actions {
		items = append(items, runBeat{
			ID:        fmt.Sprintf("%s-action-%d", record.ID, index),
			Type:      "handoff",
			Title:     firstNonEmpty(action.Kind, "action"),
			Summary:   firstNonEmpty(action.Body, action.Kind, "action"),
			Zone:      "sync_port",
			ActorRole: firstNonEmpty(action.Role, "human"),
			CreatedAt: action.CreatedAt.Format(time.RFC3339Nano),
		})
	}
	for index, message := range messages {
		beatType := "briefing"
		zone := "command_deck"
		if message.Kind == "instruction" {
			beatType = "briefing"
			zone = "command_deck"
		}
		items = append(items, runBeat{
			ID:        fmt.Sprintf("%s-message-%d", record.ID, index),
			Type:      beatType,
			Title:     firstNonEmpty(message.Kind, "message"),
			Summary:   firstNonEmpty(message.Body, message.Kind, "message"),
			Zone:      zone,
			ActorRole: firstNonEmpty(message.Role, "human"),
			CreatedAt: message.CreatedAt.Format(time.RFC3339Nano),
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt < items[j].CreatedAt
	})
	return items
}

func streamRunEvents(w http.ResponseWriter, r *http.Request, store runlog.EventStore, runID string) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("response writer does not support flushing")
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	last := 0
	for {
		events, err := store.LoadEvents(runID)
		if err != nil {
			return err
		}
		for _, event := range events[last:] {
			data, err := json.Marshal(event)
			if err != nil {
				return err
			}
			if _, err := fmt.Fprintf(w, "event: %s\n", event.Kind); err != nil {
				return nil
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				return nil
			}
			flusher.Flush()
		}
		last = len(events)

		select {
		case <-r.Context().Done():
			return nil
		case <-time.After(50 * time.Millisecond):
		}
	}
}
