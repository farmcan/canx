package review

type Result struct {
	Validated bool
	InScope   bool
	Approved  bool
	Reason    string
}

func Evaluate(result Result) Result {
	switch {
	case !result.Validated:
		result.Approved = false
		result.Reason = "missing validation"
	case !result.InScope:
		result.Approved = false
		result.Reason = "out of scope"
	default:
		result.Approved = true
		result.Reason = "approved"
	}

	return result
}
