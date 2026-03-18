package runlog

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type ActionRecord struct {
	RunID         string    `json:"run_id"`
	RoomID        string    `json:"room_id,omitempty"`
	TaskID        string    `json:"task_id,omitempty"`
	ParticipantID string    `json:"participant_id,omitempty"`
	Role          string    `json:"role,omitempty"`
	Kind          string    `json:"kind,omitempty"`
	Body          string    `json:"body,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

func (s EventStore) AppendAction(runID string, action ActionRecord) error {
	if action.RunID == "" {
		action.RunID = runID
	}
	if action.CreatedAt.IsZero() {
		action.CreatedAt = time.Now()
	}
	dir := filepath.Join(s.Root, ".canx", "runs", runID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(filepath.Join(dir, "actions.jsonl"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	data, err := json.Marshal(action)
	if err != nil {
		return err
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		return err
	}
	return nil
}

func (s EventStore) ListActions(runID string) ([]ActionRecord, error) {
	file, err := os.Open(filepath.Join(s.Root, ".canx", "runs", runID, "actions.jsonl"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()
	var actions []ActionRecord
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var action ActionRecord
		if err := json.Unmarshal(scanner.Bytes(), &action); err != nil {
			return nil, err
		}
		actions = append(actions, action)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return actions, nil
}
