package tui

import (
	"regexp"
	"strings"
	"testing"
)

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}

func TestPrettyJSON_IndentsValidJSON(t *testing.T) {
	got, ok := PrettyJSON([]byte(`{"a":1,"b":[2,3]}`))
	if !ok {
		t.Fatal("expected ok=true for valid JSON")
	}
	if !strings.Contains(got, "\n") {
		t.Errorf("expected indented output, got %q", got)
	}
}

func TestPrettyJSON_RejectsInvalidJSON(t *testing.T) {
	_, ok := PrettyJSON([]byte(`not json`))
	if ok {
		t.Error("expected ok=false for invalid JSON")
	}
}

func TestPrettyXML_IndentsValidXML(t *testing.T) {
	got, ok := PrettyXML([]byte(`<root><a>1</a></root>`))
	if !ok {
		t.Fatal("expected ok=true for valid XML")
	}
	if !strings.Contains(got, "\n") {
		t.Errorf("expected indented output, got %q", got)
	}
	if !strings.Contains(got, "<a>1</a>") {
		t.Errorf("expected content preserved, got %q", got)
	}
}

func TestPrettyXML_RejectsInvalidXML(t *testing.T) {
	_, ok := PrettyXML([]byte(`{"not":"xml"}`))
	if ok {
		t.Error("expected ok=false for non-XML input")
	}
}

func TestHighlightJSON_PreservesContent(t *testing.T) {
	pretty, _ := PrettyJSON([]byte(`{"name":"ada","age":36,"admin":true,"note":null}`))
	highlighted := HighlightJSON(pretty)
	if stripANSI(highlighted) != pretty {
		t.Errorf("highlighting changed content:\ngot  %q\nwant %q", stripANSI(highlighted), pretty)
	}
}

func TestDetectAndRender_RawModeReturnsBodyUnchanged(t *testing.T) {
	body := []byte(`{"a":1}`)
	got := DetectAndRender(body, "application/json", false, true)
	if got != string(body) {
		t.Errorf("got %q, want raw body unchanged", got)
	}
}

func TestDetectAndRender_PrettyJSONByContentType(t *testing.T) {
	got := DetectAndRender([]byte(`{"a":1}`), "application/json; charset=utf-8", true, true)
	if !strings.Contains(stripANSI(got), "\n") {
		t.Errorf("expected pretty-printed JSON, got %q", got)
	}
}

func TestDetectAndRender_SniffsJSONWhenContentTypeMissing(t *testing.T) {
	got := DetectAndRender([]byte(`{"a":1}`), "", true, true)
	if !strings.Contains(stripANSI(got), "\n") {
		t.Errorf("expected sniffed pretty JSON, got %q", got)
	}
}

func TestDetectAndRender_FallsBackToRawForPlainText(t *testing.T) {
	got := DetectAndRender([]byte(`just plain text`), "text/plain", true, true)
	if got != "just plain text" {
		t.Errorf("got %q, want unchanged plain text", got)
	}
}

func TestDetectAndRender_MasksSecretLookingJSONFieldsUnlessRevealed(t *testing.T) {
	body := []byte(`{"token":"abc123","name":"ada"}`)
	masked := DetectAndRender(body, "application/json", true, false)
	if strings.Contains(stripANSI(masked), "abc123") {
		t.Errorf("expected token value masked, got %q", masked)
	}
	if !strings.Contains(stripANSI(masked), "****") {
		t.Errorf("expected **** mask, got %q", masked)
	}
	if !strings.Contains(stripANSI(masked), `"name": "ada"`) {
		t.Errorf("expected non-secret field left untouched, got %q", masked)
	}

	revealed := DetectAndRender(body, "application/json", true, true)
	if !strings.Contains(stripANSI(revealed), "abc123") {
		t.Errorf("expected token value revealed, got %q", revealed)
	}
}

func TestDetectAndRender_MasksSecretLookingPlainTextLinesUnlessRevealed(t *testing.T) {
	body := []byte("Authorization: Bearer abc123\nContent-Length: 42")
	masked := DetectAndRender(body, "text/plain", true, false)
	if strings.Contains(masked, "abc123") {
		t.Errorf("expected Authorization value masked, got %q", masked)
	}
	if !strings.Contains(masked, "Content-Length: 42") {
		t.Errorf("expected non-secret line left untouched, got %q", masked)
	}
}
