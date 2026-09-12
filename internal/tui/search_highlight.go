package tui

import "strings"

// highlightSearch wraps every case-insensitive occurrence of term in content
// with searchHighlightStyle and returns the styled string.
func highlightSearch(content, term string) string {
	if term == "" {
		return content
	}
	lowerTerm := strings.ToLower(term)
	// strings.ToLower can change a rune's byte length (e.g. U+0130 lowercases
	// to two runes), so byte offsets in the lowercased copy do not line up with
	// content. Build the lowercased copy alongside offsets, which maps every
	// byte index in lower back to the content byte index it came from. offsets
	// gets one trailing entry mapping len(lower) -> len(content) so match end
	// offsets resolve cleanly.
	var lowerBuilder strings.Builder
	offsets := make([]int, 0, len(content)+1)
	for bytePos, r := range content {
		lr := strings.ToLower(string(r))
		lowerBuilder.WriteString(lr)
		for j := 0; j < len(lr); j++ {
			offsets = append(offsets, bytePos)
		}
	}
	offsets = append(offsets, len(content))
	lower := lowerBuilder.String()

	var b strings.Builder
	i := 0
	for {
		idx := strings.Index(lower[i:], lowerTerm)
		if idx < 0 {
			b.WriteString(content[offsets[i]:])
			break
		}
		start := i + idx
		end := start + len(lowerTerm)
		b.WriteString(content[offsets[i]:offsets[start]])
		b.WriteString(searchHighlightStyle.Render(content[offsets[start]:offsets[end]]))
		i = end
	}
	return b.String()
}
