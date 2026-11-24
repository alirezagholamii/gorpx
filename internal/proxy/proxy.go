package proxy

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/alirezagholamii/gorpx/internal/config"
)

type selectedUpstream struct {
	target *url.URL
}

type selectedUpstreamKey struct{}

func New(targets []*url.URL, responseHeaderTimeout time.Duration) http.Handler {
	selector := newRoundRobin(targets)
	return newHandler(selector, responseHeaderTimeout)
}

func newHandler(
	selector *roundRobin,
	responseHeaderTimeout time.Duration,
) http.Handler {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = responseHeaderTimeout

	reverseProxy := &httputil.ReverseProxy{
		Transport: transport,

		Rewrite: func(pr *httputil.ProxyRequest) {
			selected := pr.In.Context().Value(selectedUpstreamKey{}).(*selectedUpstream)

			pr.SetURL(selected.target)
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
				"upstream", selected.target.Host,
				"path", r.URL.Path,
				"request_id", r.Header.Get("X-Request-ID"),
				"error", err,
			)

			http.Error(w, "Bad Gateway", http.StatusBadGateway)
		},
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target, ok := selector.Next()
		if !ok {
			http.Error(
				w,
				"Service Unavailable",
				http.StatusServiceUnavailable,
			)
			return
		}

		selected := &selectedUpstream{
			target: target,
		}

		ctx := context.WithValue(
			r.Context(),
			selectedUpstreamKey{},
			selected,
		)

		reverseProxy.ServeHTTP(w, r.WithContext(ctx))
	})
}

func NewWithHealth(
	targets []*url.URL,
	responseHeaderTimeout time.Duration,
	hc *config.HealthCheck,
) (http.Handler, HealthRunner, error) {
	selector := newRoundRobin(targets)
	handler := newHandler(selector, responseHeaderTimeout)

	if hc == nil {
		return handler, nil, nil
	}

	interval, err := time.ParseDuration(hc.Interval)
	if err != nil || interval <= 0 {
		return nil, nil, fmt.Errorf(
			"invalid health check interval %q",
			hc.Interval,
		)
	}

	timeout, err := time.ParseDuration(hc.Timeout)
	if err != nil || timeout <= 0 {
		return nil, nil, fmt.Errorf(
			"invalid health check timeout %q",
			hc.Timeout,
		)
	}

	if timeout > interval {
		return nil, nil, fmt.Errorf(
			"health check timeout must not exceed interval",
		)
	}

	scheduler := newHealthScheduler(
		selector,
		hc.Path,
		interval,
		timeout,
	)

	return handler, scheduler, nil
}
