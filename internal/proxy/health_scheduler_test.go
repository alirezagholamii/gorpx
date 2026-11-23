package proxy

import (
	"context"
	"net"
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

func TestHealthSchedulerCancelsActiveProbe(t *testing.T) {
	started := make(chan struct{})
	canceled := make(chan struct{})
	release := make(chan struct{})
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		select {
		case <-r.Context().Done():
			close(canceled)
		case <-release:
		}
	}))
	t.Cleanup(backend.Close)
	defer close(release)

	target, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatal(err)
	}
	selector := newRoundRobin([]*url.URL{target})
	scheduler := newHealthScheduler(selector, "/health", time.Minute, time.Minute)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		scheduler.Run(ctx)
	}()

	waitForProbeSignal(t, started, "active probe to start")
	cancel()
	waitForProbeSignal(t, canceled, "backend to observe probe cancellation")
	waitForProbeSignal(t, done, "scheduler to stop after cancellation")
}

func TestHealthSchedulerClosesIdleConnections(t *testing.T) {
	idle := make(chan struct{})
	closed := make(chan struct{})
	backend := httptest.NewUnstartedServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
	))
	backend.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		switch state {
		case http.StateIdle:
			close(idle)
		case http.StateClosed:
			close(closed)
		}
	}
	backend.Start()
	t.Cleanup(backend.Close)

	target, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatal(err)
	}
	selector := newRoundRobin([]*url.URL{target})
	selector.SetHealthy(0, false)
	scheduler := newHealthScheduler(selector, "/health", time.Minute, time.Minute)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		scheduler.Run(ctx)
	}()

	waitForProbeSignal(t, idle, "probe connection to become idle")
	// Wait for the scheduler to process the completed probe before canceling.
	waitForHealth(t, selector, true)
	select {
	case <-closed:
		t.Fatal("probe connection closed before scheduler cancellation")
	default:
	}

	cancel()
	waitForProbeSignal(t, done, "scheduler to stop after cancellation")
	waitForProbeSignal(t, closed, "idle probe connection to close")
}

func waitForProbeSignal(t *testing.T, signal <-chan struct{}, event string) {
	t.Helper()

	select {
	case <-signal:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", event)
	}
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
