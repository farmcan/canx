package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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
	mux.HandleFunc("/api/context", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := workspace.Load(store.Root)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, map[string]any{
			"root":   ctx.Root,
			"readme": ctx.Readme,
			"agents": ctx.Agents,
			"docs":   ctx.Docs,
		})
	})

	staticRoot, err := fs.Sub(uiFiles, "ui")
	if err != nil {
		return nil, err
	}
	mux.Handle("/", http.FileServer(http.FS(staticRoot)))
	return mux, nil
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
