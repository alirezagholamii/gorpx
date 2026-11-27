# Gorpx agent instructions

Gorpx is a lightweight HTTP reverse proxy and static file server.
The module is `github.com/alirezagholamii/gorpx`. Use the Go version in `go.mod`.

## Repository map

| Path | Responsibility |
| --- | --- |
| `cmd/gorpx/` | CLI, config loading, listener and process lifecycle, health-runner startup, graceful shutdown |
| `internal/config/` | Strict YAML decoding, defaults, config types and validation |
| `internal/server/` | HTTP server settings, built-in endpoints and middleware composition |
| `internal/router/` | Route validation and mounting proxy or static handlers |
| `internal/proxy/` | Upstream transport, round-robin selection and active health checks |
| `internal/static/` | Static file serving and filesystem confinement |
| `internal/middleware/` | Request IDs, logging, body size and body-read deadlines |
| `examples/`, `public/` | Runnable example configuration and static content |
| `docs/`, `evals/` | Behavior documentation, task plans and opt-in coding agent regression cases |

## Working conventions

- Keep `main.go` focused on startup and lifecycle. Put request handling in the owning package and compose it with `http.Handler`.
- Prefer the standard library, explicit errors, focused changes and straightforward concurrency. Preserve existing behavior outside the requested scope.
- Validate configuration and upstream destinations. Treat incoming forwarding headers and paths as untrusted. Keep client errors generic and credentials, request bodies and tokens out of logs.
- Preserve timeout and shutdown behavior, streaming and response-writer capabilities. Give goroutines clear cancellation and cleanup ownership.
- Add or update Go tests for behavior changes. Use `httptest` and temporary directories. Run `go fmt ./...` after editing Go.
- Keep the README, architecture and configuration documentation accurate when changing documented behavior. Preserve the ISC license and copyright notices.
- Stabilize current functionality. Get explicit approval for broad refactors or new capabilities such as automatic HTTPS, caching, plugins or a dashboard.
- Copy `docs/plans/TEMPLATE.md` for multi-step work. Update it at checkpoints so a new session can resume from the last verified state.

## Commands

| Command | Purpose |
| --- | --- |
| `./scripts/verify.sh [base-ref]` | Check guardrails, Go formatting, vet, race tests and the CLI build. Defaults to `HEAD`. |
| `./scripts/install-hooks.sh` | Install the local pre-commit hook. It verifies the staged snapshot. |
| `./scripts/build.sh` | Build `bin/gorpx`. |
| `./scripts/run.sh [config]` | Build and run Gorpx. Defaults to `examples/basic.yaml`. |
| `GORPX_RUN_EVALS=1 go test -v -count=1 -timeout=30m ./evals` | Run live coding agent evaluations from a committed baseline. Select the CLI with `AGENT_RUNNER`. See `evals/README.md`. |

## Protected files and test coverage

- Never modify `.git/`, `LICENSE`, the verification and hook scripts, `scripts/guardrails/`, `evals/`, or CI workflows during routine tasks. Changes to these paths require a task explicitly authorizing maintenance of that path.
- Never modify `go.mod` or `go.sum` unless the maintainer explicitly authorizes a dependency or Go-version change.
- Never remove an existing test or add a skip without maintainer approval and a specific reason. The verifier reports the identifiers for reviewed exceptions.
- Never weaken a guardrail, grant an exception, bypass a hook, or edit evaluation fixtures to make a task pass. The maintainer owns those decisions.

## Shared coding agent skills

Read only the relevant `SKILL.md` when needed.

- `.agents/skills/http-handler/` for changes to routes, endpoints or middleware.
- `.agents/skills/go-tests/` for focused Go tests and HTTP integration coverage.
- `.agents/skills/code-review/` for a fresh-context diff review before handoff.
