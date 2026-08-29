package tui

import (
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

type scriptKind int

const (
	scriptPreRequest scriptKind = iota
	scriptTest
)

// scriptEditor holds the pre-request and test script textareas. Only one is
// visible/focused at a time; ctrl+b cycles between them, mirroring how
// bodyEditor cycles its own body type - same key, but zone-scoped, so there's
// no collision (only one of the two editors is ever focused at once).
type scriptEditor struct {
	kind     scriptKind
	preArea  textarea.Model
	testArea textarea.Model
	focused  bool
}

func newScriptEditor() scriptEditor {
	pre := textarea.New()
	pre.Placeholder = `pm.environment.set("token", "value");`
	pre.ShowLineNumbers = true
	// drop the default "|" prompt gutter - it rendered as a stray bright line
	// down the left of the editor (see newBodyEditor for the same fix).
	pre.Prompt = ""

	test := textarea.New()
	test.Placeholder = "pm.test(\"status is 200\", function () {\n  pm.expect(pm.response.code).to.equal(200);\n});"
	test.ShowLineNumbers = true
	test.Prompt = ""

	return scriptEditor{preArea: pre, testArea: test}
}

func (s scriptEditor) PreRequest() string { return s.preArea.Value() }
func (s scriptEditor) Test() string       { return s.testArea.Value() }

func (s *scriptEditor) SetScripts(pre, test string) {
	s.preArea.SetValue(pre)
	s.testArea.SetValue(test)
}

func (s *scriptEditor) SetFocus(focused bool) {
	s.focused = focused
	if !focused {
		s.preArea.Blur()
		s.testArea.Blur()
		return
	}
	if s.kind == scriptTest {
		s.testArea.Focus()
	} else {
		s.preArea.Focus()
	}
}

func (s *scriptEditor) SetSize(w, h int) {
	s.preArea.SetWidth(w)
	s.preArea.SetHeight(h)
	s.testArea.SetWidth(w)
	s.testArea.SetHeight(h)
}

func (s scriptEditor) Update(msg tea.Msg) (scriptEditor, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok && k.String() == "ctrl+b" {
		if s.kind == scriptPreRequest {
			s.kind = scriptTest
			s.preArea.Blur()
			s.testArea.Focus()
		} else {
			s.kind = scriptPreRequest
			s.testArea.Blur()
			s.preArea.Focus()
		}
		return s, nil
	}
	var cmd tea.Cmd
	if s.kind == scriptTest {
		s.testArea, cmd = s.testArea.Update(msg)
	} else {
		s.preArea, cmd = s.preArea.Update(msg)
	}
	return s, cmd
}

func (s scriptEditor) View() string {
	area := s.preArea
	if s.kind == scriptTest {
		area = s.testArea
	}
	// render the textarea flush (no nested border) - focus is shown by the
	// Request panel's own border, matching the Body Raw editor and the
	// params/headers tables (see reqpanel.go).
	return s.tabLine() + "\n" + area.View()
}

// tabLine is the Pre-request/Test sub-tab bar shown above the editor. It
// replaces the old "Scripts: pre-request (ctrl+b: ...)" text label: the raw
// shortcut is gone (shortcuts don't belong in the request tabs), and both
// scripts are shown as tabs - active one filled - so it's self-evident they're
// switchable. Same pill/divider styling as the DevTools tab bars (tabbar.go);
// ctrl+b still toggles which is active (see Update).
func (s scriptEditor) tabLine() string {
	pre := " Pre-request "
	test := " Test "
	preStyle, testStyle := tabBarStyle, tabBarStyle
	if s.kind == scriptTest {
		testStyle = activeTabPillStyle
	} else {
		preStyle = activeTabPillStyle
	}
	return preStyle.Render(pre) + tabDividerStyle.Render(glyphVerticalLine) + testStyle.Render(test)
}
