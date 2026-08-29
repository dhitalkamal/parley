package tui

import (
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
	scripting "github.com/dhitalkamal/parley/internal/scripting/domain"
	"strings"
	"testing"
	"time"
)

// TestResponseView_TabBarListsTestsAndTimeline guards the tab bar listing both
// Tests and the Timeline waterfall tab alongside Body/Headers/Cookies.
func TestResponseView_TabBarListsTestsAndTimeline(t *testing.T) {
	rv := newResponseView()

	got := stripANSI(rv.TabBarText())
	if !strings.Contains(got, "Tests") || !strings.Contains(got, "Timeline") {
		t.Errorf("TabBarText() = %q, want Tests and Timeline tabs listed", got)
	}
}

// TestResponseView_TestsTabShowsPassAndFailResults guards the Tests mode
// folding in what used to be a separate stacked panel (testspanel.go).
func TestResponseView_TestsTabShowsPassAndFailResults(t *testing.T) {
	rv := newResponseView()
	rv.SetSize(60, 20)
	rv.SetResponse(execution.Response{StatusCode: 200, Status: "200 OK"}, 10)
	rv.SetTestResults([]scripting.TestResult{
		{Name: "status is 200", Passed: true},
		{Name: "has id", Passed: false, Error: "expected a string"},
	}, "")
	rv.SetMode(viewTests)

	got := stripANSI(rv.ContentView())
	if !strings.Contains(got, "status is 200") || !strings.Contains(got, "PASS") {
		t.Errorf("ContentView() = %q, want the passing test listed", got)
	}
	if !strings.Contains(got, "has id") || !strings.Contains(got, "FAIL") || !strings.Contains(got, "expected a string") {
		t.Errorf("ContentView() = %q, want the failing test and its error listed", got)
	}
}

// TestResponseView_TestsTabShowsScriptError guards a script that errored
// outright (as opposed to individual test assertions failing).
func TestResponseView_TestsTabShowsScriptError(t *testing.T) {
	rv := newResponseView()
	rv.SetSize(60, 20)
	rv.SetResponse(execution.Response{StatusCode: 200, Status: "200 OK"}, 10)
	rv.SetTestResults(nil, "unexpected token")
	rv.SetMode(viewTests)

	got := stripANSI(rv.ContentView())
	if !strings.Contains(got, "unexpected token") {
		t.Errorf("ContentView() = %q, want the script error shown", got)
	}
}

// TestResponseView_TestsTabShowsPlaceholderWhenNoTestScript guards the case
// where a request has no test script at all - Tests is a real tab now
// rather than a panel that only appears when there's something to show, so
// it needs its own "nothing here" state.
func TestResponseView_TestsTabShowsPlaceholderWhenNoTestScript(t *testing.T) {
	rv := newResponseView()
	rv.SetSize(60, 20)
	rv.SetResponse(execution.Response{StatusCode: 200, Status: "200 OK"}, 10)
	rv.SetMode(viewTests)

	got := stripANSI(rv.ContentView())
	if !strings.Contains(got, "no tests") {
		t.Errorf("ContentView() = %q, want a \"no tests\" placeholder", got)
	}
}

// TestResponseView_TimelineTabShowsPhaseBreakdown guards the Timeline tab's
// data source: execution.Response.Timing, captured via httptrace.
func TestResponseView_TimelineTabShowsPhaseBreakdown(t *testing.T) {
	rv := newResponseView()
	rv.SetSize(60, 20)
	rv.SetResponse(execution.Response{
		StatusCode: 200, Status: "200 OK",
		Timing: execution.Timing{
			DNSLookup:  2 * time.Millisecond,
			TCPConnect: 3 * time.Millisecond,
			ServerWait: 15 * time.Millisecond,
		},
	}, 20)

	got := stripANSI(rv.timelineText())
	for _, want := range []string{"DNS Lookup", "TCP Connect", "Server Wait", "Total"} {
		if !strings.Contains(got, want) {
			t.Errorf("ContentView() = %q, want %q listed", got, want)
		}
	}
}

// TestResponseView_TimelineTabHidesPhasesThatDidNotHappen guards against a
// fake/estimated value: a phase left at zero (e.g. TLSHandshake for plain
// HTTP) didn't happen, so it must not render as a misleading 0s row.
func TestResponseView_TimelineTabHidesPhasesThatDidNotHappen(t *testing.T) {
	rv := newResponseView()
	rv.SetSize(60, 20)
	rv.SetResponse(execution.Response{
		StatusCode: 200, Status: "200 OK",
		Timing: execution.Timing{ServerWait: 5 * time.Millisecond},
	}, 10)

	got := stripANSI(rv.timelineText())
	if strings.Contains(got, "TLS Handshake") {
		t.Errorf("ContentView() = %q, want no TLS Handshake row when it never happened", got)
	}
}

// TestResponseView_TimelineTabShowsNoDataPlaceholder guards a response with
// no captured timing at all (e.g. one restored from the last-response disk
// cache after an app restart, which doesn't persist Timing).
func TestResponseView_TimelineTabShowsNoDataPlaceholder(t *testing.T) {
	rv := newResponseView()
	rv.SetSize(60, 20)
	rv.SetResponse(execution.Response{StatusCode: 200, Status: "200 OK"}, 10)

	got := stripANSI(rv.timelineText())
	if !strings.Contains(got, "no timing data") {
		t.Errorf("ContentView() = %q, want a \"no timing data\" placeholder", got)
	}
}
