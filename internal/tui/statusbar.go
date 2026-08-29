package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// isProdEnv reports whether an environment name looks like production, so the
// status line can paint it with the danger color instead of the normal env
// color - the terminal equivalent of posting's blinking PRODUCTION badge.
func isProdEnv(name string) bool {
	return strings.Contains(strings.ToLower(name), "prod")
}

// statusChip renders an HTTP status as a filled, color-coded chip (2xx green,
// 3xx amber, 4xx orange, 5xx red - the same class colors statusClassStyle
// uses), so a landed response reads at a glance. The fill is a padded
// background rather than a box-drawing shape, keeping to the repo's ASCII-only
// output discipline.
func statusChip(code int, status string) string {
	color := activeTheme.FgDim
	switch {
	case code >= 200 && code < 300:
		color = activeTheme.Status2xx
	case code >= 300 && code < 400:
		color = activeTheme.Status3xx
	case code >= 400 && code < 500:
		color = activeTheme.Status4xx
	case code >= 500:
		color = activeTheme.Status5xx
	}
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(activeTheme.AccentFg)).
		Background(lipgloss.Color(color)).
		Padding(0, 1).
		Render(status)
}

// envChip renders the active environment as a filled chip - the env color for
// normal envs, the error color for production so it's impossible to miss which
// environment a request is about to hit.
func envChip(name string) string {
	if name == "" {
		name = "none"
	}
	bg := envPillColor(name)
	fg := activeTheme.AccentFg
	if isProdEnv(name) {
		bg = activeTheme.Err
		fg = activeTheme.BG
	}
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(fg)).
		Background(lipgloss.Color(bg)).
		Padding(0, 1).
		Render(name)
}

// modeChip renders the current interaction mode (NORMAL, SEARCH, ...) as the
// leftmost filled accent block of the status line - yazi's mode segment.
func modeChip(mode string) string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(activeTheme.AccentFg)).
		Background(lipgloss.Color(activeTheme.Accent)).
		Padding(0, 1).
		Render(mode)
}

// segmentedStatusLine composes the bottom bar: a left cluster (mode chip, env
// chip, dim context) and a right cluster (whatever the caller passes, e.g. the
// landed response chips plus key hints), padded so the two clusters sit at
// opposite edges and the whole line is exactly width columns. If the content
// can't fit, it's clipped rather than wrapped - a wrapped status line would
// push a row off the alt-screen (the same failure the top bar guards against).
func segmentedStatusLine(width int, mode, envName, ctx, right string) string {
	if width < 1 {
		width = 1
	}
	base := modeChip(mode) + " " + envChip(envName)
	rightStyled := labelStyle.Render(right)
	rw := lipgloss.Width(rightStyled)

	// Prefer showing the context label, but shed it before the right cluster:
	// the landed response status/latency on the right matters more than a
	// breadcrumb when space is tight.
	left := base
	if ctx != "" && lipgloss.Width(base)+2+lipgloss.Width(labelStyle.Render(ctx))+rw+1 <= width {
		left = base + "  " + labelStyle.Render(ctx)
	}

	gap := width - lipgloss.Width(left) - rw
	if gap < 1 {
		// Even without the context both clusters plus a gap don't fit - keep
		// the mode/env block (always meaningful) and clip to width so nothing
		// wraps onto a second row.
		return ansi.Cut(left, 0, width)
	}
	return left + strings.Repeat(" ", gap) + rightStyled
}

// responseStatusRight builds the status line's right cluster from a landed
// response: the status chip, latency, and payload size, followed by the key
// hints - the "what just happened" summary posting keeps in the URL bar,
// placed in the bar here.
func responseStatusRight(code int, status string, latencyMS int64, size, keys string) string {
	parts := []string{
		statusChip(code, status),
		labelStyle.Render(formatDurationMS(latencyMS)),
	}
	if size != "" {
		parts = append(parts, labelStyle.Render(size))
	}
	if keys != "" {
		parts = append(parts, labelStyle.Render(keys))
	}
	return strings.Join(parts, "  ")
}
