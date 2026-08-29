package tui

import "strings"

// envPillColor is the color an environment's pill/echo renders in - LOCAL
// green, STAGING amber, PROD red, anything else (including no active
// environment) the theme's neutral dim gray. Reuses the existing Success/
// Warn/Err/FgDim theme colors rather than adding new palette entries, so
// every theme (not just Ayu Midnight) gets a sensible mapping for free.
func envPillColor(envName string) string {
	switch strings.ToLower(envName) {
	case "local":
		return activeTheme.Success
	case "staging", "stage":
		return activeTheme.Warn
	case "prod", "production":
		return activeTheme.Err
	default:
		return activeTheme.FgDim
	}
}

// envSegmentText is the top bar's environment pill segment as plain text -
// a status LED ("*" filled when an environment is active, "o" hollow when
// none is) plus the name, or "none". Kept plain (uncolored) here so mouse.go
// can find it with a simple string search, the same convention
// workspaceLabelAt/profileIconAt already rely on - color is applied
// afterward, only for display (see topBarView's colorizeEnvSegment).
func envSegmentText(envName string) string {
	label := envName
	led := "*"
	if label == "" {
		label = "none"
		led = "o"
	}
	return led + " " + label
}
