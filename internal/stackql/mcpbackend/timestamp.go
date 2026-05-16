package mcpbackend

import "time"

// nowTimestamp returns the current time in the format used by mutation/lifecycle responses.
func nowTimestamp() string {
	return time.Now().Format("2006-01-02T15:04:05-07:00 MST")
}
