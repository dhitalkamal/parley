package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	"github.com/dhitalkamal/parley/internal/importexport"
	"github.com/dhitalkamal/parley/internal/platform/fsstore"
)

// promptPurpose selects what happens when a promptState is submitted.
type promptPurpose int

const (
	promptNewRequest promptPurpose = iota
	promptNewFolder
	promptRename
	promptSaveAs
	promptSearch
	promptSaveExample
	promptImport
	promptExport
	promptNewEnvironment
	promptNewWorkspace
	promptNewWorkspacePath
	promptDashboardSearch
)

// promptState is a single-line modal text prompt used for every sidebar
// naming action (new request, new folder, rename, save-as), distinguished by
// purpose. targetPath means the parent folder for new/save-as, or the node
// being renamed for rename.
type promptState struct {
	active     bool
	purpose    promptPurpose
	targetPath string
	label      string
	input      textinput.Model
}

func newPromptState() promptState {
	ti := textinput.New()
	ti.Prompt = "> "
	ti.Width = modalWidth - 10
	return promptState{input: ti}
}

func (p *promptState) Open(purpose promptPurpose, targetPath, label, prefill string) {
	p.active = true
	p.purpose = purpose
	p.targetPath = targetPath
	p.label = label
	p.input.SetValue(prefill)
	p.input.CursorEnd()
	p.input.Focus()
}

func (p *promptState) Close() {
	p.active = false
	p.input.Blur()
	p.input.SetValue("")
}

// View renders the prompt as a centered modal dialog box rather than another
// line stacked into the normal document flow - see Model.View in view.go for
// where this gets placed over a blank backdrop.
func (p promptState) View() string {
	if !p.active {
		return ""
	}
	content := labelStyle.Render(p.label+":") + "\n" + p.input.View() +
		"\n\n" + labelStyle.Render("enter confirm - esc cancel")
	return modalStyle.Width(modalWidth).Render(content)
}

// handlePromptKey handles input while a prompt is open. Submitting dispatches
// to the store action matching p.purpose.
func (m Model) handlePromptKey(k tea.KeyMsg) (Model, tea.Cmd) {
	switch k.String() {
	case "esc":
		m.prompt.Close()
		return m, nil
	case "enter":
		return m.submitPrompt()
	}
	var cmd tea.Cmd
	m.prompt.input, cmd = m.prompt.input.Update(k)
	return m, cmd
}

func (m Model) submitPrompt() (Model, tea.Cmd) {
	name := m.prompt.input.Value()
	purpose := m.prompt.purpose
	target := m.prompt.targetPath
	m.prompt.Close()

	switch purpose {
	case promptNewRequest:
		path, err := m.store.SaveRequest(target, name, collection.Request{Method: collection.GET, FollowRedirects: true})
		if err != nil {
			m.status = "New request failed: " + err.Error()
			return m, nil
		}
		m.refreshTree()
		m.selectSidebarPath(path)
		m.loadRequestIntoEditor(path, collection.Request{Method: collection.GET, FollowRedirects: true})
		m.status = "Created " + name
	case promptNewFolder:
		if _, err := m.store.CreateFolder(target, name); err != nil {
			m.status = "New folder failed: " + err.Error()
			return m, nil
		}
		m.refreshTree()
		m.status = "Created folder " + name
	case promptRename:
		newPath, err := m.store.Rename(target, name)
		if err != nil {
			m.status = "Rename failed: " + err.Error()
			return m, nil
		}
		if m.loadedRequestPath == target {
			m.loadedRequestPath = newPath
		}
		m.refreshTree()
		m.selectSidebarPath(newPath)
		m.status = "Renamed to " + name
	case promptSaveAs:
		path, err := m.store.SaveRequest(target, name, m.buildRequestForSave())
		if err != nil {
			m.status = "Save failed: " + err.Error()
			return m, nil
		}
		m.loadedRequestPath = path
		m.refreshTree()
		m.selectSidebarPath(path)
		m.status = "Saved " + name
	case promptSearch:
		m.response.SetSearch(name)
		if name == "" {
			m.status = "Ready"
		} else {
			m.status = "Searching for " + name
		}
	case promptSaveExample:
		ex := collection.Example{
			StatusCode: m.response.statusCode,
			Status:     m.response.status,
			Headers:    m.response.headers,
			Body:       string(m.response.rawBody),
		}
		if _, err := m.store.SaveExample(target, name, ex); err != nil {
			m.status = "Save example failed: " + err.Error()
			return m, nil
		}
		m.status = "Saved example " + name
	case promptImport:
		node, err := importexport.ParseAny(name)
		if err != nil {
			// Not recognized as literal content - try it as a file path instead,
			// so a multi-line curl command or a spec too long to paste inline
			// can be saved to a file and imported by path.
			data, readErr := os.ReadFile(name)
			if readErr != nil {
				m.status = "Import failed: " + err.Error()
				return m, nil
			}
			node, err = importexport.ParseAny(string(data))
			if err != nil {
				m.status = "Import failed: " + err.Error()
				return m, nil
			}
		}
		count, err := importexport.PersistImportedNode(m.store, target, node)
		if err != nil {
			m.status = "Import failed: " + err.Error()
			return m, nil
		}
		m.refreshTree()
		m.status = fmt.Sprintf("Imported %d request(s)", count)
	case promptExport:
		if err := m.exportToFile(target, name); err != nil {
			m.status = "Export failed: " + err.Error()
			return m, nil
		}
		m.status = "Exported to " + name
	case promptNewEnvironment:
		return m.createEnvironment(name)
	case promptNewWorkspace:
		// Naming alone isn't enough - a user wants to choose exactly where
		// this gets saved, not just have it land wherever newWorkspacesDir
		// happens to point. Validate the name now (so a bad name fails
		// before ever asking about a location) and carry it via
		// targetPath into a second prompt, prefilled with the same
		// default this used to create unconditionally - accepting it with
		// another enter reproduces the old one-step behavior exactly.
		if err := fsstore.ValidateName(name); err != nil {
			m.status = "New workspace failed: " + err.Error()
			return m, nil
		}
		defaultRoot := filepath.Join(m.newWorkspacesDir, name)
		m.prompt.Open(promptNewWorkspacePath, name, "Save workspace at", defaultRoot)
		return m, nil
	case promptNewWorkspacePath:
		return m.createWorkspace(target, name)
	case promptDashboardSearch:
		m.dashboard.filter = name
		m.dashboard.cursor = 0
		if name == "" {
			m.status = "Ready"
		} else {
			m.status = "Filtering runs by " + name
		}
	}
	return m, nil
}

