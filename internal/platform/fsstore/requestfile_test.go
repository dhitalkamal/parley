package fsstore

import (
	"reflect"
	"testing"
	"time"

	collection "github.com/dhitalkamal/parley/internal/collection/domain"
)

// TestRequestFile_RoundTripFullHTTP is the core guarantee: a fully-populated
// HTTP request survives RequestToFile -> RequestFromFile unchanged. This is the
// on-disk serialization every saved request depends on.
func TestRequestFile_RoundTripFullHTTP(t *testing.T) {
	orig := collection.Request{
		Method: collection.POST,
		URL:    "https://api.example.com/v1/things",
		Params: []collection.QueryParam{
			{Key: "page", Value: "1", Enabled: true, Description: "which page"},
			{Key: "q", Value: "term", Enabled: false},
		},
		Headers: []collection.Header{
			{Key: "Authorization", Value: "Bearer x", Enabled: true},
			{Key: "X-Trace", Value: "off", Enabled: false, Description: "tracing"},
		},
		Body: collection.Body{
			Type:           collection.BodyMultipart,
			RawContentType: "application/json",
			RawText:        `{"a":1}`,
			FormFields: []collection.BodyFormField{
				{Key: "field", Value: "v", Enabled: true},
				{Key: "upload", FilePath: "/tmp/f", IsFile: true, Enabled: true},
			},
			GraphQLQuery:     "query{ x }",
			GraphQLVariables: `{"v":1}`,
			BinaryFilePath:   "/tmp/bin",
		},
		Timeout:            30 * time.Second,
		FollowRedirects:    true,
		InsecureSkipVerify: true,
		PreRequestScript:   "pm.environment.set('a','b')",
		TestScript:         "pm.test('ok', () => {})",
		AuthCapture: collection.AuthCapture{
			TokenField: "data.token", TokenVar: "TOKEN",
			ExpiresInField: "data.expiresIn", ExpiresAtVar: "TOKEN_EXP",
		},
		Refresh: collection.RefreshConfig{RequestPath: "auth/refresh", ExpiresAtVar: "TOKEN_EXP"},
	}

	got := RequestFromFile(RequestToFile(orig))
	if !reflect.DeepEqual(got, orig) {
		t.Errorf("round trip changed the request:\n orig = %+v\n got  = %+v", orig, got)
	}
}

// TestRequestFile_RoundTripWebSocket covers the WS-only fields (protocol and
// the optional second endpoint), which serialize through separate omitempty
// paths from the HTTP fields.
func TestRequestFile_RoundTripWebSocket(t *testing.T) {
	orig := collection.Request{
		Protocol: collection.ProtocolWebSocket,
		URL:      "wss://example.com/a",
		Headers:  []collection.Header{{Key: "Origin", Value: "x", Enabled: true}},
		WSPeer: &collection.WSEndpoint{
			URL:     "wss://example.com/b",
			Headers: []collection.Header{{Key: "Authorization", Value: "Bearer y", Enabled: true, Description: "peer auth"}},
		},
	}

	got := RequestFromFile(RequestToFile(orig))
	if !reflect.DeepEqual(got, orig) {
		t.Errorf("ws round trip changed the request:\n orig = %+v\n got  = %+v", orig, got)
	}
}

// TestRequestFile_RoundTripZeroValue guards that an empty request (no optional
// pointers, no slices) round-trips to an identical zero-value request rather
// than picking up empty non-nil slices or zeroed pointer structs.
func TestRequestFile_RoundTripZeroValue(t *testing.T) {
	var orig collection.Request
	got := RequestFromFile(RequestToFile(orig))
	if !reflect.DeepEqual(got, orig) {
		t.Errorf("zero round trip changed the request:\n orig = %+v\n got  = %+v", orig, got)
	}
}

// TestRequestFile_DisabledAuthAndRefreshNotPersisted verifies the Enabled()
// gating: a request whose auth-capture/refresh are not configured must not
// serialize those blocks (they stay nil in the file), and must load back as
// disabled.
func TestRequestFile_DisabledAuthAndRefreshNotPersisted(t *testing.T) {
	f := RequestToFile(collection.Request{Method: collection.GET, URL: "https://x"})
	if f.AuthCapture != nil {
		t.Errorf("disabled auth capture should not be serialized, got %+v", f.AuthCapture)
	}
	if f.Refresh != nil {
		t.Errorf("disabled refresh should not be serialized, got %+v", f.Refresh)
	}
	back := RequestFromFile(f)
	if back.AuthCapture.Enabled() || back.Refresh.Enabled() {
		t.Error("disabled auth/refresh must load back as disabled")
	}
}

// TestRequestFile_TimeoutSurvivesSubSecond guards the time.Duration <-> seconds
// conversion for a fractional value.
func TestRequestFile_TimeoutSurvivesSubSecond(t *testing.T) {
	orig := collection.Request{Timeout: 1500 * time.Millisecond}
	got := RequestFromFile(RequestToFile(orig))
	if got.Timeout != orig.Timeout {
		t.Errorf("timeout = %v, want %v", got.Timeout, orig.Timeout)
	}
}
