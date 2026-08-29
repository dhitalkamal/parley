package importexport

import (
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	"strings"
	"testing"
)

const sampleOpenAPIJSON = `{
  "openapi": "3.0.0",
  "servers": [{"url": "https://api.example.com"}],
  "paths": {
    "/users/{id}": {
      "get": {
        "operationId": "getUser",
        "tags": ["Users"],
        "parameters": [
          {"name": "id", "in": "path", "required": true},
          {"name": "verbose", "in": "query"},
          {"name": "X-Trace", "in": "header"}
        ]
      },
      "post": {
        "tags": ["Users"],
        "requestBody": {
          "content": {
            "application/json": {
              "schema": {"example": {"name": "ada"}}
            }
          }
        }
      }
    },
    "/ping": {
      "get": {"operationId": "ping"}
    }
  }
}`

const sampleOpenAPIYAML = `
openapi: 3.0.0
servers:
  - url: https://api.example.com
paths:
  /users/{id}:
    get:
      operationId: getUser
      tags: [Users]
      parameters:
        - name: id
          in: path
          required: true
        - name: verbose
          in: query
        - name: X-Trace
          in: header
    post:
      tags: [Users]
      requestBody:
        content:
          application/json:
            schema:
              example:
                name: ada
  /ping:
    get:
      operationId: ping
`

func findChild(nodes []ImportedNode, name string) (ImportedNode, bool) {
	for _, n := range nodes {
		if n.Name == name {
			return n, true
		}
	}
	return ImportedNode{}, false
}

func TestParseOpenAPICollection_JSON_PathParamsAndGrouping(t *testing.T) {
	root, err := ParseOpenAPICollection([]byte(sampleOpenAPIJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	usersFolder, ok := findChild(root.Children, "Users")
	if !ok {
		t.Fatalf("expected a Users folder, got %+v", root.Children)
	}
	getUser, ok := findChild(usersFolder.Children, "getUser")
	if !ok {
		t.Fatalf("expected getUser request, got %+v", usersFolder.Children)
	}
	if getUser.Request.Method != collection.GET {
		t.Errorf("method = %s, want GET", getUser.Request.Method)
	}
	if getUser.Request.URL != "https://api.example.com/users/{{id}}" {
		t.Errorf("url = %q, want https://api.example.com/users/{{id}}", getUser.Request.URL)
	}

	var queryFound, headerFound bool
	for _, p := range getUser.Request.Params {
		if p.Key == "verbose" {
			queryFound = true
		}
	}
	for _, h := range getUser.Request.Headers {
		if h.Key == "X-Trace" {
			headerFound = true
		}
	}
	if !queryFound {
		t.Errorf("expected verbose query param, got %+v", getUser.Request.Params)
	}
	if !headerFound {
		t.Errorf("expected X-Trace header, got %+v", getUser.Request.Headers)
	}

	pingLeaf, ok := findChild(root.Children, "ping")
	if !ok || pingLeaf.Request == nil || pingLeaf.Request.Method != collection.GET {
		t.Errorf("expected an ungrouped, untagged ping request at root, got %+v", root.Children)
	}
}

func TestParseOpenAPICollection_RequestBodyExampleFromSchema(t *testing.T) {
	root, err := ParseOpenAPICollection([]byte(sampleOpenAPIJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	usersFolder, _ := findChild(root.Children, "Users")
	postUser, ok := findChild(usersFolder.Children, "POST /users/{id}")
	if !ok {
		t.Fatalf("expected an unnamed POST operation, got %+v", usersFolder.Children)
	}
	if postUser.Request.Body.Type != collection.BodyRaw || !strings.Contains(postUser.Request.Body.RawText, "ada") {
		t.Errorf("body = %+v", postUser.Request.Body)
	}
}

func TestParseOpenAPICollection_YAMLProducesEquivalentTree(t *testing.T) {
	jsonRoot, err := ParseOpenAPICollection([]byte(sampleOpenAPIJSON))
	if err != nil {
		t.Fatalf("json parse error: %v", err)
	}
	yamlRoot, err := ParseOpenAPICollection([]byte(sampleOpenAPIYAML))
	if err != nil {
		t.Fatalf("yaml parse error: %v", err)
	}

	jsonUsers, _ := findChild(jsonRoot.Children, "Users")
	yamlUsers, _ := findChild(yamlRoot.Children, "Users")
	jsonGetUser, _ := findChild(jsonUsers.Children, "getUser")
	yamlGetUser, _ := findChild(yamlUsers.Children, "getUser")
	if jsonGetUser.Request.URL != yamlGetUser.Request.URL {
		t.Errorf("url mismatch: json=%q yaml=%q", jsonGetUser.Request.URL, yamlGetUser.Request.URL)
	}
	if jsonGetUser.Request.Method != yamlGetUser.Request.Method {
		t.Errorf("method mismatch: json=%s yaml=%s", jsonGetUser.Request.Method, yamlGetUser.Request.Method)
	}
}

func TestParseOpenAPICollection_RefSchemaDoesNotPanic(t *testing.T) {
	spec := `{
		"paths": {
			"/things": {
				"post": {
					"requestBody": {
						"content": {
							"application/json": {"schema": {"$ref": "#/components/schemas/Thing"}}
						}
					}
				}
			}
		}
	}`
	root, err := ParseOpenAPICollection([]byte(spec))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	leaf, ok := findChild(root.Children, "POST /things")
	if !ok {
		t.Fatalf("expected the operation to still import, got %+v", root.Children)
	}
	if leaf.Request.Body.Type != collection.BodyNone {
		t.Errorf("expected no body synthesized for a $ref schema, got %+v", leaf.Request.Body)
	}
}

func TestParseOpenAPICollection_InvalidInputReturnsErrorNotPanic(t *testing.T) {
	_, err := ParseOpenAPICollection([]byte("{not valid json or yaml: ][["))
	if err == nil {
		t.Fatal("expected an error for malformed input")
	}
}
