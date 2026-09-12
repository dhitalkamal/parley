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
