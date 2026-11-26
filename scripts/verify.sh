#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

repo=$PWD
base=${1:-HEAD}
staged=false
if [[ $base == --staged && $# == 2 ]]; then
    repo=$2
    base=HEAD
    staged=true
elif [[ $# -gt 1 || $base == -* ]]; then
    echo "Usage: $0 [base-ref]" >&2
    exit 2
fi

export GOWORK=off
export GOFLAGS=-mod=readonly

GO111MODULE=off go run ./scripts/guardrails/main.go -repo "$repo" -base "$base" -staged="$staged"

go vet ./...
GORPX_RUN_EVALS=0 go test -race -count=1 ./...

build_dir=$(mktemp -d "${TMPDIR:-/tmp}/gorpx-build.XXXXXXXX")
trap 'rm -rf "$build_dir"' EXIT
go build -o "$build_dir/gorpx" ./cmd/gorpx
