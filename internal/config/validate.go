package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

func Validate(cfg Config) error {
	_, port, err := net.SplitHostPort(cfg.Server.Address)
	if err != nil {
		return fmt.Errorf("invalid server address: %w", err)
	}

	timeouts := []struct {
		name  string
		value string
	}{
		{"read_header_timeout", cfg.Server.ReadHeaderTimeout},
		{"idle_timeout", cfg.Server.IdleTimeout},
	}

	for _, timeout := range timeouts {
		if timeout.value == "" {
			continue
		}

		duration, err := time.ParseDuration(timeout.value)
		if err != nil {
			return fmt.Errorf(
				"invalid %s %q: %w",
				timeout.name,
				timeout.value,
				err,
			)
		}

		if duration <= 0 {
			return fmt.Errorf(
				"%s must be greater than zero",
				timeout.name,
			)
		}
	}

	if cfg.Server.MaxHeaderBytes < 0 {
		return fmt.Errorf(
			"max_header_bytes must not be negative: %d",
			cfg.Server.MaxHeaderBytes,
		)
	}

	if cfg.Server.MaxBodyBytes < 0 {
		return fmt.Errorf(
			"max_body_bytes must not be negative: %d",
			cfg.Server.MaxBodyBytes,
		)
	}

	number, err := strconv.Atoi(port)
	if err != nil || number < 1 || number > 65535 {
		return fmt.Errorf("invalid server port: %q", port)
	}

	if len(cfg.Routes) == 0 {
		return fmt.Errorf("at least one route is required")
	}

	seen := make(map[string]bool)

	for _, route := range cfg.Routes {
		if !strings.HasPrefix(route.Path, "/") ||
			!strings.HasSuffix(route.Path, "/") {
			return fmt.Errorf("invalid route path: %q", route.Path)
		}

		if seen[route.Path] {
			return fmt.Errorf("duplicate route: %q", route.Path)
		}
		seen[route.Path] = true

		hasUpstream := route.Upstream != ""
		hasRoot := route.Root != ""
		if hasUpstream == hasRoot {
			return fmt.Errorf(
				"route %q must define exactly one of upstream or root",
				route.Path,
			)
		}

		if hasRoot {
			info, err := os.Stat(route.Root)
			if err != nil {
				return fmt.Errorf(
					"route %q: invalid static root %q: %w",
					route.Path,
					route.Root,
					err,
				)
			}

			if !info.IsDir() {
				return fmt.Errorf(
					"route %q: static root %q is not a directory",
					route.Path,
					route.Root,
				)
			}
		}

		if hasUpstream {
			target, err := url.Parse(route.Upstream)

			if err != nil {
				return fmt.Errorf(
					"route %q: invalid upstream %q: %w",
					route.Path,
					route.Upstream,
					err,
				)
			}

			if strings.HasSuffix(target.Host, ":") {
				return fmt.Errorf(
					"route %q: empty upstream port",
					route.Path,
				)
			}

			if port := target.Port(); port != "" {
				number, err := strconv.Atoi(port)
				if err != nil || number < 1 || number > 65535 {
					return fmt.Errorf(
						"route %q: invalid upstream port %q",
						route.Path,
						port,
					)
				}
			}

			if (target.Scheme != "http" && target.Scheme != "https") ||
				target.Hostname() == "" ||
				target.User != nil ||
				target.RawQuery != "" ||
				target.ForceQuery ||
				target.Fragment != "" ||
				target.RawFragment != "" ||
				(target.Path != "" && target.Path != "/") {
				return fmt.Errorf(
					"route %q: invalid upstream %q",
					route.Path,
					route.Upstream,
				)
			}
		}
	}

	return nil
}
