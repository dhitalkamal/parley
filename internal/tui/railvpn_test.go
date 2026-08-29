package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestVPNConnected covers the pure match: an expected VPN counts as connected
// only when it's one of the currently-detected tunnels; an empty expectation
// (the "any" default) is never a match to warn about.
func TestVPNConnected(t *testing.T) {
	cases := []struct {
		expected string
		detected []string
		want     bool
	}{
		{"backend-kamal", []string{"tailscale0", "backend-kamal"}, true},
		{"backend-kamal", []string{"tailscale0"}, false},
		{"backend-kamal", nil, false},
		{"", []string{"tailscale0"}, false}, // no expectation set
		{"", nil, false},
	}
	for _, c := range cases {
		if got := vpnConnected(c.expected, c.detected); got != c.want {
			t.Errorf("vpnConnected(%q, %v) = %v, want %v", c.expected, c.detected, got, c.want)
		}
	}
}

// TestRail_PickExpectedVPNCyclesAndPersists walks the pick flow: with two
// tunnels open, 'p' on the selector cycles the active env's expected VPN through
// (any) -> first -> second -> (any), and each choice is written straight to the
// store - carrying the env's variables through untouched.
func TestRail_PickExpectedVPNCyclesAndPersists(t *testing.T) {
	project := t.TempDir()
	m := New(t.TempDir(), project)
	if err := m.envStore.SaveEnvironment(environmentWithVar("dev", "baseUrl", "https://x")); err != nil {
		t.Fatal(err)
	}
	m.envNames, _ = m.envStore.ListEnvironments()
	m.setActiveEnvByName("dev")
	m.detectedVPNs = []string{"tailscale0", "backend-kamal"}
	m.width, m.height = 160, 40
	m.focus = focusRail
	m.updateFocus()
	m.railSection = railVPNSelector // move to the Expects/VPN selector

	step := func(want string) {
		t.Helper()
		var handled bool
		m, _, handled = m.handleRailKey(tea.KeyMsg{Type: tea.KeyRight})
		if !handled {
			t.Fatal("VPN selector should handle right-arrow")
		}
		if m.activeEnv.ExpectedVPN != want {
			t.Fatalf("in-memory ExpectedVPN = %q, want %q", m.activeEnv.ExpectedVPN, want)
		}
		loaded, err := m.envStore.LoadEnvironment("dev")
		if err != nil {
			t.Fatal(err)
		}
		if loaded.ExpectedVPN != want {
			t.Errorf("persisted ExpectedVPN = %q, want %q", loaded.ExpectedVPN, want)
		}
		if len(loaded.Variables) != 1 || loaded.Variables[0].Key != "baseUrl" {
			t.Errorf("pick clobbered variables, got %+v", loaded.Variables)
		}
	}

	step("tailscale0")
	step("backend-kamal")
	step("") // wraps back to (any)
}

// TestRail_VPNSelectorReachableAndCycles guards the headline: the expected VPN
// is its own navigable selector row (shift+down from Active reaches it), and
// left/right cycle it through (none) and each open tunnel - just like choosing
// the active environment.
func TestRail_VPNSelectorReachableAndCycles(t *testing.T) {
	project := t.TempDir()
	m := New(t.TempDir(), project)
	if err := m.envStore.SaveEnvironment(environmentWithVar("dev", "baseUrl", "https://x")); err != nil {
		t.Fatal(err)
	}
	m.envNames, _ = m.envStore.ListEnvironments()
	m.setActiveEnvByName("dev")
	m.detectedVPNs = []string{"tailscale0"}
	m.width, m.height = 160, 40
	m.focus = focusRail
	m.updateFocus() // starts on the Active selector

	// shift+down moves onto the VPN selector.
	m, _, _ = m.handleRailKey(tea.KeyMsg{Type: tea.KeyShiftDown})
	if m.railSection != railVPNSelector {
		t.Fatalf("shift+down from Active should reach the VPN selector, got %v", m.railSection)
	}
	// right cycles (none) -> tailscale0 -> (none); left goes back.
	m, _, _ = m.handleRailKey(tea.KeyMsg{Type: tea.KeyRight})
	if m.activeEnv.ExpectedVPN != "tailscale0" {
		t.Fatalf("right should pick tailscale0, got %q", m.activeEnv.ExpectedVPN)
	}
	m, _, _ = m.handleRailKey(tea.KeyMsg{Type: tea.KeyRight})
	if m.activeEnv.ExpectedVPN != "" {
		t.Fatalf("right again should wrap to (none), got %q", m.activeEnv.ExpectedVPN)
	}
	m, _, _ = m.handleRailKey(tea.KeyMsg{Type: tea.KeyLeft})
	if m.activeEnv.ExpectedVPN != "tailscale0" {
		t.Fatalf("left should go back to tailscale0, got %q", m.activeEnv.ExpectedVPN)
	}
}

