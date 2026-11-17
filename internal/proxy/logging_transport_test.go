package proxy

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoggingTransport(t *testing.T) {
	upstream := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	)

	address := upstream.URL
	upstream.Close()

	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))

	transport := &loggingTransport{
		base:   http.DefaultTransport.(*http.Transport).Clone(),
		logger: logger,
	}

	defer transport.base.(*http.Transport).CloseIdleConnections()

	req, err := http.NewRequest(http.MethodGet, address+"/api/test", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("X-Request-ID", "test-request-id")

	resp, err := transport.RoundTrip(req)
	if resp != nil {
		resp.Body.Close()
	}
	if err == nil {
		t.Fatal("expected upstream connection error")
	}

	var entry struct {
		Message   string `json:"msg"`
		Upstream  string `json:"upstream"`
		RequestID string `json:"request_id"`
	}

	if err := json.Unmarshal(logs.Bytes(), &entry); err != nil {
		t.Fatal(err)
	}

	if entry.Message != "upstream request failed" {
		t.Errorf("unexpected message: %q", entry.Message)
	}

	if entry.Upstream != req.URL.Host {
		t.Errorf("upstream = %q, want %q", entry.Upstream, req.URL.Host)
	}

	if entry.RequestID != "test-request-id" {
		t.Errorf("request ID = %q, want test-request-id", entry.RequestID)
	}
}
