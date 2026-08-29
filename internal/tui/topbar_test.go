package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// TestTopBarRightText_ShowsThemeClockWorkspaceAndProfile guards the top
// bar's right-hand cluster - a user asked for the platform name alone on
// the left, and theme/clock/workspace/profile grouped on the right ending
// in a profile control, rather than everything crammed onto the left the
// way it used to be.
func TestTopBarRightText_ShowsThemeClockWorkspaceAndProfile(t *testing.T) {
	now := time.Date(2026, 1, 1, 22, 1, 50, 0, time.UTC)
	got := topBarRightText("Personal", "Gruvbox", "staging", now)

	for _, want := range []string{"theme: Gruvbox", "22:01:50", "workspace: Personal", "staging", profileIconLabel} {
		if !strings.Contains(got, want) {
			t.Errorf("got %q, want it to contain %q", got, want)
		}
	}
}

// TestTopBarRightText_OmitsWorkspaceWhenUnset mirrors the old left-side
// behavior (see workspaceLabelAt's doc comment) now that the segment lives
// on the right - a bare New() in a test has no workspace registry wired up,
// so there's no name to show.
func TestTopBarRightText_OmitsWorkspaceWhenUnset(t *testing.T) {
	got := topBarRightText("", "Gruvbox", "", time.Time{})
	if strings.Contains(got, "workspace") {
		t.Errorf("got %q, want no mention of \"workspace\" when no workspace name is set", got)
	}
}

// TestTopBarRightText_ShowsEnvironmentPillOrNone guards the environment
// pill's own move into the top bar (see envDropdownAnchor) - "none" when
// nothing is active, the name when something is.
func TestTopBarRightText_ShowsEnvironmentPillOrNone(t *testing.T) {
	if got := topBarRightText("", "Gruvbox", "", time.Time{}); !strings.Contains(got, "none") {
		t.Errorf("got %q, want it to show \"none\" with no active environment", got)
	}
	if got := topBarRightText("", "Gruvbox", "PROD", time.Time{}); !strings.Contains(got, "PROD") {
		t.Errorf("got %q, want it to show the active environment name", got)
	}
}

func TestTopBarView_RendersExactlyOneLineAtRequestedWidth(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	now := time.Date(2026, 1, 1, 22, 1, 50, 0, time.UTC)

	for _, width := range []int{60, 90, 140, 200} {
		got := m.topBarView(width, now)
		if h := lipgloss.Height(got); h != 1 {
			t.Errorf("width %d: got %d lines, want exactly 1 (topBarView must never wrap)", width, h)
		}
		if w := lipgloss.Width(got); w != width {
			t.Errorf("width %d: got rendered width %d, want %d", width, w, width)
		}
	}
}

// TestTopBarView_ShowsBrandOnTheLeft guards the platform name staying the
// left-most, always-present element regardless of workspace/theme state.
func TestTopBarView_ShowsBrandOnTheLeft(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	got := stripANSI(m.topBarView(80, time.Time{}))
	if !strings.HasPrefix(strings.TrimLeft(got, " "), "parley") {
		t.Errorf("got %q, want it to start with \"parley\"", got)
	}
}
