package middleware

import (
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
