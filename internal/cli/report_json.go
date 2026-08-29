package cli

import (
	"encoding/json"
	"io"

	execution "github.com/dhitalkamal/parley/internal/execution/domain"
	scripting "github.com/dhitalkamal/parley/internal/scripting/domain"
)

type jsonReport struct {
	Total   int          `json:"total"`
	Passed  int          `json:"passed"`
	Failed  int          `json:"failed"`
	Results []jsonResult `json:"results"`
}

type jsonResult struct {
	RequestPath string           `json:"request_path"`
	Method      string           `json:"method"`
	URL         string           `json:"url"`
	StatusCode  int              `json:"status_code,omitempty"`
	Status      string           `json:"status,omitempty"`
	ElapsedMS   int64            `json:"elapsed_ms"`
	Err         string           `json:"error,omitempty"`
	Passed      bool             `json:"passed"`
	Tests       []jsonTestResult `json:"tests,omitempty"`
}

type jsonTestResult struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Error  string `json:"error,omitempty"`
}

// WriteJSON renders results as a machine-readable report: overall totals
// plus one entry per request, each with its own test assertions.
func WriteJSON(w io.Writer, results []execution.RunResult) error {
	report := jsonReport{Total: len(results)}
	for _, r := range results {
		if requestPassed(r) {
			report.Passed++
		} else {
			report.Failed++
		}
		report.Results = append(report.Results, jsonResult{
			RequestPath: r.RequestPath,
			Method:      string(r.Method),
			URL:         r.URL,
			StatusCode:  r.StatusCode,
			Status:      r.Status,
			ElapsedMS:   r.ElapsedMS,
			Err:         r.Err,
			Passed:      requestPassed(r),
			Tests:       jsonTestResults(r.Tests),
		})
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

func jsonTestResults(tests []scripting.TestResult) []jsonTestResult {
	if len(tests) == 0 {
		return nil
	}
	out := make([]jsonTestResult, len(tests))
	for i, tr := range tests {
		out[i] = jsonTestResult{Name: tr.Name, Passed: tr.Passed, Error: tr.Error}
	}
	return out
}
