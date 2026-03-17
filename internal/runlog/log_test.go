package runlog

import "testing"

func TestEntryValidateRequiresGoalAndDecision(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		entry   Entry
		wantErr bool
	}{
		{
			name: "valid entry",
			entry: Entry{
				Goal:     "ship task model",
				Decision: "continue",
			},
		},
		{
			name: "missing goal",
			entry: Entry{
				Decision: "continue",
			},
			wantErr: true,
		},
		{
			name: "missing decision",
			entry: Entry{
				Goal: "ship task model",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.entry.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
