package collectionstore

import (
	"encoding/json"

	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	"github.com/dhitalkamal/parley/internal/platform/fsstore"
)

// encodeRequest marshals a request to the on-disk JSON shape held by fsstore.
func encodeRequest(req collection.Request) ([]byte, error) {
	return json.MarshalIndent(fsstore.RequestToFile(req), "", "  ")
}

// decodeRequest parses the on-disk JSON shape back into a domain request.
func decodeRequest(data []byte) (collection.Request, error) {
	var f fsstore.RequestFile
	if err := json.Unmarshal(data, &f); err != nil {
		return collection.Request{}, err
	}
	return fsstore.RequestFromFile(f), nil
}
