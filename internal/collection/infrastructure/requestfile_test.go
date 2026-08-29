package collectionstore

import (
	"reflect"
	"strings"
	"testing"
	"time"

	collection "github.com/dhitalkamal/parley/internal/collection/domain"
)

// TestEncodeDecodeRequest_WebSocketProtocol guards that a WebSocket request
// round-trips its protocol and handshake headers.
func TestEncodeDecodeRequest_WebSocketProtocol(t *testing.T) {
	req := collection.Request{
		Protocol: collection.ProtocolWebSocket,
		URL:      "wss://echo.example.com/ws",
		Headers: []collection.Header{
			{Key: "Authorization", Value: "Bearer t", Enabled: true},
		},
	}
	encoded, err := encodeRequest(req)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if !strings.Contains(string(encoded), `"protocol": "ws"`) {
		t.Errorf("encoded form should carry the protocol, got:\n%s", encoded)
	}
	decoded, err := decodeRequest(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !decoded.IsWebSocket() {
		t.Errorf("decoded protocol = %q, want ws", decoded.Protocol)
	}
	if len(decoded.Headers) != 1 || decoded.Headers[0].Key != "Authorization" {
		t.Errorf("handshake headers lost: %+v", decoded.Headers)
	}
}

// TestEncodeDecodeRequest_WebSocketPair guards that a two-pane WS session's
// second endpoint (WSPeer) round-trips, and that a single-pane WS request omits
// it entirely.
func TestEncodeDecodeRequest_WebSocketPair(t *testing.T) {
	req := collection.Request{
		Protocol: collection.ProtocolWebSocket,
		URL:      "wss://a.example.com/ws",
		Headers:  []collection.Header{{Key: "Authorization", Value: "Bearer a", Enabled: true}},
		WSPeer: &collection.WSEndpoint{
			URL:     "wss://b.example.com/ws",
			Headers: []collection.Header{{Key: "Authorization", Value: "Bearer b", Enabled: true}},
		},
	}
	encoded, err := encodeRequest(req)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if !strings.Contains(string(encoded), `"wsPeer"`) {
		t.Errorf("encoded pair should carry wsPeer, got:\n%s", encoded)
	}
	decoded, err := decodeRequest(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.WSPeer == nil || decoded.WSPeer.URL != "wss://b.example.com/ws" {
		t.Fatalf("peer not round-tripped: %+v", decoded.WSPeer)
	}
	if len(decoded.WSPeer.Headers) != 1 || decoded.WSPeer.Headers[0].Value != "Bearer b" {
		t.Errorf("peer headers lost: %+v", decoded.WSPeer.Headers)
	}

	// single-pane WS request must not gain a wsPeer key.
	solo, _ := encodeRequest(collection.Request{Protocol: collection.ProtocolWebSocket, URL: "wss://x/ws"})
	if strings.Contains(string(solo), "wsPeer") {
		t.Errorf("single-pane WS request should omit wsPeer, got:\n%s", solo)
	}
}

// TestEncodeDecodeRequest_HTTPProtocolStaysEmpty guards additive compat: a
// plain HTTP request must not gain a protocol field on disk.
func TestEncodeDecodeRequest_HTTPProtocolStaysEmpty(t *testing.T) {
	encoded, err := encodeRequest(collection.Request{Method: collection.GET, URL: "https://x"})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if strings.Contains(string(encoded), "protocol") {
		t.Errorf("HTTP request should omit the protocol field, got:\n%s", encoded)
	}
}

func TestEncodeDecodeRequest_RoundTrips(t *testing.T) {
	req := collection.Request{
		Method: collection.POST,
		URL:    "https://api.example.com/users",
		Params: []collection.QueryParam{
			{Key: "page", Value: "2", Enabled: true, Description: "Page number"},
		},
		Headers: []collection.Header{
			{Key: "Authorization", Value: "Bearer xyz", Enabled: true, Description: "Bearer token"},
		},
		Body: collection.Body{
			Type:           collection.BodyRaw,
			RawContentType: "application/json",
			RawText:        `{"name":"ada"}`,
		},
		Timeout:            15 * time.Second,
		FollowRedirects:    true,
		InsecureSkipVerify: true,
	}

	encoded, err := encodeRequest(req)
	if err != nil {
		t.Fatalf("unexpected encode error: %v", err)
	}
	decoded, err := decodeRequest(encoded)
	if err != nil {
		t.Fatalf("unexpected decode error: %v", err)
	}

	if decoded.Method != req.Method || decoded.URL != req.URL {
		t.Errorf("method/url mismatch: got %+v", decoded)
	}
	if len(decoded.Params) != 1 || decoded.Params[0] != req.Params[0] {
		t.Errorf("params mismatch: got %+v", decoded.Params)
	}
	if len(decoded.Headers) != 1 || decoded.Headers[0] != req.Headers[0] {
		t.Errorf("headers mismatch: got %+v", decoded.Headers)
	}
	if !reflect.DeepEqual(decoded.Body, req.Body) {
		t.Errorf("body mismatch: got %+v, want %+v", decoded.Body, req.Body)
	}
	if decoded.Timeout != req.Timeout {
		t.Errorf("timeout = %v, want %v", decoded.Timeout, req.Timeout)
	}
	if decoded.FollowRedirects != req.FollowRedirects || decoded.InsecureSkipVerify != req.InsecureSkipVerify {
		t.Errorf("toggles mismatch: got %+v", decoded)
	}
}

// TestEncodeDecodeRequest_RoundTripsNewBodyTypes guards the body types added
// alongside None/Raw (form-urlencoded, multipart, GraphQL, binary) - each
// needs its own fields carried through the on-disk RequestFile shape, not
// just the original Type/RawContentType/RawText trio.
func TestEncodeDecodeRequest_RoundTripsNewBodyTypes(t *testing.T) {
	req := collection.Request{
		Method: collection.POST,
		URL:    "https://api.example.com/upload",
		Body: collection.Body{
			Type: collection.BodyMultipart,
			FormFields: []collection.BodyFormField{
				{Key: "title", Value: "photo", Enabled: true},
				{Key: "file", FilePath: "/tmp/photo.png", IsFile: true, Enabled: true},
			},
			GraphQLQuery:     "query { me { id } }",
			GraphQLVariables: `{"id": 1}`,
			BinaryFilePath:   "/tmp/upload.bin",
		},
	}

	encoded, err := encodeRequest(req)
	if err != nil {
		t.Fatalf("unexpected encode error: %v", err)
	}
	decoded, err := decodeRequest(encoded)
	if err != nil {
		t.Fatalf("unexpected decode error: %v", err)
	}
	if !reflect.DeepEqual(decoded.Body, req.Body) {
		t.Errorf("body mismatch: got %+v, want %+v", decoded.Body, req.Body)
	}
}

// TestEncodeDecodeRequest_RoundTripsAuthCaptureAndRefresh guards the auth
// automation fields (AuthCapture, RefreshConfig) added alongside the
// scripting fields - a login request's captured-token config, and a
// protected request's pointer to the saved request that refreshes it,
// both need to survive a save/reload cycle.
func TestEncodeDecodeRequest_RoundTripsAuthCaptureAndRefresh(t *testing.T) {
	req := collection.Request{
		Method: collection.POST,
		URL:    "https://api.example.com/login",
		AuthCapture: collection.AuthCapture{
			TokenField:     "data.token",
			TokenVar:       "token",
			ExpiresInField: "expiresIn",
			ExpiresAtVar:   "tokenExpiresAt",
		},
		Refresh: collection.RefreshConfig{
			RequestPath:  "Auth/refresh.json",
			ExpiresAtVar: "tokenExpiresAt",
		},
	}

	encoded, err := encodeRequest(req)
	if err != nil {
		t.Fatalf("unexpected encode error: %v", err)
	}
	decoded, err := decodeRequest(encoded)
	if err != nil {
		t.Fatalf("unexpected decode error: %v", err)
	}
	if !reflect.DeepEqual(decoded.AuthCapture, req.AuthCapture) {
		t.Errorf("AuthCapture = %+v, want %+v", decoded.AuthCapture, req.AuthCapture)
	}
	if !reflect.DeepEqual(decoded.Refresh, req.Refresh) {
		t.Errorf("Refresh = %+v, want %+v", decoded.Refresh, req.Refresh)
	}
}

func TestEncodeRequest_OmitsEmptyAuthCaptureAndRefresh(t *testing.T) {
	req := collection.Request{Method: collection.GET, URL: "https://example.com"}
	encoded, err := encodeRequest(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(string(encoded), "authCapture") || strings.Contains(string(encoded), "refresh") {
		t.Errorf("expected no auth/refresh fields for a plain request, got:\n%s", encoded)
	}
}

func TestEncodeRequest_StoresTimeoutAsSecondsNotNanoseconds(t *testing.T) {
	req := collection.Request{Method: collection.GET, URL: "https://example.com", Timeout: 30 * time.Second}
	encoded, err := encodeRequest(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(encoded), `"timeoutSeconds": 30`) {
		t.Errorf("expected human-readable timeoutSeconds field, got:\n%s", encoded)
	}
}

func TestEncodeRequest_OmitsEmptyBodyFields(t *testing.T) {
	req := collection.Request{Method: collection.GET, URL: "https://example.com"}
	encoded, err := encodeRequest(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(string(encoded), "rawContentType") {
		t.Errorf("expected no rawContentType field for a bodyless request, got:\n%s", encoded)
	}
}

func TestDecodeRequest_RejectsMalformedJSON(t *testing.T) {
	_, err := decodeRequest([]byte("not json"))
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestEncodeDecodeRequest_ScriptsRoundTrip(t *testing.T) {
	req := collection.Request{
		Method:           collection.GET,
		URL:              "https://example.com",
		PreRequestScript: `pm.environment.set("token", "abc");`,
		TestScript:       `pm.test("ok", function () { pm.expect(pm.response.code).to.equal(200); });`,
	}
	encoded, err := encodeRequest(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	decoded, err := decodeRequest(encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decoded.PreRequestScript != req.PreRequestScript {
		t.Errorf("preRequestScript = %q, want %q", decoded.PreRequestScript, req.PreRequestScript)
	}
	if decoded.TestScript != req.TestScript {
		t.Errorf("testScript = %q, want %q", decoded.TestScript, req.TestScript)
	}
}

func TestEncodeRequest_OmitsEmptyScripts(t *testing.T) {
	req := collection.Request{Method: collection.GET, URL: "https://example.com"}
	encoded, err := encodeRequest(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(string(encoded), "Script") {
		t.Errorf("expected no script fields for a scriptless request, got:\n%s", encoded)
	}
}
