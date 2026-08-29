package tui

import (
	"fmt"
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// workspaceRootPath is the sentinel path of the synthetic workspace row that
// sits at the top of the Explorer tree (the flow's step 1). It's not a real
// on-disk path, so it's namespaced with a NUL that no collection path can
// contain, and create/rename/delete deliberately treat it specially.
const workspaceRootPath = "\x00workspace"

// sidebarItem adapts one collection.TreeNode into a bubbles/list.Item, indented
// by depth so the list renders as a simple folder/request outline. The one
// exception is the synthetic workspace root (isWorkspace), which has no
// TreeNode behind it - it's the Explorer's tree root, with the collections
// nested one level under it.
type sidebarItem struct {
	kind        collection.NodeKind
	isWorkspace bool
	name        string
	path        string
	depth       int
	open        bool              // meaningful when kind == KindFolder or isWorkspace
	method      collection.Method // only set when kind == KindRequest
	// scrollOffset shifts the visible window into name, so a name too long
	// to fit can be scrolled into view a few columns at a time (see
	// sidebar.ScrollBy) instead of just sitting truncated. Only ever
	// nonzero for the currently selected row - sidebar.applyScroll clears
	// it on every other item.
	scrollOffset int
}

func (i sidebarItem) Title() string {
	// The workspace root reads as the tree's heading: a filled collapse
	// triangle plus the workspace name, accented so it stands apart from the
	// collections nested under it.
	if i.isWorkspace {
		tri := glyphTriangleRight
		if i.open {
			tri = glyphTriangleDown
		}
		return accentStyle.Bold(true).Render(tri + " " + i.name)
	}
	prefix := strings.Repeat("  ", i.depth)
	name := i.name
	if i.kind == collection.KindFolder {
		// A filled triangle carries both the nesting cue and the open/closed
		// state (down = open, right = closed), replacing the old "- " dash plus
		// "+" marker.
		tri := glyphTriangleRight
		if i.open {
			tri = glyphTriangleDown
		}
		prefix += tri + " "
		name += "/"
	} else {
		// A depth-0 request needs no marker; deeper ones get a "- " so which
		// folder an endpoint lives under reads at a glance. Pad the plain
		// method text to a fixed width before colorizing it, not after -
		// padding a string that already carries ANSI escape bytes counts those
		// bytes as visible width, throwing the alignment off (same ordering
		// history.go/runner.go already rely on for their method column).
		if i.depth > 0 {
			prefix += "- "
		}
		prefix += methodBadgeStyle(string(i.method)).Render(fmt.Sprintf("%-6s", i.method)) + " "
	}
	// The prefix (indent/marker/method badge) never scrolls - only name
	// does, so a long name can be scrolled into view without the method
	// badge or indentation drifting out of their own fixed column.
	if i.scrollOffset > 0 {
		if i.scrollOffset >= len(name) {
			name = "..."
		} else {
			name = "..." + name[i.scrollOffset:]
		}
	}
	return prefix + name
}

// Description is deliberately blank, not i.path: path is the store's opaque
// on-disk identifier (see collection.TreeNode's doc comment), and it carries the
// manual-sort-order prefix from naming.go (e.g. "010_users/020_list.json") -
// showing it raw here is what put a stray "010" under every single row. A
// URL was shown here for a while (letting two requests sharing a URL with
// different methods be recognized without opening either one), but a user
// found the extra line under every row too cluttered and asked for the
// plain single-line name back.
func (i sidebarItem) Description() string { return "" }
func (i sidebarItem) FilterValue() string { return i.name }

// flattenTree walks a collection tree depth-first into the flat item slice
// bubbles/list expects, which gets fuzzy filtering ("/" across all requests)
// for free instead of a hand-rolled tree widget. closed holds the paths of
// folders the user has collapsed - anything not in it defaults to open, so
// a freshly loaded tree (or one from before this feature existed) renders
// fully expanded exactly like before.
func flattenTree(node collection.TreeNode, depth int, closed map[string]bool) []list.Item {
	var items []list.Item
	for _, child := range node.Children {
		open := !closed[child.Path]
		items = append(items, sidebarItem{kind: child.Kind, name: child.Name, path: child.Path, depth: depth, open: open, method: child.Method})
		if child.Kind == collection.KindFolder && open {
			items = append(items, flattenTree(child, depth+1, closed)...)
		}
	}
	return items
}

type sidebar struct {
	list    list.Model
	focused bool
	root    collection.TreeNode
	// workspaceName, when set, renders as the synthetic root row of the tree
	// (the flow's step 1) with the collections nested under it. Empty means
	// "no workspace context" (e.g. a bare New() in a test), which falls back
	// to rendering the collections at the top level exactly as before.
	workspaceName string
	closed        map[string]bool // paths of folders (and the workspace root) the user has collapsed
	width         int             // content width, as given to SetSize - see View
	hScroll       int             // the selected row's current name scroll offset - see ScrollBy
}

// sidebarScrollStep is how many columns ScrollLeft/ScrollRight shift the
// selected row's name per press - a few columns at a time reads as
// scrolling; one at a time would take forever on a genuinely long name.
const sidebarScrollStep = 4

func newSidebar() sidebar {
	l := list.New(nil, newThemedDelegate(false), 0, 0)
	l.SetShowHelp(false)
	// The list's own title row duplicated the panel's own border-breaking
	// "[1] Collections" title (paneltitle.go) - one less redundant line
	// also means mouse row hit-testing (see mouse.go's sidebar.ItemIndexAt)
	// doesn't need to account for it.
	l.SetShowTitle(false)
	// The list's own status bar duplicates its empty-state message ("No
	// items" then "No items." right below it); the hint in View below
	// replaces both with something that actually says how to add one.
	l.SetShowStatusBar(false)
	return sidebar{list: l}
}

// SetFocused records whether the sidebar is the active focus zone, so View
// can render a highlighted border - otherwise nothing in the UI shows which
// panel a keypress like "n" or "N" would land in.
func (sb *sidebar) SetFocused(focused bool) {
	sb.focused = focused
}

// RefreshDelegate rebuilds the selection-highlight delegate from the
// current theme - called after cycling themes, since a list's delegate
// colors are baked in at creation time rather than re-read on every render.
func (sb *sidebar) RefreshDelegate() {
	sb.list.SetDelegate(newThemedDelegate(false))
}

func (sb *sidebar) SetTree(root collection.TreeNode) {
	sb.root = root
	sb.rebuild()
}

// SetWorkspaceName sets the workspace shown as the tree's root row and
// re-flattens. Called whenever the active workspace changes (startup, switch).
func (sb *sidebar) SetWorkspaceName(name string) {
	sb.workspaceName = name
	sb.rebuild()
}

func (sb *sidebar) rebuild() {
	sb.list.SetItems(sb.buildItems())
}

// buildItems flattens the tree into list rows. With a workspace set, a
// synthetic workspace root row goes first and the collections nest one level
// under it (collapsible via the same closed-set the folders use, keyed by
// workspaceRootPath). With no workspace (tests), it falls back to the old
// collections-at-top-level shape so nothing downstream changes.
func (sb *sidebar) buildItems() []list.Item {
	if sb.workspaceName == "" {
		return flattenTree(sb.root, 0, sb.closed)
	}
	open := !sb.closed[workspaceRootPath]
	items := []list.Item{sidebarItem{isWorkspace: true, name: sb.workspaceName, path: workspaceRootPath, depth: 0, open: open}}
	if open {
		items = append(items, flattenTree(sb.root, 1, sb.closed)...)
	}
	return items
}

// ScrollBy shifts the selected row's name scroll offset by delta columns,
// clamped to [0, len(name)] - see sidebarItem.scrollOffset. A no-op with
// nothing selected.
func (sb *sidebar) ScrollBy(delta int) {
	item, ok := sb.Selected()
	if !ok {
		return
	}
	sb.hScroll += delta
	if sb.hScroll < 0 {
		sb.hScroll = 0
	}
	if max := len(item.name); sb.hScroll > max {
		sb.hScroll = max
	}
	sb.applyScroll()
}

// resetScroll zeroes the scroll offset - called whenever the selected row
// itself changes (arrow-key navigation, a mouse click, filtering), so a
// freshly selected row always starts unscrolled rather than inheriting
// whatever offset the previous selection had drifted to.
func (sb *sidebar) resetScroll() {
	sb.hScroll = 0
	sb.applyScroll()
}

// applyScroll writes sb.hScroll onto the currently selected item and
// zeroes it on every other item - only the selected row ever shows a
// scrolled window, the same way a spreadsheet only scrolls the active
// cell, not every cell in the column.
func (sb *sidebar) applyScroll() {
	selected := sb.list.GlobalIndex()
	for i, it := range sb.list.Items() {
		si, ok := it.(sidebarItem)
		if !ok {
			continue
		}
		want := 0
		if i == selected {
			want = sb.hScroll
		}
		if si.scrollOffset != want {
			si.scrollOffset = want
			sb.list.SetItem(i, si)
		}
	}
}

// ToggleFolder flips path's expanded/collapsed state and re-flattens from
// the already-loaded tree - a pure UI toggle, so it doesn't need to re-read
// the collection from disk the way refreshTree does.
func (sb *sidebar) ToggleFolder(path string) {
	if sb.closed == nil {
		sb.closed = map[string]bool{}
	}
	sb.closed[path] = !sb.closed[path]
	sb.rebuild()
}

// ClosedPaths returns the paths of every folder currently collapsed - the
// inverse of SetClosed, used to persist this workspace's sidebar state.
func (sb sidebar) ClosedPaths() []string {
	var paths []string
	for path, closed := range sb.closed {
		if closed {
			paths = append(paths, path)
		}
	}
	return paths
}

// SetClosed replaces the set of collapsed folders with paths and
// re-flattens from the already-loaded tree - used to restore a workspace's
// sidebar state on startup/switch, the inverse of ClosedPaths.
func (sb *sidebar) SetClosed(paths []string) {
	closed := make(map[string]bool, len(paths))
	for _, path := range paths {
		closed[path] = true
	}
	sb.closed = closed
	sb.rebuild()
}

func (sb *sidebar) SetSize(w, h int) {
	sb.width = w
	// The empty-state hint (see emptyHint) is appended below whatever the
	// list itself renders - its lines have to come OUT of the list's own
	// height budget, or the sidebar renders 1-2 rows taller than h every
	// time there are no items, and wideGridView stretches every panel to
	// match the tallest one, pushing the whole grid (and the top bar with
	// it) down past the terminal's actual height.
	listHeight := h
	if len(sb.list.Items()) == 0 {
		listHeight -= lipgloss.Height(sb.emptyHint())
	}
	sb.list.SetSize(w, listHeight)
}

// emptyHint is the "n: new request   N: new folder" message shown in
// place of the list when there are no items - pre-wrapped to the sidebar's
// own known width rather than left to the border style's own .Width() call
// in View(), which word-wraps over-budget content instead of clipping it
// (see kvtable.go's emptyStateView for the same fix, applied there first).
func (sb sidebar) emptyHint() string {
	hintWidth := sb.width - 2 // -2 for this style's own Padding(0, 1)
	if hintWidth < 10 {
		hintWidth = 10
	}
	return ansi.Wordwrap("n: new request   N: new folder", hintWidth, "")
}

func (sb sidebar) Selected() (sidebarItem, bool) {
	it, ok := sb.list.SelectedItem().(sidebarItem)
	return it, ok
}

// SelectPath moves the cursor to the item with the given path, if present -
// used after create/rename so the affected node stays highlighted.
func (sb *sidebar) SelectPath(path string) {
	for i, it := range sb.list.Items() {
		if si, ok := it.(sidebarItem); ok && si.path == path {
			sb.list.Select(i)
			return
		}
	}
}

// IsFiltering reports whether the user is mid-way through typing a fuzzy
// filter, so root.go knows to pass every keystroke straight through instead
// of intercepting letters as commands (n/r/d/etc).
func (sb sidebar) IsFiltering() bool {
	return sb.list.FilterState() == list.Filtering
}

func (sb sidebar) Update(msg tea.Msg) (sidebar, tea.Cmd) {
	var cmd tea.Cmd
	sb.list, cmd = sb.list.Update(msg)
	return sb, cmd
}

func (sb sidebar) View() string {
	body := sb.list.View()
	if len(sb.list.Items()) == 0 {
		body += "\n" + labelStyle.Render(sb.emptyHint())
	}
	style := borderStyle
	if sb.focused {
		style = focusedBorder
	}
	// Explicit Width() so the box always renders at its configured size -
	// without it, this style sizes itself to whatever content it holds, so
	// an empty ("No items.") sidebar rendered far narrower than the fraction
	// of the terminal a user expected, regardless of what SetSize was told.
	// +2: this Width() budget includes the style's own Padding(0, 1), same
	// convention as the env box and method box in view.go.
	return style.Width(sb.width + 2).Render(body)
}
