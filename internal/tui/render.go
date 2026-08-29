package tui

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	jsonKeyStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("81"))
	jsonStringStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	jsonNumberStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	jsonBoolStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true)
	jsonNullStyle   = lipgloss.NewStyle().Faint(true)
)

// PrettyJSON indents body as JSON, reporting ok=false if it isn't valid JSON.
func PrettyJSON(body []byte) (string, bool) {
	var buf bytes.Buffer
	if json.Indent(&buf, body, "", "  ") != nil {
		return "", false
	}
	return buf.String(), true
}

// PrettyXML re-indents body as XML, reporting ok=false if it isn't valid XML.
// xml.Decoder alone isn't enough to detect non-XML input: it happily tokenizes
// plain text with no tags as a single CharData token and reports no error, so
// this also requires at least one real element to consider it XML.
func PrettyXML(body []byte) (string, bool) {
	decoder := xml.NewDecoder(bytes.NewReader(body))
	var buf bytes.Buffer
	encoder := xml.NewEncoder(&buf)
	encoder.Indent("", "  ")
	sawElement := false
	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", false
		}
		if _, ok := tok.(xml.StartElement); ok {
			sawElement = true
		}
		if err := encoder.EncodeToken(tok); err != nil {
			return "", false
		}
	}
	if !sawElement {
		return "", false
	}
	if err := encoder.Flush(); err != nil || buf.Len() == 0 {
		return "", false
	}
	return buf.String(), true
}

// jsonTokenPattern matches, in preference order, a quoted object key
// (quoted string immediately followed by a colon - json.Indent always
// formats keys this way), a plain quoted string, true/false, null, and
// numbers. Go's regexp alternation is leftmost-first among equal-start
// matches, so the key pattern wins over the plain-string one where both
// would apply.
var jsonTokenPattern = regexp.MustCompile(`"(?:[^"\\]|\\.)*"\s*:|"(?:[^"\\]|\\.)*"|\btrue\b|\bfalse\b|\bnull\b|-?\d+\.?\d*(?:[eE][+-]?\d+)?`)

// HighlightJSON applies lipgloss styling to a pretty-printed JSON string.
// It's a lightweight regex-based highlighter, not a full parser - good
// enough for read-only response display, not meant to survive edge cases
// like a key containing a literal `":`.
func HighlightJSON(pretty string) string {
	return jsonTokenPattern.ReplaceAllStringFunc(pretty, func(tok string) string {
		switch {
		case strings.HasSuffix(tok, ":"):
			return jsonKeyStyle.Render(tok)
		case strings.HasPrefix(tok, `"`):
			return jsonStringStyle.Render(tok)
		case tok == "true" || tok == "false":
			return jsonBoolStyle.Render(tok)
		case tok == "null":
			return jsonNullStyle.Render(tok)
		default:
			return jsonNumberStyle.Render(tok)
		}
	})
}

// DetectAndRender renders a response body for display: raw and unchanged
// when pretty is false, otherwise pretty-printed (and, for JSON,
// syntax-highlighted) based on contentType, falling back to sniffing the
// body itself when the content type doesn't say or doesn't match. Unless
// reveal is true, secret-looking key/value pairs (see secretmask.go) are
// masked before any syntax highlighting is applied - highlighting wraps
// each token in its own ANSI escape sequence, which would otherwise break
// the plain-text pattern match.
func DetectAndRender(body []byte, contentType string, pretty bool, reveal bool) string {
	if !pretty {
		return string(body)
	}
	ct := strings.ToLower(contentType)
	switch {
	case strings.Contains(ct, "json"):
		if p, ok := PrettyJSON(body); ok {
			return HighlightJSON(maskSecrets(p, reveal))
		}
	case strings.Contains(ct, "xml"):
		if p, ok := PrettyXML(body); ok {
			return maskSecrets(p, reveal)
		}
	}
	if p, ok := PrettyJSON(body); ok {
		return HighlightJSON(maskSecrets(p, reveal))
	}
	if p, ok := PrettyXML(body); ok {
		return maskSecrets(p, reveal)
	}
	return maskSecrets(string(body), reveal)
}
