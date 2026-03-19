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
