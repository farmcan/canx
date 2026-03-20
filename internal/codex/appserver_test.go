package codex

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"testing"
	"time"
)

func TestAppServerProtocolEncodesInitializeThreadAndTurnRequests(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		message  any
		method   string
		wantJSON []string
	}{
		{
			name: "initialize",
			message: appServerRequest{
				JSONRPC: "2.0",
				ID:      1,
				Method:  "initialize",
				Params: appServerInitializeParams{
					Client: appServerClientInfo{Name: "canx", Version: "dev"},
				},
			},
			method:   "initialize",
			wantJSON: []string{`"method":"initialize"`, `"client"`},
		},
		{
			name: "thread start",
			message: appServerRequest{
				JSONRPC: "2.0",
				ID:      2,
				Method:  "thread/start",
				Params:  appServerThreadStartParams{},
			},
			method:   "thread/start",
			wantJSON: []string{`"method":"thread/start"`},
		},
		{
			name: "turn start",
			message: appServerRequest{
				JSONRPC: "2.0",
				ID:      3,
				Method:  "turn/start",
				Params: appServerTurnStartParams{
					ThreadID: "thread-1",
					Input:    "hello",
				},
			},
			method:   "turn/start",
			wantJSON: []string{`"method":"turn/start"`, `"thread_id":"thread-1"`},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data, err := json.Marshal(tt.message)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			encoded := string(data)
			for _, want := range tt.wantJSON {
				if !containsJSON(encoded, want) {
					t.Fatalf("encoded json = %s, want fragment %s", encoded, want)
				}
			}
		})
	}
}

func TestAppServerProtocolDecodesNotifications(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		raw    string
		method string
	}{
		{
			name:   "thread started",
			raw:    `{"jsonrpc":"2.0","method":"thread/started","params":{"thread_id":"thread-1"}}`,
			method: "thread/started",
		},
		{
			name:   "item completed",
			raw:    `{"jsonrpc":"2.0","method":"item/completed","params":{"thread_id":"thread-1","item":{"id":"item-1","type":"message","text":"done"}}}`,
			method: "item/completed",
		},
		{
			name:   "turn completed",
			raw:    `{"jsonrpc":"2.0","method":"turn/completed","params":{"thread_id":"thread-1","turn_id":"turn-1"}}`,
			method: "turn/completed",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var notice appServerNotification
			if err := json.Unmarshal([]byte(tt.raw), &notice); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}
			if notice.Method != tt.method {
				t.Fatalf("method = %q, want %q", notice.Method, tt.method)
			}
		})
	}
}

