// Package tui holds the Bubble Tea models and views. root.go is the
// composition root: it owns focus routing between the sidebar, method
// selector, URL input, params/headers tables, and body editor, and wires
// sends through execapp.SendRequest and persistence through collection.Store.
package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	collectionstore "github.com/dhitalkamal/parley/internal/collection/infrastructure"
	environment "github.com/dhitalkamal/parley/internal/environment/domain"
	environmentstore "github.com/dhitalkamal/parley/internal/environment/infrastructure"
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
	"github.com/dhitalkamal/parley/internal/execution/infrastructure/httpclient"
	executionstore "github.com/dhitalkamal/parley/internal/execution/infrastructure/store"
	"github.com/dhitalkamal/parley/internal/execution/infrastructure/wsclient"
	history "github.com/dhitalkamal/parley/internal/history/domain"
	historystore "github.com/dhitalkamal/parley/internal/history/infrastructure"
	scriptengine "github.com/dhitalkamal/parley/internal/scripting/infrastructure"
	workspace "github.com/dhitalkamal/parley/internal/workspace/domain"
)

// Focus zones collapsed to 5 (from 7) once the Request panel became tabbed:
// Params/Headers/Body/Pre-req/Tests used to each be their own focus stop,
// but with only one visible at a time (see reqtab.go), "focused on the
// Request panel" plus "which tab is active" fully describes where input
// goes - a separate focus stop per tab would be redundant.
//
// Method..Response are the six baseline Tab stops (focusTabCount of them),
// written to match how the screen actually reads left-to-right,
// top-to-bottom: Method, URL, Env, Send along the top row, then Request/
// Response. focusSidebar (the collections drawer's own list) is spliced
// into the cycle right after Send, but only conditionally - see focus.go's
// tabStops, which includes it only while the drawer is actually open
// (there's nothing to tab into otherwise). Visibility no longer implies
// focus (see drawer.go's drawerOpen doc comment): tabbing through Sidebar
// never closes the drawer, and tabbing away from it never opens one that
// wasn't already open - only ctrl+\ (ToggleDrawer) or esc do that. Send was
// mouse/ctrl+r-only for a while (a "flat button" with no border to show
// focus on), the same way Env used to be mouse/ctrl+e-only before it got
// its own focus border and Enter/Space activation - Send now gets the
// identical treatment for the same reason: a keyboard-only user tabbing
// through the row needs to be able to reach and activate it too.
const (
	focusMethod = iota
	focusURL
	focusEnv
	focusSend
	focusRequest
	focusResponse
	focusTabCount // number of the six baseline Tab stops (Method..Response)
	focusSidebar
	// focusRail is the side rail's own focus stop, spliced into the cycle
	// after Response but only while the rail is actually on screen - the same
	// conditional-visibility-implies-a-stop pattern focusSidebar follows for
	// the drawer (see focus.go's tabStops). It only takes focus so the rail's
	// env view can be edited in place; the shortcuts view has nothing to edit
	// but is still a stop, purely so "v" (switch rail view) is reachable from
	// it (see railenv_keys.go).
	focusRail
	focusCount
	focusNone = -1
)

