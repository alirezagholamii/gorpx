package router

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alirezagholamii/gorpx/internal/config"
)

func TestRouting(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "API:", r.URL.Path)
		},
	))
	defer api.Close()

	app := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "APP:", r.URL.Path)
		},
	))
	defer app.Close()

	handler, err := New([]config.Route{
		{Path: "/api/", Upstreams: []string{api.URL, app.URL}},
		{Path: "/app/", Upstreams: []string{app.URL}},
	})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		path       string
		wantStatus int
		wantBody   string
	}{
		{"/api/users", 200, "API:/api/users"},
		{"/app/home", 200, "APP:/app/home"},
		{"/unknown", 404, "404 page not found\n"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			if got := rec.Body.String(); got != tt.wantBody {
				t.Errorf("body = %q, want %q", got, tt.wantBody)
			}
		})
	}
}

func TestRejectEquivalentEscapedRoutes(t *testing.T) {
	for _, paths := range [][]string{
		{"/api/", "/%61pi/"},
		{"/%61pi/", "/api/"},
		{"/api/", "/%61%70%69/"},
	} {
		t.Run(strings.Join(paths, ","), func(t *testing.T) {
			var routes []config.Route
			for _, path := range paths {
				routes = append(routes, config.Route{
					Path:      path,
					Upstreams: []string{"http://localhost:3000"},
				})
			}

			handler, err := New(routes)
			if err == nil || !strings.Contains(err.Error(), "must not contain percent escapes") {
				t.Fatalf("New() error = %v, want percent-escape validation error", err)
			}
			if handler != nil {
				t.Fatal("New() returned a handler for invalid routes")
			}
		})
	}
}
