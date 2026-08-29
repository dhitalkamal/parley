package cli

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	collectionstore "github.com/dhitalkamal/parley/internal/collection/infrastructure"
	environment "github.com/dhitalkamal/parley/internal/environment/domain"
	environmentstore "github.com/dhitalkamal/parley/internal/environment/infrastructure"
	httpclient "github.com/dhitalkamal/parley/internal/execution/infrastructure/httpclient"
	historystore "github.com/dhitalkamal/parley/internal/history/infrastructure"
	scriptengine "github.com/dhitalkamal/parley/internal/scripting/infrastructure"
)

func TestRun_ExitsZeroAndReportsPassWhenTestScriptPasses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	store := collectionstore.New(t.TempDir())
	if _, err := store.SaveRequest("", "ping", collection.Request{
		Method:     collection.GET,
		URL:        server.URL,
		TestScript: `pm.test("status is 200", function () { pm.expect(pm.response.code).to.equal(200); });`,
	}); err != nil {
		t.Fatalf("save error: %v", err)
	}

	var out bytes.Buffer
	code, err := Run(context.Background(), Options{
		Store:    store,
		Client:   httpclient.New(),
		Scripts:  scriptengine.New(),
		Reporter: "text",
		Out:      &out,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 0 {
		t.Errorf("exit code = %d, want 0\noutput:\n%s", code, out.String())
	}
	if !strings.Contains(out.String(), "PASS") {
		t.Errorf("expected PASS in output, got:\n%s", out.String())
	}
}

func TestRun_ExitsOneWhenATestScriptFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	store := collectionstore.New(t.TempDir())
	if _, err := store.SaveRequest("", "ping", collection.Request{
		Method:     collection.GET,
		URL:        server.URL,
		TestScript: `pm.test("status is 200", function () { pm.expect(pm.response.code).to.equal(200); });`,
	}); err != nil {
		t.Fatalf("save error: %v", err)
	}

	var out bytes.Buffer
	code, err := Run(context.Background(), Options{
		Store:    store,
		Client:   httpclient.New(),
		Scripts:  scriptengine.New(),
		Reporter: "text",
		Out:      &out,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 1 {
		t.Errorf("exit code = %d, want 1\noutput:\n%s", code, out.String())
	}
}

func TestRun_UnknownPathReturnsUsageError(t *testing.T) {
	store := collectionstore.New(t.TempDir())
	var out bytes.Buffer
	code, err := Run(context.Background(), Options{
		Store:    store,
		Client:   httpclient.New(),
		Scripts:  scriptengine.New(),
		Path:     "does-not-exist",
		Reporter: "text",
		Out:      &out,
	})
	if err == nil {
		t.Fatal("expected an error for an unresolvable path")
	}
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
}

func TestRun_UnknownReporterReturnsUsageError(t *testing.T) {
	store := collectionstore.New(t.TempDir())
	if _, err := store.SaveRequest("", "ping", collection.Request{Method: collection.GET, URL: "https://example.com"}); err != nil {
		t.Fatalf("save error: %v", err)
	}
	var out bytes.Buffer
	code, err := Run(context.Background(), Options{
		Store:    store,
		Client:   httpclient.New(),
		Scripts:  scriptengine.New(),
		Reporter: "bogus",
		Out:      &out,
	})
	if err == nil {
		t.Fatal("expected an error for an unknown reporter")
	}
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
}

func TestRun_UnknownEnvironmentNameReturnsUsageError(t *testing.T) {
	store := collectionstore.New(t.TempDir())
	if _, err := store.SaveRequest("", "ping", collection.Request{Method: collection.GET, URL: "https://example.com"}); err != nil {
		t.Fatalf("save error: %v", err)
	}
	envStore := environmentstore.New(t.TempDir())
	var out bytes.Buffer
	code, err := Run(context.Background(), Options{
		Store:    store,
		EnvStore: envStore,
		Client:   httpclient.New(),
		Scripts:  scriptengine.New(),
		EnvName:  "staging",
		Reporter: "text",
		Out:      &out,
	})
	if err == nil {
		t.Fatal("expected an error for a nonexistent environment")
	}
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
}

func TestRun_SubstitutesSelectedEnvironmentIntoRequestURL(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	store := collectionstore.New(t.TempDir())
	if _, err := store.SaveRequest("", "ping", collection.Request{
		Method: collection.GET,
		URL:    "{{base}}/ping",
	}); err != nil {
		t.Fatalf("save error: %v", err)
	}

	envDir := t.TempDir()
	envStore := environmentstore.New(envDir)
	if err := envStore.SaveEnvironment(environment.Environment{
		Name:      "staging",
		Variables: []environment.Variable{{Key: "base", Value: server.URL, Enabled: true}},
	}); err != nil {
		t.Fatalf("save environment error: %v", err)
	}

	var out bytes.Buffer
	code, err := Run(context.Background(), Options{
		Store:    store,
		EnvStore: envStore,
		Client:   httpclient.New(),
		Scripts:  scriptengine.New(),
		EnvName:  "staging",
		Reporter: "text",
		Out:      &out,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 0 {
		t.Errorf("exit code = %d, want 0\noutput:\n%s", code, out.String())
	}
	if gotPath != "/ping" {
		t.Errorf("server received path %q, want /ping (env variable was not substituted into the URL)", gotPath)
	}
}

func TestRun_AppendsToRunStoreWhenProvided(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	store := collectionstore.New(t.TempDir())
	if _, err := store.SaveRequest("", "ping", collection.Request{Method: collection.GET, URL: server.URL}); err != nil {
		t.Fatalf("save error: %v", err)
	}
	runStore := historystore.NewRunStore(t.TempDir())

	var out bytes.Buffer
	code, err := Run(context.Background(), Options{
		Store:    store,
		Client:   httpclient.New(),
		Scripts:  scriptengine.New(),
		RunStore: runStore,
		Path:     "ping",
		Reporter: "text",
		Out:      &out,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	runs, err := runStore.ListRuns()
	if err != nil {
		t.Fatalf("list runs error: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("got %d run history entries, want 1", len(runs))
	}
	if runs[0].Path != "ping" {
		t.Errorf("run path = %q, want ping", runs[0].Path)
	}
	if len(runs[0].Results) != 1 || runs[0].Results[0].URL != server.URL {
		t.Errorf("run results = %+v", runs[0].Results)
	}
}

func TestRun_NilRunStoreIsSkippedWithoutError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	store := collectionstore.New(t.TempDir())
	if _, err := store.SaveRequest("", "ping", collection.Request{Method: collection.GET, URL: server.URL}); err != nil {
		t.Fatalf("save error: %v", err)
	}

	var out bytes.Buffer
	code, err := Run(context.Background(), Options{
		Store:    store,
		Client:   httpclient.New(),
		Scripts:  scriptengine.New(),
		Reporter: "text",
		Out:      &out,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
}
