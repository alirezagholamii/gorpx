package middleware

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

func TestRequestID(t *testing.T) {
	var receivedID string

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedID = r.Header.Get(requestIDHeader)
		w.WriteHeader(http.StatusOK)
	})

	handler := RequestID(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(requestIDHeader, "client-provided-id")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	responseID := rec.Header().Get(requestIDHeader)

	if !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(responseID) {
		t.Fatalf("invalid request ID: %q", responseID)
	}

	if responseID != receivedID {
		t.Errorf(
			"response ID = %q, downstream ID = %q",
			responseID,
			receivedID,
		)
	}

	if responseID == "client-provided-id" {
		t.Error("client-supplied request ID was trusted")
	}
}

func TestRequestIDUnique(t *testing.T) {
	handler := RequestID(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	))

	seen := make(map[string]bool)

	for range 100 {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		id := rec.Header().Get(requestIDHeader)
		if id == "" {
			t.Fatal("missing request ID")
		}

		if seen[id] {
			t.Fatalf("duplicate request ID: %q", id)
		}

		seen[id] = true
	}
}
