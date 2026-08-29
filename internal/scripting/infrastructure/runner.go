// Package script embeds a goja JS runtime exposing a Postman-compatible
// pm.* API for pre-request and test scripts.
package scriptengine

import (
	_ "embed"
	"encoding/json"
	"fmt"
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
	scripting "github.com/dhitalkamal/parley/internal/scripting/domain"
	"strings"
	"time"

	"github.com/dop251/goja"
)

//go:embed chai.js
var chaiShim string

// scriptTimeout guards against a runaway script (e.g. an infinite loop)
// hanging the whole TUI.
const scriptTimeout = 5 * time.Second

// Runner implements execution.ScriptRunner. Each call gets a fresh
// goja.Runtime - scripts are short-lived enough that isolation beats reuse.
type Runner struct{}

var _ execution.ScriptRunner = (*Runner)(nil)

func New() *Runner { return &Runner{} }

func (r *Runner) RunPreRequest(script string, req collection.Request, ctx scripting.ScriptContext) (collection.Request, scripting.ScriptContext, error) {
	if strings.TrimSpace(script) == "" {
		return req, ctx, nil
	}
	ctx = ensureContext(ctx)
	vm := goja.New()
	pmObj, err := setupPM(vm, ctx)
	if err != nil {
		return req, ctx, err
	}

	headers := effectiveHeaderMap(req.Headers)
	headerAdds := make(map[string]string)
	pmObj.Set("request", map[string]interface{}{
		"method": string(req.Method),
		"url":    req.URL,
		"headers": map[string]interface{}{
			"get": func(key string) interface{} {
				if v, ok := headers[key]; ok {
					return v
				}
				return goja.Undefined()
			},
			"add": func(key, value string) {
				headerAdds[key] = value
			},
		},
	})

	if err := runScript(vm, script); err != nil {
		return req, ctx, fmt.Errorf("pre-request script: %w", err)
	}

	newReq := req
	if len(headerAdds) > 0 {
		newReq.Headers = append([]collection.Header{}, req.Headers...)
		for k, v := range headerAdds {
			newReq.Headers = upsertHeader(newReq.Headers, k, v)
		}
	}
	return newReq, ctx, nil
}

func (r *Runner) RunTest(script string, req collection.Request, resp execution.Response, ctx scripting.ScriptContext) ([]scripting.TestResult, scripting.ScriptContext, error) {
	if strings.TrimSpace(script) == "" {
		return nil, ctx, nil
	}
	ctx = ensureContext(ctx)
	vm := goja.New()
	pmObj, err := setupPM(vm, ctx)
	if err != nil {
		return nil, ctx, err
	}

	pmObj.Set("request", map[string]interface{}{
		"method": string(req.Method),
		"url":    req.URL,
	})

	respHeaders := make(map[string]string)
	for _, h := range resp.Headers {
		respHeaders[h.Key] = h.Value
	}
	pmObj.Set("response", map[string]interface{}{
		"code":   resp.StatusCode,
		"status": resp.Status,
		"text":   func() string { return string(resp.Body) },
		"json": func() interface{} {
			var parsed interface{}
			if err := json.Unmarshal(resp.Body, &parsed); err != nil {
				panic(vm.NewGoError(fmt.Errorf("response body is not valid JSON: %w", err)))
			}
			return parsed
		},
		"headers": map[string]interface{}{
			"get": func(key string) interface{} {
				if v, ok := respHeaders[key]; ok {
					return v
				}
				return goja.Undefined()
			},
		},
	})

	if err := runScript(vm, script); err != nil {
		return nil, ctx, fmt.Errorf("test script: %w", err)
	}

	results, err := extractTestResults(vm)
	if err != nil {
		return nil, ctx, err
	}
	return results, ctx, nil
}

// setupPM loads the chai/pm.test shim and attaches the environment/globals/
// variables bridges, which every script (pre-request or test) needs. The
// returned object still needs "request" (and, for test scripts, "response")
// attached by the caller.
func setupPM(vm *goja.Runtime, ctx scripting.ScriptContext) (*goja.Object, error) {
	if _, err := vm.RunString(chaiShim + "\nvar pm = {}; pm.test = pmTest; pm.expect = pmExpect;"); err != nil {
		return nil, err
	}
	pmObj := vm.Get("pm").ToObject(vm)
	pmObj.Set("environment", varBridge(ctx.Environment))
	pmObj.Set("globals", varBridge(ctx.Globals))
	pmObj.Set("variables", readOnlyVarBridge(mergeVars(ctx.Globals, ctx.Environment)))
	return pmObj, nil
}

// varBridge exposes a Go map as pm.environment/pm.globals: get/set are plain
// function calls closing over the map, so mutations are just map writes -
// no fragile live JS-property binding needed, and the caller's ScriptContext
// already reflects every change once the script returns.
func varBridge(vars map[string]string) map[string]interface{} {
	return map[string]interface{}{
		"get": func(key string) interface{} {
			if v, ok := vars[key]; ok {
				return v
			}
			return goja.Undefined()
		},
		"set": func(key, value string) {
			vars[key] = value
		},
	}
}

func readOnlyVarBridge(vars map[string]string) map[string]interface{} {
	return map[string]interface{}{
		"get": func(key string) interface{} {
			if v, ok := vars[key]; ok {
				return v
			}
			return goja.Undefined()
		},
	}
}

func mergeVars(globals, env map[string]string) map[string]string {
	out := make(map[string]string, len(globals)+len(env))
	for k, v := range globals {
		out[k] = v
	}
	for k, v := range env {
		out[k] = v
	}
	return out
}

func ensureContext(ctx scripting.ScriptContext) scripting.ScriptContext {
	if ctx.Environment == nil {
		ctx.Environment = map[string]string{}
	}
	if ctx.Globals == nil {
		ctx.Globals = map[string]string{}
	}
	return ctx
}

func effectiveHeaderMap(headers []collection.Header) map[string]string {
	m := make(map[string]string)
	for _, h := range headers {
		if h.Enabled {
			m[h.Key] = h.Value
		}
	}
	return m
}

func upsertHeader(headers []collection.Header, key, value string) []collection.Header {
	for i, h := range headers {
		if strings.EqualFold(h.Key, key) {
			headers[i].Value = value
			headers[i].Enabled = true
			return headers
		}
	}
	return append(headers, collection.Header{Key: key, Value: value, Enabled: true})
}

func runScript(vm *goja.Runtime, script string) error {
	timer := time.AfterFunc(scriptTimeout, func() {
		vm.Interrupt("script timed out after 5s (possible infinite loop)")
	})
	defer timer.Stop()
	_, err := vm.RunString(script)
	return err
}

func extractTestResults(vm *goja.Runtime) ([]scripting.TestResult, error) {
	val := vm.Get("__pmTestResults")
	if val == nil || goja.IsUndefined(val) {
		return nil, nil
	}
	var raw []map[string]interface{}
	if err := vm.ExportTo(val, &raw); err != nil {
		return nil, fmt.Errorf("reading test results: %w", err)
	}
	results := make([]scripting.TestResult, 0, len(raw))
	for _, r := range raw {
		tr := scripting.TestResult{}
		if v, ok := r["name"].(string); ok {
			tr.Name = v
		}
		if v, ok := r["passed"].(bool); ok {
			tr.Passed = v
		}
		if v, ok := r["error"].(string); ok {
			tr.Error = v
		}
		results = append(results, tr)
	}
	return results, nil
}
