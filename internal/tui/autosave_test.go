package tui

import (
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestAutosave_PersistsURLEditsWithoutExplicitSave guards the actual ask: a
// user editing an already-saved request must never lose that edit to a
// forgotten ctrl+s, a crash, or switching away before saving - every
// keystroke that changes the request now writes straight through to disk.
func TestAutosave_PersistsURLEditsWithoutExplicitSave(t *testing.T) {
	projectRoot := t.TempDir()
	base := New(t.TempDir(), projectRoot)
	path, err := base.store.SaveRequest("", "req", collection.Request{Method: collection.GET, URL: "https://example.com"})
	if err != nil {
		t.Fatalf("SaveRequest: %v", err)
	}
	base.loadRequestIntoEditor(path, collection.Request{Method: collection.GET, URL: "https://example.com"})
	base.screen = ScreenRequest
	base.focus = focusURL
	base.updateFocus()     // actually focus the URL widget (launch focus is the Explorer now)
	base.mode = modeInsert // vim: typing into a field requires INSERT (see mode.go)
	var m tea.Model = base

	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	loaded, err := base.store.LoadRequest(path)
	if err != nil {
		t.Fatalf("LoadRequest: %v", err)
	}
	if loaded.URL != "https://example.comx" {
		t.Errorf("got persisted URL %q, want the just-typed edit autosaved", loaded.URL)
	}
}

// TestAutosave_PersistsMethodChangeViaMouseClick checks the mouse-driven
// method cycle also autosaves, not just keyboard edits - a click on the
// method box is just as much an edit as typing.
func TestAutosave_PersistsMethodChangeViaMouseClick(t *testing.T) {
	projectRoot := t.TempDir()
	base := New(t.TempDir(), projectRoot)
	path, err := base.store.SaveRequest("", "req", collection.Request{Method: collection.GET, URL: "https://example.com"})
	if err != nil {
		t.Fatalf("SaveRequest: %v", err)
	}
	base.loadRequestIntoEditor(path, collection.Request{Method: collection.GET, URL: "https://example.com"})
	base.screen = ScreenRequest
	base.width, base.height = 140, 44
	// The method box is in the center column now, offset by the Explorer width.
	_, centerStart, _ := base.layoutColumns()
	var m tea.Model = base

	next, _ := m.Update(tea.MouseMsg{X: centerStart + 2, Y: topBarHeight + 1, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	m = next

	loaded, err := base.store.LoadRequest(path)
	if err != nil {
		t.Fatalf("LoadRequest: %v", err)
	}
	if loaded.Method == collection.GET {
		t.Errorf("got persisted method %q, want the mouse-driven method cycle autosaved", loaded.Method)
	}
}

// TestAutosave_DoesNothingWhenNoRequestIsLoaded guards against autosave
// inventing a save target - a brand new, never-saved request has nowhere to
// autosave to, so it must still require an explicit Save As the first time.
func TestAutosave_DoesNothingWhenNoRequestIsLoaded(t *testing.T) {
	base := New(t.TempDir(), t.TempDir())
	base.focus = focusURL
	var m tea.Model = base

	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	if got := m.(Model).status; got == "Saved" {
		t.Error("expected no autosave attempt when nothing is loaded")
	}
}

// TestAutosave_DoesNotFireWhenTheLoadedRequestChanges guards the switch
// scenario: loading a different request in the same keystroke must not
// clobber the newly loaded one by re-running buildRequest() against its
// path (harmless in practice, but not what this diff is meant to guard) -
// this locks in that the check compares the request that was loaded
// *before* the keystroke to the same path, not whatever ends up loaded.
func TestAutosave_DoesNotFireWhenTheLoadedRequestChanges(t *testing.T) {
	base := New(t.TempDir(), t.TempDir())
	pathA, err := base.store.SaveRequest("", "a", collection.Request{Method: collection.GET, URL: "https://a.example.com"})
	if err != nil {
		t.Fatalf("SaveRequest a: %v", err)
	}
	pathB, err := base.store.SaveRequest("", "b", collection.Request{Method: collection.GET, URL: "https://b.example.com"})
	if err != nil {
		t.Fatalf("SaveRequest b: %v", err)
	}
	base.refreshTree()
	base.loadRequestIntoEditor(pathA, collection.Request{Method: collection.GET, URL: "https://a.example.com"})
	base.focus = focusSidebar
	base.sidebar.SelectPath(pathB)
	var m tea.Model = base

	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	if m.(Model).loadedRequestPath != pathB {
		t.Fatalf("got loadedRequestPath %q, want %q (enter on b should load it)", m.(Model).loadedRequestPath, pathB)
	}
	loadedA, err := base.store.LoadRequest(pathA)
	if err != nil {
		t.Fatalf("LoadRequest a: %v", err)
	}
	if loadedA.URL != "https://a.example.com" {
		t.Errorf("got a's persisted URL %q, want it untouched by loading b", loadedA.URL)
	}
}
