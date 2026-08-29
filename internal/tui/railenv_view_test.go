package tui

import (
	"strings"
	"testing"
)

// TestRailEnvTop_ShowsKeysAndValues guards the selector-card design: the
// selector shows, each variable's key AND value show, but a secret's value is
// masked (never rendered literally). Uses the globals section, which is always
// visible.
func TestRailEnvTop_ShowsKeysAndValues(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.railGlobals.SetRows([]envVarRow{
		{Key: "baseUrl", Value: "http://api", Enabled: true},
		{Key: "token", Value: "s3cr3t", Enabled: true, Secret: true},
	})

	got := stripANSI(railEnvTop(m, true))

	if !strings.Contains(got, "Active:") {
		t.Errorf("want the active-env selector, got %q", got)
	}
	if !strings.Contains(got, "baseUrl") || !strings.Contains(got, "token") {
		t.Errorf("want both variable keys listed, got %q", got)
	}
	if !strings.Contains(got, "http://api") {
		t.Errorf("want the plain variable value shown, got %q", got)
	}
	if strings.Contains(got, "s3cr3t") {
		t.Errorf("secret value must be masked, never rendered literally, got %q", got)
	}
}

// TestRailEnvTop_AlwaysShowsGlobalsSection guards that GLOBALS is a section in
// every environment view, not just when a named env is active.
func TestRailEnvTop_AlwaysShowsGlobalsSection(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	got := stripANSI(railEnvTop(m, false))
	if !strings.Contains(got, "GLOBALS") {
		t.Errorf("want a GLOBALS section shown, got %q", got)
	}
}

// TestComposeRailEnv_PinsBottomToLastRows guards the layout fix: the bottom
// section (the add/edit form) sits on the final rows of the box regardless of
// how short the top section is, with blank padding filling the gap.
func TestComposeRailEnv_PinsBottomToLastRows(t *testing.T) {
	got := composeRailEnv("top1\ntop2", "formA\nformB", 8)
	lines := strings.Split(got, "\n")
	if len(lines) != 8 {
		t.Fatalf("want 8 lines to fill innerHeight, got %d: %q", len(lines), lines)
	}
	if lines[0] != "top1" || lines[1] != "top2" {
		t.Errorf("top section should stay at the top, got %q", lines[:2])
	}
	if lines[6] != "formA" || lines[7] != "formB" {
		t.Errorf("bottom section should be pinned to the last rows, got %q", lines[6:])
	}
	for i := 2; i < 6; i++ {
		if lines[i] != "" {
			t.Errorf("gap row %d should be blank padding, got %q", i, lines[i])
		}
	}
}
