package proxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"
)

func TestHealthScheduler(t *testing.T) {
	var healthy atomic.Bool
	healthy.Store(true)

	backend := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/health" {
				t.Errorf("unexpected path: %q", r.URL.Path)
			}

			if !healthy.Load() {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}

			w.WriteHeader(http.StatusOK)
		},
	))
	defer backend.Close()

	target, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatal(err)
	}

	selector := newRoundRobin([]*url.URL{target})

	scheduler := newHealthScheduler(
		selector,
		"/health",
		20*time.Millisecond,
		100*time.Millisecond,
	)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		defer close(done)
		scheduler.Run(ctx)
	}()

	defer func() {
		cancel()
		<-done
	}()

	// Simulate a failing upstream.
	healthy.Store(false)

	waitForHealth(t, selector, false)

	// Simulate recovery.
	healthy.Store(true)

	waitForHealth(t, selector, true)
}

func waitForHealth(
	t *testing.T,
	selector *roundRobin,
	want bool,
) {
	t.Helper()

	deadline := time.After(time.Second)

	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()

	for {
		if _, selectable := selector.Next(); selectable == want {
			return
		}

		select {
		case <-deadline:
			t.Fatalf(
				"upstream health did not become %v",
				want,
			)
		case <-ticker.C:
		}
	}
}
