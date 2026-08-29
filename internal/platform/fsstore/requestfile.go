package fsstore

import (
	"time"

	collection "github.com/dhitalkamal/parley/internal/collection/domain"
)

// RequestFile is the on-disk JSON shape for a saved request. It's kept
// separate from collection.Request so storage concerns (JSON tags, seconds
// instead of a raw time.Duration) don't leak into the domain layer.
type RequestFile struct {
	// Protocol is omitempty so HTTP requests (the vast majority, and every file
	// written before WebSocket support) serialize byte-for-byte as before and
	// load back as HTTP.
	Protocol string `json:"protocol,omitempty"`
	// WSPeer is omitempty so single-connection WS requests and all HTTP
	// requests serialize exactly as before.
	WSPeer             *wsEndpointFile  `json:"wsPeer,omitempty"`
	Method             string           `json:"method"`
	URL                string           `json:"url"`
	Params             []KVFile         `json:"params,omitempty"`
	Headers            []KVFile         `json:"headers,omitempty"`
	Body               bodyFile         `json:"body"`
	TimeoutSeconds     float64          `json:"timeoutSeconds,omitempty"`
	FollowRedirects    bool             `json:"followRedirects"`
	InsecureSkipVerify bool             `json:"insecureSkipVerify,omitempty"`
	PreRequestScript   string           `json:"preRequestScript,omitempty"`
	TestScript         string           `json:"testScript,omitempty"`
	AuthCapture        *authCaptureFile `json:"authCapture,omitempty"`
	Refresh            *refreshFile     `json:"refresh,omitempty"`
}

type authCaptureFile struct {
	TokenField     string `json:"tokenField"`
	TokenVar       string `json:"tokenVar"`
	ExpiresInField string `json:"expiresInField,omitempty"`
	ExpiresAtVar   string `json:"expiresAtVar,omitempty"`
}

type refreshFile struct {
	RequestPath  string `json:"requestPath"`
	ExpiresAtVar string `json:"expiresAtVar,omitempty"`
}

type wsEndpointFile struct {
	URL     string   `json:"url"`
	Headers []KVFile `json:"headers,omitempty"`
}

// KVFile is the on-disk shape for a key/value pair with an enabled toggle,
// shared by request params/headers and example headers. Description is
// omitted from example headers today (RequestToFile/RequestFromFile below
// never set it there) - only request params/headers carry one.
type KVFile struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Enabled     bool   `json:"enabled"`
	Description string `json:"description,omitempty"`
}

type bodyFile struct {
	Type             string          `json:"type"`
	RawContentType   string          `json:"rawContentType,omitempty"`
	RawText          string          `json:"rawText,omitempty"`
	FormFields       []formFieldFile `json:"formFields,omitempty"`
	GraphQLQuery     string          `json:"graphqlQuery,omitempty"`
	GraphQLVariables string          `json:"graphqlVariables,omitempty"`
	BinaryFilePath   string          `json:"binaryFilePath,omitempty"`
}

type formFieldFile struct {
	Key      string `json:"key"`
	Value    string `json:"value,omitempty"`
	FilePath string `json:"filePath,omitempty"`
	IsFile   bool   `json:"isFile,omitempty"`
	Enabled  bool   `json:"enabled"`
}

