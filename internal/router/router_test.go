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

func TestConstructorsWithHealth(t *testing.T) {
	routes := []config.Route{
		{Path: "/api/", Upstreams: []string{"http://localhost:3000"}},
		{Path: "/other/", Upstreams: []string{"http://localhost:3001"}},
	}

	if handler, err := New(routes); err != nil || handler == nil {
		t.Fatalf("New() without health checks = %v, %v, want a handler", handler, err)
	}

	routes[1].HealthCheck = &config.HealthCheck{
		Path:     "/health",
		Interval: "10s",
		Timeout:  "2s",
	}
	if handler, err := New(routes); handler != nil || err == nil ||
		!strings.Contains(err.Error(), `route "/other/": health_check requires router.NewWithHealth()`) {
		t.Fatalf("New() with health checks = %v, %v, want an error directing callers to router.NewWithHealth()", handler, err)
	}

	handler, runners, err := NewWithHealth(routes)
	if err != nil || handler == nil {
		t.Fatalf("NewWithHealth() = %v, %v, want a handler", handler, err)
	}
	if len(runners) != 1 || runners[0] == nil {
		t.Errorf("NewWithHealth() runners = %v, want one non-nil runner", runners)
	}
}

func TestNewWithHealthValidation(t *testing.T) {
	upstreamRoute := config.Route{
		Path:      "/api/",
		Upstreams: []string{"http://localhost:3000"},
	}
	valid := config.HealthCheck{Path: "/health", Interval: "10s", Timeout: "2s"}
	invalid := valid
	invalid.Path = "/../health"
	validRoute := upstreamRoute
	validRoute.HealthCheck = &valid
	invalidRoute := upstreamRoute
	invalidRoute.HealthCheck = &invalid

	tests := []struct {
		name        string
		route       config.Route
		wantErr     string
		wantRunners int
	}{
		{
			name: "static route with health check",
			route: config.Route{
				Path:        "/static/",
				Root:        t.TempDir(),
				HealthCheck: &valid,
			},
			wantErr: "health_check is only valid for upstream routes",
		},
		{
			name:    "invalid probe path",
			route:   invalidRoute,
			wantErr: "invalid health check path",
		},
		{
			name:        "valid health check",
			route:       validRoute,
			wantRunners: 1,
		},
		{
			name:  "health checks omitted",
			route: upstreamRoute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Default()
			cfg.Routes = []config.Route{tt.route}
			configErr := config.Validate(cfg)
			handler, runners, err := NewWithHealth(cfg.Routes)
			if tt.wantErr != "" {
				if configErr == nil || !strings.Contains(configErr.Error(), tt.wantErr) {
					t.Fatalf("config.Validate() error = %v, want %q", configErr, tt.wantErr)
				}
				if err == nil || err.Error() != configErr.Error() {
					t.Fatalf("NewWithHealth() error = %v, want config validation error %v", err, configErr)
				}
				if handler != nil || len(runners) != 0 {
					t.Fatal("NewWithHealth() returned a handler or runner for an invalid route")
				}
				return
			}

			if configErr != nil {
				t.Fatalf("config.Validate() rejected a valid route: %v", configErr)
			}
			if err != nil || handler == nil {
				t.Fatalf("NewWithHealth() = %v, %v, want a handler", handler, err)
			}
			if len(runners) != tt.wantRunners || (tt.wantRunners > 0 && runners[0] == nil) {
				t.Errorf("NewWithHealth() runners = %v, want %d non-nil runners", runners, tt.wantRunners)
			}
		})
	}
}
