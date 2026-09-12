package tui

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

// Theme names every color the app renders with. Every render call site goes
// through the package-level style vars below rather than a Theme field on
// Model, so cycling themes is just reassigning those vars - the alternative
// (threading a Theme through every one of the ~50 render call sites across
// the package) is a lot of churn for no behavioral difference, since there's
// only ever one active theme for the single running TUI process.
type Theme struct {
	Name string

	// Chrome. BG/Panel/PanelAlt are used sparingly (top bar, tab bar) rather
	// than painted across every content box - a terminal full of solid-color
	// panels reads as noisy rather than polished, unlike a browser canvas.
	BG            string
	Panel         string
	PanelAlt      string
	Fg            string
	FgDim         string
	Border        string
	FocusedBorder string
	Accent        string
	AccentFg      string // contrast text for a filled Accent background
	Accent2       string

	MethodGet    string
	MethodPost   string
	MethodPut    string
	MethodPatch  string
	MethodDelete string

	Success string
	Err     string
	Warn    string
	Sel     string // selected-row background
	Secret  string // masked variable value

	Status2xx string
	Status3xx string
	Status4xx string
	Status5xx string
}

// themes mirrors the four named designs from the design reference exactly,
// Ayu Midnight first as the default.
var themes = []Theme{
	{
		Name:          "Ayu Midnight",
		BG:            "#0b0e14",
		Panel:         "#0d1017",
		PanelAlt:      "#11151f",
		Fg:            "#bfbdb6",
		FgDim:         "#565b66",
		Border:        "#2b303b",
		FocusedBorder: "#59c2ff",
		Accent:        "#ffb454",
		AccentFg:      "#0b0e14",
		Accent2:       "#59c2ff",
		MethodGet:     "#aad94c", MethodPost: "#ffb454", MethodPut: "#d2a6ff", MethodPatch: "#e6b450", MethodDelete: "#f07178",
		Success: "#aad94c", Err: "#f07178", Warn: "#e6b450", Sel: "#1b2531", Secret: "#565b66",
		Status2xx: "#aad94c", Status3xx: "#e6b450", Status4xx: "#ffb454", Status5xx: "#f07178",
	},
	{
		Name:          "Gruvbox",
		BG:            "#1d2021",
		Panel:         "#1d2021",
		PanelAlt:      "#282828",
		Fg:            "#ebdbb2",
		FgDim:         "#7c6f64",
		Border:        "#3c3836",
		FocusedBorder: "#fabd2f",
		Accent:        "#fe8019",
		AccentFg:      "#1d2021",
		Accent2:       "#83a598",
		MethodGet:     "#b8bb26", MethodPost: "#fe8019", MethodPut: "#d3869b", MethodPatch: "#fabd2f", MethodDelete: "#fb4934",
		Success: "#b8bb26", Err: "#fb4934", Warn: "#fabd2f", Sel: "#32302f", Secret: "#7c6f64",
		Status2xx: "#b8bb26", Status3xx: "#fabd2f", Status4xx: "#fe8019", Status5xx: "#fb4934",
	},
	{
		Name:          "Matrix",
		BG:            "#03110a",
		Panel:         "#04160d",
		PanelAlt:      "#07200f",
		Fg:            "#7dffa1",
		FgDim:         "#2f7a4d",
		Border:        "#12482a",
		FocusedBorder: "#39ff88",
		Accent:        "#c8ff5a",
		AccentFg:      "#03110a",
		Accent2:       "#39ffc2",
		MethodGet:     "#39ff88", MethodPost: "#c8ff5a", MethodPut: "#7dffcf", MethodPatch: "#c8ff5a", MethodDelete: "#ff5f6e",
		Success: "#39ff88", Err: "#ff5f6e", Warn: "#c8ff5a", Sel: "#0c3a20", Secret: "#2f7a4d",
		Status2xx: "#39ff88", Status3xx: "#c8ff5a", Status4xx: "#c8ff5a", Status5xx: "#ff5f6e",
	},
	{
		Name:          "Paper",
		BG:            "#f4f1e8",
		Panel:         "#faf8f1",
		PanelAlt:      "#efeadc",
		Fg:            "#3a372f",
		FgDim:         "#94907f",
		Border:        "#d3ccb8",
		FocusedBorder: "#b5651d",
		Accent:        "#b5651d",
		AccentFg:      "#f4f1e8",
		Accent2:       "#3a6ea5",
		MethodGet:     "#4a7c59", MethodPost: "#b5651d", MethodPut: "#7a6bb5", MethodPatch: "#b58a1d", MethodDelete: "#b5433d",
		Success: "#4a7c59", Err: "#b5433d", Warn: "#a5761d", Sel: "#e6dcc0", Secret: "#a89f88",
		Status2xx: "#4a7c59", Status3xx: "#a5761d", Status4xx: "#b5651d", Status5xx: "#b5433d",
	},
}

var currentThemeIndex int

// modalWidth is how wide the confirm/prompt dialogs render, centered over
// the whole terminal - a real dialog box rather than another line stacked
// into the normal document flow.
const modalWidth = 50

var (
	methodStyle        lipgloss.Style
	methodStyleActive  lipgloss.Style
	labelStyle         lipgloss.Style
	statusStyle        lipgloss.Style
	errStyle           lipgloss.Style
	borderStyle        lipgloss.Style
	focusedBorder      lipgloss.Style
	disabledRowStyle   lipgloss.Style
	warnStyle          lipgloss.Style
	modalStyle         lipgloss.Style
	tabBarStyle        lipgloss.Style
	activeTabStyle     lipgloss.Style
	activeTabPillStyle lipgloss.Style // filled active-tab pill (DevTools tab bar)
	tabDividerStyle    lipgloss.Style // dim "|" tab dividers and the rule under them
	selStyle           lipgloss.Style
	secretStyle        lipgloss.Style
	accentStyle        lipgloss.Style
	accent2Style       lipgloss.Style
	activeTheme        Theme
)

