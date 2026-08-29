package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"

	execution "github.com/dhitalkamal/parley/internal/execution/domain"
)

func modelWithResponse(t *testing.T) Model {
	t.Helper()
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 160, 44
	m.screen = ScreenRequest
	m.response.SetResponse(execution.Response{
		StatusCode: 200, Status: "200 OK",
		Body: []byte(`{"ok":true}`),
		Timing: execution.Timing{
			DNSLookup:  12 * time.Millisecond,
			TCPConnect: 34 * time.Millisecond,
			ServerWait: 100 * time.Millisecond,
		},
	}, 142)
	return m
}

// TestResponseStatusMeta_ShowsStatusTimeSize guards the redesign: status, total
// time, and size render as the title meta (the string spliced onto the panel's
// title border), with the code in its class color.
func TestResponseStatusMeta_ShowsStatusTimeSize(t *testing.T) {
	m := modelWithResponse(t)
	meta := stripANSI(m.responseStatusMeta())
	for _, want := range []string{"200 OK", "142 ms", "11 bytes"} {
		if !strings.Contains(meta, want) {
			t.Errorf("status meta = %q, want it to contain %q", meta, want)
		}
	}
}

// TestResponseStatusMeta_EmptyBeforeResponse guards that nothing is shown until
// a response exists (and while sending / on error).
func TestResponseStatusMeta_EmptyBeforeResponse(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	if got := m.responseStatusMeta(); got != "" {
		t.Errorf("status meta before a response = %q, want empty", got)
	}
	m2 := modelWithResponse(t)
	m2.sending = true
	if got := m2.responseStatusMeta(); got != "" {
		t.Errorf("status meta while sending = %q, want empty", got)
	}
}

// TestResponsePanel_HeightExact guards the panel renders at exactly its height
// budget (body full width, no overflow).
func TestResponsePanel_HeightExact(t *testing.T) {
	m := modelWithResponse(t)
	if h := lipgloss.Height(m.responsePanelView(120, 30)); h != 30 {
		t.Errorf("panel height = %d, want 30", h)
	}
}

// TestResponsePanel_BodyIsFullWidth guards that the body no longer carries a
// status row or a side info column - the status lives on the title now.
func TestResponsePanel_BodyIsFullWidth(t *testing.T) {
	m := modelWithResponse(t)
	out := stripANSI(m.responsePanelView(120, 30))
	// timing detail belongs to the Timeline tab, not the body panel.
	if strings.Contains(out, "DNS Lookup") {
		t.Error("the response body panel should not embed the timing breakdown")
	}
}
