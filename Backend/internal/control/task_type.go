package control

import "strings"

func isSourceTaskType(taskType string) bool {
	switch strings.ToLower(strings.TrimSpace(taskType)) {
	case "rss", "api", "page":
		return true
	default:
		return false
	}
}
