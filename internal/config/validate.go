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

	number, err := strconv.Atoi(port)
	if err != nil || number < 1 || number > 65535 {
		return fmt.Errorf("invalid server port: %q", port)
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

	if len(cfg.Routes) == 0 {
		return fmt.Errorf("at least one route is required")
	}

	seen := make(map[string]bool)

	for _, route := range cfg.Routes {
		if err := ValidateRoutePath(route.Path); err != nil {
			return err
		}

		if seen[route.Path] {
			return fmt.Errorf("duplicate route: %q", route.Path)
		}
		seen[route.Path] = true

		if route.Upstreams != nil && len(route.Upstreams) == 0 {
			return fmt.Errorf(
				"route %q: upstreams must not be empty",
				route.Path,
			)
		}

		hasUpstreams := len(route.Upstreams) > 0
		hasRoot := route.Root != ""

		if hasUpstreams == hasRoot {
			return fmt.Errorf(
				"route %q must define exactly one of upstreams or root",
				route.Path,
			)
		}

		if hasRoot && route.ResponseHeaderTimeout != "" {
			return fmt.Errorf(
				"route %q: response_header_timeout is only valid for upstream routes",
				route.Path,
			)
		}

		if route.ResponseHeaderTimeout != "" {
			duration, err := time.ParseDuration(route.ResponseHeaderTimeout)
			if err != nil {
				return fmt.Errorf(
					"route %q: invalid response_header_timeout %q: %w",
					route.Path,
					route.ResponseHeaderTimeout,
					err,
				)
			}

			if duration <= 0 {
				return fmt.Errorf(
					"route %q: response_header_timeout must be greater than zero",
					route.Path,
				)
			}
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

		for _, upstream := range route.Upstreams {
			if err := validateUpstreamURL(upstream); err != nil {
				return fmt.Errorf(
					"route %q: %w",
					route.Path,
					err,
				)
			}
		}

		if hc := route.HealthCheck; hc != nil {
			if hasRoot {
				return fmt.Errorf(
					"route %q: health_check is only valid for upstream routes",
					route.Path,
				)
			}

			if !strings.HasPrefix(hc.Path, "/") ||
				strings.HasPrefix(hc.Path, "//") ||
				strings.ContainsAny(hc.Path, "?#{}%\\ \t\r\n") {
				return fmt.Errorf(
					"route %q: invalid health check path %q",
					route.Path,
					hc.Path,
				)
			}

			for _, segment := range strings.Split(hc.Path, "/") {
				if segment == "." || segment == ".." {
					return fmt.Errorf(
						"route %q: invalid health check path %q",
						route.Path,
						hc.Path,
					)
				}
			}

			interval, err := time.ParseDuration(hc.Interval)
			if err != nil || interval <= 0 {
				return fmt.Errorf(
					"route %q: invalid health check interval %q",
					route.Path,
					hc.Interval,
				)
			}

			timeout, err := time.ParseDuration(hc.Timeout)
			if err != nil || timeout <= 0 {
				return fmt.Errorf(
					"route %q: invalid health check timeout %q",
					route.Path,
					hc.Timeout,
				)
			}

			if timeout > interval {
				return fmt.Errorf(
					"route %q: health check timeout must not exceed interval",
					route.Path,
				)
			}
		}

	}

	return nil
}

func validateUpstreamURL(rawURL string) error {
	target, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid upstream %q: %w", rawURL, err)
	}

	if (target.Scheme != "http" && target.Scheme != "https") ||
		target.Hostname() == "" ||
		target.User != nil ||
		target.Opaque != "" ||
		target.RawQuery != "" ||
		target.ForceQuery ||
		target.Fragment != "" ||
		target.RawFragment != "" ||
		(target.Path != "" && target.Path != "/") {
		return fmt.Errorf("invalid upstream %q", rawURL)
	}

	if strings.HasSuffix(target.Host, ":") {
		return fmt.Errorf("invalid upstream %q: empty port", rawURL)
	}

	if port := target.Port(); port != "" {
		number, err := strconv.Atoi(port)
		if err != nil || number < 1 || number > 65535 {
			return fmt.Errorf("invalid upstream %q: invalid port %q", rawURL, port)
		}
	}

	return nil
}
