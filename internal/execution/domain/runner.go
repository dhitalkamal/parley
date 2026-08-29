package execution

import (
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	scripting "github.com/dhitalkamal/parley/internal/scripting/domain"
)

// RunResult is the outcome of one request executed by the collection runner.
type RunResult struct {
	RequestPath string
	Method      collection.Method
	URL         string
	StatusCode  int
	Status      string
	ElapsedMS   int64
	Err         string
	Tests       []scripting.TestResult
}
