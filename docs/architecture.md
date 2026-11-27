# Gorpx Architecture

Gorpx is a lightweight HTTP reverse proxy and web server written in Go.

## Design Principles

- Simplicity over complexity.
- Use Go's standard library whenever possible.
- Keep packages focused on a single responsibility.
- Avoid unnecessary abstractions and dependencies.
- Prioritise reliability, maintainability, and testability.

See the [repository map](../AGENTS.md#repository-map) for package ownership.

## Startup and shutdown

The CLI loads a single YAML document, rejects unknown fields, applies defaults,
and validates the configuration. It passes the configuration to
`server.NewWithHealth`, which assembles the HTTP server and returns any health
runners for the configured proxy routes. The CLI binds the listener, starts the
health runners, and serves requests.

On interrupt or termination, the CLI stops accepting new connections and
upgrades. In-flight HTTP requests and established upgrades share a ten-second
grace period. Gorpx closes connections still open when the budget expires,
including both ends of upgraded proxy sessions. On exit from serving, it
cancels the health runners and waits for them to stop.

## HTTP Request Lifecycle

Gorpx applies the header size, header read timeout, and idle timeout through
`net/http`. Requests that reach the handler pass through this chain.

```text
Client
  |
  v
net/http server
  |
  v
Logging -> Request ID -> Body size limit -> Body read deadline
  |
  v
Server mux
  |
  +-- GET / and HEAD /             -> built-in greeting
  +-- GET /health and HEAD /health -> built-in health response
  |
  +-- Other requests -> configured route mux
                          +-- Static handler -> open files within the configured root
                          +-- Proxy handler  -> select and forward to an upstream
                          +-- No match       -> 404 Not Found
```

The body read deadline applies to each read from the client. Static routes
serve files under their configured roots. Proxy routes select upstreams in
round-robin order and rebuild the forwarding headers. Optional active probes
update which upstreams are eligible for selection. If no upstream is eligible,
the proxy returns HTTP 503.

See the [configuration reference](configuration.md) for route syntax, limits,
timeouts, and health-check settings.

## Testing

Gorpx uses Go's standard `testing` and `net/http/httptest` packages.

Tests are organised into two categories:

- **Unit tests:** Verify individual packages, including configuration, routing, proxying, and middleware.
- **Integration tests:** Verify that multiple components work together to process HTTP requests correctly.

Integration tests use temporary HTTP servers and filesystem directories to avoid dependencies on external services.

Run all tests using:

```bash
go test ./...
```

Run tests with race detection:

```bash
go test -race ./...
```

## Dependencies

Gorpx uses Go's standard library wherever possible. External dependencies should be introduced only when they provide clear value.

## License

ISC License.
