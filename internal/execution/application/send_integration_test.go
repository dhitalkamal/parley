package execapp_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	execapp "github.com/dhitalkamal/parley/internal/execution/application"
	"github.com/dhitalkamal/parley/internal/execution/infrastructure/httpclient"
)

func TestSendRequest_IntegrationWithRealClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	req := collection.Request{Method: collection.GET, URL: srv.URL}
	resp, err := execapp.SendRequest(context.Background(), httpclient.New(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status code = %d, want 200", resp.StatusCode)
	}
	if string(resp.Body) != `{"status":"ok"}` {
		t.Errorf("body = %q", resp.Body)
	}
}
