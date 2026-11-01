# Gorpx

**Gorpx (Go Reverse Proxy)** is a lightweight, open-source HTTP reverse proxy and static web server written in Go.

Built for simplicity, Gorpx provides configurable routing, structured logging, and basic server lifecycle management with minimal dependencies.

> **Status:** Experimental. Gorpx is under active development and has not yet been hardened for public-facing production deployments.

## Features

- HTTP reverse proxy
- Path-based routing
- YAML configuration
- Static file serving
- Structured request logging
- Graceful shutdown
- Basic upstream error handling

## Getting Started

### Requirements

- Go 1.22 or later (use the version specified in `go.mod`)

### Installation

Clone the repository and build Gorpx:

```bash
git clone https://github.com/alirezagholamii/gorpx.git
cd gorpx
go build -o gorpx ./cmd/gorpx
```

### Configuration

Create a file named `gorpx.yaml`:

```yaml
server:
  address: "127.0.0.1:8080"

routes:
  - path: /api/
    upstream: http://localhost:3000

  - path: /static/
    root: ./public
```

### Run

```bash
./gorpx -config gorpx.yaml
```

Gorpx will listen on `127.0.0.1:8080`.

Requests to `/api/` are forwarded to the configured upstream. Requests to `/static/` are served from the local `public` directory.

Routes preserve their incoming URL prefix when proxying. For example, `/api/users` is forwarded as `/api/users`.

## Documentation

- [Configuration](docs/configuration.md)
- [Architecture](docs/architecture.md)
- [Contributing](CONTRIBUTING.md)

## Development

Run the test suite:

```bash
go test ./...
```

Run tests with race detection:

```bash
go test -race ./...
```

## License

Gorpx is licensed under the [ISC License](LICENSE).
