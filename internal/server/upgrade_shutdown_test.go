package server

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alirezagholamii/gorpx/internal/config"
)

func TestIntegrationUpgradeShutdown(t *testing.T) {
	t.Run("drains active upgrade", func(t *testing.T) {
		client, stream, startShutdown, upstreamDone := openUpgrade(t, nil)
		requireUpgrade(t, stream)
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()
		shutdownDone := startShutdown(ctx)

		if _, err := io.WriteString(client, "ping\n"); err != nil {
			t.Fatal(err)
		}
		if response, err := stream.ReadString('\n'); err != nil || response != "ping\n" {
			t.Fatalf("upgrade echo during shutdown = %q, %v", response, err)
		}
		select {
		case err := <-shutdownDone:
			t.Fatalf("shutdown returned with upgrade still active: %v", err)
		default:
		}

		if err := client.CloseWrite(); err != nil {
			t.Fatal(err)
		}
		requireUpgradeEOF(t, stream)
		if err := awaitUpgradeResult(t, upstreamDone); err != nil {
			t.Fatalf("upstream did not finish cleanly: %v", err)
		}
		if err := awaitUpgradeResult(t, shutdownDone); err != nil {
			t.Fatalf("graceful shutdown failed: %v", err)
		}
	})

	t.Run("closes upgrade at deadline", func(t *testing.T) {
		_, stream, startShutdown, upstreamDone := openUpgrade(t, nil)
		requireUpgrade(t, stream)
		ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
		defer cancel()

		if err := awaitUpgradeResult(t, startShutdown(ctx)); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("shutdown error = %v, want deadline exceeded", err)
		}
		requireUpgradeEOF(t, stream)
		if err := awaitUpgradeResult(t, upstreamDone); err != nil {
			t.Fatalf("upstream did not close cleanly: %v", err)
		}
	})

	t.Run("rejects late hijack", func(t *testing.T) {
		releaseUpgrade := make(chan struct{})
		_, stream, startShutdown, upstreamDone := openUpgrade(t, releaseUpgrade)
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()
		shutdownDone := startShutdown(ctx)
		close(releaseUpgrade)

		resp, err := http.ReadResponse(stream, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadGateway {
			t.Fatalf("late upgrade status = %d, want 502", resp.StatusCode)
		}
		if err := awaitUpgradeResult(t, upstreamDone); err != nil {
			t.Fatalf("upstream did not close cleanly: %v", err)
		}
		if err := awaitUpgradeResult(t, shutdownDone); err != nil {
			t.Fatalf("graceful shutdown failed: %v", err)
		}
	})
}

func openUpgrade(t *testing.T, releaseUpgrade <-chan struct{}) (
	*net.TCPConn, *bufio.Reader, func(context.Context) <-chan error, <-chan error,
) {
	t.Helper()
	testCtx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	t.Cleanup(cancel)
	deadline, _ := testCtx.Deadline()
	upstreamStarted := make(chan error, 1)
	upstreamDone := make(chan error, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		conn, rw, err := http.NewResponseController(w).Hijack()
		upstreamStarted <- err
		if err != nil {
			return
		}
		defer conn.Close()
		if releaseUpgrade != nil {
			select {
			case <-releaseUpgrade:
			case <-testCtx.Done():
				upstreamDone <- testCtx.Err()
				return
			}
		}
		upstreamDone <- echoUpgrade(conn, rw, deadline)
	}))
	t.Cleanup(upstream.Close)

	srv, err := New(config.Config{
		Server: config.Server{Address: "127.0.0.1:8080"},
		Routes: []config.Route{{Path: "/upgrade/", Upstreams: []string{upstream.URL}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	shutdown := TrackUpgrades(srv)
	t.Cleanup(func() {
		cancel()
		shutdown(testCtx)
	})
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	serveDone := make(chan error, 1)
	go func() { serveDone <- srv.Serve(listener) }()

	dialer := net.Dialer{Deadline: deadline}
	conn, err := dialer.DialContext(testCtx, "tcp4", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	client := conn.(*net.TCPConn)
	t.Cleanup(func() { client.Close() })
	if err := client.SetDeadline(deadline); err != nil {
		t.Fatal(err)
	}
	if _, err := fmt.Fprintf(client, "GET /upgrade/ HTTP/1.1\r\nHost: %s\r\nConnection: Upgrade\r\nUpgrade: echo\r\n\r\n", listener.Addr()); err != nil {
		t.Fatal(err)
	}
	if err := awaitUpgradeResult(t, upstreamStarted); err != nil {
		t.Fatalf("upstream hijack failed: %v", err)
	}

	startShutdown := func(ctx context.Context) <-chan error {
		t.Helper()
		shutdownDone := make(chan error, 1)
		go func() { shutdownDone <- shutdown(ctx) }()
		if err := awaitUpgradeResult(t, serveDone); !errors.Is(err, http.ErrServerClosed) {
			t.Fatalf("Serve error = %v, want server closed", err)
		}
		return shutdownDone
	}
	return client, bufio.NewReader(client), startShutdown, upstreamDone
}

func echoUpgrade(conn net.Conn, rw *bufio.ReadWriter, deadline time.Time) error {
	if err := conn.SetDeadline(deadline); err != nil {
		return err
	}
	if _, err := rw.WriteString("HTTP/1.1 101 Switching Protocols\r\nConnection: Upgrade\r\nUpgrade: echo\r\n\r\n"); err != nil {
		return err
	}
	if err := rw.Flush(); err != nil {
		return err
	}
	_, err := io.Copy(conn, rw)
	return err
}

func requireUpgrade(t *testing.T, stream *bufio.Reader) {
	t.Helper()
	resp, err := http.ReadResponse(stream, nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusSwitchingProtocols || resp.Header.Get("Upgrade") != "echo" {
		t.Fatalf("upgrade response = %s, Upgrade: %q", resp.Status, resp.Header.Get("Upgrade"))
	}
}

func requireUpgradeEOF(t *testing.T, stream *bufio.Reader) {
	t.Helper()
	if _, err := stream.ReadByte(); !errors.Is(err, io.EOF) {
		t.Fatalf("upgrade client read error = %v, want EOF", err)
	}
}

func awaitUpgradeResult(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for upgrade test result")
		return nil
	}
}
