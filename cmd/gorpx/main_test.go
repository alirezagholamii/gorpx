package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"syscall"
	"testing"
	"time"

	"github.com/alirezagholamii/gorpx/internal/proxy"
)

type healthRunnerFunc func(context.Context)

func (f healthRunnerFunc) Run(ctx context.Context) {
	f(ctx)
}

func TestServeBindFailure(t *testing.T) {
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer occupied.Close()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	started := make(chan struct{})
	runner := healthRunnerFunc(func(context.Context) {
		close(started)
	})
	srv := &http.Server{Addr: occupied.Addr().String()}

	err = serve(ctx, cancel, srv, []proxy.HealthRunner{runner}, time.Second)
	if !errors.Is(err, syscall.EADDRINUSE) {
		t.Fatalf("serve() error = %v, want address already in use", err)
	}
	select {
	case <-started:
		t.Fatal("health worker started after bind failure")
	default:
	}
}

func TestServeShutdownTimeout(t *testing.T) {
	testCtx, cancelTest := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancelTest()
	signalCtx, signal := context.WithCancel(testCtx)
	defer signal()

	addr := make(chan string, 1)
	requestStarted := make(chan struct{})
	requestStopped := make(chan struct{})
	srv := &http.Server{
		Addr: "127.0.0.1:0",
		BaseContext: func(listener net.Listener) context.Context {
			addr <- listener.Addr().String()
			return testCtx
		},
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			close(requestStarted)
			<-r.Context().Done()
			close(requestStopped)
		}),
	}

	workerStarted := make(chan struct{})
	workerStopped := make(chan struct{})
	runner := healthRunnerFunc(func(ctx context.Context) {
		close(workerStarted)
		<-ctx.Done()
		close(workerStopped)
	})

	serveDone := make(chan error, 1)
	go func() {
		serveDone <- serve(signalCtx, signal, srv, []proxy.HealthRunner{runner}, 0)
	}()

	var endpoint string
	select {
	case address := <-addr:
		endpoint = "http://" + address
	case err := <-serveDone:
		t.Fatalf("server exited before starting: %v", err)
	case <-testCtx.Done():
		t.Fatal("server did not start")
	}

	req, err := http.NewRequestWithContext(testCtx, http.MethodGet, endpoint, nil)
	if err != nil {
		t.Fatal(err)
	}
	requestDone := make(chan error, 1)
	go func() {
		resp, err := http.DefaultClient.Do(req)
		if resp != nil {
			resp.Body.Close()
		}
		requestDone <- err
	}()

	for _, started := range []<-chan struct{}{workerStarted, requestStarted} {
		select {
		case <-started:
		case <-testCtx.Done():
			t.Fatal("health worker or HTTP request did not start")
		}
	}
	signal()

	select {
	case err := <-serveDone:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("serve() error = %v, want shutdown deadline exceeded", err)
		}
	case <-testCtx.Done():
		t.Fatal("server did not exit")
	}
	select {
	case <-workerStopped:
	default:
		t.Fatal("serve returned before the health worker stopped")
	}
	select {
	case <-requestStopped:
	case <-testCtx.Done():
		t.Fatal("force-close did not cancel the active request")
	}
	select {
	case err := <-requestDone:
		if err == nil || testCtx.Err() != nil {
			t.Fatalf("HTTP request error = %v, want a closed connection before the test deadline", err)
		}
	case <-testCtx.Done():
		t.Fatal("client connection was left open")
	}
}
