package importexport

import (
	"fmt"
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	"net/url"
	"path"
	"strings"

	"gopkg.in/yaml.v3"
)

// ParseAny auto-detects an import's format from its content and dispatches
// to the matching parser: a curl command (starts with "curl"), an OpenAPI 3
// spec (JSON or YAML, has an "openapi" or "swagger" key), or a Postman v2.1
// collection (has both "info" and "item" keys). yaml.Unmarshal is used to
// sniff the keys since it also accepts JSON object syntax, so one probe
// covers both formats.
func ParseAny(input string) (ImportedNode, error) {
	trimmed := strings.TrimSpace(input)
	if strings.HasPrefix(strings.ToLower(trimmed), "curl") {
		req, err := ParseCurl(trimmed)
		if err != nil {
			return ImportedNode{}, err
		}
		return ImportedNode{Name: importNameFromRequest(req), Request: &req}, nil
	}

	var probe map[string]interface{}
	if err := yaml.Unmarshal([]byte(trimmed), &probe); err == nil {
		if _, ok := probe["openapi"]; ok {
			return ParseOpenAPICollection([]byte(trimmed))
		}
		if _, ok := probe["swagger"]; ok {
			return ParseOpenAPICollection([]byte(trimmed))
		}
		if _, hasInfo := probe["info"]; hasInfo {
			if _, hasItem := probe["item"]; hasItem {
				return ParsePostmanCollection([]byte(trimmed))
			}
		}
	}

	return ImportedNode{}, fmt.Errorf("unrecognized import format (expected a curl command, a Postman v2.1 collection, or an OpenAPI 3 spec)")
}

// importNameFromRequest derives a reasonable request name from a curl
// import, which has no name of its own.
func importNameFromRequest(req collection.Request) string {
	if u, err := url.Parse(req.URL); err == nil {
		if base := path.Base(u.Path); base != "" && base != "/" && base != "." {
			return base
		}
		if u.Host != "" {
			return u.Host
		}
	}
	return "imported-request"
}
