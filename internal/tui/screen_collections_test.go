package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
)

func TestNew_DefaultsToRequestScreenWithDrawerOpen(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	if m.screen != ScreenRequest {
		t.Errorf("default screen = %v, want ScreenRequest", m.screen)
	}
	if !m.drawerOpen() {
		t.Error("expected the collections drawer to be open by default")
	}
}

func TestActivateSidebarSelection_OnARequestSwitchesToRequestScreen(t *testing.T) {
	collectionsRoot := t.TempDir()
	m := New(collectionsRoot, t.TempDir())
	m.screen = ScreenCollections
	if _, err := m.store.SaveRequest("", "ping", collection.Request{Method: collection.GET, URL: "https://example.com"}); err != nil {
		t.Fatalf("save request error: %v", err)
	}
	m.refreshTree()
	m.selectSidebarPath("010_ping.json")

	got := m.activateSidebarSelection()

	if got.screen != ScreenRequest {
		t.Errorf("screen = %v, want ScreenRequest after opening a request", got.screen)
	}
	if got.urlInput.Value() != "https://example.com" {
		t.Errorf("expected the request to be loaded, url = %q", got.urlInput.Value())
	}
}

func TestActivateSidebarSelection_OnAFolderStaysOnCollectionsScreen(t *testing.T) {
	collectionsRoot := t.TempDir()
	m := New(collectionsRoot, t.TempDir())
	m.screen = ScreenCollections
	if _, err := m.store.CreateFolder("", "users"); err != nil {
		t.Fatalf("create folder error: %v", err)
	}
	m.refreshTree()
	m.selectSidebarPath("010_users")

	got := m.activateSidebarSelection()

	if got.screen != ScreenCollections {
		t.Errorf("screen = %v, want to stay on ScreenCollections when toggling a folder", got.screen)
	}
}

func TestCollectionsScreenView_ShowsRequestsAndHeader(t *testing.T) {
	collectionsRoot := t.TempDir()
	m := New(collectionsRoot, t.TempDir())
	m.screen = ScreenCollections
	if _, err := m.store.SaveRequest("", "ping", collection.Request{Method: collection.GET, URL: "https://example.com"}); err != nil {
		t.Fatalf("save request error: %v", err)
	}
	m.refreshTree()
	m.width, m.height = 100, 30

	out := stripANSI(m.View())
	if !strings.Contains(out, "ping") {
		t.Errorf("expected the saved request in the collections screen, got:\n%s", out)
	}
	// The top bar was removed, so the screen no longer carries a "parley"
	// header; its own "Collections" panel title is the heading now.
	if !strings.Contains(out, "Collections") {
		t.Errorf("expected the Collections panel title, got:\n%s", out)
	}
}

func TestCollectionsScreenView_RendersExactlyTheTerminalHeight(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.screen = ScreenCollections
	m.width, m.height = 100, 30

	got := m.View()
	lines := strings.Split(got, "\n")
	if len(lines) != m.height {
		t.Errorf("got %d lines, want %d", len(lines), m.height)
	}
}

func TestHandleCollectionsScreenKey_QOpensQuitConfirm(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.screen = ScreenCollections
	m.width, m.height = 100, 30

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	got := next.(Model)
	if !got.confirm.active {
		t.Error("expected q on the Collections screen to open the quit confirmation")
	}
}

func TestHandleCollectionsScreenKey_PaletteStillOpens(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.screen = ScreenCollections
	m.width, m.height = 100, 30

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	got := next.(Model)
	if !got.palette.active {
		t.Error("expected ctrl+k on the Collections screen to open the command palette")
	}
}

func TestHandleCollectionsScreenKey_F5OpensDashboard(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.screen = ScreenCollections
	m.width, m.height = 100, 30

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyF5})
	got := next.(Model)
	if got.screen != ScreenDashboard {
		t.Errorf("screen = %v, want ScreenDashboard", got.screen)
	}
	if got.previousScreen != ScreenCollections {
		t.Errorf("previousScreen = %v, want ScreenCollections", got.previousScreen)
	}
}

func TestOpenCollectionsScreen_RemembersPreviousScreen(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 100, 30

	got, _ := m.openCollectionsScreen()
	if got.screen != ScreenCollections {
		t.Errorf("screen = %v, want ScreenCollections", got.screen)
	}
	if got.previousScreen != ScreenRequest {
		t.Errorf("previousScreen = %v, want ScreenRequest", got.previousScreen)
	}
}