// Model is the root Bubble Tea model: sidebar + request editor + response viewer.
type Model struct {
	// screen is which primary full-frame view is showing - see screen.go.
	// previousScreen is whatever screen was active before switching into
	// ScreenDashboard, ScreenSettings, or ScreenCollections, so their own
	// back-key returns to it instead of always landing on the same screen
	// regardless of where the user actually was.
	screen         Screen
	previousScreen Screen

	sidebar sidebar
	store   collection.Store

	envStore      environment.EnvironmentStore
	globals       environment.Environment
	envNames      []string
	activeEnvName string
	activeEnv     environment.Environment
	showVariables bool
	envPanel      envPanelState
	envDropdown   envDropdownState

	// railMode selects which of the side rail's two views shows: the compact
	// editable environment editor (railModeEnv, the default) or the shortcuts
	// reference list. railEnv is that editor's own state - it reuses
	// envPanelState's variable-editor half (rows/table/edit inputs), scoped
	// to whichever environment is active (globals when none is), see
	// railmode.go. Both persist in the active workspace's Layout.
	railMode railMode
	// railEnv holds the active named environment's variables; railGlobals holds
	// the shared globals (always shown, and editable). railSection tracks which
	// of the panel's three stacked sections the cursor is in - the Active
	// selector, this environment's variables, or the globals - so up/down move
	// between them and edits route to the right scope. See railenv_keys.go.
	railEnv     envPanelState
	railGlobals envPanelState
	railSection railSection

	// bodyTypeDropdown/contentTypeDropdown back the Request zone header's
	// body-type/content-type controls (see bodydropdown.go) - the
	// replacement for the old inline tab row inside the body editor itself.
	bodyTypeDropdown    labelDropdownState
	contentTypeDropdown labelDropdownState

	workspaceStore      workspace.WorkspaceStore
	workspaces          []workspace.Workspace
	activeWorkspaceName string
	newWorkspacesDir    string
	workspace           workspaceState

	showCodeSnippet bool

	// revealSecrets toggles whether masked values (password/token/secret/
	// authorization keys) render as their literal value or as "****" in the
	// request/response body views - see secretmask.go.
	revealSecrets bool

	historyStore history.HistoryStore
	history      historyState
	runner       runnerState
	dashboard    dashboardState
	// runHistoryStore persists every collection run (interactive or via
	// `parley run`) so the dashboard can show pass/fail and latency trends
	// across runs over time - see runner.go's handleRunResult.
	runHistoryStore history.RunHistoryStore

	loadedRequestPath string
	responseCache     map[string]responseView
	// lastResponseStore persists each request's most recent response to
	// disk, keyed by its Store path - responseCache alone only lasts for
	// the current process, so navigating to a request after restarting the
	// app would otherwise always show the empty state even though it was
	// sent before.
	lastResponseStore execution.LastResponseStore
	prompt            promptState
	confirm           confirmState
	palette           paletteState
	kvAdd             kvAddState

	methodIdx   int
	urlInput    textinput.Model
	params      kvTable
	headers     kvTable
	body        bodyEditor
	scripts     scriptEditor
	auth        authEditor
	reqSettings requestSettingsEditor
	response    responseView
	help        help.Model
	client      execution.HTTPClient

	// ws is the WebSocket screen's state - two independent panes shown side by
	// side (see wsScreen); wsDialer opens connections for it, injected as a port
	// so tests can drive the flow with a fake instead of a real socket.
	ws       wsScreen
	wsDialer execution.WSDialer

	scriptRunner execution.ScriptRunner

	focus int
	// mode is the vim-style input mode (see mode.go). NORMAL (the zero value)
	// is command mode; INSERT is text entry into the focused field.
	mode editorMode
	// pendingG is set after a bare "g" in NORMAL, arming the vim goto prefix -
	// the next key completes it (gg top, gc/gd/gs/gw/gh a screen). Cleared as
	// soon as any key resolves it. See handleKey.
	pendingG bool
	reqTab   reqTab
	status   string
	width    int
	height   int

	// detectedVPNs are the VPN/tunnel interfaces currently up on this machine
	// (empty for none), refreshed once a second off the clock tick - see
	// netcheck.DetectVPNs. Read-only detection: parley never connects or
	// disconnects anything. detectedVPN is the same list joined for display.
	// Used by the Environment panel (both the raw list and the per-env
	// expected-VPN match, see railvpn.go) and to enrich a failed request's
	// error (the smart failure hint in send.go).
	detectedVPN  string
	detectedVPNs []string

	// sending/sendStartedAt drive the Response panel's "Sending... 123 ms"
	// indicator (see resppanel.go) - a user sent a request and had no
	// feedback at all that anything was happening until the result came
	// back. sendTickCmd (send.go) re-arms itself every 200ms while sending
	// is true purely to force a repaint so the elapsed time actually ticks;
	// it carries no data of its own.
	sending       bool
	sendStartedAt time.Time

	// orientation/requestCollapsed/responseCollapsed/shortcutsRailHidden are
	// the workspace's request/response zone arrangement - the in-memory
	// mirror of the active workspace's workspace.Layout (see layout.go).
	// Zero values (vertical, both expanded, rail shown) match Layout's own
	// zero value, so this behaves correctly even with no workspace store
	// wired up.
	orientation         workspace.Orientation
	requestCollapsed    bool
	responseCollapsed   bool
	shortcutsRailHidden bool

	// responseMaximized is a transient zoom state, not part of the saved
	// workspace.Layout (unlike the two above) - it always starts false on
	// launch/workspace switch, the same way a terminal multiplexer's pane
	// zoom doesn't persist across restarts.
	responseMaximized bool

	// drawerVisible tracks the collections drawer independently of focus -
	// a user asked to be able to Tab from its list into Request/Response
	// and back without the drawer disappearing every time focus left it, so
	// "visible" and "focused" are no longer the same fact the way an
	// earlier version had it (see drawer.go's drawerOpen doc comment).
	drawerVisible bool

	// lastEscAt tracks an at-rest esc (nothing open to back out of) so a
	// second one within escDoubleTapWindow opens the quit confirmation,
	// instead of a lone esc instantly exiting the whole app.
	lastEscAt time.Time
}

