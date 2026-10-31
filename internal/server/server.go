package server

import (
	"fmt"
	"net/http"

	"github.com/alirezagholamii/gorpx/internal/config"
	"github.com/alirezagholamii/gorpx/internal/router"
)

func New(cfg config.Config) (*http.Server, error) {
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
		Addr:    cfg.Server.Address,
		Handler: mux,
	}, nil
}
