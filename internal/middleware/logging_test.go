package middleware

import (
	"bytes"
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
