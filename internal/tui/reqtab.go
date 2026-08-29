package tui

import "fmt"

// reqTab picks which single view is showing in the Request panel - the
// design reference's tab bar (Params/Headers/Body/Auth/Scripts/Settings)
// replaces the old always-stacked layout, so only one of these renders at a
// time. Pre-request and Test used to be two separate tabs both pointing at
// the same scriptEditor widget - merged into one Scripts tab, since that
// widget already toggles which of the two it's showing via ctrl+b (see
// scripteditor.go) and having two tabs for one widget's own internal state
// was redundant.
type reqTab int

const (
	reqTabParams reqTab = iota
	reqTabHeaders
	reqTabBody
	reqTabAuth
	reqTabScripts
	reqTabSettings
	reqTabCount
)

var reqTabLabels = [...]string{"Params", "Headers", "Body", "Auth", "Scripts", "Settings"}

func (t reqTab) String() string { return reqTabLabels[t] }

// nextReqTab shifts by delta, wrapping around - same small-fixed-set cycling
// as the method selector and body/script type toggles elsewhere.
func nextReqTab(t reqTab, delta int) reqTab {
	n := (int(t) + delta) % int(reqTabCount)
	if n < 0 {
		n += int(reqTabCount)
	}
	return reqTab(n)
}

// reqTabBarText renders the tab bar: the active tab accent-colored with an
// underline beneath it (see tabbar.go), Params/Headers annotated with their
// row counts. width is the panel's own content width - see
// tabBarWithUnderline's doc comment for why this can't be left unclipped.
func reqTabBarText(active reqTab, paramsCount, headersCount int, hasBody bool, width int) string {
	labels := make([]string, reqTabCount)
	labels[reqTabParams] = fmt.Sprintf("Params %d", paramsCount)
	labels[reqTabHeaders] = fmt.Sprintf("Headers %d", headersCount)
	bodyLabel := "Body"
	if hasBody {
		bodyLabel = "Body *"
	}
	labels[reqTabBody] = bodyLabel
	labels[reqTabAuth] = "Auth"
	labels[reqTabScripts] = "Scripts"
	labels[reqTabSettings] = "Settings"

	return tabBarWithUnderline(labels, int(active), width)
}
