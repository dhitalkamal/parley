package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// TestIsProdEnv flags production-like environment names so the status line can
// render them with the danger color (posting's blinking PRODUCTION cue).
func TestIsProdEnv(t *testing.T) {
	for _, name := range []string{"prod", "PROD", "production", "live-prod"} {
		if !isProdEnv(name) {
			t.Errorf("isProdEnv(%q) = false, want true", name)
		}
	}
	for _, name := range []string{"local", "staging", "dev", ""} {
		if isProdEnv(name) {
			t.Errorf("isProdEnv(%q) = true, want false", name)
		}
	}
}

// TestStatusChip shows the response status text inside the chip.
func TestStatusChip(t *testing.T) {
	got := stripANSI(statusChip(200, "200 OK"))
	if !strings.Contains(got, "200 OK") {
		t.Errorf("statusChip() = %q, want it to contain %q", got, "200 OK")
	}
}

// TestSegmentedStatusLine_FillsWidthAndOrders guards the two-cluster layout:
// left content (mode, env) sits left of the right content, and the whole line
// is exactly width columns wide so it never wraps or leaves a ragged gap.
func TestSegmentedStatusLine_FillsWidthAndOrders(t *testing.T) {
	line := segmentedStatusLine(80, "NORMAL", "local", "Billing API", "ctrl+k palette")
	plain := stripANSI(line)
	if w := lipgloss.Width(plain); w != 80 {
		t.Fatalf("segmentedStatusLine width = %d, want 80\nline=%q", w, plain)
	}
	mode := strings.Index(plain, "NORMAL")
	right := strings.Index(plain, "ctrl+k palette")
	if mode < 0 || right < 0 || mode >= right {
		t.Fatalf("expected NORMAL left of the right cluster: mode=%d right=%d in %q", mode, right, plain)
	}
}

// TestSegmentedStatusLine_Clips guards against overflow: when the content is
// wider than the terminal, the line is clipped to width rather than wrapping
// onto a second row (which would scroll the alt-screen).
func TestSegmentedStatusLine_Clips(t *testing.T) {
	line := segmentedStatusLine(20, "NORMAL", "production", "A very long context label", "ctrl+k palette   ? keys")
	if w := lipgloss.Width(stripANSI(line)); w > 20 {
		t.Fatalf("segmentedStatusLine width = %d, want <= 20", w)
	}
}
