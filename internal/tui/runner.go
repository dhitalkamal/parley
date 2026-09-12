package tui

import (
	"context"
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	execapp "github.com/dhitalkamal/parley/internal/execution/application"
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
	history "github.com/dhitalkamal/parley/internal/history/domain"
	scripting "github.com/dhitalkamal/parley/internal/scripting/domain"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// maxRunnerResultsShown caps the runner summary display so a very large
// collection doesn't blow out the layout - the cap is surfaced explicitly
// ("...and N more"), never silently.
const maxRunnerResultsShown = 20

type runnerState struct {
	active  bool
	running bool
	results []execution.RunResult
	totalMS int64
}

type runResultMsg struct {
	// path is the run's human-readable label for the dashboard's history -
	// the selected folder's display Name, or "" for a whole-collection run.
	// Deliberately not the raw store path (order-prefixed, e.g. "020_users"):
	// that's an on-disk encoding detail the dashboard shouldn't surface.
	path    string
	results []execution.RunResult
	ctx     scripting.ScriptContext
	err     error
}

// startCollectionRun runs everything under the selected folder, or the
// whole collection if a request (or nothing) is selected.
func (m Model) startCollectionRun() (Model, tea.Cmd) {
	tree, err := m.store.Tree()
	if err != nil {
		m.status = "Run failed: " + err.Error()
		return m, nil
	}

	target := ""
	if item, ok := m.sidebar.Selected(); ok && item.kind == collection.KindFolder {
		target = item.path
	}
	node, ok := findNodeByPath(tree, target)
	if !ok {
		node = tree
	}
	recordedPath := ""
	if target != "" {
		recordedPath = node.Name
	}

	paths := collection.RequestPathsUnder(node)
	if len(paths) == 0 {
		m.status = "No requests to run"
		return m, nil
	}

	m.runner = runnerState{active: true, running: true}
	m.status = "Running collection..."
	return m, runCollectionCmd(m.store, m.client, m.scriptRunner, paths, m.scriptContext(), recordedPath)
}

// startRerunFromHistory re-executes a recorded run's exact request list -
// reconstructed from its own Results (each retains the RequestPath actually
// run), rather than re-resolving whatever folder was selected in the
// sidebar at the time, so a rerun still works even if that folder has since
// been renamed or the run was against the whole collection (Path == "").
func (m Model) startRerunFromHistory(run history.CollectionRunEntry) (Model, tea.Cmd) {
	paths := make([]string, len(run.Results))
	for i, r := range run.Results {
		paths[i] = r.RequestPath
	}
	if len(paths) == 0 {
		m.status = "No requests to rerun"
		return m, nil
	}

	m.runner = runnerState{active: true, running: true}
	m.status = "Running collection..."
	return m, runCollectionCmd(m.store, m.client, m.scriptRunner, paths, m.scriptContext(), run.Path)
}

func findNodeByPath(node collection.TreeNode, path string) (collection.TreeNode, bool) {
	if node.Path == path {
		return node, true
	}
	for _, child := range node.Children {
		if found, ok := findNodeByPath(child, path); ok {
			return found, true
		}
	}
	return collection.TreeNode{}, false
}

func runCollectionCmd(loader execapp.RequestLoader, client execution.HTTPClient, scripts execution.ScriptRunner, paths []string, initial scripting.ScriptContext, path string) tea.Cmd {
	return func() tea.Msg {
		results, ctx, err := execapp.RunCollection(context.Background(), execapp.RunOptions{
			Loader:         loader,
			Client:         client,
			Scripts:        scripts,
			RequestPaths:   paths,
			InitialContext: initial,
		})
		return runResultMsg{path: path, results: results, ctx: ctx, err: err}
	}
}

func (m Model) handleRunResult(msg runResultMsg) Model {
	m.runner.running = false
	if msg.err != nil {
		m.runner.active = false
		m.status = "Run failed: " + msg.err.Error()
		return m
	}
	m.runner.results = msg.results
	var total int64
	for _, r := range msg.results {
		total += r.ElapsedMS
	}
	m.runner.totalMS = total
	entry := history.CollectionRunEntry{
		Time:    time.Now(),
		Path:    msg.path,
		Results: msg.results,
		TotalMS: total,
	}
	// Best-effort, like historyStore.AppendHistory elsewhere - a run's
	// results are still shown even if recording them for the dashboard fails.
	_ = m.runHistoryStore.AppendRun(entry)
	// Keep the Dashboard's own in-memory run list in sync immediately - a
	// rerun triggered from there (see startRerunFromHistory) shouldn't
	// require leaving and reopening the screen to see itself in the log.
	m.dashboard.setRuns(append([]history.CollectionRunEntry{entry}, m.dashboard.runs...))
	m.applyScriptContext(msg.ctx)
	m.status = "Run complete"
	return m
}

func (m Model) handleRunnerKey(k tea.KeyMsg) (Model, tea.Cmd) {
	if k.String() == "esc" || key.Matches(k, sidebarKeys.Run) {
		m.runner.active = false
	}
	return m, nil
}

// runnerHeader is the same title+esc-hint convention every modal in the app
// uses (see envPanelState.View's "Manage Environments  (esc close)").
func runnerHeader() string {
	return activeTabStyle.Render("Collection run") + labelStyle.Render("  (esc close)")
}

func (m Model) runnerView() string {
	if m.runner.running {
		return borderStyle.Render(runnerHeader() + "\n" + labelStyle.Render("Running..."))
	}
	if len(m.runner.results) == 0 {
		return borderStyle.Render(runnerHeader() + "\n" + labelStyle.Render("No results"))
	}

	summary := runResultsSummary(m.runner.results, m.runner.totalMS)
	body := runResultsBody(m.runner.results, maxRunnerResultsShown)
	return borderStyle.Render(runnerHeader() + "\n" + labelStyle.Render(summary) + "\n" + body)
}