// TestRail_XClearsExpectedVPN guards the one-press "uncheck": x on the selector
// sets the expectation back to (any) and persists it.
func TestRail_XClearsExpectedVPN(t *testing.T) {
	project := t.TempDir()
	m := New(t.TempDir(), project)
	env := environmentWithVar("dev", "baseUrl", "https://x")
	env.ExpectedVPN = "tailscale0"
	if err := m.envStore.SaveEnvironment(env); err != nil {
		t.Fatal(err)
	}
	m.envNames, _ = m.envStore.ListEnvironments()
	m.setActiveEnvByName("dev")
	m.width, m.height = 160, 40
	m.focus = focusRail
	m.updateFocus()
	m.railSection = railVPNSelector // the Expects/VPN selector

	m, _, handled := m.handleRailKey(runeKey('x'))
	if !handled {
		t.Fatal("VPN selector should handle 'x'")
	}
	if m.activeEnv.ExpectedVPN != "" {
		t.Errorf("x should clear the expected VPN, got %q", m.activeEnv.ExpectedVPN)
	}
	loaded, _ := m.envStore.LoadEnvironment("dev")
	if loaded.ExpectedVPN != "" {
		t.Errorf("cleared expectation should persist, got %q", loaded.ExpectedVPN)
	}
}

// TestRail_CanClearWhenTunnelIsDown guards the edge case: an expected VPN whose
// tunnel isn't currently up stays in the cycle ring, so cycling still returns to
// (any). Without this, a set-but-disconnected VPN couldn't be unchecked.
func TestRail_CanClearWhenTunnelIsDown(t *testing.T) {
	project := t.TempDir()
	m := New(t.TempDir(), project)
	env := environmentWithVar("dev", "baseUrl", "https://x")
	env.ExpectedVPN = "backend-kamal" // set...
	if err := m.envStore.SaveEnvironment(env); err != nil {
		t.Fatal(err)
	}
	m.envNames, _ = m.envStore.ListEnvironments()
	m.setActiveEnvByName("dev")
	m.detectedVPNs = []string{"tailscale0"} // ...but backend-kamal is down now

	opts := m.expectedVPNOptions()
	found := false
	for _, o := range opts {
		if o == "backend-kamal" {
			found = true
		}
	}
	if !found {
		t.Fatalf("current expectation should stay in the ring even when down, got %v", opts)
	}
	// cycling the full ring returns to the start (any reachable).
	seenAny := false
	for i := 0; i < len(opts); i++ {
		m.pickExpectedVPN(1)
		if m.activeEnv.ExpectedVPN == "" {
			seenAny = true
		}
	}
	if !seenAny {
		t.Error("cycling should pass through (any) so the expectation can be cleared")
	}
}

// TestRail_PickExpectedVPNNoopWithoutActiveEnv guards that the picker does
// nothing when no named environment is active - there's nowhere to store the
// expectation.
func TestRail_PickExpectedVPNNoopWithoutActiveEnv(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.detectedVPNs = []string{"tailscale0"}
	m.width, m.height = 160, 40
	m.focus = focusRail
	m.updateFocus()

	m, _, _ = m.handleRailKey(runeKey('p'))
	if m.activeEnv.ExpectedVPN != "" {
		t.Errorf("pick with no active env should be a no-op, got %q", m.activeEnv.ExpectedVPN)
	}
}

// TestRail_EditingVariablePreservesExpectedVPN is the drop-on-save guard: the
// expected VPN lives on the same env record as the variables, so editing a
// variable through the rail must not wipe it.
func TestRail_EditingVariablePreservesExpectedVPN(t *testing.T) {
	project := t.TempDir()
	m := New(t.TempDir(), project)
	env := environmentWithVar("dev", "k", "v")
	env.ExpectedVPN = "backend-kamal"
	if err := m.envStore.SaveEnvironment(env); err != nil {
		t.Fatal(err)
	}
	m.envNames, _ = m.envStore.ListEnvironments()
	m.setActiveEnvByName("dev")
	m.width, m.height = 160, 40
	m.focus = focusRail
	m.updateFocus()
	m.railSection = railEnvVars // cursor on the env's variable

	// edit the key: enter, type, enter to commit + save.
	m, _, _ = m.handleRailKey(tea.KeyMsg{Type: tea.KeyEnter})
	m, _, _ = m.handleRailKey(runeKey('X'))
	m, _, _ = m.handleRailKey(tea.KeyMsg{Type: tea.KeyEnter})

	loaded, err := m.envStore.LoadEnvironment("dev")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ExpectedVPN != "backend-kamal" {
		t.Errorf("editing a variable dropped ExpectedVPN, got %q", loaded.ExpectedVPN)
	}
	if len(loaded.Variables) != 1 || loaded.Variables[0].Key != "kX" {
		t.Errorf("variable edit not saved, got %+v", loaded.Variables)
	}
}
