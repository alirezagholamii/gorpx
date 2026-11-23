package proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

func newHealthClient() *http.Client {
	return &http.Client{
		CheckRedirect: func(
			req *http.Request,
			via []*http.Request,
		) error {
			return http.ErrUseLastResponse
		},
	}
}

func checkHealth(
	ctx context.Context,
	client *http.Client,
	upstream *url.URL,
	path string,
	timeout time.Duration,
) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	target := *upstream
	target.Path = path
	target.RawPath = ""
	target.RawQuery = ""
	target.Fragment = ""

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		target.String(),
		nil,
	)
	if err != nil {
		return fmt.Errorf("create health request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("health request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf(
			"unhealthy response: HTTP %d",
			resp.StatusCode,
		)
	}

	return nil
}
