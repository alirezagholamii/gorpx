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

Format Go changes and run the same checks as CI before submitting a pull request.

```bash
go fmt ./...
./scripts/verify.sh
```

The verifier defaults to comparing with `HEAD`. On a branch, pass the base
commit or use `./scripts/verify.sh "$(git merge-base HEAD origin/main)"`
to check committed changes as well. CI supplies the pull request base or
previous push commit. Go formatting is the linter fallback when no standalone
linter is configured.

Install the local pre-commit hook from a checkout containing the committed
verification tooling.

```bash
./scripts/install-hooks.sh
```

It verifies the staged tree, including partially staged files. The installer
preserves an existing unmanaged hook or custom `core.hooksPath`. The hook
blocks staged changes to verification tooling, evals, CI workflows and the
license.

For authorized manifest changes, removed tests or new skip calls, the
maintainer can record the verifier's reported identifier with a reason in
`scripts/guardrails/exceptions.txt`. For an intentional protected
maintenance commit, the maintainer can run the verifier manually and
make that single commit with `git -c core.hooksPath=/dev/null commit`.
The secret check scans changed text files for recognizable credential
patterns and reports locations. Treat it as a heuristic.

The test inventory compares top-level test names and direct skip calls. Review
test diffs for removed assertions or table cases and for build constraints
that exclude tests on the CI platform.

See [Coding agent evaluations](evals/README.md) for the opt-in regression suite.

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

Use a fresh read-only review for agent-authored changes. Record actionable
findings and how they were resolved in the pull request. For changes to proxy
selection, transport, streaming, or middleware, check the cost on the request
path. See the [code-review skill](.agents/skills/code-review/SKILL.md).

Review changes to CI workflows, verification and hook scripts, guardrails
and exceptions, and eval fixtures explicitly. Ask a second maintainer to
approve these changes when available. The required CI check runs the
candidate's verification policy, so a green run alone cannot establish that
the policy still enforces the intended rules.

When reviewing solo, check the protected diff and record your decision in
the pull request. After the initial harness merge, make a disposable local
checkout of the PR commit, with the trusted PR base available in its history.
From that checkout, set `base` to the reviewed base commit SHA and run the
guard with the base's source and exception file.

```bash
git restore --source="$base" --worktree -- scripts/guardrails/main.go scripts/guardrails/exceptions.txt &&
    GO111MODULE=off GOTOOLCHAIN=local GOWORK=off go run ./scripts/guardrails/main.go -base "$base"
```

Use the original PR diff to review policy changes and any intentional
exceptions. For the initial harness pull request, review the new policy
directly and confirm that the required `Build and Test` check runs on GitHub
before merging.

For larger features or architectural changes, consider opening an issue for discussion first.

## Security

Please avoid publishing sensitive security vulnerabilities in public issues. Until a private reporting channel is configured, contact the maintainer privately.

## License

By contributing, you agree that your contributions will be distributed under the project's ISC License.
