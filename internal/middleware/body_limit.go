package middleware

import "net/http"

const MaxBodyBytes int64 = 10 << 20 // 10 MiB

func BodyLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength > MaxBodyBytes {
			http.Error(
				w,
				"Request body too large",
				http.StatusRequestEntityTooLarge,
			)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)

		next.ServeHTTP(w, r)
	})
}
