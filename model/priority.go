package model

const (
	JobPriorityLow    int = 0
	JobPriorityNormal int = 1
	JobPriorityHigh   int = 2
	JobPriorityUrgent int = 3
)

func JobPriorityIsValid(p int) bool {
	return p >= JobPriorityLow && p <= JobPriorityUrgent
}

func JobPriorityString(p int) string {
	switch p {
	case JobPriorityLow:
		return "low"
	case JobPriorityNormal:
		return "normal"
	case JobPriorityHigh:
		return "high"
	case JobPriorityUrgent:
		return "urgent"
	default:
		return "unknown"
	}
}

func JobPriorityFromString(s string) int {
	switch s {
	case "low":
		return JobPriorityLow
	case "normal":
		return JobPriorityNormal
	case "high":
		return JobPriorityHigh
	case "urgent":
		return JobPriorityUrgent
	default:
		return JobPriorityNormal
	}
}

func JobPriorityValidate(p int) int {
	if !JobPriorityIsValid(p) {
		return JobPriorityNormal
	}
	return p
}
