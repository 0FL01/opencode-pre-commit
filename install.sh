#!/usr/bin/env bash
set -euo pipefail

REPO="0FL01/opencode-pre-commit"
BRANCH="${2:-main}"
HOOK_DIR="$(git rev-parse --show-toplevel)/.git/hooks"
HOOK_PATH="${HOOK_DIR}/commit-msg"
BIN_PATH="${HOOK_DIR}/opencode-pre-commit"

if [ -f "$HOOK_PATH" ]; then
    echo "A commit-msg hook already exists at $HOOK_PATH"
    echo "Remove it first if you want to replace it."
    exit 1
fi

echo "Downloading opencode-pre-commit..."
if command -v curl &>/dev/null; then
    curl -sL --fail -o "$BIN_PATH" "https://raw.githubusercontent.com/${REPO}/${BRANCH}/opencode-pre-commit" || { echo "Download failed"; exit 1; }
elif command -v wget &>/dev/null; then
    wget -q -O "$BIN_PATH" "https://raw.githubusercontent.com/${REPO}/${BRANCH}/opencode-pre-commit" || { echo "Download failed"; exit 1; }
else
    echo "Error: curl or wget is required"
    exit 1
fi

chmod +x "$BIN_PATH"

cat > "$HOOK_PATH" << EOF
#!/usr/bin/env bash
exec "\$(dirname "\$0")/opencode-pre-commit" "\$1"
EOF

chmod +x "$HOOK_PATH"
echo "Installed commit-msg hook at $HOOK_PATH"
echo "Binary installed at $BIN_PATH"
