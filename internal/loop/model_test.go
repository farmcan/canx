package loop

import "testing"

func TestConfigValidateRequiresGoalAndMaxTurns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				Goal:     "ship task model",
				MaxTurns: 3,
			},
		},
		{
			name: "missing goal",
			config: Config{
				MaxTurns: 3,
			},
			wantErr: true,
		},
		{
			name: "missing max turns",
			config: Config{
				Goal: "ship task model",
			},
			wantErr: true,
		},
		{
			name: "invalid budget",
			config: Config{
				Goal:          "ship task model",
				MaxTurns:      1,
				BudgetSeconds: -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDecisionDoneReportsTerminalState(t *testing.T) {
	t.Parallel()

	decision := Decision{Action: ActionStop}

	if !decision.Terminal() {
		t.Fatal("expected stop decision to be terminal")
	}
}
