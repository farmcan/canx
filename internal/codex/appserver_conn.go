package codex

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"sync"
	"sync/atomic"
)

var errAppServerDecode = errors.New("app server decode error")

type appServerConn struct {
	reader io.Reader
	writer io.Writer
	cmd    *exec.Cmd

	writeMu         sync.Mutex
	notifications   chan appServerNotification
	pendingMu       sync.Mutex
	pending         map[int64]chan appServerResponse
	errMu           sync.Mutex
	errCh           chan struct{}
	nextID          atomic.Int64
	initOnce        sync.Once
	initErr         error
	readLoopStarted sync.Once
}

func newAppServerConnWithIO(reader io.Reader, writer io.Writer) *appServerConn {
	return &appServerConn{
		reader:        reader,
		writer:        writer,
		notifications: make(chan appServerNotification, 32),
		pending:       map[int64]chan appServerResponse{},
		errCh:         make(chan struct{}),
	}
}

func newAppServerConn(bin string) (*appServerConn, error) {
	cmd := exec.Command(bin, "app-server")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	conn := newAppServerConnWithIO(stdout, stdin)
	conn.cmd = cmd
	return conn, nil
}

func (c *appServerConn) Initialize(ctx context.Context) error {
	c.initOnce.Do(func() {
		c.startReadLoop()
		_, c.initErr = c.call(ctx, "initialize", appServerInitializeParams{
			Client: appServerClientInfo{Name: "canx", Version: "dev"},
		})
	})
	return c.initErr
}

func (c *appServerConn) Notifications() <-chan appServerNotification {
	return c.notifications
}

func (c *appServerConn) call(ctx context.Context, method string, params any) (appServerResponse, error) {
	if err := c.connectionError(); err != nil {
		return appServerResponse{}, err
	}
	id := c.nextID.Add(1)
	respCh := make(chan appServerResponse, 1)
	c.pendingMu.Lock()
	c.pending[id] = respCh
	c.pendingMu.Unlock()

	req := appServerRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return appServerResponse{}, err
	}

	c.writeMu.Lock()
	_, err = c.writer.Write(append(data, '\n'))
	c.writeMu.Unlock()
	if err != nil {
		return appServerResponse{}, err
	}

	select {
	case resp := <-respCh:
		if err := c.connectionError(); err != nil {
			return appServerResponse{}, err
		}
		if resp.Error != nil {
			return resp, errors.New(resp.Error.Message)
		}
		return resp, nil
	case <-c.errCh:
		return appServerResponse{}, c.connectionError()
	case <-ctx.Done():
		return appServerResponse{}, ctx.Err()
	}
}

func (c *appServerConn) startReadLoop() {
	c.readLoopStarted.Do(func() {
		go func() {
			scanner := bufio.NewScanner(c.reader)
			for scanner.Scan() {
				line := scanner.Bytes()
				var envelope map[string]json.RawMessage
				if err := json.Unmarshal(line, &envelope); err != nil {
					c.setConnectionError(errors.Join(errAppServerDecode, err))
					return
				}
				if _, ok := envelope["id"]; ok {
					var resp appServerResponse
					if err := json.Unmarshal(line, &resp); err != nil {
						c.setConnectionError(errors.Join(errAppServerDecode, err))
						return
					}
					c.pendingMu.Lock()
					respCh := c.pending[resp.ID]
					delete(c.pending, resp.ID)
					c.pendingMu.Unlock()
					if respCh != nil {
						respCh <- resp
					}
					continue
				}
				var notice appServerNotification
				if err := json.Unmarshal(line, &notice); err != nil {
					c.setConnectionError(errors.Join(errAppServerDecode, err))
					return
				}
				c.notifications <- notice
			}
		}()
	})
}

func (c *appServerConn) connectionError() error {
	c.errMu.Lock()
	defer c.errMu.Unlock()
	return c.initErr
}

func (c *appServerConn) setConnectionError(err error) {
	c.errMu.Lock()
	alreadySet := c.initErr != nil
	c.initErr = err
	c.errMu.Unlock()
	if !alreadySet {
		close(c.errCh)
	}

	c.pendingMu.Lock()
	defer c.pendingMu.Unlock()
	for id, ch := range c.pending {
		ch <- appServerResponse{JSONRPC: "2.0", ID: id, Error: &appServerError{Code: -1, Message: err.Error()}}
		delete(c.pending, id)
	}
}
