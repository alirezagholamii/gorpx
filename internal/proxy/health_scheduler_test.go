package proxy

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestHealthScheduler(t *testing.T) {
	var healthy atomic.Bool

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
		250*time.Millisecond,
		200*time.Millisecond,
	)

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})

	go func() {
		defer close(done)
		scheduler.Run(ctx)
	}()

	defer func() {
		cancel()
		waitForProbeSignal(t, done, "scheduler to stop")
	}()

	// The backend initially fails its health checks.
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
		if r.URL.Path != "/health" {
			t.Errorf("unexpected probe path: %q", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		close(started)
		select {
		case <-r.Context().Done():
			close(canceled)
		case <-release:
		}
	}))
	t.Cleanup(backend.Close)

	target, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatal(err)
	}
	selector := newRoundRobin([]*url.URL{target})
	scheduler := newHealthScheduler(selector, "/health", time.Minute, time.Minute)

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	t.Cleanup(func() {
		cancel()
		close(release)
		waitForProbeSignal(t, done, "scheduler to stop during cleanup")
	})
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

func TestHealthSchedulerLimitsStalledTLSConnections(t *testing.T) {
	handshakeStarted := make(chan struct{})
	releaseHandshake := make(chan struct{})
	release := sync.OnceFunc(func() { close(releaseHandshake) })
	var connections, handshakes atomic.Int32

	backend := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	backend.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			connections.Add(1)
		}
	}
	backend.TLS = &tls.Config{
		GetConfigForClient: func(*tls.ClientHelloInfo) (*tls.Config, error) {
			if handshakes.Add(1) == 1 {
				close(handshakeStarted)
				<-releaseHandshake
				return nil, errors.New("stalled test handshake released")
			}
			return nil, nil
		},
	}
	backend.StartTLS()
	t.Cleanup(backend.Close)
	t.Cleanup(release)

	target, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatal(err)
	}
	selector := newRoundRobin([]*url.URL{target})
	scheduler := newHealthScheduler(selector, "/health", 250*time.Millisecond, 200*time.Millisecond)
	transport := scheduler.client.Transport.(*http.Transport)
	transport.TLSClientConfig = backend.Client().Transport.(*http.Transport).TLSClientConfig.Clone()
	results := make(chan error, 8)
	scheduler.client.Transport = healthProbeRecorder{Transport: transport, results: results}

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		defer close(done)
		scheduler.Run(ctx)
	}()
	defer func() {
		cancel()
		release()
		waitForProbeSignal(t, done, "scheduler to stop after cancellation")
	}()

	waitForProbeSignal(t, handshakeStarted, "first TLS handshake to stall")
	for range 3 {
		select {
		case err := <-results:
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("stalled probe error = %v, want deadline exceeded", err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("stalled probe did not return by its deadline")
		}
	}
	if got := connections.Load(); got != 1 {
		t.Fatalf("stalled probes opened %d TCP connections, want 1", got)
	}
	waitForHealth(t, selector, false)

	// The failed handshake must free the slot for a new connection.
	release()
	waitForHealth(t, selector, true)
	if got := connections.Load(); got < 2 {
		t.Errorf("recovery did not open a new TCP connection, got %d total", got)
	}
}

type healthProbeRecorder struct {
	*http.Transport
	results chan<- error
}

func (r healthProbeRecorder) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := r.Transport.RoundTrip(req)
	select {
	case r.results <- err:
	default:
	}
	return resp, err
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

	deadline := time.After(3 * time.Second)

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
