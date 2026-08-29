package tui

import (
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestSidebarNavigation_PreviewsTheRequestUnderTheCursor guards a real
// request: navigating the sidebar with arrow keys should load whichever
// request the cursor lands on into the Request panel immediately, without
// needing an explicit Enter first - closer to a file explorer's preview
// pane than a two-step "select, then confirm."
func TestSidebarNavigation_PreviewsTheRequestUnderTheCursor(t *testing.T) {
	base := New(t.TempDir(), t.TempDir())
	if _, err := base.store.SaveRequest("", "a", collection.Request{Method: collection.GET, URL: "https://a.example.com"}); err != nil {
		t.Fatalf("SaveRequest a: %v", err)
	}
	if _, err := base.store.SaveRequest("", "b", collection.Request{Method: collection.POST, URL: "https://b.example.com"}); err != nil {
		t.Fatalf("SaveRequest b: %v", err)
	}
	base.refreshTree()
	base.focus = focusSidebar
	var m tea.Model = base

	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyDown})

	if m.(Model).urlInput.Value() == "" {
		t.Fatal("expected navigating onto a request to load it into the editor without pressing enter")
	}
}

// TestSidebarNavigation_DoesNotToggleFoldersOnNavigation checks arrow-key
// navigation over a folder must not open/close it - that stays an
// Enter/click-only action, or every arrow-key press over a collapsed
// collection would blow it open unexpectedly.
func TestSidebarNavigation_DoesNotToggleFoldersOnNavigation(t *testing.T) {
	base := New(t.TempDir(), t.TempDir())
	folderPath, err := base.store.CreateFolder("", "Folder")
	if err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}
	if _, err := base.store.SaveRequest(folderPath, "inner", collection.Request{Method: collection.GET, URL: "https://inner.example.com"}); err != nil {
		t.Fatalf("SaveRequest inner: %v", err)
	}
	base.refreshTree()
	base.sidebar.SelectPath(folderPath)
	base.focus = focusSidebar
	before := base.sidebar.closed[folderPath]
	var m tea.Model = base

	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyDown})
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyUp})

	if got := m.(Model).sidebar.closed[folderPath]; got != before {
		t.Errorf("got closed=%v after navigation, want unchanged from %v - navigation must not toggle folders", got, before)
	}
}

// TestSidebarNavigation_ResetsResponseWhenLoadingADifferentRequest guards
// the other half of the ask: the Response panel still showing a stale
// response tied to whatever was loaded *before* is misleading once a
// different request is now in view.
func TestSidebarNavigation_ResetsResponseWhenLoadingADifferentRequest(t *testing.T) {
	base := New(t.TempDir(), t.TempDir())
	if _, err := base.store.SaveRequest("", "a", collection.Request{Method: collection.GET, URL: "https://a.example.com"}); err != nil {
		t.Fatalf("SaveRequest a: %v", err)
	}
	if _, err := base.store.SaveRequest("", "b", collection.Request{Method: collection.GET, URL: "https://b.example.com"}); err != nil {
		t.Fatalf("SaveRequest b: %v", err)
	}
	base.refreshTree()
	base.focus = focusSidebar
	base.response.SetResponse(execution.Response{StatusCode: 200, Status: "200 OK", Body: []byte("from a")}, 10)
	var m tea.Model = base

	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyDown})

	got := m.(Model).response
	if got.status != "" {
		t.Errorf("got response status %q after navigating to a different request, want reset to the empty state", got.status)
	}
}

