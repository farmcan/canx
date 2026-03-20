package tasks

import "errors"

const (
	StatusPending    = "pending"
	StatusInProgress = "in_progress"
	StatusDone       = "done"
	StatusBlocked    = "blocked"
)

var (
	ErrMissingTaskID   = errors.New("missing task id")
	ErrMissingTaskGoal = errors.New("missing task goal")
)

type Task struct {
	ID                 string
	Title              string
	Goal               string
	Status             string
	Owner              string
	OwnerSessionID     string
	ParentTaskID       string
	SpawnDepth         int
	DependsOn          []string
	PlannedFiles       []string
	FilesInScope       []string
	FilesChanged       []string
	BlockedBy          []string
	ValidationCommands []string
	Summary            string
}

func (t *Task) Normalize() {
	if t.Status == "" {
		t.Status = StatusPending
	}
}

func (t Task) Validate() error {
	switch {
	case t.ID == "":
		return ErrMissingTaskID
	case t.Goal == "":
		return ErrMissingTaskGoal
	default:
		return nil
	}
}
