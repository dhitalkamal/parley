package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// clockTickMsg drives a self-rearming tea.Tick, same pattern any bubbletea
// clock/spinner uses. The visible top-bar clock was removed, but the tick
// still runs: root.go re-arms VPN detection off it (see vpnpoll.go).
type clockTickMsg time.Time

func clockTickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return clockTickMsg(t) })
}
