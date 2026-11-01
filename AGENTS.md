# Gorpx — Agent Instructions

## Project Overview

Gorpx is a lightweight HTTP reverse proxy and static web server written in Go.

Module: `github.com/alirezagholamii/gorpx`

The project prioritises simplicity, reliability, security, and maintainability.

## Development Principles

- Prefer simple, idiomatic Go.
- Use Go's standard library whenever practical.
- Keep changes small, focused, and easy to review.
- Avoid unnecessary abstractions, interfaces, and dependencies.
- Do not introduce features outside the requested scope.
- Preserve existing behaviour unless a change is explicitly required.
- Prioritise code clarity over clever implementations.

## Project Structure

- `cmd/gorpx/` — CLI entry point and application startup.
- `internal/config/` — Configuration loading and validation.
- `internal/server/` — HTTP server setup and lifecycle.
- `internal/router/` — Path-based request routing.
- `internal/proxy/` — Reverse proxy implementation.
- `internal/static/` — Static file serving.
- `internal/middleware/` — HTTP middleware.
- `docs/` — Architecture and configuration documentation.

Keep `main.go` minimal. Do not move routing, proxying, or middleware logic into it.

## Architecture Guidelines

- Keep packages focused on clear responsibilities.
- Prefer Go's existing interfaces, especially `http.Handler`.
- Avoid circular dependencies.
- Do not introduce new packages without a clear reason.
- Use explicit error handling.
- Avoid unnecessary global mutable state.
- Keep concurrency understandable and race-free.

## Security

Gorpx processes untrusted HTTP requests.

- Do not blindly trust client-supplied forwarding headers.
- Do not expose internal errors or sensitive information to clients.
- Validate configuration and upstream destinations.
- Preserve safe timeout and shutdown behaviour.
- Do not expose arbitrary filesystem paths through static-file routes.
- Avoid logging credentials, tokens, or request bodies.
- Consider HTTP streaming and connection behaviour when modifying middleware.

## Testing

Add or update tests when modifying behaviour.

Prefer standard Go testing tools and `net/http/httptest`.

Run:

```bash
go fmt ./...
go vet ./...
go test -race ./...
go build ./cmd/gorpx
```

Report any checks that could not be completed.

## Working with the Maintainer

The maintainer prioritises understanding the entire codebase.

When proposing changes:

1. Explain the problem and the proposed solution.
2. Identify the files that need modification.
3. Describe important design decisions and trade-offs.
4. Prefer incremental changes over large rewrites.
5. Explain non-obvious code and Go concepts.
6. Clearly summarise what changed and how it was tested.

Do not perform broad refactoring or introduce major architectural changes without explicit approval.

## Scope

Gorpx v0.1.0 is an experimental release.

Do not introduce load balancing, automatic HTTPS, caching, plugin systems, or a management dashboard unless explicitly requested.

Focus on stabilising the existing functionality.

## Documentation

Keep `README.md`, `docs/architecture.md`, and `docs/configuration.md` consistent with the implementation.

Document actual behaviour rather than planned functionality.

## Licence

Gorpx is licensed under the ISC License. Preserve existing copyright and licence notices.