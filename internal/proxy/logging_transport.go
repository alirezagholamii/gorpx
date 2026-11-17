package proxy

import (
	"log/slog"
	"net/http"
)

type loggingTransport struct {
	base   http.RoundTripper
	logger *slog.Logger
}

func (t *loggingTransport) RoundTrip(
	req *http.Request,
) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)

	if err != nil {
		t.logger.Error(
			"upstream request failed",
			"upstream", req.URL.Host,
			"path", req.URL.Path,
			"request_id", req.Header.Get("X-Request-ID"),
			"error", err,
		)
	}

	return resp, err
}
