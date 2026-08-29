package tui

import (
	"strings"
	"testing"
	"time"

	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	history "github.com/dhitalkamal/parley/internal/history/domain"
)

// TestAboutPanelView_ShowsPlaceholderWhenNothingSelected guards the no-
// selection state - nothing to show details about yet.
func TestAboutPanelView_ShowsPlaceholderWhenNothingSelected(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 100, 30

	got := stripANSI(m.aboutPanelView(40, 20))
	if !strings.Contains(strings.ToLower(got), "select a request") {
		t.Errorf("aboutPanelView() = %q, want a select-a-request placeholder", got)
	}
}

// TestAboutPanelView_ShowsPlaceholderForAFolder guards the other non-
// request selection state - a folder has no method/URL/tests/history of
// its own to summarize.
func TestAboutPanelView_ShowsPlaceholderForAFolder(t *testing.T) {
	collectionsRoot := t.TempDir()
	m := New(collectionsRoot, t.TempDir())
	if _, err := m.store.CreateFolder("", "users"); err != nil {
		t.Fatalf("create folder error: %v", err)
	}
	m.refreshTree()
	m.selectSidebarPath("010_users")
	m.width, m.height = 100, 30

	got := stripANSI(m.aboutPanelView(40, 20))
	if !strings.Contains(strings.ToLower(got), "select a request") {
		t.Errorf("aboutPanelView() = %q, want a select-a-request placeholder for a folder", got)
	}
}

// TestAboutPanelView_ShowsMethodAndURLForARequest guards the actual ask:
// summarizing the selected request without opening it.
func TestAboutPanelView_ShowsMethodAndURLForARequest(t *testing.T) {
	collectionsRoot := t.TempDir()
	m := New(collectionsRoot, t.TempDir())
	if _, err := m.store.SaveRequest("", "ping", collection.Request{Method: collection.GET, URL: "https://example.com/ping"}); err != nil {
		t.Fatalf("save request error: %v", err)
	}
	m.refreshTree()
	m.selectSidebarPath("010_ping.json")
	m.width, m.height = 100, 30

	got := stripANSI(m.aboutPanelView(40, 20))
	if !strings.Contains(got, "GET") || !strings.Contains(got, "https://example.com/ping") {
		t.Errorf("aboutPanelView() = %q, want method and URL shown", got)
	}
}

// TestAboutPanelView_DistinguishesTestScriptPresence guards a real (if
// simplified) piece of info: whether the request has a test script at all -
// counting individual pm.test() assertions would need parsing the script,
// which is out of scope, so this is a yes/no rather than a count.
func TestAboutPanelView_DistinguishesTestScriptPresence(t *testing.T) {
	collectionsRoot := t.TempDir()
	m := New(collectionsRoot, t.TempDir())
	if _, err := m.store.SaveRequest("", "no-tests", collection.Request{Method: collection.GET, URL: "https://example.com/a"}); err != nil {
		t.Fatalf("save error: %v", err)
	}
	if _, err := m.store.SaveRequest("", "has-tests", collection.Request{Method: collection.GET, URL: "https://example.com/b", TestScript: `pm.test("ok", function(){});`}); err != nil {
		t.Fatalf("save error: %v", err)
	}
	m.refreshTree()
	m.width, m.height = 100, 30

	m.selectSidebarPath("010_no-tests.json")
	got := stripANSI(m.aboutPanelView(40, 20))
	if !strings.Contains(got, "Tests: none") {
		t.Errorf("aboutPanelView() for a request with no script = %q, want \"Tests: none\"", got)
	}

	m.selectSidebarPath("020_has-tests.json")
	got = stripANSI(m.aboutPanelView(40, 20))
	if !strings.Contains(got, "Tests: configured") {
		t.Errorf("aboutPanelView() for a request with a script = %q, want \"Tests: configured\"", got)
	}
}

// TestAboutPanelView_ShowsNeverRunWhenNoMatchingHistory guards the default
// state before a request has ever been sent.
func TestAboutPanelView_ShowsNeverRunWhenNoMatchingHistory(t *testing.T) {
	collectionsRoot := t.TempDir()
	m := New(collectionsRoot, t.TempDir())
	if _, err := m.store.SaveRequest("", "ping", collection.Request{Method: collection.GET, URL: "https://example.com"}); err != nil {
		t.Fatalf("save error: %v", err)
	}
	m.refreshTree()
	m.selectSidebarPath("010_ping.json")
	m.width, m.height = 100, 30

	got := stripANSI(m.aboutPanelView(40, 20))
	if !strings.Contains(strings.ToLower(got), "never run") {
		t.Errorf("aboutPanelView() = %q, want \"never run\"", got)
	}
}

