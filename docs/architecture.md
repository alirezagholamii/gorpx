# Gorpx Architecture

Gorpx is a lightweight HTTP reverse proxy and web server written in Go.

## Design Principles

- Simplicity over complexity.
- Use Go's standard library whenever possible.
- Keep packages focused on a single responsibility.
- Avoid unnecessary abstractions and dependencies.
- Prioritise reliability, maintainability, and testability.

## Project Structure

### `cmd/gorpx`

The application entry point.

Responsibilities:

- Initialise application configuration.
- Initialise and start the HTTP server.
- Handle application startup errors.

### `internal/config`

Manages application configuration.

Responsibilities:

- Define configuration structures.
- Provide default values.
- Eventually load and validate YAML configuration.

## HTTP Request Lifecycle

Currently, Gorpx handles requests using Go's `net/http` package.

```text
Client
  |
  v
HTTP Server
  |
  v
ServeMux
  |
  +-- GET /health --> 200 OK
  |
  +-- GET / -------> Hello from Gorpx!
  |
  +-- Unknown path -> 404 Not Found
```

## Planned Components

The following packages will be introduced as needed:

- `internal/server`: HTTP server lifecycle and graceful shutdown.
- `internal/router`: Request routing.
- `internal/proxy`: Reverse proxying to upstream services.
- `internal/static`: Static file serving.
- `internal/middleware`: Request logging and HTTP middleware.

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
