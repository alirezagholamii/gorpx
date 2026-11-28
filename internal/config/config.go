package config

import (
	"fmt"
	"path"
	"strings"
)

type Server struct {
	Address           string `yaml:"address"`
	ReadHeaderTimeout string `yaml:"read_header_timeout"`
	BodyReadTimeout   string `yaml:"body_read_timeout"`
	IdleTimeout       string `yaml:"idle_timeout"`
	MaxHeaderBytes    int    `yaml:"max_header_bytes"`
	MaxBodyBytes      int64  `yaml:"max_body_bytes"`
}

type HealthCheck struct {
	Path     string `yaml:"path"`
	Interval string `yaml:"interval"`
	Timeout  string `yaml:"timeout"`
}

type Route struct {
	Path                  string       `yaml:"path"`
	Upstreams             []string     `yaml:"upstreams"`
	Root                  string       `yaml:"root"`
	ResponseHeaderTimeout string       `yaml:"response_header_timeout"`
	HealthCheck           *HealthCheck `yaml:"health_check"`
}

type Config struct {
	Server Server  `yaml:"server"`
	Routes []Route `yaml:"routes"`
}

func Default() Config {
	return Config{
		Server: Server{
			Address:           ":8080",
			ReadHeaderTimeout: "5s",
			BodyReadTimeout:   "30s",
			IdleTimeout:       "60s",
			MaxHeaderBytes:    16 << 10,
			MaxBodyBytes:      10 << 20,
		},
	}
}

func ValidateRoutePath(prefix string) error {
	if !strings.HasPrefix(prefix, "/") ||
		!strings.HasSuffix(prefix, "/") {
		return fmt.Errorf("invalid route path: %q", prefix)
	}

	if strings.ContainsAny(prefix, "{}") {
		return fmt.Errorf(
			"route path %q must not contain wildcard syntax",
			prefix,
		)
	}

	// ServeMux unescapes pattern segments, so differently spelled paths
	// such as /api/ and /%61pi/ would otherwise conflict at registration.
	if strings.Contains(prefix, "%") {
		return fmt.Errorf("route path %q must not contain percent escapes", prefix)
	}

	if strings.ContainsAny(prefix, " \t") {
		return fmt.Errorf("route path %q must not contain spaces or tabs", prefix)
	}

	// Restore the subtree slash after cleaning, including for the root prefix.
	cleanPrefix := strings.TrimSuffix(path.Clean(prefix), "/") + "/"
	if prefix != cleanPrefix {
		return fmt.Errorf("route path %q must not contain dot segments or repeated slashes", prefix)
	}

	return nil
}
