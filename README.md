# Gorpx

**Gorpx (Go Reverse Proxy)** is a lightweight, open-source HTTP reverse proxy and static web server written in Go.

Built for simplicity, Gorpx provides configurable routing, structured logging, and basic server lifecycle management with minimal dependencies.

> **Status:** Experimental. Gorpx is under active development and has not yet been hardened for public-facing production deployments.

## Features

- HTTP reverse proxy
- Path-based routing
- YAML configuration
- Static file serving
- Structured request logging
- Graceful shutdown for HTTP requests and upgraded connections
- Basic upstream error handling
- Optional active upstream health checks

## Getting Started

### Install a release binary

For a release that includes binaries, download the archive for your platform
and `SHA256SUMS` from that release's **Assets** section. Go is only required
when building from source. [GitHub Releases](https://github.com/alirezagholamii/gorpx/releases)

| Platform | Example archive for `v1.0.0` |
| --- | --- |
| Linux x86-64 | `gorpx_v1.0.0_linux_amd64.tar.gz` |
| Linux ARM64 (AArch64) | `gorpx_v1.0.0_linux_arm64.tar.gz` |
| macOS Intel | `gorpx_v1.0.0_darwin_amd64.tar.gz` |
| macOS Apple silicon | `gorpx_v1.0.0_darwin_arm64.tar.gz` |
| Windows x86-64 | `gorpx_v1.0.0_windows_amd64.zip` |

#### Linux and macOS

Open a terminal in the download directory, set `archive` to your downloaded
filename, and run the checksum command for your OS.

```bash
archive=gorpx_v1.0.0_darwin_arm64.tar.gz
# Linux
awk -v name="$archive" '$2 == name' SHA256SUMS | sha256sum -c -
# macOS
awk -v name="$archive" '$2 == name' SHA256SUMS | shasum -a 256 -c -
```

If verification reports `OK`, extract the binary to a directory you own and
add it to your current shell's `PATH`. To keep it on your `PATH` in new
terminals, add the export line to your shell's startup file.

```bash
mkdir -p "$HOME/.local/bin" && tar -xzf "$archive" -C "$HOME/.local/bin" gorpx
export PATH="$HOME/.local/bin:$PATH"
gorpx -h
```

#### Windows

Open PowerShell in the download directory and set `$archive` to your downloaded
filename. The commands verify the ZIP before extracting it and adding its
directory to your current session's `PATH`.

```powershell
$archive = 'gorpx_v1.0.0_windows_amd64.zip'
$entry = Get-Content -LiteralPath 'SHA256SUMS' |
    Where-Object { ($_ -split '  ', 2)[1] -ceq $archive }
$expected = ($entry -split '  ', 2)[0]
if ((Get-FileHash -LiteralPath $archive -Algorithm SHA256).Hash -ne $expected) {
    throw "SHA-256 verification failed for $archive."
}

$bin = Join-Path $HOME 'gorpx-bin'
Expand-Archive -LiteralPath $archive -DestinationPath $bin -Force -ErrorAction Stop
$env:Path = "$bin;$env:Path"
gorpx -h
```

For new terminals, add the `gorpx-bin` directory under your home directory to
your user `Path` in Windows Environment Variables.

### Build from source

Install Go 1.27.4 or later (see `go.mod`), then clone and build the repository.

```bash
git clone https://github.com/alirezagholamii/gorpx.git
cd gorpx
go build ./cmd/gorpx
```

### Configuration

Create a file named `gorpx.yaml`:

```yaml
server:
  address: "127.0.0.1:8080"

routes:
  - path: /api/
    upstreams:
      - http://localhost:3000

  - path: /static/
    root: ./public
```

### Run

Create the `public` directory next to `gorpx.yaml` if it does not exist.
From the directory containing the config, start the installed binary.

```bash
gorpx -config gorpx.yaml
```

For a source build in the current directory, use `./gorpx -config gorpx.yaml`
on Linux and macOS or `.\gorpx.exe -config gorpx.yaml` in PowerShell.

Gorpx will listen on `127.0.0.1:8080`.

Requests to `/api/` are forwarded to the configured upstream. Requests to `/static/` are served from the local `public` directory.

Routes preserve their incoming URL prefix when proxying. For example, `/api/users` is forwarded as `/api/users`.

## Documentation

- [Configuration](docs/configuration.md)
- [Architecture](docs/architecture.md)
- [Contributing](CONTRIBUTING.md)

## Development

Run the test suite:

```bash
go test ./...
```

Run tests with race detection:

```bash
go test -race ./...
```

## License

Gorpx is licensed under the [ISC License](LICENSE).
