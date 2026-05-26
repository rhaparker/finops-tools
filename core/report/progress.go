package report

// Progress reports long-running steps while building a report.
type Progress interface {
	Step(message string)
}

// noopProgress discards progress messages.
type noopProgress struct{}

func (noopProgress) Step(string) {}
