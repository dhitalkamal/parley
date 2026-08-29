package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// clockTickMsg drives the top bar's clock - a self-rearming tea.Tick, same
// pattern any bubbletea clock/spinner uses.
type clockTickMsg time.Time

func clockTickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return clockTickMsg(t) })
}

// topBarBrand is the top bar's left-most, always-present element - the
// platform name alone, with everything else (theme, clock, workspace,
// profile) grouped on the right instead (see topBarRightText).
const topBarBrand = "parley"

// profileIconLabel is the top bar's rightmost control - a plain-text
// stand-in for a profile icon (this app has no user accounts to show a
// real avatar for), clicking it cycles the theme (see mouse.go's zoneTopBar
// handling) - the closest existing concept to a profile/settings menu.
const profileIconLabel = "[Theme]"

// topBarRightText is the pure, testable half of the top bar - theme,
// clock, workspace, the environment pill, and the profile control, in that
// order. workspaceName is left out entirely when empty (no workspace
// registry wired up, e.g. a bare New() in a test) rather than rendering a
// blank "workspace: " segment. The environment pill used to be a plain url-
// row box only - it now lives here too as the primary switcher (see
// envDropdownAnchor), with the url row keeping a color-matched echo instead
// (see urlRowView).
func topBarRightText(workspaceName, themeName, envName string, now time.Time) string {
	segments := []string{"theme: " + themeName, now.Format("15:04:05")}
	if workspaceName != "" {
		segments = append(segments, "workspace: "+workspaceName)
	}
	segments = append(segments, envSegmentText(envName), profileIconLabel)
	return strings.Join(segments, "  |  ")
}

// topBarContent builds the top bar's unstyled content line - split out
// from topBarView so mouse.go's click handling can search the exact same
// text for the workspace/env/profile segments instead of recomputing the
// layout math separately, which could drift out of sync with what's
// actually rendered.
func (m Model) topBarContent(width int, now time.Time) string {
	right := topBarRightText(m.activeWorkspaceName, activeTheme.Name, m.activeEnvName, now)

	innerWidth := width - 2 // topBarStyle's Padding(0, 1)
	gap := innerWidth - lipgloss.Width(topBarBrand) - lipgloss.Width(right)
	if gap < 1 {
		// The right-hand cluster (now with the env pill added) plus a
		// 1-column gap no longer fits a narrow terminal - clip rather than
		// let it overflow: topBarView deliberately has no .Width() (see its
		// own comment), so an unclipped line here would render wider than
		// the terminal instead of wrapping or clipping visibly.
		return ansi.Cut(topBarBrand+" "+right, 0, innerWidth)
	}
	return topBarBrand + strings.Repeat(" ", gap) + right
}

// colorizeEnvSegment wraps the environment pill segment within an already-
// built plain top bar line in its environment-matched color, leaving
// everything else untouched - the same find-the-substring-then-style
// approach response.go's highlightSearch uses, so topBarContent itself
// (what mouse.go's click hit-testing searches) can stay plain, uncolored
// text.
func colorizeEnvSegment(line, envName string) string {
	seg := envSegmentText(envName)
	idx := strings.Index(line, seg)
	if idx < 0 {
		return line
	}
	styled := lipgloss.NewStyle().Foreground(lipgloss.Color(envPillColor(envName))).Render(seg)
	return line[:idx] + styled + line[idx+len(seg):]
}

// topBarView renders the persistent top row: the platform name on the
// left, theme/clock/workspace/profile on the right - the design
// reference's topbar, translated to a single-line chrome strip since a
// terminal has no draggable window furniture to separate it from.
// Workspace is shown here (rather than only behind the F4 shortcut)
// because it previously had no visible entry point anywhere on screen - a
// user only found it by asking; clicking the label opens the same switcher
// F4 does, and clicking the profile control cycles the theme (see
// workspaceLabelAt/profileIconAt/handleMouseClick in mouse.go).
func (m Model) topBarView(width int, now time.Time) string {
	content := colorizeEnvSegment(m.topBarContent(width, now), m.activeEnvName)
	// No .Width() here deliberately: content is already padded to exactly
	// innerWidth, and Style.Width() on a borderless style word-wraps
	// instead of clipping once the padded content is a hair over budget -
	// which silently split this single line into two.
	return topBarStyle.Render(content)
}
