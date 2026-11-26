package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gorpx.yaml")

	data := []byte(`
server:
  address: ":9090"
  body_read_timeout: 750ms

routes:
  - path: /api/
    upstreams:
      - http://localhost:3000
      - http://localhost:3001
`)

	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Server.Address != ":9090" {
		t.Errorf(
			"address = %q, want :9090",
			cfg.Server.Address,
		)
	}
	if cfg.Server.BodyReadTimeout != "750ms" {
		t.Errorf("body read timeout = %q, want 750ms", cfg.Server.BodyReadTimeout)
	}

	if len(cfg.Routes) != 1 {
		t.Fatalf("routes = %d, want 1", len(cfg.Routes))
	}

	if got := cfg.Routes[0].Upstreams; len(got) != 2 || got[0] != "http://localhost:3000" || got[1] != "http://localhost:3001" {
		t.Errorf("upstreams = %v, want both configured destinations in order", got)
	}

	if cfg.Routes[0].Path != "/api/" {
		t.Errorf(
			"path = %q, want /api/",
			cfg.Routes[0].Path,
		)
	}
}

func TestLoadDefaultTimeouts(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	content := []byte(`
server:
  address: ":8080"

routes:
  - path: /api/
    upstreams:
      - http://localhost:3000
`)

	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Server.ReadHeaderTimeout != "5s" {
		t.Errorf(
			"ReadHeaderTimeout = %q, want 5s",
			cfg.Server.ReadHeaderTimeout,
		)
	}
	if cfg.Server.BodyReadTimeout != "30s" {
		t.Errorf("body read timeout = %q, want 30s", cfg.Server.BodyReadTimeout)
	}

	if cfg.Server.IdleTimeout != "60s" {
		t.Errorf(
			"IdleTimeout = %q, want 60s",
			cfg.Server.IdleTimeout,
		)
	}
}

func TestLoadDefaultRequestLimits(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	content := []byte(`
server:
  address: ":8080"

routes:
  - path: /api/
    upstreams:
      - http://localhost:3000
`)

	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Server.MaxHeaderBytes != 16<<10 {
		t.Errorf(
			"MaxHeaderBytes = %d, want %d",
			cfg.Server.MaxHeaderBytes,
			16<<10,
		)
	}

	if cfg.Server.MaxBodyBytes != 10<<20 {
		t.Errorf(
			"MaxBodyBytes = %d, want %d",
			cfg.Server.MaxBodyBytes,
			10<<20,
		)
	}
}
