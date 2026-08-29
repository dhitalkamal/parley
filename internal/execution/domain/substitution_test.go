package execution

import (
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	environment "github.com/dhitalkamal/parley/internal/environment/domain"

	"reflect"
	"testing"
)

func TestSubstitute_ReplacesKnownVariable(t *testing.T) {
	got, unresolved := Substitute("{{baseUrl}}/users", map[string]string{"baseUrl": "https://api.example.com"})
	if got != "https://api.example.com/users" {
		t.Errorf("got %q", got)
	}
	if len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}

func TestSubstitute_ToleratesWhitespaceInsideBraces(t *testing.T) {
	got, _ := Substitute("{{ baseUrl }}/users", map[string]string{"baseUrl": "https://api.example.com"})
	if got != "https://api.example.com/users" {
		t.Errorf("got %q", got)
	}
}

func TestSubstitute_ReplacesMultipleOccurrences(t *testing.T) {
	got, _ := Substitute("{{host}}/{{host}}", map[string]string{"host": "x"})
	if got != "x/x" {
		t.Errorf("got %q", got)
	}
}

func TestSubstitute_LeavesUnknownPlaceholderAndReportsIt(t *testing.T) {
	got, unresolved := Substitute("{{missing}}/users", map[string]string{})
	if got != "{{missing}}/users" {
		t.Errorf("got %q, want placeholder left intact", got)
	}
	if !reflect.DeepEqual(unresolved, []string{"missing"}) {
		t.Errorf("unresolved = %v, want [missing]", unresolved)
	}
}

func TestSubstitute_PlainTextUnaffected(t *testing.T) {
	got, unresolved := Substitute("no variables here", nil)
	if got != "no variables here" || len(unresolved) != 0 {
		t.Errorf("got %q, unresolved %v", got, unresolved)
	}
}

func TestResolveVariables_EnvironmentOverridesGlobals(t *testing.T) {
	globals := environment.Environment{Variables: []environment.Variable{
		{Key: "host", Value: "global-host", Enabled: true},
		{Key: "onlyGlobal", Value: "g", Enabled: true},
	}}
	env := environment.Environment{Name: "dev", Variables: []environment.Variable{
		{Key: "host", Value: "dev-host", Enabled: true},
	}}

	resolved := ResolveVariables(globals, env)
	if resolved["host"].Value != "dev-host" {
		t.Errorf("host = %q, want dev-host (env overrides globals)", resolved["host"].Value)
	}
	if resolved["onlyGlobal"].Value != "g" {
		t.Errorf("onlyGlobal = %q, want g", resolved["onlyGlobal"].Value)
	}
}

func TestResolveVariables_SkipsDisabled(t *testing.T) {
	globals := environment.Environment{Variables: []environment.Variable{
		{Key: "off", Value: "x", Enabled: false},
	}}
	resolved := ResolveVariables(globals, environment.Environment{})
	if _, ok := resolved["off"]; ok {
		t.Errorf("expected disabled variable to be excluded, got %+v", resolved["off"])
	}
}

func TestFlattenVariables_ExtractsValues(t *testing.T) {
	vars := map[string]environment.Variable{"a": {Key: "a", Value: "1"}, "b": {Key: "b", Value: "2"}}
	flat := FlattenVariables(vars)
	if flat["a"] != "1" || flat["b"] != "2" {
		t.Errorf("flat = %v", flat)
	}
}

func TestSubstituteRequest_ResolvesAcrossAllFields(t *testing.T) {
	req := collection.Request{
		Method: collection.GET,
		URL:    "{{baseUrl}}/users",
		Params: []collection.QueryParam{{Key: "page", Value: "{{page}}", Enabled: true}},
		Headers: []collection.Header{
			{Key: "Authorization", Value: "Bearer {{token}}", Enabled: true},
		},
		Body: collection.Body{Type: collection.BodyRaw, RawText: `{"id":"{{id}}"}`},
	}
	vars := map[string]environment.Variable{
		"baseUrl": {Value: "https://api.example.com"},
		"page":    {Value: "2"},
		"token":   {Value: "xyz"},
		"id":      {Value: "42"},
	}

	got, unresolved := SubstituteRequest(req, vars)
	if len(unresolved) != 0 {
		t.Fatalf("unresolved = %v, want none", unresolved)
	}
	if got.URL != "https://api.example.com/users" {
		t.Errorf("URL = %q", got.URL)
	}
	if got.Params[0].Value != "2" {
		t.Errorf("param value = %q", got.Params[0].Value)
	}
	if got.Headers[0].Value != "Bearer xyz" {
		t.Errorf("header value = %q", got.Headers[0].Value)
	}
	if got.Body.RawText != `{"id":"42"}` {
		t.Errorf("body = %q", got.Body.RawText)
	}
}

// TestSubstituteRequest_ResolvesFormFieldsAndGraphQL guards the newer body
// types (form-urlencoded/multipart's key/value fields, GraphQL's query and
// variables text) - added alongside collection.BodyRaw, which SubstituteRequest already
// covered. BinaryFilePath is deliberately left out: it's a filesystem path,
// not a templated value, so it isn't substituted.
func TestSubstituteRequest_ResolvesFormFieldsAndGraphQL(t *testing.T) {
	req := collection.Request{
		Body: collection.Body{
			Type: collection.BodyMultipart,
			FormFields: []collection.BodyFormField{
				{Key: "{{fieldKey}}", Value: "{{fieldValue}}", Enabled: true},
			},
			GraphQLQuery:     `query { user(id: "{{id}}") { name } }`,
			GraphQLVariables: `{"id": "{{id}}"}`,
		},
	}
	vars := map[string]environment.Variable{
		"fieldKey":   {Value: "role"},
		"fieldValue": {Value: "admin"},
		"id":         {Value: "42"},
	}

	got, unresolved := SubstituteRequest(req, vars)
	if len(unresolved) != 0 {
		t.Fatalf("unresolved = %v, want none", unresolved)
	}
	if got.Body.FormFields[0].Key != "role" || got.Body.FormFields[0].Value != "admin" {
		t.Errorf("form field = %+v", got.Body.FormFields[0])
	}
	if got.Body.GraphQLQuery != `query { user(id: "42") { name } }` {
		t.Errorf("GraphQLQuery = %q", got.Body.GraphQLQuery)
	}
	if got.Body.GraphQLVariables != `{"id": "42"}` {
		t.Errorf("GraphQLVariables = %q", got.Body.GraphQLVariables)
	}
}

func TestSubstituteRequest_ReportsSortedDedupedUnresolved(t *testing.T) {
	req := collection.Request{
		URL: "{{missing}}/{{missing}}",
		Headers: []collection.Header{
			{Key: "X-A", Value: "{{alsoMissing}}", Enabled: true},
		},
	}
	_, unresolved := SubstituteRequest(req, nil)
	if !reflect.DeepEqual(unresolved, []string{"alsoMissing", "missing"}) {
		t.Errorf("unresolved = %v, want [alsoMissing missing]", unresolved)
	}
}

func TestSubstituteRequest_DoesNotMutateOriginalSlices(t *testing.T) {
	req := collection.Request{
		URL:     "{{x}}",
		Params:  []collection.QueryParam{{Key: "k", Value: "{{x}}", Enabled: true}},
		Headers: []collection.Header{{Key: "h", Value: "{{x}}", Enabled: true}},
	}
	_, _ = SubstituteRequest(req, map[string]environment.Variable{"x": {Value: "resolved"}})
	if req.Params[0].Value != "{{x}}" || req.Headers[0].Value != "{{x}}" {
		t.Errorf("original request was mutated: %+v", req)
	}
}
