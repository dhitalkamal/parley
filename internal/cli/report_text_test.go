package cli

import (
	"bytes"
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
	scripting "github.com/dhitalkamal/parley/internal/scripting/domain"
	"strings"
	"testing"
)

func TestWriteText_PassingRequestShowsPassAndStatus(t *testing.T) {
	var buf bytes.Buffer
	results := []execution.RunResult{
		{RequestPath: "a.json", Method: "GET", URL: "https://example.com/a", StatusCode: 200, Status: "200 OK", ElapsedMS: 42},
	}
	WriteText(&buf, results)
	out := buf.String()
	if !strings.Contains(out, "PASS") {
		t.Errorf("expected PASS marker, got:\n%s", out)
	}
	if !strings.Contains(out, "GET") || !strings.Contains(out, "https://example.com/a") {
		t.Errorf("expected method and url, got:\n%s", out)
	}
}

func TestWriteText_FailingRequestShowsFailAndError(t *testing.T) {
	var buf bytes.Buffer
	results := []execution.RunResult{
		{RequestPath: "a.json", Method: "GET", URL: "https://example.com/a", Err: "connection refused"},
	}
	WriteText(&buf, results)
	out := buf.String()
	if !strings.Contains(out, "FAIL") {
		t.Errorf("expected FAIL marker, got:\n%s", out)
	}
	if !strings.Contains(out, "connection refused") {
		t.Errorf("expected error message, got:\n%s", out)
	}
}

func TestWriteText_FailingTestShowsTestNameAndError(t *testing.T) {
	var buf bytes.Buffer
	results := []execution.RunResult{
		{RequestPath: "a.json", Method: "GET", URL: "https://example.com/a", StatusCode: 500,
			Tests: []scripting.TestResult{{Name: "status is 200", Passed: false, Error: "expected 200, got 500"}}},
	}
	WriteText(&buf, results)
	out := buf.String()
	if !strings.Contains(out, "status is 200") || !strings.Contains(out, "expected 200, got 500") {
		t.Errorf("expected failing test name and error, got:\n%s", out)
	}
}

func TestWriteText_SummaryLineCountsPassAndFail(t *testing.T) {
	var buf bytes.Buffer
	results := []execution.RunResult{
		{RequestPath: "a.json", StatusCode: 200},
		{RequestPath: "b.json", Err: "boom"},
	}
	WriteText(&buf, results)
	out := buf.String()
	if !strings.Contains(out, "1 passed") || !strings.Contains(out, "1 failed") {
		t.Errorf("expected summary counts, got:\n%s", out)
	}
}
