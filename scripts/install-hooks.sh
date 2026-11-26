#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."
if git config --get core.hooksPath > /dev/null; then
    echo "core.hooksPath is configured. Keep the existing hook setup and integrate scripts/hooks/pre-commit there." >&2
    exit 1
fi

hook_path=$(git rev-parse --git-path hooks/pre-commit)
mkdir -p "$(dirname "$hook_path")"
if [[ -L $hook_path ]] || { [[ -e $hook_path ]] && ! grep -Fxq '# gorpx-managed-pre-commit' "$hook_path"; }; then
    echo "An existing pre-commit hook is installed at $hook_path." >&2
    exit 1
fi
if ! git cat-file -e HEAD:scripts/verify.sh; then
    echo "Commit the verification tooling before installing the hook." >&2
    exit 1
fi
if ! git diff --quiet HEAD -- scripts/hooks/pre-commit scripts/install-hooks.sh; then
    echo "Commit the hook and installer changes before installing the hook." >&2
    exit 1
fi

temp_hook=$(mktemp "${hook_path}.XXXXXXXX")
trap 'rm -f "$temp_hook"' EXIT
git show HEAD:scripts/hooks/pre-commit > "$temp_hook"
chmod 755 "$temp_hook"
mv "$temp_hook" "$hook_path"
echo "Installed $hook_path"
