package middleware

import (
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"net/textproto"
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

func TestRequestIDInformationalResponse(t *testing.T) {
	handler := RequestID(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusEarlyHints)

			w.Header().Set(requestIDHeader, "incorrect-id")

			w.WriteHeader(http.StatusOK)
		},
	))

	server := httptest.NewServer(handler)
	defer server.Close()

	var earlyID string

	client := server.Client()
	client.Transport = &http.Transport{}

	req, err := http.NewRequest(
		http.MethodGet,
		server.URL,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	trace := &httptrace.ClientTrace{
		Got1xxResponse: func(
			code int,
			header textproto.MIMEHeader,
		) error {
			if code == http.StatusEarlyHints {
				earlyID = header.Get(requestIDHeader)
			}
			return nil
		},
	}

	req = req.WithContext(
		httptrace.WithClientTrace(req.Context(), trace),
	)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	finalID := resp.Header.Get(requestIDHeader)

	if earlyID == "" {
		t.Fatal("103 response missing request ID")
	}

	if finalID == "" {
		t.Fatal("final response missing request ID")
	}

	if earlyID != finalID {
		t.Errorf(
			"early ID = %q, final ID = %q",
			earlyID,
			finalID,
		)
	}

	if finalID == "incorrect-id" {
		t.Error("request ID was overwritten")
	}
}
