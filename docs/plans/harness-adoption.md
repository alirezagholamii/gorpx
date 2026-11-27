# Harness adoption

Update the architecture guide, document how maintainers independently review
harness policy changes, and verify the proposal locally and in its first
GitHub pull request.

## Done

- Updated `docs/architecture.md` against the CLI, server, router, and proxy
  implementation.
- Updated `CONTRIBUTING.md` with review expectations for protected changes
  and the limits of the test inventory.
- Verified the trusted-base guard procedure in a disposable checkout.
- Passed `./scripts/verify.sh HEAD` with loopback access and
  `git diff --check`. The independent documentation review found no
  remaining issues.

## Failed / Dead Ends

- None.

## Next Steps

- Obtain authorization to publish the existing harness proposal, then confirm
  the required GitHub check runs on the resulting pull request.
