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

func TestEnvSegmentText_ShowsNoneWhenNoEnvironmentActive(t *testing.T) {
	got := envSegmentText("")
	if got != "o none" {
		t.Errorf("envSegmentText(\"\") = %q, want %q", got, "o none")
	}
}

func TestEnvSegmentText_ShowsTheActiveEnvironmentName(t *testing.T) {
	got := envSegmentText("staging")
	if got != "* staging" {
		t.Errorf("envSegmentText(\"staging\") = %q, want %q", got, "* staging")
	}
}