func TestAppServerConnInitializeAndNotificationFlow(t *testing.T) {
	t.Parallel()

	server := newFakeAppServer()
	conn := newAppServerConnWithIO(server.reader(), server.writer())

	if err := conn.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	if got, want := server.methodAt(0), "initialize"; got != want {
		t.Fatalf("first method = %q, want %q", got, want)
	}

	server.sendNotification(appServerNotification{
		JSONRPC: "2.0",
		Method:  "thread/started",
		Params:  mustRawJSON(t, appServerThreadStartedParams{ThreadID: "thread-1"}),
	})

	select {
	case notice := <-conn.Notifications():
		if notice.Method != "thread/started" {
			t.Fatalf("notification method = %q, want thread/started", notice.Method)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for notification")
	}
}

func TestAppServerConnReturnsDecodeErrorOnMalformedServerOutput(t *testing.T) {
	t.Parallel()

	server := newFakeAppServer()
	conn := newAppServerConnWithIO(server.reader(), server.writer())
	go server.sendRaw("{not-json}\n")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := conn.Initialize(ctx)
	if err == nil {
		t.Fatal("expected initialize error")
	}
	if !errors.Is(err, errAppServerDecode) {
		t.Fatalf("Initialize() error = %v, want decode error", err)
	}
}

func TestAppServerRunnerReusesThreadPerSessionKey(t *testing.T) {
	t.Parallel()

	server := newFakeAppServer()
	runner := newAppServerRunnerWithConn(newAppServerConnWithIO(server.reader(), server.writer()))

	first, err := runner.Run(context.Background(), Request{Prompt: "first", SessionKey: "session-a"})
	if err != nil {
		t.Fatalf("Run(first) error = %v", err)
	}
	second, err := runner.Run(context.Background(), Request{Prompt: "second", SessionKey: "session-a"})
	if err != nil {
		t.Fatalf("Run(second) error = %v", err)
	}

	if first.Runtime.SessionID == "" || second.Runtime.SessionID == "" {
		t.Fatal("expected runtime session ids to be populated")
	}
	if first.Runtime.SessionID != second.Runtime.SessionID {
		t.Fatalf("session ids differ: %q vs %q", first.Runtime.SessionID, second.Runtime.SessionID)
	}
	if got, want := server.countMethod("thread/start"), 1; got != want {
		t.Fatalf("thread/start count = %d, want %d", got, want)
	}
}

func TestAppServerRunnerSeparatesThreadsAcrossSessionKeys(t *testing.T) {
	t.Parallel()

	server := newFakeAppServer()
	runner := newAppServerRunnerWithConn(newAppServerConnWithIO(server.reader(), server.writer()))

	first, err := runner.Run(context.Background(), Request{Prompt: "first", SessionKey: "session-a"})
	if err != nil {
		t.Fatalf("Run(first) error = %v", err)
	}
	second, err := runner.Run(context.Background(), Request{Prompt: "second", SessionKey: "session-b"})
	if err != nil {
		t.Fatalf("Run(second) error = %v", err)
	}

	if first.Runtime.SessionID == second.Runtime.SessionID {
		t.Fatalf("expected different session ids, got %q", first.Runtime.SessionID)
	}
	if got, want := server.countMethod("thread/start"), 2; got != want {
		t.Fatalf("thread/start count = %d, want %d", got, want)
	}
}

func containsJSON(input, fragment string) bool {
	return json.Valid([]byte(input)) && len(input) > 0 && len(fragment) > 0 && stringContains(input, fragment)
}

type fakeAppServer struct {
	toConnReader   *io.PipeReader
	toConnWriter   *io.PipeWriter
	fromConnReader *io.PipeReader
	fromConnWriter *io.PipeWriter
	methods        []string
	threadCounter  int
	mu             sync.Mutex
}

func newFakeAppServer() *fakeAppServer {
	toConnReader, toConnWriter := io.Pipe()
	fromConnReader, fromConnWriter := io.Pipe()
	server := &fakeAppServer{
		toConnReader:   toConnReader,
		toConnWriter:   toConnWriter,
		fromConnReader: fromConnReader,
		fromConnWriter: fromConnWriter,
	}
	go server.loop()
	return server
}

func (s *fakeAppServer) reader() io.Reader { return s.toConnReader }
func (s *fakeAppServer) writer() io.Writer { return s.fromConnWriter }

func (s *fakeAppServer) loop() {
	scanner := bufio.NewScanner(s.fromConnReader)
	for scanner.Scan() {
		var req appServerRequest
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			continue
		}
		s.mu.Lock()
		s.methods = append(s.methods, req.Method)
		s.mu.Unlock()
		switch req.Method {
		case "initialize":
			s.sendResponse(appServerResponse{JSONRPC: "2.0", ID: req.ID, Result: mustMarshalRaw(appServerInitializeResult{Server: appServerClientInfo{Name: "codex", Version: "dev"}})})
		case "thread/start":
			s.mu.Lock()
			s.threadCounter++
			threadID := "thread-" + string(rune('0'+s.threadCounter))
			s.mu.Unlock()
			s.sendResponse(appServerResponse{JSONRPC: "2.0", ID: req.ID, Result: mustMarshalRaw(appServerThreadStartedParams{ThreadID: threadID})})
		case "turn/start":
			var params appServerTurnStartParams
			_ = remarshal(req.Params, &params)
			s.sendResponse(appServerResponse{JSONRPC: "2.0", ID: req.ID, Result: mustMarshalRaw(appServerTurnCompletedParams{ThreadID: params.ThreadID, TurnID: "turn-1", Output: "done"})})
		}
	}
}

func (s *fakeAppServer) methodAt(index int) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index >= len(s.methods) {
		return ""
	}
	return s.methods[index]
}

func (s *fakeAppServer) countMethod(method string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	total := 0
	for _, item := range s.methods {
		if item == method {
			total++
		}
	}
	return total
}

func (s *fakeAppServer) sendResponse(resp appServerResponse) {
	data, _ := json.Marshal(resp)
	_, _ = s.toConnWriter.Write(append(data, '\n'))
}

func (s *fakeAppServer) sendNotification(notice appServerNotification) {
	data, _ := json.Marshal(notice)
	_, _ = s.toConnWriter.Write(append(data, '\n'))
}

func (s *fakeAppServer) sendRaw(raw string) {
	_, _ = io.WriteString(s.toConnWriter, raw)
}

func mustRawJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()
	return mustMarshalRaw(value)
}

func mustMarshalRaw(value any) json.RawMessage {
	data, _ := json.Marshal(value)
	return data
}

func remarshal(input any, out any) error {
	data, err := json.Marshal(input)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}

func stringContains(input, fragment string) bool {
	return len(fragment) == 0 || (len(input) >= len(fragment) && indexOf(input, fragment) >= 0)
}

func indexOf(input, fragment string) int {
	for i := 0; i+len(fragment) <= len(input); i++ {
		if input[i:i+len(fragment)] == fragment {
			return i
		}
	}
	return -1
}
