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
)

type Config struct {
	Goal               string
	MaxTurns           int
	ValidationCommands []string
}

func (c Config) Validate() error {
	switch {
	case c.Goal == "":
		return ErrMissingGoal
	case c.MaxTurns <= 0:
		return ErrInvalidMaxTurns
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
