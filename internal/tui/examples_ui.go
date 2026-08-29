package tui

import tea "github.com/charmbracelet/bubbletea"

// startSaveExample opens the save-as-example prompt, provided there's a
// saved request to attach the example to and a response to save. Browsing
// previously saved examples back into the response viewer is left for a
// follow-up - this covers the "save a response as a named example" bullet
// from the brief; examples otherwise live as plain JSON files next to their
// request, readable by hand like the rest of the collection.
func (m Model) startSaveExample() (Model, tea.Cmd) {
	if m.loadedRequestPath == "" {
		m.status = "Save or load a request before saving an example"
		return m, nil
	}
	if m.response.statusCode == 0 {
		m.status = "Send a request before saving an example"
		return m, nil
	}
	m.prompt.Open(promptSaveExample, m.loadedRequestPath, "Example name", "")
	return m, nil
}
