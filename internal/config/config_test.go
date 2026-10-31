package config

import "testing"

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.Server.Address != ":8080" {
		t.Errorf(
			"default address = %q, want %q",
			cfg.Server.Address,
			":8080",
		)
	}
}
