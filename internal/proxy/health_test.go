package proxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != "/health" {
						t.Errorf(
							"path = %q, want /health",
							r.URL.Path,
						)
					}

					if tt.delay > 0 {
						time.Sleep(tt.delay)
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
