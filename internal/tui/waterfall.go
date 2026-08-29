package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// timingPhase is one segment of the HTTP round trip drawn in the waterfall.
// color is a theme hex string; label is the fixed left-column caption.
type timingPhase struct {
	label string
	d     time.Duration
	color string
}

// waterfallLabelWidth is the fixed left column every phase label is padded
// into, so the bars all start at the same column and the offsets read as a
// clean waterfall.
const waterfallLabelWidth = 16

// waterfallTrackWidth is the default column budget for the bar region after
// the label - the response Timeline tab passes its own measured width, this
// is the fallback for callers that don't.
const waterfallTrackWidth = 28

// timingWaterfall renders phases as a stacked waterfall: each phase's bar is
// proportional to its share of elapsed and is offset from the left by the sum
// of the phases before it, so the bars step to the right like a real request
// timeline. elapsed is the denominator, so any untracked tail time shows as
// empty track rather than stretching the bars. Bars use a visible ASCII fill
// ('=') colored via the theme rather than a colored background, so they still
// read on a terminal with no truecolor support (the repo's ASCII-only
// discipline, same reason sparkline.go uses an ASCII ramp).
func timingWaterfall(phases []timingPhase, elapsed time.Duration, track int) string {
	if track < 1 {
		track = 1
	}
	// col maps a duration onto a column count within the track.
	col := func(d time.Duration) int {
		if elapsed <= 0 {
			return 0
		}
		c := int(float64(d) / float64(elapsed) * float64(track))
		if c < 0 {
			c = 0
		}
		return c
	}
	var b strings.Builder
	var cum time.Duration
	for _, p := range phases {
		start := col(cum)
		length := col(p.d)
		if p.d > 0 && length < 1 {
			// a phase that happened is never invisible - round it up to one
			// cell so a fast-but-real phase doesn't render as a blank row.
			length = 1
		}
		if start > track {
			start = track
		}
		if start+length > track {
			length = track - start
		}
		bar := strings.Repeat(" ", start) +
			lipgloss.NewStyle().Foreground(lipgloss.Color(p.color)).Render(strings.Repeat("=", length))
		fmt.Fprintf(&b, "%-*s%s %s\n", waterfallLabelWidth, p.label, bar, p.d.Round(time.Microsecond))
		cum += p.d
	}
	return strings.TrimRight(b.String(), "\n")
}

// waterfallPhaseColors maps the round-trip phases to distinct theme colors so
// the eye can tell DNS from TLS from server-wait at a glance. Kept as a
// function (not a package var) so it always reflects the current theme after
// a cycle.
func waterfallPhaseColors() (dns, tcp, tls, wait, transfer string) {
	return activeTheme.Accent2, activeTheme.MethodPut, activeTheme.MethodPost, activeTheme.Warn, activeTheme.Success
}
