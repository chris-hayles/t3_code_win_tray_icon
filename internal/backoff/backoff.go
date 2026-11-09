package backoff

import "time"

// Delay returns the wait before restart attempt n (1-based).
func Delay(attempt int) time.Duration {
	switch attempt {
	case 1:
		return time.Second
	case 2:
		return 2 * time.Second
	case 3:
		return 4 * time.Second
	default:
		return 15 * time.Second
	}
}

// ShouldResetUptime reports whether a successful run duration clears the attempt counter.
func ShouldResetUptime(uptime time.Duration) bool {
	return uptime >= 60*time.Second
}

// NextAttemptAfterReset returns 1 after a reset.
func NextAttemptAfterReset() int {
	return 1
}
