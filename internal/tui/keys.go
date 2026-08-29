package tui

import "github.com/charmbracelet/bubbles/key"

// keyMap defines every keybinding the root model recognizes. It implements
// help.KeyMap so the same bindings drive both key handling and the help bar.
type keyMap struct {
	NextFocus          key.Binding
	PrevFocus          key.Binding
	JumpRequest        key.Binding
	JumpResponse       key.Binding
	JumpRail           key.Binding
	ToggleDrawer       key.Binding
	ToggleOrientation  key.Binding
	ToggleRequestZone  key.Binding
	ToggleResponseZone key.Binding
	ToggleSideRail     key.Binding
	MaximizeResponse   key.Binding
	PrevReqTab         key.Binding
	NextReqTab         key.Binding
	Send               key.Binding
	Save               key.Binding
	ToggleTab          key.Binding
	Search             key.Binding
	SaveExample        key.Binding
	History            key.Binding
	AddRow             key.Binding
	DeleteRow          key.Binding
	ToggleRow          key.Binding
	EditRow            key.Binding
	Environments       key.Binding
	GoEnvPanel         key.Binding
	Workspaces         key.Binding
	Dashboard          key.Binding
	Settings           key.Binding
	ToggleVars         key.Binding
	Import             key.Binding
	Export             key.Binding
	CodeSnippet        key.Binding
	RevealSecrets      key.Binding
	Palette            key.Binding
	Help               key.Binding
	RestEsc            key.Binding
	Quit               key.Binding
}

var keys = keyMap{
	NextFocus:    key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next panel")),
	PrevFocus:    key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev panel")),
	JumpRequest:  key.NewBinding(key.WithKeys("f2"), key.WithHelp("f2", "request")),
	JumpResponse: key.NewBinding(key.WithKeys("f3"), key.WithHelp("f3", "response")),
	// f10, the next free function key after f1-f9 (same never-collides-with-
	// typing series): jumps focus to the side rail's env editor, unhiding it
	// if needed - the third panel's counterpart to f2/f3's Request/Response
	// jumps.
	JumpRail: key.NewBinding(key.WithKeys("f10"), key.WithHelp("f10", "environment rail")),
	// ctrl+\ (a real control code, unlike ctrl+<digit> below) rather than
	// f5: f5 used to force a single-panel view of whichever of 3 grid
	// panels had focus, back when Collections was a permanent grid column -
	// now that it's an overlay drawer (see drawer.go) instead, there's no
	// longer a 3-way panel switch to force down to one, and independently
	// collapsing whichever zone isn't wanted (ToggleRequestZone/
	// ToggleResponseZone below) already covers "see just one zone" for
	// Request/Response. This is also the ONLY way to open the drawer, and
	// the only way to close it besides esc while it has focus - it used to
	// also be reachable via f1 (JumpCollections), which was removed so
	// there's exactly one dedicated shortcut to open it (see root.go's
	// focusTabCount doc comment for how Tab now interacts with it).
	ToggleDrawer: key.NewBinding(key.WithKeys("ctrl+\\"), key.WithHelp("ctrl+\\", "collections drawer")),
	// f6/f7/f8, not ctrl+1/ctrl+2 as an early spec draft asked for: ctrl+
	// <digit> has no standard control-code representation in a terminal
	// (unlike ctrl+a..z or ctrl+\) - most terminals deliver a bare "1" or
	// "2" with no way to tell a ctrl chord was held at all, so those
	// bindings could never actually fire. f6-f8 follow the same
	// never-collides-with-typing reasoning as f1-f5 above.
	ToggleOrientation:  key.NewBinding(key.WithKeys("f6"), key.WithHelp("f6", "toggle orientation")),
	ToggleRequestZone:  key.NewBinding(key.WithKeys("f7"), key.WithHelp("f7", "toggle request zone")),
	ToggleResponseZone: key.NewBinding(key.WithKeys("f8"), key.WithHelp("f8", "toggle response zone")),
	// f1, reclaimed: it used to be JumpCollections (removed when the drawer
	// became ctrl+\-only, see ToggleDrawer's comment) and has been free
	// since - the same never-collides-with-typing function-key reasoning as
	// f2-f8. Hides/shows the whole side rail; "v" while the rail has focus
	// switches which view it shows (env config vs shortcuts).
	ToggleSideRail: key.NewBinding(key.WithKeys("f1"), key.WithHelp("f1", "toggle side rail")),
	// ctrl+z, not another function key: f9 is already Settings and there's
	// no f10+ left in the never-collides-with-typing series above, but
	// ctrl+z is itself never bound by bubbles/textarea or textinput's
	// defaults - free the same way ctrl+t/ctrl+f (also chords, also unused
	// by those widgets) already are.
	MaximizeResponse: key.NewBinding(key.WithKeys("ctrl+z"), key.WithHelp("ctrl+z", "zoom response")),
	// shift+left/shift+right rather than ctrl+left/ctrl+right: this app
	// tried ctrl+left/right first, and it silently switched tabs out from
	// under someone mid-edit instead of moving their cursor a word over -
	// bubbles/textarea and textinput both bind that combo as word
	// navigation by default. shift+left/right isn't bound by either, and
	// unlike the f4/f5 that replaced ctrl+left/right, it never needs a Fn
	// modifier on a laptop keyboard.
	PrevReqTab: key.NewBinding(key.WithKeys("shift+left"), key.WithHelp("shift+left", "prev tab")),
	NextReqTab: key.NewBinding(key.WithKeys("shift+right"), key.WithHelp("shift+right", "next tab")),
	// ctrl+j alongside ctrl+r: ctrl+j is the terminal-safe "send" the researched
	// TUI clients settle on (it's ctrl+enter's legacy code, survives every
	// emulator - posting uses it), while ctrl+r stays as a familiar alias.
	Send:         key.NewBinding(key.WithKeys("ctrl+j", "ctrl+r"), key.WithHelp("ctrl+j", "send")),
	Save:         key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("ctrl+s", "save")),
	ToggleTab:    key.NewBinding(key.WithKeys("ctrl+t"), key.WithHelp("ctrl+t", "body/headers/cookies")),
	Search:       key.NewBinding(key.WithKeys("ctrl+f"), key.WithHelp("ctrl+f", "search response")),
	SaveExample:  key.NewBinding(key.WithKeys("ctrl+x"), key.WithHelp("ctrl+x", "save example")),
	History:      key.NewBinding(key.WithKeys("ctrl+y"), key.WithHelp("ctrl+y", "history")),
	AddRow:       key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add row")),
	DeleteRow:    key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete row")),
	ToggleRow:    key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "enable/disable")),
	EditRow:      key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "edit row")),
	Environments: key.NewBinding(key.WithKeys("ctrl+e"), key.WithHelp("ctrl+e", "environments")),
	// alt+e (not super+e): the Super/Cmd key is grabbed by the window manager
	// and never reaches a terminal app, so alt+e is the deliverable binding for
	// jumping into the Environment panel. f10 also lands there.
	GoEnvPanel: key.NewBinding(key.WithKeys("alt+e"), key.WithHelp("alt+e", "environment panel")),
	// f4 rather than a ctrl-chord: freed up when prev/next-tab moved to
	// shift+left/right (see PrevReqTab/NextReqTab above), and function keys
	// are never bound by bubbles/textarea or textinput's own default editing
	// keys, so this can never collide with typing.
	Workspaces: key.NewBinding(key.WithKeys("f4"), key.WithHelp("f4", "workspaces")),
	// f5, not a ctrl-chord: freed up when the old single-panel-focus binding
	// was removed (see ToggleDrawer's comment) and never reused since - the
	// same never-collides-with-typing reasoning as f2-f4/f6-f8.
	Dashboard: key.NewBinding(key.WithKeys("f5"), key.WithHelp("f5", "dashboard")),
	// f9, the next free function key in the same never-collides-with-typing
	// series as f2-f8 above.
	Settings:   key.NewBinding(key.WithKeys("f9"), key.WithHelp("f9", "settings")),
	ToggleVars: key.NewBinding(key.WithKeys("ctrl+w"), key.WithHelp("ctrl+w", "variables")),
	Import:     key.NewBinding(key.WithKeys("ctrl+l"), key.WithHelp("ctrl+l", "import")),
	Export:     key.NewBinding(key.WithKeys("ctrl+o"), key.WithHelp("ctrl+o", "export")),
	// ctrl+n rather than the code snippet panel's old ctrl+u: ctrl+u was
	// reassigned to RevealSecrets (masked values in the request/response
	// body need a quick toggle, and ctrl+u was the spec's chosen chord for
	// it) - both chords already collide with a bubbles/textarea emacs
	// default (DeleteBeforeCursor / LineNext respectively) and are handled
	// by the same textEntryFocused() guard as every other colliding chord.
	CodeSnippet:   key.NewBinding(key.WithKeys("ctrl+n"), key.WithHelp("ctrl+n", "code snippet")),
	RevealSecrets: key.NewBinding(key.WithKeys("ctrl+u"), key.WithHelp("ctrl+u", "reveal secrets")),
	Palette:       key.NewBinding(key.WithKeys("ctrl+k"), key.WithHelp("ctrl+k", "command palette")),
	Help:          key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	RestEsc:       key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	Quit:          key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "force quit")),
}

