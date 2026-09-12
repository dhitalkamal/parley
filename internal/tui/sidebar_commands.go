package tui

import (
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	"path/filepath"
	"reflect"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// dirOf is filepath.Dir but returns "" instead of "." for a top-level path,
// matching the convention (shared with infrastructure/store) that the
// collections root is the empty path.
func dirOf(p string) string {
	d := filepath.Dir(p)
	if d == "." {
		return ""
	}
	return d
}

var sidebarKeys = struct {
	NewRequest    key.Binding
	NewFolder     key.Binding
	NewCollection key.Binding
	Rename        key.Binding
	Delete        key.Binding
	MoveUp        key.Binding
	MoveDown      key.Binding
	Run           key.Binding
	ScrollLeft    key.Binding
	ScrollRight   key.Binding
}{
	NewRequest:    key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new request")),
	NewFolder:     key.NewBinding(key.WithKeys("N"), key.WithHelp("N", "new folder")),
	NewCollection: key.NewBinding(key.WithKeys("C"), key.WithHelp("C", "new collection")),
	Rename:        key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "rename")),
	Delete:        key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
	MoveUp:        key.NewBinding(key.WithKeys("ctrl+up"), key.WithHelp("ctrl+up", "move up")),
	MoveDown:      key.NewBinding(key.WithKeys("ctrl+down"), key.WithHelp("ctrl+down", "move down")),
	Run:           key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "run folder/collection")),
	// left/right take over from bubbles/list's own default paging binding
	// for just these two keys (pgup/pgdown/b/f/u/d still page) - a selected
	// row's name too long to fit needs some way to see the rest of it, and
	// this is the one already-idle-for-that-purpose pair on the keyboard.
	ScrollLeft:  key.NewBinding(key.WithKeys("left"), key.WithHelp("left", "scroll name left")),
	ScrollRight: key.NewBinding(key.WithKeys("right"), key.WithHelp("right", "scroll name right")),
}

func (m *Model) refreshTree() {
	tree, err := m.store.Tree()
	if err != nil {
		m.status = "Tree error: " + err.Error()
		return
	}
	// Keep the Explorer's root row in step with the active workspace; set the
	// field directly (not SetWorkspaceName) so SetTree's rebuild runs once.
	m.sidebar.workspaceName = m.activeWorkspaceName
	m.sidebar.SetTree(tree)
}

func (m *Model) selectSidebarPath(path string) {
	m.sidebar.SelectPath(path)
}

func (m *Model) loadRequestIntoEditor(path string, req collection.Request) {
	// A WebSocket request opens on its own screen, not the HTTP editor.
	if req.IsWebSocket() {
		m.loadWSRequest(path, req)
		return
	}
	// Opening an HTTP request while on the WebSocket screen switches back.
	if m.screen == ScreenWebSocket {
		m.screen = ScreenRequest
		m.updateFocus()
	}
	// The Response panel belongs to whichever request was loaded before -
	// leaving a stale response on screen once a genuinely different request
	// is in view reads as "this is what THIS request returned," which isn't
	// true. Only swap it out on an actual switch, not a reselect of the same
	// request (e.g. the cursor briefly moving off and back onto it), which
	// would otherwise wipe a response the user just fetched.
	if path != m.loadedRequestPath {
		// Stash the outgoing request's response before dropping it, so
		// navigating back later restores it instead of showing empty state
		// again - a response is worth just as much as the request fields
		// autosave already protects.
		if m.loadedRequestPath != "" {
			m.responseCache[m.loadedRequestPath] = m.response
		}
		if cached, ok := m.responseCache[path]; ok {
			m.response = cached
		} else if resp, elapsedMS, ok, err := m.lastResponseStore.Load(path); err == nil && ok {
			// Nothing cached in memory for this path yet (e.g. the app just
			// restarted) - fall back to what was last persisted to disk
			// (see send.go's handleSendResult) rather than the empty state.
			rv := newResponseView()
			rv.SetResponse(resp, elapsedMS)
			m.response = rv
		} else {
			m.response = newResponseView()
		}
	}
	m.loadedRequestPath = path
	m.methodIdx = 0
	for i, meth := range collection.Methods {
		if meth == req.Method {
			m.methodIdx = i
		}
	}
	m.urlInput.SetValue(req.URL)

	prows := make([]kvRow, len(req.Params))
	for i, p := range req.Params {
		prows[i] = kvRow{Key: p.Key, Value: p.Value, Enabled: p.Enabled, Description: p.Description}
	}
	m.params.SetRows(prows)

	hrows := make([]kvRow, len(req.Headers))
	for i, h := range req.Headers {
		hrows[i] = kvRow{Key: h.Key, Value: h.Value, Enabled: h.Enabled, Description: h.Description}
	}
	m.headers.SetRows(hrows)

	m.body.SetBody(req.Body)
	m.scripts.SetScripts(req.PreRequestScript, req.TestScript)
	m.auth.SetAuth(req.AuthCapture, req.Refresh)
	m.reqSettings.SetSettings(req.Timeout, req.FollowRedirects, req.InsecureSkipVerify)
	m.persistSidebarState()
}

