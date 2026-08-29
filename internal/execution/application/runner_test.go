package execapp

import (
	"context"
	"errors"
	"fmt"
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
	scripting "github.com/dhitalkamal/parley/internal/scripting/domain"
	scriptengine "github.com/dhitalkamal/parley/internal/scripting/infrastructure"
	"testing"
)

type fakeLoader struct {
	requests map[string]collection.Request
}

func (f *fakeLoader) LoadRequest(path string) (collection.Request, error) {
	req, ok := f.requests[path]
	if !ok {
		return collection.Request{}, fmt.Errorf("not found: %s", path)
	}
	return req, nil
}

type fakeSequentialClient struct {
	responses []execution.Response
	errs      []error
	calls     []collection.Request
}

func (f *fakeSequentialClient) Do(ctx context.Context, req collection.Request) (execution.Response, error) {
	idx := len(f.calls)
	f.calls = append(f.calls, req)
	var resp execution.Response
	var err error
	if idx < len(f.responses) {
		resp = f.responses[idx]
	}
	if idx < len(f.errs) {
		err = f.errs[idx]
	}
	return resp, err
}

func TestRunCollection_RunsEachRequestInOrder(t *testing.T) {
	loader := &fakeLoader{requests: map[string]collection.Request{
		"a.json": {Method: collection.GET, URL: "https://example.com/a"},
		"b.json": {Method: collection.GET, URL: "https://example.com/b"},
	}}
	client := &fakeSequentialClient{responses: []execution.Response{
		{StatusCode: 200, Status: "200 OK"},
		{StatusCode: 201, Status: "201 Created"},
	}}

	results, _, err := RunCollection(context.Background(), RunOptions{
		Loader:       loader,
		Client:       client,
		RequestPaths: []string{"a.json", "b.json"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	if results[0].RequestPath != "a.json" || results[0].StatusCode != 200 {
		t.Errorf("result[0] = %+v", results[0])
	}
	if results[1].RequestPath != "b.json" || results[1].StatusCode != 201 {
		t.Errorf("result[1] = %+v", results[1])
	}
	if len(client.calls) != 2 || client.calls[0].URL != "https://example.com/a" || client.calls[1].URL != "https://example.com/b" {
		t.Errorf("calls = %+v", client.calls)
	}
}

func TestRunCollection_ContinuesAfterSendError(t *testing.T) {
	loader := &fakeLoader{requests: map[string]collection.Request{
		"a.json": {Method: collection.GET, URL: "https://example.com/a"},
		"b.json": {Method: collection.GET, URL: "https://example.com/b"},
	}}
	client := &fakeSequentialClient{
		errs:      []error{errors.New("connection refused")},
		responses: []execution.Response{{}, {StatusCode: 200}},
	}

	results, _, err := RunCollection(context.Background(), RunOptions{
		Loader:       loader,
		Client:       client,
		RequestPaths: []string{"a.json", "b.json"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2 (run should continue past a failure)", len(results))
	}
	if results[0].Err == "" {
		t.Error("expected result[0] to record the send error")
	}
	if results[1].Err != "" || results[1].StatusCode != 200 {
		t.Errorf("expected result[1] to succeed normally, got %+v", results[1])
	}
}

func TestRunCollection_LoadFailureRecordsErrorAndContinues(t *testing.T) {
	loader := &fakeLoader{requests: map[string]collection.Request{
		"b.json": {Method: collection.GET, URL: "https://example.com/b"},
	}}
	client := &fakeSequentialClient{responses: []execution.Response{{StatusCode: 200}}}

	results, _, err := RunCollection(context.Background(), RunOptions{
		Loader:       loader,
		Client:       client,
		RequestPaths: []string{"missing.json", "b.json"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	if results[0].Err == "" {
		t.Error("expected result[0] to record the load error")
	}
	if results[1].StatusCode != 200 {
		t.Errorf("expected result[1] to still run, got %+v", results[1])
	}
	if len(client.calls) != 1 {
		t.Errorf("expected only the loadable request to reach the client, got %d calls", len(client.calls))
	}
}

func TestRunCollection_EmptyPathsReturnsEmptyResults(t *testing.T) {
	results, _, err := RunCollection(context.Background(), RunOptions{
		Loader: &fakeLoader{},
		Client: &fakeSequentialClient{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("got %+v, want none", results)
	}
}

func TestRunCollection_RunsTestScriptAndRecordsResults(t *testing.T) {
	loader := &fakeLoader{requests: map[string]collection.Request{
		"a.json": {
			Method:     collection.GET,
			URL:        "https://example.com/a",
			TestScript: `pm.test("status is 200", function () { pm.expect(pm.response.code).to.equal(200); });`,
		},
	}}
	client := &fakeSequentialClient{responses: []execution.Response{{StatusCode: 200}}}

	results, _, err := RunCollection(context.Background(), RunOptions{
		Loader:       loader,
		Client:       client,
		Scripts:      scriptengine.New(),
		RequestPaths: []string{"a.json"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || len(results[0].Tests) != 1 || !results[0].Tests[0].Passed {
		t.Errorf("got %+v", results)
	}
}

func TestRunCollection_PreRequestScriptErrorRecordsErrorAndContinues(t *testing.T) {
	loader := &fakeLoader{requests: map[string]collection.Request{
		"a.json": {Method: collection.GET, URL: "https://example.com/a", PreRequestScript: `throw new Error("boom");`},
		"b.json": {Method: collection.GET, URL: "https://example.com/b"},
	}}
	client := &fakeSequentialClient{responses: []execution.Response{{StatusCode: 200}}}

	results, _, err := RunCollection(context.Background(), RunOptions{
		Loader:       loader,
		Client:       client,
		Scripts:      scriptengine.New(),
		RequestPaths: []string{"a.json", "b.json"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	if results[0].Err == "" {
		t.Error("expected result[0] to record the pre-request script error")
	}
	if len(client.calls) != 1 {
		t.Errorf("expected the pre-request-failed request to never reach the client, got %d calls", len(client.calls))
	}
	if results[1].StatusCode != 200 {
		t.Errorf("expected result[1] to still run, got %+v", results[1])
	}
}

func TestRunCollection_ChainsVariablesAcrossRequests(t *testing.T) {
	loader := &fakeLoader{requests: map[string]collection.Request{
		"login.json": {
			Method:     collection.GET,
			URL:        "https://example.com/login",
			TestScript: `pm.environment.set("token", pm.response.json().token);`,
		},
		"profile.json": {
			Method:  collection.GET,
			URL:     "https://example.com/profile",
			Headers: []collection.Header{{Key: "Authorization", Value: "Bearer {{token}}", Enabled: true}},
		},
	}}
	client := &fakeSequentialClient{responses: []execution.Response{
		{StatusCode: 200, Body: []byte(`{"token":"chained-xyz"}`)},
		{StatusCode: 200},
	}}

	_, finalCtx, err := RunCollection(context.Background(), RunOptions{
		Loader:         loader,
		Client:         client,
		Scripts:        scriptengine.New(),
		RequestPaths:   []string{"login.json", "profile.json"},
		InitialContext: scripting.ScriptContext{Environment: map[string]string{}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(client.calls) != 2 {
		t.Fatalf("got %d calls, want 2", len(client.calls))
	}
	var gotAuth string
	for _, h := range client.calls[1].Headers {
		if h.Key == "Authorization" {
			gotAuth = h.Value
		}
	}
	if gotAuth != "Bearer chained-xyz" {
		t.Errorf("second request's Authorization header = %q, want %q (chaining across the run)", gotAuth, "Bearer chained-xyz")
	}
	if finalCtx.Environment["token"] != "chained-xyz" {
		t.Errorf("final context token = %q, want chained-xyz", finalCtx.Environment["token"])
	}
}
