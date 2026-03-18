package loop

import "errors"

const (
	ActionContinue = "continue"
	ActionStop     = "stop"
	ActionEscalate = "escalate"
)

var (
	ErrMissingGoal     = errors.New("missing goal")
	ErrInvalidMaxTurns = errors.New("invalid max turns")
	ErrInvalidBudget   = errors.New("invalid budget")
)

type Config struct {
	Goal               string
	MaxTurns           int
	BudgetSeconds      int
	ValidationCommands []string
}

func (c Config) Validate() error {
	switch {
	case c.Goal == "":
		return ErrMissingGoal
	case c.MaxTurns <= 0:
		return ErrInvalidMaxTurns
	case c.BudgetSeconds < 0:
		return ErrInvalidBudget
	default:
		return nil
	}
}

type Decision struct {
	Action string
	Reason string
}

func (d Decision) Terminal() bool {
	return d.Action == ActionStop || d.Action == ActionEscalate
}
