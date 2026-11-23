package proxy

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type healthScheduler struct {
	selector *roundRobin
	client   *http.Client
	path     string
	interval time.Duration
	timeout  time.Duration
}

type HealthRunner interface {
	Run(ctx context.Context)
}

func newHealthScheduler(
	selector *roundRobin,
	path string,
	interval time.Duration,
	timeout time.Duration,
) *healthScheduler {
	return &healthScheduler{
		selector: selector,
		client:   newHealthClient(),
		path:     path,
		interval: interval,
		timeout:  timeout,
	}
}

func (s *healthScheduler) Run(ctx context.Context) {
	var wg sync.WaitGroup

	for i, upstream := range s.selector.targets {
		wg.Add(1)

		go func(index int, target *url.URL) {
			defer wg.Done()

			s.monitor(ctx, index, target)
		}(i, upstream)
	}

	wg.Wait()
}

func (s *healthScheduler) monitor(
	ctx context.Context,
	index int,
	upstream *url.URL,
) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		if ctx.Err() != nil {
			return
		}

		err := checkHealth(
			ctx,
			s.client,
			upstream,
			s.path,
			s.timeout,
		)

		if ctx.Err() != nil {
			return
		}

		healthy := err == nil
		if s.selector.SetHealthy(index, healthy) {
			slog.Info(
				"upstream health changed",
				"upstream", upstream.Host,
				"healthy", healthy,
				"error", err,
			)
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
