package httpclient

import (
	"bytes"
	"encoding/json"
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	"io"
	"mime/multipart"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// buildRequestBody turns a collection.Body into the io.Reader http.NewRequest
// needs plus the Content-Type it implies - callers only set that header
// when the request doesn't already have one, same as BodyRaw's existing
// RawContentType. BodyNone yields an empty reader and no content type.
func buildRequestBody(body collection.Body) (io.Reader, string, error) {
	switch body.Type {
	case collection.BodyRaw:
		return strings.NewReader(body.RawText), body.RawContentType, nil
	case collection.BodyURLEncoded:
		return buildURLEncodedBody(body.FormFields), "application/x-www-form-urlencoded", nil
	case collection.BodyMultipart:
		return buildMultipartBody(body.FormFields)
	case collection.BodyGraphQL:
		return buildGraphQLBody(body.GraphQLQuery, body.GraphQLVariables)
	case collection.BodyBinary:
		return buildBinaryBody(body.BinaryFilePath)
	default:
		return strings.NewReader(""), "", nil
	}
}

func buildURLEncodedBody(fields []collection.BodyFormField) io.Reader {
	values := url.Values{}
	for _, f := range fields {
		if f.Enabled {
			values.Add(f.Key, f.Value)
		}
	}
	return strings.NewReader(values.Encode())
}

// buildMultipartBody writes enabled fields in order, opening each file field
// from disk as it goes rather than up front - a missing file fails fast
// with a clear error instead of silently sending a partial body.
func buildMultipartBody(fields []collection.BodyFormField) (io.Reader, string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for _, f := range fields {
		if !f.Enabled {
			continue
		}
		if !f.IsFile {
			if err := w.WriteField(f.Key, f.Value); err != nil {
				return nil, "", err
			}
			continue
		}
		file, err := os.Open(f.FilePath)
		if err != nil {
			return nil, "", err
		}
		part, err := w.CreateFormFile(f.Key, filepath.Base(f.FilePath))
		if err != nil {
			file.Close()
			return nil, "", err
		}
		if _, err := io.Copy(part, file); err != nil {
			file.Close()
			return nil, "", err
		}
		file.Close()
	}
	if err := w.Close(); err != nil {
		return nil, "", err
	}
	return &buf, w.FormDataContentType(), nil
}

// buildGraphQLBody wraps the query/variables pair in the {query, variables}
// envelope every GraphQL HTTP endpoint expects. Invalid or empty variables
// JSON is sent as an empty query with no variables key, matching a GraphQL
// query with no variables, rather than failing the whole request over a
// typo in a JSON textarea.
func buildGraphQLBody(query, variablesJSON string) (io.Reader, string, error) {
	payload := map[string]any{"query": query}
	if strings.TrimSpace(variablesJSON) != "" {
		var vars any
		if err := json.Unmarshal([]byte(variablesJSON), &vars); err == nil {
			payload["variables"] = vars
		}
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, "", err
	}
	return bytes.NewReader(data), "application/json", nil
}

func buildBinaryBody(path string) (io.Reader, string, error) {
	if path == "" {
		return strings.NewReader(""), "", nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	return bytes.NewReader(data), "application/octet-stream", nil
}
