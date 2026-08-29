package cli

import (
	"bytes"
	"encoding/json"
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
	scripting "github.com/dhitalkamal/parley/internal/scripting/domain"
	"testing"
)

func TestWriteJSON_EncodesTotalsAndPerRequestResults(t *testing.T) {
	var buf bytes.Buffer
	results := []execution.RunResult{
		{RequestPath: "a.json", Method: "GET", URL: "https://example.com/a", StatusCode: 200, ElapsedMS: 10,
			Tests: []scripting.TestResult{{Name: "status is 200", Passed: true}}},
		{RequestPath: "b.json", Method: "POST", URL: "https://example.com/b", Err: "connection refused"},
	}
	if err := WriteJSON(&buf, results); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var decoded struct {
		Total   int `json:"total"`
		Passed  int `json:"passed"`
		Failed  int `json:"failed"`
		Results []struct {
			RequestPath string `json:"request_path"`
			Passed      bool   `json:"passed"`
			Err         string `json:"error"`
			Tests       []struct {
				Name   string `json:"name"`
				Passed bool   `json:"passed"`
			} `json:"tests"`
		} `json:"results"`
	}
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("output is not valid json: %v\n%s", err, buf.String())
	}
	if decoded.Total != 2 || decoded.Passed != 1 || decoded.Failed != 1 {
		t.Errorf("totals = %+v, want total=2 passed=1 failed=1", decoded)
	}
	if len(decoded.Results) != 2 {
		t.Fatalf("got %d results, want 2", len(decoded.Results))
	}
	if decoded.Results[0].RequestPath != "a.json" || !decoded.Results[0].Passed {
		t.Errorf("results[0] = %+v", decoded.Results[0])
	}
	if len(decoded.Results[0].Tests) != 1 || decoded.Results[0].Tests[0].Name != "status is 200" {
		t.Errorf("results[0].Tests = %+v", decoded.Results[0].Tests)
	}
	if decoded.Results[1].RequestPath != "b.json" || decoded.Results[1].Passed || decoded.Results[1].Err != "connection refused" {
		t.Errorf("results[1] = %+v", decoded.Results[1])
	}
}
