package tui

import (
	"fmt"
	"sort"
	"strings"

	environment "github.com/dhitalkamal/parley/internal/environment/domain"
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
)

func (m Model) envLabel() string {
	name := m.activeEnvName
	if name == "" {
		name = "none"
	}
	return labelStyle.Render("Env: ") + statusStyle.Render(name)
}

// resolvedVars merges globals and the active environment per the
// env-overrides-globals precedence in execution.ResolveVariables.
func (m Model) resolvedVars() map[string]environment.Variable {
	return execution.ResolveVariables(m.globals, m.activeEnv)
}

// unresolvedWarning reports any {{var}} in the current (unsubstituted)
// editor state that isn't defined in globals/the active environment, so it
// can be surfaced before the user sends.
func (m Model) unresolvedWarning() string {
	_, unresolved := execution.SubstituteRequest(m.buildRequest(), m.resolvedVars())
	if len(unresolved) == 0 {
		return ""
	}
	names := make([]string, len(unresolved))
	for i, u := range unresolved {
		names[i] = "{{" + u + "}}"
	}
	return warnStyle.Render("Unresolved: " + strings.Join(names, ", "))
}

// variableFrom reports which scope key came from - the active environment
// if it defines an enabled variable by that name, Globals otherwise (env
// overrides globals for the same key, per execution.ResolveVariables).
func (m Model) variableFrom(key string) string {
	if m.activeEnvName != "" {
		for _, v := range m.activeEnv.Variables {
			if v.Key == key && v.Enabled {
				return m.activeEnvName
			}
		}
	}
	return "Globals"
}

// variablesPanelView renders the merged, read-only variable list with
// secrets masked and each row's source scope (Globals vs. the active
// environment) - a quick-glance view of what's actually in scope while
// building a request, distinct from the editable per-scope table in the
// ctrl+e environments panel (envpanel.go). Anything the current request
// references via {{var}} but that resolves nowhere is listed too, in error
// color, instead of only surfacing in the separate unresolvedWarning banner.
func (m Model) variablesPanelView() string {
	header := activeTabStyle.Render("Merged Variables") + labelStyle.Render("  read-only, secrets masked  (esc close)")
	vars := m.resolvedVars()
	_, unresolved := execution.SubstituteRequest(m.buildRequest(), vars)

	if len(vars) == 0 && len(unresolved) == 0 {
		return borderStyle.Render(header + "\n" + labelStyle.Render("No variables defined"))
	}

	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	fmt.Fprintf(&b, "%-20s %-24s %s\n", "VARIABLE", "VALUE", "FROM")
	for _, k := range keys {
		v := vars[k]
		valueText := v.Value
		if v.Secret {
			valueText = "****"
		}
		// Pad the plain value first, then colorize - padding text that
		// already carries ANSI escape bytes counts those bytes as visible
		// width and throws column alignment off.
		valueCell := fmt.Sprintf("%-24s", valueText)
		if v.Secret {
			valueCell = secretStyle.Render(valueCell)
		}
		fmt.Fprintf(&b, "%-20s %s %s\n", k, valueCell, m.variableFrom(k))
	}
	for _, k := range unresolved {
		valueCell := errStyle.Render(fmt.Sprintf("%-24s", "unset"))
		fmt.Fprintf(&b, "%-20s %s %s\n", k, valueCell, "-")
	}
	return borderStyle.Render(header + "\n" + strings.TrimRight(b.String(), "\n"))
}
