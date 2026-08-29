package tui

// sparklineRamp is the ASCII density ramp latencySparkline maps values onto,
// lowest-weight first - plain ASCII punctuation rather than the usual
// Unicode block-drawing glyphs (U+2581-2588), which this project's own
// file-content policy forbids (ASCII punctuation only - see CLAUDE.md).
var sparklineRamp = []rune{'_', '.', '-', '=', '+', '*', '#', '@'}

// latencySparkline renders a sequence of millisecond latencies (oldest to
// newest) as a single-line ASCII density ramp: the fastest value in the
// sequence maps to the ramp's lightest character, the slowest to its
// heaviest, everything else scaled linearly in between - so a request
// getting slower over time reads as a rising trend at a glance, without
// opening each run.
func latencySparkline(values []int64) string {
	if len(values) == 0 {
		return ""
	}
	min, max := values[0], values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	span := max - min

	out := make([]rune, len(values))
	for i, v := range values {
		if span == 0 {
			// Every value identical - nothing to scale, so every point gets
			// the same mid-ramp character rather than a misleading spread.
			out[i] = sparklineRamp[len(sparklineRamp)/2]
			continue
		}
		idx := int(float64(v-min) / float64(span) * float64(len(sparklineRamp)-1))
		out[i] = sparklineRamp[idx]
	}
	return string(out)
}
