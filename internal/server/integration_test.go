package server

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
			Address: "127.0.0.1:8080",
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

func TestIntegrationStreamedBodyLimit(t *testing.T) {
	received := make(chan int64, 1)

	// Backend counts the bytes it actually receives.
	upstream := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			n, _ := io.Copy(io.Discard, r.Body)
			received <- n
			w.WriteHeader(http.StatusOK)
		},
	))
	defer upstream.Close()

	cfg := config.Config{
		Server: config.Server{
			Address: "127.0.0.1:8080",
		},
		Routes: []config.Route{
			{
				Path:     "/api/",
				Upstream: upstream.URL,
			},
		},
	}

	srv, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	gorpx := httptest.NewServer(srv.Handler)
	defer gorpx.Close()

	// Send one byte more than the 10 MiB limit.
	body := strings.NewReader(
		strings.Repeat("x", (10<<20)+1),
	)

	req, err := http.NewRequest(
		http.MethodPost,
		gorpx.URL+"/api/upload",
		body,
	)
	if err != nil {
		t.Fatal(err)
	}

	// Force an unknown-length, chunked request.
	req.ContentLength = -1

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)

	// The proxy may close the connection after the limit
	// is exceeded. A transport error is acceptable here.
	if err == nil {
		resp.Body.Close()
	}

	// Check what actually reached the upstream.
	select {
	case n := <-received:
		if n > 10<<20 {
			t.Errorf(
				"upstream received %d bytes, limit is %d",
				n,
				10<<20,
			)
		}

		t.Logf("upstream received %d bytes", n)

	case <-time.After(5 * time.Second):
		t.Fatal("upstream did not finish reading")
	}
}

func TestIntegrationUpstreamUnavailable(t *testing.T) {
	// Start an upstream to obtain a valid address.
	upstream := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	))

	upstreamURL := upstream.URL

	// Stop the upstream before sending a request.
	upstream.Close()

	cfg := config.Config{
		Server: config.Server{
			Address: "127.0.0.1:8080",
		},
		Routes: []config.Route{
			{
				Path:     "/api/",
				Upstream: upstreamURL,
			},
		},
	}

	srv, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	gorpx := httptest.NewServer(srv.Handler)
	defer gorpx.Close()

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Request a resource from the unavailable upstream.
	resp, err := client.Get(gorpx.URL + "/api/users")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf(
			"status = %d, want %d",
			resp.StatusCode,
			http.StatusBadGateway,
		)
	}

	// Verify that Gorpx continues handling requests.
	health, err := client.Get(gorpx.URL + "/health")
	if err != nil {
		t.Fatalf("health check failed: %v", err)
	}
	defer health.Body.Close()

	if health.StatusCode != http.StatusOK {
		t.Errorf(
			"health status = %d, want %d",
			health.StatusCode,
			http.StatusOK,
		)
	}
}

func TestIntegrationUpstreamTimeout(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-r.Context().Done():
				return

			case <-time.After(20 * time.Second):
				w.WriteHeader(http.StatusOK)
			}
		},
	))
	defer upstream.Close()

	cfg := config.Config{
		Server: config.Server{
			Address: "127.0.0.1:8080",
		},
		Routes: []config.Route{
			{
				Path:     "/api/",
				Upstream: upstream.URL,
			},
		},
	}

	srv, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	gorpx := httptest.NewServer(srv.Handler)
	defer gorpx.Close()

	client := &http.Client{
		Timeout: 25 * time.Second,
	}

	resp, err := client.Get(gorpx.URL + "/api/slow")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf(
			"status = %d, want %d",
			resp.StatusCode,
			http.StatusBadGateway,
		)
	}

	// Verify Gorpx remains responsive.
	health, err := client.Get(gorpx.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer health.Body.Close()

	if health.StatusCode != http.StatusOK {
		t.Errorf(
			"health status = %d, want %d",
			health.StatusCode,
			http.StatusOK,
		)
	}
}

func TestIntegrationStaticSecurity(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "public")

	if err := os.Mkdir(root, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(root, "safe.txt"),
		[]byte("safe content"),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(base, "secret.txt"),
		[]byte("PRIVATE_SECRET"),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink(
		"safe.txt",
		filepath.Join(root, "internal-link.txt"),
	); err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink(
		"../secret.txt",
		filepath.Join(root, "escape-link.txt"),
	); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{
		Server: config.Server{
			Address: "127.0.0.1:8080",
		},
		Routes: []config.Route{
			{
				Path: "/static/",
				Root: root,
			},
		},
	}

	srv, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	gorpx := httptest.NewServer(srv.Handler)
	defer gorpx.Close()

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "regular file",
			path:       "/static/safe.txt",
			wantStatus: http.StatusOK,
			wantBody:   "safe content",
		},
		{
			name:       "internal symlink",
			path:       "/static/internal-link.txt",
			wantStatus: http.StatusOK,
			wantBody:   "safe content",
		},
		{
			name:       "escaping symlink",
			path:       "/static/escape-link.txt",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "directory listing disabled",
			path:       "/static/",
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := gorpx.Client().Get(gorpx.URL + tt.path)
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

			if tt.wantBody != "" &&
				string(body) != tt.wantBody {
				t.Errorf(
					"body = %q, want %q",
					body,
					tt.wantBody,
				)
			}

			if bytes.Contains(body, []byte("PRIVATE_SECRET")) {
				t.Fatal("security failure: secret file exposed")
			}
		})
	}
}

func TestIntegrationRequestID(t *testing.T) {
	var upstreamID string

	upstream := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			upstreamID = r.Header.Get("X-Request-ID")

			w.Header().Set("X-Request-ID", "upstream-id")
			w.WriteHeader(http.StatusOK)
		},
	))
	defer upstream.Close()

	cfg := config.Config{
		Server: config.Server{
			Address: "127.0.0.1:8080",
		},
		Routes: []config.Route{
			{
				Path:     "/api/",
				Upstream: upstream.URL,
			},
		},
	}

	srv, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	gorpx := httptest.NewServer(srv.Handler)
	defer gorpx.Close()

	req, err := http.NewRequest(
		http.MethodGet,
		gorpx.URL+"/api/test",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("X-Request-ID", "client-id")

	resp, err := gorpx.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	responseID := resp.Header.Get("X-Request-ID")

	if responseID == "" {
		t.Fatal("missing response request ID")
	}

	if responseID == "client-id" {
		t.Error("client-provided request ID was trusted")
	}

	if responseID == "upstream-id" {
		t.Error("upstream overwrote Gorpx request ID")
	}

	if responseID != upstreamID {
		t.Errorf(
			"response ID = %q, upstream received ID = %q",
			responseID,
			upstreamID,
		)
	}
}