// New builds the root model, backed by a collections tree rooted at
// collectionsRoot and environments/globals rooted at projectRoot, ready to
// run under tea.NewProgram.
func New(collectionsRoot, projectRoot string) Model {
	ti := textinput.New()
	ti.Placeholder = "https://api.example.com/users"
	ti.Prompt = ""

	m := Model{
		screen:            ScreenRequest,
		drawerVisible:     true,
		sidebar:           newSidebar(),
		store:             collectionstore.New(collectionsRoot),
		envStore:          environmentstore.New(projectRoot),
		historyStore:      historystore.New(projectRoot),
		runHistoryStore:   historystore.NewRunStore(projectRoot),
		lastResponseStore: executionstore.New(projectRoot),
		prompt:            newPromptState(),
		palette:           newPaletteState(),
		kvAdd:             newKVAddState(),
		urlInput:          ti,
		params:            newKVTable("Query params", "Key", "Query params are appended to the URL after ?key=value"),
		headers:           newKVTable("Headers", "Key", "Headers are sent as literal HTTP request headers"),
		body:              newBodyEditor(),
		scripts:           newScriptEditor(),
		auth:              newAuthEditor(),
		reqSettings:       newRequestSettingsEditor(),
		response:          newResponseView(),
		responseCache:     map[string]responseView{},
		help:              help.New(),
		client:            httpclient.New(),
		ws:                newWSScreen(),
		wsDialer:          wsclient.New(),
		scriptRunner:      scriptengine.New(),
		// Start on the Explorer, the head of the flow (Workspace > Collection >
		// Folder > Request) - so "n" (new) works immediately and a fresh
		// session lands where creation begins, rather than on the URL bar of a
		// request that may not exist yet.
		focus:  focusSidebar,
		status: "Ready",
	}
	// params/headers render flush inside the Request panel's own border, no
	// second box (see kvTable.borderless / reqpanel.go).
	m.params.SetBorderless(true)
	m.headers.SetBorderless(true)
	m.refreshTree()
	if globals, err := m.envStore.LoadGlobals(); err == nil {
		m.globals = globals
	}
	if names, err := m.envStore.ListEnvironments(); err == nil {
		m.envNames = names
	}
	// Restore whichever environment was active last session, instead of
	// always resetting to "none" - a user shouldn't have to reselect it via
	// ctrl+e on every single launch.
	if name, err := m.envStore.ActiveName(); err == nil && name != "" {
		if env, err := m.envStore.LoadEnvironment(name); err == nil {
			m.activeEnvName = name
			m.activeEnv = env
		}
	}
	// Prime the rail's env editor from whichever scope is active now, so the
	// rail shows the right variables on the very first frame instead of only
	// after it's been focused once.
	m.refreshRailEnv()
	// VPN detection runs off-loop right after Init (see vpnpoll.go); leaving it
	// out here means one frame with no VPN list rather than a possible startup
	// stall on net.Interfaces.
	m.updateFocus()
	return m
}

