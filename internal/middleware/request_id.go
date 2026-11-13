package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

const requestIDHeader = "X-Request-ID"

type requestIDWriter struct {
	http.ResponseWriter
	id string
}

func (w *requestIDWriter) WriteHeader(status int) {
	w.Header().Set(requestIDHeader, w.id)
	w.ResponseWriter.WriteHeader(status)
}

func (w *requestIDWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var random [16]byte

		if _, err := rand.Read(random[:]); err != nil {
			http.Error(
				w,
				"Internal Server Error",
				http.StatusInternalServerError,
			)
			return
		}

		id := hex.EncodeToString(random[:])

		r.Header.Set(requestIDHeader, id)

		writer := &requestIDWriter{
			ResponseWriter: w,
			id:             id,
		}

		writer.Header().Set(requestIDHeader, id)

		next.ServeHTTP(writer, r)
	})
}
