package tui

import (
	"fmt"
	"strings"
	"time"
)

// testsText renders pm.test(...) results for the Response panel's Tests
// mode (see response.go's refresh).
func (rv responseView) testsText() string {
	if rv.testScriptError != "" {
		return errStyle.Render("Script error: " + rv.testScriptError)
	}
	if len(rv.testResults) == 0 {
		return labelStyle.Render("(no tests)")
	}
	passed := 0
	var b strings.Builder
	for _, r := range rv.testResults {
		mark := errStyle.Render("FAIL")
		if r.Passed {
			mark = statusStyle.Render("PASS")
			passed++
		}
		fmt.Fprintf(&b, "%s  %s\n", mark, r.Name)
		if !r.Passed && r.Error != "" {
			fmt.Fprintf(&b, "      %s\n", errStyle.Render(r.Error))
		}
	}
	header := labelStyle.Render(fmt.Sprintf("%d/%d passed", passed, len(rv.testResults)))
	return header + "\n\n" + strings.TrimRight(b.String(), "\n")
}

// timelineText renders the HTTP round trip's phase breakdown for the
// Response panel's Timeline mode. A phase left at zero didn't happen (a
// literal-IP URL has no DNS lookup, plain HTTP has no TLS handshake, a
// reused connection has no TCP connect) rather than being instant, so it's
// left out instead of shown as 0s.
func (rv responseView) timelineText() string {
	return rv.timelineTextWidth(rv.vp.Width)
}

// timelineTextWidth is timelineText sized to an explicit width, so the same
// waterfall can render full-size in the Timeline tab and compact in the
// Response info column (see resppanel.go).
func (rv responseView) timelineTextWidth(width int) string {
	dnsC, tcpC, tlsC, waitC, xferC := waterfallPhaseColors()
	all := []timingPhase{
		{"DNS Lookup", rv.timing.DNSLookup, dnsC},
		{"TCP Connect", rv.timing.TCPConnect, tcpC},
		{"TLS Handshake", rv.timing.TLSHandshake, tlsC},
		{"Server Wait", rv.timing.ServerWait, waitC},
		{"Content Transfer", rv.timing.ContentTransfer, xferC},
	}
	// A phase left at zero didn't happen (literal-IP URL has no DNS, plain
	// HTTP has no TLS, a reused connection has no TCP connect), so it's left
	// out rather than drawn as a misleading instant bar.
	var phases []timingPhase
	for _, p := range all {
		if p.d > 0 {
			phases = append(phases, p)
		}
	}
	if len(phases) == 0 {
		return labelStyle.Render("(no timing data)")
	}
	// Size the bar track to the given width, leaving room for the fixed label
	// column, the trailing duration text, and a little breathing room.
	track := width - waterfallLabelWidth - 12
	if track < 8 {
		track = waterfallTrackWidth
	}
	wf := timingWaterfall(phases, time.Duration(rv.elapsedMS)*time.Millisecond, track)
	var b strings.Builder
	b.WriteString(wf)
	fmt.Fprintf(&b, "\n\n%-*s%d ms\n", waterfallLabelWidth, "Total", rv.elapsedMS)
	return strings.TrimRight(b.String(), "\n")
}
