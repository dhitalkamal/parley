package cli

import (
	"encoding/xml"
	"fmt"
	"io"

	execution "github.com/dhitalkamal/parley/internal/execution/domain"
)

type junitTestSuites struct {
	XMLName xml.Name         `xml:"testsuites"`
	Suites  []junitTestSuite `xml:"testsuite"`
}

type junitTestSuite struct {
	Name     string          `xml:"name,attr"`
	Tests    int             `xml:"tests,attr"`
	Failures int             `xml:"failures,attr"`
	Cases    []junitTestCase `xml:"testcase"`
}

type junitTestCase struct {
	Name      string        `xml:"name,attr"`
	Classname string        `xml:"classname,attr"`
	Time      string        `xml:"time,attr"`
	Failure   *junitFailure `xml:"failure,omitempty"`
}

type junitFailure struct {
	Message string `xml:"message,attr"`
}

// WriteJUnit renders results as JUnit XML, one testsuite named suiteName
// containing one testcase per pm.test assertion. A request with no test
// script still gets a single testcase (pass/fail on whether it sent
// successfully), so every request shows up in CI test-result dashboards
// even if it defines no assertions of its own.
func WriteJUnit(w io.Writer, results []execution.RunResult, suiteName string) error {
	suite := junitTestSuite{Name: suiteName}
	for _, r := range results {
		label := requestLabel(r)
		elapsed := fmt.Sprintf("%.3f", float64(r.ElapsedMS)/1000)

		if r.Err != "" {
			suite.Cases = append(suite.Cases, junitTestCase{
				Name: label, Classname: "request", Time: elapsed,
				Failure: &junitFailure{Message: r.Err},
			})
			suite.Failures++
			suite.Tests++
			continue
		}

		if len(r.Tests) == 0 {
			suite.Cases = append(suite.Cases, junitTestCase{Name: label, Classname: "request", Time: elapsed})
			suite.Tests++
			continue
		}

		for _, tr := range r.Tests {
			tc := junitTestCase{Name: fmt.Sprintf("%s: %s", label, tr.Name), Classname: label, Time: elapsed}
			if !tr.Passed {
				tc.Failure = &junitFailure{Message: tr.Error}
				suite.Failures++
			}
			suite.Cases = append(suite.Cases, tc)
			suite.Tests++
		}
	}

	doc := junitTestSuites{Suites: []junitTestSuite{suite}}
	if _, err := io.WriteString(w, xml.Header); err != nil {
		return err
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	return enc.Encode(doc)
}

func requestLabel(r execution.RunResult) string {
	if r.Method != "" || r.URL != "" {
		return fmt.Sprintf("%s %s", r.Method, r.URL)
	}
	return r.RequestPath
}
