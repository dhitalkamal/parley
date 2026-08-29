package collection

import "time"

// Protocol is the wire protocol a request speaks. The empty value means HTTP,
// so every request saved before this field existed (and every plain HTTP one)
// stays HTTP without needing the field set.
type Protocol string

const (
	ProtocolHTTP      Protocol = ""   // default: a normal HTTP request/response
	ProtocolWebSocket Protocol = "ws" // a long-lived ws:// or wss:// connection
)

// WSEndpoint is one side of a WebSocket connection: a URL and its handshake
// headers. A saved WebSocket request's primary endpoint is the Request's own
// URL/Headers (pane A); WSPeer holds an optional second endpoint (pane B) so a
// two-client session can be saved and reopened as a pair.
type WSEndpoint struct {
	URL     string
	Headers []Header
}

// Method is an HTTP request method.
type Method string

const (
	GET     Method = "GET"
	POST    Method = "POST"
	PUT     Method = "PUT"
	PATCH   Method = "PATCH"
	DELETE  Method = "DELETE"
	HEAD    Method = "HEAD"
	OPTIONS Method = "OPTIONS"
)

// Methods lists the methods offered in the editor's method selector, in display order.
var Methods = []Method{GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS}

// IsWebSocket reports whether this request is a WebSocket connection rather
// than a normal HTTP call.
func (r Request) IsWebSocket() bool { return r.Protocol == ProtocolWebSocket }

// QueryParam is one row in the URL's query-param editor.
type QueryParam struct {
	Key         string
	Value       string
	Enabled     bool
	Description string
}

// Header is one row in the header editor.
type Header struct {
	Key         string
	Value       string
	Enabled     bool
	Description string
}

// BodyType selects which body editor is active for a request.
type BodyType string

const (
	BodyNone       BodyType = "none"
	BodyRaw        BodyType = "raw"
	BodyURLEncoded BodyType = "urlencoded"
	BodyMultipart  BodyType = "multipart"
	BodyGraphQL    BodyType = "graphql"
	BodyBinary     BodyType = "binary"
)

// BodyFormField is one row of a URLEncoded or Multipart body. IsFile only
// applies to Multipart (URLEncoded has no notion of a file field) - Value
// holds the field's text value when IsFile is false, and FilePath holds the
// path to the file to upload when it's true.
type BodyFormField struct {
	Key      string
	Value    string
	FilePath string
	IsFile   bool
	Enabled  bool
}

// Body holds the request payload. Only the fields for the active Type are meaningful.
type Body struct {
	Type             BodyType
	RawContentType   string
	RawText          string
	FormFields       []BodyFormField // BodyURLEncoded, BodyMultipart
	GraphQLQuery     string
	GraphQLVariables string
	BinaryFilePath   string
}

// AuthCapture auto-extracts a value out of this request's own response and
// saves it as an environment variable - the structured equivalent of a
// user hand-writing pm.environment.set(...) in a test script (see
// send.go's applyAuthCapture), for the common "login response contains a
// token" case without requiring any JS. TokenField/ExpiresInField are
// dot-paths into the response's parsed JSON body, e.g. "data.token" or
// "expiresIn". Disabled when TokenField is empty - there's no separate
// on/off flag to keep in sync with it.
type AuthCapture struct {
	TokenField     string
	TokenVar       string
	ExpiresInField string // optional: seconds until expiry
	ExpiresAtVar   string // required only if ExpiresInField is set
}

// Enabled reports whether this request should attempt to capture
// anything from its own response after a send.
func (a AuthCapture) Enabled() bool {
	return a.TokenField != "" && a.TokenVar != ""
}

// RefreshesExpiry reports whether ExpiresInField/ExpiresAtVar are both
// configured, so send.go knows whether to also track an expiry alongside
// the captured token.
func (a AuthCapture) TracksExpiry() bool {
	return a.ExpiresInField != "" && a.ExpiresAtVar != ""
}

// RefreshConfig points a request at an existing saved request that
// refreshes its auth token - RequestPath is a path within the same
// collection Store (see Store.LoadRequest), reusing the request
// model instead of a second, separately-typed place to define an HTTP
// call. Disabled when RequestPath is empty.
type RefreshConfig struct {
	RequestPath  string
	ExpiresAtVar string // env var send.go checks before sending, to refresh proactively
}

// Enabled reports whether this request should refresh its token (both
// proactively, before sending, and reactively, on a 401 - see send.go).
func (r RefreshConfig) Enabled() bool {
	return r.RequestPath != ""
}

// Request is the full, editable definition of an API call.
type Request struct {
	// Protocol selects HTTP (default) vs WebSocket. For a WebSocket request only
	// URL and Headers (the handshake headers) are meaningful; the HTTP-only
	// fields below are ignored.
	Protocol Protocol
	// WSPeer is the optional second endpoint of a two-pane WebSocket session
	// (pane B); nil for a single-connection WS request or any HTTP request.
	WSPeer             *WSEndpoint
	Method             Method
	URL                string
	Params             []QueryParam
	Headers            []Header
	Body               Body
	Timeout            time.Duration
	FollowRedirects    bool
	InsecureSkipVerify bool
	PreRequestScript   string
	TestScript         string
	AuthCapture        AuthCapture
	Refresh            RefreshConfig
}
