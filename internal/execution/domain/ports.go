package execution

import (
	"context"

	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	scripting "github.com/dhitalkamal/parley/internal/scripting/domain"
)

// HTTPClient executes a resolved collection.Request. Implemented by
// execution/infrastructure/httpclient.
type HTTPClient interface {
	Do(ctx context.Context, req collection.Request) (Response, error)
}

// LastResponseStore persists the most recent response each saved request
// (identified by its collection.Store path) actually returned, so the
// Response panel can restore it after the app restarts - not just for the
// rest of the current session. Implemented by execution/infrastructure/store.
type LastResponseStore interface {
	Save(requestPath string, resp Response, elapsedMS int64) error
	Load(requestPath string) (resp Response, elapsedMS int64, ok bool, err error)
	Delete(requestPath string) error
}

// ScriptRunner executes pre-request and test scripts against a resolved
// collection.Request. Implemented by scripting/infrastructure via goja.
type ScriptRunner interface {
	RunPreRequest(script string, req collection.Request, ctx scripting.ScriptContext) (collection.Request, scripting.ScriptContext, error)
	RunTest(script string, req collection.Request, resp Response, ctx scripting.ScriptContext) ([]scripting.TestResult, scripting.ScriptContext, error)
}
