package middleware

import "net/http"

func BodyLimit(next http.Handler, maxBytes int64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength > maxBytes {
			http.Error(
				w,
				"Request body too large",
				http.StatusRequestEntityTooLarge,
			)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

		next.ServeHTTP(w, r)
	})
}
