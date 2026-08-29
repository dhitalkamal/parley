package execapp

import (
	"context"
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	environment "github.com/dhitalkamal/parley/internal/environment/domain"
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
	scripting "github.com/dhitalkamal/parley/internal/scripting/domain"
	"strings"
	"time"
)

// RequestLoader is the narrow capability RunCollection needs from a
// collection.Store - a real Store satisfies it structurally, and tests can use
// a minimal fake instead of implementing the whole Store interface.
type RequestLoader interface {
	LoadRequest(path string) (collection.Request, error)
}

// RunOptions configures a collection run. Scripts may be nil if no request
// being run has a pre-request/test script - RunCollection skips scripting
// entirely in that case.
type RunOptions struct {
	Loader         RequestLoader
	Client         execution.HTTPClient
	Scripts        execution.ScriptRunner
	RequestPaths   []string
	InitialContext scripting.ScriptContext
}

// RunCollection executes every path in opts.RequestPaths in order: load,
// run the pre-request script, resolve {{variables}}, send, run the test
// script - threading a single mutable ScriptContext across the whole run so
// one request's pm.environment.set(...) is visible to the next, the same
// chaining behavior a manual send gets, just across an entire run. A
// failure at any step for one request is recorded on that request's
// RunResult and the run continues; it never aborts the whole run.
func RunCollection(ctx context.Context, opts RunOptions) ([]execution.RunResult, scripting.ScriptContext, error) {
	scriptCtx := opts.InitialContext
	var results []execution.RunResult

	for _, path := range opts.RequestPaths {
		start := time.Now()
		req, err := opts.Loader.LoadRequest(path)
		if err != nil {
			results = append(results, execution.RunResult{RequestPath: path, Err: err.Error()})
			continue
		}

		if opts.Scripts != nil && strings.TrimSpace(req.PreRequestScript) != "" {
			newReq, newCtx, err := opts.Scripts.RunPreRequest(req.PreRequestScript, req, scriptCtx)
			if err != nil {
				results = append(results, execution.RunResult{
					RequestPath: path, Method: req.Method, URL: req.URL,
					Err: "pre-request script: " + err.Error(),
				})
				continue
			}
			req = newReq
			scriptCtx = newCtx
		}

		resolved, _ := execution.SubstituteRequest(req, scriptContextVars(scriptCtx))
		resp, sendErr := SendRequest(ctx, opts.Client, resolved)
		elapsed := time.Since(start).Milliseconds()
		result := execution.RunResult{RequestPath: path, Method: req.Method, URL: resolved.URL, ElapsedMS: elapsed}
		if sendErr != nil {
			result.Err = sendErr.Error()
			results = append(results, result)
			continue
		}
		result.StatusCode = resp.StatusCode
		result.Status = resp.Status

		if opts.Scripts != nil && strings.TrimSpace(req.TestScript) != "" {
			testResults, newCtx, err := opts.Scripts.RunTest(req.TestScript, resolved, resp, scriptCtx)
			if err != nil {
				result.Err = "test script: " + err.Error()
			} else {
				result.Tests = testResults
				scriptCtx = newCtx
			}
		}
		results = append(results, result)
	}

	return results, scriptCtx, nil
}

// scriptContextVars adapts a ScriptContext's plain maps into the
// map[string]Variable shape execution.SubstituteRequest expects.
func scriptContextVars(ctx scripting.ScriptContext) map[string]environment.Variable {
	vars := make(map[string]environment.Variable, len(ctx.Globals)+len(ctx.Environment))
	for k, v := range ctx.Globals {
		vars[k] = environment.Variable{Key: k, Value: v, Enabled: true}
	}
	for k, v := range ctx.Environment {
		vars[k] = environment.Variable{Key: k, Value: v, Enabled: true}
	}
	return vars
}
