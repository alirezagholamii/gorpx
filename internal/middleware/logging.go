package middleware

import (
	"log/slog"
	"net"
	"net/http"
	"time"
)

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) WriteHeader(status int) {
	// Informational responses don't finalise the status,
	// except 101 Switching Protocols.
	if status >= 100 && status < 200 && status != 101 {
		w.ResponseWriter.WriteHeader(status)
		return
	}

	// Record only the first final status.
	if w.status != 0 {
		return
	}

	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}

	return w.ResponseWriter.Write(data)
}

func (w *responseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func Logging(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &responseWriter{
			ResponseWriter: w,
		}

		var aborted bool

		defer func() {
			if recovered := recover(); recovered != nil {
				aborted = true

				defer func() {
					panic(recovered)
				}()
			}

			status := recorder.status
			if status == 0 {
				status = http.StatusOK
			}

			logger.Info(
				"http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", status,
				"duration_ms", time.Since(start).Milliseconds(),
				"client_ip", clientIP(r.RemoteAddr),
				"request_id", r.Header.Get("X-Request-ID"),
				"aborted", aborted,
			)
		}()

		next.ServeHTTP(recorder, r)
	})
}

func clientIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}

	return host
}
