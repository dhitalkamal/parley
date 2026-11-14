package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	collectionstore "github.com/dhitalkamal/parley/internal/collection/infrastructure"
	environment "github.com/dhitalkamal/parley/internal/environment/domain"
	environmentstore "github.com/dhitalkamal/parley/internal/environment/infrastructure"
	httpclient "github.com/dhitalkamal/parley/internal/execution/infrastructure/httpclient"
	scriptengine "github.com/dhitalkamal/parley/internal/scripting/infrastructure"
)

// wantAll fails unless msg contains every fragment.
func wantAll(t *testing.T, msg string, fragments ...string) {
	t.Helper()
	for _, f := range fragments {
		if !strings.Contains(msg, f) {
			t.Errorf("message %q is missing %q", msg, f)
		}
	}
}

func TestResolveMessage_UnknownTopLevelNameListsChoicesAndExplainsMatching(t *testing.T) {
	_, err := ResolveRequestPaths(sampleTree(), "nope")
	if err == nil {
		t.Fatal("expected an error")
	}
	wantAll(t, err.Error(), `"nope"`, "folder and request names", "available here: top, users", "omit the path to run everything")
}

func TestResolveMessage_UnknownNestedNameListsThatLevel(t *testing.T) {
	_, err := ResolveRequestPaths(sampleTree(), "users/nope")
	if err == nil {
		t.Fatal("expected an error")
	}
	wantAll(t, err.Error(), `"nope"`, `"users/nope"`, "available here: list, get")
}

func TestResolveMessage_EmptyFolderNamesTheFolder(t *testing.T) {
	tree := collection.TreeNode{
		Kind:     collection.KindFolder,
		Children: []collection.TreeNode{{Kind: collection.KindFolder, Name: "empty", Path: "010_empty"}},
	}
	_, err := ResolveRequestPaths(tree, "empty")
	if err == nil {
		t.Fatal("expected an error")
	}
	wantAll(t, err.Error(), `folder "empty" has no requests`)
}

func TestResolveMessage_EmptyCollectionSaysHowToAddRequests(t *testing.T) {
	_, err := ResolveRequestPaths(collection.TreeNode{Kind: collection.KindFolder}, "")
	if err == nil {
		t.Fatal("expected an error")
	}
	wantAll(t, err.Error(), "collection has no requests", "parley")
}

func runWithEnv(t *testing.T, envStore *environmentstore.EnvStore, envName string) error {
	t.Helper()
	store := collectionstore.New(t.TempDir())
	if _, err := store.SaveRequest("", "ping", collection.Request{Method: collection.GET, URL: "https://example.com"}); err != nil {
		t.Fatalf("save error: %v", err)
	}
	var out bytes.Buffer
	_, err := Run(context.Background(), Options{
		Store: store, EnvStore: envStore, Client: httpclient.New(),
		Scripts: scriptengine.New(), EnvName: envName, Reporter: "text", Out: &out,
	})
	return err
}

func TestRunMessage_UnknownEnvironmentListsExistingOnes(t *testing.T) {
	envStore := environmentstore.New(t.TempDir())
	if err := envStore.SaveEnvironment(environment.Environment{Name: "dev"}); err != nil {
		t.Fatalf("save env: %v", err)
	}
	err := runWithEnv(t, envStore, "staging")
	if err == nil {
		t.Fatal("expected an error")
	}
	wantAll(t, err.Error(), `no environment named "staging"`, "available: dev")
}

func TestRunMessage_UnknownEnvironmentWithNoneSaved(t *testing.T) {
	err := runWithEnv(t, environmentstore.New(t.TempDir()), "staging")
	if err == nil {
		t.Fatal("expected an error")
	}
	wantAll(t, err.Error(), `no environment named "staging"`, "none exist")
}
