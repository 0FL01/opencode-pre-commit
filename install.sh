#!/usr/bin/env bash
set -euo pipefail

REPO="0FL01/opencode-pre-commit"
BRANCH="${2:-main}"
HOOK_PATH="$(git rev-parse --show-toplevel)/.git/hooks/commit-msg"

if [ -f "$HOOK_PATH" ]; then
    echo "A commit-msg hook already exists at $HOOK_PATH"
    echo "Remove it first if you want to replace it."
    exit 1
fi

BIN_URL="https://raw.githubusercontent.com/${REPO}/${BRANCH}/opencode-pre-commit"
BIN_DIR="$(mktemp -d)"
BIN_PATH="${BIN_DIR}/opencode-pre-commit"

echo "Downloading opencode-pre-commit..."
if command -v curl &>/dev/null; then
    curl -sL --fail -o "$BIN_PATH" "$BIN_URL" || { echo "Download failed"; exit 1; }
elif command -v wget &>/dev/null; then
    wget -q -O "$BIN_PATH" "$BIN_URL" || { echo "Download failed"; exit 1; }
else
    echo "Error: curl or wget is required"
    exit 1
fi

chmod +x "$BIN_PATH"

cat > "$HOOK_PATH" << EOF
#!/usr/bin/env bash
exec $BIN_PATH "\$1"
EOF

chmod +x "$HOOK_PATH"
echo "Installed commit-msg hook at $HOOK_PATH"