func (m Model) Init() tea.Cmd {
	// clock re-arms itself; VPN detection runs off-loop and paces itself (see
	// vpnpoll.go) - kick off the first detection here.
	return tea.Batch(textinput.Blink, clockTickCmd(), detectVPNsCmd())
}

// applyLayoutWidths sizes the url bar from the current terminal width -
// called on resize. Every other width-dependent widget (sidebar, request
// panel tabs, response viewport) is (re)sized fresh on every render inside
// mainView/requestPanelView/responsePanelView instead, since their budgets
// depend on dynamic per-render state (which tab is active, whether a
// warning line is showing) that a resize-only hook can't see.
func (m *Model) applyLayoutWidths() {
	// urlBoxOuterWidth(...) - 2 for the box's own border, - 2 more for its
	// Padding(0,1) - matches urlRowView's own box-width math so the input's
	// internal scrolling stays in sync with what's actually rendered.
	m.urlInput.Width = urlBoxOuterWidth(urlRowWidth(m.width)) - 4
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.applyLayoutWidths()
		m.resizeWS()
		return m, nil

	case wsConnectedMsg:
		return m.onWSConnected(msg)
	case wsConnectErrMsg:
		return m.onWSConnectErr(msg)
	case wsIncomingMsg:
		return m.onWSIncoming(msg)
	case wsRecvErrMsg:
		return m.onWSRecvErr(msg)
	case wsClosedMsg:
		return m.onWSClosed(msg)
	case wsSendErrMsg:
		return m.onWSSendErr(msg)
	case wsPingResultMsg:
		return m.onWSPingResult(msg)

	case clockTickMsg:
		// just re-arm the clock - VPN detection is off-loop now (see vpnpoll.go),
		// because running that syscall here every second could freeze the UI.
		return m, clockTickCmd()

	case vpnDetectedMsg:
		m.detectedVPNs = []string(msg)
		m.detectedVPN = strings.Join(m.detectedVPNs, ", ")
		// wait, then poll again - only ever one detection in flight.
		return m, scheduleVPNPollCmd()

	case vpnPollMsg:
		return m, detectVPNsCmd()

	case sendTickMsg:
		if !m.sending {
			return m, nil
		}
		return m, sendTickCmd()

	case sendResultMsg:
		return m.handleSendResult(msg)

	case refreshResultMsg:
		return m.handleRefreshResult(msg)

	case runResultMsg:
		return m.handleRunResult(msg), nil

	case tea.KeyMsg:
		return m.autosaveAfter(m.handleKey(msg))

	case tea.MouseMsg:
		return m.autosaveAfter(m.handleMouse(msg))
	}

	// Anything else (e.g. a list widget's async FilterMatchesMsg from its
	// fuzzy filter, or a future spinner tick) isn't a type root.go cares
	// about directly, but the sidebar's and palette's own tea.Cmd calls still
	// need it delivered back in, or their results never land - each list is
	// its own list.Model instance, so both need the same forwarding.
	var sidebarCmd, paletteCmd tea.Cmd
	m.sidebar, sidebarCmd = m.sidebar.Update(msg)
	m.palette.list, paletteCmd = m.palette.list.Update(msg)
	return m, tea.Batch(sidebarCmd, paletteCmd)
}
