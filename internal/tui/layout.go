package tui

import workspace "github.com/dhitalkamal/parley/internal/workspace/domain"

// currentLayout builds a workspace.Layout snapshot from the model's current
// orientation/collapse state - the inverse of applyLayout.
func (m Model) currentLayout() workspace.Layout {
	return workspace.Layout{
		Orientation:         m.orientation,
		RequestCollapsed:    m.requestCollapsed,
		ResponseCollapsed:   m.responseCollapsed,
		ShortcutsRailHidden: m.shortcutsRailHidden,
		RailMode:            int(m.railMode),
	}
}

// applyLayout adopts l as the model's current orientation/collapse state -
// called on startup (SetWorkspaces) and on switching workspaces, so each
// workspace's own saved arrangement takes effect immediately rather than
// leaking over from whichever workspace was active before.
func (m *Model) applyLayout(l workspace.Layout) {
	m.orientation = l.Orientation
	m.requestCollapsed = l.RequestCollapsed
	m.responseCollapsed = l.ResponseCollapsed
	m.shortcutsRailHidden = l.ShortcutsRailHidden
	m.railMode = railMode(l.RailMode)
	// The restored mode may switch the rail back into its env view - reload
	// its rows so it shows the active environment straight away.
	if m.railEnvMode() {
		m.refreshRailEnv()
	}
}

// responseZoneVisible reports whether the response zone should show at all -
// derived from the response itself (HasContent) rather than a separate
// stored flag, so there's no bookkeeping to keep in sync at every send/load/
// reset site: loading a request with no cached response already resets
// m.response to an empty responseView (see loadRequestIntoEditor), and a
// completed send already calls SetResponse/SetError - both naturally flip
// this without any extra wiring.
func (m Model) responseZoneVisible() bool {
	return m.response.HasContent()
}

// persistLayout saves the model's current layout into the active
// workspace's registry entry. A no-op with no workspace store wired up
// (most tests, or a plain single-workspace run) or no matching active
// entry yet - the same nil-store-is-a-no-op convention every workspace
// action in workspace_actions.go already follows.
func (m *Model) persistLayout() {
	if m.workspaceStore == nil {
		return
	}
	for i, ws := range m.workspaces {
		if ws.Name == m.activeWorkspaceName {
			ws.Layout = m.currentLayout()
			m.workspaces[i] = ws
			_ = m.workspaceStore.Save(ws)
			return
		}
	}
}

// restoreSidebarState adopts ws's saved sidebar state - which folders are
// collapsed and which request was last loaded - called on startup
// (SetWorkspaces) and on switching to ws (switchWorkspace), mirroring
// applyLayout's restore of the same workspace's layout. Loads the saved
// request straight from the store rather than going through the sidebar's
// current selection, since the request may sit behind a folder that's
// itself collapsed (also being restored here) and so wouldn't be reachable
// via the flattened list.
func (m *Model) restoreSidebarState(ws workspace.Workspace) {
	m.sidebar.SetClosed(ws.ClosedFolders)
	if ws.SelectedPath == "" {
		return
	}
	req, err := m.store.LoadRequest(ws.SelectedPath)
	if err != nil {
		return
	}
	m.loadRequestIntoEditor(ws.SelectedPath, req)
	m.sidebar.SelectPath(ws.SelectedPath)
}

// persistSidebarState saves which request is currently loaded and which
// folders are collapsed into the active workspace's registry entry, the
// same read-modify-write pattern persistLayout uses - so both restore
// automatically the next time this workspace is opened.
func (m *Model) persistSidebarState() {
	if m.workspaceStore == nil {
		return
	}
	for i, ws := range m.workspaces {
		if ws.Name == m.activeWorkspaceName {
			ws.SelectedPath = m.loadedRequestPath
			ws.ClosedFolders = m.sidebar.ClosedPaths()
			m.workspaces[i] = ws
			_ = m.workspaceStore.Save(ws)
			return
		}
	}
}
