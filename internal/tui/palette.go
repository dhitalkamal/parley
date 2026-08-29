package tui

import (
	workspace "github.com/dhitalkamal/parley/internal/workspace/domain"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// paletteCommand is one fuzzy-searchable entry in the command palette: a
// human name plus what to do when it's chosen. category and keyHint are
// shown as the item's subtitle - a category tag for grouping/discovery, and
// the direct keybinding (if any) as a forgot-the-shortcut reminder.
type paletteCommand struct {
	name     string
	category string
	keyHint  string
	run      func(Model) (Model, tea.Cmd)
}

func (c paletteCommand) Title() string { return c.name }
func (c paletteCommand) Description() string {
	if c.keyHint == "" {
		return c.category
	}
	return c.category + "  " + c.keyHint
}
func (c paletteCommand) FilterValue() string { return c.name }

type paletteState struct {
	active bool
	list   list.Model
}

func newPaletteState() paletteState {
	l := list.New(nil, newThemedDelegate(true), 0, 0)
	l.Title = "Commands"
	l.SetShowHelp(false)
	return paletteState{list: l}
}

// RefreshDelegate rebuilds the selection-highlight delegate from the
// current theme - see sidebar.RefreshDelegate for why this is needed at all.
func (p *paletteState) RefreshDelegate() {
	p.list.SetDelegate(newThemedDelegate(true))
}

// paletteCommands lists every action reachable from the command palette -
// the same ground as the direct keybindings, so it works both as a
// forgot-the-shortcut fallback and a discovery mechanism for new users. Row-
// level table actions (add/delete/toggle/edit row) are left out since they
// only make sense with a specific focused widget already in view.
func paletteCommands() []paletteCommand {
	return []paletteCommand{
		{name: "Send request", category: "Request", keyHint: "ctrl+r", run: func(m Model) (Model, tea.Cmd) { return m.trySend() }},
		{name: "Save request", category: "Request", keyHint: "ctrl+s", run: func(m Model) (Model, tea.Cmd) { return m.saveCurrentRequest() }},
		{name: "Format body (JSON)", category: "Request", keyHint: "ctrl+p", run: func(m Model) (Model, tea.Cmd) {
			m.body.formatJSON()
			return m, nil
		}},
		{name: "New request", category: "Collections", run: func(m Model) (Model, tea.Cmd) {
			m.prompt.Open(promptNewRequest, m.selectedFolderPath(), "New request name", "")
			return m, nil
		}},
		{name: "New folder", category: "Collections", run: func(m Model) (Model, tea.Cmd) {
			m.prompt.Open(promptNewFolder, m.selectedFolderPath(), "New folder name", "")
			return m, nil
		}},
		{name: "New collection", category: "Collections", keyHint: "C", run: func(m Model) (Model, tea.Cmd) {
			m.prompt.Open(promptNewFolder, "", "New collection name", "")
			return m, nil
		}},
		{name: "New WebSocket connection", category: "Request", run: func(m Model) (Model, tea.Cmd) { return m.openWebSocket() }},
		{name: "Run collection/folder", category: "Collections", run: func(m Model) (Model, tea.Cmd) { return m.startCollectionRun() }},
		{name: "Run dashboard", category: "Collections", keyHint: "f5", run: func(m Model) (Model, tea.Cmd) { return m.openDashboard() }},
		{name: "Open Collections screen", category: "Collections", run: func(m Model) (Model, tea.Cmd) { return m.openCollectionsScreen() }},
		{name: "Import", category: "I/O", keyHint: "ctrl+l", run: func(m Model) (Model, tea.Cmd) { return m.startImport() }},
		{name: "Export", category: "I/O", keyHint: "ctrl+o", run: func(m Model) (Model, tea.Cmd) { return m.startExport() }},
		{name: "Toggle code snippet", category: "Code", keyHint: "ctrl+n", run: func(m Model) (Model, tea.Cmd) {
			m.showCodeSnippet = !m.showCodeSnippet
			return m, nil
		}},
		{name: "Toggle body/headers/cookies", category: "Response", keyHint: "ctrl+t", run: func(m Model) (Model, tea.Cmd) {
			m.response.CycleMode()
			return m, nil
		}},
		{name: "Search response", category: "Response", keyHint: "ctrl+f", run: func(m Model) (Model, tea.Cmd) {
			m.prompt.Open(promptSearch, "", "Search response", m.response.searchTerm)
			return m, nil
		}},
		{name: "Save example", category: "Response", keyHint: "ctrl+x", run: func(m Model) (Model, tea.Cmd) { return m.startSaveExample() }},
		{name: "Reveal secrets", category: "Response", keyHint: "ctrl+u", run: func(m Model) (Model, tea.Cmd) {
			m.revealSecrets = !m.revealSecrets
			m.response.SetRevealSecrets(m.revealSecrets)
			m.body.formTable.SetRevealSecrets(m.revealSecrets)
			return m, nil
		}},
		{name: "Maximize response", category: "Response", keyHint: "ctrl+z", run: func(m Model) (Model, tea.Cmd) {
			if m.responseZoneVisible() {
				m.responseMaximized = !m.responseMaximized
				m.focus = focusResponse
				m.updateFocus()
			}
			return m, nil
		}},
		// Go-to entries: the top tab strip was removed, so the palette is now
		// the discoverable menu for reaching every screen (alongside the
		// dedicated keys and the 1-6 accelerators). Collections/Environments/
		// History/Runner already appear under their own categories above/below;
		// these two fill the gaps (home and Settings) so nothing is orphaned.
		{name: "Go to Request", category: "Nav", keyHint: "2", run: func(m Model) (Model, tea.Cmd) {
			m.screen = ScreenRequest
			return m, nil
		}},
		{name: "Settings", category: "Nav", keyHint: "f9", run: func(m Model) (Model, tea.Cmd) { return m.openSettings() }},
		{name: "History", category: "History", keyHint: "ctrl+y", run: func(m Model) (Model, tea.Cmd) { return m.openHistory() }},
		{name: "Environments", category: "Env", keyHint: "ctrl+e", run: func(m Model) (Model, tea.Cmd) {
			m.openEnvDropdown()
			return m, nil
		}},
		{name: "Toggle variables panel", category: "Env", keyHint: "ctrl+w", run: func(m Model) (Model, tea.Cmd) {
			m.showVariables = !m.showVariables
			return m, nil
		}},
		{name: "Toggle theme", category: "UI", run: func(m Model) (Model, tea.Cmd) {
			m.status = "Theme: " + cycleTheme()
			m.sidebar.RefreshDelegate()
			m.palette.RefreshDelegate()
			return m, nil
		}},
		{name: "Toggle help", category: "UI", keyHint: "?", run: func(m Model) (Model, tea.Cmd) {
			m.help.ShowAll = !m.help.ShowAll
			return m, nil
		}},
		{name: "Toggle orientation", category: "UI", keyHint: "f6", run: func(m Model) (Model, tea.Cmd) {
			if m.orientation == workspace.OrientationVertical {
				m.orientation = workspace.OrientationHorizontal
			} else {
				m.orientation = workspace.OrientationVertical
			}
			m.persistLayout()
			return m, nil
		}},
		{name: "Toggle request zone", category: "UI", keyHint: "f7", run: func(m Model) (Model, tea.Cmd) {
			m.requestCollapsed = !m.requestCollapsed
			m.persistLayout()
			return m, nil
		}},
		{name: "Toggle response zone", category: "UI", keyHint: "f8", run: func(m Model) (Model, tea.Cmd) {
			m.responseCollapsed = !m.responseCollapsed
			m.persistLayout()
			return m, nil
		}},
		{name: "Toggle collections drawer", category: "UI", keyHint: "ctrl+\\", run: func(m Model) (Model, tea.Cmd) {
			m.toggleDrawer()
			return m, nil
		}},
		{name: "Toggle shortcuts rail", category: "UI", keyHint: "f1", run: func(m Model) (Model, tea.Cmd) {
			m.toggleSideRail()
			return m, nil
		}},
		{name: "Toggle rail view (env/shortcuts)", category: "UI", keyHint: "v", run: func(m Model) (Model, tea.Cmd) {
			m.toggleRailMode()
			return m, nil
		}},
		{name: "Quit", category: "UI", keyHint: "ctrl+c", run: func(m Model) (Model, tea.Cmd) { return m, tea.Quit }},
	}
}

// Open resets the palette to the full command list, cursor at the top, and
// drops straight into filtering - so typing immediately narrows the list
// instead of needing a "/" first like the sidebar's own filter does.
func (p *paletteState) Open() {
	cmds := paletteCommands()
	items := make([]list.Item, len(cmds))
	for i, c := range cmds {
		items[i] = c
	}
	p.list.SetItems(items)
	p.list.ResetFilter()
	p.list.Select(0)
	// Simulate pressing "/" (the list's own filter key) rather than calling
	// SetFilterState directly - only the real key handler populates
	// filteredItems with "everything matches" for an empty query; skipping it
	// leaves the palette showing "Nothing matched" until the first keystroke.
	p.list, _ = p.list.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	p.active = true
}

func (p *paletteState) Close() {
	p.active = false
}

// paletteListHeight is a fixed number of visible rows - the palette is a
// floating modal now (see overlay.go), not sized from leftover terminal
// height the way the old inline-stacked version was.
const paletteListHeight = 14

// View renders the palette at width - an explicit width, not just
// borderStyle.Render's default content-hugging behavior, because every
// command name here is short enough that the box would otherwise shrink to
// fit the text instead of filling the available column. Width() sizes the
// padding+content area together (border is the only thing it adds on top
// of), confirmed empirically - -2 not -4.
func (p paletteState) View(width int) string {
	p.list.SetSize(width-4, paletteListHeight)
	return borderStyle.Width(width - 2).Render(p.list.View())
}

// handlePaletteKey handles input while the palette is open. Enter runs the
// selected command's action and closes the palette either way - a command
// that opens another prompt (import, search, new request, ...) needs the
// palette out of the way first.
func (m Model) handlePaletteKey(k tea.KeyMsg) (Model, tea.Cmd) {
	switch k.String() {
	case "esc", "ctrl+k":
		m.palette.Close()
		return m, nil
	case "enter":
		item, ok := m.palette.list.SelectedItem().(paletteCommand)
		m.palette.Close()
		if !ok {
			return m, nil
		}
		return item.run(m)
	}
	var cmd tea.Cmd
	m.palette.list, cmd = m.palette.list.Update(k)
	return m, cmd
}
