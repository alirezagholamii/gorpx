package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

func New(target *url.URL) http.Handler {
	return &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(target)
			r.SetXForwarded()
		},
	}
}
