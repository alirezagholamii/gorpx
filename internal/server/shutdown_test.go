package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestGracefulShutdown(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("completed"))
	})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	srv := &http.Server{Handler: handler}
	serverDone := make(chan error, 1)

	go func() {
		serverDone <- srv.Serve(listener)
	}()

	client := &http.Client{Timeout: 5 * time.Second}
	requestDone := make(chan error, 1)

	go func() {
		resp, err := client.Get("http://" + listener.Addr().String())
		if err != nil {
			requestDone <- err
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err == nil && string(body) != "completed" {
			err = fmt.Errorf("unexpected response body: %q", body)
		}
		requestDone <- err
	}()

	select {
	case <-started:
		// Request is now active.
	case <-time.After(5 * time.Second):
		t.Fatal("request did not start")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	shutdownDone := make(chan error, 1)

	go func() {
		shutdownDone <- srv.Shutdown(ctx)
	}()

	// Wait for the listener to close.
	// This confirms shutdown has started.
	select {
	case err := <-serverDone:
		if err != http.ErrServerClosed {
			t.Fatalf("server stopped unexpectedly: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not stop accepting connections")
	}

	select {
	case err := <-shutdownDone:
		t.Fatalf("shutdown returned before request finished: %v", err)
	default:
		// Expected: request is still active.
	}

	close(release)

	if err := <-requestDone; err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if err := <-shutdownDone; err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}
}
