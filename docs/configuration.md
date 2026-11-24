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
  read_header_timeout: 5s
  idle_timeout: 60s
  max_header_bytes: 16384
  max_body_bytes: 10485760

routes:
  - path: /api/
    upstreams:
      - http://localhost:3000
      - http://localhost:3001
    response_header_timeout: 15s

  - path: /app/
    upstreams:
      - http://localhost:4200

  - path: /static/
    root: ./public
```

## Server

| Field | Description |
|---|---|
| `server.address` | Address and port on which Gorpx listens |

Set `server.address` in the configuration file. Use `:8080` to listen on all interfaces.

For local development, prefer `127.0.0.1:8080` to avoid exposing the listener on external network interfaces.

## Routes

Each route contains a `path` and exactly one destination type.

| Field | Description |
|---|---|
| `path` | URL path prefix, beginning and ending with `/` |
| `upstreams` | Non-empty list of HTTP or HTTPS destinations |
| `root` | Local directory containing static files |
| `response_header_timeout` | Optional timeout for reverse-proxy routes |
| `health_check` | Optional [upstream health checks](#upstream-health-checks) for reverse-proxy routes |

Routes use literal URL path prefixes. Wildcard patterns are not supported.

### Reverse proxy

```yaml
routes:
  - path: /api/
    upstreams:
      - http://localhost:3000
```

A request to `/api/users` is forwarded to `http://localhost:3000/api/users`.

Gorpx preserves the incoming request path.

The `upstreams` field must contain at least one URL, even when a route has only one backend.

### Round-robin load balancing

Gorpx supports multiple upstream servers using round-robin load balancing.

```yaml
routes:
  - path: /api/
    upstreams:
      - http://localhost:3000
      - http://localhost:3001
      - http://localhost:3002
```

For sequential requests, Gorpx selects upstreams in the following order:

| Request | Selected upstream |
|---|---|
| 1 | `localhost:3000` |
| 2 | `localhost:3001` |
| 3 | `localhost:3002` |
| 4 | `localhost:3000` |
| 5 | `localhost:3001` |
| 6 | `localhost:3002` |

Each route maintains an independent, concurrency-safe round-robin selector.

Under concurrent traffic, selection remains balanced, but responses may complete in a different order.

**Failure behaviour:**

- When health checks are disabled, unavailable upstreams remain in the round-robin rotation.
- When Gorpx cannot connect to the selected upstream or times out waiting for its response headers, it normally returns HTTP 502 Bad Gateway.
- Gorpx does not retry failed requests against another upstream.

Round-robin distributes requests by selection count. It does not consider backend load, response times or connection counts.

### Upstream health checks

Starting in v0.5.0, reverse-proxy routes can enable active health checks with a `health_check` block. The block works with one or more upstreams. Omitting it disables health checks for the route.

```yaml
routes:
  - path: /api/
    upstreams:
      - http://localhost:9001
      - http://localhost:9002
    health_check:
      path: /health
      interval: 10s
      timeout: 2s
```

Within an enabled block, omitted or empty settings use these defaults.

| Setting | Default | Description |
|---|---|---|
| `path` | `/health` | Path requested directly on each upstream |
| `interval` | `10s` | Interval between probe ticks |
| `timeout` | `2s` | Time limit for the probe to receive response headers |

All upstreams are initially treated as healthy. Gorpx sends an HTTP GET probe to the configured path on each upstream as soon as monitoring starts, then probes periodically. An HTTP 2xx status marks the upstream healthy. Any other status, a timeout or a connection error marks it unhealthy. Gorpx does not follow redirects.

Round-robin selection skips unhealthy upstreams. A successful probe returns a recovered upstream to the rotation. When all upstreams on a route are marked unhealthy, Gorpx returns HTTP 503 Service Unavailable for requests to that route.

**Validation rules**

- Only reverse-proxy routes with `upstreams` can use `health_check`.
- The probe path must start with `/`. It cannot start with `//` or contain `?`, `#`, `{`, `}`, `%`, backslashes, spaces, tabs, carriage returns or line feeds. Path segments of `.` or `..` are also rejected.
- `interval` and `timeout` must be positive Go durations. The timeout must not exceed the interval.

Invalid health-check settings prevent Gorpx from starting.

### Static files

```yaml
routes:
  - path: /static/
    root: ./public
```

A request to `/static/style.css` serves `./public/style.css`.

Relative filesystem paths are resolved from the process's working directory.

Static-file serving is restricted to the configured root directory. Directory listings are disabled, and paths that escape the configured root through symbolic links are rejected.

Only use directories containing files intended for public access.

## Route validation

Gorpx validates routes at startup.

- Route paths must start and end with `/`.
- Route paths must use supported literal-prefix syntax; wildcard patterns are rejected.
- Duplicate route paths are rejected.
- Each route must define exactly one of `upstreams` or `root`.
- The `upstreams` list must contain at least one URL.
- Every upstream URL must use `http` or `https`.
- Upstream URLs cannot contain custom base paths, query strings, fragments or user information.
- Explicit upstream ports must be valid TCP port numbers.
- Static-file roots must exist and be directories.
- `response_header_timeout` is allowed only on reverse-proxy routes.

