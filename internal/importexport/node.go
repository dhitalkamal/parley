// Package importexport converts between collection.Request/collections and
// external formats: curl commands, Postman Collection v2.1, and OpenAPI 3.
// Every importer produces, and the exporter consumes, the same
// format-agnostic ImportedNode tree - format parsers/generators stay pure
// and store-free; only persist.go touches collection.Store.
package importexport

import collection "github.com/dhitalkamal/parley/internal/collection/domain"

// ImportedNode is a parsed folder/request tree, ready to persist via a
// collection.Store or to build back into an export format. Request is nil for
// folder nodes.
type ImportedNode struct {
	Name     string
	Request  *collection.Request
	Children []ImportedNode
}
