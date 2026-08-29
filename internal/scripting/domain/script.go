package scripting

// ScriptContext is the variable state exposed to pre-request/test scripts
// via pm.environment/pm.globals/pm.variables. Plain resolved values, not the
// full Variable (scripts only read/write values; Enabled/Secret bookkeeping
// stays the caller's responsibility when merging mutations back).
type ScriptContext struct {
	Environment map[string]string
	Globals     map[string]string
}

// TestResult is the outcome of one pm.test(name, fn) call.
type TestResult struct {
	Name   string
	Passed bool
	Error  string
}
