package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/alirezagholamii/gorpx/internal/config"
	"github.com/alirezagholamii/gorpx/internal/server"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	configPath := flag.String("config", "gorpx.yaml", "config file path")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	srv, runners, err := server.NewWithHealth(cfg)
	if err != nil {
		log.Fatal(err)
	}

	signalCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

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
		serverErr <- srv.ListenAndServe()
	}()

	log.Printf("Gorpx listening on %s", srv.Addr)

	select {
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP server failed: %v", err)
			return
		}
		return

	case <-signalCtx.Done():
		log.Println("Shutting down Gorpx...")
	}

	stop()

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Graceful shutdown failed: %v", err)
		return
	}

	log.Println("Gorpx stopped")
}
