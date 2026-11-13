
#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

mkdir -p bin

echo "Building Gorpx..."
go build -o bin/gorpx ./cmd/gorpx

echo "Build complete: bin/gorpx"
