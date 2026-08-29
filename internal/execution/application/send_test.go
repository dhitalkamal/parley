package execapp

import (
	"context"
	"errors"
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
	"testing"
	"time"
)

type fakeClient struct {
	gotReq collection.Request
	resp   execution.Response
	err    error
}

func (f *fakeClient) Do(ctx context.Context, req collection.Request) (execution.Response, error) {
	f.gotReq = req
	return f.resp, f.err
}

func TestSendRequest_AppliesDefaultTimeoutWhenUnset(t *testing.T) {
	fc := &fakeClient{resp: execution.Response{StatusCode: 200}}
	req := collection.Request{Method: collection.GET, URL: "https://example.com"}

	_, err := SendRequest(context.Background(), fc, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fc.gotReq.Timeout != DefaultTimeout {
		t.Errorf("timeout = %v, want default %v", fc.gotReq.Timeout, DefaultTimeout)
	}
}

func TestSendRequest_KeepsExplicitTimeout(t *testing.T) {
	fc := &fakeClient{resp: execution.Response{StatusCode: 200}}
	req := collection.Request{Method: collection.GET, URL: "https://example.com", Timeout: 3 * time.Second}

	_, err := SendRequest(context.Background(), fc, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fc.gotReq.Timeout != 3*time.Second {
		t.Errorf("timeout = %v, want 3s", fc.gotReq.Timeout)
	}
}

func TestSendRequest_PropagatesClientError(t *testing.T) {
	wantErr := errors.New("boom")
	fc := &fakeClient{err: wantErr}
	req := collection.Request{Method: collection.GET, URL: "https://example.com"}

	_, err := SendRequest(context.Background(), fc, req)
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want %v", err, wantErr)
	}
}

func TestSendRequest_ReturnsClientResponse(t *testing.T) {
	want := execution.Response{StatusCode: 201, Body: []byte("hi")}
	fc := &fakeClient{resp: want}
	req := collection.Request{Method: collection.GET, URL: "https://example.com"}

	got, err := SendRequest(context.Background(), fc, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.StatusCode != want.StatusCode || string(got.Body) != string(want.Body) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}
