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

## Server timeouts

Gorpx supports configurable HTTP server timeouts.

```yaml
server:
  address: ":8080"
  read_header_timeout: 5s
  idle_timeout: 60s
```

Both timeout settings are optional.

| Setting               | Default | Description                                                               |
| --------------------- | ------- | ------------------------------------------------------------------------- |
| `read_header_timeout` | `5s`    | Maximum time allowed to read incoming HTTP request headers                |
| `idle_timeout`        | `60s`   | Maximum time to wait for another request on an idle keep-alive connection |

### Supported duration formats

Timeout values use Go duration syntax:

- `500ms` — 500 milliseconds
- `10s` — 10 seconds
- `2m` — 2 minutes
- `1h` — 1 hour

Timeout values must be greater than zero. Invalid or negative durations cause Gorpx to reject the configuration at startup.

### Backwards compatibility

Existing configuration files that omit timeout settings continue to work.

Gorpx applies the default values of `5s` and `60s`.

### Streaming

These settings do not impose a maximum duration on an entire proxied request or response.

Gorpx deliberately avoids a general write timeout for now to preserve streaming behaviour.

## HTTP request size limits

Gorpx supports configurable HTTP request-header and body-size limits.

```yaml
server:
  address: ":8080"
  max_header_bytes: 16384
  max_body_bytes: 10485760
```

Both settings are optional.

| Setting | Default | Description |
|---|---|---|
| `max_header_bytes` | 16384 (16 KiB) | Maximum HTTP request-header size in bytes |
| `max_body_bytes` | 10485760 (10 MiB) | Maximum HTTP request-body size in bytes |

### Configuration rules

- Values must be non-negative integers representing bytes.
- Omitting a setting uses its default value.
- Setting a value to `0` also uses the default.
- Negative values are rejected during configuration validation.
- Values such as `10MB` are not supported.

### Request-body behaviour

Requests with a declared `Content-Length` greater than `max_body_bytes` are rejected with HTTP 413 (Request Entity Too Large).

For requests with an unknown body size, Gorpx enforces the limit while streaming the body.

Gorpx does not buffer complete request bodies in memory.

If a streamed request exceeds the limit after forwarding has begun, the client may receive a transport error or a different HTTP status, such as 502.

The upstream server may receive part of the request before the limit is reached.

### Header limits

Gorpx uses Go's `http.Server.MaxHeaderBytes` setting to limit incoming request headers.

The HTTP server may use a small amount of additional internal buffering beyond the configured header limit.

## Limitations

- No automatic HTTPS or certificate management.
- No load balancing or health checks.
- No configuration hot reloading.
- No URL prefix rewriting.
- No authentication or access-control system.

Gorpx is currently intended for development and controlled testing environments.
