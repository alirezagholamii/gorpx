package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/alirezagholamii/gorpx/internal/config"
	"github.com/alirezagholamii/gorpx/internal/middleware"
	"github.com/alirezagholamii/gorpx/internal/router"
)

func New(cfg config.Config) (*http.Server, error) {
	if err := config.Validate(cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	routes, err := router.New(cfg.Routes)
	if err != nil {
		return nil, fmt.Errorf("create router: %w", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello from Gorpx!")
	})

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "OK")
	})

	mux.Handle("/", routes)

	return &http.Server{
		Addr: cfg.Server.Address,
		Handler: middleware.Logging(
			middleware.BodyLimit(mux),
			slog.Default(),
		), ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 16 << 10,
	}, nil
}
