package tui

import "testing"

func TestEnvPillColor_MatchesEnvironmentNameCaseInsensitively(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"local", activeTheme.Success},
		{"LOCAL", activeTheme.Success},
		{"staging", activeTheme.Warn},
		{"Staging", activeTheme.Warn},
		{"prod", activeTheme.Err},
		{"PROD", activeTheme.Err},
		{"production", activeTheme.Err},
		{"", activeTheme.FgDim},
		{"qa", activeTheme.FgDim},
	}
	for _, c := range cases {
		if got := envPillColor(c.name); got != c.want {
			t.Errorf("envPillColor(%q) = %q, want %q", c.name, got, c.want)
		}
	}
}