// newFromExplorer is the flow-aware "n" (New) in the Explorer: it opens the
// prompt for whatever the next step in Workspace -> Collection -> Folder ->
// Request is, given what already exists. With a workspace store but no
// workspace yet, n creates the workspace (step 1); with a workspace but no
// collections, n creates the first collection (step 2); otherwise n creates a
// request under the current selection (the everyday case). N (folder) and C
// (collection) stay as explicit accelerators for the in-between steps.
func (m Model) newFromExplorer() (Model, tea.Cmd, bool) {
	if m.workspaceStore != nil && m.activeWorkspaceName == "" {
		m.prompt.Open(promptNewWorkspace, "", "New workspace name", "")
		return m, nil, true
	}
	if len(m.sidebar.root.Children) == 0 {
		m.prompt.Open(promptNewFolder, "", "New collection name", "")
		return m, nil, true
	}
	m.prompt.Open(promptNewRequest, m.selectedFolderPath(), "New request name", "")
	return m, nil, true
}

// selectedFolderPath resolves the parent folder a new node should be
// created under: the selected folder itself, the parent of a selected
// request, or the collections root if nothing is selected.
func (m Model) selectedFolderPath() string {
	item, ok := m.sidebar.Selected()
	if !ok {
		return ""
	}
	// The workspace root is the collections root itself - a new collection or
	// request created "here" lands at the top level, not inside a folder.
	if item.isWorkspace {
		return ""
	}
	if item.kind == collection.KindFolder {
		return item.path
	}
	return dirOf(item.path)
}

// handleSidebarKey handles the sidebar's own commands (new/rename/delete/
// reorder/load). It returns handled=false for keys that should fall through
// to sidebar.Update instead (navigation, "/" filter, etc).
func (m Model) handleSidebarKey(k tea.KeyMsg) (Model, tea.Cmd, bool) {
	switch {
	case key.Matches(k, sidebarKeys.Run):
		newM, cmd := m.startCollectionRun()
		return newM, cmd, true
	case key.Matches(k, sidebarKeys.NewRequest):
		return m.newFromExplorer()
	case key.Matches(k, sidebarKeys.NewFolder):
		m.prompt.Open(promptNewFolder, m.selectedFolderPath(), "New folder name", "")
		return m, nil, true
	case key.Matches(k, sidebarKeys.NewCollection):
		// Always targets the root, regardless of what's selected - unlike
		// New folder, which deliberately nests under the current selection.
		// A collection created moments ago becoming selected (the list
		// widget's own cursor semantics, not deliberate code) used to mean
		// every subsequent "new collection" attempt via New folder silently
		// landed inside the first one instead of beside it.
		m.prompt.Open(promptNewFolder, "", "New collection name", "")
		return m, nil, true
	case key.Matches(k, sidebarKeys.Rename):
		item, ok := m.sidebar.Selected()
		if !ok || item.isWorkspace {
			return m, nil, true
		}
		m.prompt.Open(promptRename, item.path, "Rename to", item.name)
		return m, nil, true
	case key.Matches(k, sidebarKeys.Delete):
		item, ok := m.sidebar.Selected()
		if !ok || item.isWorkspace {
			return m, nil, true
		}
		m.confirm.Open(confirmDelete, "Delete '"+item.name+"'?", item.path)
		return m, nil, true
	case key.Matches(k, sidebarKeys.MoveUp):
		item, ok := m.sidebar.Selected()
		if !ok || item.isWorkspace {
			return m, nil, true
		}
		if err := m.store.MoveUp(item.path); err != nil {
			m.status = "Move failed: " + err.Error()
			return m, nil, true
		}
		m.refreshTree()
		m.selectSidebarPath(item.path)
		return m, nil, true
	case key.Matches(k, sidebarKeys.MoveDown):
		item, ok := m.sidebar.Selected()
		if !ok || item.isWorkspace {
			return m, nil, true
		}
		if err := m.store.MoveDown(item.path); err != nil {
			m.status = "Move failed: " + err.Error()
			return m, nil, true
		}
		m.refreshTree()
		m.selectSidebarPath(item.path)
		return m, nil, true
	case key.Matches(k, sidebarKeys.ScrollLeft):
		m.sidebar.ScrollBy(-sidebarScrollStep)
		return m, nil, true
	case key.Matches(k, sidebarKeys.ScrollRight):
		m.sidebar.ScrollBy(sidebarScrollStep)
		return m, nil, true
	case k.String() == "enter":
		m = m.activateSidebarSelection()
		return m, nil, true
	}
	return m, nil, false
}

