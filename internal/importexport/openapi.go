package importexport

import (
	"bytes"
	"encoding/json"
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// This importer covers the OpenAPI 3 subset that matters for building a
// request collection: servers[0].url as a base, one request per operation
// with its query/header parameters and a best-effort JSON body example.
// $ref is not resolved (a request body using one imports with no body
// rather than failing), and deep schema-to-example synthesis only goes one
// property level deep - real-world specs often need touch-up after import.
type openAPISpec struct {
	Servers []openAPIServer            `json:"servers,omitempty" yaml:"servers,omitempty"`
	Paths   map[string]openAPIPathItem `json:"paths,omitempty" yaml:"paths,omitempty"`
}

type openAPIServer struct {
	URL string `json:"url" yaml:"url"`
}

type openAPIPathItem struct {
	Get     *openAPIOperation `json:"get,omitempty" yaml:"get,omitempty"`
	Post    *openAPIOperation `json:"post,omitempty" yaml:"post,omitempty"`
	Put     *openAPIOperation `json:"put,omitempty" yaml:"put,omitempty"`
	Patch   *openAPIOperation `json:"patch,omitempty" yaml:"patch,omitempty"`
	Delete  *openAPIOperation `json:"delete,omitempty" yaml:"delete,omitempty"`
	Head    *openAPIOperation `json:"head,omitempty" yaml:"head,omitempty"`
	Options *openAPIOperation `json:"options,omitempty" yaml:"options,omitempty"`
}

type openAPIOperation struct {
	OperationID string              `json:"operationId,omitempty" yaml:"operationId,omitempty"`
	Tags        []string            `json:"tags,omitempty" yaml:"tags,omitempty"`
	Parameters  []openAPIParameter  `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	RequestBody *openAPIRequestBody `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
}

type openAPIParameter struct {
	Name string `json:"name" yaml:"name"`
	In   string `json:"in" yaml:"in"`
}

type openAPIRequestBody struct {
	Content map[string]openAPIMediaType `json:"content,omitempty" yaml:"content,omitempty"`
}

type openAPIMediaType struct {
	Schema openAPISchema `json:"schema,omitempty" yaml:"schema,omitempty"`
}

type openAPISchema struct {
	Ref        string                   `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Type       string                   `json:"type,omitempty" yaml:"type,omitempty"`
	Example    interface{}              `json:"example,omitempty" yaml:"example,omitempty"`
	Properties map[string]openAPISchema `json:"properties,omitempty" yaml:"properties,omitempty"`
}

var openAPIPathParam = regexp.MustCompile(`\{([^}]+)\}`)

// ParseOpenAPICollection parses an OpenAPI 3 document (JSON or YAML,
// detected by content) into an ImportedNode tree, one request per
// operation, grouped by each operation's first tag (ungrouped/flat at the
// root if it has none).
func ParseOpenAPICollection(data []byte) (ImportedNode, error) {
	spec, err := decodeOpenAPISpec(data)
	if err != nil {
		return ImportedNode{}, err
	}

	baseURL := ""
	if len(spec.Servers) > 0 {
		baseURL = strings.TrimRight(spec.Servers[0].URL, "/")
	}

	var flat []ImportedNode
	var folderOrder []string
	folders := map[string][]ImportedNode{}

	addOperation := func(method, path string, op *openAPIOperation) {
		if op == nil {
			return
		}
		req := buildOpenAPIRequest(baseURL, method, path, op)
		name := op.OperationID
		if name == "" {
			name = method + " " + path
		}
		leaf := ImportedNode{Name: name, Request: &req}

		if len(op.Tags) == 0 {
			flat = append(flat, leaf)
			return
		}
		tag := op.Tags[0]
		if _, ok := folders[tag]; !ok {
			folderOrder = append(folderOrder, tag)
		}
		folders[tag] = append(folders[tag], leaf)
	}

	for path, item := range spec.Paths {
		addOperation("GET", path, item.Get)
		addOperation("POST", path, item.Post)
		addOperation("PUT", path, item.Put)
		addOperation("PATCH", path, item.Patch)
		addOperation("DELETE", path, item.Delete)
		addOperation("HEAD", path, item.Head)
		addOperation("OPTIONS", path, item.Options)
	}

	root := ImportedNode{Children: flat}
	for _, tag := range folderOrder {
		root.Children = append(root.Children, ImportedNode{Name: tag, Children: folders[tag]})
	}
	return root, nil
}

func decodeOpenAPISpec(data []byte) (openAPISpec, error) {
	var spec openAPISpec
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) > 0 && trimmed[0] == '{' {
		if err := json.Unmarshal(data, &spec); err != nil {
			return openAPISpec{}, err
		}
		return spec, nil
	}
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return openAPISpec{}, err
	}
	return spec, nil
}

func buildOpenAPIRequest(baseURL, method, path string, op *openAPIOperation) collection.Request {
	templatedPath := openAPIPathParam.ReplaceAllString(path, "{{$1}}")
	req := collection.Request{
		Method:          collection.Method(method),
		URL:             baseURL + templatedPath,
		FollowRedirects: true,
		Body:            collection.Body{Type: collection.BodyNone},
	}
	for _, p := range op.Parameters {
		switch p.In {
		case "query":
			req.Params = append(req.Params, collection.QueryParam{Key: p.Name, Enabled: true})
		case "header":
			req.Headers = append(req.Headers, collection.Header{Key: p.Name, Enabled: true})
		}
	}
	if op.RequestBody != nil {
		if mt, ok := op.RequestBody.Content["application/json"]; ok {
			if example := openAPIExample(mt.Schema); example != "" {
				req.Body = collection.Body{Type: collection.BodyRaw, RawContentType: "application/json", RawText: example}
			}
		}
	}
	return req
}

func openAPIExample(schema openAPISchema) string {
	if schema.Ref != "" {
		return ""
	}
	if schema.Example != nil {
		if b, err := json.MarshalIndent(schema.Example, "", "  "); err == nil {
			return string(b)
		}
	}
	if schema.Type == "object" && len(schema.Properties) > 0 {
		example := make(map[string]interface{}, len(schema.Properties))
		for key, prop := range schema.Properties {
			example[key] = openAPIZeroHint(prop.Type)
		}
		if b, err := json.MarshalIndent(example, "", "  "); err == nil {
			return string(b)
		}
	}
	return ""
}

func openAPIZeroHint(t string) interface{} {
	switch t {
	case "integer", "number":
		return 0
	case "boolean":
		return false
	case "array":
		return []interface{}{}
	case "object":
		return map[string]interface{}{}
	default:
		return ""
	}
}
