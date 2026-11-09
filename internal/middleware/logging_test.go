package middleware

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLogging(t *testing.T) {
	var output bytes.Buffer

	logger := slog.New(slog.NewJSONHandler(&output, nil))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("created"))
	})

	wrapped := Logging(handler, logger)

	req := httptest.NewRequest(http.MethodPost, "/api/items", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201", rec.Code)
	}

	logOutput := output.String()

	for _, expected := range []string{
		`"method":"POST"`,
		`"path":"/api/items"`,
		`"status":201`,
		`"duration_ms":`,
	} {
		if !strings.Contains(logOutput, expected) {
			t.Errorf("log missing %q", expected)
		}
	}
}

func TestLoggingKeepsFirstStatus(t *testing.T) {
	var output bytes.Buffer

	logger := slog.New(
		slog.NewJSONHandler(&output, nil),
	)

	handler := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
			w.WriteHeader(http.StatusInternalServerError)
		},
	)

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	Logging(handler, logger).ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", rec.Code)
	}

	if !strings.Contains(output.String(), `"status":201`) {
		t.Errorf("incorrect status log: %s", output.String())
	}
}

func TestLoggingSupportsFlush(t *testing.T) {
	logger := slog.New(
		slog.NewJSONHandler(io.Discard, nil),
	)

	handler := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("event: hello\n\n"))

			err := http.NewResponseController(w).Flush()
			if err != nil {
				t.Errorf("flush failed: %v", err)
			}
		},
	)

	req := httptest.NewRequest("GET", "/events", nil)
	rec := httptest.NewRecorder()

	Logging(handler, logger).ServeHTTP(rec, req)

	if !rec.Flushed {
		t.Error("response was not flushed")
	}

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}
