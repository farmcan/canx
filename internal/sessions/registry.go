package sessions

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sort"
	"time"
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
	ID          string
	Label       string
	Mode        string
	CWD         string
	Turns       []string
	LastSummary string
	Closed      bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
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
		ID:        newSessionID(),
		Label:     req.Label,
		Mode:      req.Mode,
		CWD:       req.CWD,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
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
	session.LastSummary = summary
	session.UpdatedAt = time.Now()
	r.sessions[id] = session
	return session, nil
}

func (r *Registry) Close(id string) (Session, error) {
	session, err := r.Get(id)
	if err != nil {
		return Session{}, err
	}
	session.Closed = true
	session.UpdatedAt = time.Now()
	r.sessions[id] = session
	return session, nil
}

func (r *Registry) List() []Session {
	sessions := make([]Session, 0, len(r.sessions))
	for _, session := range r.sessions {
		sessions = append(sessions, session)
	}

	sort.Slice(sessions, func(i, j int) bool {
		if sessions[i].Label == sessions[j].Label {
			return sessions[i].ID < sessions[j].ID
		}
		return sessions[i].Label < sessions[j].Label
	})

	return sessions
}

func newSessionID() string {
	var buf [8]byte
	_, _ = rand.Read(buf[:])
	return "session-" + hex.EncodeToString(buf[:])
}
