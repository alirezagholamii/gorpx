package router

import (
	"fmt"
	"net/http"
	"net/http/httptest"
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
		{Path: "/api/", Upstream: api.URL},
		{Path: "/app/", Upstream: app.URL},
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
