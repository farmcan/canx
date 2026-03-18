package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/farmcan/canx/internal/runlog"
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

	staticRoot, err := fs.Sub(uiFiles, "ui")
	if err != nil {
		return nil, err
	}
	mux.Handle("/", http.FileServer(http.FS(staticRoot)))
	return mux, nil
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	writeJSON(w, map[string]string{"error": err.Error()})
}
