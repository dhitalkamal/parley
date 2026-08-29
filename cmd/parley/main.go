// Command parley is a terminal api client. `parley` alone launches the TUI;
// `parley run [flags] <path>` executes a saved collection or request
// headlessly, for CI.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dhitalkamal/parley/internal/cli"
	collectionstore "github.com/dhitalkamal/parley/internal/collection/infrastructure"
	environmentstore "github.com/dhitalkamal/parley/internal/environment/infrastructure"
	httpclient "github.com/dhitalkamal/parley/internal/execution/infrastructure/httpclient"
	historystore "github.com/dhitalkamal/parley/internal/history/infrastructure"
	scriptengine "github.com/dhitalkamal/parley/internal/scripting/infrastructure"
	"github.com/dhitalkamal/parley/internal/tui"
	workspace "github.com/dhitalkamal/parley/internal/workspace/domain"
	workspacestore "github.com/dhitalkamal/parley/internal/workspace/infrastructure"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "run" {
		home, err := dataHome()
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(cli.ExitUsageError)
		}
		os.Exit(runCLICommand(os.Args[2:], home, os.Stdout, os.Stderr))
	}

	m, err := buildModel()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// runCLICommand implements `parley run`: it resolves the workspace the same
// way the TUI does (active workspace, or --workspace to pick another one),
// then hands off to cli.Run for the send/script/report loop. home, stdout,
// and stderr are parameters (rather than reading os.Args/os.Stdout/os.Stderr
// directly) so this is testable without a subprocess.
func runCLICommand(args []string, home string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(stderr)
	workspaceName := fs.String("workspace", "", "workspace to run against (default: the active workspace)")
	envName := fs.String("env", "", "environment to substitute {{variables}} from (default: none)")
	reporter := fs.String("reporter", "text", "report format: text, json, or junit")
	outPath := fs.String("out", "", "write the report to this file instead of stdout")
	if err := fs.Parse(args); err != nil {
		return cli.ExitUsageError
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "usage: parley run [flags] <collection-or-request-path>")
		return cli.ExitUsageError
	}
	path := fs.Arg(0)

	wsStore := workspacestore.New(filepath.Join(home, "workspaces.json"))
	workspaces, err := wsStore.List()
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return cli.ExitUsageError
	}

	name := *workspaceName
	if name == "" {
		if name, err = wsStore.ActiveName(); err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return cli.ExitUsageError
		}
	} else if !workspaceExists(workspaces, name) {
		// An explicit --workspace typo must fail loudly rather than silently
		// falling back to another workspace (selectWorkspace's soft fallback
		// is only right for the persisted active-name case, e.g. after that
		// workspace was deleted) - a CI run silently testing the wrong
		// environment is worse than one that refuses to start.
		fmt.Fprintf(stderr, "error: no workspace named %q\n", name)
		return cli.ExitUsageError
	}
	active, err := selectWorkspace(workspaces, name)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return cli.ExitUsageError
	}

	out := io.Writer(stdout)
	if *outPath != "" {
		f, err := os.Create(*outPath)
		if err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return cli.ExitUsageError
		}
		defer f.Close()
		out = f
	}

	code, err := cli.Run(context.Background(), cli.Options{
		Store:    collectionstore.New(active.CollectionsRoot),
		EnvStore: environmentstore.New(active.ProjectRoot),
		Client:   httpclient.New(),
		Scripts:  scriptengine.New(),
		RunStore: historystore.NewRunStore(active.ProjectRoot),
		Path:     path,
		EnvName:  *envName,
		Reporter: *reporter,
		Out:      out,
	})
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
	}
	return code
}

// selectWorkspace picks the workspace named name, or workspaces[0] if name
// is empty or doesn't match any known workspace - the same soft fallback
// buildModel has always used for a stale or unset active-workspace name.
func selectWorkspace(workspaces []workspace.Workspace, name string) (workspace.Workspace, error) {
	if len(workspaces) == 0 {
		return workspace.Workspace{}, fmt.Errorf("no workspaces configured - launch parley once to create one")
	}
	for _, w := range workspaces {
		if w.Name == name {
			return w, nil
		}
	}
	return workspaces[0], nil
}

func workspaceExists(workspaces []workspace.Workspace, name string) bool {
	for _, w := range workspaces {
		if w.Name == name {
			return true
		}
	}
	return false
}

// buildModel resolves the workspace registry under the parley home and seeds a
// default "Personal" workspace pointing at <home>/collections, so parley opens
// the same workspace regardless of the directory it is launched from. the home
// is PARLEY_HOME, else $XDG_CONFIG_HOME/parley, else ~/.config/parley - see
// dataHome.
func buildModel() (tui.Model, error) {
	home, err := dataHome()
	if err != nil {
		return tui.Model{}, err
	}

	collectionsRoot := filepath.Join(home, "collections")
	// ensure the workspace exists on first launch so the store has somewhere
	// to read and write; collections/ is the tree root, home holds
	// environments/, globals.json and history.jsonl as siblings.
	if err := os.MkdirAll(collectionsRoot, 0o755); err != nil {
		return tui.Model{}, err
	}

	wsStore := workspacestore.New(filepath.Join(home, "workspaces.json"))
	workspaces, err := wsStore.List()
	if err != nil {
		return tui.Model{}, err
	}
	if len(workspaces) == 0 {
		personal := workspace.Workspace{Name: "Personal", CollectionsRoot: collectionsRoot, ProjectRoot: home}
		if err := wsStore.Save(personal); err != nil {
			return tui.Model{}, err
		}
		if err := wsStore.SetActiveName(personal.Name); err != nil {
			return tui.Model{}, err
		}
		workspaces = []workspace.Workspace{personal}
	}

	activeName, err := wsStore.ActiveName()
	if err != nil {
		return tui.Model{}, err
	}
	active, err := selectWorkspace(workspaces, activeName)
	if err != nil {
		return tui.Model{}, err
	}

	m := tui.New(active.CollectionsRoot, active.ProjectRoot)
	m.SetWorkspaces(wsStore, workspaces, active.Name, filepath.Join(home, "workspaces"))
	return m, nil
}

// dataHome resolves the parley home directory. precedence: PARLEY_HOME, then
// $XDG_CONFIG_HOME/parley, then ~/.config/parley. this keeps one shared
// workspace regardless of the directory parley is launched from.
func dataHome() (string, error) {
	if h := os.Getenv("PARLEY_HOME"); h != "" {
		return h, nil
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "parley"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "parley"), nil
}
