package proxy

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

type selectedUpstream struct {
	host string
}

type selectedUpstreamKey struct{}

func New(targets []*url.URL, responseHeaderTimeout time.Duration) http.Handler {
	selector := newRoundRobin(targets)
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = responseHeaderTimeout

	reverseProxy := &httputil.ReverseProxy{
		Transport: transport,

		Rewrite: func(pr *httputil.ProxyRequest) {
			target := selector.Next()

			selected := pr.In.Context().Value(selectedUpstreamKey{}).(*selectedUpstream)
			selected.host = target.Host

			pr.SetURL(target)
			pr.SetXForwarded()

			// Preserve Gorpx's trusted request ID.
			if requestID := pr.In.Header.Get("X-Request-ID"); requestID != "" {
				pr.Out.Header.Set("X-Request-ID", requestID)
			}
		},

		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			selected := r.Context().Value(selectedUpstreamKey{}).(*selectedUpstream)

			slog.Error(
				"upstream request failed",
				"upstream", selected.host,
				"path", r.URL.Path,
				"request_id", r.Header.Get("X-Request-ID"),
				"error", err,
			)

			http.Error(w, "Bad Gateway", http.StatusBadGateway)
		},
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		selected := &selectedUpstream{}

		ctx := context.WithValue(
			r.Context(),
			selectedUpstreamKey{},
			selected,
		)

		reverseProxy.ServeHTTP(w, r.WithContext(ctx))
	})
}
