package proxy

import (
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

func New(target *url.URL) http.Handler {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = 15 * time.Second

	return &httputil.ReverseProxy{
		Transport: transport,

		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(target)
			r.SetXForwarded()
		},

		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			slog.Error(
				"upstream request failed",
				"upstream", target.Host,
				"path", r.URL.Path,
				"request_id", r.Header.Get("X-Request-ID"),
				"error", err,
			)

			http.Error(w, "Bad Gateway", http.StatusBadGateway)
		},
	}
}
