package sessions

import "testing"

func TestRegistrySpawnPersistentSession(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	session, err := registry.Spawn(SpawnRequest{
		Label: "main",
		Mode:  ModePersistent,
		CWD:   "/repo",
	})
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}

	if session.ID == "" {
		t.Fatal("expected session id")
	}
	if session.Mode != ModePersistent {
		t.Fatalf("Mode = %q, want %q", session.Mode, ModePersistent)
	}
}

func TestRegistrySteerAppendsTurnSummary(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	session, err := registry.Spawn(SpawnRequest{Label: "main", Mode: ModePersistent})
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}

	updated, err := registry.Steer(session.ID, "implemented task model")
	if err != nil {
		t.Fatalf("Steer() error = %v", err)
	}

	if got, want := len(updated.Turns), 1; got != want {
		t.Fatalf("Turns = %d, want %d", got, want)
	}
}

func TestRegistryCloseMarksSessionClosed(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	session, err := registry.Spawn(SpawnRequest{Label: "main", Mode: ModePersistent})
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}

	closed, err := registry.Close(session.ID)
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if !closed.Closed {
		t.Fatal("expected closed session")
	}
}
