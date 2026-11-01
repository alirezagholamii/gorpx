package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/alirezagholamii/gorpx/internal/config"
)

func TestIntegration(t *testing.T) {
	// Create a test backend.
	backend := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("API: " + r.URL.Path))
		},
	))
	defer backend.Close()

	// Create a temporary static file.
	dir := t.TempDir()

	err := os.WriteFile(
		filepath.Join(dir, "hello.txt"),
		[]byte("Hello from static"),
		0600,
	)
	if err != nil {
		t.Fatal(err)
	}

	// Configure Gorpx.
	cfg := config.Config{
		Server: config.Server{
			Address: ":0",
		},
		Routes: []config.Route{
			{
				Path:     "/api/",
				Upstream: backend.URL,
			},
			{
				Path: "/static/",
				Root: dir,
			},
		},
	}

	srv, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Start Gorpx using a test HTTP server.
	gorpx := httptest.NewServer(srv.Handler)
	defer gorpx.Close()

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "health",
			path:       "/health",
			wantStatus: 200,
			wantBody:   "OK",
		},
		{
			name:       "reverse proxy",
			path:       "/api/users",
			wantStatus: 200,
			wantBody:   "API: /api/users",
		},
		{
			name:       "static file",
			path:       "/static/hello.txt",
			wantStatus: 200,
			wantBody:   "Hello from static",
		},
		{
			name:       "not found",
			path:       "/unknown",
			wantStatus: 404,
			wantBody:   "404 page not found\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Get(gorpx.URL + tt.path)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}

			if resp.StatusCode != tt.wantStatus {
				t.Errorf(
					"status = %d, want %d",
					resp.StatusCode,
					tt.wantStatus,
				)
			}

			if string(body) != tt.wantBody {
				t.Errorf(
					"body = %q, want %q",
					string(body),
					tt.wantBody,
				)
			}
		})
	}
}
