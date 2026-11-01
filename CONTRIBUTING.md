# Contributing to Gorpx

Thanks for your interest in contributing to Gorpx!

Gorpx aims to remain small, understandable, and maintainable.

## Development Setup

1. Fork and clone the repository.
2. Install the Go version specified in `go.mod`.
3. Make your changes in a dedicated branch.
4. Add or update tests where appropriate.
5. Open a pull request explaining your changes.

## Code Guidelines

- Follow idiomatic Go conventions.
- Prefer Go's standard library.
- Keep changes small and focused.
- Avoid unnecessary abstractions and external dependencies.
- Write clear, maintainable code.
- Handle errors explicitly.
- Add tests for new functionality and bug fixes.

## Testing

Before submitting a pull request, run:

```bash
go fmt ./...
go vet ./...
go test -race ./...
go build ./cmd/gorpx
```

All checks should pass.

## Commit Messages

Use concise, descriptive commit messages.

Examples:

```text
feat: add request logging
fix: handle upstream connection errors
test: cover invalid configurations
docs: improve configuration examples
```

## Pull Requests

Please describe:

- What changed.
- Why the change is needed.
- How it was tested.

For larger features or architectural changes, consider opening an issue for discussion first.

## Security

Please avoid publishing sensitive security vulnerabilities in public issues. Until a private reporting channel is configured, contact the maintainer privately.

## License

By contributing, you agree that your contributions will be distributed under the project's ISC License.