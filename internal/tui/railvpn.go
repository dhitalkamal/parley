package tui

// vpnConnected reports whether the environment's expected VPN is one of the
// tunnels currently up. An empty expectation ("any") is never a match - there's
// nothing to warn about - so callers can treat false-with-empty-expected as "no
// opinion" rather than "disconnected".
func vpnConnected(expected string, detected []string) bool {
	if expected == "" {
		return false
	}
	for _, d := range detected {
		if d == expected {
			return true
		}
	}
	return false
}

// expectedVPNOptions is the ring the picker cycles through: "(any)" first
// (represented as ""), then each currently-detected tunnel. Built fresh each
// press so a tunnel that just came up is immediately pickable. The current
// expectation is always kept in the ring even if its tunnel is down right now -
// otherwise cycling could never pass back through it to "(any)", so a
// set-but-disconnected VPN could never be cleared.
func (m Model) expectedVPNOptions() []string {
	opts := append([]string{""}, m.detectedVPNs...)
	cur := m.activeEnv.ExpectedVPN
	if cur == "" {
		return opts
	}
	for _, o := range opts {
		if o == cur {
			return opts
		}
	}
	return append(opts, cur)
}

// clearExpectedVPN sets the active environment back to "(any)" - the one-press
// "uncheck" for the expected VPN. A no-op when nothing's active or set.
func (m *Model) clearExpectedVPN() {
	if m.activeEnvName == "" || m.activeEnv.ExpectedVPN == "" {
		return
	}
	m.activeEnv.ExpectedVPN = ""
	m.saveRailEnvScope()
}

// pickExpectedVPN cycles the active environment's expected VPN one step through
// the options ring (any -> each open tunnel -> any) and persists it. Display
// only: this never touches the VPN itself, it just records which tunnel this
// environment is meant to run behind so the panel can show connected vs not. A
// no-op when no named environment is active (nowhere to store it).
func (m *Model) pickExpectedVPN(delta int) {
	if m.activeEnvName == "" {
		return
	}
	opts := m.expectedVPNOptions()
	cur := 0
	for i, o := range opts {
		if o == m.activeEnv.ExpectedVPN {
			cur = i
			break
		}
	}
	next := ((cur+delta)%len(opts) + len(opts)) % len(opts)
	m.activeEnv.ExpectedVPN = opts[next]
	m.saveRailEnvScope()
}
