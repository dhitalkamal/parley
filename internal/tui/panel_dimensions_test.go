package tui

import (
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
	scripting "github.com/dhitalkamal/parley/internal/scripting/domain"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestRequestPanelView_MatchesRequestedDimensionsAcrossEveryTab(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44

	for tab := reqTabParams; tab < reqTabCount; tab++ {
		m.reqTab = tab
		got := m.requestPanelView(60, 30)
		if w := lipgloss.Width(got); w != 60 {
			t.Errorf("tab %v: got width %d, want 60", tab, w)
		}
		if h := lipgloss.Height(got); h != 30 {
			t.Errorf("tab %v: got height %d, want 30", tab, h)
		}
	}
}

func TestRequestPanelView_MatchesDimensionsWithUnresolvedWarningShown(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.urlInput.SetValue("https://example.com/{{missing}}")

	got := m.requestPanelView(60, 30)
	if w := lipgloss.Width(got); w != 60 {
		t.Errorf("got width %d, want 60", w)
	}
	if h := lipgloss.Height(got); h != 30 {
		t.Errorf("got height %d, want 30", h)
	}
}

func TestRequestPanelView_MatchesDimensionsWhileEditingAParamRow(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.reqTab = reqTabParams
	m.params.rows = []kvRow{{Key: "a", Value: "b", Enabled: true}}
	m.params = m.params.startEdit()

	got := m.requestPanelView(60, 30)
	if h := lipgloss.Height(got); h != 30 {
		t.Errorf("got height %d, want 30 (editing a row must not overflow the panel)", h)
	}
}

func TestRequestPanelView_MatchesDimensionsWithEmptyHeadersTable(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.reqTab = reqTabHeaders

	got := m.requestPanelView(60, 30)
	if h := lipgloss.Height(got); h != 30 {
		t.Errorf("got height %d, want 30 (empty-state hint must not overflow the panel)", h)
	}
}

func TestUrlRowView_MatchesRequestedWidthAndIsOneLine(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	for _, width := range []int{60, 90, 140, 200} {
		// applyLayoutWidths is what keeps urlInput.Width in sync with the
		// url box's actual budget - the real app always runs it (every
		// startup fires a WindowSizeMsg before the first render), so a test
		// that skips it is rendering a state real usage never hits.
		m.width = width
		m.applyLayoutWidths()
		got := m.urlRowView(width)
		if w := lipgloss.Width(got); w != width {
			t.Errorf("width %d: got rendered width %d, want %d", width, w, width)
		}
		if h := lipgloss.Height(got); h != 3 {
			t.Errorf("width %d: got %d lines, want 3 (top/content/bottom of the bordered boxes)", width, h)
		}
	}
}

func TestResponsePanelView_MatchesRequestedDimensions(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44

	got := m.responsePanelView(60, 30)
	if w := lipgloss.Width(got); w != 60 {
		t.Errorf("got width %d, want 60", w)
	}
	if h := lipgloss.Height(got); h != 30 {
		t.Errorf("got height %d, want 30", h)
	}
}

func TestResponsePanelView_MatchesDimensionsWithTestResults(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.response.SetResponse(execution.Response{StatusCode: 200, Status: "200 OK"}, 10)
	m.response.SetTestResults([]scripting.TestResult{
		{Name: "status is 200", Passed: true},
		{Name: "has id", Passed: false, Error: "expected a string"},
	}, "")
	m.response.SetMode(viewTests)

	got := m.responsePanelView(60, 30)
	if w := lipgloss.Width(got); w != 60 {
		t.Errorf("got width %d, want 60", w)
	}
	if h := lipgloss.Height(got); h != 30 {
		t.Errorf("got height %d, want 30", h)
	}
}
