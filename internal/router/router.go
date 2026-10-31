package router

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/alirezagholamii/gorpx/internal/config"
	"github.com/alirezagholamii/gorpx/internal/proxy"
)

func New(routes []config.Route) (http.Handler, error) {
	mux := http.NewServeMux()
	seen := make(map[string]bool)

	for _, route := range routes {
		if !strings.HasPrefix(route.Path, "/") ||
			!strings.HasSuffix(route.Path, "/") {
			return nil, fmt.Errorf("invalid route path: %q", route.Path)
		}

		if seen[route.Path] {
			return nil, fmt.Errorf("duplicate route: %q", route.Path)
		}
		seen[route.Path] = true

		target, err := url.Parse(route.Upstream)
		if err != nil {
			return nil, fmt.Errorf("parse upstream %q: %w", route.Upstream, err)
		}

		if (target.Scheme != "http" && target.Scheme != "https") ||
			target.Host == "" ||
			target.User != nil ||
			target.RawQuery != "" ||
			target.Fragment != "" ||
			(target.Path != "" && target.Path != "/") {
			return nil, fmt.Errorf("invalid upstream: %q", route.Upstream)
		}

		mux.Handle(route.Path, proxy.New(target))
	}

	return mux, nil
}
