package cli

import (
	"bytes"
	"encoding/xml"
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
	scripting "github.com/dhitalkamal/parley/internal/scripting/domain"
	"testing"
)

type junitSuitesDoc struct {
	XMLName xml.Name        `xml:"testsuites"`
	Suites  []junitSuiteDoc `xml:"testsuite"`
}

type junitSuiteDoc struct {
	Tests    int            `xml:"tests,attr"`
	Failures int            `xml:"failures,attr"`
	Cases    []junitCaseDoc `xml:"testcase"`
}

type junitCaseDoc struct {
	Name    string        `xml:"name,attr"`
	Failure *junitFailDoc `xml:"failure"`
}

type junitFailDoc struct {
	Message string `xml:"message,attr"`
}

func TestWriteJUnit_OneTestcasePerAssertion(t *testing.T) {
	var buf bytes.Buffer
	results := []execution.RunResult{
		{RequestPath: "a.json", Method: "GET", URL: "https://example.com/a", StatusCode: 200,
			Tests: []scripting.TestResult{
				{Name: "status is 200", Passed: true},
				{Name: "has body", Passed: false, Error: "body was empty"},
			}},
	}
	if err := WriteJUnit(&buf, results, "parley"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var doc junitSuitesDoc
	if err := xml.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("output is not valid xml: %v\n%s", err, buf.String())
	}
	if len(doc.Suites) != 1 {
		t.Fatalf("got %d suites, want 1", len(doc.Suites))
	}
	suite := doc.Suites[0]
	if suite.Tests != 2 || suite.Failures != 1 {
		t.Errorf("suite totals = %+v, want tests=2 failures=1", suite)
	}
	if len(suite.Cases) != 2 {
		t.Fatalf("got %d testcases, want 2", len(suite.Cases))
	}
	if suite.Cases[0].Failure != nil {
		t.Errorf("expected first testcase to pass, got failure %+v", suite.Cases[0].Failure)
	}
	if suite.Cases[1].Failure == nil || suite.Cases[1].Failure.Message != "body was empty" {
		t.Errorf("expected second testcase to fail with message, got %+v", suite.Cases[1].Failure)
	}
}

func TestWriteJUnit_RequestWithNoTestsStillProducesOneTestcase(t *testing.T) {
	var buf bytes.Buffer
	results := []execution.RunResult{
		{RequestPath: "a.json", Method: "GET", URL: "https://example.com/a", StatusCode: 200},
	}
	if err := WriteJUnit(&buf, results, "parley"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var doc junitSuitesDoc
	if err := xml.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("output is not valid xml: %v", err)
	}
	if len(doc.Suites) != 1 || len(doc.Suites[0].Cases) != 1 {
		t.Fatalf("got %+v, want exactly one testcase for a request with no test script", doc)
	}
	if doc.Suites[0].Cases[0].Failure != nil {
		t.Errorf("expected the testcase to pass, got %+v", doc.Suites[0].Cases[0].Failure)
	}
}

func TestWriteJUnit_SendErrorProducesFailingTestcase(t *testing.T) {
	var buf bytes.Buffer
	results := []execution.RunResult{
		{RequestPath: "a.json", Method: "GET", URL: "https://example.com/a", Err: "connection refused"},
	}
	if err := WriteJUnit(&buf, results, "parley"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var doc junitSuitesDoc
	if err := xml.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("output is not valid xml: %v", err)
	}
	if doc.Suites[0].Failures != 1 {
		t.Errorf("suite failures = %d, want 1", doc.Suites[0].Failures)
	}
	if doc.Suites[0].Cases[0].Failure == nil || doc.Suites[0].Cases[0].Failure.Message != "connection refused" {
		t.Errorf("expected a failing testcase with the send error, got %+v", doc.Suites[0].Cases[0])
	}
}
