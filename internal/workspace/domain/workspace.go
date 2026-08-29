package workspace

// Orientation is how the request/response workspace zones are arranged -
// see Layout.
type Orientation int

const (
	// OrientationVertical stacks request over response, each full width -
	// the default, and the zero value so a Workspace loaded from a
	// registry file written before Layout existed defaults to it.
	OrientationVertical Orientation = iota
	// OrientationHorizontal places request and response side by side, each
	// full height.
	OrientationHorizontal
)

// Layout is a workspace's saved request/response zone arrangement - which
// way they're stacked, and whether either is collapsed to just its header
// bar. Its zero value (vertical, both expanded) is also what a Workspace
// loaded from a registry file predating Layout gets, so this field is
// additive: old registry files need no migration.
type Layout struct {
	Orientation         Orientation
	RequestCollapsed    bool
	ResponseCollapsed   bool
	ShortcutsRailHidden bool
	// RailMode is which of the side rail's two views is showing - the
	// compact environment editor or the shortcuts reference list. Stored as
	// a plain int (the tui package owns the railMode* constants) so this
	// domain type stays free of any presentation dependency. Additive like
	// the fields above: its zero value is the environment view, which is
	// also the default a registry file predating this field loads as, so
	// old registry files need no migration.
	RailMode int
}

// Workspace is a named, self-contained set of collections and environments -
// each workspace points at its own CollectionsRoot and ProjectRoot (the
// project root also backs environments/history, same as it does today for a
// single-workspace setup). Switching workspaces means re-pointing the
// app's store/envStore/historyStore at a different Workspace's roots.
type Workspace struct {
	Name            string
	CollectionsRoot string
	ProjectRoot     string
	Layout          Layout
	// SelectedPath is the request last loaded into the editor in this
	// workspace, restored on relaunch so a session picks back up on the
	// same request instead of starting with nothing loaded. Additive like
	// Layout: a registry file predating this field loads it as "", which
	// already means "nothing loaded" today.
	SelectedPath string
	// ClosedFolders is the set of folder paths this workspace's sidebar has
	// collapsed, restored on relaunch so a folder left collapsed (or left
	// expanded) stays that way across restarts. Additive like Layout: a
	// registry file predating this field loads it as nil, which already
	// means "fully expanded" today.
	ClosedFolders []string
}

// WorkspaceStore persists the workspace registry (the list of known
// workspaces plus which one is active) - implemented by
// infrastructure/store, independent of any single workspace's own data.
type WorkspaceStore interface {
	List() ([]Workspace, error)
	Save(ws Workspace) error
	Delete(name string) error
	ActiveName() (string, error)
	SetActiveName(name string) error
}
