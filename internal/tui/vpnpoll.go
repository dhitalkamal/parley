package tui

import (
	"time"

	"github.com/dhitalkamal/parley/internal/netcheck"

	tea "github.com/charmbracelet/bubbletea"
)

// VPN detection used to run synchronously inside the clockTickMsg handler, once
// a second, right on bubbletea's single Update goroutine. netcheck.DetectVPNs
// shells down to net.Interfaces (a syscall that enumerates every network
// interface) - usually fast, but on macOS across a sleep/wake or a VPN coming
// up/down it can stall, and any stall there froze the whole UI ("the app stops
// working if I stay away for a while"). It now runs off-loop as a tea.Cmd and
// paces itself: the next poll is only scheduled after the previous one returns,
// so a slow probe delays the next check instead of piling up goroutines or
// blocking input.

// vpnPollInterval is how long to wait after one detection finishes before
// starting the next. Interfaces change rarely, so a second was always overkill.
const vpnPollInterval = 3 * time.Second

// vpnDetectedMsg carries the result of an off-loop netcheck.DetectVPNs run.
type vpnDetectedMsg []string

// vpnPollMsg fires vpnPollInterval after the last detection completed - its
// handler kicks off the next detectVPNsCmd.
type vpnPollMsg struct{}

// detectVPNsCmd runs netcheck.DetectVPNs in bubbletea's own command goroutine
// (a tea.Cmd is invoked off the Update loop), so a slow or hung syscall can
// never block key/mouse handling.
func detectVPNsCmd() tea.Cmd {
	return func() tea.Msg {
		return vpnDetectedMsg(netcheck.DetectVPNs())
	}
}

// scheduleVPNPollCmd waits vpnPollInterval then emits vpnPollMsg - the pacing
// half of the loop (detect -> store -> wait -> detect ...).
func scheduleVPNPollCmd() tea.Cmd {
	return tea.Tick(vpnPollInterval, func(time.Time) tea.Msg { return vpnPollMsg{} })
}
