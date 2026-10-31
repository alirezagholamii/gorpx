# Gorpx

**Gorpx** (Go Reverse Proxy) is a lightweight, open-source HTTP reverse proxy and web server written in Go.

Designed for simplicity, performance, and reliability, Gorpx makes it easy to route HTTP traffic, serve static files, and manage backend services with minimal configuration.

## Features

- HTTP reverse proxy
- Path-based routing
- Static file serving
- YAML-based configuration
- Structured request logging
- Graceful shutdown
- Lightweight with minimal dependencies

> **Status:** 🚧 Early development. Features listed above are planned for the initial release.

## Example Configuration

```yaml
server:
  address: ":8080"

routes:
  - path: /api/
    upstream: http://localhost:3000

  - path: /
    upstream: http://localhost:4200
```

## Getting Started

Installation and usage instructions will be available with the first release.

## Contributing

Contributions, bug reports, and feature suggestions are welcome.

## License

Gorpx is licensed under the [ISC License](LICENSE).