// confirmPurpose selects what happens when a confirmState is answered "y".
type confirmPurpose int

const (
	confirmDelete confirmPurpose = iota
	confirmQuit
	confirmDeleteEnvironment
	confirmDeleteWorkspace
	confirmDeleteRun
)

// confirmState is a single yes/no modal, used for both deleting a sidebar
// node and confirming an at-rest double-esc quit.
type confirmState struct {
	active     bool
	purpose    confirmPurpose
	label      string
	targetPath string
}

func (c *confirmState) Open(purpose confirmPurpose, label, targetPath string) {
	c.active = true
	c.purpose = purpose
	c.label = label
	c.targetPath = targetPath
}

func (c *confirmState) Close() {
	c.active = false
}

// View renders the confirmation as a centered modal dialog box - see
// Model.View in view.go for where this gets placed over a blank backdrop.
func (c confirmState) View() string {
	if !c.active {
		return ""
	}
	content := errStyle.Bold(true).Render(c.label) + "\n\n" + labelStyle.Render("y = yes    n = no")
	return modalStyle.Width(modalWidth).Align(lipgloss.Center).Render(content)
}

// pathIsAtOrUnder reports whether p is ancestor itself or a descendant of it.
// Store paths use filepath separators, so a descendant is prefixed by
// ancestor + separator. An empty p (nothing loaded) is never under anything.
func pathIsAtOrUnder(p, ancestor string) bool {
	if p == "" {
		return false
	}
	if p == ancestor {
		return true
	}
	return strings.HasPrefix(p, ancestor+string(filepath.Separator))
}

func (m Model) handleConfirmKey(k tea.KeyMsg) (Model, tea.Cmd) {
	switch k.String() {
	case "y":
		purpose := m.confirm.purpose
		target := m.confirm.targetPath
		m.confirm.Close()
		switch purpose {
		case confirmQuit:
			return m, tea.Quit
		case confirmDeleteEnvironment:
			return m.deleteEnvironment(target)
		case confirmDeleteWorkspace:
			return m.deleteWorkspace(target)
		case confirmDeleteRun:
			return m.deleteSelectedDashboardRun()
		}
		if err := m.store.Delete(target); err != nil {
			m.status = "Delete failed: " + err.Error()
			return m, nil
		}
		// target may be a folder; store.Delete removes it and all children
		// recursively, so clear any loaded request and cached response that
		// lives at target or underneath it, not just an exact match.
		if pathIsAtOrUnder(m.loadedRequestPath, target) {
			m.loadedRequestPath = ""
		}
		for cached := range m.responseCache {
			if pathIsAtOrUnder(cached, target) {
				delete(m.responseCache, cached)
			}
		}
		_ = m.lastResponseStore.Delete(target)
		m.refreshTree()
		m.status = "Deleted"
		return m, nil
	case "n", "esc":
		m.confirm.Close()
		return m, nil
	}
	return m, nil
}
