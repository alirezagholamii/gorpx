---
name: go-tests
description: Write or debug focused table-driven Go tests and HTTP integration tests for Gorpx.
---

Use this skill when a coding agent adds or diagnoses tests.

1. Start with the nearest existing package test. Use named `t.Run` cases for inputs that share setup and assertions. Use `t.Helper` when a helper reports assertion failures.
2. Use `httptest.NewRecorder` for handler responses and `httptest.NewServer` for actual transport, middleware deadlines or streaming. Use `t.TempDir` for YAML and static files.
3. Synchronize concurrent work with channels or contexts. Bound waits and register cleanup for servers, bodies and background workers. Use parallel tests only when fixtures and global state are independent.
4. Assert observable behavior, including errors and preserved defaults. Run the targeted test uncached, then run `./scripts/verify.sh` for the race suite and build.
