#!/usr/bin/env bash
set -euo pipefail

HOOK_PATH="$(git rev-parse --show-toplevel)/.git/hooks/commit-msg"

if [ -f "$HOOK_PATH" ]; then
    echo "A commit-msg hook already exists at $HOOK_PATH"
    echo "Remove it first if you want to replace it."
    exit 1
fi

# Check if binary is on PATH or build it.
if command -v opencode-pre-commit &>/dev/null; then
    BIN="opencode-pre-commit"
else
    echo "Building opencode-pre-commit..."
    go install github.com/plutov/opencode-pre-commit@latest
    BIN="$(go env GOPATH)/bin/opencode-pre-commit"
fi

cat > "$HOOK_PATH" << EOF
#!/usr/bin/env bash
exec $BIN "\$1"
EOF

chmod +x "$HOOK_PATH"
echo "Installed commit-msg hook at $HOOK_PATH"
