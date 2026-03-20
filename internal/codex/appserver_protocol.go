package codex

import "encoding/json"

type appServerRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type appServerResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *appServerError `json:"error,omitempty"`
}

type appServerNotification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type appServerError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type appServerClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type appServerInitializeParams struct {
	Client appServerClientInfo `json:"client"`
}

type appServerInitializeResult struct {
	Server appServerClientInfo `json:"server,omitempty"`
}

type appServerThreadStartParams struct{}

type appServerThreadStartedParams struct {
	ThreadID string `json:"thread_id"`
}

type appServerTurnStartParams struct {
	ThreadID string `json:"thread_id"`
	Input    string `json:"input"`
}

type appServerItem struct {
	ID   string `json:"id,omitempty"`
	Type string `json:"type,omitempty"`
	Text string `json:"text,omitempty"`
}

type appServerItemCompletedParams struct {
	ThreadID string        `json:"thread_id"`
	Item     appServerItem `json:"item"`
}

type appServerTurnCompletedParams struct {
	ThreadID string `json:"thread_id"`
	TurnID   string `json:"turn_id"`
	Output   string `json:"output,omitempty"`
}