// TestAboutPanelView_ShowsLastRunAndRecentHistoryForThisRequestOnly guards
// the core of the panel: history scoped to the selected request (by
// RequestPath), most recent first, and unrelated requests' history left
// out even when they share the same store.
func TestAboutPanelView_ShowsLastRunAndRecentHistoryForThisRequestOnly(t *testing.T) {
	collectionsRoot := t.TempDir()
	projectRoot := t.TempDir()
	m := New(collectionsRoot, projectRoot)
	if _, err := m.store.SaveRequest("", "ping", collection.Request{Method: collection.GET, URL: "https://example.com/ping"}); err != nil {
		t.Fatalf("save error: %v", err)
	}
	if _, err := m.store.SaveRequest("", "other", collection.Request{Method: collection.GET, URL: "https://example.com/other"}); err != nil {
		t.Fatalf("save error: %v", err)
	}
	m.refreshTree()

	entries := []history.HistoryEntry{
		{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), RequestPath: "010_ping.json", StatusCode: 200, Status: "200 OK"},
		{Time: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), RequestPath: "020_other.json", StatusCode: 500, Status: "500 Internal Server Error"},
		{Time: time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC), RequestPath: "010_ping.json", StatusCode: 404, Status: "404 Not Found"},
	}
	for _, e := range entries {
		if err := m.historyStore.AppendHistory(e); err != nil {
			t.Fatalf("append error: %v", err)
		}
	}
	m.selectSidebarPath("010_ping.json")
	m.width, m.height = 100, 30

	got := stripANSI(m.aboutPanelView(40, 20))
	if !strings.Contains(got, "404 Not Found") {
		t.Errorf("aboutPanelView() = %q, want the most recent matching run (404) as the last run", got)
	}
	if strings.Contains(got, "500 Internal Server Error") {
		t.Errorf("aboutPanelView() = %q, want no history from the other request (500)", got)
	}
	if !strings.Contains(got, "200 OK") {
		t.Errorf("aboutPanelView() = %q, want the older matching run (200) still listed in Recent History", got)
	}
}

// TestCollectionsScreenView_ShowsSidebarAndAboutPanelSideBySide guards the
// end-to-end wiring on a wide enough terminal.
func TestCollectionsScreenView_ShowsSidebarAndAboutPanelSideBySide(t *testing.T) {
	collectionsRoot := t.TempDir()
	m := New(collectionsRoot, t.TempDir())
	if _, err := m.store.SaveRequest("", "ping", collection.Request{Method: collection.GET, URL: "https://example.com/ping"}); err != nil {
		t.Fatalf("save error: %v", err)
	}
	m.refreshTree()
	m.selectSidebarPath("010_ping.json")
	m.width, m.height = 120, 34

	got := stripANSI(m.collectionsScreenView())
	if !strings.Contains(got, "About") {
		t.Errorf("collectionsScreenView() = %q, want the About panel present", got)
	}
	if !strings.Contains(got, "Recent History") {
		t.Errorf("collectionsScreenView() = %q, want the Recent History panel present", got)
	}
}

// TestCollectionsScreenView_RendersExactlyTheTerminalDimensionsWithAboutPanel
// extends the existing height guard to the new split layout.
func TestCollectionsScreenView_RendersExactlyTheTerminalDimensionsWithAboutPanel(t *testing.T) {
	collectionsRoot := t.TempDir()
	m := New(collectionsRoot, t.TempDir())
	if _, err := m.store.SaveRequest("", "ping", collection.Request{Method: collection.GET, URL: "https://example.com/ping"}); err != nil {
		t.Fatalf("save error: %v", err)
	}
	m.refreshTree()
	m.selectSidebarPath("010_ping.json")
	m.width, m.height = 120, 34

	got := m.collectionsScreenView()
	lines := strings.Split(got, "\n")
	if len(lines) != m.height {
		t.Errorf("got %d lines, want %d", len(lines), m.height)
	}
}
