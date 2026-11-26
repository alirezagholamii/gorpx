package middleware

import (
	"io"
	"net/http"
	"time"
)

type bodyReadDeadline struct {
	io.ReadCloser
	controller *http.ResponseController
	timeout    time.Duration
}

func (b *bodyReadDeadline) Read(p []byte) (int, error) {
	if err := b.controller.SetReadDeadline(time.Now().Add(b.timeout)); err != nil {
		return 0, err
	}

	n, err := b.ReadCloser.Read(p)
	if err != nil && err != io.EOF {
		// Leave a failed read's deadline in effect in case Go drains the body.
		return n, err
	}
	if clearErr := b.controller.SetReadDeadline(time.Time{}); clearErr != nil {
		return n, clearErr
	}
	return n, err
}

func BodyReadTimeout(next http.Handler, timeout time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength != 0 {
			r.Body = &bodyReadDeadline{
				ReadCloser: r.Body,
				controller: http.NewResponseController(w),
				timeout:    timeout,
			}
		}
		next.ServeHTTP(w, r)
	})
}
