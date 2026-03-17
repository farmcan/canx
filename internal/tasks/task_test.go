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
