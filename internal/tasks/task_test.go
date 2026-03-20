package tasks

import "testing"

func TestTaskValidateRequiresIDAndGoal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		task    Task
		wantErr bool
	}{
		{
			name: "valid task",
			task: Task{
				ID:    "task-1",
				Goal:  "add task model",
				Title: "Task model",
			},
		},
		{
			name: "missing id",
			task: Task{
				Goal: "add task model",
			},
			wantErr: true,
		},
		{
			name: "missing goal",
			task: Task{
				ID: "task-1",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.task.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTaskDefaultsStatusToPending(t *testing.T) {
	t.Parallel()

	task := Task{ID: "task-1", Goal: "add task model"}
	task.Normalize()

	if got, want := task.Status, StatusPending; got != want {
		t.Fatalf("Normalize() status = %q, want %q", got, want)
	}
}

func TestTaskNormalizePreservesSchedulerMetadata(t *testing.T) {
	t.Parallel()

	task := Task{
		ID:             "task-child",
		Goal:           "write regression test",
		ParentTaskID:   "task-parent",
		SpawnDepth:     1,
		OwnerSessionID: "session-123",
		DependsOn:      []string{"task-parent"},
		PlannedFiles:   []string{"internal/loop/engine_test.go"},
	}

	task.Normalize()

	if task.ParentTaskID != "task-parent" {
		t.Fatalf("Normalize() parent task id = %q, want task-parent", task.ParentTaskID)
	}
	if task.SpawnDepth != 1 {
		t.Fatalf("Normalize() spawn depth = %d, want 1", task.SpawnDepth)
	}
	if task.OwnerSessionID != "session-123" {
		t.Fatalf("Normalize() owner session id = %q, want session-123", task.OwnerSessionID)
	}
	if len(task.DependsOn) != 1 || task.DependsOn[0] != "task-parent" {
		t.Fatalf("Normalize() depends_on = %#v, want [task-parent]", task.DependsOn)
	}
	if len(task.PlannedFiles) != 1 || task.PlannedFiles[0] != "internal/loop/engine_test.go" {
		t.Fatalf("Normalize() planned files = %#v, want engine test path", task.PlannedFiles)
	}
}

func TestTaskValidateAllowsEmptyPlannedFiles(t *testing.T) {
	t.Parallel()

	task := Task{
		ID:           "task-1",
		Goal:         "implement scheduler",
		PlannedFiles: nil,
	}

	if err := task.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}
