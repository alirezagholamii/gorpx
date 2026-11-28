package server

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
)

type upgrades struct {
	mu       sync.Mutex
	stopping bool
	active   map[*upgradeWriter]struct{}
	drained  chan struct{}
}

// TrackUpgrades installs hijack tracking before srv starts serving. It expects
// handlers such as ReverseProxy to own the connection until they return.
// The returned shutdown function drains HTTP requests and upgrades on one context.
func TrackUpgrades(srv *http.Server) func(context.Context) error {
	tracker := &upgrades{
		active:  make(map[*upgradeWriter]struct{}),
		drained: make(chan struct{}),
	}
	next := srv.Handler
	if next == nil {
		next = http.DefaultServeMux
	}
	srv.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithCancel(r.Context())
		writer := &upgradeWriter{ResponseWriter: w, tracker: tracker, cancel: cancel}
		defer func() {
			cancel()
			if writer.conn != nil {
				tracker.finished(writer)
			}
		}()
		next.ServeHTTP(writer, r.WithContext(ctx))
	})
	return func(ctx context.Context) error {
		tracker.stop()
		httpErr := srv.Shutdown(ctx)
		var closeErr error
		if httpErr != nil {
			if err := srv.Close(); err != nil {
				closeErr = fmt.Errorf("force-close HTTP server: %w", err)
			}
		}

		upgradeErr := tracker.wait(ctx)
		if httpErr == nil {
			httpErr = upgradeErr
		}
		if httpErr != nil {
			shutdownErr := fmt.Errorf("graceful shutdown failed: %w", httpErr)
			if closeErr != nil {
				return errors.Join(shutdownErr, closeErr)
			}
			return shutdownErr
		}
		return nil
	}
}

type upgradeWriter struct {
	http.ResponseWriter
	tracker *upgrades
	cancel  context.CancelFunc
	conn    net.Conn
}

func (w *upgradeWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *upgradeWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	tracker := w.tracker
	tracker.mu.Lock()
	defer tracker.mu.Unlock()

	// Shutdown must see the connection in active as soon as net/http
	// hands it off, or reject the handoff before it happens.
	if tracker.stopping {
		return nil, nil, errors.New("server is shutting down")
	}
	conn, rw, err := http.NewResponseController(w.ResponseWriter).Hijack()
	if err != nil {
		return nil, nil, err
	}
	w.conn = conn
	tracker.active[w] = struct{}{}
	return conn, rw, nil
}

func (u *upgrades) stop() {
	u.mu.Lock()
	defer u.mu.Unlock()

	if !u.stopping {
		u.stopping = true
		if len(u.active) == 0 {
			close(u.drained)
		}
	}
}

func (u *upgrades) finished(w *upgradeWriter) {
	u.mu.Lock()
	defer u.mu.Unlock()

	delete(u.active, w)
	if u.stopping && len(u.active) == 0 {
		close(u.drained)
	}
}

func (u *upgrades) wait(ctx context.Context) error {
	select {
	case <-u.drained:
		return nil
	case <-ctx.Done():
		u.mu.Lock()
		defer u.mu.Unlock()

		if len(u.active) == 0 {
			return nil
		}
		for w := range u.active {
			// ReverseProxy closes the upgraded upstream on cancellation.
			w.cancel()
			w.conn.Close()
		}
		return ctx.Err()
	}
}
