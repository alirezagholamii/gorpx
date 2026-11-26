# Agent engineering for Gorpx

Deliver the Learn, Guide, Review and Score layers for coding agent CLIs while preserving
the proxy's runtime behavior.

## Done

- Inspected the Go module, packages, existing scripts, Git hooks and CI.
- Added the repository map, workflow skills, verification entrypoint and
  staged pre-commit hook.
- Added the worktree eval harness, two fixture cases and a scoring rubric.
- Ran Go formatting, vet, the full race suite and the CLI build.
- Verified hook rejection cases and a valid partial commit in a disposable
  repository. Both eval cases passed with a controlled agent CLI fixture.
- Added selectable adapters and a custom executable protocol. Verified all
  built-in trace formats with offline fixtures and exercised a two-turn custom
  run through worktree cleanup and acceptance tests.
- Confirmed that verification stays offline even when the caller enables live
  evaluations.

## Failed / Dead Ends

- The managed workspace blocked writes to the default Go build cache and
  loopback binds. Validation used a temporary cache and approved loopback
  access.
- Automatic approval review blocked the initial authenticated agent run because
  it could transmit repository contents to the service.

## Next Steps

- Install the hook after committing the tooling.
- Run the authenticated evals when repository access for the selected provider is authorized.
