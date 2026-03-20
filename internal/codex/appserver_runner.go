package codex

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

type AppServerRunner struct {
	conn *appServerConn

	mu      sync.Mutex
	threads map[string]string
}

func NewAppServerRunner(bin string) (*AppServerRunner, error) {
	conn, err := newAppServerConn(bin)
	if err != nil {
		return nil, err
	}
	return newAppServerRunnerWithConn(conn), nil
}

func newAppServerRunnerWithConn(conn *appServerConn) *AppServerRunner {
	return &AppServerRunner{
		conn:    conn,
		threads: map[string]string{},
	}
}

func (r *AppServerRunner) Run(ctx context.Context, req Request) (Result, error) {
	if err := req.Validate(); err != nil {
		return Result{}, err
	}
	if err := r.conn.Initialize(ctx); err != nil {
		return Result{}, err
	}

	sessionKey := req.SessionKey
	if sessionKey == "" {
		sessionKey = "ephemeral"
	}
	threadID, err := r.ensureThread(ctx, sessionKey)
	if err != nil {
		return Result{}, err
	}

	resp, err := r.conn.call(ctx, "turn/start", appServerTurnStartParams{
		ThreadID: threadID,
		Input:    req.Prompt,
	})
	if err != nil {
		return Result{}, err
	}

	var completed appServerTurnCompletedParams
	if err := json.Unmarshal(resp.Result, &completed); err != nil {
		return Result{}, err
	}
	return Result{
		Output:   completed.Output,
		ExitCode: 0,
		Runtime: Runtime{
			SessionID: threadID,
		},
	}, nil
}

func (r *AppServerRunner) ensureThread(ctx context.Context, sessionKey string) (string, error) {
	r.mu.Lock()
	threadID := r.threads[sessionKey]
	r.mu.Unlock()
	if threadID != "" {
		return threadID, nil
	}

	resp, err := r.conn.call(ctx, "thread/start", appServerThreadStartParams{})
	if err != nil {
		return "", err
	}
	var started appServerThreadStartedParams
	if err := json.Unmarshal(resp.Result, &started); err != nil {
		return "", err
	}
	if started.ThreadID == "" {
		return "", fmt.Errorf("missing thread id")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if existing := r.threads[sessionKey]; existing != "" {
		return existing, nil
	}
	r.threads[sessionKey] = started.ThreadID
	return started.ThreadID, nil
}
