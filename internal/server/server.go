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
	cfg = config.ApplyDefaults(cfg)

	if err := config.Validate(cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	readHeaderTimeout, err := time.ParseDuration(
		cfg.Server.ReadHeaderTimeout,
	)
	if err != nil {
		return nil, fmt.Errorf("parse read header timeout: %w", err)
	}

	idleTimeout, err := time.ParseDuration(
		cfg.Server.IdleTimeout,
	)
	if err != nil {
		return nil, fmt.Errorf("parse idle timeout: %w", err)
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
			middleware.RequestID(
				middleware.BodyLimit(mux, cfg.Server.MaxBodyBytes),
			),
			slog.Default(),
		),
		ReadHeaderTimeout: readHeaderTimeout,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    cfg.Server.MaxHeaderBytes,
	}, nil
}
