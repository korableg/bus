package metrics

// ErrToLabel returns the value of the "error" label: "true" if err is not nil, "false" otherwise.
func ErrToLabel(err error) string {
	if err == nil {
		return "false"
	}

	return "true"
}
