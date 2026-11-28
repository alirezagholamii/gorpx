package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/alirezagholamii/gorpx/internal/config"
	"github.com/alirezagholamii/gorpx/internal/proxy"
	"github.com/alirezagholamii/gorpx/internal/server"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := run(); err != nil {
		slog.Error("Gorpx failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	configPath := flag.String("config", "gorpx.yaml", "config file path")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	srv, runners, err := server.NewWithHealth(cfg)
	if err != nil {
		return err
	}

	signalCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	return serve(signalCtx, stop, srv, runners, 10*time.Second)
}

func serve(
	signalCtx context.Context,
	stop context.CancelFunc,
	srv *http.Server,
	runners []proxy.HealthRunner,
	shutdownTimeout time.Duration,
) error {
	listener, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", srv.Addr, err)
	}

	shutdown := server.TrackUpgrades(srv)
	healthCtx, cancelHealth := context.WithCancel(context.Background())

	var wg sync.WaitGroup

	for _, runner := range runners {
		wg.Add(1)

		go func() {
			defer wg.Done()
			runner.Run(healthCtx)
		}()
	}

	defer func() {
		cancelHealth()
		wg.Wait()
	}()

	serverErr := make(chan error, 1)

	go func() {
		err := srv.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- fmt.Errorf("HTTP server failed: %w", err)
			return
		}
		serverErr <- nil
	}()

	slog.Info("Gorpx listening", "address", listener.Addr().String())
	select {
	case err := <-serverErr:
		return err

	case <-signalCtx.Done():
		slog.Info("Shutting down Gorpx...")
	}

	stop()

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		shutdownTimeout,
	)
	defer cancel()

	if err := shutdown(shutdownCtx); err != nil {
		return err
	}

	if err := <-serverErr; err != nil {
		return err
	}

	slog.Info("Gorpx stopped")
	return nil
}
