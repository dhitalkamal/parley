package cli

import (
	"fmt"
	"io"

	execution "github.com/dhitalkamal/parley/internal/execution/domain"
)

// WriteText renders results as a human-readable summary for a terminal or
// CI log: one line per request, one indented line per failing assertion,
// and a final pass/fail count.
func WriteText(w io.Writer, results []execution.RunResult) {
	passed := 0
	for _, r := range results {
		ok := requestPassed(r)
		if ok {
			passed++
			fmt.Fprintf(w, "PASS  %s  %s  %s  (%dms)\n", r.Method, r.URL, r.Status, r.ElapsedMS)
		} else {
			fmt.Fprintf(w, "FAIL  %s  %s  (%dms)\n", r.Method, r.URL, r.ElapsedMS)
		}
		if r.Err != "" {
			fmt.Fprintf(w, "  error: %s\n", r.Err)
		}
		for _, tr := range r.Tests {
			if tr.Passed {
				fmt.Fprintf(w, "  ok %s\n", tr.Name)
			} else {
				fmt.Fprintf(w, "  fail %s: %s\n", tr.Name, tr.Error)
			}
		}
	}
	fmt.Fprintf(w, "\n%d passed, %d failed, %d total\n", passed, len(results)-passed, len(results))
}
