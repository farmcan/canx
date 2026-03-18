package sessions

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
)

const (
	ModeOneshot    = "oneshot"
	ModePersistent = "persistent"
)

var (
	ErrMissingLabel    = errors.New("missing session label")
	ErrInvalidMode     = errors.New("invalid session mode")
	ErrSessionNotFound = errors.New("session not found")
)

type SpawnRequest struct {
	Label string
	Mode  string
	CWD   string
}

type Session struct {
	ID     string
	Label  string
	Mode   string
	CWD    string
	Turns  []string
	Closed bool
}

type Registry struct {
	sessions map[string]Session
}

func NewRegistry() *Registry {
	return &Registry{sessions: make(map[string]Session)}
}

func (r *Registry) Spawn(req SpawnRequest) (Session, error) {
	if req.Label == "" {
		return Session{}, ErrMissingLabel
	}
	if req.Mode == "" {
		req.Mode = ModeOneshot
	}
	if req.Mode != ModeOneshot && req.Mode != ModePersistent {
		return Session{}, ErrInvalidMode
	}

	session := Session{
		ID:    newSessionID(),
		Label: req.Label,
		Mode:  req.Mode,
		CWD:   req.CWD,
	}
	r.sessions[session.ID] = session
	return session, nil
}

func (r *Registry) Get(id string) (Session, error) {
	session, ok := r.sessions[id]
	if !ok {
		return Session{}, ErrSessionNotFound
	}
	return session, nil
}

func (r *Registry) Steer(id, summary string) (Session, error) {
	session, err := r.Get(id)
	if err != nil {
		return Session{}, err
	}
	session.Turns = append(session.Turns, summary)
	r.sessions[id] = session
	return session, nil
}

func (r *Registry) Close(id string) (Session, error) {
	session, err := r.Get(id)
	if err != nil {
		return Session{}, err
	}
	session.Closed = true
	r.sessions[id] = session
	return session, nil
}

func newSessionID() string {
	var buf [8]byte
	_, _ = rand.Read(buf[:])
	return "session-" + hex.EncodeToString(buf[:])
}
