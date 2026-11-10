package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alirezagholamii/gorpx/internal/config"
)

func TestHandler(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "root",
			path:       "/",
			wantStatus: http.StatusOK,
			wantBody:   "Hello from Gorpx!",
		},
		{
			name:       "health",
			path:       "/health",
			wantStatus: http.StatusOK,
			wantBody:   "OK",
		},
		{
			name:       "not found",
			path:       "/unknown",
			wantStatus: http.StatusNotFound,
			wantBody:   "404 page not found\n",
		},
	}

	cfg := config.Default()

	cfg.Routes = []config.Route{
		{
			Path:     "/api/",
			Upstream: "http://localhost:3000",
		},
	}

	srv, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	handler := srv.Handler

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			if rec.Body.String() != tt.wantBody {
				t.Errorf("body = %q, want %q", rec.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestMaxHeaderBytes(t *testing.T) {
	cfg := config.Config{
		Server: config.Server{
			Address: "127.0.0.1:8080",
		},
		Routes: []config.Route{
			{
				Path:     "/api/",
				Upstream: "http://localhost:3000",
			},
		},
	}

	srv, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	testServer := httptest.NewUnstartedServer(srv.Handler)
	testServer.Config.MaxHeaderBytes = srv.MaxHeaderBytes
	testServer.Start()
	defer testServer.Close()

	req, err := http.NewRequest(
		http.MethodGet,
		testServer.URL+"/health",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("X-Large-Header", strings.Repeat("x", 64<<10))

	resp, err := testServer.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusRequestHeaderFieldsTooLarge {
		t.Errorf(
			"status = %d, want 431",
			resp.StatusCode,
		)
	}
}