func (k keyMap) ShortHelp() []key.Binding {
	// NextReqTab (not its reverse, PrevReqTab - same one-direction-only
	// convention as NextFocus/PrevFocus below) was previously only visible
	// in the expanded ('?') help, which a user read as it not existing at
	// all - the collapsed bar is the only help most people ever see.
	return []key.Binding{k.NextFocus, k.NextReqTab, k.Send, k.Save, k.Palette, k.Help, k.RestEsc}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.NextFocus, k.PrevFocus, k.JumpRequest, k.JumpResponse, k.JumpRail, k.ToggleDrawer},
		{k.ToggleOrientation, k.ToggleRequestZone, k.ToggleResponseZone, k.ToggleSideRail, k.MaximizeResponse},
		{k.PrevReqTab, k.NextReqTab, k.Send, k.Save, k.ToggleTab},
		{k.Search, k.SaveExample, k.History},
		{k.AddRow, k.DeleteRow, k.ToggleRow, k.EditRow},
		{sidebarKeys.NewRequest, sidebarKeys.NewFolder, sidebarKeys.NewCollection, sidebarKeys.Rename},
		{sidebarKeys.Delete, sidebarKeys.MoveUp, sidebarKeys.MoveDown, sidebarKeys.Run},
		{sidebarKeys.ScrollLeft, sidebarKeys.ScrollRight},
		{k.Environments, k.Workspaces, k.Dashboard, k.Settings},
		{k.ToggleVars, k.Import, k.Export, k.CodeSnippet, k.RevealSecrets},
		{k.Palette, k.Help, k.RestEsc, k.Quit},
	}
}
