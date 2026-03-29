package runlog

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/farmcan/canx/internal/tasks"
)

type RunRecord struct {
	ID         string       `json:"id"`
	Goal       string       `json:"goal"`
	RepoRoot   string       `json:"repo_root"`
	Status     string       `json:"status"`
	Reason     string       `json:"reason,omitempty"`
	SessionID  string       `json:"session_id,omitempty"`
	TurnCount  int          `json:"turn_count"`
	TaskCount  int          `json:"task_count"`
	Tasks      []tasks.Task `json:"tasks,omitempty"`
	StartedAt  time.Time    `json:"started_at"`
	UpdatedAt  time.Time    `json:"updated_at"`
	FinishedAt *time.Time   `json:"finished_at,omitempty"`
}

type Event struct {
	RunID      string         `json:"run_id"`
	Timestamp  time.Time      `json:"timestamp"`
	Kind       string         `json:"kind"`
	SessionID  string         `json:"session_id,omitempty"`
	TaskID     string         `json:"task_id,omitempty"`
	Turn       int            `json:"turn,omitempty"`
	Message    string         `json:"message,omitempty"`
	Decision   string         `json:"decision,omitempty"`
	Reason     string         `json:"reason,omitempty"`
	Validated  bool           `json:"validated,omitempty"`
	Tasks      []tasks.Task   `json:"tasks,omitempty"`
	Runtime    map[string]any `json:"runtime,omitempty"`
	Validation string         `json:"validation,omitempty"`
	Output     string         `json:"output,omitempty"`
}

type EventStore struct {
	Root string
}

func NewEventStore(root string) EventStore {
	return EventStore{Root: root}
}

func NewRunID() string {
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "run-" + time.Now().Format("20060102T150405.000000000")
	}
	return "run-" + hex.EncodeToString(raw[:])
}

func (s EventStore) SaveRun(record RunRecord) error {
	record.UpdatedAt = time.Now()
	if record.StartedAt.IsZero() {
		record.StartedAt = record.UpdatedAt
	}

	dir := filepath.Join(s.Root, ".canx", "runs", record.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "run.json"), data, 0o644)
}

func (s EventStore) AppendEvent(runID string, event Event) error {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	if event.RunID == "" {
		event.RunID = runID
	}

	dir := filepath.Join(s.Root, ".canx", "runs", runID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	file, err := os.OpenFile(filepath.Join(dir, "events.jsonl"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		return err
	}
	return nil
}

func (s EventStore) ListRuns() ([]RunRecord, error) {
	root := filepath.Join(s.Root, ".canx", "runs")
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	runs := make([]RunRecord, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		record, err := s.LoadRun(entry.Name())
		if err != nil {
			return nil, err
		}
		runs = append(runs, record)
	}
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].StartedAt.After(runs[j].StartedAt)
	})
	return runs, nil
}

func (s EventStore) LoadRun(runID string) (RunRecord, error) {
	data, err := os.ReadFile(filepath.Join(s.Root, ".canx", "runs", runID, "run.json"))
	if err != nil {
		return RunRecord{}, err
	}
	var record RunRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return RunRecord{}, err
	}
	return record, nil
}

func (s EventStore) LoadEvents(runID string) ([]Event, error) {
	file, err := os.Open(filepath.Join(s.Root, ".canx", "runs", runID, "events.jsonl"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	var events []Event
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		var event Event
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return events, nil
}
