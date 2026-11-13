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

func TestLoggingInformationalResponse(t *testing.T) {
	var output bytes.Buffer

	logger := slog.New(slog.NewJSONHandler(&output, nil))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusEarlyHints)
		w.WriteHeader(http.StatusOK)
	})

	server := httptest.NewServer(Logging(handler, logger))
	defer server.Close()

	resp, err := server.Client().Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	if !strings.Contains(output.String(), `"status":200`) {
		t.Errorf("incorrect log: %s", output.String())
	}
}

func TestLoggingImplicitStatus(t *testing.T) {
	var output bytes.Buffer

	logger := slog.New(
		slog.NewJSONHandler(&output, nil),
	)

	handler := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("Hello"))
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	Logging(handler, logger).ServeHTTP(rec, req)

	if !strings.Contains(output.String(), `"status":200`) {
		t.Errorf("incorrect log: %s", output.String())
	}
}

func TestClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		want       string
	}{
		{
			name:       "IPv4",
			remoteAddr: "192.168.1.10:54321",
			want:       "192.168.1.10",
		},
		{
			name:       "IPv6",
			remoteAddr: "[::1]:54321",
			want:       "::1",
		},
		{
			name:       "IPv6 full address",
			remoteAddr: "[2001:db8::1]:443",
			want:       "2001:db8::1",
		},
		{
			name:       "missing port",
			remoteAddr: "192.168.1.10",
			want:       "192.168.1.10",
		},
		{
			name:       "empty address",
			remoteAddr: "",
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := clientIP(tt.remoteAddr)

			if got != tt.want {
				t.Errorf(
					"clientIP(%q) = %q, want %q",
					tt.remoteAddr,
					got,
					tt.want,
				)
			}
		})
	}
}
