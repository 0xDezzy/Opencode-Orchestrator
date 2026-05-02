package orchestrator

func nextAttempt(current, maxAttempts int) (int, bool) {
	if maxAttempts <= 0 {
		maxAttempts = 1
	}
	if current >= maxAttempts {
		return current, false
	}
	return current + 1, true
}