// activateSidebarSelection is "enter" on the currently selected sidebar row:
// toggle a folder open/closed, or load a request into the editor. Shared by
// the enter key and a mouse click on a sidebar row (mouse.go).
func (m Model) activateSidebarSelection() Model {
	item, ok := m.sidebar.Selected()
	if !ok {
		return m
	}
	// The workspace root and folders both just expand/collapse on enter.
	if item.isWorkspace || item.kind == collection.KindFolder {
		m.sidebar.ToggleFolder(item.path)
		m.persistSidebarState()
		return m
	}
	req, err := m.store.LoadRequest(item.path)
	if err != nil {
		m.status = "Load failed: " + err.Error()
		return m
	}
	m.loadRequestIntoEditor(item.path, req)
	m.status = "Loaded " + item.name
	// The Explorer is a permanent column now, not a pop-over drawer, so
	// picking a request no longer hides it - it just hands focus to the
	// Request editor so the next keystroke edits what was picked. (Earlier,
	// when this was a floating drawer, picking a request closed it so the
	// user wasn't left staring at the list they just chose from; a permanent
	// side column doesn't have that problem.) previewSidebarSelection below
	// (plain cursor movement) deliberately doesn't move focus - that's just
	// browsing, not a commit to edit.
	if m.drawerOpen() {
		m.focus = focusRequest
		m.updateFocus()
	}
	// Picking a request from the full Collections screen (as opposed to the
	// drawer overlaid on an already-open Request screen) means "go edit this
	// now" the same way - switch screens so the editor is what's shown.
	if m.screen == ScreenCollections {
		m.screen = ScreenRequest
		m.focus = focusURL
		m.updateFocus()
	}
	return m
}

// previewSidebarSelection loads whichever request the cursor now rests on
// into the Request panel immediately - called after every sidebar
// navigation keypress, so moving the cursor previews a request live
// instead of requiring an explicit enter first (closer to a file
// explorer's preview pane). Folders are deliberately left alone here -
// unlike activateSidebarSelection, arrow-key navigation over a folder must
// never toggle it open/closed, or every press over a collapsed collection
// would blow it open unexpectedly.
func (m Model) previewSidebarSelection() Model {
	item, ok := m.sidebar.Selected()
	if !ok || item.kind != collection.KindRequest {
		return m
	}
	req, err := m.store.LoadRequest(item.path)
	if err != nil {
		m.status = "Load failed: " + err.Error()
		return m
	}
	m.loadRequestIntoEditor(item.path, req)
	m.status = "Loaded " + item.name
	return m
}

// saveCurrentRequest overwrites the currently loaded request, or opens a
// save-as prompt if nothing is loaded yet.
func (m Model) saveCurrentRequest() (Model, tea.Cmd) {
	if m.loadedRequestPath == "" {
		m.prompt.Open(promptSaveAs, m.selectedFolderPath(), "Save as", "")
		return m, nil
	}
	if err := m.store.UpdateRequest(m.loadedRequestPath, m.buildRequestForSave()); err != nil {
		m.status = "Save failed: " + err.Error()
		return m, nil
	}
	// The tree caches each request's method for the sidebar's badge (see
	// buildNode in store/fs.go) - without refreshing here, a method change
	// on an already-saved request wouldn't show up until the app restarted.
	m.refreshTree()
	m.selectSidebarPath(m.loadedRequestPath)
	m.status = "Saved"
	return m, nil
}

// autosaveAfter wraps a key/mouse handler's result: if the same request is
// still loaded afterward and its content actually changed, persists it
// immediately - so an edit (typed or mouse-driven, e.g. cycling the method)
// is never lost to a forgotten ctrl+s or a crash. Comparing against m (the
// model from *before* the keystroke), not just checking nm.loadedRequestPath
// != "", is what keeps this from misfiring when the keystroke itself loaded
// a *different* request - that transition's own outgoing edits already went
// through this same path on the keystroke that made them, so there's
// nothing left unsaved to catch here. Quiet on success (an editing
// keystroke shouldn't keep overwriting the status line the way an explicit
// ctrl+s does); surfaces a failure, since losing data silently would be
// worse than not autosaving at all.
func (m Model) autosaveAfter(next tea.Model, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	nm, ok := next.(Model)
	if !ok || m.loadedRequestPath == "" || nm.loadedRequestPath != m.loadedRequestPath {
		return next, cmd
	}
	// A screen switch during this keystroke (e.g. esc off the WebSocket screen)
	// means the before/after would compare two different protocols' forms of the
	// request - which would clobber the file with the wrong shape. Only autosave
	// when we stayed put; the switch itself carries no unsaved edit.
	if m.screen != nm.screen {
		return next, cmd
	}
	// buildRequestForSave picks the HTTP or WebSocket form by the current
	// screen, so WS edits (URL/handshake headers) autosave just like HTTP ones.
	before := m.buildRequestForSave()
	after := nm.buildRequestForSave()
	if reflect.DeepEqual(before, after) {
		return next, cmd
	}
	if err := nm.store.UpdateRequest(nm.loadedRequestPath, after); err != nil {
		nm.status = "Autosave failed: " + err.Error()
		return nm, cmd
	}
	// The tree caches each request's method for the sidebar's badge, and that
	// is the only request field it holds - so only a method change needs a
	// refresh. Skipping it otherwise keeps the hot editing path (typing into
	// URL/body/params) off store.Tree, which re-reads every request file in the
	// whole collection to recompute badges - O(files) blocking disk reads per
	// keystroke, and painfully visible input lag on a large collection.
	if before.Method != after.Method {
		nm.refreshTree()
	}
	return nm, cmd
}
