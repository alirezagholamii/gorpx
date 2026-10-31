package main

import (
	"errors"
	"flag"
	"log"
	"net/http"

	"github.com/alirezagholamii/gorpx/internal/config"
	"github.com/alirezagholamii/gorpx/internal/server"
)

func main() {
	configPath := flag.String(
		"config",
		"gorpx.yaml",
		"path to configuration file",
	)
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	srv, err := server.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Gorpx listening on %s", srv.Addr)

	if err := srv.ListenAndServe(); err != nil &&
		!errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
