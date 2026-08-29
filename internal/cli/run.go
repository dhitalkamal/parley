package cli

import (
	"context"
	"fmt"
	"io"
	"time"

	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	environment "github.com/dhitalkamal/parley/internal/environment/domain"
	execapp "github.com/dhitalkamal/parley/internal/execution/application"
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
	history "github.com/dhitalkamal/parley/internal/history/domain"
	scripting "github.com/dhitalkamal/parley/internal/scripting/domain"
)

// Options configures a headless collection run. EnvStore may be nil if the
// caller has no environments configured - Globals/Environment are then left
// empty and only literal (non-{{var}}) values resolve. RunStore may be nil
// if the caller doesn't want this run recorded for the dashboard's history.
type Options struct {
	Store    collection.Store
	EnvStore environment.EnvironmentStore
	Client   execution.HTTPClient
	Scripts  execution.ScriptRunner
	RunStore history.RunHistoryStore
	Path     string
	EnvName  string
	Reporter string
	Out      io.Writer
}

// Exit codes: 0 means every request sent and every assertion passed, 1
// means the run completed but something failed, 2 means the run couldn't
// even start (bad path, bad reporter, bad environment name).
const (
	ExitSuccess     = 0
	ExitTestFailure = 1
	ExitUsageError  = 2
)

// Run resolves opts.Path against the collection's tree, executes every
// matching request through the same send/pre-request/test-script pipeline
// the TUI uses, writes a report in opts.Reporter's format to opts.Out, and
// returns the process exit code to use.
func Run(ctx context.Context, opts Options) (int, error) {
	tree, err := opts.Store.Tree()
	if err != nil {
		return ExitUsageError, err
	}
	paths, err := ResolveRequestPaths(tree, opts.Path)
	if err != nil {
		return ExitUsageError, err
	}
	if err := validateReporter(opts.Reporter); err != nil {
		return ExitUsageError, err
	}
	scriptCtx, err := loadScriptContext(opts)
	if err != nil {
		return ExitUsageError, err
	}

	results, _, err := execapp.RunCollection(ctx, execapp.RunOptions{
		Loader:         opts.Store,
		Client:         opts.Client,
		Scripts:        opts.Scripts,
		RequestPaths:   paths,
		InitialContext: scriptCtx,
	})
	if err != nil {
		return ExitUsageError, err
	}

	if opts.RunStore != nil {
		if err := opts.RunStore.AppendRun(history.CollectionRunEntry{
			Time:    time.Now(),
			Path:    opts.Path,
			Results: results,
			TotalMS: totalElapsedMS(results),
		}); err != nil {
			return ExitUsageError, fmt.Errorf("cli: recording run history: %w", err)
		}
	}

	if err := writeReport(opts.Reporter, opts.Out, results); err != nil {
		return ExitUsageError, err
	}
	if !Success(results) {
		return ExitTestFailure, nil
	}
	return ExitSuccess, nil
}

func totalElapsedMS(results []execution.RunResult) int64 {
	var total int64
	for _, r := range results {
		total += r.ElapsedMS
	}
	return total
}

func loadScriptContext(opts Options) (scripting.ScriptContext, error) {
	scriptCtx := scripting.ScriptContext{Environment: map[string]string{}, Globals: map[string]string{}}
	if opts.EnvStore == nil {
		return scriptCtx, nil
	}

	globals, err := opts.EnvStore.LoadGlobals()
	if err != nil {
		return scripting.ScriptContext{}, fmt.Errorf("cli: loading globals: %w", err)
	}
	scriptCtx.Globals = enabledVars(globals.Variables)

	if opts.EnvName != "" {
		env, err := opts.EnvStore.LoadEnvironment(opts.EnvName)
		if err != nil {
			return scripting.ScriptContext{}, fmt.Errorf("cli: loading environment %q: %w", opts.EnvName, err)
		}
		scriptCtx.Environment = enabledVars(env.Variables)
	}
	return scriptCtx, nil
}

func enabledVars(vars []environment.Variable) map[string]string {
	out := make(map[string]string, len(vars))
	for _, v := range vars {
		if v.Enabled {
			out[v.Key] = v.Value
		}
	}
	return out
}

func validateReporter(reporter string) error {
	switch reporter {
	case "", "text", "json", "junit":
		return nil
	default:
		return unknownReporterErr(reporter)
	}
}

func writeReport(reporter string, out io.Writer, results []execution.RunResult) error {
	switch reporter {
	case "", "text":
		WriteText(out, results)
		return nil
	case "json":
		return WriteJSON(out, results)
	case "junit":
		return WriteJUnit(out, results, "parley")
	default:
		return unknownReporterErr(reporter)
	}
}

func unknownReporterErr(reporter string) error {
	return fmt.Errorf("cli: unknown reporter %q (want text, json, or junit)", reporter)
}
