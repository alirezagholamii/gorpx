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
	const limit int64 = 1024

	tests := []struct {
		name       string
		size       int64
		wantStatus int
	}{
		{"small body", 512, http.StatusOK},
		{"at limit", limit, http.StatusOK},
		{"over limit", limit + 1, http.StatusRequestEntityTooLarge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next := http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					if _, err := io.Copy(io.Discard, r.Body); err != nil {
						t.Fatalf("reading body: %v", err)
					}

					w.WriteHeader(http.StatusOK)
				},
			)

			handler := BodyLimit(next, limit)

			body := strings.NewReader(
				strings.Repeat("x", int(tt.size)),
			)

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
	const limit int64 = 1024

	var readErr error

	next := http.HandlerFunc(
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
	)

	handler := BodyLimit(next, limit)

	body := strings.NewReader(
		strings.Repeat("x", int(limit+1)),
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/upload",
		body,
	)

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

	if maxErr.Limit != limit {
		t.Errorf(
			"limit = %d, want %d",
			maxErr.Limit,
			limit,
		)
	}

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf(
			"status = %d, want 413",
			rec.Code,
		)
	}
}
