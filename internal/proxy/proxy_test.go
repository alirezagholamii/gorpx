package proxy

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestProxy(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(r.URL.Path))
		},
	))
	defer backend.Close()

	target, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/proxy/hello", nil)
	rec := httptest.NewRecorder()

	New(target).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	if got := rec.Body.String(); got != "/proxy/hello" {
		t.Errorf("body = %q, want /proxy/hello", got)
	}
}

func TestUnavailableUpstream(t *testing.T) {
	backend := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	target, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatal(err)
	}

	backend.Close()

	handler := New(target)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/users",
		nil,
	)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Errorf(
			"status = %d, want 502",
			rec.Code,
		)
	}
}

func TestForwardedFor(t *testing.T) {
	backend := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, r.Header.Get("X-Forwarded-For"))
		}),
	)
	defer backend.Close()

	target, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/users",
		nil,
	)

	req.RemoteAddr = "192.0.2.10:12345"

	req.Header.Set(
		"X-Forwarded-For",
		"203.0.113.99",
	)

	rec := httptest.NewRecorder()

	New(target).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	if got := rec.Body.String(); got != "192.0.2.10" {
		t.Errorf(
			"X-Forwarded-For = %q, want 192.0.2.10",
			got,
		)
	}
}
