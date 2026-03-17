package runlog

import "errors"

var (
	ErrMissingGoal     = errors.New("missing goal")
	ErrMissingDecision = errors.New("missing decision")
)

type Entry struct {
	Goal     string
	Decision string
	Summary  string
}

func (e Entry) Validate() error {
	switch {
	case e.Goal == "":
		return ErrMissingGoal
	case e.Decision == "":
		return ErrMissingDecision
	default:
		return nil
	}
}
