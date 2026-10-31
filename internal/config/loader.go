package config

import (
	"fmt"
	"io"
	"os"

	"go.yaml.in/yaml/v3"
)

func Load(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("open config: %w", err)
	}
	defer file.Close()

	cfg := Default()

	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)

	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return Config{}, fmt.Errorf(
			"config must contain exactly one YAML document",
		)
	}

	if cfg.Server.Address == "" {
		return Config{}, fmt.Errorf("server address is required")
	}

	if len(cfg.Routes) == 0 {
		return Config{}, fmt.Errorf("at least one route is required")
	}

	return cfg, nil
}
