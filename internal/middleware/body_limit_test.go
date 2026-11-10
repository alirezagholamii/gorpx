package middleware

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBodyLimit(t *testing.T) {
	tests := []struct {
		name       string
		size       int64
		wantStatus int
	}{
		{"small body", 1024, http.StatusOK},
		{"at limit", MaxBodyBytes, http.StatusOK},
		{"over limit", MaxBodyBytes + 1, http.StatusRequestEntityTooLarge},
	}

	handler := BodyLimit(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if _, err := io.Copy(io.Discard, r.Body); err != nil {
				t.Errorf("reading body: %v", err)
				return
			}
			w.WriteHeader(http.StatusOK)
		},
	))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := strings.NewReader(strings.Repeat("x", int(tt.size)))

			req := httptest.NewRequest(
				http.MethodPost,
				"/api/upload",
				body,
			)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf(
					"status = %d, want %d",
					rec.Code,
					tt.wantStatus,
				)
			}
		})
	}
}

func TestBodyLimitUnknownLength(t *testing.T) {
	var readErr error

	handler := BodyLimit(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			_, readErr = io.Copy(io.Discard, r.Body)

			if readErr != nil {
				var maxErr *http.MaxBytesError

				if errors.As(readErr, &maxErr) {
					http.Error(
						w,
						"Request body too large",
						http.StatusRequestEntityTooLarge,
					)
					return
				}

				http.Error(
					w,
					"Failed to read request",
					http.StatusBadRequest,
				)
				return
			}

			w.WriteHeader(http.StatusOK)
		},
	))

	body := strings.NewReader(
		strings.Repeat("x", int(MaxBodyBytes+1)),
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/upload",
		body,
	)

	// Simulate an unknown-length request.
	req.ContentLength = -1

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	var maxErr *http.MaxBytesError
	if !errors.As(readErr, &maxErr) {
		t.Fatalf(
			"read error = %v, want *http.MaxBytesError",
			readErr,
		)
	}

	if maxErr.Limit != MaxBodyBytes {
		t.Errorf(
			"limit = %d, want %d",
			maxErr.Limit,
			MaxBodyBytes,
		)
	}

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf(
			"status = %d, want 413",
			rec.Code,
		)
	}
}