func init() {
	applyTheme(themes[0])
}

// applyTheme rebuilds every package-level style from t's colors.
func applyTheme(t Theme) {
	activeTheme = t
	methodStyle = lipgloss.NewStyle().Bold(true).Padding(0, 1).Foreground(lipgloss.Color(t.Accent))
	methodStyleActive = methodStyle.Copy().Background(lipgloss.Color(t.Accent)).Foreground(lipgloss.Color(t.AccentFg))
	labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgDim))
	statusStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(t.Success))
	errStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Err))
	borderStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).BorderForeground(lipgloss.Color(t.Border))
	// Thick border instead of just a color swap: color alone is invisible
	// over terminals/SSH sessions without truecolor support, so a focused
	// panel needs a shape change too or there's no way to tell which panel
	// a keypress will land in.
	focusedBorder = lipgloss.NewStyle().Border(lipgloss.ThickBorder()).Padding(0, 1).BorderForeground(lipgloss.Color(t.FocusedBorder))
	disabledRowStyle = lipgloss.NewStyle().Faint(true).Strikethrough(true)
	warnStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(t.Warn))
	// BorderForeground matches borderStyle's dim t.Border, not the bright
	// t.FocusedBorder a focused grid panel uses - confirm/prompt/kvAdd/
	// workspace used to stand out more than every other floating panel
	// (palette/history/runner/env dropdown/panel, all already on
	// borderStyle) for no functional reason; a border here is still needed
	// (see overlay.go), just not a bright one.
	modalStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.Border)).
		Padding(1, 3)
	tabBarStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.FgDim))
	activeTabStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(t.Accent))
	activeTabPillStyle = lipgloss.NewStyle().Bold(true).Background(lipgloss.Color(t.Accent)).Foreground(lipgloss.Color(t.AccentFg))
	tabDividerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Border))
	selStyle = lipgloss.NewStyle().Background(lipgloss.Color(t.Sel))
	secretStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Secret))
	accentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Accent))
	accent2Style = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Accent2))
}

// focusMarker is the flat-redesign's replacement for border-shape focus
// signals on anything that dropped its border (method/env boxes, panel
// headings): a leading ">" when focused, matching the reference
// screenshot's own row-selection marker, vs 2 blank columns otherwise -
// same width either way so callers don't need a focused/unfocused case in
// their own width budget, and a real character difference (not just
// color) so it still shows on a non-truecolor terminal and survives
// stripANSI in tests.
func focusMarker(focused bool) string {
	if focused {
		return "> "
	}
	return "  "
}

// nextThemeIndex advances index by one, wrapping around at count.
func nextThemeIndex(index, count int) int {
	return (index + 1) % count
}

// cycleTheme switches to the next theme, wrapping around, and returns its
// name for the status line.
func cycleTheme() string {
	currentThemeIndex = nextThemeIndex(currentThemeIndex, len(themes))
	t := themes[currentThemeIndex]
	applyTheme(t)
	return t.Name
}

// methodColor returns the theme color for an HTTP method's badge - HEAD and
// OPTIONS fall back to the dim foreground, matching the design reference's
// ".m.HEAD,.m.OPTIONS{color:var(--fg-dim)}".
func methodColor(method string) string {
	switch method {
	case "GET":
		return activeTheme.MethodGet
	case "POST":
		return activeTheme.MethodPost
	case "PUT":
		return activeTheme.MethodPut
	case "PATCH":
		return activeTheme.MethodPatch
	case "DELETE":
		return activeTheme.MethodDelete
	default:
		return activeTheme.FgDim
	}
}

// methodBadgeStyle renders method as a bold, color-coded badge (GET green,
// POST/PATCH warm, PUT/DELETE distinct) instead of one flat accent color.
func methodBadgeStyle(method string) lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(methodColor(method)))
}

// newThemedDelegate builds a bubbles/list item delegate with the selected
// row highlighted using the current theme's Sel color, instead of the
// library default - every list in the app (sidebar, command palette) shares
// this so selection always reads the same way. showDescription is false for
// the sidebar (its items have nothing useful in Description - see
// sidebarItem.Description - so a second blank line per row would just waste
// space) and true for the command palette (category + keybind hint).
func newThemedDelegate(showDescription bool) list.DefaultDelegate {
	d := list.NewDefaultDelegate()
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.Copy().Background(lipgloss.Color(activeTheme.Sel))
	d.Styles.SelectedDesc = d.Styles.SelectedDesc.Copy().Background(lipgloss.Color(activeTheme.Sel))
	d.ShowDescription = showDescription
	if !showDescription {
		// The default delegate still inserts a blank spacer row between
		// single-line items even with the description off - wastes half the
		// sidebar's vertical space and throws off mouse row hit-testing
		// (see mouse.go's sidebar.ItemIndexAt, which assumes 1 row/item).
		d.SetSpacing(0)
	}
	return d
}

// statusClassStyle color-codes an HTTP status code by its class (2xx/3xx/4xx/5xx).
func statusClassStyle(code int) lipgloss.Style {
	switch {
	case code >= 200 && code < 300:
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(activeTheme.Status2xx))
	case code >= 300 && code < 400:
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(activeTheme.Status3xx))
	case code >= 400 && code < 500:
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(activeTheme.Status4xx))
	case code >= 500:
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(activeTheme.Status5xx))
	default:
		return lipgloss.NewStyle().Bold(true)
	}
}
