# Configuration

Gorpx uses YAML configuration files.

Specify the configuration file when starting the application:

```bash
./gorpx -config gorpx.yaml
```

## Example

```yaml
server:
  address: "127.0.0.1:8080"

routes:
  - path: /api/
    upstream: http://localhost:3000

  - path: /app/
    upstream: http://localhost:4200

  - path: /static/
    root: ./public
```

## Server

| Field            | Description                             |
| ---------------- | --------------------------------------- |
| `server.address` | Address and port on which Gorpx listens |

If the address is omitted, Gorpx uses the current default of `:8080`.

For local development, prefer `127.0.0.1:8080` to avoid exposing the listener on external network interfaces.

## Routes

Each route contains a `path` and exactly one destination.

| Field      | Description                                    |
| ---------- | ---------------------------------------------- |
| `path`     | URL path prefix, beginning and ending with `/` |
| `upstream` | Destination HTTP or HTTPS server               |
| `root`     | Local directory containing static files        |

### Reverse proxy

```yaml
routes:
  - path: /api/
    upstream: http://localhost:3000
```

A request to `/api/users` is forwarded to `http://localhost:3000/api/users`.

Gorpx currently preserves the incoming request path.

### Static files

```yaml
routes:
  - path: /static/
    root: ./public
```

A request to `/static/style.css` serves `./public/style.css`.

Relative filesystem paths are resolved from the process's working directory.

Only use directories containing files intended for public access. The current implementation may serve directory listings.

## Route validation

- Route paths must start and end with `/`.
- Duplicate route paths are rejected.
- Each route must define either `upstream` or `root`, but not both.
- Upstream URLs must use `http` or `https`.
- Upstream URLs currently cannot include custom paths, query strings, fragments or user information.

## Limitations

- No automatic HTTPS or certificate management.
- No load balancing or health checks.
- No configuration hot reloading.
- No URL prefix rewriting.
- No authentication or access-control system.

Gorpx is currently intended for development and controlled testing environments.
