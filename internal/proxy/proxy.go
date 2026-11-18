package proxy

import (
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

func New(targets []*url.URL, responseHeaderTimeout time.Duration) http.Handler {
	selector := newRoundRobin(targets)
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = responseHeaderTimeout

	loggedTransport := &loggingTransport{
		base:   transport,
		logger: slog.Default(),
	}

	return &httputil.ReverseProxy{
		Transport: loggedTransport,

		Rewrite: func(pr *httputil.ProxyRequest) {
			target := selector.Next()

			pr.SetURL(target)
			pr.SetXForwarded()
			
			// Restore the trusted request ID after hop-by-hop
			// headers have been removed.
			if requestID := pr.In.Header.Get("X-Request-ID"); requestID != "" {
				pr.Out.Header.Set("X-Request-ID", requestID)
			}
		},

		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
		},
	}
}
