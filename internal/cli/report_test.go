package cli

import (
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
	scripting "github.com/dhitalkamal/parley/internal/scripting/domain"
	"testing"
)

func TestSuccess_AllPassedNoTests(t *testing.T) {
	results := []execution.RunResult{
		{RequestPath: "a.json", StatusCode: 200},
		{RequestPath: "b.json", StatusCode: 201},
	}
	if !Success(results) {
		t.Error("expected success when no request errored and no test failed")
	}
}

func TestSuccess_FalseWhenARequestErrored(t *testing.T) {
	results := []execution.RunResult{
		{RequestPath: "a.json", StatusCode: 200},
		{RequestPath: "b.json", Err: "connection refused"},
	}
	if Success(results) {
		t.Error("expected failure when a request errored")
	}
}

func TestSuccess_FalseWhenATestFailed(t *testing.T) {
	results := []execution.RunResult{
		{RequestPath: "a.json", StatusCode: 200, Tests: []scripting.TestResult{
			{Name: "status is 200", Passed: false, Error: "expected 200, got 500"},
		}},
	}
	if Success(results) {
		t.Error("expected failure when a test assertion failed")
	}
}

func TestSuccess_EmptyResultsIsSuccess(t *testing.T) {
	if !Success(nil) {
		t.Error("expected an empty run to count as success")
	}
}
