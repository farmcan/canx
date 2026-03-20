package loop

import "errors"

const (
	ActionContinue = "continue"
	ActionStop     = "stop"
	ActionEscalate = "escalate"
)

var (
	ErrMissingGoal               = errors.New("missing goal")
	ErrInvalidMaxTurns           = errors.New("invalid max turns")
	ErrInvalidBudget             = errors.New("invalid budget")
	ErrInvalidMaxWorkers         = errors.New("invalid max workers")
	ErrInvalidMaxSpawnDepth      = errors.New("invalid max spawn depth")
	ErrInvalidMaxChildrenPerTask = errors.New("invalid max children per task")
)

type Config struct {
	Goal               string
	MaxTurns           int
	BudgetSeconds      int
	ValidationCommands []string
	MaxWorkers         int
	MaxSpawnDepth      int
	MaxChildrenPerTask int
}

func (c Config) WithDefaults() Config {
	if c.MaxWorkers == 0 {
		c.MaxWorkers = 2
	}
	if c.MaxSpawnDepth == 0 {
		c.MaxSpawnDepth = 1
	}
	if c.MaxChildrenPerTask == 0 {
		c.MaxChildrenPerTask = 2
	}
	return c
}

func (c Config) Validate() error {
	switch {
	case c.Goal == "":
		return ErrMissingGoal
	case c.MaxTurns <= 0:
		return ErrInvalidMaxTurns
	case c.BudgetSeconds < 0:
		return ErrInvalidBudget
	case c.MaxWorkers < 0:
		return ErrInvalidMaxWorkers
	case c.MaxSpawnDepth < 0:
		return ErrInvalidMaxSpawnDepth
	case c.MaxChildrenPerTask < 0:
		return ErrInvalidMaxChildrenPerTask
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
