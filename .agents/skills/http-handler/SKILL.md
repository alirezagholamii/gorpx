---
name: http-handler
description: Add or change a Gorpx HTTP endpoint, route, or middleware while preserving request and response semantics.
---

Use this skill when a coding agent changes HTTP behavior.

1. Read `internal/server/server.go` and the owning package. Mount built-in endpoints in the outer server mux and configured routes in `internal/router`. Keep the CLI focused on lifecycle.
2. Check exact versus subtree patterns, method matching, redirects, and the middleware order. For config changes, follow the fields through strict YAML decoding, defaults and validation.
3. Preserve client-facing error behavior and request IDs. For response-writer wrappers, retain `Unwrap` and exercise the capabilities the handler uses, such as flushing or setting read deadlines.
4. Add focused tests in the owning package. Drive wiring changes through `internal/server/integration_test.go` with loopback upstreams and temporary static roots. Run `./scripts/verify.sh`.
