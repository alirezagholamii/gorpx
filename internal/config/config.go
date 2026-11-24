package config

import (
	"fmt"
	"strings"
)

type Server struct {
	Address           string `yaml:"address"`
	ReadHeaderTimeout string `yaml:"read_header_timeout"`
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
			IdleTimeout:       "60s",
			MaxHeaderBytes:    16 << 10,
			MaxBodyBytes:      10 << 20,
		},
	}
}

func ValidateRoutePath(path string) error {
	if !strings.HasPrefix(path, "/") ||
		!strings.HasSuffix(path, "/") {
		return fmt.Errorf("invalid route path: %q", path)
	}

	if strings.ContainsAny(path, "{}") {
		return fmt.Errorf(
			"route path %q must not contain wildcard syntax",
			path,
		)
	}

	// ServeMux unescapes pattern segments, so differently spelled paths
	// such as /api/ and /%61pi/ would otherwise conflict at registration.
	if strings.Contains(path, "%") {
		return fmt.Errorf("route path %q must not contain percent escapes", path)
	}

	return nil
}