Invalid configurations cause Gorpx to return an error during startup.

### Breaking configuration change in v0.4.0

Gorpx v0.4.0 uses `upstreams` for all reverse-proxy routes.

The previous singular `upstream` field is no longer supported.

Old configuration:

```yaml
routes:
  - path: /api/
    upstream: http://localhost:3000
```

New configuration:

```yaml
routes:
  - path: /api/
    upstreams:
      - http://localhost:3000
```

Existing configuration files must be updated to use the new format.

## Server timeouts

Gorpx supports configurable HTTP server timeouts.

```yaml
server:
  address: ":8080"
  read_header_timeout: 5s
  idle_timeout: 60s
```

Both timeout settings are optional.

| Setting | Default | Description |
|---|---|---|
| `read_header_timeout` | `5s` | Maximum time allowed to read incoming HTTP request headers |
| `idle_timeout` | `60s` | Maximum time to wait for another request on an idle keep-alive connection |

### Supported duration formats

Timeout values use Go duration syntax:

- `500ms` — 500 milliseconds
- `10s` — 10 seconds
- `2m` — 2 minutes
- `1h` — 1 hour

Timeout values must be greater than zero. Invalid, zero or negative durations cause Gorpx to reject the configuration at startup.

### Default values

Configurations that omit timeout settings use the defaults of `5s` and `60s`.

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

Requests with a declared `Content-Length` greater than `max_body_bytes` are rejected with HTTP 413 Content Too Large.

For requests with an unknown body size, Gorpx enforces the limit while streaming the body.

Gorpx does not buffer complete request bodies in memory.

If a streamed request exceeds the limit after forwarding has begun, the client may receive a transport error or a different HTTP status, such as 502.

The upstream server may receive part of the request before the limit is reached.

### Header limits

Gorpx uses Go's `http.Server.MaxHeaderBytes` setting to limit incoming request headers.

The HTTP server may use a small amount of additional internal buffering beyond the configured header limit.

## HTTP request logging

Gorpx uses Go's `log/slog` package for structured HTTP request logging.

Access logs include:

| Field | Description |
|---|---|
| `method` | HTTP request method |
| `path` | Request URL path |
| `status` | HTTP response status code |
| `duration_ms` | Request duration in milliseconds |
| `client_ip` | IP address of the direct client connection |
| `request_id` | Gorpx-generated request correlation ID |
| `aborted` | Whether the handler terminated through a panic, including an aborted response |

Gorpx generates a unique request ID for each incoming request.

The ID is returned to clients through the `X-Request-ID` response header and forwarded to upstream services.

Client-provided request IDs are replaced by Gorpx-generated identifiers.

The generated request ID is restored in outgoing proxy requests after hop-by-hop headers have been removed.

### Upstream error logging

When a request to a selected upstream fails before response headers are sent, Gorpx records a structured error log containing:

- The selected upstream host.
- The request path.
- The generated request ID.
- The underlying error details.

The selected upstream is tracked per request, so concurrent requests do not overwrite each other's error context.

### Aborted responses

A proxied response may begin successfully but fail while its body is being transferred.

For example, an upstream might send HTTP 200 and then close its connection before completing the response.

In this situation, Gorpx preserves the committed HTTP status and records the aborted outcome separately.

An aborted response is not necessarily an HTTP 502 response, because response headers may already have been sent to the client.

### Privacy considerations

Gorpx does not intentionally log request bodies, authorisation headers or URL query parameters in its access logs.

The `client_ip` field is derived from the direct connection address rather than untrusted forwarding headers.

## Upstream response-header timeout

Gorpx allows each reverse-proxy route to configure its own upstream response-header timeout.

```yaml
routes:
  - path: /api/
    upstreams:
      - http://localhost:3000
      - http://localhost:3001
    response_header_timeout: 30s

  - path: /reports/
    upstreams:
      - http://localhost:4000
    response_header_timeout: 2m
```

### Configuration

| Setting | Default | Description |
|---|---|---|
| `response_header_timeout` | `15s` | Maximum time to wait for upstream response headers |

The setting is optional and applies only to reverse-proxy routes.

Supported values follow Go's duration syntax, including `500ms`, `15s`, `1m` and `2m`.

Timeout values must be greater than zero. Invalid, zero or negative values cause Gorpx to reject the configuration at startup.

### Behaviour

If the selected upstream does not return response headers before the configured timeout expires, Gorpx normally returns HTTP 502 Bad Gateway.

The timeout does not limit the time required to stream the upstream response body.

For routes with multiple upstreams, the same timeout applies to each selected upstream.

A timeout does not automatically trigger a retry against another upstream.

### Static routes

`response_header_timeout` is not supported for static-file routes.

The following configuration is invalid:

```yaml
routes:
  - path: /static/
    root: ./public
    response_header_timeout: 30s
```

Configurations that omit the timeout use the default of 15 seconds.

## Limitations

- No automatic HTTPS or certificate management.
- No passive upstream health checks or circuit breakers.
- No automatic retries of failed proxied requests at the routing layer.
- No weighted or least-connections load balancing.
- No configuration hot reloading.
- No URL prefix rewriting.
- No built-in authentication or access-control system.

Gorpx is currently intended for development and controlled testing environments.
