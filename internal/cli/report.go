package cli

import execution "github.com/dhitalkamal/parley/internal/execution/domain"

// requestPassed reports whether a request's run counts as passing: it sent
// without error and every pm.test assertion in it (if any) passed.
func requestPassed(r execution.RunResult) bool {
	if r.Err != "" {
		return false
	}
	for _, tr := range r.Tests {
		if !tr.Passed {
			return false
		}
	}
	return true
}

// Success reports whether every request in results passed - the same
// condition Run uses to decide between exit code 0 and 1.
func Success(results []execution.RunResult) bool {
	for _, r := range results {
		if !requestPassed(r) {
			return false
		}
	}
	return true
}