// TestSidebarNavigation_RestoresTheCachedResponseWhenNavigatingBackToARequest
// guards the follow-up ask: a response fetched for a request must not be
// lost just because the cursor moved on to preview a different request -
// navigating back should bring that response back, not leave the empty
// state from the request visited in between.
func TestSidebarNavigation_RestoresTheCachedResponseWhenNavigatingBackToARequest(t *testing.T) {
	base := New(t.TempDir(), t.TempDir())
	pathA, err := base.store.SaveRequest("", "a", collection.Request{Method: collection.GET, URL: "https://a.example.com"})
	if err != nil {
		t.Fatalf("SaveRequest a: %v", err)
	}
	if _, err := base.store.SaveRequest("", "b", collection.Request{Method: collection.GET, URL: "https://b.example.com"}); err != nil {
		t.Fatalf("SaveRequest b: %v", err)
	}
	base.refreshTree()
	base.sidebar.SelectPath(pathA)
	base.loadRequestIntoEditor(pathA, collection.Request{Method: collection.GET, URL: "https://a.example.com"})
	base.response.SetResponse(execution.Response{StatusCode: 200, Status: "200 OK", Body: []byte("from a")}, 10)
	base.focus = focusSidebar
	var m tea.Model = base

	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyDown}) // preview b - response resets
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyUp})   // back to a - response should return

	got := m.(Model).response
	if got.status != "200 OK" || string(got.rawBody) != "from a" {
		t.Errorf("got response status %q body %q after navigating back to a, want the cached 200 OK/from a response restored", got.status, got.rawBody)
	}
}

// TestSidebarNavigation_KeepsResponseWhenReselectingTheSameRequest checks
// the reset above doesn't wipe a response just because the cursor briefly
// moved off and back onto the same request.
func TestSidebarNavigation_KeepsResponseWhenReselectingTheSameRequest(t *testing.T) {
	base := New(t.TempDir(), t.TempDir())
	path, err := base.store.SaveRequest("", "a", collection.Request{Method: collection.GET, URL: "https://a.example.com"})
	if err != nil {
		t.Fatalf("SaveRequest a: %v", err)
	}
	base.refreshTree()
	base.sidebar.SelectPath(path)
	base.loadRequestIntoEditor(path, collection.Request{Method: collection.GET, URL: "https://a.example.com"})
	base.response.SetResponse(execution.Response{StatusCode: 200, Status: "200 OK", Body: []byte("hello")}, 10)
	base.focus = focusSidebar

	// Re-loading the already-loaded request (same path) must not reset it.
	req, err := base.store.LoadRequest(path)
	if err != nil {
		t.Fatalf("LoadRequest: %v", err)
	}
	base.loadRequestIntoEditor(path, req)

	if base.response.status == "" {
		t.Error("expected re-loading the same already-loaded request to leave its response untouched")
	}
}

// TestLoadRequestIntoEditor_RestoresPersistedResponseAfterRestart guards
// the actual ask: a response must survive not just navigating away and
// back within the same session (see the responseCache-backed tests above),
// but the app being closed and reopened entirely - responseCache alone is
// in-memory only and would lose it.
func TestLoadRequestIntoEditor_RestoresPersistedResponseAfterRestart(t *testing.T) {
	collectionsRoot, projectRoot := t.TempDir(), t.TempDir()
	base := New(collectionsRoot, projectRoot)
	path, err := base.store.SaveRequest("", "a", collection.Request{Method: collection.GET, URL: "https://a.example.com"})
	if err != nil {
		t.Fatalf("SaveRequest: %v", err)
	}
	base.refreshTree()
	base.loadRequestIntoEditor(path, collection.Request{Method: collection.GET, URL: "https://a.example.com"})
	base, _ = base.handleSendResult(sendResultMsg{
		req:  collection.Request{Method: collection.GET, URL: "https://a.example.com"},
		resp: execution.Response{StatusCode: 200, Status: "200 OK", Body: []byte("hello")},
	})

	// Simulate closing and reopening the app: a fresh Model built against
	// the same roots, with nothing carried over in memory.
	reopened := New(collectionsRoot, projectRoot)
	reopened.refreshTree()
	reopened.loadRequestIntoEditor(path, collection.Request{Method: collection.GET, URL: "https://a.example.com"})

	if reopened.response.status != "200 OK" || string(reopened.response.rawBody) != "hello" {
		t.Errorf("got status %q body %q, want the persisted 200 OK/hello response restored after restart", reopened.response.status, reopened.response.rawBody)
	}
}
