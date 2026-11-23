package proxy

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"
)

func TestCheckHealth(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		delay   time.Duration
		timeout time.Duration
		wantErr bool
	}{
		{
			name:    "healthy",
			status:  http.StatusOK,
			timeout: time.Second,
		},
		{
			name:    "healthy no content",
			status:  http.StatusNoContent,
			timeout: time.Second,
		},
		{
			name:    "service unavailable",
			status:  http.StatusServiceUnavailable,
			timeout: time.Second,
			wantErr: true,
		},
		{
			name:    "redirect",
			status:  http.StatusFound,
			timeout: time.Second,
			wantErr: true,
		},
		{
			name:    "timeout",
			status:  http.StatusOK,
			delay:   100 * time.Millisecond,
			timeout: 10 * time.Millisecond,
			wantErr: true,
		},
	}

	client := newHealthClient()
	t.Cleanup(client.CloseIdleConnections)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/redirect-target" {
						w.WriteHeader(http.StatusNoContent)
						return
					}

					if r.URL.Path != "/health" {
						t.Errorf(
							"path = %q, want /health",
							r.URL.Path,
						)
					}

					if tt.delay > 0 {
						time.Sleep(tt.delay)
					}

					if tt.status == http.StatusFound {
						w.Header().Set("Location", "/redirect-target")
					}
					w.WriteHeader(tt.status)
				},
			))
			defer server.Close()

			upstream, err := url.Parse(server.URL)
			if err != nil {
				t.Fatal(err)
			}

			err = checkHealth(
				context.Background(),
				client,
				upstream,
				"/health",
				tt.timeout,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf(
					"checkHealth() error = %v, wantErr %v",
					err,
					tt.wantErr,
				)
			}
		})
	}
}

func TestCheckHealthReusesConnection(t *testing.T) {
	var connections atomic.Int32
	backend := httptest.NewUnstartedServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
	))
	backend.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			connections.Add(1)
		}
	}
	backend.Start()
	t.Cleanup(backend.Close)

	target, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatal(err)
	}
	client := newHealthClient()
	t.Cleanup(client.CloseIdleConnections)
	if client.Transport == nil || client.Transport == http.DefaultTransport {
		t.Fatal("probe client must own its transport")
	}

	for range 2 {
		if err := checkHealth(t.Context(), client, target, "/health", 5*time.Second); err != nil {
			t.Fatal(err)
		}
	}
	if got := connections.Load(); got != 1 {
		t.Errorf("two successful probes opened %d connections, want 1", got)
	}
}

func TestCheckHealthConnectionFailure(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	)

	upstream, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	server.Close()

	err = checkHealth(
		context.Background(),
		newHealthClient(),
		upstream,
		"/health",
		time.Second,
	)

	if err == nil {
		t.Fatal("expected a connection error")
	}
}
