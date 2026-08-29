package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
	"github.com/dhitalkamal/parley/internal/importexport"
)

// startImport opens a prompt accepting either a raw curl command, a pasted
// Postman/OpenAPI document, or a path to a file containing one of those -
// format is auto-detected on submit by importexport.ParseAny. Imported
// requests land under the currently selected folder.
func (m Model) startImport() (Model, tea.Cmd) {
	m.prompt.Open(promptImport, m.selectedFolderPath(), "Import (curl, Postman/OpenAPI text, or file path)", "")
	return m, nil
}

// startExport opens a prompt for a destination file path and exports the
// selected folder (or the whole collection if nothing folder-shaped is
// selected) as a Postman v2.1 collection.
func (m Model) startExport() (Model, tea.Cmd) {
	target := ""
	if item, ok := m.sidebar.Selected(); ok && item.kind == collection.KindFolder {
		target = item.path
	}
	m.prompt.Open(promptExport, target, "Export to file", "collection.json")
	return m, nil
}

// exportToFile builds an ImportedNode tree from everything under sourcePath
// (the whole collection if empty) and writes it to destPath as a Postman
// v2.1 collection.
func (m Model) exportToFile(sourcePath, destPath string) error {
	tree, err := m.store.Tree()
	if err != nil {
		return err
	}
	source, ok := findNodeByPath(tree, sourcePath)
	if !ok {
		source = tree
	}
	node, err := importexport.BuildImportedNode(m.store, source)
	if err != nil {
		return err
	}
	name := source.Name
	if name == "" {
		name = "collection"
	}
	data, err := importexport.ExportPostmanCollection(name, node)
	if err != nil {
		return err
	}
	return os.WriteFile(destPath, data, 0o644)
}

// codeSnippetView renders the current editor request, with variables
// substituted the same way a real send would resolve them, as both a curl
// command and a Go net/http program.
func (m Model) codeSnippetView() string {
	req, _ := execution.SubstituteRequest(m.buildRequest(), m.resolvedVars())
	curlSnippet := importexport.GenerateCurl(req)
	goSnippet := importexport.GenerateGo(req)
	return labelStyle.Render("curl:") + "\n" + curlSnippet + "\n\n" +
		labelStyle.Render("Go (net/http):") + "\n" + goSnippet
}

// truncateLines caps s at max lines, noting how many were dropped instead of
// silently cutting them - the same explicit-cap convention as
// maxRunnerResultsShown's "...and N more".
func truncateLines(s string, max int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= max {
		return s
	}
	kept := strings.Join(lines[:max], "\n")
	return fmt.Sprintf("%s\n... (%d more line(s) - enlarge the terminal to see the rest)", kept, len(lines)-max)
}
