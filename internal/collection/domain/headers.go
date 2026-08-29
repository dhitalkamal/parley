package collection

// EffectiveHeaders returns only the enabled rows from the header editor, in order.
func EffectiveHeaders(headers []Header) []Header {
	var out []Header
	for _, h := range headers {
		if h.Enabled {
			out = append(out, h)
		}
	}
	return out
}
