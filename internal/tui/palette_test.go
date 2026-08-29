package tui

import (
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
	"testing"
)

// findPaletteCommand is a small test helper - fails the test immediately if
// name isn't registered, so a typo/rename in paletteCommands() surfaces as
// a clear failure rather than a silent no-op t.Run.
func findPaletteCommand(t *testing.T, name string) paletteCommand {
	t.Helper()
	for _, c := range paletteCommands() {
		if c.name == name {
			return c
		}
	}
	t.Fatalf("no palette command named %q", name)
	return paletteCommand{}
}

// TestPaletteCommands_ToggleCodeSnippetHasCorrectKeyHint guards a real
// mismatch found during the overlay polish pass: this entry's keyHint said
// ctrl+u (RevealSecrets' actual binding), not ctrl+n (CodeSnippet's) - a
// wrong "forgot the shortcut" reminder is worse than none at all.
func TestPaletteCommands_ToggleCodeSnippetHasCorrectKeyHint(t *testing.T) {
	c := findPaletteCommand(t, "Toggle code snippet")
	if c.keyHint != "ctrl+n" {
		t.Errorf("keyHint = %q, want %q (CodeSnippet's real binding)", c.keyHint, keys.CodeSnippet.Help().Key)
	}
}

// TestPaletteCommands_RevealSecretsToggles guards a genuine discoverability
// gap: ctrl+u had no palette entry at all before this pass.
func TestPaletteCommands_RevealSecretsToggles(t *testing.T) {
	c := findPaletteCommand(t, "Reveal secrets")
	m := New(t.TempDir(), t.TempDir())

	got, _ := c.run(m)
	if !got.revealSecrets {
		t.Error("expected running the command to set revealSecrets")
	}
}

// TestPaletteCommands_MaximizeResponseTogglesWhenResponseExists guards the
// other new discoverability gap: ctrl+z (added this session's redesign) had
// no palette entry either.
func TestPaletteCommands_MaximizeResponseTogglesWhenResponseExists(t *testing.T) {
	c := findPaletteCommand(t, "Maximize response")
	m := New(t.TempDir(), t.TempDir())
	m.response.SetResponse(execution.Response{StatusCode: 200, Status: "200 OK"}, 10)

	got, _ := c.run(m)
	if !got.responseMaximized {
		t.Error("expected running the command to set responseMaximized")
	}
}

// TestPaletteCommands_MaximizeResponseIsANoOpWithoutAResponse guards the
// same guard the direct ctrl+z binding has - nothing to maximize yet.
func TestPaletteCommands_MaximizeResponseIsANoOpWithoutAResponse(t *testing.T) {
	c := findPaletteCommand(t, "Maximize response")
	m := New(t.TempDir(), t.TempDir())

	got, _ := c.run(m)
	if got.responseMaximized {
		t.Error("expected the command to do nothing before a response has arrived")
	}
}

func TestPaletteCommands_NonEmpty(t *testing.T) {
	cmds := paletteCommands()
	if len(cmds) == 0 {
		t.Fatal("expected at least one command")
	}
}

func TestPaletteCommands_NoDuplicateNames(t *testing.T) {
	cmds := paletteCommands()
	seen := map[string]bool{}
	for _, c := range cmds {
		if c.name == "" {
			t.Error("command with empty name")
		}
		if seen[c.name] {
			t.Errorf("duplicate command name %q", c.name)
		}
		seen[c.name] = true
	}
}

func TestPaletteCommands_EveryCommandHasARunFunc(t *testing.T) {
	for _, c := range paletteCommands() {
		if c.run == nil {
			t.Errorf("command %q has a nil run func", c.name)
		}
	}
}

func TestPaletteCommands_EveryCommandHasACategory(t *testing.T) {
	for _, c := range paletteCommands() {
		if c.category == "" {
			t.Errorf("command %q has no category", c.name)
		}
	}
}
