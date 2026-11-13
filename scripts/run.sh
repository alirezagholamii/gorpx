
#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

CONFIG="${1:-examples/basic.yaml}"

if [[ ! -f "$CONFIG" ]]; then
    echo "Configuration file not found: $CONFIG" >&2
    exit 1
fi

./scripts/build.sh

echo "Starting Gorpx with $CONFIG..."
exec ./bin/gorpx -config "$CONFIG"
