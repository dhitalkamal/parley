package tui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestNextThemeIndex_AdvancesByOne(t *testing.T) {
	got := nextThemeIndex(0, 2)
	if got != 1 {
		t.Errorf("got %d, want 1", got)
	}
}

func TestNextThemeIndex_WrapsAroundAtEnd(t *testing.T) {
	got := nextThemeIndex(1, 2)
	if got != 0 {
		t.Errorf("got %d, want 0", got)
	}
}

func TestThemes_HasAtLeastTwoNamedThemes(t *testing.T) {
	if len(themes) < 2 {
		t.Fatalf("got %d themes, want at least 2", len(themes))
	}
	seen := map[string]bool{}
	for _, th := range themes {
		if th.Name == "" {
			t.Error("theme with empty name")
		}
		if seen[th.Name] {
			t.Errorf("duplicate theme name %q", th.Name)
		}
		seen[th.Name] = true
	}
}

func TestThemes_MatchesTheFourNamedDesigns(t *testing.T) {
	want := []string{"Ayu Midnight", "Gruvbox", "Matrix", "Paper"}
	if len(themes) != len(want) {
		t.Fatalf("got %d themes, want %d: %v", len(themes), len(want), want)
	}
	for i, name := range want {
		if themes[i].Name != name {
			t.Errorf("theme %d: got %q, want %q", i, themes[i].Name, name)
		}
	}
}

func TestThemes_EveryFieldIsSet(t *testing.T) {
	for _, th := range themes {
		fields := map[string]string{
			"BG": th.BG, "Panel": th.Panel, "PanelAlt": th.PanelAlt,
			"Fg": th.Fg, "FgDim": th.FgDim, "Border": th.Border,
			"FocusedBorder": th.FocusedBorder, "Accent": th.Accent, "Accent2": th.Accent2,
			"AccentFg":  th.AccentFg,
			"MethodGet": th.MethodGet, "MethodPost": th.MethodPost, "MethodPut": th.MethodPut,
			"MethodPatch": th.MethodPatch, "MethodDelete": th.MethodDelete,
			"Success": th.Success, "Err": th.Err, "Warn": th.Warn,
			"Sel": th.Sel, "Secret": th.Secret,
			"Status2xx": th.Status2xx, "Status3xx": th.Status3xx, "Status4xx": th.Status4xx, "Status5xx": th.Status5xx,
		}
		for name, v := range fields {
			if v == "" {
				t.Errorf("theme %q: field %s is empty", th.Name, name)
			}
		}
	}
}

func TestDefaultTheme_IsAyuMidnight(t *testing.T) {
	if themes[0].Name != "Ayu Midnight" {
		t.Errorf("got default theme %q, want \"Ayu Midnight\"", themes[0].Name)
	}
}

// TestModalStyle_UsesTheSameDimBorderColorAsEveryOtherFloatingPanel guards
// the flat redesign's border-weight reduction for confirm/prompt/kvAdd/
// workspace: they used to stand out with the same bright FocusedBorder
// color a focused panel uses, while every other floating panel (palette,
// history, runner, env dropdown/panel, all on borderStyle already) used
// the dim t.Border color - two different "floating over the background"
// looks for no functional reason. They keep a border at all (unlike the
// grid panels) because overlay.go's whole reason for existing is that a
// floating dialog needs a visible edge over the still-rendered background.
func TestModalStyle_UsesTheSameDimBorderColorAsEveryOtherFloatingPanel(t *testing.T) {
	applyTheme(themes[0])
	got := modalStyle.GetBorderTopForeground()
	want := lipgloss.Color(activeTheme.Border)
	if got != want {
		t.Errorf("modalStyle border color = %v, want %v (activeTheme.Border, same as borderStyle)", got, want)
	}
}

// TestFocusMarker_DiffersByFocusAndSameWidth guards the flat-redesign's
// replacement for border-shape focus signals: elements that dropped their
// border (method/env boxes, panel headings) need some other visible,
// non-color character difference to show focus, since color alone doesn't
// survive stripANSI or a non-truecolor terminal. Both variants must render
// to the same width so callers can budget for it without a focused/
// unfocused case of their own.
func TestFocusMarker_DiffersByFocusAndSameWidth(t *testing.T) {
	unfocused, focused := focusMarker(false), focusMarker(true)
	if unfocused == focused {
		t.Errorf("focusMarker(false) == focusMarker(true) == %q, want them to differ", unfocused)
	}
	if lipgloss.Width(unfocused) != lipgloss.Width(focused) {
		t.Errorf("focusMarker width differs: unfocused=%d focused=%d, want equal", lipgloss.Width(unfocused), lipgloss.Width(focused))
	}
}
