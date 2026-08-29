package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Field indices into requestSettingsEditor's cursor - see
// requestSettingsFieldLabels for what each one means to the user.
const (
	reqSettingsFieldTimeout = iota
	reqSettingsFieldFollowRedirects
	reqSettingsFieldInsecureSkipVerify
	reqSettingsFieldCount
)

var requestSettingsFieldLabels = [reqSettingsFieldCount]string{
	"Timeout",
	"Follow redirects",
	"Skip TLS verify",
}

// requestSettingsEditor configures a request's Timeout/FollowRedirects/
// InsecureSkipVerify - collection.Request fields that existed since before
// the TUI redesign but had no editor anywhere; every request silently sent
// with FollowRedirects hardcoded true and no way to set a timeout or skip
// TLS verification. Timeout is free text parsed as a Go duration (e.g.
// "30s") - empty or unparseable both mean "use the default", the same as
// leaving Timeout at its zero value already does in execapp.SendRequest.
type requestSettingsEditor struct {
	timeout            textinput.Model
	followRedirects    bool
	insecureSkipVerify bool
	fieldIdx           int
	focused            bool
}

func newRequestSettingsEditor() requestSettingsEditor {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = "30s"
	return requestSettingsEditor{timeout: ti, followRedirects: true}
}

// Timeout parses the timeout field as a Go duration - 0 (unparseable or
// empty) means "let execapp.SendRequest apply its own default" rather than
// blocking on invalid input.
func (e requestSettingsEditor) Timeout() time.Duration {
	d, err := time.ParseDuration(e.timeout.Value())
	if err != nil {
		return 0
	}
	return d
}

func (e requestSettingsEditor) FollowRedirects() bool    { return e.followRedirects }
func (e requestSettingsEditor) InsecureSkipVerify() bool { return e.insecureSkipVerify }

// SetSettings loads a saved request's values into the editor - timeout of 0
// renders as an empty field (matching Timeout()'s own "0 means unset"
// convention) rather than a literal "0s".
func (e *requestSettingsEditor) SetSettings(timeout time.Duration, followRedirects, insecureSkipVerify bool) {
	if timeout == 0 {
		e.timeout.SetValue("")
	} else {
		e.timeout.SetValue(timeout.String())
	}
	e.followRedirects = followRedirects
	e.insecureSkipVerify = insecureSkipVerify
}

func (e *requestSettingsEditor) SetFocus(focused bool) {
	e.focused = focused
	if focused && e.fieldIdx == reqSettingsFieldTimeout {
		e.timeout.Focus()
	} else {
		e.timeout.Blur()
	}
}

func (e *requestSettingsEditor) SetSize(w, h int) {
	e.timeout.Width = w - lipgloss.Width(longestReqSettingsLabel()) - 4
}

func longestReqSettingsLabel() string {
	longest := ""
	for _, l := range requestSettingsFieldLabels {
		if len(l) > len(longest) {
			longest = l
		}
	}
	return longest
}

// Update cycles fields with up/down (wrapping, same convention as
// authEditor), toggles the two boolean fields with space/enter, and
// forwards everything else to the timeout text field.
func (e requestSettingsEditor) Update(msg tea.Msg) (requestSettingsEditor, tea.Cmd, bool) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "down":
			e.timeout.Blur()
			e.fieldIdx = (e.fieldIdx + 1) % reqSettingsFieldCount
			e.SetFocus(true)
			return e, nil, true
		case "up":
			e.timeout.Blur()
			e.fieldIdx = (e.fieldIdx + reqSettingsFieldCount - 1) % reqSettingsFieldCount
			e.SetFocus(true)
			return e, nil, true
		case " ", "enter":
			switch e.fieldIdx {
			case reqSettingsFieldFollowRedirects:
				e.followRedirects = !e.followRedirects
				return e, nil, true
			case reqSettingsFieldInsecureSkipVerify:
				e.insecureSkipVerify = !e.insecureSkipVerify
				return e, nil, true
			}
		}
	}
	if e.fieldIdx != reqSettingsFieldTimeout {
		return e, nil, true
	}
	var cmd tea.Cmd
	e.timeout, cmd = e.timeout.Update(msg)
	return e, cmd, true
}

func checkbox(checked bool) string {
	if checked {
		return "[x]"
	}
	return "[ ]"
}

func (e requestSettingsEditor) View() string {
	labelWidth := lipgloss.Width(longestReqSettingsLabel()) + 1
	rows := make([]string, reqSettingsFieldCount)
	for i, label := range requestSettingsFieldLabels {
		marker := focusMarker(e.focused && e.fieldIdx == i)
		padded := fmt.Sprintf("%-*s", labelWidth, label)
		var value string
		switch i {
		case reqSettingsFieldTimeout:
			value = e.timeout.View()
		case reqSettingsFieldFollowRedirects:
			value = checkbox(e.followRedirects)
		case reqSettingsFieldInsecureSkipVerify:
			value = checkbox(e.insecureSkipVerify)
		}
		rows[i] = marker + labelStyle.Render(padded) + "  " + value
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}