// RequestToFile converts a domain request into its on-disk shape.
func RequestToFile(req collection.Request) RequestFile {
	f := RequestFile{
		Protocol: string(req.Protocol),
		Method:   string(req.Method),
		URL:      req.URL,
		Body: bodyFile{
			Type:             string(req.Body.Type),
			RawContentType:   req.Body.RawContentType,
			RawText:          req.Body.RawText,
			GraphQLQuery:     req.Body.GraphQLQuery,
			GraphQLVariables: req.Body.GraphQLVariables,
			BinaryFilePath:   req.Body.BinaryFilePath,
		},
		TimeoutSeconds:     req.Timeout.Seconds(),
		FollowRedirects:    req.FollowRedirects,
		InsecureSkipVerify: req.InsecureSkipVerify,
		PreRequestScript:   req.PreRequestScript,
		TestScript:         req.TestScript,
	}
	if req.AuthCapture.Enabled() {
		f.AuthCapture = &authCaptureFile{
			TokenField:     req.AuthCapture.TokenField,
			TokenVar:       req.AuthCapture.TokenVar,
			ExpiresInField: req.AuthCapture.ExpiresInField,
			ExpiresAtVar:   req.AuthCapture.ExpiresAtVar,
		}
	}
	if req.Refresh.Enabled() {
		f.Refresh = &refreshFile{RequestPath: req.Refresh.RequestPath, ExpiresAtVar: req.Refresh.ExpiresAtVar}
	}
	if req.WSPeer != nil {
		peer := &wsEndpointFile{URL: req.WSPeer.URL}
		for _, h := range req.WSPeer.Headers {
			peer.Headers = append(peer.Headers, KVFile{Key: h.Key, Value: h.Value, Enabled: h.Enabled, Description: h.Description})
		}
		f.WSPeer = peer
	}
	for _, p := range req.Params {
		f.Params = append(f.Params, KVFile{Key: p.Key, Value: p.Value, Enabled: p.Enabled, Description: p.Description})
	}
	for _, h := range req.Headers {
		f.Headers = append(f.Headers, KVFile{Key: h.Key, Value: h.Value, Enabled: h.Enabled, Description: h.Description})
	}
	for _, ff := range req.Body.FormFields {
		f.Body.FormFields = append(f.Body.FormFields, formFieldFile{
			Key: ff.Key, Value: ff.Value, FilePath: ff.FilePath, IsFile: ff.IsFile, Enabled: ff.Enabled,
		})
	}
	return f
}

// RequestFromFile converts an on-disk request shape back into a domain request.
func RequestFromFile(f RequestFile) collection.Request {
	req := collection.Request{
		Protocol: collection.Protocol(f.Protocol),
		Method:   collection.Method(f.Method),
		URL:      f.URL,
		Body: collection.Body{
			Type:             collection.BodyType(f.Body.Type),
			RawContentType:   f.Body.RawContentType,
			RawText:          f.Body.RawText,
			GraphQLQuery:     f.Body.GraphQLQuery,
			GraphQLVariables: f.Body.GraphQLVariables,
			BinaryFilePath:   f.Body.BinaryFilePath,
		},
		Timeout:            time.Duration(f.TimeoutSeconds * float64(time.Second)),
		FollowRedirects:    f.FollowRedirects,
		InsecureSkipVerify: f.InsecureSkipVerify,
		PreRequestScript:   f.PreRequestScript,
		TestScript:         f.TestScript,
	}
	if f.AuthCapture != nil {
		req.AuthCapture = collection.AuthCapture{
			TokenField:     f.AuthCapture.TokenField,
			TokenVar:       f.AuthCapture.TokenVar,
			ExpiresInField: f.AuthCapture.ExpiresInField,
			ExpiresAtVar:   f.AuthCapture.ExpiresAtVar,
		}
	}
	if f.Refresh != nil {
		req.Refresh = collection.RefreshConfig{RequestPath: f.Refresh.RequestPath, ExpiresAtVar: f.Refresh.ExpiresAtVar}
	}
	if f.WSPeer != nil {
		peer := &collection.WSEndpoint{URL: f.WSPeer.URL}
		for _, h := range f.WSPeer.Headers {
			peer.Headers = append(peer.Headers, collection.Header{Key: h.Key, Value: h.Value, Enabled: h.Enabled, Description: h.Description})
		}
		req.WSPeer = peer
	}
	for _, p := range f.Params {
		req.Params = append(req.Params, collection.QueryParam{Key: p.Key, Value: p.Value, Enabled: p.Enabled, Description: p.Description})
	}
	for _, h := range f.Headers {
		req.Headers = append(req.Headers, collection.Header{Key: h.Key, Value: h.Value, Enabled: h.Enabled, Description: h.Description})
	}
	for _, ff := range f.Body.FormFields {
		req.Body.FormFields = append(req.Body.FormFields, collection.BodyFormField{
			Key: ff.Key, Value: ff.Value, FilePath: ff.FilePath, IsFile: ff.IsFile, Enabled: ff.Enabled,
		})
	}
	return req
}
