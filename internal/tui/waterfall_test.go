package tui

import (
	"strings"
	"testing"
	"time"
)

// TestTimingWaterfall_LongerPhaseGetsWiderBar guards the core waterfall
// property: a phase that took more time renders a wider bar than one that
// took less.
func TestTimingWaterfall_LongerPhaseGetsWiderBar(t *testing.T) {
	phases := []timingPhase{
		{label: "DNS Lookup", d: 2 * time.Millisecond, color: "#59c2ff"},
		{label: "Server Wait", d: 16 * time.Millisecond, color: "#e6b450"},
	}
	out := stripANSI(timingWaterfall(phases, 20*time.Millisecond, 20))
	lines := strings.Split(out, "\n")
	dns := strings.Count(lines[0], "=")
	wait := strings.Count(lines[1], "=")
	if wait <= dns {
		t.Fatalf("expected Server Wait bar (%d cells) wider than DNS bar (%d cells)", wait, dns)
	}
}

// TestTimingWaterfall_PhasesStepToTheRight guards the waterfall shape: each
// later phase's bar starts further right, offset by the phases before it.
func TestTimingWaterfall_PhasesStepToTheRight(t *testing.T) {
	phases := []timingPhase{
		{label: "DNS Lookup", d: 5 * time.Millisecond, color: "#59c2ff"},
		{label: "Server Wait", d: 5 * time.Millisecond, color: "#e6b450"},
	}
	out := stripANSI(timingWaterfall(phases, 10*time.Millisecond, 20))
	lines := strings.Split(out, "\n")
	off := func(s string) int { return strings.Index(s, "=") }
	if off(lines[1]) <= off(lines[0]) {
		t.Fatalf("expected second phase bar to start further right: line0=%q line1=%q", lines[0], lines[1])
	}
}

// TestTimingWaterfall_PhaseThatHappenedIsNeverInvisible guards the min-one-cell
// rule: a tiny phase against a huge total still shows at least one bar cell,
// never a blank row that reads as "did not happen".
func TestTimingWaterfall_PhaseThatHappenedIsNeverInvisible(t *testing.T) {
	phases := []timingPhase{{label: "DNS Lookup", d: 1 * time.Microsecond, color: "#59c2ff"}}
	out := stripANSI(timingWaterfall(phases, 10*time.Second, 20))
	if !strings.Contains(out, "=") {
		t.Fatalf("expected at least one bar cell for a phase that happened, got %q", out)
	}
}
