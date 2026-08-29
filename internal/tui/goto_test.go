package tui

import "testing"

// pressGoto sends g then the given follow-up key through Update, returning the
// resulting model.
func pressGoto(t *testing.T, m Model, second rune) Model {
	t.Helper()
	next, _ := m.Update(rune1('g'))
	armed := next.(Model)
	if !armed.pendingG {
		t.Fatal("bare g should arm the goto prefix")
	}
	next, _ = armed.Update(rune1(second))
	return next.(Model)
}

func TestGoto_ScreenJumps(t *testing.T) {
	t.Run("gd dashboard", func(t *testing.T) {
		m := New(t.TempDir(), t.TempDir())
		m.width, m.height = 160, 44
		m.screen = ScreenRequest
		if got := pressGoto(t, m, 'd'); got.screen != ScreenDashboard {
			t.Errorf("gd -> screen %v, want Dashboard", got.screen)
		}
	})
	t.Run("gs settings", func(t *testing.T) {
		m := New(t.TempDir(), t.TempDir())
		m.width, m.height = 160, 44
		m.screen = ScreenRequest
		if got := pressGoto(t, m, 's'); got.screen != ScreenSettings {
			t.Errorf("gs -> screen %v, want Settings", got.screen)
		}
	})
	t.Run("gc collections", func(t *testing.T) {
		m := New(t.TempDir(), t.TempDir())
		m.width, m.height = 160, 44
		m.screen = ScreenRequest
		if got := pressGoto(t, m, 'c'); got.screen != ScreenCollections {
			t.Errorf("gc -> screen %v, want Collections", got.screen)
		}
	})
	t.Run("gh history", func(t *testing.T) {
		m := New(t.TempDir(), t.TempDir())
		m.width, m.height = 160, 44
		m.screen = ScreenRequest
		if got := pressGoto(t, m, 'h'); !got.history.active {
			t.Error("gh should open the history overlay")
		}
	})
	t.Run("gw workspaces routes", func(t *testing.T) {
		// The switcher itself needs a workspace store to actually open; here we
		// just guard that gw is wired to the goto handler (consumes the prefix).
		m := New(t.TempDir(), t.TempDir())
		m.width, m.height = 160, 44
		m.screen = ScreenRequest
		if got := pressGoto(t, m, 'w'); got.pendingG {
			t.Error("gw should consume the goto prefix (routes to the workspace switcher)")
		}
	})
}

// TestGoto_UnknownKeyCancels guards that an unrecognized follow-up clears the
// prefix and doesn't jump anywhere.
func TestGoto_UnknownKeyCancels(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 160, 44
	m.screen = ScreenRequest
	m.focus = focusResponse
	m.updateFocus()

	next, _ := m.Update(rune1('g'))
	next, _ = next.(Model).Update(rune1('x'))
	got := next.(Model)
	if got.pendingG {
		t.Error("an unknown follow-up should clear the goto prefix")
	}
	if got.screen != ScreenRequest {
		t.Errorf("gx should not change screens, got %v", got.screen)
	}
}
