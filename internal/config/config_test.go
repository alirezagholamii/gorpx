package config

import "testing"

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.Server.ReadHeaderTimeout != "5s" {
		t.Errorf(
			"ReadHeaderTimeout = %q, want 5s",
			cfg.Server.ReadHeaderTimeout,
		)
	}

	if cfg.Server.IdleTimeout != "60s" {
		t.Errorf(
			"IdleTimeout = %q, want 60s",
			cfg.Server.IdleTimeout,
		)
	}

	if cfg.Server.Address != ":8080" {
		t.Errorf(
			"default address = %q, want %q",
			cfg.Server.Address,
			":8080",
		)
	}
}
