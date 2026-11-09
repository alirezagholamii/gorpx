package server

import (
	"net/http"
	"net/http/httptest"
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
