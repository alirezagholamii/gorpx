package router

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/alirezagholamii/gorpx/internal/config"
	"github.com/alirezagholamii/gorpx/internal/proxy"
	"github.com/alirezagholamii/gorpx/internal/static"
)

func NewWithHealth(
	routes []config.Route,
) (http.Handler, []proxy.HealthRunner, error) {
	mux := http.NewServeMux()
	seen := make(map[string]bool)
	var runners []proxy.HealthRunner

	for _, route := range routes {
		if err := config.ValidateRoutePath(route.Path); err != nil {
			return nil, nil, err
		}

		if seen[route.Path] {
			return nil, nil, fmt.Errorf("duplicate route: %q", route.Path)
		}
		seen[route.Path] = true

		hasUpstreams := len(route.Upstreams) > 0
		hasRoot := route.Root != ""

		if hasUpstreams == hasRoot {
			return nil, nil, fmt.Errorf(
				"route %q must define exactly one of upstreams or root",
				route.Path,
			)
		}

		if err := config.ValidateRouteHealthCheck(route); err != nil {
			return nil, nil, err
		}

		if hasRoot {
			if route.ResponseHeaderTimeout != "" {
				return nil, nil, fmt.Errorf(
					"route %q: response_header_timeout is only valid for upstream routes",
					route.Path,
				)
			}

			mux.Handle(route.Path, static.New(route.Path, route.Root))
			continue
		}

		targets := make([]*url.URL, 0, len(route.Upstreams))

		for _, upstream := range route.Upstreams {
			target, err := url.Parse(upstream)
			if err != nil {
				return nil, nil, fmt.Errorf(
					"route %q: parse upstream %q: %w",
					route.Path,
					upstream,
					err,
				)
			}

			if (target.Scheme != "http" && target.Scheme != "https") ||
				target.Hostname() == "" ||
				target.User != nil ||
				target.Opaque != "" ||
				target.RawQuery != "" ||
				target.ForceQuery ||
				target.Fragment != "" ||
				target.RawFragment != "" ||
				(target.Path != "" && target.Path != "/") ||
				strings.HasSuffix(target.Host, ":") {
				return nil, nil, fmt.Errorf(
					"route %q: invalid upstream %q",
					route.Path,
					upstream,
				)
			}

			if port := target.Port(); port != "" {
				number, err := strconv.Atoi(port)
				if err != nil || number < 1 || number > 65535 {
					return nil, nil, fmt.Errorf(
						"route %q: invalid upstream port %q",
						route.Path,
						port,
					)
				}
			}

			targets = append(targets, target)
		}

		responseHeaderTimeout := 15 * time.Second

		if route.ResponseHeaderTimeout != "" {
			parsed, err := time.ParseDuration(route.ResponseHeaderTimeout)
			if err != nil {
				return nil, nil, fmt.Errorf(
					"route %q: parse response header timeout: %w",
					route.Path,
					err,
				)
			}

			if parsed <= 0 {
				return nil, nil, fmt.Errorf(
					"route %q: response header timeout must be positive",
					route.Path,
				)
			}

			responseHeaderTimeout = parsed
		}

		handler, runner, err := proxy.NewWithHealth(
			targets,
			responseHeaderTimeout,
			route.HealthCheck,
		)
		if err != nil {
			return nil, nil, fmt.Errorf(
				"route %q: create proxy: %w",
				route.Path,
				err,
			)
		}

		mux.Handle(route.Path, handler)

		if runner != nil {
			runners = append(runners, runner)
		}
	}

	return mux, runners, nil
}

func New(routes []config.Route) (http.Handler, error) {
	for _, route := range routes {
		if route.HealthCheck != nil {
			return nil, fmt.Errorf("route %q: health_check requires router.NewWithHealth()", route.Path)
		}
	}

	handler, _, err := NewWithHealth(routes)
	return handler, err
}
