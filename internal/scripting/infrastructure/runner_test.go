package scriptengine

import (
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
	scripting "github.com/dhitalkamal/parley/internal/scripting/domain"
	"strings"
	"testing"
	"time"
)

func TestRunPreRequest_EmptyScriptIsNoOp(t *testing.T) {
	r := New()
	req := collection.Request{Method: collection.GET, URL: "https://example.com"}
	got, _, err := r.RunPreRequest("", req, scripting.ScriptContext{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.URL != req.URL {
		t.Errorf("got %+v, want unchanged", got)
	}
}

func TestRunPreRequest_SetsEnvironmentVariable(t *testing.T) {
	r := New()
	req := collection.Request{Method: collection.GET, URL: "https://example.com"}
	ctx := scripting.ScriptContext{Environment: map[string]string{}}

	_, newCtx, err := r.RunPreRequest(`pm.environment.set("token", "abc123");`, req, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newCtx.Environment["token"] != "abc123" {
		t.Errorf("environment = %v, want token=abc123", newCtx.Environment)
	}
}

func TestRunPreRequest_ReadsExistingVariable(t *testing.T) {
	r := New()
	req := collection.Request{Method: collection.GET, URL: "https://example.com"}
	ctx := scripting.ScriptContext{
		Environment: map[string]string{"envKey": "envVal"},
		Globals:     map[string]string{"globalKey": "globalVal"},
	}

	_, newCtx, err := r.RunPreRequest(
		`pm.environment.set("gotEnv", pm.environment.get("envKey"));
		 pm.environment.set("gotGlobal", pm.globals.get("globalKey"));
		 pm.environment.set("gotMerged", pm.variables.get("globalKey"));`,
		req, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newCtx.Environment["gotEnv"] != "envVal" {
		t.Errorf("gotEnv = %q", newCtx.Environment["gotEnv"])
	}
	if newCtx.Environment["gotGlobal"] != "globalVal" {
		t.Errorf("gotGlobal = %q", newCtx.Environment["gotGlobal"])
	}
	if newCtx.Environment["gotMerged"] != "globalVal" {
		t.Errorf("gotMerged (via pm.variables) = %q", newCtx.Environment["gotMerged"])
	}
}

func TestRunPreRequest_AddsComputedHeader(t *testing.T) {
	r := New()
	req := collection.Request{Method: collection.GET, URL: "https://example.com"}

	newReq, _, err := r.RunPreRequest(`pm.request.headers.add("X-Computed", "hello-" + pm.request.method);`, req, scripting.ScriptContext{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, h := range newReq.Headers {
		if h.Key == "X-Computed" && h.Value == "hello-GET" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected computed header, got %+v", newReq.Headers)
	}
}

func TestRunPreRequest_OverwritesExistingHeaderCaseInsensitively(t *testing.T) {
	r := New()
	req := collection.Request{
		Method:  collection.GET,
		URL:     "https://example.com",
		Headers: []collection.Header{{Key: "Authorization", Value: "old", Enabled: true}},
	}
	newReq, _, err := r.RunPreRequest(`pm.request.headers.add("authorization", "Bearer new");`, req, scripting.ScriptContext{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(newReq.Headers) != 1 || newReq.Headers[0].Value != "Bearer new" {
		t.Errorf("got %+v, want single overwritten header", newReq.Headers)
	}
}

func TestRunPreRequest_SyntaxErrorReturnsGoErrorNotPanic(t *testing.T) {
	r := New()
	req := collection.Request{Method: collection.GET, URL: "https://example.com"}
	_, _, err := r.RunPreRequest(`this is not valid javascript {{{`, req, scripting.ScriptContext{})
	if err == nil {
		t.Fatal("expected an error for invalid script syntax")
	}
}

func TestRunPreRequest_RuntimeExceptionReturnsGoError(t *testing.T) {
	r := New()
	req := collection.Request{Method: collection.GET, URL: "https://example.com"}
	_, _, err := r.RunPreRequest(`throw new Error("boom");`, req, scripting.ScriptContext{})
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("error = %v, want it to mention 'boom'", err)
	}
}

func TestRunPreRequest_InfiniteLoopIsInterrupted(t *testing.T) {
	r := New()
	req := collection.Request{Method: collection.GET, URL: "https://example.com"}

	done := make(chan error, 1)
	go func() {
		_, _, err := r.RunPreRequest(`while (true) {}`, req, scripting.ScriptContext{})
		done <- err
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Error("expected an interrupt error for an infinite loop")
		}
	case <-time.After(8 * time.Second):
		t.Fatal("script was not interrupted within the watchdog window")
	}
}

func TestRunTest_PassingAssertionRecordsPass(t *testing.T) {
	r := New()
	resp := execution.Response{StatusCode: 200, Status: "200 OK"}
	results, _, err := r.RunTest(
		`pm.test("status is 200", function () { pm.expect(pm.response.code).to.equal(200); });`,
		collection.Request{}, resp, scripting.ScriptContext{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || !results[0].Passed || results[0].Name != "status is 200" {
		t.Errorf("got %+v", results)
	}
}

func TestRunTest_FailingAssertionRecordsFailureWithMessage(t *testing.T) {
	r := New()
	resp := execution.Response{StatusCode: 500, Status: "500 Internal Server Error"}
	results, _, err := r.RunTest(
		`pm.test("status is 200", function () { pm.expect(pm.response.code).to.equal(200); });`,
		collection.Request{}, resp, scripting.ScriptContext{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || results[0].Passed {
		t.Fatalf("got %+v, want a failed test", results)
	}
	if results[0].Error == "" {
		t.Error("expected a non-empty failure message")
	}
}

func TestRunTest_MultipleTestsAllRecorded(t *testing.T) {
	r := New()
	resp := execution.Response{StatusCode: 200}
	results, _, err := r.RunTest(
		`pm.test("a", function () { pm.expect(1).to.equal(1); });
		 pm.test("b", function () { pm.expect(1).to.equal(2); });`,
		collection.Request{}, resp, scripting.ScriptContext{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 || !results[0].Passed || results[1].Passed {
		t.Errorf("got %+v", results)
	}
}

func TestRunTest_ResponseJSONAccessible(t *testing.T) {
	r := New()
	resp := execution.Response{StatusCode: 200, Body: []byte(`{"token":"xyz","count":3}`)}
	results, newCtx, err := r.RunTest(
		`pm.environment.set("token", pm.response.json().token);
		 pm.test("count is 3", function () { pm.expect(pm.response.json().count).to.equal(3); });`,
		collection.Request{}, resp, scripting.ScriptContext{Environment: map[string]string{}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newCtx.Environment["token"] != "xyz" {
		t.Errorf("environment = %v, want token=xyz (chaining example from the brief)", newCtx.Environment)
	}
	if len(results) != 1 || !results[0].Passed {
		t.Errorf("got %+v", results)
	}
}

func TestRunTest_InvalidJSONBodyFailsTestInsteadOfCrashing(t *testing.T) {
	r := New()
	resp := execution.Response{StatusCode: 200, Body: []byte("not json")}
	results, _, err := r.RunTest(
		`pm.test("parses", function () { pm.response.json(); });`,
		collection.Request{}, resp, scripting.ScriptContext{})
	if err != nil {
		t.Fatalf("unexpected Go error (should be a failed test, not a script error): %v", err)
	}
	if len(results) != 1 || results[0].Passed {
		t.Fatalf("got %+v, want a failed test for invalid JSON", results)
	}
}

func TestRunTest_ResponseHeadersAccessible(t *testing.T) {
	r := New()
	resp := execution.Response{StatusCode: 200, Headers: []collection.Header{{Key: "X-Custom", Value: "hello", Enabled: true}}}
	results, _, err := r.RunTest(
		`pm.test("header present", function () { pm.expect(pm.response.headers.get("X-Custom")).to.equal("hello"); });`,
		collection.Request{}, resp, scripting.ScriptContext{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || !results[0].Passed {
		t.Errorf("got %+v", results)
	}
}

func TestRunTest_EmptyScriptReturnsNoResults(t *testing.T) {
	r := New()
	results, _, err := r.RunTest("", collection.Request{}, execution.Response{}, scripting.ScriptContext{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("got %+v, want none", results)
	}
}

func TestExpect_NotNegatesAssertion(t *testing.T) {
	r := New()
	resp := execution.Response{StatusCode: 200}
	results, _, err := r.RunTest(
		`pm.test("not equal", function () { pm.expect(pm.response.code).not.to.equal(500); });`,
		collection.Request{}, resp, scripting.ScriptContext{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || !results[0].Passed {
		t.Errorf("got %+v", results)
	}
}
