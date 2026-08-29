package tui

import (
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
	history "github.com/dhitalkamal/parley/internal/history/domain"
	"reflect"
	"strings"
	"testing"
	"time"
)

// TestLatencySparkline_MapsLowToHighAcrossTheRamp guards the actual point of
// a sparkline: a clearly rising latency sequence should render as a rising
// sequence of ramp characters, not something flat or reversed.
func TestLatencySparkline_MapsLowToHighAcrossTheRamp(t *testing.T) {
	got := latencySparkline([]int64{10, 20, 30, 40, 50})
	if len(got) != 5 {
		t.Fatalf("got %d chars, want 5 (one per value)", len(got))
	}
	if rune(got[0]) != sparklineRamp[0] {
		t.Errorf("first char = %q, want the ramp's lowest-weight char %q for the smallest value", got[0], sparklineRamp[0])
	}
	last := sparklineRamp[len(sparklineRamp)-1]
	if rune(got[len(got)-1]) != last {
		t.Errorf("last char = %q, want the ramp's highest-weight char %q for the largest value", got[len(got)-1], last)
	}
}

// TestLatencySparkline_ConstantValuesDoNotPanic guards the zero-span divide
// case: every value identical must still render, not crash or divide by
// zero.
func TestLatencySparkline_ConstantValuesDoNotPanic(t *testing.T) {
	got := latencySparkline([]int64{15, 15, 15})
	if len(got) != 3 {
		t.Fatalf("got %d chars, want 3", len(got))
	}
	if got[0] != got[1] || got[1] != got[2] {
		t.Errorf("got %q, want identical chars for identical values", got)
	}
}

// TestLatencySparkline_EmptyInputReturnsEmpty guards the no-data case (a
// request with no recorded runs in the trend window).
func TestLatencySparkline_EmptyInputReturnsEmpty(t *testing.T) {
	if got := latencySparkline(nil); got != "" {
		t.Errorf("latencySparkline(nil) = %q, want empty", got)
	}
}

// TestLatencySparkline_UsesOnlyASCII guards this project's file-content
// policy: no Unicode block-drawing glyphs (the usual sparkline approach),
// plain ASCII density ramp instead.
func TestLatencySparkline_UsesOnlyASCII(t *testing.T) {
	got := latencySparkline([]int64{5, 100, 5, 100, 50})
	for _, r := range got {
		if r > 127 {
			t.Errorf("got %q, want ASCII-only characters (found %q)", got, r)
		}
	}
}

// TestComputeDashboardPerfRows_CarriesTheLatencyHistoryForItsSparkline
// guards the dashboard-level wiring: each per-request performance row
// should carry the full ordered latency history a sparkline is rendered
// from, alongside its pass rate.
func TestComputeDashboardPerfRows_CarriesTheLatencyHistoryForItsSparkline(t *testing.T) {
	runs := []history.CollectionRunEntry{
		{Time: time.Date(2026, 1, 1, 3, 0, 0, 0, time.UTC), Results: []execution.RunResult{
			{RequestPath: "010_get.json", Method: "GET", URL: "https://example.com/users", StatusCode: 200, ElapsedMS: 50},
		}},
		{Time: time.Date(2026, 1, 1, 2, 0, 0, 0, time.UTC), Results: []execution.RunResult{
			{RequestPath: "010_get.json", Method: "GET", URL: "https://example.com/users", StatusCode: 200, ElapsedMS: 5},
		}},
	}

	rows := computeDashboardPerfRows(runs, trendRunWindow)
	if len(rows) != 1 {
		t.Fatalf("got %d trend rows, want 1", len(rows))
	}
	want := []int64{5, 50} // oldest-to-newest, so the sparkline reads left-to-right
	if !reflect.DeepEqual(rows[0].ms, want) {
		t.Errorf("ms = %v, want %v", rows[0].ms, want)
	}
	if got := latencySparkline(rows[0].ms); !strings.Contains(got, string(sparklineRamp[0])) {
		t.Errorf("sparkline %q built from ms, want it to include the ramp's lowest char", got)
	}
}
