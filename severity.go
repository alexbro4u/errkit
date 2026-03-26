package errkit

// Severity represents the severity level of an error.
type Severity int

const (
	SeverityLow      Severity = iota // SeverityLow indicates a minor, non-critical error.
	SeverityMedium                   // SeverityMedium indicates a moderate error that may need attention.
	SeverityHigh                     // SeverityHigh indicates a serious error that requires prompt attention.
	SeverityCritical                 // SeverityCritical indicates a critical error that demands immediate action.
)

var severityNames = [...]string{
	SeverityLow:      "low",
	SeverityMedium:   "medium",
	SeverityHigh:     "high",
	SeverityCritical: "critical",
}

// String returns the string representation of the severity level.
func (s Severity) String() string {
	if int(s) < len(severityNames) {
		return severityNames[s]
	}
	return "unknown"
}
