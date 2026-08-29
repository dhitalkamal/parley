package importexport

import (
	"encoding/json"
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	"strings"
	"testing"
)

const samplePostmanCollection = `{
  "info": {
    "name": "Sample",
    "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
  },
  "item": [
    {
      "name": "get-user",
      "request": {
        "method": "GET",
        "header": [
          {"key": "Authorization", "value": "Bearer xyz", "disabled": false},
          {"key": "X-Off", "value": "nope", "disabled": true}
        ],
        "url": {
          "raw": "https://api.example.com/users?page=2",
          "query": [{"key": "page", "value": "2", "disabled": false}]
        }
      },
      "event": [
        {"listen": "prerequest", "script": {"exec": ["pm.environment.set(\"a\", \"1\");"]}},
        {"listen": "test", "script": {"exec": ["pm.test(\"ok\", function () {});"]}}
      ]
    },
    {
      "name": "users",
      "item": [
        {
          "name": "create-user",
          "request": {
            "method": "POST",
            "header": [{"key": "Content-Type", "value": "application/json", "disabled": false}],
            "url": {"raw": "https://api.example.com/users"},
            "body": {"mode": "raw", "raw": "{\"name\":\"ada\"}"}
          }
        }
      ]
    }
  ]
}`

func TestParsePostmanCollection_TopLevelRequest(t *testing.T) {
	root, err := ParsePostmanCollection([]byte(samplePostmanCollection))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(root.Children) != 2 {
		t.Fatalf("got %d children, want 2: %+v", len(root.Children), root.Children)
	}
	first := root.Children[0]
	if first.Name != "get-user" || first.Request == nil {
		t.Fatalf("first child = %+v", first)
	}
	if first.Request.Method != collection.GET {
		t.Errorf("method = %s, want GET", first.Request.Method)
	}
	if first.Request.URL != "https://api.example.com/users" {
		t.Errorf("url = %q", first.Request.URL)
	}
	if len(first.Request.Params) != 1 || first.Request.Params[0].Value != "2" {
		t.Errorf("params = %+v", first.Request.Params)
	}
}

func TestParsePostmanCollection_DisabledHeaderMapsToEnabledFalse(t *testing.T) {
	root, err := ParsePostmanCollection([]byte(samplePostmanCollection))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	headers := root.Children[0].Request.Headers
	var off *collection.Header
	for i, h := range headers {
		if h.Key == "X-Off" {
			off = &headers[i]
		}
	}
	if off == nil {
		t.Fatal("expected X-Off header to be present")
	}
	if off.Enabled {
		t.Error("expected disabled Postman header to map to Enabled=false")
	}
}

func TestParsePostmanCollection_ScriptsFromEvents(t *testing.T) {
	root, err := ParsePostmanCollection([]byte(samplePostmanCollection))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	req := root.Children[0].Request
	if !strings.Contains(req.PreRequestScript, `pm.environment.set`) {
		t.Errorf("preRequestScript = %q", req.PreRequestScript)
	}
	if !strings.Contains(req.TestScript, `pm.test`) {
		t.Errorf("testScript = %q", req.TestScript)
	}
}

func TestParsePostmanCollection_NestedFolder(t *testing.T) {
	root, err := ParsePostmanCollection([]byte(samplePostmanCollection))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	folder := root.Children[1]
	if folder.Name != "users" || folder.Request != nil {
		t.Fatalf("folder = %+v", folder)
	}
	if len(folder.Children) != 1 || folder.Children[0].Name != "create-user" {
		t.Fatalf("folder children = %+v", folder.Children)
	}
	if folder.Children[0].Request.Body.RawText != `{"name":"ada"}` {
		t.Errorf("body = %+v", folder.Children[0].Request.Body)
	}
}

func TestParsePostmanCollection_InvalidJSONReturnsError(t *testing.T) {
	_, err := ParsePostmanCollection([]byte("not json"))
	if err == nil {
		t.Fatal("expected an error for invalid JSON")
	}
}

func TestExportPostmanCollection_ProducesValidSchema(t *testing.T) {
	req := collection.Request{Method: collection.GET, URL: "https://api.example.com/users"}
	root := ImportedNode{Children: []ImportedNode{{Name: "get-user", Request: &req}}}

	data, err := ExportPostmanCollection("My Collection", root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var col postmanCollection
	if err := json.Unmarshal(data, &col); err != nil {
		t.Fatalf("exported data doesn't parse as a Postman collection: %v", err)
	}
	if col.Info.Name != "My Collection" {
		t.Errorf("info.name = %q", col.Info.Name)
	}
	if !strings.Contains(col.Info.Schema, "v2.1.0") {
		t.Errorf("schema = %q, want v2.1.0", col.Info.Schema)
	}
}

func TestExportImportRoundTrip_PreservesStructureAndScripts(t *testing.T) {
	leaf := collection.Request{
		Method:           collection.POST,
		URL:              "https://api.example.com/users",
		Params:           []collection.QueryParam{{Key: "page", Value: "2", Enabled: true}},
		Headers:          []collection.Header{{Key: "Authorization", Value: "Bearer xyz", Enabled: true}},
		Body:             collection.Body{Type: collection.BodyRaw, RawText: `{"name":"ada"}`},
		PreRequestScript: "pm.environment.set(\"a\", \"1\");",
		TestScript:       "pm.test(\"ok\", function () {});",
	}
	original := ImportedNode{
		Children: []ImportedNode{
			{
				Name: "users",
				Children: []ImportedNode{
					{Name: "create-user", Request: &leaf},
				},
			},
		},
	}

	data, err := ExportPostmanCollection("Round Trip", original)
	if err != nil {
		t.Fatalf("export error: %v", err)
	}
	imported, err := ParsePostmanCollection(data)
	if err != nil {
		t.Fatalf("import error: %v", err)
	}

	if len(imported.Children) != 1 || imported.Children[0].Name != "users" {
		t.Fatalf("imported = %+v", imported.Children)
	}
	folder := imported.Children[0]
	if len(folder.Children) != 1 || folder.Children[0].Name != "create-user" {
		t.Fatalf("folder children = %+v", folder.Children)
	}
	got := folder.Children[0].Request
	if got.Method != leaf.Method || got.URL != leaf.URL {
		t.Errorf("got %+v, want method/url matching %+v", got, leaf)
	}
	if len(got.Params) != 1 || got.Params[0].Value != "2" {
		t.Errorf("params = %+v", got.Params)
	}
	if got.Body.RawText != leaf.Body.RawText {
		t.Errorf("body = %q, want %q", got.Body.RawText, leaf.Body.RawText)
	}
	if got.PreRequestScript != leaf.PreRequestScript {
		t.Errorf("preRequestScript = %q, want %q", got.PreRequestScript, leaf.PreRequestScript)
	}
	if got.TestScript != leaf.TestScript {
		t.Errorf("testScript = %q, want %q", got.TestScript, leaf.TestScript)
	}
}
