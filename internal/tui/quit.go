package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// escDoubleTapWindow is how quickly a second at-rest esc must follow the
// first to count as "really means quit" rather than a stray keypress.
const escDoubleTapWindow = 600 * time.Millisecond

// escIsDoubleTap reports whether now falls within window of last - last
// being the zero time means there was no prior esc to pair with.
func escIsDoubleTap(last, now time.Time, window time.Duration) bool {
	if last.IsZero() {
		return false
	}
	return now.Sub(last) < window
}

// handleRestEsc runs when esc reaches the root with nothing open to back out
// of (no prompt/confirm/history/runner/palette, no sidebar filter) - a single
// esc there is a no-op instead of an instant, easy-to-fat-finger quit; only a
// second esc within escDoubleTapWindow opens the "really quit?" dialog.
func (m Model) handleRestEsc() (Model, tea.Cmd) {
	if escIsDoubleTap(m.lastEscAt, time.Now(), escDoubleTapWindow) {
		m.lastEscAt = time.Time{}
		m.confirm.Open(confirmQuit, "Quit parley?", "")
		return m, nil
	}
	m.lastEscAt = time.Now()
	return m, nil
}
