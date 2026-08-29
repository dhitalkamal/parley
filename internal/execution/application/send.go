// Package application holds the use cases the tui drives: sending a request,
// running a collection, chaining values between requests. Later phases add
// variable substitution and pre-request/test scripts here, ahead of the
// execution.HTTPClient call.
package execapp

import (
	"context"
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
	"time"
)

// DefaultTimeout applies when a request doesn't specify one.
const DefaultTimeout = 30 * time.Second

// SendRequest resolves and executes req against client.
func SendRequest(ctx context.Context, client execution.HTTPClient, req collection.Request) (execution.Response, error) {
	if req.Timeout == 0 {
		req.Timeout = DefaultTimeout
	}
	return client.Do(ctx, req)
}
