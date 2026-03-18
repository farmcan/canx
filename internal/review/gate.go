package review

type Result struct {
	Validated bool
	Approved  bool
	Reason    string
}

func Evaluate(result Result) Result {
	if !result.Validated {
		result.Approved = false
		result.Reason = "missing validation"
		return result
	}
	result.Approved = true
	result.Reason = "approved"
	return result
}
