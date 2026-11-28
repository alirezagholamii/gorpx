# Phase 3. Automated release builds

Build downloadable Gorpx binaries for Linux amd64 and arm64, macOS amd64 and
arm64, and Windows amd64 when an approved version tag is pushed to GitHub.
Work continues on the local `release/automated-builds` branch. Creating tags,
pushing changes, and publishing releases require explicit approval.

## Done

- [x] **Task 1. Verify local cross-compilation.** The CLI entry point is
  `./cmd/gorpx`, and `scripts/build.sh` builds it for the host. The module
  specifies Go 1.27.4 and requires `go.yaml.in/yaml/v3` v3.0.5.
  `.github/workflows/ci.yml` verifies pushes and pull requests to `main` on
  Ubuntu with `contents: read` and Go selected from `go.mod`.
- [x] Built all five targets from clean commit `f82ef1c` on an arm64 Mac
  with Go 1.27.4. Verified the executable format and architecture with
  `file` and inspected Go build metadata with `go version -m`. The native
  macOS arm64 binary displayed CLI help with `-h`. The other binaries were
  inspected as files. Temporary build outputs were removed.

| Target | Verified executable |
| --- | --- |
| Linux amd64 | ELF x86-64 |
| Linux arm64 | ELF AArch64 |
| macOS amd64 | Mach-O x86-64 |
| macOS arm64 | Mach-O arm64 |
| Windows amd64 | PE32+ x86-64 |

The build command for each pair was
`CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build -trimpath -mod=readonly -o "$binary" ./cmd/gorpx`.
`GOOS` sets the target operating system (`darwin` for macOS), and `GOARCH`
sets the target CPU architecture. Disabling cgo avoids a target C toolchain.
`-trimpath` removes absolute build paths, `-mod=readonly` requires the module
requirements to be usable without updating `go.mod`, and `-o` names the
output. The Windows output used `.exe`.

### Task 2. Binary and archive names

Use `gorpx_<version>_<goos>_<goarch>.<extension>` for release archives.
Use the validated Git tag verbatim as the version, including its leading
`v`. Use Go's target names in filenames, including `darwin` for macOS.

| Target | Archive for `v1.0.0` | Executable inside |
| --- | --- | --- |
| Linux amd64 | `gorpx_v1.0.0_linux_amd64.tar.gz` | `gorpx` |
| Linux arm64 | `gorpx_v1.0.0_linux_arm64.tar.gz` | `gorpx` |
| macOS amd64 | `gorpx_v1.0.0_darwin_amd64.tar.gz` | `gorpx` |
| macOS arm64 | `gorpx_v1.0.0_darwin_arm64.tar.gz` | `gorpx` |
| Windows amd64 | `gorpx_v1.0.0_windows_amd64.zip` | `gorpx.exe` |

Place the executable and `LICENSE` at each archive's root. Preserve executable
permissions in the tarballs. Produce a single `SHA256SUMS` beside the archives,
using one GNU `sha256sum` entry per archive basename in bytewise filename order.

For manual practice builds, substitute `dev-<commit>` for the version, with
`<commit>` set to the first 12 hexadecimal characters of the checked-out Git
commit. On tagged builds, check the embedded version and revision with
`go version -m`.

### Task 3. GitHub Actions release workflow

`.github/workflows/release.yml` builds and packages all five targets in one
Ubuntu job using Go from `go.mod`. The tag path accepts stable
`vMAJOR.MINOR.PATCH` versions and checks the Go version metadata and Git
revision on every binary. The manual path uploads practice archives to
Actions with the default read-only token.

The tag-only publish job downloads the successful build's artifact by ID,
verifies the checksums and checks that the remote tag still points to the
built commit. That job uses `contents: write` to call
`gh release create --verify-tag` using the built-in token. Actions are pinned
by commit. Protect version tags against updates and deletions on GitHub so
the tag cannot move between the remote check and release creation.

Local checks passed for two manual builds with matching archive checksums,
all five builds using the existing `v0.5.0` tag in a temporary checkout,
archive contents and permissions, and a stubbed publish step with both a
matching and a moved tag. The repository verifier passed.

## Failed / Dead Ends

- The local sandbox denied writes to Go's module version stat cache. Each
  build exited successfully and retained the expected Git revision and clean
  tree metadata, as well as a Go-generated module pseudo-version.

## Next Steps

- [ ] **Task 4. Run a GitHub-hosted rehearsal.** With approval, push the
  changes and make the workflow available on the default branch, which
  GitHub requires for manual dispatch. Download the workflow artifact and
  verify all five archive names, contents, build metadata, and SHA-256
  checksums. Confirm the publish job was skipped.
- [ ] **Task 5. Document binary installation.** Cover platform selection,
  checksum verification, extraction, adding the executable to `PATH`, and
  running Gorpx with a config file.

Tasks 1 through 3 are complete locally. The GitHub-hosted rehearsal awaits
approval to push and land the workflow on the default branch.
